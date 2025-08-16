package monitoring

import (
	"log"
	"os"
	"sync"
	"time"
)

// Global logger instance
var globalLogger *Logger
var globalLoggerOnce sync.Once

// Logger provides structured logging functionality
type Logger struct {
	infoLogger    *log.Logger
	errorLogger   *log.Logger
	warningLogger *log.Logger
	debugLogger   *log.Logger
	startTime     time.Time
}

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
)

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	flags := log.LstdFlags | log.Lshortfile

	return &Logger{
		infoLogger:    log.New(os.Stdout, "[INFO] ", flags),
		errorLogger:   log.New(os.Stderr, "[ERROR] ", flags),
		warningLogger: log.New(os.Stdout, "[WARNING] ", flags),
		debugLogger:   log.New(os.Stdout, "[DEBUG] ", flags),
		startTime:     time.Now(),
	}
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.infoLogger.Printf(format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.errorLogger.Printf(format, v...)
}

// Warning logs a warning message
func (l *Logger) Warning(format string, v ...interface{}) {
	l.warningLogger.Printf(format, v...)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.debugLogger.Printf(format, v...)
}

// LogRequest logs an HTTP request
func (l *Logger) LogRequest(method, path, remoteAddr string, statusCode int, duration time.Duration) {
	l.Info("HTTP %s %s from %s - %d (%v)", method, path, remoteAddr, statusCode, duration)
}

// LogError logs an error with context
func (l *Logger) LogError(err error, context string) {
	l.Error("Error in %s: %v", context, err)
}

// GetUptime returns the uptime of the logger
func (l *Logger) GetUptime() time.Duration {
	return time.Since(l.startTime)
}

// GetStats returns logger statistics
func (l *Logger) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"uptime":  l.GetUptime().String(),
		"started": l.startTime.Format(time.RFC3339),
	}
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	globalLoggerOnce.Do(func() {
		globalLogger = NewLogger()
	})
	return globalLogger
}

// WithField creates a new logger with an additional field (simplified implementation)
func (l *Logger) WithField(key, value string) *Logger {
	// In a more sophisticated implementation, this would create a new logger with structured fields
	// For now, we'll just return the same logger
	return l
}

// SetLogLevel sets the minimum log level (not implemented in this simple version)
func (l *Logger) SetLogLevel(level LogLevel) {
	// In a more sophisticated implementation, this would control which messages are logged
	l.Info("Log level set to %d", level)
}
