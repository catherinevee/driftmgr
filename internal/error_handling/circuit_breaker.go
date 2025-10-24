package error_handling

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState string

const (
	CircuitStateClosed   CircuitState = "closed"
	CircuitStateOpen     CircuitState = "open"
	CircuitStateHalfOpen CircuitState = "half_open"
)

// CircuitBreaker provides circuit breaker functionality for error handling
type CircuitBreaker struct {
	name          string
	threshold     int
	timeout       time.Duration
	state         CircuitState
	failureCount  int
	lastFailTime  time.Time
	mutex         sync.RWMutex
	onStateChange func(name string, from, to CircuitState)
}

// CircuitBreakerConfig represents configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name              string        `json:"name"`
	FailureThreshold  int           `json:"failure_threshold"`
	Timeout           time.Duration `json:"timeout"`
	OnStateChange     func(name string, from, to CircuitState) `json:"-"`
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:          config.Name,
		threshold:     config.FailureThreshold,
		timeout:       config.Timeout,
		state:         CircuitStateClosed,
		onStateChange: config.OnStateChange,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Check if circuit is open
	if cb.state == CircuitStateOpen {
		if time.Since(cb.lastFailTime) >= cb.timeout {
			// Timeout reached, transition to half-open
			cb.setState(CircuitStateHalfOpen)
		} else {
			// Circuit is still open, return error
			return fmt.Errorf("circuit breaker %s is open", cb.name)
		}
	}

	// Execute the operation
	err := operation()
	
	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetFailureCount returns the current failure count
func (cb *CircuitBreaker) GetFailureCount() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.failureCount
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.setState(CircuitStateClosed)
	cb.failureCount = 0
}

// IsAvailable returns true if the circuit breaker allows operations
func (cb *CircuitBreaker) IsAvailable() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	if cb.state == CircuitStateClosed {
		return true
	}
	
	if cb.state == CircuitStateHalfOpen {
		return true
	}
	
	if cb.state == CircuitStateOpen {
		return time.Since(cb.lastFailTime) >= cb.timeout
	}
	
	return false
}

// Helper methods

func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailTime = time.Now()
	
	if cb.failureCount >= cb.threshold {
		cb.setState(CircuitStateOpen)
	}
}

func (cb *CircuitBreaker) onSuccess() {
	if cb.state == CircuitStateHalfOpen {
		cb.setState(CircuitStateClosed)
	}
	cb.failureCount = 0
}

func (cb *CircuitBreaker) setState(newState CircuitState) {
	if cb.state != newState {
		oldState := cb.state
		cb.state = newState
		
		if cb.onStateChange != nil {
			cb.onStateChange(cb.name, oldState, newState)
		}
	}
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config CircuitBreakerConfig) *CircuitBreaker {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()
	
	if breaker, exists := cbm.breakers[name]; exists {
		return breaker
	}
	
	config.Name = name
	breaker := NewCircuitBreaker(config)
	cbm.breakers[name] = breaker
	return breaker
}

// Get returns an existing circuit breaker
func (cbm *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	breaker, exists := cbm.breakers[name]
	return breaker, exists
}

// GetAll returns all circuit breakers
func (cbm *CircuitBreakerManager) GetAll() map[string]*CircuitBreaker {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	result := make(map[string]*CircuitBreaker)
	for name, breaker := range cbm.breakers {
		result[name] = breaker
	}
	return result
}

// Reset resets a specific circuit breaker
func (cbm *CircuitBreakerManager) Reset(name string) {
	cbm.mutex.RLock()
	breaker, exists := cbm.breakers[name]
	cbm.mutex.RUnlock()
	
	if exists {
		breaker.Reset()
	}
}

// ResetAll resets all circuit breakers
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	for _, breaker := range cbm.breakers {
		breaker.Reset()
	}
}

// GetStatistics returns statistics about all circuit breakers
func (cbm *CircuitBreakerManager) GetStatistics() map[string]interface{} {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	
	for name, breaker := range cbm.breakers {
		breakerStats := map[string]interface{}{
			"state":         string(breaker.GetState()),
			"failure_count": breaker.GetFailureCount(),
			"available":     breaker.IsAvailable(),
		}
		stats[name] = breakerStats
	}
	
	return stats
}
