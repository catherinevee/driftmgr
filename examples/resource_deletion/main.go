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
	// Initialize the deletion engine
	deletionEngine := deletion.NewDeletionEngine()

	// Register cloud providers
	if awsProvider, err := deletion.NewAWSProvider(); err == nil {
		deletionEngine.RegisterProvider("aws", awsProvider)
		log.Println("AWS provider registered successfully")
	} else {
		log.Printf("Warning: Failed to register AWS provider: %v", err)
	}

	if azureProvider, err := deletion.NewAzureProvider(); err == nil {
		deletionEngine.RegisterProvider("azure", azureProvider)
		log.Println("Azure provider registered successfully")
	} else {
		log.Printf("Warning: Failed to register Azure provider: %v", err)
	}

	if gcpProvider, err := deletion.NewGCPProvider(); err == nil {
		deletionEngine.RegisterProvider("gcp", gcpProvider)
		log.Println("GCP provider registered successfully")
	} else {
		log.Printf("Warning: Failed to register GCP provider: %v", err)
	}

	// Example 1: Preview deletion (dry run)
	fmt.Println("\n=== Example 1: Preview Deletion (Dry Run) ===")
	previewOptions := deletion.DeletionOptions{
		DryRun:        true,
		Force:         false,
		ResourceTypes: []string{"ec2_instance", "s3_bucket"},
		Regions:       []string{"us-east-1", "us-west-2"},
		Timeout:       10 * time.Minute,
		BatchSize:     5,
		ProgressCallback: func(update deletion.ProgressUpdate) {
			fmt.Printf("Progress: %s - %d/%d - %s\n",
				update.Type, update.Progress, update.Total, update.Message)
		},
	}

	result, err := deletionEngine.DeleteAccountResources(context.Background(), "aws", "123456789012", previewOptions)
	if err != nil {
		log.Printf("Preview failed: %v", err)
	} else {
		fmt.Printf("Preview completed: %d resources would be deleted\n", result.DeletedResources)
	}

	// Example 2: Get supported providers
	fmt.Println("\n=== Example 2: Supported Providers ===")
	providers := deletionEngine.GetSupportedProviders()
	fmt.Printf("Supported providers: %v\n", providers)

	// Example 3: Force deletion (use with extreme caution!)
	fmt.Println("\n=== Example 3: Force Deletion (DANGEROUS!) ===")
	fmt.Println("Force deletion options would be configured here")
	fmt.Println("WARNING: Force deletion bypasses all safety checks!")
	fmt.Println("This example is intentionally disabled for safety reasons.")

	// Example of force deletion options (commented out for safety)
	/*
		forceOptions := deletion.DeletionOptions{
			DryRun:           false,
			Force:            true, // Bypass safety checks
			ResourceTypes:    []string{"ec2_instance"},
			Regions:          []string{"us-east-1"},
			Timeout:          30 * time.Minute,
			BatchSize:        10,
			ExcludeResources: []string{"critical-instance-1", "production-db"},
			ProgressCallback: func(update deletion.ProgressUpdate) {
				fmt.Printf("Progress: %s - %d/%d - %s\n",
					update.Type, update.Progress, update.Total, update.Message)
			},
		}

		// WARNING: This will actually delete resources!
		// Uncomment the following lines only if you're absolutely sure
		result, err = deletionEngine.DeleteAccountResources(context.Background(), "aws", "123456789012", forceOptions)
		if err != nil {
			log.Printf("Deletion failed: %v", err)
		} else {
			fmt.Printf("Deletion completed: %d resources deleted, %d failed\n",
				result.DeletedResources, result.FailedResources)

			// Print errors if any
			for _, deletionError := range result.Errors {
				fmt.Printf("Error deleting %s (%s): %s\n",
					deletionError.ResourceID, deletionError.ResourceType, deletionError.Error)
			}
		}
	*/

	// Example 4: Azure resource deletion
	fmt.Println("\n=== Example 4: Azure Resource Deletion ===")
	azureOptions := deletion.DeletionOptions{
		DryRun:        true,
		Force:         false,
		ResourceTypes: []string{"microsoft.compute/virtualmachines", "microsoft.storage/storageaccounts"},
		Regions:       []string{"eastus", "westus"},
		Timeout:       15 * time.Minute,
		BatchSize:     5,
		ProgressCallback: func(update deletion.ProgressUpdate) {
			fmt.Printf("Azure Progress: %s - %d/%d - %s\n",
				update.Type, update.Progress, update.Total, update.Message)
		},
	}

	result, err = deletionEngine.DeleteAccountResources(context.Background(), "azure", "subscription-id", azureOptions)
	if err != nil {
		log.Printf("Azure preview failed: %v", err)
	} else {
		fmt.Printf("Azure preview completed: %d resources would be deleted\n", result.DeletedResources)
	}

	// Example 5: GCP resource deletion
	fmt.Println("\n=== Example 5: GCP Resource Deletion ===")
	gcpOptions := deletion.DeletionOptions{
		DryRun:        true,
		Force:         false,
		ResourceTypes: []string{"compute.googleapis.com/instances", "storage.googleapis.com/buckets"},
		Regions:       []string{"us-central1-a", "us-west1-a"},
		Timeout:       20 * time.Minute,
		BatchSize:     5,
		ProgressCallback: func(update deletion.ProgressUpdate) {
			fmt.Printf("GCP Progress: %s - %d/%d - %s\n",
				update.Type, update.Progress, update.Total, update.Message)
		},
	}

	result, err = deletionEngine.DeleteAccountResources(context.Background(), "gcp", "project-id", gcpOptions)
	if err != nil {
		log.Printf("GCP preview failed: %v", err)
	} else {
		fmt.Printf("GCP preview completed: %d resources would be deleted\n", result.DeletedResources)
	}

	fmt.Println("\n=== Resource Deletion Example Completed ===")
	fmt.Println("Remember: Always use dry-run mode first to preview what will be deleted!")
	fmt.Println("Use force mode only when you're absolutely certain about the deletion.")
}

// Example API usage functions
func exampleAPICalls() {
	fmt.Println("\n=== API Usage Examples ===")

	// Example 1: Preview deletion via API
	previewRequest := map[string]interface{}{
		"provider":   "aws",
		"account_id": "123456789012",
		"options": map[string]interface{}{
			"dry_run":        true,
			"force":          false,
			"resource_types": []string{"ec2_instance", "s3_bucket"},
			"regions":        []string{"us-east-1"},
			"timeout":        "10m",
			"batch_size":     5,
		},
	}

	previewJSON, _ := json.MarshalIndent(previewRequest, "", "  ")
	fmt.Printf("Preview API Request:\n%s\n", string(previewJSON))

	// Example 2: Actual deletion via API
	deletionRequest := map[string]interface{}{
		"provider":   "aws",
		"account_id": "123456789012",
		"options": map[string]interface{}{
			"dry_run":           false,
			"force":             true,
			"resource_types":    []string{"ec2_instance"},
			"regions":           []string{"us-east-1"},
			"timeout":           "30m",
			"batch_size":        10,
			"exclude_resources": []string{"critical-instance-1"},
		},
	}

	deletionJSON, _ := json.MarshalIndent(deletionRequest, "", "  ")
	fmt.Printf("Deletion API Request:\n%s\n", string(deletionJSON))

	// Example 3: Get supported providers
	fmt.Println("Get Supported Providers API: GET /api/v1/delete/providers")
}
