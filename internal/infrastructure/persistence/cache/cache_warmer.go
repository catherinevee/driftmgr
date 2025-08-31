package cache

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// NewCacheWarmer creates a new cache warmer instance
func NewCacheWarmer() *CacheWarmer {
	return &CacheWarmer{
		strategies: make([]WarmingStrategy, 0),
		schedule:   make(map[string]time.Duration),
		stopCh:     make(chan struct{}),
	}
}

// Start begins the cache warming process
func (cw *CacheWarmer) Start(cache *EnhancedCache) {
	cw.mu.Lock()
	if cw.running {
		cw.mu.Unlock()
		return
	}
	cw.running = true
	cw.mu.Unlock()
	
	// Initialize default strategies
	cw.initializeStrategies()
	
	// Start warming workers
	go cw.runWarmingLoop(cache)
	go cw.runPreemptiveWarming(cache)
	go cw.runPredictiveWarming(cache)
}

// Stop stops the cache warming process
func (cw *CacheWarmer) Stop() {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	if !cw.running {
		return
	}
	
	close(cw.stopCh)
	cw.running = false
}

// AddStrategy adds a warming strategy
func (cw *CacheWarmer) AddStrategy(strategy WarmingStrategy) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	cw.strategies = append(cw.strategies, strategy)
	
	// Sort strategies by priority
	sort.Slice(cw.strategies, func(i, j int) bool {
		return cw.strategies[i].Priority() > cw.strategies[j].Priority()
	})
}

// SetSchedule sets the warming schedule for a pattern
func (cw *CacheWarmer) SetSchedule(pattern string, interval time.Duration) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	cw.schedule[pattern] = interval
}

// initializeStrategies sets up default warming strategies
func (cw *CacheWarmer) initializeStrategies() {
	// TTL-based warming
	cw.AddStrategy(&TTLWarmingStrategy{
		threshold: 0.8, // Warm when 80% of TTL has passed
	})
	
	// Access pattern warming
	cw.AddStrategy(&AccessPatternWarmingStrategy{
		minAccessCount: 5,
		timeWindow:     1 * time.Hour,
	})
	
	// Priority warming for critical resources
	cw.AddStrategy(&PriorityWarmingStrategy{
		criticalPatterns: []string{
			"state_file:",
			"terraform:",
			"compliance:",
		},
	})
	
	// Dependency warming
	cw.AddStrategy(&DependencyWarmingStrategy{})
	
	// Time-based warming for peak hours
	cw.AddStrategy(&TimeBasedWarmingStrategy{
		peakHours: []int{9, 10, 11, 14, 15, 16}, // Business hours
	})
}

// runWarmingLoop runs the main warming loop
func (cw *CacheWarmer) runWarmingLoop(cache *EnhancedCache) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cw.warmExpiring(cache)
		case <-cw.stopCh:
			return
		}
	}
}

// runPreemptiveWarming performs preemptive cache warming
func (cw *CacheWarmer) runPreemptiveWarming(cache *EnhancedCache) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cw.preemptiveWarm(cache)
		case <-cw.stopCh:
			return
		}
	}
}

// runPredictiveWarming performs predictive cache warming
func (cw *CacheWarmer) runPredictiveWarming(cache *EnhancedCache) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cw.predictiveWarm(cache)
		case <-cw.stopCh:
			return
		}
	}
}

// warmExpiring warms cache entries that are about to expire
func (cw *CacheWarmer) warmExpiring(cache *EnhancedCache) {
	cw.mu.RLock()
	strategies := make([]WarmingStrategy, len(cw.strategies))
	copy(strategies, cw.strategies)
	cw.mu.RUnlock()
	
	// Get all cache entries
	entries := cache.getAllEntries()
	
	for _, entry := range entries {
		for _, strategy := range strategies {
			if strategy.ShouldWarm(entry.Key, entry) {
				go func(key string) {
					if err := strategy.Warm(cache, key); err != nil {
						// Log error but continue
						fmt.Printf("Failed to warm cache entry %s: %v\n", key, err)
					}
				}(entry.Key)
				break // Only use first matching strategy
			}
		}
	}
}

