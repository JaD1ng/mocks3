package service

import (
	"context"
	"fmt"
	"mocks3/services/storage/internal/config"
	"mocks3/services/storage/internal/repository"
	"mocks3/shared/client"
	"mocks3/shared/models"
	"mocks3/shared/observability"
	"time"
)

// StorageService 存储服务实现
type StorageService struct {
	config           *config.Config
	storageManager   *repository.StorageManager
	metadataClient   *client.MetadataClient
	thirdPartyClient *client.ThirdPartyClient
	logger           *observability.Logger
}

// NewStorageService 创建存储服务
func NewStorageService(cfg *config.Config, logger *observability.Logger) (*StorageService, error) {
	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// 创建存储管理器
	storageManager := repository.NewStorageManager()

	// 初始化存储节点
	for _, nodeConfig := range cfg.Storage.Nodes {
		node, err := repository.NewFileStorageNode(nodeConfig.ID, nodeConfig.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage node %s: %w", nodeConfig.ID, err)
		}
		storageManager.AddNode(node)
		logger.Info(context.Background(), "Storage node created", 
			observability.String("node_id", nodeConfig.ID), 
			observability.String("path", nodeConfig.Path))
	}

	// 创建元数据客户端
	metadataTimeout, err := time.ParseDuration(cfg.Metadata.Timeout)
	if err != nil {
		metadataTimeout = 30 * time.Second
	}
	metadataClient := client.NewMetadataClient(cfg.Metadata.ServiceURL, metadataTimeout)

	// 创建第三方服务客户端
	var thirdPartyClient *client.ThirdPartyClient
	if cfg.ThirdParty.Enabled {
		thirdPartyTimeout, err := time.ParseDuration(cfg.ThirdParty.Timeout)
		if err != nil {
			thirdPartyTimeout = 30 * time.Second
		}
		thirdPartyClient = client.NewThirdPartyClient(cfg.ThirdParty.ServiceURL, thirdPartyTimeout)
		logger.Info(context.Background(), "Third-party service client initialized", 
			observability.String("url", cfg.ThirdParty.ServiceURL))
	} else {
		logger.Info(context.Background(), "Third-party service disabled")
	}

	return &StorageService{
		config:           cfg,
		storageManager:   storageManager,
		metadataClient:   metadataClient,
		thirdPartyClient: thirdPartyClient,
		logger:           logger,
	}, nil
}

