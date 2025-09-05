//go:build windows
// +build windows

package compliance

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

// WindowsWorker implements cleanup operations for Windows
type WindowsWorker struct {
	kernel32 *syscall.LazyDLL
}

// NewWindowsWorker creates a new Windows cleanup worker
func NewWindowsWorker() *WindowsWorker {
	return &WindowsWorker{
		kernel32: syscall.NewLazyDLL("kernel32.dll"),
	}
}

// TryDelete attempts to delete a file with Windows-specific handling
func (w *WindowsWorker) TryDelete(path string) error {
	// First try normal deletion
	err := os.Remove(path)
	if err == nil {
		return nil
	}
	
	// If it's a permission error, try to change attributes
	if os.IsPermission(err) {
		// Remove read-only attribute
		if err := os.Chmod(path, 0666); err == nil {
			// Retry deletion
			return os.Remove(path)
		}
	}
	
	return err
}

// ForceUnlock attempts to unlock a file using Windows APIs
func (w *WindowsWorker) ForceUnlock(path string) error {
	// Convert path to UTF16
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	
	// Try to open the file with delete permission
	handle, err := syscall.CreateFile(
		pathPtr,
		syscall.GENERIC_WRITE|syscall.GENERIC_READ,
		syscall.FILE_SHARE_DELETE|syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		0x04000000, // FILE_FLAG_DELETE_ON_CLOSE
		0,
	)
	
	if err != nil {
		// Try alternative method using MoveFileEx
		return w.scheduleForDeletion(path)
	}
	
	// Close handle which should delete the file
	syscall.CloseHandle(handle)
	return nil
}

// scheduleForDeletion schedules file deletion on next reboot
func (w *WindowsWorker) scheduleForDeletion(path string) error {
	moveFileEx := w.kernel32.NewProc("MoveFileExW")
	
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	
	// MOVEFILE_DELAY_UNTIL_REBOOT = 0x4
	ret, _, err := moveFileEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		0x4,
	)
	
	if ret == 0 {
		return fmt.Errorf("failed to schedule deletion: %v", err)
	}
	
	return nil
}

// IsLocked checks if a file is locked on Windows
func (w *WindowsWorker) IsLocked(path string) bool {
	// Try to open the file exclusively
	file, err := os.OpenFile(path, os.O_RDWR|os.O_EXCL, 0)
	if err != nil {
		// Check if it's a sharing violation
		if pathErr, ok := err.(*os.PathError); ok {
			if errno, ok := pathErr.Err.(syscall.Errno); ok {
				// ERROR_SHARING_VIOLATION = 32
				if errno == 32 {
					return true
				}
			}
		}
		// Other errors might also indicate the file is in use
		return true
	}
	
	// File opened successfully, it's not locked
	file.Close()
	return false
}

// GetLockingProcesses attempts to find which processes have the file locked
func (w *WindowsWorker) GetLockingProcesses(path string) ([]string, error) {
	// This would require using the Restart Manager API
	// For now, return empty list
	return []string{}, nil
}