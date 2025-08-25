package observability

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// HTTPMiddleware HTTP中间件
type HTTPMiddleware struct {
	collector *MetricCollector
	logger    *Logger
}

// NewHTTPMiddleware 创建HTTP中间件
func NewHTTPMiddleware(collector *MetricCollector, logger *Logger) *HTTPMiddleware {
	return &HTTPMiddleware{
		collector: collector,
		logger:    logger,
	}
}

// GinMetricsMiddleware 返回Gin指标中间件
func (m *HTTPMiddleware) GinMetricsMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		start := time.Now()

		// 增加活跃连接数
		m.collector.IncrementActiveConnections(c.Request.Context())
		defer m.collector.DecrementActiveConnections(c.Request.Context())

		// 处理请求
		c.Next()

		// 记录指标
		duration := time.Since(start)
		
		requestSize := int64(0)
		if c.Request.ContentLength > 0 {
			requestSize = c.Request.ContentLength
		}

		responseSize := int64(c.Writer.Size())

		m.collector.RecordHTTPRequest(
			c.Request.Context(),
			c.Request.Method,
			c.FullPath(),
			c.Writer.Status(),
			duration,
			requestSize,
			responseSize,
		)

		// 记录错误
		if c.Writer.Status() >= 400 {
			errorType := "client_error"
			if c.Writer.Status() >= 500 {
				errorType = "server_error"
			}
			m.collector.RecordError(c.Request.Context(), errorType)
		}

		// 记录访问日志
		m.logger.Info(c.Request.Context(), "HTTP request",
			String("method", c.Request.Method),
			String("path", c.FullPath()),
			String("remote_addr", c.ClientIP()),
			String("user_agent", c.Request.UserAgent()),
			Int("status", c.Writer.Status()),
			Duration("duration", duration),
			Int64("request_size", requestSize),
			Int64("response_size", responseSize),
		)
	})
}

// GinTracingMiddleware 返回Gin追踪中间件
func (m *HTTPMiddleware) GinTracingMiddleware() gin.HandlerFunc {
	return otelgin.Middleware("http-server")
}

// GinRecoveryMiddleware 返回Gin恢复中间件
func (m *HTTPMiddleware) GinRecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered interface{}) {
		// 记录panic
		m.logger.Error(c.Request.Context(), "Request panic recovered",
			String("panic", recovered.(error).Error()),
			String("method", c.Request.Method),
			String("path", c.FullPath()),
			String("remote_addr", c.ClientIP()),
		)

		// 记录错误指标
		m.collector.RecordError(c.Request.Context(), "panic")

		// 返回500错误
		c.AbortWithStatusJSON(500, gin.H{
			"error":   "Internal Server Error",
			"message": "The server encountered an internal error and was unable to complete your request",
		})
	})
}

// GinCORSMiddleware CORS中间件
func (m *HTTPMiddleware) GinCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}