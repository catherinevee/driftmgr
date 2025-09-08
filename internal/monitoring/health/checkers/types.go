package checkers

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

// HealthChecker interface for performing health checks
type HealthChecker interface {
	Check(ctx context.Context, resource *models.Resource) (*HealthCheck, error)
	GetType() string
	GetDescription() string
}
