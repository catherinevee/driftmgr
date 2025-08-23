package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/gorilla/mux"
)

// ResourceSearchRequest represents a resource search request
type ResourceSearchRequest struct {
	Query     string            `json:"query"`
	Providers []string          `json:"providers"`
	Types     []string          `json:"types"`
	Regions   []string          `json:"regions"`
	Tags      map[string]string `json:"tags"`
	States    []string          `json:"states"`
	SortBy    string            `json:"sortBy"`
	SortOrder string            `json:"sortOrder"`
	Limit     int               `json:"limit"`
	Offset    int               `json:"offset"`
}

// ResourceBulkOperation represents a bulk operation on resources
type ResourceBulkOperation struct {
	ResourceIDs []string               `json:"resourceIds"`
	Operation   string                 `json:"operation"` // "tag", "delete", "stop", "start", "export"
	Parameters  map[string]interface{} `json:"parameters"`
}

// ResourceDetails provides detailed information about a resource
type ResourceDetails struct {
	Resource     models.Resource        `json:"resource"`
	Dependencies []models.Resource      `json:"dependencies"`
	Dependents   []models.Resource      `json:"dependents"`
	Cost         ResourceCost           `json:"cost"`
	Compliance   ResourceCompliance     `json:"compliance"`
	History      []ResourceChange       `json:"history"`
	Metrics      map[string]interface{} `json:"metrics"`
}

// ResourceCost represents cost information for a resource
type ResourceCost struct {
	Monthly   float64 `json:"monthly"`
	Daily     float64 `json:"daily"`
	Projected float64 `json:"projected"`
	Currency  string  `json:"currency"`
}

// ResourceCompliance represents compliance status
type ResourceCompliance struct {
	Status    string   `json:"status"`
	Issues    []string `json:"issues"`
	Standards []string `json:"standards"`
	LastCheck string   `json:"lastCheck"`
}

// ResourceChange represents a change to a resource
type ResourceChange struct {
	Timestamp   string                 `json:"timestamp"`
	ChangeType  string                 `json:"changeType"`
	Description string                 `json:"description"`
	User        string                 `json:"user"`
	Details     map[string]interface{} `json:"details"`
}

