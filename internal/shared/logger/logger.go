package logger

import (
	"fmt"
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
	currentLevel  LogLevel
	mu            sync.RWMutex
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
		currentLevel:  INFO, // Default to INFO level
	}
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	l.mu.RLock()
	level := l.currentLevel
	l.mu.RUnlock()

	if level <= INFO {
		l.infoLogger.Printf(format, v...)
	}
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.mu.RLock()
	level := l.currentLevel
	l.mu.RUnlock()

	if level <= ERROR {
		l.errorLogger.Printf(format, v...)
	}
}

// Warning logs a warning message
func (l *Logger) Warning(format string, v ...interface{}) {
	l.mu.RLock()
	level := l.currentLevel
	l.mu.RUnlock()

	if level <= WARNING {
		l.warningLogger.Printf(format, v...)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.mu.RLock()
	level := l.currentLevel
	l.mu.RUnlock()

	if level <= DEBUG {
		l.debugLogger.Printf(format, v...)
	}
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

// SetLogLevel sets the minimum log level
func (l *Logger) SetLogLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()

	oldLevel := l.currentLevel
	l.currentLevel = level

	// Log the change at the appropriate level
	levelName := l.getLevelName(level)
	oldLevelName := l.getLevelName(oldLevel)

	// Force log this message regardless of level since it's important
	l.infoLogger.Printf("Log level changed from %s to %s", oldLevelName, levelName)
}

// GetLogLevel returns the current log level
func (l *Logger) GetLogLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.currentLevel
}

// getLevelName returns the string representation of a log level
func (l *Logger) getLevelName(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARNING"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// SetLogLevelFromString sets the log level from a string
func (l *Logger) SetLogLevelFromString(levelStr string) error {
	var level LogLevel
	switch levelStr {
	case "DEBUG", "debug":
		level = DEBUG
	case "INFO", "info":
		level = INFO
	case "WARNING", "warning", "WARN", "warn":
		level = WARNING
	case "ERROR", "error":
		level = ERROR
	default:
		return fmt.Errorf("invalid log level: %s", levelStr)
	}

	l.SetLogLevel(level)
	return nil
}
