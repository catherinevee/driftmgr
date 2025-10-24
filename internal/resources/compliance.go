package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// ComplianceManager manages resource compliance
type ComplianceManager struct {
	compliance map[string]*models.ComplianceStatus
	policies   map[string]*models.ResourcePolicy
	mu         sync.RWMutex
}

// NewComplianceManager creates a new compliance manager
func NewComplianceManager() *ComplianceManager {
	return &ComplianceManager{
		compliance: make(map[string]*models.ComplianceStatus),
		policies:   make(map[string]*models.ResourcePolicy),
	}
}

// CheckCompliance checks compliance for a resource
func (m *ComplianceManager) CheckCompliance(ctx context.Context, resource *models.CloudResource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create compliance status
	status := m.compliance[resource.ID]
	if status == nil {
		status = &models.ComplianceStatus{
			Status:      models.ComplianceLevelUnknown,
			LastChecked: time.Now(),
		}
	}

	// Perform compliance checks
	violations := m.performComplianceChecks(resource)

	if len(violations) == 0 {
		status.Status = models.ComplianceLevelCompliant
	} else {
		// Check if any violations are critical
		hasCritical := false
		for _, violation := range violations {
			if violation.Severity == "critical" {
				hasCritical = true
				break
			}
		}

		if hasCritical {
			status.Status = models.ComplianceLevelNonCompliant
		} else {
			status.Status = models.ComplianceLevelWarning
		}
	}

	status.Violations = violations
	status.LastChecked = time.Now()
	status.NextCheck = time.Now().Add(24 * time.Hour) // Check daily

	m.compliance[resource.ID] = status
	return nil
}

// Enrich enriches a resource with compliance data
func (m *ComplianceManager) Enrich(ctx context.Context, resource *models.CloudResource) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := m.compliance[resource.ID]
	if status == nil {
		return nil // No compliance data available
	}

	// Create a copy of the compliance status
	resource.Compliance = *status
	return nil
}

// GetCompliance retrieves compliance status for a resource
func (m *ComplianceManager) GetCompliance(ctx context.Context, resourceID string) (*models.ComplianceStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := m.compliance[resourceID]
	if status == nil {
		return nil, fmt.Errorf("compliance status for resource %s not found", resourceID)
	}

	return status, nil
}

// DeleteCompliance deletes compliance data for a resource
func (m *ComplianceManager) DeleteCompliance(ctx context.Context, resourceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.compliance, resourceID)
	return nil
}

// ApplyPolicy applies a policy to a resource
func (m *ComplianceManager) ApplyPolicy(ctx context.Context, resourceID string, policy *models.ResourcePolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	policy.ResourceID = resourceID
	policy.AppliedAt = time.Now()
	policy.LastChecked = time.Now()
	policy.NextCheck = time.Now().Add(24 * time.Hour)

	m.policies[resourceID] = policy
	return nil
}

// GetPolicy retrieves a policy for a resource
func (m *ComplianceManager) GetPolicy(ctx context.Context, resourceID string) (*models.ResourcePolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy := m.policies[resourceID]
	if policy == nil {
		return nil, fmt.Errorf("policy for resource %s not found", resourceID)
	}

	return policy, nil
}

// GetStatistics returns compliance statistics
func (m *ComplianceManager) GetStatistics(ctx context.Context) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"total_resources":         len(m.compliance),
		"compliant_resources":     0,
		"non_compliant_resources": 0,
		"warning_resources":       0,
		"unknown_resources":       0,
		"total_violations":        0,
		"critical_violations":     0,
		"high_violations":         0,
		"medium_violations":       0,
		"low_violations":          0,
		"total_policies":          len(m.policies),
	}

	// Count compliance levels
	for _, status := range m.compliance {
		switch status.Status {
		case models.ComplianceLevelCompliant:
			stats["compliant_resources"] = stats["compliant_resources"].(int) + 1
		case models.ComplianceLevelNonCompliant:
			stats["non_compliant_resources"] = stats["non_compliant_resources"].(int) + 1
		case models.ComplianceLevelWarning:
			stats["warning_resources"] = stats["warning_resources"].(int) + 1
		case models.ComplianceLevelUnknown:
			stats["unknown_resources"] = stats["unknown_resources"].(int) + 1
		}

		// Count violations
		stats["total_violations"] = stats["total_violations"].(int) + len(status.Violations)
		for _, violation := range status.Violations {
			switch violation.Severity {
			case "critical":
				stats["critical_violations"] = stats["critical_violations"].(int) + 1
			case "high":
				stats["high_violations"] = stats["high_violations"].(int) + 1
			case "medium":
				stats["medium_violations"] = stats["medium_violations"].(int) + 1
			case "low":
				stats["low_violations"] = stats["low_violations"].(int) + 1
			}
		}
	}

	return stats
}

// Health checks the health of the compliance manager
func (m *ComplianceManager) Health(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Basic health check
	if len(m.compliance) < 0 {
		return fmt.Errorf("compliance manager has negative count")
	}

	return nil
}

// performComplianceChecks performs compliance checks on a resource
func (m *ComplianceManager) performComplianceChecks(resource *models.CloudResource) []models.Violation {
	var violations []models.Violation

	// Check for required tags
	if len(resource.Tags) == 0 {
		violations = append(violations, models.Violation{
			ID:          fmt.Sprintf("missing-tags-%s", resource.ID),
			RuleID:      "required-tags",
			RuleName:    "Required Tags",
			Severity:    "medium",
			Description: "Resource must have at least one tag",
			Remediation: "Add appropriate tags to the resource",
			DetectedAt:  time.Now(),
		})
	}

	// Check for encryption
	if resource.Type == "aws_s3_bucket" {
		if config := resource.Configuration; config != nil {
			if encrypted, ok := config["encrypted"].(bool); !ok || !encrypted {
				violations = append(violations, models.Violation{
					ID:          fmt.Sprintf("encryption-%s", resource.ID),
					RuleID:      "s3-encryption",
					RuleName:    "S3 Bucket Encryption",
					Severity:    "high",
					Description: "S3 bucket must be encrypted",
					Remediation: "Enable encryption on the S3 bucket",
					DetectedAt:  time.Now(),
				})
			}
		}
	}

	// Check for public access
	if resource.Type == "aws_s3_bucket" {
		if config := resource.Configuration; config != nil {
			if publicAccess, ok := config["public_access"].(bool); ok && publicAccess {
				violations = append(violations, models.Violation{
					ID:          fmt.Sprintf("public-access-%s", resource.ID),
					RuleID:      "s3-public-access",
					RuleName:    "S3 Bucket Public Access",
					Severity:    "critical",
					Description: "S3 bucket should not have public access",
					Remediation: "Remove public access from the S3 bucket",
					DetectedAt:  time.Now(),
				})
			}
		}
	}

	// Check for backup
	if resource.Type == "aws_rds_instance" {
		if config := resource.Configuration; config != nil {
			if backupEnabled, ok := config["backup_enabled"].(bool); !ok || !backupEnabled {
				violations = append(violations, models.Violation{
					ID:          fmt.Sprintf("backup-%s", resource.ID),
					RuleID:      "rds-backup",
					RuleName:    "RDS Backup",
					Severity:    "high",
					Description: "RDS instance must have backup enabled",
					Remediation: "Enable automated backups on the RDS instance",
					DetectedAt:  time.Now(),
				})
			}
		}
	}

	return violations
}
