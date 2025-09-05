package backend

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// S3Backend implements the Backend interface for AWS S3
type S3Backend struct {
	bucket         string
	key            string
	region         string
	dynamoTable    string
	encrypt        bool
	profile        string
	roleARN        string
	workspace      string
	
	s3Client       *s3.Client
	dynamoClient   *dynamodb.Client
	stsClient      *sts.Client
	
	mu             sync.RWMutex
	config         aws.Config
	metadata       *BackendMetadata
	connectionPool *S3ConnectionPool
}

// NewS3Backend creates a new S3 backend instance
func NewS3Backend(cfg *BackendConfig) (*S3Backend, error) {
	// Extract S3-specific configuration
	bucket, _ := cfg.Config["bucket"].(string)
	key, _ := cfg.Config["key"].(string)
	region, _ := cfg.Config["region"].(string)
	dynamoTable, _ := cfg.Config["dynamodb_table"].(string)
	encrypt, _ := cfg.Config["encrypt"].(bool)
	profile, _ := cfg.Config["profile"].(string)
	roleARN, _ := cfg.Config["role_arn"].(string)
	workspace, _ := cfg.Config["workspace"].(string)
	
	if bucket == "" || key == "" {
		return nil, fmt.Errorf("bucket and key are required for S3 backend")
	}
	
	if workspace == "" {
		workspace = "default"
	}
	
	backend := &S3Backend{
		bucket:      bucket,
		key:         key,
		region:      region,
		dynamoTable: dynamoTable,
		encrypt:     encrypt,
		profile:     profile,
		roleARN:     roleARN,
		workspace:   workspace,
		metadata: &BackendMetadata{
			Type:               "s3",
			SupportsLocking:    dynamoTable != "",
			SupportsVersions:   true,
			SupportsWorkspaces: true,
			Configuration: map[string]string{
				"bucket":         bucket,
				"key":            key,
				"region":         region,
				"dynamodb_table": dynamoTable,
			},
			Workspace: workspace,
			StateKey:  key,
			LockTable: dynamoTable,
		},
	}
	
	// Initialize AWS clients
	if err := backend.initializeClients(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize AWS clients: %w", err)
	}
	
	// Initialize connection pool
	backend.connectionPool = NewS3ConnectionPool(10, 5, 30*time.Minute)
	
	return backend, nil
}

// initializeClients sets up AWS service clients
func (s *S3Backend) initializeClients(ctx context.Context) error {
	var configOptions []func(*config.LoadOptions) error
	
	if s.region != "" {
		configOptions = append(configOptions, config.WithRegion(s.region))
	}
	
	if s.profile != "" {
		configOptions = append(configOptions, config.WithSharedConfigProfile(s.profile))
	}
	
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	// Handle role assumption if specified
	if s.roleARN != "" {
		stsClient := sts.NewFromConfig(cfg)
		result, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         aws.String(s.roleARN),
			RoleSessionName: aws.String("driftmgr-backend"),
		})
		if err != nil {
			return fmt.Errorf("failed to assume role %s: %w", s.roleARN, err)
		}
		
		// Update config with assumed role credentials
		cfg.Credentials = aws.NewCredentialsCache(
			aws.NewStaticCredentialsProvider(
				*result.Credentials.AccessKeyId,
				*result.Credentials.SecretAccessKey,
				*result.Credentials.SessionToken,
			),
		)
	}
	
	s.config = cfg
	s.s3Client = s3.NewFromConfig(cfg)
	s.stsClient = sts.NewFromConfig(cfg)
	
	if s.dynamoTable != "" {
		s.dynamoClient = dynamodb.NewFromConfig(cfg)
	}
	
	return nil
}

