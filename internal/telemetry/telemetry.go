package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Telemetry provides OpenTelemetry instrumentation
type Telemetry struct {
	tracer         trace.Tracer
	meter          metric.Meter
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider

	// Metrics
	discoveryDuration   metric.Float64Histogram
	discoveryErrors     metric.Int64Counter
	resourcesDiscovered metric.Int64Counter
	driftDetected       metric.Int64Counter
	remediationAttempts metric.Int64Counter
	remediationSuccess  metric.Int64Counter
	apiRequests         metric.Int64Counter
	apiLatency          metric.Float64Histogram
	activeConnections   metric.Int64UpDownCounter
	cacheHits           metric.Int64Counter
	cacheMisses         metric.Int64Counter
}

var (
	globalTelemetry *Telemetry
	serviceName     = "driftmgr"
)

// Config represents telemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	JaegerEndpoint string
	PrometheusPort int
	SampleRate     float64
	EnableTracing  bool
	EnableMetrics  bool
}

// Initialize sets up OpenTelemetry
func Initialize(ctx context.Context, config Config) (*Telemetry, error) {
	if config.ServiceName != "" {
		serviceName = config.ServiceName
	}

	// Create resource
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	t := &Telemetry{}

	// Initialize tracing
	if config.EnableTracing {
		if err := t.initTracing(ctx, config, res); err != nil {
			return nil, fmt.Errorf("failed to initialize tracing: %w", err)
		}
	}

	// Initialize metrics
	if config.EnableMetrics {
		if err := t.initMetrics(ctx, config, res); err != nil {
			return nil, fmt.Errorf("failed to initialize metrics: %w", err)
		}
	}

	// Create instruments
	if err := t.createInstruments(); err != nil {
		return nil, fmt.Errorf("failed to create instruments: %w", err)
	}

	globalTelemetry = t
	return t, nil
}

// initTracing initializes tracing
func (t *Telemetry) initTracing(ctx context.Context, config Config, res *resource.Resource) error {
	// Create Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.JaegerEndpoint)))
	if err != nil {
		return err
	}

	// Create tracer provider
	t.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(config.SampleRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(t.tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t.tracer = t.tracerProvider.Tracer(serviceName)

	return nil
}

// initMetrics initializes metrics
func (t *Telemetry) initMetrics(ctx context.Context, config Config, res *resource.Resource) error {
	// Create Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return err
	}

	// Create meter provider
	t.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
		sdkmetric.WithResource(res),
	)

	// Set global meter provider
	otel.SetMeterProvider(t.meterProvider)

	t.meter = t.meterProvider.Meter(serviceName)

	return nil
}

