package main

import (
	"context"
	"log"
	"mocks3/services/third-party/internal/config"
	"mocks3/services/third-party/internal/handler"
	"mocks3/services/third-party/internal/repository"
	"mocks3/services/third-party/internal/service"
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
		ServiceName:    "third-party-service",
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
	consulManager, err := middleware.NewDefaultConsulManager("third-party-service")
	if err != nil {
		log.Fatalf("Failed to initialize consul: %v", err)
	}

	// 初始化仓库
	dataSourceRepo := repository.NewDataSourceRepository(cfg.DataSources)
	cacheRepo := repository.NewCacheRepository(&cfg.Cache)

	// 初始化服务
	thirdPartyService := service.NewThirdPartyService(dataSourceRepo, cacheRepo, logger)

	// 初始化处理器
	thirdPartyHandler := handler.NewThirdPartyHandler(thirdPartyService, logger)

	// 注册服务到Consul
	ctx := context.Background()
	consulConfig := &middleware.ConsulConfig{
		ServiceName: "third-party-service",
		ServicePort: cfg.Server.Port,
		HealthPath:  "/health",
		Tags:        []string{"third-party", "external", "cache"},
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
	thirdPartyHandler.RegisterRoutes(router)

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		if err := thirdPartyService.HealthCheck(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "unhealthy",
				"service": "third-party-service",
				"error":   err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "third-party-service",
			"version":   cfg.Server.Version,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// 显示启动信息
	logger.Info(context.Background(), "Starting third-party service", 
		observability.String("address", cfg.Server.GetAddress()))

	// 打印数据源信息
	dataSources, _ := dataSourceRepo.GetAll(ctx)
	for _, ds := range dataSources {
		logger.Info(context.Background(), "Configured data source",
			observability.String("name", ds.Name),
			observability.String("type", ds.Type),
			observability.Bool("enabled", ds.Enabled),
			observability.Int("priority", ds.Priority))
	}

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
		logger.Info(context.Background(), "Third-party service started", 
			observability.String("address", cfg.Server.GetAddress()))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(context.Background(), "Shutting down third-party service...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info(context.Background(), "Third-party service stopped")
}
