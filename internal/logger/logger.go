package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	WithContext(ctx context.Context) Logger
	WithFields(fields ...Field) Logger
	WithError(err error) Logger
	WithTraceID(traceID string) Logger
}

// Field represents a logging field
type Field struct {
	Key   string
	Value interface{}
}

// ZeroLogger implements Logger using zerolog
type ZeroLogger struct {
	logger  zerolog.Logger
	fields  []Field
	context context.Context
}

var (
	globalLogger *ZeroLogger
	once         sync.Once
)

// LogConfig represents logger configuration
type LogConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	Output     string `json:"output"`
	TimeFormat string `json:"time_format"`
	Caller     bool   `json:"caller"`
	Stacktrace bool   `json:"stacktrace"`
}

// Initialize initializes the global logger
func Initialize(config LogConfig) {
	once.Do(func() {
		var output io.Writer

		// Set output
		switch config.Output {
		case "stdout":
			output = os.Stdout
		case "stderr":
			output = os.Stderr
		default:
			file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				output = os.Stdout
			} else {
				output = file
			}
		}

		// Set format
		if config.Format == "console" {
			output = zerolog.ConsoleWriter{
				Out:        output,
				TimeFormat: config.TimeFormat,
			}
		}

		// Set level
		level := parseLevel(config.Level)
		zerolog.SetGlobalLevel(level)

		// Create logger
		logger := zerolog.New(output).With().Timestamp()

		if config.Caller {
			logger = logger.Caller()
		}

		globalLogger = &ZeroLogger{
			logger: logger.Logger(),
		}

		// Set global logger
		log.Logger = globalLogger.logger
	})
}

// Get returns the global logger
func Get() Logger {
	if globalLogger == nil {
		// Initialize with defaults if not initialized
		Initialize(LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			TimeFormat: time.RFC3339,
			Caller:     true,
		})
	}
	return globalLogger
}

// New creates a new logger instance
func New(name string) Logger {
	logger := Get().WithFields(String("component", name))
	return logger
}

// WithContext creates a logger with context
func (l *ZeroLogger) WithContext(ctx context.Context) Logger {
	newLogger := &ZeroLogger{
		logger:  l.logger,
		fields:  append([]Field{}, l.fields...),
		context: ctx,
	}

	// Extract trace ID if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		newLogger.fields = append(newLogger.fields, Field{
			Key:   "trace_id",
			Value: span.SpanContext().TraceID().String(),
		})
	}

	return newLogger
}

// WithFields adds fields to the logger
func (l *ZeroLogger) WithFields(fields ...Field) Logger {
	newLogger := &ZeroLogger{
		logger:  l.logger,
		fields:  append(append([]Field{}, l.fields...), fields...),
		context: l.context,
	}
	return newLogger
}

// WithError adds an error to the logger
func (l *ZeroLogger) WithError(err error) Logger {
	if err == nil {
		return l
	}

	fields := []Field{
		String("error", err.Error()),
		String("error_type", fmt.Sprintf("%T", err)),
	}

	// Add stack trace for errors
	if _, file, line, ok := runtime.Caller(1); ok {
		fields = append(fields, String("error_location", fmt.Sprintf("%s:%d", file, line)))
	}

	return l.WithFields(fields...)
}

// WithTraceID adds a trace ID to the logger
func (l *ZeroLogger) WithTraceID(traceID string) Logger {
	return l.WithFields(String("trace_id", traceID))
}

// Debug logs a debug message
func (l *ZeroLogger) Debug(msg string, fields ...Field) {
	event := l.logger.Debug()
	l.logEvent(event, msg, fields...)
}

// Info logs an info message
func (l *ZeroLogger) Info(msg string, fields ...Field) {
	event := l.logger.Info()
	l.logEvent(event, msg, fields...)
}

// Warn logs a warning message
func (l *ZeroLogger) Warn(msg string, fields ...Field) {
	event := l.logger.Warn()
	l.logEvent(event, msg, fields...)
}

// Error logs an error message
func (l *ZeroLogger) Error(msg string, fields ...Field) {
	event := l.logger.Error()
	l.logEvent(event, msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *ZeroLogger) Fatal(msg string, fields ...Field) {
	event := l.logger.Fatal()
	l.logEvent(event, msg, fields...)
}

// logEvent logs an event with fields
func (l *ZeroLogger) logEvent(event *zerolog.Event, msg string, fields ...Field) {
	// Add stored fields
	for _, field := range l.fields {
		event = addField(event, field)
	}

	// Add new fields
	for _, field := range fields {
		event = addField(event, field)
	}

	event.Msg(msg)
}

// addField adds a field to an event
func addField(event *zerolog.Event, field Field) *zerolog.Event {
	switch v := field.Value.(type) {
	case string:
		return event.Str(field.Key, v)
	case int:
		return event.Int(field.Key, v)
	case int64:
		return event.Int64(field.Key, v)
	case float64:
		return event.Float64(field.Key, v)
	case bool:
		return event.Bool(field.Key, v)
	case time.Time:
		return event.Time(field.Key, v)
	case time.Duration:
		return event.Dur(field.Key, v)
	case error:
		return event.Err(v)
	default:
		return event.Interface(field.Key, v)
	}
}

// parseLevel parses a log level string
func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// Field constructors

func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Helper functions for migration

// ReplaceStdLog replaces standard library log with our logger
func ReplaceStdLog() {
	log.Logger = globalLogger.logger
}

// Printf is a compatibility function for fmt.Printf replacement
func Printf(format string, args ...interface{}) {
	Get().Info(fmt.Sprintf(format, args...))
}

// Println is a compatibility function for fmt.Println replacement
func Println(args ...interface{}) {
	Get().Info(fmt.Sprint(args...))
}

// Fatalf is a compatibility function for log.Fatalf replacement
func Fatalf(format string, args ...interface{}) {
	Get().Fatal(fmt.Sprintf(format, args...))
}
