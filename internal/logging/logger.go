package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/constants"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
	FATAL
)

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warning(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	WithContext(ctx context.Context) Logger
	WithFields(fields ...Field) Logger
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// StructuredLogger implements the Logger interface
type StructuredLogger struct {
	mu         sync.RWMutex
	level      LogLevel
	output     io.Writer
	fields     map[string]interface{}
	timeFormat string
	context    context.Context
}

// NewLogger creates a new structured logger
func NewLogger(level string, output io.Writer) *StructuredLogger {
	if output == nil {
		output = os.Stdout
	}

	return &StructuredLogger{
		level:      parseLogLevel(level),
		output:     output,
		fields:     make(map[string]interface{}),
		timeFormat: time.RFC3339,
	}
}

// parseLogLevel converts string to LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case constants.LogLevelDebug:
		return DEBUG
	case constants.LogLevelInfo:
		return INFO
	case constants.LogLevelWarning:
		return WARNING
	case constants.LogLevelError:
		return ERROR
	case constants.LogLevelFatal:
		return FATAL
	default:
		return INFO
	}
}

// levelToString converts LogLevel to string
func levelToString(level LogLevel) string {
	switch level {
	case DEBUG:
		return constants.LogLevelDebug
	case INFO:
		return constants.LogLevelInfo
	case WARNING:
		return constants.LogLevelWarning
	case ERROR:
		return constants.LogLevelError
	case FATAL:
		return constants.LogLevelFatal
	default:
		return "UNKNOWN"
	}
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(msg string, fields ...Field) {
	l.log(DEBUG, msg, fields...)
}

// Info logs an info message
func (l *StructuredLogger) Info(msg string, fields ...Field) {
	l.log(INFO, msg, fields...)
}

// Warning logs a warning message
func (l *StructuredLogger) Warning(msg string, fields ...Field) {
	l.log(WARNING, msg, fields...)
}

// Error logs an error message
func (l *StructuredLogger) Error(msg string, fields ...Field) {
	l.log(ERROR, msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *StructuredLogger) Fatal(msg string, fields ...Field) {
	l.log(FATAL, msg, fields...)
	os.Exit(1)
}

// WithContext returns a logger with context
func (l *StructuredLogger) WithContext(ctx context.Context) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &StructuredLogger{
		level:      l.level,
		output:     l.output,
		fields:     make(map[string]interface{}),
		timeFormat: l.timeFormat,
		context:    ctx,
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add context values if available
	if ctx != nil {
		if traceID := ctx.Value("trace_id"); traceID != nil {
			newLogger.fields["trace_id"] = traceID
		}
		if userID := ctx.Value("user_id"); userID != nil {
			newLogger.fields["user_id"] = userID
		}
		if requestID := ctx.Value("request_id"); requestID != nil {
			newLogger.fields["request_id"] = requestID
		}
	}

	return newLogger
}

// WithFields returns a logger with additional fields
func (l *StructuredLogger) WithFields(fields ...Field) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &StructuredLogger{
		level:      l.level,
		output:     l.output,
		fields:     make(map[string]interface{}),
		timeFormat: l.timeFormat,
		context:    l.context,
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for _, field := range fields {
		newLogger.fields[field.Key] = field.Value
	}

	return newLogger
}

// log performs the actual logging
func (l *StructuredLogger) log(level LogLevel, msg string, fields ...Field) {
	if level < l.level {
		return
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	// Build log entry
	entry := make(map[string]interface{})
	entry["timestamp"] = time.Now().Format(l.timeFormat)
	entry["level"] = levelToString(level)
	entry["message"] = msg

	// Add caller information for error and fatal levels
	if level >= ERROR {
		if pc, file, line, ok := runtime.Caller(2); ok {
			entry["caller"] = fmt.Sprintf("%s:%d", file, line)
			if fn := runtime.FuncForPC(pc); fn != nil {
				entry["function"] = fn.Name()
			}
		}
	}

	// Add persistent fields
	for k, v := range l.fields {
		entry[k] = v
	}

	// Add temporary fields
	for _, field := range fields {
		entry[field.Key] = field.Value
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(l.output, `{"timestamp":"%s","level":"ERROR","message":"Failed to marshal log entry","error":"%v"}`+"\n",
			time.Now().Format(l.timeFormat), err)
		return
	}

	// Write to output
	fmt.Fprintln(l.output, string(data))
}

// Helper functions for creating fields

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a boolean field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Time creates a time field
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value.Format(time.RFC3339)}
}

// Err creates an error field
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Global logger instance
var (
	globalLogger Logger
	once         sync.Once
)

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(level string, output io.Writer) {
	once.Do(func() {
		globalLogger = NewLogger(level, output)
	})
}

// GetLogger returns the global logger
func GetLogger() Logger {
	if globalLogger == nil {
		InitGlobalLogger(constants.LogLevelInfo, os.Stdout)
	}
	return globalLogger
}

// Package-level logging functions using global logger

// Debug logs a debug message
func Debug(msg string, fields ...Field) {
	GetLogger().Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...Field) {
	GetLogger().Info(msg, fields...)
}

// Warning logs a warning message
func Warning(msg string, fields ...Field) {
	GetLogger().Warning(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...Field) {
	GetLogger().Fatal(msg, fields...)
}