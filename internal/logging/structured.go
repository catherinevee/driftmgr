package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	Stack     string                 `json:"stack,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Provider  string                 `json:"provider,omitempty"`
	Operation string                 `json:"operation,omitempty"`
	Duration  float64                `json:"duration_ms,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// Logger provides structured logging capabilities
type Logger struct {
	level       LogLevel
	output      io.Writer
	errorOutput io.Writer
	mu          sync.Mutex
	fields      map[string]interface{}
	logFile     *os.File
	requestID   string
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// InitLogger initializes the global logger
func InitLogger(logPath string, level LogLevel) error {
	var err error
	once.Do(func() {
		defaultLogger = &Logger{
			level:       level,
			output:      os.Stdout,
			errorOutput: os.Stderr,
			fields:      make(map[string]interface{}),
		}

		if logPath != "" {
			// Ensure log directory exists
			logDir := filepath.Dir(logPath)
			if err = os.MkdirAll(logDir, 0755); err != nil {
				return
			}

			// Open log file with append mode
			defaultLogger.logFile, err = os.OpenFile(logPath,
				os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				return
			}

			// Write to both file and stdout
			defaultLogger.output = io.MultiWriter(os.Stdout, defaultLogger.logFile)
			defaultLogger.errorOutput = io.MultiWriter(os.Stderr, defaultLogger.logFile)
		}
	})
	return err
}

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	if defaultLogger == nil {
		InitLogger("", INFO)
	}
	return defaultLogger
}

// WithField adds a field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := &Logger{
		level:       l.level,
		output:      l.output,
		errorOutput: l.errorOutput,
		fields:      make(map[string]interface{}),
		requestID:   l.requestID,
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value

	return newLogger
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		level:       l.level,
		output:      l.output,
		errorOutput: l.errorOutput,
		fields:      make(map[string]interface{}),
		requestID:   l.requestID,
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithRequestID sets the request ID for tracing
func (l *Logger) WithRequestID(id string) *Logger {
	newLogger := &Logger{
		level:       l.level,
		output:      l.output,
		errorOutput: l.errorOutput,
		fields:      make(map[string]interface{}),
		requestID:   id,
		logFile:     l.logFile,
		// mu is not copied - new logger gets its own mutex
	}
	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// log writes a log entry
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level.String(),
		Message:   message,
		Fields:    make(map[string]interface{}),
		RequestID: l.requestID,
	}

	// Add caller information
	if _, file, line, ok := runtime.Caller(2); ok {
		entry.Caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// Merge fields
	for k, v := range l.fields {
		entry.Fields[k] = v
	}
	for k, v := range fields {
		entry.Fields[k] = v
	}

	// Add stack trace for errors
	if level >= ERROR {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		entry.Stack = string(buf[:n])
	}

	// Output based on level
	output := l.output
	if level >= ERROR {
		output = l.errorOutput
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(output, "Failed to marshal log entry: %v\n", err)
		return
	}

	fmt.Fprintln(output, string(jsonData))
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(DEBUG, message, f)
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(INFO, message, f)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(WARN, message, f)
}

// Error logs an error message
func (l *Logger) Error(message string, err error, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	if err != nil {
		f["error"] = err.Error()
		f["error_type"] = fmt.Sprintf("%T", err)
	}
	l.log(ERROR, message, f)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string, err error, fields ...map[string]interface{}) {
	f := make(map[string]interface{})
	if len(fields) > 0 {
		f = fields[0]
	}
	if err != nil {
		f["error"] = err.Error()
		f["error_type"] = fmt.Sprintf("%T", err)
	}
	l.log(FATAL, message, f)

	// Ensure logs are flushed
	if l.logFile != nil {
		l.logFile.Sync()
	}

	os.Exit(1)
}

// Close closes the logger and flushes any pending writes
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		if err := l.logFile.Sync(); err != nil {
			return err
		}
		return l.logFile.Close()
	}
	return nil
}

// Audit logs security-sensitive operations
func (l *Logger) Audit(operation string, user string, result string, fields map[string]interface{}) {
	auditFields := map[string]interface{}{
		"audit":     true,
		"operation": operation,
		"user":      user,
		"result":    result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	for k, v := range fields {
		auditFields[k] = v
	}

	l.log(INFO, fmt.Sprintf("AUDIT: %s by %s - %s", operation, user, result), auditFields)
}

// Metric logs performance metrics
func (l *Logger) Metric(name string, value float64, unit string, tags map[string]string) {
	metricFields := map[string]interface{}{
		"metric":    true,
		"name":      name,
		"value":     value,
		"unit":      unit,
		"tags":      tags,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	l.log(INFO, fmt.Sprintf("METRIC: %s = %.2f %s", name, value, unit), metricFields)
}

// Global convenience functions

// Debug logs a debug message using the default logger
func Debug(message string, fields ...map[string]interface{}) {
	GetLogger().Debug(message, fields...)
}

// Info logs an info message using the default logger
func Info(message string, fields ...map[string]interface{}) {
	GetLogger().Info(message, fields...)
}

// Warn logs a warning message using the default logger
func Warn(message string, fields ...map[string]interface{}) {
	GetLogger().Warn(message, fields...)
}

// Error logs an error message using the default logger
func Error(message string, err error, fields ...map[string]interface{}) {
	GetLogger().Error(message, err, fields...)
}

// Fatal logs a fatal message using the default logger and exits
func Fatal(message string, err error, fields ...map[string]interface{}) {
	GetLogger().Fatal(message, err, fields...)
}

// Audit logs an audit event using the default logger
func Audit(operation string, user string, result string, fields map[string]interface{}) {
	GetLogger().Audit(operation, user, result, fields)
}

// Metric logs a metric using the default logger
func Metric(name string, value float64, unit string, tags map[string]string) {
	GetLogger().Metric(name, value, unit, tags)
}
