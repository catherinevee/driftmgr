package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	
	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/gorilla/mux"
)

// RegisterEnrichedHandlers registers handlers for enriched data endpoints
func (s *Server) RegisterEnrichedHandlers() {
	// Enriched data endpoints
	s.router.HandleFunc("/api/enriched/stats", s.handleEnrichedStats).Methods("GET")
	s.router.HandleFunc("/api/enriched/resources/{id}", s.handleEnrichedResource).Methods("GET")
	s.router.HandleFunc("/api/enriched/search", s.handleEnrichedSearch).Methods("GET", "POST")
	s.router.HandleFunc("/api/enriched/related/{id}", s.handleRelatedResources).Methods("GET")
	s.router.HandleFunc("/api/enriched/visualization/{type}", s.handleVisualizationData).Methods("GET")
	s.router.HandleFunc("/api/enriched/aggregations/{type}", s.handleAggregations).Methods("GET")
	s.router.HandleFunc("/api/enriched/cost-analysis", s.handleCostAnalysis).Methods("GET")
	s.router.HandleFunc("/api/enriched/compliance", s.handleComplianceAnalysis).Methods("GET")
	s.router.HandleFunc("/api/enriched/drift-patterns", s.handleDriftPatterns).Methods("GET")
	s.router.HandleFunc("/api/enriched/recommendations/{id}", s.handleRecommendations).Methods("GET")
	s.router.HandleFunc("/api/enriched/impact-analysis/{id}", s.handleImpactAnalysis).Methods("GET")
}

// handleEnrichedStats returns enriched statistics
func (s *Server) handleEnrichedStats(w http.ResponseWriter, r *http.Request) {
	if s.cacheIntegration == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Cache integration not available")
		return
	}
	
	// Get enriched stats from cache integration
	stats := s.cacheIntegration.GetEnrichedStats()
	
	// Add additional computed stats
	if providerSummary, ok := stats["provider_summary"].(map[string]interface{}); ok {
		totalResources := 0
		driftedResources := 0
		activeProviders := 0
		
		for _, summary := range providerSummary {
			if s, ok := summary.(map[string]interface{}); ok {
				if total, ok := s["total_resources"].(int); ok {
					totalResources += total
				}
				if drifted, ok := s["drifted_resources"].(int); ok {
					driftedResources += drifted
				}
				activeProviders++
			}
		}
		
		stats["totalResources"] = totalResources
		stats["driftedResources"] = driftedResources
		stats["activeProviders"] = activeProviders
		stats["complianceRate"] = 100.0
		if totalResources > 0 {
			stats["complianceRate"] = float64(totalResources-driftedResources) / float64(totalResources) * 100
		}
	}
	
	// Add cache performance metrics
	if cachePerf, ok := stats["cache_performance"].(map[string]interface{}); ok {
		stats["cacheHitRate"] = cachePerf["hit_rate"]
		stats["cacheMissRate"] = cachePerf["miss_rate"]
		stats["cachedItems"] = cachePerf["cached_items"]
	}
	
	// Add timestamp
	stats["timestamp"] = fmt.Sprintf("%d", time.Now().Unix())
	stats["lastUpdated"] = time.Now().Format(time.RFC3339)
	
	s.respondJSON(w, http.StatusOK, stats)
}

// handleEnrichedResource returns enriched data for a specific resource
func (s *Server) handleEnrichedResource(w http.ResponseWriter, r *http.Request) {
	if s.cacheIntegration == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Cache integration not available")
		return
	}
	
	vars := mux.Vars(r)
	resourceID := vars["id"]
	
	entry, err := s.cacheIntegration.GetEnrichedResource(resourceID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}
	
	// Convert cache entry to response format
	response := map[string]interface{}{
		"resource_id":  resourceID,
		"value":        entry.Value,
		"metadata":     entry.Metadata,
		"enrichments":  entry.Enrichments,
		"metrics":      entry.Metrics,
		"tags":         entry.Tags,
		"confidence":   entry.ConfidenceScore,
		"last_updated": entry.UpdatedAt,
	}
	
	// Add relationships if available
	if _, relationships, found := s.cacheIntegration.enhancedCache.GetWithRelationships(resourceID); found {
		relData := make([]map[string]interface{}, 0)
		for _, rel := range relationships {
			relData = append(relData, map[string]interface{}{
				"type":      rel.Type,
				"direction": rel.Direction,
				"target":    rel.TargetKey,
				"strength":  rel.Strength,
			})
		}
		response["relationships"] = relData
	}
	
	s.respondJSON(w, http.StatusOK, response)
}