// Pull retrieves the current state from S3
func (s *S3Backend) Pull(ctx context.Context) (*StateData, error) {
	stateKey := s.getStateKey()
	
	// Get object from S3
	result, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(stateKey),
	})
	if err != nil {
		// Check if state doesn't exist yet
		if strings.Contains(err.Error(), "NoSuchKey") {
			return &StateData{
				Version:      4, // Terraform state version
				Serial:       0,
				LastModified: time.Now(),
				Data:         []byte("{}"),
			}, nil
		}
		return nil, fmt.Errorf("failed to get state from S3: %w", err)
	}
	defer result.Body.Close()
	
	// Read state data
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state data: %w", err)
	}
	
	// Parse state metadata
	var stateMetadata map[string]interface{}
	if err := json.Unmarshal(data, &stateMetadata); err != nil {
		return nil, fmt.Errorf("failed to parse state metadata: %w", err)
	}
	
	state := &StateData{
		Data:         data,
		LastModified: *result.LastModified,
		Size:         result.ContentLength,
	}
	
	// Extract version and serial from metadata
	if version, ok := stateMetadata["version"].(float64); ok {
		state.Version = int(version)
	}
	if serial, ok := stateMetadata["serial"].(float64); ok {
		state.Serial = uint64(serial)
	}
	if lineage, ok := stateMetadata["lineage"].(string); ok {
		state.Lineage = lineage
	}
	if tfVersion, ok := stateMetadata["terraform_version"].(string); ok {
		state.TerraformVersion = tfVersion
	}
	
	// Calculate checksum
	h := md5.New()
	h.Write(data)
	state.Checksum = base64.StdEncoding.EncodeToString(h.Sum(nil))
	
	return state, nil
}

// Push uploads state to S3
func (s *S3Backend) Push(ctx context.Context, state *StateData) error {
	stateKey := s.getStateKey()
	
	// Prepare the state data
	var data []byte
	if state.Data != nil {
		data = state.Data
	} else {
		// Marshal state if Data is not set
		var err error
		data, err = json.MarshalIndent(state, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal state: %w", err)
		}
	}
	
	// Calculate MD5 for content verification
	h := md5.New()
	h.Write(data)
	contentMD5 := base64.StdEncoding.EncodeToString(h.Sum(nil))
	
	// Prepare put object input
	putInput := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(stateKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
		ContentMD5:  aws.String(contentMD5),
	}
	
	// Add encryption if enabled
	if s.encrypt {
		putInput.ServerSideEncryption = s3types.ServerSideEncryptionAes256
	}
	
	// Add metadata
	putInput.Metadata = map[string]string{
		"terraform-version": state.TerraformVersion,
		"serial":            fmt.Sprintf("%d", state.Serial),
		"lineage":           state.Lineage,
	}
	
	// Upload to S3
	_, err := s.s3Client.PutObject(ctx, putInput)
	if err != nil {
		return fmt.Errorf("failed to push state to S3: %w", err)
	}
	
	return nil
}

// Lock acquires a lock on the state using DynamoDB
func (s *S3Backend) Lock(ctx context.Context, info *LockInfo) (string, error) {
	if s.dynamoTable == "" {
		return "", fmt.Errorf("locking not supported: no DynamoDB table configured")
	}
	
	lockID := fmt.Sprintf("%s-%d", info.ID, time.Now().UnixNano())
	stateKey := s.getStateKey()
	
	// Prepare lock info JSON
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return "", fmt.Errorf("failed to marshal lock info: %w", err)
	}
	
	// Try to acquire lock in DynamoDB
	_, err = s.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.dynamoTable),
		Item: map[string]types.AttributeValue{
			"LockID": &types.AttributeValueMemberS{Value: stateKey},
			"Info":   &types.AttributeValueMemberS{Value: string(infoJSON)},
		},
		ConditionExpression: aws.String("attribute_not_exists(LockID)"),
	})
	
	if err != nil {
		// Check if lock already exists
		if strings.Contains(err.Error(), "ConditionalCheckFailedException") {
			// Get existing lock info
			existingLock, _ := s.GetLockInfo(ctx)
			if existingLock != nil {
				return "", fmt.Errorf("state is already locked by %s since %s", 
					existingLock.Who, existingLock.Created.Format(time.RFC3339))
			}
			return "", fmt.Errorf("state is already locked")
		}
		return "", fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	return lockID, nil
}

// Unlock releases the lock on the state
func (s *S3Backend) Unlock(ctx context.Context, lockID string) error {
	if s.dynamoTable == "" {
		return nil // No locking configured
	}
	
	stateKey := s.getStateKey()
	
	// Delete lock from DynamoDB
	_, err := s.dynamoClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.dynamoTable),
		Key: map[string]types.AttributeValue{
			"LockID": &types.AttributeValueMemberS{Value: stateKey},
		},
	})
	
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	
	return nil
}

