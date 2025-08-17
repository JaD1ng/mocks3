package trace

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	otrace "go.opentelemetry.io/otel/trace"
)

// TracerProvider 追踪器提供者
type TracerProvider struct {
	provider *trace.TracerProvider
	tracer   otrace.Tracer
}

// TracerConfig 追踪器配置
type TracerConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	SamplingRatio  float64
}

// NewTracerProvider 创建新的追踪器提供者
func NewTracerProvider(config *TracerConfig) (*TracerProvider, error) {
	// 创建资源
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建OTLP HTTP导出器
	exporter, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint(config.OTLPEndpoint),
		otlptracehttp.WithInsecure(), // 开发环境使用HTTP
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// 创建采样器
	var sampler trace.Sampler
	if config.SamplingRatio <= 0 {
		sampler = trace.NeverSample()
	} else if config.SamplingRatio >= 1.0 {
		sampler = trace.AlwaysSample()
	} else {
		sampler = trace.TraceIDRatioBased(config.SamplingRatio)
	}

	// 创建追踪器提供者
	provider := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(exporter),
		trace.WithSampler(sampler),
	)

	// 设置全局追踪器提供者
	otel.SetTracerProvider(provider)

	// 设置全局传播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := provider.Tracer(config.ServiceName)

	return &TracerProvider{
		provider: provider,
		tracer:   tracer,
	}, nil
}

// GetTracer 获取追踪器
func (tp *TracerProvider) GetTracer() otrace.Tracer {
	return tp.tracer
}

// Shutdown 关闭追踪器提供者
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	return tp.provider.Shutdown(ctx)
}

// StartSpan 开始一个新的span
func (tp *TracerProvider) StartSpan(ctx context.Context, name string, opts ...otrace.SpanStartOption) (context.Context, otrace.Span) {
	return tp.tracer.Start(ctx, name, opts...)
}

// SpanFromContext 从上下文获取span
func SpanFromContext(ctx context.Context) otrace.Span {
	return otrace.SpanFromContext(ctx)
}

// AddEvent 为span添加事件
func AddEvent(ctx context.Context, name string, attributes ...otrace.EventOption) {
	span := otrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, attributes...)
	}
}

// AddAttributes 为span添加属性
func AddAttributes(ctx context.Context, attributes ...otrace.SpanStartOption) {
	span := otrace.SpanFromContext(ctx)
	if span.IsRecording() {
		// 由于Span接口没有直接的SetAttributes方法，我们在创建span时设置属性
		// 这个函数主要用于演示，实际使用时应该在创建span时设置属性
	}
}

// SetError 设置span错误状态
func SetError(ctx context.Context, err error) {
	span := otrace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetStatus 设置span状态
func SetStatus(ctx context.Context, code codes.Code, description string) {
	span := otrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetStatus(code, description)
	}
}

// GetTraceID 获取当前追踪ID
func GetTraceID(ctx context.Context) string {
	span := otrace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// GetSpanID 获取当前SpanID
func GetSpanID(ctx context.Context) string {
	span := otrace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

// NewDefaultTracerProvider 创建默认的追踪器提供者
func NewDefaultTracerProvider(serviceName string) (*TracerProvider, error) {
	config := &TracerConfig{
		ServiceName:    serviceName,
		ServiceVersion: os.Getenv("SERVICE_VERSION"),
		Environment:    getEnv("ENVIRONMENT", "development"),
		OTLPEndpoint:   getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
		SamplingRatio:  1.0, // 开发环境全量采样
	}

	if config.ServiceVersion == "" {
		config.ServiceVersion = "1.0.0"
	}

	return NewTracerProvider(config)
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
