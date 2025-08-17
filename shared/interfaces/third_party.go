package interfaces

import (
	"context"
	"mocks3/shared/models"
)

// ThirdPartyService 第三方服务接口
type ThirdPartyService interface {
	// 对象操作
	GetObject(ctx context.Context, bucket, key string) (*models.Object, error)
	PutObject(ctx context.Context, object *models.Object) error
	DeleteObject(ctx context.Context, bucket, key string) error

	// 元数据操作
	GetObjectMetadata(ctx context.Context, bucket, key string) (*models.Metadata, error)
	ListObjects(ctx context.Context, bucket, prefix string, limit int) ([]*models.Metadata, error)

	// 数据源管理
	SetDataSource(ctx context.Context, name, config string) error
	GetDataSources(ctx context.Context) ([]models.DataSource, error)

	// 缓存管理
	CacheObject(ctx context.Context, object *models.Object) error
	InvalidateCache(ctx context.Context, bucket, key string) error

	// 统计信息
	GetStats(ctx context.Context) (map[string]interface{}, error)

	// 健康检查
	HealthCheck(ctx context.Context) error
}
