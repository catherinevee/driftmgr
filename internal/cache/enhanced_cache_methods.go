package cache

import (
	"fmt"
	"time"
)

// Additional methods for EnhancedCache

// getAllEntries returns all cache entries across all tiers
func (ec *EnhancedCache) getAllEntries() []*CacheEntry {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	
	entries := make([]*CacheEntry, 0)
	
	// Collect from all tiers
	entries = append(entries, ec.l1Cache.getAllEntries()...)
	entries = append(entries, ec.l2Cache.getAllEntries()...)
	entries = append(entries, ec.l3Cache.getAllEntries()...)
	
	return entries
}

// getAllEntries for TierCache
func (tc *TierCache) getAllEntries() []*CacheEntry {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	
	entries := make([]*CacheEntry, 0, len(tc.data))
	for _, entry := range tc.data {
		entries = append(entries, entry)
	}
	
	return entries
}

// promoteToTier promotes an entry to a specific tier
func (ec *EnhancedCache) promoteToTier(key string, entry *CacheEntry, tier string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	switch tier {
	case "L1":
		entry.ExpiresAt = time.Now().Add(ec.l1Cache.defaultTTL)
		ec.l1Cache.Set(key, entry)
	case "L2":
		entry.ExpiresAt = time.Now().Add(ec.l2Cache.defaultTTL)
		ec.l2Cache.Set(key, entry)
	case "L3":
		entry.ExpiresAt = time.Now().Add(ec.l3Cache.defaultTTL)
		ec.l3Cache.Set(key, entry)
	}
}

// getEntriesMatchingPattern returns entries matching a pattern
func (ec *EnhancedCache) getEntriesMatchingPattern(pattern string) []*CacheEntry {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	
	entries := make([]*CacheEntry, 0)
	
	// Check all tiers
	entries = append(entries, ec.l1Cache.getEntriesMatchingPattern(pattern)...)
	entries = append(entries, ec.l2Cache.getEntriesMatchingPattern(pattern)...)
	entries = append(entries, ec.l3Cache.getEntriesMatchingPattern(pattern)...)
	
	return entries
}

// getEntriesMatchingPattern for TierCache
func (tc *TierCache) getEntriesMatchingPattern(pattern string) []*CacheEntry {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	
	entries := make([]*CacheEntry, 0)
	for key, entry := range tc.data {
		if matchesPattern(key, pattern) {
			entries = append(entries, entry)
		}
	}
	
	return entries
}

// NewAggregationCache creates a new aggregation cache
func NewAggregationCache() *AggregationCache {
	return &AggregationCache{
		providerSummaries:    make(map[string]*ProviderSummary),
		regionalDistribution: make(map[string]*RegionalDistribution),
		typeCategories:       make(map[string]*TypeCategory),
		driftPatterns:        make(map[string]*DriftPattern),
		complianceScores:     make(map[string]*ComplianceScore),
		costAnalysis:         make(map[string]*CostAnalysis),
	}
}

// Get retrieves an aggregation by type
func (ac *AggregationCache) Get(aggregationType string) (interface{}, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	
	switch aggregationType {
	case "provider_summary":
		return ac.providerSummaries, len(ac.providerSummaries) > 0
	case "regional_distribution":
		return ac.regionalDistribution, len(ac.regionalDistribution) > 0
	case "type_categories":
		return ac.typeCategories, len(ac.typeCategories) > 0
	case "drift_patterns":
		return ac.driftPatterns, len(ac.driftPatterns) > 0
	case "compliance_scores":
		return ac.complianceScores, len(ac.complianceScores) > 0
	case "cost_analysis":
		return ac.costAnalysis, len(ac.costAnalysis) > 0
	default:
		return nil, false
	}
}

// InvalidatePattern invalidates entries matching a pattern
func (ac *AggregationCache) InvalidatePattern(pattern string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	
	// Clear matching entries
	for key := range ac.providerSummaries {
		if matchesPattern(key, pattern) {
			delete(ac.providerSummaries, key)
		}
	}
	
	for key := range ac.regionalDistribution {
		if matchesPattern(key, pattern) {
			delete(ac.regionalDistribution, key)
		}
	}
}

// NewVisualizationCache creates a new visualization cache
func NewVisualizationCache() *VisualizationCache {
	return &VisualizationCache{
		graphLayouts:   make(map[string]*GraphLayout),
		hierarchyTrees: make(map[string]*HierarchyTree),
		heatMapData:    make(map[string]*HeatMapData),
		timeSeriesData: make(map[string]*TimeSeriesData),
		sankeyData:     make(map[string]*SankeyData),
	}
}