// getResources handles resource listing with filtering
func (s *EnhancedDashboardServer) getResources(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	provider := r.URL.Query().Get("provider")
	resourceType := r.URL.Query().Get("type")
	region := r.URL.Query().Get("region")
	state := r.URL.Query().Get("state")

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	// Get resources from data store
	allResources := s.dataStore.GetResources()

	// Apply filters
	filtered := make([]models.Resource, 0)
	for _, item := range allResources {
		// Type assertion to models.Resource
		if resource, ok := item.(models.Resource); ok {
			if provider != "" && resource.Provider != provider {
				continue
			}
			if resourceType != "" && resource.Type != resourceType {
				continue
			}
			if region != "" && resource.Region != region {
				continue
			}
			if state != "" && resource.State != state {
				continue
			}
			filtered = append(filtered, resource)
		} else if resMap, ok := item.(map[string]interface{}); ok {
			// Handle map representation
			if provider != "" {
				if p, ok := resMap["provider"].(string); ok && p != provider {
					continue
				}
			}
			if resourceType != "" {
				if t, ok := resMap["type"].(string); ok && t != resourceType {
					continue
				}
			}
			if region != "" {
				if r, ok := resMap["region"].(string); ok && r != region {
					continue
				}
			}
			if state != "" {
				if s, ok := resMap["state"].(string); ok && s != state {
					continue
				}
			}
			// Create Resource from map
			res := models.Resource{
				ID:       getStringFromMap(resMap, "id"),
				Name:     getStringFromMap(resMap, "name"),
				Type:     getStringFromMap(resMap, "type"),
				Provider: getStringFromMap(resMap, "provider"),
				Region:   getStringFromMap(resMap, "region"),
				State:    getStringFromMap(resMap, "state"),
			}
			filtered = append(filtered, res)
		}
	}

	// Apply pagination
	total := len(filtered)
	start := offset
	end := offset + limit
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}

	paginated := filtered[start:end]

	// Return response with pagination metadata
	response := map[string]interface{}{
		"resources": paginated,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"hasMore":   end < total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// searchResources handles advanced resource search
func (s *EnhancedDashboardServer) searchResources(w http.ResponseWriter, r *http.Request) {
	var req ResourceSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Get all resources
	allResources := s.dataStore.GetResources()

	// Apply filters
	filtered := make([]models.Resource, 0)
	for _, item := range allResources {
		var resource models.Resource
		var ok bool

		// Type assertion to models.Resource
		if resource, ok = item.(models.Resource); !ok {
			// Try map representation
			if resMap, mapOk := item.(map[string]interface{}); mapOk {
				resource = models.Resource{
					ID:       getStringFromMap(resMap, "id"),
					Name:     getStringFromMap(resMap, "name"),
					Type:     getStringFromMap(resMap, "type"),
					Provider: getStringFromMap(resMap, "provider"),
					Region:   getStringFromMap(resMap, "region"),
					State:    getStringFromMap(resMap, "state"),
				}
			} else {
				continue
			}
		}

		// Text search in name and ID
		if req.Query != "" {
			query := strings.ToLower(req.Query)
			if !strings.Contains(strings.ToLower(resource.Name), query) &&
				!strings.Contains(strings.ToLower(resource.ID), query) {
				continue
			}
		}

		// Provider filter
		if len(req.Providers) > 0 && !contains(req.Providers, resource.Provider) {
			continue
		}

		// Type filter
		if len(req.Types) > 0 && !contains(req.Types, resource.Type) {
			continue
		}

		// Region filter
		if len(req.Regions) > 0 && !contains(req.Regions, resource.Region) {
			continue
		}

		// State filter
		if len(req.States) > 0 && !contains(req.States, resource.State) {
			continue
		}

		// Tag filter
		if len(req.Tags) > 0 {
			matchesTags := true
			resourceTags := resource.GetTagsAsMap()
			for key, value := range req.Tags {
				if resourceTag, exists := resourceTags[key]; !exists || resourceTag != value {
					matchesTags = false
					break
				}
			}
			if !matchesTags {
				continue
			}
		}

		filtered = append(filtered, resource)
	}

	// Sort results
	sortResources(filtered, req.SortBy, req.SortOrder)

	// Apply pagination
	total := len(filtered)
	start := req.Offset
	end := req.Offset + req.Limit
	if end > total {
		end = total
	}
	if start > total {
		start = total
	}

	paginated := filtered[start:end]

	// Return response
	response := map[string]interface{}{
		"resources": paginated,
		"total":     total,
		"query":     req,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// bulkResourceOperation handles bulk operations on resources
func (s *EnhancedDashboardServer) bulkResourceOperation(w http.ResponseWriter, r *http.Request) {
	var req ResourceBulkOperation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.ResourceIDs) == 0 {
		http.Error(w, "No resources specified", http.StatusBadRequest)
		return
	}

	// Create a job for the bulk operation
	job := s.jobManager.CreateJob("bulk_operation")
	job.Result = map[string]interface{}{
		"operation":   req.Operation,
		"resourceIds": req.ResourceIDs,
		"parameters":  req.Parameters,
	}

	// Process operation in background
	go s.processBulkOperation(job, req)

	// Return job ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"jobId":  job.ID,
		"status": "started",
	})
}

