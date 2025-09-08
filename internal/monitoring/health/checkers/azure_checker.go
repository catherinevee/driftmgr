package checkers

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// AzureChecker performs health checks for Azure resources
type AzureChecker struct {
	checkType string
}

// NewAzureChecker creates a new Azure health checker
func NewAzureChecker() *AzureChecker {
	return &AzureChecker{
		checkType: "azure",
	}
}

// Check performs health checks on Azure resources
func (ac *AzureChecker) Check(ctx context.Context, resource *models.Resource) (*HealthCheck, error) {
	check := &HealthCheck{
		ID:          fmt.Sprintf("azure-%s", resource.ID),
		Name:        "Azure Resource Health",
		Type:        ac.checkType,
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
	case ac.isVirtualMachine(resource):
		return ac.checkVirtualMachine(ctx, resource, check)
	case ac.isStorageAccount(resource):
		return ac.checkStorageAccount(ctx, resource, check)
	case ac.isSQLDatabase(resource):
		return ac.checkSQLDatabase(ctx, resource, check)
	case ac.isFunctionApp(resource):
		return ac.checkFunctionApp(ctx, resource, check)
	case ac.isLoadBalancer(resource):
		return ac.checkLoadBalancer(ctx, resource, check)
	default:
		return ac.checkGenericAzureResource(ctx, resource, check)
	}
}

// GetType returns the checker type
func (ac *AzureChecker) GetType() string {
	return ac.checkType
}

// GetDescription returns the checker description
func (ac *AzureChecker) GetDescription() string {
	return "Azure resource health checker"
}

// isVirtualMachine checks if the resource is a virtual machine
func (ac *AzureChecker) isVirtualMachine(resource *models.Resource) bool {
	return resource.Type == "azurerm_virtual_machine" || resource.Type == "azurerm_linux_virtual_machine" || resource.Type == "azurerm_windows_virtual_machine"
}

// isStorageAccount checks if the resource is a storage account
func (ac *AzureChecker) isStorageAccount(resource *models.Resource) bool {
	return resource.Type == "azurerm_storage_account"
}

// isSQLDatabase checks if the resource is a SQL database
func (ac *AzureChecker) isSQLDatabase(resource *models.Resource) bool {
	return resource.Type == "azurerm_sql_database" || resource.Type == "azurerm_mssql_database"
}

// isFunctionApp checks if the resource is a function app
func (ac *AzureChecker) isFunctionApp(resource *models.Resource) bool {
	return resource.Type == "azurerm_function_app"
}

// isLoadBalancer checks if the resource is a load balancer
func (ac *AzureChecker) isLoadBalancer(resource *models.Resource) bool {
	return resource.Type == "azurerm_lb" || resource.Type == "azurerm_application_gateway"
}

// checkVirtualMachine performs health checks on virtual machines
func (ac *AzureChecker) checkVirtualMachine(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	// Check VM status
	if status, ok := resource.Attributes["provisioning_state"].(string); ok {
		switch status {
		case "Succeeded":
			check.Status = HealthStatusHealthy
			check.Message = "Virtual machine is running normally"
		case "Creating":
			check.Status = HealthStatusWarning
			check.Message = "Virtual machine is being created"
		case "Deleting":
			check.Status = HealthStatusCritical
			check.Message = "Virtual machine is being deleted"
		case "Failed":
			check.Status = HealthStatusCritical
			check.Message = "Virtual machine creation failed"
		default:
			check.Status = HealthStatusUnknown
			check.Message = fmt.Sprintf("Unknown VM status: %s", status)
		}
	} else {
		check.Status = HealthStatusUnknown
		check.Message = "VM status not available"
	}

	// Check VM size
	if vmSize, ok := resource.Attributes["vm_size"].(string); ok {
		check.Metadata["vm_size"] = vmSize
	}

	// Check OS disk
	if osDisk, ok := resource.Attributes["os_disk"].([]interface{}); ok {
		if len(osDisk) > 0 {
			check.Metadata["os_disk_configured"] = true
		}
	}

	// Check network security groups
	if nsgId, ok := resource.Attributes["network_security_group_id"].(string); ok {
		if nsgId != "" {
			check.Metadata["network_security_group"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - No network security group attached"
		}
	}

	// Check tags
	if tags, ok := resource.Attributes["tags"].(map[string]interface{}); ok {
		check.Metadata["tags"] = len(tags)
		if len(tags) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No tags applied"
		}
	}

	return check, nil
}

