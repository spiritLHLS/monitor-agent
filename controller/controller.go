package controller

import (
	pb "agent/proto"
	"context"
	"fmt"
	"github.com/imroc/req/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"sync"
	"time"
)

const (
	modeGRPC = "grpc"
	modeAPI  = "api"
)

// 与主控通信的客户端结构体
type ControllerClient struct {
	Token       string
	Host        string
	GrpcPort    string
	ApiPort     string
	GrpcClient  pb.SpiderServiceClient
	Ctx         context.Context
	Cancel      context.CancelFunc
	CurrentMode string
	ModeMutex   sync.RWMutex
	LastError   time.Time
	LastSuccess time.Time
	HttpClient  *req.Client
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

// NewControllerClient 创建主控客户端
func NewControllerClient(token, host, grpcPort, apiPort string) (*ControllerClient, error) {
	client := &ControllerClient{
		Token:       token,
		Host:        host,
		GrpcPort:    grpcPort,
		ApiPort:     apiPort,
		CurrentMode: modeGRPC,
		HttpClient:  req.C().SetTimeout(10 * time.Second),
		LastSuccess: time.Now(),
	}
	// 初始化 gRPC 客户端
	if err := client.InitGRPCClient(); err != nil {
		log.Printf("gRPC 客户端初始化失败: %v, 将使用 API 模式", err)
		client.SetMode(modeAPI)
	}
	return client, nil
}

// InitGRPCClient 初始化 gRPC 客户端
func (c *ControllerClient) InitGRPCClient() error {
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%s", c.Host, c.GrpcPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("无法连接到gRPC服务器: %v", err)
	}
	c.GrpcClient = pb.NewSpiderServiceClient(conn)
	return nil
}

// SetMode 设置当前模式
func (c *ControllerClient) SetMode(mode string) {
	c.ModeMutex.Lock()
	defer c.ModeMutex.Unlock()
	c.CurrentMode = mode
	if mode == modeAPI {
		c.LastSuccess = time.Now()
	}
	c.LastError = time.Now()
}

// GetMode 获取当前模式
func (c *ControllerClient) GetMode() string {
	c.ModeMutex.RLock()
	defer c.ModeMutex.RUnlock()
	return c.CurrentMode
}

// UpdateLastSuccess 更新最后一次成功的时间
func (c *ControllerClient) UpdateLastSuccess() {
	c.ModeMutex.Lock()
	defer c.ModeMutex.Unlock()
	c.LastSuccess = time.Now()
}

// GetTaskGRPC 通过 gRPC 获取任务
func (c *ControllerClient) GetTaskGRPC() (*pb.CrawlerTask, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	request := &pb.TaskRequest{
		Token: c.Token,
	}
	response, err := c.GrpcClient.GetTask(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("gRPC获取任务失败: %v", err)
	}
	return response, nil
}

// GetTaskAPI 通过 API 获取任务
func (c *ControllerClient) GetTaskAPI() (*pb.CrawlerTask, error) {
	url := fmt.Sprintf("http://%s:%s/spiders/getonetask", c.Host, c.ApiPort)
	resp, err := c.HttpClient.R().
		SetBody(map[string]string{"token": c.Token}).
		SetHeader("Content-Type", "application/json").
		Post(url)
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}
	var taskData TaskFromData
	if err := resp.UnmarshalJson(&taskData); err != nil {
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

// HandleTaskGRPC 通过 gRPC 处理任务
func (c *ControllerClient) HandleTaskGRPC(task *pb.CrawlerTask, webData string, success bool, runtime int32, startTime string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result := &pb.CrawlerResult{
		Token:       c.Token,
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
	response, err := c.GrpcClient.HandleTask(ctx, result)
	if err != nil {
		return fmt.Errorf("gRPC处理任务失败: %v", err)
	}
	log.Printf("任务处理结果 - 成功: %v, 消息: %s", response.Success, response.Message)
	return nil
}

// HandleTaskAPI 通过 API 处理任务
func (c *ControllerClient) HandleTaskAPI(task *pb.CrawlerTask, webData string, success bool, runtime int32, startTime string) error {
	result := CrawlerResult{
		Token:       c.Token,
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
	url := fmt.Sprintf("http://%s:%s/spiders/handletask", c.Host, c.ApiPort)
	resp, err := c.HttpClient.R().
		SetBody(result).
		SetHeader("Content-Type", "application/json").
		Post(url)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	if !resp.IsSuccessState() {
		return fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}
	log.Printf("API任务处理结果: %s", resp.String())
	return nil
}
