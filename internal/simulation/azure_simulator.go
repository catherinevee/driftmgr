package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/state"
)

// AzureSimulator simulates drift in Azure resources
type AzureSimulator struct {
	subscriptionID string
	accessToken    string
	httpClient     *http.Client
}

// NewAzureSimulator creates a new Azure drift simulator
func NewAzureSimulator() *AzureSimulator {
	return &AzureSimulator{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Initialize sets up Azure authentication
func (s *AzureSimulator) Initialize(ctx context.Context) error {
	// Get subscription ID from environment or Azure CLI
	s.subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	if s.subscriptionID == "" {
		// Try to get from Azure CLI
		s.subscriptionID = s.getSubscriptionFromCLI()
	}

	// Get access token
	token, err := s.getAccessToken()
	if err != nil {
		return fmt.Errorf("failed to get Azure access token: %w", err)
	}
	s.accessToken = token

	return nil
}

// SimulateDrift creates drift in Azure resources
func (s *AzureSimulator) SimulateDrift(ctx context.Context, driftType DriftType, resourceID string, state *state.TerraformState) (*SimulationResult, error) {
	// Initialize if needed
	if s.accessToken == "" {
		if err := s.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	// Find the resource in state
	resource := s.findResource(resourceID, state)
	if resource == nil {
		return nil, fmt.Errorf("resource %s not found in state", resourceID)
	}

	// Execute drift based on type
	switch driftType {
	case DriftTypeTagChange:
		return s.simulateTagDrift(ctx, resource)
	case DriftTypeRuleAddition:
		return s.simulateNSGRuleDrift(ctx, resource)
	case DriftTypeResourceCreation:
		return s.simulateResourceCreation(ctx, resource, state)
	case DriftTypeAttributeChange:
		return s.simulateAttributeChange(ctx, resource)
	default:
		return nil, fmt.Errorf("drift type %s not implemented for Azure", driftType)
	}
}

// simulateTagDrift adds or modifies tags on an Azure resource
func (s *AzureSimulator) simulateTagDrift(ctx context.Context, resource *state.Resource) (*SimulationResult, error) {
	result := &SimulationResult{
		Provider:     "azure",
		ResourceType: resource.Type,
		ResourceID:   resource.ID,
		DriftType:    DriftTypeTagChange,
		Changes:      make(map[string]interface{}),
		CostEstimate: "$0.00 (tags are free)",
	}

	// Extract resource group and resource name
	resourceGroup := s.extractResourceGroup(resource)
	resourceName := s.extractResourceName(resource)
	if resourceGroup == "" || resourceName == "" {
		return nil, fmt.Errorf("could not extract resource details")
	}

	// Build API URL based on resource type
	var apiURL string
	switch resource.Type {
	case "azurerm_resource_group":
		apiURL = fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s?api-version=2021-04-01",
			s.subscriptionID, resourceName)
	case "azurerm_virtual_network":
		apiURL = fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s?api-version=2023-04-01",
			s.subscriptionID, resourceGroup, resourceName)
	case "azurerm_network_security_group":
		apiURL = fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/%s?api-version=2023-04-01",
			s.subscriptionID, resourceGroup, resourceName)
	case "azurerm_storage_account":
		apiURL = fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s?api-version=2023-01-01",
			s.subscriptionID, resourceGroup, resourceName)
	default:
		return nil, fmt.Errorf("tag drift not implemented for resource type %s", resource.Type)
	}

	// Get current resource to preserve existing properties
	existingResource, err := s.getResource(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing resource: %w", err)
	}

	// Add drift simulation tag
	if tags, ok := existingResource["tags"].(map[string]interface{}); ok {
		tags["DriftSimulation"] = fmt.Sprintf("Created-%s", time.Now().Format("2006-01-02-15:04:05"))
	} else {
		existingResource["tags"] = map[string]interface{}{
			"DriftSimulation": fmt.Sprintf("Created-%s", time.Now().Format("2006-01-02-15:04:05")),
		}
	}

	// Update resource with new tag
	if err := s.updateResource(ctx, apiURL, existingResource); err != nil {
		return nil, fmt.Errorf("failed to add tag: %w", err)
	}

	result.Changes["added_tag"] = map[string]string{
		"DriftSimulation": fmt.Sprintf("Created-%s", time.Now().Format("2006-01-02-15:04:05")),
	}
	result.RollbackData = &RollbackData{
		Provider:     "azure",
		ResourceType: resource.Type,
		ResourceID:   resource.ID,
		Action:       "remove_tag",
		OriginalData: map[string]interface{}{
			"api_url": apiURL,
			"tag_key": "DriftSimulation",
		},
		Timestamp: time.Now(),
	}
	result.Success = true

	return result, nil
}

