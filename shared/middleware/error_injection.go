package middleware

import (
	"context"
	"fmt"
	"math/rand"
	"mocks3/shared/interfaces"
	"mocks3/shared/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorInjectionMiddleware 错误注入中间件
type ErrorInjectionMiddleware struct {
	injectorService interfaces.ErrorInjectorService
	enabled         bool
}

// NewErrorInjectionMiddleware 创建错误注入中间件
func NewErrorInjectionMiddleware(injectorService interfaces.ErrorInjectorService) *ErrorInjectionMiddleware {
	return &ErrorInjectionMiddleware{
		injectorService: injectorService,
		enabled:         true,
	}
}

// GinMiddleware 返回Gin中间件
func (m *ErrorInjectionMiddleware) GinMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.enabled {
			c.Next()
			return
		}

		// 提取操作名
		operation := m.extractOperation(c)

		// 检查是否应该注入错误
		action, shouldInject := m.injectorService.ShouldInjectError(c.Request.Context(), serviceName, operation)
		if !shouldInject {
			c.Next()
			return
		}

		// 注入错误
		if m.injectError(c, action) {
			return // 错误已注入，停止处理
		}

		c.Next()
	}
}

// HTTPMiddleware 返回标准HTTP中间件
func (m *ErrorInjectionMiddleware) HTTPMiddleware(serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !m.enabled {
				next.ServeHTTP(w, r)
				return
			}

			// 提取操作名
			operation := m.extractOperationFromRequest(r)

			// 检查是否应该注入错误
			action, shouldInject := m.injectorService.ShouldInjectError(r.Context(), serviceName, operation)
			if !shouldInject {
				next.ServeHTTP(w, r)
				return
			}

			// 注入错误
			if m.injectHTTPError(w, r, action) {
				return // 错误已注入，停止处理
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Enable 启用错误注入
func (m *ErrorInjectionMiddleware) Enable() {
	m.enabled = true
}

// Disable 禁用错误注入
func (m *ErrorInjectionMiddleware) Disable() {
	m.enabled = false
}

// IsEnabled 检查是否启用
func (m *ErrorInjectionMiddleware) IsEnabled() bool {
	return m.enabled
}

// extractOperation 从Gin上下文提取操作名
func (m *ErrorInjectionMiddleware) extractOperation(c *gin.Context) string {
	// 使用路由路径作为操作名
	if path := c.FullPath(); path != "" {
		return fmt.Sprintf("%s %s", c.Request.Method, path)
	}
	return fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
}

// extractOperationFromRequest 从HTTP请求提取操作名
func (m *ErrorInjectionMiddleware) extractOperationFromRequest(r *http.Request) string {
	return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
}

// injectError 在Gin上下文中注入错误
func (m *ErrorInjectionMiddleware) injectError(c *gin.Context, action *models.ErrorAction) bool {
	switch action.Type {
	case models.ErrorActionTypeHTTPError:
		return m.injectHTTPErrorGin(c, action)
	case models.ErrorActionTypeDelay:
		return m.injectDelay(c, action)
	case models.ErrorActionTypeTimeout:
		return m.injectTimeout(c, action)
	case models.ErrorActionTypeCorruption:
		return m.injectCorruption(c, action)
	default:
		return false
	}
}

// injectHTTPError 在标准HTTP中注入错误
func (m *ErrorInjectionMiddleware) injectHTTPError(w http.ResponseWriter, r *http.Request, action *models.ErrorAction) bool {
	switch action.Type {
	case models.ErrorActionTypeHTTPError:
		return m.injectHTTPErrorStandard(w, r, action)
	case models.ErrorActionTypeDelay:
		return m.injectDelayStandard(w, r, action)
	case models.ErrorActionTypeTimeout:
		return m.injectTimeoutStandard(w, r, action)
	default:
		return false
	}
}

// injectHTTPErrorGin 注入HTTP错误到Gin
func (m *ErrorInjectionMiddleware) injectHTTPErrorGin(c *gin.Context, action *models.ErrorAction) bool {
	statusCode := action.HTTPCode
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}

	// 设置自定义响应头
	for key, value := range action.Headers {
		c.Header(key, value)
	}

	// 设置错误响应
	if action.Body != "" {
		c.String(statusCode, action.Body)
	} else if action.Message != "" {
		c.JSON(statusCode, gin.H{
			"error":    action.Message,
			"code":     statusCode,
			"injected": true,
		})
	} else {
		c.JSON(statusCode, gin.H{
			"error":    "Injected error",
			"code":     statusCode,
			"injected": true,
		})
	}

	c.Abort()
	return true
}

// injectHTTPErrorStandard 注入HTTP错误到标准HTTP
func (m *ErrorInjectionMiddleware) injectHTTPErrorStandard(w http.ResponseWriter, r *http.Request, action *models.ErrorAction) bool {
	statusCode := action.HTTPCode
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}

	// 设置自定义响应头
	for key, value := range action.Headers {
		w.Header().Set(key, value)
	}

	w.WriteHeader(statusCode)

	// 写入响应体
	if action.Body != "" {
		w.Write([]byte(action.Body))
	} else if action.Message != "" {
		w.Write([]byte(fmt.Sprintf(`{"error": "%s", "code": %d, "injected": true}`, action.Message, statusCode)))
	} else {
		w.Write([]byte(fmt.Sprintf(`{"error": "Injected error", "code": %d, "injected": true}`, statusCode)))
	}

	return true
}

