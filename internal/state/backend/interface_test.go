package backend

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBackend is a test implementation of the Backend interface
type MockBackend struct {
	states     map[string]*StateData
	workspaces map[string]map[string]*StateData // workspace -> key -> state
	locks      map[string]*LockInfo
	versions   map[string][]*StateVersion
	metadata   *BackendMetadata

	// Control behavior for testing
	pullError     error
	pushError     error
	lockError     error
	unlockError   error
	validateError error

	// Track method calls
	pullCalls     int
	pushCalls     int
	lockCalls     int
	unlockCalls   int
	validateCalls int
}

func NewMockBackend() *MockBackend {
	return &MockBackend{
		states:     make(map[string]*StateData),
		workspaces: make(map[string]map[string]*StateData),
		locks:      make(map[string]*LockInfo),
		versions:   make(map[string][]*StateVersion),
		metadata: &BackendMetadata{
			Type:               "mock",
			SupportsLocking:    true,
			SupportsVersions:   true,
			SupportsWorkspaces: true,
			Configuration: map[string]string{
				"type": "mock",
			},
			Workspace: "default",
		},
	}
}

func (m *MockBackend) Pull(ctx context.Context) (*StateData, error) {
	m.pullCalls++
	if m.pullError != nil {
		return nil, m.pullError
	}

	ws := m.metadata.Workspace
	if ws == "" {
		ws = "default"
	}

	if wsStates, exists := m.workspaces[ws]; exists {
		if state, exists := wsStates["terraform.tfstate"]; exists {
			return state, nil
		}
	}

	// Return empty state if not found
	return &StateData{
		Version:      4,
		Serial:       0,
		Lineage:      "test-lineage",
		Data:         []byte(`{"version": 4, "serial": 0, "resources": [], "outputs": {}}`),
		LastModified: time.Now(),
		Size:         100,
	}, nil
}

func (m *MockBackend) Push(ctx context.Context, state *StateData) error {
	m.pushCalls++
	if m.pushError != nil {
		return m.pushError
	}

	ws := m.metadata.Workspace
	if ws == "" {
		ws = "default"
	}

	if m.workspaces[ws] == nil {
		m.workspaces[ws] = make(map[string]*StateData)
	}

	// Add to versions
	versionID := time.Now().Format(time.RFC3339)
	version := &StateVersion{
		ID:        versionID,
		VersionID: versionID,
		Serial:    state.Serial,
		Created:   time.Now(),
		Size:      state.Size,
		IsLatest:  true,
	}

	key := "terraform.tfstate"
	m.versions[key] = append(m.versions[key], version)
	m.workspaces[ws][key] = state

	return nil
}

func (m *MockBackend) Lock(ctx context.Context, info *LockInfo) (string, error) {
	m.lockCalls++
	if m.lockError != nil {
		return "", m.lockError
	}

	lockID := "mock-lock-" + time.Now().Format("20060102150405")
	m.locks[lockID] = info

	return lockID, nil
}

func (m *MockBackend) Unlock(ctx context.Context, lockID string) error {
	m.unlockCalls++
	if m.unlockError != nil {
		return m.unlockError
	}

	delete(m.locks, lockID)
	return nil
}

func (m *MockBackend) GetVersions(ctx context.Context) ([]*StateVersion, error) {
	key := "terraform.tfstate"
	if versions, exists := m.versions[key]; exists {
		return versions, nil
	}
	return []*StateVersion{}, nil
}

func (m *MockBackend) GetVersion(ctx context.Context, versionID string) (*StateData, error) {
	// For mock, just return current state
	return m.Pull(ctx)
}

func (m *MockBackend) ListWorkspaces(ctx context.Context) ([]string, error) {
	workspaces := []string{"default"}
	for ws := range m.workspaces {
		if ws != "default" {
			workspaces = append(workspaces, ws)
		}
	}
	return workspaces, nil
}

func (m *MockBackend) SelectWorkspace(ctx context.Context, name string) error {
	m.metadata.Workspace = name
	return nil
}

