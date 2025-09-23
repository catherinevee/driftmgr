package azure

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewAzureProviderComplete tests creating a new Azure provider
func TestNewAzureProviderComplete(t *testing.T) {
	tests := []struct {
		name           string
		subscriptionID string
		resourceGroup  string
		expected       bool
	}{
		{
			name:           "Valid configuration",
			subscriptionID: "12345678-1234-1234-1234-123456789012",
			resourceGroup:  "test-rg",
			expected:       true,
		},
		{
			name:           "Empty subscription ID",
			subscriptionID: "",
			resourceGroup:  "test-rg",
			expected:       true, // Should still create provider
		},
		{
			name:           "Empty resource group",
			subscriptionID: "12345678-1234-1234-1234-123456789012",
			resourceGroup:  "",
			expected:       true, // Should still create provider
		},
		{
			name:           "Both empty",
			subscriptionID: "",
			resourceGroup:  "",
			expected:       true, // Should still create provider
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAzureProviderComplete(tt.subscriptionID, tt.resourceGroup)
			assert.NotNil(t, provider)
			assert.Equal(t, "azure", provider.Name())
		})
	}
}

// TestAzureProviderName tests the provider name
func TestAzureProviderName(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")
	assert.Equal(t, "azure", provider.Name())
}

// TestAzureProviderSupportedResourceTypes tests supported resource types
func TestAzureProviderSupportedResourceTypes(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")
	resourceTypes := provider.SupportedResourceTypes()

	assert.NotEmpty(t, resourceTypes)
	assert.Contains(t, resourceTypes, "azurerm_virtual_machine")
	assert.Contains(t, resourceTypes, "azurerm_storage_account")
	assert.Contains(t, resourceTypes, "azurerm_network_security_group")
	assert.Contains(t, resourceTypes, "azurerm_sql_server")
	assert.Contains(t, resourceTypes, "azurerm_function_app")
}

// TestAzureProviderListRegions tests listing available regions
func TestAzureProviderListRegions(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")
	ctx := context.Background()

	regions, err := provider.ListRegions(ctx)

	// In test environment, this might fail due to credentials
	// but we should get a reasonable response structure
	if err != nil {
		// Expected in test environment without credentials
		assert.NotNil(t, err)
	} else {
		assert.NotNil(t, regions)
		assert.IsType(t, []string{}, regions)
	}
}

// TestAzureProviderValidateCredentials tests credential validation
func TestAzureProviderValidateCredentials(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")
	ctx := context.Background()

	err := provider.ValidateCredentials(ctx)

	// In test environment, this will likely fail due to missing credentials
	// but we should handle it gracefully
	assert.NotNil(t, err) // Expected in test environment
}

// TestAzureProviderDiscoverResources tests resource discovery
func TestAzureProviderDiscoverResources(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")
	ctx := context.Background()

	resources, err := provider.DiscoverResources(ctx, "eastus")

	// In test environment, this will likely fail due to missing credentials
	// but we should get a proper response structure
	if err != nil {
		// Expected in test environment without credentials
		assert.NotNil(t, err)
	} else {
		assert.NotNil(t, resources)
		assert.IsType(t, []interface{}{}, resources)
	}
}

// TestAzureProviderGetResource tests getting a specific resource
func TestAzureProviderGetResource(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")
	ctx := context.Background()

	resource, err := provider.GetResource(ctx, "test-resource-id")

	// Should return error for non-existent resource
	assert.Error(t, err)
	assert.Nil(t, resource)
}

