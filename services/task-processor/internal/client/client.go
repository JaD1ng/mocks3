package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mocks3/shared/logger"
	"net/http"
	"time"
)

// StorageClient 存储服务客户端
type StorageClient struct {
	baseURL    string
	httpClient *http.Client
}

// Clients 所有客户端
type Clients struct {
	Storage *StorageClient
}

// NewStorageClient 创建存储客户端
func NewStorageClient(baseURL string) *StorageClient {
	return &StorageClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DeleteObject 删除对象
func (c *StorageClient) DeleteObject(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s/storage/delete/%s", c.baseURL, key)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// GetStats 获取存储统计信息
func (c *StorageClient) GetStats(ctx context.Context) (map[string]any, error) {
	url := fmt.Sprintf("%s/stats", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var stats map[string]any
	err = json.NewDecoder(resp.Body).Decode(&stats)
	return stats, err
}

// GetNodeStatus 获取存储节点状态
func (c *StorageClient) GetNodeStatus(ctx context.Context, nodeID string) (map[string]any, error) {
	url := fmt.Sprintf("%s/nodes/%s/status", c.baseURL, nodeID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var status map[string]any
	err = json.NewDecoder(resp.Body).Decode(&status)
	return status, err
}

// NotifyTaskCompletion 通知任务完成（用于统计）
func (c *StorageClient) NotifyTaskCompletion(ctx context.Context, taskType string, success bool) error {
	// 这是一个可选的通知接口，用于统计任务完成情况
	notification := map[string]any{
		"task_type": taskType,
		"success":   success,
		"timestamp": time.Now(),
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/notifications/task-completion", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 通知失败不应该影响主要流程
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 只记录警告，不返回错误
		logger.Warnf("Task completion notification failed with HTTP %d", resp.StatusCode)
	}

	return nil
}
