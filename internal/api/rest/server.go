package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/api/rest/handlers"
	"github.com/catherinevee/driftmgr/internal/api/rest/middleware"
	// "github.com/catherinevee/driftmgr/internal/api/websocket"
	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/drift"
	"github.com/catherinevee/driftmgr/internal/core/remediation"
	providers "github.com/catherinevee/driftmgr/internal/discovery"
	// "github.com/catherinevee/driftmgr/internal/storage"
	"github.com/gorilla/mux"
)

// Server represents the REST API server
type Server struct {
	router *mux.Router
	port   string

	// Core services
	discovery   *discovery.Service
	drift       *drift.Detector
	remediation *remediation.Engine

	// Dashboard server handles all the endpoints
	dashboardServer *handlers.EnhancedDashboardServer

	// WebSocket support
	// wsServer *websocket.Server

	// Server metadata
	startTime time.Time
	version   string
}

// Config represents server configuration
type Config struct {
	Port            string
	EnableWebSocket bool
	EnableMetrics   bool
	EnableSwagger   bool
	CORSOrigins     []string
	AuthEnabled     bool
	TLSCert         string
	TLSKey          string
}

// NewServer creates a new REST API server
func NewServer(config Config) *Server {
	if config.Port == "" {
		config.Port = "8080"
	}

	server := &Server{
		router:      mux.NewRouter(),
		port:        config.Port,
		discovery:   discovery.NewService(),
		drift:       drift.NewDetector(),
		remediation: remediation.NewEngine(),
		// dataStore:   storage.NewDataStore(),
		startTime: time.Now(),
		version:   "2.0.0",
	}

	// Initialize dashboard server
	server.dashboardServer = handlers.NewEnhancedDashboardServer(
		server.discovery,
		server.drift,
		server.remediation,
	)

	// Setup routes
	server.setupRoutes()

	// Setup middleware
	server.setupMiddleware(config)

	// Initialize WebSocket if enabled
	// if config.EnableWebSocket {
	// 	server.wsServer = websocket.NewServer()
	// }

	return server
}