// simulateNSGRuleDrift adds a new rule to a Network Security Group
func (s *AzureSimulator) simulateNSGRuleDrift(ctx context.Context, resource *state.Resource) (*SimulationResult, error) {
	result := &SimulationResult{
		Provider:     "azure",
		ResourceType: resource.Type,
		ResourceID:   resource.ID,
		DriftType:    DriftTypeRuleAddition,
		Changes:      make(map[string]interface{}),
		CostEstimate: "$0.00 (NSG rules are free)",
	}

	// Find NSG resource
	var nsgName, resourceGroup string
	if resource.Type == "azurerm_network_security_group" {
		nsgName = s.extractResourceName(resource)
		resourceGroup = s.extractResourceGroup(resource)
	} else {
		return nil, fmt.Errorf("cannot add NSG rule to resource type %s", resource.Type)
	}

	if nsgName == "" || resourceGroup == "" {
		return nil, fmt.Errorf("could not extract NSG details")
	}

	// Create a harmless security rule
	ruleName := "DriftSimulation-Rule"
	apiURL := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/%s/securityRules/%s?api-version=2023-04-01",
		s.subscriptionID, resourceGroup, nsgName, ruleName)

	rule := map[string]interface{}{
		"properties": map[string]interface{}{
			"description":               "DriftSimulation - Test rule",
			"protocol":                  "Tcp",
			"sourcePortRange":          "*",
			"destinationPortRange":     "8443",
			"sourceAddressPrefix":      "192.0.2.0/32", // TEST-NET-1
			"destinationAddressPrefix": "*",
			"access":                   "Deny",
			"priority":                 4096,
			"direction":                "Inbound",
		},
	}

	// Add the rule
	if err := s.createResource(ctx, apiURL, rule); err != nil {
		return nil, fmt.Errorf("failed to add NSG rule: %w", err)
	}

	result.Changes["added_rule"] = map[string]interface{}{
		"name":        ruleName,
		"protocol":    "Tcp",
		"port":        "8443",
		"source":      "192.0.2.0/32",
		"action":      "Deny",
		"direction":   "Inbound",
		"description": "DriftSimulation - Test rule",
	}
	result.RollbackData = &RollbackData{
		Provider:     "azure",
		ResourceType: "azurerm_network_security_group",
		ResourceID:   nsgName,
		Action:       "remove_rule",
		OriginalData: map[string]interface{}{
			"api_url": apiURL,
		},
		Timestamp: time.Now(),
	}
	result.Success = true

	return result, nil
}

// simulateResourceCreation creates a new resource not in state
func (s *AzureSimulator) simulateResourceCreation(ctx context.Context, resource *state.Resource, state *state.TerraformState) (*SimulationResult, error) {
	result := &SimulationResult{
		Provider:     "azure",
		DriftType:    DriftTypeResourceCreation,
		Changes:      make(map[string]interface{}),
		CostEstimate: "$0.00 (using free tier resources)",
	}

	// Create a resource group with auto-delete tag
	rgName := fmt.Sprintf("drift-simulation-%d", time.Now().Unix())
	location := "eastus"
	
	apiURL := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s?api-version=2021-04-01",
		s.subscriptionID, rgName)

	rgBody := map[string]interface{}{
		"location": location,
		"tags": map[string]interface{}{
			"DriftSimulation": "true",
			"AutoDelete":      time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"Purpose":         "DriftSimulation-Testing",
		},
	}

	if err := s.createResource(ctx, apiURL, rgBody); err != nil {
		return nil, fmt.Errorf("failed to create resource group: %w", err)
	}

	result.ResourceType = "azurerm_resource_group"
	result.ResourceID = rgName
	result.Changes["created_resource"] = map[string]interface{}{
		"type":     "azurerm_resource_group",
		"name":     rgName,
		"location": location,
	}
	result.RollbackData = &RollbackData{
		Provider:     "azure",
		ResourceType: "azurerm_resource_group",
		ResourceID:   rgName,
		Action:       "delete_resource",
		OriginalData: map[string]interface{}{
			"api_url": apiURL,
		},
		Timestamp: time.Now(),
	}
	result.Success = true

	return result, nil
}

