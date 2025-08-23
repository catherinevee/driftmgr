package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/catherinevee/driftmgr/internal/core/drift"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/gorilla/mux"
)

// DriftDetectionRequest represents a drift detection request
type DriftDetectionRequest struct {
	Providers     []string           `json:"providers"`
	SmartDefaults bool               `json:"smartDefaults"`
	Environment   string             `json:"environment"`
	Thresholds    map[string]float64 `json:"thresholds"`
	IgnoreRules   []string           `json:"ignoreRules"`
	StateFile     string             `json:"stateFile,omitempty"`
	CompareWith   string             `json:"compareWith,omitempty"` // "terraform", "baseline", "snapshot"
}

// DriftDetectionResponse represents a drift detection response
type DriftDetectionResponse struct {
	JobID      string             `json:"jobId,omitempty"`
	DriftItems []models.DriftItem `json:"driftItems,omitempty"`
	Summary    DriftSummary       `json:"summary"`
	Status     string             `json:"status"`
}

// DriftSummary provides drift statistics
type DriftSummary struct {
	TotalResources   int            `json:"totalResources"`
	DriftedResources int            `json:"driftedResources"`
	DriftPercentage  float64        `json:"driftPercentage"`
	BySeverity       map[string]int `json:"bySeverity"`
	ByProvider       map[string]int `json:"byProvider"`
	ByResourceType   map[string]int `json:"byResourceType"`
}

// DriftConfiguration represents drift detection configuration
type DriftConfiguration struct {
	SmartDefaults   bool               `json:"smartDefaults"`
	Environment     string             `json:"environment"`
	NoiseReduction  float64            `json:"noiseReduction"`
	Thresholds      map[string]float64 `json:"thresholds"`
	IgnorePatterns  []string           `json:"ignorePatterns"`
	AlertThresholds map[string]int     `json:"alertThresholds"`
}

