package utils

import (
	"fmt"
	"strings"
	"time"
)

// LoadingBar represents a progress bar for long-running operations
type LoadingBar struct {
	width      int
	progress   float64
	message    string
	startTime  time.Time
	isComplete bool
}

// NewLoadingBar creates a new loading bar with the specified width and message
func NewLoadingBar(width int, message string) *LoadingBar {
	return &LoadingBar{
		width:     width,
		message:   message,
		startTime: time.Now(),
	}
}

// Update updates the progress bar with a new progress value (0.0 to 1.0)
func (lb *LoadingBar) Update(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	lb.progress = progress
	lb.render()
}

// Complete marks the loading bar as complete and shows final state
func (lb *LoadingBar) Complete() {
	lb.progress = 1.0
	lb.isComplete = true
	lb.render()
	fmt.Println() // Add newline after completion
}

// render renders the current state of the loading bar
func (lb *LoadingBar) render() {
	// Calculate filled width
	filledWidth := int(float64(lb.width) * lb.progress)

	// Create the progress bar
	bar := "["
	bar += strings.Repeat("=", filledWidth)
	if lb.progress < 1.0 {
		bar += ">"
		bar += strings.Repeat(" ", lb.width-filledWidth-1)
	} else {
		bar += strings.Repeat("=", lb.width-filledWidth)
	}
	bar += "]"

	// Calculate percentage
	percentage := int(lb.progress * 100)

	// Calculate elapsed time
	elapsed := time.Since(lb.startTime)

	// Create status message
	status := fmt.Sprintf("%s %s %d%% (%v)", lb.message, bar, percentage, elapsed.Round(time.Millisecond))

	// Clear line and print status
	fmt.Printf("\r%s", status)

	// If complete, add a checkmark
	if lb.isComplete {
		fmt.Print(" ✓")
	}
}

// LoadingSpinner represents an animated spinner for indeterminate operations
type LoadingSpinner struct {
	message   string
	spinner   []string
	position  int
	startTime time.Time
	isRunning bool
	stopChan  chan bool
}

// NewLoadingSpinner creates a new loading spinner
func NewLoadingSpinner(message string) *LoadingSpinner {
	return &LoadingSpinner{
		message:   message,
		spinner:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		position:  0,
		startTime: time.Now(),
		stopChan:  make(chan bool, 1), // Buffered channel to prevent deadlock
	}
}

// Start starts the spinner animation
func (ls *LoadingSpinner) Start() {
	ls.isRunning = true
	go ls.animate()
}

// Stop stops the spinner animation
func (ls *LoadingSpinner) Stop() {
	if ls.isRunning {
		ls.isRunning = false
		select {
		case ls.stopChan <- true:
			// Successfully sent stop signal
		default:
			// Channel is full, continue anyway
		}
		elapsed := time.Since(ls.startTime)
		fmt.Printf("\r%s ✓ (%v)\n", ls.message, elapsed.Round(time.Millisecond))
	}
}

// animate runs the spinner animation
func (ls *LoadingSpinner) animate() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !ls.isRunning {
				return
			}
			elapsed := time.Since(ls.startTime)
			spinner := ls.spinner[ls.position]
			fmt.Printf("\r%s %s (%v)", ls.message, spinner, elapsed.Round(time.Millisecond))
			ls.position = (ls.position + 1) % len(ls.spinner)
		case <-ls.stopChan:
			return
		}
	}
}

// LoadingManager provides a simple interface for showing loading states
type LoadingManager struct{}

// NewLoadingManager creates a new loading manager
func NewLoadingManager() *LoadingManager {
	return &LoadingManager{}
}

// ShowProgress shows a progress bar for operations with known progress
func (lm *LoadingManager) ShowProgress(message string, progressFunc func(updateFunc func(float64))) {
	bar := NewLoadingBar(30, message)
	progressFunc(bar.Update)
	bar.Complete()
}

// ShowSpinner shows a spinner for operations with unknown duration
func (lm *LoadingManager) ShowSpinner(message string, operation func()) {
	spinner := NewLoadingSpinner(message)
	spinner.Start()
	operation()
	spinner.Stop()
}

// ShowSimpleMessage shows a simple loading message
func (lm *LoadingManager) ShowSimpleMessage(message string) {
	fmt.Printf("%s... ", message)
}

// CompleteMessage completes a simple message
func (lm *LoadingManager) CompleteMessage() {
	fmt.Println("✓")
}

// ErrorMessage shows an error message
func (lm *LoadingManager) ErrorMessage(err error) {
	fmt.Printf("✗ Error: %v\n", err)
}

// Global loading manager instance
var GlobalLoadingManager = NewLoadingManager()
