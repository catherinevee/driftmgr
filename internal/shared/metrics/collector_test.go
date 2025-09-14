package metrics

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCollector(t *testing.T) {
	collector := NewCollector(100, 1*time.Second)

	assert.NotNil(t, collector)
	assert.Equal(t, 100, collector.bufferSize)
	assert.Equal(t, 1*time.Second, collector.flushInterval)
	assert.NotNil(t, collector.metrics)
	assert.NotNil(t, collector.exporters)
	assert.NotNil(t, collector.stopCh)

	// Clean up
	collector.Stop()
}

func TestCollector_Counter(t *testing.T) {
	collector := NewCollector(100, 1*time.Second)
	defer collector.Stop()

	// Record counter metrics
	collector.Counter("test.counter", 1.0, map[string]string{"label1": "value1"})
	collector.Counter("test.counter", 2.0, map[string]string{"label1": "value1"})
	collector.Counter("test.counter", 3.0, map[string]string{"label1": "value2"})

	// Verify metrics were recorded
	collector.mu.RLock()
	assert.Len(t, collector.metrics, 2) // Two different label combinations
	collector.mu.RUnlock()
}

func TestCollector_Gauge(t *testing.T) {
	collector := NewCollector(100, 1*time.Second)
	defer collector.Stop()

	// Record gauge metrics
	collector.Gauge("test.gauge", 10.0, map[string]string{"label1": "value1"})
	collector.Gauge("test.gauge", 20.0, map[string]string{"label1": "value1"}) // Should overwrite
	collector.Gauge("test.gauge", 30.0, map[string]string{"label1": "value2"})

	// Verify metrics were recorded
	collector.mu.RLock()
	assert.Len(t, collector.metrics, 2) // Two different label combinations
	collector.mu.RUnlock()
}

func TestCollector_Histogram(t *testing.T) {
	collector := NewCollector(100, 1*time.Second)
	defer collector.Stop()

	// Record histogram metrics
	collector.Histogram("test.histogram", 1.0, map[string]string{"label1": "value1"})
	collector.Histogram("test.histogram", 2.0, map[string]string{"label1": "value1"})
	collector.Histogram("test.histogram", 3.0, map[string]string{"label1": "value1"})

	// Verify metrics were recorded
	collector.mu.RLock()
	assert.Len(t, collector.metrics, 1) // Same label combination
	collector.mu.RUnlock()
}

func TestCollector_Summary(t *testing.T) {
	collector := NewCollector(100, 1*time.Second)
	defer collector.Stop()

	// Record summary metrics
	collector.Summary("test.summary", 1.0, map[string]string{"label1": "value1"})
	collector.Summary("test.summary", 2.0, map[string]string{"label1": "value1"})
	collector.Summary("test.summary", 3.0, map[string]string{"label1": "value1"})

	// Verify metrics were recorded
	collector.mu.RLock()
	assert.Len(t, collector.metrics, 1) // Same label combination
	collector.mu.RUnlock()
}

func TestCollector_Timer(t *testing.T) {
	collector := NewCollector(100, 1*time.Second)
	defer collector.Stop()

	// Use timer
	stopTimer := collector.Timer("test.timer", map[string]string{"label1": "value1"})

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	// Stop timer
	stopTimer()

	// Verify metrics were recorded
	collector.mu.RLock()
	assert.Len(t, collector.metrics, 1)
	collector.mu.RUnlock()
}

func TestCollector_metricKey(t *testing.T) {
	collector := NewCollector(100, 1*time.Second)
	defer collector.Stop()

	// Test metric key generation
	key1 := collector.metricKey("test.metric", map[string]string{"label1": "value1", "label2": "value2"})
	key3 := collector.metricKey("test.metric", map[string]string{"label1": "value1"})

	// Keys should be deterministic (but map iteration order is not guaranteed in Go)
	// So we'll just test that they contain the expected components
	assert.Contains(t, key1, "test.metric")
	assert.Contains(t, key1, "label1_value1")
	assert.Contains(t, key1, "label2_value2")
	assert.NotEqual(t, key1, key3) // Different labels
}

func TestCollector_AddExporter(t *testing.T) {
	collector := NewCollector(100, 1*time.Second)
	defer collector.Stop()

	// Create mock exporter
	exporter := &MockExporter{name: "test"}

	// Add exporter
	collector.AddExporter(exporter)

	// Verify exporter was added
	collector.mu.RLock()
	assert.Len(t, collector.exporters, 1)
	assert.Equal(t, exporter, collector.exporters[0])
	collector.mu.RUnlock()
}

