package api

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"github.com/catherinevee/driftmgr/internal/api/handlers/discovery"
	apimodels "github.com/catherinevee/driftmgr/internal/api/models"
	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// CacheIntegration manages the integration between API and enhanced cache
type CacheIntegration struct {
	enhancedCache    *cache.EnhancedCache
	discoveryHub     *discovery.DiscoveryHub
	enrichmentQueue  chan *apimodels.Resource
	mu               sync.RWMutex
	initialized      bool
}

// NewCacheIntegration creates a new cache integration
func NewCacheIntegration(hub *discovery.DiscoveryHub) *CacheIntegration {
	ci := &CacheIntegration{
		enhancedCache:   cache.NewEnhancedCache(),
		discoveryHub:    hub,
		enrichmentQueue: make(chan *apimodels.Resource, 1000),
	}
	
	// Start background workers
	go ci.enrichmentWorker()
	go ci.cacheMaintenanceWorker()
	
	return ci
}

// Initialize sets up the cache with existing data
func (ci *CacheIntegration) Initialize(ctx context.Context) error {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	
	if ci.initialized {
		return nil
	}
	
	// Load existing resources into cache
	resources := ci.discoveryHub.GetAllResources()
	
	for _, resource := range resources {
		modelResource := ConvertAPIToModelResource(&resource)
		if err := ci.cacheResource(modelResource); err != nil {
			fmt.Printf("Failed to cache resource %s: %v\n", resource.ID, err)
		}
	}
	
	// Load existing drifts
	// TODO: Fix type mismatch between discovery.DriftRecord and api.DriftRecord
	// drifts := ci.discoveryHub.GetDriftRecords()
	// for _, drift := range drifts {
	// 	if err := ci.cacheDrift(drift); err != nil {
	// 		fmt.Printf("Failed to cache drift %s: %v\n", drift.ID, err)
	// 	}
	// }
	
	ci.initialized = true
	return nil
}

// GetEnrichedResource retrieves an enriched resource from cache
func (ci *CacheIntegration) GetEnrichedResource(resourceID string) (*cache.CacheEntry, error) {
	// Try cache first
	entry, found := ci.enhancedCache.Get(resourceID)
	if found {
		return entry, nil
	}
	
	// If not in cache, fetch and enrich
	resource, found := ci.discoveryHub.GetResourceByID(resourceID)
	if !found || resource == nil {
		return nil, fmt.Errorf("resource not found: %s", resourceID)
	}
	
	// Cache and enrich
	modelResource := ConvertAPIToModelResource(resource)
	if err := ci.cacheResource(modelResource); err != nil {
		return nil, err
	}
	
	// Retrieve enriched version
	entry, _ = ci.enhancedCache.Get(resourceID)
	return entry, nil
}

// GetEnrichedStats returns enriched statistics
func (ci *CacheIntegration) GetEnrichedStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	// Get basic stats from discovery hub
	resources := ci.discoveryHub.GetAllResources()
	drifts := ci.discoveryHub.GetDriftRecords()
	
	// Basic counts
	stats["total_resources"] = len(resources)
	stats["total_drifts"] = len(drifts)
	
	// Get aggregated data from cache
	if providerSummary, found := ci.enhancedCache.GetAggregation("provider_summary"); found {
		stats["provider_summary"] = providerSummary
	}
	
	if regionalDist, found := ci.enhancedCache.GetAggregation("regional_distribution"); found {
		stats["regional_distribution"] = regionalDist
	}
	
	if costAnalysis, found := ci.enhancedCache.GetAggregation("cost_analysis"); found {
		stats["cost_analysis"] = costAnalysis
	}
	
	if complianceScores, found := ci.enhancedCache.GetAggregation("compliance_scores"); found {
		stats["compliance_scores"] = complianceScores
	}
	
	if driftPatterns, found := ci.enhancedCache.GetAggregation("drift_patterns"); found {
		stats["drift_patterns"] = driftPatterns
	}
	
	// Get cache analytics
	analytics := ci.enhancedCache.GetAnalytics()
	stats["cache_performance"] = map[string]interface{}{
		"hit_rate":     analytics.GetHitRate(),
		"miss_rate":    analytics.GetMissRate(),
		"avg_latency":  analytics.GetAverageLatency(),
		"cached_items": analytics.GetCachedItemCount(),
	}
	
	// Calculate enrichment coverage
	enrichedCount := 0
	for _, resource := range resources {
		if entry, found := ci.enhancedCache.Get(resource.ID); found {
			if len(entry.Enrichments) > 0 {
				enrichedCount++
			}
		}
	}
	stats["enrichment_coverage"] = float64(enrichedCount) / float64(len(resources)) * 100
	
	return stats
}

