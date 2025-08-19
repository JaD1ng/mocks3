package service

import (
	"context"
	"fmt"
	"mocks3/shared/interfaces"
	"mocks3/shared/models"
	"mocks3/shared/observability/log"
	"strings"
	"time"
)

// MetadataService 元数据服务实现
type MetadataService struct {
	repo   interfaces.MetadataRepository
	logger *log.Logger
}

// NewMetadataService 创建元数据服务
func NewMetadataService(repo interfaces.MetadataRepository, logger *log.Logger) *MetadataService {
	return &MetadataService{
		repo:   repo,
		logger: logger,
	}
}

// SaveMetadata 保存元数据
func (s *MetadataService) SaveMetadata(ctx context.Context, metadata *models.Metadata) error {
	s.logger.InfoContext(ctx, "Saving metadata", "bucket", metadata.Bucket, "key", metadata.Key)

	// 验证元数据
	if err := s.validateMetadata(metadata); err != nil {
		s.logger.ErrorContext(ctx, "Invalid metadata", "error", err, "bucket", metadata.Bucket, "key", metadata.Key)
		return fmt.Errorf("invalid metadata: %w", err)
	}

	// 设置默认值
	s.setDefaults(metadata)

	// 检查是否已存在
	existing, err := s.repo.GetByKey(ctx, metadata.Bucket, metadata.Key)
	if err != nil && !strings.Contains(err.Error(), "not found") {
		s.logger.ErrorContext(ctx, "Failed to check existing metadata", "error", err)
		return fmt.Errorf("failed to check existing metadata: %w", err)
	}

	if existing != nil {
		// 更新现有元数据
		metadata.ID = existing.ID
		metadata.Version = existing.Version
		metadata.CreatedAt = existing.CreatedAt

		if err := s.repo.Update(ctx, metadata); err != nil {
			s.logger.ErrorContext(ctx, "Failed to update metadata", "error", err)
			return fmt.Errorf("failed to update metadata: %w", err)
		}

		s.logger.InfoContext(ctx, "Metadata updated", "bucket", metadata.Bucket, "key", metadata.Key, "version", metadata.Version)
	} else {
		// 创建新元数据
		if err := s.repo.Create(ctx, metadata); err != nil {
			s.logger.ErrorContext(ctx, "Failed to create metadata", "error", err)
			return fmt.Errorf("failed to create metadata: %w", err)
		}

		s.logger.InfoContext(ctx, "Metadata created", "bucket", metadata.Bucket, "key", metadata.Key, "id", metadata.ID)
	}

	return nil
}

// GetMetadata 获取元数据
func (s *MetadataService) GetMetadata(ctx context.Context, bucket, key string) (*models.Metadata, error) {
	s.logger.DebugContext(ctx, "Getting metadata", "bucket", bucket, "key", key)

	if err := s.validateBucketKey(bucket, key); err != nil {
		return nil, fmt.Errorf("invalid bucket or key: %w", err)
	}

	metadata, err := s.repo.GetByKey(ctx, bucket, key)
	if err != nil {
		s.logger.WarnContext(ctx, "Metadata not found", "bucket", bucket, "key", key, "error", err)
		return nil, fmt.Errorf("metadata not found: %w", err)
	}

	s.logger.DebugContext(ctx, "Metadata retrieved", "bucket", bucket, "key", key, "size", metadata.Size)
	return metadata, nil
}