// Get retrieves visualization data
func (vc *VisualizationCache) Get(vizType string, params map[string]interface{}) (interface{}, bool) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	
	// Create cache key from type and params
	key := vizType
	if params != nil {
		// Simple param serialization for cache key
		for k, v := range params {
			key += fmt.Sprintf("_%s_%v", k, v)
		}
	}
	
	switch vizType {
	case "graph":
		if layout, exists := vc.graphLayouts[key]; exists {
			return layout, true
		}
	case "hierarchy":
		if tree, exists := vc.hierarchyTrees[key]; exists {
			return tree, true
		}
	case "heatmap":
		if data, exists := vc.heatMapData[key]; exists {
			return data, true
		}
	case "timeseries":
		if data, exists := vc.timeSeriesData[key]; exists {
			return data, true
		}
	case "sankey":
		if data, exists := vc.sankeyData[key]; exists {
			return data, true
		}
	}
	
	return nil, false
}

// UpdateVisualizationData updates visualization cache with new data
func (vc *VisualizationCache) UpdateVisualizationData(entry *CacheEntry) {
	// This would be implemented to update visualization data based on the entry
	// For now, it's a placeholder
}

// InvalidatePattern invalidates visualization entries matching a pattern
func (vc *VisualizationCache) InvalidatePattern(pattern string) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	
	// Clear matching entries
	for key := range vc.graphLayouts {
		if matchesPattern(key, pattern) {
			delete(vc.graphLayouts, key)
		}
	}
	
	for key := range vc.hierarchyTrees {
		if matchesPattern(key, pattern) {
			delete(vc.hierarchyTrees, key)
		}
	}
}

// NewSearchIndexCache creates a new search index cache
func NewSearchIndexCache() *SearchIndexCache {
	return &SearchIndexCache{
		fullTextIndex: make(map[string][]string),
		facetIndex:    make(map[string]map[string][]string),
		fuzzyIndex:    &FuzzyIndex{
			trigrams: make(map[string][]string),
			phonetic: make(map[string][]string),
		},
		suggestions: &SuggestionEngine{
			frequentQueries: make([]string, 0),
			queryPatterns:   make(map[string]int),
			completions:     make(map[string][]string),
		},
	}
}

// IndexEntry indexes a cache entry for search
func (sic *SearchIndexCache) IndexEntry(key string, entry *CacheEntry) {
	sic.mu.Lock()
	defer sic.mu.Unlock()
	
	// Index for full-text search
	if entry.Metadata != nil {
		searchText := entry.Metadata.ResourceType + " " + entry.Metadata.Provider
		if entry.Metadata.Custom != nil {
			if name, exists := entry.Metadata.Custom["name"]; exists {
				searchText += " " + name
			}
		}
		
		sic.fullTextIndex[key] = []string{searchText}
		
		// Index facets
		if sic.facetIndex["provider"] == nil {
			sic.facetIndex["provider"] = make(map[string][]string)
		}
		sic.facetIndex["provider"][entry.Metadata.Provider] = append(
			sic.facetIndex["provider"][entry.Metadata.Provider], key)
		
		if sic.facetIndex["type"] == nil {
			sic.facetIndex["type"] = make(map[string][]string)
		}
		sic.facetIndex["type"][entry.Metadata.ResourceType] = append(
			sic.facetIndex["type"][entry.Metadata.ResourceType], key)
	}
}

// Search performs a search across the index
func (sic *SearchIndexCache) Search(query string, filters map[string]interface{}) ([]*CacheEntry, error) {
	sic.mu.RLock()
	defer sic.mu.RUnlock()
	
	// Simple search implementation
	// In production, this would use a proper search algorithm
	results := make([]*CacheEntry, 0)
	
	// For now, return empty results
	// Full implementation would search the indices
	
	return results, nil
}

// InvalidatePattern invalidates search index entries matching a pattern
func (sic *SearchIndexCache) InvalidatePattern(pattern string) {
	sic.mu.Lock()
	defer sic.mu.Unlock()
	
	// Clear matching entries
	for key := range sic.fullTextIndex {
		if matchesPattern(key, pattern) {
			delete(sic.fullTextIndex, key)
		}
	}
}

// CacheAnalytics methods

