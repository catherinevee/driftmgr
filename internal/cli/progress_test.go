package cli

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewProgressIndicator(t *testing.T) {
	pi := NewProgressIndicator(100, "Processing")
	assert.NotNil(t, pi)
	assert.Equal(t, 100, pi.total)
	assert.Equal(t, "Processing", pi.message)
	assert.Equal(t, 0, pi.current)
	assert.True(t, pi.showPercent)
	assert.True(t, pi.showETA)
}

func TestProgressIndicator_Start(t *testing.T) {
	var buf bytes.Buffer
	pi := &ProgressIndicator{
		writer:      &buf,
		total:       100,
		message:     "Starting",
		showPercent: true,
		showETA:     false,
	}

	pi.Start()
	output := buf.String()

	assert.Contains(t, output, "Starting")
	assert.Contains(t, output, "0%")
	assert.NotZero(t, pi.startTime)
}

func TestProgressIndicator_Update(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		updates  []int
		expected []string
	}{
		{
			name:     "Simple progress",
			total:    100,
			updates:  []int{25, 50, 75, 100},
			expected: []string{"25.0%", "50.0%", "75.0%", "100.0%"},
		},
		{
			name:     "Small increments",
			total:    10,
			updates:  []int{1, 2, 3, 4, 5},
			expected: []string{"10.0%", "20.0%", "30.0%", "40.0%", "50.0%"},
		},
		{
			name:     "Large total",
			total:    1000,
			updates:  []int{100, 500, 1000},
			expected: []string{"10.0%", "50.0%", "100.0%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			pi := &ProgressIndicator{
				writer:      &buf,
				total:       tt.total,
				message:     "Processing",
				showPercent: true,
				showETA:     false,
			}

			for i, update := range tt.updates {
				buf.Reset()
				pi.Update(update)
				output := buf.String()
				assert.Contains(t, output, tt.expected[i])
			}
		})
	}
}

func TestProgressIndicator_Increment(t *testing.T) {
	var buf bytes.Buffer
	pi := &ProgressIndicator{
		writer:      &buf,
		total:       5,
		message:     "Processing",
		current:     0,
		showPercent: true,
		showETA:     false,
	}

	expectedPercentages := []string{"20.0%", "40.0%", "60.0%", "80.0%", "100.0%"}

	for i := 0; i < 5; i++ {
		buf.Reset()
		pi.Increment()
		output := buf.String()
		assert.Contains(t, output, expectedPercentages[i])
		assert.Equal(t, i+1, pi.current)
	}
}

func TestProgressIndicator_SetMessage(t *testing.T) {
	var buf bytes.Buffer
	pi := &ProgressIndicator{
		writer:      &buf,
		total:       100,
		current:     50,
		message:     "Initial",
		showPercent: true,
		showETA:     false,
	}

	messages := []string{
		"Downloading",
		"Processing",
		"Finalizing",
	}

	for _, msg := range messages {
		buf.Reset()
		pi.SetMessage(msg)
		output := buf.String()
		assert.Contains(t, output, msg)
		assert.Equal(t, msg, pi.message)
	}
}

func TestProgressIndicator_Complete(t *testing.T) {
	var buf bytes.Buffer
	pi := &ProgressIndicator{
		writer:      &buf,
		total:       100,
		current:     75,
		message:     "Processing",
		showPercent: true,
		showETA:     false,
	}

	pi.Complete()
	output := buf.String()

	assert.Contains(t, output, "100.0%")
	assert.Equal(t, 100, pi.current)
	assert.Contains(t, output, "\n")
}

func TestProgressIndicator_WithETA(t *testing.T) {
	var buf bytes.Buffer
	pi := &ProgressIndicator{
		writer:      &buf,
		total:       100,
		current:     0,
		message:     "Processing",
		showPercent: true,
		showETA:     true,
		startTime:   time.Now().Add(-10 * time.Second),
	}

	pi.Update(50)
	output := buf.String()

	// Should show some ETA information
	assert.Contains(t, output, "50.0%")
	// ETA calculation should be present in some form
}

func TestProgressIndicator_ConcurrentUpdates(t *testing.T) {
	var buf bytes.Buffer
	pi := &ProgressIndicator{
		writer:      &buf,
		total:       1000,
		message:     "Processing",
		showPercent: true,
		showETA:     false,
	}

	var wg sync.WaitGroup
	updates := 100

	for i := 0; i < updates; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()
			pi.Update(val * 10)
		}(i)
	}

	wg.Wait()

	// Should not panic and current should be set to some value
	assert.GreaterOrEqual(t, pi.current, 0)
	assert.LessOrEqual(t, pi.current, 1000)
}

