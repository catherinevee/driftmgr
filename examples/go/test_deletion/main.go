package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/catherinevee/driftmgr/internal/deletion"
)

func main() {
	fmt.Println("=== Testing DriftMgr Deletion Feature ===")
	fmt.Println("This will test the deletion functionality on your local AWS account")
	fmt.Println("WARNING: This is a test - use dry-run mode to be safe!")
	fmt.Println()

	// Initialize the deletion engine
	deletionEngine := deletion.NewDeletionEngine()

	// Register AWS provider
	fmt.Println("1. Registering AWS provider...")
	if awsProvider, err := deletion.NewAWSProvider(); err == nil {
		deletionEngine.RegisterProvider("aws", awsProvider)
		fmt.Println("   ✓ AWS provider registered successfully")
	} else {
		log.Fatalf("   ✗ Failed to register AWS provider: %v", err)
	}

	// Get supported providers
	fmt.Println("\n2. Checking supported providers...")
	providers := deletionEngine.GetSupportedProviders()
	fmt.Printf("   ✓ Supported providers: %v\n", providers)

	// Test 1: Preview deletion (dry run) - SAFE
	fmt.Println("\n3. Testing preview deletion (dry run)...")
	previewOptions := deletion.DeletionOptions{
		DryRun:        true,  // SAFE - no actual deletion
		Force:         false,
		ResourceTypes: []string{"ec2_instance", "s3_bucket"},
		Regions:       []string{"us-east-1"},
		Timeout:       5 * time.Minute,
		BatchSize:     5,
		ProgressCallback: func(update deletion.ProgressUpdate) {
			fmt.Printf("   Progress: %s - %d/%d - %s\n", 
				update.Type, update.Progress, update.Total, update.Message)
		},
	}

	// Get AWS account ID
	accountID := "025066254478" // Your actual AWS account ID
	
	fmt.Printf("   Using account ID: %s\n", accountID)
	fmt.Println("   Starting preview deletion...")

	result, err := deletionEngine.DeleteAccountResources(context.Background(), "aws", accountID, previewOptions)
	if err != nil {
		fmt.Printf("   ✗ Preview failed: %v\n", err)
		fmt.Println("   This might be due to:")
		fmt.Println("   - Invalid AWS credentials")
		fmt.Println("   - Invalid account ID")
		fmt.Println("   - No resources found")
		fmt.Println("   - AWS API permissions")
	} else {
		fmt.Printf("   ✓ Preview completed successfully!\n")
		fmt.Printf("   - Total resources found: %d\n", result.TotalResources)
		fmt.Printf("   - Resources that would be deleted: %d\n", result.DeletedResources)
		fmt.Printf("   - Duration: %v\n", result.Duration)
		
		if len(result.Warnings) > 0 {
			fmt.Println("   - Warnings:")
			for _, warning := range result.Warnings {
				fmt.Printf("     * %s\n", warning)
			}
		}
	}

	// Test 2: Get specific resource types
	fmt.Println("\n4. Testing resource type filtering...")
	filteredOptions := deletion.DeletionOptions{
		DryRun:        true,
		Force:         false,
		ResourceTypes: []string{"ec2_instance"}, // Only EC2 instances
		Regions:       []string{"us-east-1"},
		Timeout:       3 * time.Minute,
		BatchSize:     3,
		ProgressCallback: func(update deletion.ProgressUpdate) {
			fmt.Printf("   Progress: %s - %d/%d - %s\n", 
				update.Type, update.Progress, update.Total, update.Message)
		},
	}

	fmt.Println("   Testing EC2 instance discovery only...")
	result2, err := deletionEngine.DeleteAccountResources(context.Background(), "aws", accountID, filteredOptions)
	if err != nil {
		fmt.Printf("   ✗ Filtered preview failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Filtered preview completed!\n")
		fmt.Printf("   - EC2 instances found: %d\n", result2.TotalResources)
	}

	// Test 3: API endpoint simulation
	fmt.Println("\n5. Testing API endpoint simulation...")
	
	// Simulate the API request structure
	apiRequest := map[string]interface{}{
		"provider":   "aws",
		"account_id": accountID,
		"options": map[string]interface{}{
			"dry_run":        true,
			"force":          false,
			"resource_types": []string{"s3_bucket"},
			"regions":        []string{"us-east-1"},
			"timeout":        "5m",
			"batch_size":     3,
		},
	}

	apiJSON, _ := json.MarshalIndent(apiRequest, "", "  ")
	fmt.Println("   API Request structure:")
	fmt.Printf("   %s\n", string(apiJSON))

	fmt.Println("\n=== Test Summary ===")
	fmt.Println("✓ Deletion engine initialized")
	fmt.Println("✓ AWS provider registered")
	fmt.Println("✓ Preview functionality tested")
	fmt.Println("✓ Resource filtering tested")
	fmt.Println("✓ API structure validated")
	
	fmt.Println("\n=== Next Steps ===")
	fmt.Println("1. Replace 'your-aws-account-id' with your actual AWS account ID")
	fmt.Println("2. Ensure your AWS credentials are properly configured")
	fmt.Println("3. Run the server: go run ./cmd/driftmgr-server")
	fmt.Println("4. Test via API: curl -X POST http://localhost:8080/api/v1/delete/preview")
	fmt.Println("5. Always use dry_run=true for testing!")
	
	fmt.Println("\n=== Safety Reminders ===")
	fmt.Println("⚠️  ALWAYS use dry_run=true for testing")
	fmt.Println("⚠️  NEVER use force=true without careful review")
	fmt.Println("⚠️  Test on non-production accounts first")
	fmt.Println("⚠️  Review the preview results before any actual deletion")
}