// NewCacheAnalytics creates new cache analytics
func NewCacheAnalytics() *CacheAnalytics {
	return &CacheAnalytics{
		accessPatterns: make(map[string]*AccessPattern),
	}
}

// Start starts the analytics collection
func (ca *CacheAnalytics) Start(cache *EnhancedCache) {
	// This would start background analytics collection
	// For now, it's a placeholder
}

// GetHitRate returns the cache hit rate
func (ca *CacheAnalytics) GetHitRate() float64 {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	
	if ca.hitRate == 0 && ca.missRate == 0 {
		return 0
	}
	
	total := ca.hitRate + ca.missRate
	if total == 0 {
		return 0
	}
	
	return ca.hitRate / total
}

// GetMissRate returns the cache miss rate
func (ca *CacheAnalytics) GetMissRate() float64 {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	
	if ca.hitRate == 0 && ca.missRate == 0 {
		return 0
	}
	
	total := ca.hitRate + ca.missRate
	if total == 0 {
		return 0
	}
	
	return ca.missRate / total
}

// GetAverageLatency returns the average cache latency
func (ca *CacheAnalytics) GetAverageLatency() time.Duration {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	
	return ca.avgLatency
}

// GetCachedItemCount returns the number of cached items
func (ca *CacheAnalytics) GetCachedItemCount() int {
	// This would count items across all cache tiers
	// For now, return 0
	return 0
}

// GetHotKeys returns the most frequently accessed keys
func (ca *CacheAnalytics) GetHotKeys() []string {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	
	if len(ca.hotKeys) > 0 {
		return ca.hotKeys
	}
	
	// Calculate hot keys from access patterns
	hotKeys := make([]string, 0)
	for key, pattern := range ca.accessPatterns {
		if pattern.AccessCount > 10 {
			hotKeys = append(hotKeys, key)
		}
	}
	
	return hotKeys
}

// GetAccessPatterns returns access patterns
func (ca *CacheAnalytics) GetAccessPatterns() map[string]*AccessPattern {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	
	// Return a copy to avoid external modifications
	patterns := make(map[string]*AccessPattern)
	for k, v := range ca.accessPatterns {
		patterns[k] = v
	}
	
	return patterns
}

// GetRecentlyAccessed returns recently accessed keys
func (ca *CacheAnalytics) GetRecentlyAccessed(duration time.Duration) []string {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	
	recent := make([]string, 0)
	cutoff := time.Now().Add(-duration)
	
	for key, pattern := range ca.accessPatterns {
		if pattern.LastAccessed.After(cutoff) {
			recent = append(recent, key)
		}
	}
	
	return recent
}

// IsHotKey checks if a key is frequently accessed
func (ca *CacheAnalytics) IsHotKey(key string) bool {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	
	for _, hotKey := range ca.hotKeys {
		if hotKey == key {
			return true
		}
	}
	
	if pattern, exists := ca.accessPatterns[key]; exists {
		return pattern.AccessCount > 10
	}
	
	return false
}

// GetAccessCount returns the access count for a key
func (ca *CacheAnalytics) GetAccessCount(key string) int {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	
	if pattern, exists := ca.accessPatterns[key]; exists {
		return pattern.AccessCount
	}
	
	return 0
}

// RecordSet records a cache set operation
func (ca *CacheAnalytics) RecordSet(key string, tier string) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	
	if ca.accessPatterns[key] == nil {
		ca.accessPatterns[key] = &AccessPattern{
			Key: key,
		}
	}
	
	ca.accessPatterns[key].LastAccessed = time.Now()
}

// RecordHit records a cache hit
func (ca *CacheAnalytics) RecordHit(key string, tier string) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	
	ca.hitRate++
	
	if ca.accessPatterns[key] == nil {
		ca.accessPatterns[key] = &AccessPattern{
			Key: key,
		}
	}
	
	ca.accessPatterns[key].AccessCount++
	ca.accessPatterns[key].LastAccessed = time.Now()
}

// RecordMiss records a cache miss
func (ca *CacheAnalytics) RecordMiss(key string) {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	
	ca.missRate++
}

// SetAggregation sets an aggregation in the cache
func (ec *EnhancedCache) SetAggregation(key string, data interface{}) {
	ec.GetAggregationCache().Set(key, data)
}

// GetAggregationCache returns the aggregation cache
func (ec *EnhancedCache) GetAggregationCache() *AggregationCache {
	if ec.aggregationCache == nil {
		ec.aggregationCache = NewAggregationCache()
	}
	return ec.aggregationCache
}

