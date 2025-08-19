package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mocks3/services/third-party/internal/repository"
	"mocks3/shared/interfaces"
	"mocks3/shared/models"
	"mocks3/shared/observability/log"
	"net/http"
	"strings"
	"time"
)

// ThirdPartyService 第三方服务实现
type ThirdPartyService struct {
	dataSourceRepo *repository.DataSourceRepository
	cacheRepo      *repository.CacheRepository
	logger         *log.Logger
	httpClient     *http.Client
}

// NewThirdPartyService 创建第三方服务
func NewThirdPartyService(
	dataSourceRepo *repository.DataSourceRepository,
	cacheRepo *repository.CacheRepository,
	logger *log.Logger,
) *ThirdPartyService {
	return &ThirdPartyService{
		dataSourceRepo: dataSourceRepo,
		cacheRepo:      cacheRepo,
		logger:         logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetObject 获取对象
func (s *ThirdPartyService) GetObject(ctx context.Context, bucket, key string) (*models.Object, error) {
	s.logger.InfoContext(ctx, "Getting object from third-party sources", "bucket", bucket, "key", key)

	// 1. 首先尝试从缓存获取
	if cachedObj, err := s.cacheRepo.Get(ctx, bucket, key); err == nil {
		s.logger.InfoContext(ctx, "Object found in cache", "bucket", bucket, "key", key)
		return cachedObj, nil
	}

	// 2. 按优先级从数据源获取
	dataSources, err := s.dataSourceRepo.GetByPriority(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get data sources: %w", err)
	}

	for _, ds := range dataSources {
		s.logger.InfoContext(ctx, "Trying data source", "source", ds.Name, "type", ds.Type)

		object, err := s.fetchFromDataSource(ctx, ds, bucket, key)
		if err != nil {
			s.logger.WarnContext(ctx, "Failed to fetch from data source",
				"source", ds.Name, "error", err)
			continue
		}

		// 成功获取，缓存对象
		if cacheErr := s.cacheRepo.Set(ctx, bucket, key, object); cacheErr != nil {
			s.logger.WarnContext(ctx, "Failed to cache object", "error", cacheErr)
		}

		s.logger.InfoContext(ctx, "Object retrieved successfully",
			"bucket", bucket, "key", key, "source", ds.Name)
		return object, nil
	}

	return nil, fmt.Errorf("object not found in any data source: %s/%s", bucket, key)
}

// PutObject 存储对象
func (s *ThirdPartyService) PutObject(ctx context.Context, object *models.Object) error {
	s.logger.InfoContext(ctx, "Storing object to third-party sources",
		"bucket", object.Bucket, "key", object.Key)

	// 获取所有可用数据源
	dataSources, err := s.dataSourceRepo.GetByPriority(ctx)
	if err != nil {
		return fmt.Errorf("failed to get data sources: %w", err)
	}

	// 存储到第一个可用的数据源
	for _, ds := range dataSources {
		if err := s.putToDataSource(ctx, ds, object); err != nil {
			s.logger.WarnContext(ctx, "Failed to store to data source",
				"source", ds.Name, "error", err)
			continue
		}

		// 存储成功，同时缓存
		if cacheErr := s.cacheRepo.Set(ctx, object.Bucket, object.Key, object); cacheErr != nil {
			s.logger.WarnContext(ctx, "Failed to cache object", "error", cacheErr)
		}

		s.logger.InfoContext(ctx, "Object stored successfully",
			"bucket", object.Bucket, "key", object.Key, "source", ds.Name)
		return nil
	}

	return fmt.Errorf("failed to store object to any data source")
}

// DeleteObject 删除对象
func (s *ThirdPartyService) DeleteObject(ctx context.Context, bucket, key string) error {
	s.logger.InfoContext(ctx, "Deleting object from third-party sources", "bucket", bucket, "key", key)

	// 从缓存中删除
	s.cacheRepo.Delete(ctx, bucket, key)

	// 从所有数据源删除
	dataSources, err := s.dataSourceRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get data sources: %w", err)
	}

	var lastErr error
	successCount := 0

	for _, ds := range dataSources {
		if err := s.deleteFromDataSource(ctx, &ds, bucket, key); err != nil {
			s.logger.WarnContext(ctx, "Failed to delete from data source",
				"source", ds.Name, "error", err)
			lastErr = err
		} else {
			successCount++
		}
	}

	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to delete from any data source: %w", lastErr)
	}

	s.logger.InfoContext(ctx, "Object deletion completed",
		"bucket", bucket, "key", key, "success_count", successCount)
	return nil
}