// WriteObject 写入对象
func (s *StorageService) WriteObject(ctx context.Context, object *models.Object) error {
	s.logger.InfoContext(ctx, "Writing object", "bucket", object.Bucket, "key", object.Key, "size", object.Size)

	// 验证对象
	if err := s.validateObject(object); err != nil {
		s.logger.ErrorContext(ctx, "Invalid object", "error", err)
		return fmt.Errorf("invalid object: %w", err)
	}

	// 写入存储节点
	if err := s.storageManager.WriteToAllNodes(ctx, object); err != nil {
		s.logger.ErrorContext(ctx, "Failed to write to storage nodes", "error", err)
		return fmt.Errorf("failed to write to storage: %w", err)
	}

	// 保存元数据
	metadata := s.objectToMetadata(object)
	metadata.StorageNodes = s.storageManager.GetNodeIDs()

	if err := s.metadataClient.SaveMetadata(ctx, metadata); err != nil {
		s.logger.ErrorContext(ctx, "Failed to save metadata", "error", err)
		// 如果元数据保存失败，应该考虑回滚存储操作
		s.rollbackStorage(ctx, object.Bucket, object.Key)
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	s.logger.InfoContext(ctx, "Object written successfully", "bucket", object.Bucket, "key", object.Key)
	return nil
}

// ReadObject 读取对象
func (s *StorageService) ReadObject(ctx context.Context, bucket, key string) (*models.Object, error) {
	s.logger.DebugContext(ctx, "Reading object", "bucket", bucket, "key", key)

	if err := s.validateBucketKey(bucket, key); err != nil {
		return nil, fmt.Errorf("invalid bucket or key: %w", err)
	}

	// 首先检查元数据是否存在
	metadata, err := s.metadataClient.GetMetadata(ctx, bucket, key)
	if err != nil {
		s.logger.WarnContext(ctx, "Metadata not found, trying storage directly", "bucket", bucket, "key", key)
	}

	// 从存储读取对象
	object, err := s.storageManager.ReadFromBestNode(ctx, bucket, key)
	if err != nil {
		s.logger.WarnContext(ctx, "Failed to read from storage nodes", "error", err, "bucket", bucket, "key", key)

		// 如果本地存储失败且第三方服务可用，尝试从第三方服务获取
		if s.thirdPartyClient != nil {
			s.logger.InfoContext(ctx, "Trying to read from third-party service", "bucket", bucket, "key", key)

			thirdPartyObject, thirdPartyErr := s.thirdPartyClient.GetObject(ctx, bucket, key)
			if thirdPartyErr != nil {
				s.logger.WarnContext(ctx, "Failed to read from third-party service", "error", thirdPartyErr)
				return nil, fmt.Errorf("failed to read object from storage and third-party: storage_err=%w, third_party_err=%v", err, thirdPartyErr)
			}

			s.logger.InfoContext(ctx, "Object retrieved from third-party service", "bucket", bucket, "key", key, "size", thirdPartyObject.Size)

			// 异步缓存到本地存储
			go func() {
				cacheCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if writeErr := s.storageManager.WriteToAllNodes(cacheCtx, thirdPartyObject); writeErr != nil {
					s.logger.WarnContext(cacheCtx, "Failed to cache third-party object to local storage",
						"error", writeErr, "bucket", bucket, "key", key)
				} else {
					s.logger.InfoContext(cacheCtx, "Third-party object cached to local storage",
						"bucket", bucket, "key", key)
				}
			}()

			object = thirdPartyObject
		} else {
			return nil, fmt.Errorf("failed to read object: %w", err)
		}
	}

	// 如果元数据存在，合并一些信息
	if metadata != nil {
		object.Headers = metadata.Headers
		object.Tags = metadata.Tags
		object.CreatedAt = metadata.CreatedAt
		object.UpdatedAt = metadata.UpdatedAt
	}

	s.logger.DebugContext(ctx, "Object read successfully", "bucket", bucket, "key", key, "size", object.Size)
	return object, nil
}

// DeleteObject 删除对象
func (s *StorageService) DeleteObject(ctx context.Context, bucket, key string) error {
	s.logger.InfoContext(ctx, "Deleting object", "bucket", bucket, "key", key)

	if err := s.validateBucketKey(bucket, key); err != nil {
		return fmt.Errorf("invalid bucket or key: %w", err)
	}

	// 先删除元数据
	if err := s.metadataClient.DeleteMetadata(ctx, bucket, key); err != nil {
		s.logger.WarnContext(ctx, "Failed to delete metadata", "error", err)
		// 元数据删除失败不阻止存储删除
	}

	// 删除存储文件
	if err := s.storageManager.DeleteFromAllNodes(ctx, bucket, key); err != nil {
		s.logger.ErrorContext(ctx, "Failed to delete from storage", "error", err)
		return fmt.Errorf("failed to delete from storage: %w", err)
	}

	s.logger.InfoContext(ctx, "Object deleted successfully", "bucket", bucket, "key", key)
	return nil
}

// ListObjects 列出对象
func (s *StorageService) ListObjects(ctx context.Context, req *models.ListObjectsRequest) (*models.ListObjectsResponse, error) {
	s.logger.DebugContext(ctx, "Listing objects", "bucket", req.Bucket, "prefix", req.Prefix, "max_keys", req.MaxKeys)

	// 参数验证
	if req.MaxKeys <= 0 {
		req.MaxKeys = 1000
	}
	if req.MaxKeys > 1000 {
		req.MaxKeys = 1000
	}

	// 从存储管理器获取对象列表
	objects, err := s.storageManager.ListObjects(ctx, req.Bucket, req.Prefix, req.MaxKeys)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to list objects", "error", err)
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// 构建响应
	objectInfos := make([]models.ObjectInfo, len(objects))
	for i, obj := range objects {
		objectInfos[i] = *obj
	}

	response := &models.ListObjectsResponse{
		Bucket:      req.Bucket,
		Prefix:      req.Prefix,
		MaxKeys:     req.MaxKeys,
		IsTruncated: len(objects) >= req.MaxKeys,
		Objects:     objectInfos,
		Count:       len(objects),
	}

	if response.IsTruncated && len(objects) > 0 {
		response.NextMarker = objects[len(objects)-1].Key
	}

	s.logger.DebugContext(ctx, "Objects listed", "count", len(objects))
	return response, nil
}

// GetStats 获取存储统计信息
func (s *StorageService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	s.logger.DebugContext(ctx, "Getting storage statistics")

	stats, err := s.storageManager.GetStats(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get storage stats", "error", err)
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}

	// 添加服务级别的统计信息
	stats["service_name"] = "storage-service"
	stats["service_version"] = s.config.Server.Version
	stats["timestamp"] = time.Now().Format(time.RFC3339)

	s.logger.DebugContext(ctx, "Statistics retrieved")
	return stats, nil
}

// HealthCheck 健康检查
func (s *StorageService) HealthCheck(ctx context.Context) error {
	s.logger.DebugContext(ctx, "Performing health check")

	// 检查存储节点健康状态
	healthyNodes := s.storageManager.GetHealthyNodes()
	totalNodes := len(s.storageManager.GetAllNodes())

	if len(healthyNodes) == 0 {
		return fmt.Errorf("no healthy storage nodes available")
	}

	// 检查元数据服务连接
	if err := s.metadataClient.HealthCheck(ctx); err != nil {
		s.logger.WarnContext(ctx, "Metadata service health check failed", "error", err)
		// 元数据服务异常不影响存储服务的健康状态
	}

	s.logger.DebugContext(ctx, "Health check passed", "healthy_nodes", len(healthyNodes), "total_nodes", totalNodes)
	return nil
}

// validateObject 验证对象
func (s *StorageService) validateObject(object *models.Object) error {
	if object == nil {
		return fmt.Errorf("object cannot be nil")
	}

	if object.Bucket == "" {
		return fmt.Errorf("bucket cannot be empty")
	}

	if object.Key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if object.Data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	if object.Size != int64(len(object.Data)) {
		return fmt.Errorf("size mismatch: declared %d, actual %d", object.Size, len(object.Data))
	}

	return nil
}

// validateBucketKey 验证bucket和key
func (s *StorageService) validateBucketKey(bucket, key string) error {
	if bucket == "" {
		return fmt.Errorf("bucket cannot be empty")
	}

	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	return nil
}

// objectToMetadata 将对象转换为元数据
func (s *StorageService) objectToMetadata(object *models.Object) *models.Metadata {
	return &models.Metadata{
		ID:          object.ID,
		Key:         object.Key,
		Bucket:      object.Bucket,
		Size:        object.Size,
		ContentType: object.ContentType,
		MD5Hash:     object.MD5Hash,
		ETag:        object.ETag,
		Headers:     object.Headers,
		Tags:        object.Tags,
		Status:      "active",
		Version:     1,
		CreatedAt:   object.CreatedAt,
		UpdatedAt:   object.UpdatedAt,
	}
}

// rollbackStorage 回滚存储操作
func (s *StorageService) rollbackStorage(ctx context.Context, bucket, key string) {
	s.logger.WarnContext(ctx, "Rolling back storage operation", "bucket", bucket, "key", key)

	if err := s.storageManager.DeleteFromAllNodes(ctx, bucket, key); err != nil {
		s.logger.ErrorContext(ctx, "Failed to rollback storage", "error", err)
	}
}
