package main

import (
	"context"
	"fmt"
	"mocks3/shared/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"mocks3/services/task-processor/internal/client"
	"mocks3/services/task-processor/internal/config"
	"mocks3/services/task-processor/internal/handler"
	"mocks3/services/task-processor/internal/processor"
	"mocks3/services/task-processor/internal/queue"
	"mocks3/services/task-processor/internal/worker"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 连接 Redis 队列
	queueClient, err := queue.NewRedisQueue(cfg.RedisQueueURL)
	if err != nil {
		logger.Fatal("Failed to connect to Redis queue", err)
	}

	// 初始化服务客户端
	clients := initializeClients(cfg)

	// 创建任务处理器
	taskProcessor := processor.NewTaskProcessor(clients)

	// 创建工作节点管理器
	workerManager := worker.NewWorkerManager(queueClient, taskProcessor, cfg.WorkerCount)

	// 启动工作节点
	err = workerManager.Start()
	if err != nil {
		logger.Fatal("Failed to start workers", err)
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
	h := handler.NewTaskHandler(queueClient, workerManager)

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
		logger.Infof("异步任务处理服务启动在端口 %d", cfg.Port)
		logger.Infof("工作节点数量: %d", cfg.WorkerCount)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务启动失败", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭异步任务处理服务...")

	// 停止工作节点
	workerManager.Stop()

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

	// 关闭队列连接
	if err := queueClient.Close(); err != nil {
		logger.Error("关闭队列连接失败", err)
	}

	logger.Info("异步任务处理服务已关闭")
}

// setupRoutes 设置路由
func setupRoutes(router *gin.Engine, h *handler.TaskHandler) {
	// 健康检查
	router.GET("/health", h.HealthCheck)

	// 任务管理
	router.POST("/tasks", h.SubmitTask)
	router.GET("/tasks", h.GetTasks)
	router.GET("/tasks/:taskId", h.GetTask)
	router.DELETE("/tasks/:taskId", h.CancelTask)

	// 队列统计
	router.GET("/queue/stats", h.GetQueueStats)
	router.GET("/workers/status", h.GetWorkerStatus)
}

// initializeClients 初始化服务客户端
func initializeClients(cfg *config.Config) *client.Clients {
	return &client.Clients{
		Storage: client.NewStorageClient(cfg.StorageServiceURL),
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