// GetVisualizationData returns pre-computed visualization data
func (ci *CacheIntegration) GetVisualizationData(vizType string, params map[string]interface{}) (interface{}, error) {
	data, found := ci.enhancedCache.GetVisualizationData(vizType, params)
	if !found {
		// Generate visualization data on demand
		data = ci.generateVisualizationData(vizType, params)
		// Cache for future use
		ci.cacheVisualizationData(vizType, params, data)
	}
	
	return data, nil
}

// SearchResources performs an enriched search
func (ci *CacheIntegration) SearchResources(query string, filters map[string]interface{}) ([]*cache.CacheEntry, error) {
	return ci.enhancedCache.Search(query, filters)
}

// GetRelatedResources returns resources related to a given resource
func (ci *CacheIntegration) GetRelatedResources(resourceID string) ([]apimodels.Resource, error) {
	entry, relationships, found := ci.enhancedCache.GetWithRelationships(resourceID)
	if !found {
		return nil, fmt.Errorf("resource not found: %s", resourceID)
	}
	
	relatedResources := make([]apimodels.Resource, 0)
	for _, rel := range relationships {
		if relEntry, found := ci.enhancedCache.Get(rel.TargetKey); found {
			// Convert cache entry back to resource
			if resource, ok := relEntry.Value.(*models.Resource); ok {
				relatedResources = append(relatedResources, ConvertModelToAPIResource(resource))
			}
		}
	}
	
	// Add entry's resource itself
	if resource, ok := entry.Value.(*models.Resource); ok {
		relatedResources = append(relatedResources, ConvertModelToAPIResource(resource))
	}
	
	return relatedResources, nil
}

// InvalidateResource invalidates a specific resource in cache
func (ci *CacheIntegration) InvalidateResource(resourceID string) {
	ci.enhancedCache.InvalidatePartial(resourceID)
}

// InvalidateProvider invalidates all resources from a provider
func (ci *CacheIntegration) InvalidateProvider(provider string) {
	pattern := fmt.Sprintf("%s:", provider)
	ci.enhancedCache.InvalidatePartial(pattern)
}

// Private methods

func (ci *CacheIntegration) cacheResource(resource *models.Resource) error {
	// Create cache key
	key := fmt.Sprintf("%s:%s:%s", resource.Provider, resource.Type, resource.ID)
	
	// Create metadata
	metadata := &cache.EntryMetadata{
		Source:         "discovery",
		Provider:       resource.Provider,
		Region:         resource.Region,
		ResourceType:   resource.Type,
		DataType:       "resource",
		CollectionTime: time.Now(),
		Custom: map[string]string{
			"name":   resource.Name,
			"status": resource.Status,
		},
	}
	
	// Set in cache (will trigger enrichment)
	return ci.enhancedCache.Set(key, resource, metadata)
}

func (ci *CacheIntegration) cacheDrift(drift *DriftRecord) error {
	// Create cache key
	key := fmt.Sprintf("drift:%s:%s", drift.Provider, drift.ID)
	
	// Create metadata
	metadata := &cache.EntryMetadata{
		Source:         "drift_detection",
		Provider:       drift.Provider,
		Region:         drift.Region,
		ResourceType:   drift.ResourceType,
		DataType:       "drift",
		CollectionTime: drift.DetectedAt,
		Custom: map[string]string{
			"severity":    drift.Severity,
			"drift_type":  drift.DriftType,
			"resource_id": drift.ResourceID,
		},
	}
	
	// Set in cache
	return ci.enhancedCache.Set(key, drift, metadata)
}

func (ci *CacheIntegration) enrichmentWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case resource := <-ci.enrichmentQueue:
			// Process resource enrichment asynchronously
			go func(r *apimodels.Resource) {
				// Convert API resource to core model for caching
				coreResource := ConvertAPIToModelResource(r)
				if err := ci.cacheResource(coreResource); err != nil {
					fmt.Printf("Enrichment failed for %s: %v\n", r.ID, err)
				}
			}(resource)
			
		case <-ticker.C:
			// Periodic re-enrichment of stale data
			ci.reEnrichStaleData()
		}
	}
}

func (ci *CacheIntegration) cacheMaintenanceWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		// Update aggregations
		ci.updateAggregations()
		
		// Update visualization data
		ci.updateVisualizationCache()
		
		// Clean up expired entries
		ci.cleanupExpiredEntries()
	}
}

