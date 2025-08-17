package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// Logger 结构化日志器
type Logger struct {
	logger      *slog.Logger
	serviceName string
	level       slog.Level
}

// LogLevel 日志级别
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// NewLogger 创建新的日志器
func NewLogger(serviceName string, level LogLevel) *Logger {
	var slogLevel slog.Level
	switch level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 自定义时间格式
			if a.Key == slog.TimeKey {
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339Nano))
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &Logger{
		logger:      logger,
		serviceName: serviceName,
		level:       slogLevel,
	}
}

// WithContext 为日志添加追踪上下文
func (l *Logger) WithContext(ctx context.Context) *Logger {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return l
	}

	spanCtx := span.SpanContext()
	logger := l.logger.With(
		"trace_id", spanCtx.TraceID().String(),
		"span_id", spanCtx.SpanID().String(),
	)

	return &Logger{
		logger:      logger,
		serviceName: l.serviceName,
		level:       l.level,
	}
}

// With 添加字段
func (l *Logger) With(args ...interface{}) *Logger {
	return &Logger{
		logger:      l.logger.With(args...),
		serviceName: l.serviceName,
		level:       l.level,
	}
}

// WithFields 添加多个字段
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return l.With(args...)
}

// Debug 调试日志
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.logger.Debug(msg, l.prepareArgs(args...)...)
}

// Info 信息日志
func (l *Logger) Info(msg string, args ...interface{}) {
	l.logger.Info(msg, l.prepareArgs(args...)...)
}

// Warn 警告日志
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.logger.Warn(msg, l.prepareArgs(args...)...)
}

// Error 错误日志
func (l *Logger) Error(msg string, args ...interface{}) {
	l.logger.Error(msg, l.prepareArgs(args...)...)
}

// DebugContext 带上下文的调试日志
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Debug(msg, args...)
}

// InfoContext 带上下文的信息日志
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Info(msg, args...)
}

// WarnContext 带上下文的警告日志
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Warn(msg, args...)
}

// ErrorContext 带上下文的错误日志
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...interface{}) {
	l.WithContext(ctx).Error(msg, args...)
}

// LogError 记录错误
func (l *Logger) LogError(ctx context.Context, err error, msg string, args ...interface{}) {
	if err == nil {
		return
	}

	fields := map[string]interface{}{
		"error": err.Error(),
	}

	l.WithContext(ctx).WithFields(fields).Error(msg, args...)
}

// LogPanic 记录panic
func (l *Logger) LogPanic(ctx context.Context, r interface{}, msg string, args ...interface{}) {
	fields := map[string]interface{}{
		"panic": fmt.Sprintf("%v", r),
	}

	l.WithContext(ctx).WithFields(fields).Error(msg, args...)
}

// prepareArgs 准备日志参数，添加服务名
func (l *Logger) prepareArgs(args ...interface{}) []interface{} {
	result := make([]interface{}, 0, len(args)+2)
	result = append(result, "service", l.serviceName)
	result = append(result, args...)
	return result
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	var slogLevel slog.Level
	switch level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	l.level = slogLevel
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() LogLevel {
	switch l.level {
	case slog.LevelDebug:
		return LevelDebug
	case slog.LevelInfo:
		return LevelInfo
	case slog.LevelWarn:
		return LevelWarn
	case slog.LevelError:
		return LevelError
	default:
		return LevelInfo
	}
}

// IsDebugEnabled 检查是否启用调试级别
func (l *Logger) IsDebugEnabled() bool {
	return l.level <= slog.LevelDebug
}

// IsInfoEnabled 检查是否启用信息级别
func (l *Logger) IsInfoEnabled() bool {
	return l.level <= slog.LevelInfo
}

// IsWarnEnabled 检查是否启用警告级别
func (l *Logger) IsWarnEnabled() bool {
	return l.level <= slog.LevelWarn
}

// IsErrorEnabled 检查是否启用错误级别
func (l *Logger) IsErrorEnabled() bool {
	return l.level <= slog.LevelError
}
