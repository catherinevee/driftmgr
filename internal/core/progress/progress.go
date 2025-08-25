package progress

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/catherinevee/driftmgr/internal/core/color"
)

// Spinner represents an animated spinner
type Spinner struct {
	frames  []string
	current int
	message string
	active  bool
	mu      sync.Mutex
	writer  io.Writer
}

// Bar represents a progress bar
type Bar struct {
	total     int
	current   int
	width     int
	message   string
	startTime time.Time
	mu        sync.Mutex
	writer    io.Writer
}

// NewSpinner creates a new spinner
func NewSpinner(message string) *Spinner {
	return &Spinner{
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		message: message,
		writer:  os.Stdout,
	}
}

// NewDotSpinner creates a dot-based spinner
func NewDotSpinner(message string) *Spinner {
	return &Spinner{
		frames:  []string{"   ", ".  ", ".. ", "..."},
		message: message,
		writer:  os.Stdout,
	}
}

// NewBarSpinner creates a bar-based spinner
func NewBarSpinner(message string) *Spinner {
	return &Spinner{
		frames:  []string{"|", "/", "-", "\\"},
		message: message,
		writer:  os.Stdout,
	}
}

// Start starts the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		for {
			s.mu.Lock()
			if !s.active {
				s.mu.Unlock()
				break
			}
			frame := s.frames[s.current]
			fmt.Fprintf(s.writer, "\r%s %s", color.Spinner(frame), s.message)
			s.current = (s.current + 1) % len(s.frames)
			s.mu.Unlock()
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

// Stop stops the spinner with proper cleanup
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}
	s.active = false

	// Give the spinner goroutine time to exit
	time.Sleep(150 * time.Millisecond)

	// Clear the line completely using ANSI escape sequences
	// \r moves cursor to beginning, \033[K clears from cursor to end of line
	fmt.Fprintf(s.writer, "\r\033[K")

	// Alternative fallback for terminals that don't support ANSI
	if runtime.GOOS == "windows" {
		// Use spaces to clear for Windows terminals that might not support ANSI
		fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", getTerminalWidth()))
	}
}

// Success stops the spinner with a success message
func (s *Spinner) Success(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "%s %s\n", color.CheckMark(), color.Success(message))
}

// Error stops the spinner with an error message
func (s *Spinner) Error(message string) {
	s.Stop()
	fmt.Fprintf(s.writer, "%s %s\n", color.CrossMark(), color.Error(message))
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// NewBar creates a new progress bar
func NewBar(total int, message string) *Bar {
	return &Bar{
		total:     total,
		width:     40,
		message:   message,
		startTime: time.Now(),
		writer:    os.Stdout,
	}
}

// NewCustomBar creates a progress bar with custom width
func NewCustomBar(total int, width int, message string) *Bar {
	return &Bar{
		total:     total,
		width:     width,
		message:   message,
		startTime: time.Now(),
		writer:    os.Stdout,
	}
}

// Update updates the progress bar
func (b *Bar) Update(current int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.current = current
	b.render()
}

// Increment increments the progress bar by 1
func (b *Bar) Increment() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.current++
	if b.current > b.total {
		b.current = b.total
	}
	b.render()
}

// render renders the progress bar
func (b *Bar) render() {
	if b.total == 0 {
		return
	}

	percent := float64(b.current) / float64(b.total)
	filled := int(percent * float64(b.width))

	// Create the bar with colors
	bar := strings.Builder{}
	bar.WriteString(color.Sprint(color.Gray, "["))

	filledStr := strings.Builder{}
	emptyStr := strings.Builder{}

	for i := 0; i < b.width; i++ {
		if i < filled {
			filledStr.WriteString("█")
		} else if i == filled {
			filledStr.WriteString("▓")
		} else {
			emptyStr.WriteString("░")
		}
	}

	bar.WriteString(color.Sprint(color.Green, filledStr.String()))
	bar.WriteString(color.Sprint(color.Gray, emptyStr.String()))
	bar.WriteString(color.Sprint(color.Gray, "]"))

	// Calculate ETA
	elapsed := time.Since(b.startTime)
	eta := ""
	if b.current > 0 && b.current < b.total {
		remaining := elapsed * time.Duration(b.total-b.current) / time.Duration(b.current)
		eta = fmt.Sprintf(" ETA: %s", formatDuration(remaining))
	}

	// Print the bar
	fmt.Fprintf(b.writer, "\r%s %s %3.0f%% (%d/%d)%s",
		b.message,
		bar.String(),
		percent*100,
		b.current,
		b.total,
		eta)

	// If complete, add newline
	if b.current >= b.total {
		fmt.Fprintln(b.writer)
	}
}

