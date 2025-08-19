package models

import "time"

// DataSource 数据源模型
type DataSource struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"` // s3, azure, gcs, http等
	Config    map[string]string `json:"config"`
	Enabled   bool              `json:"enabled"`
	Priority  int               `json:"priority"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// DataSourceConfig 数据源配置
type DataSourceConfig struct {
	Endpoint   string `json:"endpoint,omitempty"`
	AccessKey  string `json:"access_key,omitempty"`
	SecretKey  string `json:"secret_key,omitempty"`
	Region     string `json:"region,omitempty"`
	BucketName string `json:"bucket_name,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
}

// CachePolicy 缓存策略
type CachePolicy struct {
	TTL      time.Duration `json:"ttl"`
	MaxSize  int64         `json:"max_size"`
	Strategy string        `json:"strategy"` // lru, lfu, fifo
	Enabled  bool          `json:"enabled"`
}
