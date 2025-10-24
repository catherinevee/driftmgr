package integrations_test

import (
	"context"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/integrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationManager_CreateIntegration(t *testing.T) {
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	integration := &integrations.Integration{
		Name:     "Test Webhook",
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/test",
		},
		Enabled: true,
	}

	err := manager.CreateIntegration(ctx, integration)
	require.NoError(t, err)
	assert.NotEmpty(t, integration.ID)
	assert.Equal(t, "inactive", integration.Status)
	assert.False(t, integration.CreatedAt.IsZero())
	assert.False(t, integration.UpdatedAt.IsZero())
}

func TestIntegrationManager_GetIntegration(t *testing.T) {
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	// Create integration
	integration := &integrations.Integration{
		Name:     "Test Webhook",
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/test",
		},
		Enabled: true,
	}

	err := manager.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	// Retrieve integration
	retrievedIntegration, err := manager.GetIntegration(ctx, integration.ID)
	require.NoError(t, err)
	assert.Equal(t, integration.ID, retrievedIntegration.ID)
	assert.Equal(t, integration.Name, retrievedIntegration.Name)
	assert.Equal(t, integration.Type, retrievedIntegration.Type)
	assert.Equal(t, integration.Provider, retrievedIntegration.Provider)
}

func TestIntegrationManager_ListIntegrations(t *testing.T) {
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	// Create multiple integrations
	integration1 := &integrations.Integration{
		Name:     "Test Webhook 1",
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/test1",
		},
		Enabled: true,
	}

	integration2 := &integrations.Integration{
		Name:     "Test Webhook 2",
		Type:     "webhook",
		Provider: "teams",
		Config: map[string]interface{}{
			"url": "https://teams.microsoft.com/test2",
		},
		Enabled: true,
	}

	err := manager.CreateIntegration(ctx, integration1)
	require.NoError(t, err)

	err = manager.CreateIntegration(ctx, integration2)
	require.NoError(t, err)

	// List integrations
	integrationsList, err := manager.ListIntegrations(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(integrationsList), 2)

	// Verify both integrations are in the list
	integrationIDs := make(map[string]bool)
	for _, i := range integrationsList {
		integrationIDs[i.ID] = true
	}
	assert.True(t, integrationIDs[integration1.ID])
	assert.True(t, integrationIDs[integration2.ID])
}

func TestIntegrationManager_UpdateIntegration(t *testing.T) {
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	// Create integration
	integration := &integrations.Integration{
		Name:     "Test Webhook",
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/test",
		},
		Enabled: true,
	}

	err := manager.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	// Update integration
	updates := &integrations.Integration{
		Name:    "Updated Webhook",
		Enabled: false,
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/updated",
		},
	}

	err = manager.UpdateIntegration(ctx, integration.ID, updates)
	require.NoError(t, err)

	// Verify updates
	updatedIntegration, err := manager.GetIntegration(ctx, integration.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Webhook", updatedIntegration.Name)
	assert.False(t, updatedIntegration.Enabled)
	assert.Equal(t, "https://hooks.slack.com/updated", updatedIntegration.Config["url"])
}

