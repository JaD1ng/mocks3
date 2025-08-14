package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// MetadataClient 元数据服务客户端
type MetadataClient struct {
	baseURL    string
	httpClient *http.Client
}

// StorageClient 存储服务客户端
type StorageClient struct {
	baseURL    string
	httpClient *http.Client
}

// TaskClient 任务服务客户端
type TaskClient struct {
	baseURL    string
	httpClient *http.Client
}

// Clients 所有客户端
type Clients struct {
	Metadata *MetadataClient
	Storage  *StorageClient
	Task     *TaskClient
}

// NewMetadataClient 创建元数据客户端
func NewMetadataClient(baseURL string) *MetadataClient {
	return &MetadataClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// NewStorageClient 创建存储客户端
func NewStorageClient(baseURL string) *StorageClient {
	return &StorageClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewTaskClient 创建任务客户端
func NewTaskClient(baseURL string) *TaskClient {
	return &TaskClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FileObject 文件对象
type FileObject struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Bucket      string    `json:"bucket"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	MD5Hash     string    `json:"md5_hash"`
	Data        []byte    `json:"data,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// MetadataEntry 元数据条目
type MetadataEntry struct {
	ID           string    `json:"id"`
	Key          string    `json:"key"`
	Bucket       string    `json:"bucket"`
	Size         int64     `json:"size"`
	ContentType  string    `json:"content_type"`
	MD5Hash      string    `json:"md5_hash"`
	StorageNodes []string  `json:"storage_nodes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TaskMessage 任务消息
type TaskMessage struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	ObjectKey string         `json:"object_key"`
	Data      map[string]any `json:"data"`
	CreatedAt time.Time      `json:"created_at"`
}

// 元数据客户端方法

func (c *MetadataClient) GetMetadata(ctx context.Context, key string) (*MetadataEntry, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/metadata/%s", c.baseURL, key))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("metadata not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var entry MetadataEntry
	err = json.NewDecoder(resp.Body).Decode(&entry)
	return &entry, err
}

func (c *MetadataClient) SaveMetadata(ctx context.Context, entry *MetadataEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/metadata", c.baseURL),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

func (c *MetadataClient) DeleteMetadata(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("%s/metadata/%s", c.baseURL, key), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// 存储客户端方法

func (c *StorageClient) WriteObject(ctx context.Context, obj *FileObject) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/storage/write", c.baseURL),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

func (c *StorageClient) ReadObject(ctx context.Context, key string) (*FileObject, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/storage/read/%s", c.baseURL, key))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("object not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var obj FileObject
	err = json.NewDecoder(resp.Body).Decode(&obj)
	return &obj, err
}

func (c *StorageClient) DeleteObject(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("%s/storage/delete/%s", c.baseURL, key), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// 任务客户端方法

func (c *TaskClient) SubmitTask(ctx context.Context, task *TaskMessage) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/tasks", c.baseURL),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