// Complete marks the progress bar as complete
func (b *Bar) Complete() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.current = b.total
	b.render()
}

// Clear clears the progress bar line
func (b *Bar) Clear() {
	fmt.Fprintf(b.writer, "\r%s\r", strings.Repeat(" ", 80))
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "< 1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// MultiProgress manages multiple progress items
type MultiProgress struct {
	items  []*ProgressItem
	mu     sync.Mutex
	writer io.Writer
}

// ProgressItem represents a single progress item
type ProgressItem struct {
	name    string
	status  string
	spinner *Spinner
}

// NewMultiProgress creates a new multi-progress display
func NewMultiProgress() *MultiProgress {
	return &MultiProgress{
		items:  make([]*ProgressItem, 0),
		writer: os.Stdout,
	}
}

// AddItem adds a new progress item
func (m *MultiProgress) AddItem(name string) *ProgressItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := &ProgressItem{
		name:    name,
		status:  "pending",
		spinner: NewSpinner(name),
	}
	m.items = append(m.items, item)
	return item
}

// Render renders all progress items
func (m *MultiProgress) Render() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear previous output
	fmt.Fprintf(m.writer, "\033[%dA", len(m.items))

	for _, item := range m.items {
		icon := "⠿"
		switch item.status {
		case "running":
			icon = "⠿"
		case "success":
			icon = "✓"
		case "error":
			icon = "✗"
		case "pending":
			icon = "○"
		}

		fmt.Fprintf(m.writer, "%s %s\n", icon, item.name)
	}
}

// LoadingAnimation shows a loading animation with custom message
type LoadingAnimation struct {
	message   string
	animation []string
	current   int
	active    bool
	mu        sync.Mutex
}

// NewLoadingAnimation creates a new loading animation
func NewLoadingAnimation(message string) *LoadingAnimation {
	return &LoadingAnimation{
		message: message,
		animation: []string{
			"[■□□□□□□□□□]",
			"[■■□□□□□□□□]",
			"[■■■□□□□□□□]",
			"[■■■■□□□□□□]",
			"[■■■■■□□□□□]",
			"[■■■■■■□□□□]",
			"[■■■■■■■□□□]",
			"[■■■■■■■■□□]",
			"[■■■■■■■■■□]",
			"[■■■■■■■■■■]",
		},
	}
}

// Start starts the loading animation
func (l *LoadingAnimation) Start() {
	l.mu.Lock()
	if l.active {
		l.mu.Unlock()
		return
	}
	l.active = true
	l.mu.Unlock()

	go func() {
		for {
			l.mu.Lock()
			if !l.active {
				l.mu.Unlock()
				break
			}
			frame := l.animation[l.current]
			fmt.Printf("\r%s %s", l.message, frame)
			l.current = (l.current + 1) % len(l.animation)
			l.mu.Unlock()
			time.Sleep(150 * time.Millisecond)
		}
	}()
}

// Stop stops the loading animation
func (l *LoadingAnimation) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.active {
		return
	}
	l.active = false
	fmt.Printf("\r%s\r", strings.Repeat(" ", len(l.message)+15))
}

// Complete completes the loading animation with a message
func (l *LoadingAnimation) Complete(message string) {
	l.Stop()
	fmt.Printf("✓ %s\n", message)
}

// getTerminalWidth returns the terminal width, defaulting to 80 if unable to determine
func getTerminalWidth() int {
	if runtime.GOOS == "windows" {
		// Windows-specific terminal width detection
		type coord struct {
			x int16
			y int16
		}
		type small_rect struct {
			left   int16
			top    int16
			right  int16
			bottom int16
		}
		type consoleScreenBufferInfo struct {
			size              coord
			cursorPosition    coord
			attributes        uint16
			window            small_rect
			maximumWindowSize coord
		}

		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		getConsoleScreenBufferInfo := kernel32.NewProc("GetConsoleScreenBufferInfo")

		handle, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
		if err != nil {
			return 80
		}

		var info consoleScreenBufferInfo
		ret, _, _ := getConsoleScreenBufferInfo.Call(uintptr(handle), uintptr(unsafe.Pointer(&info)))
		if ret == 0 {
			return 80
		}

		return int(info.size.x)
	}

	// Unix-like systems
	width := 80
	if term := os.Getenv("TERM"); term != "" {
		// Try to get terminal width from environment
		if cols := os.Getenv("COLUMNS"); cols != "" {
			if w, err := strconv.Atoi(cols); err == nil && w > 0 {
				width = w
			}
		}
	}

	return width
}
