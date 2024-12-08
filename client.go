package main

import (
	pb "agent/proto"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/parnurzeal/gorequest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	modeGRPC = "grpc"
	modeAPI  = "api"
)

// SpiderClient 统一的客户端结构体
type SpiderClient struct {
	token       string
	host        string
	grpcPort    string
	apiPort     string
	grpcClient  pb.SpiderServiceClient
	ctx         context.Context
	cancel      context.CancelFunc
	currentMode string
	modeMutex   sync.RWMutex
	lastError   time.Time
}

// TaskFromData API 模式的任务响应结构
type TaskFromData struct {
	Data CrawlerTask `json:"data"`
}

// CrawlerTask 任务结构
type CrawlerTask struct {
	Token       string `json:"token"`
	Tag         string `json:"tag"`
	URL         string `json:"url"`
	BillingType string `json:"billing_type"`
	CrawlNum    int    `json:"crawl_num"`
	ExtraHeader string `json:"extra_header"`
	ReqMethod   string `json:"req_method"`
}

// CrawlerResult 结果结构
type CrawlerResult struct {
	Token       string `json:"token"`
	Tag         string `json:"tag"`
	URL         string `json:"url"`
	BillingType string `json:"billing_type"`
	CrawlNum    int    `json:"crawl_num"`
	Runtime     int    `json:"runtime"`
	StartTime   string `json:"start_time"`
	Success     bool   `json:"success"`
	ReqMethod   string `json:"req_method"`
	WebData     string `json:"webdata,omitempty"`
}

// NewSpiderClient 创建新的客户端实例
func NewSpiderClient(token, host, grpcPort, apiPort string) (*SpiderClient, error) {
	client := &SpiderClient{
		token:       token,
		host:        host,
		grpcPort:    grpcPort,
		apiPort:     apiPort,
		currentMode: modeGRPC,
	}
	// 初始化 gRPC 客户端
	if err := client.initGRPCClient(); err != nil {
		log.Printf("gRPC 客户端初始化失败: %v, 将使用 API 模式", err)
		client.setMode(modeAPI)
	}
	return client, nil
}

// initGRPCClient 初始化 gRPC 客户端
func (c *SpiderClient) initGRPCClient() error {
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%s", c.host, c.grpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("无法连接到gRPC服务器: %v", err)
	}
	c.grpcClient = pb.NewSpiderServiceClient(conn)
	c.ctx, c.cancel = context.WithTimeout(context.Background(), 30*time.Second)
	return nil
}

// setMode 设置当前模式
func (c *SpiderClient) setMode(mode string) {
	c.modeMutex.Lock()
	defer c.modeMutex.Unlock()
	c.currentMode = mode
	c.lastError = time.Now()
}

// getMode 获取当前模式
func (c *SpiderClient) getMode() string {
	c.modeMutex.RLock()
	defer c.modeMutex.RUnlock()
	return c.currentMode
}

// switchMode 切换模式
func (c *SpiderClient) switchMode() {
	currentMode := c.getMode()
	if currentMode == modeGRPC {
		c.setMode(modeAPI)
	} else {
		// 重新初始化 gRPC 客户端
		if err := c.initGRPCClient(); err != nil {
			log.Printf("gRPC 客户端重新初始化失败: %v", err)
			return
		}
		c.setMode(modeGRPC)
	}
	log.Printf("切换到 %s 模式", c.getMode())
}

// GetTask 获取任务
func (c *SpiderClient) GetTask() (*pb.CrawlerTask, error) {
	mode := c.getMode()
	var task *pb.CrawlerTask
	var err error
	if mode == modeGRPC {
		task, err = c.getTaskGRPC()
	} else {
		task, err = c.getTaskAPI()
	}
	if err != nil {
		log.Printf("%s 模式获取任务失败: %v", mode, err)
		c.switchMode()
		return nil, err
	}
	return task, nil
}

// getTaskGRPC 通过 gRPC 获取任务
func (c *SpiderClient) getTaskGRPC() (*pb.CrawlerTask, error) {
	request := &pb.TaskRequest{
		Token: c.token,
	}
	response, err := c.grpcClient.GetTask(c.ctx, request)
	if err != nil {
		return nil, fmt.Errorf("gRPC获取任务失败: %v", err)
	}
	return response, nil
}

