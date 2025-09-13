package monitoring

import (
	"bytes"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogLevel(t *testing.T) {
	levels := []LogLevel{
		DEBUG,
		INFO,
		WARNING,
		ERROR,
	}

	expectedValues := []int{
		0,
		1,
		2,
		3,
	}

	for i, level := range levels {
		assert.Equal(t, LogLevel(expectedValues[i]), level)
	}
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger()

	assert.NotNil(t, logger)
	assert.NotNil(t, logger.infoLogger)
	assert.NotNil(t, logger.errorLogger)
	assert.NotNil(t, logger.warningLogger)
	assert.NotNil(t, logger.debugLogger)
	assert.Equal(t, INFO, logger.currentLevel)
	assert.NotZero(t, logger.startTime)
}

func TestLogger_SetLogLevel(t *testing.T) {
	logger := NewLogger()

	logger.SetLogLevel(DEBUG)
	assert.Equal(t, DEBUG, logger.currentLevel)

	logger.SetLogLevel(ERROR)
	assert.Equal(t, ERROR, logger.currentLevel)
}

func TestLogger_GetLogLevel(t *testing.T) {
	logger := NewLogger()

	logger.SetLogLevel(WARNING)
	assert.Equal(t, WARNING, logger.GetLogLevel())
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
		{"invalid", INFO, true},
	}

	for _, tt := range tests {
		err := logger.SetLogLevelFromString(tt.input)
		if tt.hasError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, logger.GetLogLevel())
		}
	}
}

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.infoLogger = log.New(&buf, "[INFO] ", 0)

	buf.Reset()
	logger.Info("info message")
	assert.Contains(t, buf.String(), "info message")
	assert.Contains(t, buf.String(), "[INFO]")
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.errorLogger = log.New(&buf, "[ERROR] ", 0)

	buf.Reset()
	logger.Error("error message")
	assert.Contains(t, buf.String(), "error message")
	assert.Contains(t, buf.String(), "[ERROR]")
}

func TestLogger_Warning(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.warningLogger = log.New(&buf, "[WARNING] ", 0)

	buf.Reset()
	logger.Warning("warning message")
	assert.Contains(t, buf.String(), "warning message")
	assert.Contains(t, buf.String(), "[WARNING]")
}

func TestLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.debugLogger = log.New(&buf, "[DEBUG] ", 0)
	logger.SetLogLevel(DEBUG)

	buf.Reset()
	logger.Debug("debug message")
	assert.Contains(t, buf.String(), "debug message")
	assert.Contains(t, buf.String(), "[DEBUG]")
}

func TestLogger_FilterByLevel(t *testing.T) {
	var infoBuf, errorBuf, warnBuf, debugBuf bytes.Buffer
	logger := NewLogger()
	logger.infoLogger = log.New(&infoBuf, "[INFO] ", 0)
	logger.errorLogger = log.New(&errorBuf, "[ERROR] ", 0)
	logger.warningLogger = log.New(&warnBuf, "[WARNING] ", 0)
	logger.debugLogger = log.New(&debugBuf, "[DEBUG] ", 0)

	// Set to WARNING level
	logger.SetLogLevel(WARNING)

	// Debug should not log
	debugBuf.Reset()
	logger.Debug("debug")
	assert.Empty(t, debugBuf.String())

	// Info should not log
	infoBuf.Reset()
	logger.Info("info")
	assert.Empty(t, infoBuf.String())

	// Warning should log
	warnBuf.Reset()
	logger.Warning("warning")
	assert.Contains(t, warnBuf.String(), "warning")

	// Error should log
	errorBuf.Reset()
	logger.Error("error")
	assert.Contains(t, errorBuf.String(), "error")
}

func TestLogger_LogRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.infoLogger = log.New(&buf, "[INFO] ", 0)

	buf.Reset()
	logger.LogRequest("GET", "/api/health", "192.168.1.1", 200, 100*time.Millisecond)
	output := buf.String()
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/api/health")
	assert.Contains(t, output, "192.168.1.1")
	assert.Contains(t, output, "200")
}

func TestLogger_LogError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.errorLogger = log.New(&buf, "[ERROR] ", 0)

	buf.Reset()
	testErr := fmt.Errorf("test error")
	logger.LogError(testErr, "test context")
	output := buf.String()
	assert.Contains(t, output, "test error")
	assert.Contains(t, output, "test context")
}

func TestLogger_GetUptime(t *testing.T) {
	logger := NewLogger()
	logger.startTime = time.Now().Add(-5 * time.Second)

	uptime := logger.GetUptime()
	assert.True(t, uptime >= 5*time.Second)
	assert.True(t, uptime < 6*time.Second)
}

func TestLogger_GetStats(t *testing.T) {
	logger := NewLogger()

	stats := logger.GetStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "uptime")
	assert.Contains(t, stats, "started")
}

func TestGetGlobalLogger(t *testing.T) {
	logger1 := GetGlobalLogger()
	logger2 := GetGlobalLogger()

	// Should return the same instance
	assert.Equal(t, logger1, logger2)
	assert.NotNil(t, logger1)
}

func TestLogger_WithField(t *testing.T) {
	logger := NewLogger()

	newLogger := logger.WithField("key", "value")
	assert.NotNil(t, newLogger)
	// Current implementation just returns the same logger
	assert.Equal(t, logger, newLogger)
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
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, logger.getLevelName(tt.level))
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.infoLogger = log.New(&buf, "[INFO] ", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message %d", i)
	}
}

func BenchmarkLogger_FilteredLog(b *testing.B) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.debugLogger = log.New(&buf, "[DEBUG] ", 0)
	logger.SetLogLevel(INFO) // Debug messages will be filtered

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Debug("filtered message %d", i)
	}
}
