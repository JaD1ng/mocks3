package main

import (
	"context"
	"fmt"
	"mocks3/services/chaos-engineering/internal/config"
	"mocks3/services/chaos-engineering/internal/handler"
	"mocks3/services/chaos-engineering/internal/injector"
	"mocks3/services/chaos-engineering/internal/rules"
	"mocks3/shared/logger"
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

	// 初始化规则管理器
	ruleManager := rules.NewRuleManager()

	// 加载默认规则
	err := ruleManager.LoadDefaultRules()
	if err != nil {
		logger.Error("Warning: Failed to load default rules", err)
	}

	// 初始化故障注入器
	chaosInjector := injector.NewChaosInjector(ruleManager)

	// 启动故障注入器
	err = chaosInjector.Start()
	if err != nil {
		logger.Fatal("Failed to start chaos injector", err)
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
	h := handler.NewChaosHandler(ruleManager, chaosInjector)

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
		logger.Infof("混沌工程服务启动在端口 %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("服务启动失败", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("正在关闭混沌工程服务...")

	// 停止故障注入器
	chaosInjector.Stop()

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

	logger.Info("混沌工程服务已关闭")
}

// setupRoutes 设置路由
func setupRoutes(router *gin.Engine, h *handler.ChaosHandler) {
	// 健康检查
	router.GET("/health", h.HealthCheck)

	// 规则管理
	router.GET("/rules", h.GetRules)
	router.POST("/rules", h.CreateRule)
	router.GET("/rules/:ruleId", h.GetRule)
	router.PUT("/rules/:ruleId", h.UpdateRule)
	router.DELETE("/rules/:ruleId", h.DeleteRule)

	// 规则控制
	router.POST("/rules/:ruleId/enable", h.EnableRule)
	router.POST("/rules/:ruleId/disable", h.DisableRule)

	// 故障注入控制
	router.POST("/chaos/start", h.StartChaos)
	router.POST("/chaos/stop", h.StopChaos)
	router.GET("/chaos/status", h.GetChaosStatus)

	// 统计信息
	router.GET("/stats", h.GetStats)
	router.GET("/logs", h.GetLogs)
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