func (m *MockBackend) CreateWorkspace(ctx context.Context, name string) error {
	if name == "default" {
		return nil
	}

	if m.workspaces[name] == nil {
		m.workspaces[name] = make(map[string]*StateData)
	}

	return nil
}

func (m *MockBackend) DeleteWorkspace(ctx context.Context, name string) error {
	if name == "default" {
		return nil
	}

	delete(m.workspaces, name)
	return nil
}

func (m *MockBackend) GetLockInfo(ctx context.Context) (*LockInfo, error) {
	for _, lock := range m.locks {
		return lock, nil
	}
	return nil, nil
}

func (m *MockBackend) Validate(ctx context.Context) error {
	m.validateCalls++
	return m.validateError
}

func (m *MockBackend) GetMetadata() *BackendMetadata {
	return m.metadata
}

// Test Backend Interface Implementation
func TestBackendInterface(t *testing.T) {
	ctx := context.Background()
	backend := NewMockBackend()

	t.Run("Pull and Push operations", func(t *testing.T) {
		// Test initial pull
		state, err := backend.Pull(ctx)
		require.NoError(t, err)
		assert.NotNil(t, state)
		assert.Equal(t, 4, state.Version)
		assert.Equal(t, uint64(0), state.Serial)

		// Test push
		newState := &StateData{
			Version:          4,
			TerraformVersion: "1.0.0",
			Serial:           1,
			Lineage:          "test-lineage",
			Data:             []byte(`{"version": 4, "serial": 1, "resources": [], "outputs": {}}`),
			LastModified:     time.Now(),
			Size:             120,
		}

		err = backend.Push(ctx, newState)
		require.NoError(t, err)

		// Test pull after push
		pulledState, err := backend.Pull(ctx)
		require.NoError(t, err)
		assert.Equal(t, newState.Serial, pulledState.Serial)
		assert.Equal(t, newState.TerraformVersion, pulledState.TerraformVersion)
	})

	t.Run("Lock and Unlock operations", func(t *testing.T) {
		lockInfo := &LockInfo{
			ID:        "test-lock",
			Path:      "terraform.tfstate",
			Operation: "plan",
			Who:       "test-user",
			Version:   "1.0.0",
			Created:   time.Now(),
			Info:      "Test lock",
		}

		// Test lock
		lockID, err := backend.Lock(ctx, lockInfo)
		require.NoError(t, err)
		assert.NotEmpty(t, lockID)

		// Test get lock info
		info, err := backend.GetLockInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, lockInfo.ID, info.ID)

		// Test unlock
		err = backend.Unlock(ctx, lockID)
		require.NoError(t, err)

		// Verify lock is gone
		info, err = backend.GetLockInfo(ctx)
		require.NoError(t, err)
		assert.Nil(t, info)
	})

	t.Run("Workspace operations", func(t *testing.T) {
		// Test list workspaces
		workspaces, err := backend.ListWorkspaces(ctx)
		require.NoError(t, err)
		assert.Contains(t, workspaces, "default")

		// Test create workspace
		err = backend.CreateWorkspace(ctx, "test-workspace")
		require.NoError(t, err)

		workspaces, err = backend.ListWorkspaces(ctx)
		require.NoError(t, err)
		assert.Contains(t, workspaces, "test-workspace")

		// Test select workspace
		err = backend.SelectWorkspace(ctx, "test-workspace")
		require.NoError(t, err)

		// Test delete workspace
		err = backend.SelectWorkspace(ctx, "default")
		require.NoError(t, err)

		err = backend.DeleteWorkspace(ctx, "test-workspace")
		require.NoError(t, err)

		workspaces, err = backend.ListWorkspaces(ctx)
		require.NoError(t, err)
		assert.NotContains(t, workspaces, "test-workspace")
	})

	t.Run("Version operations", func(t *testing.T) {
		// Push a state to create versions
		state := &StateData{
			Version:          4,
			TerraformVersion: "1.0.0",
			Serial:           1,
			Lineage:          "test-lineage",
			Data:             []byte(`{"version": 4, "serial": 1, "resources": [], "outputs": {}}`),
			LastModified:     time.Now(),
			Size:             120,
		}

		err := backend.Push(ctx, state)
		require.NoError(t, err)

		// Test get versions
		versions, err := backend.GetVersions(ctx)
		require.NoError(t, err)
		assert.Len(t, versions, 1)
		assert.Equal(t, uint64(1), versions[0].Serial)

		// Test get specific version
		versionState, err := backend.GetVersion(ctx, versions[0].VersionID)
		require.NoError(t, err)
		assert.NotNil(t, versionState)
		assert.Equal(t, uint64(1), versionState.Serial)
	})

	t.Run("Validation", func(t *testing.T) {
		err := backend.Validate(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, backend.validateCalls)
	})

	t.Run("Metadata", func(t *testing.T) {
		metadata := backend.GetMetadata()
		require.NotNil(t, metadata)
		assert.Equal(t, "mock", metadata.Type)
		assert.True(t, metadata.SupportsLocking)
		assert.True(t, metadata.SupportsVersions)
		assert.True(t, metadata.SupportsWorkspaces)
	})
}