// initializeHandlers initializes all request handlers
func (s *Server) initializeHandlers() {
	s.discoveryHandler = handlers.NewDiscoveryHandler(s.discovery, s.dataStore)
	s.driftHandler = handlers.NewDriftHandler(s.drift, s.dataStore)
	s.remediationHandler = handlers.NewRemediationHandler(s.remediation, s.dataStore)
	s.resourceHandler = handlers.NewResourceHandler(s.dataStore)
	s.credentialHandler = handlers.NewCredentialHandler(s.dataStore)
	s.analyticsHandler = handlers.NewAnalyticsHandler(s.dataStore)
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// API version prefix
	api := s.router.PathPrefix("/api/v2").Subrouter()

	// Health and status endpoints
	api.HandleFunc("/health", s.healthCheck).Methods("GET")
	api.HandleFunc("/status", s.getStatus).Methods("GET")
	api.HandleFunc("/version", s.getVersion).Methods("GET")

	// Discovery endpoints
	discovery := api.PathPrefix("/discovery").Subrouter()
	discovery.HandleFunc("/start", s.discoveryHandler.StartDiscovery).Methods("POST")
	discovery.HandleFunc("/jobs", s.discoveryHandler.ListJobs).Methods("GET")
	discovery.HandleFunc("/jobs/{jobId}", s.discoveryHandler.GetJob).Methods("GET")
	discovery.HandleFunc("/jobs/{jobId}/cancel", s.discoveryHandler.CancelJob).Methods("POST")
	discovery.HandleFunc("/providers", s.discoveryHandler.ListProviders).Methods("GET")
	discovery.HandleFunc("/providers/{provider}/status", s.discoveryHandler.GetProviderStatus).Methods("GET")

	// Resource endpoints
	resources := api.PathPrefix("/resources").Subrouter()
	resources.HandleFunc("", s.resourceHandler.ListResources).Methods("GET")
	resources.HandleFunc("/search", s.resourceHandler.SearchResources).Methods("POST")
	resources.HandleFunc("/bulk", s.resourceHandler.BulkOperation).Methods("POST")
	resources.HandleFunc("/{resourceId}", s.resourceHandler.GetResource).Methods("GET")
	resources.HandleFunc("/{resourceId}", s.resourceHandler.UpdateResource).Methods("PUT")
	resources.HandleFunc("/{resourceId}", s.resourceHandler.DeleteResource).Methods("DELETE")
	resources.HandleFunc("/{resourceId}/dependencies", s.resourceHandler.GetDependencies).Methods("GET")

	// Drift endpoints
	driftRoutes := api.PathPrefix("/drift").Subrouter()
	driftRoutes.HandleFunc("/detect", s.driftHandler.DetectDrift).Methods("POST")
	driftRoutes.HandleFunc("/items", s.driftHandler.ListDriftItems).Methods("GET")
	driftRoutes.HandleFunc("/items/{driftId}", s.driftHandler.GetDriftItem).Methods("GET")
	driftRoutes.HandleFunc("/analyze", s.driftHandler.AnalyzeDrift).Methods("POST")
	driftRoutes.HandleFunc("/history", s.driftHandler.GetHistory).Methods("GET")
	driftRoutes.HandleFunc("/patterns", s.driftHandler.GetPatterns).Methods("GET")
	driftRoutes.HandleFunc("/predict", s.driftHandler.PredictDrift).Methods("POST")

	// Remediation endpoints
	remediationRoutes := api.PathPrefix("/remediation").Subrouter()
	remediationRoutes.HandleFunc("/plans", s.remediationHandler.CreatePlan).Methods("POST")
	remediationRoutes.HandleFunc("/plans", s.remediationHandler.ListPlans).Methods("GET")
	remediationRoutes.HandleFunc("/plans/{planId}", s.remediationHandler.GetPlan).Methods("GET")
	remediationRoutes.HandleFunc("/plans/{planId}/approve", s.remediationHandler.ApprovePlan).Methods("POST")
	remediationRoutes.HandleFunc("/plans/{planId}/execute", s.remediationHandler.ExecutePlan).Methods("POST")
	remediationRoutes.HandleFunc("/plans/{planId}/rollback", s.remediationHandler.RollbackPlan).Methods("POST")
	remediationRoutes.HandleFunc("/history", s.remediationHandler.GetHistory).Methods("GET")

	// Credential endpoints
	credentials := api.PathPrefix("/credentials").Subrouter()
	credentials.HandleFunc("/providers", s.credentialHandler.ListProviders).Methods("GET")
	credentials.HandleFunc("/providers/{provider}/configure", s.credentialHandler.ConfigureProvider).Methods("POST")
	credentials.HandleFunc("/providers/{provider}/test", s.credentialHandler.TestCredentials).Methods("POST")
	credentials.HandleFunc("/providers/{provider}/status", s.credentialHandler.GetProviderStatus).Methods("GET")
	credentials.HandleFunc("/providers/{provider}", s.credentialHandler.DeleteProvider).Methods("DELETE")

	// Analytics endpoints
	analytics := api.PathPrefix("/analytics").Subrouter()
	analytics.HandleFunc("/summary", s.analyticsHandler.GetSummary).Methods("GET")
	analytics.HandleFunc("/trends", s.analyticsHandler.GetTrends).Methods("GET")
	analytics.HandleFunc("/costs", s.analyticsHandler.GetCostAnalysis).Methods("GET")
	analytics.HandleFunc("/compliance", s.analyticsHandler.GetComplianceReport).Methods("GET")
	analytics.HandleFunc("/utilization", s.analyticsHandler.GetUtilization).Methods("GET")
	analytics.HandleFunc("/reports/generate", s.analyticsHandler.GenerateReport).Methods("POST")
	analytics.HandleFunc("/reports/{reportId}", s.analyticsHandler.GetReport).Methods("GET")

	// WebSocket endpoint
	if s.wsServer != nil {
		s.router.HandleFunc("/ws", s.wsServer.HandleConnection)
	}

	// Static files (if needed for UI)
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/dist/")))
}

