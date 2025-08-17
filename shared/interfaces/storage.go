package interfaces

import (
	"context"
	"mocks3/shared/models"
)

// StorageService 存储服务接口
type StorageService interface {
	// 文件操作
	WriteObject(ctx context.Context, object *models.Object) error
	ReadObject(ctx context.Context, bucket, key string) (*models.Object, error)
	DeleteObject(ctx context.Context, bucket, key string) error
	ListObjects(ctx context.Context, req *models.ListObjectsRequest) (*models.ListObjectsResponse, error)

	// 统计信息
	GetStats(ctx context.Context) (map[string]interface{}, error)

	// 健康检查
	HealthCheck(ctx context.Context) error
}

// StorageNode 存储节点接口
type StorageNode interface {
	GetNodeID() string
	Write(ctx context.Context, object *models.Object) error
	Read(ctx context.Context, bucket, key string) (*models.Object, error)
	Delete(ctx context.Context, bucket, key string) error
	IsHealthy(ctx context.Context) bool
}

// StorageManager 存储管理器接口
type StorageManager interface {
	AddNode(node StorageNode)
	WriteToAllNodes(ctx context.Context, object *models.Object) error
	ReadFromBestNode(ctx context.Context, bucket, key string) (*models.Object, error)
	DeleteFromAllNodes(ctx context.Context, bucket, key string) error
	GetHealthyNodes() []StorageNode
}