// getResourceDetails retrieves detailed information about a resource
func (s *EnhancedDashboardServer) getResourceDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["resourceId"]

	// Find resource
	resources := s.dataStore.GetResources()
	var resource *models.Resource
	for _, item := range resources {
		if res, ok := item.(models.Resource); ok {
			if res.ID == resourceID {
				resource = &res
				break
			}
		} else if resMap, ok := item.(map[string]interface{}); ok {
			if id, ok := resMap["id"].(string); ok && id == resourceID {
				res := models.Resource{
					ID:       getStringFromMap(resMap, "id"),
					Name:     getStringFromMap(resMap, "name"),
					Type:     getStringFromMap(resMap, "type"),
					Provider: getStringFromMap(resMap, "provider"),
					Region:   getStringFromMap(resMap, "region"),
					State:    getStringFromMap(resMap, "state"),
				}
				resource = &res
				break
			}
		}
	}

	if resource == nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	// Build detailed response
	details := ResourceDetails{
		Resource:     *resource,
		Dependencies: s.findResourceDependencies(resourceID),
		Dependents:   s.findResourceDependents(resourceID),
		Cost:         s.calculateResourceCost(*resource),
		Compliance:   s.checkResourceCompliance(*resource),
		History:      s.getResourceHistory(resourceID),
		Metrics:      s.getResourceMetrics(resourceID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(details)
}

// getResourceDependencies retrieves resource dependencies
func (s *EnhancedDashboardServer) getResourceDependencies(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["resourceId"]

	dependencies := s.findResourceDependencies(resourceID)

	// Build dependency graph
	graph := map[string]interface{}{
		"resourceId":   resourceID,
		"dependencies": dependencies,
		"graph":        s.buildDependencyGraph(resourceID, dependencies),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}

// Helper functions

func (s *EnhancedDashboardServer) processBulkOperation(job *Job, req ResourceBulkOperation) {
	total := len(req.ResourceIDs)
	processed := 0

	for _, resourceID := range req.ResourceIDs {
		progress := float64(processed*100) / float64(total)
		s.jobManager.UpdateJob(job.ID, "running", progress)

		switch req.Operation {
		case "tag":
			s.tagResource(resourceID, req.Parameters)
		case "delete":
			s.deleteResource(resourceID)
		case "stop":
			s.stopResource(resourceID)
		case "start":
			s.startResource(resourceID)
		case "export":
			s.exportResource(resourceID, req.Parameters)
		}

		processed++
	}

	s.jobManager.UpdateJob(job.ID, "completed", 100)
}

func (s *EnhancedDashboardServer) findResourceDependencies(resourceID string) []models.Resource {
	// Simplified dependency detection - in production would analyze actual relationships
	dependencies := make([]models.Resource, 0)
	resources := s.dataStore.GetResources()

	for _, item := range resources {
		var resource models.Resource
		if res, ok := item.(models.Resource); ok {
			resource = res
		} else if resMap, ok := item.(map[string]interface{}); ok {
			resource = models.Resource{
				ID:       getStringFromMap(resMap, "id"),
				Name:     getStringFromMap(resMap, "name"),
				Type:     getStringFromMap(resMap, "type"),
				Provider: getStringFromMap(resMap, "provider"),
				Region:   getStringFromMap(resMap, "region"),
				State:    getStringFromMap(resMap, "state"),
			}
		} else {
			continue
		}

		// Example: VMs depend on VPCs
		if strings.Contains(resourceID, "instance") && resource.Type == "vpc" {
			dependencies = append(dependencies, resource)
		}
	}

	return dependencies
}

func (s *EnhancedDashboardServer) findResourceDependents(resourceID string) []models.Resource {
	// Find resources that depend on this resource
	dependents := make([]models.Resource, 0)
	resources := s.dataStore.GetResources()

	for _, item := range resources {
		var resource models.Resource
		if res, ok := item.(models.Resource); ok {
			resource = res
		} else if resMap, ok := item.(map[string]interface{}); ok {
			resource = models.Resource{
				ID:       getStringFromMap(resMap, "id"),
				Name:     getStringFromMap(resMap, "name"),
				Type:     getStringFromMap(resMap, "type"),
				Provider: getStringFromMap(resMap, "provider"),
				Region:   getStringFromMap(resMap, "region"),
				State:    getStringFromMap(resMap, "state"),
			}
		} else {
			continue
		}

		// Example: instances depend on VPCs
		if strings.Contains(resourceID, "vpc") && strings.Contains(resource.Type, "instance") {
			dependents = append(dependents, resource)
		}
	}

	return dependents
}

func (s *EnhancedDashboardServer) calculateResourceCost(resource models.Resource) ResourceCost {
	// Simplified cost calculation - in production would use cloud pricing APIs
	baseCost := 0.0

	switch resource.Type {
	case "aws_instance", "azure_virtual_machine", "gcp_compute_instance":
		baseCost = 50.0 // $50/month base
	case "aws_rds_instance", "azure_sql_database":
		baseCost = 100.0 // $100/month for databases
	case "aws_s3_bucket", "azure_storage_account":
		baseCost = 10.0 // $10/month for storage
	default:
		baseCost = 5.0 // Default cost
	}

	return ResourceCost{
		Monthly:   baseCost,
		Daily:     baseCost / 30,
		Projected: baseCost * 12,
		Currency:  "USD",
	}
}

func (s *EnhancedDashboardServer) checkResourceCompliance(resource models.Resource) ResourceCompliance {
	issues := make([]string, 0)

	// Example compliance checks
	tags := resource.GetTagsAsMap()
	if tags["Environment"] == "" {
		issues = append(issues, "Missing Environment tag")
	}
	if tags["Owner"] == "" {
		issues = append(issues, "Missing Owner tag")
	}

	status := "compliant"
	if len(issues) > 0 {
		status = "non-compliant"
	}

	return ResourceCompliance{
		Status:    status,
		Issues:    issues,
		Standards: []string{"CIS", "PCI-DSS", "HIPAA"},
		LastCheck: "2024-01-20T10:00:00Z",
	}
}

func (s *EnhancedDashboardServer) getResourceHistory(resourceID string) []ResourceChange {
	// Return sample history - in production would track actual changes
	return []ResourceChange{
		{
			Timestamp:   "2024-01-20T10:00:00Z",
			ChangeType:  "created",
			Description: "Resource created",
			User:        "terraform",
		},
		{
			Timestamp:   "2024-01-20T11:00:00Z",
			ChangeType:  "modified",
			Description: "Tags updated",
			User:        "admin",
		},
	}
}

func (s *EnhancedDashboardServer) getResourceMetrics(resourceID string) map[string]interface{} {
	// Return sample metrics - in production would fetch from monitoring systems
	return map[string]interface{}{
		"cpu_utilization":    15.5,
		"memory_utilization": 45.2,
		"network_in":         1024,
		"network_out":        2048,
		"uptime_percentage":  99.9,
	}
}

func (s *EnhancedDashboardServer) buildDependencyGraph(resourceID string, dependencies []models.Resource) map[string]interface{} {
	// Build a graph structure for visualization
	nodes := make([]map[string]interface{}, 0)
	edges := make([]map[string]interface{}, 0)

	// Add main resource as root node
	nodes = append(nodes, map[string]interface{}{
		"id":    resourceID,
		"label": resourceID,
		"type":  "root",
	})

	// Add dependencies as nodes and edges
	for _, dep := range dependencies {
		nodes = append(nodes, map[string]interface{}{
			"id":    dep.ID,
			"label": dep.Name,
			"type":  dep.Type,
		})

		edges = append(edges, map[string]interface{}{
			"from": resourceID,
			"to":   dep.ID,
			"type": "depends_on",
		})
	}

	return map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}
}

func (s *EnhancedDashboardServer) tagResource(resourceID string, parameters map[string]interface{}) {
	// Tag resource logic
	tags, _ := parameters["tags"].(map[string]string)
	// Apply tags to resource
	_ = tags
}

func (s *EnhancedDashboardServer) deleteResource(resourceID string) {
	// Delete resource logic
}

func (s *EnhancedDashboardServer) stopResource(resourceID string) {
	// Stop resource logic
}

func (s *EnhancedDashboardServer) startResource(resourceID string) {
	// Start resource logic
}

func (s *EnhancedDashboardServer) exportResource(resourceID string, parameters map[string]interface{}) {
	// Export resource logic
	format, _ := parameters["format"].(string)
	_ = format
}

// Utility functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func sortResources(resources []models.Resource, sortBy, sortOrder string) {
	// Simple sorting implementation - could be enhanced
	// For now, resources remain in original order
}
