package utils

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int           // 最大重试次数
	InitialDelay    time.Duration // 初始延迟
	MaxDelay        time.Duration // 最大延迟
	BackoffFactor   float64       // 退避因子
	Jitter          bool          // 是否添加随机抖动
	RetryableErrors []string      // 可重试的错误类型
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// RetryFunc 重试函数类型
type RetryFunc func() error

// RetryWithResult 带返回值的重试函数类型
type RetryWithResult[T any] func() (T, error)

// IsRetryable 检查错误是否可重试的函数类型
type IsRetryable func(error) bool

// Retry 执行带重试的操作
func Retry(ctx context.Context, config *RetryConfig, fn RetryFunc) error {
	_, err := RetryWithResultFunc[struct{}](ctx, config, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}

// RetryWithResultFunc 执行带重试的操作并返回结果
func RetryWithResultFunc[T any](ctx context.Context, config *RetryConfig, fn RetryWithResult[T]) (T, error) {
	var zero T

	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// 如果是最后一次尝试，直接返回错误
		if attempt == config.MaxRetries {
			break
		}

		// 计算延迟时间
		delay := calculateDelay(config, attempt)

		// 等待重试
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
	}

	return zero, fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// RetryWithCondition 带条件检查的重试
func RetryWithCondition(ctx context.Context, config *RetryConfig, fn RetryFunc, isRetryable IsRetryable) error {
	_, err := RetryWithResultAndConditionFunc[struct{}](ctx, config, func() (struct{}, error) {
		return struct{}{}, fn()
	}, isRetryable)
	return err
}

// RetryWithResultAndConditionFunc 带条件检查的重试（有返回值）
func RetryWithResultAndConditionFunc[T any](ctx context.Context, config *RetryConfig, fn RetryWithResult[T], isRetryable IsRetryable) (T, error) {
	var zero T

	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// 检查错误是否可重试
		if isRetryable != nil && !isRetryable(err) {
			return zero, fmt.Errorf("non-retryable error: %w", err)
		}

		// 如果是最后一次尝试，直接返回错误
		if attempt == config.MaxRetries {
			break
		}

		// 计算延迟时间
		delay := calculateDelay(config, attempt)

		// 等待重试
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
	}

	return zero, fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// calculateDelay 计算延迟时间
func calculateDelay(config *RetryConfig, attempt int) time.Duration {
	// 指数退避
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt))

	// 应用最大延迟限制
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// 添加随机抖动
	if config.Jitter {
		jitter := delay * 0.1 * (rand.Float64()*2 - 1) // +/- 10%
		delay += jitter
	}

	// 确保延迟为正数
	if delay < 0 {
		delay = float64(config.InitialDelay)
	}

	return time.Duration(delay)
}

// ExponentialBackoff 指数退避重试
func ExponentialBackoff(ctx context.Context, maxRetries int, fn RetryFunc) error {
	config := &RetryConfig{
		MaxRetries:    maxRetries,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
	return Retry(ctx, config, fn)
}

// LinearBackoff 线性退避重试
func LinearBackoff(ctx context.Context, maxRetries int, delay time.Duration, fn RetryFunc) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// 如果是最后一次尝试，直接返回错误
		if attempt == maxRetries {
			break
		}

		// 等待固定延迟
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries+1, lastErr)
}

// ConstantBackoff 固定间隔重试
func ConstantBackoff(ctx context.Context, maxRetries int, delay time.Duration, fn RetryFunc) error {
	return LinearBackoff(ctx, maxRetries, delay, fn)
}

// RetryOnError 基于错误类型的重试
func RetryOnError(ctx context.Context, maxRetries int, retryableErrors []string, fn RetryFunc) error {
	isRetryable := func(err error) bool {
		if err == nil {
			return false
		}

		errMsg := err.Error()
		for _, retryableErr := range retryableErrors {
			if containsString(errMsg, retryableErr) {
				return true
			}
		}
		return false
	}

	config := DefaultRetryConfig()
	config.MaxRetries = maxRetries

	return RetryWithCondition(ctx, config, fn, isRetryable)
}

// containsString 检查字符串是否包含子字符串（忽略大小写）
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexString(s, substr) >= 0))
}

// indexString 查找子字符串索引
func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// RetryDecorator 重试装饰器
type RetryDecorator struct {
	config *RetryConfig
}

// NewRetryDecorator 创建重试装饰器
func NewRetryDecorator(config *RetryConfig) *RetryDecorator {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryDecorator{config: config}
}

// Wrap 包装函数使其具有重试能力
func (rd *RetryDecorator) Wrap(fn RetryFunc) RetryFunc {
	return func() error {
		return Retry(context.Background(), rd.config, fn)
	}
}

// WrapWithContext 包装函数使其具有重试能力（带上下文）
func (rd *RetryDecorator) WrapWithContext(fn func(context.Context) error) func(context.Context) error {
	return func(ctx context.Context) error {
		return Retry(ctx, rd.config, func() error {
			return fn(ctx)
		})
	}
}

// Circuit 熔断器状态
type Circuit struct {
	maxFailures  int
	resetTimeout time.Duration
	failures     int
	lastFailTime time.Time
	state        CircuitState
}

// CircuitState 熔断器状态
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewCircuit 创建熔断器
func NewCircuit(maxFailures int, resetTimeout time.Duration) *Circuit {
	return &Circuit{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
	}
}

// Execute 执行操作（带熔断器）
func (c *Circuit) Execute(ctx context.Context, fn RetryFunc) error {
	if c.state == CircuitOpen {
		if time.Since(c.lastFailTime) > c.resetTimeout {
			c.state = CircuitHalfOpen
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	err := fn()

	if err != nil {
		c.failures++
		c.lastFailTime = time.Now()

		if c.failures >= c.maxFailures {
			c.state = CircuitOpen
		}
		return err
	}

	// 成功执行，重置计数器
	c.failures = 0
	c.state = CircuitClosed
	return nil
}