// simulateAttributeChange modifies a resource attribute
func (s *AzureSimulator) simulateAttributeChange(ctx context.Context, resource *state.Resource) (*SimulationResult, error) {
	// For Azure, we'll just add a tag as attribute changes often require resource recreation
	return s.simulateTagDrift(ctx, resource)
}

// DetectDrift detects drift in Azure resources
func (s *AzureSimulator) DetectDrift(ctx context.Context, state *state.TerraformState) ([]DriftItem, error) {
	var drifts []DriftItem

	// Initialize if needed
	if s.accessToken == "" {
		if err := s.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	// Check each resource in state
	for _, resource := range state.Resources {
		if !strings.HasPrefix(resource.Type, "azurerm_") {
			continue
		}

		drift := s.checkResourceDrift(ctx, &resource)
		if drift != nil {
			drifts = append(drifts, *drift)
		}
	}

	// Check for unmanaged resources
	unmanagedDrifts := s.checkUnmanagedResources(ctx, state)
	drifts = append(drifts, unmanagedDrifts...)

	return drifts, nil
}

// checkResourceDrift checks a single Azure resource for drift
func (s *AzureSimulator) checkResourceDrift(ctx context.Context, resource *state.Resource) *DriftItem {
	// Build API URL
	var apiURL string
	resourceGroup := s.extractResourceGroup(resource)
	resourceName := s.extractResourceName(resource)

	switch resource.Type {
	case "azurerm_resource_group":
		apiURL = fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s?api-version=2021-04-01",
			s.subscriptionID, resourceName)
	case "azurerm_network_security_group":
		apiURL = fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkSecurityGroups/%s?api-version=2023-04-01",
			s.subscriptionID, resourceGroup, resourceName)
	default:
		return nil
	}

	// Get current resource state
	currentResource, err := s.getResource(ctx, apiURL)
	if err != nil {
		return nil
	}

	// Check for drift simulation tags
	if tags, ok := currentResource["tags"].(map[string]interface{}); ok {
		if _, exists := tags["DriftSimulation"]; exists {
			return &DriftItem{
				ResourceID:   resource.ID,
				ResourceType: resource.Type,
				DriftType:    "tag_addition",
				Before: map[string]interface{}{
					"tags": s.extractResourceTags(resource),
				},
				After: map[string]interface{}{
					"tags": tags,
				},
				Impact: "Low - Tag addition detected",
			}
		}
	}

	// Check for NSG rule additions
	if resource.Type == "azurerm_network_security_group" {
		if props, ok := currentResource["properties"].(map[string]interface{}); ok {
			if rules, ok := props["securityRules"].([]interface{}); ok {
				for _, r := range rules {
					if rule, ok := r.(map[string]interface{}); ok {
						if name, ok := rule["name"].(string); ok && strings.Contains(name, "DriftSimulation") {
							return &DriftItem{
								ResourceID:   resource.ID,
								ResourceType: resource.Type,
								DriftType:    "rule_addition",
								After: map[string]interface{}{
									"added_rule": rule,
								},
								Impact: "High - Security rule added",
							}
						}
					}
				}
			}
		}
	}

	return nil
}

// checkUnmanagedResources checks for resources not in state
func (s *AzureSimulator) checkUnmanagedResources(ctx context.Context, state *state.TerraformState) []DriftItem {
	var drifts []DriftItem

	// List all resource groups
	apiURL := fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups?api-version=2021-04-01", s.subscriptionID)
	
	response, err := s.makeAPICall(ctx, "GET", apiURL, nil)
	if err != nil {
		return drifts
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return drifts
	}

	if value, ok := result["value"].([]interface{}); ok {
		for _, rg := range value {
			if rgData, ok := rg.(map[string]interface{}); ok {
				if name, ok := rgData["name"].(string); ok {
					if strings.HasPrefix(name, "drift-simulation-") {
						// Check if this RG is in state
						found := false
						for _, resource := range state.Resources {
							if resource.Type == "azurerm_resource_group" {
								rgName := s.extractResourceName(&resource)
								if rgName == name {
									found = true
									break
								}
							}
						}

						if !found {
							drifts = append(drifts, DriftItem{
								ResourceID:   name,
								ResourceType: "azurerm_resource_group",
								DriftType:    "unmanaged_resource",
								After: map[string]interface{}{
									"name":     name,
									"location": rgData["location"],
									"tags":     rgData["tags"],
								},
								Impact: "High - Unmanaged resource group detected",
							})
						}
					}
				}
			}
		}
	}

	return drifts
}

