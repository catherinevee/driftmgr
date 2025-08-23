package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name            string
	config          *CircuitBreakerConfig
	state           State
	failures        uint32
	successes       uint32
	requests        uint32
	lastFailureTime time.Time
	lastStateChange time.Time
	mu              sync.RWMutex
	metrics         *CircuitBreakerMetrics
	stateListeners  []StateChangeListener
}

// State represents circuit breaker state
type State int

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

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	MaxRequests           uint32        `yaml:"max_requests"`            // Max requests in half-open state
	Interval              time.Duration `yaml:"interval"`                // Interval for closed state
	Timeout               time.Duration `yaml:"timeout"`                 // Timeout to transition from open to half-open
	FailureThreshold      uint32        `yaml:"failure_threshold"`      // Failures to open circuit
	SuccessThreshold      uint32        `yaml:"success_threshold"`      // Successes to close circuit
	FailureRatio          float64       `yaml:"failure_ratio"`           // Failure ratio to open circuit
	ObservabilityWindow   time.Duration `yaml:"observability_window"`   // Window for metrics
	MinimumRequestCount   uint32        `yaml:"minimum_request_count"`  // Min requests before evaluation
}

// CircuitBreakerMetrics tracks circuit breaker metrics
type CircuitBreakerMetrics struct {
	TotalRequests     int64
	TotalFailures     int64
	TotalSuccesses    int64
	TotalTimeouts     int64
	ConsecutiveErrors int64
	LastFailureTime   time.Time
	StateChanges      int64
	CurrentState      string
}

// StateChangeListener listens for state changes
type StateChangeListener func(from, to State, metrics *CircuitBreakerMetrics)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = &CircuitBreakerConfig{
			MaxRequests:         10,
			Interval:            60 * time.Second,
			Timeout:             30 * time.Second,
			FailureThreshold:    5,
			SuccessThreshold:    2,
			FailureRatio:        0.5,
			MinimumRequestCount: 10,
		}
	}

	return &CircuitBreaker{
		name:            name,
		config:          config,
		state:           StateClosed,
		lastStateChange: time.Now(),
		metrics:         &CircuitBreakerMetrics{},
		stateListeners:  make([]StateChangeListener, 0),
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	return cb.ExecuteContext(context.Background(), func(ctx context.Context) (interface{}, error) {
		return fn()
	})
}

// ExecuteContext executes a function with context and circuit breaker protection
func (cb *CircuitBreaker) ExecuteContext(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	if err := cb.beforeRequest(); err != nil {
		return nil, err
	}

	// Execute with timeout if configured
	if cb.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cb.config.Timeout)
		defer cancel()
	}

	result, err := fn(ctx)
	cb.afterRequest(err)

	return result, err
}

// beforeRequest checks if request should be allowed
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state := cb.state

	switch state {
	case StateOpen:
		// Check if we should transition to half-open
		if now.Sub(cb.lastStateChange) > cb.config.Timeout {
			cb.transitionTo(StateHalfOpen)
			cb.requests = 0
			return nil
		}
		cb.metrics.TotalRequests++
		return ErrCircuitBreakerOpen

	case StateHalfOpen:
		// Allow limited requests
		if cb.requests >= cb.config.MaxRequests {
			return ErrTooManyRequests
		}
		cb.requests++
		cb.metrics.TotalRequests++
		return nil

	case StateClosed:
		// Check if we need to reset counters based on interval
		if cb.config.Interval > 0 && now.Sub(cb.lastStateChange) > cb.config.Interval {
			cb.failures = 0
			cb.successes = 0
			cb.requests = 0
		}
		cb.requests++
		cb.metrics.TotalRequests++
		return nil

	default:
		return nil
	}
}

// afterRequest updates circuit breaker state after request
func (cb *CircuitBreaker) afterRequest(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onSuccess handles successful request
func (cb *CircuitBreaker) onSuccess() {
	cb.successes++
	atomic.AddInt64(&cb.metrics.TotalSuccesses, 1)
	atomic.StoreInt64(&cb.metrics.ConsecutiveErrors, 0)

	switch cb.state {
	case StateHalfOpen:
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(StateClosed)
			cb.failures = 0
			cb.successes = 0
			cb.requests = 0
		}
	}
}

// onFailure handles failed request
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()
	atomic.AddInt64(&cb.metrics.TotalFailures, 1)
	atomic.AddInt64(&cb.metrics.ConsecutiveErrors, 1)
	cb.metrics.LastFailureTime = cb.lastFailureTime

	switch cb.state {
	case StateClosed:
		if cb.shouldOpen() {
			cb.transitionTo(StateOpen)
		}

	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	}
}

// shouldOpen determines if circuit should open
func (cb *CircuitBreaker) shouldOpen() bool {
	// Check minimum request count
	if cb.requests < cb.config.MinimumRequestCount {
		return false
	}

	// Check failure threshold
	if cb.failures >= cb.config.FailureThreshold {
		return true
	}

	// Check failure ratio
	if cb.config.FailureRatio > 0 && cb.requests > 0 {
		ratio := float64(cb.failures) / float64(cb.requests)
		return ratio >= cb.config.FailureRatio
	}

	return false
}