// getTaskAPI 通过 API 获取任务
func (c *SpiderClient) getTaskAPI() (*pb.CrawlerTask, error) {
	url := fmt.Sprintf("http://%s:%s/spiders/getonetask", c.host, c.apiPort)
	data := map[string]string{"token": c.token}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var taskData TaskFromData
	if err := json.Unmarshal(body, &taskData); err != nil {
		return nil, err
	}
	return &pb.CrawlerTask{
		Token:       taskData.Data.Token,
		Tag:         taskData.Data.Tag,
		Url:         taskData.Data.URL,
		BillingType: taskData.Data.BillingType,
		CrawlNum:    int32(taskData.Data.CrawlNum),
		ExtraHeader: taskData.Data.ExtraHeader,
		ReqMethod:   taskData.Data.ReqMethod,
	}, nil
}

// fetchWebData 获取网页数据
func (c *SpiderClient) fetchWebData(url string) (string, bool) {
	startTime := time.Now()
	request := gorequest.New()
	req := request.Get(url).Retry(3, 6*time.Second, http.StatusBadRequest, http.StatusInternalServerError)
	req.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.75 Safari/537.36")
	req.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	_, body, errs := req.End()
	if len(errs) > 0 {
		log.Printf("获取页面数据失败: %v, URL: %s", errs, url)
		return "", false
	}

	log.Printf("获取页面成功 - URL: %s, 耗时: %v", url, time.Since(startTime))
	return body, true
}

// HandleTask 处理任务
func (c *SpiderClient) HandleTask(task *pb.CrawlerTask) error {
	if task.Token != c.token {
		return fmt.Errorf("无效的Token")
	}
	if task.Url == "" || task.Tag == "" {
		return fmt.Errorf("无效的URL或Tag")
	}
	startTime := time.Now()
	webData, success := c.fetchWebData(task.Url)
	runtime := int32(time.Since(startTime).Seconds())
	loc, _ := time.LoadLocation("Asia/Shanghai")
	beijingTime := time.Now().In(loc)
	formattedTime := beijingTime.Format("2006-01-02 15:04:05")
	mode := c.getMode()
	var err error
	if mode == modeGRPC {
		err = c.handleTaskGRPC(task, webData, success, runtime, formattedTime)
	} else {
		err = c.handleTaskAPI(task, webData, success, runtime, formattedTime)
	}
	if err != nil {
		log.Printf("%s 模式处理任务失败: %v", mode, err)
		c.switchMode()
		return err
	}
	return nil
}

// handleTaskGRPC 通过 gRPC 处理任务
func (c *SpiderClient) handleTaskGRPC(task *pb.CrawlerTask, webData string, success bool, runtime int32, startTime string) error {
	result := &pb.CrawlerResult{
		Token:       c.token,
		Tag:         task.Tag,
		Url:         task.Url,
		BillingType: task.BillingType,
		CrawlNum:    task.CrawlNum,
		Runtime:     runtime,
		StartTime:   startTime,
		Success:     success,
		ReqMethod:   task.ReqMethod,
		WebData:     webData,
	}
	response, err := c.grpcClient.HandleTask(c.ctx, result)
	if err != nil {
		return fmt.Errorf("gRPC处理任务失败: %v", err)
	}
	log.Printf("任务处理结果 - 成功: %v, 消息: %s", response.Success, response.Message)
	return nil
}

// handleTaskAPI 通过 API 处理任务
func (c *SpiderClient) handleTaskAPI(task *pb.CrawlerTask, webData string, success bool, runtime int32, startTime string) error {
	result := CrawlerResult{
		Token:       c.token,
		Tag:         task.Tag,
		URL:         task.Url,
		BillingType: task.BillingType,
		CrawlNum:    int(task.CrawlNum),
		Runtime:     int(runtime),
		StartTime:   startTime,
		Success:     success,
		ReqMethod:   task.ReqMethod,
		WebData:     webData,
	}
	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("JSON编码失败: %v", err)
	}
	url := fmt.Sprintf("http://%s:%s/spiders/handletask", c.host, c.apiPort)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}
	log.Printf("API任务处理结果: %s", string(body))
	return nil
}

func main() {
	var (
		token    string
		host     string
		grpcPort string
		apiPort  string
	)
	flag.StringVar(&token, "token", "", "爬虫校验的Token")
	flag.StringVar(&host, "host", "", "主控的IP地址")
	flag.StringVar(&grpcPort, "grpc-port", "", "主控的gRPC通信端口")
	flag.StringVar(&apiPort, "api-port", "", "主控的API通信端口")
	flag.Parse()
	if token == "" || host == "" || grpcPort == "" || apiPort == "" {
		log.Fatal("请提供所有必需的参数: -token, -host, -grpc-port, -api-port")
	}
	client, err := NewSpiderClient(token, host, grpcPort, apiPort)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	if client.cancel != nil {
		defer client.cancel()
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
