package terraform

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// BackendConfig represents a Terraform backend configuration
type BackendConfig struct {
	Type          string             `json:"type"`
	WorkingDir    string             `json:"working_dir"`
	ConfigFile    string             `json:"config_file"`
	IsInitialized bool               `json:"is_initialized"`
	IsStateFile   bool               `json:"is_state_file"`
	IsTerragrunt  bool               `json:"is_terragrunt"`
	Workspaces    []string           `json:"workspaces"`
	Configuration map[string]string  `json:"configuration"`
	Environment   string             `json:"environment"`
	Region        string             `json:"region"`
	ResourceType  string             `json:"resource_type"`
	RemoteState   *RemoteStateConfig `json:"remote_state,omitempty"`
	StateContent  []byte             `json:"-"` // Cached state content
}

// RemoteStateConfig represents Terragrunt remote state configuration
type RemoteStateConfig struct {
	Backend string            `json:"backend"`
	Config  map[string]string `json:"config"`
}

// GetStateFilePath returns the state file path for a backend
func (b *BackendConfig) GetStateFilePath(workspace string) string {
	if b.IsStateFile {
		return b.ConfigFile
	}
	if workspace == "" {
		workspace = "default"
	}
	return filepath.Join(b.WorkingDir, fmt.Sprintf("terraform.%s.tfstate", workspace))
}

// BackendScanner scans for Terraform backends
type BackendScanner struct {
	rootDir string
}

// NewBackendScanner creates a new backend scanner
func NewBackendScanner(rootDir string) *BackendScanner {
	return &BackendScanner{
		rootDir: rootDir,
	}
}

// ScanDirectory scans a directory for backend configurations
func (s *BackendScanner) ScanDirectory() ([]*BackendConfig, error) {
	configs := []*BackendConfig{}

	// Scan for both Terraform state files and Terragrunt configurations
	err := filepath.Walk(s.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Check for Terraform state files
		if filepath.Ext(path) == ".tfstate" {
			config := &BackendConfig{
				Type:        "local",
				WorkingDir:  filepath.Dir(path),
				ConfigFile:  path,
				IsStateFile: true,
				Configuration: map[string]string{
					"path": path,
				},
			}
			configs = append(configs, config)
		}

		// Check for Terragrunt files
		if filepath.Base(path) == "terragrunt.hcl" {
			config := &BackendConfig{
				Type:         "terragrunt",
				WorkingDir:   filepath.Dir(path),
				ConfigFile:   path,
				IsTerragrunt: true,
				Configuration: map[string]string{
					"path": path,
					"dir":  filepath.Dir(path),
				},
			}

			// Parse Terragrunt file for remote state configuration
			if remoteState := s.parseTerragruntRemoteState(path); remoteState != nil {
				config.RemoteState = remoteState
			}

			configs = append(configs, config)
		}

		return nil
	})

	return configs, err
}

// parseTerragruntRemoteState parses remote_state block from terragrunt.hcl
func (s *BackendScanner) parseTerragruntRemoteState(path string) *RemoteStateConfig {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inRemoteState := false
	braceCount := 0
	var backend string
	config := make(map[string]string)

	// Regex patterns for parsing
	backendPattern := regexp.MustCompile(`backend\s*=\s*"([^"]+)"`)
	bucketPattern := regexp.MustCompile(`bucket\s*=\s*"([^"]+)"`)
	keyPattern := regexp.MustCompile(`key\s*=\s*"([^"]+)"`)
	regionPattern := regexp.MustCompile(`region\s*=\s*"([^"]+)"`)
	containerPattern := regexp.MustCompile(`container_name\s*=\s*"([^"]+)"`)
	storageAccountPattern := regexp.MustCompile(`storage_account_name\s*=\s*"([^"]+)"`)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if we're entering remote_state block
		if strings.Contains(line, "remote_state") && strings.Contains(line, "{") {
			inRemoteState = true
			braceCount = 1
			continue
		}

		if inRemoteState {
			// Count braces to track nesting
			braceCount += strings.Count(line, "{")
			braceCount -= strings.Count(line, "}")

			if braceCount <= 0 {
				inRemoteState = false
				break
			}

			// Parse backend type
			if matches := backendPattern.FindStringSubmatch(line); len(matches) > 1 {
				backend = matches[1]
			}

			// Parse S3 configuration
			if matches := bucketPattern.FindStringSubmatch(line); len(matches) > 1 {
				config["bucket"] = matches[1]
			}
			if matches := keyPattern.FindStringSubmatch(line); len(matches) > 1 {
				// Handle ${path_relative_to_include()} syntax
				key := matches[1]
				if strings.Contains(key, "path_relative_to_include") {
					// Calculate relative path from root terragrunt
					relPath, _ := filepath.Rel(s.rootDir, filepath.Dir(path))
					if relPath == "." {
						relPath = ""
					}
					key = strings.Replace(key, "${path_relative_to_include()}", relPath, -1)
					key = strings.Replace(key, "path_relative_to_include()", relPath, -1)
					key = strings.TrimPrefix(key, "/")
				}
				config["key"] = key
			}
			if matches := regionPattern.FindStringSubmatch(line); len(matches) > 1 {
				config["region"] = matches[1]
			}

			// Parse Azure configuration
			if matches := containerPattern.FindStringSubmatch(line); len(matches) > 1 {
				config["container"] = matches[1]
			}
			if matches := storageAccountPattern.FindStringSubmatch(line); len(matches) > 1 {
				config["storage_account"] = matches[1]
			}
		}
	}

	if backend != "" {
		return &RemoteStateConfig{
			Backend: backend,
			Config:  config,
		}
	}

	return nil
}

