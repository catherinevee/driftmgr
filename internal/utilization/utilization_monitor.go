package utilization

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// UtilizationMonitor tracks resource utilization and performance metrics
type UtilizationMonitor struct {
	providers map[string]UtilizationProvider
}

// UtilizationProvider interface for different cloud providers
type UtilizationProvider interface {
	GetResourceUtilization(ctx context.Context, resource models.Resource) (*UtilizationMetrics, error)
	GetPerformanceMetrics(ctx context.Context, resource models.Resource) (*PerformanceMetrics, error)
}

// UtilizationMetrics represents resource utilization data
type UtilizationMetrics struct {
	ResourceID         string
	CPUUtilization     float64
	MemoryUtilization  float64
	DiskUtilization    float64
	NetworkUtilization float64
	LastUpdated        time.Time
	Trend              UtilizationTrend
}

// PerformanceMetrics represents performance data
type PerformanceMetrics struct {
	ResourceID   string
	ResponseTime time.Duration
	Throughput   float64
	ErrorRate    float64
	Availability float64
	LastUpdated  time.Time
}

// UtilizationTrend shows utilization movement over time
type UtilizationTrend struct {
	Direction  string // "increasing", "decreasing", "stable"
	Percentage float64
	Period     string
}

// NewUtilizationMonitor creates a new utilization monitor
func NewUtilizationMonitor() *UtilizationMonitor {
	return &UtilizationMonitor{
		providers: make(map[string]UtilizationProvider),
	}
}

// RegisterProvider registers a utilization provider
func (um *UtilizationMonitor) RegisterProvider(name string, provider UtilizationProvider) {
	um.providers[name] = provider
}

// GetResourceUtilization gets utilization metrics for a resource
func (um *UtilizationMonitor) GetResourceUtilization(ctx context.Context, resource models.Resource) (*UtilizationMetrics, error) {
	provider, exists := um.providers[resource.Provider]
	if !exists {
		return nil, fmt.Errorf("utilization provider for %s not registered", resource.Provider)
	}

	return provider.GetResourceUtilization(ctx, resource)
}

// GetPerformanceMetrics gets performance metrics for a resource
func (um *UtilizationMonitor) GetPerformanceMetrics(ctx context.Context, resource models.Resource) (*PerformanceMetrics, error) {
	provider, exists := um.providers[resource.Provider]
	if !exists {
		return nil, fmt.Errorf("utilization provider for %s not registered", resource.Provider)
	}

	return provider.GetPerformanceMetrics(ctx, resource)
}
