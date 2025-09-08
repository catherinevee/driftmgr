package checkers

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// GenericChecker performs generic health checks for any resource
type GenericChecker struct {
	checkType string
}

// NewGenericChecker creates a new generic health checker
func NewGenericChecker() *GenericChecker {
	return &GenericChecker{
		checkType: "generic",
	}
}

// Check performs generic health checks on any resource
func (gc *GenericChecker) Check(ctx context.Context, resource *models.Resource) (*HealthCheck, error) {
	check := &HealthCheck{
		ID:          fmt.Sprintf("generic-%s", resource.ID),
		Name:        "Generic Resource Health",
		Type:        gc.checkType,
		ResourceID:  resource.ID,
		LastChecked: time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	start := time.Now()
	defer func() {
		check.Duration = time.Since(start)
	}()

	// Start with healthy status
	check.Status = HealthStatusHealthy
	check.Message = "Resource is configured"

	// Check basic resource properties
	gc.checkBasicProperties(resource, check)

	// Check for common issues
	gc.checkCommonIssues(resource, check)

	// Check resource-specific attributes
	gc.checkResourceSpecificAttributes(resource, check)

	return check, nil
}

// GetType returns the checker type
func (gc *GenericChecker) GetType() string {
	return gc.checkType
}

// GetDescription returns the checker description
func (gc *GenericChecker) GetDescription() string {
	return "Generic resource health checker"
}

// checkBasicProperties checks basic resource properties
func (gc *GenericChecker) checkBasicProperties(resource *models.Resource, check *HealthCheck) {
	// Check if resource has an ID
	if resource.ID == "" {
		check.Status = HealthStatusCritical
		check.Message = "Resource has no ID"
		return
	}

	// Check if resource has a type
	if resource.Type == "" {
		check.Status = HealthStatusCritical
		check.Message = "Resource has no type"
		return
	}

	// Check if resource has a provider
	if resource.Provider == "" {
		check.Status = HealthStatusWarning
		check.Message += " - No provider specified"
	}

	// Check if resource has a region
	if resource.Region == "" {
		check.Status = HealthStatusWarning
		check.Message += " - No region specified"
	}

	// Store basic metadata
	check.Metadata["resource_type"] = resource.Type
	check.Metadata["provider"] = resource.Provider
	check.Metadata["region"] = resource.Region
}

// checkCommonIssues checks for common resource issues
func (gc *GenericChecker) checkCommonIssues(resource *models.Resource, check *HealthCheck) {
	// Check for tags/labels
	gc.checkTags(resource, check)

	// Check for naming conventions
	gc.checkNamingConventions(resource, check)

	// Check for security configurations
	gc.checkSecurityConfigurations(resource, check)

	// Check for monitoring configurations
	gc.checkMonitoringConfigurations(resource, check)
}

// checkTags checks for tag/label configurations
func (gc *GenericChecker) checkTags(resource *models.Resource, check *HealthCheck) {
	hasTags := false
	tagCount := 0

	// Check for various tag/label attributes
	tagAttributes := []string{"tags", "labels", "annotations", "metadata"}

	for _, attr := range tagAttributes {
		if tags, ok := resource.Attributes[attr].(map[string]interface{}); ok {
			hasTags = true
			tagCount += len(tags)
		} else if tags, ok := resource.Attributes[attr].([]interface{}); ok {
			hasTags = true
			tagCount += len(tags)
		}
	}

	check.Metadata["has_tags"] = hasTags
	check.Metadata["tag_count"] = tagCount

	if !hasTags {
		check.Status = HealthStatusWarning
		check.Message += " - No tags/labels found"
	}
}

// checkNamingConventions checks for naming convention compliance
func (gc *GenericChecker) checkNamingConventions(resource *models.Resource, check *HealthCheck) {
	// Check for name attribute
	if name, ok := resource.Attributes["name"].(string); ok {
		check.Metadata["name"] = name

		// Basic naming convention checks
		if len(name) < 3 {
			check.Status = HealthStatusWarning
			check.Message += " - Resource name is too short"
		}

		if len(name) > 63 {
			check.Status = HealthStatusWarning
			check.Message += " - Resource name is too long"
		}
	} else {
		check.Status = HealthStatusWarning
		check.Message += " - No name attribute found"
	}
}

// checkSecurityConfigurations checks for security-related configurations
func (gc *GenericChecker) checkSecurityConfigurations(resource *models.Resource, check *HealthCheck) {
	// Check for encryption configurations
	encryptionAttributes := []string{
		"encryption", "encrypted", "storage_encrypted", "transparent_data_encryption_enabled",
		"enable_encryption", "encryption_at_rest", "encryption_in_transit",
	}

	hasEncryption := false
	for _, attr := range encryptionAttributes {
		if value, ok := resource.Attributes[attr]; ok {
			if boolVal, ok := value.(bool); ok && boolVal {
				hasEncryption = true
				break
			} else if strVal, ok := value.(string); ok && strVal != "" {
				hasEncryption = true
				break
			}
		}
	}

	check.Metadata["has_encryption"] = hasEncryption
	if !hasEncryption {
		check.Status = HealthStatusWarning
		check.Message += " - No encryption configuration found"
	}

	// Check for public access configurations
	publicAccessAttributes := []string{
		"public", "publicly_accessible", "public_access_block", "public_access_prevention",
		"allow_public_access", "is_public",
	}

	hasPublicAccessControl := false
	for _, attr := range publicAccessAttributes {
		if _, ok := resource.Attributes[attr]; ok {
			hasPublicAccessControl = true
			break
		}
	}

	check.Metadata["has_public_access_control"] = hasPublicAccessControl
	if !hasPublicAccessControl {
		check.Status = HealthStatusWarning
		check.Message += " - No public access control found"
	}
}

// checkMonitoringConfigurations checks for monitoring-related configurations
func (gc *GenericChecker) checkMonitoringConfigurations(resource *models.Resource, check *HealthCheck) {
	// Check for monitoring configurations
	monitoringAttributes := []string{
		"monitoring", "enable_monitoring", "detailed_monitoring", "cloudwatch_enabled",
		"logging", "enable_logging", "audit_logging",
	}

	hasMonitoring := false
	for _, attr := range monitoringAttributes {
		if value, ok := resource.Attributes[attr]; ok {
			if boolVal, ok := value.(bool); ok && boolVal {
				hasMonitoring = true
				break
			}
		}
	}

	check.Metadata["has_monitoring"] = hasMonitoring
	if !hasMonitoring {
		check.Status = HealthStatusWarning
		check.Message += " - No monitoring configuration found"
	}
}

// checkResourceSpecificAttributes checks for resource-specific attributes
func (gc *GenericChecker) checkResourceSpecificAttributes(resource *models.Resource, check *HealthCheck) {
	// Check for status-related attributes
	statusAttributes := []string{"status", "state", "provisioning_state", "lifecycle_state"}

	for _, attr := range statusAttributes {
		if status, ok := resource.Attributes[attr].(string); ok {
			check.Metadata["status"] = status

			// Check for problematic statuses
			problematicStatuses := []string{
				"failed", "error", "deleting", "terminated", "stopped", "inactive",
				"deprecated", "disabled", "suspended",
			}

			for _, problematic := range problematicStatuses {
				if status == problematic {
					check.Status = HealthStatusCritical
					check.Message += fmt.Sprintf(" - Status is %s", status)
					return
				}
			}

			// Check for warning statuses
			warningStatuses := []string{
				"creating", "updating", "pending", "stopping", "starting",
				"maintenance", "degraded",
			}

			for _, warning := range warningStatuses {
				if status == warning {
					check.Status = HealthStatusWarning
					check.Message += fmt.Sprintf(" - Status is %s", status)
					return
				}
			}
			break
		}
	}

	// Check for backup configurations
	backupAttributes := []string{
		"backup", "backup_enabled", "backup_retention", "backup_retention_period",
		"backup_configuration", "snapshot_enabled",
	}

	hasBackup := false
	for _, attr := range backupAttributes {
		if value, ok := resource.Attributes[attr]; ok {
			if boolVal, ok := value.(bool); ok && boolVal {
				hasBackup = true
				break
			} else if intVal, ok := value.(int); ok && intVal > 0 {
				hasBackup = true
				break
			}
		}
	}

	check.Metadata["has_backup"] = hasBackup
	if !hasBackup {
		check.Status = HealthStatusWarning
		check.Message += " - No backup configuration found"
	}

	// Check for high availability configurations
	haAttributes := []string{
		"multi_az", "high_availability", "replication", "redundancy",
		"availability_zone", "fault_tolerance",
	}

	hasHA := false
	for _, attr := range haAttributes {
		if value, ok := resource.Attributes[attr]; ok {
			if boolVal, ok := value.(bool); ok && boolVal {
				hasHA = true
				break
			} else if strVal, ok := value.(string); ok && strVal != "" {
				hasHA = true
				break
			}
		}
	}

	check.Metadata["has_high_availability"] = hasHA
	if !hasHA {
		check.Status = HealthStatusWarning
		check.Message += " - No high availability configuration found"
	}
}