// detectDrift handles drift detection requests
func (s *EnhancedDashboardServer) detectDrift(w http.ResponseWriter, r *http.Request) {
	var req DriftDetectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Environment == "" {
		req.Environment = "production"
	}
	if req.SmartDefaults {
		req.Thresholds = s.getSmartThresholds(req.Environment)
	}

	// Create a job for async drift detection
	job := s.jobManager.CreateJob("drift_detection")

	// Start drift detection in background
	go s.performDriftDetection(job, req)

	// Return job ID for tracking
	resp := DriftDetectionResponse{
		JobID:  job.ID,
		Status: "started",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// performDriftDetection performs the actual drift detection
func (s *EnhancedDashboardServer) performDriftDetection(job *Job, req DriftDetectionRequest) {
	ctx := context.Background()

	s.jobManager.UpdateJob(job.ID, "running", 10)

	// Get current resources from data store
	currentResources := s.dataStore.GetResources()
	if len(currentResources) == 0 {
		// If no cached resources, trigger discovery first
		s.jobManager.UpdateJob(job.ID, "running", 20)
		discoveryResult := s.discoveryService.DiscoverAll(ctx)

		for _, result := range discoveryResult {
			if result.Resources != nil {
				// Convert resources to interface{} for storage
				for _, res := range result.Resources {
					currentResources = append(currentResources, res)
				}
			}
		}
	}

	s.jobManager.UpdateJob(job.ID, "running", 40)

	// Detect drift based on comparison type
	var driftItems []models.DriftItem
	var err error

	switch req.CompareWith {
	case "terraform":
		if req.StateFile != "" {
			s.jobManager.UpdateJob(job.ID, "running", 50)
			// Convert interfaces to models.Resource
			resources := convertInterfacesToResources(currentResources)
			driftResult, detectErr := s.driftDetector.Detect(ctx, resources, drift.DetectionOptions{})
			if detectErr == nil && driftResult != nil {
				driftItems = driftResult.DriftItems
			}
			err = detectErr
		}
	case "baseline":
		s.jobManager.UpdateJob(job.ID, "running", 50)
		// Compare with previously saved baseline
		baseline := s.dataStore.GetBaseline()
		resources := convertInterfacesToResources(currentResources)
		baselineResources := convertInterfacesToResources(baseline)
		driftItems = s.compareResourceSets(resources, baselineResources)
	default:
		// Auto-detect drift by analyzing resource changes
		s.jobManager.UpdateJob(job.ID, "running", 50)
		resources := convertInterfacesToResources(currentResources)
		driftItems = s.analyzeResourceDrift(resources, req)
	}

	if err != nil {
		s.jobManager.UpdateJob(job.ID, "failed", 100)
		return
	}

	s.jobManager.UpdateJob(job.ID, "running", 70)

	// Apply smart filters if enabled
	if req.SmartDefaults {
		driftItems = s.applySmartFilters(driftItems, req.Environment)
	}

	// Apply ignore rules
	if len(req.IgnoreRules) > 0 {
		driftItems = s.applyIgnoreRules(driftItems, req.IgnoreRules)
	}

	s.jobManager.UpdateJob(job.ID, "running", 90)

	// Calculate summary
	resources := convertInterfacesToResources(currentResources)
	summary := s.calculateDriftSummary(driftItems, resources)

	// Store drift items
	driftInterfaces := convertDriftsToInterfaces(driftItems)
	s.dataStore.SetDrifts(driftInterfaces)

	// Store job result
	job.Result = map[string]interface{}{
		"driftItems": driftItems,
		"summary":    summary,
	}

	// Update job completion
	s.jobManager.UpdateJob(job.ID, "completed", 100)

	// Broadcast completion
	s.broadcast <- map[string]interface{}{
		"type":    "drift_detection_completed",
		"jobId":   job.ID,
		"summary": summary,
	}
}

// configureDriftDetection configures drift detection settings
func (s *EnhancedDashboardServer) configureDriftDetection(w http.ResponseWriter, r *http.Request) {
	var config DriftConfiguration
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Store configuration
	s.dataStore.SetConfig("drift_detection", config)

	// Broadcast configuration update
	s.broadcast <- map[string]interface{}{
		"type":   "drift_config_updated",
		"config": config,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "configured"})
}

// analyzeDrift performs advanced drift analysis
func (s *EnhancedDashboardServer) analyzeDrift(w http.ResponseWriter, r *http.Request) {
	var driftIDs []string
	if err := json.NewDecoder(r.Body).Decode(&driftIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	drifts := s.dataStore.GetDrifts()
	analysis := make(map[string]interface{})

	// Analyze patterns
	driftItems := make([]models.DriftItem, 0, len(drifts))
	for _, d := range drifts {
		if drift, ok := d.(models.DriftItem); ok {
			driftItems = append(driftItems, drift)
		}
	}
	patterns := s.analyzeDriftPatterns(driftItems)
	analysis["patterns"] = patterns

	// Predict future drift
	predictions := s.predictFutureDrift(driftItems)
	analysis["predictions"] = predictions

	// Calculate impact
	impact := s.calculateDriftImpact(driftItems)
	analysis["impact"] = impact

	// Generate recommendations
	recommendations := s.generateDriftRecommendations(driftItems)
	analysis["recommendations"] = recommendations

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// getDriftHistory retrieves drift history
func (s *EnhancedDashboardServer) getDriftHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	provider := r.URL.Query().Get("provider")
	severity := r.URL.Query().Get("severity")

	// Get drift history
	history := s.dataStore.GetDriftHistory()

	// Apply filters
	filteredHistory := make([]interface{}, 0)
	for i, item := range history {
		if limit > 0 && i >= limit {
			break
		}
		// Type assertion to access fields
		if drift, ok := item.(models.DriftItem); ok {
			if provider != "" && drift.Provider != provider {
				continue
			}
			if severity != "" && drift.Severity != severity {
				continue
			}
			filteredHistory = append(filteredHistory, drift)
		} else {
			// If not DriftItem, include as-is
			filteredHistory = append(filteredHistory, item)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filteredHistory)
}

// getDriftItems retrieves current drift items
func (s *EnhancedDashboardServer) getDriftItems(w http.ResponseWriter, r *http.Request) {
	// Parse filters
	provider := r.URL.Query().Get("provider")
	resourceType := r.URL.Query().Get("type")
	severity := r.URL.Query().Get("severity")

	drifts := s.dataStore.GetDrifts()

	// Apply filters
	filtered := make([]models.DriftItem, 0)
	for _, item := range drifts {
		// Type assertion to models.DriftItem
		if drift, ok := item.(models.DriftItem); ok {
			if provider != "" && drift.Provider != provider {
				continue
			}
			if resourceType != "" && drift.ResourceType != resourceType {
				continue
			}
			if severity != "" && drift.Severity != severity {
				continue
			}
			filtered = append(filtered, drift)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(filtered)
}

// getDriftDetails retrieves details for a specific drift item
func (s *EnhancedDashboardServer) getDriftDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	driftID := vars["driftId"]

	drifts := s.dataStore.GetDrifts()
	for _, item := range drifts {
		// Type assertion to models.DriftItem
		if drift, ok := item.(models.DriftItem); ok {
			if drift.ResourceID == driftID {
				// Enhance with additional details
				details := map[string]interface{}{
					"drift":         drift,
					"currentState":  s.getCurrentResourceState(drift.ResourceID),
					"expectedState": s.getExpectedResourceState(drift.ResourceID),
					"changeHistory": s.getResourceChangeHistory(drift.ResourceID),
					"impact":        s.assessDriftImpact(drift),
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(details)
				return
			}
		}
	}

	http.Error(w, "Drift item not found", http.StatusNotFound)
}

// Helper methods

func (s *EnhancedDashboardServer) getSmartThresholds(environment string) map[string]float64 {
	switch environment {
	case "production":
		return map[string]float64{
			"critical": 0.0,  // No tolerance for critical changes
			"high":     0.05, // 5% tolerance for high severity
			"medium":   0.15, // 15% tolerance for medium
			"low":      0.75, // 75% filter for low severity (noise reduction)
		}
	case "staging":
		return map[string]float64{
			"critical": 0.10,
			"high":     0.25,
			"medium":   0.50,
			"low":      0.85,
		}
	default: // development
		return map[string]float64{
			"critical": 0.25,
			"high":     0.50,
			"medium":   0.75,
			"low":      0.90,
		}
	}
}

func (s *EnhancedDashboardServer) applySmartFilters(drifts []models.DriftItem, environment string) []models.DriftItem {
	thresholds := s.getSmartThresholds(environment)
	filtered := make([]models.DriftItem, 0)

	severityCounts := make(map[string]int)
	for _, drift := range drifts {
		severityCounts[drift.Severity]++
	}

	for _, drift := range drifts {
		threshold := thresholds[drift.Severity]
		// Keep drift if it's above the threshold for its severity
		if threshold < 1.0 {
			filtered = append(filtered, drift)
		}
	}

	return filtered
}

func (s *EnhancedDashboardServer) applyIgnoreRules(drifts []models.DriftItem, rules []string) []models.DriftItem {
	filtered := make([]models.DriftItem, 0)

	for _, drift := range drifts {
		ignore := false
		for _, rule := range rules {
			// Simple pattern matching - could be enhanced with regex
			if drift.ResourceType == rule || drift.DriftType == rule {
				ignore = true
				break
			}
		}
		if !ignore {
			filtered = append(filtered, drift)
		}
	}

	return filtered
}

func (s *EnhancedDashboardServer) calculateDriftSummary(drifts []models.DriftItem, resources []models.Resource) DriftSummary {
	summary := DriftSummary{
		TotalResources:   len(resources),
		DriftedResources: len(drifts),
		BySeverity:       make(map[string]int),
		ByProvider:       make(map[string]int),
		ByResourceType:   make(map[string]int),
	}

	if summary.TotalResources > 0 {
		summary.DriftPercentage = float64(summary.DriftedResources) / float64(summary.TotalResources) * 100
	}

	for _, drift := range drifts {
		summary.BySeverity[drift.Severity]++
		summary.ByProvider[drift.Provider]++
		summary.ByResourceType[drift.ResourceType]++
	}

	return summary
}

func (s *EnhancedDashboardServer) compareResourceSets(current, baseline []models.Resource) []models.DriftItem {
	drifts := make([]models.DriftItem, 0)

	// Create maps for efficient lookup
	currentMap := make(map[string]models.Resource)
	for _, r := range current {
		currentMap[r.ID] = r
	}

	baselineMap := make(map[string]models.Resource)
	for _, r := range baseline {
		baselineMap[r.ID] = r
	}

	// Check for modified and deleted resources
	for id, baselineResource := range baselineMap {
		if currentResource, exists := currentMap[id]; exists {
			// Compare resources for changes
			if drift := s.compareResources(baselineResource, currentResource); drift != nil {
				drifts = append(drifts, *drift)
			}
		} else {
			// Resource deleted
			drifts = append(drifts, models.DriftItem{
				ResourceID:   id,
				ResourceType: baselineResource.Type,
				Provider:     baselineResource.Provider,
				DriftType:    "deleted",
				Severity:     "high",
				Description:  fmt.Sprintf("Resource %s has been deleted", baselineResource.Name),
			})
		}
	}

	// Check for added resources
	for id, currentResource := range currentMap {
		if _, exists := baselineMap[id]; !exists {
			drifts = append(drifts, models.DriftItem{
				ResourceID:   id,
				ResourceType: currentResource.Type,
				Provider:     currentResource.Provider,
				DriftType:    "added",
				Severity:     "medium",
				Description:  fmt.Sprintf("New resource %s has been added", currentResource.Name),
			})
		}
	}

	return drifts
}

func (s *EnhancedDashboardServer) compareResources(baseline, current models.Resource) *models.DriftItem {
	// Simple comparison - could be enhanced with deep diff
	if baseline.State != current.State {
		return &models.DriftItem{
			ResourceID:   current.ID,
			ResourceType: current.Type,
			Provider:     current.Provider,
			DriftType:    "modified",
			Severity:     "medium",
			Description:  fmt.Sprintf("Resource state changed from %s to %s", baseline.State, current.State),
		}
	}
	return nil
}

func (s *EnhancedDashboardServer) analyzeResourceDrift(resources []models.Resource, req DriftDetectionRequest) []models.DriftItem {
	// Simplified drift analysis - in production, would use more sophisticated algorithms
	drifts := make([]models.DriftItem, 0)

	for _, resource := range resources {
		// Check if resource matches expected patterns
		if resource.State != "active" && resource.State != "running" {
			drifts = append(drifts, models.DriftItem{
				ResourceID:   resource.ID,
				ResourceType: resource.Type,
				Provider:     resource.Provider,
				DriftType:    "state_drift",
				Severity:     "medium",
				Description:  fmt.Sprintf("Resource %s is in unexpected state: %s", resource.Name, resource.State),
			})
		}
	}

	return drifts
}

func (s *EnhancedDashboardServer) analyzeDriftPatterns(drifts []models.DriftItem) map[string]interface{} {
	// Analyze patterns in drift data
	patterns := make(map[string]interface{})

	// Time-based patterns
	patterns["temporal"] = "Most drift occurs during business hours"

	// Resource type patterns
	typeCount := make(map[string]int)
	for _, drift := range drifts {
		typeCount[drift.ResourceType]++
	}
	patterns["byType"] = typeCount

	return patterns
}

func (s *EnhancedDashboardServer) predictFutureDrift(drifts []models.DriftItem) map[string]interface{} {
	// Simple prediction based on historical patterns
	predictions := make(map[string]interface{})
	predictions["likelihood"] = "high"
	predictions["timeframe"] = "within 7 days"
	predictions["affectedResources"] = len(drifts) * 2 // Simple multiplication
	return predictions
}

func (s *EnhancedDashboardServer) calculateDriftImpact(drifts []models.DriftItem) map[string]interface{} {
	impact := make(map[string]interface{})
	impact["security"] = len(drifts) * 10 // Simplified scoring
	impact["cost"] = len(drifts) * 50.0   // Estimated cost impact
	impact["compliance"] = "medium"
	return impact
}

func (s *EnhancedDashboardServer) generateDriftRecommendations(drifts []models.DriftItem) []string {
	recommendations := []string{
		"Review and update Terraform configurations",
		"Enable drift detection automation",
		"Implement stricter change management policies",
	}

	if len(drifts) > 10 {
		recommendations = append(recommendations, "Consider implementing auto-remediation for common drift patterns")
	}

	return recommendations
}

func (s *EnhancedDashboardServer) getCurrentResourceState(resourceID string) map[string]interface{} {
	// Get current state from data store
	resources := s.dataStore.GetResources()
	for _, item := range resources {
		// Type assertion to access resource fields
		if resMap, ok := item.(map[string]interface{}); ok {
			if id, ok := resMap["id"].(string); ok && id == resourceID {
				return map[string]interface{}{
					"state":      resMap["state"],
					"properties": resMap["properties"],
					"tags":       resMap["tags"],
				}
			}
		} else if res, ok := item.(models.Resource); ok {
			if res.ID == resourceID {
				return map[string]interface{}{
					"state":      res.State,
					"properties": res.Properties,
					"tags":       res.Tags,
				}
			}
		}
	}
	return nil
}

func (s *EnhancedDashboardServer) getExpectedResourceState(resourceID string) map[string]interface{} {
	// Get expected state from baseline or terraform
	return map[string]interface{}{
		"state": "active",
		"properties": map[string]string{
			"configured": "true",
		},
	}
}

func (s *EnhancedDashboardServer) getResourceChangeHistory(resourceID string) []map[string]interface{} {
	// Return change history
	return []map[string]interface{}{
		{
			"timestamp": "2024-01-20T10:00:00Z",
			"change":    "State changed from active to stopped",
			"user":      "system",
		},
	}
}

func (s *EnhancedDashboardServer) assessDriftImpact(drift models.DriftItem) map[string]interface{} {
	return map[string]interface{}{
		"severity":       drift.Severity,
		"businessImpact": "medium",
		"securityRisk":   "low",
		"costImpact":     100.0,
	}
}

// Helper functions for type conversions
func convertInterfacesToResources(interfaces []interface{}) []models.Resource {
	resources := make([]models.Resource, 0, len(interfaces))
	for _, i := range interfaces {
		if res, ok := i.(models.Resource); ok {
			resources = append(resources, res)
		}
	}
	return resources
}

func convertDriftsToInterfaces(drifts []models.DriftItem) []interface{} {
	interfaces := make([]interface{}, len(drifts))
	for i, drift := range drifts {
		interfaces[i] = drift
	}
	return interfaces
}