// preemptiveWarm performs preemptive warming based on patterns
func (cw *CacheWarmer) preemptiveWarm(cache *EnhancedCache) {
	analytics := cache.GetAnalytics()
	
	// Warm hot keys that are not currently cached
	hotKeys := analytics.GetHotKeys()
	for _, key := range hotKeys {
		if _, found := cache.Get(key); !found {
			// Key was hot but is now missing, warm it
			go cw.warmKey(cache, key)
		}
	}
	
	// Warm frequently accessed patterns
	patterns := analytics.GetAccessPatterns()
	for pattern, stats := range patterns {
		if stats.AccessCount > 10 && time.Since(stats.LastAccessed) < 10*time.Minute {
			go cw.warmPattern(cache, pattern)
		}
	}
}

// predictiveWarm performs predictive warming based on historical patterns
func (cw *CacheWarmer) predictiveWarm(cache *EnhancedCache) {
	analytics := cache.GetAnalytics()
	
	// Predict keys likely to be accessed soon
	currentHour := time.Now().Hour()
	patterns := analytics.GetAccessPatterns()
	
	for pattern, stats := range patterns {
		// If this pattern is typically accessed at this hour
		if stats.PeakHour == currentHour {
			go cw.warmPattern(cache, pattern)
		}
	}
	
	// Warm related resources when primary resources are accessed
	recentlyAccessed := analytics.GetRecentlyAccessed(5 * time.Minute)
	for _, key := range recentlyAccessed {
		go cw.warmRelated(cache, key)
	}
}

// warmKey warms a specific cache key
func (cw *CacheWarmer) warmKey(cache *EnhancedCache, key string) error {
	// This would typically fetch fresh data from the source
	// For now, we'll just ensure it's in a higher tier
	if entry, found := cache.Get(key); found {
		// Promote to L1 cache
		cache.promoteToTier(key, entry, "L1")
	}
	return nil
}

// warmPattern warms all entries matching a pattern
func (cw *CacheWarmer) warmPattern(cache *EnhancedCache, pattern string) error {
	entries := cache.getEntriesMatchingPattern(pattern)
	
	for _, entry := range entries {
		if err := cw.warmKey(cache, entry.Key); err != nil {
			return err
		}
	}
	
	return nil
}

// warmRelated warms resources related to a key
func (cw *CacheWarmer) warmRelated(cache *EnhancedCache, key string) error {
	relationships := cache.relationshipCache.GetRelationships(key)
	
	for _, rel := range relationships {
		if rel.Strength > 0.5 { // Only warm strongly related resources
			go cw.warmKey(cache, rel.TargetKey)
		}
	}
	
	return nil
}

// TTLWarmingStrategy warms entries based on TTL
type TTLWarmingStrategy struct {
	threshold float64 // Percentage of TTL elapsed
}

func (s *TTLWarmingStrategy) ShouldWarm(key string, entry *CacheEntry) bool {
	if entry.ExpiresAt.IsZero() {
		return false
	}
	
	totalTTL := entry.ExpiresAt.Sub(entry.CreatedAt)
	elapsed := time.Since(entry.CreatedAt)
	
	return elapsed.Seconds()/totalTTL.Seconds() >= s.threshold
}

func (s *TTLWarmingStrategy) Warm(cache *EnhancedCache, key string) error {
	// Refresh the entry with fresh data
	// This would typically involve fetching from the source
	return nil
}

func (s *TTLWarmingStrategy) Priority() int {
	return 80
}

// AccessPatternWarmingStrategy warms based on access patterns
type AccessPatternWarmingStrategy struct {
	minAccessCount int
	timeWindow     time.Duration
}

func (s *AccessPatternWarmingStrategy) ShouldWarm(key string, entry *CacheEntry) bool {
	// Check if frequently accessed and about to expire
	if entry.AccessCount < s.minAccessCount {
		return false
	}
	
	timeSinceLastAccess := time.Since(entry.LastAccessed)
	return timeSinceLastAccess < s.timeWindow && 
	       time.Until(entry.ExpiresAt) < 2*time.Minute
}

func (s *AccessPatternWarmingStrategy) Warm(cache *EnhancedCache, key string) error {
	// Refresh frequently accessed entries
	return nil
}

