package performance

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// AdaptiveConcurrency provides adaptive concurrency control
type AdaptiveConcurrency struct {
	loadMonitor     *LoadMonitor
	concurrencyManager *ConcurrencyManager
	config          *AdaptiveConfig
	mu              sync.RWMutex
}

// AdaptiveConfig defines adaptive concurrency behavior
type AdaptiveConfig struct {
	Enabled           bool          `yaml:"enabled"`
	MinConcurrency    int           `yaml:"min_concurrency"`
	MaxConcurrency    int           `yaml:"max_concurrency"`
	TargetCPUPercent  float64       `yaml:"target_cpu_percent"`
	TargetMemoryPercent float64     `yaml:"target_memory_percent"`
	AdjustmentInterval time.Duration `yaml:"adjustment_interval"`
	StabilizationPeriod time.Duration `yaml:"stabilization_period"`
	LoadThreshold     float64       `yaml:"load_threshold"`
}

// LoadMetrics represents system load metrics
type LoadMetrics struct {
	Timestamp       time.Time `json:"timestamp"`
	CPUPercent      float64   `json:"cpu_percent"`
	MemoryPercent   float64   `json:"memory_percent"`
	GoroutineCount  int       `json:"goroutine_count"`
	LoadAverage     float64   `json:"load_average"`
	ResponseTime    time.Duration `json:"response_time"`
	Throughput      float64   `json:"throughput"`
}

// ConcurrencyState represents current concurrency state
type ConcurrencyState struct {
	CurrentConcurrency int           `json:"current_concurrency"`
	TargetConcurrency  int           `json:"target_concurrency"`
	LastAdjustment    time.Time      `json:"last_adjustment"`
	AdjustmentReason  string         `json:"adjustment_reason"`
	LoadMetrics       *LoadMetrics   `json:"load_metrics"`
	StabilityScore    float64        `json:"stability_score"`
}

// NewAdaptiveConcurrency creates a new adaptive concurrency controller
func NewAdaptiveConcurrency(config *AdaptiveConfig) *AdaptiveConcurrency {
	if config == nil {
		config = &AdaptiveConfig{
			Enabled:           true,
			MinConcurrency:    1,
			MaxConcurrency:    runtime.NumCPU() * 4,
			TargetCPUPercent:  70.0,
			TargetMemoryPercent: 80.0,
			AdjustmentInterval: 30 * time.Second,
			StabilizationPeriod: 2 * time.Minute,
			LoadThreshold:     0.8,
		}
	}

	ac := &AdaptiveConcurrency{
		loadMonitor:       NewLoadMonitor(),
		concurrencyManager: NewConcurrencyManager(config),
		config:            config,
	}

	// Start monitoring and adjustment loop
	go ac.adjustmentLoop()

	return ac
}

// GetOptimalConcurrency returns the optimal concurrency level
func (ac *AdaptiveConcurrency) GetOptimalConcurrency() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	return ac.concurrencyManager.GetCurrentConcurrency()
}

// AdjustConcurrency manually adjusts concurrency
func (ac *AdaptiveConcurrency) AdjustConcurrency(delta int, reason string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.concurrencyManager.AdjustConcurrency(delta, reason)
}

// GetLoadMetrics returns current load metrics
func (ac *AdaptiveConcurrency) GetLoadMetrics() *LoadMetrics {
	return ac.loadMonitor.GetCurrentMetrics()
}

// GetConcurrencyState returns current concurrency state
func (ac *AdaptiveConcurrency) GetConcurrencyState() *ConcurrencyState {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	return ac.concurrencyManager.GetState()
}

// adjustmentLoop continuously monitors and adjusts concurrency
func (ac *AdaptiveConcurrency) adjustmentLoop() {
	ticker := time.NewTicker(ac.config.AdjustmentInterval)
	defer ticker.Stop()

	for range ticker.C {
		ac.performAdjustment()
	}
}

// performAdjustment performs automatic concurrency adjustment
func (ac *AdaptiveConcurrency) performAdjustment() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	metrics := ac.loadMonitor.GetCurrentMetrics()
	state := ac.concurrencyManager.GetState()

	// Calculate adjustment based on load metrics
	adjustment := ac.calculateAdjustment(metrics, state)
	
	if adjustment != 0 {
		reason := ac.determineAdjustmentReason(metrics, state)
		ac.concurrencyManager.AdjustConcurrency(adjustment, reason)
		
		fmt.Printf("Adaptive concurrency adjustment: %+d (reason: %s, CPU: %.1f%%, Memory: %.1f%%)\n",
			adjustment, reason, metrics.CPUPercent, metrics.MemoryPercent)
	}
}

