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

func TestLogger_SetLevel(t *testing.T) {
	logger := NewLogger()

	logger.SetLevel(DEBUG)
	assert.Equal(t, DEBUG, logger.currentLevel)

	logger.SetLevel(ERROR)
	assert.Equal(t, ERROR, logger.currentLevel)
}

func TestLogger_Methods(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.infoLogger = log.New(&buf, "[INFO] ", 0)
	logger.errorLogger = log.New(&buf, "[ERROR] ", 0)
	logger.warningLogger = log.New(&buf, "[WARNING] ", 0)
	logger.debugLogger = log.New(&buf, "[DEBUG] ", 0)
	logger.SetLevel(DEBUG)

	// Test Info
	buf.Reset()
	logger.Info("info message")
	assert.Contains(t, buf.String(), "info message")
	assert.Contains(t, buf.String(), "[INFO]")

	// Test Error
	buf.Reset()
	logger.Error("error message")
	assert.Contains(t, buf.String(), "error message")
	assert.Contains(t, buf.String(), "[ERROR]")

	// Test Warning
	buf.Reset()
	logger.Warning("warning message")
	assert.Contains(t, buf.String(), "warning message")
	assert.Contains(t, buf.String(), "[WARNING]")

	// Test Debug
	buf.Reset()
	logger.Debug("debug message")
	assert.Contains(t, buf.String(), "debug message")
	assert.Contains(t, buf.String(), "[DEBUG]")
}

func TestLogger_Infof(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.infoLogger = log.New(&buf, "[INFO] ", 0)

	buf.Reset()
	logger.Infof("formatted %s %d", "message", 123)
	assert.Contains(t, buf.String(), "formatted message 123")
}

func TestLogger_Errorf(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.errorLogger = log.New(&buf, "[ERROR] ", 0)

	buf.Reset()
	logger.Errorf("error: %s", "something went wrong")
	assert.Contains(t, buf.String(), "error: something went wrong")
}

func TestLogger_Warningf(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.warningLogger = log.New(&buf, "[WARNING] ", 0)

	buf.Reset()
	logger.Warningf("warning: %s", "be careful")
	assert.Contains(t, buf.String(), "warning: be careful")
}

func TestLogger_Debugf(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.debugLogger = log.New(&buf, "[DEBUG] ", 0)
	logger.SetLevel(DEBUG)

	buf.Reset()
	logger.Debugf("debug: %v", map[string]int{"count": 5})
	assert.Contains(t, buf.String(), "debug: map[count:5]")
}

func TestLogger_FilterByLevel(t *testing.T) {
	var infoBuf, errorBuf, warnBuf, debugBuf bytes.Buffer
	logger := NewLogger()
	logger.infoLogger = log.New(&infoBuf, "[INFO] ", 0)
	logger.errorLogger = log.New(&errorBuf, "[ERROR] ", 0)
	logger.warningLogger = log.New(&warnBuf, "[WARNING] ", 0)
	logger.debugLogger = log.New(&debugBuf, "[DEBUG] ", 0)

	// Set to WARNING level
	logger.SetLevel(WARNING)

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

func TestGetLogger(t *testing.T) {
	logger1 := GetLogger()
	logger2 := GetLogger()

	// Should return the same instance
	assert.Equal(t, logger1, logger2)
	assert.NotNil(t, logger1)
}

func TestLogger_ElapsedTime(t *testing.T) {
	logger := NewLogger()
	logger.startTime = time.Now().Add(-5 * time.Second)

	elapsed := logger.ElapsedTime()
	assert.True(t, elapsed >= 5*time.Second)
	assert.True(t, elapsed < 6*time.Second)
}

func BenchmarkLogger_Info(b *testing.B) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.infoLogger = log.New(&buf, "[INFO] ", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLogger_Infof(b *testing.B) {
	var buf bytes.Buffer
	logger := NewLogger()
	logger.infoLogger = log.New(&buf, "[INFO] ", 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Infof("benchmark %s %d", "message", i)
	}
}
