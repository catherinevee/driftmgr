package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsTracker tracks various application metrics
type MetricsTracker struct {
	// Detection metrics
	activeDetections    int64
	completedDetections int64
	failedDetections    int64

	// Cache metrics
	cacheHits      int64
	cacheMisses    int64
	cacheEvictions int64

	// WebSocket metrics
	messagesSent     int64
	messagesReceived int64
	messagesQueued   int64

	// API metrics
	apiRequests     int64
	apiErrors       int64
	apiLatencySum   int64
	apiLatencyCount int64

	// Discovery metrics
	discoveryRuns     int64
	discoveryErrors   int64
	resourcesFound    int64
	lastDiscoveryTime atomic.Value // stores *time.Time

	// Remediation metrics
	remediationRuns     int64
	remediationSuccess  int64
	remediationFailures int64

	mu sync.RWMutex
}

var (
	globalTracker *MetricsTracker
	trackerOnce   sync.Once
)

// GetGlobalTracker returns the global metrics tracker instance
func GetGlobalTracker() *MetricsTracker {
	trackerOnce.Do(func() {
		globalTracker = &MetricsTracker{}
	})
	return globalTracker
}

// IncrementActiveDetections increments the active detections counter
func (mt *MetricsTracker) IncrementActiveDetections() {
	atomic.AddInt64(&mt.activeDetections, 1)
}

// DecrementActiveDetections decrements the active detections counter
func (mt *MetricsTracker) DecrementActiveDetections() {
	atomic.AddInt64(&mt.activeDetections, -1)
}

// GetActiveDetections returns the current number of active detections
func (mt *MetricsTracker) GetActiveDetections() int64 {
	return atomic.LoadInt64(&mt.activeDetections)
}

// IncrementCompletedDetections increments the completed detections counter
func (mt *MetricsTracker) IncrementCompletedDetections() {
	atomic.AddInt64(&mt.completedDetections, 1)
}

// GetCompletedDetections returns the total number of completed detections
func (mt *MetricsTracker) GetCompletedDetections() int64 {
	return atomic.LoadInt64(&mt.completedDetections)
}

// IncrementFailedDetections increments the failed detections counter
func (mt *MetricsTracker) IncrementFailedDetections() {
	atomic.AddInt64(&mt.failedDetections, 1)
}

// GetFailedDetections returns the total number of failed detections
func (mt *MetricsTracker) GetFailedDetections() int64 {
	return atomic.LoadInt64(&mt.failedDetections)
}

// IncrementCacheHit increments the cache hit counter
func (mt *MetricsTracker) IncrementCacheHit() {
	atomic.AddInt64(&mt.cacheHits, 1)
}

// GetCacheHits returns the total number of cache hits
func (mt *MetricsTracker) GetCacheHits() int64 {
	return atomic.LoadInt64(&mt.cacheHits)
}

// IncrementCacheMiss increments the cache miss counter
func (mt *MetricsTracker) IncrementCacheMiss() {
	atomic.AddInt64(&mt.cacheMisses, 1)
}

// GetCacheMisses returns the total number of cache misses
func (mt *MetricsTracker) GetCacheMisses() int64 {
	return atomic.LoadInt64(&mt.cacheMisses)
}

// IncrementCacheEviction increments the cache eviction counter
func (mt *MetricsTracker) IncrementCacheEviction() {
	atomic.AddInt64(&mt.cacheEvictions, 1)
}

// GetCacheEvictions returns the total number of cache evictions
func (mt *MetricsTracker) GetCacheEvictions() int64 {
	return atomic.LoadInt64(&mt.cacheEvictions)
}

// IncrementMessagesSent increments the messages sent counter
func (mt *MetricsTracker) IncrementMessagesSent() {
	atomic.AddInt64(&mt.messagesSent, 1)
}

// GetMessagesSent returns the total number of messages sent
func (mt *MetricsTracker) GetMessagesSent() int64 {
	return atomic.LoadInt64(&mt.messagesSent)
}

// IncrementMessagesReceived increments the messages received counter
func (mt *MetricsTracker) IncrementMessagesReceived() {
	atomic.AddInt64(&mt.messagesReceived, 1)
}

// GetMessagesReceived returns the total number of messages received
func (mt *MetricsTracker) GetMessagesReceived() int64 {
	return atomic.LoadInt64(&mt.messagesReceived)
}

// SetMessagesQueued sets the current number of queued messages
func (mt *MetricsTracker) SetMessagesQueued(count int64) {
	atomic.StoreInt64(&mt.messagesQueued, count)
}

// GetMessagesQueued returns the current number of queued messages
func (mt *MetricsTracker) GetMessagesQueued() int64 {
	return atomic.LoadInt64(&mt.messagesQueued)
}

// RecordAPIRequest records an API request with latency
func (mt *MetricsTracker) RecordAPIRequest(latencyMs int64, isError bool) {
	atomic.AddInt64(&mt.apiRequests, 1)
	if isError {
		atomic.AddInt64(&mt.apiErrors, 1)
	}
	atomic.AddInt64(&mt.apiLatencySum, latencyMs)
	atomic.AddInt64(&mt.apiLatencyCount, 1)
}

