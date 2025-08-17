package interfaces

import (
	"context"
	"mocks3/shared/models"
	"net/http"
)

// GatewayService 网关服务接口
type GatewayService interface {
	// S3 API 处理
	HandlePutObject(w http.ResponseWriter, r *http.Request)
	HandleGetObject(w http.ResponseWriter, r *http.Request)
	HandleDeleteObject(w http.ResponseWriter, r *http.Request)
	HandleHeadObject(w http.ResponseWriter, r *http.Request)
	HandleListObjects(w http.ResponseWriter, r *http.Request)

	// 管理 API
	HandleGetStats(w http.ResponseWriter, r *http.Request)
	HandleSearchObjects(w http.ResponseWriter, r *http.Request)

	// 健康检查
	HandleHealthCheck(w http.ResponseWriter, r *http.Request)
}

// ServiceDiscovery 服务发现接口
type ServiceDiscovery interface {
	// 服务注册
	RegisterService(ctx context.Context, service *models.ServiceInfo) error
	DeregisterService(ctx context.Context, serviceID string) error

	// 服务发现
	DiscoverService(ctx context.Context, serviceName string) ([]*models.ServiceInfo, error)
	GetHealthyServices(ctx context.Context, serviceName string) ([]*models.ServiceInfo, error)

	// 健康检查
	SetHealthCheck(ctx context.Context, serviceID string, healthy bool) error
	GetServiceHealth(ctx context.Context, serviceID string) (bool, error)

	// 配置管理
	GetConfig(ctx context.Context, key string) (string, error)
	SetConfig(ctx context.Context, key, value string) error
	WatchConfig(ctx context.Context, key string) (<-chan string, error)
}

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	SelectService(services []*models.ServiceInfo) (*models.ServiceInfo, error)
	UpdateWeights(weights map[string]int) error
	GetStrategy() string
	SetStrategy(strategy string) error
}

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	SetLimit(ctx context.Context, key string, limit int64, window int64) error
	GetLimit(ctx context.Context, key string) (*models.RateLimit, error)
	RemoveLimit(ctx context.Context, key string) error
}
