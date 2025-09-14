package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Backend interface for state storage operations
type Backend interface {
	Connect(ctx context.Context) error
	GetState(ctx context.Context, key string) ([]byte, error)
	PutState(ctx context.Context, key string, data []byte) error
	DeleteState(ctx context.Context, key string) error
	ListStates(ctx context.Context) ([]string, error)
	LockState(ctx context.Context, key string) (string, error)
	UnlockState(ctx context.Context, key string, lockID string) error
}

// BackendType represents the type of backend
type BackendType string

const (
	BackendLocal     BackendType = "local"
	BackendS3        BackendType = "s3"
	BackendAzureBlob BackendType = "azurerm"
	BackendGCS       BackendType = "gcs"
	BackendRemote    BackendType = "remote"
)

// LocalBackend implements local file system backend
type LocalBackend struct {
	path     string
	lockFile string
}

// NewLocalBackend creates a new local backend
func NewLocalBackend(path string) *LocalBackend {
	return &LocalBackend{
		path:     path,
		lockFile: path + ".lock",
	}
}

// Connect initializes the local backend
func (b *LocalBackend) Connect(ctx context.Context) error {
	// Ensure directory exists
	dir := filepath.Dir(b.path)
	return os.MkdirAll(dir, 0755)
}

// GetState retrieves state from local file
func (b *LocalBackend) GetState(ctx context.Context, key string) ([]byte, error) {
	path := b.path
	if key != "" {
		path = filepath.Join(filepath.Dir(b.path), key)
	}
	return os.ReadFile(path)
}

// PutState writes state to local file
func (b *LocalBackend) PutState(ctx context.Context, key string, data []byte) error {
	path := b.path
	if key != "" {
		path = filepath.Join(filepath.Dir(b.path), key)
	}

	// Write to temp file first
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempPath, path)
}

// DeleteState removes state file
func (b *LocalBackend) DeleteState(ctx context.Context, key string) error {
	path := b.path
	if key != "" {
		path = filepath.Join(filepath.Dir(b.path), key)
	}
	err := os.Remove(path)
	if err != nil && os.IsNotExist(err) {
		// File doesn't exist, which is fine for delete operations
		return nil
	}
	return err
}

// ListStates lists all state files
func (b *LocalBackend) ListStates(ctx context.Context) ([]string, error) {
	dir := filepath.Dir(b.path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var states []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".tfstate" {
			states = append(states, entry.Name())
		}
	}
	return states, nil
}

// LockState creates a lock file
func (b *LocalBackend) LockState(ctx context.Context, key string) (string, error) {
	lockID := fmt.Sprintf("lock-%d", time.Now().Unix())
	lockData := map[string]interface{}{
		"ID":      lockID,
		"Created": time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(lockData)
	if err != nil {
		return "", err
	}

	// Try to create lock file exclusively
	f, err := os.OpenFile(b.lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("state is already locked")
		}
		return "", err
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		os.Remove(b.lockFile)
		return "", err
	}

	return lockID, nil
}

// UnlockState removes the lock file
func (b *LocalBackend) UnlockState(ctx context.Context, key string, lockID string) error {
	// Check if lock file exists
	if _, err := os.Stat(b.lockFile); os.IsNotExist(err) {
		// Lock file doesn't exist, consider it already unlocked
		return nil
	}

	// Verify lock ID matches
	data, err := os.ReadFile(b.lockFile)
	if err != nil {
		return err
	}

	var lockData map[string]interface{}
	if err := json.Unmarshal(data, &lockData); err != nil {
		return err
	}

	if lockData["ID"] != lockID {
		return fmt.Errorf("lock ID mismatch")
	}

	return os.Remove(b.lockFile)
}

// S3Backend implements AWS S3 backend
type S3Backend struct {
	bucket        string
	key           string
	region        string
	dynamoDBTable string
	client        *s3.Client
	dynamoClient  *dynamodb.Client
}

