package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/analytics"
	"github.com/catherinevee/driftmgr/internal/automation"
	"github.com/catherinevee/driftmgr/internal/bi"
	"github.com/catherinevee/driftmgr/internal/cost"
	"github.com/catherinevee/driftmgr/internal/remediation"
	"github.com/catherinevee/driftmgr/internal/security"
	"github.com/catherinevee/driftmgr/internal/tenant"
)

// Server represents the API server
type Server struct {
	httpServer *http.Server
	router     *Router
	services   *Services
	config     *Config
	mu         sync.RWMutex
}

// Services represents all available services
type Services struct {
	Analytics   *analytics.AnalyticsService
	Automation  *automation.AutomationService
	BI          *bi.BIService
	Cost        *cost.CostAnalyzer
	Remediation *remediation.IntelligentRemediationService
	Security    *security.SecurityService
	Tenant      *tenant.TenantService
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
}

// Router represents the HTTP router
type Router struct {
	routes map[string]map[string]http.HandlerFunc
	mu     sync.RWMutex
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
	// Health check
	s.router.GET("/health", s.handleHealth)

	// API version
	s.router.GET("/api/v1/version", s.handleVersion)

	// Resources
	s.router.GET("/api/v1/resources", s.handleGetResources)
	s.router.GET("/api/v1/resources/{id}", s.handleGetResource)
	s.router.POST("/api/v1/resources", s.handleCreateResource)
	s.router.PUT("/api/v1/resources/{id}", s.handleUpdateResource)
	s.router.DELETE("/api/v1/resources/{id}", s.handleDeleteResource)

	// Drift detection
	s.router.POST("/api/v1/drift/detect", s.handleDetectDrift)
	s.router.GET("/api/v1/drift/reports", s.handleGetDriftReports)
	s.router.GET("/api/v1/drift/reports/{id}", s.handleGetDriftReport)

	// Health monitoring
	s.router.GET("/api/v1/health", s.handleGetHealthStatus)
	s.router.GET("/api/v1/health/checks", s.handleGetHealthChecks)
	s.router.POST("/api/v1/health/checks", s.handleCreateHealthCheck)

	// Cost analysis
	s.router.GET("/api/v1/cost/analysis", s.handleGetCostAnalysis)
	s.router.GET("/api/v1/cost/optimization", s.handleGetCostOptimization)
	s.router.GET("/api/v1/cost/forecast", s.handleGetCostForecast)

	// Remediation
	s.router.POST("/api/v1/remediation/plan", s.handleCreateRemediationPlan)
	s.router.POST("/api/v1/remediation/execute", s.handleExecuteRemediation)
	s.router.GET("/api/v1/remediation/plans", s.handleGetRemediationPlans)

	// Security
	s.router.GET("/api/v1/security/scan", s.handleSecurityScan)
	s.router.GET("/api/v1/security/compliance", s.handleGetCompliance)
	s.router.GET("/api/v1/security/policies", s.handleGetSecurityPolicies)

	// Analytics
	s.router.GET("/api/v1/analytics/models", s.handleGetAnalyticsModels)
	s.router.POST("/api/v1/analytics/forecast", s.handleGenerateForecast)
	s.router.GET("/api/v1/analytics/trends", s.handleGetTrends)
	s.router.GET("/api/v1/analytics/anomalies", s.handleGetAnomalies)

	// Automation
	s.router.GET("/api/v1/automation/workflows", s.handleGetWorkflows)
	s.router.POST("/api/v1/automation/workflows", s.handleCreateWorkflow)
	s.router.POST("/api/v1/automation/workflows/{id}/execute", s.handleExecuteWorkflow)
	s.router.GET("/api/v1/automation/rules", s.handleGetRules)
	s.router.POST("/api/v1/automation/rules", s.handleCreateRule)

	// Business Intelligence
	s.router.GET("/api/v1/bi/dashboards", s.handleGetDashboards)
	s.router.POST("/api/v1/bi/dashboards", s.handleCreateDashboard)
	s.router.GET("/api/v1/bi/reports", s.handleGetReports)
	s.router.POST("/api/v1/bi/reports", s.handleCreateReport)
	s.router.GET("/api/v1/bi/queries", s.handleGetQueries)
	s.router.POST("/api/v1/bi/queries", s.handleCreateQuery)
	s.router.POST("/api/v1/bi/queries/{id}/execute", s.handleExecuteQuery)

	// Multi-tenant
	s.router.GET("/api/v1/tenants", s.handleGetTenants)
	s.router.POST("/api/v1/tenants", s.handleCreateTenant)
	s.router.GET("/api/v1/tenants/{id}/accounts", s.handleGetTenantAccounts)
	s.router.POST("/api/v1/tenants/{id}/accounts", s.handleAddTenantAccount)

	// WebSocket endpoints
	s.router.GET("/ws/health", s.handleHealthWebSocket)
	s.router.GET("/ws/drift", s.handleDriftWebSocket)
	s.router.GET("/ws/automation", s.handleAutomationWebSocket)
}
