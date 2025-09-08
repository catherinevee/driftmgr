package health

import (
	"context"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// HealthStatus represents the health status of a resource
type HealthStatus string

const (
	HealthStatusHealthy  HealthStatus = "healthy"
	HealthStatusWarning  HealthStatus = "warning"
	HealthStatusCritical HealthStatus = "critical"
	HealthStatusUnknown  HealthStatus = "unknown"
	HealthStatusDegraded HealthStatus = "degraded"
)

// HealthMetric represents a health metric
type HealthMetric struct {
	Name        string                 `json:"name"`
	Value       float64                `json:"value"`
	Unit        string                 `json:"unit"`
	Threshold   float64                `json:"threshold"`
	Status      HealthStatus           `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// HealthCheck represents a health check
type HealthCheck struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	ResourceID  string                 `json:"resource_id"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message"`
	LastChecked time.Time              `json:"last_checked"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// HealthReport represents a comprehensive health report
type HealthReport struct {
	ResourceID      string                 `json:"resource_id"`
	ResourceType    string                 `json:"resource_type"`
	Provider        string                 `json:"provider"`
	Region          string                 `json:"region"`
	OverallStatus   HealthStatus           `json:"overall_status"`
	HealthScore     float64                `json:"health_score"`
	LastUpdated     time.Time              `json:"last_updated"`
	Checks          []HealthCheck          `json:"checks"`
	Metrics         []HealthMetric         `json:"metrics"`
	Recommendations []string               `json:"recommendations,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// HealthChecker interface for performing health checks
type HealthChecker interface {
	Check(ctx context.Context, resource *models.Resource) (*HealthCheck, error)
	GetType() string
	GetDescription() string
}

// AlertThreshold defines alerting thresholds
type AlertThreshold struct {
	Warning  float64 `json:"warning"`
	Critical float64 `json:"critical"`
}

// EventBus interface for health events
type EventBus interface {
	PublishHealthEvent(event HealthEvent) error
}

// HealthEvent represents a health-related event
type HealthEvent struct {
	Type       string                 `json:"type"`
	ResourceID string                 `json:"resource_id"`
	Status     HealthStatus           `json:"status"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// HealthTrend represents a health trend over time
type HealthTrend struct {
	Timestamp   time.Time    `json:"timestamp"`
	HealthScore float64      `json:"health_score"`
	Status      HealthStatus `json:"status"`
}

// HealthAlert represents a health alert
type HealthAlert struct {
	ID         string    `json:"id"`
	ResourceID string    `json:"resource_id"`
	Type       string    `json:"type"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
	Status     string    `json:"status"`
}
