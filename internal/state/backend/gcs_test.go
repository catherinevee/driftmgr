package backend

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test GCS Backend Creation
func TestNewGCSBackend(t *testing.T) {
	tests := []struct {
		name        string
		config      *BackendConfig
		expectError bool
	}{
		{
			name: "valid configuration",
			config: &BackendConfig{
				Type: "gcs",
				Config: map[string]interface{}{
					"bucket":  "test-bucket",
					"prefix":  "terraform/state",
					"project": "test-project",
				},
			},
			expectError: false,
		},
		{
			name: "minimal configuration",
			config: &BackendConfig{
				Type: "gcs",
				Config: map[string]interface{}{
					"bucket": "test-bucket",
				},
			},
			expectError: false,
		},
		{
			name: "configuration with workspace",
			config: &BackendConfig{
				Type: "gcs",
				Config: map[string]interface{}{
					"bucket":    "test-bucket",
					"prefix":    "terraform/state",
					"project":   "test-project",
					"workspace": "production",
				},
			},
			expectError: false,
		},
		{
			name: "configuration with credentials",
			config: &BackendConfig{
				Type: "gcs",
				Config: map[string]interface{}{
					"bucket":      "test-bucket",
					"prefix":      "terraform/state",
					"project":     "test-project",
					"credentials": "/path/to/credentials.json",
				},
			},
			expectError: false,
		},
		{
			name: "missing bucket",
			config: &BackendConfig{
				Type: "gcs",
				Config: map[string]interface{}{
					"prefix":  "terraform/state",
					"project": "test-project",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := NewGCSBackend(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, backend)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, backend)

				// Validate configuration
				bucket, _ := tt.config.Config["bucket"].(string)
				prefix, _ := tt.config.Config["prefix"].(string)
				project, _ := tt.config.Config["project"].(string)
				workspace, _ := tt.config.Config["workspace"].(string)
				credentials, _ := tt.config.Config["credentials"].(string)

				assert.Equal(t, bucket, backend.bucket)
				if prefix != "" {
					assert.Equal(t, prefix, backend.prefix)
				} else {
					assert.Equal(t, "terraform/state", backend.prefix)
				}
				assert.Equal(t, project, backend.project)
				assert.Equal(t, credentials, backend.credentials)
				if workspace != "" {
					assert.Equal(t, workspace, backend.workspace)
				} else {
					assert.Equal(t, "default", backend.workspace)
				}
			}
		})
	}
}

