package checkers

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// DigitalOceanChecker performs health checks for DigitalOcean resources
type DigitalOceanChecker struct {
	checkType string
}

// NewDigitalOceanChecker creates a new DigitalOcean health checker
func NewDigitalOceanChecker() *DigitalOceanChecker {
	return &DigitalOceanChecker{
		checkType: "digitalocean",
	}
}

// Check performs health checks on DigitalOcean resources
func (dc *DigitalOceanChecker) Check(ctx context.Context, resource *models.Resource) (*HealthCheck, error) {
	check := &HealthCheck{
		ID:          fmt.Sprintf("digitalocean-%s", resource.ID),
		Name:        "DigitalOcean Resource Health",
		Type:        dc.checkType,
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
	case dc.isDroplet(resource):
		return dc.checkDroplet(ctx, resource, check)
	case dc.isVolume(resource):
		return dc.checkVolume(ctx, resource, check)
	case dc.isLoadBalancer(resource):
		return dc.checkLoadBalancer(ctx, resource, check)
	case dc.isDatabase(resource):
		return dc.checkDatabase(ctx, resource, check)
	default:
		return dc.checkGenericDigitalOceanResource(ctx, resource, check)
	}
}

// GetType returns the checker type
func (dc *DigitalOceanChecker) GetType() string {
	return dc.checkType
}

// GetDescription returns the checker description
func (dc *DigitalOceanChecker) GetDescription() string {
	return "DigitalOcean resource health checker"
}

// isDroplet checks if the resource is a droplet
func (dc *DigitalOceanChecker) isDroplet(resource *models.Resource) bool {
	return resource.Type == "digitalocean_droplet"
}

// isVolume checks if the resource is a volume
func (dc *DigitalOceanChecker) isVolume(resource *models.Resource) bool {
	return resource.Type == "digitalocean_volume"
}

// isLoadBalancer checks if the resource is a load balancer
func (dc *DigitalOceanChecker) isLoadBalancer(resource *models.Resource) bool {
	return resource.Type == "digitalocean_loadbalancer"
}

// isDatabase checks if the resource is a database
func (dc *DigitalOceanChecker) isDatabase(resource *models.Resource) bool {
	return resource.Type == "digitalocean_database_cluster"
}

// checkDroplet performs health checks on droplets
func (dc *DigitalOceanChecker) checkDroplet(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	// Check droplet status
	if status, ok := resource.Attributes["status"].(string); ok {
		switch status {
		case "active":
			check.Status = HealthStatusHealthy
			check.Message = "Droplet is running normally"
		case "new":
			check.Status = HealthStatusWarning
			check.Message = "Droplet is being created"
		case "off":
			check.Status = HealthStatusWarning
			check.Message = "Droplet is powered off"
		case "archive":
			check.Status = HealthStatusCritical
			check.Message = "Droplet is archived"
		default:
			check.Status = HealthStatusUnknown
			check.Message = fmt.Sprintf("Unknown droplet status: %s", status)
		}
	} else {
		check.Status = HealthStatusUnknown
		check.Message = "Droplet status not available"
	}

	// Check droplet size
	if size, ok := resource.Attributes["size_slug"].(string); ok {
		check.Metadata["size"] = size
	}

	// Check memory
	if memory, ok := resource.Attributes["memory"].(int); ok {
		check.Metadata["memory_mb"] = memory
	}

	// Check VCPUs
	if vcpus, ok := resource.Attributes["vcpus"].(int); ok {
		check.Metadata["vcpus"] = vcpus
	}

	// Check disk
	if disk, ok := resource.Attributes["disk"].(int); ok {
		check.Metadata["disk_gb"] = disk
	}

	// Check image
	if image, ok := resource.Attributes["image"].(map[string]interface{}); ok {
		if imageName, ok := image["name"].(string); ok {
			check.Metadata["image"] = imageName
		}
	}

	// Check region
	if region, ok := resource.Attributes["region"].(map[string]interface{}); ok {
		if regionSlug, ok := region["slug"].(string); ok {
			check.Metadata["region"] = regionSlug
		}
	}

	// Check tags
	if tags, ok := resource.Attributes["tags"].([]interface{}); ok {
		check.Metadata["tags"] = len(tags)
		if len(tags) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No tags applied"
		}
	}

	// Check monitoring
	if monitoring, ok := resource.Attributes["monitoring"].(bool); ok {
		check.Metadata["monitoring"] = monitoring
		if !monitoring {
			check.Status = HealthStatusWarning
			check.Message += " - Monitoring not enabled"
		}
	}

	return check, nil
}

// checkVolume performs health checks on volumes
func (dc *DigitalOceanChecker) checkVolume(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Volume is accessible"

	// Check volume size
	if size, ok := resource.Attributes["size_gigabytes"].(int); ok {
		check.Metadata["size_gb"] = size
	}

	// Check filesystem type
	if filesystemType, ok := resource.Attributes["filesystem_type"].(string); ok {
		check.Metadata["filesystem_type"] = filesystemType
	}

	// Check filesystem label
	if filesystemLabel, ok := resource.Attributes["filesystem_label"].(string); ok {
		check.Metadata["filesystem_label"] = filesystemLabel
	}

	// Check droplet attachments
	if dropletIDs, ok := resource.Attributes["droplet_ids"].([]interface{}); ok {
		check.Metadata["attached_droplets"] = len(dropletIDs)
		if len(dropletIDs) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - Volume not attached to any droplets"
		}
	}

	// Check region
	if region, ok := resource.Attributes["region"].(map[string]interface{}); ok {
		if regionSlug, ok := region["slug"].(string); ok {
			check.Metadata["region"] = regionSlug
		}
	}

	// Check tags
	if tags, ok := resource.Attributes["tags"].([]interface{}); ok {
		check.Metadata["tags"] = len(tags)
		if len(tags) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No tags applied"
		}
	}

	return check, nil
}

