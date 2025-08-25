package main

import (
	"context"
	"log"
	"mocks3/services/storage/internal/config"
	"mocks3/services/storage/internal/handler"
	"mocks3/services/storage/internal/service"
	"mocks3/shared/middleware"
	"mocks3/shared/observability"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 初始化统一可观测性
	obsConfig := &observability.Config{
		ServiceName:    "storage-service",
		ServiceVersion: "1.0.0",
		Environment:    cfg.Server.Environment,
		OTLPEndpoint:   "http://localhost:4318",
		LogLevel:       cfg.LogLevel,
	}

	obs, err := observability.New(context.Background(), obsConfig)
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}
	defer obs.Shutdown(context.Background())

	loggerInstance := obs.Logger()

	// 初始化Consul管理器
	consulManager, err := middleware.NewDefaultConsulManager("storage-service")
	if err != nil {
		log.Fatalf("Failed to initialize consul: %v", err)
	}

	// 初始化存储服务
	storageService, err := service.NewStorageService(cfg, loggerInstance)
	if err != nil {
		log.Fatalf("Failed to initialize storage service: %v", err)
	}

	// 初始化处理器
	storageHandler := handler.NewStorageHandler(storageService, loggerInstance)

	// 注册服务到Consul
	ctx := context.Background()
	consulConfig := &middleware.ConsulConfig{
		ServiceName: "storage-service",
		ServicePort: cfg.Server.Port,
		HealthPath:  "/health",
		Tags:        []string{"storage", "api"},
		Metadata: map[string]string{
			"version": cfg.Server.Version,
		},
	}

	err = consulManager.RegisterService(ctx, consulConfig)
	if err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}
	defer consulManager.DeregisterService(ctx)

	// 设置Gin模式
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	router := gin.New()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(middleware.GinRecoveryMiddleware(middleware.DefaultRecoveryConfig()))
	// 使用统一可观测性中间件
	router.Use(obs.GinMiddleware())

	// 设置路由
	storageHandler.RegisterRoutes(router)

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "storage-service",
			"version":   cfg.Server.Version,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// 创建HTTP服务器
	server := &http.Server{
		Addr:         cfg.Server.GetAddress(),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器
	go func() {
		loggerInstance.Info(context.Background(), "Starting storage service", 
			observability.String("address", cfg.Server.GetAddress()))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	loggerInstance.Info(context.Background(), "Shutting down storage service...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	loggerInstance.Info(context.Background(), "Storage service stopped")
}