// transitionTo transitions to a new state
func (cb *CircuitBreaker) transitionTo(state State) {
	if cb.state == state {
		return
	}

	from := cb.state
	cb.state = state
	cb.lastStateChange = time.Now()
	atomic.AddInt64(&cb.metrics.StateChanges, 1)
	cb.metrics.CurrentState = state.String()

	// Notify listeners
	for _, listener := range cb.stateListeners {
		go listener(from, state, cb.metrics)
	}
}

// GetState returns current state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns current metrics
func (cb *CircuitBreaker) GetMetrics() *CircuitBreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return &CircuitBreakerMetrics{
		TotalRequests:     atomic.LoadInt64(&cb.metrics.TotalRequests),
		TotalFailures:     atomic.LoadInt64(&cb.metrics.TotalFailures),
		TotalSuccesses:    atomic.LoadInt64(&cb.metrics.TotalSuccesses),
		TotalTimeouts:     atomic.LoadInt64(&cb.metrics.TotalTimeouts),
		ConsecutiveErrors: atomic.LoadInt64(&cb.metrics.ConsecutiveErrors),
		LastFailureTime:   cb.metrics.LastFailureTime,
		StateChanges:      atomic.LoadInt64(&cb.metrics.StateChanges),
		CurrentState:      cb.state.String(),
	}
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.requests = 0
	cb.lastStateChange = time.Now()
}

// AddStateChangeListener adds a state change listener
func (cb *CircuitBreaker) AddStateChangeListener(listener StateChangeListener) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.stateListeners = append(cb.stateListeners, listener)
}

// Errors
var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	ErrTooManyRequests    = errors.New("too many requests in half-open state")
)

// CircuitBreakerGroup manages multiple circuit breakers
type CircuitBreakerGroup struct {
	breakers map[string]*CircuitBreaker
	config   *CircuitBreakerConfig
	mu       sync.RWMutex
}

// NewCircuitBreakerGroup creates a new circuit breaker group
func NewCircuitBreakerGroup(config *CircuitBreakerConfig) *CircuitBreakerGroup {
	return &CircuitBreakerGroup{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}
}

// Get gets or creates a circuit breaker for a key
func (cbg *CircuitBreakerGroup) Get(key string) *CircuitBreaker {
	cbg.mu.RLock()
	cb, exists := cbg.breakers[key]
	cbg.mu.RUnlock()

	if exists {
		return cb
	}

	cbg.mu.Lock()
	defer cbg.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists = cbg.breakers[key]; exists {
		return cb
	}

	cb = NewCircuitBreaker(key, cbg.config)
	cbg.breakers[key] = cb
	return cb
}

// Execute executes a function with circuit breaker for a key
func (cbg *CircuitBreakerGroup) Execute(key string, fn func() (interface{}, error)) (interface{}, error) {
	cb := cbg.Get(key)
	return cb.Execute(fn)
}

// GetAll returns all circuit breakers
func (cbg *CircuitBreakerGroup) GetAll() map[string]*CircuitBreaker {
	cbg.mu.RLock()
	defer cbg.mu.RUnlock()

	result := make(map[string]*CircuitBreaker, len(cbg.breakers))
	for k, v := range cbg.breakers {
		result[k] = v
	}
	return result
}

// GetMetrics returns metrics for all circuit breakers
func (cbg *CircuitBreakerGroup) GetMetrics() map[string]*CircuitBreakerMetrics {
	cbg.mu.RLock()
	defer cbg.mu.RUnlock()

	result := make(map[string]*CircuitBreakerMetrics, len(cbg.breakers))
	for k, v := range cbg.breakers {
		result[k] = v.GetMetrics()
	}
	return result
}

// Reset resets a specific circuit breaker
func (cbg *CircuitBreakerGroup) Reset(key string) {
	cbg.mu.RLock()
	cb, exists := cbg.breakers[key]
	cbg.mu.RUnlock()

	if exists {
		cb.Reset()
	}
}

// ResetAll resets all circuit breakers
func (cbg *CircuitBreakerGroup) ResetAll() {
	cbg.mu.RLock()
	defer cbg.mu.RUnlock()

	for _, cb := range cbg.breakers {
		cb.Reset()
	}
}

// HystrixCircuitBreaker implements Netflix Hystrix-style circuit breaker
type HystrixCircuitBreaker struct {
	*CircuitBreaker
	commandKey      string
	commandGroup    string
	threadPool      *ThreadPool
	fallbackFunc    func() (interface{}, error)
	metricsCollector *MetricsCollector
}

// ThreadPool manages thread pool for Hystrix commands
type ThreadPool struct {
	size       int
	queueSize  int
	semaphore  chan struct{}
}

// MetricsCollector collects Hystrix metrics
type MetricsCollector struct {
	successCount     int64
	errorCount       int64
	timeoutCount     int64
	rejectedCount    int64
	shortCircuitCount int64
	totalDuration    int64
	mu               sync.RWMutex
}

