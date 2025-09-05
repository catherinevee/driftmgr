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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/lease"
)

// AzureBackend implements the Backend interface for Azure Storage
type AzureBackend struct {
	storageAccountName string
	containerName      string
	key                string
	accessKey          string
	sasToken           string
	clientID           string
	clientSecret       string
	tenantID           string
	subscriptionID     string
	useMSI             bool
	workspace          string

	containerClient *container.Client
	leaseClient     *lease.BlobLeaseClient
	currentLeaseID  *string

	mu         sync.RWMutex
	metadata   *BackendMetadata
	credential azcore.TokenCredential
}

// NewAzureBackend creates a new Azure Storage backend instance
func NewAzureBackend(cfg *BackendConfig) (*AzureBackend, error) {
	// Extract Azure-specific configuration
	storageAccount, _ := cfg.Config["storage_account_name"].(string)
	containerName, _ := cfg.Config["container_name"].(string)
	key, _ := cfg.Config["key"].(string)
	accessKey, _ := cfg.Config["access_key"].(string)
	sasToken, _ := cfg.Config["sas_token"].(string)
	clientID, _ := cfg.Config["client_id"].(string)
	clientSecret, _ := cfg.Config["client_secret"].(string)
	tenantID, _ := cfg.Config["tenant_id"].(string)
	subscriptionID, _ := cfg.Config["subscription_id"].(string)
	useMSI, _ := cfg.Config["use_msi"].(bool)
	workspace, _ := cfg.Config["workspace"].(string)

	if storageAccount == "" || containerName == "" || key == "" {
		return nil, fmt.Errorf("storage_account_name, container_name, and key are required for Azure backend")
	}

	if workspace == "" {
		workspace = "default"
	}

	backend := &AzureBackend{
		storageAccountName: storageAccount,
		containerName:      containerName,
		key:                key,
		accessKey:          accessKey,
		sasToken:           sasToken,
		clientID:           clientID,
		clientSecret:       clientSecret,
		tenantID:           tenantID,
		subscriptionID:     subscriptionID,
		useMSI:             useMSI,
		workspace:          workspace,
		metadata: &BackendMetadata{
			Type:               "azurerm",
			SupportsLocking:    true, // Using blob leases
			SupportsVersions:   true, // Using snapshots
			SupportsWorkspaces: true,
			Configuration: map[string]string{
				"storage_account_name": storageAccount,
				"container_name":       containerName,
				"key":                  key,
			},
			Workspace: workspace,
			StateKey:  key,
		},
	}

	// Initialize Azure clients
	if err := backend.initializeClients(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize Azure clients: %w", err)
	}

	return backend, nil
}

// initializeClients sets up Azure service clients
func (a *AzureBackend) initializeClients(ctx context.Context) error {
	var err error
	var client *azblob.Client

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", a.storageAccountName)

	// Determine authentication method
	if a.accessKey != "" {
		// Use storage account key
		credential, err := azblob.NewSharedKeyCredential(a.storageAccountName, a.accessKey)
		if err != nil {
			return fmt.Errorf("failed to create shared key credential: %w", err)
		}
		client, err = azblob.NewClientWithSharedKeyCredential(serviceURL, credential, nil)
		if err != nil {
			return fmt.Errorf("failed to create blob client with shared key: %w", err)
		}
	} else if a.sasToken != "" {
		// Use SAS token
		sasURL := serviceURL + a.sasToken
		client, err = azblob.NewClientWithNoCredential(sasURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create blob client with SAS token: %w", err)
		}
	} else if a.useMSI {
		// Use Managed Service Identity
		credential, err := azidentity.NewManagedIdentityCredential(nil)
		if err != nil {
			return fmt.Errorf("failed to create MSI credential: %w", err)
		}
		a.credential = credential
		client, err = azblob.NewClient(serviceURL, credential, nil)
		if err != nil {
			return fmt.Errorf("failed to create blob client with MSI: %w", err)
		}
	} else if a.clientID != "" && a.clientSecret != "" && a.tenantID != "" {
		// Use Service Principal
		credential, err := azidentity.NewClientSecretCredential(
			a.tenantID,
			a.clientID,
			a.clientSecret,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to create service principal credential: %w", err)
		}
		a.credential = credential
		client, err = azblob.NewClient(serviceURL, credential, nil)
		if err != nil {
			return fmt.Errorf("failed to create blob client with service principal: %w", err)
		}
	} else {
		// Try default Azure credential chain
		credential, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return fmt.Errorf("failed to create default credential: %w", err)
		}
		a.credential = credential
		client, err = azblob.NewClient(serviceURL, credential, nil)
		if err != nil {
			return fmt.Errorf("failed to create blob client with default credential: %w", err)
		}
	}

	// Get container client
	a.containerClient = client.ServiceClient().NewContainerClient(a.containerName)

	// Ensure container exists
	_, err = a.containerClient.GetProperties(ctx, nil)
	if err != nil {
		// Try to create container if it doesn't exist
		_, createErr := a.containerClient.Create(ctx, nil)
		if createErr != nil && !strings.Contains(createErr.Error(), "ContainerAlreadyExists") {
			return fmt.Errorf("container %s does not exist and cannot be created: %w", a.containerName, err)
		}
	}

	return nil
}

