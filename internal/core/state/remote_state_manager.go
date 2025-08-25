package state

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// RemoteStateManager handles remote Terraform state files
type RemoteStateManager struct {
	awsConfig *aws.Config
}

// RemoteStateConfig represents remote state configuration
type RemoteStateConfig struct {
	Backend        string            `json:"backend"`
	Config         map[string]string `json:"config"`
	Workspace      string            `json:"workspace,omitempty"`
	Key            string            `json:"key"`
	Region         string            `json:"region,omitempty"`
	Bucket         string            `json:"bucket,omitempty"`
	StorageAccount string            `json:"storage_account,omitempty"`
	Container      string            `json:"container,omitempty"`
	Project        string            `json:"project,omitempty"`
}

// NewRemoteStateManager creates a new remote state manager
func NewRemoteStateManager() (*RemoteStateManager, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &RemoteStateManager{
		awsConfig: &cfg,
	}, nil
}

// ParseRemoteState parses remote state from various backends
func (rsm *RemoteStateManager) ParseRemoteState(config *RemoteStateConfig) (*models.StateFile, error) {
	switch config.Backend {
	case "s3":
		return rsm.parseS3State(config)
	case "azurerm":
		return rsm.parseAzureState(config)
	case "gcs":
		return rsm.parseGCSState(config)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", config.Backend)
	}
}

// parseS3State parses Terraform state from S3
func (rsm *RemoteStateManager) parseS3State(config *RemoteStateConfig) (*models.StateFile, error) {
	client := s3.NewFromConfig(*rsm.awsConfig)

	// Determine the state file key
	stateKey := config.Key
	if config.Workspace != "" && config.Workspace != "default" {
		stateKey = path.Join("env:", config.Workspace, config.Key)
	}

	// Get the state file from S3
	resp, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(stateKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get state file from S3: %w", err)
	}
	defer resp.Body.Close()

	// Read and parse the state file
	stateData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	return rsm.parseStateData(stateData, fmt.Sprintf("s3://%s/%s", config.Bucket, stateKey))
}

// parseAzureState parses Terraform state from Azure Storage
func (rsm *RemoteStateManager) parseAzureState(config *RemoteStateConfig) (*models.StateFile, error) {
	ctx := context.Background()

	// Create Azure credential
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	// Create blob service client
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", config.StorageAccount)
	client, err := azblob.NewClient(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure blob client: %w", err)
	}

	// Determine the state file blob name
	blobName := config.Key
	if config.Workspace != "" && config.Workspace != "default" {
		blobName = fmt.Sprintf("env:%s/%s", config.Workspace, config.Key)
	}

	// Download the blob
	downloadResponse, err := client.DownloadStream(ctx, config.Container, blobName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download state file from Azure: %w", err)
	}
	defer downloadResponse.Body.Close()

	// Read the state file
	stateData, err := io.ReadAll(downloadResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	return rsm.parseStateData(stateData, fmt.Sprintf("azurerm://%s/%s/%s", config.StorageAccount, config.Container, blobName))
}

// parseGCSState parses Terraform state from Google Cloud Storage
func (rsm *RemoteStateManager) parseGCSState(config *RemoteStateConfig) (*models.StateFile, error) {
	ctx := context.Background()

	// Create GCS client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	// Determine the state file object name
	objectName := config.Key
	if config.Workspace != "" && config.Workspace != "default" {
		objectName = fmt.Sprintf("env:%s/%s", config.Workspace, config.Key)
	}

	// Get the bucket handle
	bucket := client.Bucket(config.Bucket)

	// Get the object handle
	obj := bucket.Object(objectName)

	// Read the object
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file from GCS: %w", err)
	}
	defer reader.Close()

	// Read the state file
	stateData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	return rsm.parseStateData(stateData, fmt.Sprintf("gcs://%s/%s", config.Bucket, objectName))
}

// parseStateData parses state file data and converts to our model
func (rsm *RemoteStateManager) parseStateData(data []byte, source string) (*models.StateFile, error) {
	var rawState map[string]interface{}
	if err := json.Unmarshal(data, &rawState); err != nil {
		return nil, fmt.Errorf("failed to parse state JSON: %w", err)
	}

	stateFile := &models.StateFile{
		Path:      source,
		Version:   4, // Default Terraform state version
		Serial:    1,
		Lineage:   "",
		Outputs:   make(map[string]interface{}),
		Resources: []models.TerraformResource{},
	}

	// Extract resources
	if resources, ok := rawState["resources"].([]interface{}); ok {
		for _, resource := range resources {
			if resourceMap, ok := resource.(map[string]interface{}); ok {
				tfResource := rsm.convertToTerraformResource(resourceMap)
				stateFile.Resources = append(stateFile.Resources, tfResource)
			}
		}
	}

	return stateFile, nil
}

// convertToTerraformResource converts raw state resource to our model
func (rsm *RemoteStateManager) convertToTerraformResource(resourceMap map[string]interface{}) models.TerraformResource {
	resource := models.TerraformResource{
		Instances: []models.TerraformResourceInstance{},
	}

	// Extract basic fields
	if name, ok := resourceMap["name"].(string); ok {
		resource.Name = name
	}
	if resourceType, ok := resourceMap["type"].(string); ok {
		resource.Type = resourceType
	}
	if mode, ok := resourceMap["mode"].(string); ok {
		resource.Mode = mode
	}

	// Extract provider
	if provider, ok := resourceMap["provider"].(string); ok {
		resource.Provider = provider
	}

	// Extract instances
	if instances, ok := resourceMap["instances"].([]interface{}); ok {
		for _, instance := range instances {
			if instanceMap, ok := instance.(map[string]interface{}); ok {
				tfInstance := rsm.convertToTerraformInstance(instanceMap)
				resource.Instances = append(resource.Instances, tfInstance)
			}
		}
	}

	return resource
}

