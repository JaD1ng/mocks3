package main

import (
	"context"
	"log"
	"mocks3/services/mock-error/internal/config"
	"mocks3/services/mock-error/internal/handler"
	"mocks3/services/mock-error/internal/repository"
	"mocks3/services/mock-error/internal/service"
	"mocks3/shared/middleware"
	"mocks3/shared/models"
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

	// 验证配置
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// 初始化日志器
	logger := logger.NewLogger("mock-error-service", logger.LogLevel(cfg.LogLevel))

	// 初始化追踪器
	tracerProvider, err := trace.NewDefaultTracerProvider("mock-error-service")
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer tracerProvider.Shutdown(context.Background())

	// 初始化指标收集器
	metricCollector, err := metric.NewDefaultCollector("mock-error-service")
	if err != nil {
		log.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer metricCollector.Shutdown(context.Background())

	// 初始化Consul管理器
	var consulManager *middleware.ConsulManager
	if cfg.Consul.Enabled {
		consulManager, err = middleware.NewDefaultConsulManager("mock-error-service")
		if err != nil {
			log.Fatalf("Failed to initialize consul: %v", err)
		}
	}

	// 初始化仓库
	ruleRepo := repository.NewRuleRepository()
	statsRepo := repository.NewStatsRepository(10000, cfg.ErrorEngine.StatRetentionHours)

	// 初始化规则引擎
	ruleEngine := service.NewRuleEngine(logger)

	// 初始化错误注入服务
	errorService := service.NewErrorInjectorService(cfg, ruleRepo, statsRepo, ruleEngine, logger)

	// 初始化处理器
	errorHandler := handler.NewErrorHandler(errorService, logger)

	// 注册服务到Consul
	ctx := context.Background()
	if consulManager != nil {
		consulConfig := &middleware.ConsulConfig{
			ServiceName: "mock-error-service",
			ServicePort: cfg.Server.Port,
			HealthPath:  "/health",
			Tags:        []string{"mock", "error", "injection", "chaos"},
			Metadata: map[string]string{
				"version":     cfg.Server.Version,
				"environment": cfg.Server.Environment,
			},
		}

		err = consulManager.RegisterService(ctx, consulConfig)
		if err != nil {
			log.Fatalf("Failed to register service: %v", err)
		}
		defer consulManager.DeregisterService(ctx)
	}

	// 设置Gin模式
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由器
	router := gin.New()

	// 添加中间件
	router.Use(gin.Logger())
	router.Use(middleware.GinRecoveryMiddleware(middleware.DefaultRecoveryConfig()))
	router.Use(trace.GinMiddleware("mock-error-service"))

	// 添加指标中间件
	metricsMiddleware := metric.NewDefaultMiddlewareConfig(metricCollector)
	router.Use(metricsMiddleware.GinMiddleware())

	// 设置路由
	errorHandler.RegisterRoutes(router)

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		if err := errorService.HealthCheck(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "unhealthy",
				"service": "mock-error-service",
				"error":   err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "mock-error-service",
			"version":   cfg.Server.Version,
			"timestamp": time.Now().Format(time.RFC3339),
			"config": gin.H{
				"max_rules":              cfg.ErrorEngine.MaxRules,
				"enable_statistics":      cfg.ErrorEngine.EnableStatistics,
				"enable_scheduling":      cfg.ErrorEngine.EnableScheduling,
				"global_probability":     cfg.Injection.GlobalProbability,
				"enable_http_errors":     cfg.Injection.EnableHTTPErrors,
				"enable_network_errors":  cfg.Injection.EnableNetworkErrors,
				"enable_database_errors": cfg.Injection.EnableDatabaseErrors,
				"enable_storage_errors":  cfg.Injection.EnableStorageErrors,
			},
		})
	})

	// 显示启动信息
	logger.Info("Starting mock error service", "address", cfg.Server.GetAddress())
	logger.Info("Service configuration",
		"max_rules", cfg.ErrorEngine.MaxRules,
		"default_probability", cfg.ErrorEngine.DefaultProbability,
		"enable_statistics", cfg.ErrorEngine.EnableStatistics,
		"global_probability", cfg.Injection.GlobalProbability)

	// 添加一些示例规则（仅在开发环境）
	if cfg.Server.Environment == "development" {
		addSampleRules(ctx, errorService, logger)
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
		logger.Info("Mock error service started", "address", cfg.Server.GetAddress())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down mock error service...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Mock error service stopped")
}

// addSampleRules 添加示例规则
func addSampleRules(ctx context.Context, service *service.ErrorInjectorService, logger *logger.Logger) {
	logger.Info("Adding sample error injection rules for development")

	// 示例规则1: 存储服务随机错误
	delay1 := 500 * time.Millisecond
	rule1 := &models.ErrorRule{
		Name:        "Storage Service Random Error",
		Description: "Randomly inject 500 errors into storage service operations",
		Service:     "storage-service",
		Enabled:     true,
		Priority:    1,
		Conditions: []models.ErrorCondition{
			{
				Type:     models.ErrorConditionTypeProbability,
				Operator: "eq",
				Value:    0.1, // 10% 概率
			},
		},
		Action: models.ErrorAction{
			Type:     models.ErrorActionTypeHTTPError,
			HTTPCode: 500,
			Message:  "Internal server error injected for testing",
			Delay:    &delay1,
		},
	}

	// 示例规则2: 元数据服务延迟
	delay2 := 2 * time.Second
	rule2 := &models.ErrorRule{
		Name:        "Metadata Service Delay",
		Description: "Add delay to metadata service operations",
		Service:     "metadata-service",
		Operation:   "GetMetadata",
		Enabled:     true,
		Priority:    2,
		Conditions: []models.ErrorCondition{
			{
				Type:     models.ErrorConditionTypeProbability,
				Operator: "eq",
				Value:    0.2, // 20% 概率
			},
		},
		Action: models.ErrorAction{
			Type:  models.ErrorActionTypeDelay,
			Delay: &delay2,
		},
	}

	// 示例规则3: 队列服务网络错误
	rule3 := &models.ErrorRule{
		Name:        "Queue Service Network Error",
		Description: "Inject network errors into queue service",
		Service:     "queue-service",
		Enabled:     false, // 默认禁用
		Priority:    3,
		MaxTriggers: 10, // 最多触发10次
		Conditions: []models.ErrorCondition{
			{
				Type:     models.ErrorConditionTypeProbability,
				Operator: "eq",
				Value:    0.05, // 5% 概率
			},
		},
		Action: models.ErrorAction{
			Type:    models.ErrorActionTypeNetworkError,
			Message: "Network timeout injected",
		},
	}

	// 添加规则
	rules := []*models.ErrorRule{rule1, rule2, rule3}
	for _, rule := range rules {
		if err := service.AddErrorRule(ctx, rule); err != nil {
			logger.Warn("Failed to add sample rule", "rule_name", rule.Name, "error", err)
		} else {
			logger.Info("Added sample rule", "rule_name", rule.Name, "enabled", rule.Enabled)
		}
	}
}
