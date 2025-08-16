package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressTracker manages progress indicators for long-running discoveries
type ProgressTracker struct {
	mu              sync.RWMutex
	bar             *progressbar.ProgressBar
	status          map[string]string
	errors          map[string]error
	warnings        map[string]string
	startTime       time.Time
	totalSteps      int
	completedSteps  int
	providers       []string
	regions         []string
	services        map[string][]string
	ctx             context.Context
	cancel          context.CancelFunc
	progressChannel chan ProgressEvent
}

// ProgressEvent represents a progress update event
type ProgressEvent struct {
	Type      ProgressType `json:"type"`
	Provider  string       `json:"provider"`
	Region    string       `json:"region"`
	Service   string       `json:"service"`
	Message   string       `json:"message"`
	Progress  float64      `json:"progress"`
	Error     error        `json:"error,omitempty"`
	Warning   string       `json:"warning,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

// ProgressType defines the type of progress update
type ProgressType string

const (
	ProgressStart    ProgressType = "start"
	ProgressUpdate   ProgressType = "update"
	ProgressComplete ProgressType = "complete"
	ProgressError    ProgressType = "error"
	ProgressWarning  ProgressType = "warning"
	ProgressCancel   ProgressType = "cancel"
)

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(providers, regions []string, services map[string][]string) *ProgressTracker {
	ctx, cancel := context.WithCancel(context.Background())

	tracker := &ProgressTracker{
		status:          make(map[string]string),
		errors:          make(map[string]error),
		warnings:        make(map[string]string),
		startTime:       time.Now(),
		providers:       providers,
		regions:         regions,
		services:        services,
		ctx:             ctx,
		cancel:          cancel,
		progressChannel: make(chan ProgressEvent, 100),
	}

	// Calculate total steps
	totalSteps := 0
	for _, provider := range providers {
		if serviceList, exists := services[provider]; exists {
			totalSteps += len(serviceList) * len(regions)
		}
	}
	tracker.totalSteps = totalSteps

	// Initialize progress bar
	tracker.bar = progressbar.NewOptions(totalSteps,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription("[cyan][1/1][reset] Starting discovery..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Println("\n[green]Discovery completed successfully![reset]")
		}),
	)

	return tracker
}

// Start begins the progress tracking
func (pt *ProgressTracker) Start() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.startTime = time.Now()
	pt.completedSteps = 0

	// Send start event
	pt.progressChannel <- ProgressEvent{
		Type:      ProgressStart,
		Message:   fmt.Sprintf("Starting discovery for %d providers across %d regions", len(pt.providers), len(pt.regions)),
		Progress:  0.0,
		Timestamp: time.Now(),
	}

	fmt.Printf("[cyan]Starting discovery for %d providers across %d regions...[reset]\n", len(pt.providers), len(pt.regions))
}

// UpdateProgress updates the progress for a specific service
func (pt *ProgressTracker) UpdateProgress(provider, region, service, message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.completedSteps++
	progress := float64(pt.completedSteps) / float64(pt.totalSteps) * 100

	key := fmt.Sprintf("%s:%s:%s", provider, region, service)
	pt.status[key] = message

	// Update progress bar
	pt.bar.Set(pt.completedSteps)
	pt.bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] %s - %s - %s", pt.completedSteps, pt.totalSteps, provider, region, service))

	// Send progress update
	pt.progressChannel <- ProgressEvent{
		Type:      ProgressUpdate,
		Provider:  provider,
		Region:    region,
		Service:   service,
		Message:   message,
		Progress:  progress,
		Timestamp: time.Now(),
	}
}

// ReportError reports an error for a specific service
func (pt *ProgressTracker) ReportError(provider, region, service string, err error) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", provider, region, service)
	pt.errors[key] = err

	// Send error event
	pt.progressChannel <- ProgressEvent{
		Type:      ProgressError,
		Provider:  provider,
		Region:    region,
		Service:   service,
		Message:   fmt.Sprintf("Error: %v", err),
		Error:     err,
		Timestamp: time.Now(),
	}

	fmt.Printf("[red]Error in %s:%s:%s: %v[reset]\n", provider, region, service, err)
}

// ReportWarning reports a warning for a specific service
func (pt *ProgressTracker) ReportWarning(provider, region, service, warning string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", provider, region, service)
	pt.warnings[key] = warning

	// Send warning event
	pt.progressChannel <- ProgressEvent{
		Type:      ProgressWarning,
		Provider:  provider,
		Region:    region,
		Service:   service,
		Message:   warning,
		Warning:   warning,
		Timestamp: time.Now(),
	}

	fmt.Printf("[yellow]Warning in %s:%s:%s: %s[reset]\n", provider, region, service, warning)
}

// Complete marks the discovery as complete
func (pt *ProgressTracker) Complete() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	duration := time.Since(pt.startTime)

	// Send completion event
	pt.progressChannel <- ProgressEvent{
		Type:      ProgressComplete,
		Message:   fmt.Sprintf("Discovery completed in %v", duration),
		Progress:  100.0,
		Timestamp: time.Now(),
	}

	pt.bar.Finish()
	fmt.Printf("[green]Discovery completed in %v[reset]\n", duration)
}

// Cancel cancels the progress tracking
func (pt *ProgressTracker) Cancel() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.cancel()

	// Send cancel event
	pt.progressChannel <- ProgressEvent{
		Type:      ProgressCancel,
		Message:   "Discovery cancelled by user",
		Timestamp: time.Now(),
	}

	// Clean up resources
	pt.cleanup()

	fmt.Println("[yellow]Discovery cancelled[reset]")
}

// cleanup performs resource cleanup
func (pt *ProgressTracker) cleanup() {
	// Close progress channel to prevent goroutine leaks
	close(pt.progressChannel)

	// Clear maps to free memory
	pt.status = make(map[string]string)
	pt.errors = make(map[string]error)
	pt.warnings = make(map[string]string)
}

// GetProgressChannel returns the progress update channel
func (pt *ProgressTracker) GetProgressChannel() <-chan ProgressEvent {
	return pt.progressChannel
}

// GetStatus returns the current status
func (pt *ProgressTracker) GetStatus() map[string]string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	status := make(map[string]string)
	for k, v := range pt.status {
		status[k] = v
	}
	return status
}

// GetErrors returns all errors
func (pt *ProgressTracker) GetErrors() map[string]error {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	errors := make(map[string]error)
	for k, v := range pt.errors {
		errors[k] = v
	}
	return errors
}

// GetWarnings returns all warnings
func (pt *ProgressTracker) GetWarnings() map[string]string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	warnings := make(map[string]string)
	for k, v := range pt.warnings {
		warnings[k] = v
	}
	return warnings
}

// GetDuration returns the total duration
func (pt *ProgressTracker) GetDuration() time.Duration {
	return time.Since(pt.startTime)
}

// GetProgressPercentage returns the current progress percentage
func (pt *ProgressTracker) GetProgressPercentage() float64 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	if pt.totalSteps == 0 {
		return 0.0
	}
	return float64(pt.completedSteps) / float64(pt.totalSteps) * 100
}

// IsCancelled checks if the discovery has been cancelled
func (pt *ProgressTracker) IsCancelled() bool {
	select {
	case <-pt.ctx.Done():
		return true
	default:
		return false
	}
}
