package state

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/catherinevee/driftmgr/internal/models"
)

// RemoteStateManager handles remote Terraform state files
type RemoteStateManager struct {
	awsConfig *aws.Config
}

// RemoteStateConfig represents remote state configuration
type RemoteStateConfig struct {
	Backend     string            `json:"backend"`
	Config      map[string]string `json:"config"`
	Workspace   string            `json:"workspace,omitempty"`
	Key         string            `json:"key"`
	Region      string            `json:"region,omitempty"`
	Bucket      string            `json:"bucket,omitempty"`
	StorageAccount string         `json:"storage_account,omitempty"`
	Container   string            `json:"container,omitempty"`
	Project     string            `json:"project,omitempty"`
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
	// TODO: Implement Azure Storage client
	// This would use Azure Storage SDK to fetch state files
	return nil, fmt.Errorf("Azure Storage support not yet implemented")
}

// parseGCSState parses Terraform state from Google Cloud Storage
func (rsm *RemoteStateManager) parseGCSState(config *RemoteStateConfig) (*models.StateFile, error) {
	// TODO: Implement GCS client
	// This would use Google Cloud Storage SDK to fetch state files
	return nil, fmt.Errorf("Google Cloud Storage support not yet implemented")
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
	// TODO: Implement Azure Storage listing
	return nil, fmt.Errorf("Azure Storage listing not yet implemented")
}

// listGCSStates lists available state files in GCS
func (rsm *RemoteStateManager) listGCSStates(config *RemoteStateConfig) ([]string, error) {
	// TODO: Implement GCS listing
	return nil, fmt.Errorf("GCS listing not yet implemented")
}