// UpdateMetadata 更新元数据
func (s *MetadataService) UpdateMetadata(ctx context.Context, metadata *models.Metadata) error {
	s.logger.InfoContext(ctx, "Updating metadata", "bucket", metadata.Bucket, "key", metadata.Key)

	if err := s.validateMetadata(metadata); err != nil {
		return fmt.Errorf("invalid metadata: %w", err)
	}

	if err := s.repo.Update(ctx, metadata); err != nil {
		s.logger.ErrorContext(ctx, "Failed to update metadata", "error", err)
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	s.logger.InfoContext(ctx, "Metadata updated successfully", "bucket", metadata.Bucket, "key", metadata.Key)
	return nil
}

// DeleteMetadata 删除元数据
func (s *MetadataService) DeleteMetadata(ctx context.Context, bucket, key string) error {
	s.logger.InfoContext(ctx, "Deleting metadata", "bucket", bucket, "key", key)

	if err := s.validateBucketKey(bucket, key); err != nil {
		return fmt.Errorf("invalid bucket or key: %w", err)
	}

	if err := s.repo.Delete(ctx, bucket, key); err != nil {
		s.logger.ErrorContext(ctx, "Failed to delete metadata", "error", err, "bucket", bucket, "key", key)
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	s.logger.InfoContext(ctx, "Metadata deleted successfully", "bucket", bucket, "key", key)
	return nil
}

// ListMetadata 列出元数据
func (s *MetadataService) ListMetadata(ctx context.Context, bucket, prefix string, limit, offset int) ([]*models.Metadata, error) {
	s.logger.DebugContext(ctx, "Listing metadata", "bucket", bucket, "prefix", prefix, "limit", limit, "offset", offset)

	// 参数验证
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	metadataList, err := s.repo.List(ctx, bucket, prefix, limit, offset)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list metadata", "error", err)
		return nil, fmt.Errorf("failed to list metadata: %w", err)
	}

	s.logger.DebugContext(ctx, "Metadata listed", "count", len(metadataList))
	return metadataList, nil
}

// SearchMetadata 搜索元数据
func (s *MetadataService) SearchMetadata(ctx context.Context, query string, limit int) ([]*models.Metadata, error) {
	s.logger.DebugContext(ctx, "Searching metadata", "query", query, "limit", limit)

	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	metadataList, err := s.repo.Search(ctx, query, limit)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to search metadata", "error", err)
		return nil, fmt.Errorf("failed to search metadata: %w", err)
	}

	s.logger.DebugContext(ctx, "Metadata search completed", "query", query, "count", len(metadataList))
	return metadataList, nil
}

// GetStats 获取统计信息
func (s *MetadataService) GetStats(ctx context.Context) (*models.Stats, error) {
	s.logger.DebugContext(ctx, "Getting statistics")

	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get statistics", "error", err)
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	s.logger.DebugContext(ctx, "Statistics retrieved",
		"total_objects", stats.TotalObjects,
		"total_size", stats.TotalSize,
		"buckets", len(stats.BucketStats))

	return stats, nil
}

// CountObjects 计算对象数量
func (s *MetadataService) CountObjects(ctx context.Context, bucket, prefix string) (int64, error) {
	s.logger.DebugContext(ctx, "Counting objects", "bucket", bucket, "prefix", prefix)

	count, err := s.repo.Count(ctx, bucket, prefix)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to count objects", "error", err)
		return 0, fmt.Errorf("failed to count objects: %w", err)
	}

	s.logger.DebugContext(ctx, "Objects counted", "count", count)
	return count, nil
}

// HealthCheck 健康检查
func (s *MetadataService) HealthCheck(ctx context.Context) error {
	s.logger.DebugContext(ctx, "Performing health check")

	// 可以添加更多健康检查逻辑
	// 例如检查数据库连接、缓存等

	s.logger.DebugContext(ctx, "Health check passed")
	return nil
}

// validateMetadata 验证元数据
func (s *MetadataService) validateMetadata(metadata *models.Metadata) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	if strings.TrimSpace(metadata.Bucket) == "" {
		return fmt.Errorf("bucket cannot be empty")
	}

	if strings.TrimSpace(metadata.Key) == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if metadata.Size < 0 {
		return fmt.Errorf("size cannot be negative")
	}

	// 验证bucket名称格式（简单验证）
	if len(metadata.Bucket) < 3 || len(metadata.Bucket) > 63 {
		return fmt.Errorf("bucket name must be between 3 and 63 characters")
	}

	// 验证key格式
	if len(metadata.Key) > 1024 {
		return fmt.Errorf("key cannot exceed 1024 characters")
	}

	// 检查非法字符
	if strings.Contains(metadata.Bucket, "..") || strings.Contains(metadata.Key, "..") {
		return fmt.Errorf("bucket and key cannot contain '..'")
	}

	return nil
}

// validateBucketKey 验证bucket和key
func (s *MetadataService) validateBucketKey(bucket, key string) error {
	if strings.TrimSpace(bucket) == "" {
		return fmt.Errorf("bucket cannot be empty")
	}

	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("key cannot be empty")
	}

	return nil
}

// setDefaults 设置默认值
func (s *MetadataService) setDefaults(metadata *models.Metadata) {
	if metadata.Status == "" {
		metadata.Status = "active"
	}

	if metadata.Headers == nil {
		metadata.Headers = make(map[string]string)
	}

	if metadata.Tags == nil {
		metadata.Tags = make(map[string]string)
	}

	if metadata.StorageNodes == nil {
		metadata.StorageNodes = make([]string, 0)
	}

	if metadata.Version == 0 {
		metadata.Version = 1
	}

	now := time.Now()
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = now
	}
	metadata.UpdatedAt = now
}
