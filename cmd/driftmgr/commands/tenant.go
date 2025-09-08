package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/catherinevee/driftmgr/internal/tenant"
)

// TenantCommand represents the tenant management command
type TenantCommand struct {
	service *tenant.TenantService
}

// NewTenantCommand creates a new tenant command
func NewTenantCommand() *TenantCommand {
	// Create a mock event bus for demonstration
	eventBus := &MockEventBus{}

	// Create tenant service
	service := tenant.NewTenantService(eventBus)

	return &TenantCommand{
		service: service,
	}
}

// MockEventBus is a mock implementation of the EventBus interface
type MockEventBus struct{}

func (m *MockEventBus) PublishTenantEvent(event tenant.TenantEvent) error {
	fmt.Printf("Event: %s - %s\n", event.Type, event.Message)
	return nil
}

// HandleTenant handles the tenant command
func HandleTenant(args []string) {
	cmd := NewTenantCommand()

	// Start the service
	ctx := context.Background()
	if err := cmd.service.Start(ctx); err != nil {
		fmt.Printf("Error starting tenant service: %v\n", err)
		return
	}
	defer cmd.service.Stop(ctx)

	if len(args) == 0 {
		cmd.showHelp()
		return
	}

	switch args[0] {
	case "create":
		cmd.handleCreateTenant(args[1:])
	case "list":
		cmd.handleListTenants(args[1:])
	case "add-account":
		cmd.handleAddAccount(args[1:])
	case "sync":
		cmd.handleSyncTenant(args[1:])
	case "summary":
		cmd.handleTenantSummary(args[1:])
	case "account-summary":
		cmd.handleAccountSummary(args[1:])
	default:
		fmt.Printf("Unknown tenant command: %s\n", args[0])
		cmd.showHelp()
	}
}

// showHelp shows the help for tenant commands
func (cmd *TenantCommand) showHelp() {
	fmt.Println("Tenant Management Commands:")
	fmt.Println("  create <name> <description>     - Create a new tenant")
	fmt.Println("  list                           - List all tenants")
	fmt.Println("  add-account <tenant-id> <name> <provider> <region> - Add account to tenant")
	fmt.Println("  sync <tenant-id>               - Sync tenant accounts")
	fmt.Println("  summary <tenant-id>            - Show tenant summary")
	fmt.Println("  account-summary <account-id>   - Show account summary")
}

// handleCreateTenant handles tenant creation
func (cmd *TenantCommand) handleCreateTenant(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: tenant create <name> <description>")
		return
	}

	name := args[0]
	description := args[1]

	tenant := &tenant.Tenant{
		Name:        name,
		Description: description,
	}

	ctx := context.Background()
	if err := cmd.service.CreateTenant(ctx, tenant); err != nil {
		fmt.Printf("Error creating tenant: %v\n", err)
		return
	}

	fmt.Printf("Tenant '%s' created successfully with ID: %s\n", name, tenant.ID)
}

// handleListTenants handles listing tenants
func (cmd *TenantCommand) handleListTenants(args []string) {
	ctx := context.Background()
	summaries, err := cmd.service.ListTenants(ctx)
	if err != nil {
		fmt.Printf("Error listing tenants: %v\n", err)
		return
	}

	if len(summaries) == 0 {
		fmt.Println("No tenants found.")
		return
	}

	// Create tabwriter for formatted output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tAccounts\tResources\tCost\tHealth\tLast Sync")
	fmt.Fprintln(w, "---\t----\t--------\t---------\t----\t------\t---------")

	for _, summary := range summaries {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t$%.2f\t%s\t%s\n",
			summary.Tenant.ID,
			summary.Tenant.Name,
			summary.AccountCount,
			summary.ResourceCount,
			summary.TotalCost,
			summary.HealthStatus,
			summary.LastSync.Format("2006-01-02 15:04:05"),
		)
	}

	w.Flush()
}

// handleAddAccount handles adding an account to a tenant
func (cmd *TenantCommand) handleAddAccount(args []string) {
	if len(args) < 4 {
		fmt.Println("Usage: tenant add-account <tenant-id> <name> <provider> <region>")
		return
	}

	tenantID := args[0]
	name := args[1]
	provider := args[2]
	region := args[3]

	account := &tenant.Account{
		Name:     name,
		Provider: provider,
		Region:   region,
		Credentials: &tenant.AccountCredentials{
			AccessKey: "mock_access_key",
			SecretKey: "mock_secret_key",
			Region:    region,
		},
	}

	ctx := context.Background()
	if err := cmd.service.AddAccount(ctx, tenantID, account); err != nil {
		fmt.Printf("Error adding account: %v\n", err)
		return
	}

	fmt.Printf("Account '%s' added successfully to tenant %s with ID: %s\n", name, tenantID, account.ID)
}

