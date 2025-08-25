package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/catherinevee/driftmgr/internal/observability/logging"
)

// Telemetry provides observability capabilities
type Telemetry struct {
	tracer   trace.Tracer
	meter    metric.Meter
	provider *sdktrace.TracerProvider

	// Common metrics
	requestCounter  metric.Int64Counter
	requestDuration metric.Float64Histogram
	errorCounter    metric.Int64Counter

	// Resource metrics
	resourcesDiscovered metric.Int64Counter
	driftDetected       metric.Int64Counter
	remediationSuccess  metric.Int64Counter
	remediationFailure  metric.Int64Counter

	// Performance metrics
	apiLatency       metric.Float64Histogram
	discoveryLatency metric.Float64Histogram
	dbQueryDuration  metric.Float64Histogram
}

// Config represents telemetry configuration
type Config struct {
	ServiceName    string `json:"service_name" yaml:"service_name"`
	ServiceVersion string `json:"service_version" yaml:"service_version"`
	Environment    string `json:"environment" yaml:"environment"`

	// Tracing
	TracingEnabled  bool    `json:"tracing_enabled" yaml:"tracing_enabled"`
	JaegerEndpoint  string  `json:"jaeger_endpoint" yaml:"jaeger_endpoint"`
	TraceSampleRate float64 `json:"trace_sample_rate" yaml:"trace_sample_rate"`

	// Metrics
	MetricsEnabled bool `json:"metrics_enabled" yaml:"metrics_enabled"`
	MetricsPort    int  `json:"metrics_port" yaml:"metrics_port"`

	// Additional attributes
	Attributes map[string]string `json:"attributes" yaml:"attributes"`
}

// DefaultConfig returns default telemetry configuration
func DefaultConfig() *Config {
	return &Config{
		ServiceName:     "driftmgr",
		ServiceVersion:  "1.0.0",
		Environment:     "development",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceSampleRate: 1.0,
		MetricsEnabled:  true,
		MetricsPort:     9090,
	}
}

var globalTelemetry *Telemetry

// Init initializes telemetry with the given configuration
func Init(ctx context.Context, cfg *Config) (*Telemetry, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Create resource
	attrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(cfg.ServiceVersion),
		attribute.String("environment", cfg.Environment),
	}
	attrs = append(attrs, attributesFromMap(cfg.Attributes)...)

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			attrs...,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	t := &Telemetry{}

	// Initialize tracing
	if cfg.TracingEnabled {
		if err := t.initTracing(ctx, cfg, res); err != nil {
			return nil, fmt.Errorf("failed to initialize tracing: %w", err)
		}
	}

	// Initialize metrics
	if cfg.MetricsEnabled {
		if err := t.initMetrics(ctx, cfg, res); err != nil {
			return nil, fmt.Errorf("failed to initialize metrics: %w", err)
		}
	}

	// Set global telemetry
	globalTelemetry = t

	logging.Info("Telemetry initialized", map[string]interface{}{
		"tracing_enabled": cfg.TracingEnabled,
		"metrics_enabled": cfg.MetricsEnabled,
		"service_name":    cfg.ServiceName,
		"environment":     cfg.Environment,
	})

	return t, nil
}

// initTracing initializes distributed tracing
func (t *Telemetry) initTracing(ctx context.Context, cfg *Config, res *resource.Resource) error {
	// Create OTLP exporter
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(cfg.JaegerEndpoint),
		otlptracehttp.WithInsecure(),
	)

	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create tracer provider
	t.provider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.TraceSampleRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(t.provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create tracer
	t.tracer = t.provider.Tracer(cfg.ServiceName)

	return nil
}

// initMetrics initializes metrics collection
func (t *Telemetry) initMetrics(ctx context.Context, cfg *Config, res *resource.Resource) error {
	// Create Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create meter provider
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(exporter),
	)

	// Set global meter provider
	otel.SetMeterProvider(provider)

	// Create meter
	t.meter = provider.Meter(cfg.ServiceName)

	// Initialize common metrics
	if err := t.initCommonMetrics(); err != nil {
		return fmt.Errorf("failed to initialize common metrics: %w", err)
	}

	return nil
}

