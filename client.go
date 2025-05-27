package main

import (
	"agent/controller"
	"agent/crawler"
	pb "agent/proto"
	"context"
	"math/rand"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	modeGRPC = "grpc"
	modeAPI  = "api"
	maxConcurrentTasks = 10
)

type SpiderClient struct {
	controller *controller.ControllerClient
	crawler    *crawler.Crawler
	semaphore  chan struct{}
	modeMutex  sync.RWMutex
	token      string
	host       string
	grpcPort   string
	apiPort    string
}

// NewSpiderClient 创建新的客户端实例
func NewSpiderClient(token, host, grpcPort, apiPort string) (*SpiderClient, error) {
	controller, err := controller.NewControllerClient(token, host, grpcPort, apiPort)
	if err != nil {
		return nil, err
	}
	crawler := crawler.NewCrawler()
	return &SpiderClient{
		controller: controller,
		crawler:    crawler,
		semaphore:  make(chan struct{}, maxConcurrentTasks),
		token:      token,
		host:       host,
		grpcPort:   grpcPort,
		apiPort:    apiPort,
	}, nil
}

// GetTask 获取任务
func (c *SpiderClient) GetTask() (*pb.CrawlerTask, error) {
	c.modeMutex.RLock()
	mode := c.controller.GetMode()
	c.modeMutex.RUnlock()
	var task *pb.CrawlerTask
	var err error
	if mode == modeGRPC {
		task, err = c.controller.GetTaskGRPC()
	} else {
		task, err = c.controller.GetTaskAPI()
		if err == nil {
			c.controller.UpdateLastSuccess()
			c.checkAndSwitchToGRPC()
		}
	}
	if err != nil {
		log.Printf("%s 模式获取任务失败: %v", mode, err)
		c.switchMode()
		return nil, err
	}
	return task, nil
}

// HandleTask 处理任务
func (c *SpiderClient) HandleTask(task *pb.CrawlerTask) error {
	if task.Token != c.controller.Token {
		return fmt.Errorf("无效的Token: 传入=%s，期望=%s", task.Token, c.controller.Token)
	}
	if task.Url == "" || task.Tag == "" {
		return fmt.Errorf("无效的URL或Tag")
	}
	startTime := time.Now()
	webData, success := c.crawler.FetchWebData(task.Url)
	runtime := int32(time.Since(startTime).Seconds())
	loc, _ := time.LoadLocation("Asia/Shanghai")
	beijingTime := time.Now().In(loc)
	formattedTime := beijingTime.Format("2006-01-02 15:04:05")
	c.modeMutex.RLock()
	mode := c.controller.GetMode()
	c.modeMutex.RUnlock()
	var err error
	if mode == modeGRPC {
		err = c.controller.HandleTaskGRPC(task, webData, success, runtime, formattedTime)
	} else {
		err = c.controller.HandleTaskAPI(task, webData, success, runtime, formattedTime)
		if err == nil {
			c.controller.UpdateLastSuccess()
			// 只在API模式成功时检查是否切换到gRPC
			c.checkAndSwitchToGRPC()
		}
	}
	if err != nil {
		log.Printf("%s 模式处理任务失败: %v", mode, err)
		c.switchMode()
		return err
	}
	return nil
}

// checkAndSwitchToGRPC 检查是否需要切换回gRPC模式
func (c *SpiderClient) checkAndSwitchToGRPC() {
	c.modeMutex.Lock()
	defer c.modeMutex.Unlock()
	if c.controller.GetMode() == modeAPI {
		c.controller.ModeMutex.RLock()
		stableTime := time.Since(c.controller.LastSuccess)
		c.controller.ModeMutex.RUnlock()
		if stableTime >= 5*time.Minute {
			// 重新创建整个controller，确保token等参数正确
			if newController, err := controller.NewControllerClient(c.token, c.host, c.grpcPort, c.apiPort); err == nil {
				// 尝试gRPC连接
				if err := newController.InitGRPCClient(); err == nil {
					c.controller = newController
					c.controller.SetMode(modeGRPC)
					log.Printf("API模式稳定运行5分钟，成功切换回gRPC模式")
				} else {
					log.Printf("gRPC连接失败，继续使用API模式: %v", err)
				}
			} else {
				log.Printf("重新创建controller失败: %v", err)
			}
		}
	}
}

// switchMode 切换模式
func (c *SpiderClient) switchMode() {
	c.modeMutex.Lock()
	defer c.modeMutex.Unlock()
	currentMode := c.controller.GetMode()
	if currentMode == modeGRPC {
		c.controller.SetMode(modeAPI)
		log.Printf("切换到 %s 模式", modeAPI)
	} else {
		// 重新创建整个controller，确保所有参数正确
		if newController, err := controller.NewControllerClient(c.token, c.host, c.grpcPort, c.apiPort); err == nil {
			if err := newController.InitGRPCClient(); err == nil {
				c.controller = newController
				c.controller.SetMode(modeGRPC)
				log.Printf("切换到 %s 模式", modeGRPC)
			} else {
				log.Printf("gRPC 客户端重新初始化失败，保持API模式: %v", err)
			}
		} else {
			log.Printf("重新创建controller失败: %v", err)
		}
	}
}

// 异步任务处理函数
func (c *SpiderClient) handleTaskAsync(ctx context.Context, t *pb.CrawlerTask) {
	// 获取信号量，控制并发数量
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return
	}
	if err := c.HandleTask(t); err != nil {
		log.Printf("处理任务失败: %v", err)
	}
}

// 在指定时长上加入随机抖动，避免集群雪崩
func addJitter(duration time.Duration) time.Duration {
	jitter := time.Duration(rand.Int63n(int64(duration / 2)))
	return duration + jitter
}

func main() {
	var (
		token        string
		host         string
		grpcPort     string
		apiPort      string
	)
	flag.StringVar(&token, "token", "", "爬虫校验的Token")
	flag.StringVar(&host, "host", "", "主控的IP地址")
	flag.StringVar(&grpcPort, "grpc-port", "", "主控的gRPC通信端口")
	flag.StringVar(&apiPort, "api-port", "", "主控的API通信端口")
	flag.Parse()
	if token == "" || host == "" || grpcPort == "" || apiPort == "" {
		log.Fatal("请提供所有必需的参数: -token, -host, -grpc-port, -api-port")
	}
	
	log.Printf("启动参数: token=%s, host=%s, grpc-port=%s, api-port=%s", 
		token, host, grpcPort, apiPort)
	
	client, err := NewSpiderClient(token, host, grpcPort, apiPort)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const (
		initialBackoff = 6 * time.Second
		maxBackoff     = 90 * time.Second
	)
	for {
		backoff := initialBackoff
		for {
			task, err := client.GetTask()
			if err == nil {
				go client.handleTaskAsync(ctx, task)
				break
			}
			log.Printf("获取任务失败: %v，%v后重试...", err, backoff)
			time.Sleep(addJitter(backoff))
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
				// 重新创建整个客户端，确保参数正确
				client, err = NewSpiderClient(token, host, grpcPort, apiPort)
				if err != nil {
				    	log.Fatalf("创建客户端失败: %v", err)
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}
