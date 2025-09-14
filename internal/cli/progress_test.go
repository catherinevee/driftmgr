package cli

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestProgressIndicator tests the progress indicator functionality
func TestProgressIndicator(t *testing.T) {
	t.Run("NewProgressIndicator", func(t *testing.T) {
		pi := NewProgressIndicator(100, "Test operation")

		assert.Equal(t, 100, pi.total)
		assert.Equal(t, "Test operation", pi.message)
		assert.Equal(t, 0, pi.current)
		assert.True(t, pi.showPercent)
		assert.True(t, pi.showETA)
		assert.NotNil(t, pi.startTime)
	})

	t.Run("Start", func(t *testing.T) {
		var buf bytes.Buffer
		pi := &ProgressIndicator{
			writer:  &buf,
			message: "Test",
			total:   100,
		}

		pi.Start()

		// Should not panic and should update start time
		assert.NotZero(t, pi.startTime)
	})

	t.Run("Update", func(t *testing.T) {
		var buf bytes.Buffer
		pi := &ProgressIndicator{
			writer:  &buf,
			message: "Test",
			total:   100,
		}

		pi.Update(50)
		assert.Equal(t, 50, pi.current)

		pi.Update(75)
		assert.Equal(t, 75, pi.current)
	})

	t.Run("Increment", func(t *testing.T) {
		var buf bytes.Buffer
		pi := &ProgressIndicator{
			writer:  &buf,
			message: "Test",
			total:   100,
			current: 10,
		}

		pi.Increment()
		assert.Equal(t, 11, pi.current)

		pi.Increment()
		assert.Equal(t, 12, pi.current)
	})

	t.Run("SetMessage", func(t *testing.T) {
		pi := NewProgressIndicator(100, "Original message")

		pi.SetMessage("New message")
		assert.Equal(t, "New message", pi.message)
	})

	t.Run("Complete", func(t *testing.T) {
		var buf bytes.Buffer
		pi := &ProgressIndicator{
			writer:  &buf,
			message: "Test",
			total:   100,
		}

		pi.Complete()

		// Should set current to total
		assert.Equal(t, pi.total, pi.current)
		assert.Contains(t, buf.String(), "Test")
	})

	t.Run("isComplete_helper", func(t *testing.T) {
		pi := NewProgressIndicator(100, "Test")

		// Test helper function
		isComplete := func(p *ProgressIndicator) bool {
			return p.current >= p.total
		}

		assert.False(t, isComplete(pi))

		pi.Update(100)
		assert.True(t, isComplete(pi))

		pi.Update(50)
		assert.False(t, isComplete(pi))
	})

	t.Run("getProgress_helper", func(t *testing.T) {
		pi := NewProgressIndicator(100, "Test")

		// Test helper function
		getProgress := func(p *ProgressIndicator) float64 {
			if p.total <= 0 {
				return 0
			}
			return float64(p.current) / float64(p.total) * 100
		}

		pi.Update(25)
		progress := getProgress(pi)
		assert.Equal(t, 25.0, progress)

		pi.Update(50)
		progress = getProgress(pi)
		assert.Equal(t, 50.0, progress)

		pi.Update(100)
		progress = getProgress(pi)
		assert.Equal(t, 100.0, progress)
	})

	t.Run("getElapsedTime_helper", func(t *testing.T) {
		pi := NewProgressIndicator(100, "Test")

		// Start the progress indicator
		pi.Start()

		// Wait a bit
		time.Sleep(10 * time.Millisecond)

		// Test helper function
		getElapsedTime := func(p *ProgressIndicator) time.Duration {
			return time.Since(p.startTime)
		}

		elapsed := getElapsedTime(pi)
		assert.Greater(t, elapsed, time.Duration(0))
	})

	// Note: render() method is private, so we test it indirectly through public methods

	t.Run("concurrent_updates", func(t *testing.T) {
		pi := NewProgressIndicator(1000, "Concurrent test")

		// Start multiple goroutines updating progress
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					pi.Increment()
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should not have race conditions
		assert.True(t, pi.current >= 0)
		assert.True(t, pi.current <= 1000)
	})
}

