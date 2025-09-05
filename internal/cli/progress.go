package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ProgressIndicator provides visual feedback during long-running operations
type ProgressIndicator struct {
	writer      io.Writer
	message     string
	total       int
	current     int
	startTime   time.Time
	spinner     *Spinner
	showPercent bool
	showETA     bool
	mu          sync.Mutex
}

// NewProgressIndicator creates a new progress indicator
func NewProgressIndicator(total int, message string) *ProgressIndicator {
	return &ProgressIndicator{
		writer:      os.Stdout,
		message:     message,
		total:       total,
		current:     0,
		startTime:   time.Now(),
		showPercent: true,
		showETA:     true,
	}
}

// Start begins the progress display
func (p *ProgressIndicator) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.startTime = time.Now()
	p.render()
}

// Update updates the current progress
func (p *ProgressIndicator) Update(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = current
	p.render()
}

// Increment increments the progress by 1
func (p *ProgressIndicator) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current++
	p.render()
}

// SetMessage updates the progress message
func (p *ProgressIndicator) SetMessage(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.message = message
	p.render()
}

// Complete marks the progress as complete
func (p *ProgressIndicator) Complete() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = p.total
	p.render()
	fmt.Fprintln(p.writer)
}

// render displays the progress bar
func (p *ProgressIndicator) render() {
	if p.total <= 0 {
		return
	}

	percent := float64(p.current) / float64(p.total) * 100
	barWidth := 40
	filled := int(float64(barWidth) * float64(p.current) / float64(p.total))

	// Build progress bar
	bar := strings.Builder{}
	bar.WriteString("\r")
	bar.WriteString(p.message)
	bar.WriteString(" [")

	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar.WriteString("=")
		} else if i == filled {
			bar.WriteString(">")
		} else {
			bar.WriteString(" ")
		}
	}

	bar.WriteString("] ")

	if p.showPercent {
		bar.WriteString(fmt.Sprintf("%.1f%% ", percent))
	}

	bar.WriteString(fmt.Sprintf("(%d/%d)", p.current, p.total))

	if p.showETA && p.current > 0 {
		elapsed := time.Since(p.startTime)
		eta := time.Duration(float64(elapsed) / float64(p.current) * float64(p.total-p.current))
		if eta > 0 {
			bar.WriteString(fmt.Sprintf(" ETA: %s", formatDuration(eta)))
		}
	}

	// Clear to end of line
	bar.WriteString("\033[K")

	fmt.Fprint(p.writer, bar.String())
}

// Spinner provides an animated spinner for indeterminate operations
type Spinner struct {
	writer   io.Writer
	message  string
	frames   []string
	current  int
	active   bool
	stopChan chan bool
	mu       sync.Mutex
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		writer:   os.Stdout,
		message:  message,
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		stopChan: make(chan bool),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopChan:
				return
			case <-ticker.C:
				s.mu.Lock()
				s.render()
				s.current = (s.current + 1) % len(s.frames)
				s.mu.Unlock()
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false
	s.stopChan <- true
	fmt.Fprintf(s.writer, "\r\033[K")
}

// Success stops the spinner with a success message
func (s *Spinner) Success(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "✓ %s\n", message)
}

// Error stops the spinner with an error message
func (s *Spinner) Error(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "✗ %s\n", message)
}

// SetMessage updates the spinner message
func (s *Spinner) SetMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// render displays the current spinner frame
func (s *Spinner) render() {
	fmt.Fprintf(s.writer, "\r%s %s", s.frames[s.current], s.message)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

// MultiProgress manages multiple progress indicators
type MultiProgress struct {
	indicators []*ProgressIndicator
	spinners   []*Spinner
	mu         sync.Mutex
}

// NewMultiProgress creates a new multi-progress manager
func NewMultiProgress() *MultiProgress {
	return &MultiProgress{
		indicators: make([]*ProgressIndicator, 0),
		spinners:   make([]*Spinner, 0),
	}
}

// AddProgress adds a progress indicator
func (m *MultiProgress) AddProgress(total int, message string) *ProgressIndicator {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := NewProgressIndicator(total, message)
	m.indicators = append(m.indicators, p)
	return p
}

// AddSpinner adds a spinner
func (m *MultiProgress) AddSpinner(message string) *Spinner {
	m.mu.Lock()
	defer m.mu.Unlock()

	s := NewSpinner(message)
	m.spinners = append(m.spinners, s)
	return s
}

// StopAll stops all progress indicators and spinners
func (m *MultiProgress) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, s := range m.spinners {
		s.Stop()
	}

	for _, p := range m.indicators {
		p.Complete()
	}
}