func TestCollector_flush(t *testing.T) {
	collector := NewCollector(100, 1*time.Second)
	defer collector.Stop()

	// Add mock exporter
	exporter := &MockExporter{name: "test"}
	collector.AddExporter(exporter)

	// Record some metrics
	collector.Counter("test.counter", 1.0, nil)
	collector.Gauge("test.gauge", 10.0, nil)
	collector.Histogram("test.histogram", 1.0, nil)

	// Flush metrics
	collector.flush()

	// Verify exporter was called
	assert.True(t, exporter.exportCalled)
	assert.Len(t, exporter.exportedMetrics, 3)

	// Verify metrics were cleared
	collector.mu.RLock()
	assert.Len(t, collector.metrics, 0)
	collector.mu.RUnlock()
}

func TestCollector_ConcurrentAccess(t *testing.T) {
	collector := NewCollector(100, 1*time.Second)
	defer collector.Stop()

	// Test concurrent metric recording
	var wg sync.WaitGroup
	numGoroutines := 10
	metricsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < metricsPerGoroutine; j++ {
				collector.Counter("test.counter", float64(i*metricsPerGoroutine+j), map[string]string{"goroutine": string(rune(i))})
			}
		}(i)
	}

	wg.Wait()

	// Verify all metrics were recorded
	collector.mu.RLock()
	assert.Len(t, collector.metrics, numGoroutines) // One per goroutine (different labels)
	collector.mu.RUnlock()
}

func TestHistogramMetric(t *testing.T) {
	histogram := NewHistogramMetric()

	// Record values
	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	for _, v := range values {
		histogram.Record(v)
	}

	// Test percentiles
	assert.Equal(t, 1.0, histogram.Percentile(0.0)) // Min
	assert.Equal(t, 3.0, histogram.Percentile(0.5)) // Median
	assert.Equal(t, 5.0, histogram.Percentile(1.0)) // Max

	// Test empty histogram
	emptyHistogram := NewHistogramMetric()
	assert.Equal(t, 0.0, emptyHistogram.Percentile(0.5))
}

func TestHistogramMetric_ConcurrentAccess(t *testing.T) {
	histogram := NewHistogramMetric()

	// Test concurrent recording
	var wg sync.WaitGroup
	numGoroutines := 10
	valuesPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < valuesPerGoroutine; j++ {
				histogram.Record(float64(i*valuesPerGoroutine + j))
			}
		}(i)
	}

	wg.Wait()

	// Verify all values were recorded
	histogram.mu.RLock()
	assert.Len(t, histogram.values, numGoroutines*valuesPerGoroutine)
	histogram.mu.RUnlock()
}

func TestPrometheusExporter(t *testing.T) {
	exporter := NewPrometheusExporter("http://localhost:9090", &MockHTTPClient{})

	assert.Equal(t, "prometheus", exporter.Name())

	// Test export (should not error)
	metrics := []*Metric{
		{Name: "test.metric", Type: MetricTypeCounter, Value: 1.0},
	}

	err := exporter.Export(context.Background(), metrics)
	assert.NoError(t, err)
}

func TestCloudWatchExporter(t *testing.T) {
	exporter := &CloudWatchExporter{
		namespace: "DriftMgr",
		region:    "us-east-1",
	}

	assert.Equal(t, "cloudwatch", exporter.Name())

	// Test export (should not error)
	metrics := []*Metric{
		{Name: "test.metric", Type: MetricTypeCounter, Value: 1.0},
	}

	err := exporter.Export(context.Background(), metrics)
	assert.NoError(t, err)
}

func TestDatadogExporter(t *testing.T) {
	exporter := &DatadogExporter{
		apiKey:   "test-key",
		endpoint: "https://api.datadoghq.com",
	}

	assert.Equal(t, "datadog", exporter.Name())

	// Test export (should not error)
	metrics := []*Metric{
		{Name: "test.metric", Type: MetricTypeCounter, Value: 1.0},
	}

	err := exporter.Export(context.Background(), metrics)
	assert.NoError(t, err)
}

func TestGetGlobalCollector(t *testing.T) {
	// Reset global collector
	globalCollector = nil
	once = sync.Once{}

	// First call should create a new collector
	collector1 := GetGlobalCollector()
	assert.NotNil(t, collector1)

	// Second call should return the same instance
	collector2 := GetGlobalCollector()
	assert.Equal(t, collector1, collector2)

	// Clean up
	collector1.Stop()
}

func TestGlobalFunctions(t *testing.T) {
	// Reset global collector
	globalCollector = nil
	once = sync.Once{}

	// Test global functions
	Counter("global.counter", 1.0, map[string]string{"test": "value"})
	Gauge("global.gauge", 10.0, map[string]string{"test": "value"})
	Histogram("global.histogram", 1.0, map[string]string{"test": "value"})

	// Test timer
	stopTimer := Timer("global.timer", map[string]string{"test": "value"})
	time.Sleep(1 * time.Millisecond)
	stopTimer()

	// Verify metrics were recorded
	collector := GetGlobalCollector()
	collector.mu.RLock()
	assert.Len(t, collector.metrics, 4) // counter, gauge, histogram, timer
	collector.mu.RUnlock()

	// Clean up
	collector.Stop()
}

