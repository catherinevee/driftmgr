package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
)

// handleAccountsCommand handles listing and managing cloud accounts
func handleAccountsCommand(args []string) {
	var provider, format, output string
	var showDetails, testAccess bool
	format = "table" // default format

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider":
			if i+1 < len(args) {
				provider = args[i+1]
				i++
			}
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "--output":
			if i+1 < len(args) {
				output = args[i+1]
				i++
			}
		case "--details":
			showDetails = true
		case "--test-access":
			testAccess = true
		case "--help", "-h":
			showAccountsHelp()
			return
		}
	}

	// Collect accounts from all providers if none specified
	providers := []string{"aws", "azure", "gcp", "digitalocean"}
	if provider != "" {
		providers = []string{provider}
	}

	type AccountInfo struct {
		Provider    string                 `json:"provider"`
		AccountID   string                 `json:"account_id"`
		AccountName string                 `json:"account_name"`
		Status      string                 `json:"status"`
		Accessible  bool                   `json:"accessible"`
		Regions     []string               `json:"regions,omitempty"`
		Tags        map[string]string      `json:"tags,omitempty"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
		Error       string                 `json:"error,omitempty"`
	}

	var allAccounts []AccountInfo

	fmt.Println("Discovering cloud accounts...")
	fmt.Println()

	for _, p := range providers {
		// Try to create multi-account discoverer for this provider
		discoverer, err := discovery.NewMultiAccountDiscoverer(p)
		if err != nil {
			// Provider not configured
			if provider != "" {
				fmt.Fprintf(os.Stderr, "Error: %s provider not configured: %v\n", p, err)
			}
			continue
		}

		ctx := context.Background()
		accounts, _ := discoverer.GetAccounts(ctx)

		for _, account := range accounts {
			accountInfo := AccountInfo{
				Provider:    p,
				AccountID:   account.ID,
				AccountName: account.Name,
				Status:      "Active",
				Accessible:  true,
				Regions:     account.Regions,
				Tags:        make(map[string]string),
			}

			// Test access if requested
			if testAccess {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// Try to discover a minimal set of resources to test access
				_, err := discoverer.DiscoverAccountResources(ctx, account.ID)
				if err != nil {
					accountInfo.Accessible = false
					accountInfo.Error = err.Error()
					accountInfo.Status = "error"
				} else {
					accountInfo.Status = "active"
				}
			}

			allAccounts = append(allAccounts, accountInfo)
		}
	}

	if len(allAccounts) == 0 {
		fmt.Println("No cloud accounts found. Please configure your cloud credentials.")
		fmt.Println()
		fmt.Println("Configuration instructions:")
		fmt.Println("  AWS:          Set AWS_PROFILE or AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY")
		fmt.Println("  Azure:        Run 'az login' or set AZURE_SUBSCRIPTION_ID")
		fmt.Println("  GCP:          Run 'gcloud auth login' or set GCP_PROJECT_ID")
		fmt.Println("  DigitalOcean: Set DIGITALOCEAN_TOKEN environment variable")
		return
	}

	// Display or save results
	switch format {
	case "json":
		data, _ := json.MarshalIndent(allAccounts, "", "  ")
		if output != "" {
			if err := os.WriteFile(output, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Results saved to %s\n", output)
		} else {
			fmt.Println(string(data))
		}

	case "csv":
		// CSV output
		fmt.Println("Provider,Account ID,Account Name,Status,Accessible")
		for _, account := range allAccounts {
			fmt.Printf("%s,%s,%s,%s,%v\n",
				account.Provider,
				account.AccountID,
				account.AccountName,
				account.Status,
				account.Accessible)
		}

	default: // table format
		fmt.Println("Cloud Accounts")
		fmt.Println("=" + strings.Repeat("=", 70))
		fmt.Println()

		// Group by provider
		byProvider := make(map[string][]AccountInfo)
		for _, account := range allAccounts {
			byProvider[account.Provider] = append(byProvider[account.Provider], account)
		}

		for provider, accounts := range byProvider {
			fmt.Printf("%s ACCOUNTS (%d)\n", strings.ToUpper(provider), len(accounts))
			fmt.Println("-" + strings.Repeat("-", 70))

			for _, account := range accounts {
				statusIcon := "✓"
				if !account.Accessible || account.Status == "error" {
					statusIcon = "✗"
				} else if account.Status == "inactive" {
					statusIcon = "○"
				}

				fmt.Printf("%s %s (%s)\n", statusIcon, account.AccountName, account.AccountID)

				if showDetails {
					if account.Status != "" && account.Status != "active" {
						fmt.Printf("  Status: %s\n", account.Status)
					}
					if len(account.Regions) > 0 {
						fmt.Printf("  Regions: %s\n", strings.Join(account.Regions, ", "))
					}
					if len(account.Tags) > 0 {
						fmt.Printf("  Tags: ")
						for k, v := range account.Tags {
							fmt.Printf("%s=%s ", k, v)
						}
						fmt.Println()
					}
					if account.Error != "" {
						fmt.Printf("  Error: %s\n", account.Error)
					}
				}
			}
			fmt.Println()
		}

		// Summary
		fmt.Println("Summary")
		fmt.Println("-" + strings.Repeat("-", 70))
		fmt.Printf("Total Accounts: %d\n", len(allAccounts))

		activeCount := 0
		errorCount := 0
		for _, account := range allAccounts {
			if account.Accessible && account.Status != "error" {
				activeCount++
			}
			if account.Status == "error" || !account.Accessible {
				errorCount++
			}
		}

		fmt.Printf("Active: %d\n", activeCount)
		if errorCount > 0 {
			fmt.Printf("Errors: %d\n", errorCount)
		}

		if testAccess {
			fmt.Println()
			fmt.Println("Access test completed. Use --details to see any errors.")
		}
	}
}

// showAccountsHelp displays help for accounts command
func showAccountsHelp() {
	fmt.Println("Usage: driftmgr accounts [flags]")
	fmt.Println()
	fmt.Println("List all accessible cloud accounts/subscriptions/projects")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --provider string  Specific provider (aws, azure, gcp, digitalocean)")
	fmt.Println("  --format string    Output format: table, json, csv (default: table)")
	fmt.Println("  --output string    Output file path")
	fmt.Println("  --details          Show detailed account information")
	fmt.Println("  --test-access      Test access to each account")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # List all accounts")
	fmt.Println("  driftmgr accounts")
	fmt.Println()
	fmt.Println("  # List AWS accounts with details")
	fmt.Println("  driftmgr accounts --provider aws --details")
	fmt.Println()
	fmt.Println("  # Test access to all accounts")
	fmt.Println("  driftmgr accounts --test-access")
	fmt.Println()
	fmt.Println("  # Export accounts to JSON")
	fmt.Println("  driftmgr accounts --format json --output accounts.json")
}
