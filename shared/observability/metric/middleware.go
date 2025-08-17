package metric

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	Collector       *Collector
	RequestDuration string
	RequestSize     string
	ResponseSize    string
	RequestsTotal   string
	ActiveRequests  string
}

// NewDefaultMiddlewareConfig 创建默认中间件配置
func NewDefaultMiddlewareConfig(collector *Collector) *MiddlewareConfig {
	return &MiddlewareConfig{
		Collector:       collector,
		RequestDuration: "http_request_duration_seconds",
		RequestSize:     "http_request_size_bytes",
		ResponseSize:    "http_response_size_bytes",
		RequestsTotal:   "http_requests_total",
		ActiveRequests:  "http_active_connections",
	}
}

// GinMiddleware 返回Gin的指标中间件
func (config *MiddlewareConfig) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 增加活跃请求数
		config.Collector.SetGauge(c.Request.Context(), config.ActiveRequests, 1,
			metric.WithAttributes(
				attribute.String("method", c.Request.Method),
				attribute.String("handler", c.HandlerName()),
			),
		)

		// 记录请求大小
		if c.Request.ContentLength > 0 {
			config.Collector.RecordHistogram(c.Request.Context(), config.RequestSize, float64(c.Request.ContentLength),
				metric.WithAttributes(
					attribute.String("method", c.Request.Method),
					attribute.String("endpoint", c.FullPath()),
				),
			)
		}

		// 处理请求
		c.Next()

		// 计算持续时间
		duration := time.Since(start)

		// 获取状态码
		statusCode := c.Writer.Status()
		statusClass := getStatusClass(statusCode)

		// 记录请求计数
		config.Collector.IncCounter(c.Request.Context(), config.RequestsTotal, 1,
			metric.WithAttributes(
				attribute.String("method", c.Request.Method),
				attribute.String("endpoint", c.FullPath()),
				attribute.String("status_code", strconv.Itoa(statusCode)),
				attribute.String("status_class", statusClass),
			),
		)

		// 记录请求持续时间
		config.Collector.RecordDuration(c.Request.Context(), config.RequestDuration, duration,
			metric.WithAttributes(
				attribute.String("method", c.Request.Method),
				attribute.String("endpoint", c.FullPath()),
				attribute.String("status_code", strconv.Itoa(statusCode)),
				attribute.String("status_class", statusClass),
			),
		)

		// 记录响应大小
		responseSize := c.Writer.Size()
		if responseSize > 0 {
			config.Collector.RecordHistogram(c.Request.Context(), config.ResponseSize, float64(responseSize),
				metric.WithAttributes(
					attribute.String("method", c.Request.Method),
					attribute.String("endpoint", c.FullPath()),
					attribute.String("status_code", strconv.Itoa(statusCode)),
				),
			)
		}

		// 减少活跃请求数
		config.Collector.SetGauge(c.Request.Context(), config.ActiveRequests, -1,
			metric.WithAttributes(
				attribute.String("method", c.Request.Method),
				attribute.String("handler", c.HandlerName()),
			),
		)
	}
}

// DatabaseMiddleware 数据库操作指标中间件
type DatabaseMiddleware struct {
	collector *Collector
}

// NewDatabaseMiddleware 创建数据库中间件
func NewDatabaseMiddleware(collector *Collector) *DatabaseMiddleware {
	return &DatabaseMiddleware{
		collector: collector,
	}
}

// RecordOperation 记录数据库操作
func (dm *DatabaseMiddleware) RecordOperation(operation string, duration time.Duration, success bool) {
	// 记录操作计数
	status := "success"
	if !success {
		status = "error"
	}

	dm.collector.IncCounter(nil, "database_operations_total", 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("status", status),
		),
	)

	// 记录操作持续时间
	dm.collector.RecordDuration(nil, "database_operation_duration_seconds", duration,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("status", status),
		),
	)
}

// QueueMiddleware 队列操作指标中间件
type QueueMiddleware struct {
	collector *Collector
}

// NewQueueMiddleware 创建队列中间件
func NewQueueMiddleware(collector *Collector) *QueueMiddleware {
	return &QueueMiddleware{
		collector: collector,
	}
}

// RecordEnqueue 记录入队操作
func (qm *QueueMiddleware) RecordEnqueue(queueName string, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	qm.collector.IncCounter(nil, "queue_enqueue_total", 1,
		metric.WithAttributes(
			attribute.String("queue", queueName),
			attribute.String("status", status),
		),
	)
}

// RecordDequeue 记录出队操作
func (qm *QueueMiddleware) RecordDequeue(queueName string, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	qm.collector.IncCounter(nil, "queue_dequeue_total", 1,
		metric.WithAttributes(
			attribute.String("queue", queueName),
			attribute.String("status", status),
		),
	)
}

// RecordTaskProcessing 记录任务处理
func (qm *QueueMiddleware) RecordTaskProcessing(taskType string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	qm.collector.IncCounter(nil, "tasks_processed_total", 1,
		metric.WithAttributes(
			attribute.String("task_type", taskType),
			attribute.String("status", status),
		),
	)

	qm.collector.RecordDuration(nil, "task_processing_duration_seconds", duration,
		metric.WithAttributes(
			attribute.String("task_type", taskType),
			attribute.String("status", status),
		),
	)
}

// UpdateQueueLength 更新队列长度
func (qm *QueueMiddleware) UpdateQueueLength(queueName string, length int64) {
	qm.collector.SetGauge(nil, "queue_length", length,
		metric.WithAttributes(
			attribute.String("queue", queueName),
		),
	)
}

// StorageMiddleware 存储操作指标中间件
type StorageMiddleware struct {
	collector *Collector
}

// NewStorageMiddleware 创建存储中间件
func NewStorageMiddleware(collector *Collector) *StorageMiddleware {
	return &StorageMiddleware{
		collector: collector,
	}
}

// RecordOperation 记录存储操作
func (sm *StorageMiddleware) RecordOperation(operation string, duration time.Duration, size int64, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	sm.collector.IncCounter(nil, "storage_operations_total", 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("status", status),
		),
	)

	sm.collector.RecordDuration(nil, "storage_operation_duration_seconds", duration,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("status", status),
		),
	)

	if size > 0 {
		sm.collector.RecordHistogram(nil, "storage_operation_size_bytes", float64(size),
			metric.WithAttributes(
				attribute.String("operation", operation),
				attribute.String("status", status),
			),
		)
	}
}

// getStatusClass 获取状态码类别
func getStatusClass(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
