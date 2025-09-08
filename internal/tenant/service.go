package tenant

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// TenantService provides a unified interface for tenant management
type TenantService struct {
	tenantManager    *TenantManager
	accountManager   *AccountManager
	isolationManager *ResourceIsolationManager
	mu               sync.RWMutex
	eventBus         EventBus
	config           *ServiceConfig
}

// ServiceConfig represents configuration for the tenant service
type ServiceConfig struct {
	AutoSyncInterval    time.Duration `json:"auto_sync_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	IsolationEnabled    bool          `json:"isolation_enabled"`
	AuditLogging        bool          `json:"audit_logging"`
	MaxConcurrentSyncs  int           `json:"max_concurrent_syncs"`
}

// TenantSummary represents a summary of tenant information
type TenantSummary struct {
	Tenant          *Tenant                `json:"tenant"`
	AccountCount    int                    `json:"account_count"`
	ResourceCount   int                    `json:"resource_count"`
	TotalCost       float64                `json:"total_cost"`
	HealthStatus    string                 `json:"health_status"`
	LastSync        time.Time              `json:"last_sync"`
	IsolationStatus string                 `json:"isolation_status"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// AccountSummary represents a summary of account information
type AccountSummary struct {
	Account          *Account               `json:"account"`
	ResourceCount    int                    `json:"resource_count"`
	TotalCost        float64                `json:"total_cost"`
	HealthStatus     string                 `json:"health_status"`
	LastSync         time.Time              `json:"last_sync"`
	ConnectionStatus string                 `json:"connection_status"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// NewTenantService creates a new tenant service
func NewTenantService(eventBus EventBus) *TenantService {
	config := &ServiceConfig{
		AutoSyncInterval:    30 * time.Minute,
		HealthCheckInterval: 5 * time.Minute,
		IsolationEnabled:    true,
		AuditLogging:        true,
		MaxConcurrentSyncs:  5,
	}

	// Create managers
	tenantManager := NewTenantManager(eventBus)
	accountManager := NewAccountManager(tenantManager, eventBus)
	isolationManager := NewResourceIsolationManager(eventBus)

	return &TenantService{
		tenantManager:    tenantManager,
		accountManager:   accountManager,
		isolationManager: isolationManager,
		eventBus:         eventBus,
		config:           config,
	}
}

// Start starts the tenant service
func (ts *TenantService) Start(ctx context.Context) error {
	// Start auto-sync
	go ts.autoSync(ctx)

	// Start health checks
	go ts.healthCheck(ctx)

	// Create default isolation rules
	if ts.config.IsolationEnabled {
		if err := ts.isolationManager.CreateDefaultIsolationRules(ctx); err != nil {
			return fmt.Errorf("failed to create default isolation rules: %w", err)
		}
	}

	// Publish event
	if ts.eventBus != nil {
		event := TenantEvent{
			Type:      "service_started",
			Message:   "Tenant service started",
			Timestamp: time.Now(),
		}
		ts.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// Stop stops the tenant service
func (ts *TenantService) Stop(ctx context.Context) error {
	// Publish event
	if ts.eventBus != nil {
		event := TenantEvent{
			Type:      "service_stopped",
			Message:   "Tenant service stopped",
			Timestamp: time.Now(),
		}
		ts.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// CreateTenant creates a new tenant with default settings
func (ts *TenantService) CreateTenant(ctx context.Context, tenant *Tenant) error {
	// Create tenant
	if err := ts.tenantManager.CreateTenant(ctx, tenant); err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	// Create tenant-specific isolation rules
	if ts.config.IsolationEnabled {
		rule := &IsolationRule{
			TenantID:    tenant.ID,
			Name:        fmt.Sprintf("Tenant %s Isolation", tenant.Name),
			Description: fmt.Sprintf("Isolation rules for tenant %s", tenant.Name),
			Type:        "tenant_specific",
			Conditions: []IsolationCondition{
				{
					Field:    "tenant_id",
					Operator: "equals",
					Value:    tenant.ID,
					Type:     "string",
				},
			},
			Actions: []IsolationAction{
				{
					Type:        "enforce_tenant_isolation",
					Description: "Enforce isolation for tenant resources",
					Parameters: map[string]interface{}{
						"tenant_id": tenant.ID,
					},
				},
			},
			Priority: 100,
			Enabled:  true,
		}

		if err := ts.isolationManager.CreateIsolationRule(ctx, rule); err != nil {
			// Log error but don't fail tenant creation
			fmt.Printf("Warning: failed to create isolation rule for tenant %s: %v\n", tenant.ID, err)
		}
	}

	return nil
}

// AddAccount adds an account to a tenant
func (ts *TenantService) AddAccount(ctx context.Context, tenantID string, account *Account) error {
	// Set tenant ID
	account.TenantID = tenantID

	// Create account in tenant manager
	if err := ts.tenantManager.CreateAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	// Register account in account manager
	if err := ts.accountManager.RegisterAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to register account: %w", err)
	}

	// Tag account resources for isolation
	if ts.config.IsolationEnabled {
		if err := ts.isolationManager.TagResource(ctx, account.ID, tenantID, account.ID, map[string]string{
			"tenant_id":  tenantID,
			"account_id": account.ID,
			"provider":   account.Provider,
			"region":     account.Region,
		}); err != nil {
			// Log error but don't fail account creation
			fmt.Printf("Warning: failed to tag account %s: %v\n", account.ID, err)
		}
	}

	return nil
}

// SyncTenant synchronizes all accounts for a tenant
func (ts *TenantService) SyncTenant(ctx context.Context, tenantID string) error {
	// Get tenant accounts
	accounts, err := ts.tenantManager.ListAccounts(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to list accounts: %w", err)
	}

	// Sync each account
	var errors []error
	for _, account := range accounts {
		if err := ts.accountManager.SyncAccount(ctx, account.ID); err != nil {
			errors = append(errors, fmt.Errorf("failed to sync account %s: %w", account.ID, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("sync completed with errors: %v", errors)
	}

	return nil
}

// GetTenantSummary retrieves a summary of tenant information
func (ts *TenantService) GetTenantSummary(ctx context.Context, tenantID string) (*TenantSummary, error) {
	// Get tenant
	tenant, err := ts.tenantManager.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	// Get accounts
	accounts, err := ts.tenantManager.ListAccounts(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}

	// Get account summaries
	accountSummaries, err := ts.accountManager.ListAccountsByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account summaries: %w", err)
	}

	// Calculate totals
	accountCount := len(accounts)
	resourceCount := 0
	totalCost := 0.0
	healthStatus := "healthy"
	lastSync := time.Time{}

	for _, accountSummary := range accountSummaries {
		resourceCount += accountSummary.ResourceCount
		totalCost += accountSummary.CostData.TotalCost

		// Update health status
		if accountSummary.HealthStatus.OverallStatus == "critical" {
			healthStatus = "critical"
		} else if accountSummary.HealthStatus.OverallStatus == "warning" && healthStatus != "critical" {
			healthStatus = "warning"
		}

		// Update last sync
		if accountSummary.LastSync.After(lastSync) {
			lastSync = accountSummary.LastSync
		}
	}

	// Determine isolation status
	isolationStatus := "enforced"
	if !ts.config.IsolationEnabled {
		isolationStatus = "disabled"
	}

	summary := &TenantSummary{
		Tenant:          tenant,
		AccountCount:    accountCount,
		ResourceCount:   resourceCount,
		TotalCost:       totalCost,
		HealthStatus:    healthStatus,
		LastSync:        lastSync,
		IsolationStatus: isolationStatus,
		Metadata: map[string]interface{}{
			"total_accounts":  accountCount,
			"total_resources": resourceCount,
			"total_cost":      totalCost,
		},
	}

	return summary, nil
}

// GetAccountSummary retrieves a summary of account information
func (ts *TenantService) GetAccountSummary(ctx context.Context, accountID string) (*AccountSummary, error) {
	// Get account
	account, err := ts.tenantManager.GetAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Get account info
	accountInfo, err := ts.accountManager.GetAccountInfo(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	// Get connection status
	connection, err := ts.accountManager.GetAccountConnection(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	summary := &AccountSummary{
		Account:          account,
		ResourceCount:    accountInfo.ResourceCount,
		TotalCost:        accountInfo.CostData.TotalCost,
		HealthStatus:     accountInfo.HealthStatus.OverallStatus,
		LastSync:         accountInfo.LastSync,
		ConnectionStatus: connection.Status,
		Metadata: map[string]interface{}{
			"provider":     account.Provider,
			"region":       account.Region,
			"sync_status":  accountInfo.SyncStatus,
			"health_score": accountInfo.HealthStatus.HealthScore,
			"issue_count":  len(accountInfo.HealthStatus.Issues),
		},
	}

	return summary, nil
}

// ListTenants lists all tenants with summaries
func (ts *TenantService) ListTenants(ctx context.Context) ([]*TenantSummary, error) {
	// Get all tenants
	tenants, err := ts.tenantManager.ListTenants(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}

	// Get summaries for each tenant
	var summaries []*TenantSummary
	for _, tenant := range tenants {
		summary, err := ts.GetTenantSummary(ctx, tenant.ID)
		if err != nil {
			// Log error but continue with other tenants
			fmt.Printf("Warning: failed to get summary for tenant %s: %v\n", tenant.ID, err)
			continue
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// CheckResourceAccess checks if a user can access a resource
func (ts *TenantService) CheckResourceAccess(ctx context.Context, userID, tenantID, accountID, resourceID, action string) (bool, error) {
	// Check permission
	hasPermission, err := ts.tenantManager.CheckPermission(ctx, userID, tenantID, accountID, resourceID, action)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	if !hasPermission {
		return false, nil
	}

	// Check isolation if enabled
	if ts.config.IsolationEnabled {
		// Get resource
		accountInfo, err := ts.accountManager.GetAccountInfo(ctx, accountID)
		if err != nil {
			return false, fmt.Errorf("failed to get account info: %w", err)
		}

		var resource *models.Resource
		for _, res := range accountInfo.Resources {
			if res.ID == resourceID {
				resource = res
				break
			}
		}

		if resource == nil {
			return false, fmt.Errorf("resource %s not found", resourceID)
		}

		// Check isolation
		violations, err := ts.isolationManager.CheckIsolation(ctx, resource, tenantID)
		if err != nil {
			return false, fmt.Errorf("failed to check isolation: %w", err)
		}

		if len(violations) > 0 {
			return false, nil
		}
	}

	return true, nil
}

// Helper methods

// autoSync performs automatic synchronization of accounts
func (ts *TenantService) autoSync(ctx context.Context) {
	ticker := time.NewTicker(ts.config.AutoSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Sync all accounts
			if err := ts.accountManager.SyncAllAccounts(ctx); err != nil {
				fmt.Printf("Warning: auto-sync failed: %v\n", err)
			}
		}
	}
}

// healthCheck performs health checks on accounts
func (ts *TenantService) healthCheck(ctx context.Context) {
	ticker := time.NewTicker(ts.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Perform health checks
			ts.performHealthChecks(ctx)
		}
	}
}

// performHealthChecks performs health checks on all accounts
func (ts *TenantService) performHealthChecks(ctx context.Context) {
	// Get all tenants
	tenants, err := ts.tenantManager.ListTenants(ctx)
	if err != nil {
		fmt.Printf("Warning: failed to list tenants for health check: %v\n", err)
		return
	}

	// Check each tenant
	for _, tenant := range tenants {
		// Get accounts
		accounts, err := ts.tenantManager.ListAccounts(ctx, tenant.ID)
		if err != nil {
			fmt.Printf("Warning: failed to list accounts for tenant %s: %v\n", tenant.ID, err)
			continue
		}

		// Check each account
		for _, account := range accounts {
			// Get account info
			accountInfo, err := ts.accountManager.GetAccountInfo(ctx, account.ID)
			if err != nil {
				fmt.Printf("Warning: failed to get account info for %s: %v\n", account.ID, err)
				continue
			}

			// Check health status
			if accountInfo.HealthStatus.OverallStatus == "critical" {
				// Publish alert
				if ts.eventBus != nil {
					event := TenantEvent{
						Type:      "health_alert",
						TenantID:  tenant.ID,
						AccountID: account.ID,
						Message:   fmt.Sprintf("Critical health status for account %s", account.Name),
						Timestamp: time.Now(),
						Metadata: map[string]interface{}{
							"health_status": accountInfo.HealthStatus.OverallStatus,
							"health_score":  accountInfo.HealthStatus.HealthScore,
							"issue_count":   len(accountInfo.HealthStatus.Issues),
						},
					}
					ts.eventBus.PublishTenantEvent(event)
				}
			}
		}
	}
}

// SetConfig updates the service configuration
func (ts *TenantService) SetConfig(config *ServiceConfig) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.config = config
}

// GetConfig returns the current service configuration
func (ts *TenantService) GetConfig() *ServiceConfig {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.config
}
