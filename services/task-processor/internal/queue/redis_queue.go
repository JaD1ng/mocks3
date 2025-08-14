package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TaskMessage 任务消息
type TaskMessage struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	ObjectKey  string         `json:"object_key"`
	Data       map[string]any `json:"data"`
	CreatedAt  time.Time      `json:"created_at"`
	Status     string         `json:"status"`
	RetryCount int            `json:"retry_count"`
	MaxRetries int            `json:"max_retries"`
	Error      string         `json:"error,omitempty"`
}

// RedisQueue Redis 队列实现
type RedisQueue struct {
	client     *redis.Client
	streamName string
	groupName  string
}

// NewRedisQueue 创建 Redis 队列
func NewRedisQueue(redisURL string) (*RedisQueue, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	queue := &RedisQueue{
		client:     client,
		streamName: "tasks",
		groupName:  "task-processors",
	}

	// 初始化 Stream 和 Consumer Group
	err = queue.initializeStream()
	if err != nil {
		return nil, err
	}

	return queue, nil
}

// initializeStream 初始化 Stream 和 Consumer Group
func (q *RedisQueue) initializeStream() error {
	ctx := context.Background()

	// 检查 Stream 是否存在，不存在则创建
	_, err := q.client.XInfoStream(ctx, q.streamName).Result()
	if err != nil {
		// Stream 不存在，创建一个
		_, err = q.client.XAdd(ctx, &redis.XAddArgs{
			Stream: q.streamName,
			Values: map[string]any{"init": "true"},
		}).Result()
		if err != nil {
			return fmt.Errorf("failed to create stream: %v", err)
		}
	}

	// 创建 Consumer Group（如果不存在）
	_, err = q.client.XGroupCreate(ctx, q.streamName, q.groupName, "0").Result()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %v", err)
	}

	return nil
}

// SubmitTask 提交任务
func (q *RedisQueue) SubmitTask(ctx context.Context, task *TaskMessage) error {
	task.Status = "pending"
	task.CreatedAt = time.Now()

	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	_, err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamName,
		Values: map[string]any{
			"task_id": task.ID,
			"data":    string(data),
		},
	}).Result()

	return err
}

// ReadTasks 读取任务（用于工作节点）
func (q *RedisQueue) ReadTasks(ctx context.Context, consumerName string, count int64) ([]*TaskMessage, error) {
	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.groupName,
		Consumer: consumerName,
		Streams:  []string{q.streamName, ">"},
		Count:    count,
		Block:    time.Second,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return []*TaskMessage{}, nil // 没有新任务
		}
		return nil, err
	}

	var tasks []*TaskMessage

	for _, stream := range streams {
		for _, message := range stream.Messages {
			taskData, exists := message.Values["data"]
			if !exists {
				continue
			}

			var task TaskMessage
			err = json.Unmarshal([]byte(taskData.(string)), &task)
			if err != nil {
				continue
			}

			// 设置消息 ID 用于确认
			task.ID = message.ID
			tasks = append(tasks, &task)
		}
	}

	return tasks, nil
}

// AckTask 确认任务完成
func (q *RedisQueue) AckTask(ctx context.Context, messageID string) error {
	_, err := q.client.XAck(ctx, q.streamName, q.groupName, messageID).Result()
	return err
}

// GetPendingTasks 获取待处理任务
func (q *RedisQueue) GetPendingTasks(ctx context.Context, consumerName string) ([]*TaskMessage, error) {
	pending, err := q.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: q.streamName,
		Group:  q.groupName,
		Start:  "-",
		End:    "+",
		Count:  100,
	}).Result()

	if err != nil {
		return nil, err
	}

	var tasks []*TaskMessage
	for _, p := range pending {
		// 获取消息详情
		messages, err := q.client.XRange(ctx, q.streamName, p.ID, p.ID).Result()
		if err != nil || len(messages) == 0 {
			continue
		}

		message := messages[0]
		taskData, exists := message.Values["data"]
		if !exists {
			continue
		}

		var task TaskMessage
		err = json.Unmarshal([]byte(taskData.(string)), &task)
		if err != nil {
			continue
		}

		task.ID = message.ID
		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// GetQueueStats 获取队列统计信息
func (q *RedisQueue) GetQueueStats(ctx context.Context) (map[string]any, error) {
	stats := make(map[string]any)

	// Stream 信息
	streamInfo, err := q.client.XInfoStream(ctx, q.streamName).Result()
	if err != nil {
		return nil, err
	}

	stats["stream_length"] = streamInfo.Length
	stats["stream_name"] = q.streamName

	// Consumer Group 信息
	groupInfo, err := q.client.XInfoGroups(ctx, q.streamName).Result()
	if err != nil {
		return nil, err
	}

	var totalPending int64 = 0
	var consumerCount int64 = 0

	for _, group := range groupInfo {
		if group.Name == q.groupName {
			totalPending = group.Pending

			// 获取消费者信息
			consumers, err := q.client.XInfoConsumers(ctx, q.streamName, q.groupName).Result()
			if err == nil {
				consumerCount = int64(len(consumers))
			}
			break
		}
	}

	stats["pending_tasks"] = totalPending
	stats["consumer_count"] = consumerCount
	stats["group_name"] = q.groupName

	return stats, nil
}

// Close 关闭队列连接
func (q *RedisQueue) Close() error {
	return q.client.Close()
}

// CleanupOldTasks 清理旧任务（定期维护）
func (q *RedisQueue) CleanupOldTasks(ctx context.Context, maxAge time.Duration) error {
	// 删除超过指定时间的已确认消息
	cutoff := time.Now().Add(-maxAge).UnixMilli()
	_, err := q.client.XTrimMaxLenApprox(ctx, q.streamName, int64(cutoff), 100).Result()
	return err
}
