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

// MetadataClient 元数据服务客户端
type MetadataClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewMetadataClient 创建元数据服务客户端
func NewMetadataClient(baseURL string, timeout time.Duration) *MetadataClient {
	return &MetadataClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// SaveMetadata 保存元数据
func (c *MetadataClient) SaveMetadata(ctx context.Context, metadata *models.Metadata) error {
	url := fmt.Sprintf("%s/metadata", c.baseURL)

	body, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
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

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// GetMetadata 获取元数据
func (c *MetadataClient) GetMetadata(ctx context.Context, bucket, key string) (*models.Metadata, error) {
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

// UpdateMetadata 更新元数据
func (c *MetadataClient) UpdateMetadata(ctx context.Context, metadata *models.Metadata) error {
	url := fmt.Sprintf("%s/metadata/%s/%s", c.baseURL, url.PathEscape(metadata.Bucket), url.PathEscape(metadata.Key))

	body, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// DeleteMetadata 删除元数据
func (c *MetadataClient) DeleteMetadata(ctx context.Context, bucket, key string) error {
	url := fmt.Sprintf("%s/metadata/%s/%s", c.baseURL, url.PathEscape(bucket), url.PathEscape(key))

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

// ListMetadata 列出元数据
func (c *MetadataClient) ListMetadata(ctx context.Context, bucket, prefix string, limit, offset int) ([]*models.Metadata, error) {
	u, err := url.Parse(fmt.Sprintf("%s/metadata", c.baseURL))
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
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
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

	var metadataList []*models.Metadata
	if err := json.NewDecoder(resp.Body).Decode(&metadataList); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return metadataList, nil
}

// SearchMetadata 搜索元数据
func (c *MetadataClient) SearchMetadata(ctx context.Context, req *models.SearchObjectsRequest) (*models.SearchObjectsResponse, error) {
	u, err := url.Parse(fmt.Sprintf("%s/metadata/search", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	q.Set("q", req.Query)
	if req.Bucket != "" {
		q.Set("bucket", req.Bucket)
	}
	if req.Limit > 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Offset > 0 {
		q.Set("offset", strconv.Itoa(req.Offset))
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

	var searchResp models.SearchObjectsResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &searchResp, nil
}

// GetStats 获取统计信息
func (c *MetadataClient) GetStats(ctx context.Context) (*models.Stats, error) {
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

	var stats models.Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &stats, nil
}

// CountObjects 计算对象数量
func (c *MetadataClient) CountObjects(ctx context.Context, bucket, prefix string) (int64, error) {
	u, err := url.Parse(fmt.Sprintf("%s/count", c.baseURL))
	if err != nil {
		return 0, fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	if bucket != "" {
		q.Set("bucket", bucket)
	}
	if prefix != "" {
		q.Set("prefix", prefix)
	}
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var countResp struct {
		Count int64 `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&countResp); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	return countResp.Count, nil
}

// HealthCheck 健康检查
func (c *MetadataClient) HealthCheck(ctx context.Context) error {
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
