package tenant

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TenantManager manages multi-tenant operations
type TenantManager struct {
	tenants     map[string]*Tenant
	accounts    map[string]*Account
	permissions map[string]*Permission
	mu          sync.RWMutex
	eventBus    EventBus
	config      *TenantConfig
}

// Tenant represents a tenant in the system
type Tenant struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Settings    *TenantSettings        `json:"settings"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Account represents a cloud account within a tenant
type Account struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	Name        string                 `json:"name"`
	Provider    string                 `json:"provider"`
	Region      string                 `json:"region"`
	Credentials *AccountCredentials    `json:"credentials"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AccountCredentials represents encrypted credentials for an account
type AccountCredentials struct {
	AccessKey     string `json:"access_key,omitempty"`
	SecretKey     string `json:"secret_key,omitempty"`
	Token         string `json:"token,omitempty"`
	Region        string `json:"region,omitempty"`
	Encrypted     bool   `json:"encrypted"`
	EncryptionKey string `json:"encryption_key,omitempty"`
}

// Permission represents a permission for a tenant or account
type Permission struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	AccountID  string                 `json:"account_id,omitempty"`
	UserID     string                 `json:"user_id"`
	Role       string                 `json:"role"`
	Resources  []string               `json:"resources"`
	Actions    []string               `json:"actions"`
	Conditions []PermissionCondition  `json:"conditions"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// PermissionCondition represents a condition for a permission
type PermissionCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// TenantSettings represents tenant-specific settings
type TenantSettings struct {
	DefaultRegion        string                `json:"default_region"`
	AllowedRegions       []string              `json:"allowed_regions"`
	AllowedProviders     []string              `json:"allowed_providers"`
	ResourceLimits       *ResourceLimits       `json:"resource_limits"`
	SecuritySettings     *SecuritySettings     `json:"security_settings"`
	CostSettings         *CostSettings         `json:"cost_settings"`
	NotificationSettings *NotificationSettings `json:"notification_settings"`
}

// ResourceLimits represents resource limits for a tenant
type ResourceLimits struct {
	MaxInstances     int `json:"max_instances"`
	MaxStorage       int `json:"max_storage_gb"`
	MaxNetworks      int `json:"max_networks"`
	MaxDatabases     int `json:"max_databases"`
	MaxLoadBalancers int `json:"max_load_balancers"`
}

// SecuritySettings represents security settings for a tenant
type SecuritySettings struct {
	RequireMFA       bool     `json:"require_mfa"`
	SessionTimeout   int      `json:"session_timeout_minutes"`
	AllowedIPs       []string `json:"allowed_ips"`
	EncryptionAtRest bool     `json:"encryption_at_rest"`
	AuditLogging     bool     `json:"audit_logging"`
	ComplianceMode   string   `json:"compliance_mode"`
}

// CostSettings represents cost settings for a tenant
type CostSettings struct {
	BudgetLimit      float64 `json:"budget_limit"`
	AlertThreshold   float64 `json:"alert_threshold"`
	Currency         string  `json:"currency"`
	CostOptimization bool    `json:"cost_optimization"`
	AutoScaling      bool    `json:"auto_scaling"`
}

// NotificationSettings represents notification settings for a tenant
type NotificationSettings struct {
	EmailEnabled   bool     `json:"email_enabled"`
	SlackEnabled   bool     `json:"slack_enabled"`
	WebhookEnabled bool     `json:"webhook_enabled"`
	EmailAddresses []string `json:"email_addresses"`
	SlackChannels  []string `json:"slack_channels"`
	WebhookURLs    []string `json:"webhook_urls"`
}

// TenantConfig represents configuration for the tenant manager
type TenantConfig struct {
	DefaultTenantSettings *TenantSettings `json:"default_tenant_settings"`
	MaxTenants            int             `json:"max_tenants"`
	MaxAccountsPerTenant  int             `json:"max_accounts_per_tenant"`
	EncryptionEnabled     bool            `json:"encryption_enabled"`
	AuditLogging          bool            `json:"audit_logging"`
}

// EventBus interface for tenant events
type EventBus interface {
	PublishTenantEvent(event TenantEvent) error
}

// TenantEvent represents a tenant-related event
type TenantEvent struct {
	Type      string                 `json:"type"`
	TenantID  string                 `json:"tenant_id"`
	AccountID string                 `json:"account_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewTenantManager creates a new tenant manager
