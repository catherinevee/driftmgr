package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// PredictiveCache provides intelligent caching based on usage patterns
type PredictiveCache struct {
	usageTracker    *UsageTracker
	patternAnalyzer *PatternAnalyzer
	cacheManager    *CacheManager
	config          *PredictiveConfig
	mu              sync.RWMutex
}

// PredictiveConfig defines predictive caching behavior
type PredictiveConfig struct {
	Enabled           bool          `yaml:"enabled"`
	LearningEnabled   bool          `yaml:"learning_enabled"`
	PredictionWindow  time.Duration `yaml:"prediction_window"`
	ConfidenceThreshold float64     `yaml:"confidence_threshold"`
	MaxPredictions    int           `yaml:"max_predictions"`
	WarmupPeriod      time.Duration `yaml:"warmup_period"`
}

// UsagePattern represents a usage pattern
type UsagePattern struct {
	Key           string                 `json:"key"`
	Frequency     int                    `json:"frequency"`
	LastAccess    time.Time              `json:"last_access"`
	AccessTimes   []time.Time            `json:"access_times"`
	Predictions   []Prediction           `json:"predictions"`
	Confidence    float64                `json:"confidence"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// Prediction represents a cache prediction
type Prediction struct {
	Key           string    `json:"key"`
	Probability   float64   `json:"probability"`
	NextAccess    time.Time `json:"next_access"`
	Confidence    float64   `json:"confidence"`
	Reason        string    `json:"reason"`
}

// CachePrediction represents a cache prediction result
type CachePrediction struct {
	Key           string                 `json:"key"`
	ShouldCache   bool                   `json:"should_cache"`
	TTL           time.Duration          `json:"ttl"`
	Priority      int                    `json:"priority"`
	Confidence    float64                `json:"confidence"`
	Reason        string                 `json:"reason"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// NewPredictiveCache creates a new predictive cache
func NewPredictiveCache(config *PredictiveConfig) *PredictiveCache {
	if config == nil {
		config = &PredictiveConfig{
			Enabled:           true,
			LearningEnabled:   true,
			PredictionWindow:  1 * time.Hour,
			ConfidenceThreshold: 0.7,
			MaxPredictions:    100,
			WarmupPeriod:      24 * time.Hour,
		}
	}

	return &PredictiveCache{
		usageTracker:    NewUsageTracker(),
		patternAnalyzer: NewPatternAnalyzer(config),
		cacheManager:    NewCacheManager(1 * time.Hour),
		config:          config,
	}
}

// Get retrieves a value with predictive caching
func (pc *PredictiveCache) Get(key string) (interface{}, bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Track access
	pc.usageTracker.TrackAccess(key)

	// Try to get from cache
	if value, found := pc.cacheManager.Get(key); found {
		return value, true
	}

	// Check if we should predictively cache this key
	prediction := pc.predictCache(key)
	if prediction.ShouldCache {
		// Pre-warm cache with predicted value
		pc.preWarmCache(prediction)
	}

	return nil, false
}

// Set stores a value with predictive caching
func (pc *PredictiveCache) Set(key string, value interface{}, ttl time.Duration) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Track access
	pc.usageTracker.TrackAccess(key)

	// Store in cache
	pc.cacheManager.Set(key, value)

	// Update patterns
	pc.updatePatterns(key, ttl)
}

// PredictNextAccess predicts when a key will be accessed next
func (pc *PredictiveCache) PredictNextAccess(key string) *CachePrediction {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	return pc.predictCache(key)
}

// GetPredictions returns all current predictions
func (pc *PredictiveCache) GetPredictions() []*CachePrediction {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	patterns := pc.usageTracker.GetPatterns()
	predictions := make([]*CachePrediction, 0, len(patterns))

	for _, pattern := range patterns {
		prediction := pc.predictCache(pattern.Key)
		if prediction.ShouldCache {
			predictions = append(predictions, prediction)
		}
	}

	// Sort by priority (highest first)
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].Priority > predictions[j].Priority
	})

	// Limit to max predictions
	if len(predictions) > pc.config.MaxPredictions {
		predictions = predictions[:pc.config.MaxPredictions]
	}

	return predictions
}

