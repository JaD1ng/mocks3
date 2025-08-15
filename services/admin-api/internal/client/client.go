package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ServiceClient 通用服务客户端
type ServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// Clients 所有客户端
type Clients struct {
	Metadata *ServiceClient
	Storage  *ServiceClient
	Task     *ServiceClient
	Chaos    *ServiceClient
}

// NewServiceClient 创建服务客户端
func NewServiceClient(baseURL string) *ServiceClient {
	return &ServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Get 发送GET请求
// ctx: 请求上下文，用于控制请求超时和取消
// path: API路径，会与baseURL拼接形成完整URL
// result: 用于接收JSON响应的结构体指针，如果为nil则忽略响应体
// 返回: 请求错误或JSON解析错误
func (c *ServiceClient) Get(ctx context.Context, path string, result any) error {
	// 构建完整的请求URL
	url := c.baseURL + path

	// 创建GET请求，支持上下文取消
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	// 发送HTTP请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，非2xx状态码视为错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 如果提供了result参数，则解析JSON响应
	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Post 发送POST请求
// ctx: 请求上下文，用于控制请求超时和取消
// path: API路径，会与baseURL拼接形成完整URL
// body: 请求体数据，会被序列化为JSON
// result: 用于接收JSON响应的结构体指针，如果为nil则忽略响应体
// 返回: 请求错误、JSON序列化/解析错误
func (c *ServiceClient) Post(ctx context.Context, path string, body, result any) error {
	// 构建完整的请求URL
	url := c.baseURL + path

	// 处理请求体：如果提供了body，将其序列化为JSON
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// 创建POST请求，支持上下文取消
	req, err := http.NewRequestWithContext(ctx, "POST", url, reqBody)
	if err != nil {
		return err
	}
	// 设置Content-Type为JSON
	req.Header.Set("Content-Type", "application/json")

	// 发送HTTP请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，非2xx状态码视为错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 如果提供了result参数，则解析JSON响应
	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Put 发送PUT请求
// ctx: 请求上下文，用于控制请求超时和取消
// path: API路径，会与baseURL拼接形成完整URL
// body: 请求体数据，会被序列化为JSON
// result: 用于接收JSON响应的结构体指针，如果为nil则忽略响应体
// 返回: 请求错误、JSON序列化/解析错误
func (c *ServiceClient) Put(ctx context.Context, path string, body, result any) error {
	// 构建完整的请求URL
	url := c.baseURL + path

	// 处理请求体：如果提供了body，将其序列化为JSON
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// 创建PUT请求，支持上下文取消
	req, err := http.NewRequestWithContext(ctx, "PUT", url, reqBody)
	if err != nil {
		return err
	}
	// 设置Content-Type为JSON
	req.Header.Set("Content-Type", "application/json")

	// 发送HTTP请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，非2xx状态码视为错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 如果提供了result参数，则解析JSON响应
	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Delete 发送DELETE请求
// ctx: 请求上下文，用于控制请求超时和取消
// path: API路径，会与baseURL拼接形成完整URL
// 返回: 请求错误
func (c *ServiceClient) Delete(ctx context.Context, path string) error {
	// 构建完整的请求URL
	url := c.baseURL + path

	// 创建DELETE请求，支持上下文取消
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	// 发送HTTP请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，非2xx状态码视为错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetHealth 获取服务健康状态
// ctx: 请求上下文，用于控制请求超时和取消
// 返回: 健康状态数据和错误信息
// 健康状态数据通常包含服务状态、版本信息、依赖检查等
func (c *ServiceClient) GetHealth(ctx context.Context) (map[string]any, error) {
	var health map[string]any
	// 调用统一的GET方法获取健康检查数据
	err := c.Get(ctx, "/health", &health)
	return health, err
}
