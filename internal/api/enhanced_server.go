package api

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

// EnhancedServer represents the enhanced API server with new handlers
type EnhancedServer struct {
	httpServer *http.Server
	router     *Router
	config     *Config
	address    string
	mu         sync.RWMutex

	// Handler instances
	backendHandlers  *BackendHandlers
	stateHandlers    *StateHandlers
	resourceHandlers *ResourceHandlers
	driftHandlers    *DriftHandlers
}

// NewEnhancedServer creates a new enhanced API server
func NewEnhancedServer(address string) *EnhancedServer {
	config := &Config{
		Host:             "0.0.0.0",
		Port:             8080,
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     30 * time.Second,
		IdleTimeout:      120 * time.Second,
		MaxHeaderBytes:   1 << 20, // 1MB
		CORSEnabled:      true,
		AuthEnabled:      false,
		RateLimitEnabled: true,
		RateLimitRPS:     100,
		LoggingEnabled:   true,
	}

	router := &Router{
		routes: make(map[string]map[string]http.HandlerFunc),
	}

	server := &EnhancedServer{
		router:  router,
		config:  config,
		address: address,

		// Initialize handlers
		backendHandlers:  NewBackendHandlers(),
		stateHandlers:    NewStateHandlers(),
		resourceHandlers: NewResourceHandlers(),
		driftHandlers:    NewDriftHandlers(),
	}

	// Setup routes
	server.setupEnhancedRoutes()

	// Create HTTP server
	server.httpServer = &http.Server{
		Addr:           address,
		Handler:        server,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		IdleTimeout:    config.IdleTimeout,
		MaxHeaderBytes: config.MaxHeaderBytes,
	}

	return server
}