// Test GCS Backend Operations
func TestGCSBackend_Operations(t *testing.T) {
	config := &BackendConfig{
		Type: "gcs",
		Config: map[string]interface{}{
			"bucket":  "test-bucket",
			"prefix":  "terraform/state",
			"project": "test-project",
		},
	}

	backend, err := NewGCSBackend(config)
	require.NoError(t, err)
	require.NotNil(t, backend)

	ctx := context.Background()

	t.Run("Pull non-existent state", func(t *testing.T) {
		state, err := backend.Pull(ctx)
		require.NoError(t, err)
		assert.NotNil(t, state)
		assert.Equal(t, 4, state.Version)
		assert.Equal(t, uint64(0), state.Serial)
		assert.NotEmpty(t, state.Lineage)
		assert.Contains(t, string(state.Data), `"serial": 0`)
	})

	t.Run("Push and Pull state", func(t *testing.T) {
		testState := &StateData{
			Version:          4,
			TerraformVersion: "1.5.0",
			Serial:           1,
			Lineage:          "test-lineage",
			Data:             []byte(`{"version": 4, "serial": 1, "terraform_version": "1.5.0", "lineage": "test-lineage", "resources": [], "outputs": {}}`),
			LastModified:     time.Now(),
			Size:             100,
		}

		// Push state
		err := backend.Push(ctx, testState)
		require.NoError(t, err)

		// Verify object was stored
		objectName := backend.getObjectName()
		assert.Contains(t, backend.objects, objectName)
		assert.Contains(t, backend.metadata, objectName)

		// Pull state
		pulledState, err := backend.Pull(ctx)
		require.NoError(t, err)
		assert.Equal(t, testState.Version, pulledState.Version)
		assert.Equal(t, testState.Serial, pulledState.Serial)
		assert.Equal(t, testState.Lineage, pulledState.Lineage)
		assert.Equal(t, testState.TerraformVersion, pulledState.TerraformVersion)
		assert.NotEmpty(t, pulledState.Checksum)
	})

	t.Run("Lock and Unlock operations", func(t *testing.T) {
		lockInfo := &LockInfo{
			ID:        "test-lock",
			Path:      "default.tfstate",
			Operation: "plan",
			Who:       "test-user",
			Version:   "1.5.0",
			Created:   time.Now(),
			Info:      "Test lock",
		}

		// Acquire lock
		lockID, err := backend.Lock(ctx, lockInfo)
		require.NoError(t, err)
		assert.NotEmpty(t, lockID)
		assert.Contains(t, lockID, "gcs-lock")

		// Verify lock was stored
		lockKey := backend.getLockKey()
		assert.Contains(t, backend.locks, lockKey)

		// Try to acquire lock again (should fail)
		_, err = backend.Lock(ctx, lockInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already locked")

		// Get lock info
		info, err := backend.GetLockInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, lockID, info.ID)
		assert.Equal(t, lockInfo.Operation, info.Operation)
		assert.Equal(t, lockInfo.Who, info.Who)

		// Release lock
		err = backend.Unlock(ctx, lockID)
		require.NoError(t, err)

		// Verify lock was removed
		assert.NotContains(t, backend.locks, lockKey)

		// Verify lock info is cleared
		info, err = backend.GetLockInfo(ctx)
		require.NoError(t, err)
		assert.Nil(t, info)
	})

	t.Run("Workspace operations", func(t *testing.T) {
		// List initial workspaces
		workspaces, err := backend.ListWorkspaces(ctx)
		require.NoError(t, err)
		assert.Contains(t, workspaces, "default")

		// Create new workspace
		err = backend.CreateWorkspace(ctx, "test-workspace")
		require.NoError(t, err)

		// List workspaces should include new one
		workspaces, err = backend.ListWorkspaces(ctx)
		require.NoError(t, err)
		assert.Contains(t, workspaces, "test-workspace")

		// Select new workspace
		err = backend.SelectWorkspace(ctx, "test-workspace")
		require.NoError(t, err)
		assert.Equal(t, "test-workspace", backend.workspace)

		// Push state to new workspace
		testState := &StateData{
			Version:          4,
			TerraformVersion: "1.5.0",
			Serial:           1,
			Lineage:          "test-workspace-lineage",
			Data:             []byte(`{"version": 4, "serial": 1, "terraform_version": "1.5.0", "lineage": "test-workspace-lineage", "resources": [], "outputs": {}}`),
			LastModified:     time.Now(),
			Size:             100,
		}

		err = backend.Push(ctx, testState)
		require.NoError(t, err)

		// Verify workspace object was created
		workspaceObjectName := backend.getObjectName()
		assert.Contains(t, backend.objects, workspaceObjectName)
		assert.Contains(t, workspaceObjectName, "env:/test-workspace")

		// Pull from new workspace
		pulledState, err := backend.Pull(ctx)
		require.NoError(t, err)
		assert.Equal(t, "test-workspace-lineage", pulledState.Lineage)

		// Switch back to default
		err = backend.SelectWorkspace(ctx, "default")
		require.NoError(t, err)

		// Delete workspace
		err = backend.DeleteWorkspace(ctx, "test-workspace")
		require.NoError(t, err)

		// Verify workspace object was removed
		assert.NotContains(t, backend.objects, workspaceObjectName)

		// Verify workspace is not in list
		workspaces, err = backend.ListWorkspaces(ctx)
		require.NoError(t, err)
		assert.NotContains(t, workspaces, "test-workspace")
	})

	t.Run("Version operations", func(t *testing.T) {
		// Push multiple states to create versions
		for i := 1; i <= 3; i++ {
			state := &StateData{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           uint64(i),
				Lineage:          "version-test-lineage",
				Data:             []byte(fmt.Sprintf(`{"version": 4, "serial": %d, "terraform_version": "1.5.0", "lineage": "version-test-lineage", "resources": [], "outputs": {}}`, i)),
				LastModified:     time.Now(),
				Size:             100,
			}

			err := backend.Push(ctx, state)
			require.NoError(t, err)

			// Small delay to ensure different generations
			time.Sleep(10 * time.Millisecond)
		}

		// Get versions
		versions, err := backend.GetVersions(ctx)
		require.NoError(t, err)
		assert.Len(t, versions, 3)

		// Verify versions have different generations
		generations := make(map[string]bool)
		for _, v := range versions {
			generations[v.VersionID] = true
		}
		assert.Len(t, generations, 3)

		// Find latest version
		var latestVersion *StateVersion
		for _, v := range versions {
			if v.IsLatest {
				latestVersion = v
				break
			}
		}
		assert.NotNil(t, latestVersion)

		// Get specific version
		versionState, err := backend.GetVersion(ctx, latestVersion.VersionID)
		require.NoError(t, err)
		assert.NotNil(t, versionState)
		assert.Equal(t, uint64(3), versionState.Serial)

		// Get current version
		currentState, err := backend.GetVersion(ctx, "current")
		require.NoError(t, err)
		assert.NotNil(t, currentState)
		assert.Equal(t, uint64(3), currentState.Serial)
	})

	t.Run("Validation", func(t *testing.T) {
		err := backend.Validate(ctx)
		require.NoError(t, err)
	})

	t.Run("Metadata", func(t *testing.T) {
		metadata := backend.GetMetadata()
		require.NotNil(t, metadata)
		assert.Equal(t, "gcs", metadata.Type)
		assert.True(t, metadata.SupportsLocking)
		assert.True(t, metadata.SupportsVersions)
		assert.True(t, metadata.SupportsWorkspaces)
		assert.Equal(t, "test-bucket", metadata.Configuration["bucket"])
		assert.Equal(t, "terraform/state", metadata.Configuration["prefix"])
		assert.Equal(t, "test-project", metadata.Configuration["project"])
	})
}

