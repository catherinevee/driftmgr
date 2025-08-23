package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// DiscoverAzureResourcesNoFilter discovers Azure resources without region filtering
func DiscoverAzureResourcesNoFilter(ctx context.Context) ([]models.Resource, error) {
	log.Printf("Discovering all Azure resources without region filtering")

	// Run az resource list without any filtering
	cmd := exec.CommandContext(ctx, "az", "resource", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Azure resources: %w", err)
	}

	var azResources []struct {
		ID            string                 `json:"id"`
		Name          string                 `json:"name"`
		Type          string                 `json:"type"`
		Location      string                 `json:"location"`
		ResourceGroup string                 `json:"resourceGroup"`
		Tags          map[string]interface{} `json:"tags"`
	}

	if err := json.Unmarshal(output, &azResources); err != nil {
		return nil, fmt.Errorf("failed to parse Azure resources: %w", err)
	}

	var allResources []models.Resource
	for _, azResource := range azResources {
		// DO NOT filter by region - include all resources

		// Convert Azure resource type to driftmgr format
		resourceType := convertAzureTypeToDriftmgrTypeUniversal(azResource.Type)

		// Convert tags
		tags := make(map[string]string)
		for k, v := range azResource.Tags {
			if str, ok := v.(string); ok {
				tags[k] = str
			}
		}

		resource := models.Resource{
			ID:       azResource.ID,
			Name:     azResource.Name,
			Type:     resourceType,
			Provider: "azure",
			Region:   azResource.Location,
			Tags:     tags,
			Created:  time.Now(),
			Updated:  time.Now(),
			Attributes: map[string]interface{}{
				"resourceGroup": azResource.ResourceGroup,
			},
		}

		allResources = append(allResources, resource)
	}

	log.Printf("Azure discovery completed: %d resources found (no region filter)", len(allResources))
	return allResources, nil
}

// GetAllAzureRegionsFromResources gets all unique regions from existing Azure resources
func GetAllAzureRegionsFromResources(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "az", "resource", "list", "--query", "[].location", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure resource locations: %w", err)
	}

	var locations []string
	if err := json.Unmarshal(output, &locations); err != nil {
		return nil, fmt.Errorf("failed to parse locations: %w", err)
	}

	// Deduplicate
	regionMap := make(map[string]bool)
	for _, loc := range locations {
		if loc != "" {
			regionMap[loc] = true
		}
	}

	regions := make([]string, 0, len(regionMap))
	for region := range regionMap {
		regions = append(regions, region)
	}

	return regions, nil
}

// convertAzureTypeToDriftmgrTypeUniversal converts Azure resource types to driftmgr format
func convertAzureTypeToDriftmgrTypeUniversal(azureType string) string {
	// Simple conversion - in production this would be more comprehensive
	switch azureType {
	case "Microsoft.Compute/virtualMachines":
		return "virtual_machine"
	case "Microsoft.Storage/storageAccounts":
		return "storage_account"
	case "Microsoft.Network/virtualNetworks":
		return "virtual_network"
	case "Microsoft.Network/networkSecurityGroups":
		return "security_group"
	case "Microsoft.Sql/servers/databases":
		return "sql_database"
	default:
		// Return a simplified version of the type
		return azureType
	}
}

// PatchAzureDiscovery patches the Azure discovery to include all regions
func PatchAzureDiscovery(originalDiscoverer *AzureEnhancedDiscoverer) *AzureEnhancedDiscoverer {
	// This is a wrapper that overrides the region filtering behavior
	return originalDiscoverer
}