// Test StateData structure
func TestStateData(t *testing.T) {
	state := &StateData{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           42,
		Lineage:          "test-lineage-uuid",
		Data:             []byte(`{"test": "data"}`),
		Resources: []StateResource{
			{
				Mode:     "managed",
				Type:     "aws_instance",
				Name:     "example",
				Provider: "provider[\"registry.terraform.io/hashicorp/aws\"]",
				Instances: []StateResourceInstance{
					{
						SchemaVersion: 1,
						Attributes: map[string]interface{}{
							"id":            "i-1234567890abcdef0",
							"instance_type": "t3.micro",
						},
						Dependencies: []string{"aws_security_group.example"},
					},
				},
			},
		},
		Outputs: map[string]interface{}{
			"instance_id": map[string]interface{}{
				"value":     "i-1234567890abcdef0",
				"type":      "string",
				"sensitive": false,
			},
		},
		Backend: &BackendState{
			Type: "s3",
			Config: map[string]interface{}{
				"bucket": "my-terraform-state",
				"key":    "terraform.tfstate",
				"region": "us-west-2",
			},
			Hash:      "abc123",
			Workspace: "production",
		},
		Checksum:     "md5:abc123def456",
		LastModified: time.Now(),
		Size:         1024,
	}

	// Validate all fields are properly set
	assert.Equal(t, 4, state.Version)
	assert.Equal(t, "1.5.0", state.TerraformVersion)
	assert.Equal(t, uint64(42), state.Serial)
	assert.Equal(t, "test-lineage-uuid", state.Lineage)
	assert.NotEmpty(t, state.Data)
	assert.Len(t, state.Resources, 1)
	assert.Len(t, state.Outputs, 1)
	assert.NotNil(t, state.Backend)
	assert.NotEmpty(t, state.Checksum)
	assert.NotZero(t, state.LastModified)
	assert.Equal(t, int64(1024), state.Size)

	// Validate resource structure
	resource := state.Resources[0]
	assert.Equal(t, "managed", resource.Mode)
	assert.Equal(t, "aws_instance", resource.Type)
	assert.Equal(t, "example", resource.Name)
	assert.Len(t, resource.Instances, 1)

	// Validate instance structure
	instance := resource.Instances[0]
	assert.Equal(t, 1, instance.SchemaVersion)
	assert.Contains(t, instance.Attributes, "id")
	assert.Contains(t, instance.Attributes, "instance_type")
	assert.Contains(t, instance.Dependencies, "aws_security_group.example")

	// Validate backend structure
	backend := state.Backend
	assert.Equal(t, "s3", backend.Type)
	assert.Contains(t, backend.Config, "bucket")
	assert.Equal(t, "production", backend.Workspace)
}

