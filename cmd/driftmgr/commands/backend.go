package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	// "github.com/catherinevee/driftmgr/internal/discovery/backend/discovery" // Not yet implemented
	"github.com/catherinevee/driftmgr/internal/discovery/backend"
	// "github.com/catherinevee/driftmgr/internal/discovery/backend/providers/s3" // Not yet implemented
	// "github.com/catherinevee/driftmgr/internal/discovery/backend/providers/azure" // Not yet implemented
	// "github.com/catherinevee/driftmgr/internal/discovery/backend/providers/gcs" // Not yet implemented
)

var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "Manage Terraform backend configurations",
	Long:  `Discover, validate, and manage Terraform backend configurations across multiple providers.`,
}

var discoverBackendsCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover Terraform backends in the repository",
	RunE:  runDiscoverBackends,
}

var validateBackendCmd = &cobra.Command{
	Use:   "validate [backend-id]",
	Short: "Validate backend configuration and connectivity",
	Args:  cobra.ExactArgs(1),
	RunE:  runValidateBackend,
}

var listBackendsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all registered backends",
	RunE:  runListBackends,
}

var (
	backendPath      string
	backendRecursive bool
	backendProvider  string
	backendRegion    string
	backendTimeout   time.Duration
)

func init() {
	backendCmd.AddCommand(discoverBackendsCmd)
	backendCmd.AddCommand(validateBackendCmd)
	backendCmd.AddCommand(listBackendsCmd)

	discoverBackendsCmd.Flags().StringVarP(&backendPath, "path", "p", ".", "Path to search for backend configurations")
	discoverBackendsCmd.Flags().BoolVarP(&backendRecursive, "recursive", "r", true, "Search recursively")
	discoverBackendsCmd.Flags().StringVar(&backendProvider, "provider", "", "Filter by provider (s3, azurerm, gcs)")
	
	validateBackendCmd.Flags().DurationVar(&backendTimeout, "timeout", 30*time.Second, "Validation timeout")
	validateBackendCmd.Flags().StringVar(&backendRegion, "region", "", "Override region for validation")
}

func runDiscoverBackends(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	
	// Create backend scanner
	scanner := discovery.NewBackendScanner()
	
	// Configure scanner options
	opts := discovery.ScanOptions{
		Path:      backendPath,
		Recursive: backendRecursive,
		MaxDepth:  10,
		Workers:   4,
	}
	
	if backendProvider != "" {
		opts.FilterTypes = []string{backendProvider}
	}
	
	fmt.Printf("Scanning for Terraform backends in %s...\n", backendPath)
	
	// Perform discovery
	backends, err := scanner.Scan(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to scan for backends: %w", err)
	}
	
	if len(backends) == 0 {
		fmt.Println("No backend configurations found")
		return nil
	}
	
	// Display discovered backends
	fmt.Printf("\nDiscovered %d backend configuration(s):\n\n", len(backends))
	
	for _, backend := range backends {
		fmt.Printf("Backend: %s\n", backend.ID)
		fmt.Printf("  Type: %s\n", backend.Type)
		fmt.Printf("  Path: %s\n", backend.ConfigPath)
		
		switch backend.Type {
		case "s3":
			if bucket, ok := backend.Config["bucket"].(string); ok {
				fmt.Printf("  Bucket: %s\n", bucket)
			}
			if key, ok := backend.Config["key"].(string); ok {
				fmt.Printf("  Key: %s\n", key)
			}
			if region, ok := backend.Config["region"].(string); ok {
				fmt.Printf("  Region: %s\n", region)
			}
			
		case "azurerm":
			if account, ok := backend.Config["storage_account_name"].(string); ok {
				fmt.Printf("  Storage Account: %s\n", account)
			}
			if container, ok := backend.Config["container_name"].(string); ok {
				fmt.Printf("  Container: %s\n", container)
			}
			if key, ok := backend.Config["key"].(string); ok {
				fmt.Printf("  Key: %s\n", key)
			}
			
		case "gcs":
			if bucket, ok := backend.Config["bucket"].(string); ok {
				fmt.Printf("  Bucket: %s\n", bucket)
			}
			if prefix, ok := backend.Config["prefix"].(string); ok {
				fmt.Printf("  Prefix: %s\n", prefix)
			}
		}
		
		if backend.WorkspaceDir != "" {
			fmt.Printf("  Workspace: %s\n", backend.WorkspaceDir)
		}
		
		fmt.Println()
	}
	
	// Register backends
	registry := registry.NewBackendRegistry()
	for _, backend := range backends {
		if err := registry.Register(backend.ID, createBackendProvider(backend)); err != nil {
			fmt.Printf("Warning: Failed to register backend %s: %v\n", backend.ID, err)
		}
	}
	
	return nil
}

