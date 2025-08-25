package service

import (
	"context"
	"fmt"
	"mocks3/services/queue/internal/repository"
	"mocks3/shared/interfaces"
	"mocks3/shared/models"
	"mocks3/shared/observability"
	"sync"
	"time"
)

// QueueService 队列服务实现
type QueueService struct {
	repo    *repository.RedisRepository
	logger  *observability.Logger
	workers map[string]*Worker
	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
}

// Worker 工作节点
type Worker struct {
	ID      string
	service *QueueService
	logger  *observability.Logger
	stopCh  chan struct{}
	running bool
	mu      sync.RWMutex
}

// NewQueueService 创建队列服务
func NewQueueService(repo *repository.RedisRepository, logger *observability.Logger) *QueueService {
	ctx, cancel := context.WithCancel(context.Background())

	return &QueueService{
		repo:    repo,
		logger:  logger,
		workers: make(map[string]*Worker),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// AddTask 添加任务到队列
func (qs *QueueService) AddTask(ctx context.Context, task *models.Task) error {
	qs.logger.Info(ctx, "Adding task to queue", 
		observability.String("task_id", task.ID), 
		observability.String("type", task.Type))

	// 设置任务状态和时间戳
	task.Status = "pending"
	task.CreatedAt = time.Now()
	task.UpdatedAt = task.CreatedAt

	if err := qs.repo.AddTask(ctx, task); err != nil {
		qs.logger.Error(ctx, "Failed to add task", 
			observability.String("error", err.Error()), 
			observability.String("task_id", task.ID))
		return fmt.Errorf("failed to add task: %w", err)
	}

	qs.logger.Info(ctx, "Task added successfully", 
		observability.String("task_id", task.ID), 
		observability.String("stream_id", task.StreamID))
	return nil
}

// GetTask 获取任务
func (qs *QueueService) GetTask(ctx context.Context, taskID string) (*models.Task, error) {
	qs.logger.Debug(ctx, "Getting task", 
		observability.String("task_id", taskID))

	task, err := qs.repo.GetTaskStatus(ctx, taskID)
	if err != nil {
		qs.logger.Warn(ctx, "Task not found", 
			observability.String("task_id", taskID), 
			observability.String("error", err.Error()))
		return nil, fmt.Errorf("task not found: %w", err)
	}

	return task, nil
}

// ListTasks 列出任务
func (qs *QueueService) ListTasks(ctx context.Context, status string, limit int) ([]*models.Task, error) {
	qs.logger.Debug(ctx, "Listing tasks", 
		observability.String("status", status), 
		observability.Int("limit", limit))

	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	tasks, err := qs.repo.ListTasks(ctx, status, int64(limit))
	if err != nil {
		qs.logger.Error(ctx, "Failed to list tasks", 
			observability.String("error", err.Error()))
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	qs.logger.Debug(ctx, "Tasks listed", 
		observability.Int("count", len(tasks)))
	return tasks, nil
}

// GetStats 获取队列统计信息
func (qs *QueueService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	qs.logger.Debug(ctx, "Getting queue statistics")

	stats, err := qs.repo.GetStats(ctx)
	if err != nil {
		qs.logger.Error(ctx, "Failed to get statistics", 
			observability.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	// 添加工作节点信息
	qs.mu.RLock()
	workerStats := make(map[string]interface{})
	for id, worker := range qs.workers {
		worker.mu.RLock()
		workerStats[id] = map[string]interface{}{
			"running": worker.running,
		}
		worker.mu.RUnlock()
	}
	qs.mu.RUnlock()

	stats["workers"] = workerStats
	stats["worker_count"] = len(qs.workers)

	return stats, nil
}

// StartWorker 启动工作节点
func (qs *QueueService) StartWorker(ctx context.Context, workerID string) error {
	qs.logger.Info(ctx, "Starting worker", 
		observability.String("worker_id", workerID))

	qs.mu.Lock()
	defer qs.mu.Unlock()

	if _, exists := qs.workers[workerID]; exists {
		return fmt.Errorf("worker %s already exists", workerID)
	}

	worker := &Worker{
		ID:      workerID,
		service: qs,
		logger:  qs.logger,
		stopCh:  make(chan struct{}),
	}

	qs.workers[workerID] = worker
	go worker.start()

	qs.logger.Info(ctx, "Worker started", 
		observability.String("worker_id", workerID))
	return nil
}

// StopWorker 停止工作节点
func (qs *QueueService) StopWorker(ctx context.Context, workerID string) error {
	qs.logger.Info(ctx, "Stopping worker", 
		observability.String("worker_id", workerID))

	qs.mu.Lock()
	defer qs.mu.Unlock()

	worker, exists := qs.workers[workerID]
	if !exists {
		return fmt.Errorf("worker %s not found", workerID)
	}

	worker.stop()
	delete(qs.workers, workerID)

	qs.logger.Info(ctx, "Worker stopped", 
		observability.String("worker_id", workerID))
	return nil
}

// Stop 停止队列服务
func (qs *QueueService) Stop() error {
	qs.logger.Info(context.Background(), "Stopping queue service")

	// 停止所有工作节点
	qs.mu.Lock()
	for id, worker := range qs.workers {
		qs.logger.Info(context.Background(), "Stopping worker", 
			observability.String("worker_id", id))
		worker.stop()
	}
	qs.workers = make(map[string]*Worker)
	qs.mu.Unlock()

	// 取消上下文
	qs.cancel()

	// 关闭仓库连接
	if err := qs.repo.Close(); err != nil {
		qs.logger.Error(context.Background(), "Failed to close repository", 
			observability.String("error", err.Error()))
		return err
	}

	qs.logger.Info(context.Background(), "Queue service stopped")
	return nil
}

// HealthCheck 健康检查
func (qs *QueueService) HealthCheck(ctx context.Context) error {
	qs.logger.DebugContext(ctx, "Performing health check")

	// 检查Redis连接
	_, err := qs.repo.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}

// EnqueueTask 入队任务 (接口方法)
func (qs *QueueService) EnqueueTask(ctx context.Context, task *models.Task) error {
	return qs.AddTask(ctx, task)
}

// DequeueTask 出队任务 (接口方法)
func (qs *QueueService) DequeueTask(ctx context.Context, queueName string) (*models.Task, error) {
	tasks, err := qs.repo.GetTasks(ctx, queueName, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue task: %w", err)
	}
	if len(tasks) == 0 {
		return nil, nil
	}
	return tasks[0], nil
}

// CreateQueue 创建队列 (接口方法)
func (qs *QueueService) CreateQueue(ctx context.Context, queueName string, config *models.QueueConfig) error {
	qs.logger.InfoContext(ctx, "Creating queue", "queue_name", queueName)
	// Redis Streams会在第一次添加消息时自动创建
	// 这里我们可以记录队列配置或进行验证
	return nil
}

// DeleteQueue 删除队列 (接口方法)
func (qs *QueueService) DeleteQueue(ctx context.Context, queueName string) error {
	qs.logger.InfoContext(ctx, "Deleting queue", "queue_name", queueName)
	// TODO: 实现队列删除逻辑
	return nil
}

// ListQueues 列出队列 (接口方法)
func (qs *QueueService) ListQueues(ctx context.Context) ([]string, error) {
	// TODO: 实现从Redis获取所有队列名称
	return []string{}, nil
}

// GetQueueStats 获取队列统计 (接口方法)
func (qs *QueueService) GetQueueStats(ctx context.Context, queueName string) (*models.QueueStats, error) {
	stats, err := qs.repo.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	// 构造QueueStats
	queueStats := &models.QueueStats{
		QueueName:   queueName,
		Length:      int64(stats["pending_count"].(int64)),
		FailedCount: int64(stats["failed_count"].(int64)),
		LastMessage: time.Now(), // TODO: 从实际数据获取
	}

	return queueStats, nil
}

// GetQueueLength 获取队列长度 (接口方法)
func (qs *QueueService) GetQueueLength(ctx context.Context, queueName string) (int64, error) {
	stats, err := qs.GetQueueStats(ctx, queueName)
	if err != nil {
		return 0, err
	}
	return stats.Length, nil
}

// RegisterWorker 注册工作节点 (接口方法)
func (qs *QueueService) RegisterWorker(ctx context.Context, workerID string, queues []string) error {
	return qs.StartWorker(ctx, workerID)
}

// UnregisterWorker 注销工作节点 (接口方法)
func (qs *QueueService) UnregisterWorker(ctx context.Context, workerID string) error {
	return qs.StopWorker(ctx, workerID)
}

// ListWorkers 列出工作节点 (接口方法)
func (qs *QueueService) ListWorkers(ctx context.Context) ([]*models.Worker, error) {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	workers := make([]*models.Worker, 0, len(qs.workers))
	for _, worker := range qs.workers {
		status := models.WorkerStatusStopped
		if worker.running {
			status = models.WorkerStatusRunning
		}

		modelWorker := &models.Worker{
			ID:        worker.ID,
			Name:      worker.ID,
			Status:    status,
			LastSeen:  time.Now(),
			StartedAt: time.Now(), // TODO: 记录实际启动时间
		}
		workers = append(workers, modelWorker)
	}

	return workers, nil
}

// Worker methods

// start 启动工作节点
func (w *Worker) start() {
	w.mu.Lock()
	w.running = true
	w.mu.Unlock()

	w.logger.Info(context.Background(), "Worker started", 
		observability.String("worker_id", w.ID))

	for {
		select {
		case <-w.stopCh:
			w.logger.Info(context.Background(), "Worker stopped", 
			observability.String("worker_id", w.ID))
			return
		case <-w.service.ctx.Done():
			w.logger.Info(context.Background(), "Worker stopping due to service shutdown", 
			observability.String("worker_id", w.ID))
			return
		default:
			w.processTasks()
		}
	}
}

// stop 停止工作节点
func (w *Worker) stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		close(w.stopCh)
		w.running = false
	}
}

// processTasks 处理任务
func (w *Worker) processTasks() {
	ctx, cancel := context.WithTimeout(w.service.ctx, 30*time.Second)
	defer cancel()

	// 获取待处理任务
	tasks, err := w.service.repo.GetTasks(ctx, w.ID, 5)
	if err != nil {
		if err != context.Canceled {
			w.logger.Error(context.Background(), "Failed to get tasks", 
				observability.String("worker_id", w.ID), 
				observability.String("error", err.Error()))
		}
		time.Sleep(1 * time.Second)
		return
	}

	// 处理每个任务
	for _, task := range tasks {
		w.processTask(ctx, task)
	}

	// 如果没有任务，短暂休眠
	if len(tasks) == 0 {
		time.Sleep(2 * time.Second)
	}
}

// processTask 处理单个任务
func (w *Worker) processTask(ctx context.Context, task *models.Task) {
	w.logger.InfoContext(ctx, "Processing task",
		"worker_id", w.ID,
		"task_id", task.ID,
		"task_type", task.Type)

	// 更新任务状态
	task.Status = "processing"
	task.UpdatedAt = time.Now()

	// 根据任务类型处理
	var err error
	switch task.Type {
	case "file_deletion":
		err = w.processFileDeletion(ctx, task)
	case "metadata_cleanup":
		err = w.processMetadataCleanup(ctx, task)
	case "storage_optimization":
		err = w.processStorageOptimization(ctx, task)
	default:
		err = fmt.Errorf("unknown task type: %s", task.Type)
	}

	if err != nil {
		w.logger.ErrorContext(ctx, "Task processing failed",
			"worker_id", w.ID,
			"task_id", task.ID,
			"error", err)

		// 拒绝任务（重试或标记失败）
		if rejectErr := w.service.repo.RejectTask(ctx, task); rejectErr != nil {
			w.logger.ErrorContext(ctx, "Failed to reject task", "task_id", task.ID, "error", rejectErr)
		}
		return
	}

	// 确认任务完成
	if ackErr := w.service.repo.AckTask(ctx, task.StreamID); ackErr != nil {
		w.logger.ErrorContext(ctx, "Failed to ack task", "task_id", task.ID, "error", ackErr)
		return
	}

	w.logger.InfoContext(ctx, "Task completed successfully",
		"worker_id", w.ID,
		"task_id", task.ID)
}

// processFileDeletion 处理文件删除任务
func (w *Worker) processFileDeletion(ctx context.Context, task *models.Task) error {
	w.logger.InfoContext(ctx, "Processing file deletion", "task_id", task.ID)

	// 解析任务数据
	if task.Data == nil {
		return fmt.Errorf("task data is nil")
	}

	bucket, exists := task.Data["bucket"]
	if !exists {
		return fmt.Errorf("bucket not specified in task data")
	}

	key, exists := task.Data["key"]
	if !exists {
		return fmt.Errorf("key not specified in task data")
	}

	w.logger.InfoContext(ctx, "Deleting file",
		"bucket", bucket,
		"key", key,
		"task_id", task.ID)

	// 这里应该调用存储服务来删除文件
	// 由于我们还没有实现存储服务的接口调用，先模拟处理
	// TODO: 实现实际的文件删除逻辑

	// 模拟处理时间
	time.Sleep(100 * time.Millisecond)

	w.logger.InfoContext(ctx, "File deletion completed",
		"bucket", bucket,
		"key", key,
		"task_id", task.ID)

	return nil
}

// processMetadataCleanup 处理元数据清理任务
func (w *Worker) processMetadataCleanup(ctx context.Context, task *models.Task) error {
	w.logger.InfoContext(ctx, "Processing metadata cleanup", "task_id", task.ID)

	// TODO: 实现元数据清理逻辑
	time.Sleep(50 * time.Millisecond)

	return nil
}

// processStorageOptimization 处理存储优化任务
func (w *Worker) processStorageOptimization(ctx context.Context, task *models.Task) error {
	w.logger.InfoContext(ctx, "Processing storage optimization", "task_id", task.ID)

	// TODO: 实现存储优化逻辑
	time.Sleep(200 * time.Millisecond)

	return nil
}

// 确保QueueService实现了QueueService接口
var _ interfaces.QueueService = (*QueueService)(nil)
