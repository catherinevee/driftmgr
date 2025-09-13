package checkers

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestHealthStatus(t *testing.T) {
	statuses := []HealthStatus{
		HealthStatusHealthy,
		HealthStatusWarning,
		HealthStatusCritical,
		HealthStatusUnknown,
		HealthStatusDegraded,
	}

	expectedStrings := []string{
		"healthy",
		"warning",
		"critical",
		"unknown",
		"degraded",
	}

	for i, status := range statuses {
		assert.Equal(t, HealthStatus(expectedStrings[i]), status)
		assert.NotEmpty(t, string(status))
	}
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name  string
		check HealthCheck
	}{
		{
			name: "healthy check",
			check: HealthCheck{
				ID:          "check-1",
				Name:        "CPU Usage",
				Type:        "performance",
				ResourceID:  "i-12345",
				Status:      HealthStatusHealthy,
				Message:     "CPU usage is within normal range (15%)",
				LastChecked: time.Now(),
				Duration:    100 * time.Millisecond,
				Metadata: map[string]interface{}{
					"cpu_percent": 15,
					"threshold":   80,
				},
				Tags: []string{"performance", "cpu"},
			},
		},
		{
			name: "warning check",
			check: HealthCheck{
				ID:          "check-2",
				Name:        "Memory Usage",
				Type:        "performance",
				ResourceID:  "i-12345",
				Status:      HealthStatusWarning,
				Message:     "Memory usage is high (75%)",
				LastChecked: time.Now(),
				Duration:    50 * time.Millisecond,
				Metadata: map[string]interface{}{
					"memory_percent": 75,
					"threshold":      70,
				},
			},
		},
		{
			name: "critical check",
			check: HealthCheck{
				ID:          "check-3",
				Name:        "Disk Space",
				Type:        "storage",
				ResourceID:  "vol-12345",
				Status:      HealthStatusCritical,
				Message:     "Disk space critically low (95% used)",
				LastChecked: time.Now(),
				Duration:    200 * time.Millisecond,
				Metadata: map[string]interface{}{
					"disk_used_percent": 95,
					"threshold":         90,
				},
			},
		},
		{
			name: "degraded service",
			check: HealthCheck{
				ID:          "check-4",
				Name:        "Service Health",
				Type:        "availability",
				ResourceID:  "svc-12345",
				Status:      HealthStatusDegraded,
				Message:     "Service is responding slowly",
				LastChecked: time.Now(),
				Duration:    1 * time.Second,
			},
		},
		{
			name: "unknown status",
			check: HealthCheck{
				ID:          "check-5",
				Name:        "Network Connectivity",
				Type:        "network",
				ResourceID:  "vpc-12345",
				Status:      HealthStatusUnknown,
				Message:     "Unable to determine network status",
				LastChecked: time.Now(),
				Duration:    5 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, tt.check.ID)
			assert.NotEmpty(t, tt.check.Name)
			assert.NotEmpty(t, tt.check.Type)
			assert.NotEmpty(t, tt.check.ResourceID)
			assert.NotEmpty(t, tt.check.Status)
			assert.NotEmpty(t, tt.check.Message)
			assert.NotZero(t, tt.check.LastChecked)
			assert.Greater(t, tt.check.Duration, time.Duration(0))

			// Check status-specific assertions
			switch tt.check.Status {
			case HealthStatusHealthy:
				assert.Contains(t, tt.check.Message, "normal")
			case HealthStatusWarning:
				assert.Contains(t, tt.check.Message, "high")
			case HealthStatusCritical:
				assert.Contains(t, tt.check.Message, "critical")
			case HealthStatusDegraded:
				assert.Contains(t, tt.check.Message, "slow")
			case HealthStatusUnknown:
				assert.Contains(t, tt.check.Message, "Unable")
			}
		})
	}
}

// Mock health checker for testing
type mockHealthChecker struct {
	checkType   string
	description string
	status      HealthStatus
	err         error
}