// injectDelay 注入延迟
func (m *ErrorInjectionMiddleware) injectDelay(c *gin.Context, action *models.ErrorAction) bool {
	if action.Delay == nil {
		return false
	}

	time.Sleep(*action.Delay)
	return false // 继续处理请求
}

// injectDelayStandard 在标准HTTP中注入延迟
func (m *ErrorInjectionMiddleware) injectDelayStandard(w http.ResponseWriter, r *http.Request, action *models.ErrorAction) bool {
	if action.Delay == nil {
		return false
	}

	time.Sleep(*action.Delay)
	return false // 继续处理请求
}

// injectTimeout 注入超时
func (m *ErrorInjectionMiddleware) injectTimeout(c *gin.Context, action *models.ErrorAction) bool {
	if action.Delay == nil {
		return false
	}

	// 模拟超时：延迟然后返回超时错误
	time.Sleep(*action.Delay)

	c.JSON(http.StatusRequestTimeout, gin.H{
		"error":    "Request timeout (injected)",
		"code":     http.StatusRequestTimeout,
		"injected": true,
	})
	c.Abort()
	return true
}

// injectTimeoutStandard 在标准HTTP中注入超时
func (m *ErrorInjectionMiddleware) injectTimeoutStandard(w http.ResponseWriter, r *http.Request, action *models.ErrorAction) bool {
	if action.Delay == nil {
		return false
	}

	// 模拟超时：延迟然后返回超时错误
	time.Sleep(*action.Delay)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusRequestTimeout)
	w.Write([]byte(`{"error": "Request timeout (injected)", "code": 408, "injected": true}`))
	return true
}

// injectCorruption 注入数据损坏
func (m *ErrorInjectionMiddleware) injectCorruption(c *gin.Context, action *models.ErrorAction) bool {
	// 这是一个复杂的错误类型，需要在响应中随机修改数据
	// 这里提供一个基本实现，实际使用时可能需要更复杂的逻辑

	// 在响应写入器中注入损坏
	originalWriter := c.Writer
	c.Writer = &corruptedResponseWriter{
		ResponseWriter: originalWriter,
		corruptionRate: 0.1, // 10%的字节损坏率
	}

	return false // 继续处理请求
}

// corruptedResponseWriter 损坏的响应写入器
type corruptedResponseWriter struct {
	gin.ResponseWriter
	corruptionRate float64
}

func (w *corruptedResponseWriter) Write(data []byte) (int, error) {
	// 随机损坏一些字节
	corrupted := make([]byte, len(data))
	copy(corrupted, data)

	for i := range corrupted {
		if rand.Float64() < w.corruptionRate {
			corrupted[i] = byte(rand.Intn(256))
		}
	}

	return w.ResponseWriter.Write(corrupted)
}

// DatabaseErrorInjector 数据库错误注入器
type DatabaseErrorInjector struct {
	injectorService interfaces.ErrorInjectorService
	serviceName     string
}

// NewDatabaseErrorInjector 创建数据库错误注入器
func NewDatabaseErrorInjector(injectorService interfaces.ErrorInjectorService, serviceName string) *DatabaseErrorInjector {
	return &DatabaseErrorInjector{
		injectorService: injectorService,
		serviceName:     serviceName,
	}
}

// ShouldInjectError 检查是否应该注入数据库错误
func (d *DatabaseErrorInjector) ShouldInjectError(ctx context.Context, operation string) error {
	action, shouldInject := d.injectorService.ShouldInjectError(ctx, d.serviceName, operation)
	if !shouldInject {
		return nil
	}

	switch action.Type {
	case models.ErrorActionTypeDatabaseError:
		if action.Message != "" {
			return fmt.Errorf("database error (injected): %s", action.Message)
		}
		return fmt.Errorf("database connection failed (injected)")
	case models.ErrorActionTypeTimeout:
		if action.Delay != nil {
			time.Sleep(*action.Delay)
		}
		return fmt.Errorf("database operation timeout (injected)")
	default:
		return nil
	}
}

// StorageErrorInjector 存储错误注入器
type StorageErrorInjector struct {
	injectorService interfaces.ErrorInjectorService
	serviceName     string
}

// NewStorageErrorInjector 创建存储错误注入器
func NewStorageErrorInjector(injectorService interfaces.ErrorInjectorService, serviceName string) *StorageErrorInjector {
	return &StorageErrorInjector{
		injectorService: injectorService,
		serviceName:     serviceName,
	}
}

// ShouldInjectError 检查是否应该注入存储错误
func (s *StorageErrorInjector) ShouldInjectError(ctx context.Context, operation string) error {
	action, shouldInject := s.injectorService.ShouldInjectError(ctx, s.serviceName, operation)
	if !shouldInject {
		return nil
	}

	switch action.Type {
	case models.ErrorActionTypeStorageError:
		if action.Message != "" {
			return fmt.Errorf("storage error (injected): %s", action.Message)
		}
		return fmt.Errorf("storage operation failed (injected)")
	case models.ErrorActionTypeTimeout:
		if action.Delay != nil {
			time.Sleep(*action.Delay)
		}
		return fmt.Errorf("storage operation timeout (injected)")
	case models.ErrorActionTypeCorruption:
		return fmt.Errorf("storage data corruption detected (injected)")
	default:
		return nil
	}
}