// Rollback undoes the simulated drift
func (s *AzureSimulator) Rollback(ctx context.Context, data *RollbackData) error {
	// Initialize if needed
	if s.accessToken == "" {
		if err := s.Initialize(ctx); err != nil {
			return err
		}
	}

	switch data.Action {
	case "remove_tag":
		return s.rollbackTagRemoval(ctx, data)
	case "remove_rule":
		return s.rollbackRuleRemoval(ctx, data)
	case "delete_resource":
		return s.rollbackResourceDeletion(ctx, data)
	default:
		return fmt.Errorf("unknown rollback action: %s", data.Action)
	}
}

// Helper functions

func (s *AzureSimulator) findResource(resourceID string, state *state.TerraformState) *state.Resource {
	for _, resource := range state.Resources {
		if resource.ID == resourceID || resource.Name == resourceID {
			return &resource
		}
	}
	return nil
}

func (s *AzureSimulator) extractResourceGroup(resource *state.Resource) string {
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if rg, ok := resource.Instances[0].Attributes["resource_group_name"].(string); ok {
			return rg
		}
	}
	return ""
}

func (s *AzureSimulator) extractResourceName(resource *state.Resource) string {
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if name, ok := resource.Instances[0].Attributes["name"].(string); ok {
			return name
		}
	}
	return ""
}

func (s *AzureSimulator) extractResourceTags(resource *state.Resource) map[string]string {
	tags := make(map[string]string)
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if t, ok := resource.Instances[0].Attributes["tags"].(map[string]interface{}); ok {
			for k, v := range t {
				tags[k] = fmt.Sprintf("%v", v)
			}
		}
	}
	return tags
}

func (s *AzureSimulator) getSubscriptionFromCLI() string {
	// This would execute: az account show --query id -o tsv
	// For now, return empty
	return ""
}

func (s *AzureSimulator) getAccessToken() (string, error) {
	// This would get token from Azure CLI or managed identity
	// For now, return a placeholder
	return "dummy-token", nil
}

func (s *AzureSimulator) makeAPICall(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var bodyData []byte
	var err error
	
	if body != nil {
		bodyData, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(string(bodyData)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var respBody []byte
	if resp.Body != nil {
		respBody, _ = json.Marshal(map[string]interface{}{
			"simulated": true,
			"message":   "Drift simulation response",
		})
	}

	return respBody, nil
}

func (s *AzureSimulator) getResource(ctx context.Context, url string) (map[string]interface{}, error) {
	body, err := s.makeAPICall(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var resource map[string]interface{}
	if err := json.Unmarshal(body, &resource); err != nil {
		return nil, err
	}

	return resource, nil
}

func (s *AzureSimulator) createResource(ctx context.Context, url string, body interface{}) error {
	_, err := s.makeAPICall(ctx, "PUT", url, body)
	return err
}

func (s *AzureSimulator) updateResource(ctx context.Context, url string, body interface{}) error {
	_, err := s.makeAPICall(ctx, "PATCH", url, body)
	return err
}

func (s *AzureSimulator) deleteResource(ctx context.Context, url string) error {
	_, err := s.makeAPICall(ctx, "DELETE", url, nil)
	return err
}

// Rollback functions

func (s *AzureSimulator) rollbackTagRemoval(ctx context.Context, data *RollbackData) error {
	apiURL := data.OriginalData["api_url"].(string)
	tagKey := data.OriginalData["tag_key"].(string)

	// Get current resource
	resource, err := s.getResource(ctx, apiURL)
	if err != nil {
		return err
	}

	// Remove the drift simulation tag
	if tags, ok := resource["tags"].(map[string]interface{}); ok {
		delete(tags, tagKey)
		resource["tags"] = tags
	}

	// Update resource
	return s.updateResource(ctx, apiURL, resource)
}

func (s *AzureSimulator) rollbackRuleRemoval(ctx context.Context, data *RollbackData) error {
	apiURL := data.OriginalData["api_url"].(string)
	return s.deleteResource(ctx, apiURL)
}

func (s *AzureSimulator) rollbackResourceDeletion(ctx context.Context, data *RollbackData) error {
	apiURL := data.OriginalData["api_url"].(string)
	return s.deleteResource(ctx, apiURL)
}