// setupMiddleware configures middleware
func (s *Server) setupMiddleware(config Config) {
	// Logging middleware
	s.router.Use(middleware.LoggingMiddleware)

	// Recovery middleware
	s.router.Use(middleware.RecoveryMiddleware)

	// CORS middleware
	if len(config.CORSOrigins) > 0 {
		s.router.Use(middleware.CORSMiddleware(config.CORSOrigins))
	}

	// Authentication middleware
	if config.AuthEnabled {
		s.router.Use(middleware.AuthMiddleware)
	}

	// Rate limiting middleware
	s.router.Use(middleware.RateLimitMiddleware)

	// Metrics middleware
	if config.EnableMetrics {
		s.router.Use(middleware.MetricsMiddleware)
	}
}

// Start starts the API server
func (s *Server) Start() error {
	log.Printf("Starting REST API server on port %s", s.port)
	log.Printf("API version: %s", s.version)

	// Initialize providers
	go s.initializeProviders()

	// Start WebSocket server if enabled
	if s.wsServer != nil {
		go s.wsServer.Start()
	}

	// Start HTTP server
	server := &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServe()
}

// StartTLS starts the API server with TLS
func (s *Server) StartTLS(certFile, keyFile string) error {
	log.Printf("Starting REST API server with TLS on port %s", s.port)

	server := &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server.ListenAndServeTLS(certFile, keyFile)
}

// initializeProviders initializes cloud providers
func (s *Server) initializeProviders() {
	ctx := context.Background()

	// Register providers with discovery service
	// This would be done based on configuration
	log.Println("Initializing cloud providers...")

	// Register cloud providers
	awsProvider, err := providers.NewAWSProvider()
	if err != nil {
		log.Printf("Warning: Failed to initialize AWS provider: %v", err)
	} else {
		s.discovery.RegisterProvider("aws", awsProvider)
		log.Println("AWS provider registered")
	}

	azureProvider, err := providers.NewAzureProvider()
	if err != nil {
		log.Printf("Warning: Failed to initialize Azure provider: %v", err)
	} else {
		s.discovery.RegisterProvider("azure", azureProvider)
		log.Println("Azure provider registered")
	}

	gcpProvider, err := providers.NewGCPProvider()
	if err != nil {
		log.Printf("Warning: Failed to initialize GCP provider: %v", err)
	} else {
		s.discovery.RegisterProvider("gcp", gcpProvider)
		log.Println("GCP provider registered")
	}

	digitalOceanProvider, err := providers.NewDigitalOceanProvider()
	if err != nil {
		log.Printf("Warning: Failed to initialize DigitalOcean provider: %v", err)
	} else {
		s.discovery.RegisterProvider("digitalocean", digitalOceanProvider)
		log.Println("DigitalOcean provider registered")
	}

	log.Println("Cloud providers initialization complete")
}

// Health check endpoints

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(s.startTime).Seconds(),
		"version":   s.version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (s *Server) getStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"server": map[string]interface{}{
			"version":   s.version,
			"uptime":    time.Since(s.startTime).Seconds(),
			"startTime": s.startTime,
		},
		"services": map[string]interface{}{
			"discovery":   s.discovery != nil,
			"drift":       s.drift != nil,
			"remediation": s.remediation != nil,
			"websocket":   s.wsServer != nil,
		},
		"resources": map[string]interface{}{
			"total":     s.dataStore.GetResourceCount(),
			"providers": s.discovery.GetProviders(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	version := map[string]interface{}{
		"version":    s.version,
		"apiVersion": "v2",
		"buildTime":  s.startTime,
		"features": []string{
			"multi-cloud-discovery",
			"drift-detection",
			"auto-remediation",
			"real-time-updates",
			"cost-analysis",
			"compliance-monitoring",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down REST API server...")

	// Stop WebSocket server
	if s.wsServer != nil {
		s.wsServer.Shutdown()
	}

	// Clean up resources
	s.dataStore.Clear()

	return nil
}