func (ci *CacheIntegration) updateAggregations() {
	apiResources := ci.discoveryHub.GetAllResources()
	
	// Convert to model resources
	resources := make([]*models.Resource, len(apiResources))
	for i, r := range apiResources {
		resources[i] = ConvertAPIToModelResource(&r)
	}
	
	// Provider summaries
	providerSummaries := ci.calculateProviderSummaries(resources)
	ci.SetAggregation("provider_summary", providerSummaries)
	
	// Regional distribution
	regionalDist := ci.calculateRegionalDistribution(resources)
	ci.SetAggregation("regional_distribution", regionalDist)
	
	// Cost analysis
	costAnalysis := ci.calculateCostAnalysis(resources)
	ci.SetAggregation("cost_analysis", costAnalysis)
	
	// Compliance scores
	complianceScores := ci.calculateComplianceScores(resources)
	ci.SetAggregation("compliance_scores", complianceScores)
	
	// Drift patterns
	discoveryDrifts := ci.discoveryHub.GetDriftRecords()
	// Convert discovery.DriftRecord to api.DriftRecord
	apiDrifts := make([]*DriftRecord, 0, len(discoveryDrifts))
	for _, dd := range discoveryDrifts {
		apiDrift := &DriftRecord{
			ID:           dd.ID,
			ResourceID:   dd.ResourceID,
			ResourceType: dd.ResourceType,
			Provider:     dd.Provider,
			Region:       dd.Region,
			DriftType:    dd.DriftType,
			Severity:     dd.Severity,
			DetectedAt:   dd.DetectedAt,
			Status:       dd.Status,
		}
		apiDrifts = append(apiDrifts, apiDrift)
	}
	driftPatterns := ci.analyzeDriftPatterns(apiDrifts)
	ci.SetAggregation("drift_patterns", driftPatterns)
}

func (ci *CacheIntegration) calculateProviderSummaries(resources []*models.Resource) map[string]interface{} {
	summaries := make(map[string]interface{})
	
	// Group by provider
	byProvider := make(map[string][]*models.Resource)
	for _, r := range resources {
		byProvider[r.Provider] = append(byProvider[r.Provider], r)
	}
	
	for provider, providerResources := range byProvider {
		driftCount := 0
		totalCost := 0.0
		
		for _, r := range providerResources {
			// Check if resource has drift
			if ci.discoveryHub.HasDrift(r.ID) {
				driftCount++
			}
			
			// Get cost from cache
			if entry, found := ci.enhancedCache.Get(r.ID); found {
				if cost, ok := entry.Metrics["monthly_cost"]; ok {
					totalCost += cost
				}
			}
		}
		
		summaries[provider] = map[string]interface{}{
			"total_resources":   len(providerResources),
			"drifted_resources": driftCount,
			"compliance_rate":   float64(len(providerResources)-driftCount) / float64(len(providerResources)) * 100,
			"estimated_cost":    totalCost,
		}
	}
	
	return summaries
}

func (ci *CacheIntegration) calculateRegionalDistribution(resources []*models.Resource) map[string]interface{} {
	distribution := make(map[string]interface{})
	
	// Group by region
	byRegion := make(map[string][]*models.Resource)
	for _, r := range resources {
		region := r.Region
		if region == "" {
			region = "global"
		}
		byRegion[region] = append(byRegion[region], r)
	}
	
	for region, regionResources := range byRegion {
		// Count resource types
		typeCount := make(map[string]int)
		for _, r := range regionResources {
			typeCount[r.Type]++
		}
		
		distribution[region] = map[string]interface{}{
			"resource_count": len(regionResources),
			"resource_types": typeCount,
		}
	}
	
	return distribution
}

func (ci *CacheIntegration) calculateCostAnalysis(resources []*models.Resource) map[string]interface{} {
	totalCost := 0.0
	costByProvider := make(map[string]float64)
	costByRegion := make(map[string]float64)
	costByType := make(map[string]float64)
	unusedResources := make([]string, 0)
	
	for _, r := range resources {
		if entry, found := ci.enhancedCache.Get(r.ID); found {
			if cost, ok := entry.Metrics["monthly_cost"]; ok {
				totalCost += cost
				costByProvider[r.Provider] += cost
				
				region := r.Region
				if region == "" {
					region = "global"
				}
				costByRegion[region] += cost
				costByType[r.Type] += cost
				
				// Check if resource is unused
				if waste, ok := entry.Metrics["waste_percentage"]; ok && waste > 80 {
					unusedResources = append(unusedResources, r.ID)
				}
			}
		}
	}
	
	return map[string]interface{}{
		"total_cost":        totalCost,
		"cost_by_provider":  costByProvider,
		"cost_by_region":    costByRegion,
		"cost_by_type":      costByType,
		"unused_resources":  unusedResources,
		"potential_savings": totalCost * 0.2, // Estimate 20% savings potential
	}
}

