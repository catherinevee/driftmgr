package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// AdaptiveConcurrencyManager manages dynamic concurrency levels based on system load
type AdaptiveConcurrencyManager struct {
	mu                 sync.RWMutex
	maxConcurrency     int32
	currentConcurrency int32
	activeTasks        int32
	resourcePool       *ResourcePool
	backpressure       *BackpressureHandler

	// Configuration
	minConcurrency    int32
	maxConcurrencyLim int32
	adjustmentFactor  float64
	monitorInterval   time.Duration

	// System metrics
	cpuThreshold   float64
	memThreshold   float64
	lastAdjustment time.Time

	// Statistics
	totalTasks      int64
	completedTasks  int64
	failedTasks     int64
	avgResponseTime time.Duration

	// Metrics
	metrics *ConcurrencyMetrics

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

// ConcurrencyMetrics holds Prometheus metrics for concurrency management
type ConcurrencyMetrics struct {
	activeTasks        prometheus.Gauge
	completedTasks     prometheus.Counter
	failedTasks        prometheus.Counter
	concurrencyLevel   prometheus.Gauge
	responseTime       prometheus.Histogram
	resourcePoolSize   prometheus.Gauge
	backpressureEvents prometheus.Counter
}

// ResourcePool manages a pool of reusable resources
type ResourcePool struct {
	mu        sync.RWMutex
	resources chan Resource
	factory   ResourceFactory
	destroyer ResourceDestroyer
	maxSize   int32
	current   int32
	metrics   *ResourcePoolMetrics
}

// Resource represents a pooled resource
type Resource interface {
	Close() error
	IsValid() bool
	Reset() error
}

// ResourceFactory creates new resources
type ResourceFactory func() (Resource, error)

// ResourceDestroyer cleans up resources
type ResourceDestroyer func(Resource) error

// ResourcePoolMetrics holds metrics for resource pool
type ResourcePoolMetrics struct {
	poolSize    prometheus.Gauge
	hitRate     prometheus.Counter
	missRate    prometheus.Counter
	createTime  prometheus.Histogram
	destroyTime prometheus.Histogram
}

// BackpressureHandler manages system backpressure
type BackpressureHandler struct {
	mu              sync.RWMutex
	enabled         bool
	threshold       float64
	currentPressure float64
	windowSize      time.Duration
	samples         []float64
	lastSample      time.Time
	metrics         *BackpressureMetrics
}

// BackpressureMetrics holds metrics for backpressure handling
type BackpressureMetrics struct {
	pressureLevel prometheus.Gauge
	rejectedTasks prometheus.Counter
	delayedTasks  prometheus.Counter
	adaptations   prometheus.Counter
}

// Task represents a unit of work to be executed
type Task interface {
	Execute(ctx context.Context) error
	Priority() int
	EstimatedDuration() time.Duration
}

// ConcurrencyConfig holds configuration for adaptive concurrency
type ConcurrencyConfig struct {
	MinConcurrency   int32
	MaxConcurrency   int32
	CPUThreshold     float64
	MemoryThreshold  float64
	AdjustmentFactor float64
	MonitorInterval  time.Duration
}

// NewAdaptiveConcurrencyManager creates a new adaptive concurrency manager
func NewAdaptiveConcurrencyManager(config ConcurrencyConfig) *AdaptiveConcurrencyManager {
	ctx, cancel := context.WithCancel(context.Background())

	metrics := &ConcurrencyMetrics{
		activeTasks: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "driftmgr_active_tasks_total",
			Help: "Number of currently active tasks",
		}),
		completedTasks: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_completed_tasks_total",
			Help: "Total number of completed tasks",
		}),
		failedTasks: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_failed_tasks_total",
			Help: "Total number of failed tasks",
		}),
		concurrencyLevel: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "driftmgr_concurrency_level",
			Help: "Current concurrency level",
		}),
		responseTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_task_duration_seconds",
			Help:    "Task execution duration",
			Buckets: prometheus.DefBuckets,
		}),
		resourcePoolSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "driftmgr_resource_pool_size",
			Help: "Current resource pool size",
		}),
		backpressureEvents: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_backpressure_events_total",
			Help: "Total number of backpressure events",
		}),
	}

	poolMetrics := &ResourcePoolMetrics{
		poolSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "driftmgr_pool_size",
			Help: "Current pool size",
		}),
		hitRate: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_pool_hits_total",
			Help: "Total pool hits",
		}),
		missRate: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_pool_misses_total",
			Help: "Total pool misses",
		}),
		createTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "driftmgr_resource_create_duration_seconds",
			Help: "Resource creation duration",
		}),
		destroyTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "driftmgr_resource_destroy_duration_seconds",
			Help: "Resource destruction duration",
		}),
	}

	backpressureMetrics := &BackpressureMetrics{
		pressureLevel: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "driftmgr_backpressure_level",
			Help: "Current backpressure level",
		}),
		rejectedTasks: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_rejected_tasks_total",
			Help: "Total number of rejected tasks due to backpressure",
		}),
		delayedTasks: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_delayed_tasks_total",
			Help: "Total number of delayed tasks due to backpressure",
		}),
		adaptations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_concurrency_adaptations_total",
			Help: "Total number of concurrency adaptations",
		}),
	}

	manager := &AdaptiveConcurrencyManager{
		maxConcurrency:     config.MinConcurrency,
		currentConcurrency: config.MinConcurrency,
		minConcurrency:     config.MinConcurrency,
		maxConcurrencyLim:  config.MaxConcurrency,
		adjustmentFactor:   config.AdjustmentFactor,
		monitorInterval:    config.MonitorInterval,
		cpuThreshold:       config.CPUThreshold,
		memThreshold:       config.MemoryThreshold,
		metrics:            metrics,
		ctx:                ctx,
		cancel:             cancel,
		done:               make(chan struct{}),
	}

	// Initialize resource pool
	manager.resourcePool = &ResourcePool{
		resources: make(chan Resource, config.MaxConcurrency),
		maxSize:   config.MaxConcurrency,
		metrics:   poolMetrics,
	}

	// Initialize backpressure handler
	manager.backpressure = &BackpressureHandler{
		enabled:    true,
		threshold:  0.8,
		windowSize: time.Minute,
		samples:    make([]float64, 0, 60),
		metrics:    backpressureMetrics,
	}

	// Start monitoring goroutine
	go manager.monitor()

	return manager
}

