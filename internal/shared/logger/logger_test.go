package monitoring

import (
	"bytes"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger()

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.infoLogger)
	assert.NotNil(t, logger.errorLogger)
	assert.NotNil(t, logger.warningLogger)
	assert.NotNil(t, logger.debugLogger)
	assert.Equal(t, INFO, logger.currentLevel)
	assert.False(t, logger.startTime.IsZero())
}

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		infoLogger:   createTestLogger(&buf, "[INFO] "),
		currentLevel: INFO,
	}

	logger.Info("Test info message")
	output := buf.String()
	assert.Contains(t, output, "[INFO]")
	assert.Contains(t, output, "Test info message")
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		errorLogger:  createTestLogger(&buf, "[ERROR] "),
		currentLevel: ERROR,
	}

	logger.Error("Test error message")
	output := buf.String()
	assert.Contains(t, output, "[ERROR]")
	assert.Contains(t, output, "Test error message")
}

func TestLogger_Warning(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		warningLogger: createTestLogger(&buf, "[WARNING] "),
		currentLevel:  WARNING,
	}

	logger.Warning("Test warning message")
	output := buf.String()
	assert.Contains(t, output, "[WARNING]")
	assert.Contains(t, output, "Test warning message")
}

func TestLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		debugLogger:  createTestLogger(&buf, "[DEBUG] "),
		currentLevel: DEBUG,
	}

	logger.Debug("Test debug message")
	output := buf.String()
	assert.Contains(t, output, "[DEBUG]")
	assert.Contains(t, output, "Test debug message")
}

func TestLogger_LogLevelFiltering(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		method   func(*Logger)
		expected bool
	}{
		{"DEBUG level with Debug", DEBUG, func(l *Logger) { l.Debug("test") }, true},
		{"DEBUG level with Info", DEBUG, func(l *Logger) { l.Info("test") }, true},
		{"DEBUG level with Warning", DEBUG, func(l *Logger) { l.Warning("test") }, true},
		{"DEBUG level with Error", DEBUG, func(l *Logger) { l.Error("test") }, true},
		{"INFO level with Debug", INFO, func(l *Logger) { l.Debug("test") }, false},
		{"INFO level with Info", INFO, func(l *Logger) { l.Info("test") }, true},
		{"INFO level with Warning", INFO, func(l *Logger) { l.Warning("test") }, true},
		{"INFO level with Error", INFO, func(l *Logger) { l.Error("test") }, true},
		{"WARNING level with Debug", WARNING, func(l *Logger) { l.Debug("test") }, false},
		{"WARNING level with Info", WARNING, func(l *Logger) { l.Info("test") }, false},
		{"WARNING level with Warning", WARNING, func(l *Logger) { l.Warning("test") }, true},
		{"WARNING level with Error", WARNING, func(l *Logger) { l.Error("test") }, true},
		{"ERROR level with Debug", ERROR, func(l *Logger) { l.Debug("test") }, false},
		{"ERROR level with Info", ERROR, func(l *Logger) { l.Info("test") }, false},
		{"ERROR level with Warning", ERROR, func(l *Logger) { l.Warning("test") }, false},
		{"ERROR level with Error", ERROR, func(l *Logger) { l.Error("test") }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &Logger{
				infoLogger:    createTestLogger(&buf, "[INFO] "),
				errorLogger:   createTestLogger(&buf, "[ERROR] "),
				warningLogger: createTestLogger(&buf, "[WARNING] "),
				debugLogger:   createTestLogger(&buf, "[DEBUG] "),
				currentLevel:  tt.level,
			}

			tt.method(logger)

			output := buf.String()
			if tt.expected {
				assert.NotEmpty(t, output)
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

func TestLogger_LogRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		infoLogger:   createTestLogger(&buf, "[INFO] "),
		currentLevel: INFO,
	}

	logger.LogRequest("GET", "/api/test", "127.0.0.1:8080", 200, 100*time.Millisecond)

	output := buf.String()
	assert.Contains(t, output, "[INFO]")
	assert.Contains(t, output, "HTTP GET /api/test from 127.0.0.1:8080 - 200")
	assert.Contains(t, output, "100ms")
}

func TestLogger_LogError(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		errorLogger:  createTestLogger(&buf, "[ERROR] "),
		currentLevel: ERROR,
	}

	err := assert.AnError
	logger.LogError(err, "test context")

	output := buf.String()
	assert.Contains(t, output, "[ERROR]")
	assert.Contains(t, output, "Error in test context:")
	assert.Contains(t, output, "assert.AnError")
}

func TestLogger_GetUptime(t *testing.T) {
	logger := NewLogger()

	// Wait a bit to ensure uptime is measurable
	time.Sleep(10 * time.Millisecond)

	uptime := logger.GetUptime()
	assert.Greater(t, uptime, time.Duration(0))
	assert.Less(t, uptime, time.Second) // Should be less than a second
}

