package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger provides structured logging capabilities
type Logger struct {
	name   string
	level  LogLevel
	output *log.Logger
}

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// NewLogger creates a new logger instance
func NewLogger(name string) *Logger {
	return &Logger{
		name:   name,
		level:  LogLevelInfo,
		output: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.level <= LogLevelDebug {
		l.log("DEBUG", msg, args...)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		l.log("INFO", msg, args...)
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		l.log("WARN", msg, args...)
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	if l.level <= LogLevelError {
		l.log("ERROR", msg, args...)
	}
}

// log formats and outputs a log message
func (l *Logger) log(level, msg string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")

	// Format the message with arguments
	formattedMsg := msg
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(msg, args...)
	}

	logEntry := fmt.Sprintf("[%s] %s [%s] %s", timestamp, level, l.name, formattedMsg)
	l.output.Println(logEntry)
}
