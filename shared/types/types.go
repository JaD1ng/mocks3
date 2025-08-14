package types

import "time"

// FileObject 文件对象
type FileObject struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Bucket      string    `json:"bucket"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	MD5Hash     string    `json:"md5_hash"`
	Data        []byte    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MetadataEntry 元数据条目
type MetadataEntry struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	Key          string    `json:"key" gorm:"uniqueIndex"`
	Bucket       string    `json:"bucket"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
	MD5Hash      string    `json:"md5_hash"`
	StorageNodes []string  `json:"storage_nodes" gorm:"serializer:json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UploadRequest 上传请求
type UploadRequest struct {
	Key         string            `json:"key"`
	Bucket      string            `json:"bucket"`
	ContentType string            `json:"content_type"`
	Size        int64             `json:"size"`
	MD5Hash     string            `json:"md5_hash"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// UploadResponse 上传响应
type UploadResponse struct {
	Success  bool   `json:"success"`
	ObjectID string `json:"object_id,omitempty"`
	Key      string `json:"key,omitempty"`
	Bucket   string `json:"bucket,omitempty"`
	Message  string `json:"message,omitempty"`
	Size     int64  `json:"size,omitempty"`
	MD5Hash  string `json:"md5_hash,omitempty"`
	ETag     string `json:"etag,omitempty"`
}

// TaskMessage 队列任务消息
type TaskMessage struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	ObjectKey  string         `json:"object_key"`
	Data       map[string]any `json:"data"`
	CreatedAt  time.Time      `json:"created_at"`
	RetryCount int            `json:"retry_count"`
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	Address string `json:"address"`
	Port    int    `json:"port"`
	Health  string `json:"health"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HealthCheck 健康检查响应
type HealthCheck struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
}

// StorageNode 存储节点
type StorageNode struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Status   string `json:"status"`
	Capacity int64  `json:"capacity"`
	Used     int64  `json:"used"`
}

// ChaosRule 混沌工程规则
type ChaosRule struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Service     string         `json:"service"`
	Endpoint    string         `json:"endpoint,omitempty"`
	FailureType string         `json:"failure_type"`
	FailureRate float64        `json:"failure_rate"`
	Duration    string         `json:"duration,omitempty"`
	Enabled     bool           `json:"enabled"`
	Config      map[string]any `json:"config,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}
