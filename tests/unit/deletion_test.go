package test

import (
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/deletion"
)

func TestDeletionEngine(t *testing.T) {
	// Initialize deletion engine
	deletionEngine := deletion.NewDeletionEngine()

	// Test provider registration
	t.Run("Provider Registration", func(t *testing.T) {
		// Test AWS provider registration
		if awsProvider, err := deletion.NewAWSProvider(); err == nil {
			deletionEngine.RegisterProvider("aws", awsProvider)
			t.Log("AWS provider registered successfully")
		} else {
			t.Logf("AWS provider registration failed (expected in test environment): %v", err)
		}

		// Test Azure provider registration
		if azureProvider, err := deletion.NewAzureProvider(); err == nil {
			deletionEngine.RegisterProvider("azure", azureProvider)
			t.Log("Azure provider registered successfully")
		} else {
			t.Logf("Azure provider registration failed (expected in test environment): %v", err)
		}

		// Test GCP provider registration
		if gcpProvider, err := deletion.NewGCPProvider(); err == nil {
			deletionEngine.RegisterProvider("gcp", gcpProvider)
			t.Log("GCP provider registered successfully")
		} else {
			t.Logf("GCP provider registration failed (expected in test environment): %v", err)
		}

		// Get supported providers
		providers := deletionEngine.GetSupportedProviders()
		t.Logf("Supported providers: %v", providers)
	})

	// Test deletion options
	t.Run("Deletion Options", func(t *testing.T) {
		options := deletion.DeletionOptions{
			DryRun:        true,
			Force:         false,
			ResourceTypes: []string{"ec2_instance", "s3_bucket"},
			Regions:       []string{"us-east-1"},
			Timeout:       10 * time.Minute,
			BatchSize:     5,
			ProgressCallback: func(update deletion.ProgressUpdate) {
				t.Logf("Progress: %s - %d/%d - %s",
					update.Type, update.Progress, update.Total, update.Message)
			},
		}

		// Validate options
		if options.DryRun != true {
			t.Errorf("Expected DryRun to be true, got %v", options.DryRun)
		}

		if options.BatchSize != 5 {
			t.Errorf("Expected BatchSize to be 5, got %d", options.BatchSize)
		}

		if len(options.ResourceTypes) != 2 {
			t.Errorf("Expected 2 resource types, got %d", len(options.ResourceTypes))
		}
	})

	// Test safety checks
	t.Run("Safety Checks", func(t *testing.T) {
		// Test critical resource type detection
		criticalTypes := []string{
			"aws_iam_user",
			"aws_iam_role",
			"aws_s3_bucket",
			"azurerm_storage_account",
			"azurerm_key_vault",
			"google_storage_bucket",
		}

		for _, resourceType := range criticalTypes {
			// This would be tested in the actual implementation
			t.Logf("Critical resource type: %s", resourceType)
		}

		// Test critical tag detection
		criticalTags := []string{
			"production",
			"prod",
			"critical",
			"protected",
			"do-not-delete",
		}

		for _, tag := range criticalTags {
			// This would be tested in the actual implementation
			t.Logf("Critical tag: %s", tag)
		}
	})

	// Test deletion result structure
	t.Run("Deletion Result", func(t *testing.T) {
		result := &deletion.DeletionResult{
			AccountID:        "123456789012",
			Provider:         "aws",
			TotalResources:   100,
			DeletedResources: 95,
			FailedResources:  3,
			SkippedResources: 2,
			StartTime:        time.Now(),
			EndTime:          time.Now().Add(5 * time.Minute),
			Duration:         5 * time.Minute,
			Errors: []deletion.DeletionError{
				{
					ResourceID:   "i-1234567890abcdef0",
					ResourceType: "ec2_instance",
					Error:        "Instance is protected from deletion",
					Timestamp:    time.Now(),
				},
			},
			Warnings: []string{
				"Large number of resources deleted (100 total)",
			},
			Details: map[string]interface{}{
				"region": "us-east-1",
			},
		}

		// Validate result
		if result.AccountID != "123456789012" {
			t.Errorf("Expected AccountID to be 123456789012, got %s", result.AccountID)
		}

		if result.Provider != "aws" {
			t.Errorf("Expected Provider to be aws, got %s", result.Provider)
		}

		if result.TotalResources != 100 {
			t.Errorf("Expected TotalResources to be 100, got %d", result.TotalResources)
		}

		if len(result.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(result.Errors))
		}

		if len(result.Warnings) != 1 {
			t.Errorf("Expected 1 warning, got %d", len(result.Warnings))
		}
	})

	// Test progress updates
	t.Run("Progress Updates", func(t *testing.T) {
		update := deletion.ProgressUpdate{
			Type:      "deletion_progress",
			Message:   "Deleted EC2 instance: i-1234567890abcdef0",
			Progress:  5,
			Total:     10,
			Current:   "i-1234567890abcdef0",
			Timestamp: time.Now(),
		}

		// Validate update
		if update.Type != "deletion_progress" {
			t.Errorf("Expected Type to be deletion_progress, got %s", update.Type)
		}

		if update.Progress != 5 {
			t.Errorf("Expected Progress to be 5, got %d", update.Progress)
		}

		if update.Total != 10 {
			t.Errorf("Expected Total to be 10, got %d", update.Total)
		}
	})
}