// checkStorageAccount performs health checks on storage accounts
func (ac *AzureChecker) checkStorageAccount(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Storage account is accessible"

	// Check account kind
	if accountKind, ok := resource.Attributes["account_kind"].(string); ok {
		check.Metadata["account_kind"] = accountKind
	}

	// Check replication type
	if replication, ok := resource.Attributes["account_replication_type"].(string); ok {
		check.Metadata["replication_type"] = replication
		if replication == "LRS" {
			check.Status = HealthStatusWarning
			check.Message += " - Using local replication (LRS)"
		}
	}

	// Check encryption
	if encryption, ok := resource.Attributes["enable_https_traffic_only"].(bool); ok {
		check.Metadata["https_only"] = encryption
		if !encryption {
			check.Status = HealthStatusCritical
			check.Message += " - HTTPS traffic not enforced"
		}
	}

	// Check blob encryption
	if blobEncryption, ok := resource.Attributes["blob_properties"].([]interface{}); ok {
		if len(blobEncryption) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No blob encryption configured"
		} else {
			check.Metadata["blob_encryption"] = true
		}
	}

	// Check access tier
	if accessTier, ok := resource.Attributes["access_tier"].(string); ok {
		check.Metadata["access_tier"] = accessTier
	}

	return check, nil
}

// checkSQLDatabase performs health checks on SQL databases
func (ac *AzureChecker) checkSQLDatabase(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "SQL database is configured"

	// Check database status
	if status, ok := resource.Attributes["status"].(string); ok {
		check.Metadata["status"] = status
		if status != "Online" {
			check.Status = HealthStatusWarning
			check.Message += fmt.Sprintf(" - Database status: %s", status)
		}
	}

	// Check service tier
	if sku, ok := resource.Attributes["sku_name"].(string); ok {
		check.Metadata["sku"] = sku
	}

	// Check encryption
	if transparentDataEncryption, ok := resource.Attributes["transparent_data_encryption_enabled"].(bool); ok {
		if !transparentDataEncryption {
			check.Status = HealthStatusCritical
			check.Message += " - Transparent data encryption not enabled"
		} else {
			check.Metadata["encryption_enabled"] = true
		}
	}

	// Check backup retention
	if backupRetentionDays, ok := resource.Attributes["backup_retention_days"].(int); ok {
		check.Metadata["backup_retention_days"] = backupRetentionDays
		if backupRetentionDays == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No backup retention configured"
		}
	}

	// Check geo-replication
	if geoReplication, ok := resource.Attributes["geo_replication_enabled"].(bool); ok {
		check.Metadata["geo_replication"] = geoReplication
		if !geoReplication {
			check.Status = HealthStatusWarning
			check.Message += " - Geo-replication not enabled"
		}
	}

	return check, nil
}

// checkFunctionApp performs health checks on function apps
func (ac *AzureChecker) checkFunctionApp(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Function app is configured"

	// Check runtime
	if runtime, ok := resource.Attributes["site_config"].([]interface{}); ok {
		if len(runtime) > 0 {
			check.Metadata["runtime_configured"] = true
		}
	}

	// Check HTTPS only
	if httpsOnly, ok := resource.Attributes["https_only"].(bool); ok {
		check.Metadata["https_only"] = httpsOnly
		if !httpsOnly {
			check.Status = HealthStatusWarning
			check.Message += " - HTTPS not enforced"
		}
	}

	// Check identity
	if identity, ok := resource.Attributes["identity"].([]interface{}); ok {
		if len(identity) > 0 {
			check.Metadata["managed_identity"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - No managed identity configured"
		}
	}

	// Check app settings
	if appSettings, ok := resource.Attributes["app_settings"].(map[string]interface{}); ok {
		check.Metadata["app_settings"] = len(appSettings)
	}

	return check, nil
}

// checkLoadBalancer performs health checks on load balancers
func (ac *AzureChecker) checkLoadBalancer(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Load balancer is configured"

	// Check SKU
	if sku, ok := resource.Attributes["sku"].(string); ok {
		check.Metadata["sku"] = sku
	}

	// Check frontend IP configurations
	if frontendIPConfigs, ok := resource.Attributes["frontend_ip_configuration"].([]interface{}); ok {
		check.Metadata["frontend_ip_configs"] = len(frontendIPConfigs)
		if len(frontendIPConfigs) == 0 {
			check.Status = HealthStatusCritical
			check.Message += " - No frontend IP configurations"
		}
	}

	// Check backend address pools
	if backendPools, ok := resource.Attributes["backend_address_pool"].([]interface{}); ok {
		check.Metadata["backend_pools"] = len(backendPools)
		if len(backendPools) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No backend address pools"
		}
	}

	// Check load balancing rules
	if lbRules, ok := resource.Attributes["loadbalancing_rule"].([]interface{}); ok {
		check.Metadata["load_balancing_rules"] = len(lbRules)
		if len(lbRules) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No load balancing rules"
		}
	}

	return check, nil
}

// checkGenericAzureResource performs generic health checks for Azure resources
func (ac *AzureChecker) checkGenericAzureResource(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Azure resource is configured"

	// Check if resource has tags
	if tags, ok := resource.Attributes["tags"].(map[string]interface{}); ok {
		check.Metadata["tags"] = len(tags)
		if len(tags) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No tags applied"
		}
	}

	// Check resource group
	if resourceGroup, ok := resource.Attributes["resource_group_name"].(string); ok {
		check.Metadata["resource_group"] = resourceGroup
	}

	// Check location
	if location, ok := resource.Attributes["location"].(string); ok {
		check.Metadata["location"] = location
	}

	return check, nil
}