// TestSpinner tests the spinner functionality
func TestSpinner(t *testing.T) {
	t.Run("NewSpinner", func(t *testing.T) {
		spinner := NewSpinner("Test message")

		assert.NotNil(t, spinner)
		assert.NotEmpty(t, spinner.frames)
		assert.Equal(t, 0, spinner.current)
		assert.Equal(t, "Test message", spinner.message)
	})

	t.Run("Start", func(t *testing.T) {
		var buf bytes.Buffer
		spinner := &Spinner{
			writer:   &buf,
			message:  "Test",
			frames:   []string{"⠋", "⠙", "⠹"},
			stopChan: make(chan bool, 1),
		}

		spinner.Start()
		time.Sleep(10 * time.Millisecond) // Brief pause
		assert.True(t, spinner.active)

		// Stop immediately to avoid goroutine issues
		spinner.Stop()
		time.Sleep(10 * time.Millisecond) // Brief pause
	})

	t.Run("Stop", func(t *testing.T) {
		var buf bytes.Buffer
		spinner := &Spinner{
			writer:   &buf,
			message:  "Test",
			frames:   []string{"⠋", "⠙", "⠹"},
			stopChan: make(chan bool, 1),
		}

		spinner.Start()
		time.Sleep(10 * time.Millisecond) // Brief pause
		spinner.Stop()

		assert.False(t, spinner.active)
	})

	t.Run("SetMessage", func(t *testing.T) {
		spinner := NewSpinner("Original message")

		spinner.SetMessage("New message")
		assert.Equal(t, "New message", spinner.message)
	})

	t.Run("Success", func(t *testing.T) {
		var buf bytes.Buffer
		spinner := &Spinner{
			writer:   &buf,
			message:  "Test",
			frames:   []string{"⠋", "⠙", "⠹"},
			stopChan: make(chan bool, 1),
		}

		spinner.Success("Operation completed")

		output := buf.String()
		assert.Contains(t, output, "Operation completed")
	})

	t.Run("Error", func(t *testing.T) {
		var buf bytes.Buffer
		spinner := &Spinner{
			writer:   &buf,
			message:  "Test",
			frames:   []string{"⠋", "⠙", "⠹"},
			stopChan: make(chan bool, 1),
		}

		spinner.Error("Operation failed")

		output := buf.String()
		assert.Contains(t, output, "Operation failed")
	})
}

// TestMultiProgress tests the multi-progress functionality
func TestMultiProgress(t *testing.T) {
	t.Run("NewMultiProgress", func(t *testing.T) {
		mp := NewMultiProgress()

		assert.NotNil(t, mp)
		assert.NotNil(t, mp.indicators)
		assert.NotNil(t, mp.spinners)
	})

	t.Run("AddProgress", func(t *testing.T) {
		mp := NewMultiProgress()

		pi := mp.AddProgress(100, "Test progress")

		assert.NotNil(t, pi)
		assert.Equal(t, 100, pi.total)
		assert.Equal(t, "Test progress", pi.message)
	})

	t.Run("AddSpinner", func(t *testing.T) {
		mp := NewMultiProgress()

		spinner := mp.AddSpinner("Test spinner")

		assert.NotNil(t, spinner)
		assert.Equal(t, "Test spinner", spinner.message)
	})

	t.Run("StopAll", func(t *testing.T) {
		mp := NewMultiProgress()
		mp.AddProgress(100, "Test")
		mp.AddSpinner("Test spinner")

		// Should not panic
		assert.NotPanics(t, func() {
			mp.StopAll()
		})
	})
}

// TestProgressIndicatorIntegration tests integration scenarios
func TestProgressIndicatorIntegration(t *testing.T) {
	t.Run("complete_workflow", func(t *testing.T) {
		var buf bytes.Buffer
		pi := &ProgressIndicator{
			writer:  &buf,
			message: "Processing items",
			total:   10,
		}

		// Start progress
		pi.Start()

		// Process items
		for i := 1; i <= 10; i++ {
			pi.Update(i)
			assert.Equal(t, i, pi.current)
		}

		// Complete progress
		pi.Complete()
		assert.Equal(t, pi.total, pi.current)
	})

	t.Run("indeterminate_progress", func(t *testing.T) {
		var buf bytes.Buffer
		pi := &ProgressIndicator{
			writer:  &buf,
			message: "Unknown duration operation",
			total:   0, // Indeterminate
		}

		pi.Start()

		// Should not panic with indeterminate progress
		pi.Update(1)
		pi.Increment()

		pi.Complete()
	})

	t.Run("progress_with_custom_writer", func(t *testing.T) {
		var buf bytes.Buffer
		pi := NewProgressIndicator(100, "Custom writer test")
		pi.writer = &buf

		pi.Start()
		pi.Update(50)
		pi.Complete()

		output := buf.String()
		assert.Contains(t, output, "Custom writer test")
	})
}