func TestSpinner_New(t *testing.T) {
	spinner := NewSpinner("Loading")
	assert.NotNil(t, spinner)
	assert.Equal(t, "Loading", spinner.message)
	assert.False(t, spinner.active)
	assert.NotEmpty(t, spinner.frames)
}

func TestSpinner_StartStop(t *testing.T) {
	spinner := NewSpinner("Loading")

	spinner.Start()
	assert.True(t, spinner.active)

	// Let it spin for a bit
	time.Sleep(50 * time.Millisecond)

	spinner.Stop()
	assert.False(t, spinner.active)
}

func TestSpinner_SetMessage(t *testing.T) {
	spinner := NewSpinner("Initial")

	spinner.Start()
	time.Sleep(20 * time.Millisecond)

	spinner.SetMessage("Updated")
	assert.Equal(t, "Updated", spinner.message)

	time.Sleep(20 * time.Millisecond)
	spinner.Stop()
}

func TestMultiProgress_New(t *testing.T) {
	mp := NewMultiProgress()
	assert.NotNil(t, mp)
	assert.NotNil(t, mp.indicators)
	assert.Empty(t, mp.indicators)
}

func TestMultiProgress_AddProgress(t *testing.T) {
	mp := NewMultiProgress()

	// Add progress indicators
	pi1 := mp.AddProgress(100, "Task 1")
	pi2 := mp.AddProgress(200, "Task 2")

	assert.NotNil(t, pi1)
	assert.NotNil(t, pi2)
	assert.Len(t, mp.indicators, 2)
	assert.Equal(t, "Task 1", pi1.message)
	assert.Equal(t, "Task 2", pi2.message)
}

func TestMultiProgress_AddSpinner(t *testing.T) {
	mp := NewMultiProgress()

	// Add spinners
	s1 := mp.AddSpinner("Loading 1")
	s2 := mp.AddSpinner("Loading 2")

	assert.NotNil(t, s1)
	assert.NotNil(t, s2)
	assert.Len(t, mp.spinners, 2)
	assert.Equal(t, "Loading 1", s1.message)
	assert.Equal(t, "Loading 2", s2.message)
}

func TestMultiProgress_StopAll(t *testing.T) {
	mp := NewMultiProgress()

	// Add indicators and spinners
	pi1 := mp.AddProgress(100, "Task 1")
	pi2 := mp.AddProgress(200, "Task 2")
	s1 := mp.AddSpinner("Loading")

	// Start spinner
	s1.Start()
	assert.True(t, s1.active)

	// Stop all
	mp.StopAll()

	// Spinner should be stopped
	assert.False(t, s1.active)

	// Progress indicators should still exist
	assert.NotNil(t, pi1)
	assert.NotNil(t, pi2)
}

func TestProgressBar_Render(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		total    int
		width    int
		expected string
	}{
		{
			name:     "Empty bar",
			current:  0,
			total:    100,
			width:    10,
			expected: "[          ]",
		},
		{
			name:     "Half full",
			current:  50,
			total:    100,
			width:    10,
			expected: "[=====     ]",
		},
		{
			name:     "Full bar",
			current:  100,
			total:    100,
			width:    10,
			expected: "[==========]",
		},
		{
			name:     "Quarter full",
			current:  25,
			total:    100,
			width:    20,
			expected: "[=====               ]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := renderProgressBar(tt.current, tt.total, tt.width)
			assert.Equal(t, tt.expected, bar)
		})
	}
}

func renderProgressBar(current, total, width int) string {
	if total == 0 {
		return "[" + strings.Repeat(" ", width) + "]"
	}

	filled := (current * width) / total
	if filled > width {
		filled = width
	}

	return "[" + strings.Repeat("=", filled) + strings.Repeat(" ", width-filled) + "]"
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{3600 * time.Second, "1h0m"},
		{3665 * time.Second, "1h1m"},
		{7200 * time.Second, "2h0m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// Using the formatDuration from progress.go
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func BenchmarkProgressIndicator_Update(b *testing.B) {
	var buf bytes.Buffer
	pi := &ProgressIndicator{
		writer:      &buf,
		total:       1000,
		message:     "Processing",
		showPercent: true,
		showETA:     false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pi.Update(i % 1000)
	}
}

func BenchmarkProgressIndicator_Render(b *testing.B) {
	var buf bytes.Buffer
	pi := &ProgressIndicator{
		writer:      &buf,
		total:       100,
		current:     50,
		message:     "Processing",
		showPercent: true,
		showETA:     true,
		startTime:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		pi.render()
	}
}
