package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"mocks3/services/queue/internal/config"
	"mocks3/shared/models"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRepository Redis队列仓库
type RedisRepository struct {
	client *redis.Client
	config *config.QueueConfig
}

// NewRedisRepository 创建Redis仓库
func NewRedisRepository(redisConfig *config.RedisConfig, queueConfig *config.QueueConfig) (*RedisRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisConfig.GetAddress(),
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisRepository{
		client: client,
		config: queueConfig,
	}, nil
}

// AddTask 添加任务到队列
func (r *RedisRepository) AddTask(ctx context.Context, task *models.Task) error {
	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	args := &redis.XAddArgs{
		Stream: r.config.StreamName,
		Values: map[string]interface{}{
			"task_id":    task.ID,
			"task_type":  task.Type,
			"priority":   task.Priority,
			"data":       string(taskData),
			"created_at": task.CreatedAt.Format(time.RFC3339),
		},
	}

	msgID, err := r.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("failed to add task to stream: %w", err)
	}

	task.StreamID = msgID
	return nil
}

// GetTasks 获取待处理任务
func (r *RedisRepository) GetTasks(ctx context.Context, consumerName string, count int64) ([]*models.Task, error) {
	// 创建消费者组（如果不存在）
	err := r.ensureConsumerGroup(ctx)
	if err != nil {
		return nil, err
	}

	// 读取消息
	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    r.config.ConsumerGroup,
		Consumer: consumerName,
		Streams:  []string{r.config.StreamName, ">"},
		Count:    count,
		Block:    time.Duration(r.config.ProcessTimeout) * time.Second,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return []*models.Task{}, nil
		}
		return nil, fmt.Errorf("failed to read from stream: %w", err)
	}

	var tasks []*models.Task
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			task, err := r.messageToTask(msg)
			if err != nil {
				// 记录错误但继续处理其他消息
				continue
			}
			task.StreamID = msg.ID
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// AckTask 确认任务完成
func (r *RedisRepository) AckTask(ctx context.Context, streamID string) error {
	err := r.client.XAck(ctx, r.config.StreamName, r.config.ConsumerGroup, streamID).Err()
	if err != nil {
		return fmt.Errorf("failed to ack message %s: %w", streamID, err)
	}
	return nil
}

// RejectTask 拒绝任务（重新入队）
func (r *RedisRepository) RejectTask(ctx context.Context, task *models.Task) error {
	// 增加重试次数
	task.RetryCount++

	if task.RetryCount >= r.config.MaxRetries {
		// 超过最大重试次数，标记为失败
		task.Status = models.TaskStatusFailed
		task.UpdatedAt = time.Now()

		// 记录到失败队列（可选）
		failedData, _ := json.Marshal(task)
		r.client.LPush(ctx, r.config.StreamName+":failed", failedData)

		// 确认原消息
		return r.AckTask(ctx, task.StreamID)
	}

	// 重新添加到队列
	task.Status = models.TaskStatusRetrying
	task.UpdatedAt = time.Now()

	return r.AddTask(ctx, task)
}

// GetTaskStatus 获取任务状态
func (r *RedisRepository) GetTaskStatus(ctx context.Context, taskID string) (*models.Task, error) {
	// 从待处理队列查找
	result, err := r.client.XRevRange(ctx, r.config.StreamName, "+", "-").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to search stream: %w", err)
	}

	for _, msg := range result {
		if taskIDValue, exists := msg.Values["task_id"]; exists {
			if taskIDValue == taskID {
				return r.messageToTask(msg)
			}
		}
	}

	// 从失败队列查找
	failedTasks, err := r.client.LRange(ctx, r.config.StreamName+":failed", 0, -1).Result()
	if err == nil {
		for _, taskData := range failedTasks {
			var task models.Task
			if json.Unmarshal([]byte(taskData), &task) == nil && task.ID == taskID {
				return &task, nil
			}
		}
	}

	return nil, fmt.Errorf("task not found: %s", taskID)
}

// ListTasks 列出任务
func (r *RedisRepository) ListTasks(ctx context.Context, status string, limit int64) ([]*models.Task, error) {
	var tasks []*models.Task

	switch status {
	case "pending", "processing", "":
		// 从主队列获取
		result, err := r.client.XRevRange(ctx, r.config.StreamName, "+", "-").Result()
		if err != nil {
			return nil, fmt.Errorf("failed to list pending tasks: %w", err)
		}

		count := int64(0)
		for _, msg := range result {
			if limit > 0 && count >= limit {
				break
			}

			task, err := r.messageToTask(msg)
			if err != nil {
				continue
			}

			if status == "" || string(task.Status) == status {
				task.StreamID = msg.ID
				tasks = append(tasks, task)
				count++
			}
		}

	case "failed":
		// 从失败队列获取
		failedTasks, err := r.client.LRange(ctx, r.config.StreamName+":failed", 0, limit-1).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to list failed tasks: %w", err)
		}

		for _, taskData := range failedTasks {
			var task models.Task
			if json.Unmarshal([]byte(taskData), &task) == nil {
				tasks = append(tasks, &task)
			}
		}
	}

	return tasks, nil
}

// GetStats 获取队列统计信息
func (r *RedisRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 待处理任务数
	pendingCount, err := r.client.XLen(ctx, r.config.StreamName).Result()
	if err == nil {
		stats["pending_count"] = pendingCount
	}

	// 失败任务数
	failedCount, err := r.client.LLen(ctx, r.config.StreamName+":failed").Result()
	if err == nil {
		stats["failed_count"] = failedCount
	}

	// 消费者组信息
	groups, err := r.client.XInfoGroups(ctx, r.config.StreamName).Result()
	if err == nil {
		for _, group := range groups {
			if group.Name == r.config.ConsumerGroup {
				stats["consumer_group"] = map[string]interface{}{
					"name":      group.Name,
					"consumers": group.Consumers,
					"pending":   group.Pending,
				}
				break
			}
		}
	}

	stats["stream_name"] = r.config.StreamName
	stats["max_retries"] = r.config.MaxRetries

	return stats, nil
}

// Close 关闭连接
func (r *RedisRepository) Close() error {
	return r.client.Close()
}

// ensureConsumerGroup 确保消费者组存在
func (r *RedisRepository) ensureConsumerGroup(ctx context.Context) error {
	// 检查消费者组是否存在
	groups, err := r.client.XInfoGroups(ctx, r.config.StreamName).Result()
	if err != nil {
		// 如果stream不存在，先创建一个空的消息
		if err.Error() == "ERR no such key" {
			r.client.XAdd(ctx, &redis.XAddArgs{
				Stream: r.config.StreamName,
				Values: map[string]interface{}{"init": "true"},
			})
		}
	}

	// 检查消费者组是否已存在
	for _, group := range groups {
		if group.Name == r.config.ConsumerGroup {
			return nil
		}
	}

	// 创建消费者组
	err = r.client.XGroupCreate(ctx, r.config.StreamName, r.config.ConsumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	return nil
}

// messageToTask 将Redis消息转换为Task
func (r *RedisRepository) messageToTask(msg redis.XMessage) (*models.Task, error) {
	taskData, exists := msg.Values["data"]
	if !exists {
		return nil, fmt.Errorf("task data not found in message")
	}

	var task models.Task
	err := json.Unmarshal([]byte(taskData.(string)), &task)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal task data: %w", err)
	}

	// 设置优先级（如果存在）
	if priorityStr, exists := msg.Values["priority"]; exists {
		if priority, err := strconv.Atoi(priorityStr.(string)); err == nil {
			task.Priority = priority
		}
	}

	return &task, nil
}
