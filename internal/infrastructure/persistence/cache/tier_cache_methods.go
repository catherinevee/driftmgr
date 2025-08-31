package cache

import (
	"time"
)

// Clear removes all entries from the tier cache
func (tc *TierCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	tc.data = make(map[string]*CacheEntry)
	tc.ttlIndex = make(map[string]time.Time)
	tc.accessLog = make(map[string][]time.Time)
}

// Delete removes a specific entry from the tier cache
func (tc *TierCache) Delete(key string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	
	delete(tc.data, key)
	delete(tc.ttlIndex, key)
	delete(tc.accessLog, key)
}

// Keys returns all keys matching a pattern
func (tc *TierCache) Keys(pattern string) []string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	
	var keys []string
	for key := range tc.data {
		if matchPatternSimple(key, pattern) {
			keys = append(keys, key)
		}
	}
	return keys
}

// Clear methods for specialized caches

// Clear removes all entries from the relationship cache
func (rc *RelationshipCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	rc.dependencies = make(map[string][]string)
	rc.parentChild = make(map[string][]string)
	rc.crossProvider = make(map[string][]string)
	rc.networkTopology = make(map[string]*NetworkNode)
	rc.iamRelations = make(map[string]*IAMRelation)
}

// Clear removes all entries from the aggregation cache
func (ac *AggregationCache) Clear() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	
	ac.providerSummaries = make(map[string]*ProviderSummary)
	ac.regionalDistribution = make(map[string]*RegionalDistribution)
	ac.typeCategories = make(map[string]*TypeCategory)
	ac.driftPatterns = make(map[string]*DriftPattern)
	ac.complianceScores = make(map[string]*ComplianceScore)
	ac.costAnalysis = make(map[string]*CostAnalysis)
}

// Clear removes all entries from the visualization cache
func (vc *VisualizationCache) Clear() {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	
	vc.graphLayouts = make(map[string]*GraphLayout)
	vc.hierarchyTrees = make(map[string]*HierarchyTree)
	vc.heatMapData = make(map[string]*HeatMapData)
	vc.timeSeriesData = make(map[string]*TimeSeriesData)
	vc.sankeyData = make(map[string]*SankeyData)
}

// Clear removes all entries from the search index cache
func (sic *SearchIndexCache) Clear() {
	sic.mu.Lock()
	defer sic.mu.Unlock()
	
	sic.fullTextIndex = make(map[string][]string)
	sic.facetIndex = make(map[string]map[string][]string)
	if sic.fuzzyIndex != nil {
		sic.fuzzyIndex.trigrams = make(map[string][]string)
	}
	if sic.suggestions != nil {
		sic.suggestions = &SuggestionEngine{}
	}
}

// Helper function for simple pattern matching
func matchPatternSimple(key, pattern string) bool {
	if pattern == "*" || pattern == "" {
		return true
	}
	
	// Check for prefix match with *
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}
	
	// Check for suffix match with *
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(key) >= len(suffix) && key[len(key)-len(suffix):] == suffix
	}
	
	// Exact match
	return key == pattern
}

// Additional helper methods can be added here as needed