// handleSyncTenant handles syncing a tenant
func (cmd *TenantCommand) handleSyncTenant(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: tenant sync <tenant-id>")
		return
	}

	tenantID := args[0]

	ctx := context.Background()
	if err := cmd.service.SyncTenant(ctx, tenantID); err != nil {
		fmt.Printf("Error syncing tenant: %v\n", err)
		return
	}

	fmt.Printf("Tenant %s synchronized successfully\n", tenantID)
}

// handleTenantSummary handles showing tenant summary
func (cmd *TenantCommand) handleTenantSummary(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: tenant summary <tenant-id>")
		return
	}

	tenantID := args[0]

	ctx := context.Background()
	summary, err := cmd.service.GetTenantSummary(ctx, tenantID)
	if err != nil {
		fmt.Printf("Error getting tenant summary: %v\n", err)
		return
	}

	fmt.Printf("Tenant Summary for '%s'\n", summary.Tenant.Name)
	fmt.Printf("ID: %s\n", summary.Tenant.ID)
	fmt.Printf("Description: %s\n", summary.Tenant.Description)
	fmt.Printf("Status: %s\n", summary.Tenant.Status)
	fmt.Printf("Accounts: %d\n", summary.AccountCount)
	fmt.Printf("Resources: %d\n", summary.ResourceCount)
	fmt.Printf("Total Cost: $%.2f\n", summary.TotalCost)
	fmt.Printf("Health Status: %s\n", summary.HealthStatus)
	fmt.Printf("Isolation Status: %s\n", summary.IsolationStatus)
	fmt.Printf("Last Sync: %s\n", summary.LastSync.Format("2006-01-02 15:04:05"))

	if summary.Tenant.Settings != nil {
		fmt.Printf("\nSettings:\n")
		fmt.Printf("  Default Region: %s\n", summary.Tenant.Settings.DefaultRegion)
		fmt.Printf("  Allowed Regions: %v\n", summary.Tenant.Settings.AllowedRegions)
		fmt.Printf("  Allowed Providers: %v\n", summary.Tenant.Settings.AllowedProviders)

		if summary.Tenant.Settings.ResourceLimits != nil {
			fmt.Printf("  Resource Limits:\n")
			fmt.Printf("    Max Instances: %d\n", summary.Tenant.Settings.ResourceLimits.MaxInstances)
			fmt.Printf("    Max Storage: %d GB\n", summary.Tenant.Settings.ResourceLimits.MaxStorage)
			fmt.Printf("    Max Networks: %d\n", summary.Tenant.Settings.ResourceLimits.MaxNetworks)
			fmt.Printf("    Max Databases: %d\n", summary.Tenant.Settings.ResourceLimits.MaxDatabases)
			fmt.Printf("    Max Load Balancers: %d\n", summary.Tenant.Settings.ResourceLimits.MaxLoadBalancers)
		}

		if summary.Tenant.Settings.CostSettings != nil {
			fmt.Printf("  Cost Settings:\n")
			fmt.Printf("    Budget Limit: $%.2f\n", summary.Tenant.Settings.CostSettings.BudgetLimit)
			fmt.Printf("    Alert Threshold: %.1f%%\n", summary.Tenant.Settings.CostSettings.AlertThreshold*100)
			fmt.Printf("    Currency: %s\n", summary.Tenant.Settings.CostSettings.Currency)
		}
	}
}

// handleAccountSummary handles showing account summary
func (cmd *TenantCommand) handleAccountSummary(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: tenant account-summary <account-id>")
		return
	}

	accountID := args[0]

	ctx := context.Background()
	summary, err := cmd.service.GetAccountSummary(ctx, accountID)
	if err != nil {
		fmt.Printf("Error getting account summary: %v\n", err)
		return
	}

	fmt.Printf("Account Summary for '%s'\n", summary.Account.Name)
	fmt.Printf("ID: %s\n", summary.Account.ID)
	fmt.Printf("Tenant ID: %s\n", summary.Account.TenantID)
	fmt.Printf("Provider: %s\n", summary.Account.Provider)
	fmt.Printf("Region: %s\n", summary.Account.Region)
	fmt.Printf("Status: %s\n", summary.Account.Status)
	fmt.Printf("Resources: %d\n", summary.ResourceCount)
	fmt.Printf("Total Cost: $%.2f\n", summary.TotalCost)
	fmt.Printf("Health Status: %s\n", summary.HealthStatus)
	fmt.Printf("Connection Status: %s\n", summary.ConnectionStatus)
	fmt.Printf("Last Sync: %s\n", summary.LastSync.Format("2006-01-02 15:04:05"))

	if summary.Metadata != nil {
		fmt.Printf("\nMetadata:\n")
		for key, value := range summary.Metadata {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
}
