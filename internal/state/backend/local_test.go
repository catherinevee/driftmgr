package backend

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Local Backend Creation
func TestNewLocalBackend(t *testing.T) {
	tests := []struct {
		name        string
		config      *BackendConfig
		expectError bool
	}{
		{
			name: "valid configuration with path",
			config: &BackendConfig{
				Type: "local",
				Config: map[string]interface{}{
					"path": t.TempDir(),
				},
			},
			expectError: false,
		},
		{
			name: "default configuration",
			config: &BackendConfig{
				Type:   "local",
				Config: map[string]interface{}{},
			},
			expectError: false,
		},
		{
			name: "configuration with workspace",
			config: &BackendConfig{
				Type: "local",
				Config: map[string]interface{}{
					"path":      t.TempDir(),
					"workspace": "test",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := NewLocalBackend(tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, backend)
				assert.NotEmpty(t, backend.basePath)

				workspace, _ := tt.config.Config["workspace"].(string)
				if workspace == "" {
					workspace = "default"
				}
				assert.Equal(t, workspace, backend.workspace)
			}
		})
	}
}

// Test Local Backend Operations
func TestLocalBackend_Operations(t *testing.T) {
	tempDir := t.TempDir()
	config := &BackendConfig{
		Type: "local",
		Config: map[string]interface{}{
			"path": tempDir,
		},
	}

	backend, err := NewLocalBackend(config)
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

		// Verify state file exists
		statePath := filepath.Join(tempDir, "terraform.tfstate")
		_, err = os.Stat(statePath)
		require.NoError(t, err)

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
			Path:      "terraform.tfstate",
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
		assert.Contains(t, lockID, "test-lock")

		// Verify lock file exists
		lockPath := filepath.Join(tempDir, "terraform.tfstate.lock")
		_, err = os.Stat(lockPath)
		require.NoError(t, err)

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

		// Verify lock file is removed
		_, err = os.Stat(lockPath)
		assert.True(t, os.IsNotExist(err))

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

		// Verify workspace directory exists
		workspaceDir := filepath.Join(tempDir, "workspaces", "test-workspace")
		_, err = os.Stat(workspaceDir)
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

		// Verify workspace state file exists
		workspaceStatePath := filepath.Join(workspaceDir, "terraform.tfstate")
		_, err = os.Stat(workspaceStatePath)
		require.NoError(t, err)

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

		// Verify workspace directory is removed
		_, err = os.Stat(workspaceDir)
		assert.True(t, os.IsNotExist(err))

		// Verify workspace is not in list
		workspaces, err = backend.ListWorkspaces(ctx)
		require.NoError(t, err)
		assert.NotContains(t, workspaces, "test-workspace")
	})

	t.Run("Version operations and backup", func(t *testing.T) {
		// Push multiple states to create backups
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

			// Small delay to ensure different backup names
			time.Sleep(10 * time.Millisecond)
		}

		// Get versions
		versions, err := backend.GetVersions(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(versions), 1)

		// Find current version
		var currentVersion *StateVersion
		for _, v := range versions {
			if v.IsLatest {
				currentVersion = v
				break
			}
		}
		assert.NotNil(t, currentVersion)
		assert.Equal(t, "current", currentVersion.VersionID)

		// Get current version
		versionState, err := backend.GetVersion(ctx, "current")
		require.NoError(t, err)
		assert.NotNil(t, versionState)
		assert.Equal(t, uint64(3), versionState.Serial)

		// Check backup directory exists
		backupDir := filepath.Join(tempDir, ".terraform", "backups")
		_, err = os.Stat(backupDir)
		require.NoError(t, err)
	})

	t.Run("Validation", func(t *testing.T) {
		err := backend.Validate(ctx)
		require.NoError(t, err)
	})

	t.Run("Metadata", func(t *testing.T) {
		metadata := backend.GetMetadata()
		require.NotNil(t, metadata)
		assert.Equal(t, "local", metadata.Type)
		assert.True(t, metadata.SupportsLocking)
		assert.True(t, metadata.SupportsVersions)
		assert.True(t, metadata.SupportsWorkspaces)
		assert.Equal(t, tempDir, metadata.Configuration["path"])
	})
}