// Test GCS Backend Error Handling
func TestGCSBackend_ErrorHandling(t *testing.T) {
	config := &BackendConfig{
		Type: "gcs",
		Config: map[string]interface{}{
			"bucket":  "test-bucket",
			"prefix":  "terraform/state",
			"project": "test-project",
		},
	}

	backend, err := NewGCSBackend(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Cannot create default workspace", func(t *testing.T) {
		err := backend.CreateWorkspace(ctx, "default")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot create default workspace")
	})

	t.Run("Cannot delete default workspace", func(t *testing.T) {
		err := backend.DeleteWorkspace(ctx, "default")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete default workspace")
	})

	t.Run("Cannot delete current workspace", func(t *testing.T) {
		// Create and select workspace
		err := backend.CreateWorkspace(ctx, "test")
		require.NoError(t, err)

		err = backend.SelectWorkspace(ctx, "test")
		require.NoError(t, err)

		// Try to delete current workspace
		err = backend.DeleteWorkspace(ctx, "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete current workspace")
	})

	t.Run("Select non-existent workspace", func(t *testing.T) {
		err := backend.SelectWorkspace(ctx, "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace non-existent does not exist")
	})

	t.Run("Create existing workspace", func(t *testing.T) {
		// Create workspace first
		err := backend.CreateWorkspace(ctx, "existing-test")
		require.NoError(t, err)

		// Try to create again
		err = backend.CreateWorkspace(ctx, "existing-test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace existing-test already exists")
	})

	t.Run("Get non-existent version", func(t *testing.T) {
		_, err := backend.GetVersion(ctx, "non-existent-generation")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version non-existent-generation not found")
	})
}

