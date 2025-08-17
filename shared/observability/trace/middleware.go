package trace

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	otrace "go.opentelemetry.io/otel/trace"
)

// GinMiddleware 返回Gin的追踪中间件
func GinMiddleware(serviceName string) gin.HandlerFunc {
	return otelgin.Middleware(serviceName)
}

// HTTPMiddleware HTTP追踪中间件
func HTTPMiddleware(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从HTTP头中提取追踪上下文
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// 创建span
			spanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			ctx, span := tracer.Start(ctx, spanName,
				otrace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.URLScheme(r.URL.Scheme),
					semconv.ServerAddress(r.Host),
					semconv.URLPath(r.URL.Path),
					semconv.UserAgentOriginal(r.UserAgent()),
					semconv.HTTPRequestBodySize(int(r.ContentLength)),
				),
			)
			defer span.End()

			// 创建响应写入器包装器来捕获状态码
			wrapper := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// 将上下文传递给下一个处理器
			r = r.WithContext(ctx)
			next.ServeHTTP(wrapper, r)

			// 设置响应相关的属性
			span.SetAttributes(
				semconv.HTTPResponseStatusCode(wrapper.statusCode),
				semconv.HTTPResponseBodySize(wrapper.written),
			)

			// 根据状态码设置span状态
			if wrapper.statusCode >= 400 {
				span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", wrapper.statusCode))
			}
		})
	}
}

// responseWriter 响应写入器包装器
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	rw.written += n
	return n, err
}

// AddCustomAttributes 为当前span添加自定义属性
func AddCustomAttributes(ctx *gin.Context, attrs ...attribute.KeyValue) {
	span := otrace.SpanFromContext(ctx.Request.Context())
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// AddEvent 为当前span添加事件
func AddEventToGinContext(ctx *gin.Context, name string, attrs ...attribute.KeyValue) {
	span := otrace.SpanFromContext(ctx.Request.Context())
	if span.IsRecording() {
		span.AddEvent(name, otrace.WithAttributes(attrs...))
	}
}

// SetSpanError 设置span错误状态
func SetSpanError(ctx *gin.Context, err error) {
	span := otrace.SpanFromContext(ctx.Request.Context())
	if span.IsRecording() && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// StartChildSpan 在Gin上下文中开始子span
func StartChildSpan(ctx *gin.Context, name string, opts ...otrace.SpanStartOption) otrace.Span {
	tracer := otel.Tracer("gin-custom")
	_, span := tracer.Start(ctx.Request.Context(), name, opts...)
	return span
}

// ExtractHeaders 从HTTP头中提取追踪信息
func ExtractHeaders(headers http.Header) map[string]string {
	traceHeaders := make(map[string]string)

	// 提取W3C Trace Context头
	if traceParent := headers.Get("traceparent"); traceParent != "" {
		traceHeaders["traceparent"] = traceParent
	}

	if traceState := headers.Get("tracestate"); traceState != "" {
		traceHeaders["tracestate"] = traceState
	}

	// 提取Baggage头
	if baggage := headers.Get("baggage"); baggage != "" {
		traceHeaders["baggage"] = baggage
	}

	return traceHeaders
}

// InjectHeaders 向HTTP头中注入追踪信息
func InjectHeaders(ctx *gin.Context, headers http.Header) {
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx.Request.Context(), propagation.HeaderCarrier(headers))
}