// Execute executes a task with adaptive concurrency control
func (acm *AdaptiveConcurrencyManager) Execute(ctx context.Context, task Task) error {
	// Check backpressure
	if acm.backpressure.ShouldReject() {
		acm.backpressure.metrics.rejectedTasks.Inc()
		return ErrBackpressureRejection
	}

	// Acquire concurrency slot
	if !acm.acquireSlot(ctx) {
		return ErrConcurrencyLimitReached
	}
	defer acm.releaseSlot()

	// Get resource from pool
	resource, err := acm.resourcePool.Get(ctx)
	if err != nil {
		return err
	}
	defer acm.resourcePool.Put(resource)

	// Execute task
	start := time.Now()
	atomic.AddInt64(&acm.totalTasks, 1)
	acm.metrics.activeTasks.Inc()

	err = task.Execute(ctx)

	duration := time.Since(start)
	acm.metrics.responseTime.Observe(duration.Seconds())
	acm.metrics.activeTasks.Dec()

	if err != nil {
		atomic.AddInt64(&acm.failedTasks, 1)
		acm.metrics.failedTasks.Inc()
		return err
	}

	atomic.AddInt64(&acm.completedTasks, 1)
	acm.metrics.completedTasks.Inc()

	// Update average response time
	acm.updateAvgResponseTime(duration)

	return nil
}

// acquireSlot attempts to acquire a concurrency slot
func (acm *AdaptiveConcurrencyManager) acquireSlot(ctx context.Context) bool {
	for {
		current := atomic.LoadInt32(&acm.activeTasks)
		max := atomic.LoadInt32(&acm.maxConcurrency)

		if current >= max {
			// Check if we should wait or reject
			select {
			case <-ctx.Done():
				return false
			case <-time.After(time.Millisecond * 10):
				continue
			}
		}

		if atomic.CompareAndSwapInt32(&acm.activeTasks, current, current+1) {
			return true
		}
	}
}

