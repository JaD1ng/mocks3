package models

import (
	"fmt"
	"time"
)

// Task 任务模型
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`        // task type
	Queue       string                 `json:"queue"`       // queue name
	ObjectKey   string                 `json:"object_key"`  // related object key
	Data        map[string]interface{} `json:"data"`        // task payload
	Priority    int                    `json:"priority"`    // task priority (higher number = higher priority)
	MaxRetries  int                    `json:"max_retries"` // maximum retry attempts
	RetryCount  int                    `json:"retry_count"` // current retry count
	Status      TaskStatus             `json:"status"`      // task status
	ScheduledAt time.Time              `json:"scheduled_at"` // when to execute
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	FailedAt    *time.Time             `json:"failed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	WorkerID    string                 `json:"worker_id,omitempty"`
	StreamID    string                 `json:"stream_id,omitempty"`  // Redis stream message ID
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// GenerateID 生成任务ID
func (t *Task) GenerateID() {
	if t.ID == "" {
		t.ID = generateTaskID()
	}
}

// generateTaskID 生成随机任务ID
func generateTaskID() string {
	// 简单的ID生成实现
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusRetrying   TaskStatus = "retrying"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// TaskType 任务类型
const (
	TaskTypeDeleteFile      = "delete_file"
	TaskTypeReplicateFile   = "replicate_file"
	TaskTypeCleanupTemp     = "cleanup_temp"
	TaskTypeGenerateThumbnail = "generate_thumbnail"
	TaskTypeBackupMetadata  = "backup_metadata"
	TaskTypeSyncMetadata    = "sync_metadata"
	TaskTypeHealthCheck     = "health_check"
)

// QueueConfig 队列配置
type QueueConfig struct {
	Name            string        `json:"name"`
	MaxLength       int64         `json:"max_length"`
	MaxConsumers    int           `json:"max_consumers"`
	VisibilityTimeout time.Duration `json:"visibility_timeout"`
	RetentionPeriod time.Duration `json:"retention_period"`
	DeadLetterQueue string        `json:"dead_letter_queue,omitempty"`
	Priority        bool          `json:"priority"` // whether queue supports priority
}

// QueueStats 队列统计
type QueueStats struct {
	QueueName       string    `json:"queue_name"`
	Length          int64     `json:"length"`
	ConsumerCount   int       `json:"consumer_count"`
	ProcessingCount int64     `json:"processing_count"`
	CompletedCount  int64     `json:"completed_count"`
	FailedCount     int64     `json:"failed_count"`
	LastMessage     time.Time `json:"last_message"`
	ThroughputPerSecond float64 `json:"throughput_per_second"`
}

// Worker 工作节点模型
type Worker struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Queues      []string          `json:"queues"`
	Status      WorkerStatus      `json:"status"`
	Metadata    map[string]string `json:"metadata"`
	LastSeen    time.Time         `json:"last_seen"`
	StartedAt   time.Time         `json:"started_at"`
	TasksRun    int64             `json:"tasks_run"`
	TasksFailed int64             `json:"tasks_failed"`
	CurrentTask *Task             `json:"current_task,omitempty"`
}

// WorkerStatus 工作节点状态
type WorkerStatus string

const (
	WorkerStatusIdle    WorkerStatus = "idle"
	WorkerStatusRunning WorkerStatus = "running"
	WorkerStatusStopped WorkerStatus = "stopped"
	WorkerStatusError   WorkerStatus = "error"
)

// TaskMessage 任务消息（用于队列传输）
type TaskMessage struct {
	Task      *Task     `json:"task"`
	Timestamp time.Time `json:"timestamp"`
	Attempts  int       `json:"attempts"`
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID    string                 `json:"task_id"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Duration  time.Duration          `json:"duration"`
	WorkerID  string                 `json:"worker_id"`
	Timestamp time.Time              `json:"timestamp"`
}