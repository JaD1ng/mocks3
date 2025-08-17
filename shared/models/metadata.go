package models

import (
	"time"
)

// Metadata 元数据模型
type Metadata struct {
	ID           string            `json:"id" db:"id"`
	Key          string            `json:"key" db:"key"`
	Bucket       string            `json:"bucket" db:"bucket"`
	Size         int64             `json:"size" db:"size"`
	ContentType  string            `json:"content_type" db:"content_type"`
	MD5Hash      string            `json:"md5_hash" db:"md5_hash"`
	ETag         string            `json:"etag" db:"etag"`
	StorageNodes []string          `json:"storage_nodes" db:"storage_nodes"` // JSON 存储
	Headers      map[string]string `json:"headers" db:"headers"`             // JSON 存储
	Tags         map[string]string `json:"tags" db:"tags"`                   // JSON 存储
	Status       string            `json:"status" db:"status"`               // active, deleted, corrupted
	Version      int64             `json:"version" db:"version"`
	LastModified time.Time         `json:"last_modified" db:"last_modified"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time        `json:"deleted_at,omitempty" db:"deleted_at"`
}

// MetadataFilter 元数据过滤器
type MetadataFilter struct {
	Bucket      string            `json:"bucket,omitempty"`
	Prefix      string            `json:"prefix,omitempty"`
	Status      string            `json:"status,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	SizeMin     *int64            `json:"size_min,omitempty"`
	SizeMax     *int64            `json:"size_max,omitempty"`
	CreatedFrom *time.Time        `json:"created_from,omitempty"`
	CreatedTo   *time.Time        `json:"created_to,omitempty"`
}

// Stats 统计信息
type Stats struct {
	TotalObjects   int64             `json:"total_objects"`
	TotalSize      int64             `json:"total_size"`
	AverageSize    float64           `json:"average_size"`
	BucketStats    map[string]int64  `json:"bucket_stats"`
	ContentTypes   map[string]int64  `json:"content_types"`
	StorageNodes   map[string]int64  `json:"storage_nodes"`
	StatusCounts   map[string]int64  `json:"status_counts"`
	DailyUploads   []DailyUploadStat `json:"daily_uploads"`
	LastUpdated    time.Time         `json:"last_updated"`
}

// DailyUploadStat 每日上传统计
type DailyUploadStat struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
	Size  int64  `json:"size"`
}

// MetadataBackup 元数据备份
type MetadataBackup struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
	Size      int64     `json:"size"`
	FilePath  string    `json:"file_path"`
	Status    string    `json:"status"` // creating, completed, failed
	CreatedBy string    `json:"created_by"`
}

// MetadataSyncEvent 元数据同步事件
type MetadataSyncEvent struct {
	EventID     string            `json:"event_id"`
	EventType   string            `json:"event_type"` // create, update, delete
	ObjectKey   string            `json:"object_key"`
	Metadata    *Metadata         `json:"metadata,omitempty"`
	Changes     map[string]string `json:"changes,omitempty"` // field -> old_value
	Timestamp   time.Time         `json:"timestamp"`
	SourceNode  string            `json:"source_node"`
	ReplicatedTo []string         `json:"replicated_to"`
}