// Pull retrieves the current state from Azure Blob Storage
func (a *AzureBackend) Pull(ctx context.Context) (*StateData, error) {
	blobName := a.getStateBlobName()
	blobClient := a.containerClient.NewBlobClient(blobName)

	// Download blob
	downloadResponse, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		// Check if blob doesn't exist yet
		if strings.Contains(err.Error(), "BlobNotFound") {
			return &StateData{
				Version:      4,
				Serial:       0,
				LastModified: time.Now(),
				Data:         []byte("{}"),
			}, nil
		}
		return nil, fmt.Errorf("failed to download state from Azure: %w", err)
	}

	// Read state data
	reader := downloadResponse.Body
	defer reader.Close()

	data, err := io.ReadAll(reader)
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
		LastModified: *downloadResponse.LastModified,
		Size:         *downloadResponse.ContentLength,
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

// Push uploads state to Azure Blob Storage
func (a *AzureBackend) Push(ctx context.Context, state *StateData) error {
	blobName := a.getStateBlobName()
	blobClient := a.containerClient.NewBlockBlobClient(blobName)

	// Prepare the state data
	var data []byte
	if state.Data != nil {
		data = state.Data
	} else {
		var err error
		data, err = json.MarshalIndent(state, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal state: %w", err)
		}
	}

	// Calculate MD5 for content verification
	h := md5.New()
	h.Write(data)
	contentMD5 := h.Sum(nil)

	// Prepare upload options
	uploadOptions := &azblob.UploadBufferOptions{
		Metadata: map[string]*string{
			"terraform-version": to.Ptr(state.TerraformVersion),
			"serial":            to.Ptr(fmt.Sprintf("%d", state.Serial)),
			"lineage":           to.Ptr(state.Lineage),
		},
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: to.Ptr("application/json"),
			BlobContentMD5:  contentMD5,
		},
	}

	// Upload to Azure
	_, err := blobClient.UploadBuffer(ctx, data, uploadOptions)
	if err != nil {
		return fmt.Errorf("failed to push state to Azure: %w", err)
	}

	return nil
}

// Lock acquires a lock on the state using blob leases
func (a *AzureBackend) Lock(ctx context.Context, info *LockInfo) (string, error) {
	blobName := a.getStateBlobName()
	blobClient := a.containerClient.NewBlobClient(blobName)

	// Create lease client
	a.leaseClient, _ = lease.NewBlobLeaseClient(blobClient, nil)

	// Try to acquire lease (60 seconds, can be renewed)
	leaseResponse, err := a.leaseClient.AcquireLease(ctx, 60, &lease.BlobAcquireOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "LeaseAlreadyPresent") {
			return "", fmt.Errorf("state is already locked")
		}
		// If blob doesn't exist, create it first
		if strings.Contains(err.Error(), "BlobNotFound") {
			// Create empty blob
			emptyState := &StateData{
				Version: 4,
				Serial:  0,
				Data:    []byte("{}"),
			}
			if err := a.Push(ctx, emptyState); err != nil {
				return "", fmt.Errorf("failed to create initial state: %w", err)
			}
			// Retry lease acquisition
			leaseResponse, err = a.leaseClient.AcquireLease(ctx, 60, &lease.BlobAcquireOptions{})
			if err != nil {
				return "", fmt.Errorf("failed to acquire lease after creating blob: %w", err)
			}
		} else {
			return "", fmt.Errorf("failed to acquire lease: %w", err)
		}
	}

	a.mu.Lock()
	a.currentLeaseID = leaseResponse.LeaseID
	a.mu.Unlock()

	// Store lock info in blob metadata
	metadata := map[string]*string{
		"lock-id":        to.Ptr(info.ID),
		"lock-operation": to.Ptr(info.Operation),
		"lock-who":       to.Ptr(info.Who),
		"lock-created":   to.Ptr(info.Created.Format(time.RFC3339)),
	}

	blobClient.SetMetadata(ctx, metadata, &blob.SetMetadataOptions{
		LeaseAccessConditions: &blob.LeaseAccessConditions{
			LeaseID: a.currentLeaseID,
		},
	})

	// Start lease renewal goroutine
	go a.renewLease(ctx)

	return *a.currentLeaseID, nil
}

