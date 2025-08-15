package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"mocks3/shared/types"
	"time"
)

// HTTPClient 通用HTTP客户端
type HTTPClient struct {
	client  *http.Client
	baseURL string
}

// NewHTTPClient 创建HTTP客户端
func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// Get 发送GET请求
func (c *HTTPClient) Get(ctx context.Context, path string, result any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// Post 发送POST请求
func (c *HTTPClient) Post(ctx context.Context, path string, body, result any) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Put 发送PUT请求
func (c *HTTPClient) Put(ctx context.Context, path string, body, result any) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", c.baseURL+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// Delete 发送DELETE请求
func (c *HTTPClient) Delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+path, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// MetadataClient 元数据服务客户端
type MetadataClient struct {
	*HTTPClient
}

// NewMetadataClient 创建元数据客户端
func NewMetadataClient(baseURL string) *MetadataClient {
	return &MetadataClient{
		HTTPClient: NewHTTPClient(baseURL, 5*time.Second),
	}
}

// SaveMetadata 保存元数据
func (c *MetadataClient) SaveMetadata(ctx context.Context, entry *types.MetadataEntry) error {
	return c.Post(ctx, "/metadata", entry, nil)
}

// GetMetadata 获取元数据
func (c *MetadataClient) GetMetadata(ctx context.Context, key string) (*types.MetadataEntry, error) {
	var entry types.MetadataEntry
	err := c.Get(ctx, "/metadata/"+key, &entry)
	return &entry, err
}

// DeleteMetadata 删除元数据
func (c *MetadataClient) DeleteMetadata(ctx context.Context, key string) error {
	return c.Delete(ctx, "/metadata/"+key)
}

// StorageClient 存储服务客户端
type StorageClient struct {
	*HTTPClient
}

// NewStorageClient 创建存储客户端
func NewStorageClient(baseURL string) *StorageClient {
	return &StorageClient{
		HTTPClient: NewHTTPClient(baseURL, 30*time.Second),
	}
}

// WriteObject 写入对象
func (c *StorageClient) WriteObject(ctx context.Context, obj *types.FileObject) error {
	return c.Post(ctx, "/storage/write", obj, nil)
}

// ReadObject 读取对象
func (c *StorageClient) ReadObject(ctx context.Context, key string) (*types.FileObject, error) {
	var obj types.FileObject
	err := c.Get(ctx, "/storage/read/"+key, &obj)
	return &obj, err
}

// DeleteObject 删除对象
func (c *StorageClient) DeleteObject(ctx context.Context, key string) error {
	return c.Delete(ctx, "/storage/delete/"+key)
}

// TaskClient 任务服务客户端
type TaskClient struct {
	*HTTPClient
}

// NewTaskClient 创建任务客户端
func NewTaskClient(baseURL string) *TaskClient {
	return &TaskClient{
		HTTPClient: NewHTTPClient(baseURL, 10*time.Second),
	}
}

// SubmitTask 提交任务
func (c *TaskClient) SubmitTask(ctx context.Context, task *types.TaskMessage) error {
	return c.Post(ctx, "/tasks", task, nil)
}