func (ci *CacheIntegration) calculateComplianceScores(resources []*models.Resource) map[string]interface{} {
	scores := make(map[string]interface{})
	frameworks := []string{"SOC2", "HIPAA", "PCI-DSS", "ISO27001"}
	
	for _, framework := range frameworks {
		violations := 0
		critical := 0
		high := 0
		
		for _, r := range resources {
			if entry, found := ci.enhancedCache.Get(r.ID); found {
				if vList, ok := entry.Enrichments["compliance_violations"].([]interface{}); ok {
					for _, v := range vList {
						if violation, ok := v.(map[string]interface{}); ok {
							if isApplicable(violation, framework) {
								violations++
								severity := violation["severity"].(string)
								if severity == "critical" {
									critical++
								} else if severity == "high" {
									high++
								}
							}
						}
					}
				}
			}
		}
		
		score := 100.0
		if len(resources) > 0 {
			score = float64(len(resources)-violations) / float64(len(resources)) * 100
		}
		
		scores[framework] = map[string]interface{}{
			"score":      score,
			"violations": violations,
			"critical":   critical,
			"high":       high,
		}
	}
	
	return scores
}

func (ci *CacheIntegration) analyzeDriftPatterns(drifts []*DriftRecord) map[string]interface{} {
	patterns := make(map[string]interface{})
	
	// Group drifts by type
	byType := make(map[string][]*DriftRecord)
	for _, d := range drifts {
		byType[d.DriftType] = append(byType[d.DriftType], d)
	}
	
	// Analyze each pattern
	for driftType, typeDrifts := range byType {
		resources := make([]string, 0)
		for _, d := range typeDrifts {
			resources = append(resources, d.ResourceID)
		}
		
		patterns[driftType] = map[string]interface{}{
			"frequency":     len(typeDrifts),
			"resources":     resources,
			"last_detected": typeDrifts[0].DetectedAt,
		}
	}
	
	return patterns
}

func (ci *CacheIntegration) generateVisualizationData(vizType string, params map[string]interface{}) interface{} {
	switch vizType {
	case "dependency_graph":
		return ci.generateDependencyGraph()
	case "cost_treemap":
		return ci.generateCostTreeMap()
	case "drift_timeline":
		return ci.generateDriftTimeline()
	case "compliance_heatmap":
		return ci.generateComplianceHeatMap()
	default:
		return nil
	}
}

func (ci *CacheIntegration) generateDependencyGraph() interface{} {
	// Get relationship data from cache
	resources := ci.discoveryHub.GetAllResources()
	nodes := make([]interface{}, 0)
	edges := make([]interface{}, 0)
	
	for _, r := range resources {
		// Add node
		nodes = append(nodes, map[string]interface{}{
			"id":    r.ID,
			"label": r.Name,
			"type":  r.Type,
		})
		
		// Get relationships
		if _, relationships, found := ci.enhancedCache.GetWithRelationships(r.ID); found {
			for _, rel := range relationships {
				edges = append(edges, map[string]interface{}{
					"source": r.ID,
					"target": rel.TargetKey,
					"type":   rel.Type,
				})
			}
		}
	}
	
	return map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}
}

func (ci *CacheIntegration) generateCostTreeMap() interface{} {
	resources := ci.discoveryHub.GetAllResources()
	data := make([]interface{}, 0)
	
	// Group by provider and type
	byProvider := make(map[string]map[string]float64)
	
	for _, r := range resources {
		if entry, found := ci.enhancedCache.Get(r.ID); found {
			if cost, ok := entry.Metrics["monthly_cost"]; ok && cost > 0 {
				if byProvider[r.Provider] == nil {
					byProvider[r.Provider] = make(map[string]float64)
				}
				byProvider[r.Provider][r.Type] += cost
			}
		}
	}
	
	for provider, types := range byProvider {
		children := make([]interface{}, 0)
		for resourceType, cost := range types {
			children = append(children, map[string]interface{}{
				"name":  resourceType,
				"value": cost,
			})
		}
		
		data = append(data, map[string]interface{}{
			"name":     provider,
			"children": children,
		})
	}
	
	return map[string]interface{}{
		"name":     "Total Cost",
		"children": data,
	}
}

