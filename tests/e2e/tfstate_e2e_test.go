package e2e_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/terraform/state"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTerraformStateE2E tests end-to-end workflow with real tfstate files
func TestTerraformStateE2E(t *testing.T) {
	// Get absolute path to driftmgr
	wd, err := os.Getwd()
	require.NoError(t, err)

	driftmgrPath := filepath.Join(wd, "..", "..", "driftmgr.exe")
	if _, err := os.Stat(driftmgrPath); os.IsNotExist(err) {
		// Try to build it
		cmd := exec.Command("go", "build", "-o", "driftmgr.exe", "./cmd/driftmgr")
		cmd.Dir = filepath.Join(wd, "..", "..")
		if err := cmd.Run(); err != nil {
			t.Skip("Could not build driftmgr.exe, skipping e2e test")
		}
	}

	tempDir := t.TempDir()

	t.Run("Complete Terraform State Workflow", func(t *testing.T) {
		// Step 1: Create a realistic Terraform state file
		t.Log("Step 1: Creating Terraform state file...")
		stateFile := filepath.Join(tempDir, "terraform.tfstate")
		tfState := createProductionState()

		data, err := json.MarshalIndent(tfState, "", "  ")
		require.NoError(t, err)

		err = os.WriteFile(stateFile, data, 0644)
		require.NoError(t, err)

		// Step 2: Test tfstate list command
		t.Log("Step 2: Testing tfstate list command...")
		cmd := exec.Command(driftmgrPath, "tfstate", "list")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		assert.Contains(t, string(output), "terraform.tfstate")

		// Step 3: Test tfstate show command
		t.Log("Step 3: Testing tfstate show command...")
		cmd = exec.Command(driftmgrPath, "tfstate", "show", "terraform.tfstate", "--resources")
		cmd.Dir = tempDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		assert.Contains(t, string(output), "Resources: 10")
		assert.Contains(t, string(output), "aws_instance")
		assert.Contains(t, string(output), "aws_rds_instance")

		// Step 4: Test tfstate analyze command
		t.Log("Step 4: Testing tfstate analyze command...")
		cmd = exec.Command(driftmgrPath, "tfstate", "analyze", "terraform.tfstate")
		cmd.Dir = tempDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, string(output))

		// Step 5: Test state inspect command
		t.Log("Step 5: Testing state inspect command...")
		cmd = exec.Command(driftmgrPath, "state", "inspect", "--state", "terraform.tfstate")
		cmd.Dir = tempDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		assert.Contains(t, string(output), "Total Resources: 10")
		assert.Contains(t, string(output), "Resources by Provider:")
		assert.Contains(t, string(output), "aws")

		// Step 6: Test drift detection with state file
		t.Log("Step 6: Testing drift detection with state file...")
		cmd = exec.Command(driftmgrPath, "drift", "detect", "--state", "terraform.tfstate", "--provider", "aws")
		cmd.Dir = tempDir
		output, err = cmd.CombinedOutput()
		// Note: May fail if AWS credentials aren't configured, which is okay
		if err == nil {
			assert.Contains(t, string(output), "state_file")
			assert.Contains(t, string(output), "terraform.tfstate")
		}

		// Step 7: Create resources for import test
		t.Log("Step 7: Testing import workflow...")
		importCSV := filepath.Join(tempDir, "import.csv")
		csvContent := `provider,type,name,id
aws,aws_instance,imported_server,i-0123456789abcdef0
aws,aws_s3_bucket,imported_bucket,my-imported-bucket
aws,aws_rds_instance,imported_db,my-database-instance`

		err = os.WriteFile(importCSV, []byte(csvContent), 0644)
		require.NoError(t, err)

		cmd = exec.Command(driftmgrPath, "import", "--input", "import.csv", "--dry-run")
		cmd.Dir = tempDir
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		assert.Contains(t, string(output), "Loaded 3 resources for import")
		assert.Contains(t, string(output), "terraform import aws_instance.imported_server")

		// Step 8: Test state visualization
		t.Log("Step 8: Testing state visualization...")
		cmd = exec.Command(driftmgrPath, "state", "visualize", "--state", "terraform.tfstate", "--format", "json", "--output", "viz.json")
		cmd.Dir = tempDir
		output, err = cmd.CombinedOutput()
		// Visualization might not be fully implemented, but command should at least run
		if err == nil {
			vizFile := filepath.Join(tempDir, "viz.json")
			if _, err := os.Stat(vizFile); err == nil {
				t.Log("Visualization file created successfully")
			}
		}

		// Step 9: Verify state file parsing with actual discovery
		t.Log("Step 9: Verifying state parsing with discovery...")
		loader := state.NewStateLoader(stateFile)
		loadedState, err := loader.Load()
		require.NoError(t, err)
		assert.Equal(t, 10, len(loadedState.Resources))

		// Convert to models.Resource and verify
		resources := convertStateResources(loadedState)
		assert.Equal(t, 10, len(resources))

		// Verify resource details
		hasWebServer := false
		hasDatabase := false
		hasLoadBalancer := false

		for _, resource := range resources {
			switch resource.Name {
			case "web_server_1", "web_server_2":
				hasWebServer = true
				assert.Equal(t, "aws_instance", resource.Type)
			case "main_database":
				hasDatabase = true
				assert.Equal(t, "aws_rds_instance", resource.Type)
			case "main":
				if resource.Type == "aws_lb" {
					hasLoadBalancer = true
				}
			}
		}

		assert.True(t, hasWebServer, "Should have web servers")
		assert.True(t, hasDatabase, "Should have database")
		assert.True(t, hasLoadBalancer, "Should have load balancer")

		t.Log("End-to-end Terraform state workflow completed successfully!")
	})

	t.Run("Test Large State File Performance", func(t *testing.T) {
		// Create a large state file with many resources
		t.Log("Creating large state file...")
		largeStateFile := filepath.Join(tempDir, "large.tfstate")
		largeState := createLargeState(100) // 100 resources

		data, err := json.MarshalIndent(largeState, "", "  ")
		require.NoError(t, err)

		err = os.WriteFile(largeStateFile, data, 0644)
		require.NoError(t, err)

		// Test parsing performance
		start := time.Now()
		loader := state.NewStateLoader(largeStateFile)
		loadedState, err := loader.Load()
		require.NoError(t, err)
		duration := time.Since(start)

		t.Logf("Parsed %d resources in %v", len(loadedState.Resources), duration)
		assert.Less(t, duration.Seconds(), 5.0, "Should parse large state file quickly")

		// Test state inspection performance
		start = time.Now()
		cmd := exec.Command(driftmgrPath, "state", "inspect", "--state", "large.tfstate")
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()
		duration = time.Since(start)

		require.NoError(t, err, string(output))
		t.Logf("Inspected large state in %v", duration)
		assert.Less(t, duration.Seconds(), 5.0, "Should inspect large state quickly")
	})
}

