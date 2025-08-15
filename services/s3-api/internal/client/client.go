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
// baseURL: 元数据服务的基础URL
// 返回: 初始化后的元数据客户端，超时时间为5秒
func NewMetadataClient(baseURL string) *MetadataClient {
	return &MetadataClient{
		baseURL: baseURL,
		// 设置5秒超时，适合元数据查询操作
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// NewStorageClient 创建存储客户端
// baseURL: 存储服务的基础URL
// 返回: 初始化后的存储客户端，超时时间为30秒
func NewStorageClient(baseURL string) *StorageClient {
	return &StorageClient{
		baseURL: baseURL,
		// 设置30秒超时，适合大文件上传下载操作
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewTaskClient 创建任务客户端
// baseURL: 任务服务的基础URL
// 返回: 初始化后的任务客户端，超时时间为10秒
func NewTaskClient(baseURL string) *TaskClient {
	return &TaskClient{
		baseURL: baseURL,
		// 设置10秒超时，适合任务提交操作
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

// GetMetadata 获取指定键的元数据
// ctx: 请求上下文，用于控制请求超时和取消
// key: 对象键名
// 返回: 元数据条目和错误信息，如果找不到返回错误
func (c *MetadataClient) GetMetadata(ctx context.Context, key string) (*MetadataEntry, error) {
	// 发送GET请求获取元数据
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/metadata/%s", c.baseURL, key))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 处理404状态码，表示元数据不存在
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("metadata not found")
	}

	// 检查其他非成功状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// 解析JSON响应为元数据结构
	var entry MetadataEntry
	err = json.NewDecoder(resp.Body).Decode(&entry)
	return &entry, err
}

// SaveMetadata 保存元数据条目
// ctx: 请求上下文，用于控制请求超时和取消
// entry: 要保存的元数据条目
// 返回: 错误信息，成功时为nil
func (c *MetadataClient) SaveMetadata(ctx context.Context, entry *MetadataEntry) error {
	// 将元数据条目序列化为JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// 发送POST请求保存元数据
	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/metadata", c.baseURL),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，非2xx视为错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// DeleteMetadata 删除指定键的元数据
// ctx: 请求上下文，用于控制请求超时和取消
// key: 要删除的对象键名
// 返回: 错误信息，成功时为nil
func (c *MetadataClient) DeleteMetadata(ctx context.Context, key string) error {
	// 创建DELETE请求，支持上下文取消
	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("%s/metadata/%s", c.baseURL, key), nil)
	if err != nil {
		return err
	}

	// 执行删除请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，非2xx视为错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// 存储客户端方法

// WriteObject 写入文件对象到存储
// ctx: 请求上下文，用于控制请求超时和取消
// obj: 要存储的文件对象，包含文件数据和元信息
// 返回: 错误信息，成功时为nil
func (c *StorageClient) WriteObject(ctx context.Context, obj *FileObject) error {
	// 将文件对象序列化为JSON
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	// 发送POST请求写入文件
	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/storage/write", c.baseURL),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，非2xx视为错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// ReadObject 从存储读取文件对象
// ctx: 请求上下文，用于控制请求超时和取消
// key: 要读取的文件键名
// 返回: 文件对象和错误信息，如果找不到文件返回错误
func (c *StorageClient) ReadObject(ctx context.Context, key string) (*FileObject, error) {
	// 发送GET请求读取文件
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/storage/read/%s", c.baseURL, key))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 处理404状态码，表示文件不存在
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("object not found")
	}

	// 检查其他非成功状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// 解析JSON响应为文件对象
	var obj FileObject
	err = json.NewDecoder(resp.Body).Decode(&obj)
	return &obj, err
}

// DeleteObject 从存储删除文件对象
// ctx: 请求上下文，用于控制请求超时和取消
// key: 要删除的文件键名
// 返回: 错误信息，成功时为nil
func (c *StorageClient) DeleteObject(ctx context.Context, key string) error {
	// 创建DELETE请求，支持上下文取消
	req, err := http.NewRequestWithContext(ctx, "DELETE",
		fmt.Sprintf("%s/storage/delete/%s", c.baseURL, key), nil)
	if err != nil {
		return err
	}

	// 执行删除请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，非2xx视为错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// 任务客户端方法

// SubmitTask 提交任务到任务队列
// ctx: 请求上下文，用于控制请求超时和取消
// task: 要提交的任务消息，包含任务类型和数据
// 返回: 错误信息，成功时为nil
func (c *TaskClient) SubmitTask(ctx context.Context, task *TaskMessage) error {
	// 将任务消息序列化为JSON
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}

	// 发送POST请求提交任务
	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/tasks", c.baseURL),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP状态码，非2xx视为错误，包含详细错误信息
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
