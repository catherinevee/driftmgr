package graceful

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/catherinevee/driftmgr/internal/utils/errors"
	"github.com/catherinevee/driftmgr/internal/observability/logging"
	"github.com/rs/zerolog"
)

// Handler manages graceful shutdown and error handling
type Handler struct {
	logger       *zerolog.Logger
	shutdownFunc []func() error
	mu           sync.Mutex
	exitCode     int
	isDone       chan struct{}
}

// Global handler instance
var defaultHandler = &Handler{
	logger:   func() *zerolog.Logger { l := logging.WithComponent("graceful"); return &l }(),
	isDone:   make(chan struct{}),
	exitCode: 0,
}

// OnShutdown registers a function to be called during shutdown
func OnShutdown(fn func() error) {
	defaultHandler.mu.Lock()
	defer defaultHandler.mu.Unlock()
	defaultHandler.shutdownFunc = append(defaultHandler.shutdownFunc, fn)
}

// HandleError handles an error gracefully without panic or log.Fatal
func HandleError(err error, message string) {
	if err == nil {
		return
	}

	defaultHandler.logger.Error().
		Err(err).
		Msg(message)

	// Set exit code based on error type
	if errors.Is(err, errors.ErrorTypeValidation) {
		defaultHandler.exitCode = 2
	} else if errors.Is(err, errors.ErrorTypeNotFound) {
		defaultHandler.exitCode = 3
	} else {
		defaultHandler.exitCode = 1
	}

	// Trigger graceful shutdown
	Shutdown()
}

// HandleErrorf handles an error with formatted message
func HandleErrorf(err error, format string, args ...interface{}) {
	HandleError(err, fmt.Sprintf(format, args...))
}

// HandleCritical handles critical errors that require immediate shutdown
func HandleCritical(err error, message string) {
	if err == nil {
		return
	}

	defaultHandler.logger.Error().
		Err(err).
		Str("severity", "CRITICAL").
		Msg(message)

	defaultHandler.exitCode = 1

	// Immediate shutdown for critical errors
	performShutdown(5 * time.Second)
	os.Exit(defaultHandler.exitCode)
}

// Shutdown initiates graceful shutdown
func Shutdown() {
	performShutdown(30 * time.Second)
	os.Exit(defaultHandler.exitCode)
}

// performShutdown executes all shutdown functions
func performShutdown(timeout time.Duration) {
	defaultHandler.logger.Info().Msg("Initiating graceful shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan bool, 1)

	go func() {
		defaultHandler.mu.Lock()
		funcs := defaultHandler.shutdownFunc
		defaultHandler.mu.Unlock()

		for i := len(funcs) - 1; i >= 0; i-- {
			if err := funcs[i](); err != nil {
				defaultHandler.logger.Error().
					Err(err).
					Int("handler", i).
					Msg("Shutdown handler failed")
			}
		}
		done <- true
	}()

	select {
	case <-done:
		defaultHandler.logger.Info().Msg("Graceful shutdown completed")
	case <-ctx.Done():
		defaultHandler.logger.Warn().Msg("Shutdown timeout exceeded, forcing exit")
	}

	close(defaultHandler.isDone)
}

// WaitForSignal waits for interrupt signals and handles graceful shutdown
func WaitForSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		defaultHandler.logger.Info().
			Str("signal", sig.String()).
			Msg("Received shutdown signal")
		Shutdown()
	case <-defaultHandler.isDone:
		// Already shutting down
	}
}

// Exit sets the exit code and initiates shutdown
func Exit(code int) {
	defaultHandler.exitCode = code
	Shutdown()
}

// SetLogger sets a custom logger for the handler
func SetLogger(logger zerolog.Logger) {
	defaultHandler.logger = &logger
}

// RecoverPanic recovers from panics and converts them to graceful errors
func RecoverPanic() {
	if r := recover(); r != nil {
		err := fmt.Errorf("panic recovered: %v", r)
		HandleCritical(err, "Application panic")
	}
}

// Wrap wraps a function with panic recovery
func Wrap(fn func() error) error {
	defer RecoverPanic()
	return fn()
}

// WrapMain wraps the main function with graceful error handling
func WrapMain(mainFunc func() error) {
	defer RecoverPanic()

	// Set up signal handling
	go WaitForSignal()

	// Run the main function
	if err := mainFunc(); err != nil {
		HandleError(err, "Application error")
	}
}