// GetVersions returns available state versions using S3 versioning
func (s *S3Backend) GetVersions(ctx context.Context) ([]*StateVersion, error) {
	stateKey := s.getStateKey()
	
	// List object versions
	result, err := s.s3Client.ListObjectVersions(ctx, &s3.ListObjectVersionsInput{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(stateKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list state versions: %w", err)
	}
	
	var versions []*StateVersion
	for i, version := range result.Versions {
		if *version.Key != stateKey {
			continue
		}
		
		sv := &StateVersion{
			ID:        *version.VersionId,
			VersionID: *version.VersionId,
			Created:   *version.LastModified,
			Size:      version.Size,
			IsLatest:  *version.IsLatest,
		}
		
		if version.ETag != nil {
			sv.Checksum = strings.Trim(*version.ETag, "\"")
		}
		
		// Get metadata for serial number
		if i < 5 { // Only get metadata for recent versions to avoid too many API calls
			headResult, err := s.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
				Bucket:    aws.String(s.bucket),
				Key:       aws.String(stateKey),
				VersionId: version.VersionId,
			})
			if err == nil && headResult.Metadata != nil {
				if serial, ok := headResult.Metadata["serial"]; ok {
					var s uint64
					fmt.Sscanf(serial, "%d", &s)
					sv.Serial = s
				}
			}
		}
		
		versions = append(versions, sv)
	}
	
	return versions, nil
}

// GetVersion retrieves a specific version of the state
func (s *S3Backend) GetVersion(ctx context.Context, versionID string) (*StateData, error) {
	stateKey := s.getStateKey()
	
	// Get specific version from S3
	result, err := s.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket:    aws.String(s.bucket),
		Key:       aws.String(stateKey),
		VersionId: aws.String(versionID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get state version %s: %w", versionID, err)
	}
	defer result.Body.Close()
	
	// Read state data
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state data: %w", err)
	}
	
	state := &StateData{
		Data:         data,
		LastModified: *result.LastModified,
		Size:         result.ContentLength,
	}
	
	// Parse state metadata
	var stateMetadata map[string]interface{}
	if err := json.Unmarshal(data, &stateMetadata); err == nil {
		if version, ok := stateMetadata["version"].(float64); ok {
			state.Version = int(version)
		}
		if serial, ok := stateMetadata["serial"].(float64); ok {
			state.Serial = uint64(serial)
		}
		if lineage, ok := stateMetadata["lineage"].(string); ok {
			state.Lineage = lineage
		}
	}
	
	return state, nil
}

// ListWorkspaces returns available workspaces
func (s *S3Backend) ListWorkspaces(ctx context.Context) ([]string, error) {
	// List all state files with env: prefix
	prefix := path.Dir(s.key) + "/env:/"
	
	result, err := s.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(s.bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}
	
	workspaces := []string{"default"}
	for _, prefix := range result.CommonPrefixes {
		parts := strings.Split(strings.TrimSuffix(*prefix.Prefix, "/"), "env:/")
		if len(parts) > 1 {
			workspaces = append(workspaces, parts[len(parts)-1])
		}
	}
	
	return workspaces, nil
}

// SelectWorkspace switches to a different workspace
func (s *S3Backend) SelectWorkspace(ctx context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if workspace exists
	workspaces, err := s.ListWorkspaces(ctx)
	if err != nil {
		return err
	}
	
	found := false
	for _, ws := range workspaces {
		if ws == name {
			found = true
			break
		}
	}
	
	if !found && name != "default" {
		return fmt.Errorf("workspace %s does not exist", name)
	}
	
	s.workspace = name
	s.metadata.Workspace = name
	
	return nil
}