// createInstruments creates metric instruments
func (t *Telemetry) createInstruments() error {
	var err error

	// Discovery metrics
	t.discoveryDuration, err = t.meter.Float64Histogram(
		"driftmgr.discovery.duration",
		metric.WithDescription("Discovery operation duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	t.discoveryErrors, err = t.meter.Int64Counter(
		"driftmgr.discovery.errors",
		metric.WithDescription("Number of discovery errors"),
	)
	if err != nil {
		return err
	}

	t.resourcesDiscovered, err = t.meter.Int64Counter(
		"driftmgr.resources.discovered",
		metric.WithDescription("Number of resources discovered"),
	)
	if err != nil {
		return err
	}

	// Drift metrics
	t.driftDetected, err = t.meter.Int64Counter(
		"driftmgr.drift.detected",
		metric.WithDescription("Number of drift items detected"),
	)
	if err != nil {
		return err
	}

	// Remediation metrics
	t.remediationAttempts, err = t.meter.Int64Counter(
		"driftmgr.remediation.attempts",
		metric.WithDescription("Number of remediation attempts"),
	)
	if err != nil {
		return err
	}

	t.remediationSuccess, err = t.meter.Int64Counter(
		"driftmgr.remediation.success",
		metric.WithDescription("Number of successful remediations"),
	)
	if err != nil {
		return err
	}

	// API metrics
	t.apiRequests, err = t.meter.Int64Counter(
		"driftmgr.api.requests",
		metric.WithDescription("Number of API requests"),
	)
	if err != nil {
		return err
	}

	t.apiLatency, err = t.meter.Float64Histogram(
		"driftmgr.api.latency",
		metric.WithDescription("API request latency in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	t.activeConnections, err = t.meter.Int64UpDownCounter(
		"driftmgr.connections.active",
		metric.WithDescription("Number of active connections"),
	)
	if err != nil {
		return err
	}

	// Cache metrics
	t.cacheHits, err = t.meter.Int64Counter(
		"driftmgr.cache.hits",
		metric.WithDescription("Number of cache hits"),
	)
	if err != nil {
		return err
	}

	t.cacheMisses, err = t.meter.Int64Counter(
		"driftmgr.cache.misses",
		metric.WithDescription("Number of cache misses"),
	)
	if err != nil {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down telemetry
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t.tracerProvider != nil {
		if err := t.tracerProvider.Shutdown(ctx); err != nil {
			return err
		}
	}

	if t.meterProvider != nil {
		if err := t.meterProvider.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Get returns the global telemetry instance
func Get() *Telemetry {
	return globalTelemetry
}

// StartSpan starts a new span
func (t *Telemetry) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// RecordDiscovery records discovery metrics
func (t *Telemetry) RecordDiscovery(ctx context.Context, provider string, resourceCount int, duration time.Duration, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}

	t.discoveryDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	t.resourcesDiscovered.Add(ctx, int64(resourceCount), metric.WithAttributes(attrs...))

	if err != nil {
		t.discoveryErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordDrift records drift detection metrics
func (t *Telemetry) RecordDrift(ctx context.Context, provider string, driftCount int) {
	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}

	t.driftDetected.Add(ctx, int64(driftCount), metric.WithAttributes(attrs...))
}

// RecordRemediation records remediation metrics
func (t *Telemetry) RecordRemediation(ctx context.Context, resourceType string, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("resource_type", resourceType),
		attribute.Bool("success", success),
	}

	t.remediationAttempts.Add(ctx, 1, metric.WithAttributes(attrs...))

	if success {
		t.remediationSuccess.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordAPIRequest records API request metrics
func (t *Telemetry) RecordAPIRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.String("path", path),
		attribute.Int("status_code", statusCode),
	}

	t.apiRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
	t.apiLatency.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordCacheHit records a cache hit
func (t *Telemetry) RecordCacheHit(ctx context.Context, cacheType string) {
	attrs := []attribute.KeyValue{
		attribute.String("cache_type", cacheType),
	}

	t.cacheHits.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCacheMiss records a cache miss
func (t *Telemetry) RecordCacheMiss(ctx context.Context, cacheType string) {
	attrs := []attribute.KeyValue{
		attribute.String("cache_type", cacheType),
	}

	t.cacheMisses.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// IncrementActiveConnections increments active connections
func (t *Telemetry) IncrementActiveConnections(ctx context.Context, connectionType string) {
	attrs := []attribute.KeyValue{
		attribute.String("connection_type", connectionType),
	}

	t.activeConnections.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// DecrementActiveConnections decrements active connections
func (t *Telemetry) DecrementActiveConnections(ctx context.Context, connectionType string) {
	attrs := []attribute.KeyValue{
		attribute.String("connection_type", connectionType),
	}

	t.activeConnections.Add(ctx, -1, metric.WithAttributes(attrs...))
}

// TracedHTTPHandler wraps an HTTP handler with tracing
func TracedHTTPHandler(pattern string, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := globalTelemetry.StartSpan(
			r.Context(),
			fmt.Sprintf("HTTP %s %s", r.Method, pattern),
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.path", r.URL.Path),
				attribute.String("http.user_agent", r.UserAgent()),
			),
		)
		defer span.End()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}

		// Time the request
		start := time.Now()

		// Call the handler
		handler(wrapped, r.WithContext(ctx))

		// Record metrics
		duration := time.Since(start)
		globalTelemetry.RecordAPIRequest(ctx, r.Method, pattern, wrapped.statusCode, duration)

		// Set span attributes
		span.SetAttributes(
			attribute.Int("http.status_code", wrapped.statusCode),
			attribute.Int64("http.response_size", wrapped.written),
		)

		if wrapped.statusCode >= 400 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", wrapped.statusCode))
		}
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(data)
	rw.written += int64(n)
	return n, err
}
