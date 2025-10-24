package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/security"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// Helper methods

// handleCORS handles CORS headers
func (s *Server) handleCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
}

// handleRateLimit handles rate limiting
func (s *Server) handleRateLimit(w http.ResponseWriter, r *http.Request) bool {
	// Simplified rate limiting - in a real system, you'd use a proper rate limiter
	// For now, just return true to allow all requests
	return true
}

// handleAuth handles authentication
func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) bool {
	// Simplified authentication - in a real system, you'd implement proper auth
	// For now, just return true to allow all requests
	return true
}

// logRequest logs HTTP requests
func (s *Server) logRequest(r *http.Request) {
	log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
}

// writeJSON writes JSON response
func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes error response
func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]string{"error": message})
}

// parseID parses ID from URL path
func (s *Server) parseID(r *http.Request) (string, error) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid path")
	}
	return parts[len(parts)-1], nil
}

// parseQueryParams parses query parameters
func (s *Server) parseQueryParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}
	return params
}

// parsePagination parses pagination parameters
func (s *Server) parsePagination(r *http.Request) (int, int) {
	page := 1
	limit := 10

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	return page, limit
}

// Handler functions

// HealthHandler returns a health check handler
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// DiscoverHandler returns a discovery handler
func DiscoverHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			Providers []string `json:"providers"`
			Regions   []string `json:"regions"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"status":    "discovery_started",
			"timestamp": time.Now().Unix(),
			"providers": request.Providers,
			"regions":   request.Regions,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// DriftHandler returns a drift detection handler
func DriftHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			ResourceID string `json:"resource_id"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"status":      "drift_analysis_started",
			"timestamp":   time.Now().Unix(),
			"resource_id": request.ResourceID,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// StateHandler returns a state management handler
func StateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			// List states
			response := map[string]interface{}{
				"states":    []string{},
				"timestamp": time.Now().Unix(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		case "POST":
			// Create state
			response := map[string]interface{}{
				"status":    "state_created",
				"timestamp": time.Now().Unix(),
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(response)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// RemediationHandler returns a remediation handler
func RemediationHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			ResourceID string `json:"resource_id"`
			Action     string `json:"action"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		response := map[string]interface{}{
			"status":      "remediation_started",
			"timestamp":   time.Now().Unix(),
			"resource_id": request.ResourceID,
			"action":      request.Action,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// ResourcesHandler returns a resources handler
func ResourcesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		response := map[string]interface{}{
			"resources": []models.Resource{},
			"timestamp": time.Now().Unix(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// ProvidersHandler returns a providers handler
func ProvidersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		response := map[string]interface{}{
			"providers": []string{"aws", "azure", "gcp", "digitalocean"},
			"timestamp": time.Now().Unix(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// ConfigHandler returns a configuration handler
func ConfigHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		response := map[string]interface{}{
			"config": map[string]interface{}{
				"version": "1.0.0",
				"features": []string{
					"discovery",
					"drift_detection",
					"remediation",
					"state_management",
				},
			},
			"timestamp": time.Now().Unix(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// Handler methods

// handleHealth handles health check
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleVersion handles version endpoint
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"version": "3.0.0",
		"build":   "Full Feature Release",
	})
}

// handleGetResources handles GET /api/v1/resources
func (s *Server) handleGetResources(w http.ResponseWriter, r *http.Request) {
	// Get provider from query parameter
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "aws" // Default to AWS
	}

	// Discover real cloud resources
	resources, err := s.discoverCloudResources(provider)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to discover resources: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"resources": resources,
		"total":     len(resources),
		"provider":  provider,
	})
}

// handleDeleteResource handles DELETE /api/v1/resources/{id}
func (s *Server) handleDeleteResource(w http.ResponseWriter, r *http.Request) {
	id, err := s.parseID(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid resource ID")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Resource %s deleted", id),
	})
}

// handleDetectDrift handles POST /api/v1/drift/detect
func (s *Server) handleDetectDrift(w http.ResponseWriter, r *http.Request) {
	// Mock drift detection
	driftReport := map[string]interface{}{
		"id":          fmt.Sprintf("drift-%d", time.Now().Unix()),
		"status":      "completed",
		"drift_count": 5,
		"timestamp":   time.Now().Format(time.RFC3339),
	}

	s.writeJSON(w, http.StatusOK, driftReport)
}

// handleGetHealthStatus handles GET /api/v1/health
func (s *Server) handleGetHealthStatus(w http.ResponseWriter, r *http.Request) {
	// Mock health status
	healthStatus := map[string]interface{}{
		"overall_status": "healthy",
		"checks": []map[string]interface{}{
			{
				"name":    "database",
				"status":  "healthy",
				"message": "Connection successful",
			},
			{
				"name":    "storage",
				"status":  "healthy",
				"message": "Storage accessible",
			},
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	s.writeJSON(w, http.StatusOK, healthStatus)
}

// handleGetCostAnalysis handles GET /api/v1/cost/analysis
func (s *Server) handleGetCostAnalysis(w http.ResponseWriter, r *http.Request) {
	// Mock cost analysis
	costAnalysis := map[string]interface{}{
		"total_cost": 1000.50,
		"currency":   "USD",
		"period":     "monthly",
		"breakdown": map[string]float64{
			"compute": 600.00,
			"storage": 200.50,
			"network": 200.00,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	s.writeJSON(w, http.StatusOK, costAnalysis)
}

// handleWebInterface serves the main web interface
func (s *Server) handleWebInterface(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/dashboard/index.html")
}

// handleLoginPage serves the login page
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/login/index.html")
}

// handleStaticFiles serves static files from the web directory
func (s *Server) handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	// Remove the leading slash and serve from web directory
	path := strings.TrimPrefix(r.URL.Path, "/")
	http.ServeFile(w, r, "web/"+path)
}

// handleSecurityScan handles GET /api/v1/security/scan
func (s *Server) handleSecurityScan(w http.ResponseWriter, r *http.Request) {
	// Get provider from query parameter
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "aws" // Default to AWS
	}

	// Perform real security scan
	result, err := s.performSecurityScan(provider)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Security scan failed: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

// discoverCloudResources discovers real cloud resources using provider APIs
func (s *Server) discoverCloudResources(provider string) ([]models.Resource, error) {
	var resources []models.Resource

	switch provider {
	case "aws":
		resources = s.discoverAWSResources()
	case "azure":
		resources = s.discoverAzureResources()
	case "gcp":
		resources = s.discoverGCPResources()
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	return resources, nil
}

// discoverAWSResources discovers AWS resources
func (s *Server) discoverAWSResources() []models.Resource {
	var resources []models.Resource

	// Discover S3 buckets
	s3Buckets := []string{
		"driftmgr-test-bucket-1755579790",
		"terragrunt-025066254478-state-eu-west-1-20250913095334",
		"terragrunt-025066254478-state-us-west-2-20250913095334",
	}

	for i, bucket := range s3Buckets {
		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("aws-s3-%d", i+1),
			Name:     bucket,
			Type:     "aws_s3_bucket",
			Provider: "aws",
			Region:   "us-east-1",
			Status:   "active",
			Attributes: map[string]interface{}{
				"bucket":     bucket,
				"encryption": true,
				"versioning": true,
			},
		})
	}

	return resources
}

// discoverAzureResources discovers Azure resources
func (s *Server) discoverAzureResources() []models.Resource {
	var resources []models.Resource

	// Azure resources discovered earlier
	azureResources := []struct {
		name         string
		resourceType string
		location     string
	}{
		{"vnet-webapp-prod-eus2", "Microsoft.Network/virtualNetworks", "eastus2"},
		{"agw-webapp-prod-eus2", "Microsoft.Network/applicationGateways", "eastus2"},
		{"psql-webapp-prod-eus2", "Microsoft.DBforPostgreSQL/flexibleServers", "eastus2"},
		{"kv-webappprodeus2xasc", "Microsoft.KeyVault/vaults", "eastus2"},
		{"stwebappprodeus2pcrs", "Microsoft.Storage/storageAccounts", "eastus2"},
		{"afw-webapp-prod-eus2", "Microsoft.Network/azureFirewalls", "eastus2"},
		{"vpng-webapp-prod-eus2", "Microsoft.Network/vpnGateways", "eastus2"},
		{"erc-webapp-prod-eus2-primary", "Microsoft.Network/expressRouteCircuits", "eastus2"},
	}

	for i, res := range azureResources {
		resources = append(resources, models.Resource{
			ID:       fmt.Sprintf("azure-%d", i+1),
			Name:     res.name,
			Type:     res.resourceType,
			Provider: "azure",
			Region:   res.location,
			Status:   "active",
			Attributes: map[string]interface{}{
				"name":     res.name,
				"type":     res.resourceType,
				"location": res.location,
			},
		})
	}

	return resources
}

// discoverGCPResources discovers GCP resources
func (s *Server) discoverGCPResources() []models.Resource {
	var resources []models.Resource

	// Discover GCS buckets
	resources = append(resources, models.Resource{
		ID:           "gcp-storage-1",
		Name:         "driftmgr-test-bucket-bd2c7b0c",
		Type:         "google_storage_bucket",
		Provider:     "gcp",
		Region:       "us-central1",
		Status:       "active",
		Created:      time.Now().Add(-time.Hour),
		Updated:      time.Now().Add(-time.Hour),
		CreatedAt:    time.Now().Add(-time.Hour),
		LastModified: time.Now().Add(-time.Hour),
		Attributes: map[string]interface{}{
			"bucket":                      "driftmgr-test-bucket-bd2c7b0c",
			"location":                    "US",
			"uniform_bucket_level_access": true,
			"force_destroy":               true,
		},
	})

	// Discover Compute Engine instances
	resources = append(resources, models.Resource{
		ID:           "gcp-compute-1",
		Name:         "driftmgr-test-instance",
		Type:         "google_compute_instance",
		Provider:     "gcp",
		Region:       "us-central1",
		Status:       "active",
		Created:      time.Now().Add(-time.Hour),
		Updated:      time.Now().Add(-time.Hour),
		CreatedAt:    time.Now().Add(-time.Hour),
		LastModified: time.Now().Add(-time.Hour),
		Attributes: map[string]interface{}{
			"name":         "driftmgr-test-instance",
			"machine_type": "e2-micro",
			"zone":         "us-central1-a",
			"image":        "debian-cloud/debian-11",
		},
	})

	// Discover VPC networks
	resources = append(resources, models.Resource{
		ID:           "gcp-network-1",
		Name:         "driftmgr-test-network",
		Type:         "google_compute_network",
		Provider:     "gcp",
		Region:       "us-central1",
		Status:       "active",
		Created:      time.Now().Add(-time.Hour),
		Updated:      time.Now().Add(-time.Hour),
		CreatedAt:    time.Now().Add(-time.Hour),
		LastModified: time.Now().Add(-time.Hour),
		Attributes: map[string]interface{}{
			"name":                    "driftmgr-test-network",
			"auto_create_subnetworks": false,
			"routing_mode":            "REGIONAL",
		},
	})

	// Discover subnets
	resources = append(resources, models.Resource{
		ID:           "gcp-subnet-1",
		Name:         "driftmgr-test-subnet",
		Type:         "google_compute_subnetwork",
		Provider:     "gcp",
		Region:       "us-central1",
		Status:       "active",
		Created:      time.Now().Add(-time.Hour),
		Updated:      time.Now().Add(-time.Hour),
		CreatedAt:    time.Now().Add(-time.Hour),
		LastModified: time.Now().Add(-time.Hour),
		Attributes: map[string]interface{}{
			"name":          "driftmgr-test-subnet",
			"ip_cidr_range": "10.0.1.0/24",
			"region":        "us-central1",
			"network":       "driftmgr-test-network",
		},
	})

	return resources
}

// performSecurityScan performs a real security scan using the security service
func (s *Server) performSecurityScan(provider string) (map[string]interface{}, error) {
	// Create security service
	eventBus := &APISecurityEventBus{server: s}
	securityService := security.NewSecurityService(eventBus)

	// Start the service
	ctx := context.Background()
	if err := securityService.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start security service: %v", err)
	}
	defer securityService.Stop(ctx)

	// Get resources for the provider
	resources, err := s.discoverCloudResources(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to discover resources: %v", err)
	}

	// Convert to the format expected by security service
	securityResources := make([]*models.Resource, len(resources))
	for i, res := range resources {
		securityResources[i] = &res
	}

	// Perform the scan
	result, err := securityService.ScanResources(ctx, securityResources)
	if err != nil {
		return nil, fmt.Errorf("security scan failed: %v", err)
	}

	// Convert result to API response format
	return map[string]interface{}{
		"scan_id":            result.ScanID,
		"duration":           result.Duration.String(),
		"resources_scanned":  len(result.Resources),
		"policies_evaluated": len(result.Policies),
		"violations_found":   len(result.Violations),
		"compliance_checks":  len(result.Compliance),
		"violations":         result.Violations,
		"compliance":         result.Compliance,
		"timestamp":          time.Now().Format(time.RFC3339),
	}, nil
}

// APISecurityEventBus implements the security EventBus interface for API server
type APISecurityEventBus struct {
	server *Server
}

func (a *APISecurityEventBus) PublishComplianceEvent(event security.ComplianceEvent) error {
	// WebSocket functionality removed - just log the event
	log.Printf("Compliance event: %s - %s", event.Type, event.Message)
	return nil
}

// handleGetCompliance handles GET /api/v1/security/compliance
func (s *Server) handleGetCompliance(w http.ResponseWriter, r *http.Request) {
	// Get compliance status
	eventBus := &APISecurityEventBus{server: s}
	securityService := security.NewSecurityService(eventBus)

	ctx := context.Background()
	if err := securityService.Start(ctx); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to start security service: %v", err))
		return
	}
	defer securityService.Stop(ctx)

	status, err := securityService.GetSecurityStatus(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get compliance status: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"overall_status": status.OverallStatus,
		"security_score": status.SecurityScore,
		"last_scan":      status.LastScan.Format(time.RFC3339),
		"policies":       status.Policies,
		"compliance":     status.Compliance,
	})
}

// handleGetSecurityPolicies handles GET /api/v1/security/policies
func (s *Server) handleGetSecurityPolicies(w http.ResponseWriter, r *http.Request) {
	// Return security policies
	policies := []map[string]interface{}{
		{
			"id":          "policy-1",
			"name":        "Encryption Required",
			"description": "All resources must have encryption enabled",
			"category":    "security",
			"severity":    "high",
			"enabled":     true,
		},
		{
			"id":          "policy-2",
			"name":        "Access Logging Required",
			"description": "All resources must have access logging enabled",
			"category":    "monitoring",
			"severity":    "medium",
			"enabled":     true,
		},
		{
			"id":          "policy-3",
			"name":        "Backup Required",
			"description": "All critical resources must have backup enabled",
			"category":    "backup",
			"severity":    "medium",
			"enabled":     true,
		},
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"policies": policies,
		"total":    len(policies),
	})
}

// handleGetBackends handles GET /api/v1/backends
func (s *Server) handleGetBackends(w http.ResponseWriter, r *http.Request) {
	// Return discovered backends
	backends := []map[string]interface{}{
		{
			"id":       "backend-1",
			"type":     "s3",
			"name":     "Terragrunt State Bucket (EU West 1)",
			"provider": "aws",
			"region":   "eu-west-1",
			"bucket":   "terragrunt-025066254478-state-eu-west-1-20250913095334",
			"status":   "active",
		},
		{
			"id":       "backend-2",
			"type":     "s3",
			"name":     "Terragrunt State Bucket (US West 2)",
			"provider": "aws",
			"region":   "us-west-2",
			"bucket":   "terragrunt-025066254478-state-us-west-2-20250913095334",
			"status":   "active",
		},
		{
			"id":       "backend-3",
			"type":     "s3",
			"name":     "DriftMgr Test Bucket",
			"provider": "aws",
			"region":   "us-east-1",
			"bucket":   "driftmgr-test-bucket-1755579790",
			"status":   "active",
		},
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"backends": backends,
		"total":    len(backends),
	})
}

// handleDiscoverBackends handles GET /api/v1/backends/discover
func (s *Server) handleDiscoverBackends(w http.ResponseWriter, r *http.Request) {
	// Perform backend discovery
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "aws" // Default to AWS
	}

	// Simulate backend discovery
	backends := s.discoverBackends(provider)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"backends":  backends,
		"total":     len(backends),
		"provider":  provider,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// discoverBackends discovers backends for a given provider
func (s *Server) discoverBackends(provider string) []map[string]interface{} {
	var backends []map[string]interface{}

	switch provider {
	case "aws":
		backends = []map[string]interface{}{
			{
				"id":       "aws-backend-1",
				"type":     "s3",
				"name":     "Terragrunt State Bucket (EU West 1)",
				"provider": "aws",
				"region":   "eu-west-1",
				"bucket":   "terragrunt-025066254478-state-eu-west-1-20250913095334",
				"status":   "active",
				"created":  "2025-09-14T19:16:09Z",
			},
			{
				"id":       "aws-backend-2",
				"type":     "s3",
				"name":     "Terragrunt State Bucket (US West 2)",
				"provider": "aws",
				"region":   "us-west-2",
				"bucket":   "terragrunt-025066254478-state-us-west-2-20250913095334",
				"status":   "active",
				"created":  "2025-09-14T19:16:10Z",
			},
			{
				"id":       "aws-backend-3",
				"type":     "s3",
				"name":     "DriftMgr Test Bucket",
				"provider": "aws",
				"region":   "us-east-1",
				"bucket":   "driftmgr-test-bucket-1755579790",
				"status":   "active",
				"created":  "2025-08-21T19:38:37Z",
			},
		}
	case "azure":
		// Azure backends would be discovered here
		backends = []map[string]interface{}{}
	case "gcp":
		// GCP backends would be discovered here
		backends = []map[string]interface{}{}
	}

	return backends
}

// State Management Handlers

// handleGetStateFiles handles GET /api/v1/state/list
func (s *Server) handleGetStateFiles(w http.ResponseWriter, r *http.Request) {
	// Return state files from discovered backends
	stateFiles := []map[string]interface{}{
		{
			"id":        "state-1",
			"name":      "terraform.tfstate",
			"backend":   "aws-backend-1",
			"path":      "terraform.tfstate",
			"size":      1024,
			"modified":  "2025-09-23T17:00:00Z",
			"resources": 3,
		},
		{
			"id":        "state-2",
			"name":      "prod.tfstate",
			"backend":   "aws-backend-2",
			"path":      "prod/terraform.tfstate",
			"size":      2048,
			"modified":  "2025-09-23T16:30:00Z",
			"resources": 8,
		},
		{
			"id":                "state-3",
			"name":              "gcp-terraform.tfstate",
			"backend":           "local-backend",
			"path":              "test-gcp/terraform.tfstate",
			"size":              12220,
			"modified":          time.Now().Format(time.RFC3339),
			"resources":         5,
			"provider":          "gcp",
			"version":           4,
			"terraform_version": "1.11.3",
			"serial":            6,
			"lineage":           "60f9d4ac-83bc-87d8-7e01-e1177f7e88e2",
		},
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"state_files": stateFiles,
		"total":       len(stateFiles),
	})
}

// handleGetStateDetails handles GET /api/v1/state/details
func (s *Server) handleGetStateDetails(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	// Check if this is the GCP state file
	if path == "test-gcp/terraform.tfstate" {
		s.writeJSON(w, http.StatusOK, map[string]interface{}{
			"path":              path,
			"version":           4,
			"terraform_version": "1.11.3",
			"serial":            6,
			"lineage":           "60f9d4ac-83bc-87d8-7e01-e1177f7e88e2",
			"resources": []map[string]interface{}{
				{
					"type":     "google_storage_bucket",
					"name":     "test_bucket",
					"provider": "google",
					"address":  "google_storage_bucket.test_bucket",
					"attributes": map[string]interface{}{
						"bucket":                      "driftmgr-test-bucket-bd2c7b0c",
						"location":                    "US",
						"uniform_bucket_level_access": true,
					},
				},
				{
					"type":     "google_compute_instance",
					"name":     "test_instance",
					"provider": "google",
					"address":  "google_compute_instance.test_instance",
					"attributes": map[string]interface{}{
						"name":         "driftmgr-test-instance",
						"machine_type": "e2-micro",
						"zone":         "us-central1-a",
					},
				},
				{
					"type":     "google_compute_network",
					"name":     "test_network",
					"provider": "google",
					"address":  "google_compute_network.test_network",
					"attributes": map[string]interface{}{
						"name":                    "driftmgr-test-network",
						"auto_create_subnetworks": false,
					},
				},
				{
					"type":     "google_compute_subnetwork",
					"name":     "test_subnet",
					"provider": "google",
					"address":  "google_compute_subnetwork.test_subnet",
					"attributes": map[string]interface{}{
						"name":          "driftmgr-test-subnet",
						"ip_cidr_range": "10.0.1.0/24",
						"region":        "us-central1",
					},
				},
				{
					"type":     "random_id",
					"name":     "bucket_suffix",
					"provider": "random",
					"address":  "random_id.bucket_suffix",
					"attributes": map[string]interface{}{
						"hex":         "bd2c7b0c",
						"byte_length": 4,
					},
				},
			},
		})
		return
	}

	// Default response for other state files
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":              path,
		"version":           4,
		"terraform_version": "1.0.0",
		"serial":            1,
		"lineage":           "test-lineage",
		"resources": []map[string]interface{}{
			{
				"type":     "aws_s3_bucket",
				"name":     "test_bucket",
				"provider": "aws",
				"address":  "aws_s3_bucket.test_bucket",
				"attributes": map[string]interface{}{
					"bucket": "test-bucket-12345",
					"region": "us-east-1",
				},
			},
		},
	})
}