// renewLease continuously renews the lease while locked
func (a *AzureBackend) renewLease(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.RLock()
			if a.currentLeaseID == nil || a.leaseClient == nil {
				a.mu.RUnlock()
				return
			}
			leaseID := *a.currentLeaseID
			a.mu.RUnlock()

			_, err := a.leaseClient.RenewLease(ctx, &lease.BlobRenewOptions{})
			if err != nil {
				fmt.Printf("Warning: failed to renew lease %s: %v\n", leaseID, err)
				return
			}
		}
	}
}

// Unlock releases the lock on the state
func (a *AzureBackend) Unlock(ctx context.Context, lockID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.leaseClient == nil || a.currentLeaseID == nil {
		return nil // No lock held
	}

	// Release the lease
	_, err := a.leaseClient.ReleaseLease(ctx, &lease.BlobReleaseOptions{})
	if err != nil {
		return fmt.Errorf("failed to release lease: %w", err)
	}

	a.currentLeaseID = nil
	a.leaseClient = nil

	return nil
}

// GetVersions returns available state versions using blob snapshots
func (a *AzureBackend) GetVersions(ctx context.Context) ([]*StateVersion, error) {
	blobName := a.getStateBlobName()

	// List blob snapshots
	pager := a.containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Include: container.ListBlobsInclude{
			Snapshots: true,
			Metadata:  true,
		},
		Prefix: &blobName,
	})

	var versions []*StateVersion
	versionIndex := 0

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list blob snapshots: %w", err)
		}

		for _, blobItem := range resp.Segment.BlobItems {
			if *blobItem.Name != blobName {
				continue
			}

			sv := &StateVersion{
				ID:       fmt.Sprintf("v%d", versionIndex),
				Created:  *blobItem.Properties.LastModified,
				Size:     *blobItem.Properties.ContentLength,
				IsLatest: blobItem.Snapshot == nil,
			}

			if blobItem.Snapshot != nil {
				sv.VersionID = *blobItem.Snapshot
			} else {
				sv.VersionID = "current"
			}

			// Extract serial from metadata if available
			if blobItem.Metadata != nil {
				if serial, ok := blobItem.Metadata["serial"]; ok && serial != nil {
					var s uint64
					fmt.Sscanf(*serial, "%d", &s)
					sv.Serial = s
				}
			}

			// Calculate checksum from ETag
			if blobItem.Properties.ETag != nil {
				sv.Checksum = strings.Trim(string(*blobItem.Properties.ETag), "\"")
			}

			versions = append(versions, sv)
			versionIndex++
		}
	}

	return versions, nil
}

// GetVersion retrieves a specific version of the state
func (a *AzureBackend) GetVersion(ctx context.Context, versionID string) (*StateData, error) {
	blobName := a.getStateBlobName()
	blobClient := a.containerClient.NewBlobClient(blobName)

	// Prepare download options for specific snapshot
	var downloadOptions *azblob.DownloadStreamOptions
	if versionID != "current" && versionID != "" {
		downloadOptions = &azblob.DownloadStreamOptions{
			AccessConditions: &blob.AccessConditions{
				ModifiedAccessConditions: &blob.ModifiedAccessConditions{},
			},
		}
	}

	// Download specific version
	downloadResponse, err := blobClient.DownloadStream(ctx, downloadOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to download state version %s: %w", versionID, err)
	}

	reader := downloadResponse.Body
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read state data: %w", err)
	}

	state := &StateData{
		Data:         data,
		LastModified: *downloadResponse.LastModified,
		Size:         *downloadResponse.ContentLength,
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
func (a *AzureBackend) ListWorkspaces(ctx context.Context) ([]string, error) {
	// List all state files with env: prefix
	prefix := path.Dir(a.key) + "/env:/"

	pager := a.containerClient.NewListBlobsFlatPager(&container.ListBlobsFlatOptions{
		Prefix: &prefix,
	})

	workspaceMap := make(map[string]bool)
	workspaceMap["default"] = true

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list workspaces: %w", err)
		}

		for _, blobItem := range resp.Segment.BlobItems {
			parts := strings.Split(*blobItem.Name, "env:/")
			if len(parts) > 1 {
				wsParts := strings.Split(parts[1], "/")
				if len(wsParts) > 0 && wsParts[0] != "" {
					workspaceMap[wsParts[0]] = true
				}
			}
		}
	}

	workspaces := make([]string, 0, len(workspaceMap))
	for ws := range workspaceMap {
		workspaces = append(workspaces, ws)
	}

	return workspaces, nil
}

