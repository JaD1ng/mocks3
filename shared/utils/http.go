package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient HTTP客户端配置
type HTTPClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
}

// HTTPClientConfig HTTP客户端配置
type HTTPClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
	Headers    map[string]string
	UserAgent  string
}

// NewHTTPClient 创建HTTP客户端
func NewHTTPClient(config *HTTPClientConfig) *HTTPClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	if config.UserAgent == "" {
		config.UserAgent = "MockS3-Client/1.0"
	}

	// 创建传输层配置
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	headers := make(map[string]string)
	for k, v := range config.Headers {
		headers[k] = v
	}
	headers["User-Agent"] = config.UserAgent

	return &HTTPClient{
		client:  client,
		baseURL: strings.TrimRight(config.BaseURL, "/"),
		headers: headers,
	}
}

// Get 发送GET请求
func (c *HTTPClient) Get(ctx context.Context, path string, params map[string]string) (*http.Response, error) {
	return c.Request(ctx, "GET", path, params, nil, nil)
}

// Post 发送POST请求
func (c *HTTPClient) Post(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.Request(ctx, "POST", path, nil, body, headers)
}

// Put 发送PUT请求
func (c *HTTPClient) Put(ctx context.Context, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	return c.Request(ctx, "PUT", path, nil, body, headers)
}

// Delete 发送DELETE请求
func (c *HTTPClient) Delete(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return c.Request(ctx, "DELETE", path, nil, nil, headers)
}

// Request 发送HTTP请求
func (c *HTTPClient) Request(ctx context.Context, method, path string, params map[string]string, body interface{}, headers map[string]string) (*http.Response, error) {
	// 构建URL
	reqURL := c.baseURL + "/" + strings.TrimLeft(path, "/")

	// 添加查询参数
	if len(params) > 0 {
		u, err := url.Parse(reqURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}

		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		reqURL = u.String()
	}

	// 构建请求体
	var bodyReader io.Reader
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(v)
		case string:
			bodyReader = strings.NewReader(v)
		case io.Reader:
			bodyReader = v
		default:
			jsonData, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置默认头部
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// 设置自定义头部
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 如果是JSON请求体，设置Content-Type
	if body != nil && req.Header.Get("Content-Type") == "" {
		if _, ok := body.([]byte); !ok {
			if _, ok := body.(string); !ok {
				if _, ok := body.(io.Reader); !ok {
					req.Header.Set("Content-Type", "application/json")
				}
			}
		}
	}

	return c.client.Do(req)
}

// GetJSON 获取JSON响应
func (c *HTTPClient) GetJSON(ctx context.Context, path string, params map[string]string, result interface{}) error {
	resp, err := c.Get(ctx, path, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.parseJSONResponse(resp, result)
}

// PostJSON 发送JSON请求并获取JSON响应
func (c *HTTPClient) PostJSON(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.Post(ctx, path, body, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.parseJSONResponse(resp, result)
}

// PutJSON 发送JSON PUT请求并获取JSON响应
func (c *HTTPClient) PutJSON(ctx context.Context, path string, body interface{}, result interface{}) error {
	resp, err := c.Put(ctx, path, body, map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.parseJSONResponse(resp, result)
}

// parseJSONResponse 解析JSON响应
func (c *HTTPClient) parseJSONResponse(resp *http.Response, result interface{}) error {
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	if result == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// DownloadFile 下载文件
func (c *HTTPClient) DownloadFile(ctx context.Context, path string, writer io.Writer) error {
	resp, err := c.Get(ctx, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	_, err = io.Copy(writer, resp.Body)
	return err
}

// UploadFile 上传文件
func (c *HTTPClient) UploadFile(ctx context.Context, path string, reader io.Reader, headers map[string]string) (*http.Response, error) {
	return c.Request(ctx, "PUT", path, nil, reader, headers)
}

// IsRetryableError 检查是否是可重试的错误
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 网络错误通常可以重试
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}

	// 上下文错误不应该重试
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// 字符串匹配一些常见的可重试错误
	errMsg := strings.ToLower(err.Error())
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"no such host",
		"network is unreachable",
		"broken pipe",
	}

	for _, retryableErr := range retryableErrors {
		if strings.Contains(errMsg, retryableErr) {
			return true
		}
	}

	return false
}

// IsRetryableStatusCode 检查HTTP状态码是否可重试
func IsRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// GetClientIP 获取客户端IP地址
func GetClientIP(r *http.Request) string {
	// 检查X-Forwarded-For头
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// 检查X-Real-IP头
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// 使用RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// SetJSONResponse 设置JSON响应
func SetJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// SetErrorResponse 设置错误响应
func SetErrorResponse(w http.ResponseWriter, statusCode int, message string) error {
	return SetJSONResponse(w, statusCode, map[string]interface{}{
		"error":   message,
		"code":    statusCode,
		"success": false,
	})
}

// ParseJSONBody 解析JSON请求体
func ParseJSONBody(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