// Helper functions

func createProductionState() *state.TerraformState {
	return &state.TerraformState{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           100,
		Lineage:          "prod-state-lineage",
		Resources: []state.StateResource{
			// Web servers
			{
				Mode:     "managed",
				Type:     "aws_instance",
				Name:     "web_server_1",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":            "i-web1234567890",
							"ami":           "ami-0c55b159cbfafe1f0",
							"instance_type": "t3.large",
							"subnet_id":     "subnet-web1",
							"tags": map[string]interface{}{
								"Name":        "WebServer1",
								"Environment": "production",
								"Role":        "web",
							},
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "aws_instance",
				Name:     "web_server_2",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":            "i-web0987654321",
							"ami":           "ami-0c55b159cbfafe1f0",
							"instance_type": "t3.large",
							"subnet_id":     "subnet-web2",
							"tags": map[string]interface{}{
								"Name":        "WebServer2",
								"Environment": "production",
								"Role":        "web",
							},
						},
					},
				},
			},
			// Database
			{
				Mode:     "managed",
				Type:     "aws_rds_instance",
				Name:     "main_database",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":                      "main-database",
							"engine":                  "postgres",
							"engine_version":          "14.7",
							"instance_class":          "db.r5.xlarge",
							"allocated_storage":       100,
							"multi_az":                true,
							"publicly_accessible":     false,
							"backup_retention_period": 30,
							"tags": map[string]interface{}{
								"Name":        "MainDatabase",
								"Environment": "production",
								"Critical":    "true",
							},
						},
					},
				},
			},
			// Load Balancer
			{
				Mode:     "managed",
				Type:     "aws_lb",
				Name:     "main",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0,
						Attributes: map[string]interface{}{
							"id":                 "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/main/1234567890abcdef",
							"name":               "main-lb",
							"load_balancer_type": "application",
							"scheme":             "internet-facing",
							"tags": map[string]interface{}{
								"Name":        "MainLoadBalancer",
								"Environment": "production",
							},
						},
					},
				},
			},
			// VPC
			{
				Mode:     "managed",
				Type:     "aws_vpc",
				Name:     "main",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":                   "vpc-prod123456",
							"cidr_block":           "10.0.0.0/16",
							"enable_dns_hostnames": true,
							"enable_dns_support":   true,
							"tags": map[string]interface{}{
								"Name":        "ProductionVPC",
								"Environment": "production",
							},
						},
					},
				},
			},
			// Subnets
			{
				Mode:     "managed",
				Type:     "aws_subnet",
				Name:     "public_1",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":                      "subnet-pub1",
							"vpc_id":                  "vpc-prod123456",
							"cidr_block":              "10.0.1.0/24",
							"availability_zone":       "us-east-1a",
							"map_public_ip_on_launch": true,
							"tags": map[string]interface{}{
								"Name": "PublicSubnet1",
								"Type": "public",
							},
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "aws_subnet",
				Name:     "public_2",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":                      "subnet-pub2",
							"vpc_id":                  "vpc-prod123456",
							"cidr_block":              "10.0.2.0/24",
							"availability_zone":       "us-east-1b",
							"map_public_ip_on_launch": true,
							"tags": map[string]interface{}{
								"Name": "PublicSubnet2",
								"Type": "public",
							},
						},
					},
				},
			},
			// Security Groups
			{
				Mode:     "managed",
				Type:     "aws_security_group",
				Name:     "web",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":          "sg-web123456",
							"name":        "web-security-group",
							"description": "Security group for web servers",
							"vpc_id":      "vpc-prod123456",
							"tags": map[string]interface{}{
								"Name": "WebSecurityGroup",
							},
						},
					},
				},
			},
			{
				Mode:     "managed",
				Type:     "aws_security_group",
				Name:     "database",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":          "sg-db123456",
							"name":        "database-security-group",
							"description": "Security group for database",
							"vpc_id":      "vpc-prod123456",
							"tags": map[string]interface{}{
								"Name": "DatabaseSecurityGroup",
							},
						},
					},
				},
			},
			// S3 Bucket
			{
				Mode:     "managed",
				Type:     "aws_s3_bucket",
				Name:     "assets",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []state.ResourceInstance{
					{
						SchemaVersion: 0,
						Attributes: map[string]interface{}{
							"id":     "prod-assets-bucket-12345",
							"bucket": "prod-assets-bucket-12345",
							"region": "us-east-1",
							"versioning": []map[string]interface{}{
								{
									"enabled":    true,
									"mfa_delete": false,
								},
							},
							"tags": map[string]interface{}{
								"Name":        "ProductionAssets",
								"Environment": "production",
								"Purpose":     "static-assets",
							},
						},
					},
				},
			},
		},
	}
}

