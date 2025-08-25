package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/catherinevee/driftmgr/internal/logger"
	"github.com/catherinevee/driftmgr/internal/telemetry"
)

// ShutdownHandler manages graceful shutdown of the application
type ShutdownHandler struct {
	timeout    time.Duration
	callbacks  []ShutdownCallback
	mu         sync.RWMutex
	shutdownCh chan struct{}
	done       chan struct{}
	log        logger.Logger
}

// ShutdownCallback is a function called during shutdown
type ShutdownCallback struct {
	Name     string
	Priority int // Lower numbers run first
	Fn       func(context.Context) error
}

// NewShutdownHandler creates a new shutdown handler
func NewShutdownHandler(timeout time.Duration) *ShutdownHandler {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &ShutdownHandler{
		timeout:    timeout,
		callbacks:  make([]ShutdownCallback, 0),
		shutdownCh: make(chan struct{}),
		done:       make(chan struct{}),
		log:        logger.New("shutdown_handler"),
	}
}

// Register adds a shutdown callback
func (h *ShutdownHandler) Register(callback ShutdownCallback) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Insert in priority order
	inserted := false
	for i, cb := range h.callbacks {
		if callback.Priority < cb.Priority {
			h.callbacks = append(h.callbacks[:i], append([]ShutdownCallback{callback}, h.callbacks[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		h.callbacks = append(h.callbacks, callback)
	}

	h.log.Debug("Registered shutdown callback",
		logger.String("name", callback.Name),
		logger.Int("priority", callback.Priority),
	)
}

// Listen starts listening for shutdown signals
func (h *ShutdownHandler) Listen() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		sig := <-sigCh
		h.log.Info("Received shutdown signal",
			logger.String("signal", sig.String()),
		)
		h.Shutdown()
	}()

	h.log.Info("Shutdown handler listening for signals")
}

// Shutdown initiates graceful shutdown
func (h *ShutdownHandler) Shutdown() {
	select {
	case <-h.shutdownCh:
		// Already shutting down
		return
	default:
		close(h.shutdownCh)
	}

	h.log.Info("Starting graceful shutdown",
		logger.Duration("timeout", h.timeout),
	)

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	// Track shutdown with telemetry
	if telemetry.Get() != nil {
		_, span := telemetry.Get().StartSpan(ctx, "application.shutdown")
		defer span.End()
	}

	h.mu.RLock()
	callbacks := make([]ShutdownCallback, len(h.callbacks))
	copy(callbacks, h.callbacks)
	h.mu.RUnlock()

	// Execute callbacks in priority order
	var wg sync.WaitGroup
	errCh := make(chan error, len(callbacks))

	for _, callback := range callbacks {
		wg.Add(1)
		go func(cb ShutdownCallback) {
			defer wg.Done()

			h.log.Info("Executing shutdown callback",
				logger.String("name", cb.Name),
			)

			start := time.Now()
			if err := cb.Fn(ctx); err != nil {
				h.log.Error("Shutdown callback failed",
					logger.String("name", cb.Name),
					logger.Error(err),
					logger.Duration("duration", time.Since(start)),
				)
				errCh <- err
			} else {
				h.log.Info("Shutdown callback completed",
					logger.String("name", cb.Name),
					logger.Duration("duration", time.Since(start)),
				)
			}
		}(callback)
	}

	// Wait for all callbacks or timeout
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		h.log.Info("Graceful shutdown completed successfully")
	case <-ctx.Done():
		h.log.Warn("Graceful shutdown timed out, forcing exit")
	}

	close(h.done)
}

// Wait blocks until shutdown is complete
func (h *ShutdownHandler) Wait() {
	<-h.done
}

// IsShuttingDown returns true if shutdown has been initiated
func (h *ShutdownHandler) IsShuttingDown() bool {
	select {
	case <-h.shutdownCh:
		return true
	default:
		return false
	}
}

// Manager manages application lifecycle
type Manager struct {
	shutdown     *ShutdownHandler
	healthChecks []HealthCheck
	mu           sync.RWMutex
	log          logger.Logger
}

// HealthCheck represents a health check function
type HealthCheck struct {
	Name    string
	Check   func(context.Context) error
	Timeout time.Duration
}

// NewManager creates a new lifecycle manager
func NewManager(shutdownTimeout time.Duration) *Manager {
	return &Manager{
		shutdown:     NewShutdownHandler(shutdownTimeout),
		healthChecks: make([]HealthCheck, 0),
		log:          logger.New("lifecycle_manager"),
	}
}

// RegisterShutdown registers a shutdown callback
func (m *Manager) RegisterShutdown(name string, priority int, fn func(context.Context) error) {
	m.shutdown.Register(ShutdownCallback{
		Name:     name,
		Priority: priority,
		Fn:       fn,
	})
}

// RegisterHealthCheck registers a health check
func (m *Manager) RegisterHealthCheck(check HealthCheck) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if check.Timeout <= 0 {
		check.Timeout = 5 * time.Second
	}

	m.healthChecks = append(m.healthChecks, check)

	m.log.Debug("Registered health check",
		logger.String("name", check.Name),
		logger.Duration("timeout", check.Timeout),
	)
}

// CheckHealth runs all health checks
func (m *Manager) CheckHealth(ctx context.Context) (HealthStatus, error) {
	m.mu.RLock()
	checks := make([]HealthCheck, len(m.healthChecks))
	copy(checks, m.healthChecks)
	m.mu.RUnlock()

	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Checks:    make([]HealthCheckResult, 0),
	}

	for _, check := range checks {
		checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)

		start := time.Now()
		err := check.Check(checkCtx)
		duration := time.Since(start)

		cancel()

		result := HealthCheckResult{
			Name:     check.Name,
			Duration: duration,
		}

		if err != nil {
			result.Status = "unhealthy"
			result.Error = err.Error()
			status.Status = "unhealthy"
		} else {
			result.Status = "healthy"
		}

		status.Checks = append(status.Checks, result)
	}

	return status, nil
}

// Start starts the lifecycle manager
func (m *Manager) Start() {
	m.shutdown.Listen()
	m.log.Info("Lifecycle manager started")
}

// Shutdown initiates graceful shutdown
func (m *Manager) Shutdown() {
	m.shutdown.Shutdown()
}

// Wait waits for shutdown to complete
func (m *Manager) Wait() {
	m.shutdown.Wait()
}

// IsShuttingDown returns true if shutdown is in progress
func (m *Manager) IsShuttingDown() bool {
	return m.shutdown.IsShuttingDown()
}

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    string              `json:"status"`
	Timestamp time.Time           `json:"timestamp"`
	Checks    []HealthCheckResult `json:"checks"`
}

// HealthCheckResult represents a single health check result
type HealthCheckResult struct {
	Name     string        `json:"name"`
	Status   string        `json:"status"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}
