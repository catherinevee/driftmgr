package integrations

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// IntegrationManager manages external integrations
type IntegrationManager struct {
	integrations map[string]Integration
	mu           sync.RWMutex
	config       *IntegrationConfig
}

// Integration represents an external integration
type Integration struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`     // webhook, api, sdk, etc.
	Provider  string                 `json:"provider"` // slack, teams, pagerduty, etc.
	Config    map[string]interface{} `json:"config"`
	Enabled   bool                   `json:"enabled"`
	Status    string                 `json:"status"` // active, inactive, error
	LastSync  time.Time              `json:"last_sync"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// IntegrationConfig represents configuration for the integration manager
type IntegrationConfig struct {
	MaxIntegrations     int           `json:"max_integrations"`
	SyncInterval        time.Duration `json:"sync_interval"`
	RetryAttempts       int           `json:"retry_attempts"`
	RetryDelay          time.Duration `json:"retry_delay"`
	Timeout             time.Duration `json:"timeout"`
	AutoSync            bool          `json:"auto_sync"`
	NotificationEnabled bool          `json:"notification_enabled"`
}

// IntegrationEvent represents an integration event
type IntegrationEvent struct {
	ID            string                 `json:"id"`
	IntegrationID string                 `json:"integration_id"`
	Type          string                 `json:"type"` // webhook, notification, sync, etc.
	Data          map[string]interface{} `json:"data"`
	Status        string                 `json:"status"` // pending, sent, failed, delivered
	Timestamp     time.Time              `json:"timestamp"`
	RetryCount    int                    `json:"retry_count"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// NewIntegrationManager creates a new integration manager
func NewIntegrationManager() *IntegrationManager {
	config := &IntegrationConfig{
		MaxIntegrations:     100,
		SyncInterval:        5 * time.Minute,
		RetryAttempts:       3,
		RetryDelay:          30 * time.Second,
		Timeout:             30 * time.Second,
		AutoSync:            true,
		NotificationEnabled: true,
	}

	return &IntegrationManager{
		integrations: make(map[string]Integration),
		config:       config,
	}
}

// CreateIntegration creates a new integration
func (im *IntegrationManager) CreateIntegration(ctx context.Context, integration *Integration) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	// Check integration limit
	if len(im.integrations) >= im.config.MaxIntegrations {
		return fmt.Errorf("maximum number of integrations reached (%d)", im.config.MaxIntegrations)
	}

	// Validate integration
	if err := im.validateIntegration(integration); err != nil {
		return fmt.Errorf("invalid integration: %w", err)
	}

	// Set defaults
	if integration.ID == "" {
		integration.ID = fmt.Sprintf("integration_%d", time.Now().Unix())
	}
	if integration.Status == "" {
		integration.Status = "inactive"
	}
	integration.CreatedAt = time.Now()
	integration.UpdatedAt = time.Now()

	// Store integration
	im.integrations[integration.ID] = *integration

	return nil
}

// GetIntegration retrieves an integration
func (im *IntegrationManager) GetIntegration(ctx context.Context, integrationID string) (*Integration, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	integration, exists := im.integrations[integrationID]
	if !exists {
		return nil, fmt.Errorf("integration %s not found", integrationID)
	}

	return &integration, nil
}

// ListIntegrations lists all integrations
func (im *IntegrationManager) ListIntegrations(ctx context.Context) ([]*Integration, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	integrations := make([]*Integration, 0, len(im.integrations))
	for _, integration := range im.integrations {
		integrations = append(integrations, &integration)
	}

	return integrations, nil
}

// UpdateIntegration updates an integration
func (im *IntegrationManager) UpdateIntegration(ctx context.Context, integrationID string, updates *Integration) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	integration, exists := im.integrations[integrationID]
	if !exists {
		return fmt.Errorf("integration %s not found", integrationID)
	}

	// Update fields
	if updates.Name != "" {
		integration.Name = updates.Name
	}
	if updates.Type != "" {
		integration.Type = updates.Type
	}
	if updates.Provider != "" {
		integration.Provider = updates.Provider
	}
	if updates.Config != nil {
		integration.Config = updates.Config
	}
	integration.Enabled = updates.Enabled
	integration.UpdatedAt = time.Now()

	// Store updated integration
	im.integrations[integrationID] = integration

	return nil
}

// DeleteIntegration deletes an integration
func (im *IntegrationManager) DeleteIntegration(ctx context.Context, integrationID string) error {
	im.mu.Lock()
	defer im.mu.Unlock()

	_, exists := im.integrations[integrationID]
	if !exists {
		return fmt.Errorf("integration %s not found", integrationID)
	}

	delete(im.integrations, integrationID)
	return nil
}

// SendEvent sends an event to an integration
func (im *IntegrationManager) SendEvent(ctx context.Context, integrationID string, event *IntegrationEvent) error {
	im.mu.RLock()
	integration, exists := im.integrations[integrationID]
	im.mu.RUnlock()

	if !exists {
		return fmt.Errorf("integration %s not found", integrationID)
	}

	if !integration.Enabled {
		return fmt.Errorf("integration %s is disabled", integrationID)
	}

	// Send event based on integration type
	switch integration.Type {
	case "webhook":
		return im.sendWebhookEvent(ctx, &integration, event)
	case "api":
		return im.sendAPIEvent(ctx, &integration, event)
	case "sdk":
		return im.sendSDKEvent(ctx, &integration, event)
	default:
		return fmt.Errorf("unknown integration type: %s", integration.Type)
	}
}

// TestIntegration tests an integration
func (im *IntegrationManager) TestIntegration(ctx context.Context, integrationID string) error {
	im.mu.RLock()
	_, exists := im.integrations[integrationID]
	im.mu.RUnlock()

	if !exists {
		return fmt.Errorf("integration %s not found", integrationID)
	}

	// Create test event
	testEvent := &IntegrationEvent{
		ID:            fmt.Sprintf("test_%d", time.Now().Unix()),
		IntegrationID: integrationID,
		Type:          "test",
		Data: map[string]interface{}{
			"message":   "Test integration",
			"timestamp": time.Now().Format(time.RFC3339),
		},
		Status:    "pending",
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Send test event
	return im.SendEvent(ctx, integrationID, testEvent)
}

// GetIntegrationStatus returns the status of all integrations
func (im *IntegrationManager) GetIntegrationStatus(ctx context.Context) (*IntegrationStatus, error) {
	im.mu.RLock()
	defer im.mu.RUnlock()

	status := &IntegrationStatus{
		TotalIntegrations:      len(im.integrations),
		ActiveIntegrations:     0,
		InactiveIntegrations:   0,
		ErrorIntegrations:      0,
		IntegrationsByType:     make(map[string]int),
		IntegrationsByProvider: make(map[string]int),
		LastSync:               time.Time{},
		Metadata:               make(map[string]interface{}),
	}

	for _, integration := range im.integrations {
		// Count by status
		switch integration.Status {
		case "active":
			status.ActiveIntegrations++
		case "inactive":
			status.InactiveIntegrations++
		case "error":
			status.ErrorIntegrations++
		}

		// Count by type
		status.IntegrationsByType[integration.Type]++

		// Count by provider
		status.IntegrationsByProvider[integration.Provider]++

		// Track last sync
		if integration.LastSync.After(status.LastSync) {
			status.LastSync = integration.LastSync
		}
	}

	return status, nil
}

// IntegrationStatus represents the status of integrations
type IntegrationStatus struct {
	TotalIntegrations      int                    `json:"total_integrations"`
	ActiveIntegrations     int                    `json:"active_integrations"`
	InactiveIntegrations   int                    `json:"inactive_integrations"`
	ErrorIntegrations      int                    `json:"error_integrations"`
	IntegrationsByType     map[string]int         `json:"integrations_by_type"`
	IntegrationsByProvider map[string]int         `json:"integrations_by_provider"`
	LastSync               time.Time              `json:"last_sync"`
	Metadata               map[string]interface{} `json:"metadata,omitempty"`
}

// Helper methods

// validateIntegration validates an integration
func (im *IntegrationManager) validateIntegration(integration *Integration) error {
	if integration.Name == "" {
		return fmt.Errorf("integration name is required")
	}
	if integration.Type == "" {
		return fmt.Errorf("integration type is required")
	}
	if integration.Provider == "" {
		return fmt.Errorf("integration provider is required")
	}
	return nil
}

// sendWebhookEvent sends a webhook event
func (im *IntegrationManager) sendWebhookEvent(ctx context.Context, integration *Integration, event *IntegrationEvent) error {
	// Simplified webhook implementation
	// In a real system, you would make an HTTP POST request
	_ = integration // Use the integration variable to avoid linting error
	fmt.Printf("Sending webhook event to %s: %s\n", integration.Name, event.Type)
	return nil
}

// sendAPIEvent sends an API event
func (im *IntegrationManager) sendAPIEvent(ctx context.Context, integration *Integration, event *IntegrationEvent) error {
	// Simplified API implementation
	// In a real system, you would make an API call
	_ = integration // Use the integration variable to avoid linting error
	fmt.Printf("Sending API event to %s: %s\n", integration.Name, event.Type)
	return nil
}

// sendSDKEvent sends an SDK event
func (im *IntegrationManager) sendSDKEvent(ctx context.Context, integration *Integration, event *IntegrationEvent) error {
	// Simplified SDK implementation
	// In a real system, you would use the provider's SDK
	_ = integration // Use the integration variable to avoid linting error
	fmt.Printf("Sending SDK event to %s: %s\n", integration.Name, event.Type)
	return nil
}

// SetConfig updates the integration manager configuration
func (im *IntegrationManager) SetConfig(config *IntegrationConfig) {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.config = config
}

// GetConfig returns the current integration manager configuration
func (im *IntegrationManager) GetConfig() *IntegrationConfig {
	im.mu.RLock()
	defer im.mu.RUnlock()
	return im.config
}