func TestLogger_GetStats(t *testing.T) {
	logger := NewLogger()

	stats := logger.GetStats()

	assert.Contains(t, stats, "uptime")
	assert.Contains(t, stats, "started")

	uptime, ok := stats["uptime"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, uptime)

	started, ok := stats["started"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, started)

	// Verify the started time is in RFC3339 format
	_, err := time.Parse(time.RFC3339, started)
	assert.NoError(t, err)
}

func TestGetGlobalLogger(t *testing.T) {
	// Reset global logger
	globalLogger = nil
	globalLoggerOnce = sync.Once{}

	// First call should create a new logger
	logger1 := GetGlobalLogger()
	assert.NotNil(t, logger1)

	// Second call should return the same instance
	logger2 := GetGlobalLogger()
	assert.Equal(t, logger1, logger2)
}

func TestLogger_WithField(t *testing.T) {
	logger := NewLogger()

	// WithField should return the same logger (simplified implementation)
	fieldLogger := logger.WithField("key", "value")
	assert.Equal(t, logger, fieldLogger)
}

func TestLogger_SetLogLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		infoLogger:   createTestLogger(&buf, "[INFO] "),
		currentLevel: INFO,
	}

	// Set to DEBUG level
	logger.SetLogLevel(DEBUG)
	assert.Equal(t, DEBUG, logger.GetLogLevel())

	// Set to ERROR level
	logger.SetLogLevel(ERROR)
	assert.Equal(t, ERROR, logger.GetLogLevel())
}

func TestLogger_GetLogLevel(t *testing.T) {
	logger := NewLogger()

	// Default level should be INFO
	assert.Equal(t, INFO, logger.GetLogLevel())

	// Set to DEBUG and verify
	logger.SetLogLevel(DEBUG)
	assert.Equal(t, DEBUG, logger.GetLogLevel())

	// Set to WARNING and verify
	logger.SetLogLevel(WARNING)
	assert.Equal(t, WARNING, logger.GetLogLevel())

	// Set to ERROR and verify
	logger.SetLogLevel(ERROR)
	assert.Equal(t, ERROR, logger.GetLogLevel())
}

func TestLogger_getLevelName(t *testing.T) {
	logger := NewLogger()

	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARNING, "WARNING"},
		{ERROR, "ERROR"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := logger.getLevelName(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogger_SetLogLevelFromString(t *testing.T) {
	logger := NewLogger()

	tests := []struct {
		input    string
		expected LogLevel
		hasError bool
	}{
		{"DEBUG", DEBUG, false},
		{"debug", DEBUG, false},
		{"INFO", INFO, false},
		{"info", INFO, false},
		{"WARNING", WARNING, false},
		{"warning", WARNING, false},
		{"WARN", WARNING, false},
		{"warn", WARNING, false},
		{"ERROR", ERROR, false},
		{"error", ERROR, false},
		{"INVALID", ERROR, true}, // Will use current level
		{"", ERROR, true},        // Will use current level
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := logger.SetLogLevelFromString(tt.input)

			if tt.hasError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid log level")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, logger.GetLogLevel())
			}
		})
	}
}

func TestLogger_ConcurrentAccess(t *testing.T) {
	logger := NewLogger()

	// Test concurrent logging
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()
			logger.Info("Concurrent message %d", i)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent level changes
	for i := 0; i < 5; i++ {
		go func(i int) {
			defer func() { done <- true }()
			level := LogLevel(i % 4)
			logger.SetLogLevel(level)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}
}

func TestLogger_FormatString(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		infoLogger:   createTestLogger(&buf, "[INFO] "),
		currentLevel: INFO,
	}

	// Test format string with multiple arguments
	logger.Info("User %s logged in with ID %d", "john", 123)

	output := buf.String()
	assert.Contains(t, output, "[INFO]")
	assert.Contains(t, output, "User john logged in with ID 123")
}

func TestLogger_EmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		infoLogger:   createTestLogger(&buf, "[INFO] "),
		currentLevel: INFO,
	}

	logger.Info("")

	output := buf.String()
	assert.Contains(t, output, "[INFO]")
}

func TestLogger_LogLevelConstants(t *testing.T) {
	// Test that log level constants are in the correct order
	assert.True(t, DEBUG < INFO)
	assert.True(t, INFO < WARNING)
	assert.True(t, WARNING < ERROR)
}

// Helper function to create a test logger with a custom writer
func createTestLogger(w io.Writer, prefix string) *log.Logger {
	return log.New(w, prefix, 0) // No flags for cleaner test output
}