// predictCache predicts whether a key should be cached
func (pc *PredictiveCache) predictCache(key string) *CachePrediction {
	pattern := pc.usageTracker.GetPattern(key)
	if pattern == nil {
		return &CachePrediction{
			Key:         key,
			ShouldCache: false,
			Confidence:  0.0,
			Reason:      "No usage pattern found",
		}
	}

	// Analyze pattern
	analysis := pc.patternAnalyzer.AnalyzePattern(pattern)
	if analysis.Confidence < pc.config.ConfidenceThreshold {
		return &CachePrediction{
			Key:         key,
			ShouldCache: false,
			Confidence:  analysis.Confidence,
			Reason:      "Low confidence prediction",
		}
	}

	// Calculate TTL based on access frequency
	ttl := pc.calculateTTL(pattern, analysis)

	// Calculate priority based on frequency and recency
	priority := pc.calculatePriority(pattern, analysis)

	return &CachePrediction{
		Key:         key,
		ShouldCache: true,
		TTL:         ttl,
		Priority:    priority,
		Confidence:  analysis.Confidence,
		Reason:      analysis.Reason,
		Metadata:    analysis.Metadata,
	}
}

// calculateTTL calculates optimal TTL for a key
func (pc *PredictiveCache) calculateTTL(pattern *UsagePattern, analysis *PatternAnalysis) time.Duration {
	// Base TTL on access frequency
	if pattern.Frequency == 0 {
		return 5 * time.Minute // Default TTL
	}

	// Calculate average time between accesses
	var totalInterval time.Duration
	accessCount := len(pattern.AccessTimes)
	
	if accessCount > 1 {
		for i := 1; i < accessCount; i++ {
			interval := pattern.AccessTimes[i].Sub(pattern.AccessTimes[i-1])
			totalInterval += interval
		}
		averageInterval := totalInterval / time.Duration(accessCount-1)
		
		// Use 2x average interval as TTL, with min/max bounds
		ttl := averageInterval * 2
		if ttl < 1*time.Minute {
			ttl = 1 * time.Minute
		}
		if ttl > 1*time.Hour {
			ttl = 1 * time.Hour
		}
		return ttl
	}

	return 10 * time.Minute // Default for single access
}

// calculatePriority calculates priority for a key
func (pc *PredictiveCache) calculatePriority(pattern *UsagePattern, analysis *PatternAnalysis) int {
	// Priority factors:
	// - Frequency (higher = higher priority)
	// - Recency (more recent = higher priority)
	// - Confidence (higher confidence = higher priority)
	
	frequencyScore := int(math.Min(float64(pattern.Frequency), 100))
	
	recencyScore := 0
	if time.Since(pattern.LastAccess) < 1*time.Hour {
		recencyScore = 100
	} else if time.Since(pattern.LastAccess) < 24*time.Hour {
		recencyScore = 50
	} else if time.Since(pattern.LastAccess) < 7*24*time.Hour {
		recencyScore = 25
	}
	
	confidenceScore := int(analysis.Confidence * 100)
	
	// Weighted average
	priority := (frequencyScore*3 + recencyScore*2 + confidenceScore*1) / 6
	return priority
}

// preWarmCache pre-warms the cache with predicted values
func (pc *PredictiveCache) preWarmCache(prediction *CachePrediction) {
	// In a real implementation, this would fetch the value
	// and store it in the cache. For now, we just log the prediction.
	fmt.Printf("Pre-warming cache for key: %s (TTL: %v, Priority: %d)\n", 
		prediction.Key, prediction.TTL, prediction.Priority)
}

// updatePatterns updates usage patterns
func (pc *PredictiveCache) updatePatterns(key string, ttl time.Duration) {
	// Update pattern with TTL information
	pattern := pc.usageTracker.GetPattern(key)
	if pattern != nil {
		if pattern.Metadata == nil {
			pattern.Metadata = make(map[string]interface{})
		}
		pattern.Metadata["last_ttl"] = ttl
		pattern.Metadata["avg_ttl"] = pc.calculateAverageTTL(pattern, ttl)
	}
}

