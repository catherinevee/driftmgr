//go:build !windows
// +build !windows

package compliance

import (
	"os"
	"syscall"
)

// UnixWorker implements cleanup operations for Unix-like systems
type UnixWorker struct{}

// NewUnixWorker creates a new Unix cleanup worker
func NewUnixWorker() *UnixWorker {
	return &UnixWorker{}
}

// TryDelete attempts to delete a file
func (w *UnixWorker) TryDelete(path string) error {
	return os.Remove(path)
}

// ForceUnlock is a no-op on Unix systems
func (w *UnixWorker) ForceUnlock(path string) error {
	// Unix doesn't have the same file locking issues as Windows
	return nil
}

// IsLocked checks if a file is locked using flock
func (w *UnixWorker) IsLocked(path string) bool {
	file, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return true // Can't open, assume locked
	}
	defer file.Close()

	// Try to acquire an exclusive lock
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return true // Can't lock, file is in use
	}

	// Release the lock
	syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	return false
}

// newPlatformWorker creates a Unix worker on Unix platforms
func newPlatformWorker() CleanupWorker {
	return NewUnixWorker()
}
