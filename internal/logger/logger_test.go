package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogger(t *testing.T) {
	logger := New("test")
	assert.NotNil(t, logger)
}

func TestLoggerLevels(t *testing.T) {
	logger := New("test")
	assert.NotNil(t, logger)

	// Test different log levels
	logger.Debug("debug message", String("key", "value"))
	logger.Info("info message", Int("count", 42))
	logger.Warn("warning message", Bool("flag", true))
	logger.Error("error message", Float64("value", 3.14))
}

func TestLoggerWithContext(t *testing.T) {
	logger := New("test")
	ctx := context.WithValue(context.Background(), "trace_id", "12345")
	
	contextLogger := logger.WithContext(ctx)
	assert.NotNil(t, contextLogger)
	
	contextLogger.Info("message with context", String("operation", "test"))
}

func TestLoggerFields(t *testing.T) {
	logger := New("test")
	
	// Test various field types
	logger.Info("test fields",
		String("string", "value"),
		Int("int", 42),
		Int64("int64", int64(999)),
		Float64("float", 3.14),
		Bool("bool", true),
		Error(nil),
		Any("any", map[string]interface{}{"key": "value"}),
	)
}

func TestLoggerConcurrency(t *testing.T) {
	logger := New("test")
	
	// Test concurrent logging
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			logger.Info("concurrent log", Int("goroutine", id))
			done <- true
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGlobalLogger(t *testing.T) {
	// Test global logger functions
	Printf("formatted %s %d", "message", 42)
	Println("line message")
	
	// These should not panic
	assert.NotPanics(t, func() {
		Printf("safe message %d", 123)
		Println("safe line")
	})
}

func BenchmarkLogger(b *testing.B) {
	logger := New("bench")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			String("key", "value"),
			Int("count", i),
		)
	}
}

func BenchmarkLoggerWithFields(b *testing.B) {
	logger := New("bench")
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			String("field1", "value1"),
			String("field2", "value2"),
			Int("field3", i),
			Bool("field4", true),
			Float64("field5", 3.14),
		)
	}
}