// RetrieveRemoteState downloads state from cloud backend
func (b *BackendConfig) RetrieveRemoteState(ctx context.Context) ([]byte, error) {
	if b.RemoteState == nil {
		return nil, fmt.Errorf("no remote state configuration")
	}

	switch b.RemoteState.Backend {
	case "s3":
		return b.retrieveS3State(ctx)
	case "azurerm":
		return b.retrieveAzureState(ctx)
	case "gcs":
		return b.retrieveGCSState(ctx)
	default:
		return nil, fmt.Errorf("unsupported backend: %s", b.RemoteState.Backend)
	}
}

// retrieveS3State downloads state from S3
func (b *BackendConfig) retrieveS3State(ctx context.Context) ([]byte, error) {
	bucket := b.RemoteState.Config["bucket"]
	key := b.RemoteState.Config["key"]
	region := b.RemoteState.Config["region"]

	if bucket == "" || key == "" {
		return nil, fmt.Errorf("missing S3 bucket or key")
	}

	// If no region specified, use default
	if region == "" {
		region = "us-east-1"
	}

	// Create AWS session using local credentials
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Create S3 client
	svc := s3.New(sess)

	// First, try to get the bucket location to ensure we're using the right region
	locResult, err := svc.GetBucketLocation(&s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	})
	if err == nil && locResult.LocationConstraint != nil {
		// Bucket is in a different region, recreate session
		actualRegion := *locResult.LocationConstraint
		if actualRegion == "" {
			actualRegion = "us-east-1" // Empty constraint means us-east-1
		}
		if actualRegion != region {
			sess, err = session.NewSession(&aws.Config{
				Region: aws.String(actualRegion),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create AWS session for region %s: %w", actualRegion, err)
			}
			svc = s3.New(sess)
		}
	}

	// Download state file
	result, err := svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// If the key doesn't exist, return a more helpful error
		if strings.Contains(err.Error(), "NoSuchKey") {
			return nil, fmt.Errorf("state file not found at s3://%s/%s", bucket, key)
		}
		return nil, fmt.Errorf("failed to download state from S3: %w", err)
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

// retrieveAzureState downloads state from Azure Blob Storage
func (b *BackendConfig) retrieveAzureState(ctx context.Context) ([]byte, error) {
	containerName := b.RemoteState.Config["container"]
	storageAccount := b.RemoteState.Config["storage_account"]
	key := b.RemoteState.Config["key"]

	if containerName == "" || storageAccount == "" {
		return nil, fmt.Errorf("missing Azure storage configuration (container/storage_account)")
	}

	// Default key if not specified
	if key == "" {
		key = "terraform.tfstate"
	}

	// Try multiple authentication methods
	accountKey := os.Getenv("ARM_ACCESS_KEY")
	if accountKey == "" {
		accountKey = os.Getenv("AZURE_STORAGE_ACCESS_KEY")
	}
	if accountKey == "" {
		accountKey = os.Getenv("AZURE_STORAGE_KEY")
	}
	if accountKey == "" {
		// TODO: Try Azure CLI authentication
		return nil, fmt.Errorf("Azure storage key not found. Set ARM_ACCESS_KEY or AZURE_STORAGE_ACCESS_KEY environment variable")
	}

	// Create Azure pipeline
	credential, err := azblob.NewSharedKeyCredential(storageAccount, accountKey)
	if err != nil {
		return nil, fmt.Errorf("invalid Azure credentials: %w", err)
	}

	pipeline := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	// Create URL
	u, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", storageAccount, containerName, key))
	blobURL := azblob.NewBlobURL(*u, pipeline)

	// Download blob
	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to download state from Azure: %w", err)
	}

	body := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: 3})
	defer body.Close()

	return io.ReadAll(body)
}

// retrieveGCSState downloads state from Google Cloud Storage
func (b *BackendConfig) retrieveGCSState(ctx context.Context) ([]byte, error) {
	bucket := b.RemoteState.Config["bucket"]
	prefix := b.RemoteState.Config["prefix"]
	path := b.RemoteState.Config["path"]

	if bucket == "" {
		return nil, fmt.Errorf("missing GCS bucket")
	}

	// Create GCS client using default credentials (gcloud auth)
	client, err := storage.NewClient(ctx)
	if err != nil {
		// Provide more helpful error message
		return nil, fmt.Errorf("failed to create GCS client. Ensure you're authenticated with 'gcloud auth application-default login': %w", err)
	}
	defer client.Close()

	// Build object name - try different key patterns
	var objectName string
	if path != "" {
		objectName = path
	} else if prefix != "" {
		objectName = fmt.Sprintf("%s/terraform.tfstate", prefix)
	} else {
		objectName = "terraform.tfstate"
	}

	// Try to download object
	reader, err := client.Bucket(bucket).Object(objectName).NewReader(ctx)
	if err != nil {
		// If not found, try default.tfstate
		if strings.Contains(err.Error(), "object doesn't exist") && objectName != "default.tfstate" {
			objectName = "default.tfstate"
			if prefix != "" {
				objectName = fmt.Sprintf("%s/default.tfstate", prefix)
			}
			reader, err = client.Bucket(bucket).Object(objectName).NewReader(ctx)
			if err != nil {
				return nil, fmt.Errorf("state file not found at gs://%s/%s or gs://%s/default.tfstate", bucket, path, bucket)
			}
		} else {
			return nil, fmt.Errorf("failed to download state from GCS: %w", err)
		}
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// HasRemoteState checks if this backend has remote state configuration
func (b *BackendConfig) HasRemoteState() bool {
	return b.RemoteState != nil && b.RemoteState.Backend != ""
}
