package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/catherinevee/driftmgr/internal/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingManager manages distributed tracing
type TracingManager struct {
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
	logger   *logging.Logger
}

// TracingConfig holds tracing configuration
type TracingConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	ExporterType   string // "jaeger", "otlp", "console"
	Endpoint       string
	SampleRate     float64
	Enabled        bool
}

// InitTracing initializes the global tracing provider
func InitTracing(config *TracingConfig) (*TracingManager, error) {
	if !config.Enabled {
		return &TracingManager{
			tracer: trace.NewNoopTracerProvider().Tracer("noop"),
			logger: logging.GetLogger(),
		}, nil
	}

	// Check environment variables for overrides
	if endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); endpoint != "" {
		config.Endpoint = endpoint
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
			attribute.String("service.namespace", "driftmgr"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on type
	var exporter sdktrace.SpanExporter
	switch config.ExporterType {
	case "jaeger":
		// Jaeger is deprecated, use OTLP instead
		exporter, err = createOTLPExporter(config.Endpoint)
	case "otlp":
		exporter, err = createOTLPExporter(config.Endpoint)
	default:
		exporter, err = createConsoleExporter()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create tracer provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
	)

	// Set global provider
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	manager := &TracingManager{
		tracer:   provider.Tracer(config.ServiceName),
		provider: provider,
		logger:   logging.GetLogger(),
	}

	manager.logger.Info("Tracing initialized", map[string]interface{}{
		"service":     config.ServiceName,
		"environment": config.Environment,
		"exporter":    config.ExporterType,
		"sample_rate": config.SampleRate,
	})

	return manager, nil
}

// createOTLPExporter creates an OTLP exporter
func createOTLPExporter(endpoint string) (sdktrace.SpanExporter, error) {
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)

	return otlptrace.New(context.Background(), client)
}

// createConsoleExporter creates a console exporter for development
func createConsoleExporter() (sdktrace.SpanExporter, error) {
	return stdouttrace.New(stdouttrace.WithPrettyPrint())
}

// StartSpan starts a new span
func (tm *TracingManager) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, name, opts...)
}

// TraceOperation traces a generic operation
func (tm *TracingManager) TraceOperation(ctx context.Context, operation string, attrs map[string]interface{}) (context.Context, trace.Span) {
	spanAttrs := []attribute.KeyValue{
		attribute.String("operation.type", operation),
		attribute.String("operation.timestamp", time.Now().Format(time.RFC3339)),
	}

	for k, v := range attrs {
		spanAttrs = append(spanAttrs, attributeFromInterface(k, v))
	}

	return tm.tracer.Start(ctx, operation,
		trace.WithAttributes(spanAttrs...),
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

// TraceDiscovery traces a discovery operation
func (tm *TracingManager) TraceDiscovery(ctx context.Context, provider string, region string) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, fmt.Sprintf("discovery.%s", provider),
		trace.WithAttributes(
			attribute.String("provider", provider),
			attribute.String("region", region),
			attribute.String("operation", "discovery"),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
}

// TraceDrift traces a drift detection operation
func (tm *TracingManager) TraceDrift(ctx context.Context, provider string, resourceCount int) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, "drift.detection",
		trace.WithAttributes(
			attribute.String("provider", provider),
			attribute.Int("resource.count", resourceCount),
			attribute.String("operation", "drift_detection"),
		),
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

// TraceRemediation traces a remediation operation
func (tm *TracingManager) TraceRemediation(ctx context.Context, planID string, actionCount int) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, "remediation.execute",
		trace.WithAttributes(
			attribute.String("plan.id", planID),
			attribute.Int("action.count", actionCount),
			attribute.String("operation", "remediation"),
		),
		trace.WithSpanKind(trace.SpanKindInternal),
	)
}

// TraceHTTPRequest traces an HTTP request
func (tm *TracingManager) TraceHTTPRequest(ctx context.Context, method string, url string) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, fmt.Sprintf("http.%s", method),
		trace.WithAttributes(
			semconv.HTTPMethodKey.String(method),
			semconv.HTTPURLKey.String(url),
			semconv.HTTPSchemeKey.String("https"),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
}