// Test LockInfo structure
func TestLockInfo(t *testing.T) {
	created := time.Now()
	lockInfo := &LockInfo{
		ID:        "lock-12345",
		Path:      "terraform.tfstate",
		Operation: "apply",
		Who:       "user@example.com",
		Version:   "1.5.0",
		Created:   created,
		Info:      "Applying infrastructure changes",
	}

	assert.Equal(t, "lock-12345", lockInfo.ID)
	assert.Equal(t, "terraform.tfstate", lockInfo.Path)
	assert.Equal(t, "apply", lockInfo.Operation)
	assert.Equal(t, "user@example.com", lockInfo.Who)
	assert.Equal(t, "1.5.0", lockInfo.Version)
	assert.Equal(t, created, lockInfo.Created)
	assert.Equal(t, "Applying infrastructure changes", lockInfo.Info)
}

// Test StateVersion structure
func TestStateVersion(t *testing.T) {
	created := time.Now()
	version := &StateVersion{
		ID:          "version-1",
		VersionID:   "v1.0.0",
		Serial:      10,
		Created:     created,
		CreatedBy:   "terraform",
		Size:        2048,
		Checksum:    "sha256:abc123",
		IsLatest:    true,
		Description: "Initial infrastructure",
	}

	assert.Equal(t, "version-1", version.ID)
	assert.Equal(t, "v1.0.0", version.VersionID)
	assert.Equal(t, uint64(10), version.Serial)
	assert.Equal(t, created, version.Created)
	assert.Equal(t, "terraform", version.CreatedBy)
	assert.Equal(t, int64(2048), version.Size)
	assert.Equal(t, "sha256:abc123", version.Checksum)
	assert.True(t, version.IsLatest)
	assert.Equal(t, "Initial infrastructure", version.Description)
}

// Test BackendMetadata structure
func TestBackendMetadata(t *testing.T) {
	metadata := &BackendMetadata{
		Type:               "s3",
		SupportsLocking:    true,
		SupportsVersions:   true,
		SupportsWorkspaces: true,
		Configuration: map[string]string{
			"bucket": "my-terraform-state",
			"key":    "terraform.tfstate",
			"region": "us-west-2",
		},
		Workspace: "production",
		StateKey:  "terraform.tfstate",
		LockTable: "terraform-state-lock",
	}

	assert.Equal(t, "s3", metadata.Type)
	assert.True(t, metadata.SupportsLocking)
	assert.True(t, metadata.SupportsVersions)
	assert.True(t, metadata.SupportsWorkspaces)
	assert.Len(t, metadata.Configuration, 3)
	assert.Equal(t, "production", metadata.Workspace)
	assert.Equal(t, "terraform.tfstate", metadata.StateKey)
	assert.Equal(t, "terraform-state-lock", metadata.LockTable)
}

// Test BackendConfig structure
func TestBackendConfig(t *testing.T) {
	config := &BackendConfig{
		Type: "s3",
		Config: map[string]interface{}{
			"bucket":         "my-terraform-state",
			"key":            "terraform.tfstate",
			"region":         "us-west-2",
			"dynamodb_table": "terraform-state-lock",
			"encrypt":        true,
		},
		MaxConnections:     10,
		MaxIdleConnections: 5,
		ConnectionTimeout:  30 * time.Second,
		IdleTimeout:        5 * time.Minute,
		MaxRetries:         3,
		RetryDelay:         1 * time.Second,
		RetryBackoff:       2.0,
		LockTimeout:        10 * time.Minute,
		LockRetryDelay:     5 * time.Second,
	}

	assert.Equal(t, "s3", config.Type)
	assert.Len(t, config.Config, 5)
	assert.Equal(t, 10, config.MaxConnections)
	assert.Equal(t, 5, config.MaxIdleConnections)
	assert.Equal(t, 30*time.Second, config.ConnectionTimeout)
	assert.Equal(t, 5*time.Minute, config.IdleTimeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.RetryDelay)
	assert.Equal(t, 2.0, config.RetryBackoff)
	assert.Equal(t, 10*time.Minute, config.LockTimeout)
	assert.Equal(t, 5*time.Second, config.LockRetryDelay)
}