// Test Local Backend Error Handling
func TestLocalBackend_ErrorHandling(t *testing.T) {
	t.Run("Invalid base path", func(t *testing.T) {
		config := &BackendConfig{
			Type: "local",
			Config: map[string]interface{}{
				"path": "/invalid/path/that/does/not/exist/and/cannot/be/created",
			},
		}

		_, err := NewLocalBackend(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create base directory")
	})

	t.Run("Validation with inaccessible path", func(t *testing.T) {
		backend := &LocalBackend{
			basePath: "/invalid/path",
			metadata: &BackendMetadata{},
		}

		err := backend.Validate(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot access base path")
	})

	t.Run("Cannot create default workspace", func(t *testing.T) {
		tempDir := t.TempDir()
		config := &BackendConfig{
			Type: "local",
			Config: map[string]interface{}{
				"path": tempDir,
			},
		}

		backend, err := NewLocalBackend(config)
		require.NoError(t, err)

		err = backend.CreateWorkspace(context.Background(), "default")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot create default workspace")
	})

	t.Run("Cannot delete default workspace", func(t *testing.T) {
		tempDir := t.TempDir()
		config := &BackendConfig{
			Type: "local",
			Config: map[string]interface{}{
				"path": tempDir,
			},
		}

		backend, err := NewLocalBackend(config)
		require.NoError(t, err)

		err = backend.DeleteWorkspace(context.Background(), "default")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete default workspace")
	})

	t.Run("Cannot delete current workspace", func(t *testing.T) {
		tempDir := t.TempDir()
		config := &BackendConfig{
			Type: "local",
			Config: map[string]interface{}{
				"path": tempDir,
			},
		}

		backend, err := NewLocalBackend(config)
		require.NoError(t, err)

		// Create and select workspace
		err = backend.CreateWorkspace(context.Background(), "test")
		require.NoError(t, err)

		err = backend.SelectWorkspace(context.Background(), "test")
		require.NoError(t, err)

		// Try to delete current workspace
		err = backend.DeleteWorkspace(context.Background(), "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete current workspace")
	})

	t.Run("Select non-existent workspace", func(t *testing.T) {
		tempDir := t.TempDir()
		config := &BackendConfig{
			Type: "local",
			Config: map[string]interface{}{
				"path": tempDir,
			},
		}

		backend, err := NewLocalBackend(config)
		require.NoError(t, err)

		err = backend.SelectWorkspace(context.Background(), "non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace non-existent does not exist")
	})
}

// Test Local Backend Helper Methods
func TestLocalBackend_HelperMethods(t *testing.T) {
	tempDir := t.TempDir()
	backend := &LocalBackend{
		basePath:  tempDir,
		workspace: "default",
	}

	t.Run("getStatePath for default workspace", func(t *testing.T) {
		path := backend.getStatePath()
		expected := filepath.Join(tempDir, "terraform.tfstate")
		assert.Equal(t, expected, path)
	})

	t.Run("getStatePath for custom workspace", func(t *testing.T) {
		backend.workspace = "production"
		path := backend.getStatePath()
		expected := filepath.Join(tempDir, "workspaces", "production", "terraform.tfstate")
		assert.Equal(t, expected, path)
	})

	t.Run("getLockPath", func(t *testing.T) {
		backend.workspace = "default"
		lockPath := backend.getLockPath()
		expected := filepath.Join(tempDir, "terraform.tfstate.lock")
		assert.Equal(t, expected, lockPath)
	})

	t.Run("generateLineage", func(t *testing.T) {
		lineage1 := generateLineage()
		lineage2 := generateLineage()

		assert.NotEmpty(t, lineage1)
		assert.NotEmpty(t, lineage2)
		assert.NotEqual(t, lineage1, lineage2)
		assert.Contains(t, lineage1, "lineage-")
		assert.Contains(t, lineage2, "lineage-")
	})
}

// Test Concurrent Operations
func TestLocalBackend_ConcurrentOperations(t *testing.T) {
	tempDir := t.TempDir()
	config := &BackendConfig{
		Type: "local",
		Config: map[string]interface{}{
			"path": tempDir,
		},
	}

	backend, err := NewLocalBackend(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Concurrent locking", func(t *testing.T) {
		var wg sync.WaitGroup
		lockResults := make(chan error, 10)

		// Try to acquire lock from multiple goroutines
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				lockInfo := &LockInfo{
					ID:        fmt.Sprintf("lock-%d", id),
					Operation: "test",
					Who:       fmt.Sprintf("user-%d", id),
					Created:   time.Now(),
				}

				_, err := backend.Lock(ctx, lockInfo)
				lockResults <- err
			}(i)
		}

		wg.Wait()
		close(lockResults)

		// Count successful and failed locks
		successful := 0
		failed := 0
		for err := range lockResults {
			if err == nil {
				successful++
			} else {
				failed++
			}
		}

		// Only one should succeed
		assert.Equal(t, 1, successful)
		assert.Equal(t, 9, failed)

		// Clean up lock
		_ = backend.Unlock(ctx, "")
	})

	t.Run("Concurrent workspace creation", func(t *testing.T) {
		var wg sync.WaitGroup
		workspaceResults := make(chan error, 5)
		workspaceName := "concurrent-test"

		// Try to create same workspace from multiple goroutines
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := backend.CreateWorkspace(ctx, workspaceName)
				workspaceResults <- err
			}()
		}

		wg.Wait()
		close(workspaceResults)

		// Count successful and failed operations
		successful := 0
		failed := 0
		for err := range workspaceResults {
			if err == nil {
				successful++
			} else if strings.Contains(err.Error(), "already exists") {
				failed++
			}
		}

		// Only one should succeed, others should fail with "already exists"
		assert.Equal(t, 1, successful)
		assert.Equal(t, 4, failed)
	})
}

