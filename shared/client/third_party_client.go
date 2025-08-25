package client

import (
	"context"
	"fmt"
	"mocks3/shared/models"
	"net/http"
	"time"
)

// ThirdPartyClient 第三方服务客户端
type ThirdPartyClient struct {
	*BaseHTTPClient
}

// NewThirdPartyClient 创建第三方服务客户端
func NewThirdPartyClient(baseURL string, timeout time.Duration) *ThirdPartyClient {
	return &ThirdPartyClient{
		BaseHTTPClient: NewBaseHTTPClient(baseURL, timeout),
	}
}

// GetObject 获取对象
func (c *ThirdPartyClient) GetObject(ctx context.Context, bucket, key string) (*models.Object, error) {
	path := fmt.Sprintf("/objects/%s/%s", PathEscape(bucket), PathEscape(key))
	var object models.Object
	err := c.Get(ctx, path, nil, &object)
	if err != nil {
		return nil, fmt.Errorf("object not found: %s/%s", bucket, key)
	}
	return &object, nil
}

// PutObject 存储对象
func (c *ThirdPartyClient) PutObject(ctx context.Context, object *models.Object) error {
	return c.PostExpectStatus(ctx, "/objects", object, http.StatusCreated, http.StatusOK)
}

// DeleteObject 删除对象
func (c *ThirdPartyClient) DeleteObject(ctx context.Context, bucket, key string) error {
	path := fmt.Sprintf("/objects/%s/%s", PathEscape(bucket), PathEscape(key))
	return c.Delete(ctx, path)
}

// GetObjectMetadata 获取对象元数据
func (c *ThirdPartyClient) GetObjectMetadata(ctx context.Context, bucket, key string) (*models.Metadata, error) {
	path := fmt.Sprintf("/metadata/%s/%s", PathEscape(bucket), PathEscape(key))
	var metadata models.Metadata
	err := c.Get(ctx, path, nil, &metadata)
	if err != nil {
		return nil, fmt.Errorf("metadata not found: %s/%s", bucket, key)
	}
	return &metadata, nil
}

// ListObjects 列出对象
func (c *ThirdPartyClient) ListObjects(ctx context.Context, bucket, prefix string, limit int) ([]*models.Metadata, error) {
	queryParams := BuildQueryParams(map[string]any{
		"bucket": bucket,
		"prefix": prefix,
		"limit":  limit,
	})

	var listResp struct {
		Objects []*models.Metadata `json:"objects"`
		Count   int                `json:"count"`
	}
	err := c.Get(ctx, "/objects", queryParams, &listResp)
	return listResp.Objects, err
}

// SetDataSource 设置数据源
func (c *ThirdPartyClient) SetDataSource(ctx context.Context, name, config string) error {
	reqBody := map[string]any{
		"name":   name,
		"config": config,
	}
	return c.PostExpectStatus(ctx, "/datasources", reqBody, http.StatusCreated, http.StatusOK)
}

// GetDataSources 获取数据源列表
func (c *ThirdPartyClient) GetDataSources(ctx context.Context) ([]models.DataSource, error) {
	var sourcesResp struct {
		DataSources []models.DataSource `json:"datasources"`
	}
	err := c.Get(ctx, "/datasources", nil, &sourcesResp)
	return sourcesResp.DataSources, err
}

// CacheObject 缓存对象
func (c *ThirdPartyClient) CacheObject(ctx context.Context, object *models.Object) error {
	return c.PostExpectStatus(ctx, "/cache", object, http.StatusCreated, http.StatusOK)
}

// InvalidateCache 清除缓存
func (c *ThirdPartyClient) InvalidateCache(ctx context.Context, bucket, key string) error {
	path := fmt.Sprintf("/cache/%s/%s", PathEscape(bucket), PathEscape(key))
	return c.Delete(ctx, path)
}

// GetStats 获取统计信息
func (c *ThirdPartyClient) GetStats(ctx context.Context) (map[string]any, error) {
	var stats map[string]any
	err := c.Get(ctx, "/stats", nil, &stats)
	return stats, err
}

// HealthCheck 健康检查
func (c *ThirdPartyClient) HealthCheck(ctx context.Context) error {
	return c.BaseHTTPClient.HealthCheck(ctx)
}