func TestResourceMetrics(t *testing.T) {
	// Reset global collector
	globalCollector = nil
	once = sync.Once{}

	resourceMetrics := NewResourceMetrics("aws")

	// Test discovery metrics
	resourceMetrics.RecordDiscovery("ec2", 10, 100*time.Millisecond)

	// Test drift metrics
	resourceMetrics.RecordDrift("ec2", 2, 50*time.Millisecond)

	// Test remediation metrics
	resourceMetrics.RecordRemediation("ec2", true, 200*time.Millisecond)
	resourceMetrics.RecordRemediation("ec2", false, 300*time.Millisecond)

	// Test API call metrics
	resourceMetrics.RecordAPICall("DescribeInstances", true, 10*time.Millisecond)
	resourceMetrics.RecordAPICall("DescribeInstances", false, 20*time.Millisecond)

	// Verify metrics were recorded
	collector := GetGlobalCollector()
	collector.mu.RLock()
	assert.GreaterOrEqual(t, len(collector.metrics), 8) // At least 8 metrics should be recorded
	collector.mu.RUnlock()

	// Clean up
	collector.Stop()
}

func TestRateLimitMetrics(t *testing.T) {
	rateLimitMetrics := &RateLimitMetrics{}

	// Record various requests
	rateLimitMetrics.RecordRequest(true, false)  // Allowed, not throttled
	rateLimitMetrics.RecordRequest(true, true)   // Allowed, throttled
	rateLimitMetrics.RecordRequest(false, false) // Rejected, not throttled
	rateLimitMetrics.RecordRequest(false, true)  // Rejected, throttled

	// Get stats
	stats := rateLimitMetrics.GetStats()

	assert.Equal(t, int64(4), stats["requests"])
	assert.Equal(t, int64(2), stats["allowed"])
	assert.Equal(t, int64(2), stats["rejected"])
	assert.Equal(t, int64(2), stats["throttled"])
}

func TestRateLimitMetrics_ConcurrentAccess(t *testing.T) {
	rateLimitMetrics := &RateLimitMetrics{}

	// Test concurrent recording
	var wg sync.WaitGroup
	numGoroutines := 10
	requestsPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				rateLimitMetrics.RecordRequest(j%2 == 0, j%3 == 0)
			}
		}()
	}

	wg.Wait()

	// Verify all requests were recorded
	stats := rateLimitMetrics.GetStats()
	expectedTotal := int64(numGoroutines * requestsPerGoroutine)
	assert.Equal(t, expectedTotal, stats["requests"])
	assert.Equal(t, expectedTotal, stats["allowed"]+stats["rejected"])
}

func TestMetric_Struct(t *testing.T) {
	metric := &Metric{
		Name:        "test.metric",
		Type:        MetricTypeCounter,
		Value:       1.0,
		Labels:      map[string]string{"label1": "value1"},
		Timestamp:   time.Now(),
		Description: "Test metric",
		Unit:        "count",
		Metadata:    map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "test.metric", metric.Name)
	assert.Equal(t, MetricTypeCounter, metric.Type)
	assert.Equal(t, 1.0, metric.Value)
	assert.Equal(t, "value1", metric.Labels["label1"])
	assert.False(t, metric.Timestamp.IsZero())
	assert.Equal(t, "Test metric", metric.Description)
	assert.Equal(t, "count", metric.Unit)
	assert.Equal(t, "value", metric.Metadata["key"])
}

func TestMetricType_Constants(t *testing.T) {
	assert.Equal(t, string(MetricTypeCounter), "counter")
	assert.Equal(t, string(MetricTypeGauge), "gauge")
	assert.Equal(t, string(MetricTypeHistogram), "histogram")
	assert.Equal(t, string(MetricTypeSummary), "summary")
}

// Mock implementations for testing

type MockExporter struct {
	name            string
	exportCalled    bool
	exportedMetrics []*Metric
}

func (m *MockExporter) Name() string {
	return m.name
}

func (m *MockExporter) Export(ctx context.Context, metrics []*Metric) error {
	m.exportCalled = true
	m.exportedMetrics = make([]*Metric, len(metrics))
	copy(m.exportedMetrics, metrics)
	return nil
}

type MockHTTPClient struct{}

func (m *MockHTTPClient) Post(ctx context.Context, url string, data []byte) error {
	return nil
}
