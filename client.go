package main

import (
	"agent/controller"
	"agent/crawler"
	pb "agent/proto"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
)

const (
	modeGRPC           = "grpc"
	modeAPI            = "api"
	maxConcurrentTasks = 10 // 限制并发任务数量
)

type SpiderClient struct {
	controller *controller.ControllerClient
	crawler    *crawler.Crawler
	semaphore  chan struct{} // 用于控制并发数量
	modeMutex  sync.RWMutex  // 保护模式切换的互斥锁
	token      string
	host       string
	grpcPort   string
	apiPort    string
	taskFlag   string
}

// NewSpiderClient 创建新的客户端实例
func NewSpiderClient(token, host, grpcPort, apiPort string) (*SpiderClient, error) {
	controllerClient, err := controller.NewControllerClient(token, host, grpcPort, apiPort)
	if err != nil {
		return nil, err
	}
	newCrawler := crawler.NewCrawler()
	return &SpiderClient{
		controller: controllerClient,
		crawler:    newCrawler,
		semaphore:  make(chan struct{}, maxConcurrentTasks),
		token:      token,
		host:       host,
		grpcPort:   grpcPort,
		apiPort:    apiPort,
		taskFlag:   "",
	}, nil
}

// NewSpiderClientWithFlag 创建带任务类型的客户端实例
func NewSpiderClientWithFlag(token, host, grpcPort, apiPort, taskFlag string) (*SpiderClient, error) {
	controllerClientWithFlag, err := controller.NewControllerClientWithFlag(token, host, grpcPort, apiPort, taskFlag)
	if err != nil {
		return nil, err
	}
	newCrawler := crawler.NewCrawler()
	return &SpiderClient{
		controller: controllerClientWithFlag,
		crawler:    newCrawler,
		semaphore:  make(chan struct{}, maxConcurrentTasks),
		token:      token,
		host:       host,
		grpcPort:   grpcPort,
		apiPort:    apiPort,
		taskFlag:   taskFlag,
	}, nil
}

// SetTaskFlag 设置任务类型
func (c *SpiderClient) SetTaskFlag(flag string) {
	c.taskFlag = flag
	c.controller.SetTaskFlag(flag)
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
		// 只有在连接错误时才切换模式，队列为空不切换
		if !isQueueEmptyError(err) {
			c.switchMode()
		}
		return nil, err
	}
	return task, nil
}

// isQueueEmptyError 判断是否为队列为空的错误
func isQueueEmptyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "队列为空") ||
		strings.Contains(errStr, "queue is empty") ||
		strings.Contains(errStr, "默认任务队列为空") ||
		strings.Contains(errStr, "cf5s 任务队列为空") ||
		strings.Contains(errStr, "dynamic 任务队列为空")
}

// HandleTask 处理任务
func (c *SpiderClient) HandleTask(task *pb.CrawlerTask) error {
	if task == nil {
		return fmt.Errorf("任务为空")
	}
	if task.Token == "" {
		return fmt.Errorf("任务Token为空，可能是服务端问题")
	}
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
		// 只有在连接错误时才切换模式，业务错误不切换
		if !isBusinessError(err) {
			c.switchMode()
		}
		return err
	}
	return nil
}

// isBusinessError 判断是否为业务错误（不需要切换模式的错误）
func isBusinessError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "无效的Token") ||
		strings.Contains(errStr, "无效的URL") ||
		strings.Contains(errStr, "任务为空") ||
		strings.Contains(errStr, "任务Token为空")
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
			if newController, err := controller.NewControllerClientWithFlag(c.token, c.host, c.grpcPort, c.apiPort, c.taskFlag); err == nil {
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
		if newController, err := controller.NewControllerClientWithFlag(c.token, c.host, c.grpcPort, c.apiPort, c.taskFlag); err == nil {
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
		token    string
		host     string
		grpcPort string
		apiPort  string
		taskFlag string
	)
	flag.StringVar(&token, "token", "", "爬虫校验的Token")
	flag.StringVar(&host, "host", "", "主控的IP地址")
	flag.StringVar(&grpcPort, "grpc-port", "", "主控的gRPC通信端口")
	flag.StringVar(&apiPort, "api-port", "", "主控的API通信端口")
	flag.StringVar(&taskFlag, "task-flag", "", "任务类型标识 (可选: cf5s, dynamic, 默认为空)")
	flag.Parse()
	if token == "" || host == "" || grpcPort == "" || apiPort == "" {
		log.Fatal("请提供所有必需的参数: -token, -host, -grpc-port, -api-port")
	}
	log.Printf("启动参数: token=%s, host=%s, grpc-port=%s, api-port=%s, task-flag=%s",
		maskToken(token), host, grpcPort, apiPort, taskFlag)
	var client *SpiderClient
	var err error
	if taskFlag != "" {
		client, err = NewSpiderClientWithFlag(token, host, grpcPort, apiPort, taskFlag)
	} else {
		client, err = NewSpiderClient(token, host, grpcPort, apiPort)
	}
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
				// 添加任务验证日志
				log.Printf("获取到任务: URL=%s, Token=%s, Tag=%s, BillingType=%s, ReqMethod=%s",
					task.Url, maskToken(task.Token), task.Tag, task.BillingType, task.ReqMethod)
				go client.handleTaskAsync(ctx, task)
				break
			}
			// 如果是队列为空，减少日志频率
			if isQueueEmptyError(err) {
				if backoff == initialBackoff {
					if taskFlag != "" {
						log.Printf("%s 任务队列为空，等待新任务...", taskFlag)
					} else {
						log.Printf("任务队列为空，等待新任务...")
					}
				}
				time.Sleep(addJitter(initialBackoff))
				break
			}
			log.Printf("获取任务失败: %v，%v后重试...", err, backoff)
			time.Sleep(addJitter(backoff))
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
				if taskFlag != "" {
					client, err = NewSpiderClientWithFlag(token, host, grpcPort, apiPort, taskFlag)
				} else {
					client, err = NewSpiderClient(token, host, grpcPort, apiPort)
				}
				if err != nil {
					log.Fatalf("创建客户端失败: %v", err)
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// maskToken 遮蔽token用于日志输出
func maskToken(token string) string {
	if len(token) <= 4 {
		return "****"
	}
	return token[:2] + "****" + token[len(token)-2:]
}