// initCommonMetrics initializes commonly used metrics
func (t *Telemetry) initCommonMetrics() error {
	var err error

	// Request metrics
	t.requestCounter, err = t.meter.Int64Counter(
		"driftmgr.requests.total",
		metric.WithDescription("Total number of requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	t.requestDuration, err = t.meter.Float64Histogram(
		"driftmgr.requests.duration",
		metric.WithDescription("Request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	t.errorCounter, err = t.meter.Int64Counter(
		"driftmgr.errors.total",
		metric.WithDescription("Total number of errors"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	// Resource metrics
	t.resourcesDiscovered, err = t.meter.Int64Counter(
		"driftmgr.resources.discovered",
		metric.WithDescription("Number of resources discovered"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	t.driftDetected, err = t.meter.Int64Counter(
		"driftmgr.drift.detected",
		metric.WithDescription("Number of drift instances detected"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	t.remediationSuccess, err = t.meter.Int64Counter(
		"driftmgr.remediation.success",
		metric.WithDescription("Number of successful remediations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	t.remediationFailure, err = t.meter.Int64Counter(
		"driftmgr.remediation.failure",
		metric.WithDescription("Number of failed remediations"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return err
	}

	// Performance metrics
	t.apiLatency, err = t.meter.Float64Histogram(
		"driftmgr.api.latency",
		metric.WithDescription("API endpoint latency in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	t.discoveryLatency, err = t.meter.Float64Histogram(
		"driftmgr.discovery.latency",
		metric.WithDescription("Resource discovery latency in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	t.dbQueryDuration, err = t.meter.Float64Histogram(
		"driftmgr.database.query.duration",
		metric.WithDescription("Database query duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	return nil
}

// StartSpan starts a new span
func (t *Telemetry) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if t.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.tracer.Start(ctx, name, opts...)
}

// RecordRequest records a request metric
func (t *Telemetry) RecordRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration) {
	if t.requestCounter == nil || t.requestDuration == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.String("path", path),
		attribute.Int("status_code", statusCode),
	}

	t.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	t.requestDuration.Record(ctx, duration.Seconds()*1000, metric.WithAttributes(attrs...))

	if statusCode >= 400 {
		t.errorCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordResourcesDiscovered records discovered resources
func (t *Telemetry) RecordResourcesDiscovered(ctx context.Context, provider string, count int64) {
	if t.resourcesDiscovered == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
	}

	t.resourcesDiscovered.Add(ctx, count, metric.WithAttributes(attrs...))
}

// RecordDriftDetected records detected drift
func (t *Telemetry) RecordDriftDetected(ctx context.Context, provider, resourceType string, count int64) {
	if t.driftDetected == nil {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("provider", provider),
		attribute.String("resource_type", resourceType),
	}

	t.driftDetected.Add(ctx, count, metric.WithAttributes(attrs...))
}

// RecordRemediation records remediation result
func (t *Telemetry) RecordRemediation(ctx context.Context, provider string, success bool) {
	if success && t.remediationSuccess != nil {
		t.remediationSuccess.Add(ctx, 1, metric.WithAttributes(
			attribute.String("provider", provider),
		))
	} else if !success && t.remediationFailure != nil {
		t.remediationFailure.Add(ctx, 1, metric.WithAttributes(
			attribute.String("provider", provider),
		))
	}
}

// RecordAPILatency records API endpoint latency
func (t *Telemetry) RecordAPILatency(ctx context.Context, endpoint string, duration time.Duration) {
	if t.apiLatency == nil {
		return
	}

	t.apiLatency.Record(ctx, duration.Seconds()*1000, metric.WithAttributes(
		attribute.String("endpoint", endpoint),
	))
}

// RecordDiscoveryLatency records discovery latency
func (t *Telemetry) RecordDiscoveryLatency(ctx context.Context, provider string, duration time.Duration) {
	if t.discoveryLatency == nil {
		return
	}

	t.discoveryLatency.Record(ctx, duration.Seconds()*1000, metric.WithAttributes(
		attribute.String("provider", provider),
	))
}

// RecordDBQuery records database query duration
func (t *Telemetry) RecordDBQuery(ctx context.Context, operation string, duration time.Duration) {
	if t.dbQueryDuration == nil {
		return
	}

	t.dbQueryDuration.Record(ctx, duration.Seconds()*1000, metric.WithAttributes(
		attribute.String("operation", operation),
	))
}

// Shutdown gracefully shuts down telemetry
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// Get returns the global telemetry instance
func Get() *Telemetry {
	return globalTelemetry
}

// attributesFromMap converts a map to OpenTelemetry attributes
func attributesFromMap(m map[string]string) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(m))
	for k, v := range m {
		attrs = append(attrs, attribute.String(k, v))
	}
	return attrs
}
