package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/analytics"
	"github.com/catherinevee/driftmgr/internal/auth"
	"github.com/catherinevee/driftmgr/internal/automation"
	"github.com/catherinevee/driftmgr/internal/bi"
	"github.com/catherinevee/driftmgr/internal/cost"
	"github.com/catherinevee/driftmgr/internal/remediation"
	"github.com/catherinevee/driftmgr/internal/repositories"
	"github.com/catherinevee/driftmgr/internal/security"
	"github.com/catherinevee/driftmgr/internal/services"
	"github.com/catherinevee/driftmgr/internal/tenant"
	"github.com/catherinevee/driftmgr/internal/websocket"
)

// Server represents the API server
type Server struct {
	httpServer *http.Server
	router     *Router
	services   *Services
	config     *Config
	address    string
	mu         sync.RWMutex
}

// Services represents all available services
type Services struct {
	Auth           *auth.Service
	Analytics      *analytics.AnalyticsService
	Automation     *automation.AutomationService
	BI             *bi.BIService
	Cost           *cost.CostAnalyzer
	Remediation    *remediation.IntelligentRemediationService
	Security       *security.SecurityService
	Tenant         *tenant.TenantService
	WebSocket      *websocket.Service
	BackendService *services.BackendService
	StateService   *services.StateService
	ResourceService *services.ResourceService
	DriftService   *services.DriftService
}

// Config represents server configuration
type Config struct {
	Host             string        `json:"host"`
	Port             int           `json:"port"`
	ReadTimeout      time.Duration `json:"read_timeout"`
	WriteTimeout     time.Duration `json:"write_timeout"`
	IdleTimeout      time.Duration `json:"idle_timeout"`
	MaxHeaderBytes   int           `json:"max_header_bytes"`
	CORSEnabled      bool          `json:"cors_enabled"`
	AuthEnabled      bool          `json:"auth_enabled"`
	RateLimitEnabled bool          `json:"rate_limit_enabled"`
	RateLimitRPS     int           `json:"rate_limit_rps"`
	LoggingEnabled   bool          `json:"logging_enabled"`

	// Authentication configuration
	JWTSecret          string        `json:"jwt_secret"`
	JWTIssuer          string        `json:"jwt_issuer"`
	JWTAudience        string        `json:"jwt_audience"`
	AccessTokenExpiry  time.Duration `json:"access_token_expiry"`
	RefreshTokenExpiry time.Duration `json:"refresh_token_expiry"`
}

// Router represents the HTTP router
type Router struct {
	routes map[string]map[string]http.HandlerFunc
	mu     sync.RWMutex
}

// NewAPIServer creates a new API server with default configuration
func NewAPIServer(address string) *Server {
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

	server := &Server{
		router:  router,
		config:  config,
		address: address,
	}

	// Setup routes
	server.setupRoutes()

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

// NewServer creates a new API server
func NewServer(config *Config, services *Services) *Server {
	if config == nil {
		config = &Config{
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

			// Default authentication configuration
			JWTSecret:          "driftmgr-secret-key-change-in-production",
			JWTIssuer:          "driftmgr",
			JWTAudience:        "driftmgr-api",
			AccessTokenExpiry:  15 * time.Minute,
			RefreshTokenExpiry: 7 * 24 * time.Hour,
		}
	}

	router := &Router{
		routes: make(map[string]map[string]http.HandlerFunc),
	}

	server := &Server{
		router:   router,
		services: services,
		config:   config,
	}

	// Initialize authentication services if enabled
	if config.AuthEnabled && services.Auth == nil {
		server.initializeAuth()
	}

	// Initialize WebSocket service
	if services.WebSocket == nil {
		server.initializeWebSocket()
	}

	// Initialize business logic services
	server.initializeBusinessServices()

	// Setup routes
	server.setupRoutes()

	// Create HTTP server
	server.httpServer = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:        server,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		IdleTimeout:    config.IdleTimeout,
		MaxHeaderBytes: config.MaxHeaderBytes,
	}

	return server
}

