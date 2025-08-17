package middleware

import (
	"context"
	"fmt"
	"log"
	"mocks3/shared/models"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
)

// ConsulManager Consul管理器
type ConsulManager struct {
	client      *api.Client
	serviceName string
	serviceID   string
	servicePort int
	healthCheck *api.AgentServiceCheck
}

// ConsulConfig Consul配置
type ConsulConfig struct {
	Address     string
	ServiceName string
	ServicePort int
	HealthPath  string
	Tags        []string
	Metadata    map[string]string
}

// NewConsulManager 创建Consul管理器
func NewConsulManager(config *ConsulConfig) (*ConsulManager, error) {
	consulConfig := api.DefaultConfig()
	consulConfig.Address = config.Address

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	// 生成唯一的服务ID
	hostname, _ := os.Hostname()
	serviceID := fmt.Sprintf("%s-%s-%d", config.ServiceName, hostname, config.ServicePort)

	// 创建健康检查
	healthCheck := &api.AgentServiceCheck{
		HTTP:                           fmt.Sprintf("http://localhost:%d%s", config.ServicePort, config.HealthPath),
		Interval:                       "10s",
		Timeout:                        "5s",
		DeregisterCriticalServiceAfter: "30s",
	}

	return &ConsulManager{
		client:      client,
		serviceName: config.ServiceName,
		serviceID:   serviceID,
		servicePort: config.ServicePort,
		healthCheck: healthCheck,
	}, nil
}

// RegisterService 注册服务
func (cm *ConsulManager) RegisterService(ctx context.Context, config *ConsulConfig) error {
	service := &api.AgentServiceRegistration{
		ID:      cm.serviceID,
		Name:    cm.serviceName,
		Port:    cm.servicePort,
		Address: "localhost",
		Tags:    config.Tags,
		Meta:    config.Metadata,
		Check:   cm.healthCheck,
	}

	err := cm.client.Agent().ServiceRegister(service)
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	log.Printf("Service registered: %s (ID: %s)", cm.serviceName, cm.serviceID)
	return nil
}

// DeregisterService 注销服务
func (cm *ConsulManager) DeregisterService(ctx context.Context) error {
	err := cm.client.Agent().ServiceDeregister(cm.serviceID)
	if err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}

	log.Printf("Service deregistered: %s (ID: %s)", cm.serviceName, cm.serviceID)
	return nil
}

// DiscoverServices 发现服务
func (cm *ConsulManager) DiscoverServices(ctx context.Context, serviceName string) ([]*models.ServiceInfo, error) {
	services, _, err := cm.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	var serviceInfos []*models.ServiceInfo
	for _, service := range services {
		serviceInfo := &models.ServiceInfo{
			ID:       service.Service.ID,
			Name:     service.Service.Service,
			Address:  service.Service.Address,
			Port:     service.Service.Port,
			Tags:     service.Service.Tags,
			Metadata: service.Service.Meta,
			Health: models.ServiceHealth{
				Status: models.HealthStatusHealthy, // 这里只返回健康的服务
			},
		}
		serviceInfos = append(serviceInfos, serviceInfo)
	}

	return serviceInfos, nil
}

// GetConfig 获取配置
func (cm *ConsulManager) GetConfig(ctx context.Context, key string) (string, error) {
	kv, _, err := cm.client.KV().Get(key, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	if kv == nil {
		return "", fmt.Errorf("config key not found: %s", key)
	}

	return string(kv.Value), nil
}

// SetConfig 设置配置
func (cm *ConsulManager) SetConfig(ctx context.Context, key, value string) error {
	kv := &api.KVPair{
		Key:   key,
		Value: []byte(value),
	}

	_, err := cm.client.KV().Put(kv, nil)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	return nil
}

// WatchConfig 监听配置变化
func (cm *ConsulManager) WatchConfig(ctx context.Context, key string) (<-chan string, error) {
	ch := make(chan string, 1)

	go func() {
		defer close(ch)

		var lastIndex uint64
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			queryOptions := &api.QueryOptions{
				WaitIndex: lastIndex,
				WaitTime:  30 * time.Second,
			}

			kv, meta, err := cm.client.KV().Get(key, queryOptions)
			if err != nil {
				log.Printf("Error watching config %s: %v", key, err)
				time.Sleep(5 * time.Second)
				continue
			}

			if meta.LastIndex == lastIndex {
				continue
			}

			lastIndex = meta.LastIndex

			if kv != nil {
				select {
				case ch <- string(kv.Value):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// GetServiceHealth 获取服务健康状态
func (cm *ConsulManager) GetServiceHealth(ctx context.Context, serviceID string) (bool, error) {
	checks, _, err := cm.client.Health().Checks(cm.serviceName, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get service health: %w", err)
	}

	for _, check := range checks {
		if check.ServiceID == serviceID {
			return check.Status == api.HealthPassing, nil
		}
	}

	return false, fmt.Errorf("service not found: %s", serviceID)
}

// SetServiceHealth 设置服务健康状态
func (cm *ConsulManager) SetServiceHealth(ctx context.Context, serviceID string, healthy bool) error {
	status := api.HealthPassing
	if !healthy {
		status = api.HealthCritical
	}

	err := cm.client.Agent().UpdateTTL(fmt.Sprintf("service:%s", serviceID), "Manual health update", status)
	if err != nil {
		return fmt.Errorf("failed to set service health: %w", err)
	}

	return nil
}

// NewDefaultConsulManager 创建默认的Consul管理器
func NewDefaultConsulManager(serviceName string) (*ConsulManager, error) {
	port, err := strconv.Atoi(getEnv("SERVICE_PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid service port: %w", err)
	}

	config := &ConsulConfig{
		Address:     getEnv("CONSUL_ADDR", "localhost:8500"),
		ServiceName: serviceName,
		ServicePort: port,
		HealthPath:  "/health",
		Tags:        []string{"api", "http"},
		Metadata: map[string]string{
			"version":     getEnv("SERVICE_VERSION", "1.0.0"),
			"environment": getEnv("ENVIRONMENT", "development"),
		},
	}

	return NewConsulManager(config)
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