func NewTenantManager(eventBus EventBus) *TenantManager {
	config := &TenantConfig{
		DefaultTenantSettings: &TenantSettings{
			DefaultRegion:    "us-east-1",
			AllowedRegions:   []string{"us-east-1", "us-west-2", "eu-west-1"},
			AllowedProviders: []string{"aws", "azure", "gcp", "digitalocean"},
			ResourceLimits: &ResourceLimits{
				MaxInstances:     100,
				MaxStorage:       1000,
				MaxNetworks:      10,
				MaxDatabases:     20,
				MaxLoadBalancers: 5,
			},
			SecuritySettings: &SecuritySettings{
				RequireMFA:       true,
				SessionTimeout:   60,
				EncryptionAtRest: true,
				AuditLogging:     true,
				ComplianceMode:   "standard",
			},
			CostSettings: &CostSettings{
				BudgetLimit:      10000.0,
				AlertThreshold:   0.8,
				Currency:         "USD",
				CostOptimization: true,
				AutoScaling:      true,
			},
			NotificationSettings: &NotificationSettings{
				EmailEnabled:   true,
				SlackEnabled:   false,
				WebhookEnabled: false,
			},
		},
		MaxTenants:           100,
		MaxAccountsPerTenant: 10,
		EncryptionEnabled:    true,
		AuditLogging:         true,
	}

	return &TenantManager{
		tenants:     make(map[string]*Tenant),
		accounts:    make(map[string]*Account),
		permissions: make(map[string]*Permission),
		eventBus:    eventBus,
		config:      config,
	}
}

// CreateTenant creates a new tenant
func (tm *TenantManager) CreateTenant(ctx context.Context, tenant *Tenant) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check tenant limit
	if len(tm.tenants) >= tm.config.MaxTenants {
		return fmt.Errorf("maximum number of tenants reached (%d)", tm.config.MaxTenants)
	}

	// Validate tenant
	if err := tm.validateTenant(tenant); err != nil {
		return fmt.Errorf("invalid tenant: %w", err)
	}

	// Set defaults
	if tenant.ID == "" {
		tenant.ID = fmt.Sprintf("tenant_%d", time.Now().Unix())
	}
	if tenant.Settings == nil {
		tenant.Settings = tm.config.DefaultTenantSettings
	}
	tenant.Status = "active"
	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = time.Now()

	// Store tenant
	tm.tenants[tenant.ID] = tenant

	// Publish event
	if tm.eventBus != nil {
		event := TenantEvent{
			Type:      "tenant_created",
			TenantID:  tenant.ID,
			Message:   fmt.Sprintf("Tenant '%s' created", tenant.Name),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"tenant_name": tenant.Name,
				"status":      tenant.Status,
			},
		}
		tm.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// GetTenant retrieves a tenant by ID
func (tm *TenantManager) GetTenant(ctx context.Context, tenantID string) (*Tenant, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tenant, exists := tm.tenants[tenantID]
	if !exists {
		return nil, fmt.Errorf("tenant %s not found", tenantID)
	}

	return tenant, nil
}

// UpdateTenant updates an existing tenant
func (tm *TenantManager) UpdateTenant(ctx context.Context, tenantID string, updates *Tenant) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tenant, exists := tm.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	// Update fields
	if updates.Name != "" {
		tenant.Name = updates.Name
	}
	if updates.Description != "" {
		tenant.Description = updates.Description
	}
	if updates.Status != "" {
		tenant.Status = updates.Status
	}
	if updates.Settings != nil {
		tenant.Settings = updates.Settings
	}
	tenant.UpdatedAt = time.Now()

	// Publish event
	if tm.eventBus != nil {
		event := TenantEvent{
			Type:      "tenant_updated",
			TenantID:  tenant.ID,
			Message:   fmt.Sprintf("Tenant '%s' updated", tenant.Name),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"tenant_name": tenant.Name,
				"status":      tenant.Status,
			},
		}
		tm.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// DeleteTenant deletes a tenant