func (ci *CacheIntegration) generateDriftTimeline() interface{} {
	drifts := ci.discoveryHub.GetDriftRecords()
	timeline := make([]interface{}, 0)
	
	for _, d := range drifts {
		timeline = append(timeline, map[string]interface{}{
			"time":        d.DetectedAt,
			"resource_id": d.ResourceID,
			"drift_type":  d.DriftType,
			"severity":    d.Severity,
		})
	}
	
	return timeline
}

func (ci *CacheIntegration) generateComplianceHeatMap() interface{} {
	resources := ci.discoveryHub.GetAllResources()
	
	// Create matrix: providers x compliance rules
	providers := []string{"aws", "azure", "gcp"}
	rules := []string{"encryption", "backup", "tagging", "access_control"}
	
	matrix := make([][]float64, len(providers))
	for i := range matrix {
		matrix[i] = make([]float64, len(rules))
	}
	
	// Calculate compliance for each cell
	for i, provider := range providers {
		for j, rule := range rules {
			compliantCount := 0
			totalCount := 0
			
			for _, r := range resources {
				if r.Provider == provider {
					totalCount++
					if entry, found := ci.enhancedCache.Get(r.ID); found {
						if isCompliant(entry, rule) {
							compliantCount++
						}
					}
				}
			}
			
			if totalCount > 0 {
				matrix[i][j] = float64(compliantCount) / float64(totalCount) * 100
			}
		}
	}
	
	return map[string]interface{}{
		"matrix":   matrix,
		"x_labels": rules,
		"y_labels": providers,
	}
}

func (ci *CacheIntegration) cacheVisualizationData(vizType string, params map[string]interface{}, data interface{}) {
	// This would be implemented to cache visualization data
	// For now, just a placeholder
}

func (ci *CacheIntegration) reEnrichStaleData() {
	// Re-enrich data older than 1 hour
	// This would check cache entries and re-enrich stale ones
}

func (ci *CacheIntegration) cleanupExpiredEntries() {
	// Clean up expired cache entries
	// This would be implemented based on TTL
}

// Helper functions

func isApplicable(violation map[string]interface{}, framework string) bool {
	// Check if a compliance violation applies to a framework
	// This would have actual logic based on violation type
	return true
}

func isCompliant(entry *cache.CacheEntry, rule string) bool {
	// Check if a resource is compliant with a specific rule
	if violations, ok := entry.Enrichments["compliance_violations"].([]interface{}); ok {
		for _, v := range violations {
			if violation, ok := v.(map[string]interface{}); ok {
				if violation["rule"] == rule {
					return false
				}
			}
		}
	}
	return true
}

// These cache methods are now in enhanced_cache_methods.go

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

// ConvertModelToAPIResource converts a models.Resource to an API Resource
func ConvertModelToAPIResource(r *models.Resource) apimodels.Resource {
	// Convert tags if needed
	var tags map[string]string
	if r.Tags != nil {
		switch t := r.Tags.(type) {
		case map[string]string:
			tags = t
		case map[string]interface{}:
			tags = make(map[string]string)
			for k, v := range t {
				if str, ok := v.(string); ok {
					tags[k] = str
				}
			}
		default:
			tags = make(map[string]string)
		}
	} else {
		tags = make(map[string]string)
	}
	
	return apimodels.Resource{
		ID:         r.ID,
		Name:       r.Name,
		Type:       r.Type,
		Provider:   r.Provider,
		Region:     r.Region,
		Status:     r.Status,
		Tags:       tags,
		ModifiedAt: r.LastModified,
		Properties: r.Properties,
	}
}

// ConvertAPIToModelResource converts an API Resource to models.Resource
func ConvertAPIToModelResource(r *apimodels.Resource) *models.Resource {
	return &models.Resource{
		ID:           r.ID,
		Name:         r.Name,
		Type:         r.Type,
		Provider:     r.Provider,
		Region:       r.Region,
		Status:       r.Status,
		Tags:         r.Tags,
		LastModified: r.ModifiedAt,
		Properties:   r.Properties,
	}
}

// SetAggregation sets an aggregation in the cache
func (ci *CacheIntegration) SetAggregation(key string, data interface{}) {
	if aggCache := ci.enhancedCache.GetAggregationCache(); aggCache != nil {
		aggCache.Set(key, data)
	}
}

// updateVisualizationCache updates the visualization cache (placeholder)
func (ci *CacheIntegration) updateVisualizationCache() {
	// TODO: Implement visualization cache updates
}