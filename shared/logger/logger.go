package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level  string   `json:"level" yaml:"level" default:"info"`
	Format string   `json:"format" yaml:"format" default:"json"`
	Output []string `json:"output" yaml:"output" default:"[stdout]"`
}

// Logger 统一的日志接口
type Logger interface {
	Info(ctx context.Context, message string, fields map[string]any)
	Error(ctx context.Context, message string, err error, fields map[string]any)
	Debug(ctx context.Context, message string, fields map[string]any)
	Warn(ctx context.Context, message string, fields map[string]any)
}

// LogLevel 日志级别
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// standardLogger 标准日志实现
type standardLogger struct {
	level  LogLevel
	format string
	writer io.Writer
	logger *log.Logger
}

// NewLogger 创建日志器
func NewLogger(config LoggingConfig) Logger {
	level := parseLogLevel(config.Level)
	writer := createWriter(config.Output)

	return &standardLogger{
		level:  level,
		format: config.Format,
		writer: writer,
		logger: log.New(writer, "", 0),
	}
}

// Info 记录信息日志
func (l *standardLogger) Info(ctx context.Context, message string, fields map[string]any) {
	if l.level <= LogLevelInfo {
		l.writeLog(ctx, LogLevelInfo, message, nil, fields)
	}
}

// Error 记录错误日志
func (l *standardLogger) Error(ctx context.Context, message string, err error, fields map[string]any) {
	if l.level <= LogLevelError {
		l.writeLog(ctx, LogLevelError, message, err, fields)
	}
}

// Debug 记录调试日志
func (l *standardLogger) Debug(ctx context.Context, message string, fields map[string]any) {
	if l.level <= LogLevelDebug {
		l.writeLog(ctx, LogLevelDebug, message, nil, fields)
	}
}

// Warn 记录警告日志
func (l *standardLogger) Warn(ctx context.Context, message string, fields map[string]any) {
	if l.level <= LogLevelWarn {
		l.writeLog(ctx, LogLevelWarn, message, nil, fields)
	}
}

// writeLog 写入日志
func (l *standardLogger) writeLog(ctx context.Context, level LogLevel, message string, err error, fields map[string]any) {
	entry := l.createLogEntry(ctx, level, message, err, fields)

	switch strings.ToLower(l.format) {
	case "json":
		l.writeJSONLog(entry)
	default:
		l.writeTextLog(entry)
	}
}

// LogEntry 日志条目
type LogEntry struct {
	Timestamp string         `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Error     string         `json:"error,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// createLogEntry 创建日志条目
func (l *standardLogger) createLogEntry(ctx context.Context, level LogLevel, message string, err error, fields map[string]any) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level.String(),
		Message:   message,
		Fields:    fields,
	}

	// 添加错误信息
	if err != nil {
		entry.Error = err.Error()
	}

	// 从上下文中提取请求ID
	if requestID, ok := ctx.Value("request_id").(string); ok {
		entry.RequestID = requestID
	}

	return entry
}

// writeJSONLog 写入JSON格式日志
func (l *standardLogger) writeJSONLog(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		// 降级到文本日志
		l.writeTextLog(entry)
		return
	}

	l.logger.Println(string(data))
}

// writeTextLog 写入文本格式日志
func (l *standardLogger) writeTextLog(entry LogEntry) {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("[%s] %s %s",
		entry.Level,
		entry.Timestamp,
		entry.Message,
	))

	if entry.RequestID != "" {
		builder.WriteString(fmt.Sprintf(" request_id=%s", entry.RequestID))
	}

	if entry.Error != "" {
		builder.WriteString(fmt.Sprintf(" error=%s", entry.Error))
	}

	if len(entry.Fields) > 0 {
		builder.WriteString(" fields=")
		fieldsData, _ := json.Marshal(entry.Fields)
		builder.Write(fieldsData)
	}

	l.logger.Println(builder.String())
}

// parseLogLevel 解析日志级别
func parseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarn
	case "error":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

// createWriter 创建输出写入器
func createWriter(outputs []string) io.Writer {
	var writers []io.Writer

	for _, output := range outputs {
		switch strings.ToLower(output) {
		case "stdout":
			writers = append(writers, os.Stdout)
		case "stderr":
			writers = append(writers, os.Stderr)
		default:
			// 文件输出
			if file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
				writers = append(writers, file)
			}
		}
	}

	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	if len(writers) == 1 {
		return writers[0]
	}

	return io.MultiWriter(writers...)
}

// DefaultLogger 默认日志器实例
var DefaultLogger = NewLogger(LoggingConfig{
	Level:  "info",
	Format: "json",
	Output: []string{"stdout"},
})