func (s *AccessPatternWarmingStrategy) Priority() int {
	return 70
}

// PriorityWarmingStrategy warms critical resources
type PriorityWarmingStrategy struct {
	criticalPatterns []string
}

func (s *PriorityWarmingStrategy) ShouldWarm(key string, entry *CacheEntry) bool {
	for _, pattern := range s.criticalPatterns {
		if matchesPattern(key, pattern) {
			return time.Until(entry.ExpiresAt) < 5*time.Minute
		}
	}
	return false
}

func (s *PriorityWarmingStrategy) Warm(cache *EnhancedCache, key string) error {
	// Ensure critical resources are always fresh
	return nil
}

func (s *PriorityWarmingStrategy) Priority() int {
	return 100
}

// DependencyWarmingStrategy warms dependencies together
type DependencyWarmingStrategy struct{}

func (s *DependencyWarmingStrategy) ShouldWarm(key string, entry *CacheEntry) bool {
	// Warm if entry has dependencies that are fresher
	return len(entry.Relationships) > 0 && 
	       time.Until(entry.ExpiresAt) < 3*time.Minute
}

func (s *DependencyWarmingStrategy) Warm(cache *EnhancedCache, key string) error {
	// Warm entry and its dependencies
	relationships := cache.relationshipCache.GetRelationships(key)
	
	for _, rel := range relationships {
		if rel.Type == "dependency" {
			// Warm dependent resources
			cache.Get(rel.TargetKey) // This triggers warming if needed
		}
	}
	
	return nil
}

func (s *DependencyWarmingStrategy) Priority() int {
	return 60
}

// TimeBasedWarmingStrategy warms during peak hours
type TimeBasedWarmingStrategy struct {
	peakHours []int
}

func (s *TimeBasedWarmingStrategy) ShouldWarm(key string, entry *CacheEntry) bool {
	currentHour := time.Now().Hour()
	
	for _, hour := range s.peakHours {
		if currentHour == hour {
			return time.Until(entry.ExpiresAt) < 10*time.Minute
		}
	}
	
	return false
}

func (s *TimeBasedWarmingStrategy) Warm(cache *EnhancedCache, key string) error {
	// Proactively warm during peak hours
	return nil
}

func (s *TimeBasedWarmingStrategy) Priority() int {
	return 50
}

// GranularTTLManager manages per-entry TTL settings
type GranularTTLManager struct {
	ttlRules   []TTLRule
	defaultTTL time.Duration
	mu         sync.RWMutex
}

// TTLRule defines TTL for specific patterns
type TTLRule struct {
	Pattern  string
	TTL      time.Duration
	Priority int
	Condition func(*CacheEntry) bool
}

// NewGranularTTLManager creates a new TTL manager
func NewGranularTTLManager(defaultTTL time.Duration) *GranularTTLManager {
	manager := &GranularTTLManager{
		ttlRules:   make([]TTLRule, 0),
		defaultTTL: defaultTTL,
	}
	
	// Add default rules
	manager.initializeDefaultRules()
	
	return manager
}

// initializeDefaultRules sets up default TTL rules
func (m *GranularTTLManager) initializeDefaultRules() {
	// Critical resources get longer TTL
	m.AddRule(TTLRule{
		Pattern:  "state_file:",
		TTL:      1 * time.Hour,
		Priority: 100,
	})
	
	// Compliance data needs frequent refresh
	m.AddRule(TTLRule{
		Pattern:  "compliance:",
		TTL:      5 * time.Minute,
		Priority: 90,
	})
	
	// Network topology changes infrequently
	m.AddRule(TTLRule{
		Pattern:  "network_topology:",
		TTL:      30 * time.Minute,
		Priority: 80,
	})
	
	// IAM relationships are relatively stable
	m.AddRule(TTLRule{
		Pattern:  "iam:",
		TTL:      20 * time.Minute,
		Priority: 70,
	})
	
	// Metrics need frequent updates
	m.AddRule(TTLRule{
		Pattern:  "metrics:",
		TTL:      1 * time.Minute,
		Priority: 95,
	})
	
	// Cost data updates daily
	m.AddRule(TTLRule{
		Pattern:  "cost:",
		TTL:      24 * time.Hour,
		Priority: 60,
	})
	
	// Add confidence-based TTL
	m.AddRule(TTLRule{
		Pattern: "*",
		TTL:     10 * time.Minute,
		Priority: 10,
		Condition: func(entry *CacheEntry) bool {
			return entry.ConfidenceScore > 0.9
		},
	})
	
	// Error entries get shorter TTL
	m.AddRule(TTLRule{
		Pattern: "*",
		TTL:     2 * time.Minute,
		Priority: 20,
		Condition: func(entry *CacheEntry) bool {
			return entry.Metadata != nil && entry.Metadata.ErrorCount > 0
		},
	})
}