// Test GCS Backend Helper Methods
func TestGCSBackend_HelperMethods(t *testing.T) {
	config := &BackendConfig{
		Type: "gcs",
		Config: map[string]interface{}{
			"bucket":  "test-bucket",
			"prefix":  "terraform/state",
			"project": "test-project",
		},
	}

	backend, err := NewGCSBackend(config)
	require.NoError(t, err)

	t.Run("getObjectName for default workspace", func(t *testing.T) {
		backend.workspace = "default"
		objectName := backend.getObjectName()
		assert.Equal(t, "terraform/state/default.tfstate", objectName)
	})

	t.Run("getObjectName for custom workspace", func(t *testing.T) {
		backend.workspace = "production"
		objectName := backend.getObjectName()
		assert.Equal(t, "terraform/state/env:/production/default.tfstate", objectName)
	})

	t.Run("getLockKey", func(t *testing.T) {
		backend.workspace = "default"
		lockKey := backend.getLockKey()
		assert.Equal(t, "terraform/state/default.tfstate.lock", lockKey)
	})

	t.Run("isWorkspaceObject", func(t *testing.T) {
		backend.workspace = "default"

		// Test default workspace object
		assert.True(t, backend.isWorkspaceObject("terraform/state/default.tfstate"))

		// Test custom workspace object
		assert.True(t, backend.isWorkspaceObject("terraform/state/env:/production/default.tfstate"))

		// Test non-workspace object
		assert.False(t, backend.isWorkspaceObject("terraform/state/other.txt"))
	})

	t.Run("extractWorkspaceFromObject", func(t *testing.T) {
		// Test default workspace
		workspace := backend.extractWorkspaceFromObject("terraform/state/default.tfstate")
		assert.Equal(t, "default", workspace)

		// Test custom workspace
		workspace = backend.extractWorkspaceFromObject("terraform/state/env:/production/default.tfstate")
		assert.Equal(t, "production", workspace)

		// Test malformed workspace object
		workspace = backend.extractWorkspaceFromObject("terraform/state/env:/")
		assert.Equal(t, "default", workspace)
	})
}

// Benchmark GCS Backend Operations
func BenchmarkGCSBackend_Pull(b *testing.B) {
	config := &BackendConfig{
		Type: "gcs",
		Config: map[string]interface{}{
			"bucket":  "test-bucket",
			"prefix":  "terraform/state",
			"project": "test-project",
		},
	}

	backend, err := NewGCSBackend(config)
	require.NoError(b, err)

	// Prepare test data
	testData := []byte(`{"version": 4, "serial": 1, "resources": [], "outputs": {}}`)
	objectName := backend.getObjectName()
	backend.objects[objectName] = testData

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := backend.Pull(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGCSBackend_Push(b *testing.B) {
	config := &BackendConfig{
		Type: "gcs",
		Config: map[string]interface{}{
			"bucket":  "test-bucket",
			"prefix":  "terraform/state",
			"project": "test-project",
		},
	}

	backend, err := NewGCSBackend(config)
	require.NoError(b, err)

	state := &StateData{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           1,
		Lineage:          "test-lineage",
		Data:             []byte(`{"version": 4, "serial": 1, "resources": [], "outputs": {}}`),
		LastModified:     time.Now(),
		Size:             100,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.Serial = uint64(i + 1)
		err := backend.Push(ctx, state)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGCSBackend_Lock(b *testing.B) {
	config := &BackendConfig{
		Type: "gcs",
		Config: map[string]interface{}{
			"bucket":  "test-bucket",
			"prefix":  "terraform/state",
			"project": "test-project",
		},
	}

	backend, err := NewGCSBackend(config)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lockInfo := &LockInfo{
			ID:        fmt.Sprintf("bench-lock-%d", i),
			Operation: "benchmark",
			Who:       "benchmark-user",
			Created:   time.Now(),
		}

		lockID, err := backend.Lock(ctx, lockInfo)
		if err != nil {
			b.Fatal(err)
		}

		err = backend.Unlock(ctx, lockID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGCSBackend_LargeState(b *testing.B) {
	config := &BackendConfig{
		Type: "gcs",
		Config: map[string]interface{}{
			"bucket":  "test-bucket",
			"prefix":  "terraform/state",
			"project": "test-project",
		},
	}

	backend, err := NewGCSBackend(config)
	require.NoError(b, err)

	// Create large state data (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	state := &StateData{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           1,
		Lineage:          "test-lineage",
		Data:             largeData,
		LastModified:     time.Now(),
		Size:             int64(len(largeData)),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state.Serial = uint64(i + 1)
		err := backend.Push(ctx, state)
		if err != nil {
			b.Fatal(err)
		}

		_, err = backend.Pull(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}