// checkLoadBalancer performs health checks on load balancers
func (dc *DigitalOceanChecker) checkLoadBalancer(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Load balancer is configured"

	// Check load balancer status
	if status, ok := resource.Attributes["status"].(string); ok {
		check.Metadata["status"] = status
		if status != "active" {
			check.Status = HealthStatusWarning
			check.Message += fmt.Sprintf(" - Load balancer status: %s", status)
		}
	}

	// Check algorithm
	if algorithm, ok := resource.Attributes["algorithm"].(string); ok {
		check.Metadata["algorithm"] = algorithm
	}

	// Check forwarding rules
	if forwardingRules, ok := resource.Attributes["forwarding_rules"].([]interface{}); ok {
		check.Metadata["forwarding_rules"] = len(forwardingRules)
		if len(forwardingRules) == 0 {
			check.Status = HealthStatusCritical
			check.Message += " - No forwarding rules configured"
		}
	}

	// Check health check
	if healthCheck, ok := resource.Attributes["health_check"].(map[string]interface{}); ok {
		if len(healthCheck) > 0 {
			check.Metadata["health_check_configured"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - No health check configured"
		}
	}

	// Check sticky sessions
	if stickySessions, ok := resource.Attributes["sticky_sessions"].(map[string]interface{}); ok {
		if len(stickySessions) > 0 {
			check.Metadata["sticky_sessions"] = true
		}
	}

	// Check droplet attachments
	if dropletIDs, ok := resource.Attributes["droplet_ids"].([]interface{}); ok {
		check.Metadata["attached_droplets"] = len(dropletIDs)
		if len(dropletIDs) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No droplets attached"
		}
	}

	// Check region
	if region, ok := resource.Attributes["region"].(map[string]interface{}); ok {
		if regionSlug, ok := region["slug"].(string); ok {
			check.Metadata["region"] = regionSlug
		}
	}

	return check, nil
}

// checkDatabase performs health checks on databases
func (dc *DigitalOceanChecker) checkDatabase(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Database cluster is configured"

	// Check database status
	if status, ok := resource.Attributes["status"].(string); ok {
		check.Metadata["status"] = status
		if status != "online" {
			check.Status = HealthStatusWarning
			check.Message += fmt.Sprintf(" - Database status: %s", status)
		}
	}

	// Check engine
	if engine, ok := resource.Attributes["engine"].(string); ok {
		check.Metadata["engine"] = engine
	}

	// Check version
	if version, ok := resource.Attributes["version"].(string); ok {
		check.Metadata["version"] = version
	}

	// Check number of nodes
	if numNodes, ok := resource.Attributes["num_nodes"].(int); ok {
		check.Metadata["num_nodes"] = numNodes
		if numNodes < 2 {
			check.Status = HealthStatusWarning
			check.Message += " - Single node cluster (no high availability)"
		}
	}

	// Check size
	if size, ok := resource.Attributes["size"].(string); ok {
		check.Metadata["size"] = size
	}

	// Check region
	if region, ok := resource.Attributes["region"].(string); ok {
		check.Metadata["region"] = region
	}

	// Check maintenance window
	if maintenanceWindow, ok := resource.Attributes["maintenance_window"].(map[string]interface{}); ok {
		if len(maintenanceWindow) > 0 {
			check.Metadata["maintenance_window_configured"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - No maintenance window configured"
		}
	}

	// Check tags
	if tags, ok := resource.Attributes["tags"].([]interface{}); ok {
		check.Metadata["tags"] = len(tags)
		if len(tags) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No tags applied"
		}
	}

	// Check private network
	if privateNetworkUUID, ok := resource.Attributes["private_network_uuid"].(string); ok {
		if privateNetworkUUID != "" {
			check.Metadata["private_network"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - Not in private network"
		}
	}

	return check, nil
}

// checkGenericDigitalOceanResource performs generic health checks for DigitalOcean resources
func (dc *DigitalOceanChecker) checkGenericDigitalOceanResource(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "DigitalOcean resource is configured"

	// Check if resource has tags
	if tags, ok := resource.Attributes["tags"].([]interface{}); ok {
		check.Metadata["tags"] = len(tags)
		if len(tags) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No tags applied"
		}
	}

	// Check region
	if region, ok := resource.Attributes["region"].(interface{}); ok {
		if regionMap, ok := region.(map[string]interface{}); ok {
			if regionSlug, ok := regionMap["slug"].(string); ok {
				check.Metadata["region"] = regionSlug
			}
		} else if regionStr, ok := region.(string); ok {
			check.Metadata["region"] = regionStr
		}
	}

	return check, nil
}
