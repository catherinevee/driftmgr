package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTelemetry(t *testing.T) {
	config := &Config{
		Enabled: true,
		Service: ServiceConfig{
			Name:        "test-service",
			Version:     "1.0.0",
			Environment: "test",
		},
		Tracing: TracingConfig{
			Enabled:  true,
			Endpoint: "http://localhost:14268/api/traces",
			Sampler:  1.0,
		},
		Metrics: MetricsConfig{
			Enabled:  true,
			Endpoint: "localhost:4317",
			Interval: 30 * time.Second,
		},
	}

	telemetry, err := New(context.Background(), config)
	require.NoError(t, err)
	assert.NotNil(t, telemetry)
	
	// Clean up
	err = telemetry.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestTelemetryDisabled(t *testing.T) {
	config := &Config{
		Enabled: false,
	}

	telemetry, err := New(context.Background(), config)
	require.NoError(t, err)
	assert.NotNil(t, telemetry)
	assert.Nil(t, telemetry.tracer)
	assert.Nil(t, telemetry.meter)
}

func TestStartSpan(t *testing.T) {
	config := &Config{
		Enabled: true,
		Service: ServiceConfig{
			Name: "test-service",
		},
		Tracing: TracingConfig{
			Enabled: true,
		},
	}

	telemetry, err := New(context.Background(), config)
	require.NoError(t, err)
	defer telemetry.Shutdown(context.Background())

	ctx := context.Background()
	newCtx, span := telemetry.StartSpan(ctx, "test-operation")
	assert.NotNil(t, newCtx)
	assert.NotNil(t, span)
	
	span.End()
}

func TestRecordDiscovery(t *testing.T) {
	config := &Config{
		Enabled: true,
		Service: ServiceConfig{
			Name: "test-service",
		},
		Metrics: MetricsConfig{
			Enabled: true,
		},
	}

	telemetry, err := New(context.Background(), config)
	require.NoError(t, err)
	defer telemetry.Shutdown(context.Background())

	ctx := context.Background()
	
	// Test successful discovery
	telemetry.RecordDiscovery(ctx, "aws", 100, 5*time.Second, nil)
	
	// Test failed discovery
	telemetry.RecordDiscovery(ctx, "azure", 0, 10*time.Second, assert.AnError)
}

func TestRecordDrift(t *testing.T) {
	config := &Config{
		Enabled: true,
		Service: ServiceConfig{
			Name: "test-service",
		},
		Metrics: MetricsConfig{
			Enabled: true,
		},
	}

	telemetry, err := New(context.Background(), config)
	require.NoError(t, err)
	defer telemetry.Shutdown(context.Background())

	ctx := context.Background()
	telemetry.RecordDrift(ctx, "aws", 10, "high")
}

func TestRecordError(t *testing.T) {
	config := &Config{
		Enabled: true,
		Service: ServiceConfig{
			Name: "test-service",
		},
		Metrics: MetricsConfig{
			Enabled: true,
		},
	}

	telemetry, err := New(context.Background(), config)
	require.NoError(t, err)
	defer telemetry.Shutdown(context.Background())

	ctx := context.Background()
	telemetry.RecordError(ctx, "operation", assert.AnError)
}

func TestRecordDuration(t *testing.T) {
	config := &Config{
		Enabled: true,
		Service: ServiceConfig{
			Name: "test-service",
		},
		Metrics: MetricsConfig{
			Enabled: true,
		},
	}

	telemetry, err := New(context.Background(), config)
	require.NoError(t, err)
	defer telemetry.Shutdown(context.Background())

	ctx := context.Background()
	telemetry.RecordDuration(ctx, "operation", 100*time.Millisecond)
}

func TestGlobalTelemetry(t *testing.T) {
	config := &Config{
		Enabled: true,
		Service: ServiceConfig{
			Name: "test-service",
		},
	}

	telemetry, err := New(context.Background(), config)
	require.NoError(t, err)
	
	Set(telemetry)
	assert.Equal(t, telemetry, Get())
	
	defer telemetry.Shutdown(context.Background())
}

func TestTelemetryWithLabels(t *testing.T) {
	config := &Config{
		Enabled: true,
		Service: ServiceConfig{
			Name: "test-service",
		},
		Metrics: MetricsConfig{
			Enabled: true,
		},
	}

	telemetry, err := New(context.Background(), config)
	require.NoError(t, err)
	defer telemetry.Shutdown(context.Background())

	ctx := context.Background()
	
	// Create span with attributes
	ctx, span := telemetry.StartSpan(ctx, "test-operation",
		WithAttributes(map[string]interface{}{
			"user_id": "12345",
			"action":  "create",
		}),
	)
	span.End()
}

func TestConcurrentTelemetry(t *testing.T) {
	config := &Config{
		Enabled: true,
		Service: ServiceConfig{
			Name: "test-service",
		},
		Metrics: MetricsConfig{
			Enabled: true,
		},
	}

	telemetry, err := New(context.Background(), config)
	require.NoError(t, err)
	defer telemetry.Shutdown(context.Background())

	ctx := context.Background()
	
	// Test concurrent metric recording
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			telemetry.RecordDiscovery(ctx, "aws", id*10, time.Duration(id)*time.Second, nil)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkStartSpan(b *testing.B) {
	config := &Config{
		Enabled: true,
		Service: ServiceConfig{
			Name: "test-service",
		},
		Tracing: TracingConfig{
			Enabled: true,
		},
	}

	telemetry, _ := New(context.Background(), config)
	defer telemetry.Shutdown(context.Background())
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := telemetry.StartSpan(ctx, "benchmark-operation")
		span.End()
	}
}

func BenchmarkRecordMetric(b *testing.B) {
	config := &Config{
		Enabled: true,
		Service: ServiceConfig{
			Name: "test-service",
		},
		Metrics: MetricsConfig{
			Enabled: true,
		},
	}

	telemetry, _ := New(context.Background(), config)
	defer telemetry.Shutdown(context.Background())
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		telemetry.RecordDiscovery(ctx, "aws", 100, time.Second, nil)
	}
}