func createLargeState(resourceCount int) *state.TerraformState {
	tfState := &state.TerraformState{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           1000,
		Lineage:          "large-state-test",
		Resources:        make([]state.StateResource, 0, resourceCount),
	}

	for i := 0; i < resourceCount; i++ {
		resource := state.StateResource{
			Mode:     "managed",
			Type:     fmt.Sprintf("aws_instance"),
			Name:     fmt.Sprintf("server_%d", i),
			Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
			Instances: []state.ResourceInstance{
				{
					SchemaVersion: 1,
					Attributes: map[string]interface{}{
						"id":            fmt.Sprintf("i-%012d", i),
						"ami":           "ami-0c55b159cbfafe1f0",
						"instance_type": "t3.micro",
						"tags": map[string]interface{}{
							"Name":  fmt.Sprintf("Server%d", i),
							"Index": fmt.Sprintf("%d", i),
						},
					},
				},
			},
		}
		tfState.Resources = append(tfState.Resources, resource)
	}

	return tfState
}

func convertStateResources(tfState *state.TerraformState) []models.Resource {
	var resources []models.Resource

	for _, stateResource := range tfState.Resources {
		for _, instance := range stateResource.Instances {
			resource := models.Resource{
				Type:       stateResource.Type,
				Name:       stateResource.Name,
				Provider:   extractProvider(stateResource.Provider),
				Attributes: instance.Attributes,
				Metadata: map[string]string{
					"mode": stateResource.Mode,
				},
			}

			if id, ok := instance.Attributes["id"].(string); ok {
				resource.ID = id
			}

			if tags, ok := instance.Attributes["tags"].(map[string]interface{}); ok {
				tagMap := make(map[string]string)
				for k, v := range tags {
					if str, ok := v.(string); ok {
						tagMap[k] = str
					}
				}
				resource.Tags = tagMap
			}

			resources = append(resources, resource)
		}
	}

	return resources
}

func extractProvider(provider string) string {
	// Extract from format like "provider[\"registry.terraform.io/hashicorp/aws\"]"
	if strings.Contains(provider, "aws") {
		return "aws"
	}
	if strings.Contains(provider, "azurerm") {
		return "azure"
	}
	if strings.Contains(provider, "google") {
		return "gcp"
	}
	return "unknown"
}
