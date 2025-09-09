package metrics

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// MetricType represents the type of metric
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a single metric
type Metric struct {
	Name        string                 `json:"name"`
	Type        MetricType             `json:"type"`
	Value       float64                `json:"value"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Description string                 `json:"description,omitempty"`
	Unit        string                 `json:"unit,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Collector collects and aggregates metrics
type Collector struct {
	metrics       map[string]*metricData
	exporters     []Exporter
	mu            sync.RWMutex
	bufferSize    int
	flushInterval time.Duration
	stopCh        chan struct{}
}

// metricData holds metric data and statistics
type metricData struct {
	metric     *Metric
	count      int64
	sum        float64
	min        float64
	max        float64
	histogram  *HistogramMetric
	lastUpdate time.Time
	mu         sync.RWMutex
}

// Exporter interface for metric exporters
type Exporter interface {
	Export(ctx context.Context, metrics []*Metric) error
	Name() string
}

// NewCollector creates a new metrics collector
func NewCollector(bufferSize int, flushInterval time.Duration) *Collector {
	c := &Collector{
		metrics:       make(map[string]*metricData),
		exporters:     make([]Exporter, 0),
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
	}

	// Start background flusher
	go c.backgroundFlusher()

	// Register system metrics
	c.registerSystemMetrics()

	return c
}

// Counter increments a counter metric
func (c *Collector) Counter(name string, value float64, labels map[string]string) {
	c.recordMetric(name, MetricTypeCounter, value, labels)
}

// Gauge sets a gauge metric
func (c *Collector) Gauge(name string, value float64, labels map[string]string) {
	c.recordMetric(name, MetricTypeGauge, value, labels)
}

// Histogram records a histogram metric
func (c *Collector) Histogram(name string, value float64, labels map[string]string) {
	c.recordMetric(name, MetricTypeHistogram, value, labels)
}

// Summary records a summary metric
func (c *Collector) Summary(name string, value float64, labels map[string]string) {
	c.recordMetric(name, MetricTypeSummary, value, labels)
}

// recordMetric records a metric
func (c *Collector) recordMetric(name string, metricType MetricType, value float64, labels map[string]string) {
	key := c.metricKey(name, labels)

	c.mu.Lock()
	defer c.mu.Unlock()

	data, exists := c.metrics[key]
	if !exists {
		data = &metricData{
			metric: &Metric{
				Name:   name,
				Type:   metricType,
				Labels: labels,
			},
			min:       value,
			max:       value,
			histogram: NewHistogramMetric(),
		}
		c.metrics[key] = data
	}

	data.mu.Lock()
	defer data.mu.Unlock()

	// Update statistics
	switch metricType {
	case MetricTypeCounter:
		data.sum += value
		data.count++
	case MetricTypeGauge:
		data.metric.Value = value
	case MetricTypeHistogram, MetricTypeSummary:
		data.histogram.Record(value)
		data.count++
		data.sum += value
		if value < data.min {
			data.min = value
		}
		if value > data.max {
			data.max = value
		}
	}

	data.lastUpdate = time.Now()

	// Check if buffer is full
	if len(c.metrics) >= c.bufferSize {
		go c.flush()
	}
}

// Timer records the duration of an operation
func (c *Collector) Timer(name string, labels map[string]string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start).Seconds()
		c.Histogram(name, duration, labels)
	}
}

// metricKey generates a unique key for a metric
func (c *Collector) metricKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += fmt.Sprintf("_%s_%s", k, v)
	}
	return key
}

// AddExporter adds a metric exporter
func (c *Collector) AddExporter(exporter Exporter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.exporters = append(c.exporters, exporter)
}

// flush exports all metrics
func (c *Collector) flush() {
	c.mu.Lock()

	if len(c.metrics) == 0 {
		c.mu.Unlock()
		return
	}

	// Prepare metrics for export
	metrics := make([]*Metric, 0, len(c.metrics))
	for _, data := range c.metrics {
		data.mu.RLock()
		m := &Metric{
			Name:      data.metric.Name,
			Type:      data.metric.Type,
			Labels:    data.metric.Labels,
			Timestamp: data.lastUpdate,
		}

		switch data.metric.Type {
		case MetricTypeCounter:
			m.Value = data.sum
		case MetricTypeGauge:
			m.Value = data.metric.Value
		case MetricTypeHistogram, MetricTypeSummary:
			m.Metadata = map[string]interface{}{
				"count": data.count,
				"sum":   data.sum,
				"min":   data.min,
				"max":   data.max,
				"p50":   data.histogram.Percentile(0.5),
				"p90":   data.histogram.Percentile(0.9),
				"p95":   data.histogram.Percentile(0.95),
				"p99":   data.histogram.Percentile(0.99),
			}
		}
		data.mu.RUnlock()

		metrics = append(metrics, m)
	}

	// Clear metrics after collecting
	c.metrics = make(map[string]*metricData)
	exporters := c.exporters
	c.mu.Unlock()

	// Export to all exporters
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for _, exporter := range exporters {
		wg.Add(1)
		go func(exp Exporter) {
			defer wg.Done()
			if err := exp.Export(ctx, metrics); err != nil {
				fmt.Printf("Failed to export metrics to %s: %v\n", exp.Name(), err)
			}
		}(exporter)
	}
	wg.Wait()
}

