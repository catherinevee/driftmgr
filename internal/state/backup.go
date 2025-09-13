package state

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type BackupManager struct {
	backupDir     string
	maxBackups    int
	compression   bool
	encryption    bool
	encryptionKey []byte
	mu            sync.RWMutex
	metadata      map[string]*BackupMetadata
}

type BackupMetadata struct {
	ID           string            `json:"id"`
	Timestamp    time.Time         `json:"timestamp"`
	Size         int64             `json:"size"`
	Compressed   bool              `json:"compressed"`
	Encrypted    bool              `json:"encrypted"`
	StateVersion int               `json:"state_version"`
	Description  string            `json:"description"`
	Tags         map[string]string `json:"tags"`
}

type BackupOptions struct {
	Compress    bool              `json:"compress"`
	Encrypt     bool              `json:"encrypt"`
	Description string            `json:"description"`
	Tags        map[string]string `json:"tags"`
}

func NewBackupManager(backupDir string) *BackupManager {
	return &BackupManager{
		backupDir:   backupDir,
		maxBackups:  10,
		compression: true,
		metadata:    make(map[string]*BackupMetadata),
	}
}

func (bm *BackupManager) CreateBackup(backupID string, state interface{}) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Validate inputs
	if backupID == "" {
		return errors.New("backup ID cannot be empty")
	}
	if state == nil {
		return errors.New("state cannot be nil")
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(bm.backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Marshal state
	stateData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Create backup file path
	timestamp := time.Now().Unix()
	backupFile := filepath.Join(bm.backupDir, fmt.Sprintf("%s_%d.json", backupID, timestamp))

	if bm.compression {
		backupFile += ".gz"
	}

	// Write backup
	if err := bm.writeBackupFile(backupFile, stateData); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(backupFile)
	if err != nil {
		return fmt.Errorf("failed to stat backup file: %w", err)
	}

	// Extract state version if it's a TerraformState
	stateVersion := 0
	if tfState, ok := state.(*TerraformState); ok {
		stateVersion = tfState.Version
	}

	// Store metadata
	bm.metadata[backupID] = &BackupMetadata{
		ID:           backupID,
		Timestamp:    time.Now(),
		Size:         fileInfo.Size(),
		Compressed:   bm.compression,
		Encrypted:    bm.encryption,
		StateVersion: stateVersion,
		Description:  fmt.Sprintf("Backup created at %s", time.Now().Format(time.RFC3339)),
	}

	// Cleanup old backups
	if err := bm.cleanupOldBackups(); err != nil {
		// Log but don't fail
		fmt.Fprintf(os.Stderr, "failed to cleanup old backups: %v\n", err)
	}

	return nil
}

func (bm *BackupManager) RestoreBackup(backupID string) error {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	// Find backup file
	backupFile, err := bm.findBackupFile(backupID)
	if err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	// Read backup
	stateData, err := bm.readBackupFile(backupFile)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Parse state
	var state interface{}
	if err := json.Unmarshal(stateData, &state); err != nil {
		return fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	// Write restored state to current state file
	currentStateFile := filepath.Join(filepath.Dir(bm.backupDir), "terraform.tfstate")
	if err := os.WriteFile(currentStateFile, stateData, 0644); err != nil {
		return fmt.Errorf("failed to restore state: %w", err)
	}

	return nil
}

func (bm *BackupManager) ListBackups() ([]*BackupMetadata, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	backups := make([]*BackupMetadata, 0, len(bm.metadata))
	for _, meta := range bm.metadata {
		backups = append(backups, meta)
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

func (bm *BackupManager) DeleteBackup(backupID string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Find and delete backup file
	backupFile, err := bm.findBackupFile(backupID)
	if err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	if err := os.Remove(backupFile); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	// Remove metadata
	delete(bm.metadata, backupID)

	return nil
}

func (bm *BackupManager) writeBackupFile(filename string, data []byte) error {
	if bm.compression {
		return bm.writeCompressedFile(filename, data)
	}
	return os.WriteFile(filename, data, 0644)
}

func (bm *BackupManager) writeCompressedFile(filename string, data []byte) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	_, err = gz.Write(data)
	return err
}

func (bm *BackupManager) readBackupFile(filename string) ([]byte, error) {
	if strings.HasSuffix(filename, ".gz") {
		return bm.readCompressedFile(filename)
	}
	return os.ReadFile(filename)
}

func (bm *BackupManager) readCompressedFile(filename string) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	return io.ReadAll(gz)
}

func (bm *BackupManager) findBackupFile(backupID string) (string, error) {
	pattern := filepath.Join(bm.backupDir, fmt.Sprintf("%s_*", backupID))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", errors.New("backup not found")
	}

	// Return most recent backup
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

func (bm *BackupManager) cleanupOldBackups() error {
	// List all backup files
	files, err := os.ReadDir(bm.backupDir)
	if err != nil {
		return err
	}

	// Collect all backup files
	type backupFile struct {
		name string
		info os.DirEntry
		id   string
	}

	var backups []backupFile
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Skip non-backup files like metadata.json
		if !strings.Contains(file.Name(), "_") {
			continue
		}

		// Extract backup ID from filename
		parts := strings.Split(file.Name(), "_")
		if len(parts) < 2 {
			continue
		}

		backups = append(backups, backupFile{
			name: file.Name(),
			info: file,
			id:   parts[0],
		})
	}

	// If we have more than maxBackups total, delete oldest
	if len(backups) > bm.maxBackups {
		// Sort by name (timestamp is in the name)
		sort.Slice(backups, func(i, j int) bool {
			return backups[i].name < backups[j].name
		})

		// Delete oldest backups to keep only maxBackups total
		for i := 0; i < len(backups)-bm.maxBackups; i++ {
			oldFile := filepath.Join(bm.backupDir, backups[i].name)
			if err := os.Remove(oldFile); err != nil {
				fmt.Fprintf(os.Stderr, "failed to delete old backup %s: %v\n", oldFile, err)
			}
			// Also remove from metadata
			delete(bm.metadata, backups[i].id)
		}
	}

	return nil
}

func (bm *BackupManager) LoadMetadata() error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	metadataFile := filepath.Join(bm.backupDir, "metadata.json")

	data, err := os.ReadFile(metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No metadata file yet
		}
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	if err := json.Unmarshal(data, &bm.metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return nil
}

func (bm *BackupManager) SaveMetadata() error {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	metadataFile := filepath.Join(bm.backupDir, "metadata.json")

	data, err := json.MarshalIndent(bm.metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

func (bm *BackupManager) SetMaxBackups(max int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.maxBackups = max
}

func (bm *BackupManager) SetCompression(enabled bool) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.compression = enabled
}

func (bm *BackupManager) SetEncryption(enabled bool, key []byte) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.encryption = enabled
	bm.encryptionKey = key
}
