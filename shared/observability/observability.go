package observability

import (
	"context"
	"fmt"

	"mocks3/shared/utils"

	"github.com/gin-gonic/gin"
)

// Config 简化的可观测性配置
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	LogLevel       string
}

// Observability 统一的可观测性实例
type Observability struct {
	providers  *Providers
	logger     *Logger
	collector  *MetricCollector
	middleware *HTTPMiddleware
}

// New 创建可观测性实例
func New(ctx context.Context, config *Config) (*Observability, error) {
	// 转换配置格式
	utilsConfig := &utils.Config{
		ServiceName:    config.ServiceName,
		ServiceVersion: config.ServiceVersion,
		Environment:    config.Environment,
		OTLPEndpoint:   config.OTLPEndpoint,
		LogLevel:       config.LogLevel,
		SamplingRatio:  1.0,
		ExportInterval: 30_000_000_000, // 30 seconds in nanoseconds
	}

	// 创建providers
	providers, err := NewProviders(utilsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create providers: %w", err)
	}

	// 创建指标收集器
	collector, err := NewMetricCollector(providers.Meter, providers.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric collector: %w", err)
	}

	// 创建HTTP中间件
	httpMiddleware := NewHTTPMiddleware(collector, providers.Logger)

	obs := &Observability{
		providers:  providers,
		logger:     providers.Logger,
		collector:  collector,
		middleware: httpMiddleware,
	}

	// 启动系统指标收集
	go collector.RecordSystemMetrics(ctx)

	return obs, nil
}

// Logger 获取日志器
func (o *Observability) Logger() *Logger {
	return o.logger
}

// Tracer 获取追踪器
func (o *Observability) Tracer() interface{} {
	return o.providers.Tracer
}

// Meter 获取指标器
func (o *Observability) Meter() interface{} {
	return o.providers.Meter
}

// GinMiddleware 获取Gin中间件
func (o *Observability) GinMiddleware() gin.HandlerFunc {
	return o.middleware.GinMetricsMiddleware()
}

// Shutdown 关闭可观测性组件
func (o *Observability) Shutdown(ctx context.Context) error {
	return o.providers.Shutdown(ctx)
}