func (tm *TenantManager) DeleteTenant(ctx context.Context, tenantID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tenant, exists := tm.tenants[tenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", tenantID)
	}

	// Check if tenant has accounts
	hasAccounts := false
	for _, account := range tm.accounts {
		if account.TenantID == tenantID {
			hasAccounts = true
			break
		}
	}

	if hasAccounts {
		return fmt.Errorf("cannot delete tenant with existing accounts")
	}

	// Delete tenant
	delete(tm.tenants, tenantID)

	// Publish event
	if tm.eventBus != nil {
		event := TenantEvent{
			Type:      "tenant_deleted",
			TenantID:  tenantID,
			Message:   fmt.Sprintf("Tenant '%s' deleted", tenant.Name),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"tenant_name": tenant.Name,
			},
		}
		tm.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// ListTenants returns all tenants
func (tm *TenantManager) ListTenants(ctx context.Context) ([]*Tenant, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tenants := make([]*Tenant, 0, len(tm.tenants))
	for _, tenant := range tm.tenants {
		tenants = append(tenants, tenant)
	}

	return tenants, nil
}

// CreateAccount creates a new account for a tenant
func (tm *TenantManager) CreateAccount(ctx context.Context, account *Account) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if tenant exists
	tenant, exists := tm.tenants[account.TenantID]
	if !exists {
		return fmt.Errorf("tenant %s not found", account.TenantID)
	}

	// Check account limit for tenant
	accountCount := 0
	for _, acc := range tm.accounts {
		if acc.TenantID == account.TenantID {
			accountCount++
		}
	}

	if accountCount >= tm.config.MaxAccountsPerTenant {
		return fmt.Errorf("maximum number of accounts reached for tenant %s (%d)", account.TenantID, tm.config.MaxAccountsPerTenant)
	}

	// Validate account
	if err := tm.validateAccount(account, tenant); err != nil {
		return fmt.Errorf("invalid account: %w", err)
	}

	// Set defaults
	if account.ID == "" {
		account.ID = fmt.Sprintf("account_%d", time.Now().Unix())
	}
	account.Status = "active"
	account.CreatedAt = time.Now()
	account.UpdatedAt = time.Now()

	// Encrypt credentials if enabled
	if tm.config.EncryptionEnabled && account.Credentials != nil {
		if err := tm.encryptCredentials(account.Credentials); err != nil {
			return fmt.Errorf("failed to encrypt credentials: %w", err)
		}
	}

	// Store account
	tm.accounts[account.ID] = account

	// Publish event
	if tm.eventBus != nil {
		event := TenantEvent{
			Type:      "account_created",
			TenantID:  account.TenantID,
			AccountID: account.ID,
			Message:   fmt.Sprintf("Account '%s' created for tenant '%s'", account.Name, tenant.Name),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"account_name": account.Name,
				"provider":     account.Provider,
				"region":       account.Region,
			},
		}
		tm.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// GetAccount retrieves an account by ID
func (tm *TenantManager) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	account, exists := tm.accounts[accountID]
	if !exists {
		return nil, fmt.Errorf("account %s not found", accountID)
	}

	// Decrypt credentials if needed
	if tm.config.EncryptionEnabled && account.Credentials != nil && account.Credentials.Encrypted {
		decryptedAccount := *account
		if err := tm.decryptCredentials(decryptedAccount.Credentials); err != nil {
			return nil, fmt.Errorf("failed to decrypt credentials: %w", err)
		}
		return &decryptedAccount, nil
	}

	return account, nil
}

// ListAccounts returns all accounts for a tenant
func (tm *TenantManager) ListAccounts(ctx context.Context, tenantID string) ([]*Account, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var accounts []*Account
	for _, account := range tm.accounts {
		if account.TenantID == tenantID {
			accounts = append(accounts, account)
		}
	}

	return accounts, nil
}

// CreatePermission creates a new permission
func (tm *TenantManager) CreatePermission(ctx context.Context, permission *Permission) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Validate permission
	if err := tm.validatePermission(permission); err != nil {
		return fmt.Errorf("invalid permission: %w", err)
	}

	// Set defaults
	if permission.ID == "" {
		permission.ID = fmt.Sprintf("permission_%d", time.Now().Unix())
	}
	permission.CreatedAt = time.Now()
	permission.UpdatedAt = time.Now()

	// Store permission
	tm.permissions[permission.ID] = permission

	// Publish event
	if tm.eventBus != nil {
		event := TenantEvent{
			Type:      "permission_created",
			TenantID:  permission.TenantID,
			AccountID: permission.AccountID,
			UserID:    permission.UserID,
			Message:   fmt.Sprintf("Permission created for user %s", permission.UserID),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"role":      permission.Role,
				"resources": permission.Resources,
				"actions":   permission.Actions,
			},
		}
		tm.eventBus.PublishTenantEvent(event)
	}

	return nil
}