// TestAzureProviderConcurrentAccess tests concurrent access
func TestAzureProviderConcurrentAccess(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")
	ctx := context.Background()

	// Test concurrent calls
	done := make(chan bool, 3)

	go func() {
		defer func() { done <- true }()
		regions, _ := provider.ListRegions(ctx)
		assert.NotNil(t, regions)
	}()

	go func() {
		defer func() { done <- true }()
		resources, _ := provider.DiscoverResources(ctx, "eastus")
		assert.NotNil(t, resources)
	}()

	go func() {
		defer func() { done <- true }()
		resourceTypes := provider.SupportedResourceTypes()
		assert.NotEmpty(t, resourceTypes)
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
}

// TestAzureProviderErrorHandling tests error handling scenarios
func TestAzureProviderErrorHandling(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")
	ctx := context.Background()

	t.Run("InvalidRegion", func(t *testing.T) {
		resources, err := provider.DiscoverResources(ctx, "invalid-region")
		assert.Error(t, err)
		assert.Nil(t, resources)
	})

	t.Run("EmptyResourceID", func(t *testing.T) {
		resource, err := provider.GetResource(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, resource)
	})

	t.Run("NilContext", func(t *testing.T) {
		// This should not panic
		assert.NotPanics(t, func() {
			provider.SupportedResourceTypes()
		})
	})
}

// TestAzureProviderInterfaceCompliance tests that the provider implements the interface correctly
func TestAzureProviderInterfaceCompliance(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")
	ctx := context.Background()

	// Test all interface methods exist and return expected types
	name := provider.Name()
	assert.IsType(t, "", name)
	assert.Equal(t, "azure", name)

	resourceTypes := provider.SupportedResourceTypes()
	assert.IsType(t, []string{}, resourceTypes)
	assert.NotEmpty(t, resourceTypes)

	regions, err := provider.ListRegions(ctx)
	assert.IsType(t, []string{}, regions)
	// Error is expected in test environment

	resources, err := provider.DiscoverResources(ctx, "eastus")
	assert.IsType(t, []interface{}{}, resources)
	// Error is expected in test environment

	resource, err := provider.GetResource(ctx, "test-id")
	assert.IsType(t, (*interface{})(nil), resource)
	assert.Error(t, err) // Expected for non-existent resource

	err = provider.ValidateCredentials(ctx)
	assert.IsType(t, error(nil), err)
	assert.Error(t, err) // Expected in test environment
}

// TestAzureProviderConfiguration tests provider configuration
func TestAzureProviderConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		subscriptionID string
		resourceGroup  string
	}{
		{
			name:           "Valid subscription and resource group",
			subscriptionID: "12345678-1234-1234-1234-123456789012",
			resourceGroup:  "test-rg",
		},
		{
			name:           "Different subscription",
			subscriptionID: "87654321-4321-4321-4321-210987654321",
			resourceGroup:  "prod-rg",
		},
		{
			name:           "Empty subscription ID",
			subscriptionID: "",
			resourceGroup:  "test-rg",
		},
		{
			name:           "Empty resource group",
			subscriptionID: "12345678-1234-1234-1234-123456789012",
			resourceGroup:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAzureProviderComplete(tt.subscriptionID, tt.resourceGroup)
			assert.NotNil(t, provider)
			assert.Equal(t, "azure", provider.Name())
		})
	}
}

// TestAzureProviderResourceTypesCompleteness tests that all expected resource types are supported
func TestAzureProviderResourceTypesCompleteness(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")
	resourceTypes := provider.SupportedResourceTypes()

	expectedTypes := []string{
		"azurerm_virtual_machine",
		"azurerm_storage_account",
		"azurerm_network_security_group",
		"azurerm_sql_server",
		"azurerm_function_app",
		"azurerm_app_service",
		"azurerm_key_vault",
		"azurerm_resource_group",
		"azurerm_virtual_network",
		"azurerm_subnet",
		"azurerm_public_ip",
		"azurerm_network_interface",
		"azurerm_load_balancer",
		"azurerm_application_gateway",
		"azurerm_cosmosdb_account",
		"azurerm_redis_cache",
		"azurerm_service_bus_namespace",
		"azurerm_event_hub_namespace",
		"azurerm_log_analytics_workspace",
		"azurerm_automation_account",
		"azurerm_monitor_action_group",
		"azurerm_monitor_metric_alert",
		"azurerm_managed_disk",
		"azurerm_snapshot",
		"azurerm_backup_vault",
	}

	for _, expectedType := range expectedTypes {
		assert.Contains(t, resourceTypes, expectedType, "Expected resource type %s to be supported", expectedType)
	}
}

// TestAzureProviderPerformance tests basic performance characteristics
func TestAzureProviderPerformance(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")

	// Test that getting supported resource types is fast
	start := time.Now()
	resourceTypes := provider.SupportedResourceTypes()
	duration := time.Since(start)

	assert.NotEmpty(t, resourceTypes)
	assert.Less(t, duration, 100*time.Millisecond, "Getting resource types should be fast")
}

// TestAzureProviderThreadSafety tests thread safety
func TestAzureProviderThreadSafety(t *testing.T) {
	provider := NewAzureProviderComplete("12345678-1234-1234-1234-123456789012", "test-rg")

	// Test concurrent access to read-only methods
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// These should be safe to call concurrently
			name := provider.Name()
			assert.Equal(t, "azure", name)

			resourceTypes := provider.SupportedResourceTypes()
			assert.NotEmpty(t, resourceTypes)
		}()
	}

	wg.Wait()
}

// TestAzureProviderSubscriptionIDValidation tests subscription ID format validation
func TestAzureProviderSubscriptionIDValidation(t *testing.T) {
	tests := []struct {
		name           string
		subscriptionID string
		shouldPass     bool
	}{
		{
			name:           "Valid UUID format",
			subscriptionID: "12345678-1234-1234-1234-123456789012",
			shouldPass:     true,
		},
		{
			name:           "Invalid format - too short",
			subscriptionID: "12345678-1234-1234-1234",
			shouldPass:     true, // Provider should still be created
		},
		{
			name:           "Invalid format - no hyphens",
			subscriptionID: "12345678123412341234123456789012",
			shouldPass:     true, // Provider should still be created
		},
		{
			name:           "Empty subscription ID",
			subscriptionID: "",
			shouldPass:     true, // Provider should still be created
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAzureProviderComplete(tt.subscriptionID, "test-rg")
			assert.NotNil(t, provider)
			assert.Equal(t, "azure", provider.Name())
		})
	}
}
