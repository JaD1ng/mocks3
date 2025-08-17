package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// RecoveryConfig 恢复中间件配置
type RecoveryConfig struct {
	EnableStackTrace bool
	EnableLogging    bool
	CustomHandler    func(*gin.Context, interface{})
}

// DefaultRecoveryConfig 默认恢复配置
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		EnableStackTrace: true,
		EnableLogging:    true,
		CustomHandler:    nil,
	}
}

// GinRecoveryMiddleware Gin恢复中间件
func GinRecoveryMiddleware(config *RecoveryConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if config.EnableLogging {
			log.Printf("Panic recovered: %v", recovered)
			if config.EnableStackTrace {
				log.Printf("Stack trace:\n%s", debug.Stack())
			}
		}

		if config.CustomHandler != nil {
			config.CustomHandler(c, recovered)
			return
		}

		// 默认处理
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "An unexpected error occurred",
		})
		c.Abort()
	})
}

// HTTPRecoveryMiddleware 标准HTTP恢复中间件
func HTTPRecoveryMiddleware(config *RecoveryConfig) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					if config.EnableLogging {
						log.Printf("Panic recovered: %v", recovered)
						if config.EnableStackTrace {
							log.Printf("Stack trace:\n%s", debug.Stack())
						}
					}

					// 设置响应
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "Internal Server Error", "message": "An unexpected error occurred"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingRecoveryHandler 记录panic的恢复处理器
func LoggingRecoveryHandler(enableStackTrace bool) func(*gin.Context, interface{}) {
	return func(c *gin.Context, recovered interface{}) {
		log.Printf("Panic recovered in %s %s: %v", c.Request.Method, c.Request.URL.Path, recovered)
		
		if enableStackTrace {
			log.Printf("Stack trace:\n%s", debug.Stack())
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     "Internal Server Error",
			"message":   "An unexpected error occurred",
			"path":      c.Request.URL.Path,
			"method":    c.Request.Method,
			"timestamp": gin.H{"error": fmt.Sprintf("%v", recovered)},
		})
		c.Abort()
	}
}

// DetailedRecoveryHandler 详细的恢复处理器（开发环境使用）
func DetailedRecoveryHandler() func(*gin.Context, interface{}) {
	return func(c *gin.Context, recovered interface{}) {
		log.Printf("Panic recovered: %v", recovered)
		log.Printf("Stack trace:\n%s", debug.Stack())

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Internal Server Error",
			"message":    "An unexpected error occurred",
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
			"panic":      fmt.Sprintf("%v", recovered),
			"stack":      string(debug.Stack()),
			"headers":    c.Request.Header,
		})
		c.Abort()
	}
}

// ProductionRecoveryHandler 生产环境恢复处理器
func ProductionRecoveryHandler() func(*gin.Context, interface{}) {
	return func(c *gin.Context, recovered interface{}) {
		// 只记录错误，不暴露敏感信息
		log.Printf("Panic recovered: %v", recovered)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "An unexpected error occurred",
		})
		c.Abort()
	}
}