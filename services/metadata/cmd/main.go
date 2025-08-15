package main

import (
	"context"
	"fmt"
	"mocks3/services/metadata/internal/config"
	"mocks3/services/metadata/internal/db"
	"mocks3/services/metadata/internal/handler"
	"github.com/mocks3/shared/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 连接数据库
	database, err := db.NewPostgresDB(cfg.PostgresURL)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", err)
	}

	// 连接 Redis
	redisClient, err := db.NewRedisClient(cfg.RedisCacheURL)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", err)
	}

	// 自动迁移数据库
	err = db.Migrate(database)
	if err != nil {
		logger.Fatal("Failed to migrate database", err)
	}

	// 连接 Consul
	consulClient, err := connectConsul(cfg.ConsulAddress)
	if err != nil {
		logger.Fatal("Failed to connect to Consul", err)
	}

	// 创建 Gin 引擎
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

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
	h := handler.NewMetadataHandler(database, redisClient)

	// 设置路由
	setupRoutes(router, h)

	// 注册服务到 Consul
	serviceID := fmt.Sprintf("%s-%d", cfg.ServiceName, time.Now().Unix())
	err = registerService(consulClient, cfg, serviceID)
	if err != nil {
		logger.Fatal("Failed to register service", err)
	}

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	// 启动服务器
	go func() {
		logger.Infof("元数据管理服务启动在端口 %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务启动失败", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭元数据管理服务...")

	// 注销服务
	err = consulClient.Agent().ServiceDeregister(serviceID)
	if err != nil {
		logger.Error("Failed to deregister service", err)
	}

	// 关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("服务关闭失败", err)
	}

	logger.Info("元数据管理服务已关闭")
}

// setupRoutes 设置路由
func setupRoutes(router *gin.Engine, h *handler.MetadataHandler) {
	// 健康检查
	router.GET("/health", h.HealthCheck)

	// 元数据 CRUD
	router.POST("/metadata", h.SaveMetadata)
	router.GET("/metadata/:key", h.GetMetadata)
	router.PUT("/metadata/:key", h.UpdateMetadata)
	router.DELETE("/metadata/:key", h.DeleteMetadata)

	// 元数据查询
	router.GET("/metadata", h.ListMetadata)
	router.GET("/metadata/search", h.SearchMetadata)
	router.GET("/stats", h.GetStats)
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
