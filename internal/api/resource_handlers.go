package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/services"
)

// ResourceHandlers handles resource management API endpoints
type ResourceHandlers struct {
	resourceService *services.ResourceService
}

// NewResourceHandlers creates a new ResourceHandlers instance
func NewResourceHandlers(resourceService *services.ResourceService) *ResourceHandlers {
	return &ResourceHandlers{
		resourceService: resourceService,
	}
}

// ListResources handles GET /api/v1/resources
func (h *ResourceHandlers) ListResources(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse query parameters
	queryParams := ParseQueryParams(r)
	provider := queryParams["provider"]
	region := queryParams["region"]
	resourceType := queryParams["type"]
	page, limit := ParsePaginationParams(r)

	// Create filters for resource service
	filters := services.ResourceFilters{
		Provider: provider,
		Region:   region,
		Type:     resourceType,
		Limit:    limit,
		Offset:   (page - 1) * limit,
	}

	// Use real service to get resources
	resourceModels, err := h.resourceService.ListResources(r.Context(), filters)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteInternalError("Failed to list resources: " + err.Error())
		return
	}

	// Convert to API response format
	filtered := make([]Resource, 0, len(resourceModels))
	for _, resource := range resourceModels {
		apiResource := Resource{
			ID:           resource.ID,
			Name:         resource.Name,
			Type:         resource.Type,
			Provider:     string(resource.Provider),
			Region:       resource.Region,
			AccountID:    resource.AccountID,
			State:        "present", // This would need to be determined from resource state
			Tags:         resource.Tags,
			Metadata:     convertMetadataToStringMap(resource.Metadata),
			DiscoveredAt: resource.LastDiscovered,
			UpdatedAt:    resource.UpdatedAt,
		}
		filtered = append(filtered, apiResource)
	}

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= len(filtered) {
		filtered = []Resource{}
	} else {
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[start:end]
	}

	response := NewResponseWriter(w)
	err := response.WritePaginationResponse(filtered, page, limit, len(filtered))
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// GetResource handles GET /api/v1/resources/{id}
func (h *ResourceHandlers) GetResource(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Extract resource ID from URL path
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 3 {
		response := NewResponseWriter(w)
		response.WriteBadRequest("Invalid resource ID")
		return
	}

	resourceID := parts[2]

	// Use real service to get resource
	resourceModel, err := h.resourceService.GetResource(r.Context(), resourceID)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteNotFound("Resource")
		return
	}

	// Convert to API response format
	resource := Resource{
		ID:           resourceModel.ID,
		Name:         resourceModel.Name,
		Type:         resourceModel.Type,
		Provider:     string(resourceModel.Provider),
		Region:       resourceModel.Region,
		AccountID:    resourceModel.AccountID,
		State:        "present", // This would need to be determined from resource state
		Tags:         resourceModel.Tags,
		Metadata:     convertMetadataToStringMap(resourceModel.Metadata),
		DiscoveredAt: resourceModel.LastDiscovered,
		UpdatedAt:    resourceModel.UpdatedAt,
	}

	response := NewResponseWriter(w)
	err := response.WriteSuccess(resource, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// SearchResources handles GET /api/v1/resources/search
func (h *ResourceHandlers) SearchResources(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Parse query parameters
	queryParams := ParseQueryParams(r)
	query := queryParams["q"]
	page, limit := ParsePaginationParams(r)

	if query == "" {
		response := NewResponseWriter(w)
		response.WriteValidationError("Search query parameter 'q' is required", "")
		return
	}

	// Create search query for resource service
	searchQuery := services.ResourceSearchQuery{
		Query:  query,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}

	// Use real service to search resources
	resourceModels, err := h.resourceService.SearchResources(r.Context(), searchQuery)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteInternalError("Failed to search resources: " + err.Error())
		return
	}

	// Convert to API response format
	results := make([]Resource, 0, len(resourceModels))
	for _, resource := range resourceModels {
		apiResource := Resource{
			ID:           resource.ID,
			Name:         resource.Name,
			Type:         resource.Type,
			Provider:     string(resource.Provider),
			Region:       resource.Region,
			AccountID:    resource.AccountID,
			State:        "present", // This would need to be determined from resource state
			Tags:         resource.Tags,
			Metadata:     convertMetadataToStringMap(resource.Metadata),
			DiscoveredAt: resource.LastDiscovered,
			UpdatedAt:    resource.UpdatedAt,
		}
		results = append(results, apiResource)
	}

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= len(results) {
		results = []Resource{}
	} else {
		if end > len(results) {
			end = len(results)
		}
		results = results[start:end]
	}

	response := NewResponseWriter(w)
	err := response.WritePaginationResponse(results, page, limit, len(results))
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// UpdateResourceTags handles PUT /api/v1/resources/{id}/tags
func (h *ResourceHandlers) UpdateResourceTags(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Extract resource ID from URL path
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 4 {
		response := NewResponseWriter(w)
		response.WriteBadRequest("Invalid resource ID")
		return
	}

	resourceID := parts[2]

	// Parse request body
	var tagRequest struct {
		Tags map[string]string `json:"tags"`
	}

	if err := json.NewDecoder(r.Body).Decode(&tagRequest); err != nil {
		response := NewResponseWriter(w)
		response.WriteValidationError("Invalid request body", err.Error())
		return
	}

	// Use real service to update resource tags
	err := h.resourceService.UpdateResourceTags(r.Context(), resourceID, tagRequest.Tags)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteNotFound("Resource")
		return
	}

	// Successful update
	updateResult := map[string]interface{}{
		"resource_id": resourceID,
		"tags":        tagRequest.Tags,
		"status":      "success",
		"message":     "Resource tags updated successfully",
		"updated_at":  time.Now().UTC().Format(time.RFC3339),
	}

	response := NewResponseWriter(w)
	err := response.WriteSuccess(updateResult, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// GetResourceCost handles GET /api/v1/resources/{id}/cost
func (h *ResourceHandlers) GetResourceCost(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Extract resource ID from URL path
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 4 {
		response := NewResponseWriter(w)
		response.WriteBadRequest("Invalid resource ID")
		return
	}

	resourceID := parts[2]

	// Use real service to get resource cost
	costData, err := h.resourceService.GetResourceCost(r.Context(), resourceID)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteNotFound("Resource")
		return
	}

	// Convert to API response format
	apiCostData := ResourceCost{
		MonthlyCost:    costData.CostPerMonth,
		DailyCost:      costData.CostPerHour * 24, // Approximate daily cost
		Currency:       costData.Currency,
		LastCalculated: costData.LastUpdated,
	}

	// Add cost breakdown
	costBreakdown := map[string]interface{}{
		"resource_id": resourceID,
		"cost":        apiCostData,
		"breakdown":   costData.Details,
		"trend": map[string]interface{}{
			"daily":   []float64{}, // This would need to be calculated from historical data
			"monthly": []float64{}, // This would need to be calculated from historical data
		},
	}

	response := NewResponseWriter(w)
	err := response.WriteSuccess(costBreakdown, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// GetResourceCompliance handles GET /api/v1/resources/{id}/compliance
func (h *ResourceHandlers) GetResourceCompliance(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Extract resource ID from URL path
	path := r.URL.Path
	parts := splitPath(path)
	if len(parts) < 4 {
		response := NewResponseWriter(w)
		response.WriteBadRequest("Invalid resource ID")
		return
	}

	resourceID := parts[2]

	// Use real service to get resource compliance
	complianceData, err := h.resourceService.GetResourceCompliance(r.Context(), resourceID)
	if err != nil {
		response := NewResponseWriter(w)
		response.WriteNotFound("Resource")
		return
	}

	// Convert violations to API format
	var apiViolations []ComplianceViolation
	for _, violation := range complianceData.Violations {
		apiViolations = append(apiViolations, ComplianceViolation{
			Rule:        violation.RuleID,
			Severity:    violation.Severity,
			Description: violation.Description,
			Remediation: violation.Remediation,
		})
	}

	// Convert to API response format
	apiComplianceData := ComplianceStatus{
		Status:      "compliant",
		Score:       int(complianceData.Score),
		Violations:  apiViolations,
		LastChecked: complianceData.LastChecked,
	}

	if !complianceData.Compliant {
		apiComplianceData.Status = "non_compliant"
	}

	// Add compliance details
	complianceDetails := map[string]interface{}{
		"resource_id": resourceID,
		"compliance":  apiComplianceData,
		"frameworks": map[string]interface{}{
			"soc2": map[string]interface{}{
				"status": "compliant", // This would need to be calculated from actual compliance data
				"score":  98,
			},
			"hipaa": map[string]interface{}{
				"status": "compliant", // This would need to be calculated from actual compliance data
				"score":  92,
			},
		},
		"recommendations": []string{
			"Enable detailed monitoring",
			"Review access policies quarterly",
		},
	}

	response := NewResponseWriter(w)
	err = response.WriteSuccess(complianceDetails, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode response")
		return
	}
}

// Helper function to convert metadata to string map
func convertMetadataToStringMap(metadata map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		if str, ok := v.(string); ok {
			result[k] = str
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}