// NewHystrixCircuitBreaker creates a new Hystrix-style circuit breaker
func NewHystrixCircuitBreaker(commandKey, commandGroup string, config *CircuitBreakerConfig) *HystrixCircuitBreaker {
	return &HystrixCircuitBreaker{
		CircuitBreaker:   NewCircuitBreaker(commandKey, config),
		commandKey:       commandKey,
		commandGroup:     commandGroup,
		threadPool:       &ThreadPool{size: 10, queueSize: 100},
		metricsCollector: &MetricsCollector{},
	}
}

// Execute executes command with Hystrix circuit breaker
func (hcb *HystrixCircuitBreaker) Execute(fn func() (interface{}, error)) (interface{}, error) {
	// Try to acquire semaphore
	select {
	case hcb.threadPool.semaphore <- struct{}{}:
		defer func() { <-hcb.threadPool.semaphore }()
	default:
		atomic.AddInt64(&hcb.metricsCollector.rejectedCount, 1)
		if hcb.fallbackFunc != nil {
			return hcb.fallbackFunc()
		}
		return nil, errors.New("thread pool rejected")
	}

	// Execute with circuit breaker
	result, err := hcb.CircuitBreaker.Execute(fn)
	
	// Update metrics
	if err != nil {
		if err == context.DeadlineExceeded {
			atomic.AddInt64(&hcb.metricsCollector.timeoutCount, 1)
		} else if err == ErrCircuitBreakerOpen {
			atomic.AddInt64(&hcb.metricsCollector.shortCircuitCount, 1)
		} else {
			atomic.AddInt64(&hcb.metricsCollector.errorCount, 1)
		}
		
		// Try fallback
		if hcb.fallbackFunc != nil {
			return hcb.fallbackFunc()
		}
	} else {
		atomic.AddInt64(&hcb.metricsCollector.successCount, 1)
	}

	return result, err
}

// SetFallback sets fallback function
func (hcb *HystrixCircuitBreaker) SetFallback(fn func() (interface{}, error)) {
	hcb.fallbackFunc = fn
}

// GetHystrixMetrics returns Hystrix-specific metrics
func (hcb *HystrixCircuitBreaker) GetHystrixMetrics() map[string]int64 {
	return map[string]int64{
		"success_count":       atomic.LoadInt64(&hcb.metricsCollector.successCount),
		"error_count":         atomic.LoadInt64(&hcb.metricsCollector.errorCount),
		"timeout_count":       atomic.LoadInt64(&hcb.metricsCollector.timeoutCount),
		"rejected_count":      atomic.LoadInt64(&hcb.metricsCollector.rejectedCount),
		"short_circuit_count": atomic.LoadInt64(&hcb.metricsCollector.shortCircuitCount),
	}
}

// AdaptiveCircuitBreaker adjusts thresholds based on system behavior
type AdaptiveCircuitBreaker struct {
	*CircuitBreaker
	learningRate float64
	history      []float64
	windowSize   int
}

// NewAdaptiveCircuitBreaker creates a new adaptive circuit breaker
func NewAdaptiveCircuitBreaker(name string, config *CircuitBreakerConfig) *AdaptiveCircuitBreaker {
	return &AdaptiveCircuitBreaker{
		CircuitBreaker: NewCircuitBreaker(name, config),
		learningRate:   0.1,
		history:        make([]float64, 0),
		windowSize:     100,
	}
}

// Adapt adjusts thresholds based on historical performance
func (acb *AdaptiveCircuitBreaker) Adapt() {
	acb.mu.Lock()
	defer acb.mu.Unlock()

	if len(acb.history) < acb.windowSize {
		return
	}

	// Calculate average failure rate
	var sum float64
	for _, rate := range acb.history {
		sum += rate
	}
	avgFailureRate := sum / float64(len(acb.history))

	// Adjust failure threshold
	if avgFailureRate > acb.config.FailureRatio {
		// System is struggling, be more lenient
		acb.config.FailureThreshold = uint32(float64(acb.config.FailureThreshold) * (1 + acb.learningRate))
		acb.config.FailureRatio = min(0.9, acb.config.FailureRatio+0.05)
	} else if avgFailureRate < acb.config.FailureRatio*0.5 {
		// System is healthy, be more strict
		acb.config.FailureThreshold = uint32(float64(acb.config.FailureThreshold) * (1 - acb.learningRate))
		acb.config.FailureRatio = max(0.1, acb.config.FailureRatio-0.05)
	}

	// Reset history
	acb.history = acb.history[acb.windowSize/2:]
}

// recordFailureRate records current failure rate
func (acb *AdaptiveCircuitBreaker) recordFailureRate() {
	if acb.requests > 0 {
		rate := float64(acb.failures) / float64(acb.requests)
		acb.history = append(acb.history, rate)
		
		if len(acb.history) >= acb.windowSize {
			acb.Adapt()
		}
	}
}

// CircuitBreakerMiddleware creates middleware for HTTP handlers
func CircuitBreakerMiddleware(cb *CircuitBreaker) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			_, err := cb.Execute(func() (interface{}, error) {
				next(w, r)
				return nil, nil
			})

			if err != nil {
				if err == ErrCircuitBreakerOpen {
					http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
				} else {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
			}
		}
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}