// convertToTerraformInstance converts raw state instance to our model
func (rsm *RemoteStateManager) convertToTerraformInstance(instanceMap map[string]interface{}) models.TerraformResourceInstance {
	instance := models.TerraformResourceInstance{
		Attributes: make(map[string]interface{}),
	}

	// Extract attributes
	if attributes, ok := instanceMap["attributes"].(map[string]interface{}); ok {
		instance.Attributes = attributes
	}

	// Extract schema version
	if schemaVersion, ok := instanceMap["schema_version"].(float64); ok {
		instance.SchemaVersion = int(schemaVersion)
	}

	// Extract private
	if private, ok := instanceMap["private"].(string); ok {
		instance.Private = private
	}

	return instance
}

// DetectRemoteStateConfig detects remote state configuration from Terraform files
func (rsm *RemoteStateManager) DetectRemoteStateConfig(terraformPath string) ([]*RemoteStateConfig, error) {
	// TODO: Parse terraform.tfstate.backup, .terraform/terraform.tfstate, and .terraform.tfstate.backup
	// This would scan for remote state configurations in Terraform files
	return nil, fmt.Errorf("remote state detection not yet implemented")
}

// ListRemoteStates lists available remote state files
func (rsm *RemoteStateManager) ListRemoteStates(config *RemoteStateConfig) ([]string, error) {
	switch config.Backend {
	case "s3":
		return rsm.listS3States(config)
	case "azurerm":
		return rsm.listAzureStates(config)
	case "gcs":
		return rsm.listGCSStates(config)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", config.Backend)
	}
}

// listS3States lists available state files in S3
func (rsm *RemoteStateManager) listS3States(config *RemoteStateConfig) ([]string, error) {
	client := s3.NewFromConfig(*rsm.awsConfig)

	var stateFiles []string
	prefix := strings.TrimSuffix(config.Key, path.Ext(config.Key))

	resp, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(config.Bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	for _, obj := range resp.Contents {
		if strings.HasSuffix(*obj.Key, ".tfstate") {
			stateFiles = append(stateFiles, *obj.Key)
		}
	}

	return stateFiles, nil
}

// listAzureStates lists available state files in Azure Storage
func (rsm *RemoteStateManager) listAzureStates(config *RemoteStateConfig) ([]string, error) {
	// Get Azure credentials
	accountName := os.Getenv("ARM_STORAGE_ACCOUNT_NAME")
	if accountName == "" {
		accountName = os.Getenv("AZURE_STORAGE_ACCOUNT")
	}

	accountKey := os.Getenv("ARM_ACCESS_KEY")
	if accountKey == "" {
		accountKey = os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	}

	if accountName == "" || accountKey == "" {
		// Try using Azure CLI
		cmd := exec.Command("az", "storage", "blob", "list",
			"--container-name", config.Container,
			"--account-name", config.StorageAccount,
			"--query", "[?ends_with(name, '.tfstate')].name",
			"--output", "json")

		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to list Azure blobs via CLI: %w", err)
		}

		var stateFiles []string
		if err := json.Unmarshal(output, &stateFiles); err != nil {
			return nil, fmt.Errorf("failed to parse Azure CLI output: %w", err)
		}

		return stateFiles, nil
	}

	// Use Azure SDK if credentials are available
	// Note: This requires adding Azure Storage SDK dependency
	// For now, we'll use the CLI approach as primary method
	return nil, fmt.Errorf("Azure SDK implementation pending - use Azure CLI")
}

// listGCSStates lists available state files in GCS
func (rsm *RemoteStateManager) listGCSStates(config *RemoteStateConfig) ([]string, error) {
	// Try using gcloud CLI first
	cmd := exec.Command("gcloud", "storage", "ls",
		fmt.Sprintf("gs://%s/", config.Bucket),
		"--format=json")

	output, err := cmd.Output()
	if err != nil {
		// Fallback to gsutil
		cmd = exec.Command("gsutil", "ls",
			fmt.Sprintf("gs://%s/*.tfstate", config.Bucket))
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to list GCS objects via CLI: %w", err)
		}

		// Parse gsutil output (one file per line)
		lines := strings.Split(string(output), "\n")
		var stateFiles []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasSuffix(line, ".tfstate") {
				// Extract just the filename from the full GCS path
				parts := strings.Split(line, "/")
				if len(parts) > 0 {
					stateFiles = append(stateFiles, parts[len(parts)-1])
				}
			}
		}
		return stateFiles, nil
	}

	// Parse gcloud JSON output
	var objects []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(output, &objects); err != nil {
		return nil, fmt.Errorf("failed to parse gcloud output: %w", err)
	}

	var stateFiles []string
	for _, obj := range objects {
		if strings.HasSuffix(obj.Name, ".tfstate") {
			stateFiles = append(stateFiles, obj.Name)
		}
	}

	return stateFiles, nil
}
