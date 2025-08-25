package observability

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// LogLevel 日志级别
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Field 日志字段
type Field struct {
	Key   string
	Value any
}

// String 创建字符串字段
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 创建整数字段
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 创建Int64字段
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 创建浮点数字段
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Error 创建错误字段
func Error(err error) Field {
	return Field{Key: "error", Value: err.Error()}
}

// Duration 创建持续时间字段
func Duration(key string, duration time.Duration) Field {
	return Field{Key: key, Value: duration.String()}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Logger 优化后的日志器 - 兼容现有接口
type Logger struct {
	logger      *slog.Logger
	serviceName string
	level       LogLevel
	baseAttrs   []slog.Attr
}

// NewLogger 创建新的日志器
func NewLogger(serviceName string, level string) *Logger {
	logLevel := parseLogLevel(level)

	var slogLevel slog.Level
	switch logLevel {
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

	// 预创建基础属性
	baseAttrs := []slog.Attr{
		slog.String("service", serviceName),
	}

	return &Logger{
		logger:      logger,
		serviceName: serviceName,
		level:       logLevel,
		baseAttrs:   baseAttrs,
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level string) {
	l.level = parseLogLevel(level)
}

// Debug 调试日志
func (l *Logger) Debug(ctx context.Context, msg string, fields ...Field) {
	if l.level > LevelDebug {
		return
	}
	l.emit(ctx, slog.LevelDebug, msg, fields...)
}

// Info 信息日志
func (l *Logger) Info(ctx context.Context, msg string, fields ...Field) {
	if l.level > LevelInfo {
		return
	}
	l.emit(ctx, slog.LevelInfo, msg, fields...)
}

// Warn 警告日志
func (l *Logger) Warn(ctx context.Context, msg string, fields ...Field) {
	if l.level > LevelWarn {
		return
	}
	l.emit(ctx, slog.LevelWarn, msg, fields...)
}

// Error 错误日志
func (l *Logger) Error(ctx context.Context, msg string, fields ...Field) {
	if l.level > LevelError {
		return
	}
	l.emit(ctx, slog.LevelError, msg, fields...)
}

// ErrorWithErr 记录错误，包含错误对象
func (l *Logger) ErrorWithErr(ctx context.Context, err error, msg string, fields ...Field) {
	if err == nil || l.level > LevelError {
		return
	}

	// 添加错误字段
	allFields := append(fields, Error(err))
	l.emit(ctx, slog.LevelError, msg, allFields...)
}

// emit 发送日志
func (l *Logger) emit(ctx context.Context, level slog.Level, msg string, fields ...Field) {
	// 复用基础属性，避免重复分配
	attrs := make([]slog.Attr, 0, len(l.baseAttrs)+len(fields)+3)
	attrs = append(attrs, l.baseAttrs...)

	// 处理额外字段
	for _, field := range fields {
		attrs = append(attrs, slog.String(field.Key, fmt.Sprintf("%v", field.Value)))
	}

	// 添加追踪信息（如果存在）
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		spanCtx := span.SpanContext()
		attrs = append(attrs,
			slog.String("trace_id", spanCtx.TraceID().String()),
			slog.String("span_id", spanCtx.SpanID().String()),
		)
	}

	// 创建并发送日志记录
	l.logger.LogAttrs(ctx, level, msg, attrs...)
}

// 兼容性方法 - 支持现有的字符串参数接口
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...interface{}) {
	fields := l.argsToFields(args...)
	l.Debug(ctx, msg, fields...)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, args ...interface{}) {
	fields := l.argsToFields(args...)
	l.Info(ctx, msg, fields...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, args ...interface{}) {
	fields := l.argsToFields(args...)
	l.Warn(ctx, msg, fields...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...interface{}) {
	fields := l.argsToFields(args...)
	l.Error(ctx, msg, fields...)
}

// argsToFields 将key-value参数对转换为Field切片
func (l *Logger) argsToFields(args ...interface{}) []Field {
	fields := make([]Field, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		key := fmt.Sprintf("%v", args[i])
		value := args[i+1]
		fields = append(fields, Field{Key: key, Value: value})
	}
	return fields
}

// parseLogLevel 解析日志级别字符串
func parseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}