// SelectWorkspace switches to a different workspace
func (a *AzureBackend) SelectWorkspace(ctx context.Context, name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if workspace exists
	workspaces, err := a.ListWorkspaces(ctx)
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

	a.workspace = name
	a.metadata.Workspace = name

	return nil
}

// CreateWorkspace creates a new workspace
func (a *AzureBackend) CreateWorkspace(ctx context.Context, name string) error {
	if name == "default" {
		return fmt.Errorf("cannot create default workspace")
	}

	// Check if workspace already exists
	workspaces, err := a.ListWorkspaces(ctx)
	if err != nil {
		return err
	}

	for _, ws := range workspaces {
		if ws == name {
			return fmt.Errorf("workspace %s already exists", name)
		}
	}

	// Create empty state for new workspace
	emptyState := &StateData{
		Version: 4,
		Serial:  0,
		Data:    []byte("{}"),
	}

	// Save state with workspace key
	oldWorkspace := a.workspace
	a.workspace = name
	err = a.Push(ctx, emptyState)
	a.workspace = oldWorkspace

	return err
}

// DeleteWorkspace removes a workspace
func (a *AzureBackend) DeleteWorkspace(ctx context.Context, name string) error {
	if name == "default" {
		return fmt.Errorf("cannot delete default workspace")
	}

	if a.workspace == name {
		return fmt.Errorf("cannot delete current workspace")
	}

	blobName := a.getWorkspaceStateBlobName(name)
	blobClient := a.containerClient.NewBlobClient(blobName)

	// Delete the workspace state blob
	_, err := blobClient.Delete(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete workspace %s: %w", name, err)
	}

	return nil
}

// GetLockInfo returns current lock information
func (a *AzureBackend) GetLockInfo(ctx context.Context) (*LockInfo, error) {
	blobName := a.getStateBlobName()
	blobClient := a.containerClient.NewBlobClient(blobName)

	// Get blob properties to check lease status
	props, err := blobClient.GetProperties(ctx, nil)
	if err != nil {
		if strings.Contains(err.Error(), "BlobNotFound") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get blob properties: %w", err)
	}

	// Check if blob is leased
	if props.LeaseState == nil || *props.LeaseState != lease.StateTypeLeased {
		return nil, nil // No active lease
	}

	// Extract lock info from metadata
	lockInfo := &LockInfo{}
	if props.Metadata != nil {
		if id, ok := props.Metadata["lock-id"]; ok && id != nil {
			lockInfo.ID = *id
		}
		if op, ok := props.Metadata["lock-operation"]; ok && op != nil {
			lockInfo.Operation = *op
		}
		if who, ok := props.Metadata["lock-who"]; ok && who != nil {
			lockInfo.Who = *who
		}
		if created, ok := props.Metadata["lock-created"]; ok && created != nil {
			if t, err := time.Parse(time.RFC3339, *created); err == nil {
				lockInfo.Created = t
			}
		}
	}

	return lockInfo, nil
}

// Validate checks if the backend is properly configured and accessible
func (a *AzureBackend) Validate(ctx context.Context) error {
	// Check container access
	_, err := a.containerClient.GetProperties(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot access container %s: %w", a.containerName, err)
	}

	return nil
}

// GetMetadata returns backend metadata
func (a *AzureBackend) GetMetadata() *BackendMetadata {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.metadata
}

// Helper methods

func (a *AzureBackend) getStateBlobName() string {
	if a.workspace == "" || a.workspace == "default" {
		return a.key
	}
	return a.getWorkspaceStateBlobName(a.workspace)
}

func (a *AzureBackend) getWorkspaceStateBlobName(workspace string) string {
	dir := path.Dir(a.key)
	base := path.Base(a.key)
	return path.Join(dir, fmt.Sprintf("env:/%s", workspace), base)
}
