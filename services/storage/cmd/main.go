package main

import (
	"context"
	"fmt"
	"mocks3/shared/logger"
	"net/http"
	"os"
	"os/signal"
	"mocks3/services/storage/internal/config"
	"mocks3/services/storage/internal/handler"
	"mocks3/services/storage/internal/nodes"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化存储节点
	storageManager, err := initializeStorageNodes(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize storage nodes", err)
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
	h := handler.NewStorageHandler(storageManager)

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
		logger.Infof("分布式存储服务启动在端口 %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务启动失败", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭分布式存储服务...")

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

	logger.Info("分布式存储服务已关闭")
}

// setupRoutes 设置路由
func setupRoutes(router *gin.Engine, h *handler.StorageHandler) {
	// 健康检查
	router.GET("/health", h.HealthCheck)

	// 存储操作
	router.POST("/storage/write", h.WriteObject)
	router.GET("/storage/read/:key", h.ReadObject)
	router.DELETE("/storage/delete/:key", h.DeleteObject)

	// 存储节点管理
	router.GET("/nodes", h.GetNodes)
	router.GET("/nodes/:nodeId/status", h.GetNodeStatus)
	router.GET("/stats", h.GetStats)
}

// initializeStorageNodes 初始化存储节点
func initializeStorageNodes(cfg *config.Config) (*nodes.StorageManager, error) {
	manager := nodes.NewStorageManager()

	// 创建存储节点
	nodeConfigs := []struct {
		ID   string
		Path string
	}{
		{"stg1", "/data/stg1"},
		{"stg2", "/data/stg2"},
		{"stg3", "/data/stg3"},
	}

	for _, nodeConfig := range nodeConfigs {
		node, err := nodes.NewFileStorageNode(nodeConfig.ID, nodeConfig.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to create storage node %s: %v", nodeConfig.ID, err)
		}
		manager.AddNode(node)
		logger.Infof("创建存储节点: %s (%s)", nodeConfig.ID, nodeConfig.Path)
	}

	// 设置第三方服务
	thirdParty := nodes.NewMockThirdPartyService()
	manager.SetThirdPartyService(thirdParty)
	logger.Infof("设置第三方服务: %s", thirdParty.GetName())

	return manager, nil
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
