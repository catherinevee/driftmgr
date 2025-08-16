package utils

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ProgressTracker tracks progress for long-running operations
type ProgressTracker struct {
	mu           sync.Mutex
	total        int
	completed    int
	currentItem  string
	startTime    time.Time
	description  string
	showProgress bool
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(total int, description string) *ProgressTracker {
	return &ProgressTracker{
		total:        total,
		description:  description,
		startTime:    time.Now(),
		showProgress: true,
	}
}

// Update updates the progress
func (pt *ProgressTracker) Update(completed int, currentItem string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	// Ensure completed doesn't exceed total
	if completed > pt.total {
		pt.completed = pt.total
	} else {
		pt.completed = completed
	}
	pt.currentItem = currentItem

	if pt.showProgress {
		pt.displayProgress()
	}
}

// Increment increments the progress by 1
func (pt *ProgressTracker) Increment(currentItem string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.completed++
	pt.currentItem = currentItem

	if pt.showProgress {
		pt.displayProgress()
	}
}

// Complete marks the operation as complete
func (pt *ProgressTracker) Complete() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.completed = pt.total
	pt.currentItem = "Complete"

	if pt.showProgress {
		pt.displayProgress()
		fmt.Println() // Add newline after progress
	}
}

// displayProgress displays the current progress
func (pt *ProgressTracker) displayProgress() {
	elapsed := time.Since(pt.startTime)
	percentage := float64(pt.completed) / float64(pt.total) * 100

	// Calculate ETA
	var eta time.Duration
	if pt.completed > 0 {
		eta = time.Duration(float64(elapsed) * float64(pt.total-pt.completed) / float64(pt.completed))
	}

	// Create progress bar
	barWidth := 30
	filled := int(float64(barWidth) * percentage / 100)

	// Ensure filled doesn't exceed barWidth
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}

	remaining := barWidth - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", remaining)

	// Format time
	etaStr := formatDuration(eta)

	// Clear line and display progress
	fmt.Printf("\r%s [%s] %d/%d (%.1f%%) | %s | ETA: %s",
		pt.description, bar, pt.completed, pt.total, percentage, pt.currentItem, etaStr)
}

// formatDuration formats duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.0fh", d.Hours())
	}
}

// DisableProgress disables progress display
func (pt *ProgressTracker) DisableProgress() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.showProgress = false
}

// EnableProgress enables progress display
func (pt *ProgressTracker) EnableProgress() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.showProgress = true
}

// GetProgress returns current progress information
func (pt *ProgressTracker) GetProgress() (completed, total int, percentage float64, currentItem string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	percentage = float64(pt.completed) / float64(pt.total) * 100
	return pt.completed, pt.total, percentage, pt.currentItem
}
