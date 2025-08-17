package interfaces

import (
	"context"
	"mocks3/shared/models"
)

// MetadataService 元数据服务接口
type MetadataService interface {
	// 元数据操作
	SaveMetadata(ctx context.Context, metadata *models.Metadata) error
	GetMetadata(ctx context.Context, bucket, key string) (*models.Metadata, error)
	UpdateMetadata(ctx context.Context, metadata *models.Metadata) error
	DeleteMetadata(ctx context.Context, bucket, key string) error

	// 查询操作
	ListMetadata(ctx context.Context, bucket, prefix string, limit, offset int) ([]*models.Metadata, error)
	SearchMetadata(ctx context.Context, query string, limit int) ([]*models.Metadata, error)

	// 统计操作
	GetStats(ctx context.Context) (*models.Stats, error)
	CountObjects(ctx context.Context, bucket, prefix string) (int64, error)

	// 健康检查
	HealthCheck(ctx context.Context) error
}

// MetadataRepository 元数据存储接口
type MetadataRepository interface {
	Create(ctx context.Context, metadata *models.Metadata) error
	GetByKey(ctx context.Context, bucket, key string) (*models.Metadata, error)
	Update(ctx context.Context, metadata *models.Metadata) error
	Delete(ctx context.Context, bucket, key string) error
	List(ctx context.Context, bucket, prefix string, limit, offset int) ([]*models.Metadata, error)
	Search(ctx context.Context, query string, limit int) ([]*models.Metadata, error)
	Count(ctx context.Context, bucket, prefix string) (int64, error)
	GetStats(ctx context.Context) (*models.Stats, error)
}