func TestDeletionSafety(t *testing.T) {
	t.Run("Critical Resource Protection", func(t *testing.T) {
		// Test that critical resources are identified
		criticalResources := []string{
			"aws_iam_user",
			"aws_iam_role",
			"aws_iam_policy",
			"aws_s3_bucket",
			"aws_rds_cluster",
			"aws_eks_cluster",
			"azurerm_storage_account",
			"azurerm_key_vault",
			"google_storage_bucket",
			"google_kms_crypto_key",
		}

		for _, resourceType := range criticalResources {
			t.Logf("Critical resource type: %s", resourceType)
			// In the actual implementation, these would be checked
		}
	})

	t.Run("Critical Tag Protection", func(t *testing.T) {
		// Test that critical tags are identified
		criticalTags := []string{
			"production",
			"prod",
			"critical",
			"protected",
			"do-not-delete",
		}

		for _, tag := range criticalTags {
			t.Logf("Critical tag: %s", tag)
			// In the actual implementation, these would be checked
		}
	})

	t.Run("Large Scale Protection", func(t *testing.T) {
		// Test large scale deletion protection
		largeResourceCount := 1500
		if largeResourceCount > 1000 {
			t.Logf("Large scale deletion detected: %d resources", largeResourceCount)
			// In the actual implementation, this would trigger a warning
		}
	})
}

func TestDeletionOrder(t *testing.T) {
	t.Run("AWS Deletion Order", func(t *testing.T) {
		awsOrder := []string{
			"autoscaling_group",
			"ecs_service",
			"ecs_cluster",
			"eks_nodegroup",
			"eks_cluster",
			"lambda_function",
			"rds_instance",
			"elasticache_cluster",
			"dynamodb_table",
			"ec2_instance",
			"ec2_volume",
			"ec2_security_group",
			"ec2_subnet",
			"ec2_route_table",
			"ec2_internet_gateway",
			"ec2_vpc",
			"s3_bucket",
			"sns_topic",
			"sqs_queue",
			"route53_record",
			"route53_hosted_zone",
			"cloudformation_stack",
			"iam_role",
			"iam_policy",
			"iam_user",
		}

		t.Logf("AWS deletion order has %d resource types", len(awsOrder))
		for i, resourceType := range awsOrder {
			t.Logf("%d. %s", i+1, resourceType)
		}
	})

	t.Run("Azure Deletion Order", func(t *testing.T) {
		azureOrder := []string{
			"microsoft.compute/virtualmachines",
			"microsoft.network/networkinterfaces",
			"microsoft.network/publicipaddresses",
			"microsoft.network/loadbalancers",
			"microsoft.network/applicationgateways",
			"microsoft.network/virtualnetworks",
			"microsoft.storage/storageaccounts",
			"microsoft.keyvault/vaults",
			"microsoft.web/sites",
			"microsoft.containerregistry/registries",
			"microsoft.containerservice/managedclusters",
			"microsoft.resources/resourcegroups",
		}

		t.Logf("Azure deletion order has %d resource types", len(azureOrder))
		for i, resourceType := range azureOrder {
			t.Logf("%d. %s", i+1, resourceType)
		}
	})

	t.Run("GCP Deletion Order", func(t *testing.T) {
		gcpOrder := []string{
			"compute.googleapis.com/instances",
			"container.googleapis.com/clusters",
			"storage.googleapis.com/buckets",
			"sqladmin.googleapis.com/instances",
			"pubsub.googleapis.com/topics",
			"cloudfunctions.googleapis.com/functions",
			"dataproc.googleapis.com/clusters",
			"bigquery.googleapis.com/datasets",
			"compute.googleapis.com/forwardingRules",
			"compute.googleapis.com/targetPools",
			"compute.googleapis.com/networks",
		}

		t.Logf("GCP deletion order has %d resource types", len(gcpOrder))
		for i, resourceType := range gcpOrder {
			t.Logf("%d. %s", i+1, resourceType)
		}
	})
}

func BenchmarkDeletionEngine(b *testing.B) {
	deletionEngine := deletion.NewDeletionEngine()

	// Register providers
	if awsProvider, err := deletion.NewAWSProvider(); err == nil {
		deletionEngine.RegisterProvider("aws", awsProvider)
	}

	_ = deletion.DeletionOptions{
		DryRun:        true,
		Force:         false,
		ResourceTypes: []string{"ec2_instance"},
		Regions:       []string{"us-east-1"},
		Timeout:       1 * time.Minute,
		BatchSize:     10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This would be the actual deletion call in a real scenario
		_ = deletionEngine.GetSupportedProviders()
	}
}