// GetObjectMetadata 获取对象元数据
func (s *ThirdPartyService) GetObjectMetadata(ctx context.Context, bucket, key string) (*models.Metadata, error) {
	s.logger.DebugContext(ctx, "Getting object metadata", "bucket", bucket, "key", key)

	// 尝试从对象获取元数据
	object, err := s.GetObject(ctx, bucket, key)
	if err != nil {
		return nil, err
	}

	// 从对象构建元数据
	metadata := &models.Metadata{
		Bucket:       object.Bucket,
		Key:          object.Key,
		Size:         object.Size,
		ETag:         object.ETag,
		ContentType:  object.ContentType,
		LastModified: object.LastModified,
		Headers:      make(map[string]string),
		Tags:         make(map[string]string),
		Status:       "active",
		CreatedAt:    object.LastModified,
		UpdatedAt:    time.Now(),
	}

	// 添加第三方来源标记
	metadata.Tags["source"] = "third-party"

	return metadata, nil
}

// ListObjects 列出对象
func (s *ThirdPartyService) ListObjects(ctx context.Context, bucket, prefix string, limit int) ([]*models.Metadata, error) {
	s.logger.DebugContext(ctx, "Listing objects", "bucket", bucket, "prefix", prefix, "limit", limit)

	// 这里简化实现，实际应该查询数据源
	// 目前返回空列表，表示第三方服务不支持列表操作
	return []*models.Metadata{}, nil
}

// SetDataSource 设置数据源
func (s *ThirdPartyService) SetDataSource(ctx context.Context, name, config string) error {
	s.logger.InfoContext(ctx, "Setting data source", "name", name)

	// 解析配置 (简化实现)
	dataSource := &models.DataSource{
		Name:     name,
		Type:     "custom",
		Config:   map[string]string{"config": config},
		Enabled:  true,
		Priority: 100,
	}

	return s.dataSourceRepo.Add(ctx, dataSource)
}

// GetDataSources 获取数据源列表
func (s *ThirdPartyService) GetDataSources(ctx context.Context) ([]models.DataSource, error) {
	s.logger.DebugContext(ctx, "Getting data sources")
	return s.dataSourceRepo.GetAll(ctx)
}

// CacheObject 缓存对象
func (s *ThirdPartyService) CacheObject(ctx context.Context, object *models.Object) error {
	s.logger.DebugContext(ctx, "Caching object", "bucket", object.Bucket, "key", object.Key)
	return s.cacheRepo.Set(ctx, object.Bucket, object.Key, object)
}

// InvalidateCache 清除缓存
func (s *ThirdPartyService) InvalidateCache(ctx context.Context, bucket, key string) error {
	s.logger.DebugContext(ctx, "Invalidating cache", "bucket", bucket, "key", key)
	return s.cacheRepo.Delete(ctx, bucket, key)
}

// GetStats 获取统计信息
func (s *ThirdPartyService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	s.logger.DebugContext(ctx, "Getting statistics")

	cacheStats := s.cacheRepo.GetStats()
	dataSources, _ := s.dataSourceRepo.GetAll(ctx)

	stats := map[string]interface{}{
		"cache": map[string]interface{}{
			"hits":         cacheStats.Hits,
			"misses":       cacheStats.Misses,
			"evictions":    cacheStats.Evictions,
			"total_size":   cacheStats.TotalSize,
			"item_count":   cacheStats.ItemCount,
			"last_cleanup": cacheStats.LastCleanup,
		},
		"data_sources": map[string]interface{}{
			"count":   len(dataSources),
			"sources": dataSources,
		},
	}

	return stats, nil
}

// HealthCheck 健康检查
func (s *ThirdPartyService) HealthCheck(ctx context.Context) error {
	s.logger.DebugContext(ctx, "Performing health check")

	// 检查数据源
	dataSources, err := s.dataSourceRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get data sources: %w", err)
	}

	if len(dataSources) == 0 {
		return fmt.Errorf("no data sources configured")
	}

	return nil
}

// fetchFromDataSource 从数据源获取对象
func (s *ThirdPartyService) fetchFromDataSource(ctx context.Context, ds *models.DataSource, bucket, key string) (*models.Object, error) {
	switch ds.Type {
	case "s3":
		return s.fetchFromS3(ctx, ds, bucket, key)
	case "http":
		return s.fetchFromHTTP(ctx, ds, bucket, key)
	default:
		return nil, fmt.Errorf("unsupported data source type: %s", ds.Type)
	}
}

