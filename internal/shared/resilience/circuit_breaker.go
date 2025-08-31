package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/catherinevee/driftmgr/internal/logging"
)

// State represents the circuit breaker state
type State int32

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name             string
	maxFailures      int32
	resetTimeout     time.Duration
	halfOpenRequests int32
	onStateChange    func(from, to State)

	state           int32 // atomic
	failures        int32 // atomic
	successCount    int32 // atomic
	lastFailureTime int64 // atomic (unix nano)
	totalRequests   int64 // atomic
	totalFailures   int64 // atomic

	mu     sync.RWMutex
	logger *logging.Logger
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	Name             string
	MaxFailures      int
	ResetTimeout     time.Duration
	HalfOpenRequests int
	OnStateChange    func(from, to State)
}

// ErrCircuitOpen is returned when the circuit is open
var ErrCircuitOpen = errors.New("circuit breaker is open")

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:             config.Name,
		maxFailures:      int32(config.MaxFailures),
		resetTimeout:     config.ResetTimeout,
		halfOpenRequests: int32(config.HalfOpenRequests),
		onStateChange:    config.OnStateChange,
		logger:           logging.GetLogger(),
	}

	if cb.halfOpenRequests == 0 {
		cb.halfOpenRequests = 1
	}

	cb.logger.Info("Circuit breaker created", map[string]interface{}{
		"name":          cb.name,
		"max_failures":  cb.maxFailures,
		"reset_timeout": cb.resetTimeout.String(),
	})

	return cb
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	return cb.ExecuteContext(context.Background(), func(context.Context) error {
		return fn()
	})
}

// ExecuteContext runs a function with circuit breaker protection and context
func (cb *CircuitBreaker) ExecuteContext(ctx context.Context, fn func(context.Context) error) error {
	if !cb.canExecute() {
		atomic.AddInt64(&cb.totalRequests, 1)

		cb.logger.Debug("Circuit breaker rejected request", map[string]interface{}{
			"name":  cb.name,
			"state": cb.GetState().String(),
		})

		return ErrCircuitOpen
	}

	atomic.AddInt64(&cb.totalRequests, 1)

	// Execute the function
	err := fn(ctx)

	// Record result
	cb.recordResult(err)

	return err
}

