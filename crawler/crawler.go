package crawler

import (
	"fmt"
	"github.com/imroc/req/v3"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Crawler 页面爬取客户端
type Crawler struct {
	httpClient   *req.Client
	cacheMutex   sync.RWMutex
	cacheExpiry  time.Duration
	userAgent    string
}

// NewCrawler 创建新的爬虫客户端
func NewCrawler() *Crawler {
	crawler := &Crawler{
		cacheExpiry:  2 * time.Hour,
		httpClient:   req.C(),
	}
	crawler.httpClient.SetTimeout(10 * time.Second)
	crawler.httpClient.ImpersonateChrome()
	crawler.userAgent = crawler.httpClient.Headers.Get("User-Agent")
	return crawler
}

// getCacheKey 获取缓存键
func (c *Crawler) getCacheKey(domain, userAgent string) string {
	return fmt.Sprintf("%s:%s", domain, userAgent)
}

// getDomain 从URL中获取域名
func (c *Crawler) getDomain(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return parsedURL.Host
}

// isCloudFlareChallenge 检查是否遇到CloudFlare验证
func (c *Crawler) isCloudFlareChallenge(resp *req.Response) bool {
	// 添加nil检查
	if resp == nil {
		return false
	}
	if resp.StatusCode != 403 && resp.StatusCode != 503 {
		return false
	}
	if resp.Header == nil {
		return false
	}
	for key := range resp.Header {
		if strings.Contains(strings.ToLower(key), "cf-") {
			return true
		}
	}
	body := resp.String()
	return strings.Contains(body, "Verifying you are human") ||
		strings.Contains(body, "是真人") ||
		strings.Contains(body, "Are you a human") ||
		strings.Contains(body, "Wait a moment")
}

// FetchWebData 获取网页数据
func (c *Crawler) FetchWebData(url string) (string, bool) {
	client := c.httpClient.Clone()
	startTime := time.Now()
	// 第一次请求
	resp, err := client.R().Get(url)
	// 先检查错误，再检查响应
	if err != nil {
		log.Printf("获取页面失败: %v, URL: %s", err, url)
		return "", false
	}
	// 检查是否需要处理cf5s验证
	if c.isCloudFlareChallenge(resp) {
		log.Printf("检测到 CloudFlare 验证, URL: %s", url)
		return "", false
	}
	if !resp.IsSuccessState() {
		log.Printf("请求失败，状态码: %d, URL: %s", resp.StatusCode, url)
		return "", false
	}
	log.Printf("获取页面成功 - URL: %s, 耗时: %v", url, time.Since(startTime))
	return resp.String(), true
}