// backgroundFlusher periodically flushes metrics
func (c *Collector) backgroundFlusher() {
	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.flush()
		case <-c.stopCh:
			c.flush() // Final flush
			return
		}
	}
}

// Stop stops the collector
func (c *Collector) Stop() {
	close(c.stopCh)
}

// registerSystemMetrics registers system metrics collectors
func (c *Collector) registerSystemMetrics() {
	// Start goroutine for system metrics
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.collectSystemMetrics()
			case <-c.stopCh:
				return
			}
		}
	}()
}

// collectSystemMetrics collects system metrics
func (c *Collector) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Memory metrics
	c.Gauge("system.memory.alloc", float64(m.Alloc), map[string]string{"unit": "bytes"})
	c.Gauge("system.memory.total_alloc", float64(m.TotalAlloc), map[string]string{"unit": "bytes"})
	c.Gauge("system.memory.sys", float64(m.Sys), map[string]string{"unit": "bytes"})
	c.Gauge("system.memory.heap_alloc", float64(m.HeapAlloc), map[string]string{"unit": "bytes"})
	c.Gauge("system.memory.heap_inuse", float64(m.HeapInuse), map[string]string{"unit": "bytes"})

	// GC metrics
	c.Gauge("system.gc.count", float64(m.NumGC), nil)
	c.Gauge("system.gc.pause_total", float64(m.PauseTotalNs), map[string]string{"unit": "nanoseconds"})

	// Goroutine metrics
	c.Gauge("system.goroutines", float64(runtime.NumGoroutine()), nil)

	// CPU metrics
	c.Gauge("system.cpu.count", float64(runtime.NumCPU()), nil)
}

// HistogramMetric provides histogram functionality
type HistogramMetric struct {
	values []float64
	mu     sync.RWMutex
}

// NewHistogramMetric creates a new histogram metric
func NewHistogramMetric() *HistogramMetric {
	return &HistogramMetric{
		values: make([]float64, 0, 100),
	}
}

// Record records a value
func (h *HistogramMetric) Record(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.values = append(h.values, value)

	// Keep only last 1000 values
	if len(h.values) > 1000 {
		h.values = h.values[len(h.values)-1000:]
	}
}

// Percentile calculates a percentile
func (h *HistogramMetric) Percentile(p float64) float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.values) == 0 {
		return 0
	}

	// Simple percentile calculation (not exact but fast)
	index := int(float64(len(h.values)) * p)
	if index >= len(h.values) {
		index = len(h.values) - 1
	}

	return h.values[index]
}

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	endpoint string
	client   HTTPClient
}

type HTTPClient interface {
	Post(ctx context.Context, url string, data []byte) error
}

// NewPrometheusExporter creates a Prometheus exporter
func NewPrometheusExporter(endpoint string, client HTTPClient) *PrometheusExporter {
	return &PrometheusExporter{
		endpoint: endpoint,
		client:   client,
	}
}

func (p *PrometheusExporter) Name() string {
	return "prometheus"
}

func (p *PrometheusExporter) Export(ctx context.Context, metrics []*Metric) error {
	// Convert metrics to Prometheus format
	// Implementation would convert to Prometheus text format
	return nil
}

// CloudWatchExporter exports metrics to AWS CloudWatch
type CloudWatchExporter struct {
	namespace string
	region    string
}

func (c *CloudWatchExporter) Name() string {
	return "cloudwatch"
}

func (c *CloudWatchExporter) Export(ctx context.Context, metrics []*Metric) error {
	// Export to CloudWatch
	// Implementation would use AWS SDK
	return nil
}

// DatadogExporter exports metrics to Datadog
type DatadogExporter struct {
	apiKey   string
	endpoint string
}

