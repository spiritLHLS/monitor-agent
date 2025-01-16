package main

import (
	"fmt"
	"github.com/imroc/req/v3"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type CFCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CFCookieResponse struct {
	Status  string     `json:"status"`
	Cookies []CFCookie `json:"cookies"`
}

type CFCacheEntry struct {
	Cookies   []CFCookie
	UserAgent string
	CreatedAt time.Time
}

// Crawler 页面爬取客户端
type Crawler struct {
	httpClient   *req.Client
	cfServiceURL string
	cookieCache  map[string]*CFCacheEntry
	cacheMutex   sync.RWMutex
	cacheExpiry  time.Duration
	userAgent    string
	useCF        bool
}

// NewCrawler 创建新的爬虫客户端
func NewCrawler(cfServiceURL string, useCF bool) *Crawler {
	crawler := &Crawler{
		cfServiceURL: cfServiceURL,
		cookieCache:  make(map[string]*CFCacheEntry),
		cacheExpiry:  2 * time.Hour,
		httpClient:   req.C(),
		useCF:        useCF,
	}
	crawler.httpClient.SetCommonHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	crawler.httpClient.SetTimeout(10 * time.Second)
	crawler.httpClient.ImpersonateChrome()
	crawler.userAgent = crawler.httpClient.Headers.Get("User-Agent")
	if useCF {
		go crawler.startCacheCleaner()
	}
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

// cleanExpiredCache 清理过期缓存
func (c *Crawler) cleanExpiredCache() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	now := time.Now()
	for key, entry := range c.cookieCache {
		if now.Sub(entry.CreatedAt) >= c.cacheExpiry {
			delete(c.cookieCache, key)
		}
	}
}

// startCacheCleaner 启动缓存清理器
func (c *Crawler) startCacheCleaner() {
	ticker := time.NewTicker(30 * time.Minute)
	for range ticker.C {
		c.cleanExpiredCache()
	}
}

// getCFCookies 获取CloudFlare cookies
func (c *Crawler) getCFCookies(urlStr string) ([]CFCookie, error) {
	domain := c.getDomain(urlStr)
	cacheKey := c.getCacheKey(domain, c.userAgent)
	// 检查缓存
	c.cacheMutex.RLock()
	if entry, exists := c.cookieCache[cacheKey]; exists {
		if time.Since(entry.CreatedAt) < c.cacheExpiry {
			c.cacheMutex.RUnlock()
			return entry.Cookies, nil
		}
	}
	c.cacheMutex.RUnlock()
	// 请求新的 cookies
	var response CFCookieResponse
	resp, err := c.httpClient.R().
		SetBody(map[string]string{
			"url":        urlStr,
			"user_agent": c.userAgent,
		}).
		SetSuccessResult(&response).
		Post(c.cfServiceURL + "/get_cf_cookies")
	if err != nil {
		return nil, fmt.Errorf("request CF service failed: %v", err)
	}
	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("CF service returned error: %s", resp.String())
	}
	// 更新缓存
	c.cacheMutex.Lock()
	c.cookieCache[cacheKey] = &CFCacheEntry{
		Cookies:   response.Cookies,
		UserAgent: c.userAgent,
		CreatedAt: time.Now(),
	}
	c.cacheMutex.Unlock()
	return response.Cookies, nil
}

// isCloudFlareChallenge 检查是否遇到CloudFlare验证
func (c *Crawler) isCloudFlareChallenge(resp *req.Response) bool {
	if resp == nil {
		return false
	}
	if resp.StatusCode != 403 && resp.StatusCode != 503 {
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
	// 首先尝试使用缓存的 cookies
	if c.useCF {
		cookies, err := c.getCFCookies(url)
		if err == nil && cookies != nil {
			for _, cookie := range cookies {
				client.SetCommonCookies(&http.Cookie{
					Name:  cookie.Name,
					Value: cookie.Value,
				})
			}
		}
	}
	// 第一次请求
	resp, err := client.R().Get(url)
	// 检查是否需要处理 CloudFlare 验证
	if c.useCF && (err != nil || c.isCloudFlareChallenge(resp)) {
		log.Printf("检测到 CloudFlare 验证或请求失败，正在获取新的 cookies...")
		// 强制获取新的 CF cookies
		c.cacheMutex.Lock()
		delete(c.cookieCache, c.getCacheKey(c.getDomain(url), c.userAgent))
		c.cacheMutex.Unlock()
		cookies, err := c.getCFCookies(url)
		if err != nil {
			log.Printf("获取 CF cookies 失败: %v", err)
			return "", false
		}
		// 使用新的 cookies 重试请求
		client = c.httpClient.Clone()
		for _, cookie := range cookies {
			client.SetCommonCookies(&http.Cookie{
				Name:  cookie.Name,
				Value: cookie.Value,
			})
		}
		resp, err = client.R().Get(url)
	}
	if err != nil {
		log.Printf("获取页面失败: %v, URL: %s", err, url)
		return "", false
	}
	if !resp.IsSuccessState() {
		log.Printf("请求失败，状态码: %d, URL: %s", resp.StatusCode, url)
		return "", false
	}
	log.Printf("获取页面成功 - URL: %s, 耗时: %v", url, time.Since(startTime))
	return resp.String(), true
}