// Benchmark Local Backend Operations
func BenchmarkLocalBackend_Pull(b *testing.B) {
	tempDir := b.TempDir()
	config := &BackendConfig{
		Type: "local",
		Config: map[string]interface{}{
			"path": tempDir,
		},
	}

	backend, err := NewLocalBackend(config)
	require.NoError(b, err)

	// Create test state file
	testData := []byte(`{"version": 4, "serial": 1, "resources": [], "outputs": {}}`)
	statePath := filepath.Join(tempDir, "terraform.tfstate")
	err = os.WriteFile(statePath, testData, 0644)
	require.NoError(b, err)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := backend.Pull(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLocalBackend_Push(b *testing.B) {
	tempDir := b.TempDir()
	config := &BackendConfig{
		Type: "local",
		Config: map[string]interface{}{
			"path": tempDir,
		},
	}

	backend, err := NewLocalBackend(config)
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
		err := backend.Push(ctx, state)
		if err != nil {
			b.Fatal(err)
		}
		// Update serial to create different versions
		state.Serial = uint64(i + 2)
	}
}

func BenchmarkLocalBackend_LargeState(b *testing.B) {
	tempDir := b.TempDir()
	config := &BackendConfig{
		Type: "local",
		Config: map[string]interface{}{
			"path": tempDir,
		},
	}

	backend, err := NewLocalBackend(config)
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
		err := backend.Push(ctx, state)
		if err != nil {
			b.Fatal(err)
		}

		_, err = backend.Pull(ctx)
		if err != nil {
			b.Fatal(err)
		}

		state.Serial = uint64(i + 2)
	}
}

func BenchmarkLocalBackend_Lock(b *testing.B) {
	tempDir := b.TempDir()
	config := &BackendConfig{
		Type: "local",
		Config: map[string]interface{}{
			"path": tempDir,
		},
	}

	backend, err := NewLocalBackend(config)
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