// calculateAdjustment calculates the required concurrency adjustment
func (ac *AdaptiveConcurrency) calculateAdjustment(metrics *LoadMetrics, state *ConcurrencyState) int {
	// Calculate load scores
	cpuScore := metrics.CPUPercent / ac.config.TargetCPUPercent
	memoryScore := metrics.MemoryPercent / ac.config.TargetMemoryPercent
	
	// Use the higher score to determine adjustment
	loadScore := math.Max(cpuScore, memoryScore)
	
	// Calculate stability score
	stabilityScore := ac.calculateStabilityScore(metrics, state)
	
	// Determine adjustment based on load and stability
	if loadScore > 1.2 && stabilityScore > 0.7 {
		// High load, stable system - decrease concurrency
		return -1
	} else if loadScore < 0.8 && stabilityScore > 0.7 {
		// Low load, stable system - increase concurrency
		return 1
	} else if loadScore > 1.5 {
		// Very high load - significant decrease
		return -2
	} else if loadScore < 0.5 {
		// Very low load - significant increase
		return 2
	}
	
	return 0
}

// calculateStabilityScore calculates system stability score
func (ac *AdaptiveConcurrency) calculateStabilityScore(metrics *LoadMetrics, state *ConcurrencyState) float64 {
	// Factors for stability:
	// - Response time consistency
	// - Throughput stability
	// - Resource usage consistency
	
	// For now, use a simplified stability calculation
	// In a real implementation, this would analyze historical metrics
	
	// Check if metrics are within reasonable bounds
	cpuStable := metrics.CPUPercent > 0 && metrics.CPUPercent < 95
	memoryStable := metrics.MemoryPercent > 0 && metrics.MemoryPercent < 95
	responseStable := metrics.ResponseTime > 0 && metrics.ResponseTime < 10*time.Second
	
	stabilityFactors := 0
	if cpuStable {
		stabilityFactors++
	}
	if memoryStable {
		stabilityFactors++
	}
	if responseStable {
		stabilityFactors++
	}
	
	return float64(stabilityFactors) / 3.0
}

// determineAdjustmentReason determines the reason for adjustment
func (ac *AdaptiveConcurrency) determineAdjustmentReason(metrics *LoadMetrics, state *ConcurrencyState) string {
	cpuScore := metrics.CPUPercent / ac.config.TargetCPUPercent
	memoryScore := metrics.MemoryPercent / ac.config.TargetMemoryPercent
	
	if cpuScore > 1.2 {
		return fmt.Sprintf("High CPU usage (%.1f%%)", metrics.CPUPercent)
	} else if memoryScore > 1.2 {
		return fmt.Sprintf("High memory usage (%.1f%%)", metrics.MemoryPercent)
	} else if cpuScore < 0.8 && memoryScore < 0.8 {
		return "Low resource utilization"
	} else if metrics.ResponseTime > 5*time.Second {
		return "High response time"
	}
	
	return "Load balancing"
}

// LoadMonitor monitors system load
type LoadMonitor struct {
	metrics     *LoadMetrics
	mu          sync.RWMutex
	lastUpdate  time.Time
}

// NewLoadMonitor creates a new load monitor
func NewLoadMonitor() *LoadMonitor {
	lm := &LoadMonitor{
		metrics: &LoadMetrics{
			Timestamp: time.Now(),
		},
	}
	
	// Start monitoring
	go lm.monitoringLoop()
	
	return lm
}

// GetCurrentMetrics returns current load metrics
func (lm *LoadMonitor) GetCurrentMetrics() *LoadMetrics {
	lm.mu.RLock()
	defer lm.mu.RUnlock()
	
	return lm.metrics
}

// monitoringLoop continuously monitors system load
func (lm *LoadMonitor) monitoringLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		lm.updateMetrics()
	}
}

// updateMetrics updates current load metrics
func (lm *LoadMonitor) updateMetrics() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	
	metrics := &LoadMetrics{
		Timestamp:      time.Now(),
		CPUPercent:     lm.getCPUPercent(),
		MemoryPercent:  lm.getMemoryPercent(),
		GoroutineCount: runtime.NumGoroutine(),
		LoadAverage:    lm.getLoadAverage(),
		ResponseTime:   lm.getAverageResponseTime(),
		Throughput:     lm.getThroughput(),
	}
	
	lm.metrics = metrics
	lm.lastUpdate = time.Now()
}

// getCPUPercent gets current CPU usage percentage
func (lm *LoadMonitor) getCPUPercent() float64 {
	// In a real implementation, this would use system calls
	// For now, return a simulated value based on goroutine count
	goroutines := runtime.NumGoroutine()
	cpuCores := runtime.NumCPU()
	
	// Simulate CPU usage based on active goroutines
	cpuPercent := float64(goroutines) / float64(cpuCores*10) * 100
	return math.Min(cpuPercent, 100.0)
}