// releaseSlot releases a concurrency slot
func (acm *AdaptiveConcurrencyManager) releaseSlot() {
	atomic.AddInt32(&acm.activeTasks, -1)
}

// monitor continuously monitors system metrics and adjusts concurrency
func (acm *AdaptiveConcurrencyManager) monitor() {
	ticker := time.NewTicker(acm.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-acm.ctx.Done():
			close(acm.done)
			return
		case <-ticker.C:
			acm.adjustConcurrency()
			acm.updateBackpressure()
		}
	}
}

// adjustConcurrency adjusts the concurrency level based on system metrics
func (acm *AdaptiveConcurrencyManager) adjustConcurrency() {
	cpuUsage := getCPUUsage()
	memUsage := getMemoryUsage()

	current := atomic.LoadInt32(&acm.maxConcurrency)
	var newConcurrency int32

	// Adjust based on CPU and memory usage
	if cpuUsage > acm.cpuThreshold || memUsage > acm.memThreshold {
		// Decrease concurrency
		newConcurrency = int32(float64(current) * (1.0 - acm.adjustmentFactor))
	} else {
		// Increase concurrency
		newConcurrency = int32(float64(current) * (1.0 + acm.adjustmentFactor))
	}

	// Apply bounds
	if newConcurrency < acm.minConcurrency {
		newConcurrency = acm.minConcurrency
	}
	if newConcurrency > acm.maxConcurrencyLim {
		newConcurrency = acm.maxConcurrencyLim
	}

	// Update if changed
	if newConcurrency != current {
		atomic.StoreInt32(&acm.maxConcurrency, newConcurrency)
		acm.metrics.concurrencyLevel.Set(float64(newConcurrency))
		acm.lastAdjustment = time.Now()
	}
}

// updateBackpressure updates backpressure measurements
func (acm *AdaptiveConcurrencyManager) updateBackpressure() {
	acm.backpressure.mu.Lock()
	defer acm.backpressure.mu.Unlock()

	now := time.Now()
	if now.Sub(acm.backpressure.lastSample) < time.Second {
		return
	}

	// Calculate current pressure based on active tasks vs capacity
	pressure := float64(atomic.LoadInt32(&acm.activeTasks)) / float64(atomic.LoadInt32(&acm.maxConcurrency))

	// Add to sliding window
	acm.backpressure.samples = append(acm.backpressure.samples, pressure)
	if len(acm.backpressure.samples) > 60 { // Keep 60 seconds of samples
		acm.backpressure.samples = acm.backpressure.samples[1:]
	}

	// Calculate average pressure
	var sum float64
	for _, sample := range acm.backpressure.samples {
		sum += sample
	}
	acm.backpressure.currentPressure = sum / float64(len(acm.backpressure.samples))

	acm.backpressure.metrics.pressureLevel.Set(acm.backpressure.currentPressure)
	acm.backpressure.lastSample = now
}

// updateAvgResponseTime updates the rolling average response time
func (acm *AdaptiveConcurrencyManager) updateAvgResponseTime(duration time.Duration) {
	acm.mu.Lock()
	defer acm.mu.Unlock()

	// Simple exponential moving average
	alpha := 0.1
	if acm.avgResponseTime == 0 {
		acm.avgResponseTime = duration
	} else {
		acm.avgResponseTime = time.Duration(float64(acm.avgResponseTime)*(1-alpha) + float64(duration)*alpha)
	}
}

// ShouldReject determines if a request should be rejected due to backpressure
func (bp *BackpressureHandler) ShouldReject() bool {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	if !bp.enabled {
		return false
	}

	return bp.currentPressure > bp.threshold
}