// handleEnrichedSearch handles search requests with enriched data
func (s *Server) handleEnrichedSearch(w http.ResponseWriter, r *http.Request) {
	if s.cacheIntegration == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Cache integration not available")
		return
	}
	
	var searchRequest struct {
		Query   string                 `json:"query"`
		Filters map[string]interface{} `json:"filters"`
	}
	
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&searchRequest); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
	} else {
		searchRequest.Query = r.URL.Query().Get("q")
		searchRequest.Filters = make(map[string]interface{})
		
		// Parse filters from query params
		for key, values := range r.URL.Query() {
			if key != "q" && len(values) > 0 {
				searchRequest.Filters[key] = values[0]
			}
		}
	}
	
	results, err := s.cacheIntegration.SearchResources(searchRequest.Query, searchRequest.Filters)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	// Convert results to response format
	response := map[string]interface{}{
		"query":   searchRequest.Query,
		"filters": searchRequest.Filters,
		"count":   len(results),
		"results": results,
	}
	
	s.respondJSON(w, http.StatusOK, response)
}

// handleRelatedResources returns resources related to a specific resource
func (s *Server) handleRelatedResources(w http.ResponseWriter, r *http.Request) {
	if s.cacheIntegration == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Cache integration not available")
		return
	}
	
	vars := mux.Vars(r)
	resourceID := vars["id"]
	
	related, err := s.cacheIntegration.GetRelatedResources(resourceID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}
	
	response := map[string]interface{}{
		"resource_id": resourceID,
		"related":     related,
		"count":       len(related),
	}
	
	s.respondJSON(w, http.StatusOK, response)
}

// handleVisualizationData returns pre-computed visualization data
func (s *Server) handleVisualizationData(w http.ResponseWriter, r *http.Request) {
	if s.cacheIntegration == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Cache integration not available")
		return
	}
	
	vars := mux.Vars(r)
	vizType := vars["type"]
	
	// Parse parameters from query string
	params := make(map[string]interface{})
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	
	data, err := s.cacheIntegration.GetVisualizationData(vizType, params)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	
	s.respondJSON(w, http.StatusOK, data)
}

// handleAggregations returns aggregated data
func (s *Server) handleAggregations(w http.ResponseWriter, r *http.Request) {
	if s.cacheIntegration == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Cache integration not available")
		return
	}
	
	vars := mux.Vars(r)
	aggType := vars["type"]
	
	data, found := s.cacheIntegration.enhancedCache.GetAggregation(aggType)
	if !found {
		s.respondError(w, http.StatusNotFound, "Aggregation not found")
		return
	}
	
	s.respondJSON(w, http.StatusOK, data)
}

// handleCostAnalysis returns cost analysis data
func (s *Server) handleCostAnalysis(w http.ResponseWriter, r *http.Request) {
	if s.cacheIntegration == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Cache integration not available")
		return
	}
	
	// Get cost analysis from cache
	costData, found := s.cacheIntegration.enhancedCache.GetAggregation("cost_analysis")
	if !found {
		s.respondError(w, http.StatusNotFound, "Cost analysis not available")
		return
	}
	
	// Add optimization recommendations
	resources := s.discoveryHub.GetAllResources()
	recommendations := make([]map[string]interface{}, 0)
	
	for _, resource := range resources {
		if entry, found := s.cacheIntegration.enhancedCache.Get(resource.ID); found {
			if recs, ok := entry.Enrichments["performance_recommendations"].([]string); ok && len(recs) > 0 {
				recommendations = append(recommendations, map[string]interface{}{
					"resource_id": resource.ID,
					"resource":    resource.Name,
					"type":        resource.Type,
					"provider":    resource.Provider,
					"suggestions": recs,
					"potential_savings": entry.Metrics["monthly_cost"] * 0.3, // Estimate 30% savings
				})
			}
		}
	}
	
	response := map[string]interface{}{
		"cost_analysis":    costData,
		"recommendations":  recommendations,
		"total_potential_savings": calculateTotalSavings(recommendations),
	}
	
	s.respondJSON(w, http.StatusOK, response)
}