// fetchFromS3 从S3兼容数据源获取
func (s *ThirdPartyService) fetchFromS3(ctx context.Context, ds *models.DataSource, bucket, key string) (*models.Object, error) {
	// 模拟S3访问，实际实现需要AWS SDK
	s.logger.DebugContext(ctx, "Fetching from S3 data source", "source", ds.Name)

	// 模拟数据
	object := &models.Object{
		Bucket:       bucket,
		Key:          key,
		Size:         1024,
		ETag:         fmt.Sprintf("%x", time.Now().Unix()),
		ContentType:  "application/octet-stream",
		Data:         []byte("mock data from S3 source"),
		LastModified: time.Now(),
	}

	return object, nil
}

// fetchFromHTTP 从HTTP数据源获取
func (s *ThirdPartyService) fetchFromHTTP(ctx context.Context, ds *models.DataSource, bucket, key string) (*models.Object, error) {
	s.logger.DebugContext(ctx, "Fetching from HTTP data source", "source", ds.Name)

	endpoint := ds.Config["endpoint"]
	if endpoint == "" {
		return nil, fmt.Errorf("no endpoint configured for HTTP data source")
	}

	url := fmt.Sprintf("%s/objects/%s/%s", strings.TrimRight(endpoint, "/"), bucket, key)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("object not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	object := &models.Object{
		Bucket:       bucket,
		Key:          key,
		Size:         int64(len(data)),
		ETag:         resp.Header.Get("ETag"),
		ContentType:  resp.Header.Get("Content-Type"),
		Data:         data,
		LastModified: time.Now(),
	}

	return object, nil
}

// putToDataSource 存储到数据源
func (s *ThirdPartyService) putToDataSource(ctx context.Context, ds *models.DataSource, object *models.Object) error {
	switch ds.Type {
	case "s3":
		return s.putToS3(ctx, ds, object)
	case "http":
		return s.putToHTTP(ctx, ds, object)
	default:
		return fmt.Errorf("unsupported data source type: %s", ds.Type)
	}
}

// putToS3 存储到S3
func (s *ThirdPartyService) putToS3(ctx context.Context, ds *models.DataSource, object *models.Object) error {
	s.logger.DebugContext(ctx, "Storing to S3 data source", "source", ds.Name)
	// 模拟存储成功
	return nil
}

// putToHTTP 存储到HTTP
func (s *ThirdPartyService) putToHTTP(ctx context.Context, ds *models.DataSource, object *models.Object) error {
	s.logger.DebugContext(ctx, "Storing to HTTP data source", "source", ds.Name)

	endpoint := ds.Config["endpoint"]
	if endpoint == "" {
		return fmt.Errorf("no endpoint configured for HTTP data source")
	}

	url := fmt.Sprintf("%s/objects", strings.TrimRight(endpoint, "/"))

	// 编码数据
	data := base64.StdEncoding.EncodeToString(object.Data)
	payload := map[string]interface{}{
		"bucket":       object.Bucket,
		"key":          object.Key,
		"content_type": object.ContentType,
		"data":         data,
	}

	// 这里应该发送HTTP请求，简化实现
	_ = payload
	_ = url

	return nil
}

// deleteFromDataSource 从数据源删除
func (s *ThirdPartyService) deleteFromDataSource(ctx context.Context, ds *models.DataSource, bucket, key string) error {
	switch ds.Type {
	case "s3":
		return s.deleteFromS3(ctx, ds, bucket, key)
	case "http":
		return s.deleteFromHTTP(ctx, ds, bucket, key)
	default:
		return fmt.Errorf("unsupported data source type: %s", ds.Type)
	}
}

// deleteFromS3 从S3删除
func (s *ThirdPartyService) deleteFromS3(ctx context.Context, ds *models.DataSource, bucket, key string) error {
	s.logger.DebugContext(ctx, "Deleting from S3 data source", "source", ds.Name)
	// 模拟删除成功
	return nil
}

// deleteFromHTTP 从HTTP删除
func (s *ThirdPartyService) deleteFromHTTP(ctx context.Context, ds *models.DataSource, bucket, key string) error {
	s.logger.DebugContext(ctx, "Deleting from HTTP data source", "source", ds.Name)
	// 模拟删除成功
	return nil
}

// 确保实现了接口
var _ interfaces.ThirdPartyService = (*ThirdPartyService)(nil)
