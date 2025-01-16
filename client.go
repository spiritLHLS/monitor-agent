package main

import (
	"agent/controller"
	"agent/crawler"
	pb "agent/proto"
	"flag"
	"fmt"
	"log"
	"time"
)

const (
	modeGRPC = "grpc"
	modeAPI  = "api"
)

type SpiderClient struct {
	controller *controller.ControllerClient
	crawler    *crawler.Crawler
}

// NewSpiderClient 创建新的客户端实例
func NewSpiderClient(token, host, grpcPort, apiPort, cfServiceURL string, useCF bool) (*SpiderClient, error) {
	controller, err := controller.NewControllerClient(token, host, grpcPort, apiPort)
	if err != nil {
		return nil, err
	}
	crawler := crawler.NewCrawler(cfServiceURL, useCF) // 修改函数调用
	return &SpiderClient{
		controller: controller,
		crawler:    crawler,
	}, nil
}

// GetTask 获取任务
func (c *SpiderClient) GetTask() (*pb.CrawlerTask, error) {
	mode := c.controller.GetMode()
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
		return fmt.Errorf("无效的Token")
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
	mode := c.controller.GetMode()
	var err error
	if mode == modeGRPC {
		err = c.controller.HandleTaskGRPC(task, webData, success, runtime, formattedTime)
	} else {
		err = c.controller.HandleTaskAPI(task, webData, success, runtime, formattedTime)
		if err == nil {
			c.controller.UpdateLastSuccess()
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
	if c.controller.GetMode() == modeAPI {
		c.controller.ModeMutex.RLock()
		stableTime := time.Since(c.controller.LastSuccess)
		c.controller.ModeMutex.RUnlock()

		if stableTime >= 5*time.Minute {
			if err := c.controller.InitGRPCClient(); err == nil {
				c.controller.SetMode(modeGRPC)
				log.Printf("API模式稳定运行5分钟，成功切换回gRPC模式")
			}
		}
	}
}

// switchMode 切换模式
func (c *SpiderClient) switchMode() {
	currentMode := c.controller.GetMode()
	if currentMode == modeGRPC {
		c.controller.SetMode(modeAPI)
	} else {
		if err := c.controller.InitGRPCClient(); err != nil {
			log.Printf("gRPC 客户端重新初始化失败: %v", err)
			return
		}
		c.controller.SetMode(modeGRPC)
	}
	log.Printf("切换到 %s 模式", c.controller.GetMode())
}

func main() {
	var (
		token        string
		host         string
		grpcPort     string
		apiPort      string
		cfServiceURL string
		useCF        bool // 新增参数
	)
	flag.StringVar(&token, "token", "", "爬虫校验的Token")
	flag.StringVar(&host, "host", "", "主控的IP地址")
	flag.StringVar(&grpcPort, "grpc-port", "", "主控的gRPC通信端口")
	flag.StringVar(&apiPort, "api-port", "", "主控的API通信端口")
	flag.StringVar(&cfServiceURL, "cf-service", "http://127.0.0.1:8000", "CloudFlare 绕过服务地址")
	flag.BoolVar(&useCF, "use-cf", false, "是否使用CloudFlare绕过服务") // 新增参数定义
	flag.Parse()
	if token == "" || host == "" || grpcPort == "" || apiPort == "" {
		log.Fatal("请提供所有必需的参数: -token, -host, -grpc-port, -api-port")
	}
	client, err := NewSpiderClient(token, host, grpcPort, apiPort, cfServiceURL, useCF) // 修改函数调用
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	for {
		task, err := client.GetTask()
		if err != nil {
			log.Printf("获取任务失败: %v", err)
			time.Sleep(6 * time.Second)
			continue
		}
		go func(t *pb.CrawlerTask) {
			if err := client.HandleTask(t); err != nil {
				log.Printf("处理任务失败: %v", err)
			}
		}(task)
		time.Sleep(1 * time.Second)
	}
}