// handleComplianceAnalysis returns compliance analysis data
func (s *Server) handleComplianceAnalysis(w http.ResponseWriter, r *http.Request) {
	if s.cacheIntegration == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Cache integration not available")
		return
	}
	
	// Get compliance scores from cache
	complianceData, found := s.cacheIntegration.enhancedCache.GetAggregation("compliance_scores")
	if !found {
		s.respondError(w, http.StatusNotFound, "Compliance analysis not available")
		return
	}
	
	// Get violations by resource
	resources := s.discoveryHub.GetAllResources()
	violations := make([]map[string]interface{}, 0)
	
	for _, resource := range resources {
		if entry, found := s.cacheIntegration.enhancedCache.Get(resource.ID); found {
			if vList, ok := entry.Enrichments["compliance_violations"].([]interface{}); ok && len(vList) > 0 {
				violations = append(violations, map[string]interface{}{
					"resource_id": resource.ID,
					"resource":    resource.Name,
					"type":        resource.Type,
					"provider":    resource.Provider,
					"violations":  vList,
					"risk_level":  entry.Enrichments["risk_level"],
				})
			}
		}
	}
	
	response := map[string]interface{}{
		"compliance_scores": complianceData,
		"violations":        violations,
		"violation_count":   len(violations),
		"frameworks":        []string{"SOC2", "HIPAA", "PCI-DSS", "ISO27001", "GDPR"},
	}
	
	s.respondJSON(w, http.StatusOK, response)
}

// handleDriftPatterns returns drift pattern analysis
func (s *Server) handleDriftPatterns(w http.ResponseWriter, r *http.Request) {
	if s.cacheIntegration == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Cache integration not available")
		return
	}
	
	// Get drift patterns from cache
	patterns, found := s.cacheIntegration.enhancedCache.GetAggregation("drift_patterns")
	if !found {
		s.respondError(w, http.StatusNotFound, "Drift patterns not available")
		return
	}
	
	// Add drift prediction for resources
	resources := s.discoveryHub.GetAllResources()
	predictions := make([]map[string]interface{}, 0)
	
	for _, resource := range resources {
		if entry, found := s.cacheIntegration.enhancedCache.Get(resource.ID); found {
			if likelihood, ok := entry.Enrichments["drift_likelihood"].(string); ok {
				if likelihood == "high" || likelihood == "medium" {
					predictions = append(predictions, map[string]interface{}{
						"resource_id": resource.ID,
						"resource":    resource.Name,
						"type":        resource.Type,
						"provider":    resource.Provider,
						"likelihood":  likelihood,
						"impact":      entry.Enrichments["drift_impact"],
						"prevention":  entry.Enrichments["drift_prevention"],
					})
				}
			}
		}
	}
	
	response := map[string]interface{}{
		"patterns":     patterns,
		"predictions":  predictions,
		"at_risk_count": len(predictions),
	}
	
	s.respondJSON(w, http.StatusOK, response)
}

// handleRecommendations returns recommendations for a specific resource
func (s *Server) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	if s.cacheIntegration == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Cache integration not available")
		return
	}
	
	vars := mux.Vars(r)
	resourceID := vars["id"]
	
	entry, err := s.cacheIntegration.GetEnrichedResource(resourceID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, err.Error())
		return
	}
	
	recommendations := map[string]interface{}{
		"resource_id": resourceID,
		"categories": make(map[string][]string),
	}
	
	// Collect all recommendations from enrichments
	if perf, ok := entry.Enrichments["performance_recommendations"].([]string); ok {
		recommendations["categories"].(map[string][]string)["performance"] = perf
	}
	
	if sec, ok := entry.Enrichments["security_recommendations"].([]string); ok {
		recommendations["categories"].(map[string][]string)["security"] = sec
	}
	
	if drift, ok := entry.Enrichments["drift_prevention"].([]string); ok {
		recommendations["categories"].(map[string][]string)["drift_prevention"] = drift
	}
	
	// Add cost optimization if applicable
	if score, ok := entry.Metrics["cost_optimization_score"]; ok && score < 70 {
		recommendations["categories"].(map[string][]string)["cost"] = []string{
			"Resource has low cost optimization score",
			"Consider rightsizing or reserved capacity",
		}
	}
	
	// Add compliance recommendations
	if violations, ok := entry.Enrichments["compliance_violations"].([]interface{}); ok && len(violations) > 0 {
		compRecs := []string{"Address compliance violations:"}
		for _, v := range violations {
			if violation, ok := v.(map[string]interface{}); ok {
				if msg, ok := violation["message"].(string); ok {
					compRecs = append(compRecs, "- "+msg)
				}
			}
		}
		recommendations["categories"].(map[string][]string)["compliance"] = compRecs
	}
	
	s.respondJSON(w, http.StatusOK, recommendations)
}

