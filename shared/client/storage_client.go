package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mocks3/shared/models"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// StorageClient 存储服务客户端
type StorageClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewStorageClient 创建存储服务客户端
func NewStorageClient(baseURL string, timeout time.Duration) *StorageClient {
	return &StorageClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// WriteObject 写入对象
func (c *StorageClient) WriteObject(ctx context.Context, object *models.Object) error {
	url := fmt.Sprintf("%s/objects", c.baseURL)
	
	req := &models.UploadRequest{
		Key:         object.Key,
		Bucket:      object.Bucket,
		ContentType: object.ContentType,
		Headers:     object.Headers,
		Tags:        object.Tags,
		Data:        object.Data,
	}
	
	body, err := json.Marshal(req)
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
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var uploadResp models.UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	
	if !uploadResp.Success {
		return fmt.Errorf("upload failed: %s", uploadResp.Message)
	}
	
	// 更新对象信息
	object.ID = uploadResp.ObjectID
	object.MD5Hash = uploadResp.MD5Hash
	object.ETag = uploadResp.ETag
	
	return nil
}

// ReadObject 读取对象
func (c *StorageClient) ReadObject(ctx context.Context, bucket, key string) (*models.Object, error) {
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
	
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	
	object := &models.Object{
		Key:         key,
		Bucket:      bucket,
		Data:        data,
		Size:        int64(len(data)),
		ContentType: resp.Header.Get("Content-Type"),
		ETag:        resp.Header.Get("ETag"),
		MD5Hash:     resp.Header.Get("Content-MD5"),
		Headers:     make(map[string]string),
	}
	
	// 复制所有响应头
	for k, v := range resp.Header {
		if len(v) > 0 {
			object.Headers[k] = v[0]
		}
	}
	
	return object, nil
}

// DeleteObject 删除对象
func (c *StorageClient) DeleteObject(ctx context.Context, bucket, key string) error {
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

// ListObjects 列出对象
func (c *StorageClient) ListObjects(ctx context.Context, req *models.ListObjectsRequest) (*models.ListObjectsResponse, error) {
	u, err := url.Parse(fmt.Sprintf("%s/objects", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	
	q := u.Query()
	if req.Bucket != "" {
		q.Set("bucket", req.Bucket)
	}
	if req.Prefix != "" {
		q.Set("prefix", req.Prefix)
	}
	if req.Delimiter != "" {
		q.Set("delimiter", req.Delimiter)
	}
	if req.MaxKeys > 0 {
		q.Set("max_keys", strconv.Itoa(req.MaxKeys))
	}
	if req.StartAfter != "" {
		q.Set("start_after", req.StartAfter)
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
	
	var listResp models.ListObjectsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	
	return &listResp, nil
}

// HealthCheck 健康检查
func (c *StorageClient) HealthCheck(ctx context.Context) error {
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