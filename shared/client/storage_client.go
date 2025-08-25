package client

import (
	"context"
	"fmt"
	"io"
	"mocks3/shared/models"
	"net/http"
	"time"
)

// StorageClient 存储服务客户端
type StorageClient struct {
	*BaseHTTPClient
}

// NewStorageClient 创建存储服务客户端
func NewStorageClient(baseURL string, timeout time.Duration) *StorageClient {
	return &StorageClient{
		BaseHTTPClient: NewBaseHTTPClient(baseURL, timeout),
	}
}

// WriteObject 写入对象
func (c *StorageClient) WriteObject(ctx context.Context, object *models.Object) error {
	req := &models.UploadRequest{
		Key:         object.Key,
		Bucket:      object.Bucket,
		ContentType: object.ContentType,
		Data:        object.Data,
	}

	var uploadResp models.UploadResponse
	if err := c.Post(ctx, "/objects", req, &uploadResp); err != nil {
		return err
	}

	if !uploadResp.Success {
		return fmt.Errorf("upload failed: %s", uploadResp.Message)
	}

	// 更新对象信息
	object.ID = uploadResp.ObjectID
	object.MD5Hash = uploadResp.MD5Hash

	return nil
}

// ReadObject 读取对象
func (c *StorageClient) ReadObject(ctx context.Context, bucket, key string) (*models.Object, error) {
	path := fmt.Sprintf("/objects/%s/%s", PathEscape(bucket), PathEscape(key))

	resp, err := c.DoRequest(ctx, RequestOptions{
		Method: "GET",
		Path:   path,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("object not found: %s/%s", bucket, key)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	object := &models.Object{
		Key:         key,
		Bucket:      bucket,
		Data:        data,
		Size:        int64(len(data)),
		ContentType: resp.Header.Get("Content-Type"),
		MD5Hash:     resp.Header.Get("Content-MD5"),
	}

	return object, nil
}

// DeleteObject 删除对象
func (c *StorageClient) DeleteObject(ctx context.Context, bucket, key string) error {
	path := fmt.Sprintf("/objects/%s/%s", PathEscape(bucket), PathEscape(key))
	return c.Delete(ctx, path)
}

// ListObjects 列出对象
func (c *StorageClient) ListObjects(ctx context.Context, req *models.ListObjectsRequest) (*models.ListObjectsResponse, error) {
	queryParams := BuildQueryParams(map[string]any{
		"bucket":      req.Bucket,
		"prefix":      req.Prefix,
		"delimiter":   req.Delimiter,
		"max_keys":    req.MaxKeys,
		"start_after": req.StartAfter,
	})

	var listResp models.ListObjectsResponse
	err := c.Get(ctx, "/objects", queryParams, &listResp)
	return &listResp, err
}

// HealthCheck 健康检查
func (c *StorageClient) HealthCheck(ctx context.Context) error {
	return c.BaseHTTPClient.HealthCheck(ctx)
}