// initializeAuth initializes authentication services
func (s *Server) initializeAuth() {
	// Create repositories
	userRepo := auth.NewMemoryUserRepository()
	roleRepo := auth.NewMemoryRoleRepository()
	sessionRepo := auth.NewMemorySessionRepository()
	apiKeyRepo := auth.NewMemoryAPIKeyRepository()

	// Create JWT service
	jwtService := auth.NewJWTService(
		s.config.JWTSecret,
		s.config.JWTIssuer,
		s.config.JWTAudience,
		s.config.AccessTokenExpiry,
		s.config.RefreshTokenExpiry,
	)

	// Create password service
	passwordService := auth.NewPasswordService()

	// Create auth service
	authService := auth.NewService(
		userRepo,
		roleRepo,
		sessionRepo,
		apiKeyRepo,
		jwtService,
		passwordService,
	)

	// Set auth service
	s.services.Auth = authService
}

// initializeWebSocket initializes the WebSocket service
func (s *Server) initializeWebSocket() {
	// Create WebSocket service
	wsService := websocket.NewService()

	// Set WebSocket service
	s.services.WebSocket = wsService
}

// initializeBusinessServices initializes the business logic services
func (s *Server) initializeBusinessServices() {
	// Create repositories
	backendRepo := repositories.NewMemoryBackendRepository()
	stateRepo := repositories.NewMemoryStateRepository()
	resourceRepo := repositories.NewMemoryResourceRepository()
	driftRepo := repositories.NewMemoryDriftRepository()

	// Create services
	backendService := services.NewBackendService(backendRepo)
	stateService := services.NewStateService(stateRepo, backendService)
	resourceService := services.NewResourceService(resourceRepo)
	driftService := services.NewDriftService(driftRepo, stateService, resourceService)

	// Set services
	s.services.BackendService = backendService
	s.services.StateService = stateService
	s.services.ResourceService = resourceService
	s.services.DriftService = driftService
}

// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	if s.config.LoggingEnabled {
		log.Printf("Starting API server on %s:%d", s.config.Host, s.config.Port)
	}

	// Start server in goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the API server
