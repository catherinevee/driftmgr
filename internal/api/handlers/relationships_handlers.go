package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// handleGetRelationships returns all discovered relationships
func (s *Server) handleGetRelationships(w http.ResponseWriter, r *http.Request) {
	if s.relationshipMapper == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Relationship mapper not initialized")
		return
	}
	
	relationships := s.relationshipMapper.GetRelationships()
	
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"relationships": relationships,
		"count":         len(relationships),
	})
}

// handleDiscoverRelationships discovers relationships for current resources
func (s *Server) handleDiscoverRelationships(w http.ResponseWriter, r *http.Request) {
	if s.relationshipMapper == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Relationship mapper not initialized")
		return
	}
	
	// Get resources from discovery hub or request body
	var resources []Resource
	
	// Check if resources are provided in request body
	var req struct {
		Resources []Resource `json:"resources"`
	}
	
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && len(req.Resources) > 0 {
			resources = req.Resources
		}
	}
	
	// If no resources provided, use cached resources
	if len(resources) == 0 && s.discoveryHub != nil {
		resources = s.discoveryHub.GetCachedResources()
	}
	
	if len(resources) == 0 {
		s.respondError(w, http.StatusBadRequest, "No resources available for relationship discovery")
		return
	}
	
	// Discover relationships
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	
	if err := s.relationshipMapper.DiscoverRelationships(ctx, resources); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to discover relationships")
		return
	}
	
	// Get the discovered relationships
	relationships := s.relationshipMapper.GetRelationships()
	graph := s.relationshipMapper.GetGraph()
	
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"relationships": relationships,
		"graph":         graph,
		"count":         len(relationships),
		"resources":     len(resources),
		"message":       "Relationships discovered successfully",
	})
}

// handleGetDependencyGraph returns the full dependency graph
func (s *Server) handleGetDependencyGraph(w http.ResponseWriter, r *http.Request) {
	if s.relationshipMapper == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Relationship mapper not initialized")
		return
	}
	
	graph := s.relationshipMapper.GetGraph()
	
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"graph": graph,
		"stats": map[string]interface{}{
			"nodes":         len(graph.Nodes),
			"relationships": len(graph.Relationships),
			"cycles":        len(graph.Cycles),
			"layers":        len(graph.Layers),
		},
	})
}

// handleGetResourceRelationships returns relationships for a specific resource
func (s *Server) handleGetResourceRelationships(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]
	
	if resourceID == "" {
		s.respondError(w, http.StatusBadRequest, "Resource ID required")
		return
	}
	
	if s.relationshipMapper == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Relationship mapper not initialized")
		return
	}
	
	// Get dependencies and dependents
	dependencies := s.relationshipMapper.GetDependencies(resourceID)
	dependents := s.relationshipMapper.GetDependents(resourceID)
	
	// Get relationships from persistence if available
	var persistedRelationships []map[string]interface{}
	if s.persistence != nil {
		persistedRelationships, _ = s.persistence.GetResourceRelationships(resourceID)
	}
	
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"resource_id":   resourceID,
		"dependencies":  dependencies,
		"dependents":    dependents,
		"relationships": persistedRelationships,
		"stats": map[string]interface{}{
			"dependency_count": len(dependencies),
			"dependent_count":  len(dependents),
		},
	})
}

// handleGetDeletionOrder returns the safe deletion order for resources
func (s *Server) handleGetDeletionOrder(w http.ResponseWriter, r *http.Request) {
	if s.relationshipMapper == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Relationship mapper not initialized")
		return
	}
	
	var req struct {
		ResourceIDs []string `json:"resource_ids"`
		Validate    bool     `json:"validate"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if len(req.ResourceIDs) == 0 {
		s.respondError(w, http.StatusBadRequest, "Resource IDs required")
		return
	}
	
	// Get deletion order
	order := s.relationshipMapper.GetDeletionOrder(req.ResourceIDs)
	
	response := map[string]interface{}{
		"deletion_order": order,
		"count":          len(order),
	}
	
	// Validate if requested
	if req.Validate {
		canDelete, blockers := s.relationshipMapper.ValidateDeletion(req.ResourceIDs)
		response["can_delete"] = canDelete
		response["blockers"] = blockers
		
		if !canDelete {
			response["message"] = "Some resources cannot be deleted due to dependencies"
		} else {
			response["message"] = "All resources can be safely deleted"
		}
	}
	
	s.respondJSON(w, http.StatusOK, response)
}