func (m *mockHealthChecker) Check(ctx context.Context, resource *models.Resource) (*HealthCheck, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &HealthCheck{
		ID:          "mock-check",
		Name:        "Mock Health Check",
		Type:        m.checkType,
		ResourceID:  resource.ID,
		Status:      m.status,
		Message:     "Mock check result",
		LastChecked: time.Now(),
		Duration:    10 * time.Millisecond,
	}, nil
}

func (m *mockHealthChecker) GetType() string {
	return m.checkType
}

func (m *mockHealthChecker) GetDescription() string {
	return m.description
}

func TestHealthChecker_Interface(t *testing.T) {
	checker := &mockHealthChecker{
		checkType:   "mock",
		description: "Mock health checker for testing",
		status:      HealthStatusHealthy,
	}

	// Test GetType
	assert.Equal(t, "mock", checker.GetType())

	// Test GetDescription
	assert.Equal(t, "Mock health checker for testing", checker.GetDescription())

	// Test Check
	ctx := context.Background()
	resource := &models.Resource{
		ID:       "res-123",
		Type:     "instance",
		Provider: "aws",
	}

	check, err := checker.Check(ctx, resource)
	assert.NoError(t, err)
	assert.NotNil(t, check)
	assert.Equal(t, "res-123", check.ResourceID)
	assert.Equal(t, HealthStatusHealthy, check.Status)
}

func TestHealthChecker_Error(t *testing.T) {
	checker := &mockHealthChecker{
		checkType: "mock",
		err:       assert.AnError,
	}

	ctx := context.Background()
	resource := &models.Resource{
		ID: "res-123",
	}

	check, err := checker.Check(ctx, resource)
	assert.Error(t, err)
	assert.Nil(t, check)
}

func TestHealthCheckTypes(t *testing.T) {
	types := []string{
		"performance",
		"availability",
		"security",
		"compliance",
		"cost",
		"network",
		"storage",
		"database",
	}

	for _, checkType := range types {
		t.Run(checkType, func(t *testing.T) {
			check := HealthCheck{
				Type: checkType,
			}
			assert.Equal(t, checkType, check.Type)
		})
	}
}

func TestHealthCheckMetadata(t *testing.T) {
	check := HealthCheck{
		ID:   "check-metadata",
		Name: "Metadata Test",
		Metadata: map[string]interface{}{
			"string_value": "test",
			"int_value":    42,
			"float_value":  3.14,
			"bool_value":   true,
			"array_value":  []string{"a", "b", "c"},
			"nested_object": map[string]interface{}{
				"key": "value",
			},
		},
	}

	assert.NotNil(t, check.Metadata)
	assert.Equal(t, "test", check.Metadata["string_value"])
	assert.Equal(t, 42, check.Metadata["int_value"])
	assert.Equal(t, 3.14, check.Metadata["float_value"])
	assert.Equal(t, true, check.Metadata["bool_value"])
	assert.NotNil(t, check.Metadata["array_value"])
	assert.NotNil(t, check.Metadata["nested_object"])
}

func TestHealthCheckTags(t *testing.T) {
	check := HealthCheck{
		ID:   "check-tags",
		Name: "Tags Test",
		Tags: []string{"critical", "production", "database", "performance"},
	}

	assert.Len(t, check.Tags, 4)
	assert.Contains(t, check.Tags, "critical")
	assert.Contains(t, check.Tags, "production")
	assert.Contains(t, check.Tags, "database")
	assert.Contains(t, check.Tags, "performance")
}

func BenchmarkHealthCheck(b *testing.B) {
	for i := 0; i < b.N; i++ {
		check := HealthCheck{
			ID:          "bench-check",
			Name:        "Benchmark Check",
			Type:        "performance",
			ResourceID:  "res-123",
			Status:      HealthStatusHealthy,
			Message:     "Benchmark test",
			LastChecked: time.Now(),
			Duration:    100 * time.Millisecond,
		}
		_ = check.Status
	}
}

func BenchmarkHealthChecker(b *testing.B) {
	checker := &mockHealthChecker{
		checkType: "performance",
		status:    HealthStatusHealthy,
	}

	ctx := context.Background()
	resource := &models.Resource{
		ID: "res-123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = checker.Check(ctx, resource)
	}
}