// CheckPermission checks if a user has permission to perform an action
func (tm *TenantManager) CheckPermission(ctx context.Context, userID, tenantID, accountID, resource, action string) (bool, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	for _, permission := range tm.permissions {
		if permission.UserID == userID && permission.TenantID == tenantID {
			// Check if permission applies to this account (if specified)
			if permission.AccountID != "" && permission.AccountID != accountID {
				continue
			}

			// Check if permission applies to this resource
			if !tm.resourceMatches(permission.Resources, resource) {
				continue
			}

			// Check if permission allows this action
			if !tm.actionMatches(permission.Actions, action) {
				continue
			}

			// Check if permission is expired
			if permission.ExpiresAt != nil && permission.ExpiresAt.Before(time.Now()) {
				continue
			}

			// Check conditions
			if tm.conditionsMatch(permission.Conditions) {
				return true, nil
			}
		}
	}

	return false, nil
}

// Helper methods

// validateTenant validates a tenant
func (tm *TenantManager) validateTenant(tenant *Tenant) error {
	if tenant.Name == "" {
		return fmt.Errorf("tenant name is required")
	}
	return nil
}

// validateAccount validates an account
func (tm *TenantManager) validateAccount(account *Account, tenant *Tenant) error {
	if account.Name == "" {
		return fmt.Errorf("account name is required")
	}
	if account.Provider == "" {
		return fmt.Errorf("account provider is required")
	}
	if account.Region == "" {
		return fmt.Errorf("account region is required")
	}

	// Check if provider is allowed for tenant
	allowed := false
	for _, provider := range tenant.Settings.AllowedProviders {
		if provider == account.Provider {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("provider %s is not allowed for tenant", account.Provider)
	}

	// Check if region is allowed for tenant
	allowed = false
	for _, region := range tenant.Settings.AllowedRegions {
		if region == account.Region {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("region %s is not allowed for tenant", account.Region)
	}

	return nil
}

// validatePermission validates a permission
func (tm *TenantManager) validatePermission(permission *Permission) error {
	if permission.TenantID == "" {
		return fmt.Errorf("tenant ID is required")
	}
	if permission.UserID == "" {
		return fmt.Errorf("user ID is required")
	}
	if permission.Role == "" {
		return fmt.Errorf("role is required")
	}
	return nil
}

// resourceMatches checks if a resource matches the permission resources
func (tm *TenantManager) resourceMatches(permissionResources []string, resource string) bool {
	if len(permissionResources) == 0 {
		return true // No resource restrictions
	}

	for _, permResource := range permissionResources {
		if permResource == "*" || permResource == resource {
			return true
		}
	}
	return false
}

// actionMatches checks if an action matches the permission actions
func (tm *TenantManager) actionMatches(permissionActions []string, action string) bool {
	if len(permissionActions) == 0 {
		return true // No action restrictions
	}

	for _, permAction := range permissionActions {
		if permAction == "*" || permAction == action {
			return true
		}
	}
	return false
}

// conditionsMatch checks if permission conditions are met
func (tm *TenantManager) conditionsMatch(conditions []PermissionCondition) bool {
	// For now, always return true
	// In a real implementation, you would evaluate the conditions
	return true
}

// encryptCredentials encrypts account credentials
func (tm *TenantManager) encryptCredentials(credentials *AccountCredentials) error {
	// This is a placeholder - in a real implementation, you would use proper encryption
	credentials.Encrypted = true
	credentials.EncryptionKey = "encrypted_key_placeholder"
	return nil
}

// decryptCredentials decrypts account credentials
func (tm *TenantManager) decryptCredentials(credentials *AccountCredentials) error {
	// This is a placeholder - in a real implementation, you would use proper decryption
	credentials.Encrypted = false
	credentials.EncryptionKey = ""
	return nil
}

// SetConfig updates the tenant manager configuration
func (tm *TenantManager) SetConfig(config *TenantConfig) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.config = config
}

// GetConfig returns the current tenant manager configuration
func (tm *TenantManager) GetConfig() *TenantConfig {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.config
}