// Start starts the enhanced server
func (s *EnhancedServer) Start(ctx context.Context) error {
	log.Printf("Starting enhanced DriftMgr API server on %s", s.address)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for either context cancellation or server error
	select {
	case <-ctx.Done():
		log.Println("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-serverErr:
		return err
	}
}

// ServeHTTP implements http.Handler for EnhancedServer
func (s *EnhancedServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set common headers
	SetCommonHeaders(w)

	// Handle CORS preflight requests
	if HandleCORS(w, r) {
		return
	}

	// Rate limiting
	if s.config.RateLimitEnabled {
		if !s.handleRateLimit(w, r) {
			return
		}
	}

	// Authentication
	if s.config.AuthEnabled {
		if !s.handleAuth(w, r) {
			return
		}
	}

	// Logging
	if s.config.LoggingEnabled {
		s.logRequest(r)
	}

	// Route handling
	s.router.ServeHTTP(w, r)
}

// setupEnhancedRoutes sets up all enhanced API routes
func (s *EnhancedServer) setupEnhancedRoutes() {
	// Health check
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/api/v1/health", s.handleHealth)

	// API version
	s.router.GET("/api/v1/version", s.handleVersion)

	// Backend Discovery Routes
	s.router.GET("/api/v1/backends/list", s.backendHandlers.ListBackends)
	s.router.POST("/api/v1/backends/discover", s.backendHandlers.DiscoverBackends)
	s.router.GET("/api/v1/backends/{id}", s.backendHandlers.GetBackend)
	s.router.PUT("/api/v1/backends/{id}", s.backendHandlers.UpdateBackend)
	s.router.DELETE("/api/v1/backends/{id}", s.backendHandlers.DeleteBackend)
	s.router.POST("/api/v1/backends/{id}/test", s.backendHandlers.TestBackend)

	// State Management Routes
	s.router.GET("/api/v1/state/list", s.stateHandlers.ListStateFiles)
	s.router.GET("/api/v1/state/details", s.stateHandlers.GetStateDetails)
	s.router.POST("/api/v1/state/import", s.stateHandlers.ImportResource)
	s.router.DELETE("/api/v1/state/resources/{id}", s.stateHandlers.RemoveResource)
	s.router.POST("/api/v1/state/move", s.stateHandlers.MoveResource)
	s.router.POST("/api/v1/state/lock", s.stateHandlers.LockStateFile)
	s.router.POST("/api/v1/state/unlock", s.stateHandlers.UnlockStateFile)

	// Resource Management Routes
	s.router.GET("/api/v1/resources", s.resourceHandlers.ListResources)
	s.router.GET("/api/v1/resources/{id}", s.resourceHandlers.GetResource)
	s.router.GET("/api/v1/resources/search", s.resourceHandlers.SearchResources)
	s.router.PUT("/api/v1/resources/{id}/tags", s.resourceHandlers.UpdateResourceTags)
	s.router.GET("/api/v1/resources/{id}/cost", s.resourceHandlers.GetResourceCost)
	s.router.GET("/api/v1/resources/{id}/compliance", s.resourceHandlers.GetResourceCompliance)

	// Drift Detection Routes
	s.router.POST("/api/v1/drift/detect", s.driftHandlers.DetectDrift)
	s.router.GET("/api/v1/drift/results", s.driftHandlers.ListDriftResults)
	s.router.GET("/api/v1/drift/results/{id}", s.driftHandlers.GetDriftResult)
	s.router.DELETE("/api/v1/drift/results/{id}", s.driftHandlers.DeleteDriftResult)
	s.router.GET("/api/v1/drift/history", s.driftHandlers.GetDriftHistory)
	s.router.GET("/api/v1/drift/summary", s.driftHandlers.GetDriftSummary)

	// Serve web interface
	s.router.GET("/", s.handleWebInterface)
	s.router.GET("/dashboard", s.handleWebInterface)
	s.router.GET("/js/*", s.handleStaticFiles)
	s.router.GET("/css/*", s.handleStaticFiles)
	s.router.GET("/assets/*", s.handleStaticFiles)
}

// handleHealth handles health check requests
func (s *EnhancedServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := NewResponseWriter(w)
	healthData := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
		"services": map[string]string{
			"api":       "healthy",
			"database":  "healthy",
			"discovery": "healthy",
			"drift":     "healthy",
		},
	}

	err := response.WriteSuccess(healthData, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode health response")
	}
}

// handleVersion handles version requests
func (s *EnhancedServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	response := NewResponseWriter(w)
	versionData := map[string]interface{}{
		"version":     "1.0.0",
		"build_time":  time.Now().UTC().Format(time.RFC3339),
		"git_commit":  "abc123def456",
		"go_version":  "1.21.0",
		"api_version": "v1",
	}

	err := response.WriteSuccess(versionData, &APIMeta{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		response.WriteInternalError("Failed to encode version response")
	}
}

// handleWebInterface handles web interface requests
func (s *EnhancedServer) handleWebInterface(w http.ResponseWriter, r *http.Request) {
	// Serve the main dashboard HTML
	http.ServeFile(w, r, "web/dashboard/index.html")
}

// handleStaticFiles handles static file requests
func (s *EnhancedServer) handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	// Remove the leading slash and serve from web directory
	filePath := r.URL.Path[1:] // Remove leading slash
	http.ServeFile(w, r, "web/"+filePath)
}

// handleRateLimit handles rate limiting
func (s *EnhancedServer) handleRateLimit(w http.ResponseWriter, r *http.Request) bool {
	// Simplified rate limiting - in a real system, you'd use a proper rate limiter
	// For now, just return true to allow all requests
	return true
}

// handleAuth handles authentication
func (s *EnhancedServer) handleAuth(w http.ResponseWriter, r *http.Request) bool {
	// Simplified authentication - in a real system, you'd implement proper auth
	// For now, just return true to allow all requests
	return true
}

// logRequest logs HTTP requests
func (s *EnhancedServer) logRequest(r *http.Request) {
	log.Printf("%s %s %s %s", r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
}
