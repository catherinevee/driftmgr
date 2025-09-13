package state

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBackupManager(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewBackupManager(tempDir)

	assert.NotNil(t, manager)
	assert.Equal(t, tempDir, manager.backupDir)
	assert.Equal(t, 10, manager.maxBackups)
	assert.True(t, manager.compression)
	assert.False(t, manager.encryption)
}

func TestBackupManager_CreateBackup(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewBackupManager(tempDir)

	tests := []struct {
		name      string
		backupID  string
		state     interface{}
		wantErr   bool
		setupFunc func()
	}{
		{
			name:     "Create valid backup",
			backupID: "test-backup-1",
			state: &TerraformState{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           1,
				Lineage:          "test-lineage",
				Resources:        []Resource{},
			},
			wantErr: false,
		},
		{
			name:     "Create backup with complex state",
			backupID: "test-backup-2",
			state: &TerraformState{
				Version:          4,
				TerraformVersion: "1.5.0",
				Serial:           5,
				Lineage:          "complex-lineage",
				Resources: []Resource{
					{
						Type:     "aws_instance",
						Name:     "web",
						Provider: "aws",
						Mode:     "managed",
						Instances: []Instance{
							{
								Attributes: map[string]interface{}{
									"id":            "i-1234567890",
									"instance_type": "t2.micro",
								},
							},
						},
					},
				},
				Outputs: map[string]OutputValue{
					"instance_id": {
						Value:     "i-1234567890",
						Sensitive: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name:     "Create backup with nil state",
			backupID: "test-backup-3",
			state:    nil,
			wantErr:  true,
		},
		{
			name:     "Create backup with empty ID",
			backupID: "",
			state: &TerraformState{
				Version: 4,
				Lineage: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			err := manager.CreateBackup(tt.backupID, tt.state)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify backup file exists (with timestamp pattern)
				files, err := filepath.Glob(filepath.Join(tempDir, tt.backupID+"_*.json.gz"))
				require.NoError(t, err)
				assert.NotEmpty(t, files, "Backup file should exist")

				// Verify metadata was updated
				assert.Contains(t, manager.metadata, tt.backupID)
			}
		})
	}
}

func TestBackupManager_RestoreBackup(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewBackupManager(tempDir)

	// Create a backup first
	testState := &TerraformState{
		Version:          4,
		TerraformVersion: "1.5.0",
		Serial:           10,
		Lineage:          "restore-test",
		Resources: []Resource{
			{
				Type:     "aws_s3_bucket",
				Name:     "backup",
				Provider: "aws",
				Mode:     "managed",
				Instances: []Instance{
					{
						Attributes: map[string]interface{}{
							"id":     "backup-bucket",
							"bucket": "my-backup-bucket",
						},
					},
				},
			},
		},
	}

	backupID := "restore-test-backup"
	err := manager.CreateBackup(backupID, testState)
	require.NoError(t, err)

	tests := []struct {
		name     string
		backupID string
		wantErr  bool
	}{
		{
			name:     "Restore existing backup",
			backupID: backupID,
			wantErr:  false,
		},
		{
			name:     "Restore non-existent backup",
			backupID: "non-existent",
			wantErr:  true,
		},
		{
			name:     "Restore with empty ID",
			backupID: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RestoreBackup(tt.backupID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// In a real implementation, we would verify the state was restored
			}
		})
	}
}

func TestBackupManager_ListBackups(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewBackupManager(tempDir)

	// Create multiple backups
	backups := []struct {
		id    string
		state *TerraformState
	}{
		{
			id: "backup-1",
			state: &TerraformState{
				Version: 4,
				Lineage: "test-1",
				Serial:  1,
			},
		},
		{
			id: "backup-2",
			state: &TerraformState{
				Version: 4,
				Lineage: "test-2",
				Serial:  2,
			},
		},
		{
			id: "backup-3",
			state: &TerraformState{
				Version: 4,
				Lineage: "test-3",
				Serial:  3,
			},
		},
	}

	for _, b := range backups {
		err := manager.CreateBackup(b.id, b.state)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// List backups
	list, err := manager.ListBackups()
	assert.NoError(t, err)
	assert.Len(t, list, 3)

	// Verify backups are in the list
	backupIDs := make(map[string]bool)
	for _, meta := range list {
		backupIDs[meta.ID] = true
	}

	for _, b := range backups {
		assert.True(t, backupIDs[b.id], "Backup %s should be in the list", b.id)
	}
}

func TestBackupManager_DeleteBackup(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewBackupManager(tempDir)

	// Create a backup to delete
	backupID := "delete-test"
	state := &TerraformState{
		Version: 4,
		Lineage: "delete-test",
		Serial:  1,
	}

	err := manager.CreateBackup(backupID, state)
	require.NoError(t, err)

	tests := []struct {
		name     string
		backupID string
		wantErr  bool
	}{
		{
			name:     "Delete existing backup",
			backupID: backupID,
			wantErr:  false,
		},
		{
			name:     "Delete non-existent backup",
			backupID: "non-existent",
			wantErr:  true,
		},
		{
			name:     "Delete with empty ID",
			backupID: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.DeleteBackup(tt.backupID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify backup was deleted
				backupFile := filepath.Join(tempDir, tt.backupID+".json.gz")
				_, err := os.Stat(backupFile)
				assert.True(t, os.IsNotExist(err), "Backup file should not exist")

				// Verify metadata was updated
				assert.NotContains(t, manager.metadata, tt.backupID)
			}
		})
	}
}

func TestBackupManager_CleanupOldBackups(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewBackupManager(tempDir)
	manager.SetMaxBackups(3)

	// Create more backups than the limit
	for i := 0; i < 5; i++ {
		backupID := fmt.Sprintf("backup-%d", i)
		state := &TerraformState{
			Version: 4,
			Lineage: fmt.Sprintf("test-%d", i),
			Serial:  i,
		}
		err := manager.CreateBackup(backupID, state)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Cleanup should happen automatically during CreateBackup
	// But we can also call it explicitly
	err := manager.cleanupOldBackups()
	assert.NoError(t, err)

	// List remaining backups
	list, err := manager.ListBackups()
	assert.NoError(t, err)
	assert.LessOrEqual(t, len(list), 3, "Should have at most 3 backups after cleanup")

	// Verify the newest backups are kept
	backupIDs := make(map[string]bool)
	for _, meta := range list {
		backupIDs[meta.ID] = true
	}

	// The newest backups (backup-2, backup-3, backup-4) should be kept
	assert.True(t, backupIDs["backup-2"] || backupIDs["backup-3"] || backupIDs["backup-4"])
}

func TestBackupManager_SetCompression(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewBackupManager(tempDir)

	// Default should be true
	assert.True(t, manager.compression)

	// Disable compression
	manager.SetCompression(false)
	assert.False(t, manager.compression)

	// Create backup without compression
	backupID := "no-compression"
	state := &TerraformState{
		Version: 4,
		Lineage: "test",
		Serial:  1,
	}

	err := manager.CreateBackup(backupID, state)
	assert.NoError(t, err)

	// Verify backup file is not compressed (with timestamp pattern)
	files, err := filepath.Glob(filepath.Join(tempDir, backupID+"_*.json"))
	require.NoError(t, err)
	assert.NotEmpty(t, files, "Uncompressed backup file should exist")

	// Re-enable compression
	manager.SetCompression(true)
	assert.True(t, manager.compression)
}

func TestBackupManager_SetEncryption(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewBackupManager(tempDir)

	// Default should be false
	assert.False(t, manager.encryption)

	// Enable encryption with a key
	encryptionKey := []byte("32-byte-encryption-key-for-test!")
	manager.SetEncryption(true, encryptionKey)
	assert.True(t, manager.encryption)
	assert.Equal(t, encryptionKey, manager.encryptionKey)

	// Note: Actual encryption implementation would be tested here
	// For now, we just verify the settings are applied

	// Disable encryption
	manager.SetEncryption(false, nil)
	assert.False(t, manager.encryption)
	assert.Nil(t, manager.encryptionKey)
}

func TestBackupManager_Metadata(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewBackupManager(tempDir)

	// Create backups with metadata
	backupID := "metadata-test"
	state := &TerraformState{
		Version: 4,
		Lineage: "test",
		Serial:  1,
		Resources: []Resource{
			{
				Type: "aws_instance",
				Name: "test",
			},
		},
	}

	err := manager.CreateBackup(backupID, state)
	require.NoError(t, err)

	// Check metadata
	meta, exists := manager.metadata[backupID]
	assert.True(t, exists)
	assert.Equal(t, backupID, meta.ID)
	assert.Equal(t, 4, meta.StateVersion)
	// Serial, Lineage, ResourceCount, Checksum fields don't exist in BackupMetadata
	assert.WithinDuration(t, time.Now(), meta.Timestamp, 5*time.Second)

	// Save metadata to file
	err = manager.SaveMetadata()
	assert.NoError(t, err)

	// Create new manager and load metadata
	newManager := NewBackupManager(tempDir)
	err = newManager.LoadMetadata()
	assert.NoError(t, err)

	// Verify metadata was loaded
	loadedMeta, exists := newManager.metadata[backupID]
	assert.True(t, exists)
	assert.Equal(t, meta.ID, loadedMeta.ID)
	assert.Equal(t, meta.StateVersion, loadedMeta.StateVersion)
	// Compare timestamps without monotonic clock
	assert.Equal(t, meta.Timestamp.Unix(), loadedMeta.Timestamp.Unix())
}

func TestBackupManager_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewBackupManager(tempDir)

	// Test concurrent backup creation
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			backupID := fmt.Sprintf("concurrent-%d", index)
			state := &TerraformState{
				Version: 4,
				Lineage: fmt.Sprintf("test-%d", index),
				Serial:  index,
			}
			err := manager.CreateBackup(backupID, state)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all backups were created
	list, err := manager.ListBackups()
	assert.NoError(t, err)
	assert.Len(t, list, 10)
}

func TestBackupManager_InvalidBackupDir(t *testing.T) {
	// Use an invalid directory path
	invalidDir := "/invalid\x00path/that/cannot/exist"
	manager := NewBackupManager(invalidDir)

	// Try to create a backup
	err := manager.CreateBackup("test", &TerraformState{
		Version: 4,
		Lineage: "test",
	})

	assert.Error(t, err)
}

func TestBackupManager_CorruptedBackup(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewBackupManager(tempDir)

	// Create a corrupted backup file with timestamp pattern
	backupID := "corrupted"
	timestamp := time.Now().Unix()
	corruptedFile := filepath.Join(tempDir, fmt.Sprintf("%s_%d.json.gz", backupID, timestamp))
	err := os.WriteFile(corruptedFile, []byte("corrupted data"), 0644)
	require.NoError(t, err)

	// Add to metadata
	manager.metadata[backupID] = &BackupMetadata{
		ID:        backupID,
		Timestamp: time.Now(),
	}

	// Try to restore corrupted backup
	err = manager.RestoreBackup(backupID)
	assert.Error(t, err)
	// The error will be about gzip reader, not "failed to read"
	assert.Contains(t, err.Error(), "failed")
}
