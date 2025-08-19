package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mocks3/shared/models"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ThirdPartyClient 第三方服务客户端
type ThirdPartyClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewThirdPartyClient 创建第三方服务客户端
func NewThirdPartyClient(baseURL string, timeout time.Duration) *ThirdPartyClient {
	return &ThirdPartyClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// GetObject 获取对象
func (c *ThirdPartyClient) GetObject(ctx context.Context, bucket, key string) (*models.Object, error) {
	url := fmt.Sprintf("%s/objects/%s/%s", c.baseURL, url.PathEscape(bucket), url.PathEscape(key))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("object not found: %s/%s", bucket, key)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var object models.Object
	if err := json.NewDecoder(resp.Body).Decode(&object); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &object, nil
}

// PutObject 存储对象
func (c *ThirdPartyClient) PutObject(ctx context.Context, object *models.Object) error {
	url := fmt.Sprintf("%s/objects", c.baseURL)

	body, err := json.Marshal(object)
	if err != nil {
		return fmt.Errorf("marshal object: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// DeleteObject 删除对象
func (c *ThirdPartyClient) DeleteObject(ctx context.Context, bucket, key string) error {
	url := fmt.Sprintf("%s/objects/%s/%s", c.baseURL, url.PathEscape(bucket), url.PathEscape(key))

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetObjectMetadata 获取对象元数据
func (c *ThirdPartyClient) GetObjectMetadata(ctx context.Context, bucket, key string) (*models.Metadata, error) {
	url := fmt.Sprintf("%s/metadata/%s/%s", c.baseURL, url.PathEscape(bucket), url.PathEscape(key))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("metadata not found: %s/%s", bucket, key)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var metadata models.Metadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &metadata, nil
}

// ListObjects 列出对象
func (c *ThirdPartyClient) ListObjects(ctx context.Context, bucket, prefix string, limit int) ([]*models.Metadata, error) {
	u, err := url.Parse(fmt.Sprintf("%s/objects", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	if bucket != "" {
		q.Set("bucket", bucket)
	}
	if prefix != "" {
		q.Set("prefix", prefix)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var listResp struct {
		Objects []*models.Metadata `json:"objects"`
		Count   int                `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return listResp.Objects, nil
}

// SetDataSource 设置数据源
func (c *ThirdPartyClient) SetDataSource(ctx context.Context, name, config string) error {
	url := fmt.Sprintf("%s/datasources", c.baseURL)

	reqBody := map[string]interface{}{
		"name":   name,
		"config": config,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetDataSources 获取数据源列表
func (c *ThirdPartyClient) GetDataSources(ctx context.Context) ([]models.DataSource, error) {
	url := fmt.Sprintf("%s/datasources", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var sourcesResp struct {
		DataSources []models.DataSource `json:"datasources"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sourcesResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return sourcesResp.DataSources, nil
}

// CacheObject 缓存对象
func (c *ThirdPartyClient) CacheObject(ctx context.Context, object *models.Object) error {
	url := fmt.Sprintf("%s/cache", c.baseURL)

	body, err := json.Marshal(object)
	if err != nil {
		return fmt.Errorf("marshal object: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// InvalidateCache 清除缓存
func (c *ThirdPartyClient) InvalidateCache(ctx context.Context, bucket, key string) error {
	url := fmt.Sprintf("%s/cache/%s/%s", c.baseURL, url.PathEscape(bucket), url.PathEscape(key))

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetStats 获取统计信息
func (c *ThirdPartyClient) GetStats(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/stats", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return stats, nil
}

// HealthCheck 健康检查
func (c *ThirdPartyClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)
	}

	return nil
}
