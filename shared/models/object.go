package models

import (
	"time"
)

// Object 对象模型
type Object struct {
	ID           string            `json:"id" db:"id"`
	Key          string            `json:"key" db:"key"`
	Bucket       string            `json:"bucket" db:"bucket"`
	Size         int64             `json:"size" db:"size"`
	ContentType  string            `json:"content_type" db:"content_type"`
	MD5Hash      string            `json:"md5_hash" db:"md5_hash"`
	ETag         string            `json:"etag" db:"etag"`
	Data         []byte            `json:"-"`                    // 实际数据，不序列化
	Headers      map[string]string `json:"headers,omitempty"`   // HTTP 头信息
	Tags         map[string]string `json:"tags,omitempty"`      // 用户标签
	LastModified time.Time         `json:"last_modified" db:"last_modified"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

// ObjectInfo 对象信息（不包含数据）
type ObjectInfo struct {
	ID          string            `json:"id"`
	Key         string            `json:"key"`
	Bucket      string            `json:"bucket"`
	Size        int64             `json:"size"`
	ContentType string            `json:"content_type"`
	MD5Hash     string            `json:"md5_hash"`
	ETag        string            `json:"etag"`
	Headers     map[string]string `json:"headers,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// UploadRequest 上传请求
type UploadRequest struct {
	Key         string            `json:"key" binding:"required"`
	Bucket      string            `json:"bucket" binding:"required"`
	ContentType string            `json:"content_type"`
	Headers     map[string]string `json:"headers,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Data        []byte            `json:"data"`
}

// UploadResponse 上传响应
type UploadResponse struct {
	Success   bool   `json:"success"`
	ObjectID  string `json:"object_id,omitempty"`
	Key       string `json:"key,omitempty"`
	Bucket    string `json:"bucket,omitempty"`
	Size      int64  `json:"size,omitempty"`
	MD5Hash   string `json:"md5_hash,omitempty"`
	ETag      string `json:"etag,omitempty"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// ListObjectsRequest 列表请求
type ListObjectsRequest struct {
	Bucket     string `json:"bucket" form:"bucket"`
	Prefix     string `json:"prefix" form:"prefix"`
	Delimiter  string `json:"delimiter" form:"delimiter"`
	MaxKeys    int    `json:"max_keys" form:"max_keys"`
	StartAfter string `json:"start_after" form:"start_after"`
}

// ListObjectsResponse 列表响应
type ListObjectsResponse struct {
	Bucket       string       `json:"bucket"`
	Prefix       string       `json:"prefix"`
	Delimiter    string       `json:"delimiter,omitempty"`
	MaxKeys      int          `json:"max_keys"`
	IsTruncated  bool         `json:"is_truncated"`
	NextMarker   string       `json:"next_marker,omitempty"`
	Objects      []ObjectInfo `json:"objects"`
	CommonPrefix []string     `json:"common_prefixes,omitempty"`
	Count        int          `json:"count"`
}

// SearchObjectsRequest 搜索请求
type SearchObjectsRequest struct {
	Query  string `json:"query" form:"q" binding:"required"`
	Bucket string `json:"bucket" form:"bucket"`
	Limit  int    `json:"limit" form:"limit"`
	Offset int    `json:"offset" form:"offset"`
}

// SearchObjectsResponse 搜索响应
type SearchObjectsResponse struct {
	Query   string       `json:"query"`
	Objects []ObjectInfo `json:"objects"`
	Total   int64        `json:"total"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
}