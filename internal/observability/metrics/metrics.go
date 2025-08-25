package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
	Summary   MetricType = "summary"
)

// Metric represents a single metric
type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Unit      string            `json:"unit,omitempty"`
	Help      string            `json:"help,omitempty"`
}

// MetricsCollector collects and stores metrics
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]*Metric
	history []Metric
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*Metric),
		history: make([]Metric, 0),
	}
}

// DriftMgrMetrics contains all metrics for the application
type DriftMgrMetrics struct {
	collector *MetricsCollector
	mu        sync.RWMutex
}

// NewDriftMgrMetrics creates a new metrics instance
func NewDriftMgrMetrics() *DriftMgrMetrics {
	return &DriftMgrMetrics{
		collector: NewMetricsCollector(),
	}
}

// RecordDiscoveryDuration records the duration of a discovery operation
func (m *DriftMgrMetrics) RecordDiscoveryDuration(provider string, duration time.Duration) {
	m.collector.RecordHistogram("discovery_duration_seconds", duration.Seconds(), map[string]string{
		"provider": provider,
	})
}

// RecordDiscoveredResources records the number of discovered resources
func (m *DriftMgrMetrics) RecordDiscoveredResources(provider, region string, count int) {
	m.collector.RecordGauge("discovered_resources_total", float64(count), map[string]string{
		"provider": provider,
		"region":   region,
	})
}

// RecordDriftDetected records drift detection
func (m *DriftMgrMetrics) RecordDriftDetected(resourceType, severity string) {
	m.collector.IncrementCounter("drift_detected_total", map[string]string{
		"resource_type": resourceType,
		"severity":      severity,
	})
}

// RecordRemediationAttempt records a remediation attempt
func (m *DriftMgrMetrics) RecordRemediationAttempt(resourceType string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.collector.IncrementCounter("remediation_attempts_total", map[string]string{
		"resource_type": resourceType,
		"status":        status,
	})
}

// RecordRemediationDuration records the duration of a remediation
func (m *DriftMgrMetrics) RecordRemediationDuration(resourceType string, duration time.Duration) {
	m.collector.RecordHistogram("remediation_duration_seconds", duration.Seconds(), map[string]string{
		"resource_type": resourceType,
	})
}

// RecordDeletionAttempt records a deletion attempt
func (m *DriftMgrMetrics) RecordDeletionAttempt(provider, resourceType string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.collector.IncrementCounter("deletion_attempts_total", map[string]string{
		"provider":      provider,
		"resource_type": resourceType,
		"status":        status,
	})
}

// RecordAPICall records an API call
func (m *DriftMgrMetrics) RecordAPICall(provider, operation string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.collector.IncrementCounter("api_calls_total", map[string]string{
		"provider":  provider,
		"operation": operation,
		"status":    status,
	})
	m.collector.RecordHistogram("api_call_duration_seconds", duration.Seconds(), map[string]string{
		"provider":  provider,
		"operation": operation,
	})
}

// RecordCacheHit records cache hit/miss
func (m *DriftMgrMetrics) RecordCacheHit(cacheType string, hit bool) {
	result := "hit"
	if !hit {
		result = "miss"
	}
	m.collector.IncrementCounter("cache_requests_total", map[string]string{
		"type":   cacheType,
		"result": result,
	})
}

// RecordError records an error occurrence
func (m *DriftMgrMetrics) RecordError(errorCode, component string) {
	m.collector.IncrementCounter("errors_total", map[string]string{
		"code":      errorCode,
		"component": component,
	})
}

// RecordActiveConnections records the number of active connections
func (m *DriftMgrMetrics) RecordActiveConnections(count int) {
	m.collector.RecordGauge("active_connections", float64(count), nil)
}

// RecordMemoryUsage records memory usage
func (m *DriftMgrMetrics) RecordMemoryUsage(bytes uint64) {
	m.collector.RecordGauge("memory_usage_bytes", float64(bytes), nil)
}

// RecordGoroutines records the number of goroutines
func (m *DriftMgrMetrics) RecordGoroutines(count int) {
	m.collector.RecordGauge("goroutines", float64(count), nil)
}

// MetricsCollector methods

// IncrementCounter increments a counter metric
func (c *MetricsCollector) IncrementCounter(name string, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.generateKey(name, labels)
	if metric, exists := c.metrics[key]; exists {
		metric.Value++
		metric.Timestamp = time.Now()
	} else {
		c.metrics[key] = &Metric{
			Name:      name,
			Type:      Counter,
			Value:     1,
			Labels:    labels,
			Timestamp: time.Now(),
		}
	}
}

// RecordGauge records a gauge metric
func (c *MetricsCollector) RecordGauge(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.generateKey(name, labels)
	c.metrics[key] = &Metric{
		Name:      name,
		Type:      Gauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
}

// RecordHistogram records a histogram metric
func (c *MetricsCollector) RecordHistogram(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	metric := Metric{
		Name:      name,
		Type:      Histogram,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}
	c.history = append(c.history, metric)

	// Keep history size limited
	if len(c.history) > 10000 {
		c.history = c.history[1000:]
	}
}

// GetMetrics returns all current metrics
func (c *MetricsCollector) GetMetrics() []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := make([]Metric, 0, len(c.metrics))
	for _, metric := range c.metrics {
		metrics = append(metrics, *metric)
	}
	return metrics
}

// GetHistory returns metric history
func (c *MetricsCollector) GetHistory() []Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	history := make([]Metric, len(c.history))
	copy(history, c.history)
	return history
}

// Reset resets all metrics
func (c *MetricsCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics = make(map[string]*Metric)
	c.history = make([]Metric, 0)
}

// generateKey generates a unique key for a metric
func (c *MetricsCollector) generateKey(name string, labels map[string]string) string {
	key := name
	if labels != nil {
		for k, v := range labels {
			key += fmt.Sprintf("_%s_%s", k, v)
		}
	}
	return key
}

// Global metrics instance
var (
	globalMetrics *DriftMgrMetrics
	metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics
func InitMetrics() {
	metricsOnce.Do(func() {
		globalMetrics = NewDriftMgrMetrics()
	})
}

// GetMetrics returns the global metrics instance
func GetMetrics() *DriftMgrMetrics {
	if globalMetrics == nil {
		InitMetrics()
	}
	return globalMetrics
}

// WithMetrics adds metrics to context
func WithMetrics(ctx context.Context, metrics *DriftMgrMetrics) context.Context {
	return context.WithValue(ctx, "metrics", metrics)
}

// FromContext extracts metrics from context
func FromContext(ctx context.Context) *DriftMgrMetrics {
	if metrics, ok := ctx.Value("metrics").(*DriftMgrMetrics); ok {
		return metrics
	}
	return GetMetrics()
}
