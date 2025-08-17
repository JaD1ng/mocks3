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
	logger "mocks3/shared/observability/log"
	"mocks3/shared/observability/metric"
	"mocks3/shared/observability/trace"
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

	// 初始化日志器
	logger := logger.NewLogger("metadata-service", logger.LogLevel(cfg.LogLevel))

	// 初始化追踪器
	tracerProvider, err := trace.NewDefaultTracerProvider("metadata-service")
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer tracerProvider.Shutdown(context.Background())

	// 初始化指标收集器
	metricCollector, err := metric.NewDefaultCollector("metadata-service")
	if err != nil {
		log.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer metricCollector.Shutdown(context.Background())

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
	_ = client.NewQueueClient("http://localhost:8083", 30*time.Second)

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
	router.Use(trace.GinMiddleware("metadata-service"))

	// 添加指标中间件
	metricsMiddleware := metric.NewDefaultMiddlewareConfig(metricCollector)
	router.Use(metricsMiddleware.GinMiddleware())

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
		logger.Info("Starting metadata service", "address", cfg.Server.GetAddress())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down metadata service...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Metadata service stopped")
}