// NewS3Backend creates a new S3 backend
func NewS3Backend(bucket, key, region, dynamoDBTable string) *S3Backend {
	return &S3Backend{
		bucket:        bucket,
		key:           key,
		region:        region,
		dynamoDBTable: dynamoDBTable,
	}
}

// Connect initializes AWS clients
func (b *S3Backend) Connect(ctx context.Context) error {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(b.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	b.client = s3.NewFromConfig(cfg)

	if b.dynamoDBTable != "" {
		b.dynamoClient = dynamodb.NewFromConfig(cfg)
	}

	return nil
}

// GetState retrieves state from S3
func (b *S3Backend) GetState(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		key = b.key
	}

	resp, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get state from S3: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// PutState uploads state to S3
func (b *S3Backend) PutState(ctx context.Context, key string, data []byte) error {
	if key == "" {
		key = b.key
	}

	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})

	if err != nil {
		return fmt.Errorf("failed to put state to S3: %w", err)
	}

	return nil
}

// DeleteState removes state from S3
func (b *S3Backend) DeleteState(ctx context.Context, key string) error {
	if key == "" {
		key = b.key
	}

	_, err := b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete state from S3: %w", err)
	}

	return nil
}

// ListStates lists all state files in S3
func (b *S3Backend) ListStates(ctx context.Context) ([]string, error) {
	resp, err := b.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(b.bucket),
		Prefix: aws.String(filepath.Dir(b.key)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list states from S3: %w", err)
	}

	var states []string
	for _, obj := range resp.Contents {
		if filepath.Ext(*obj.Key) == ".tfstate" {
			states = append(states, *obj.Key)
		}
	}

	return states, nil
}

// LockState creates a DynamoDB lock
func (b *S3Backend) LockState(ctx context.Context, key string) (string, error) {
	if b.dynamoClient == nil {
		return "", fmt.Errorf("DynamoDB table not configured for state locking")
	}

	lockID := fmt.Sprintf("s3://%s/%s", b.bucket, key)
	lockInfo := map[string]interface{}{
		"ID":      lockID,
		"Created": time.Now().Unix(),
	}

	infoBytes, _ := json.Marshal(lockInfo)

	_, err := b.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(b.dynamoDBTable),
		Item: map[string]types.AttributeValue{
			"LockID": &types.AttributeValueMemberS{Value: lockID},
			"Info":   &types.AttributeValueMemberS{Value: string(infoBytes)},
		},
		ConditionExpression: aws.String("attribute_not_exists(LockID)"),
	})

	if err != nil {
		return "", fmt.Errorf("failed to acquire state lock: %w", err)
	}

	return lockID, nil
}

// UnlockState removes the DynamoDB lock
func (b *S3Backend) UnlockState(ctx context.Context, key string, lockID string) error {
	if b.dynamoClient == nil {
		return nil
	}

	_, err := b.dynamoClient.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(b.dynamoDBTable),
		Key: map[string]types.AttributeValue{
			"LockID": &types.AttributeValueMemberS{Value: lockID},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to release state lock: %w", err)
	}

	return nil
}

// BackendFactory creates backends based on configuration
type BackendFactory struct{}

// NewBackendFactory creates a new backend factory
func NewBackendFactory() *BackendFactory {
	return &BackendFactory{}
}

// CreateBackend creates a backend based on type and config
func (f *BackendFactory) CreateBackend(backendType BackendType, config map[string]interface{}) (Backend, error) {
	switch backendType {
	case BackendLocal:
		path, ok := config["path"].(string)
		if !ok {
			path = "terraform.tfstate"
		}
		return NewLocalBackend(path), nil

	case BackendS3:
		bucket, _ := config["bucket"].(string)
		key, _ := config["key"].(string)
		region, _ := config["region"].(string)
		dynamoTable, _ := config["dynamodb_table"].(string)

		if bucket == "" || key == "" {
			return nil, fmt.Errorf("S3 backend requires bucket and key")
		}

		if region == "" {
			region = "us-east-1"
		}

		return NewS3Backend(bucket, key, region, dynamoTable), nil

	default:
		return nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}
}
