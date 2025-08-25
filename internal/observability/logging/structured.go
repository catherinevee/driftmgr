package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Logger is the global logger instance
	Logger zerolog.Logger

	// contextKey for storing logger in context
	contextKey = struct{}{}
)

// Config represents logging configuration
type Config struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"` // json or console
	Output     string `json:"output" yaml:"output"` // stdout, stderr, or file path
	TimeFormat string `json:"time_format" yaml:"time_format"`
	Caller     bool   `json:"caller" yaml:"caller"`
	StackTrace bool   `json:"stack_trace" yaml:"stack_trace"`
}

// DefaultConfig returns default logging configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		TimeFormat: time.RFC3339,
		Caller:     true,
		StackTrace: false,
	}
}

// Init initializes the global logger with the given configuration
func Init(cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Set log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Set time format
	zerolog.TimeFieldFormat = cfg.TimeFormat

	// Configure output
	var output io.Writer
	switch strings.ToLower(cfg.Output) {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// File output
		file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	}

	// Configure format
	if cfg.Format == "console" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: cfg.TimeFormat,
		}
	}

	// Create logger
	logContext := zerolog.New(output).With().Timestamp()

	// Add caller information if enabled
	if cfg.Caller {
		logContext = logContext.Caller()
	}

	// Add hostname
	if hostname, err := os.Hostname(); err == nil {
		logContext = logContext.Str("hostname", hostname)
	}

	// Add service info
	logContext = logContext.
		Str("service", "driftmgr").
		Str("version", getVersion())

	Logger = logContext.Logger()

	// Set global logger
	log.Logger = Logger

	return nil
}

// WithContext adds logger to context
func WithContext(ctx context.Context, logger zerolog.Logger) context.Context {
	return context.WithValue(ctx, contextKey, logger)
}

// FromContext retrieves logger from context
func FromContext(ctx context.Context) zerolog.Logger {
	if logger, ok := ctx.Value(contextKey).(zerolog.Logger); ok {
		return logger
	}
	return Logger
}

// WithFields returns a logger with additional fields
func WithFields(fields map[string]interface{}) zerolog.Logger {
	logger := Logger.With()
	for k, v := range fields {
		logger = logger.Interface(k, v)
	}
	return logger.Logger()
}

// WithError returns a logger with error field
func WithError(err error) zerolog.Logger {
	return Logger.With().Err(err).Logger()
}

// WithCorrelationID returns a logger with correlation ID
func WithCorrelationID(id string) zerolog.Logger {
	return Logger.With().Str("correlation_id", id).Logger()
}

// WithComponent returns a logger for a specific component
func WithComponent(component string) zerolog.Logger {
	return Logger.With().Str("component", component).Logger()
}

// WithProvider returns a logger for a specific cloud provider
func WithProvider(provider string) zerolog.Logger {
	return Logger.With().Str("provider", provider).Logger()
}

// WithResource returns a logger with resource information
func WithResource(resourceType, resourceID string) zerolog.Logger {
	return Logger.With().
		Str("resource_type", resourceType).
		Str("resource_id", resourceID).
		Logger()
}

// Audit logs an audit event
func Audit(ctx context.Context, action string, fields map[string]interface{}) {
	logger := FromContext(ctx).With().
		Str("audit", "true").
		Str("action", action).
		Timestamp().
		Logger()

	for k, v := range fields {
		logger = logger.With().Interface(k, v).Logger()
	}

	logger.Info().Msg("audit event")
}

// Metric logs a metric event
func Metric(name string, value float64, tags map[string]string) {
	logger := Logger.With().
		Str("metric_name", name).
		Float64("metric_value", value).
		Logger()

	for k, v := range tags {
		logger = logger.With().Str(fmt.Sprintf("tag_%s", k), v).Logger()
	}

	logger.Debug().Msg("metric")
}

// getVersion returns the application version
func getVersion() string {
	// This should be set during build time
	if version := os.Getenv("DRIFTMGR_VERSION"); version != "" {
		return version
	}
	return "dev"
}

// Fatal logs a fatal error and exits the application gracefully
func Fatal(ctx context.Context, msg string, fields ...map[string]interface{}) {
	logger := FromContext(ctx)
	event := logger.Fatal()

	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}

	// Add stack trace
	event = event.Str("stack_trace", getStackTrace())

	event.Msg(msg)
}

// getStackTrace returns current stack trace
func getStackTrace() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])

	var sb strings.Builder
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "runtime/") {
			sb.WriteString(fmt.Sprintf("%s:%d %s\n",
				filepath.Base(frame.File), frame.Line, frame.Function))
		}
		if !more {
			break
		}
	}

	return sb.String()
}

// Debug logs a debug message
func Debug(msg string, fields ...map[string]interface{}) {
	event := Logger.Debug()
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Info logs an info message
func Info(msg string, fields ...map[string]interface{}) {
	event := Logger.Info()
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Warn logs a warning message
func Warn(msg string, fields ...map[string]interface{}) {
	event := Logger.Warn()
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

// Error logs an error message
func Error(msg string, err error, fields ...map[string]interface{}) {
	event := Logger.Error().Err(err)
	if len(fields) > 0 {
		for k, v := range fields[0] {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}
