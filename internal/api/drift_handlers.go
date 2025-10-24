package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/catherinevee/driftmgr/internal/services"
)

// DriftHandlers handles drift detection API endpoints
type DriftHandlers struct {
	driftService *services.DriftService
}

// NewDriftHandlers creates a new DriftHandlers instance
func NewDriftHandlers(driftService *services.DriftService) *DriftHandlers {
	return &DriftHandlers{
		driftService: driftService,
	}
}

// DetectDrift handles POST /api/v1/drift/detect
func (h *DriftHandlers) DetectDrift(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse request body
	var detectRequest DriftDetectionRequest
	if err := json.NewDecoder(r.Body).Decode(&detectRequest); err != nil {
		response := NewResponseWriter(w)
		response.WriteValidationError("Invalid request body", err.Error())
		return
	}

	// Create drift config for service
	// For now, we'll use the first resource ID if provided, or default to empty
	var resourceID string
	if len(detectRequest.ResourceIDs) > 0 {
		resourceID = detectRequest.ResourceIDs[0]
	}

	var provider string
	if len(detectRequest.Providers) > 0 {
		provider = detectRequest.Providers[0]
	}

	var region string
	if len(detectRequest.Regions) > 0 {
		region = detectRequest.Regions[0]
	}

	driftConfig := services.DriftConfig{
		ResourceID: resourceID,
		Provider:   provider,
		Region:     region,
		Options: services.DriftOptions{
			DeepScan:        true, // Default to deep scan
			IncludeMetadata: true, // Default to include metadata
			IncludeTags:     true, // Default to include tags
			Timeout:         300,  // Default 5 minute timeout
		},
	}

	// Use real service to detect drift
	driftResults, err := h.driftService.DetectDrift(r.Context(), driftConfig)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteInternalError("Failed to detect drift: " + err.Error())
		return
	}

	// Convert to API response format
	apiResults := make([]DriftResult, 0, len(driftResults.DriftDetails))
	for _, detail := range driftResults.DriftDetails {
		apiResult := DriftResult{
			ID:           driftResults.ID,
			ResourceID:   driftResults.ResourceID,
			ResourceName: driftResults.ResourceID, // This would need to be extracted from resource
			ResourceType: driftResults.ResourceType,
			Provider:     driftResults.Provider,
			Region:       driftResults.Region,
			Severity:     detail.Severity,
			Status:       driftResults.Status,
			DriftCount:   driftResults.DriftCount,
			Drifts: []DriftDetail{
				{
					Field:       detail.Field,
					Expected:    detail.ExpectedValue,
					Actual:      detail.ActualValue,
					Severity:    detail.Severity,
					Description: detail.Description,
				},
			},
			DetectedAt: driftResults.DetectedAt,
			Metadata: map[string]string{
				"detection_method": "terraform_plan",
				"confidence":       "high",
			},
		}
		apiResults = append(apiResults, apiResult)
	}

	// Create detection response
	detectionResponse := DriftDetectionResponse{
		JobID:      driftResults.ID,
		Status:     driftResults.Status,
		DriftCount: driftResults.DriftCount,
		Results:    apiResults,
		Message:    "Drift detection completed successfully",
	}

	response := NewResponseWriter(w)
	err = response.WriteSuccess(detectionResponse, &APIMeta{
		Count:     len(apiResults),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// ListDriftResults handles GET /api/v1/drift/results
func (h *DriftHandlers) ListDriftResults(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse query parameters
	queryParams := ParseQueryParams(r)
	provider := queryParams["provider"]
	severity := queryParams["severity"]
	status := queryParams["status"]
	page, limit := ParsePaginationParams(r)

	// Create filters for drift service
	filters := services.DriftFilters{
		Provider: provider,
		Severity: severity,
		Status:   status,
		Limit:    limit,
		Offset:   (page - 1) * limit,
	}

	// Use real service to get drift results
	driftModels, err := h.driftService.GetDriftResults(r.Context(), filters)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteInternalError("Failed to get drift results: " + err.Error())
		return
	}

	// Convert to API response format
	filtered := make([]DriftResult, 0, len(driftModels))
	for _, drift := range driftModels {
		apiResult := DriftResult{
			ID:           drift.ID,
			ResourceID:   "", // This would need to be extracted from drift.Resources
			ResourceName: "", // This would need to be extracted from drift.Resources
			ResourceType: "", // This would need to be extracted from drift.Resources
			Provider:     string(drift.Provider),
			Region:       "",       // This would need to be extracted from drift.Resources
			Severity:     "medium", // This would need to be calculated from drift.Resources
			Status:       string(drift.Status),
			DriftCount:   drift.DriftCount,
			DetectedAt:   drift.Timestamp,
			Metadata: map[string]string{
				"detection_method": "terraform_plan",
				"confidence":       "high",
			},
		}
		filtered = append(filtered, apiResult)
	}

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= len(filtered) {
		filtered = []DriftResult{}
	} else {
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[start:end]
	}

	response := NewResponseWriter(w)
	err = response.WritePaginationResponse(filtered, page, limit, len(filtered))
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// GetDriftResult handles GET /api/v1/drift/results/{id}
func (h *DriftHandlers) GetDriftResult(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Extract drift result ID from URL path
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 4 {
		response := NewResponseWriter(w)
		response.WriteBadRequest("Invalid drift result ID")
		return
	}

	driftID := parts[3]

	// Use real service to get drift result
	driftModel, err := h.driftService.GetDriftResult(r.Context(), driftID)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteNotFound("Drift result")
		return
	}

	// Convert to API response format
	driftResult := DriftResult{
		ID:           driftModel.ID,
		ResourceID:   "", // This would need to be extracted from driftModel.Resources
		ResourceName: "", // This would need to be extracted from driftModel.Resources
		ResourceType: "", // This would need to be extracted from driftModel.Resources
		Provider:     string(driftModel.Provider),
		Region:       "",       // This would need to be extracted from driftModel.Resources
		Severity:     "medium", // This would need to be calculated from driftModel.Resources
		Status:       string(driftModel.Status),
		DriftCount:   driftModel.DriftCount,
		Drifts:       []DriftDetail{}, // This would need to be converted from driftModel.Resources
		DetectedAt:   driftModel.Timestamp,
		Metadata: map[string]string{
			"detection_method":  "terraform_plan",
			"confidence":        "high",
			"terraform_version": "1.5.0",
			"state_serial":      "12345",
		},
	}

	response := NewResponseWriter(w)
	err = response.WriteSuccess(driftResult, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// GetDriftHistory handles GET /api/v1/drift/history
func (h *DriftHandlers) GetDriftHistory(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse query parameters
	queryParams := ParseQueryParams(r)
	provider := queryParams["provider"]
	daysStr := queryParams["days"]

	days := 7 // Default to 7 days
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	// Mock drift history data
	historyData := map[string]interface{}{
		"provider": provider,
		"period": map[string]interface{}{
			"days":  days,
			"start": time.Now().Add(-time.Duration(days) * 24 * time.Hour).Format("2006-01-02"),
			"end":   time.Now().Format("2006-01-02"),
		},
		"summary": map[string]interface{}{
			"total_detections": 15,
			"resolved":         8,
			"pending":          7,
			"by_severity": map[string]int{
				"critical": 2,
				"high":     4,
				"medium":   6,
				"low":      3,
			},
		},
		"daily_breakdown": []map[string]interface{}{
			{
				"date":        time.Now().Add(-6 * 24 * time.Hour).Format("2006-01-02"),
				"detections":  3,
				"resolutions": 2,
				"critical":    1,
				"high":        1,
				"medium":      1,
				"low":         0,
			},
			{
				"date":        time.Now().Add(-5 * 24 * time.Hour).Format("2006-01-02"),
				"detections":  2,
				"resolutions": 1,
				"critical":    0,
				"high":        1,
				"medium":      1,
				"low":         0,
			},
			{
				"date":        time.Now().Add(-4 * 24 * time.Hour).Format("2006-01-02"),
				"detections":  4,
				"resolutions": 2,
				"critical":    1,
				"high":        1,
				"medium":      2,
				"low":         0,
			},
			{
				"date":        time.Now().Add(-3 * 24 * time.Hour).Format("2006-01-02"),
				"detections":  1,
				"resolutions": 1,
				"critical":    0,
				"high":        0,
				"medium":      1,
				"low":         0,
			},
			{
				"date":        time.Now().Add(-2 * 24 * time.Hour).Format("2006-01-02"),
				"detections":  3,
				"resolutions": 1,
				"critical":    0,
				"high":        1,
				"medium":      1,
				"low":         1,
			},
			{
				"date":        time.Now().Add(-1 * 24 * time.Hour).Format("2006-01-02"),
				"detections":  2,
				"resolutions": 1,
				"critical":    0,
				"high":        0,
				"medium":      1,
				"low":         1,
			},
			{
				"date":        time.Now().Format("2006-01-02"),
				"detections":  0,
				"resolutions": 0,
				"critical":    0,
				"high":        0,
				"medium":      0,
				"low":         0,
			},
		},
	}

	response := NewResponseWriter(w)
	err := response.WriteSuccess(historyData, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// GetDriftSummary handles GET /api/v1/drift/summary
func (h *DriftHandlers) GetDriftSummary(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse query parameters
	queryParams := ParseQueryParams(r)
	provider := queryParams["provider"]

	// Mock drift summary
	summary := map[string]interface{}{
		"provider": provider,
		"overview": map[string]interface{}{
			"total_resources":   125,
			"drifted_resources": 8,
			"drift_percentage":  6.4,
			"last_detection":    time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
		"by_severity": map[string]interface{}{
			"critical": map[string]interface{}{
				"count":      2,
				"percentage": 25.0,
			},
			"high": map[string]interface{}{
				"count":      3,
				"percentage": 37.5,
			},
			"medium": map[string]interface{}{
				"count":      2,
				"percentage": 25.0,
			},
			"low": map[string]interface{}{
				"count":      1,
				"percentage": 12.5,
			},
		},
		"by_resource_type": map[string]interface{}{
			"aws_instance": map[string]interface{}{
				"count":      3,
				"percentage": 37.5,
			},
			"aws_s3_bucket": map[string]interface{}{
				"count":      2,
				"percentage": 25.0,
			},
			"aws_db_instance": map[string]interface{}{
				"count":      2,
				"percentage": 25.0,
			},
			"aws_security_group": map[string]interface{}{
				"count":      1,
				"percentage": 12.5,
			},
		},
		"trends": map[string]interface{}{
			"weekly_change":   "+2",
			"monthly_change":  "-5",
			"trend_direction": "improving",
		},
	}

	response := NewResponseWriter(w)
	err := response.WriteSuccess(summary, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// DeleteDriftResult handles DELETE /api/v1/drift/results/{id}
func (h *DriftHandlers) DeleteDriftResult(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Extract drift result ID from URL path
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 4 {
		response := NewResponseWriter(w)
		response.WriteBadRequest("Invalid drift result ID")
		return
	}

	driftID := parts[3]

	// Use real service to delete drift result
	err := h.driftService.DeleteDriftResult(r.Context(), driftID)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteNotFound("Drift result")
		return
	}

	// Successful deletion
	deleteResult := map[string]interface{}{
		"drift_id":   driftID,
		"status":     "success",
		"message":    "Drift result deleted successfully",
		"deleted_at": time.Now().UTC().Format(time.RFC3339),
	}

	response := NewResponseWriter(w)
	err = response.WriteSuccess(deleteResult, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}
