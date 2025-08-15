package main

import (
	"context"
	"fmt"
	"mocks3/services/admin-api/internal/client"
	"mocks3/services/admin-api/internal/config"
	"mocks3/services/admin-api/internal/handler"

	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mocks3/shared/logger"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化服务客户端
	clients := initializeClients(cfg)

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
	h := handler.NewAdminHandler(clients)

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
		logger.Infof("Admin API 服务启动在端口 %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务启动失败", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭 Admin API 服务...")

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

	logger.Info("Admin API 服务已关闭")
}

// setupRoutes 设置路由
func setupRoutes(router *gin.Engine, h *handler.AdminHandler) {
	// 健康检查
	router.GET("/health", h.HealthCheck)

	// 系统概览
	router.GET("/overview", h.GetOverview)
	router.GET("/services", h.GetServices)
	router.GET("/services/:serviceName/status", h.GetServiceStatus)

	// 对象管理
	router.GET("/objects", h.ListObjects)
	router.GET("/objects/:key", h.GetObject)
	router.POST("/objects", h.CreateObject)
	router.DELETE("/objects/:key", h.DeleteObject)

	// 存储管理
	router.GET("/storage/stats", h.GetStorageStats)
	router.GET("/storage/nodes", h.GetStorageNodes)
	router.GET("/storage/nodes/:nodeId", h.GetStorageNode)

	// 任务管理
	router.GET("/tasks", h.GetTasks)
	router.POST("/tasks", h.CreateTask)
	router.GET("/tasks/stats", h.GetTaskStats)
	router.GET("/workers/status", h.GetWorkerStatus)

	// 混沌工程管理
	router.GET("/chaos/rules", h.GetChaosRules)
	router.POST("/chaos/rules", h.CreateChaosRule)
	router.PUT("/chaos/rules/:ruleId", h.UpdateChaosRule)
	router.DELETE("/chaos/rules/:ruleId", h.DeleteChaosRule)
	router.POST("/chaos/rules/:ruleId/enable", h.EnableChaosRule)
	router.POST("/chaos/rules/:ruleId/disable", h.DisableChaosRule)
	router.GET("/chaos/logs", h.GetChaosLogs)

	// 监控和统计
	router.GET("/metrics", h.GetMetrics)
	router.GET("/logs", h.GetLogs)
	router.GET("/dashboard", h.GetDashboard)
}

// initializeClients 初始化服务客户端
func initializeClients(cfg *config.Config) *client.Clients {
	return &client.Clients{
		Metadata: client.NewServiceClient(cfg.MetadataServiceURL),
		Storage:  client.NewServiceClient(cfg.StorageServiceURL),
		Task:     client.NewServiceClient(cfg.TaskServiceURL),
		Chaos:    client.NewServiceClient(cfg.ChaosServiceURL),
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