// Set stores an aggregation
func (ac *AggregationCache) Set(key string, data interface{}) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	
	switch key {
	case "provider_summary":
		// Store provider summaries
		if summaries, ok := data.(map[string]interface{}); ok {
			for provider, summary := range summaries {
				if s, ok := summary.(map[string]interface{}); ok {
					ac.providerSummaries[provider] = &ProviderSummary{
						Provider:         provider,
						TotalResources:   getInt(s, "total_resources"),
						DriftedResources: getInt(s, "drifted_resources"),
						ComplianceScore:  getFloat(s, "compliance_rate"),
						EstimatedCost:    getFloat(s, "estimated_cost"),
						LastUpdated:      time.Now(),
					}
				}
			}
		}
	case "cost_analysis":
		// Store cost analysis
		if analysis, ok := data.(map[string]interface{}); ok {
			ac.costAnalysis["global"] = &CostAnalysis{
				TotalCost:        getFloat(analysis, "total_cost"),
				CostByProvider:   getFloatMap(analysis, "cost_by_provider"),
				CostByRegion:     getFloatMap(analysis, "cost_by_region"),
				CostByType:       getFloatMap(analysis, "cost_by_type"),
				UnusedResources:  getStringSlice(analysis, "unused_resources"),
				LastCalculated:   time.Now(),
			}
		}
	}
}

// Helper functions for type conversion
func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(int); ok {
		return val
	}
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	if val, ok := m[key].(int); ok {
		return float64(val)
	}
	return 0.0
}

func getFloatMap(m map[string]interface{}, key string) map[string]float64 {
	result := make(map[string]float64)
	if val, ok := m[key].(map[string]float64); ok {
		return val
	}
	if val, ok := m[key].(map[string]interface{}); ok {
		for k, v := range val {
			if f, ok := v.(float64); ok {
				result[k] = f
			}
		}
	}
	return result
}

func getStringSlice(m map[string]interface{}, key string) []string {
	result := make([]string, 0)
	if val, ok := m[key].([]string); ok {
		return val
	}
	if val, ok := m[key].([]interface{}); ok {
		for _, v := range val {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
	}
	return result
}

// GetRelationshipCache returns the relationship cache
func (ec *EnhancedCache) GetRelationshipCache() *RelationshipCache {
	return ec.relationshipCache
}

// Clear removes all entries from all cache tiers
func (ec *EnhancedCache) Clear() error {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	// Clear all tiers
	if ec.l1Cache != nil {
		ec.l1Cache.Clear()
	}
	if ec.l2Cache != nil {
		ec.l2Cache.Clear()
	}
	if ec.l3Cache != nil {
		ec.l3Cache.Clear()
	}
	
	// Clear specialized caches
	if ec.relationshipCache != nil {
		ec.relationshipCache.Clear()
	}
	if ec.aggregationCache != nil {
		ec.aggregationCache.Clear()
	}
	if ec.visualizationCache != nil {
		ec.visualizationCache.Clear()
	}
	if ec.searchIndexCache != nil {
		ec.searchIndexCache.Clear()
	}
	
	return nil
}

// Delete removes a value from all cache tiers
func (ec *EnhancedCache) Delete(key string) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	// Delete from all tiers
	if ec.l1Cache != nil {
		ec.l1Cache.Delete(key)
	}
	if ec.l2Cache != nil {
		ec.l2Cache.Delete(key)
	}
	if ec.l3Cache != nil {
		ec.l3Cache.Delete(key)
	}
	
	return nil
}

// Keys returns all cache keys matching a pattern
func (ec *EnhancedCache) Keys(pattern string) []string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	
	keyMap := make(map[string]bool)
	
	// Collect keys from all tiers
	if ec.l1Cache != nil {
		for _, key := range ec.l1Cache.Keys(pattern) {
			keyMap[key] = true
		}
	}
	if ec.l2Cache != nil {
		for _, key := range ec.l2Cache.Keys(pattern) {
			keyMap[key] = true
		}
	}
	if ec.l3Cache != nil {
		for _, key := range ec.l3Cache.Keys(pattern) {
			keyMap[key] = true
		}
	}
	
	// Convert map to slice
	keys := make([]string, 0, len(keyMap))
	for key := range keyMap {
		keys = append(keys, key)
	}
	
	return keys
}