// Get retrieves a resource from the pool
func (rp *ResourcePool) Get(ctx context.Context) (Resource, error) {
	select {
	case resource := <-rp.resources:
		if resource.IsValid() {
			rp.metrics.hitRate.Inc()
			return resource, nil
		}
		// Resource is invalid, destroy it
		if rp.destroyer != nil {
			rp.destroyer(resource)
		}
		// Create new resource since cached one was invalid
		rp.metrics.missRate.Inc()
		if rp.factory == nil {
			return nil, ErrNoResourceFactory
		}

		start := time.Now()
		newResource, err := rp.factory()
		rp.metrics.createTime.Observe(time.Since(start).Seconds())

		return newResource, err
	default:
		rp.metrics.missRate.Inc()
		// Create new resource
		if rp.factory == nil {
			return nil, ErrNoResourceFactory
		}

		start := time.Now()
		resource, err := rp.factory()
		rp.metrics.createTime.Observe(time.Since(start).Seconds())

		return resource, err
	}
}

// Put returns a resource to the pool
func (rp *ResourcePool) Put(resource Resource) {
	if resource == nil || !resource.IsValid() {
		return
	}

	// Reset resource state
	if err := resource.Reset(); err != nil {
		// Failed to reset, destroy the resource
		if rp.destroyer != nil {
			start := time.Now()
			rp.destroyer(resource)
			rp.metrics.destroyTime.Observe(time.Since(start).Seconds())
		}
		return
	}

	select {
	case rp.resources <- resource:
		// Successfully returned to pool
	default:
		// Pool is full, destroy the resource
		if rp.destroyer != nil {
			start := time.Now()
			rp.destroyer(resource)
			rp.metrics.destroyTime.Observe(time.Since(start).Seconds())
		}
	}

	rp.metrics.poolSize.Set(float64(len(rp.resources)))
}

// SetResourceFactory sets the resource factory function
func (rp *ResourcePool) SetResourceFactory(factory ResourceFactory) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.factory = factory
}

// SetResourceDestroyer sets the resource destroyer function
func (rp *ResourcePool) SetResourceDestroyer(destroyer ResourceDestroyer) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.destroyer = destroyer
}

// GetStats returns current statistics
func (acm *AdaptiveConcurrencyManager) GetStats() ConcurrencyStats {
	acm.mu.RLock()
	defer acm.mu.RUnlock()

	return ConcurrencyStats{
		MaxConcurrency:    atomic.LoadInt32(&acm.maxConcurrency),
		ActiveTasks:       atomic.LoadInt32(&acm.activeTasks),
		TotalTasks:        atomic.LoadInt64(&acm.totalTasks),
		CompletedTasks:    atomic.LoadInt64(&acm.completedTasks),
		FailedTasks:       atomic.LoadInt64(&acm.failedTasks),
		AvgResponseTime:   acm.avgResponseTime,
		BackpressureLevel: acm.backpressure.currentPressure,
		ResourcePoolSize:  int32(len(acm.resourcePool.resources)),
		LastAdjustment:    acm.lastAdjustment,
	}
}

// ConcurrencyStats holds statistics about concurrency management
type ConcurrencyStats struct {
	MaxConcurrency    int32
	ActiveTasks       int32
	TotalTasks        int64
	CompletedTasks    int64
	FailedTasks       int64
	AvgResponseTime   time.Duration
	BackpressureLevel float64
	ResourcePoolSize  int32
	LastAdjustment    time.Time
}

// Shutdown gracefully shuts down the concurrency manager
func (acm *AdaptiveConcurrencyManager) Shutdown(ctx context.Context) error {
	acm.cancel()

	select {
	case <-acm.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// getCPUUsage returns current CPU usage percentage
func getCPUUsage() float64 {
	// Simplified CPU usage calculation
	// In production, use proper CPU monitoring
	return float64(runtime.NumGoroutine()) / float64(runtime.NumCPU()) * 10.0
}

// getMemoryUsage returns current memory usage percentage
func getMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Simplified memory usage calculation
	return float64(m.Alloc) / float64(m.Sys) * 100.0
}

// Error definitions
var (
	ErrBackpressureRejection   = fmt.Errorf("request rejected due to backpressure")
	ErrConcurrencyLimitReached = fmt.Errorf("concurrency limit reached")
	ErrNoResourceFactory       = fmt.Errorf("no resource factory configured")
)