func TestIntegrationManager_DeleteIntegration(t *testing.T) {
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	// Create integration
	integration := &integrations.Integration{
		Name:     "Test Webhook",
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/test",
		},
		Enabled: true,
	}

	err := manager.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	// Delete integration
	err = manager.DeleteIntegration(ctx, integration.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = manager.GetIntegration(ctx, integration.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestIntegrationManager_SendEvent(t *testing.T) {
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	// Create webhook integration
	integration := &integrations.Integration{
		Name:     "Test Webhook",
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/test",
		},
		Enabled: true,
	}

	err := manager.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	// Create event
	event := &integrations.IntegrationEvent{
		ID:            "test-event-1",
		IntegrationID: integration.ID,
		Type:          "alert",
		Data: map[string]interface{}{
			"message":  "Test alert",
			"severity": "high",
		},
		Status:    "pending",
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Send event
	err = manager.SendEvent(ctx, integration.ID, event)
	require.NoError(t, err)

	// Verify integration status was updated
	updatedIntegration, err := manager.GetIntegration(ctx, integration.ID)
	require.NoError(t, err)
	assert.False(t, updatedIntegration.LastSync.IsZero())
}

func TestIntegrationManager_TestIntegration(t *testing.T) {
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	// Create webhook integration
	integration := &integrations.Integration{
		Name:     "Test Webhook",
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/test",
		},
		Enabled: true,
	}

	err := manager.CreateIntegration(ctx, integration)
	require.NoError(t, err)

	// Test integration
	err = manager.TestIntegration(ctx, integration.ID)
	require.NoError(t, err)
}

func TestIntegrationManager_GetIntegrationStatus(t *testing.T) {
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	// Create multiple integrations with different statuses
	integration1 := &integrations.Integration{
		Name:     "Active Webhook",
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/test1",
		},
		Enabled: true,
		Status:  "active",
	}

	integration2 := &integrations.Integration{
		Name:     "Inactive Webhook",
		Type:     "webhook",
		Provider: "teams",
		Config: map[string]interface{}{
			"url": "https://teams.microsoft.com/test2",
		},
		Enabled: false,
		Status:  "inactive",
	}

	err := manager.CreateIntegration(ctx, integration1)
	require.NoError(t, err)

	err = manager.CreateIntegration(ctx, integration2)
	require.NoError(t, err)

	// Get status
	status, err := manager.GetIntegrationStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, status.TotalIntegrations)
	assert.Equal(t, 1, status.ActiveIntegrations)
	assert.Equal(t, 1, status.InactiveIntegrations)
	assert.Equal(t, 2, status.IntegrationsByType["webhook"])
	assert.Equal(t, 1, status.IntegrationsByProvider["slack"])
	assert.Equal(t, 1, status.IntegrationsByProvider["teams"])
}

func TestIntegrationManager_Validation(t *testing.T) {
	manager := integrations.NewIntegrationManager()
	ctx := context.Background()

	// Test missing name
	integration := &integrations.Integration{
		Type:     "webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/test",
		},
		Enabled: true,
	}

	err := manager.CreateIntegration(ctx, integration)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "integration name is required")

	// Test missing type
	integration = &integrations.Integration{
		Name:     "Test Webhook",
		Provider: "slack",
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/test",
		},
		Enabled: true,
	}

	err = manager.CreateIntegration(ctx, integration)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "integration type is required")

	// Test missing provider
	integration = &integrations.Integration{
		Name: "Test Webhook",
		Type: "webhook",
		Config: map[string]interface{}{
			"url": "https://hooks.slack.com/test",
		},
		Enabled: true,
	}

	err = manager.CreateIntegration(ctx, integration)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "integration provider is required")
}

func TestIntegrationManager_Configuration(t *testing.T) {
	manager := integrations.NewIntegrationManager()

	// Get current config
	config := manager.GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 100, config.MaxIntegrations)
	assert.Equal(t, 5*time.Minute, config.SyncInterval)
	assert.Equal(t, 3, config.RetryAttempts)

	// Update config
	newConfig := &integrations.IntegrationConfig{
		MaxIntegrations:     200,
		SyncInterval:        10 * time.Minute,
		RetryAttempts:       5,
		RetryDelay:          60 * time.Second,
		Timeout:             60 * time.Second,
		AutoSync:            false,
		NotificationEnabled: false,
	}
	manager.SetConfig(newConfig)

	// Verify config was updated
	updatedConfig := manager.GetConfig()
	assert.Equal(t, 200, updatedConfig.MaxIntegrations)
	assert.Equal(t, 10*time.Minute, updatedConfig.SyncInterval)
	assert.Equal(t, 5, updatedConfig.RetryAttempts)
	assert.Equal(t, 60*time.Second, updatedConfig.RetryDelay)
	assert.Equal(t, 60*time.Second, updatedConfig.Timeout)
	assert.False(t, updatedConfig.AutoSync)
	assert.False(t, updatedConfig.NotificationEnabled)
}