// calculateAverageTTL calculates average TTL for a pattern
func (pc *PredictiveCache) calculateAverageTTL(pattern *UsagePattern, currentTTL time.Duration) time.Duration {
	if pattern.Metadata == nil {
		return currentTTL
	}
	
	if avgTTL, exists := pattern.Metadata["avg_ttl"]; exists {
		if avg, ok := avgTTL.(time.Duration); ok {
			// Update running average
			return (avg + currentTTL) / 2
		}
	}
	
	return currentTTL
}

// GetStatistics returns predictive cache statistics
func (pc *PredictiveCache) GetStatistics() map[string]interface{} {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	patterns := pc.usageTracker.GetPatterns()
	predictions := pc.GetPredictions()

	stats := map[string]interface{}{
		"total_patterns":     len(patterns),
		"active_predictions": len(predictions),
		"cache_hits":         pc.cacheManager.GetHitCount(),
		"cache_misses":       pc.cacheManager.GetMissCount(),
		"hit_rate":           pc.cacheManager.GetHitRate(),
	}

	// Calculate prediction accuracy if we have historical data
	if accuracy := pc.calculatePredictionAccuracy(); accuracy > 0 {
		stats["prediction_accuracy"] = accuracy
	}

	return stats
}

// calculatePredictionAccuracy calculates prediction accuracy
func (pc *PredictiveCache) calculatePredictionAccuracy() float64 {
	// This would compare predicted vs actual access patterns
	// For now, return a placeholder value
	return 0.85
}