// CreateWorkspace creates a new workspace
func (s *S3Backend) CreateWorkspace(ctx context.Context, name string) error {
	if name == "default" {
		return fmt.Errorf("cannot create default workspace")
	}
	
	// Check if workspace already exists
	workspaces, err := s.ListWorkspaces(ctx)
	if err != nil {
		return err
	}
	
	for _, ws := range workspaces {
		if ws == name {
			return fmt.Errorf("workspace %s already exists", name)
		}
	}
	
	// Create empty state for new workspace
	stateKey := s.getWorkspaceStateKey(name)
	emptyState := &StateData{
		Version: 4,
		Serial:  0,
		Data:    []byte("{}"),
	}
	
	// Save state with workspace key
	oldWorkspace := s.workspace
	s.workspace = name
	err = s.Push(ctx, emptyState)
	s.workspace = oldWorkspace
	
	return err
}

// DeleteWorkspace removes a workspace
func (s *S3Backend) DeleteWorkspace(ctx context.Context, name string) error {
	if name == "default" {
		return fmt.Errorf("cannot delete default workspace")
	}
	
	if s.workspace == name {
		return fmt.Errorf("cannot delete current workspace")
	}
	
	stateKey := s.getWorkspaceStateKey(name)
	
	// Delete the workspace state file
	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(stateKey),
	})
	
	if err != nil {
		return fmt.Errorf("failed to delete workspace %s: %w", name, err)
	}
	
	return nil
}

// GetLockInfo returns current lock information
func (s *S3Backend) GetLockInfo(ctx context.Context) (*LockInfo, error) {
	if s.dynamoTable == "" {
		return nil, nil
	}
	
	stateKey := s.getStateKey()
	
	// Get lock info from DynamoDB
	result, err := s.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.dynamoTable),
		Key: map[string]types.AttributeValue{
			"LockID": &types.AttributeValueMemberS{Value: stateKey},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get lock info: %w", err)
	}
	
	if result.Item == nil {
		return nil, nil // No lock exists
	}
	
	// Extract lock info
	if infoAttr, ok := result.Item["Info"]; ok {
		if infoStr, ok := infoAttr.(*types.AttributeValueMemberS); ok {
			var lockInfo LockInfo
			if err := json.Unmarshal([]byte(infoStr.Value), &lockInfo); err != nil {
				return nil, fmt.Errorf("failed to unmarshal lock info: %w", err)
			}
			return &lockInfo, nil
		}
	}
	
	return nil, nil
}

// Validate checks if the backend is properly configured and accessible
func (s *S3Backend) Validate(ctx context.Context) error {
	// Check S3 bucket access
	_, err := s.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("cannot access S3 bucket %s: %w", s.bucket, err)
	}
	
	// Check DynamoDB table if configured
	if s.dynamoTable != "" {
		_, err = s.dynamoClient.DescribeTable(ctx, &dynamodb.DescribeTableInput{
			TableName: aws.String(s.dynamoTable),
		})
		if err != nil {
			return fmt.Errorf("cannot access DynamoDB table %s: %w", s.dynamoTable, err)
		}
	}
	
	return nil
}

// GetMetadata returns backend metadata
func (s *S3Backend) GetMetadata() *BackendMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metadata
}

// Helper methods

func (s *S3Backend) getStateKey() string {
	if s.workspace == "" || s.workspace == "default" {
		return s.key
	}
	return s.getWorkspaceStateKey(s.workspace)
}

func (s *S3Backend) getWorkspaceStateKey(workspace string) string {
	dir := path.Dir(s.key)
	base := path.Base(s.key)
	return path.Join(dir, fmt.Sprintf("env:/%s", workspace), base)
}

// S3ConnectionPool manages S3 client connections
type S3ConnectionPool struct {
	mu          sync.Mutex
	maxOpen     int
	maxIdle     int
	idleTimeout time.Duration
	connections []poolConn
	stats       PoolStats
}

type poolConn struct {
	client      interface{}
	lastUsed    time.Time
	inUse       bool
}

// NewS3ConnectionPool creates a new connection pool
func NewS3ConnectionPool(maxOpen, maxIdle int, idleTimeout time.Duration) *S3ConnectionPool {
	return &S3ConnectionPool{
		maxOpen:     maxOpen,
		maxIdle:     maxIdle,
		idleTimeout: idleTimeout,
		connections: make([]poolConn, 0, maxOpen),
		stats: PoolStats{
			MaxOpen:     maxOpen,
			MaxIdle:     maxIdle,
			IdleTimeout: idleTimeout,
		},
	}
}