// getMemoryPercent gets current memory usage percentage
func (lm *LoadMonitor) getMemoryPercent() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Calculate memory usage percentage
	totalMemory := m.Sys
	usedMemory := m.Alloc + m.StackInuse + m.HeapInuse
	
	if totalMemory == 0 {
		return 0.0
	}
	
	return float64(usedMemory) / float64(totalMemory) * 100
}

// getLoadAverage gets system load average
func (lm *LoadMonitor) getLoadAverage() float64 {
	// In a real implementation, this would read from /proc/loadavg
	// For now, simulate based on goroutine count
	goroutines := runtime.NumGoroutine()
	cpuCores := runtime.NumCPU()
	
	return float64(goroutines) / float64(cpuCores)
}

// getAverageResponseTime gets average response time
func (lm *LoadMonitor) getAverageResponseTime() time.Duration {
	// In a real implementation, this would track actual response times
	// For now, simulate based on load
	load := lm.getLoadAverage()
	
	// Simulate response time increase with load
	baseTime := 100 * time.Millisecond
	loadFactor := 1.0 + (load * 0.5)
	
	return time.Duration(float64(baseTime) * loadFactor)
}

// getThroughput gets current throughput
func (lm *LoadMonitor) getThroughput() float64 {
	// In a real implementation, this would track actual throughput
	// For now, simulate based on concurrency and response time
	goroutines := runtime.NumGoroutine()
	responseTime := lm.getAverageResponseTime()
	
	if responseTime == 0 {
		return 0.0
	}
	
	// Throughput = concurrency / response_time
	return float64(goroutines) / responseTime.Seconds()
}

// ConcurrencyManager manages concurrency levels
type ConcurrencyManager struct {
	config           *AdaptiveConfig
	currentConcurrency int32
	targetConcurrency  int32
	lastAdjustment    time.Time
	adjustmentReason  string
	loadHistory       []*LoadMetrics
	mu                sync.RWMutex
}

// NewConcurrencyManager creates a new concurrency manager
func NewConcurrencyManager(config *AdaptiveConfig) *ConcurrencyManager {
	return &ConcurrencyManager{
		config:            config,
		currentConcurrency: int32(config.MinConcurrency),
		targetConcurrency:  int32(config.MinConcurrency),
		lastAdjustment:    time.Now(),
		loadHistory:       make([]*LoadMetrics, 0, 100),
	}
}

// GetCurrentConcurrency returns current concurrency level
func (cm *ConcurrencyManager) GetCurrentConcurrency() int {
	return int(atomic.LoadInt32(&cm.currentConcurrency))
}

// AdjustConcurrency adjusts concurrency level
func (cm *ConcurrencyManager) AdjustConcurrency(delta int, reason string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	current := int(atomic.LoadInt32(&cm.currentConcurrency))
	newConcurrency := current + delta
	
	// Apply bounds
	if newConcurrency < cm.config.MinConcurrency {
		newConcurrency = cm.config.MinConcurrency
	} else if newConcurrency > cm.config.MaxConcurrency {
		newConcurrency = cm.config.MaxConcurrency
	}
	
	// Update concurrency
	atomic.StoreInt32(&cm.currentConcurrency, int32(newConcurrency))
	atomic.StoreInt32(&cm.targetConcurrency, int32(newConcurrency))
	
	cm.lastAdjustment = time.Now()
	cm.adjustmentReason = reason
}

// GetState returns current concurrency state
func (cm *ConcurrencyManager) GetState() *ConcurrencyState {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	return &ConcurrencyState{
		CurrentConcurrency: int(atomic.LoadInt32(&cm.currentConcurrency)),
		TargetConcurrency:  int(atomic.LoadInt32(&cm.targetConcurrency)),
		LastAdjustment:    cm.lastAdjustment,
		AdjustmentReason:  cm.adjustmentReason,
		StabilityScore:    0.8, // Placeholder
	}
}

// GetStatistics returns concurrency statistics
func (cs *ConcurrencyState) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"current_concurrency": cs.CurrentConcurrency,
		"target_concurrency":  cs.TargetConcurrency,
		"last_adjustment":     cs.LastAdjustment,
		"adjustment_reason":   cs.AdjustmentReason,
		"stability_score":     cs.StabilityScore,
	}
}

// GetStatistics returns concurrency statistics
func (cm *ConcurrencyManager) GetStatistics() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	return map[string]interface{}{
		"current_concurrency": int(atomic.LoadInt32(&cm.currentConcurrency)),
		"target_concurrency":  int(atomic.LoadInt32(&cm.targetConcurrency)),
		"min_concurrency":     cm.config.MinConcurrency,
		"max_concurrency":     cm.config.MaxConcurrency,
		"last_adjustment":     cm.lastAdjustment,
		"adjustment_reason":   cm.adjustmentReason,
		"load_history_size":   len(cm.loadHistory),
	}
}
