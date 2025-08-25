package client

import (
	"context"
	"fmt"
	"mocks3/shared/models"
	"net/http"
	"time"
)

// MetadataClient 元数据服务客户端
type MetadataClient struct {
	*BaseHTTPClient
}

// NewMetadataClient 创建元数据服务客户端
func NewMetadataClient(baseURL string, timeout time.Duration) *MetadataClient {
	return &MetadataClient{
		BaseHTTPClient: NewBaseHTTPClient(baseURL, timeout),
	}
}

// SaveMetadata 保存元数据
func (c *MetadataClient) SaveMetadata(ctx context.Context, metadata *models.Metadata) error {
	return c.PostExpectStatus(ctx, "/api/v1/metadata", metadata, http.StatusCreated)
}

// GetMetadata 获取元数据
func (c *MetadataClient) GetMetadata(ctx context.Context, bucket, key string) (*models.Metadata, error) {
	path := fmt.Sprintf("/api/v1/metadata/%s/%s", PathEscape(bucket), PathEscape(key))
	var metadata models.Metadata
	err := c.Get(ctx, path, nil, &metadata)
	if err != nil {
		return nil, fmt.Errorf("metadata not found: %s/%s", bucket, key)
	}
	return &metadata, nil
}

// UpdateMetadata 更新元数据
func (c *MetadataClient) UpdateMetadata(ctx context.Context, metadata *models.Metadata) error {
	path := fmt.Sprintf("/api/v1/metadata/%s/%s", PathEscape(metadata.Bucket), PathEscape(metadata.Key))
	return c.PutExpectStatus(ctx, path, metadata, http.StatusOK)
}

// DeleteMetadata 删除元数据
func (c *MetadataClient) DeleteMetadata(ctx context.Context, bucket, key string) error {
	path := fmt.Sprintf("/api/v1/metadata/%s/%s", PathEscape(bucket), PathEscape(key))
	return c.Delete(ctx, path)
}

// ListMetadata 列出元数据
func (c *MetadataClient) ListMetadata(ctx context.Context, bucket, prefix string, limit, offset int) ([]*models.Metadata, error) {
	queryParams := BuildQueryParams(map[string]any{
		"bucket": bucket,
		"prefix": prefix,
		"limit":  limit,
		"offset": offset,
	})

	var metadataList []*models.Metadata
	err := c.Get(ctx, "/api/v1/metadata", queryParams, &metadataList)
	return metadataList, err
}

// SearchMetadata 搜索元数据
func (c *MetadataClient) SearchMetadata(ctx context.Context, req *models.SearchObjectsRequest) (*models.SearchObjectsResponse, error) {
	queryParams := BuildQueryParams(map[string]any{
		"q":      req.Query,
		"bucket": req.Bucket,
		"limit":  req.Limit,
		"offset": req.Offset,
	})

	var searchResp models.SearchObjectsResponse
	err := c.Get(ctx, "/api/v1/metadata/search", queryParams, &searchResp)
	return &searchResp, err
}

// GetStats 获取统计信息
func (c *MetadataClient) GetStats(ctx context.Context) (*models.Stats, error) {
	var stats models.Stats
	err := c.Get(ctx, "/api/v1/stats", nil, &stats)
	return &stats, err
}

// CountObjects 计算对象数量
func (c *MetadataClient) CountObjects(ctx context.Context, bucket, prefix string) (int64, error) {
	queryParams := BuildQueryParams(map[string]any{
		"bucket": bucket,
		"prefix": prefix,
	})

	var countResp struct {
		Count int64 `json:"count"`
	}
	err := c.Get(ctx, "/api/v1/metadata/count", queryParams, &countResp)
	return countResp.Count, err
}

// HealthCheck 健康检查
func (c *MetadataClient) HealthCheck(ctx context.Context) error {
	return c.BaseHTTPClient.HealthCheck(ctx)
}
