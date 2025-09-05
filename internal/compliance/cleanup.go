package compliance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CleanupManager handles backup file cleanup with platform-specific implementations
type CleanupManager struct {
	backupDir       string
	retentionDays   int
	quarantineDir   string
	cleanupInterval time.Duration
	worker          CleanupWorker
	mu              sync.RWMutex
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

// CleanupWorker interface for platform-specific implementations
type CleanupWorker interface {
	// TryDelete attempts to delete a file, returns true if successful
	TryDelete(path string) error
	// ForceUnlock attempts to unlock a file (Windows-specific)
	ForceUnlock(path string) error
	// IsLocked checks if a file is locked
	IsLocked(path string) bool
}

// CleanupConfig contains configuration for cleanup manager
type CleanupConfig struct {
	BackupDir       string
	RetentionDays   int
	CleanupInterval time.Duration
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager(config CleanupConfig) *CleanupManager {
	if config.RetentionDays <= 0 {
		config.RetentionDays = 30
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 1 * time.Hour
	}
	
	quarantineDir := filepath.Join(config.BackupDir, ".quarantine")
	os.MkdirAll(quarantineDir, 0755)
	
	// Select platform-specific worker based on build tags
	var worker CleanupWorker
	worker = newPlatformWorker()
	
	return &CleanupManager{
		backupDir:       config.BackupDir,
		retentionDays:   config.RetentionDays,
		quarantineDir:   quarantineDir,
		cleanupInterval: config.CleanupInterval,
		worker:          worker,
		stopChan:        make(chan struct{}),
	}
}

// Start begins the async cleanup worker
func (cm *CleanupManager) Start(ctx context.Context) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.wg.Add(1)
	go cm.cleanupWorker(ctx)
}

// Stop stops the cleanup worker
func (cm *CleanupManager) Stop() {
	close(cm.stopChan)
	cm.wg.Wait()
}

// cleanupWorker runs periodically to clean old backups
func (cm *CleanupManager) cleanupWorker(ctx context.Context) {
	defer cm.wg.Done()
	
	ticker := time.NewTicker(cm.cleanupInterval)
	defer ticker.Stop()
	
	// Initial cleanup
	cm.performCleanup(ctx)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopChan:
			return
		case <-ticker.C:
			cm.performCleanup(ctx)
		}
	}
}

// performCleanup executes the cleanup process
func (cm *CleanupManager) performCleanup(ctx context.Context) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	now := time.Now()
	cutoffTime := now.AddDate(0, 0, -cm.retentionDays)
	
	// Walk backup directory
	filepath.Walk(cm.backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// Skip directories and quarantine
		if info.IsDir() || filepath.Dir(path) == cm.quarantineDir {
			return nil
		}
		
		// Check if file is old enough
		if info.ModTime().Before(cutoffTime) {
			cm.cleanupFile(path, info)
		}
		
		return nil
	})
}

// cleanupFile attempts to clean up a single file
func (cm *CleanupManager) cleanupFile(path string, info os.FileInfo) {
	// Try normal deletion first
	if err := cm.worker.TryDelete(path); err == nil {
		return
	}
	
	// Check if file is locked
	if cm.worker.IsLocked(path) {
		// Try to force unlock (Windows only)
		if err := cm.worker.ForceUnlock(path); err == nil {
			// Retry deletion after unlock
			if err := cm.worker.TryDelete(path); err == nil {
				return
			}
		}
		
		// Move to quarantine if still can't delete
		cm.quarantineFile(path, info)
	}
}

// quarantineFile moves a file to quarantine directory
func (cm *CleanupManager) quarantineFile(path string, info os.FileInfo) error {
	quarantinePath := filepath.Join(cm.quarantineDir, filepath.Base(path))
	
	// Add timestamp to avoid conflicts
	timestamp := time.Now().Format("20060102_150405")
	ext := filepath.Ext(quarantinePath)
	base := quarantinePath[:len(quarantinePath)-len(ext)]
	quarantinePath = fmt.Sprintf("%s_%s%s", base, timestamp, ext)
	
	return os.Rename(path, quarantinePath)
}

// CleanupNow performs immediate cleanup
func (cm *CleanupManager) CleanupNow(ctx context.Context) error {
	cm.performCleanup(ctx)
	return nil
}

// SetRetentionDays updates retention policy
func (cm *CleanupManager) SetRetentionDays(days int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.retentionDays = days
}

// GetQuarantineFiles returns list of quarantined files
func (cm *CleanupManager) GetQuarantineFiles() ([]string, error) {
	var files []string
	
	entries, err := os.ReadDir(cm.quarantineDir)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(cm.quarantineDir, entry.Name()))
		}
	}
	
	return files, nil
}

// EmptyQuarantine removes all files from quarantine
func (cm *CleanupManager) EmptyQuarantine() error {
	entries, err := os.ReadDir(cm.quarantineDir)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			path := filepath.Join(cm.quarantineDir, entry.Name())
			os.Remove(path) // Ignore errors
		}
	}
	
	return nil
}