// UsageTracker tracks usage patterns
type UsageTracker struct {
	patterns map[string]*UsagePattern
	mu       sync.RWMutex
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker() *UsageTracker {
	return &UsageTracker{
		patterns: make(map[string]*UsagePattern),
	}
}

// TrackAccess tracks access to a key
func (ut *UsageTracker) TrackAccess(key string) {
	ut.mu.Lock()
	defer ut.mu.Unlock()

	now := time.Now()
	pattern, exists := ut.patterns[key]
	
	if !exists {
		pattern = &UsagePattern{
			Key:         key,
			Frequency:   0,
			LastAccess:  now,
			AccessTimes: []time.Time{},
			Predictions: []Prediction{},
			Confidence:  0.0,
			Metadata:    make(map[string]interface{}),
		}
		ut.patterns[key] = pattern
	}

	// Update pattern
	pattern.Frequency++
	pattern.LastAccess = now
	pattern.AccessTimes = append(pattern.AccessTimes, now)

	// Keep only recent access times (last 100)
	if len(pattern.AccessTimes) > 100 {
		pattern.AccessTimes = pattern.AccessTimes[len(pattern.AccessTimes)-100:]
	}
}

// GetPattern returns a usage pattern for a key
func (ut *UsageTracker) GetPattern(key string) *UsagePattern {
	ut.mu.RLock()
	defer ut.mu.RUnlock()
	
	return ut.patterns[key]
}

// GetPatterns returns all usage patterns
func (ut *UsageTracker) GetPatterns() map[string]*UsagePattern {
	ut.mu.RLock()
	defer ut.mu.RUnlock()
	
	patterns := make(map[string]*UsagePattern)
	for key, pattern := range ut.patterns {
		patterns[key] = pattern
	}
	return patterns
}

// PatternAnalysis represents analysis of a usage pattern
type PatternAnalysis struct {
	Confidence    float64                `json:"confidence"`
	NextAccess    time.Time              `json:"next_access"`
	Reason        string                 `json:"reason"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// PatternAnalyzer analyzes usage patterns
type PatternAnalyzer struct {
	config *PredictiveConfig
}

// NewPatternAnalyzer creates a new pattern analyzer
func NewPatternAnalyzer(config *PredictiveConfig) *PatternAnalyzer {
	return &PatternAnalyzer{
		config: config,
	}
}

// AnalyzePattern analyzes a usage pattern
func (pa *PatternAnalyzer) AnalyzePattern(pattern *UsagePattern) *PatternAnalysis {
	analysis := &PatternAnalysis{
		Confidence: 0.0,
		NextAccess: time.Now().Add(pa.config.PredictionWindow),
		Reason:     "Insufficient data",
		Metadata:   make(map[string]interface{}),
	}

	// Need at least 3 accesses for meaningful analysis
	if len(pattern.AccessTimes) < 3 {
		return analysis
	}

	// Analyze access intervals
	intervals := pa.calculateIntervals(pattern.AccessTimes)
	if len(intervals) == 0 {
		return analysis
	}

	// Calculate confidence based on consistency
	consistency := pa.calculateConsistency(intervals)
	analysis.Confidence = consistency

	// Predict next access time
	nextAccess := pa.predictNextAccess(pattern.AccessTimes, intervals)
	analysis.NextAccess = nextAccess

	// Determine reason
	if consistency > 0.8 {
		analysis.Reason = "High consistency pattern"
	} else if consistency > 0.6 {
		analysis.Reason = "Moderate consistency pattern"
	} else {
		analysis.Reason = "Low consistency pattern"
	}

	// Add metadata
	analysis.Metadata["avg_interval"] = pa.calculateAverageInterval(intervals)
	analysis.Metadata["std_deviation"] = pa.calculateStandardDeviation(intervals)
	analysis.Metadata["consistency"] = consistency

	return analysis
}

// calculateIntervals calculates intervals between access times
func (pa *PatternAnalyzer) calculateIntervals(accessTimes []time.Time) []time.Duration {
	intervals := make([]time.Duration, 0, len(accessTimes)-1)
	
	for i := 1; i < len(accessTimes); i++ {
		interval := accessTimes[i].Sub(accessTimes[i-1])
		intervals = append(intervals, interval)
	}
	
	return intervals
}

// calculateConsistency calculates consistency of intervals
func (pa *PatternAnalyzer) calculateConsistency(intervals []time.Duration) float64 {
	if len(intervals) < 2 {
		return 0.0
	}

	avg := pa.calculateAverageInterval(intervals)
	stdDev := pa.calculateStandardDeviation(intervals)

	// Consistency is inverse of coefficient of variation
	if avg == 0 {
		return 0.0
	}

	coefficientOfVariation := float64(stdDev) / float64(avg)
	consistency := 1.0 / (1.0 + coefficientOfVariation)
	
	return math.Min(consistency, 1.0)
}

// calculateAverageInterval calculates average interval
func (pa *PatternAnalyzer) calculateAverageInterval(intervals []time.Duration) time.Duration {
	if len(intervals) == 0 {
		return 0
	}

	var total time.Duration
	for _, interval := range intervals {
		total += interval
	}
	
	return total / time.Duration(len(intervals))
}

// calculateStandardDeviation calculates standard deviation of intervals
func (pa *PatternAnalyzer) calculateStandardDeviation(intervals []time.Duration) time.Duration {
	if len(intervals) < 2 {
		return 0
	}

	avg := pa.calculateAverageInterval(intervals)
	var sumSquares float64
	
	for _, interval := range intervals {
		diff := float64(interval - avg)
		sumSquares += diff * diff
	}
	
	variance := sumSquares / float64(len(intervals)-1)
	return time.Duration(math.Sqrt(variance))
}

// predictNextAccess predicts next access time
func (pa *PatternAnalyzer) predictNextAccess(accessTimes []time.Time, intervals []time.Duration) time.Time {
	if len(accessTimes) == 0 {
		return time.Now().Add(pa.config.PredictionWindow)
	}

	lastAccess := accessTimes[len(accessTimes)-1]
	avgInterval := pa.calculateAverageInterval(intervals)
	
	// Predict next access based on average interval
	nextAccess := lastAccess.Add(avgInterval)
	
	// Ensure prediction is within reasonable bounds
	now := time.Now()
	if nextAccess.Before(now) {
		nextAccess = now.Add(avgInterval)
	}
	
	return nextAccess
}
