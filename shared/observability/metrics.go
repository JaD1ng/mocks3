package observability

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MetricCollector 指标收集器
type MetricCollector struct {
	meter  metric.Meter
	logger *Logger

	// HTTP 指标
	httpRequestsTotal    metric.Int64Counter
	httpRequestDuration  metric.Float64Histogram
	httpRequestSize      metric.Int64Histogram
	httpResponseSize     metric.Int64Histogram

	// 系统指标
	memoryUsage     metric.Float64ObservableGauge
	cpuUsage        metric.Float64ObservableGauge
	goroutineCount  metric.Int64ObservableGauge
	gcDuration      metric.Float64Histogram

	// 业务指标
	activeConnections metric.Int64UpDownCounter
	queueSize        metric.Int64ObservableGauge
	errorCount       metric.Int64Counter
}

// NewMetricCollector 创建指标收集器
func NewMetricCollector(meter metric.Meter, logger *Logger) (*MetricCollector, error) {
	collector := &MetricCollector{
		meter:  meter,
		logger: logger,
	}

	var err error

	// 初始化HTTP指标
	if collector.httpRequestsTotal, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	); err != nil {
		return nil, fmt.Errorf("failed to create http_requests_total counter: %w", err)
	}

	if collector.httpRequestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, fmt.Errorf("failed to create http_request_duration histogram: %w", err)
	}

	if collector.httpRequestSize, err = meter.Int64Histogram(
		"http_request_size_bytes",
		metric.WithDescription("HTTP request size in bytes"),
		metric.WithUnit("By"),
	); err != nil {
		return nil, fmt.Errorf("failed to create http_request_size histogram: %w", err)
	}

	if collector.httpResponseSize, err = meter.Int64Histogram(
		"http_response_size_bytes",
		metric.WithDescription("HTTP response size in bytes"),
		metric.WithUnit("By"),
	); err != nil {
		return nil, fmt.Errorf("failed to create http_response_size histogram: %w", err)
	}

	// 初始化系统指标
	if collector.memoryUsage, err = meter.Float64ObservableGauge(
		"memory_usage_bytes",
		metric.WithDescription("Memory usage in bytes"),
		metric.WithUnit("By"),
	); err != nil {
		return nil, fmt.Errorf("failed to create memory_usage gauge: %w", err)
	}

	if collector.cpuUsage, err = meter.Float64ObservableGauge(
		"cpu_usage_percent",
		metric.WithDescription("CPU usage percentage"),
		metric.WithUnit("%"),
	); err != nil {
		return nil, fmt.Errorf("failed to create cpu_usage gauge: %w", err)
	}

	if collector.goroutineCount, err = meter.Int64ObservableGauge(
		"goroutine_count",
		metric.WithDescription("Number of goroutines"),
	); err != nil {
		return nil, fmt.Errorf("failed to create goroutine_count gauge: %w", err)
	}

	if collector.gcDuration, err = meter.Float64Histogram(
		"gc_duration_seconds",
		metric.WithDescription("GC duration in seconds"),
		metric.WithUnit("s"),
	); err != nil {
		return nil, fmt.Errorf("failed to create gc_duration histogram: %w", err)
	}

	// 初始化业务指标
	if collector.activeConnections, err = meter.Int64UpDownCounter(
		"active_connections",
		metric.WithDescription("Number of active connections"),
	); err != nil {
		return nil, fmt.Errorf("failed to create active_connections counter: %w", err)
	}

	if collector.queueSize, err = meter.Int64ObservableGauge(
		"queue_size",
		metric.WithDescription("Current queue size"),
	); err != nil {
		return nil, fmt.Errorf("failed to create queue_size gauge: %w", err)
	}

	if collector.errorCount, err = meter.Int64Counter(
		"errors_total",
		metric.WithDescription("Total number of errors"),
	); err != nil {
		return nil, fmt.Errorf("failed to create errors_total counter: %w", err)
	}

	return collector, nil
}

// RecordHTTPRequest 记录HTTP请求指标
func (c *MetricCollector) RecordHTTPRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration, requestSize, responseSize int64) {
	labels := metric.WithAttributes(
		attribute.String("method", method),
		attribute.String("path", path),
		attribute.Int("status_code", statusCode),
	)

	c.httpRequestsTotal.Add(ctx, 1, labels)
	c.httpRequestDuration.Record(ctx, duration.Seconds(), labels)
	
	if requestSize > 0 {
		c.httpRequestSize.Record(ctx, requestSize, labels)
	}
	if responseSize > 0 {
		c.httpResponseSize.Record(ctx, responseSize, labels)
	}
}

// RecordError 记录错误
func (c *MetricCollector) RecordError(ctx context.Context, errorType string) {
	c.errorCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("error_type", errorType),
	))
}

// IncrementActiveConnections 增加活跃连接数
func (c *MetricCollector) IncrementActiveConnections(ctx context.Context) {
	c.activeConnections.Add(ctx, 1)
}

// DecrementActiveConnections 减少活跃连接数
func (c *MetricCollector) DecrementActiveConnections(ctx context.Context) {
	c.activeConnections.Add(ctx, -1)
}

// RecordSystemMetrics 记录系统指标
func (c *MetricCollector) RecordSystemMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 注册观测回调
	_, err := c.meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// 记录内存使用
			observer.ObserveFloat64(c.memoryUsage, float64(m.Alloc))

			// 记录Goroutine数量
			observer.ObserveInt64(c.goroutineCount, int64(runtime.NumGoroutine()))

			return nil
		},
		c.memoryUsage,
		c.goroutineCount,
	)

	if err != nil {
		c.logger.Error(ctx, "Failed to register system metrics callback", Error(err))
		return
	}

	// 周期性记录其他指标
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 这里可以添加其他周期性指标收集
		}
	}
}