package checkers

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// GCPChecker performs health checks for GCP resources
type GCPChecker struct {
	checkType string
}

// NewGCPChecker creates a new GCP health checker
func NewGCPChecker() *GCPChecker {
	return &GCPChecker{
		checkType: "gcp",
	}
}

// Check performs health checks on GCP resources
func (gc *GCPChecker) Check(ctx context.Context, resource *models.Resource) (*HealthCheck, error) {
	check := &HealthCheck{
		ID:          fmt.Sprintf("gcp-%s", resource.ID),
		Name:        "GCP Resource Health",
		Type:        gc.checkType,
		ResourceID:  resource.ID,
		LastChecked: time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	start := time.Now()
	defer func() {
		check.Duration = time.Since(start)
	}()

	// Perform resource-specific health checks
	switch {
	case gc.isComputeInstance(resource):
		return gc.checkComputeInstance(ctx, resource, check)
	case gc.isStorageBucket(resource):
		return gc.checkStorageBucket(ctx, resource, check)
	case gc.isSQLInstance(resource):
		return gc.checkSQLInstance(ctx, resource, check)
	case gc.isCloudFunction(resource):
		return gc.checkCloudFunction(ctx, resource, check)
	case gc.isLoadBalancer(resource):
		return gc.checkLoadBalancer(ctx, resource, check)
	default:
		return gc.checkGenericGCPResource(ctx, resource, check)
	}
}

// GetType returns the checker type
func (gc *GCPChecker) GetType() string {
	return gc.checkType
}

// GetDescription returns the checker description
func (gc *GCPChecker) GetDescription() string {
	return "GCP resource health checker"
}

// isComputeInstance checks if the resource is a compute instance
func (gc *GCPChecker) isComputeInstance(resource *models.Resource) bool {
	return resource.Type == "google_compute_instance"
}

// isStorageBucket checks if the resource is a storage bucket
func (gc *GCPChecker) isStorageBucket(resource *models.Resource) bool {
	return resource.Type == "google_storage_bucket"
}

// isSQLInstance checks if the resource is a SQL instance
func (gc *GCPChecker) isSQLInstance(resource *models.Resource) bool {
	return resource.Type == "google_sql_database_instance"
}

// isCloudFunction checks if the resource is a cloud function
func (gc *GCPChecker) isCloudFunction(resource *models.Resource) bool {
	return resource.Type == "google_cloudfunctions_function"
}

// isLoadBalancer checks if the resource is a load balancer
func (gc *GCPChecker) isLoadBalancer(resource *models.Resource) bool {
	return resource.Type == "google_compute_backend_service" || resource.Type == "google_compute_url_map"
}

// checkComputeInstance performs health checks on compute instances
func (gc *GCPChecker) checkComputeInstance(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	// Check instance status
	if status, ok := resource.Attributes["status"].(string); ok {
		switch status {
		case "RUNNING":
			check.Status = HealthStatusHealthy
			check.Message = "Compute instance is running normally"
		case "STOPPED":
			check.Status = HealthStatusWarning
			check.Message = "Compute instance is stopped"
		case "STOPPING":
			check.Status = HealthStatusWarning
			check.Message = "Compute instance is stopping"
		case "STARTING":
			check.Status = HealthStatusWarning
			check.Message = "Compute instance is starting"
		case "TERMINATED":
			check.Status = HealthStatusCritical
			check.Message = "Compute instance is terminated"
		default:
			check.Status = HealthStatusUnknown
			check.Message = fmt.Sprintf("Unknown instance status: %s", status)
		}
	} else {
		check.Status = HealthStatusUnknown
		check.Message = "Instance status not available"
	}

	// Check machine type
	if machineType, ok := resource.Attributes["machine_type"].(string); ok {
		check.Metadata["machine_type"] = machineType
	}

	// Check boot disk
	if bootDisk, ok := resource.Attributes["boot_disk"].([]interface{}); ok {
		if len(bootDisk) > 0 {
			check.Metadata["boot_disk_configured"] = true
		}
	}

	// Check network interfaces
	if networkInterfaces, ok := resource.Attributes["network_interface"].([]interface{}); ok {
		check.Metadata["network_interfaces"] = len(networkInterfaces)
		if len(networkInterfaces) == 0 {
			check.Status = HealthStatusCritical
			check.Message += " - No network interfaces configured"
		}
	}

	// Check service account
	if serviceAccount, ok := resource.Attributes["service_account"].([]interface{}); ok {
		if len(serviceAccount) > 0 {
			check.Metadata["service_account"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - No service account configured"
		}
	}

	// Check labels
	if labels, ok := resource.Attributes["labels"].(map[string]interface{}); ok {
		check.Metadata["labels"] = len(labels)
		if len(labels) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No labels applied"
		}
	}

	return check, nil
}

// checkStorageBucket performs health checks on storage buckets
func (gc *GCPChecker) checkStorageBucket(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Storage bucket is accessible"

	// Check versioning
	if versioning, ok := resource.Attributes["versioning"].([]interface{}); ok {
		if len(versioning) > 0 {
			check.Metadata["versioning_enabled"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - Versioning not enabled"
		}
	}

	// Check lifecycle rules
	if lifecycle, ok := resource.Attributes["lifecycle_rule"].([]interface{}); ok {
		check.Metadata["lifecycle_rules"] = len(lifecycle)
		if len(lifecycle) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No lifecycle rules configured"
		}
	}

	// Check encryption
	if encryption, ok := resource.Attributes["encryption"].([]interface{}); ok {
		if len(encryption) > 0 {
			check.Metadata["encryption_enabled"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - No encryption configured"
		}
	}

	// Check uniform bucket level access
	if uniformBucketLevelAccess, ok := resource.Attributes["uniform_bucket_level_access"].(bool); ok {
		check.Metadata["uniform_bucket_level_access"] = uniformBucketLevelAccess
		if !uniformBucketLevelAccess {
			check.Status = HealthStatusWarning
			check.Message += " - Uniform bucket level access not enabled"
		}
	}

	// Check public access prevention
	if publicAccessPrevention, ok := resource.Attributes["public_access_prevention"].(string); ok {
		check.Metadata["public_access_prevention"] = publicAccessPrevention
		if publicAccessPrevention != "enforced" {
			check.Status = HealthStatusCritical
			check.Message += " - Public access prevention not enforced"
		}
	}

	return check, nil
}

// checkSQLInstance performs health checks on SQL instances
func (gc *GCPChecker) checkSQLInstance(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "SQL instance is configured"

	// Check instance state
	if state, ok := resource.Attributes["state"].(string); ok {
		check.Metadata["state"] = state
		if state != "RUNNABLE" {
			check.Status = HealthStatusWarning
			check.Message += fmt.Sprintf(" - Instance state: %s", state)
		}
	}

	// Check database version
	if databaseVersion, ok := resource.Attributes["database_version"].(string); ok {
		check.Metadata["database_version"] = databaseVersion
	}

	// Check machine type
	if settings, ok := resource.Attributes["settings"].([]interface{}); ok {
		if len(settings) > 0 {
			check.Metadata["settings_configured"] = true
		}
	}

	// Check backup configuration
	if backupConfiguration, ok := resource.Attributes["settings"].([]interface{}); ok {
		if len(backupConfiguration) > 0 {
			check.Metadata["backup_configured"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - No backup configuration"
		}
	}

	// Check IP configuration
	if ipConfiguration, ok := resource.Attributes["ip_address"].([]interface{}); ok {
		if len(ipConfiguration) > 0 {
			check.Metadata["ip_configuration"] = true
		}
	}

	return check, nil
}

// checkCloudFunction performs health checks on cloud functions
func (gc *GCPChecker) checkCloudFunction(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Cloud function is configured"

	// Check runtime
	if runtime, ok := resource.Attributes["runtime"].(string); ok {
		check.Metadata["runtime"] = runtime
	}

	// Check entry point
	if entryPoint, ok := resource.Attributes["entry_point"].(string); ok {
		check.Metadata["entry_point"] = entryPoint
	}

	// Check timeout
	if timeout, ok := resource.Attributes["timeout"].(int); ok {
		check.Metadata["timeout_seconds"] = timeout
		if timeout > 540 { // 9 minutes
			check.Status = HealthStatusWarning
			check.Message += " - Timeout is very high"
		}
	}

	// Check memory
	if availableMemory, ok := resource.Attributes["available_memory_mb"].(int); ok {
		check.Metadata["memory_mb"] = availableMemory
		if availableMemory < 128 {
			check.Status = HealthStatusWarning
			check.Message += " - Memory allocation is very low"
		}
	}

	// Check VPC connector
	if vpcConnector, ok := resource.Attributes["vpc_connector"].(string); ok {
		if vpcConnector != "" {
			check.Metadata["vpc_connector"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - No VPC connector configured"
		}
	}

	return check, nil
}

// checkLoadBalancer performs health checks on load balancers
func (gc *GCPChecker) checkLoadBalancer(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Load balancer is configured"

	// Check protocol
	if protocol, ok := resource.Attributes["protocol"].(string); ok {
		check.Metadata["protocol"] = protocol
	}

	// Check port
	if port, ok := resource.Attributes["port"].(int); ok {
		check.Metadata["port"] = port
	}

	// Check health checks
	if healthChecks, ok := resource.Attributes["health_checks"].([]interface{}); ok {
		check.Metadata["health_checks"] = len(healthChecks)
		if len(healthChecks) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No health checks configured"
		}
	}

	// Check backend services
	if backendServices, ok := resource.Attributes["backend"].([]interface{}); ok {
		check.Metadata["backend_services"] = len(backendServices)
		if len(backendServices) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No backend services configured"
		}
	}

	return check, nil
}

// checkGenericGCPResource performs generic health checks for GCP resources
func (gc *GCPChecker) checkGenericGCPResource(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "GCP resource is configured"

	// Check if resource has labels
	if labels, ok := resource.Attributes["labels"].(map[string]interface{}); ok {
		check.Metadata["labels"] = len(labels)
		if len(labels) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No labels applied"
		}
	}

	// Check project
	if project, ok := resource.Attributes["project"].(string); ok {
		check.Metadata["project"] = project
	}

	// Check region/zone
	if region, ok := resource.Attributes["region"].(string); ok {
		check.Metadata["region"] = region
	}
	if zone, ok := resource.Attributes["zone"].(string); ok {
		check.Metadata["zone"] = zone
	}

	return check, nil
}