func runValidateBackend(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), backendTimeout)
	defer cancel()
	
	backendID := args[0]
	
	// Get backend from registry
	registry := registry.NewBackendRegistry()
	backend := registry.Get(backendID)
	
	if backend == nil {
		// Try to discover it first
		scanner := discovery.NewBackendScanner()
		backends, err := scanner.Scan(ctx, discovery.ScanOptions{
			Path:      ".",
			Recursive: true,
		})
		
		if err != nil {
			return fmt.Errorf("failed to scan for backend: %w", err)
		}
		
		for _, b := range backends {
			if b.ID == backendID || strings.Contains(b.ConfigPath, backendID) {
				backend = createBackendProvider(b)
				break
			}
		}
		
		if backend == nil {
			return fmt.Errorf("backend %s not found", backendID)
		}
	}
	
	fmt.Printf("Validating backend %s...\n", backendID)
	
	// Test connection
	fmt.Print("Testing connection... ")
	if err := backend.TestConnection(ctx); err != nil {
		fmt.Printf("FAILED\n  Error: %v\n", err)
		return err
	}
	fmt.Println("OK")
	
	// Check if state exists
	fmt.Print("Checking state file... ")
	exists, err := backend.StateExists(ctx, "default")
	if err != nil {
		fmt.Printf("ERROR\n  Error: %v\n", err)
		return err
	}
	
	if exists {
		fmt.Println("EXISTS")
		
		// Get state metadata
		fmt.Print("Reading state metadata... ")
		state, err := backend.GetState(ctx, "default")
		if err != nil {
			fmt.Printf("ERROR\n  Error: %v\n", err)
		} else {
			fmt.Println("OK")
			fmt.Printf("  Version: %d\n", state.Version)
			fmt.Printf("  Terraform Version: %s\n", state.TerraformVersion)
			fmt.Printf("  Serial: %d\n", state.Serial)
			fmt.Printf("  Lineage: %s\n", state.Lineage)
			
			if len(state.Outputs) > 0 {
				fmt.Printf("  Outputs: %d\n", len(state.Outputs))
			}
			
			if state.Backend != nil {
				fmt.Printf("  Backend Type: %s\n", state.Backend.Type)
			}
		}
		
		// Check locking
		fmt.Print("Checking lock support... ")
		if locker, ok := backend.(interface{ SupportsLocking() bool }); ok && locker.SupportsLocking() {
			fmt.Println("SUPPORTED")
		} else {
			fmt.Println("NOT SUPPORTED")
		}
		
	} else {
		fmt.Println("NOT FOUND")
	}
	
	fmt.Println("\nBackend validation completed successfully")
	return nil
}

func runListBackends(cmd *cobra.Command, args []string) error {
	registry := registry.NewBackendRegistry()
	backends := registry.List()
	
	if len(backends) == 0 {
		fmt.Println("No backends registered. Run 'driftmgr backend discover' first.")
		return nil
	}
	
	fmt.Printf("Registered backends (%d):\n\n", len(backends))
	
	for _, id := range backends {
		backend := registry.Get(id)
		if backend != nil {
			info := backend.GetInfo()
			fmt.Printf("ID: %s\n", id)
			fmt.Printf("  Type: %s\n", info.Type)
			fmt.Printf("  Region: %s\n", info.Region)
			
			if info.Metadata != nil {
				if bucket, ok := info.Metadata["bucket"].(string); ok {
					fmt.Printf("  Bucket: %s\n", bucket)
				}
				if account, ok := info.Metadata["storage_account"].(string); ok {
					fmt.Printf("  Storage Account: %s\n", account)
				}
			}
			fmt.Println()
		}
	}
	
	return nil
}

func createBackendProvider(config *discovery.BackendConfig) registry.BackendProvider {
	switch config.Type {
	case "s3":
		opts := []s3.Option{}
		
		if region, ok := config.Config["region"].(string); ok {
			opts = append(opts, s3.WithRegion(region))
		}
		
		if roleArn, ok := config.Config["role_arn"].(string); ok {
			opts = append(opts, s3.WithAssumeRole(roleArn, ""))
		}
		
		if encrypt, ok := config.Config["encrypt"].(bool); ok && encrypt {
			opts = append(opts, s3.WithEncryption("AES256", ""))
		}
		
		provider, _ := s3.NewS3Backend(
			config.Config["bucket"].(string),
			config.Config["key"].(string),
			opts...,
		)
		return provider
		
	case "azurerm":
		provider, _ := azure.NewAzureBackend(
			config.Config["storage_account_name"].(string),
			config.Config["container_name"].(string),
			config.Config["key"].(string),
		)
		return provider
		
	case "gcs":
		provider, _ := gcs.NewGCSBackend(
			config.Config["bucket"].(string),
			getStringValue(config.Config, "prefix"),
		)
		return provider
		
	default:
		return nil
	}
}

func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}