// GetAPIMetrics returns API metrics
func (mt *MetricsTracker) GetAPIMetrics() (requests, errors, avgLatencyMs int64) {
	requests = atomic.LoadInt64(&mt.apiRequests)
	errors = atomic.LoadInt64(&mt.apiErrors)

	sum := atomic.LoadInt64(&mt.apiLatencySum)
	count := atomic.LoadInt64(&mt.apiLatencyCount)
	if count > 0 {
		avgLatencyMs = sum / count
	}

	return requests, errors, avgLatencyMs
}

// RecordDiscoveryRun records a discovery run
func (mt *MetricsTracker) RecordDiscoveryRun(resourcesFound int, isError bool) {
	atomic.AddInt64(&mt.discoveryRuns, 1)
	if isError {
		atomic.AddInt64(&mt.discoveryErrors, 1)
	}
	atomic.AddInt64(&mt.resourcesFound, int64(resourcesFound))

	now := time.Now()
	mt.lastDiscoveryTime.Store(&now)
}

// GetDiscoveryMetrics returns discovery metrics
func (mt *MetricsTracker) GetDiscoveryMetrics() (runs, errors, resources int64, lastRun *time.Time) {
	runs = atomic.LoadInt64(&mt.discoveryRuns)
	errors = atomic.LoadInt64(&mt.discoveryErrors)
	resources = atomic.LoadInt64(&mt.resourcesFound)

	if val := mt.lastDiscoveryTime.Load(); val != nil {
		lastRun = val.(*time.Time)
	}

	return runs, errors, resources, lastRun
}

// RecordRemediationRun records a remediation run
func (mt *MetricsTracker) RecordRemediationRun(success bool) {
	atomic.AddInt64(&mt.remediationRuns, 1)
	if success {
		atomic.AddInt64(&mt.remediationSuccess, 1)
	} else {
		atomic.AddInt64(&mt.remediationFailures, 1)
	}
}

// GetRemediationMetrics returns remediation metrics
func (mt *MetricsTracker) GetRemediationMetrics() (runs, success, failures int64) {
	runs = atomic.LoadInt64(&mt.remediationRuns)
	success = atomic.LoadInt64(&mt.remediationSuccess)
	failures = atomic.LoadInt64(&mt.remediationFailures)
	return runs, success, failures
}

// Reset resets all metrics (useful for testing)
func (mt *MetricsTracker) Reset() {
	atomic.StoreInt64(&mt.activeDetections, 0)
	atomic.StoreInt64(&mt.completedDetections, 0)
	atomic.StoreInt64(&mt.failedDetections, 0)
	atomic.StoreInt64(&mt.cacheHits, 0)
	atomic.StoreInt64(&mt.cacheMisses, 0)
	atomic.StoreInt64(&mt.cacheEvictions, 0)
	atomic.StoreInt64(&mt.messagesSent, 0)
	atomic.StoreInt64(&mt.messagesReceived, 0)
	atomic.StoreInt64(&mt.messagesQueued, 0)
	atomic.StoreInt64(&mt.apiRequests, 0)
	atomic.StoreInt64(&mt.apiErrors, 0)
	atomic.StoreInt64(&mt.apiLatencySum, 0)
	atomic.StoreInt64(&mt.apiLatencyCount, 0)
	atomic.StoreInt64(&mt.discoveryRuns, 0)
	atomic.StoreInt64(&mt.discoveryErrors, 0)
	atomic.StoreInt64(&mt.resourcesFound, 0)
	atomic.StoreInt64(&mt.remediationRuns, 0)
	atomic.StoreInt64(&mt.remediationSuccess, 0)
	atomic.StoreInt64(&mt.remediationFailures, 0)
	mt.lastDiscoveryTime.Store((*time.Time)(nil))
}

// GetAllMetrics returns all metrics as a map
func (mt *MetricsTracker) GetAllMetrics() map[string]interface{} {
	apiReqs, apiErrs, apiLatency := mt.GetAPIMetrics()
	discRuns, discErrs, resources, lastDisc := mt.GetDiscoveryMetrics()
	remRuns, remSuccess, remFails := mt.GetRemediationMetrics()

	return map[string]interface{}{
		"detections": map[string]int64{
			"active":    mt.GetActiveDetections(),
			"completed": mt.GetCompletedDetections(),
			"failed":    mt.GetFailedDetections(),
		},
		"cache": map[string]int64{
			"hits":      mt.GetCacheHits(),
			"misses":    mt.GetCacheMisses(),
			"evictions": mt.GetCacheEvictions(),
		},
		"websocket": map[string]int64{
			"sent":     mt.GetMessagesSent(),
			"received": mt.GetMessagesReceived(),
			"queued":   mt.GetMessagesQueued(),
		},
		"api": map[string]int64{
			"requests":       apiReqs,
			"errors":         apiErrs,
			"avg_latency_ms": apiLatency,
		},
		"discovery": map[string]interface{}{
			"runs":            discRuns,
			"errors":          discErrs,
			"resources_found": resources,
			"last_run":        lastDisc,
		},
		"remediation": map[string]int64{
			"runs":     remRuns,
			"success":  remSuccess,
			"failures": remFails,
		},
	}
}
