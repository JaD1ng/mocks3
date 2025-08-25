package client

import (
	"context"
	"fmt"
	"mocks3/shared/models"
	"net/http"
	"time"
)

// QueueClient 队列服务客户端
type QueueClient struct {
	*BaseHTTPClient
}

// NewQueueClient 创建队列服务客户端
func NewQueueClient(baseURL string, timeout time.Duration) *QueueClient {
	return &QueueClient{
		BaseHTTPClient: NewBaseHTTPClient(baseURL, timeout),
	}
}

// EnqueueTask 入队任务
func (c *QueueClient) EnqueueTask(ctx context.Context, task *models.Task) error {
	return c.PostExpectStatus(ctx, "/tasks", task, http.StatusCreated)
}

// DequeueTask 出队任务
func (c *QueueClient) DequeueTask(ctx context.Context, queueName string) (*models.Task, error) {
	queryParams := map[string]string{"queue": queueName}

	resp, err := c.DoRequest(ctx, RequestOptions{
		Method:      "GET",
		Path:        "/tasks/dequeue",
		QueryParams: queryParams,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil // 队列为空
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var task models.Task
	if err := c.DoRequestWithJSON(ctx, RequestOptions{
		Method:      "GET",
		Path:        "/tasks/dequeue",
		QueryParams: queryParams,
	}, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// CreateQueue 创建队列
func (c *QueueClient) CreateQueue(ctx context.Context, config *models.QueueConfig) error {
	return c.PostExpectStatus(ctx, "/queues", config, http.StatusCreated)
}

// DeleteQueue 删除队列
func (c *QueueClient) DeleteQueue(ctx context.Context, queueName string) error {
	path := fmt.Sprintf("/queues/%s", PathEscape(queueName))
	return c.Delete(ctx, path)
}

// ListQueues 列出队列
func (c *QueueClient) ListQueues(ctx context.Context) ([]string, error) {
	var queues []string
	err := c.Get(ctx, "/queues", nil, &queues)
	return queues, err
}

// GetQueueStats 获取队列统计
func (c *QueueClient) GetQueueStats(ctx context.Context, queueName string) (*models.QueueStats, error) {
	path := fmt.Sprintf("/queues/%s/stats", PathEscape(queueName))
	var stats models.QueueStats
	err := c.Get(ctx, path, nil, &stats)
	return &stats, err
}

// RegisterWorker 注册工作节点
func (c *QueueClient) RegisterWorker(ctx context.Context, worker *models.Worker) error {
	return c.PostExpectStatus(ctx, "/workers", worker, http.StatusCreated)
}

// UnregisterWorker 注销工作节点
func (c *QueueClient) UnregisterWorker(ctx context.Context, workerID string) error {
	path := fmt.Sprintf("/workers/%s", PathEscape(workerID))
	return c.Delete(ctx, path)
}

// ListWorkers 列出工作节点
func (c *QueueClient) ListWorkers(ctx context.Context) ([]*models.Worker, error) {
	var workers []*models.Worker
	err := c.Get(ctx, "/workers", nil, &workers)
	return workers, err
}

// UpdateTaskStatus 更新任务状态
func (c *QueueClient) UpdateTaskStatus(ctx context.Context, taskID string, status models.TaskStatus, error string) error {
	path := fmt.Sprintf("/tasks/%s/status", PathEscape(taskID))
	req := map[string]any{
		"status": status,
	}
	if error != "" {
		req["error"] = error
	}
	return c.PutExpectStatus(ctx, path, req, http.StatusOK)
}

// HealthCheck 健康检查
func (c *QueueClient) HealthCheck(ctx context.Context) error {
	return c.BaseHTTPClient.HealthCheck(ctx)
}