// TraceDatabase traces a database operation
func (tm *TracingManager) TraceDatabase(ctx context.Context, operation string, query string) (context.Context, trace.Span) {
	return tm.tracer.Start(ctx, fmt.Sprintf("db.%s", operation),
		trace.WithAttributes(
			semconv.DBSystemKey.String("postgresql"),
			semconv.DBOperationKey.String(operation),
			semconv.DBStatementKey.String(query),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)
}

// RecordError records an error in the current span
func RecordError(ctx context.Context, err error, description string) {
	span := trace.SpanFromContext(ctx)
	if span != nil && span.IsRecording() {
		span.RecordError(err)
		span.SetStatus(codes.Error, description)
		span.SetAttributes(
			attribute.String("error.type", fmt.Sprintf("%T", err)),
			attribute.String("error.message", err.Error()),
		)
	}
}

// AddEvent adds an event to the current span
func AddEvent(ctx context.Context, name string, attrs map[string]interface{}) {
	span := trace.SpanFromContext(ctx)
	if span != nil && span.IsRecording() {
		eventAttrs := []attribute.KeyValue{}
		for k, v := range attrs {
			eventAttrs = append(eventAttrs, attributeFromInterface(k, v))
		}
		span.AddEvent(name, trace.WithAttributes(eventAttrs...))
	}
}

// SetAttributes sets attributes on the current span
func SetAttributes(ctx context.Context, attrs map[string]interface{}) {
	span := trace.SpanFromContext(ctx)
	if span != nil && span.IsRecording() {
		spanAttrs := []attribute.KeyValue{}
		for k, v := range attrs {
			spanAttrs = append(spanAttrs, attributeFromInterface(k, v))
		}
		span.SetAttributes(spanAttrs...)
	}
}

// Shutdown gracefully shuts down the tracing provider
func (tm *TracingManager) Shutdown(ctx context.Context) error {
	if tm.provider != nil {
		return tm.provider.Shutdown(ctx)
	}
	return nil
}

// Helper function to convert interface to attribute
func attributeFromInterface(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	case bool:
		return attribute.Bool(key, v)
	case []string:
		return attribute.StringSlice(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}

// TracedOperation wraps an operation with tracing
type TracedOperation struct {
	Name      string
	Operation string
	Provider  string
	Tracer    *TracingManager
}

// Execute executes the operation with tracing
func (to *TracedOperation) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Start span
	ctx, span := to.Tracer.TraceOperation(ctx, to.Operation, map[string]interface{}{
		"name":     to.Name,
		"provider": to.Provider,
	})
	defer span.End()

	// Record start time
	startTime := time.Now()
	AddEvent(ctx, "operation.started", map[string]interface{}{
		"timestamp": startTime.Format(time.RFC3339),
	})

	// Execute operation
	err := fn(ctx)

	// Record duration
	duration := time.Since(startTime)
	SetAttributes(ctx, map[string]interface{}{
		"duration.ms": duration.Milliseconds(),
	})

	// Record result
	if err != nil {
		RecordError(ctx, err, fmt.Sprintf("Operation %s failed", to.Name))
		AddEvent(ctx, "operation.failed", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		span.SetStatus(codes.Ok, "Operation completed successfully")
		AddEvent(ctx, "operation.completed", map[string]interface{}{
			"duration": duration.String(),
		})
	}

	// Record metrics
	logging.Metric("operation.duration", duration.Seconds(), "seconds", map[string]string{
		"operation": to.Operation,
		"provider":  to.Provider,
		"status":    statusFromError(err),
	})

	return err
}

func statusFromError(err error) string {
	if err != nil {
		return "error"
	}
	return "success"
}

// Middleware for HTTP tracing
type HTTPTracingMiddleware struct {
	tracer *TracingManager
}

// NewHTTPTracingMiddleware creates a new HTTP tracing middleware
func NewHTTPTracingMiddleware(tracer *TracingManager) *HTTPTracingMiddleware {
	return &HTTPTracingMiddleware{
		tracer: tracer,
	}
}

// Handle wraps an HTTP handler with tracing
func (m *HTTPTracingMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace context from headers
		ctx := otel.GetTextMapPropagator().Extract(r.Context(),
			propagation.HeaderCarrier(r.Header))

		// Start span
		ctx, span := m.tracer.TraceHTTPRequest(ctx, r.Method, r.URL.String())
		defer span.End()

		// Record request details
		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.target", r.URL.Path),
			attribute.String("http.host", r.Host),
			attribute.String("http.user_agent", r.UserAgent()),
			attribute.Int64("http.request_content_length", r.ContentLength),
		)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		// Process request
		startTime := time.Now()
		next.ServeHTTP(wrapped, r.WithContext(ctx))
		duration := time.Since(startTime)

		// Record response details
		span.SetAttributes(
			attribute.Int("http.status_code", wrapped.statusCode),
			attribute.Int64("http.response.size", wrapped.written),
			attribute.Float64("http.duration.ms", duration.Seconds()*1000),
		)

		// Set span status based on HTTP status
		if wrapped.statusCode >= 400 {
			span.SetStatus(codes.Error, http.StatusText(wrapped.statusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Record metric
		logging.Metric("http.request.duration", duration.Seconds(), "seconds", map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": fmt.Sprintf("%d", wrapped.statusCode),
		})
	})
}

// Global tracing instance
var globalTracer *TracingManager

// InitGlobalTracer initializes the global tracer
func InitGlobalTracer(config *TracingConfig) error {
	tracer, err := InitTracing(config)
	if err != nil {
		return err
	}
	globalTracer = tracer
	return nil
}

// GetTracer returns the global tracer
func GetTracer() *TracingManager {
	if globalTracer == nil {
		// Return no-op tracer if not initialized
		return &TracingManager{
			tracer: trace.NewNoopTracerProvider().Tracer("noop"),
			logger: logging.GetLogger(),
		}
	}
	return globalTracer
}