// AddRule adds a TTL rule
func (m *GranularTTLManager) AddRule(rule TTLRule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ttlRules = append(m.ttlRules, rule)
	
	// Sort by priority
	sort.Slice(m.ttlRules, func(i, j int) bool {
		return m.ttlRules[i].Priority > m.ttlRules[j].Priority
	})
}

// GetTTL calculates TTL for an entry
func (m *GranularTTLManager) GetTTL(key string, entry *CacheEntry) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, rule := range m.ttlRules {
		// Check pattern match
		if !matchesPattern(key, rule.Pattern) && rule.Pattern != "*" {
			continue
		}
		
		// Check condition if present
		if rule.Condition != nil && !rule.Condition(entry) {
			continue
		}
		
		// Apply dynamic adjustments
		ttl := m.adjustTTL(rule.TTL, entry)
		return ttl
	}
	
	return m.defaultTTL
}

// adjustTTL dynamically adjusts TTL based on entry characteristics
func (m *GranularTTLManager) adjustTTL(baseTTL time.Duration, entry *CacheEntry) time.Duration {
	ttl := baseTTL
	
	// Adjust based on confidence score
	if entry.ConfidenceScore < 0.5 {
		ttl = ttl / 2
	} else if entry.ConfidenceScore > 0.95 {
		ttl = ttl * 2
	}
	
	// Adjust based on access frequency
	if entry.AccessCount > 100 {
		ttl = ttl * 3 / 2 // Extend TTL for frequently accessed items
	} else if entry.AccessCount < 5 {
		ttl = ttl * 2 / 3 // Reduce TTL for rarely accessed items
	}
	
	// Adjust based on data size
	if entry.Metadata != nil && entry.Metadata.ProcessingTime > 5*time.Second {
		ttl = ttl * 2 // Extend TTL for expensive-to-compute entries
	}
	
	// Ensure TTL is within reasonable bounds
	if ttl < 30*time.Second {
		ttl = 30 * time.Second
	} else if ttl > 24*time.Hour {
		ttl = 24 * time.Hour
	}
	
	return ttl
}

// InvalidationStrategy defines how cache entries are invalidated
type InvalidationStrategy interface {
	ShouldInvalidate(entry *CacheEntry) bool
	InvalidationReason() string
}

// StaleDataInvalidation invalidates stale data
type StaleDataInvalidation struct {
	maxAge time.Duration
}

func (s *StaleDataInvalidation) ShouldInvalidate(entry *CacheEntry) bool {
	return time.Since(entry.UpdatedAt) > s.maxAge
}

func (s *StaleDataInvalidation) InvalidationReason() string {
	return "data too old"
}

// ErrorThresholdInvalidation invalidates entries with too many errors
type ErrorThresholdInvalidation struct {
	maxErrors int
}

func (s *ErrorThresholdInvalidation) ShouldInvalidate(entry *CacheEntry) bool {
	return entry.Metadata != nil && entry.Metadata.ErrorCount > s.maxErrors
}

func (s *ErrorThresholdInvalidation) InvalidationReason() string {
	return "error threshold exceeded"
}

// ConfidenceInvalidation invalidates low confidence entries
type ConfidenceInvalidation struct {
	minConfidence float64
}

func (s *ConfidenceInvalidation) ShouldInvalidate(entry *CacheEntry) bool {
	return entry.ConfidenceScore < s.minConfidence
}

func (s *ConfidenceInvalidation) InvalidationReason() string {
	return "confidence score too low"
}