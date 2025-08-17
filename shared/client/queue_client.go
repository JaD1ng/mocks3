package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mocks3/shared/models"
	"net/http"
	"net/url"
	"time"
)

// QueueClient 队列服务客户端
type QueueClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewQueueClient 创建队列服务客户端
func NewQueueClient(baseURL string, timeout time.Duration) *QueueClient {
	return &QueueClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// EnqueueTask 入队任务
func (c *QueueClient) EnqueueTask(ctx context.Context, task *models.Task) error {
	reqURL := fmt.Sprintf("%s/tasks", c.baseURL)
	
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(body))
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

// DequeueTask 出队任务
func (c *QueueClient) DequeueTask(ctx context.Context, queueName string) (*models.Task, error) {
	reqURL := fmt.Sprintf("%s/tasks/dequeue", c.baseURL)
	
	u, err := url.Parse(reqURL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	
	q := u.Query()
	q.Set("queue", queueName)
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
	
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil // 队列为空
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var task models.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	
	return &task, nil
}

// CreateQueue 创建队列
func (c *QueueClient) CreateQueue(ctx context.Context, config *models.QueueConfig) error {
	reqURL := fmt.Sprintf("%s/queues", c.baseURL)
	
	body, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(body))
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

// DeleteQueue 删除队列
func (c *QueueClient) DeleteQueue(ctx context.Context, queueName string) error {
	reqURL := fmt.Sprintf("%s/queues/%s", c.baseURL, url.PathEscape(queueName))
	
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", reqURL, nil)
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

// ListQueues 列出队列
func (c *QueueClient) ListQueues(ctx context.Context) ([]string, error) {
	reqURL := fmt.Sprintf("%s/queues", c.baseURL)
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
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
	
	var queues []string
	if err := json.NewDecoder(resp.Body).Decode(&queues); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	
	return queues, nil
}

// GetQueueStats 获取队列统计
func (c *QueueClient) GetQueueStats(ctx context.Context, queueName string) (*models.QueueStats, error) {
	reqURL := fmt.Sprintf("%s/queues/%s/stats", c.baseURL, url.PathEscape(queueName))
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
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
	
	var stats models.QueueStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	
	return &stats, nil
}

// RegisterWorker 注册工作节点
func (c *QueueClient) RegisterWorker(ctx context.Context, worker *models.Worker) error {
	reqURL := fmt.Sprintf("%s/workers", c.baseURL)
	
	body, err := json.Marshal(worker)
	if err != nil {
		return fmt.Errorf("marshal worker: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(body))
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

// UnregisterWorker 注销工作节点
func (c *QueueClient) UnregisterWorker(ctx context.Context, workerID string) error {
	reqURL := fmt.Sprintf("%s/workers/%s", c.baseURL, url.PathEscape(workerID))
	
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", reqURL, nil)
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

// ListWorkers 列出工作节点
func (c *QueueClient) ListWorkers(ctx context.Context) ([]*models.Worker, error) {
	reqURL := fmt.Sprintf("%s/workers", c.baseURL)
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
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
	
	var workers []*models.Worker
	if err := json.NewDecoder(resp.Body).Decode(&workers); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	
	return workers, nil
}

// UpdateTaskStatus 更新任务状态
func (c *QueueClient) UpdateTaskStatus(ctx context.Context, taskID string, status models.TaskStatus, error string) error {
	reqURL := fmt.Sprintf("%s/tasks/%s/status", c.baseURL, url.PathEscape(taskID))
	
	req := map[string]interface{}{
		"status": status,
	}
	if error != "" {
		req["error"] = error
	}
	
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", reqURL, bytes.NewReader(body))
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

// HealthCheck 健康检查
func (c *QueueClient) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
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