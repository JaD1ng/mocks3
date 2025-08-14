package processor

import (
	"context"
	"fmt"
	"time"

	"micro-s3/shared/logger"
	"task-processor/internal/client"
	"task-processor/internal/queue"
)

// TaskProcessor 任务处理器
type TaskProcessor struct {
	clients *client.Clients
	logger  logger.Logger
}

// NewTaskProcessor 创建任务处理器
func NewTaskProcessor(clients *client.Clients) *TaskProcessor {
	return &TaskProcessor{
		clients: clients,
		logger:  logger.DefaultLogger,
	}
}

// ProcessTask 处理任务
func (p *TaskProcessor) ProcessTask(ctx context.Context, task *queue.TaskMessage) error {
	p.logger.Info(ctx, "Processing task", map[string]any{
		"task_id":   task.ID,
		"task_type": task.Type,
	})

	switch task.Type {
	case "delete_object":
		return p.processDeleteObject(ctx, task)
	case "post_upload":
		return p.processPostUpload(ctx, task)
	case "cleanup_storage":
		return p.processCleanupStorage(ctx, task)
	case "health_check":
		return p.processHealthCheck(ctx, task)
	default:
		return fmt.Errorf("unknown task type: %s", task.Type)
	}
}

// processDeleteObject 处理删除对象任务
func (p *TaskProcessor) processDeleteObject(ctx context.Context, task *queue.TaskMessage) error {
	objectKey := task.ObjectKey
	if objectKey == "" {
		return fmt.Errorf("object_key is required for delete_object task")
	}

	p.logger.Info(ctx, "Deleting object from storage", map[string]any{
		"object_key": objectKey,
	})

	// 调用存储服务删除对象
	err := p.clients.Storage.DeleteObject(ctx, objectKey)
	if err != nil {
		return fmt.Errorf("failed to delete object %s: %v", objectKey, err)
	}

	// 通知任务完成
	p.clients.Storage.NotifyTaskCompletion(ctx, "delete_object", true)

	p.logger.Info(ctx, "Successfully deleted object", map[string]any{
		"object_key": objectKey,
	})
	return nil
}

// processPostUpload 处理上传后任务
func (p *TaskProcessor) processPostUpload(ctx context.Context, task *queue.TaskMessage) error {
	objectKey := task.ObjectKey
	if objectKey == "" {
		return fmt.Errorf("object_key is required for post_upload task")
	}

	p.logger.Info(ctx, "Processing post-upload tasks", map[string]any{
		"object_key": objectKey,
	})

	// 模拟后处理任务
	time.Sleep(100 * time.Millisecond)

	// 可以在这里添加:
	// 1. 生成缩略图
	// 2. 病毒扫描
	// 3. 更新索引
	// 4. 发送通知

	size, ok := task.Data["size"].(float64)
	if ok {
		p.logger.Info(ctx, "Post-upload processing completed", map[string]any{
			"object_key": objectKey,
			"size_bytes": size,
		})
	}

	// 通知任务完成
	p.clients.Storage.NotifyTaskCompletion(ctx, "post_upload", true)

	return nil
}

// processCleanupStorage 处理存储清理任务
func (p *TaskProcessor) processCleanupStorage(ctx context.Context, task *queue.TaskMessage) error {
	p.logger.Info(ctx, "Processing storage cleanup task", nil)

	// 获取存储统计信息
	stats, err := p.clients.Storage.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get storage stats: %v", err)
	}

	// 检查存储使用情况
	usagePercent, ok := stats["usage_percent"].(float64)
	if ok && usagePercent > 85.0 {
		p.logger.Warn(ctx, "Storage usage is high", map[string]any{
			"usage_percent": usagePercent,
		})

		// 在实际环境中，这里可以:
		// 1. 删除临时文件
		// 2. 清理过期文件
		// 3. 压缩旧文件
		// 4. 发送告警
	}

	// 通知任务完成
	p.clients.Storage.NotifyTaskCompletion(ctx, "cleanup_storage", true)

	p.logger.Info(ctx, "Storage cleanup task completed", nil)
	return nil
}

// processHealthCheck 处理健康检查任务
func (p *TaskProcessor) processHealthCheck(ctx context.Context, task *queue.TaskMessage) error {
	p.logger.Info(ctx, "Processing health check task", nil)

	// 检查所有存储节点状态
	nodeIDs := []string{"stg1", "stg2", "stg3"}

	for _, nodeID := range nodeIDs {
		status, err := p.clients.Storage.GetNodeStatus(ctx, nodeID)
		if err != nil {
			p.logger.Warn(ctx, "Failed to get status for node", map[string]any{
				"node_id": nodeID,
				"error":   err.Error(),
			})
			continue
		}

		health, ok := status["health"].(string)
		if ok && health != "healthy" {
			p.logger.Warn(ctx, "Node is not healthy", map[string]any{
				"node_id": nodeID,
				"health":  health,
			})
		}
	}

	// 通知任务完成
	p.clients.Storage.NotifyTaskCompletion(ctx, "health_check", true)

	p.logger.Info(ctx, "Health check task completed", nil)
	return nil
}

// GetSupportedTaskTypes 获取支持的任务类型
func (p *TaskProcessor) GetSupportedTaskTypes() []string {
	return []string{
		"delete_object",
		"post_upload",
		"cleanup_storage",
		"health_check",
	}
}
