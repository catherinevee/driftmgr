package resilience

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/catherinevee/driftmgr/internal/telemetry"
	"github.com/catherinevee/driftmgr/internal/utils/errors"
)

// State represents the state of the circuit breaker
type State int32

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

var (
	ErrCircuitOpen     = errors.New(errors.ErrorTypeRateLimit, "circuit breaker is open")
	ErrTooManyRequests = errors.New(errors.ErrorTypeRateLimit, "too many requests in half-open state")
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name             string
	maxFailures      int32
	resetTimeout     time.Duration
	halfOpenMaxCalls int32

	state           int32 // atomic
	failures        int32 // atomic
	lastFailureTime int64 // atomic (unix nano)
	halfOpenCalls   int32 // atomic

	successCount int64 // atomic
	failureCount int64 // atomic
	totalCalls   int64 // atomic

	mu            sync.RWMutex
	onStateChange func(from, to State)
	isFailure     func(error) bool
}

// Config represents circuit breaker configuration
type Config struct {
	Name             string
	MaxFailures      int32
	ResetTimeout     time.Duration
	HalfOpenMaxCalls int32
	OnStateChange    func(from, to State)
	IsFailure        func(error) bool
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config Config) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:             config.Name,
		maxFailures:      config.MaxFailures,
		resetTimeout:     config.ResetTimeout,
		halfOpenMaxCalls: config.HalfOpenMaxCalls,
		onStateChange:    config.OnStateChange,
		isFailure:        config.IsFailure,
	}

	if cb.maxFailures <= 0 {
		cb.maxFailures = 5
	}

	if cb.resetTimeout <= 0 {
		cb.resetTimeout = 60 * time.Second
	}

	if cb.halfOpenMaxCalls <= 0 {
		cb.halfOpenMaxCalls = 3
	}

	if cb.isFailure == nil {
		cb.isFailure = func(err error) bool {
			return err != nil
		}
	}

	return cb
}

// Call executes the function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	// Record telemetry
	if telemetry.Get() != nil {
		_, span := telemetry.Get().StartSpan(ctx, "circuit_breaker.call")
		defer span.End()
	}

	// Check if we can proceed
	if err := cb.canProceed(); err != nil {
		atomic.AddInt64(&cb.totalCalls, 1)
		return nil, err
	}

	// Execute the function
	atomic.AddInt64(&cb.totalCalls, 1)
	result, err := fn()

	// Update circuit breaker state based on result
	cb.onResult(err)

	return result, err
}

// canProceed checks if a call can proceed
func (cb *CircuitBreaker) canProceed() error {
	state := cb.currentState()

	switch state {
	case StateClosed:
		return nil

	case StateOpen:
		// Check if we should transition to half-open
		lastFailure := atomic.LoadInt64(&cb.lastFailureTime)
		if time.Since(time.Unix(0, lastFailure)) > cb.resetTimeout {
			cb.transitionTo(StateHalfOpen)
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// Limit the number of calls in half-open state
		calls := atomic.AddInt32(&cb.halfOpenCalls, 1)
		if calls > cb.halfOpenMaxCalls {
			atomic.AddInt32(&cb.halfOpenCalls, -1)
			return ErrTooManyRequests
		}
		return nil

	default:
		return ErrCircuitOpen
	}
}

// onResult updates the circuit breaker state based on the result
func (cb *CircuitBreaker) onResult(err error) {
	state := cb.currentState()

	if cb.isFailure(err) {
		cb.onFailure(state)
	} else {
		cb.onSuccess(state)
	}
}

// onSuccess handles successful calls
func (cb *CircuitBreaker) onSuccess(state State) {
	atomic.AddInt64(&cb.successCount, 1)

	switch state {
	case StateClosed:
		// Reset failure count on success
		atomic.StoreInt32(&cb.failures, 0)

	case StateHalfOpen:
		// Check if we should close the circuit
		calls := atomic.LoadInt32(&cb.halfOpenCalls)
		if calls >= cb.halfOpenMaxCalls {
			cb.transitionTo(StateClosed)
			atomic.StoreInt32(&cb.failures, 0)
			atomic.StoreInt32(&cb.halfOpenCalls, 0)
		}
	}
}

// onFailure handles failed calls
func (cb *CircuitBreaker) onFailure(state State) {
	atomic.AddInt64(&cb.failureCount, 1)
	atomic.StoreInt64(&cb.lastFailureTime, time.Now().UnixNano())

	switch state {
	case StateClosed:
		failures := atomic.AddInt32(&cb.failures, 1)
		if failures >= cb.maxFailures {
			cb.transitionTo(StateOpen)
		}

	case StateHalfOpen:
		// Any failure in half-open state opens the circuit
		cb.transitionTo(StateOpen)
		atomic.StoreInt32(&cb.halfOpenCalls, 0)
	}
}

// currentState returns the current state
func (cb *CircuitBreaker) currentState() State {
	return State(atomic.LoadInt32(&cb.state))
}

// transitionTo transitions to a new state
func (cb *CircuitBreaker) transitionTo(newState State) {
	oldState := State(atomic.SwapInt32(&cb.state, int32(newState)))

	if oldState != newState {
		if cb.onStateChange != nil {
			cb.onStateChange(oldState, newState)
		}
	}
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	atomic.StoreInt32(&cb.state, int32(StateClosed))
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.halfOpenCalls, 0)
	atomic.StoreInt64(&cb.lastFailureTime, 0)
}

// Stats returns circuit breaker statistics
func (cb *CircuitBreaker) Stats() Stats {
	return Stats{
		Name:         cb.name,
		State:        cb.currentState(),
		Failures:     atomic.LoadInt32(&cb.failures),
		SuccessCount: atomic.LoadInt64(&cb.successCount),
		FailureCount: atomic.LoadInt64(&cb.failureCount),
		TotalCalls:   atomic.LoadInt64(&cb.totalCalls),
		LastFailure:  time.Unix(0, atomic.LoadInt64(&cb.lastFailureTime)),
	}
}

// String returns the string representation of a state
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Stats represents circuit breaker statistics
type Stats struct {
	Name         string    `json:"name"`
	State        State     `json:"state"`
	Failures     int32     `json:"failures"`
	SuccessCount int64     `json:"success_count"`
	FailureCount int64     `json:"failure_count"`
	TotalCalls   int64     `json:"total_calls"`
	LastFailure  time.Time `json:"last_failure"`
}

// Manager manages multiple circuit breakers
type Manager struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
}

// NewManager creates a new circuit breaker manager
func NewManager() *Manager {
	return &Manager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets or creates a circuit breaker
func (m *Manager) GetOrCreate(name string, config Config) *CircuitBreaker {
	m.mu.RLock()
	if cb, exists := m.breakers[name]; exists {
		m.mu.RUnlock()
		return cb
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists := m.breakers[name]; exists {
		return cb
	}

	config.Name = name
	cb := NewCircuitBreaker(config)
	m.breakers[name] = cb

	return cb
}

// Get returns a circuit breaker by name
func (m *Manager) Get(name string) (*CircuitBreaker, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cb, exists := m.breakers[name]
	return cb, exists
}

// ResetAll resets all circuit breakers
func (m *Manager) ResetAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, cb := range m.breakers {
		cb.Reset()
	}
}

// Stats returns statistics for all circuit breakers
func (m *Manager) Stats() map[string]Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]Stats)
	for name, cb := range m.breakers {
		stats[name] = cb.Stats()
	}

	return stats
}
