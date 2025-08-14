package middleware

import (
	"time"

	"micro-s3/shared/logger"
	"github.com/gin-gonic/gin"
)

// Logger 日志中间件
func Logger() gin.HandlerFunc {
	return LoggerWithConfig(logger.DefaultLogger)
}

// LoggerWithConfig 使用指定日志器的中间件
func LoggerWithConfig(l logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 获取状态码
		status := c.Writer.Status()

		// 构建查询字符串
		if raw != "" {
			path = path + "?" + raw
		}

		// 记录请求日志
		fields := map[string]any{
			"method":     c.Request.Method,
			"path":       path,
			"status":     status,
			"latency":    latency.String(),
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}

		// 根据状态码选择日志级别
		message := "HTTP request processed"
		if status >= 400 && status < 500 {
			l.Warn(c.Request.Context(), message, fields)
		} else if status >= 500 {
			l.Error(c.Request.Context(), message, nil, fields)
		} else {
			l.Info(c.Request.Context(), message, fields)
		}
	}
}

// Recovery 恢复中间件
func Recovery() gin.HandlerFunc {
	return RecoveryWithConfig(logger.DefaultLogger)
}

// RecoveryWithConfig 使用指定日志器的恢复中间件
func RecoveryWithConfig(l logger.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		// 记录panic错误
		fields := map[string]any{
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"client_ip": c.ClientIP(),
			"panic":     recovered,
		}
		
		l.Error(c.Request.Context(), "Panic recovered in HTTP handler", 
			nil, fields)
		
		c.JSON(500, gin.H{
			"error": "Internal server error",
		})
	})
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, HEAD, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestID 请求ID中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString 生成随机字符串
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