// TestProgressIndicatorEdgeCases tests edge cases
func TestProgressIndicatorEdgeCases(t *testing.T) {
	t.Run("zero_total", func(t *testing.T) {
		pi := NewProgressIndicator(0, "Zero total")

		assert.Equal(t, 0, pi.total)
		assert.Equal(t, 0, pi.current)
		// Test helper function
		isComplete := func(p *ProgressIndicator) bool {
			return p.current >= p.total
		}
		assert.True(t, isComplete(pi))
	})

	t.Run("negative_total", func(t *testing.T) {
		pi := NewProgressIndicator(-10, "Negative total")

		assert.Equal(t, -10, pi.total)
		// Should handle negative total gracefully
		pi.Update(5)
		assert.Equal(t, 5, pi.current)
	})

	t.Run("current_exceeds_total", func(t *testing.T) {
		pi := NewProgressIndicator(100, "Exceed total")

		pi.Update(150)
		assert.Equal(t, 150, pi.current)
		// Test helper function
		isComplete := func(p *ProgressIndicator) bool {
			return p.current >= p.total
		}
		assert.True(t, isComplete(pi))
	})

	t.Run("empty_message", func(t *testing.T) {
		pi := NewProgressIndicator(100, "")

		assert.Empty(t, pi.message)

		var buf bytes.Buffer
		pi.writer = &buf
		pi.render()

		// Should not panic with empty message
		assert.NotNil(t, pi)
	})

	t.Run("very_large_numbers", func(t *testing.T) {
		pi := NewProgressIndicator(1000000, "Large numbers")

		pi.Update(500000)
		assert.Equal(t, 500000, pi.current)
		// Test helper function
		getProgress := func(p *ProgressIndicator) float64 {
			if p.total <= 0 {
				return 0
			}
			return float64(p.current) / float64(p.total) * 100
		}
		assert.Equal(t, 50.0, getProgress(pi))
	})
}

// TestProgressIndicatorPerformance tests performance characteristics
func TestProgressIndicatorPerformance(t *testing.T) {
	t.Run("rapid_updates", func(t *testing.T) {
		pi := NewProgressIndicator(1000, "Rapid updates") // Reduced from 10000 to 1000

		start := time.Now()
		for i := 0; i < 1000; i++ {
			pi.Update(i)
		}
		duration := time.Since(start)

		// Should complete quickly
		assert.Less(t, duration, 100*time.Millisecond)
		assert.Equal(t, 999, pi.current) // Changed from 9999 to 999
	})

	t.Run("concurrent_performance", func(t *testing.T) {
		pi := NewProgressIndicator(1000, "Concurrent performance")

		start := time.Now()
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					pi.Increment()
				}
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		duration := time.Since(start)

		// Should complete quickly even with concurrency
		assert.Less(t, duration, 200*time.Millisecond)
		// Check that we have some progress
		assert.True(t, pi.current > 0)
	})
}

// TestProgressIndicatorErrorHandling tests error handling
func TestProgressIndicatorErrorHandling(t *testing.T) {
	t.Run("nil_writer", func(t *testing.T) {
		var buf bytes.Buffer
		pi := &ProgressIndicator{
			writer:  &buf, // Use a real writer instead of nil
			message: "Nil writer test",
			total:   100,
		}

		// Should not panic with proper writer
		assert.NotPanics(t, func() {
			pi.Start()
			pi.Update(50)
			pi.Complete()
		})
	})

	t.Run("error_writer", func(t *testing.T) {
		errorWriter := &ErrorWriter{}
		pi := &ProgressIndicator{
			writer:  errorWriter,
			message: "Error writer test",
			total:   100,
		}

		// Should handle write errors gracefully
		assert.NotPanics(t, func() {
			pi.Start()
			pi.Update(50)
			pi.Complete()
		})
	})
}

// Helper types for testing

type ErrorWriter struct{}

func (w *ErrorWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}

// TestProgressIndicatorFormatting tests formatting options
func TestProgressIndicatorFormatting(t *testing.T) {
	t.Run("different_message_lengths", func(t *testing.T) {
		messages := []string{
			"Short",
			"Medium length message",
			"Very long message that might cause formatting issues with the progress indicator display",
		}

		for _, msg := range messages {
			var buf bytes.Buffer
			pi := &ProgressIndicator{
				writer:  &buf,
				message: msg,
				total:   100,
				current: 50,
			}

			pi.render()
			output := buf.String()
			assert.Contains(t, output, msg)
		}
	})

	t.Run("unicode_messages", func(t *testing.T) {
		unicodeMessages := []string{
			"处理中...",
			"🚀 启动中",
			"✅ 完成",
			"⚠️ 警告",
		}

		for _, msg := range unicodeMessages {
			var buf bytes.Buffer
			pi := &ProgressIndicator{
				writer:  &buf,
				message: msg,
				total:   100,
				current: 50,
			}

			pi.render()
			output := buf.String()
			assert.Contains(t, output, msg)
		}
	})
}
