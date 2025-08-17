package metric

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// Collector 指标收集器
type Collector struct {
	provider   *sdkmetric.MeterProvider
	meter      metric.Meter
	counters   map[string]metric.Int64Counter
	histograms map[string]metric.Float64Histogram
	gauges     map[string]metric.Int64UpDownCounter
}

// CollectorConfig 收集器配置
type CollectorConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	ExportInterval time.Duration
}

// NewCollector 创建新的指标收集器
func NewCollector(config *CollectorConfig) (*Collector, error) {
	// 创建资源
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建OTLP HTTP导出器
	exporter, err := otlpmetrichttp.New(context.Background(),
		otlpmetrichttp.WithEndpoint(config.OTLPEndpoint),
		otlpmetrichttp.WithInsecure(), // 开发环境使用HTTP
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// 创建指标提供者
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(config.ExportInterval),
		)),
	)

	// 设置全局指标提供者
	otel.SetMeterProvider(provider)

	meter := provider.Meter(config.ServiceName)

	return &Collector{
		provider:   provider,
		meter:      meter,
		counters:   make(map[string]metric.Int64Counter),
		histograms: make(map[string]metric.Float64Histogram),
		gauges:     make(map[string]metric.Int64UpDownCounter),
	}, nil
}

// GetMeter 获取计量器
func (c *Collector) GetMeter() metric.Meter {
	return c.meter
}

// Shutdown 关闭收集器
func (c *Collector) Shutdown(ctx context.Context) error {
	return c.provider.Shutdown(ctx)
}

// CreateCounter 创建计数器
func (c *Collector) CreateCounter(name, description string) (metric.Int64Counter, error) {
	if counter, exists := c.counters[name]; exists {
		return counter, nil
	}

	counter, err := c.meter.Int64Counter(name,
		metric.WithDescription(description),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create counter %s: %w", name, err)
	}

	c.counters[name] = counter
	return counter, nil
}

// CreateHistogram 创建直方图
func (c *Collector) CreateHistogram(name, description, unit string) (metric.Float64Histogram, error) {
	if histogram, exists := c.histograms[name]; exists {
		return histogram, nil
	}

	histogram, err := c.meter.Float64Histogram(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create histogram %s: %w", name, err)
	}

	c.histograms[name] = histogram
	return histogram, nil
}

// CreateGauge 创建仪表
func (c *Collector) CreateGauge(name, description string) (metric.Int64UpDownCounter, error) {
	if gauge, exists := c.gauges[name]; exists {
		return gauge, nil
	}

	gauge, err := c.meter.Int64UpDownCounter(name,
		metric.WithDescription(description),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gauge %s: %w", name, err)
	}

	c.gauges[name] = gauge
	return gauge, nil
}

// IncCounter 增加计数器
func (c *Collector) IncCounter(ctx context.Context, name string, value int64, attributes ...metric.AddOption) error {
	counter, exists := c.counters[name]
	if !exists {
		return fmt.Errorf("counter %s not found", name)
	}

	counter.Add(ctx, value, attributes...)
	return nil
}

// RecordHistogram 记录直方图值
func (c *Collector) RecordHistogram(ctx context.Context, name string, value float64, attributes ...metric.RecordOption) error {
	histogram, exists := c.histograms[name]
	if !exists {
		return fmt.Errorf("histogram %s not found", name)
	}

	histogram.Record(ctx, value, attributes...)
	return nil
}

// SetGauge 设置仪表值
func (c *Collector) SetGauge(ctx context.Context, name string, value int64, attributes ...metric.AddOption) error {
	gauge, exists := c.gauges[name]
	if !exists {
		return fmt.Errorf("gauge %s not found", name)
	}

	gauge.Add(ctx, value, attributes...)
	return nil
}

// RecordDuration 记录持续时间
func (c *Collector) RecordDuration(ctx context.Context, name string, duration time.Duration, attributes ...metric.RecordOption) error {
	return c.RecordHistogram(ctx, name, float64(duration.Milliseconds()), attributes...)
}

// NewDefaultCollector 创建默认的指标收集器
func NewDefaultCollector(serviceName string) (*Collector, error) {
	config := &CollectorConfig{
		ServiceName:    serviceName,
		ServiceVersion: getEnv("SERVICE_VERSION", "1.0.0"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		OTLPEndpoint:   getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
		ExportInterval: 30 * time.Second,
	}

	collector, err := NewCollector(config)
	if err != nil {
		return nil, err
	}

	// 创建常用指标
	err = collector.initCommonMetrics()
	if err != nil {
		return nil, err
	}

	return collector, nil
}

// initCommonMetrics 初始化常用指标
func (c *Collector) initCommonMetrics() error {
	// HTTP 请求计数器
	_, err := c.CreateCounter("http_requests_total", "Total number of HTTP requests")
	if err != nil {
		return err
	}

	// HTTP 请求持续时间直方图
	_, err = c.CreateHistogram("http_request_duration_seconds", "HTTP request duration in seconds", "s")
	if err != nil {
		return err
	}

	// HTTP 请求大小直方图
	_, err = c.CreateHistogram("http_request_size_bytes", "HTTP request size in bytes", "byte")
	if err != nil {
		return err
	}

	// HTTP 响应大小直方图
	_, err = c.CreateHistogram("http_response_size_bytes", "HTTP response size in bytes", "byte")
	if err != nil {
		return err
	}

	// 活跃连接数仪表
	_, err = c.CreateGauge("http_active_connections", "Number of active HTTP connections")
	if err != nil {
		return err
	}

	// 队列长度仪表
	_, err = c.CreateGauge("queue_length", "Number of items in queue")
	if err != nil {
		return err
	}

	// 处理的任务计数器
	_, err = c.CreateCounter("tasks_processed_total", "Total number of processed tasks")
	if err != nil {
		return err
	}

	// 数据库操作计数器
	_, err = c.CreateCounter("database_operations_total", "Total number of database operations")
	if err != nil {
		return err
	}

	// 数据库操作持续时间直方图
	_, err = c.CreateHistogram("database_operation_duration_seconds", "Database operation duration in seconds", "s")
	if err != nil {
		return err
	}

	return nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