func (s *Server) Stop(ctx context.Context) error {
	if s.config.LoggingEnabled {
		log.Println("Stopping API server...")
	}

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	if s.config.LoggingEnabled {
		log.Println("API server stopped")
	}

	return nil
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS handling
	if s.config.CORSEnabled {
		s.handleCORS(w, r)
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

// setupRoutes sets up all API routes
func (s *Server) setupRoutes() {
	// Create enhanced handlers with real services
	backendHandlers := NewBackendHandlers(s.services.BackendService)
	stateHandlers := NewStateHandlers(s.services.StateService)
	resourceHandlers := NewResourceHandlers(s.services.ResourceService)
	driftHandlers := NewDriftHandlers(s.services.DriftService)

	// Health check
	s.router.GET("/health", s.handleHealth)
	s.router.GET("/api/v1/health", s.handleHealth)

	// API version
	s.router.GET("/api/v1/version", s.handleVersion)

	// Authentication routes
	if s.config.AuthEnabled && s.services.Auth != nil {
		s.setupAuthRoutes()
	}

	// Backend Discovery Routes
	s.router.GET("/api/v1/backends/list", backendHandlers.ListBackends)
	s.router.POST("/api/v1/backends/discover", backendHandlers.DiscoverBackends)
	s.router.GET("/api/v1/backends/{id}", backendHandlers.GetBackend)
	s.router.PUT("/api/v1/backends/{id}", backendHandlers.UpdateBackend)
	s.router.DELETE("/api/v1/backends/{id}", backendHandlers.DeleteBackend)
	s.router.POST("/api/v1/backends/{id}/test", backendHandlers.TestBackend)

	// State Management Routes
	s.router.GET("/api/v1/state/list", stateHandlers.ListStateFiles)
	s.router.GET("/api/v1/state/details", stateHandlers.GetStateDetails)
	s.router.POST("/api/v1/state/import", stateHandlers.ImportResource)
	s.router.DELETE("/api/v1/state/resources/{id}", stateHandlers.RemoveResource)
	s.router.POST("/api/v1/state/move", stateHandlers.MoveResource)
	s.router.POST("/api/v1/state/lock", stateHandlers.LockStateFile)
	s.router.POST("/api/v1/state/unlock", stateHandlers.UnlockStateFile)

	// Resource Management Routes
	s.router.GET("/api/v1/resources", resourceHandlers.ListResources)
	s.router.GET("/api/v1/resources/{id}", resourceHandlers.GetResource)
	s.router.GET("/api/v1/resources/search", resourceHandlers.SearchResources)
	s.router.PUT("/api/v1/resources/{id}/tags", resourceHandlers.UpdateResourceTags)
	s.router.GET("/api/v1/resources/{id}/cost", resourceHandlers.GetResourceCost)
	s.router.GET("/api/v1/resources/{id}/compliance", resourceHandlers.GetResourceCompliance)

	// Drift Detection Routes
	s.router.POST("/api/v1/drift/detect", driftHandlers.DetectDrift)
	s.router.GET("/api/v1/drift/results", driftHandlers.ListDriftResults)
	s.router.GET("/api/v1/drift/results/{id}", driftHandlers.GetDriftResult)
	s.router.DELETE("/api/v1/drift/results/{id}", driftHandlers.DeleteDriftResult)
	s.router.GET("/api/v1/drift/history", driftHandlers.GetDriftHistory)
	s.router.GET("/api/v1/drift/summary", driftHandlers.GetDriftSummary)

	// WebSocket routes
	if s.services.WebSocket != nil {
		wsHandlers := s.services.WebSocket.GetHandlers()
		s.router.GET("/ws", wsHandlers.HandleWebSocket)
		s.router.GET("/api/v1/ws", wsHandlers.HandleWebSocket)
		s.router.GET("/api/v1/ws/stats", s.handleWebSocketStats)
	}

	// Serve web interface
	s.router.GET("/", s.handleLoginPage)
	s.router.GET("/login", s.handleLoginPage)
	s.router.GET("/dashboard", s.handleWebInterface)
	s.router.GET("/js/*", s.handleStaticFiles)
	s.router.GET("/css/*", s.handleStaticFiles)
	s.router.GET("/assets/*", s.handleStaticFiles)
}

// setupAuthRoutes sets up authentication routes
func (s *Server) setupAuthRoutes() {
	// Create auth handlers and middleware
	authHandlers := auth.NewAuthHandlers(s.services.Auth)
	authMiddleware := auth.NewAuthMiddleware(s.services.Auth, s.services.Auth.JWTService())

	// Public authentication routes
	s.router.POST("/api/v1/auth/login", authHandlers.Login)
	s.router.POST("/api/v1/auth/register", authHandlers.Register)
	s.router.POST("/api/v1/auth/refresh", authHandlers.RefreshToken)
	s.router.POST("/api/v1/auth/logout", authMiddleware.RequireAuth(authHandlers.Logout))

	// Protected user profile routes
	s.router.GET("/api/v1/auth/profile", authMiddleware.RequireAuth(authHandlers.GetProfile))
	s.router.PUT("/api/v1/auth/profile", authMiddleware.RequireAuth(authHandlers.UpdateProfile))
	s.router.POST("/api/v1/auth/change-password", authMiddleware.RequireAuth(authHandlers.ChangePassword))

	// API key management routes
	s.router.POST("/api/v1/auth/api-keys", authMiddleware.RequireAuth(authHandlers.CreateAPIKey))
	s.router.GET("/api/v1/auth/api-keys", authMiddleware.RequireAuth(authHandlers.ListAPIKeys))
	s.router.DELETE("/api/v1/auth/api-keys/{id}", authMiddleware.RequireAuth(authHandlers.DeleteAPIKey))

	// OAuth2 routes
	s.router.GET("/api/v1/auth/oauth2/providers", authHandlers.GetOAuth2Providers)
	s.router.POST("/api/v1/auth/oauth2/{provider}/callback", authHandlers.OAuth2Callback)

	// Health check for auth service
	s.router.GET("/api/v1/auth/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    map[string]string{"status": "healthy"},
		})
	})
}

// handleWebSocketStats handles WebSocket connection statistics
func (s *Server) handleWebSocketStats(w http.ResponseWriter, r *http.Request) {
	if s.services.WebSocket == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "WebSocket service not available",
		})
		return
	}

	stats := s.services.WebSocket.GetConnectionStats()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}