// handleImpactAnalysis returns impact analysis for a resource
func (s *Server) handleImpactAnalysis(w http.ResponseWriter, r *http.Request) {
	if s.cacheIntegration == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Cache integration not available")
		return
	}
	
	vars := mux.Vars(r)
	resourceID := vars["id"]
	
	// Get the resource and its relationships
	entry, relationships, found := s.cacheIntegration.enhancedCache.GetWithRelationships(resourceID)
	if !found {
		s.respondError(w, http.StatusNotFound, "Resource not found")
		return
	}
	
	// Calculate impact radius
	relationshipCache := s.cacheIntegration.enhancedCache.GetRelationshipCache()
	impactRadius := relationshipCache.GetImpactRadius(resourceID, 3)
	
	// Categorize impacts
	impacts := map[string]interface{}{
		"resource_id":      resourceID,
		"direct_impact":    make([]string, 0),
		"indirect_impact":  make([]string, 0),
		"total_affected":   len(impactRadius),
		"criticality":      entry.Metadata.Custom["criticality"],
		"estimated_downtime": "0 minutes",
	}
	
	// Separate direct and indirect impacts
	directDeps := make(map[string]bool)
	for _, rel := range relationships {
		if rel.Type == "dependency" || rel.Type == "parent_child" {
			directDeps[rel.TargetKey] = true
			impacts["direct_impact"] = append(impacts["direct_impact"].([]string), rel.TargetKey)
		}
	}
	
	for _, affected := range impactRadius {
		if !directDeps[affected] {
			impacts["indirect_impact"] = append(impacts["indirect_impact"].([]string), affected)
		}
	}
	
	// Estimate downtime based on criticality and dependencies
	if criticality, ok := entry.Metadata.Custom["criticality"]; ok {
		switch criticality {
		case "critical":
			impacts["estimated_downtime"] = fmt.Sprintf("%d minutes", len(impactRadius)*5)
		case "high":
			impacts["estimated_downtime"] = fmt.Sprintf("%d minutes", len(impactRadius)*3)
		case "medium":
			impacts["estimated_downtime"] = fmt.Sprintf("%d minutes", len(impactRadius)*2)
		default:
			impacts["estimated_downtime"] = fmt.Sprintf("%d minutes", len(impactRadius))
		}
	}
	
	// Add remediation suggestions
	impacts["remediation_priority"] = calculateRemediationPriority(entry, len(impactRadius))
	
	s.respondJSON(w, http.StatusOK, impacts)
}

// Helper functions

func calculateTotalSavings(recommendations []map[string]interface{}) float64 {
	total := 0.0
	for _, rec := range recommendations {
		if savings, ok := rec["potential_savings"].(float64); ok {
			total += savings
		}
	}
	return total
}

func calculateRemediationPriority(entry *cache.CacheEntry, impactCount int) string {
	score := 0
	
	// Factor in criticality
	if criticality, ok := entry.Metadata.Custom["criticality"]; ok {
		switch criticality {
		case "critical":
			score += 3
		case "high":
			score += 2
		case "medium":
			score += 1
		}
	}
	
	// Factor in impact radius
	if impactCount > 10 {
		score += 3
	} else if impactCount > 5 {
		score += 2
	} else if impactCount > 0 {
		score += 1
	}
	
	// Factor in compliance score
	if compScore, ok := entry.Metrics["compliance_score"]; ok && compScore < 50 {
		score += 2
	}
	
	if score >= 6 {
		return "immediate"
	} else if score >= 4 {
		return "high"
	} else if score >= 2 {
		return "medium"
	}
	return "low"
}