// canExecute checks if a request can be executed
func (cb *CircuitBreaker) canExecute() bool {
	state := cb.GetState()

	switch state {
	case StateClosed:
		return true

	case StateOpen:
		// Check if we should transition to half-open
		lastFailure := atomic.LoadInt64(&cb.lastFailureTime)
		if time.Since(time.Unix(0, lastFailure)) > cb.resetTimeout {
			cb.transitionTo(StateHalfOpen)
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited requests in half-open state
		successCount := atomic.LoadInt32(&cb.successCount)
		return successCount < cb.halfOpenRequests

	default:
		return false
	}
}

// recordResult records the result of an execution
func (cb *CircuitBreaker) recordResult(err error) {
	state := cb.GetState()

	if err != nil {
		cb.onFailure(state)
	} else {
		cb.onSuccess(state)
	}
}

// onSuccess handles successful execution
func (cb *CircuitBreaker) onSuccess(state State) {
	switch state {
	case StateClosed:
		// Reset failure count on success
		atomic.StoreInt32(&cb.failures, 0)

	case StateHalfOpen:
		successCount := atomic.AddInt32(&cb.successCount, 1)

		// Check if we should close the circuit
		if successCount >= cb.halfOpenRequests {
			cb.transitionTo(StateClosed)
			atomic.StoreInt32(&cb.failures, 0)
			atomic.StoreInt32(&cb.successCount, 0)

			cb.logger.Info("Circuit breaker recovered", map[string]interface{}{
				"name": cb.name,
			})
		}
	}
}

// onFailure handles failed execution
func (cb *CircuitBreaker) onFailure(state State) {
	atomic.AddInt64(&cb.totalFailures, 1)
	atomic.StoreInt64(&cb.lastFailureTime, time.Now().UnixNano())

	switch state {
	case StateClosed:
		failures := atomic.AddInt32(&cb.failures, 1)

		// Check if we should open the circuit
		if failures >= cb.maxFailures {
			cb.transitionTo(StateOpen)

			cb.logger.Warn("Circuit breaker opened", map[string]interface{}{
				"name":     cb.name,
				"failures": failures,
			})

			// Record metric
			logging.Metric("circuit_breaker.opened", 1, "count", map[string]string{
				"name": cb.name,
			})
		}

	case StateHalfOpen:
		// Any failure in half-open state reopens the circuit
		cb.transitionTo(StateOpen)
		atomic.StoreInt32(&cb.successCount, 0)

		cb.logger.Debug("Circuit breaker reopened from half-open", map[string]interface{}{
			"name": cb.name,
		})
	}
}

// transitionTo transitions to a new state
func (cb *CircuitBreaker) transitionTo(newState State) {
	oldState := State(atomic.SwapInt32(&cb.state, int32(newState)))

	if oldState != newState {
		cb.logger.Info("Circuit breaker state changed", map[string]interface{}{
			"name": cb.name,
			"from": oldState.String(),
			"to":   newState.String(),
		})

		if cb.onStateChange != nil {
			cb.onStateChange(oldState, newState)
		}

		// Record metric
		logging.Metric("circuit_breaker.state_change", 1, "count", map[string]string{
			"name": cb.name,
			"from": oldState.String(),
			"to":   newState.String(),
		})
	}
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	return State(atomic.LoadInt32(&cb.state))
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	return CircuitBreakerStats{
		Name:          cb.name,
		State:         cb.GetState().String(),
		Failures:      atomic.LoadInt32(&cb.failures),
		TotalRequests: atomic.LoadInt64(&cb.totalRequests),
		TotalFailures: atomic.LoadInt64(&cb.totalFailures),
		LastFailure:   time.Unix(0, atomic.LoadInt64(&cb.lastFailureTime)),
	}
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	atomic.StoreInt32(&cb.state, int32(StateClosed))
	atomic.StoreInt32(&cb.failures, 0)
	atomic.StoreInt32(&cb.successCount, 0)
	atomic.StoreInt64(&cb.lastFailureTime, 0)

	cb.logger.Info("Circuit breaker reset", map[string]interface{}{
		"name": cb.name,
	})
}

// CircuitBreakerStats holds circuit breaker statistics
type CircuitBreakerStats struct {
	Name          string    `json:"name"`
	State         string    `json:"state"`
	Failures      int32     `json:"failures"`
	TotalRequests int64     `json:"total_requests"`
	TotalFailures int64     `json:"total_failures"`
	LastFailure   time.Time `json:"last_failure"`
}

// ProviderCircuitBreakers manages circuit breakers for cloud providers
type ProviderCircuitBreakers struct {
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
	logger   *logging.Logger
}

// NewProviderCircuitBreakers creates circuit breakers for all providers
func NewProviderCircuitBreakers() *ProviderCircuitBreakers {
	pcb := &ProviderCircuitBreakers{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logging.GetLogger(),
	}

	// Configure provider-specific circuit breakers
	providers := []string{"aws", "azure", "gcp", "digitalocean"}

	for _, provider := range providers {
		config := &CircuitBreakerConfig{
			Name:             fmt.Sprintf("%s-circuit-breaker", provider),
			MaxFailures:      3,
			ResetTimeout:     30 * time.Second,
			HalfOpenRequests: 2,
			OnStateChange: func(from, to State) {
				pcb.logger.Info("Provider circuit breaker state changed", map[string]interface{}{
					"provider": provider,
					"from":     from.String(),
					"to":       to.String(),
				})
			},
		}

		// Adjust per provider
		switch provider {
		case "aws":
			config.MaxFailures = 5
			config.ResetTimeout = 20 * time.Second
		case "azure":
			config.MaxFailures = 4
			config.ResetTimeout = 25 * time.Second
		case "gcp":
			config.MaxFailures = 4
			config.ResetTimeout = 20 * time.Second
		case "digitalocean":
			config.MaxFailures = 3
			config.ResetTimeout = 30 * time.Second
		}

		pcb.breakers[provider] = NewCircuitBreaker(config)
	}

	// Start monitoring goroutine
	go pcb.monitorCircuits()

	return pcb
}

// Execute executes a function with circuit breaker protection for a provider
func (pcb *ProviderCircuitBreakers) Execute(ctx context.Context, provider string, fn func(context.Context) error) error {
	pcb.mu.RLock()
	breaker, exists := pcb.breakers[provider]
	pcb.mu.RUnlock()

	if !exists {
		// No circuit breaker for this provider, execute directly
		return fn(ctx)
	}

	return breaker.ExecuteContext(ctx, fn)
}

// GetBreaker returns the circuit breaker for a provider
func (pcb *ProviderCircuitBreakers) GetBreaker(provider string) *CircuitBreaker {
	pcb.mu.RLock()
	defer pcb.mu.RUnlock()

	return pcb.breakers[provider]
}

// GetAllStats returns statistics for all circuit breakers
func (pcb *ProviderCircuitBreakers) GetAllStats() map[string]CircuitBreakerStats {
	pcb.mu.RLock()
	defer pcb.mu.RUnlock()

	stats := make(map[string]CircuitBreakerStats)
	for provider, breaker := range pcb.breakers {
		stats[provider] = breaker.GetStats()
	}

	return stats
}

// ResetAll resets all circuit breakers
func (pcb *ProviderCircuitBreakers) ResetAll() {
	pcb.mu.RLock()
	defer pcb.mu.RUnlock()

	for _, breaker := range pcb.breakers {
		breaker.Reset()
	}

	pcb.logger.Info("All circuit breakers reset")
}

// monitorCircuits monitors circuit breaker health
func (pcb *ProviderCircuitBreakers) monitorCircuits() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		stats := pcb.GetAllStats()

		openCount := 0
		halfOpenCount := 0

		for _, stat := range stats {
			switch stat.State {
			case "open":
				openCount++
			case "half-open":
				halfOpenCount++
			}

			// Log if failure rate is high
			if stat.TotalRequests > 0 {
				failureRate := float64(stat.TotalFailures) / float64(stat.TotalRequests)
				if failureRate > 0.5 {
					pcb.logger.Warn("High failure rate detected", map[string]interface{}{
						"provider":       stat.Name,
						"failure_rate":   failureRate,
						"total_requests": stat.TotalRequests,
					})
				}
			}
		}

		// Alert if multiple circuits are open
		if openCount > len(pcb.breakers)/2 {
			pcb.logger.Error("Multiple circuit breakers open", nil, map[string]interface{}{
				"open_count":     openCount,
				"total_breakers": len(pcb.breakers),
			})

			// Record critical metric
			logging.Metric("circuit_breaker.critical", float64(openCount), "count", map[string]string{
				"severity": "critical",
			})
		}
	}
}

// WithCircuitBreaker wraps a function with circuit breaker protection
func WithCircuitBreaker(cb *CircuitBreaker, fn func() error) error {
	return cb.Execute(fn)
}

// Global circuit breaker instance
var globalCircuitBreakers *ProviderCircuitBreakers
var circuitBreakerOnce sync.Once

// GetCircuitBreakers returns the global circuit breaker instance
func GetCircuitBreakers() *ProviderCircuitBreakers {
	circuitBreakerOnce.Do(func() {
		globalCircuitBreakers = NewProviderCircuitBreakers()
	})
	return globalCircuitBreakers
}
