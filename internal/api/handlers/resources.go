package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/catherinevee/driftmgr/internal/remediation"
)

// DeleteResourcesRequest represents a request to delete resources
type DeleteResourcesRequest struct {
	ResourceIDs []string `json:"resource_ids"`
	DryRun      bool     `json:"dry_run,omitempty"`
}

// DeleteResourcesResponse represents the response from deleting resources
type DeleteResourcesResponse struct {
	Deleted   int      `json:"deleted"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors,omitempty"`
	DryRun    bool     `json:"dry_run,omitempty"`
}

// handleResourcesDelete handles bulk resource deletion
func (s *Server) handleResourcesDelete(w http.ResponseWriter, r *http.Request) {
	var req DeleteResourcesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if len(req.ResourceIDs) == 0 {
		sendJSONError(w, "No resources specified for deletion", http.StatusBadRequest)
		return
	}
	
	// Get resources from cache
	resources := s.discoveryHub.GetResourcesByIDs(req.ResourceIDs)
	if len(resources) == 0 {
		sendJSONError(w, "No resources found with specified IDs", http.StatusNotFound)
		return
	}
	
	// Log the deletion attempt
	username := r.Header.Get("X-Username")
	s.auditLogger.LogAction(context.Background(), "resource_deletion", username, map[string]interface{}{
		"resource_count": len(resources),
		"dry_run":        req.DryRun,
	})
	
	response := DeleteResourcesResponse{
		DryRun: req.DryRun,
		Errors: []string{},
	}
	
	// Group resources by provider
	byProvider := make(map[string][]Resource)
	for _, resource := range resources {
		byProvider[resource.Provider] = append(byProvider[resource.Provider], resource)
	}
	
	// Delete resources by provider
	for provider, providerResources := range byProvider {
		if req.DryRun {
			// Just count what would be deleted
			response.Deleted += len(providerResources)
			continue
		}
		
		// Get deletion provider
		var deleter deletion.ResourceDeleter
		switch provider {
		case "aws":
			deleter = deletion.NewAWSProvider()
		case "azure":
			deleter = deletion.NewAzureProvider()
		default:
			response.Failed += len(providerResources)
			response.Errors = append(response.Errors, fmt.Sprintf("Deletion not supported for provider: %s", provider))
			continue
		}
		
		// Delete each resource
		for _, resource := range providerResources {
			err := deleter.DeleteResource(context.Background(), resource.ID, resource.Type)
			if err != nil {
				response.Failed++
				response.Errors = append(response.Errors, fmt.Sprintf("Failed to delete %s: %v", resource.ID, err))
			} else {
				response.Deleted++
				// Remove from cache
				s.discoveryHub.RemoveResource(resource.ID)
			}
		}
	}
	
	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RemoveResource removes a resource from the cache
func (h *DiscoveryHub) RemoveResource(resourceID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	for i, resource := range h.cache {
		if resource.ID == resourceID {
			// Remove the resource
			h.cache = append(h.cache[:i], h.cache[i+1:]...)
			h.cacheVersion++
			h.updateCacheMetadata("deletion")
			return
		}
	}
}

// handleListResources returns a list of all discovered resources
func (s *Server) handleListResources(w http.ResponseWriter, r *http.Request) {
	// Get filter parameters
	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")
	resourceType := r.URL.Query().Get("type")
	
	// Get all resources from cache
	resources := s.discoveryHub.GetCachedResources()
	
	// Apply filters
	var filtered []Resource
	for _, resource := range resources {
		if provider != "" && !strings.EqualFold(resource.Provider, provider) {
			continue
		}
		if region != "" && !strings.EqualFold(resource.Region, region) {
			continue
		}
		if resourceType != "" && !strings.Contains(strings.ToLower(resource.Type), strings.ToLower(resourceType)) {
			continue
		}
		filtered = append(filtered, resource)
	}
	
	// Return results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"resources": filtered,
		"total":     len(filtered),
		"metadata": map[string]interface{}{
			"provider": provider,
			"region":   region,
			"type":     resourceType,
		},
	})
}

// handleResourceStats returns statistics about discovered resources
func (s *Server) handleResourceStats(w http.ResponseWriter, r *http.Request) {
	resources := s.discoveryHub.GetCachedResources()
	
	// Calculate statistics
	stats := map[string]interface{}{
		"total": len(resources),
		"by_provider": make(map[string]int),
		"by_region": make(map[string]int),
		"by_type": make(map[string]int),
		"by_status": make(map[string]int),
	}
	
	for _, resource := range resources {
		// Count by provider
		if byProvider, ok := stats["by_provider"].(map[string]int); ok {
			byProvider[resource.Provider]++
		}
		// Count by region
		if byRegion, ok := stats["by_region"].(map[string]int); ok {
			byRegion[resource.Region]++
		}
		// Count by type
		if byType, ok := stats["by_type"].(map[string]int); ok {
			byType[resource.Type]++
		}
		// Count by status
		if byStatus, ok := stats["by_status"].(map[string]int); ok {
			byStatus[resource.Status]++
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}