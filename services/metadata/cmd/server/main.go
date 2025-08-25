package main

import (
	"context"
	"log"
	"mocks3/services/metadata/internal/config"
	"mocks3/services/metadata/internal/handler"
	"mocks3/services/metadata/internal/repository"
	"mocks3/services/metadata/internal/service"
	"mocks3/shared/client"
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
		ServiceName:    "metadata-service",
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

	logger := obs.Logger()

	// 初始化Consul管理器
	consulManager, err := middleware.NewDefaultConsulManager("metadata-service")
	if err != nil {
		log.Fatalf("Failed to initialize consul: %v", err)
	}

	// 初始化数据库
	db, err := repository.NewDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 初始化仓库
	metadataRepo := repository.NewMetadataRepository(db)

	// 初始化队列客户端
	queueClient := client.NewQueueClient("http://localhost:8083", 30*time.Second)
	
	// TODO: 在未来版本中集成队列功能，当前仅进行连接测试
	_ = queueClient

	// 初始化服务
	metadataService := service.NewMetadataService(metadataRepo, logger)

	// 初始化处理器
	metadataHandler := handler.NewMetadataHandler(metadataService, logger)

	// 注册服务到Consul
	ctx := context.Background()
	consulConfig := &middleware.ConsulConfig{
		ServiceName: "metadata-service",
		ServicePort: cfg.Server.Port,
		HealthPath:  "/health",
		Tags:        []string{"metadata", "api"},
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
	metadataHandler.RegisterRoutes(router)

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "metadata-service",
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
		logger.Info(context.Background(), "Starting metadata service", 
			observability.String("address", cfg.Server.GetAddress()))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(context.Background(), "Shutting down metadata service...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info(context.Background(), "Metadata service stopped")
}
