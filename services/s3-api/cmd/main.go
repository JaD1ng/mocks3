package main

import (
	"context"
	"fmt"
	"mocks3/services/s3-api/internal/client"
	"mocks3/services/s3-api/internal/config"
	"mocks3/services/s3-api/internal/handler"
	"github.com/mocks3/shared/logger"
	"github.com/mocks3/shared/middleware"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
)

func main() {
	// 初始化日志器
	logger := logger.NewLogger(logger.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: []string{"stdout"},
	})
	ctx := context.Background()

	// 加载配置
	cfg := config.Load()

	// 连接 Consul
	consulClient, err := connectConsul(cfg.ConsulAddress)
	if err != nil {
		logger.Error(ctx, "Failed to connect to Consul", err, map[string]any{
			"consul_address": cfg.ConsulAddress,
		})
		os.Exit(1)
	}

	// 初始化服务客户端
	clients := initializeClients(cfg)

	// 创建 Gin 引擎
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.LoggerWithConfig(logger), middleware.RecoveryWithConfig(logger), middleware.RequestID())

	// CORS 中间件
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, HEAD, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 初始化处理器
	h := handler.NewS3Handler(clients)

	// 设置路由
	setupRoutes(router, h)

	// 注册服务到 Consul
	serviceID := fmt.Sprintf("%s-%d", cfg.ServiceName, time.Now().Unix())
	err = registerService(consulClient, cfg, serviceID)
	if err != nil {
		logger.Error(ctx, "Failed to register service", err, map[string]any{
			"service_id": serviceID,
		})
		os.Exit(1)
	}

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	// 启动服务器
	go func() {
		logger.Info(ctx, "S3 API 服务启动", map[string]any{
			"port":         cfg.Port,
			"service_name": cfg.ServiceName,
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "服务启动失败", err, map[string]any{
				"port": cfg.Port,
			})
			os.Exit(1)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "正在关闭 S3 API 服务", nil)

	// 注销服务
	err = consulClient.Agent().ServiceDeregister(serviceID)
	if err != nil {
		logger.Error(ctx, "Failed to deregister service", err, map[string]any{
			"service_id": serviceID,
		})
	}

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error(ctx, "服务关闭失败", err, nil)
		os.Exit(1)
	}

	logger.Info(ctx, "S3 API 服务已关闭", nil)
}

// setupRoutes 设置路由
func setupRoutes(router *gin.Engine, h *handler.S3Handler) {
	// 健康检查
	router.GET("/health", h.HealthCheck)

	// S3 兼容接口
	router.PUT("/:bucket/:key", h.PutObject)
	router.GET("/:bucket/:key", h.GetObject)
	router.DELETE("/:bucket/:key", h.DeleteObject)
	router.HEAD("/:bucket/:key", h.HeadObject)
	router.GET("/:bucket", h.ListObjects)
}

// initializeClients 初始化服务客户端
func initializeClients(cfg *config.Config) *client.Clients {
	return &client.Clients{
		Metadata: client.NewMetadataClient(cfg.MetadataServiceURL),
		Storage:  client.NewStorageClient(cfg.StorageServiceURL),
		Task:     client.NewTaskClient(cfg.TaskServiceURL),
	}
}

// connectConsul 连接到 Consul
func connectConsul(address string) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = address
	return api.NewClient(config)
}

// registerService 注册服务到 Consul
func registerService(client *api.Client, cfg *config.Config, serviceID string) error {
	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    cfg.ServiceName,
		Port:    cfg.Port,
		Address: cfg.ServiceAddress,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", cfg.ServiceAddress, cfg.Port),
			Timeout:                        "3s",
			Interval:                       "10s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}

	return client.Agent().ServiceRegister(registration)
}
