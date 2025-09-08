package tenant

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// AccountManager manages cloud accounts and their resources
type AccountManager struct {
	accounts      map[string]*AccountInfo
	connections   map[string]*AccountConnection
	mu            sync.RWMutex
	tenantManager *TenantManager
	eventBus      EventBus
}

// AccountInfo represents detailed information about an account
type AccountInfo struct {
	Account       *Account               `json:"account"`
	Resources     []*models.Resource     `json:"resources"`
	LastSync      time.Time              `json:"last_sync"`
	SyncStatus    string                 `json:"sync_status"`
	ResourceCount int                    `json:"resource_count"`
	CostData      *AccountCostData       `json:"cost_data"`
	HealthStatus  *AccountHealthStatus   `json:"health_status"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AccountConnection represents a connection to a cloud account
type AccountConnection struct {
	AccountID  string                  `json:"account_id"`
	Provider   string                  `json:"provider"`
	Connection providers.CloudProvider `json:"-"`
	LastUsed   time.Time               `json:"last_used"`
	Status     string                  `json:"status"`
	ErrorCount int                     `json:"error_count"`
	LastError  string                  `json:"last_error,omitempty"`
	Metadata   map[string]interface{}  `json:"metadata,omitempty"`
}

// AccountCostData represents cost data for an account
type AccountCostData struct {
	TotalCost    float64            `json:"total_cost"`
	Currency     string             `json:"currency"`
	CostByType   map[string]float64 `json:"cost_by_type"`
	CostByRegion map[string]float64 `json:"cost_by_region"`
	LastUpdated  time.Time          `json:"last_updated"`
	Trend        string             `json:"trend"`
	BudgetUsage  float64            `json:"budget_usage"`
}

// AccountHealthStatus represents health status for an account
type AccountHealthStatus struct {
	OverallStatus string                 `json:"overall_status"`
	HealthScore   float64                `json:"health_score"`
	Issues        []AccountHealthIssue   `json:"issues"`
	LastChecked   time.Time              `json:"last_checked"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AccountHealthIssue represents a health issue in an account
type AccountHealthIssue struct {
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	ResourceID  string                 `json:"resource_id,omitempty"`
	DetectedAt  time.Time              `json:"detected_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewAccountManager creates a new account manager
func NewAccountManager(tenantManager *TenantManager, eventBus EventBus) *AccountManager {
	return &AccountManager{
		accounts:      make(map[string]*AccountInfo),
		connections:   make(map[string]*AccountConnection),
		tenantManager: tenantManager,
		eventBus:      eventBus,
	}
}

// RegisterAccount registers a new account for management
func (am *AccountManager) RegisterAccount(ctx context.Context, account *Account) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Create account info
	accountInfo := &AccountInfo{
		Account:       account,
		Resources:     []*models.Resource{},
		LastSync:      time.Time{},
		SyncStatus:    "pending",
		ResourceCount: 0,
		CostData:      &AccountCostData{},
		HealthStatus: &AccountHealthStatus{
			OverallStatus: "unknown",
			HealthScore:   0.0,
			Issues:        []AccountHealthIssue{},
		},
		Metadata: make(map[string]interface{}),
	}

	am.accounts[account.ID] = accountInfo

	// Create connection
	connection, err := am.createConnection(ctx, account)
	if err != nil {
		accountInfo.SyncStatus = "error"
		accountInfo.HealthStatus.OverallStatus = "error"
		return fmt.Errorf("failed to create connection: %w", err)
	}

	am.connections[account.ID] = connection

	// Publish event
	if am.eventBus != nil {
		event := TenantEvent{
			Type:      "account_registered",
			TenantID:  account.TenantID,
			AccountID: account.ID,
			Message:   fmt.Sprintf("Account '%s' registered for management", account.Name),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"provider": account.Provider,
				"region":   account.Region,
			},
		}
		am.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// SyncAccount synchronizes resources for an account
func (am *AccountManager) SyncAccount(ctx context.Context, accountID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	accountInfo, exists := am.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	connection, exists := am.connections[accountID]
	if !exists {
		return fmt.Errorf("connection for account %s not found", accountID)
	}

	// Update sync status
	accountInfo.SyncStatus = "syncing"
	accountInfo.LastSync = time.Now()

	// Discover resources
	resources, err := connection.Connection.DiscoverResources(ctx, accountInfo.Account.Region)
	if err != nil {
		accountInfo.SyncStatus = "error"
		connection.ErrorCount++
		connection.LastError = err.Error()
		connection.Status = "error"
		return fmt.Errorf("failed to discover resources: %w", err)
	}

	// Update account info
	accountInfo.Resources = make([]*models.Resource, len(resources))
	for i, resource := range resources {
		accountInfo.Resources[i] = &resource
	}
	accountInfo.ResourceCount = len(resources)
	accountInfo.SyncStatus = "completed"
	connection.Status = "active"
	connection.LastUsed = time.Now()
	connection.ErrorCount = 0
	connection.LastError = ""

	// Update cost data
	if err := am.updateCostData(ctx, accountInfo); err != nil {
		// Log error but don't fail the sync
		fmt.Printf("Warning: failed to update cost data for account %s: %v\n", accountID, err)
	}

	// Update health status
	if err := am.updateHealthStatus(ctx, accountInfo); err != nil {
		// Log error but don't fail the sync
		fmt.Printf("Warning: failed to update health status for account %s: %v\n", accountID, err)
	}

	// Publish event
	if am.eventBus != nil {
		event := TenantEvent{
			Type:      "account_synced",
			TenantID:  accountInfo.Account.TenantID,
			AccountID: accountID,
			Message:   fmt.Sprintf("Account '%s' synchronized successfully", accountInfo.Account.Name),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"resource_count": accountInfo.ResourceCount,
				"sync_status":    accountInfo.SyncStatus,
			},
		}
		am.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// GetAccountInfo retrieves account information
func (am *AccountManager) GetAccountInfo(ctx context.Context, accountID string) (*AccountInfo, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	accountInfo, exists := am.accounts[accountID]
	if !exists {
		return nil, fmt.Errorf("account %s not found", accountID)
	}

	return accountInfo, nil
}

// ListAccountResources lists resources for an account
func (am *AccountManager) ListAccountResources(ctx context.Context, accountID string) ([]*models.Resource, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	accountInfo, exists := am.accounts[accountID]
	if !exists {
		return nil, fmt.Errorf("account %s not found", accountID)
	}

	return accountInfo.Resources, nil
}

// GetAccountConnection retrieves the connection for an account
func (am *AccountManager) GetAccountConnection(ctx context.Context, accountID string) (*AccountConnection, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	connection, exists := am.connections[accountID]
	if !exists {
		return nil, fmt.Errorf("connection for account %s not found", accountID)
	}

	return connection, nil
}

// SyncAllAccounts synchronizes all registered accounts
func (am *AccountManager) SyncAllAccounts(ctx context.Context) error {
	am.mu.RLock()
	accountIDs := make([]string, 0, len(am.accounts))
	for accountID := range am.accounts {
		accountIDs = append(accountIDs, accountID)
	}
	am.mu.RUnlock()

	var errors []error
	for _, accountID := range accountIDs {
		if err := am.SyncAccount(ctx, accountID); err != nil {
			errors = append(errors, fmt.Errorf("failed to sync account %s: %w", accountID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("sync completed with errors: %v", errors)
	}

	return nil
}

// GetAccountCostData retrieves cost data for an account
func (am *AccountManager) GetAccountCostData(ctx context.Context, accountID string) (*AccountCostData, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	accountInfo, exists := am.accounts[accountID]
	if !exists {
		return nil, fmt.Errorf("account %s not found", accountID)
	}

	return accountInfo.CostData, nil
}

// GetAccountHealthStatus retrieves health status for an account
func (am *AccountManager) GetAccountHealthStatus(ctx context.Context, accountID string) (*AccountHealthStatus, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	accountInfo, exists := am.accounts[accountID]
	if !exists {
		return nil, fmt.Errorf("account %s not found", accountID)
	}

	return accountInfo.HealthStatus, nil
}

// ListAccountsByTenant lists all accounts for a tenant
func (am *AccountManager) ListAccountsByTenant(ctx context.Context, tenantID string) ([]*AccountInfo, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var accounts []*AccountInfo
	for _, accountInfo := range am.accounts {
		if accountInfo.Account.TenantID == tenantID {
			accounts = append(accounts, accountInfo)
		}
	}

	return accounts, nil
}

// RemoveAccount removes an account from management
func (am *AccountManager) RemoveAccount(ctx context.Context, accountID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	accountInfo, exists := am.accounts[accountID]
	if !exists {
		return fmt.Errorf("account %s not found", accountID)
	}

	// Remove connection
	delete(am.connections, accountID)

	// Remove account info
	delete(am.accounts, accountID)

	// Publish event
	if am.eventBus != nil {
		event := TenantEvent{
			Type:      "account_removed",
			TenantID:  accountInfo.Account.TenantID,
			AccountID: accountID,
			Message:   fmt.Sprintf("Account '%s' removed from management", accountInfo.Account.Name),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"provider": accountInfo.Account.Provider,
			},
		}
		am.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// Helper methods

// createConnection creates a connection to a cloud account
func (am *AccountManager) createConnection(ctx context.Context, account *Account) (*AccountConnection, error) {
	// Create provider connection
	config := map[string]interface{}{
		"region": account.Region,
	}
	provider, err := providers.NewProvider(account.Provider, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Validate credentials
	if err := provider.ValidateCredentials(ctx); err != nil {
		return nil, fmt.Errorf("failed to validate credentials: %w", err)
	}

	connection := &AccountConnection{
		AccountID:  account.ID,
		Provider:   account.Provider,
		Connection: provider,
		LastUsed:   time.Now(),
		Status:     "active",
		ErrorCount: 0,
		Metadata:   make(map[string]interface{}),
	}

	return connection, nil
}

// updateCostData updates cost data for an account
func (am *AccountManager) updateCostData(ctx context.Context, accountInfo *AccountInfo) error {
	// This is a simplified cost calculation
	// In a real implementation, you would use actual cost APIs

	totalCost := 0.0
	costByType := make(map[string]float64)
	costByRegion := make(map[string]float64)

	for _, resource := range accountInfo.Resources {
		// Estimate cost based on resource type
		cost := am.estimateResourceCost(resource)
		totalCost += cost

		// Group by type
		costByType[resource.Type] += cost

		// Group by region
		costByRegion[resource.Region] += cost
	}

	accountInfo.CostData = &AccountCostData{
		TotalCost:    totalCost,
		Currency:     "USD",
		CostByType:   costByType,
		CostByRegion: costByRegion,
		LastUpdated:  time.Now(),
		Trend:        "stable", // Would calculate actual trend
		BudgetUsage:  0.0,      // Would calculate based on tenant budget
	}

	return nil
}

// updateHealthStatus updates health status for an account
func (am *AccountManager) updateHealthStatus(ctx context.Context, accountInfo *AccountInfo) error {
	// This is a simplified health check
	// In a real implementation, you would perform actual health checks

	healthScore := 100.0
	issues := []AccountHealthIssue{}

	// Check for common issues
	for _, resource := range accountInfo.Resources {
		// Check for untagged resources
		if tags, ok := resource.Attributes["tags"].(map[string]interface{}); ok {
			if len(tags) == 0 {
				healthScore -= 5.0
				issues = append(issues, AccountHealthIssue{
					Type:        "untagged_resource",
					Severity:    "low",
					Description: fmt.Sprintf("Resource %s is not tagged", resource.ID),
					ResourceID:  resource.ID,
					DetectedAt:  time.Now(),
				})
			}
		}

		// Check for resources in wrong regions
		if resource.Region != accountInfo.Account.Region {
			healthScore -= 10.0
			issues = append(issues, AccountHealthIssue{
				Type:        "wrong_region",
				Severity:    "medium",
				Description: fmt.Sprintf("Resource %s is in region %s, expected %s", resource.ID, resource.Region, accountInfo.Account.Region),
				ResourceID:  resource.ID,
				DetectedAt:  time.Now(),
			})
		}
	}

	// Determine overall status
	overallStatus := "healthy"
	if healthScore < 70 {
		overallStatus = "warning"
	}
	if healthScore < 50 {
		overallStatus = "critical"
	}

	accountInfo.HealthStatus = &AccountHealthStatus{
		OverallStatus: overallStatus,
		HealthScore:   healthScore,
		Issues:        issues,
		LastChecked:   time.Now(),
		Metadata: map[string]interface{}{
			"total_resources": accountInfo.ResourceCount,
			"issue_count":     len(issues),
		},
	}

	return nil
}

// estimateResourceCost estimates the cost of a resource
func (am *AccountManager) estimateResourceCost(resource *models.Resource) float64 {
	// Simplified cost estimation
	switch resource.Type {
	case "aws_instance", "azurerm_virtual_machine", "google_compute_instance", "digitalocean_droplet":
		return 50.0
	case "aws_s3_bucket", "azurerm_storage_account", "google_storage_bucket", "digitalocean_spaces_bucket":
		return 20.0
	case "aws_db_instance", "azurerm_sql_database", "google_sql_database_instance", "digitalocean_database_cluster":
		return 100.0
	case "aws_lb", "azurerm_lb", "google_compute_backend_service", "digitalocean_loadbalancer":
		return 35.0
	default:
		return 10.0
	}
}
