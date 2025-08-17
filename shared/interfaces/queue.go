package interfaces

import (
	"context"
	"mocks3/shared/models"
)

// QueueService 队列服务接口
type QueueService interface {
	// 任务操作
	EnqueueTask(ctx context.Context, task *models.Task) error
	DequeueTask(ctx context.Context, queueName string) (*models.Task, error)

	// 队列管理
	CreateQueue(ctx context.Context, queueName string, config *models.QueueConfig) error
	DeleteQueue(ctx context.Context, queueName string) error
	ListQueues(ctx context.Context) ([]string, error)

	// 队列状态
	GetQueueStats(ctx context.Context, queueName string) (*models.QueueStats, error)
	GetQueueLength(ctx context.Context, queueName string) (int64, error)

	// 工作节点管理
	RegisterWorker(ctx context.Context, workerID string, queues []string) error
	UnregisterWorker(ctx context.Context, workerID string) error
	ListWorkers(ctx context.Context) ([]*models.Worker, error)

	// 健康检查
	HealthCheck(ctx context.Context) error
}

// TaskProcessor 任务处理器接口
type TaskProcessor interface {
	ProcessTask(ctx context.Context, task *models.Task) error
	GetSupportedTaskTypes() []string
}

// Worker 工作节点接口
type Worker interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	IsRunning() bool
	GetWorkerID() string
	RegisterProcessor(taskType string, processor TaskProcessor)
}