func (d *DatadogExporter) Name() string {
	return "datadog"
}

func (d *DatadogExporter) Export(ctx context.Context, metrics []*Metric) error {
	// Export to Datadog
	// Implementation would use Datadog API
	return nil
}

// Global metrics instance
var (
	globalCollector *Collector
	once            sync.Once
)

// GetGlobalCollector returns the global metrics collector
func GetGlobalCollector() *Collector {
	once.Do(func() {
		globalCollector = NewCollector(1000, 30*time.Second)
	})
	return globalCollector
}

// Counter records a counter metric
func Counter(name string, value float64, labels map[string]string) {
	GetGlobalCollector().Counter(name, value, labels)
}

// Gauge records a gauge metric
func Gauge(name string, value float64, labels map[string]string) {
	GetGlobalCollector().Gauge(name, value, labels)
}

// Histogram records a histogram metric
func Histogram(name string, value float64, labels map[string]string) {
	GetGlobalCollector().Histogram(name, value, labels)
}

// Timer starts a timer and returns a function to stop it
func Timer(name string, labels map[string]string) func() {
	return GetGlobalCollector().Timer(name, labels)
}

// ResourceMetrics tracks resource-specific metrics
type ResourceMetrics struct {
	collector *Collector
	provider  string
}

// NewResourceMetrics creates resource-specific metrics
func NewResourceMetrics(provider string) *ResourceMetrics {
	return &ResourceMetrics{
		collector: GetGlobalCollector(),
		provider:  provider,
	}
}

// RecordDiscovery records discovery metrics
func (r *ResourceMetrics) RecordDiscovery(resourceType string, count int, duration time.Duration) {
	labels := map[string]string{
		"provider": r.provider,
		"type":     resourceType,
	}

	r.collector.Counter("discovery.resources.count", float64(count), labels)
	r.collector.Histogram("discovery.duration", duration.Seconds(), labels)
}

// RecordDrift records drift detection metrics
func (r *ResourceMetrics) RecordDrift(resourceType string, driftCount int, duration time.Duration) {
	labels := map[string]string{
		"provider": r.provider,
		"type":     resourceType,
	}

	r.collector.Counter("drift.detected.count", float64(driftCount), labels)
	r.collector.Histogram("drift.detection.duration", duration.Seconds(), labels)
}

// RecordRemediation records remediation metrics
func (r *ResourceMetrics) RecordRemediation(resourceType string, success bool, duration time.Duration) {
	labels := map[string]string{
		"provider": r.provider,
		"type":     resourceType,
		"success":  fmt.Sprintf("%t", success),
	}

	r.collector.Counter("remediation.attempts", 1, labels)
	r.collector.Histogram("remediation.duration", duration.Seconds(), labels)
}

// RecordAPICall records API call metrics
func (r *ResourceMetrics) RecordAPICall(operation string, success bool, duration time.Duration) {
	labels := map[string]string{
		"provider":  r.provider,
		"operation": operation,
		"success":   fmt.Sprintf("%t", success),
	}

	r.collector.Counter("api.calls", 1, labels)
	r.collector.Histogram("api.duration", duration.Seconds(), labels)

	if !success {
		r.collector.Counter("api.errors", 1, labels)
	}
}

// RateLimitMetrics tracks rate limiting metrics
type RateLimitMetrics struct {
	requests  int64
	allowed   int64
	rejected  int64
	throttled int64
}

// RecordRequest records a rate limit request
func (r *RateLimitMetrics) RecordRequest(allowed bool, throttled bool) {
	atomic.AddInt64(&r.requests, 1)

	if allowed {
		atomic.AddInt64(&r.allowed, 1)
	} else {
		atomic.AddInt64(&r.rejected, 1)
	}

	if throttled {
		atomic.AddInt64(&r.throttled, 1)
	}

	// Export metrics
	Counter("ratelimit.requests", 1, nil)
	if allowed {
		Counter("ratelimit.allowed", 1, nil)
	} else {
		Counter("ratelimit.rejected", 1, nil)
	}
	if throttled {
		Counter("ratelimit.throttled", 1, nil)
	}
}

// GetStats returns rate limit statistics
func (r *RateLimitMetrics) GetStats() map[string]int64 {
	return map[string]int64{
		"requests":  atomic.LoadInt64(&r.requests),
		"allowed":   atomic.LoadInt64(&r.allowed),
		"rejected":  atomic.LoadInt64(&r.rejected),
		"throttled": atomic.LoadInt64(&r.throttled),
	}
}
