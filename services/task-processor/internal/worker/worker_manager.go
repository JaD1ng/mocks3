package worker

import (
	"context"
	"fmt"
	"github.com/mocks3/shared/logger"
	"github.com/mocks3/shared/utils"
	"sync"
	"task-processor/internal/processor"
	"task-processor/internal/queue"
	"time"
)

// Worker 工作节点
type Worker struct {
	ID              string
	queue           *queue.RedisQueue
	processor       *processor.TaskProcessor
	ctx             context.Context
	cancel          context.CancelFunc
	running         bool
	mu              sync.RWMutex
	processedTasks  int64
	failedTasks     int64
	lastProcessedAt time.Time
	logger          logger.Logger
}

// WorkerManager 工作节点管理器
type WorkerManager struct {
	workers     []*Worker
	queue       *queue.RedisQueue
	processor   *processor.TaskProcessor
	workerCount int
	stopChan    chan struct{}
	running     bool
	mu          sync.RWMutex
	logger      logger.Logger
}

// NewWorkerManager 创建工作节点管理器
func NewWorkerManager(queue *queue.RedisQueue, processor *processor.TaskProcessor, workerCount int) *WorkerManager {
	return &WorkerManager{
		workers:     make([]*Worker, 0, workerCount),
		queue:       queue,
		processor:   processor,
		workerCount: workerCount,
		stopChan:    make(chan struct{}),
		logger:      logger.DefaultLogger,
	}
}

// Start 启动所有工作节点
func (wm *WorkerManager) Start() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if wm.running {
		return fmt.Errorf("worker manager already running")
	}

	for i := 0; i < wm.workerCount; i++ {
		worker := &Worker{
			ID:        fmt.Sprintf("worker-%s", utils.GenerateID()[:8]),
			queue:     wm.queue,
			processor: wm.processor,
			logger:    logger.DefaultLogger,
		}

		worker.ctx, worker.cancel = context.WithCancel(context.Background())
		wm.workers = append(wm.workers, worker)

		go worker.run()
		wm.logger.Info(context.Background(), "Started worker", map[string]any{
			"worker_id": worker.ID,
		})
	}

	wm.running = true
	wm.logger.Info(context.Background(), "Started workers", map[string]any{
		"worker_count": len(wm.workers),
	})
	return nil
}

// Stop 停止所有工作节点
func (wm *WorkerManager) Stop() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if !wm.running {
		return
	}

	wm.logger.Info(context.Background(), "Stopping workers", map[string]any{
		"worker_count": len(wm.workers),
	})

	// 停止所有工作节点
	for _, worker := range wm.workers {
		worker.stop()
	}

	// 发送停止信号
	close(wm.stopChan)
	wm.running = false

	wm.logger.Info(context.Background(), "All workers stopped", nil)
}

// GetWorkerStatus 获取工作节点状态
func (wm *WorkerManager) GetWorkerStatus() []map[string]any {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var status []map[string]any

	for _, worker := range wm.workers {
		worker.mu.RLock()
		workerStatus := map[string]any{
			"id":                worker.ID,
			"running":           worker.running,
			"processed_tasks":   worker.processedTasks,
			"failed_tasks":      worker.failedTasks,
			"last_processed_at": worker.lastProcessedAt,
		}
		worker.mu.RUnlock()

		status = append(status, workerStatus)
	}

	return status
}

// GetStats 获取工作节点统计信息
func (wm *WorkerManager) GetStats() map[string]any {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	var totalProcessed int64 = 0
	var totalFailed int64 = 0
	var runningWorkers int = 0

	for _, worker := range wm.workers {
		worker.mu.RLock()
		totalProcessed += worker.processedTasks
		totalFailed += worker.failedTasks
		if worker.running {
			runningWorkers++
		}
		worker.mu.RUnlock()
	}

	return map[string]any{
		"total_workers":   len(wm.workers),
		"running_workers": runningWorkers,
		"total_processed": totalProcessed,
		"total_failed":    totalFailed,
		"success_rate":    float64(totalProcessed) / float64(totalProcessed+totalFailed) * 100,
	}
}

// run 工作节点运行循环
func (w *Worker) run() {
	w.mu.Lock()
	w.running = true
	w.mu.Unlock()

	w.logger.Info(w.ctx, "Worker started", map[string]any{
		"worker_id": w.ID,
	})

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info(w.ctx, "Worker stopping due to context cancellation", map[string]any{
				"worker_id": w.ID,
			})
			goto cleanup
		default:
			// 从队列读取任务
			tasks, err := w.queue.ReadTasks(w.ctx, w.ID, 1)
			if err != nil {
				w.logger.Error(w.ctx, "Worker failed to read tasks", err, map[string]any{
					"worker_id": w.ID,
				})
				time.Sleep(1 * time.Second)
				continue
			}

			// 处理任务
			for _, task := range tasks {
				w.processTask(task)
			}

			// 如果没有任务，稍微等待
			if len(tasks) == 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

cleanup:
	w.mu.Lock()
	w.running = false
	w.mu.Unlock()
	w.logger.Info(w.ctx, "Worker stopped", map[string]any{
		"worker_id": w.ID,
	})
}

// processTask 处理单个任务
func (w *Worker) processTask(task *queue.TaskMessage) {
	w.logger.Info(w.ctx, "Worker processing task", map[string]any{
		"worker_id": w.ID,
		"task_id":   task.ID,
		"task_type": task.Type,
	})

	startTime := time.Now()

	err := w.processor.ProcessTask(w.ctx, task)

	w.mu.Lock()
	if err != nil {
		w.failedTasks++
		w.logger.Error(w.ctx, "Worker failed to process task", err, map[string]any{
			"worker_id": w.ID,
			"task_id":   task.ID,
		})

		// 检查是否需要重试
		task.RetryCount++
		if task.RetryCount < task.MaxRetries {
			w.logger.Warn(w.ctx, "Task will be retried", map[string]any{
				"task_id":     task.ID,
				"retry_count": task.RetryCount,
				"max_retries": task.MaxRetries,
			})
			// 在实际环境中，可以将任务重新放入队列
		}
	} else {
		w.processedTasks++
		w.lastProcessedAt = time.Now()

		// 确认任务完成
		err = w.queue.AckTask(w.ctx, task.ID)
		if err != nil {
			w.logger.Error(w.ctx, "Worker failed to ack task", err, map[string]any{
				"worker_id": w.ID,
				"task_id":   task.ID,
			})
		} else {
			w.logger.Info(w.ctx, "Worker completed task", map[string]any{
				"worker_id": w.ID,
				"task_id":   task.ID,
				"duration":  time.Since(startTime).String(),
			})
		}
	}
	w.mu.Unlock()
}

// stop 停止工作节点
func (w *Worker) stop() {
	w.cancel()
}
