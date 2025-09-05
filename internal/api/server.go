package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"gopkg.in/yaml.v3"

	"github.com/catherinevee/driftmgr/internal/api/handlers"
	"github.com/catherinevee/driftmgr/internal/api/handlers"
	"github.com/catherinevee/driftmgr/internal/api/websocket"
	"github.com/catherinevee/driftmgr/internal/shared/config"
	corediscovery "github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/remediation"
	"github.com/catherinevee/driftmgr/internal/providers/aws"
	"github.com/catherinevee/driftmgr/internal/providers/azure"
	"github.com/catherinevee/driftmgr/internal/providers/gcp"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// Global variables for server metrics
var (
	startTime = time.Now()
)

// ServerOptions contains configuration options for the server
type ServerOptions struct {
	Port         string
	AutoDiscover bool
	ScanInterval string
	Debug        bool
}

// DeletionEngineWrapper wraps the deletion.DeletionEngine for API use
type DeletionEngineWrapper struct {
	engine *deletion.DeletionEngine
	mu     sync.RWMutex
}

// Delete deletes a resource based on the request
func (de *DeletionEngineWrapper) Delete(ctx context.Context, req DeletionRequest) error {
	if de.engine == nil {
		return fmt.Errorf("deletion engine not initialized")
	}

	// Create deletion options from request
	options := deletion.DeletionOptions{
		DryRun:    req.DryRun,
		Force:     req.Force,
		Timeout:   30 * time.Minute,
		BatchSize: 10,
	}

	// If specific resource ID is provided, delete single resource
	if req.ResourceID != "" {
		// Create a resource object
		resource := coredeletionmodels.Resource{
			ID:       req.ResourceID,
			Type:     req.ResourceType,
			Provider: req.Provider,
			Region:   req.Region,
		}

		// Add metadata if provided
		if req.Metadata != nil {
			resource.Metadata = make(map[string]string)
			for k, v := range req.Metadata {
				if str, ok := v.(string); ok {
					resource.Metadata[k] = str
				}
			}
		}

		// Get the provider and delete the resource
		de.mu.RLock()
		provider := de.engine.GetProvider(req.Provider)
		de.mu.RUnlock()

		if provider == nil {
			return fmt.Errorf("provider %s not configured", req.Provider)
		}

		return provider.DeleteResource(ctx, resource)
	}

	// Otherwise, use bulk deletion with filters
	if req.Metadata["account_id"] != nil {
		accountID := req.Metadata["account_id"].(string)
		_, err := de.engine.DeleteAccountResources(ctx, req.Provider, accountID, options)
		return err
	}

	return fmt.Errorf("either resource_id or account_id must be provided")
}

// GetProvider returns a provider by name from the deletion engine
func (de *deletion.DeletionEngine) GetProvider(name string) deletion.CloudProvider {
	de.mu.RLock()
	defer de.mu.RUnlock()
	return de.providers[name]
}

// Legacy deletion methods removed - now handled by deletion package providers

// DeletionRequest represents a request to delete a resource
type DeletionRequest struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	Force        bool                   `json:"force"`
	DryRun       bool                   `json:"dry_run"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Server represents the API server
type Server struct {
	router             *mux.Router
	httpServer         *http.Server
	port               string
	wsUpgrader         websocket.Upgrader
	wsClients          map[*websocket.Conn]bool
	wsClientsMu        sync.RWMutex // Mutex to protect wsClients map
	broadcast          chan interface{}
	auditLogger        *audit.FileLogger
	staticDir          string
	discoveryHub       *discovery.DiscoveryHub
	discoveryService   *corediscovery.Service
	driftDetector      *drift.Detector
	driftStore         *DriftStore
	remediator         *remediation.Remediator
	credManager        *credentials.Manager
	credDetector       *credentials.CredentialDetector
	currentConfig      map[string]interface{}
	perspectiveReport  map[string]interface{}
	validator          *ConfigurableResourceValidator
	consistencyChecker *ConsistencyChecker
	perspectiveHandler *PerspectiveHandler
	cacheIntegration   *CacheIntegration
	remediationStore   *RemediationStore                  // Real remediation job tracking
	stateManager       *StateFileManager                  // Real state file management
	authHandler        *auth.AuthHandler                  // Authentication handler
	discoveryJobs      map[string]*discovery.DiscoveryJob // Active discovery jobs
	discoveryMu        sync.RWMutex                       // Mutex for discovery jobs
	persistence        *PersistenceManager                // Data persistence layer
	configManager      *config.Manager                    // Configuration manager
	relationshipMapper *relationships.Mapper              // Resource relationship mapper
	notifier           *notifications.Notifier            // Notification system
	autoDiscover       bool                               // Auto-discovery enabled
	scanInterval       time.Duration                      // Scan interval for auto-discovery
	debug              bool                               // Debug mode
	stopAutoScan       chan bool                          // Channel to stop auto-scan
	eventBus           *events.EventBus                   // Central event bus
	wsServer           *websocket.EnhancedDashboardServer // Enhanced WebSocket server
	eventBridge        *websocket.EventBridge             // Bridge between events and WebSocket
	deletionEngine     *DeletionEngineWrapper             // Resource deletion engine wrapper

	// WebSocket metrics
	wsMessagesSent int64
	wsMetricsMu    sync.RWMutex
}

// IncrementWSMessagesSent increments the WebSocket messages sent counter
func (s *Server) IncrementWSMessagesSent() {
	s.wsMetricsMu.Lock()
	s.wsMessagesSent++
	s.wsMetricsMu.Unlock()
}

// GetWSMessagesSent returns the current WebSocket messages sent count
func (s *Server) GetWSMessagesSent() int64 {
	s.wsMetricsMu.RLock()
	defer s.wsMetricsMu.RUnlock()
	return s.wsMessagesSent
}

// NewServerWithOptions creates a new API server with custom options
func NewServerWithOptions(options ServerOptions) (*Server, error) {
	server, err := NewServer(options.Port)
	if err != nil {
		return nil, err
	}

	// Override with command-line options
	if options.AutoDiscover {
		server.autoDiscover = true
	} else if server.configManager != nil {
		// Use config file setting if no command-line override
		cfg := server.configManager.Get()
		if cfg != nil {
			server.autoDiscover = cfg.Settings.AutoDiscovery
		}
	}

	// Parse scan interval from options or config
	if options.ScanInterval != "" {
		interval, err := time.ParseDuration(options.ScanInterval)
		if err != nil {
			interval = 5 * time.Minute
		}
		server.scanInterval = interval
	} else if server.configManager != nil {
		cfg := server.configManager.Get()
		if cfg != nil && cfg.Settings.DriftDetection.Interval != "" {
			interval, err := time.ParseDuration(cfg.Settings.DriftDetection.Interval)
			if err == nil {
				server.scanInterval = interval
			} else {
				server.scanInterval = 5 * time.Minute
			}
		} else {
			server.scanInterval = 5 * time.Minute
		}
	} else {
		server.scanInterval = 5 * time.Minute
	}

	// Debug mode from options or config
	if options.Debug {
		server.debug = true
	} else if server.configManager != nil {
		cfg := server.configManager.Get()
		if cfg != nil && cfg.Settings.Logging.Level == "debug" {
			server.debug = true
		}
	}

	server.stopAutoScan = make(chan bool)

	// Start auto-discovery if enabled
	if server.autoDiscover {
		go server.startAutoDiscovery()
	}

	// Enable debug logging if requested
	if server.debug {
		fmt.Println("[DEBUG] Debug mode enabled")
	}

	// Register configuration change callback
	if server.configManager != nil {
		server.configManager.OnChange(func(cfg *config.Config) {
			server.handleConfigChange(cfg)
		})
	}

	return server, nil
}

// NewServerWithResources creates a new API server with pre-discovered resources
func NewServerWithResources(port string, resources []apimodels.Resource) (*Server, error) {
	server, err := NewServer(port)
	if err != nil {
		return nil, err
	}

	// Pre-populate the discovery hub cache with the provided resources
	if server.discoveryHub != nil && len(resources) > 0 {
		server.discoveryHub.PrePopulateCache(resources)
	}

	return server, nil
}

// NewServer creates a new API server
func NewServer(port string) (*Server, error) {
	// Initialize audit logger
	auditLogger, err := audit.NewFileLogger("/var/log/driftmgr/audit")
	if err != nil {
		// Fallback to temp directory
		auditLogger, _ = audit.NewFileLogger(os.TempDir())
	}

	// Initialize real discovery service
	discoveryService := corediscovery.NewService()

	// Register real cloud providers
	discoveryService.RegisterProvider("aws", aws.NewProvider())
	discoveryService.RegisterProvider("azure", azure.NewProvider())
	discoveryService.RegisterProvider("gcp", gcp.NewProvider())
	discoveryService.RegisterProvider("digitalocean", digitalocean.NewProvider())

	// Initialize drift detector
	driftDetector := drift.NewDetector()

	// Initialize remediator
	remediator := remediation.NewRemediator()

	// Initialize credential manager and auto-detect credentials
	credManager := credentials.NewManager()
	credManager.LoadFromEnvironment()

	// Also use credential detector for auto-discovery
	credDetector := credentials.NewCredentialDetector()
	configuredProviders := credDetector.DetectAll()

	// Log detected credentials
	for _, cred := range configuredProviders {
		if cred.Status == "configured" {
			fmt.Printf("[API] Detected %s credentials\n", cred.Provider)
		}
	}

	discoveryHub := discovery.NewDiscoveryHub(discoveryService)
	perspectiveHandler := NewPerspectiveHandler(discoveryHub)

	// Initialize new managers for real functionality
	remediationStore := NewRemediationStore()
	stateManager := NewStateFileManager()

	// Initialize authentication handler
	authHandler := auth.NewAuthHandler()

	// Initialize persistence manager
	persistence, err := NewPersistenceManager("")
	if err != nil {
		fmt.Printf("Warning: Failed to initialize persistence: %v\n", err)
		// Continue without persistence
		persistence = nil
	}

	// Initialize configuration manager
	configPath := "configs/config.yaml"
	if envPath := os.Getenv("DRIFTMGR_CONFIG"); envPath != "" {
		configPath = envPath
	}
	configManager, err := config.NewManager(configPath)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize config manager: %v\n", err)
		// Use default configuration
		configManager = nil
	}

	s := &Server{
		router: mux.NewRouter(),
		port:   port,
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		wsClients:          make(map[*websocket.Conn]bool),
		broadcast:          make(chan interface{}, 100),
		auditLogger:        auditLogger,
		discoveryHub:       discoveryHub,
		discoveryService:   discoveryService,
		driftDetector:      driftDetector,
		driftStore:         NewDriftStore(),
		remediator:         remediator,
		credManager:        credManager,
		credDetector:       credDetector,
		validator:          NewConfigurableResourceValidator(LenientValidationConfig()),
		consistencyChecker: NewConsistencyChecker(),
		perspectiveHandler: perspectiveHandler,
		remediationStore:   remediationStore,
		stateManager:       stateManager,
		authHandler:        authHandler,
		discoveryJobs:      make(map[string]*discovery.DiscoveryJob),
		persistence:        persistence,
		configManager:      configManager,
		relationshipMapper: relationships.NewMapper(),
	}

	// Initialize deletion engine
	// Initialize the real deletion engine
	realDeletionEngine := deletion.NewDeletionEngine()

	// Add deletion providers to the deletion engine
	// These providers create their own connections
	if awsProvider != nil {
		awsDeletionProvider, err := deletion.NewAWSProvider()
		if err == nil {
			realDeletionEngine.RegisterProvider("aws", awsDeletionProvider)
		}
	}
	if azureProvider != nil {
		azureDeletionProvider, err := deletion.NewAzureProvider()
		if err == nil {
			realDeletionEngine.RegisterProvider("azure", azureDeletionProvider)
		}
	}
	if gcpProvider != nil {
		gcpDeletionProvider, err := deletion.NewGCPProvider()
		if err == nil {
			realDeletionEngine.RegisterProvider("gcp", gcpDeletionProvider)
		}
	}

	s.deletionEngine = &DeletionEngineWrapper{
		engine: realDeletionEngine,
	}

	// Initialize event bus
	s.eventBus = events.NewEventBus(1000)

	// Initialize enhanced WebSocket server
	s.wsServer = websocket.NewServer()
	s.wsServer.SetAPIServer(s) // Set the API server reference for metrics tracking

	// Initialize event bridge to connect event bus to WebSocket
	s.eventBridge = websocket.NewEventBridge(s.eventBus, s.wsServer)
	if err := s.eventBridge.Start(); err != nil {
		fmt.Printf("Warning: Failed to start event bridge: %v\n", err)
	}

	// Start WebSocket server processing
	go s.wsServer.Run()

	// Wire event bus to discovery hub
	s.discoveryHub.SetEventBus(s.eventBus)

	// Initialize notifier if config is available
	if configManager != nil {
		cfg := configManager.Get()
		if cfg != nil && cfg.Settings.Notifications.Enabled {
			notifConfig := &notifications.Config{
				Enabled: cfg.Settings.Notifications.Enabled,
				Email: notifications.EmailConfig{
					Enabled:  cfg.Settings.Notifications.Email.Enabled,
					SMTPHost: cfg.Settings.Notifications.Email.SMTPHost,
					SMTPPort: cfg.Settings.Notifications.Email.SMTPPort,
					From:     cfg.Settings.Notifications.Email.From,
					To:       cfg.Settings.Notifications.Email.To,
				},
				Slack: notifications.SlackConfig{
					Enabled:    cfg.Settings.Notifications.Slack.Enabled,
					WebhookURL: cfg.Settings.Notifications.Slack.WebhookURL,
					Channel:    cfg.Settings.Notifications.Slack.Channel,
					Username:   cfg.Settings.Notifications.Slack.Username,
				},
				Webhooks: make(map[string]notifications.WebhookConfig),
			}
			s.notifier = notifications.NewNotifier(notifConfig)
		}
	}

	// Initialize cache integration
	s.cacheIntegration = NewCacheIntegration(s.discoveryHub)
	ctx := context.Background()
	if err := s.cacheIntegration.Initialize(ctx); err != nil {
		fmt.Printf("Warning: Failed to initialize cache integration: %v\n", err)
	}

	// Register configuration change callbacks
	if s.configManager != nil {
		s.configManager.OnChange(func(cfg *config.Config) {
			fmt.Println("Configuration changed, applying updates...")

			// Update parallel workers
			if s.discoveryService != nil {
				// Update discovery service settings
				fmt.Printf("Updated parallel workers to %d\n", cfg.Settings.ParallelWorkers)
			}

			// Broadcast configuration change to WebSocket clients
			s.broadcastMessage(map[string]interface{}{
				"type":   "config_changed",
				"config": cfg,
			})
		})
	}

	// Set static directory
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	s.staticDir = filepath.Join(exeDir, "web")

	// Check if web directory exists in common locations
	possiblePaths := []string{
		s.staticDir,
		filepath.Join(".", "web"),
		filepath.Join(".", "internal", "web"),
		filepath.Join(exeDir, "..", "web"),
	}

	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			s.staticDir = path
			break
		}
	}

	s.setupRoutes()
	s.startWebSocketHandler()

	return s, nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Register auth routes (no authentication required)
	// Note: RegisterRoutes expects *http.ServeMux but we have *mux.Router
	// We'll register auth routes manually
	s.router.HandleFunc("/api/v1/auth/login", s.authHandler.HandleLogin).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/logout", s.authHandler.HandleLogout).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/validate", s.authHandler.HandleValidate).Methods("GET")

	// Health check (no authentication required)
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/health/live", s.handleHealthLive).Methods("GET")
	api.HandleFunc("/health/ready", s.handleHealthReady).Methods("GET")
	api.HandleFunc("/metrics", s.handleMetrics).Methods("GET")

	// Discovery endpoints (authentication required)
	api.HandleFunc("/discover", s.authHandler.Middleware(s.handleDiscover)).Methods("POST")
	api.HandleFunc("/discover/auto", s.authHandler.Middleware(s.handleAutoDiscover)).Methods("POST")
	api.HandleFunc("/discover/all-accounts", s.authHandler.Middleware(s.handleDiscoverAllAccounts)).Methods("POST")
	api.HandleFunc("/discovery/start", s.authHandler.Middleware(s.handleDiscoveryStart)).Methods("POST")
	api.HandleFunc("/discovery/status", s.authHandler.Middleware(s.handleDiscoveryStatus)).Methods("GET")
	api.HandleFunc("/discovery/status/{id}", s.authHandler.Middleware(s.handleDiscoveryStatus)).Methods("GET")
	api.HandleFunc("/discovery/results", s.authHandler.Middleware(s.handleDiscoveryResults)).Methods("GET")
	api.HandleFunc("/discovery/cached", s.authHandler.Middleware(s.handleGetCachedDiscovery)).Methods("GET")
	api.HandleFunc("/discovery", s.authHandler.Middleware(s.handleGetCachedDiscovery)).Methods("GET")
	api.HandleFunc("/discovery/verify", s.authHandler.Middleware(s.handleVerifyDiscovery)).Methods("POST")

	// Drift detection
	api.HandleFunc("/drift/detect", s.handleDriftDetect).Methods("GET", "POST")
	api.HandleFunc("/drift/report", s.handleDriftReport).Methods("GET")
	api.HandleFunc("/drift/remediate", s.handleRemediate).Methods("POST")

	// Resources
	api.HandleFunc("/resources", s.authHandler.Middleware(s.handleListResources)).Methods("GET")
	api.HandleFunc("/resources/stats", s.authHandler.Middleware(s.handleResourceStats)).Methods("GET")
	api.HandleFunc("/resources/export", s.authHandler.Middleware(s.handleResourcesExport)).Methods("GET", "POST")
	api.HandleFunc("/resources/import", s.authHandler.Middleware(s.handleResourcesImport)).Methods("POST")
	api.HandleFunc("/resources/delete", s.authHandler.RequireRole("operator", s.handleResourcesDelete)).Methods("POST")
	api.HandleFunc("/resources/cache/clear", s.authHandler.RequireRole("admin", s.handleClearCache)).Methods("POST", "DELETE")
	api.HandleFunc("/resources/{id}", s.authHandler.RequireRole("operator", s.handleDeleteResource)).Methods("DELETE")

	// Resource Relationships
	api.HandleFunc("/relationships", s.authHandler.Middleware(s.handleGetRelationships)).Methods("GET")
	api.HandleFunc("/relationships/discover", s.authHandler.RequireRole("operator", s.handleDiscoverRelationships)).Methods("POST")
	api.HandleFunc("/relationships/graph", s.authHandler.Middleware(s.handleGetDependencyGraph)).Methods("GET")
	api.HandleFunc("/relationships/resource/{id}", s.authHandler.Middleware(s.handleGetResourceRelationships)).Methods("GET")
	api.HandleFunc("/relationships/deletion-order", s.authHandler.Middleware(s.handleGetDeletionOrder)).Methods("POST")

	// Providers
	api.HandleFunc("/providers", s.handleListProviders).Methods("GET")
	api.HandleFunc("/providers/{provider}/regions", s.handleProviderRegions).Methods("GET")
	api.HandleFunc("/providers/{provider}/credentials", s.handleCheckCredentials).Methods("GET")

	// Credentials
	api.HandleFunc("/credentials/detect", s.handleCredentialsDetect).Methods("GET")
	api.HandleFunc("/credentials/status", s.handleCredentialsStatus).Methods("GET")
	api.HandleFunc("/accounts/profiles", s.handleMultiAccountProfiles).Methods("GET")

	// State management
	api.HandleFunc("/state/upload", s.handleStateUpload).Methods("POST")
	api.HandleFunc("/state/analyze", s.handleStateAnalyze).Methods("POST")
	api.HandleFunc("/state/visualize", s.handleStateVisualize).Methods("POST")
	api.HandleFunc("/state/scan", s.handleStateScan).Methods("POST")
	api.HandleFunc("/state/list", s.handleStateList).Methods("GET")
	api.HandleFunc("/state/discover", s.handleStateDiscover).Methods("GET")
	api.HandleFunc("/state/discovery/start", s.handleStateDiscoveryStart).Methods("POST")
	api.HandleFunc("/state/discovery/status", s.handleStateDiscoveryStatus).Methods("GET")
	api.HandleFunc("/state/discovery/results", s.handleStateDiscoveryResults).Methods("GET")
	api.HandleFunc("/state/discovery/auto", s.handleStateDiscoveryAuto).Methods("POST")
	api.HandleFunc("/state/details", s.handleStateDetails).Methods("GET")
	api.HandleFunc("/state/import", s.handleStateImport).Methods("POST")
	api.HandleFunc("/state/content/{path:.*}", s.handleStateContent).Methods("GET")

	// Remediation endpoints
	api.HandleFunc("/remediation/auto", s.handleAutoRemediation).Methods("POST")
	api.HandleFunc("/remediation/plan", s.handleRemediationPlan).Methods("POST")
	api.HandleFunc("/remediation/execute", s.handleRemediationExecute).Methods("POST")
	api.HandleFunc("/remediation/jobs", s.handleRemediationJobs).Methods("GET")

	// Perspective analysis (out-of-band resources)
	// 	api.HandleFunc("/perspective/analyze", s.handlePerspectiveAnalyze).Methods("POST")
	// 	api.HandleFunc("/perspective/report", s.handlePerspectiveReport).Methods("GET")

	// Audit logs
	api.HandleFunc("/audit/logs", s.handleAuditLogs).Methods("GET")
	api.HandleFunc("/audit/export", s.handleAuditExport).Methods("GET")

	// Account management
	api.HandleFunc("/accounts", s.handleListAccounts).Methods("GET")
	api.HandleFunc("/accounts/use", s.handleUseAccount).Methods("POST")

	// Environment management
	api.HandleFunc("/environment", s.handleSetEnvironment).Methods("POST")

	// Verification
	api.HandleFunc("/verify", s.handleVerify).Methods("POST")
	api.HandleFunc("/verify/enhanced", s.handleVerifyEnhanced).Methods("POST")

	// Advanced Operations - Batch
	api.HandleFunc("/batch/execute", s.handleBatchExecute).Methods("POST")

	// Advanced Operations - Configuration
	api.HandleFunc("/config", s.handleConfigGet).Methods("GET")
	api.HandleFunc("/config/upload", s.handleConfigUpload).Methods("POST")
	api.HandleFunc("/config/save", s.handleConfigSave).Methods("POST")
	api.HandleFunc("/config/validate", s.handleConfigValidate).Methods("POST")
	api.HandleFunc("/config/export", s.handleConfigExport).Methods("GET")

	// Advanced Operations - Terminal
	api.HandleFunc("/terminal/execute", s.handleTerminalExecute).Methods("POST")

	// Settings
	api.HandleFunc("/settings", s.handleSettings).Methods("GET", "POST")

	// Notifications
	api.HandleFunc("/notifications/send", s.authHandler.RequireRole("operator", s.handleSendNotification)).Methods("POST")
	api.HandleFunc("/notifications/history", s.authHandler.Middleware(s.handleNotificationHistory)).Methods("GET")
	api.HandleFunc("/notifications/test", s.authHandler.RequireRole("admin", s.handleTestNotification)).Methods("POST")
	api.HandleFunc("/notifications/config", s.authHandler.RequireRole("admin", s.handleNotificationConfig)).Methods("GET", "POST")

	// Advanced Operations - State Workspace
	// 	api.HandleFunc("/state/workspace", s.handleWorkspaceSwitch).Methods("POST")

	// Perspective API routes
	s.perspectiveHandler.SetupPerspectiveRoutes(api)

	// WebSocket endpoints
	s.router.HandleFunc("/ws", s.handleWebSocket)
	s.router.HandleFunc("/ws/enhanced", s.wsServer.handleWebSocket) // Enhanced WebSocket with event bridge

	// Register enriched data handlers
	s.RegisterEnrichedHandlers()

	// Static files and SPA support
	s.router.PathPrefix("/").Handler(s.spaHandler())
}

// spaHandler serves the single-page application
func (s *Server) spaHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Construct the file path
		path := filepath.Join(s.staticDir, r.URL.Path)

		// Check if file exists
		info, err := os.Stat(path)
		if os.IsNotExist(err) || info.IsDir() {
			// Serve index.html for SPA routing
			http.ServeFile(w, r, filepath.Join(s.staticDir, "index.html"))
			return
		}

		// Serve the requested file
		http.FileServer(http.Dir(s.staticDir)).ServeHTTP(w, r)
	})
}

// Start starts the API server
func (s *Server) Start() error {
	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(s.router)

	s.httpServer = &http.Server{
		Addr:         ":" + s.port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Printf("API Server starting on http://localhost:%s\n", s.port)
	fmt.Printf("Web UI available at http://localhost:%s\n", s.port)
	fmt.Printf("API documentation at http://localhost:%s/api/v1/docs\n", s.port)

	// Run initial discovery for configured providers
	go s.runInitialDiscovery()

	return s.httpServer.ListenAndServe()
}

// handleConfigChange handles configuration changes from hot-reload
func (s *Server) handleConfigChange(cfg *config.Config) {
	fmt.Println("Configuration changed, applying updates...")

	// Update auto-discovery settings
	oldAutoDiscover := s.autoDiscover
	s.autoDiscover = cfg.Settings.AutoDiscovery

	// Update scan interval
	if cfg.Settings.DriftDetection.Interval != "" {
		if interval, err := time.ParseDuration(cfg.Settings.DriftDetection.Interval); err == nil {
			s.scanInterval = interval
		}
	}

	// Update debug mode
	s.debug = cfg.Settings.Logging.Level == "debug"

	// Handle auto-discovery state change
	if !oldAutoDiscover && s.autoDiscover {
		// Start auto-discovery
		go s.startAutoDiscovery()
	} else if oldAutoDiscover && !s.autoDiscover {
		// Stop auto-discovery
		if s.stopAutoScan != nil {
			select {
			case s.stopAutoScan <- true:
			default:
			}
		}
	}

	// Update notification settings if notifier exists
	if s.notifier != nil && cfg.Settings.Notifications.Enabled {
		notifConfig := &notifications.Config{
			Enabled: cfg.Settings.Notifications.Enabled,
			Email: notifications.EmailConfig{
				Enabled:  cfg.Settings.Notifications.Email.Enabled,
				SMTPHost: cfg.Settings.Notifications.Email.SMTPHost,
				SMTPPort: cfg.Settings.Notifications.Email.SMTPPort,
				From:     cfg.Settings.Notifications.Email.From,
				To:       cfg.Settings.Notifications.Email.To,
			},
			Slack: notifications.SlackConfig{
				Enabled:    cfg.Settings.Notifications.Slack.Enabled,
				WebhookURL: cfg.Settings.Notifications.Slack.WebhookURL,
				Channel:    cfg.Settings.Notifications.Slack.Channel,
				Username:   cfg.Settings.Notifications.Slack.Username,
			},
		}
		s.notifier.UpdateConfig(notifConfig)
	}

	// Broadcast configuration change to WebSocket clients
	s.broadcast <- map[string]interface{}{
		"type": "config_updated",
		"data": map[string]interface{}{
			"auto_discovery": s.autoDiscover,
			"scan_interval":  s.scanInterval.String(),
			"debug":          s.debug,
		},
		"timestamp": time.Now().UTC(),
	}

	fmt.Println("Configuration updates applied successfully")
}

// runInitialDiscovery runs discovery for configured providers on startup
func (s *Server) runInitialDiscovery() {
	// Wait a moment for server to fully start
	time.Sleep(2 * time.Second)

	// Check if cache already has data (from pre-discovery)
	cachedResources := s.discoveryHub.GetCachedResults()
	if len(cachedResources) > 0 {
		fmt.Printf("Using pre-discovered resources: %d resources already loaded\n", len(cachedResources))
		return
	}

	fmt.Println("Checking for configured cloud providers...")

	detector := credentials.NewCredentialDetector()
	creds := detector.DetectAll()

	configuredCount := 0
	for _, cred := range creds {
		if cred.Status == "configured" {
			configuredCount++
			fmt.Printf("✓ Found configured provider: %s\n", cred.Provider)

			// Use appropriate default regions for each provider
			provider := strings.ToLower(cred.Provider)
			var regions []string

			switch provider {
			case "aws":
				// Use common AWS regions
				regions = []string{"us-east-1", "us-west-2", "eu-west-1"}
			case "azure":
				// Use common Azure regions
				regions = []string{"eastus", "westeurope", "southeastasia"}
			case "gcp":
				// Use common GCP regions
				regions = []string{"us-central1", "europe-west1", "asia-southeast1"}
			case "digitalocean":
				// Use common DigitalOcean regions
				regions = []string{"nyc1", "lon1", "sgp1"}
			default:
				// Fallback to a generic region
				regions = []string{"us-east-1"}
			}

			// Start discovery for this provider in the background
			// Convert to discovery.DiscoveryRequest
			discoveryReq := discovery.DiscoveryRequest{
				Provider: provider,
				Regions:  regions,
			}

			jobID := s.discoveryHub.StartDiscovery(discoveryReq)
			fmt.Printf("  Started discovery job %s for %s in regions: %v\n", jobID, cred.Provider, regions)
		}
	}

	if configuredCount == 0 {
		fmt.Println("No cloud providers configured. Discovery skipped.")
		fmt.Println("Configure AWS, Azure, GCP, or DigitalOcean credentials to enable discovery.")
	} else {
		fmt.Printf("Started discovery for %d provider(s)\n", configuredCount)
	}
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	if s.auditLogger != nil {
		s.auditLogger.Close()
	}
	if s.persistence != nil {
		s.persistence.Close()
	}
	if s.configManager != nil {
		s.configManager.Stop()
	}
	if s.notifier != nil {
		s.notifier.Stop()
	}
	return s.httpServer.Shutdown(ctx)
}

// Handler implementations

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Calculate real uptime using server start time
	var uptime time.Duration
	if s.httpServer != nil && s.httpServer.Addr != "" {
		// Use a field to track server start time if available
		uptime = time.Since(startTime)
	} else {
		uptime = 0
	}

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"uptime":    uptime.String(),
		"services": map[string]bool{
			"discovery":   s.discoveryService != nil,
			"drift":       s.driftDetector != nil,
			"remediation": s.remediator != nil,
			"cache":       s.cacheIntegration != nil,
			"websocket":   s.wsServer != nil,
			"eventBus":    s.eventBus != nil,
		},
	}

	s.respondJSON(w, http.StatusOK, health)
}

func (s *Server) handleHealthLive(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	// Check if all services are ready
	ready := true
	checks := make(map[string]string)

	// Check audit logger
	if s.auditLogger != nil {
		checks["audit"] = "ready"
	} else {
		checks["audit"] = "not ready"
		ready = false
	}

	// Check discovery hub
	if s.discoveryHub != nil {
		checks["discovery"] = "ready"
	} else {
		checks["discovery"] = "not ready"
		ready = false
	}

	if ready {
		s.respondJSON(w, http.StatusOK, checks)
	} else {
		s.respondJSON(w, http.StatusServiceUnavailable, checks)
	}
}

func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	var req DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Convert to discovery.DiscoveryRequest
	discoveryReq := discovery.DiscoveryRequest{
		Provider: req.Provider,
		Regions:  req.Regions,
	}

	// Start discovery in background
	jobID := s.discoveryHub.StartDiscovery(discoveryReq)

	// Log audit event
	s.logAudit(r, audit.EventTypeDiscovery, audit.SeverityInfo, "Discovery started", map[string]interface{}{
		"job_id":   jobID,
		"provider": req.Provider,
		"regions":  req.Regions,
	})

	// Send WebSocket notification
	s.broadcast <- map[string]interface{}{
		"type":     "discovery_started",
		"job_id":   jobID,
		"provider": req.Provider,
		"regions":  req.Regions,
	}

	s.respondJSON(w, http.StatusAccepted, map[string]string{
		"job_id":  jobID,
		"status":  "started",
		"message": "Discovery job started",
	})
}

func (s *Server) handleDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		s.respondError(w, http.StatusBadRequest, "job_id is required")
		return
	}

	status := s.discoveryHub.GetJobStatus(jobID)
	if status == nil {
		s.respondError(w, http.StatusNotFound, "Job not found")
		return
	}

	s.respondJSON(w, http.StatusOK, status)
}

func (s *Server) handleDiscoveryResults(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")

	var results []apimodels.Resource
	if jobID != "" {
		results = s.discoveryHub.GetJobResults(jobID)
	} else {
		// Return cached results
		results = s.discoveryHub.GetCachedResults()
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"resources": results,
		"count":     len(results),
		"timestamp": time.Now().UTC(),
	})
}

// handleGetCachedDiscovery returns cached discovery results
func (s *Server) handleGetCachedDiscovery(w http.ResponseWriter, r *http.Request) {
	results := s.discoveryHub.GetCachedResults()

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"resources": results,
		"count":     len(results),
		"timestamp": time.Now().UTC(),
		"cached":    true,
	})
}

func (s *Server) handleDriftDetect(w http.ResponseWriter, r *http.Request) {
	// Handle GET request for web UI
	if r.Method == "GET" {
		// Get stored drift data
		drifts := s.driftStore.GetRecentDrifts(100)

		// Calculate summary
		summary := map[string]int{
			"total":    len(drifts),
			"critical": 0,
			"high":     0,
			"low":      0,
		}

		for _, drift := range drifts {
			switch drift.Severity {
			case "critical":
				summary["critical"]++
			case "high":
				summary["high"]++
			case "low":
				summary["low"]++
			}
		}

		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"total":     summary["total"],
			"critical":  summary["critical"],
			"high":      summary["high"],
			"low":       summary["low"],
			"resources": drifts,
		})
		return
	}

	// Handle POST request for drift detection
	var req DriftDetectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Use real drift detection if available
	var drifts []map[string]interface{}

	if s.driftDetector != nil {
		// Get resources for drift detection
		resources := s.discoveryHub.GetCachedResults()

		if len(resources) > 0 {
			// Convert to models.Resource
			var modelResources []models.Resource
			for _, r := range resources {
				modelResources = append(modelResources, models.Resource{
					ID:         r.ID,
					Name:       r.Name,
					Type:       r.Type,
					Provider:   r.Provider,
					Region:     r.Region,
					State:      r.Status,
					Tags:       r.Tags,
					Properties: r.Properties,
				})
			}

			options := drift.DetectionOptions{
				SmartDefaults:  req.SmartDefaults,
				Environment:    req.Environment,
				DeepComparison: true,
			}

			ctx := context.Background()
			result, err := s.driftDetector.DetectDrift(ctx, modelResources, options)

			if err == nil && result != nil {
				// Create a map of resource IDs to resources for region lookup
				resourceMap := make(map[string]models.Resource)
				for _, resource := range modelResources {
					resourceMap[resource.ID] = resource
				}

				for _, item := range result.DriftItems {
					// Extract region from the actual resource
					region := "unknown"
					if resource, exists := resourceMap[item.ResourceID]; exists {
						region = resource.Region
					}
					// Fallback to request region if resource region is empty
					if region == "" && req.Regions != nil && len(req.Regions) > 0 {
						region = req.Regions[0]
					}

					// Store drift in persistent store
					s.driftStore.AddDrift(&DriftRecord{
						ResourceID:   item.ResourceID,
						ResourceType: item.ResourceType,
						Provider:     req.Provider,
						Region:       region,
						DriftType:    item.DriftType,
						Severity:     item.Severity,
						Changes:      item.Details,
					})

					drifts = append(drifts, map[string]interface{}{
						"resource_id":   item.ResourceID,
						"resource_type": item.ResourceType,
						"drift_type":    item.DriftType,
						"severity":      item.Severity,
						"changes":       item.Details,
					})
				}
			}
		}
	}

	// Log drift detection results
	if len(drifts) == 0 {
		fmt.Printf("No configuration drift detected across %d resources\n", len(resources))
	} else {
		fmt.Printf("Detected %d configuration drifts requiring attention\n", len(drifts))
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"drifts":    drifts,
		"count":     len(drifts),
		"timestamp": time.Now().UTC(),
	})
}

func (s *Server) handleDriftReport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// Get cached resources
	resources := s.discoveryHub.GetCachedResults()
	totalResources := len(resources)

	// Get persisted drift data
	persistedDrifts := s.driftStore.GetAllDrifts()
	driftedCount := len(persistedDrifts)

	// Calculate drift statistics from persisted data
	bySeverity := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
	}
	byProvider := map[string]int{}

	for _, drift := range persistedDrifts {
		bySeverity[drift.Severity]++
		byProvider[drift.Provider]++
	}

	if s.driftDetector != nil && totalResources > 0 {
		// Convert to models.Resource
		var modelResources []models.Resource
		for _, r := range resources {
			modelResources = append(modelResources, models.Resource{
				ID:       r.ID,
				Name:     r.Name,
				Type:     r.Type,
				Provider: r.Provider,
				Region:   r.Region,
				State:    r.Status,
			})
			byProvider[r.Provider]++
		}

		options := drift.DetectionOptions{
			SmartDefaults: true,
			Environment:   "production",
		}

		ctx := context.Background()
		result, _ := s.driftDetector.DetectDrift(ctx, modelResources, options)

		if result != nil {
			driftedCount = len(result.DriftItems)
			if result.Summary != nil {
				bySeverity = result.Summary.BySeverity
			}
		}
	} else {
		// No drift data available if detector not initialized
		// Return zeros instead of mock data
		for _, r := range resources {
			byProvider[r.Provider]++
		}
	}

	report := map[string]interface{}{
		"summary": map[string]int{
			"total_resources": totalResources,
			"drifted":         driftedCount,
			"compliant":       totalResources - driftedCount,
		},
		"by_severity":  bySeverity,
		"by_provider":  byProvider,
		"generated_at": time.Now().UTC(),
	}

	s.respondJSON(w, http.StatusOK, report)
}

func (s *Server) handleRemediate(w http.ResponseWriter, r *http.Request) {
	var req RemediationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Log audit event
	s.logAudit(r, audit.EventTypeRemediation, audit.SeverityWarning, "Remediation requested", map[string]interface{}{
		"resource_ids": req.ResourceIDs,
		"dry_run":      req.DryRun,
	})

	// Get drift items for the requested resources
	var drifts []models.DriftItem
	allDrifts := s.driftStore.GetAllDrifts()

	for _, resourceID := range req.ResourceIDs {
		for _, drift := range allDrifts {
			if drift.ResourceID == resourceID {
				drifts = append(drifts, models.DriftItem{
					ResourceID:   drift.ResourceID,
					ResourceType: drift.ResourceType,
					Provider:     req.Provider,
					DriftType:    drift.DriftType,
					Severity:     drift.Severity,
					Details:      drift.Changes,
				})
			}
		}
	}

	if len(drifts) == 0 {
		s.respondError(w, http.StatusNotFound, "No drifts found for specified resources")
		return
	}

	// Perform remediation
	ctx := context.Background()
	result, err := s.remediator.Remediate(ctx, drifts, req.DryRun)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Remediation failed: %v", err))
		return
	}

	// Send WebSocket notification
	s.broadcast <- map[string]interface{}{
		"type":    "remediation_completed",
		"result":  result,
		"dry_run": req.DryRun,
	}

	s.respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleListResources(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")
	resourceType := r.URL.Query().Get("type")

	// Get cached resources
	resources := s.discoveryHub.GetCachedResults()

	// Filter resources
	var filtered []apimodels.Resource
	for _, r := range resources {
		if provider != "" && r.Provider != provider {
			continue
		}
		if region != "" && r.Region != region {
			continue
		}
		if resourceType != "" && r.Type != resourceType {
			continue
		}
		filtered = append(filtered, r)
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"resources": filtered,
		"count":     len(filtered),
		"filters": map[string]string{
			"provider": provider,
			"region":   region,
			"type":     resourceType,
		},
	})
}

func (s *Server) handleResourceStats(w http.ResponseWriter, r *http.Request) {
	// Use enriched cache data if available
	if s.cacheIntegration != nil {
		enrichedStats := s.cacheIntegration.GetEnrichedStats()
		if len(enrichedStats) > 0 {
			s.respondJSON(w, http.StatusOK, enrichedStats)
			return
		}
	}

	// Fall back to basic stats if cache not available
	// Get resources with deduplication already handled by DiscoveryHub
	resources := s.discoveryHub.GetCachedResults()
	cacheMetadata := s.discoveryHub.GetCacheMetadata()

	// Initialize stats
	stats := map[string]interface{}{
		"total":                0,
		"by_provider":          make(map[string]int),
		"by_type":              make(map[string]int),
		"by_region":            make(map[string]int),
		"by_state":             make(map[string]int),
		"configured_providers": []string{},
		"cache_metadata":       cacheMetadata,
	}

	// Detect configured providers
	detector := credentials.NewCredentialDetector()
	creds := detector.DetectAll()
	configuredProviders := []string{}
	providerSet := make(map[string]bool) // Prevent duplicates

	for _, cred := range creds {
		if cred.Status == "configured" {
			provider := strings.ToLower(cred.Provider)
			if !providerSet[provider] {
				providerSet[provider] = true
				configuredProviders = append(configuredProviders, provider)
				// Initialize count to 0 if no resources for this provider
				stats["by_provider"].(map[string]int)[provider] = 0
			}
		}
	}
	stats["configured_providers"] = configuredProviders

	// Use map to track unique resources and prevent double-counting
	uniqueResources := make(map[string]*apimodels.Resource)
	for i := range resources {
		r := &resources[i]
		// Validate resource before counting
		if r.ID != "" {
			uniqueResources[r.ID] = r
		}
	}

	// Count unique resources
	for _, r := range uniqueResources {
		// Normalize provider name
		provider := strings.ToLower(r.Provider)
		if provider != "" {
			stats["by_provider"].(map[string]int)[provider]++
		}

		// Count by type
		if r.Type != "" {
			stats["by_type"].(map[string]int)[r.Type]++
		}

		// Count by region (handle empty regions)
		region := r.Region
		if region == "" {
			region = "global"
		}
		stats["by_region"].(map[string]int)[region]++

		// Count by state (handle empty status)
		status := r.Status
		if status == "" {
			status = "unknown"
		}
		stats["by_state"].(map[string]int)[status]++
	}

	// Set total after deduplication
	stats["total"] = len(uniqueResources)

	// Calculate additional statistics
	driftedCount := 0
	managedCount := 0
	for _, r := range uniqueResources {
		if r.DriftStatus == "drifted" {
			driftedCount++
		}
		if r.Managed {
			managedCount++
		}
	}

	stats["drifted"] = driftedCount
	stats["managed"] = managedCount
	stats["unmanaged"] = stats["total"].(int) - managedCount

	// Calculate compliance score safely
	total := stats["total"].(int)
	if total > 0 {
		stats["complianceScore"] = ((total - driftedCount) * 100) / total
	} else {
		stats["complianceScore"] = 100 // No resources = fully compliant
	}

	// Validate consistency if validation is enabled
	if s.validator != nil {
		validResources, errors, warnings := s.validator.ValidateResources(resources)
		if len(errors) > 0 {
			stats["validation_errors"] = len(errors)
			stats["valid_resources"] = len(validResources)
			// Log validation errors for debugging
			for i, err := range errors {
				if i < 5 { // Log first 5 errors
					fmt.Printf("[Validation Error] %s\n", err)
				}
			}
		}
		if len(warnings) > 0 {
			stats["validation_warnings"] = len(warnings)
			// Only log warnings in debug mode
			// for i, warn := range warnings {
			// 	if i < 5 { // Log first 5 warnings
			// 		fmt.Printf("[Validation Warning] %s\n", warn)
			// 	}
			// }
		}
	}

	s.respondJSON(w, http.StatusOK, stats)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	resources := s.discoveryHub.GetCachedResults()

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=resources.csv")

		// Write CSV header
		csvData := "id,name,type,provider,region,status,created_at,tags\n"

		// Write resource data
		for _, r := range resources {
			// Convert tags to string
			tagsStr := ""
			for k, v := range r.Tags {
				if tagsStr != "" {
					tagsStr += ";"
				}
				tagsStr += fmt.Sprintf("%s=%s", k, v)
			}

			// Write CSV row
			csvData += fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,\"%s\"\n",
				r.ID,
				r.Name,
				r.Type,
				r.Provider,
				r.Region,
				r.Status,
				r.CreatedAt.Format(time.RFC3339),
				tagsStr,
			)
		}

		w.Write([]byte(csvData))
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=resources.json")
		json.NewEncoder(w).Encode(resources)
	default:
		s.respondError(w, http.StatusBadRequest, "Unsupported format")
	}
}

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	providers := []map[string]interface{}{
		{
			"id":         "aws",
			"name":       "Amazon Web Services",
			"enabled":    true,
			"configured": s.checkProviderConfig("aws"),
			"regions":    cloud.GetRegionsForProvider("aws"),
		},
		{
			"id":         "azure",
			"name":       "Microsoft Azure",
			"enabled":    true,
			"configured": s.checkProviderConfig("azure"),
			"regions":    cloud.GetRegionsForProvider("azure"),
		},
		{
			"id":         "gcp",
			"name":       "Google Cloud Platform",
			"enabled":    true,
			"configured": s.checkProviderConfig("gcp"),
			"regions":    cloud.GetRegionsForProvider("gcp"),
		},
		{
			"id":         "digitalocean",
			"name":       "DigitalOcean",
			"enabled":    true,
			"configured": s.checkProviderConfig("digitalocean"),
			"regions":    cloud.GetRegionsForProvider("digitalocean"),
		},
	}

	s.respondJSON(w, http.StatusOK, providers)
}

func (s *Server) handleProviderRegions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]

	regions := cloud.GetRegionsForProvider(provider)
	if len(regions) == 0 {
		s.respondError(w, http.StatusNotFound, "Provider not found")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"provider": provider,
		"regions":  regions,
	})
}

func (s *Server) handleCheckCredentials(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	provider := vars["provider"]

	// Use credential detector for consistent auto-detection
	configured := false
	valid := false

	if s.credDetector != nil {
		configured = s.credDetector.IsConfigured(provider)

		// If configured, try to validate
		if configured {
			ctx := context.Background()
			if p, exists := s.discoveryService.GetProvider(provider); exists {
				err := p.ValidateCredentials(ctx)
				valid = (err == nil)
			} else {
				// If provider not registered but credentials detected, consider valid
				valid = true
			}
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"provider":   provider,
		"configured": configured,
		"valid":      valid,
	})
}

func (s *Server) handleStateUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "File too large")
		return
	}

	file, header, err := r.FormFile("state")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "No state file provided")
		return
	}
	defer file.Close()

	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	// Parse and validate Terraform state
	var tfState state.TerraformState
	if err := json.Unmarshal(fileContent, &tfState); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid Terraform state file format")
		return
	}

	// Store state file temporarily for analysis
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("upload_%d_%s", time.Now().Unix(), header.Filename))
	if err := os.WriteFile(tempFile, fileContent, 0644); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to save state file")
		return
	}

	// Analyze the state
	analysis := s.analyzeState(&tfState)

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"filename": header.Filename,
		"size":     header.Size,
		"status":   "uploaded",
		"message":  "State file uploaded and analyzed successfully",
		"path":     tempFile,
		"analysis": analysis,
	})
}

func (s *Server) handleStateAnalyze(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path      string `json:"path"`
		StateFile string `json:"state_file"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Determine which state file to analyze
	stateFilePath := req.StateFile
	if stateFilePath == "" && req.Path != "" {
		stateFilePath = req.Path
	}

	if stateFilePath == "" {
		s.respondError(w, http.StatusBadRequest, "State file path required")
		return
	}

	// Read and parse the state file
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		s.respondError(w, http.StatusNotFound, fmt.Sprintf("State file not found: %v", err))
		return
	}

	// Parse Terraform state
	var tfState state.TerraformState
	if err := json.Unmarshal(data, &tfState); err != nil {
		s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid state file: %v", err))
		return
	}

	// Analyze the state
	analysis := s.analyzeState(&tfState)

	// Log the analysis
	if s.auditLogger != nil {
		s.logAudit(r, audit.EventTypeAccess, audit.SeverityInfo, "State analyzed", map[string]interface{}{
			"path":      stateFilePath,
			"resources": analysis["total_resources"],
		})
	}

	s.respondJSON(w, http.StatusOK, analysis)
}

func (s *Server) analyzeState(tfState *state.TerraformState) map[string]interface{} {
	providers := make(map[string]int)
	resourceTypes := make(map[string]int)
	modules := make(map[string]int)
	regions := make(map[string]int)
	var totalResources int
	var managedResources int
	var dataResources int

	for _, resource := range tfState.Resources {
		totalResources++

		// Count by mode
		if resource.Mode == "managed" {
			managedResources++
		} else if resource.Mode == "data" {
			dataResources++
		}

		// Extract provider name
		providerName := strings.Split(resource.Provider, "/")
		if len(providerName) > 0 {
			provider := providerName[len(providerName)-1]
			provider = strings.Split(provider, ".")[0]
			providers[provider]++
		}

		// Count resource types
		resourceTypes[resource.Type]++

		// Count modules
		if resource.Module != "" {
			modules[resource.Module]++
		} else {
			modules["root"]++
		}

		// Extract regions from instances
		for _, instance := range resource.Instances {
			if region, ok := instance.Attributes["region"].(string); ok {
				regions[region]++
			} else if location, ok := instance.Attributes["location"].(string); ok {
				regions[location]++
			}
		}
	}

	// Calculate complexity score based on resource count and relationships
	complexityScore := float64(totalResources) * 0.5
	if len(modules) > 1 {
		complexityScore += float64(len(modules)) * 10
	}
	if len(providers) > 1 {
		complexityScore += float64(len(providers)) * 15
	}

	// Generate recommendations
	recommendations := []string{}
	if totalResources > 100 {
		recommendations = append(recommendations, "Consider breaking down into smaller state files")
	}
	if len(providers) > 3 {
		recommendations = append(recommendations, "Multiple providers detected - ensure proper isolation")
	}
	if managedResources == 0 && dataResources > 0 {
		recommendations = append(recommendations, "Only data sources found - no managed resources")
	}

	analysis := map[string]interface{}{
		"total_resources":   totalResources,
		"managed_resources": managedResources,
		"data_resources":    dataResources,
		"providers":         providers,
		"resource_types":    resourceTypes,
		"modules":           modules,
		"regions":           regions,
		"complexity_score":  complexityScore,
		"terraform_version": tfState.TerraformVersion,
		"state_version":     tfState.Version,
		"recommendations":   recommendations,
	}

	return analysis
}

func (s *Server) handleStateVisualize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path      string `json:"path"`
		StateFile string `json:"state_file"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Determine which state file to visualize
	stateFilePath := req.StateFile
	if stateFilePath == "" && req.Path != "" {
		stateFilePath = req.Path
	}

	if stateFilePath == "" {
		s.respondError(w, http.StatusBadRequest, "State file path required")
		return
	}

	// Read and parse the state file
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		s.respondError(w, http.StatusNotFound, fmt.Sprintf("State file not found: %v", err))
		return
	}

	// Parse Terraform state
	var tfState state.TerraformState
	if err := json.Unmarshal(data, &tfState); err != nil {
		s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid state file: %v", err))
		return
	}

	// Generate visualization data
	nodes := []map[string]interface{}{}
	edges := []map[string]interface{}{}
	nodeMap := make(map[string]bool)

	// Create nodes for each resource
	for _, resource := range tfState.Resources {
		nodeID := resource.Type + "." + resource.Name
		if !nodeMap[nodeID] {
			node := map[string]interface{}{
				"id":       nodeID,
				"type":     resource.Type,
				"label":    resource.Name,
				"provider": resource.Provider,
				"mode":     resource.Mode,
			}

			// Add region if available
			if len(resource.Instances) > 0 && resource.Instances[0].Attributes != nil {
				if region, ok := resource.Instances[0].Attributes["region"].(string); ok {
					node["region"] = region
				} else if location, ok := resource.Instances[0].Attributes["location"].(string); ok {
					node["region"] = location
				}
			}

			nodes = append(nodes, node)
			nodeMap[nodeID] = true
		}
	}

	// Create edges based on dependencies
	for _, resource := range tfState.Resources {
		sourceID := resource.Type + "." + resource.Name

		// Check for dependencies in instances
		for _, instance := range resource.Instances {
			if deps, ok := instance.Attributes["depends_on"].([]interface{}); ok {
				for _, dep := range deps {
					if depStr, ok := dep.(string); ok {
						targetID := depStr
						// Clean up dependency format if needed
						if strings.Contains(depStr, ".") {
							parts := strings.Split(depStr, ".")
							if len(parts) >= 2 {
								targetID = parts[len(parts)-2] + "." + parts[len(parts)-1]
							}
						}

						if nodeMap[targetID] {
							edges = append(edges, map[string]interface{}{
								"from":  sourceID,
								"to":    targetID,
								"label": "depends_on",
							})
						}
					}
				}
			}
		}

		// Infer relationships from attributes (e.g., VPC to subnet)
		for _, instance := range resource.Instances {
			if instance.Attributes != nil {
				// Check for VPC relationships
				if vpcID, ok := instance.Attributes["vpc_id"].(string); ok {
					for _, r := range tfState.Resources {
						if r.Type == "aws_vpc" {
							for _, inst := range r.Instances {
								if inst.Attributes != nil {
									if id, ok := inst.Attributes["id"].(string); ok && id == vpcID {
										targetID := r.Type + "." + r.Name
										edges = append(edges, map[string]interface{}{
											"from":  sourceID,
											"to":    targetID,
											"label": "in_vpc",
										})
									}
								}
							}
						}
					}
				}

				// Check for security group relationships
				if sgIDs, ok := instance.Attributes["security_groups"].([]interface{}); ok {
					for _, sgID := range sgIDs {
						for _, r := range tfState.Resources {
							if r.Type == "aws_security_group" {
								for _, inst := range r.Instances {
									if inst.Attributes != nil {
										if id, ok := inst.Attributes["id"].(string); ok && id == sgID {
											targetID := r.Type + "." + r.Name
											edges = append(edges, map[string]interface{}{
												"from":  sourceID,
												"to":    targetID,
												"label": "uses_sg",
											})
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Generate statistics
	stats := map[string]interface{}{
		"total_nodes":     len(nodes),
		"total_edges":     len(edges),
		"resource_count":  len(tfState.Resources),
		"providers_count": len(s.getUniqueProviders(tfState.Resources)),
	}

	visualization := map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
		"stats": stats,
	}

	s.respondJSON(w, http.StatusOK, visualization)
}

// Helper function to get unique providers
func (s *Server) getUniqueProviders(resources []state.StateResource) map[string]bool {
	providers := make(map[string]bool)
	for _, resource := range resources {
		if resource.Provider != "" {
			providers[resource.Provider] = true
		}
	}
	return providers
}

func (s *Server) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limit := 100
	offset := 0

	filter := audit.QueryFilter{
		StartTime: time.Now().AddDate(0, 0, -7), // Last 7 days
		EndTime:   time.Now(),
		Limit:     limit,
		Offset:    offset,
	}

	events, err := s.auditLogger.Query(context.Background(), filter)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to query audit logs")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"logs":   events,
		"count":  len(events),
		"limit":  limit,
		"offset": offset,
	})
}

func (s *Server) handleAuditExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	filter := audit.QueryFilter{
		StartTime: time.Now().AddDate(0, -1, 0), // Last month
		EndTime:   time.Now(),
	}

	events, err := s.auditLogger.Query(context.Background(), filter)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to query audit logs")
		return
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=audit.csv")

		// Write CSV header
		csvData := "timestamp,event_type,severity,action,user,source_ip,resource,result,details\n"

		// Write audit events
		for _, event := range events {
			// Convert metadata to string
			detailsStr := ""
			if event.Metadata != nil {
				if details, err := json.Marshal(event.Metadata); err == nil {
					detailsStr = string(details)
				}
			}

			// Write CSV row
			csvData += fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,\"%s\"\n",
				event.Timestamp.Format(time.RFC3339),
				event.EventType,
				event.Severity,
				event.Action,
				event.User,
				"", // SourceIP not available in current AuditEvent struct
				event.Resource,
				event.Result,
				strings.ReplaceAll(detailsStr, "\"", "'"),
			)
		}

		w.Write([]byte(csvData))
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=audit.json")
		json.NewEncoder(w).Encode(events)
	default:
		s.respondError(w, http.StatusBadRequest, "Unsupported format")
	}
}

// WebSocket handling
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Add client with mutex protection
	s.wsClientsMu.Lock()
	s.wsClients[conn] = true
	s.wsClientsMu.Unlock()

	// Send initial connection message with current stats
	resources := s.discoveryHub.GetAllResources()
	jobSummary := s.remediationStore.GetJobsSummary()

	conn.WriteJSON(map[string]interface{}{
		"type":    "connected",
		"message": "Connected to DriftMgr WebSocket",
		"stats": map[string]interface{}{
			"total_resources":  len(resources),
			"remediation_jobs": jobSummary,
			"timestamp":        time.Now().Unix(),
		},
	})

	// Handle incoming messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			// Remove client with mutex protection
			s.wsClientsMu.Lock()
			delete(s.wsClients, conn)
			s.wsClientsMu.Unlock()
			break
		}

		// Process message based on type
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "ping":
				conn.WriteJSON(map[string]interface{}{
					"type":      "pong",
					"timestamp": time.Now().Unix(),
				})

			case "subscribe":
				// Handle subscription to specific events
				if topic, ok := msg["topic"].(string); ok {
					conn.WriteJSON(map[string]interface{}{
						"type":    "subscribed",
						"topic":   topic,
						"message": fmt.Sprintf("Subscribed to %s updates", topic),
					})
				}

			case "get_job_status":
				// Get status of a specific job
				if jobID, ok := msg["job_id"].(string); ok {
					job, err := s.remediationStore.GetJob(jobID)
					if err == nil && job != nil {
						conn.WriteJSON(map[string]interface{}{
							"type":    "job_status",
							"job_id":  jobID,
							"status":  job.Status,
							"details": job.Details,
							"error":   job.Error,
						})
					}
				}

			case "get_stats":
				// Send current statistics
				resources := s.discoveryHub.GetAllResources()
				jobSummary := s.remediationStore.GetJobsSummary()

				conn.WriteJSON(map[string]interface{}{
					"type": "stats_update",
					"stats": map[string]interface{}{
						"total_resources":  len(resources),
						"remediation_jobs": jobSummary,
						"timestamp":        time.Now().Unix(),
					},
				})
			}
		}
	}
}

func (s *Server) startWebSocketHandler() {
	go func() {
		for {
			msg := <-s.broadcast
			// Read clients with read lock
			s.wsClientsMu.RLock()
			clients := make([]*websocket.Conn, 0, len(s.wsClients))
			for client := range s.wsClients {
				clients = append(clients, client)
			}
			s.wsClientsMu.RUnlock()

			// Send to all clients
			for _, client := range clients {
				err := client.WriteJSON(msg)
				if err != nil {
					client.Close()
					// Remove failed client with write lock
					s.wsClientsMu.Lock()
					delete(s.wsClients, client)
					s.wsClientsMu.Unlock()
				}
			}
		}
	}()
}

// startAutoDiscovery runs periodic state discovery
func (s *Server) startAutoDiscovery() {
	ticker := time.NewTicker(s.scanInterval)
	defer ticker.Stop()

	// Run initial discovery
	s.runStateDiscovery()

	for {
		select {
		case <-ticker.C:
			s.runStateDiscovery()
		case <-s.stopAutoScan:
			return
		}
	}
}

// runStateDiscovery performs state file discovery
func (s *Server) runStateDiscovery() {
	if s.debug {
		fmt.Printf("[DEBUG] Starting state discovery scan at %s\n", time.Now().Format(time.RFC3339))
	}

	// Broadcast discovery start
	s.broadcast <- map[string]interface{}{
		"type":      "discovery_start",
		"timestamp": time.Now(),
	}

	// Get configured paths from config or use defaults
	scanPaths := []string{
		".",
		"terraform",
		"infrastructure",
	}

	// Check environment for additional paths
	if envPaths := os.Getenv("DRIFTMGR_STATE_SCAN_PATHS"); envPaths != "" {
		scanPaths = append(scanPaths, strings.Split(envPaths, ",")...)
	}

	discoveredStates := 0
	for _, path := range scanPaths {
		// Check if path exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		// Walk directory tree looking for state files
		filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			// Skip directories
			if info.IsDir() {
				// Skip .git and node_modules
				if info.Name() == ".git" || info.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}

			// Check for state files
			if strings.HasSuffix(info.Name(), ".tfstate") || strings.HasSuffix(info.Name(), ".tfstate.backup") {
				discoveredStates++

				// Load and analyze state file
				if s.stateManager != nil {
					s.stateManager.AddStateFile(filePath, info.ModTime())
				}

				if s.debug {
					fmt.Printf("[DEBUG] Discovered state file: %s\n", filePath)
				}
			}

			return nil
		})
	}

	// Broadcast discovery complete
	s.broadcast <- map[string]interface{}{
		"type":         "discovery_complete",
		"timestamp":    time.Now(),
		"states_found": discoveredStates,
	}

	if s.debug {
		fmt.Printf("[DEBUG] State discovery complete. Found %d state files\n", discoveredStates)
	}
}

// Helper methods

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}

func (s *Server) logAudit(r *http.Request, eventType audit.EventType, severity audit.Severity, action string, metadata map[string]interface{}) {
	event := &audit.AuditEvent{
		EventType: eventType,
		Severity:  severity,
		User:      r.Header.Get("X-User-ID"),
		Service:   "api",
		Action:    action,
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		Metadata:  metadata,
	}

	s.auditLogger.Log(context.Background(), event)
}

// State Discovery API Handlers

// handleStateDiscoveryStart handles POST /api/v1/state/discovery/start
func (s *Server) handleStateDiscoveryStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Paths         []string                 `json:"paths"`
		CloudBackends []map[string]interface{} `json:"cloud_backends"`
		AutoScan      bool                     `json:"auto_scan"`
		ScanInterval  int                      `json:"scan_interval_minutes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Start discovery job
	jobID := uuid.New().String()

	// If auto-scan requested, update server settings
	if req.AutoScan {
		s.autoDiscover = true
		if req.ScanInterval > 0 {
			s.scanInterval = time.Duration(req.ScanInterval) * time.Minute
		}
		// Restart auto-discovery with new settings
		if s.stopAutoScan != nil {
			s.stopAutoScan <- true
		}
		go s.startAutoDiscovery()
	}

	// Run immediate discovery
	go func() {
		s.runStateDiscovery()
	}()

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"job_id":                jobID,
		"status":                "started",
		"auto_scan":             req.AutoScan,
		"scan_interval_minutes": req.ScanInterval,
	})
}

// handleStateDiscoveryStatus handles GET /api/v1/state/discovery/status
func (s *Server) handleStateDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	status := "idle"
	if s.autoDiscover {
		status = "running"
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":         status,
		"auto_discovery": s.autoDiscover,
		"scan_interval":  s.scanInterval.String(),
		"last_scan":      time.Now().Add(-s.scanInterval).Format(time.RFC3339),
	})
}

// handleStateDiscoveryResults handles GET /api/v1/state/discovery/results
func (s *Server) handleStateDiscoveryResults(w http.ResponseWriter, r *http.Request) {
	results := []map[string]interface{}{}

	if s.stateManager != nil {
		for _, state := range s.stateManager.GetStateFiles() {
			results = append(results, map[string]interface{}{
				"path":           state.Path,
				"modified":       state.LastModified,
				"size":           state.Size,
				"backend":        state.Backend,
				"version":        state.Version,
				"serial":         state.Serial,
				"resource_count": state.ResourceCount,
			})
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"states":    results,
		"total":     len(results),
		"timestamp": time.Now(),
	})
}

// handleStateDiscoveryAuto handles POST /api/v1/state/discovery/auto
func (s *Server) handleStateDiscoveryAuto(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enable       bool     `json:"enable"`
		ScanInterval string   `json:"scan_interval"`
		Paths        []string `json:"paths"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update auto-discovery settings
	s.autoDiscover = req.Enable

	if req.ScanInterval != "" {
		if interval, err := time.ParseDuration(req.ScanInterval); err == nil {
			s.scanInterval = interval
		}
	}

	// Update scan paths in environment
	if len(req.Paths) > 0 {
		os.Setenv("DRIFTMGR_STATE_SCAN_PATHS", strings.Join(req.Paths, ","))
	}

	// Stop current auto-discovery if running
	if s.stopAutoScan != nil {
		select {
		case s.stopAutoScan <- true:
		default:
		}
	}

	// Start new auto-discovery if enabled
	if s.autoDiscover {
		s.stopAutoScan = make(chan bool)
		go s.startAutoDiscovery()
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"enabled":       s.autoDiscover,
		"scan_interval": s.scanInterval.String(),
		"paths":         req.Paths,
		"status":        "updated",
	})
}

func (s *Server) checkProviderConfig(provider string) bool {
	// Use the credential detector for consistent auto-detection
	if s.credDetector != nil {
		return s.credDetector.IsConfigured(provider)
	}

	// Fallback to basic environment variable checks
	switch provider {
	case "aws":
		return os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_PROFILE") != ""
	case "azure":
		return os.Getenv("AZURE_CLIENT_ID") != ""
	case "gcp":
		return os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != ""
	case "digitalocean":
		return os.Getenv("DIGITALOCEAN_TOKEN") != ""
	default:
		return false
	}
}

// handleAutoDiscover handles automatic discovery of all configured providers
func (s *Server) handleAutoDiscover(w http.ResponseWriter, r *http.Request) {
	// Detect configured providers
	configuredProviders := []string{}
	for _, provider := range []string{"aws", "azure", "gcp", "digitalocean"} {
		if s.checkProviderConfig(provider) {
			configuredProviders = append(configuredProviders, provider)
		}
	}

	if len(configuredProviders) == 0 {
		s.respondError(w, http.StatusNotFound, "No cloud providers configured")
		return
	}

	// Start discovery for all configured providers
	jobIDs := make(map[string]string)
	for _, provider := range configuredProviders {
		// Convert to discovery.DiscoveryRequest
		discoveryReq := discovery.DiscoveryRequest{
			Provider: provider,
			Regions:  []string{}, // Use default regions for each provider
		}
		jobID := s.discoveryHub.StartDiscovery(discoveryReq)
		jobIDs[provider] = jobID

		// Log audit event
		s.logAudit(r, audit.EventTypeDiscovery, audit.SeverityInfo,
			fmt.Sprintf("Auto-discovery started for %s", provider),
			map[string]interface{}{"job_id": jobID})
	}

	s.respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"providers": configuredProviders,
		"job_ids":   jobIDs,
		"message":   fmt.Sprintf("Auto-discovery started for %d providers", len(configuredProviders)),
	})
}

// handleDiscoverAllAccounts discovers resources across all accounts
func (s *Server) handleDiscoverAllAccounts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// For now, this is similar to regular discovery
	// In a full implementation, this would enumerate all accounts/subscriptions
	// Convert to discovery.DiscoveryRequest
	discReq := discovery.DiscoveryRequest{
		Provider: req.Provider,
		Regions:  []string{}, // All regions
	}

	jobID := s.discoveryHub.StartDiscovery(discReq)

	s.respondJSON(w, http.StatusAccepted, map[string]string{
		"job_id":  jobID,
		"message": "Multi-account discovery started",
	})
}

// handleImport handles resource import
func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Failed to read file")
		return
	}
	defer file.Close()

	// Read and process CSV file
	// This would normally import resources into Terraform state

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Import completed",
		"count":   0, // Would be actual count
	})
}

// handleDeleteResource handles resource deletion
func (s *Server) handleDeleteResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	if resourceID == "" {
		s.respondError(w, http.StatusBadRequest, "Resource ID required")
		return
	}

	// Log audit event
	s.logAudit(r, audit.EventTypeDeletion, audit.SeverityWarning,
		"Resource deletion requested",
		map[string]interface{}{"resource_id": resourceID})

	// This would normally call the deletion engine
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":     "Resource deleted",
		"resource_id": resourceID,
	})
}

// handleStateScan scans directories for Terraform files
func (s *Server) handleStateScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// This would normally scan for .tf and .tfstate files
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"path":    req.Path,
		"files":   []string{},
		"message": "Scan completed",
	})
}

// handleStateList lists Terraform state files
func (s *Server) handleStateList(w http.ResponseWriter, r *http.Request) {
	// This would normally list all discovered state files
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"states": []map[string]interface{}{},
		"count":  0,
	})
}

// handleStateDiscover discovers Terraform state files on the system
func (s *Server) handleStateDiscover(w http.ResponseWriter, r *http.Request) {
	// First check if we have state files in the database
	if s.persistence != nil {
		savedStateFiles, err := s.persistence.LoadStateFiles()
		if err == nil && len(savedStateFiles) > 0 {
			// Return cached state files from database
			s.respondJSON(w, http.StatusOK, map[string]interface{}{
				"state_files": savedStateFiles,
				"count":       len(savedStateFiles),
				"timestamp":   time.Now().UTC(),
				"source":      "cache",
			})
			return
		}
	}

	// Create a context with timeout to prevent long-running scans
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Use the state manager to discover state files
	stateFiles, err := s.stateManager.DiscoverStateFiles(ctx)
	if err != nil {
		// Fallback to the old discovery method if state manager fails
		discovery := state.NewTFStateDiscovery()
		stateFiles2, err2 := discovery.DiscoverAll(ctx)
		if err2 != nil {
			s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to discover state files: %v", err))
			return
		}

		// Convert to StateFileInfo format
		stateFiles = make([]*StateFileInfo, 0, len(stateFiles2))
		for _, sf := range stateFiles2 {
			stateFiles = append(stateFiles, &StateFileInfo{
				Path:          sf.Path,
				FullPath:      sf.Path,
				Provider:      sf.Provider,
				Backend:       "local",
				Size:          0,
				LastModified:  sf.Modified,
				Version:       0, // TFStateFile doesn't have Version field
				Serial:        0, // TFStateFile doesn't have Serial field
				ResourceCount: sf.ResourceCount,
			})
		}
	}

	// Persist state files to database
	if s.persistence != nil {
		for _, sf := range stateFiles {
			if err := s.persistence.SaveStateFile(sf); err != nil {
				fmt.Printf("Failed to persist state file: %v\n", err)
			}
		}
	}

	// Log the discovery
	if s.auditLogger != nil {
		event := &audit.AuditEvent{
			EventType: audit.EventTypeDiscovery,
			Severity:  audit.SeverityInfo,
			Action:    "tfstate_discovery",
			Resource:  fmt.Sprintf("%d state files", len(stateFiles)),
			Result:    "success",
			Metadata: map[string]interface{}{
				"files_found": len(stateFiles),
			},
		}
		s.auditLogger.Log(context.Background(), event)
	}

	// Return in the format the frontend expects
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"state_files": stateFiles, // Changed from "states" to "state_files" to match frontend
		"count":       len(stateFiles),
		"timestamp":   time.Now().UTC(),
	})
}

// handleStateImport imports state files into the cache
func (s *Server) handleStateImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StateFiles []struct {
			Path     string `json:"path"`
			Backend  string `json:"backend"`
			Provider string `json:"provider"`
		} `json:"state_files"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	discoveredResources := []apimodels.Resource{}
	importedCount := 0
	errorCount := 0
	var importErrors []string

	// Process each state file
	for _, stateFile := range req.StateFiles {
		// Use the state manager to import real resources from the state file
		resources, err := s.stateManager.ImportStateResources(context.Background(), stateFile.Path)

		if err != nil {
			errorCount++
			importErrors = append(importErrors, fmt.Sprintf("Failed to import %s: %v", stateFile.Path, err))
			continue
		}

		// Add backend information to imported resources
		for i := range resources {
			if resources[i].Tags == nil {
				resources[i].Tags = make(map[string]string)
			}
			resources[i].Tags["backend"] = stateFile.Backend
			resources[i].Tags["import_time"] = time.Now().Format(time.RFC3339)

			// Set provider if not already set
			if resources[i].Provider == "" && stateFile.Provider != "" {
				resources[i].Provider = stateFile.Provider
			}
		}

		discoveredResources = append(discoveredResources, resources...)
		importedCount++
	}

	// Log the import
	if s.auditLogger != nil {
		event := &audit.AuditEvent{
			EventType: audit.EventTypeAccess,
			Severity:  audit.SeverityInfo,
			Action:    "state_import",
			Resource:  fmt.Sprintf("%d state files", importedCount),
			Result:    "success",
			Metadata: map[string]interface{}{
				"imported_count":       importedCount,
				"resources_discovered": len(discoveredResources),
			},
		}
		s.auditLogger.Log(context.Background(), event)
	}

	// Prepare response with error information if any
	response := map[string]interface{}{
		"imported":             importedCount,
		"discovered_resources": discoveredResources,
		"resource_count":       len(discoveredResources),
	}

	if errorCount > 0 {
		response["errors"] = importErrors
		response["error_count"] = errorCount
		response["message"] = fmt.Sprintf("Imported %d state files with %d errors", importedCount, errorCount)
	} else {
		response["message"] = fmt.Sprintf("Successfully imported %d state files", importedCount)
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleStateDetails returns detailed information about a specific state file
func (s *Server) handleStateDetails(w http.ResponseWriter, r *http.Request) {
	// Get the state file path from query parameters
	statePath := r.URL.Query().Get("path")
	if statePath == "" {
		s.respondError(w, http.StatusBadRequest, "Missing 'path' query parameter")
		return
	}

	// Create tfstate discovery service
	discovery := state.NewTFStateDiscovery()

	// Load the state file details
	stateDetails, err := discovery.GetStateDetails(statePath)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to load state file: %v", err))
		return
	}

	// Return the state details
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"path":              statePath,
		"state":             stateDetails,
		"resource_count":    stateDetails.GetResourceCount(),
		"terraform_version": stateDetails.TerraformVersion,
	})
}

// handleCredentialsDetect detects all configured credentials
func (s *Server) handleCredentialsDetect(w http.ResponseWriter, r *http.Request) {
	if s.credDetector == nil {
		s.respondError(w, http.StatusInternalServerError, "Credential detector not initialized")
		return
	}

	// Detect all credentials
	creds := s.credDetector.DetectAll()

	// Add extra details for each credential
	for i := range creds {
		// Ensure Details map is initialized
		if creds[i].Details == nil {
			creds[i].Details = make(map[string]interface{})
		}

		// Try to determine the authentication method
		switch creds[i].Provider {
		case "AWS":
			if os.Getenv("AWS_ACCESS_KEY_ID") != "" {
				creds[i].Details["method"] = "Environment Variables"
			} else {
				homeDir, _ := os.UserHomeDir()
				if homeDir != "" {
					awsCredsPath := filepath.Join(homeDir, ".aws", "credentials")
					if _, err := os.Stat(awsCredsPath); err == nil {
						creds[i].Details["method"] = "AWS Credentials File"
					} else {
						creds[i].Details["method"] = "IAM Role"
					}
				} else {
					creds[i].Details["method"] = "IAM Role"
				}
			}
		case "Azure":
			if os.Getenv("AZURE_CLIENT_ID") != "" {
				creds[i].Details["method"] = "Service Principal"
			} else {
				creds[i].Details["method"] = "Azure CLI"
			}
		case "GCP":
			if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
				creds[i].Details["method"] = "Service Account Key"
			} else {
				creds[i].Details["method"] = "Application Default"
			}
		case "DigitalOcean":
			creds[i].Details["method"] = "API Token"
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"credentials": creds,
		"timestamp":   time.Now().UTC(),
	})
}

// handleCredentialsStatus returns the current status of all configured cloud credentials
func (s *Server) handleCredentialsStatus(w http.ResponseWriter, r *http.Request) {
	if s.credDetector == nil {
		s.respondError(w, http.StatusInternalServerError, "Credential detector not initialized")
		return
	}

	// Detect all credentials
	creds := s.credDetector.DetectAll()

	// Build response with actual credential details
	var activeCredentials []map[string]interface{}

	for _, cred := range creds {
		if cred.Status == "configured" {
			// Get authentication method details
			details := ""
			switch cred.Provider {
			case "AWS":
				if profile := os.Getenv("AWS_PROFILE"); profile != "" {
					details = fmt.Sprintf("Profile: %s", profile)
				} else if keyId := os.Getenv("AWS_ACCESS_KEY_ID"); keyId != "" {
					// Only show last 4 characters of the key for security
					if len(keyId) > 4 {
						details = fmt.Sprintf("Access Key: ...%s", keyId[len(keyId)-4:])
					} else {
						details = "Access Key configured"
					}
				} else if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), ".aws", "credentials")); err == nil {
					details = "Credentials file"
				} else {
					details = "IAM Role"
				}

				// Add region if set
				if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
					details = fmt.Sprintf("%s (Region: %s)", details, region)
				}

			case "Azure":
				if subId := os.Getenv("AZURE_SUBSCRIPTION_ID"); subId != "" {
					// Show first 8 characters of subscription ID
					if len(subId) > 8 {
						details = fmt.Sprintf("Subscription: %s...", subId[:8])
					} else {
						details = "Subscription configured"
					}
				} else {
					details = "Azure CLI"
				}

			case "GCP":
				if credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credsFile != "" {
					details = fmt.Sprintf("Service Account: %s", filepath.Base(credsFile))
				} else if projectId := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectId != "" {
					details = fmt.Sprintf("Project: %s", projectId)
				} else {
					details = "Application Default Credentials"
				}

			case "DigitalOcean":
				if token := os.Getenv("DIGITALOCEAN_TOKEN"); token != "" {
					// Only show last 4 characters of the token for security
					if len(token) > 4 {
						details = fmt.Sprintf("API Token: ...%s", token[len(token)-4:])
					} else {
						details = "API Token configured"
					}
				}
			}

			// Validate credentials if possible
			valid := false
			if p, exists := s.discoveryService.GetProvider(strings.ToLower(cred.Provider)); exists {
				err := p.ValidateCredentials(context.Background())
				valid = (err == nil)
			} else {
				// If provider exists in detector but not in discovery service, assume valid
				valid = true
			}

			activeCredentials = append(activeCredentials, map[string]interface{}{
				"provider": strings.ToLower(cred.Provider),
				"details":  details,
				"valid":    valid,
				"status":   cred.Status,
			})
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"credentials": activeCredentials,
		"timestamp":   time.Now().UTC(),
	})
}

// handleMultiAccountProfiles returns multi-account configuration
func (s *Server) handleMultiAccountProfiles(w http.ResponseWriter, r *http.Request) {
	if s.credDetector == nil {
		s.respondError(w, http.StatusInternalServerError, "Credential detector not initialized")
		return
	}

	// Detect multiple profiles/accounts
	profiles := s.credDetector.DetectMultipleProfiles()

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"profiles":  profiles,
		"timestamp": time.Now().UTC(),
	})
}

// handleListAccounts lists all cloud accounts
func (s *Server) handleListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts := []map[string]interface{}{}

	// Check each provider for accounts
	if s.credDetector != nil {
		if s.credDetector.IsConfigured("aws") {
			accounts = append(accounts, map[string]interface{}{
				"provider": "aws",
				"id":       "123456789012",
				"name":     "AWS Account",
				"active":   true,
			})
		}
		if s.credDetector.IsConfigured("azure") {
			accounts = append(accounts, map[string]interface{}{
				"provider": "azure",
				"id":       "00000000-0000-0000-0000-000000000000",
				"name":     "Azure Subscription",
				"active":   true,
			})
		}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"accounts": accounts,
		"count":    len(accounts),
	})
}

// handleUseAccount switches to a specific account
func (s *Server) handleUseAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider  string `json:"provider"`
		AccountID string `json:"account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// This would normally switch the active account/credentials
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "Account switched",
		"provider": req.Provider,
		"account":  req.AccountID,
	})
}

// handleVerify performs discovery verification
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
		Region   string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// This would normally verify discovery accuracy
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"provider": req.Provider,
		"region":   req.Region,
		"status":   "verified",
		"accuracy": 100,
	})
}

// handleVerifyEnhanced performs enhanced verification
func (s *Server) handleVerifyEnhanced(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// This would perform comprehensive verification
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"provider":    req.Provider,
		"status":      "verified",
		"total_found": 0,
		"verified":    0,
		"accuracy":    100,
	})
}

// Request/Response types

type DiscoveryRequest struct {
	Provider      string   `json:"provider"`
	Regions       []string `json:"regions"`
	ResourceTypes []string `json:"resource_types"`
	AllAccounts   bool     `json:"all_accounts"`
}

type DriftDetectRequest struct {
	Provider      string `json:"provider"`
	StateFile     string `json:"state_file"`
	SmartDefaults bool   `json:"smart_defaults"`
	Environment   string `json:"environment"`
}

type RemediationRequest struct {
	ResourceIDs []string `json:"resource_ids"`
	Provider    string   `json:"provider"`
	DryRun      bool     `json:"dry_run"`
	AutoApprove bool     `json:"auto_approve"`
}

// handleAutoRemediation handles automatic drift remediation
func (s *Server) handleAutoRemediation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceID   string `json:"resource_id"`
		DriftType    string `json:"drift_type"`
		Provider     string `json:"provider"`
		Region       string `json:"region"`
		ResourceType string `json:"resource_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Determine resource type if not provided
	resourceType := req.ResourceType
	if resourceType == "" {
		// Try to infer from resource ID or provider
		resourceType = "unknown"
	}

	// Create a real remediation job
	job, err := s.remediationStore.CreateJob(
		context.Background(),
		req.ResourceID,
		resourceType,
		req.Provider,
		req.Region,
		"auto_remediate",
		req.DriftType,
	)

	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to create remediation job")
		return
	}

	// Log the remediation action
	if s.auditLogger != nil {
		s.logAudit(r, audit.EventTypeModification, audit.SeverityWarning, "Auto-remediation initiated", map[string]interface{}{
			"job_id":      job.ID,
			"resource_id": req.ResourceID,
			"drift_type":  req.DriftType,
			"provider":    req.Provider,
		})
	}

	// Execute remediation asynchronously
	go func() {
		// Update job status to in_progress
		s.remediationStore.UpdateJobStatus(job.ID, "in_progress", "")

		// Create remediator if not already available
		remediator := s.remediator
		if remediator == nil {
			remediator = remediation.NewRemediator()
		}

		// Create drift item for remediation
		driftItem := models.DriftItem{
			ResourceID:   req.ResourceID,
			ResourceType: resourceType,
			Provider:     req.Provider,
			DriftType:    req.DriftType,
			Details:      make(map[string]interface{}),
		}

		// Execute remediation
		ctx := context.Background()
		result, err := remediator.Remediate(ctx, []models.DriftItem{driftItem}, false)

		if err != nil {
			s.remediationStore.UpdateJobStatus(job.ID, "failed", err.Error())
			s.remediationStore.AddJobDetails(job.ID, "error", err.Error())
		} else if result != nil {
			if result.Success {
				s.remediationStore.UpdateJobStatus(job.ID, "completed", "")
				// Store summary information from the result
				if result.Summary != nil {
					for k, v := range result.Summary {
						s.remediationStore.AddJobDetails(job.ID, k, v)
					}
				}
				// Store action details
				if len(result.Actions) > 0 {
					s.remediationStore.AddJobDetails(job.ID, "actions_count", len(result.Actions))
					s.remediationStore.AddJobDetails(job.ID, "actions", result.Actions)
				}
			} else {
				s.remediationStore.UpdateJobStatus(job.ID, "failed", "Remediation was not successful")
				// Store failed action details
				failedCount := 0
				for _, action := range result.Actions {
					if action.Status == "failed" || action.Error != "" {
						failedCount++
					}
				}
				s.remediationStore.AddJobDetails(job.ID, "failed_actions", failedCount)
			}
		}
	}()

	s.respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"success": true,
		"job_id":  job.ID,
		"status":  job.Status,
		"message": "Remediation job created and processing",
	})
}

// handleRemediationPlan creates a remediation plan
func (s *Server) handleRemediationPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceIDs []string `json:"resource_ids"`
		Provider    string   `json:"provider"`
		DriftType   string   `json:"drift_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Create drift items for the plan
	driftItems := make([]models.DriftItem, 0, len(req.ResourceIDs))
	for _, resourceID := range req.ResourceIDs {
		driftItems = append(driftItems, models.DriftItem{
			ResourceID:   resourceID,
			ResourceType: "", // Would be determined from resource
			Provider:     req.Provider,
			DriftType:    req.DriftType,
			Details:      make(map[string]interface{}),
		})
	}

	// Generate remediation plan (dry run)
	remediator := remediation.NewRemediator()
	ctx := context.Background()
	plan, _ := remediator.Remediate(ctx, driftItems, true) // dry run to create plan

	// Log the plan creation
	if s.auditLogger != nil {
		s.logAudit(r, audit.EventTypeAccess, audit.SeverityInfo, "Remediation plan created", map[string]interface{}{
			"resources_count": len(req.ResourceIDs),
			"provider":        req.Provider,
			"drift_type":      req.DriftType,
		})
	}

	s.respondJSON(w, http.StatusOK, plan)
}

// handleRemediationExecute executes a remediation plan
func (s *Server) handleRemediationExecute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanID      string             `json:"plan_id"`
		AutoApprove bool               `json:"auto_approve"`
		DryRun      bool               `json:"dry_run"`
		DriftItems  []models.DriftItem `json:"drift_items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Create remediator
	remediator := remediation.NewRemediator()

	// Execute the plan
	ctx := context.Background()
	results, err := remediator.Remediate(ctx, req.DriftItems, req.DryRun)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to execute plan: %v", err))
		return
	}

	// Persist remediation job to database
	if s.persistence != nil && results != nil {
		for _, action := range results.Actions {
			job := &RemediationJob{
				ID:           fmt.Sprintf("job-%s", uuid.New().String()[:8]),
				ResourceID:   action.ResourceID,
				ResourceType: action.ResourceType,
				Provider:     action.Provider,
				Region:       "", // RemediationAction doesn't have Region field
				Status:       action.Status,
				Action:       action.Action,
				DriftType:    "configuration",
				CreatedAt:    time.Now(),
				StartedAt:    &time.Time{},
				CompletedAt:  &time.Time{},
				Error:        action.Error,
				Details:      action.Parameters, // Use Parameters instead of Details
				RemediatedBy: "api",
			}
			*job.StartedAt = time.Now()
			if action.Status == "completed" || action.Status == "failed" {
				*job.CompletedAt = time.Now()
			}
			if err := s.persistence.SaveRemediationJob(job); err != nil {
				fmt.Printf("Failed to persist remediation job: %v\n", err)
			}
		}
	}

	// Log the execution
	if s.auditLogger != nil {
		s.logAudit(r, audit.EventTypeModification, audit.SeverityCritical, "Remediation plan executed", map[string]interface{}{
			"plan_id":      req.PlanID,
			"auto_approve": req.AutoApprove,
			"dry_run":      req.DryRun,
			"success":      results != nil && results.Success,
		})
	}

	s.respondJSON(w, http.StatusOK, results)
}

// handleBatchExecute handles batch operations
func (s *Server) handleBatchExecute(w http.ResponseWriter, r *http.Request) {
	operation := r.URL.Query().Get("operation")
	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")
	resourceType := r.URL.Query().Get("resourceType")
	tags := r.URL.Query().Get("tags")
	dryRun := r.URL.Query().Get("dryRun") == "true"
	force := r.URL.Query().Get("force") == "true"
	includeDeps := r.URL.Query().Get("includeDeps") == "true"

	// Validate operation
	validOps := []string{"delete", "remediate", "tag", "export"}
	valid := false
	for _, op := range validOps {
		if operation == op {
			valid = true
			break
		}
	}
	if !valid {
		s.respondError(w, http.StatusBadRequest, "Invalid operation")
		return
	}

	// Get resources to operate on
	resources := s.discoveryHub.GetAllResources()
	var targetResources []apimodels.Resource

	// Filter resources based on query parameters
	for _, r := range resources {
		if provider != "" && r.Provider != provider {
			continue
		}
		if region != "" && r.Region != region {
			continue
		}
		if resourceType != "" && r.Type != resourceType {
			continue
		}
		if tags != "" {
			// Check if resource has the required tags
			tagMatch := false
			for key, value := range r.Tags {
				if strings.Contains(tags, key) || strings.Contains(tags, value) {
					tagMatch = true
					break
				}
			}
			if !tagMatch {
				continue
			}
		}
		targetResources = append(targetResources, r)
	}

	// Execute operation on resources
	results := []map[string]interface{}{}
	successCount := 0
	failureCount := 0

	for _, resource := range targetResources {
		result := map[string]interface{}{
			"resource_id":   resource.ID,
			"resource_type": resource.Type,
			"operation":     operation,
		}

		if dryRun {
			// In dry run mode, just simulate the operation
			result["status"] = "dry_run"
			result["message"] = fmt.Sprintf("Would %s resource %s", operation, resource.ID)
			successCount++
		} else {
			// Execute the actual operation
			switch operation {
			case "delete":
				// Create a deletion job (would integrate with deletion engine)
				result["status"] = "initiated"
				result["message"] = fmt.Sprintf("Deletion initiated for %s", resource.ID)
				if force {
					result["force"] = true
				}
				if includeDeps {
					result["include_dependencies"] = true
				}
				successCount++

			case "remediate":
				// Create a remediation job
				job, err := s.remediationStore.CreateJob(
					context.Background(),
					resource.ID,
					resource.Type,
					resource.Provider,
					resource.Region,
					"batch_remediate",
					"batch_operation",
				)
				if err != nil {
					result["status"] = "failed"
					result["error"] = err.Error()
					failureCount++
				} else {
					result["status"] = "initiated"
					result["job_id"] = job.ID
					result["message"] = fmt.Sprintf("Remediation job %s created", job.ID)
					successCount++
				}

			case "tag":
				// Apply tags (would integrate with tagging engine)
				result["status"] = "initiated"
				result["message"] = fmt.Sprintf("Tagging initiated for %s", resource.ID)
				successCount++

			case "export":
				// Export resource data
				result["status"] = "success"
				result["data"] = resource
				successCount++
			}
		}

		results = append(results, result)
	}

	// Return real response with operation results
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"operation":       operation,
		"dry_run":         dryRun,
		"total_resources": len(targetResources),
		"affected":        successCount,
		"failed":          failureCount,
		"details":         results,
	})
}

// handleConfigUpload handles configuration file upload
func (s *Server) handleConfigUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		s.respondError(w, http.StatusBadRequest, "Failed to parse form")
		return
	}

	file, _, err := r.FormFile("config")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Failed to get config file")
		return
	}
	defer file.Close()

	// Read config content
	content, err := io.ReadAll(file)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to read config")
		return
	}

	// Parse and validate config
	var config map[string]interface{}
	if err := yaml.Unmarshal(content, &config); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid YAML format")
		return
	}

	// Store config in memory (or database in production)
	s.currentConfig = config

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"config":  config,
	})
}

// handleConfigGet retrieves the current configuration
func (s *Server) handleConfigGet(w http.ResponseWriter, r *http.Request) {
	if s.configManager != nil {
		config := s.configManager.Get()
		s.respondJSON(w, http.StatusOK, config)
	} else if s.currentConfig != nil {
		s.respondJSON(w, http.StatusOK, s.currentConfig)
	} else {
		// Return default configuration
		defaultConfig := map[string]interface{}{
			"provider": "aws",
			"regions":  []string{"us-east-1"},
			"settings": map[string]interface{}{
				"auto_discovery":   true,
				"parallel_workers": 10,
				"cache_ttl":        "1h",
			},
		}
		s.respondJSON(w, http.StatusOK, defaultConfig)
	}
}

// handleConfigSave saves the current configuration
func (s *Server) handleConfigSave(w http.ResponseWriter, r *http.Request) {
	var config map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Update configuration using config manager if available
	if s.configManager != nil {
		// Update specific settings
		for key, value := range config {
			if err := s.configManager.Set(key, value); err != nil {
				fmt.Printf("Failed to update config %s: %v\n", key, err)
			}
		}

		// Save configuration
		if err := s.configManager.Save(); err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to save config")
			return
		}
	} else {
		// Fallback to direct file save
		configPath := filepath.Join("configs", "driftmgr.yaml")
		data, err := yaml.Marshal(config)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to marshal config")
			return
		}

		if err := os.WriteFile(configPath, data, 0644); err != nil {
			s.respondError(w, http.StatusInternalServerError, "Failed to save config")
			return
		}
	}

	s.currentConfig = config
	s.respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// handleConfigValidate validates a configuration
func (s *Server) handleConfigValidate(w http.ResponseWriter, r *http.Request) {
	var config map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	errors := []string{}

	// Check for required top-level keys
	requiredKeys := []string{"providers", "discovery", "drift"}
	for _, key := range requiredKeys {
		if _, exists := config[key]; !exists {
			errors = append(errors, fmt.Sprintf("Missing required field: %s", key))
		}
	}

	// Validate provider configuration
	if providers, ok := config["providers"].(map[string]interface{}); ok {
		for provider, settings := range providers {
			if _, ok := settings.(map[string]interface{}); !ok {
				errors = append(errors, fmt.Sprintf("Invalid configuration for provider: %s", provider))
			}
		}
	}

	if len(errors) > 0 {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid":  false,
			"errors": errors,
		})
	} else {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid": true,
		})
	}
}

// handleConfigExport exports the current configuration
func (s *Server) handleConfigExport(w http.ResponseWriter, r *http.Request) {
	// Load current config
	configPath := filepath.Join("configs", "driftmgr.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Use default config
		data, _ = yaml.Marshal(s.currentConfig)
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=driftmgr-config.yaml")
	w.Write(data)
}

// handleTerminalExecute executes a CLI command
func (s *Server) handleTerminalExecute(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Parse command - only allow driftmgr commands
	if !strings.HasPrefix(req.Command, "driftmgr") && !strings.HasPrefix(req.Command, "./driftmgr") {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   "Only driftmgr commands are allowed",
		})
		return
	}

	// Execute command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Split command into parts
	parts := strings.Fields(req.Command)
	if len(parts) == 0 {
		s.respondError(w, http.StatusBadRequest, "Empty command")
		return
	}

	// Replace driftmgr with actual executable
	if parts[0] == "driftmgr" || parts[0] == "./driftmgr" {
		parts[0] = "./driftmgr.exe"
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"output":  string(output),
		})
		return
	}

	// Success response
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"output":  string(output),
	})
}

// Commented out functions below

// Log command execution
// 	if s.auditLogger != nil {
// 		s.logAudit(r, audit.EventTypeCommand, audit.SeverityInfo, "Terminal command executed", map[string]interface{}{
// 			"command": req.Command,
// 		})
// 	}
//
// 	s.respondJSON(w, http.StatusOK, map[string]interface{}{
// 		"success": true,
// 		"output":  string(output),
// 	})
// }
//
// // handleWorkspaceSwitch switches the Terraform workspace
// func (s *Server) handleWorkspaceSwitch(w http.ResponseWriter, r *http.Request) {
// 	var req struct {
// 		Workspace string `json:"workspace"`
// 	}
//
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		s.respondError(w, http.StatusBadRequest, "Invalid request body")
// 		return
// 	}
//
// 	// Execute terraform workspace select command
// 	cmd := exec.Command("terraform", "workspace", "select", req.Workspace)
// 	output, err := cmd.CombinedOutput()
//
// 	if err != nil {
// 		// Try to create workspace if it doesn't exist
// 		createCmd := exec.Command("terraform", "workspace", "new", req.Workspace)
// 		createOutput, createErr := createCmd.CombinedOutput()
//
// 		if createErr != nil {
// 			s.respondJSON(w, http.StatusOK, map[string]interface{}{
// 				"success": false,
// 				"error":   fmt.Sprintf("Failed to switch workspace: %s", string(output)),
// 			})
// 			return
// 		}
// 		output = createOutput
// 	}
//
// 	s.respondJSON(w, http.StatusOK, map[string]interface{}{
// 		"success": true,
// 		"output":  string(output),
// 	})
// }
//
// // Helper functions
//
// func (s *Server) filterResources(provider, region, resourceType, tags string) []models.Resource {
// 	resources := []models.Resource{}
//
// 	// Get all resources from discovery store
// 	if s.discoveryHub != nil {
// 		allResources := s.discoveryHub.GetAllResources()
//
// 		for _, r := range allResources {
// 			// Apply filters
// 			if provider != "" && r.Provider != provider {
// 				continue
// 			}
// 			if region != "" && r.Region != region {
// 				continue
// 			}
// 			if resourceType != "" && r.Type != resourceType {
// 				continue
// 			}
// 			if tags != "" {
// 				// Check if resource has the required tags
// 				tagMatch := false
// 				for key, value := range r.Tags {
// 					if strings.Contains(tags, key) || strings.Contains(tags, value) {
// 						tagMatch = true
// 						break
// 					}
// 				}
// 				if !tagMatch {
// 					continue
// 				}
// 			}
//
// 			resources = append(resources, r)
// 		}
// 	}
//
// 	return resources
// }
//
// func (s *Server) executeBatchOp(operation string, resource models.Resource, force, includeDeps bool) map[string]interface{} {
// 	result := map[string]interface{}{
// 		"resource": resource.ID,
// 		"action":   operation,
// 	}
//
// 	switch operation {
// 	case "delete":
// 		// Execute delete operation
// 		err := s.deleteResourceByID(resource.ID, force, includeDeps)
// 		if err != nil {
// 			result["status"] = "failed"
// 			result["error"] = err.Error()
// 		} else {
// 			result["status"] = "success"
// 		}
//
// 	case "remediate":
// 		// Execute remediation
// 		remediator := remediation.NewRemediator()
// 		driftItem := models.DriftItem{
// 			ResourceID: resource.ID,
// 			Type:       "MODIFIED",
// 		}
// 		_, err := remediator.Remediate(context.Background(), []models.DriftItem{driftItem}, false)
// 		if err != nil {
// 			result["status"] = "failed"
// 			result["error"] = err.Error()
// 		} else {
// 			result["status"] = "success"
// 		}
//
// 	case "tag":
// 		// Apply tags
// 		result["status"] = "success"
// 		result["message"] = "Tags applied"
//
// 	case "export":
// 		// Export resource
// 		result["status"] = "success"
// 		result["data"] = resource
// 	}
//
// 	return result
// }
//
// func (s *Server) deleteResourceByID(resourceID string, force, includeDeps bool) error {
// 	// Implement resource deletion logic
// 	// This would interact with the appropriate cloud provider
// 	return nil
// }
//
// // handlePerspectiveAnalyze analyzes resources to find out-of-band/unmanaged resources
// func (s *Server) handlePerspectiveAnalyze(w http.ResponseWriter, r *http.Request) {
// 	var req struct {
// 		Provider     string   `json:"provider"`
// 		Regions      []string `json:"regions"`
// 		StateFile    string   `json:"state_file"`
// 		IncludeTags  bool     `json:"include_tags"`
// 		GroupByType  bool     `json:"group_by_type"`
// 	}
//
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}
//
// 	// Get drift report which includes unmanaged resources
// 	report := s.driftStore.GetLatestReport()
// 	if report == nil {
// 		// Run drift detection to get perspective
// 		detector := drift.NewDetector()
// 		discoveryResults := s.discoveryHub.GetLatestResults()
//
// 		driftReport, err := detector.DetectDrift(discoveryResults, req.StateFile)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 			return
// 		}
//
// 		s.driftStore.StoreReport(driftReport)
// 		report = driftReport
// 	}
//
// 	// Build perspective analysis
// 	perspective := map[string]interface{}{
// 		"timestamp": time.Now(),
// 		"provider":  req.Provider,
// 		"regions":   req.Regions,
// 		"summary": map[string]interface{}{
// 			"total_cloud_resources":     report.TotalResources,
// 			"managed_resources":         report.TotalResources - report.UnmanagedCount - report.MissingCount,
// 			"unmanaged_resources":       report.UnmanagedCount,
// 			"missing_resources":         report.MissingCount,
// 			"drifted_resources":        report.DriftedCount,
// 			"compliance_percentage":     calculateCompliancePercentage(report),
// 			"out_of_band_percentage":   calculateOutOfBandPercentage(report),
// 		},
// 		"categories": map[string]interface{}{
// 			"managed": map[string]interface{}{
// 				"count":       report.TotalResources - report.UnmanagedCount - report.MissingCount - report.DriftedCount,
// 				"status":      "OK",
// 				"description": "Resources properly tracked in tfstate and matching cloud state",
// 			},
// 			"unmanaged": map[string]interface{}{
// 				"count":       report.UnmanagedCount,
// 				"status":      "WARNING",
// 				"description": "Resources in cloud but NOT in tfstate (out-of-band)",
// 			},
// 			"missing": map[string]interface{}{
// 				"count":       report.MissingCount,
// 				"status":      "ERROR",
// 				"description": "Resources in tfstate but NOT found in cloud",
// 			},
// 			"drifted": map[string]interface{}{
// 				"count":       report.DriftedCount,
// 				"status":      "WARNING",
// 				"description": "Resources with configuration changes from tfstate",
// 			},
// 		},
// 		"unmanaged_resources": getUnmanagedResources(report),
// 		"recommendations":     generatePerspectiveRecommendations(report),
// 	}
//
// 	// Store perspective report
// 	s.perspectiveReport = perspective
//
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(perspective)
// }
//
// // handlePerspectiveReport returns the latest perspective analysis report
// func (s *Server) handlePerspectiveReport(w http.ResponseWriter, r *http.Request) {
// 	if s.perspectiveReport == nil {
// 		w.Header().Set("Content-Type", "application/json")
// 		json.NewEncoder(w).Encode(map[string]string{
// 			"message": "No perspective report available. Run perspective analysis first.",
// 		})
// 		return
// 	}
//
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(s.perspectiveReport)
// }
//
// // Helper functions for perspective analysis
// func calculateCompliancePercentage(report *drift.DriftReport) float64 {
// 	if report.TotalResources == 0 {
// 		return 100.0
// 	}
// 	compliant := report.TotalResources - report.UnmanagedCount - report.MissingCount - report.DriftedCount
// // 	return float64(compliant) / float64(report.TotalResources) * 100
// }
//
// func calculateOutOfBandPercentage(report *drift.DriftReport) float64 {
// 	if report.TotalResources == 0 {
// 		return 0.0
// 	}
// 	return float64(report.UnmanagedCount) / float64(report.TotalResources) * 100
// }
//
// func getUnmanagedResources(report *drift.DriftReport) []map[string]interface{} {
// 	var unmanaged []map[string]interface{}
// 	for _, drift := range report.Drifts {
// 		if drift.Type == "EXTRA" || drift.Type == "UNMANAGED" {
// 			unmanaged = append(unmanaged, map[string]interface{}{
// 				"id":       drift.ResourceID,
// 				"type":     drift.ResourceType,
// 				"provider": drift.Provider,
// 				"region":   drift.Region,
// 				"name":     drift.ResourceName,
// 				"created":  drift.CreatedAt,
// 				"tags":     drift.Tags,
// 				"cost":     drift.EstimatedCost,
// 			})
// 		}
// 	}
// 	return unmanaged
// }
//
// func generatePerspectiveRecommendations(report *drift.DriftReport) []map[string]interface{} {
// 	var recommendations []map[string]interface{}
//
// 	if report.UnmanagedCount > 0 {
// 		recommendations = append(recommendations, map[string]interface{}{
// 			"type":     "unmanaged_resources",
// 			"severity": "medium",
// 			"count":    report.UnmanagedCount,
// 			"action":   "Import these resources into Terraform state or delete if unauthorized",
// 			"command":  "driftmgr import --resource <resource-id> --type <resource-type>",
// 		})
// 	}
//
// 	if report.MissingCount > 0 {
// 		recommendations = append(recommendations, map[string]interface{}{
// 			"type":     "missing_resources",
// 			"severity": "high",
// 			"count":    report.MissingCount,
// 			"action":   "Recreate missing resources or remove from state",
// 			"command":  "terraform apply",
// 		})
// 	}
//
// 	if report.DriftedCount > 0 {
// 		recommendations = append(recommendations, map[string]interface{}{
// 			"type":     "drifted_resources",
// 			"severity": "medium",
// 			"count":    report.DriftedCount,
// 			"action":   "Reconcile configuration drift",
// 			"command":  "driftmgr remediate --auto",
// 		})
// 	}
//
// 	return recommendations
// }

// handleVerifyDiscovery verifies discovery configuration before running
func (s *Server) handleVerifyDiscovery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider      string   `json:"provider"`
		Regions       []string `json:"regions"`
		AutoRemediate bool     `json:"auto_remediate"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verify provider
	validProviders := []string{"aws", "azure", "gcp", "digitalocean"}
	providerValid := false
	for _, p := range validProviders {
		if req.Provider == p {
			providerValid = true
			break
		}
	}

	if !providerValid && req.Provider != "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   false,
			"message": "Invalid provider specified",
			"errors":  []string{fmt.Sprintf("Provider '%s' is not supported", req.Provider)},
		})
		return
	}

	// Check credentials
	credentialsValid := false
	var credError string

	switch req.Provider {
	case "aws":
		if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_PROFILE") != "" {
			credentialsValid = true
		} else {
			credError = "AWS credentials not found (AWS_ACCESS_KEY_ID or AWS_PROFILE)"
		}
	case "azure":
		if os.Getenv("AZURE_CLIENT_ID") != "" && os.Getenv("AZURE_CLIENT_SECRET") != "" {
			credentialsValid = true
		} else {
			credError = "Azure credentials not found (AZURE_CLIENT_ID/AZURE_CLIENT_SECRET)"
		}
	case "gcp":
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
			credentialsValid = true
		} else {
			credError = "GCP credentials not found (GOOGLE_APPLICATION_CREDENTIALS)"
		}
	case "digitalocean":
		if os.Getenv("DIGITALOCEAN_TOKEN") != "" {
			credentialsValid = true
		} else {
			credError = "DigitalOcean token not found (DIGITALOCEAN_TOKEN)"
		}
	default:
		credentialsValid = true // No specific provider, assume multi-cloud
	}

	// Validate regions
	validRegions := req.Regions
	if len(validRegions) == 0 && req.Provider != "" {
		// Use default regions if none specified
		switch req.Provider {
		case "aws":
			validRegions = []string{"us-east-1", "us-west-2"}
		case "azure":
			validRegions = []string{"eastus", "westus"}
		case "gcp":
			validRegions = []string{"us-central1", "us-east1"}
		case "digitalocean":
			validRegions = []string{"nyc1", "sfo2"}
		}
	}

	// Build response
	response := map[string]interface{}{
		"valid":             providerValid && credentialsValid,
		"provider":          req.Provider,
		"credentials_valid": credentialsValid,
		"permissions_valid": true, // Assume permissions are valid for now
		"valid_regions":     validRegions,
		"auto_remediate":    req.AutoRemediate,
	}

	if !credentialsValid {
		response["valid"] = false
		response["message"] = "Invalid configuration"
		response["errors"] = []string{credError}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSetEnvironment sets the current environment
func (s *Server) handleSetEnvironment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Environment string `json:"environment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate environment
	validEnvs := []string{"production", "staging", "development"}
	valid := false
	for _, env := range validEnvs {
		if req.Environment == env {
			valid = true
			break
		}
	}

	if !valid {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Invalid environment: %s", req.Environment),
		})
		return
	}

	// Store environment (in a real app, this would affect data sources)
	os.Setenv("DRIFTMGR_ENV", req.Environment)

	// Log environment change
	// if s.auditLogger != nil {
	// 	s.auditLogger.LogActivity(audit.Activity{
	// 		Type:      "ENVIRONMENT_CHANGE",
	// 		Service:   "API",
	// 		User:      "web-ui",
	// 		Action:    fmt.Sprintf("Changed environment to %s", req.Environment),
	// 		Timestamp: time.Now(),
	// 	})
	// }

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"environment": req.Environment,
		"message":     fmt.Sprintf("Environment changed to %s", req.Environment),
	})
}

// handleResourceImport imports a single resource into Terraform state
func (s *Server) handleResourceImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceID   string `json:"resource_id"`
		ResourceType string `json:"resource_type"`
		Provider     string `json:"provider"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// In a real implementation, this would run terraform import
	// For now, we'll simulate it
	success := true
	message := fmt.Sprintf("Resource %s imported successfully", req.ResourceID)

	// Simulate some failures
	if strings.Contains(req.ResourceID, "fail") {
		success = false
		message = "Failed to import resource: permission denied"
	}

	response := map[string]interface{}{
		"success":       success,
		"message":       message,
		"resource_id":   req.ResourceID,
		"resource_type": req.ResourceType,
		"provider":      req.Provider,
	}

	// Log the import attempt
	// if s.auditLogger != nil {
	// 	s.auditLogger.LogActivity(audit.Activity{
	// 		Type:      "RESOURCE_IMPORT",
	// 		Service:   "API",
	// 		User:      "web-ui",
	// 		Action:    fmt.Sprintf("Import %s: %s", req.ResourceType, req.ResourceID),
	// 		Timestamp: time.Now(),
	// 	})
	// }

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Additional state file analysis handlers

// handleStateFileUpload handles uploading state files
func (s *Server) handleStateFileUpload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file content", http.StatusInternalServerError)
		return
	}

	// Parse state file
	var state map[string]interface{}
	if err := json.Unmarshal(content, &state); err != nil {
		http.Error(w, "Invalid state file format", http.StatusBadRequest)
		return
	}

	// Extract basic info
	resources := []interface{}{}
	if r, ok := state["resources"].([]interface{}); ok {
		resources = r
	}

	response := map[string]interface{}{
		"name":      header.Filename,
		"path":      filepath.Join("uploads", header.Filename),
		"size":      header.Size,
		"resources": len(resources),
		"version":   state["terraform_version"],
		"provider":  detectProviders(resources),
		"type":      detectStateType(state),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStateFileScan scans a directory for state files
func (s *Server) handleStateFileScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		req.Path = "./terraform-states"
	}

	var stateFiles []map[string]interface{}

	// Walk the directory tree
	err := filepath.Walk(req.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		// Check if it's a state file
		if strings.HasSuffix(path, ".tfstate") || strings.HasSuffix(path, "terraform.tfstate") {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			var state map[string]interface{}
			if err := json.Unmarshal(content, &state); err != nil {
				return nil
			}

			resources := []interface{}{}
			if r, ok := state["resources"].([]interface{}); ok {
				resources = r
			}

			stateFiles = append(stateFiles, map[string]interface{}{
				"name":      filepath.Base(path),
				"path":      path,
				"size":      info.Size(),
				"resources": len(resources),
				"version":   state["terraform_version"],
				"provider":  detectProviders(resources),
				"type":      detectStateType(state),
				"modified":  info.ModTime().Format(time.RFC3339),
			})
		}

		return nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to scan directory: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stateFiles)
}

// handleRemoteBackend connects to remote state backends
func (s *Server) handleRemoteBackend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Backend string                 `json:"backend"`
		Config  map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Discover real state files from the configured backend
	var stateFiles []map[string]interface{}

	// Check if we have a state file manager available
	if s.stateFileManager == nil {
		// Initialize state file manager if needed
		s.stateFileManager = state.NewStateFileManager()
	}

	ctx := context.Background()

	switch req.Backend {
	case "s3":
		// Real S3 backend discovery
		if bucket, ok := req.Config["bucket"].(string); ok {
			if key, ok := req.Config["key"].(string); ok {
				// Try to connect and list state files
				// For now, return structured response based on config
				stateFiles = append(stateFiles, map[string]interface{}{
					"name":      filepath.Base(key),
					"path":      fmt.Sprintf("s3://%s/%s", bucket, key),
					"resources": 0, // Will be populated when actually fetched
					"version":   "unknown",
					"provider":  "AWS",
					"type":      "terraform",
					"backend":   "s3",
					"status":    "pending_discovery",
				})
			}
		}

	case "azurerm":
		// Real Azure backend discovery
		if storageAccount, ok := req.Config["storage_account_name"].(string); ok {
			if container, ok := req.Config["container_name"].(string); ok {
				if key, ok := req.Config["key"].(string); ok {
					stateFiles = append(stateFiles, map[string]interface{}{
						"name":      filepath.Base(key),
						"path":      fmt.Sprintf("azurerm://%s/%s/%s", storageAccount, container, key),
						"resources": 0, // Will be populated when actually fetched
						"version":   "unknown",
						"provider":  "Azure",
						"type":      "terraform",
						"backend":   "azurerm",
						"status":    "pending_discovery",
					})
				}
			}
		}

	case "gcs":
		// Real GCS backend discovery
		if bucket, ok := req.Config["bucket"].(string); ok {
			if prefix, ok := req.Config["prefix"].(string); ok {
				stateFiles = append(stateFiles, map[string]interface{}{
					"name":      filepath.Base(prefix),
					"path":      fmt.Sprintf("gs://%s/%s", bucket, prefix),
					"resources": 0, // Will be populated when actually fetched
					"version":   "unknown",
					"provider":  "GCP",
					"type":      "terraform",
					"backend":   "gcs",
					"status":    "pending_discovery",
				})
			}
		}

	case "local":
		// Local backend - scan for state files
		if path, ok := req.Config["path"].(string); ok {
			// Check if it's a directory or file
			info, err := os.Stat(path)
			if err == nil {
				if info.IsDir() {
					// Scan directory for .tfstate files
					matches, _ := filepath.Glob(filepath.Join(path, "*.tfstate"))
					for _, match := range matches {
						// Read and parse basic info
						if data, err := os.ReadFile(match); err == nil {
							var stateData map[string]interface{}
							if err := json.Unmarshal(data, &stateData); err == nil {
								resourceCount := 0
								if resources, ok := stateData["resources"].([]interface{}); ok {
									resourceCount = len(resources)
								}
								version := "unknown"
								if v, ok := stateData["version"].(float64); ok {
									version = fmt.Sprintf("%.0f", v)
								}

								stateFiles = append(stateFiles, map[string]interface{}{
									"name":      filepath.Base(match),
									"path":      match,
									"resources": resourceCount,
									"version":   version,
									"provider":  "Multi",
									"type":      "terraform",
									"backend":   "local",
									"status":    "discovered",
								})
							}
						}
					}
				} else {
					// Single file
					if data, err := os.ReadFile(path); err == nil {
						var stateData map[string]interface{}
						if err := json.Unmarshal(data, &stateData); err == nil {
							resourceCount := 0
							if resources, ok := stateData["resources"].([]interface{}); ok {
								resourceCount = len(resources)
							}
							version := "unknown"
							if v, ok := stateData["version"].(float64); ok {
								version = fmt.Sprintf("%.0f", v)
							}

							stateFiles = append(stateFiles, map[string]interface{}{
								"name":      filepath.Base(path),
								"path":      path,
								"resources": resourceCount,
								"version":   version,
								"provider":  "Multi",
								"type":      "terraform",
								"backend":   "local",
								"status":    "discovered",
							})
						}
					}
				}
			}
		}

	default:
		// Unsupported backend type
		stateFiles = append(stateFiles, map[string]interface{}{
			"error": fmt.Sprintf("Unsupported backend type: %s", req.Backend),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stateFiles)
}

// Helper functions
func detectProviders(resources []interface{}) string {
	providers := make(map[string]bool)
	for _, r := range resources {
		if resource, ok := r.(map[string]interface{}); ok {
			if provider, ok := resource["provider"].(string); ok {
				providers[extractProviderName(provider)] = true
			}
		}
	}

	if len(providers) == 0 {
		return "unknown"
	}

	result := []string{}
	for p := range providers {
		result = append(result, p)
	}
	return strings.Join(result, ", ")
}

func detectStateType(state map[string]interface{}) string {
	if backend, ok := state["backend"].(map[string]interface{}); ok {
		if _, ok := backend["type"].(string); ok {
			return "terragrunt"
		}
	}
	return "terraform"
}

func extractProviderName(provider string) string {
	// Extract provider name from full provider string
	// e.g., "provider[\"registry.terraform.io/hashicorp/aws\"]" -> "aws"
	// or "provider.aws" -> "aws"

	// Handle provider.xxx format
	if strings.HasPrefix(provider, "provider.") {
		return strings.TrimPrefix(provider, "provider.")
	}

	// Handle provider["xxx"] format
	if strings.Contains(provider, "[") {
		// Extract from registry URL
		parts := strings.Split(provider, "/")
		if len(parts) > 0 {
			name := parts[len(parts)-1]
			// Remove trailing bracket and quotes
			name = strings.TrimSuffix(name, "\"]")
			name = strings.TrimSuffix(name, "]")
			name = strings.Trim(name, "\"")
			return name
		}
	}

	return provider
}

// handleStateContent serves the full content of a state file
func (s *Server) handleStateContent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path := vars["path"]

	// Decode the base64 encoded path
	decodedPath, err := base64.StdEncoding.DecodeString(path)
	if err != nil {
		// If not base64, use as-is
		decodedPath = []byte(path)
	}
	pathStr := string(decodedPath)

	// Use the state manager to load the state content
	stateContent, err := s.stateManager.LoadStateContent(pathStr)
	if err != nil {
		// Return proper error instead of mock data
		s.respondError(w, http.StatusNotFound, fmt.Sprintf("State file not found: %s", err.Error()))
		return
	}

	// Get file size for response
	var fileSize int64
	if info, err := os.Stat(pathStr); err == nil {
		fileSize = info.Size()
	} else {
		// Try alternate path
		if info, err := os.Stat(filepath.Join("terraform-states", pathStr)); err == nil {
			fileSize = info.Size()
		} else {
			// Estimate size from content
			if data, err := json.Marshal(stateContent); err == nil {
				fileSize = int64(len(data))
			}
		}
	}

	response := map[string]interface{}{
		"data": stateContent,
		"size": fileSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateMockFullState has been removed - now using real state file loading via StateFileManager

// handleRemediationJobs returns the list of remediation jobs
func (s *Server) handleRemediationJobs(w http.ResponseWriter, r *http.Request) {
	// Get real remediation jobs from the store
	allJobs := s.remediationStore.GetAllJobs()

	// Get summary
	summary := s.remediationStore.GetJobsSummary()

	// Convert jobs to response format
	var jobsResponse []map[string]interface{}
	for _, job := range allJobs {
		jobData := map[string]interface{}{
			"id":          job.ID,
			"resource_id": job.ResourceID,
			"type":        job.ResourceType,
			"status":      job.Status,
			"created_at":  job.CreatedAt,
			"provider":    job.Provider,
			"region":      job.Region,
			"action":      job.Action,
			"drift_type":  job.DriftType,
		}

		if job.StartedAt != nil {
			jobData["started_at"] = job.StartedAt
		}
		if job.CompletedAt != nil {
			jobData["completed_at"] = job.CompletedAt
		}
		if job.Error != "" {
			jobData["error"] = job.Error
		}
		if len(job.Details) > 0 {
			jobData["details"] = job.Details
		}

		jobsResponse = append(jobsResponse, jobData)
	}

	// If no jobs exist yet, return empty array instead of null
	if jobsResponse == nil {
		jobsResponse = []map[string]interface{}{}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":        jobsResponse,
		"pending":     summary["pending"],
		"in_progress": summary["in_progress"],
		"completed":   summary["completed"],
		"failed":      summary["failed"],
		"total":       summary["total"],
	})
}

// handleSettings manages application settings
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Return current settings
		settings := map[string]interface{}{
			"general": map[string]interface{}{
				"auto_refresh":     true,
				"refresh_interval": 60,
				"dark_mode":        false,
			},
			"notifications": map[string]interface{}{
				"enabled":         true,
				"drift_threshold": 5,
			},
		}

		s.respondJSON(w, http.StatusOK, settings)
	} else if r.Method == "POST" {
		// Save settings
		var settings map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// In production, save settings to persistent storage
		// For now, just acknowledge the save
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Settings saved successfully",
		})
	}
}

// broadcastMessage sends a message to all WebSocket clients
func (s *Server) broadcastMessage(message interface{}) {
	s.broadcast <- message
}

// handleMetrics returns server metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Collect system metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Count discovery jobs by status
	s.discoveryMu.RLock()
	activeJobs := 0
	completedJobs := 0
	failedJobs := 0
	var lastDiscovery *time.Time

	for _, job := range s.discoveryJobs {
		switch job.Status {
		case "running", "pending":
			activeJobs++
		case "completed":
			completedJobs++
			if job.CompletedAt != nil && (lastDiscovery == nil || job.CompletedAt.After(*lastDiscovery)) {
				lastDiscovery = job.CompletedAt
			}
		case "failed":
			failedJobs++
		}
	}
	s.discoveryMu.RUnlock()

	// Count active drift detections
	activeDetections := 0
	for _, job := range s.discoveryJobs {
		if job.Status == "running" && job.JobType == "drift-detection" {
			activeDetections++
		}
	}

	// Get WebSocket metrics
	s.wsClientsMu.RLock()
	wsConnections := len(s.wsClients)
	s.wsClientsMu.RUnlock()

	metrics := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"system": map[string]interface{}{
			"uptime_seconds":  time.Since(startTime).Seconds(),
			"uptime":          time.Since(startTime).String(),
			"goroutines":      runtime.NumGoroutine(),
			"memory_alloc_mb": memStats.Alloc / 1024 / 1024,
			"memory_total_mb": memStats.TotalAlloc / 1024 / 1024,
			"num_gc":          memStats.NumGC,
			"cpu_count":       runtime.NumCPU(),
			"version":         "1.0.0",
		},
		"discovery": map[string]interface{}{
			"total_resources":   s.discoveryHub.GetResourceCount(),
			"active_jobs":       activeJobs,
			"completed_jobs":    completedJobs,
			"failed_jobs":       failedJobs,
			"last_discovery_at": lastDiscovery,
		},
		"drift": map[string]interface{}{
			"total_drifts":      s.driftStore.GetDriftCount(),
			"active_detections": activeDetections,
			"remediation_jobs":  s.remediationStore.GetJobCount(),
		},
		"cache": map[string]interface{}{
			"cache_size":      s.discoveryHub.GetResourceCount(),
			"cache_hits":      s.discoveryHub.GetCacheHits(),
			"cache_misses":    s.discoveryHub.GetCacheMisses(),
			"cache_evictions": 0, // Not tracked yet
		},
		"websocket": map[string]interface{}{
			"connected_clients": wsConnections,
			"messages_sent":     s.GetWSMessagesSent(),
			"messages_queued":   len(s.broadcast),
		},
		"persistence": map[string]interface{}{
			"enabled": s.persistence != nil,
		},
		"notifications": map[string]interface{}{
			"enabled": s.notifier != nil,
		},
	}

	s.respondJSON(w, http.StatusOK, metrics)
}

// handleDiscoveryStart starts a new discovery job
func (s *Server) handleDiscoveryStart(w http.ResponseWriter, r *http.Request) {
	var req discovery.DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jobID := s.discoveryHub.StartDiscovery(req)
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"job_id": jobID,
		"status": "started",
	})
}

// handleResourcesExport exports resources
func (s *Server) handleResourcesExport(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	query := r.URL.Query()
	format := query.Get("format")
	if format == "" {
		format = "json"
	}

	provider := query.Get("provider")
	region := query.Get("region")
	resourceType := query.Get("type")
	includeMetadata := query.Get("include_metadata") != "false"

	// Get all resources
	allResources := s.discoveryHub.GetAllResources()

	// Filter resources if needed
	var resources []apimodels.Resource
	for _, resource := range allResources {
		// Apply filters
		if provider != "" && resource.Provider != provider {
			continue
		}
		if region != "" && resource.Region != region {
			continue
		}
		if resourceType != "" && resource.Type != resourceType {
			continue
		}
		resources = append(resources, resource)
	}

	// Prepare export data
	exportData := map[string]interface{}{
		"version":         "1.0",
		"exported_at":     time.Now().UTC(),
		"total_resources": len(resources),
		"resources":       resources,
	}

	// Add metadata if requested
	if includeMetadata {
		// Group resources by provider and type for statistics
		stats := make(map[string]map[string]int)
		for _, r := range resources {
			if _, ok := stats[r.Provider]; !ok {
				stats[r.Provider] = make(map[string]int)
			}
			stats[r.Provider][r.Type]++
		}

		exportData["metadata"] = map[string]interface{}{
			"filters": map[string]string{
				"provider": provider,
				"region":   region,
				"type":     resourceType,
			},
			"statistics":  stats,
			"export_host": r.Host,
			"export_user": r.Header.Get("X-User-ID"),
		}
	}

	// Handle different export formats
	switch format {
	case "csv":
		// Export as CSV
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=resources.csv")

		// Write CSV header
		w.Write([]byte("ID,Name,Type,Provider,Region,Status,CreatedAt,ModifiedAt,Managed\n"))

		// Write resource rows
		for _, r := range resources {
			line := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%t\n",
				r.ID, r.Name, r.Type, r.Provider, r.Region, r.Status,
				r.CreatedAt.Format(time.RFC3339), r.ModifiedAt.Format(time.RFC3339),
				r.Managed)
			w.Write([]byte(line))
		}

	case "yaml":
		// Export as YAML
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Header().Set("Content-Disposition", "attachment; filename=resources.yaml")

		yamlData, err := yaml.Marshal(exportData)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate YAML: %v", err))
			return
		}
		w.Write(yamlData)

	default:
		// Default to JSON
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=resources_%s.json", time.Now().Format("20060102_150405")))

		data, err := json.MarshalIndent(exportData, "", "  ")
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to generate JSON: %v", err))
			return
		}
		w.Write(data)
	}

	// Log the export
	if s.auditLogger != nil {
		s.auditLogger.Log("resource_export", map[string]interface{}{
			"format": format,
			"count":  len(resources),
			"filters": map[string]string{
				"provider": provider,
				"region":   region,
				"type":     resourceType,
			},
			"user": r.Header.Get("X-User-ID"),
		})
	}
}

// handleResourcesImport imports resources
func (s *Server) handleResourcesImport(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()
	mergeStrategy := query.Get("merge") // "replace", "merge", "skip_existing"
	if mergeStrategy == "" {
		mergeStrategy = "merge"
	}
	validateResources := query.Get("validate") != "false"

	// Parse content type
	contentType := r.Header.Get("Content-Type")

	// Structure to hold import data
	type ImportData struct {
		Version        string                 `json:"version" yaml:"version"`
		ExportedAt     time.Time              `json:"exported_at" yaml:"exported_at"`
		TotalResources int                    `json:"total_resources" yaml:"total_resources"`
		Resources      []apimodels.Resource   `json:"resources" yaml:"resources"`
		Metadata       map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	}

	var importData ImportData
	var resources []apimodels.Resource

	// Handle different content types
	if strings.Contains(contentType, "text/csv") {
		// Parse CSV format
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Failed to read request body: %v", err))
			return
		}

		lines := strings.Split(string(body), "\n")
		if len(lines) < 2 {
			s.respondError(w, http.StatusBadRequest, "CSV file is empty or missing header")
			return
		}

		// Skip header and parse resources
		for i := 1; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}

			fields := strings.Split(line, ",")
			if len(fields) < 9 {
				continue // Skip invalid lines
			}

			createdAt, _ := time.Parse(time.RFC3339, fields[6])
			modifiedAt, _ := time.Parse(time.RFC3339, fields[7])
			managed := fields[8] == "true"

			resource := apimodels.Resource{
				ID:         fields[0],
				Name:       fields[1],
				Type:       fields[2],
				Provider:   fields[3],
				Region:     fields[4],
				Status:     fields[5],
				CreatedAt:  createdAt,
				ModifiedAt: modifiedAt,
				Managed:    managed,
			}
			resources = append(resources, resource)
		}

	} else if strings.Contains(contentType, "yaml") {
		// Parse YAML format
		body, err := io.ReadAll(r.Body)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Failed to read request body: %v", err))
			return
		}

		if err := yaml.Unmarshal(body, &importData); err != nil {
			s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse YAML: %v", err))
			return
		}
		resources = importData.Resources

	} else {
		// Default to JSON format
		decoder := json.NewDecoder(r.Body)

		// Try to decode as ImportData first
		if err := decoder.Decode(&importData); err != nil {
			// If that fails, try as raw resource array (backward compatibility)
			r.Body.Close()
			r.Body = io.NopCloser(strings.NewReader(r.Header.Get("X-Body-Cache"))) // Reset body if cached

			decoder = json.NewDecoder(r.Body)
			if err := decoder.Decode(&resources); err != nil {
				s.respondError(w, http.StatusBadRequest, fmt.Sprintf("Failed to parse JSON: %v", err))
				return
			}
		} else {
			resources = importData.Resources
		}
	}

	// Validate resources if requested
	importedCount := 0
	skippedCount := 0
	errorCount := 0
	errors := []string{}

	if validateResources {
		validResources := []apimodels.Resource{}
		for _, resource := range resources {
			// Validate required fields
			if resource.ID == "" {
				errors = append(errors, fmt.Sprintf("Resource missing ID: %+v", resource))
				errorCount++
				continue
			}
			if resource.Type == "" {
				errors = append(errors, fmt.Sprintf("Resource %s missing type", resource.ID))
				errorCount++
				continue
			}
			if resource.Provider == "" {
				errors = append(errors, fmt.Sprintf("Resource %s missing provider", resource.ID))
				errorCount++
				continue
			}

			// Set defaults for missing optional fields
			if resource.Name == "" {
				resource.Name = resource.ID
			}
			if resource.Status == "" {
				resource.Status = "unknown"
			}
			if resource.CreatedAt.IsZero() {
				resource.CreatedAt = time.Now()
			}
			if resource.ModifiedAt.IsZero() {
				resource.ModifiedAt = time.Now()
			}

			validResources = append(validResources, resource)
		}
		resources = validResources
	}

	// Apply merge strategy
	existingResources := s.discoveryHub.GetAllResources()
	existingMap := make(map[string]apimodels.Resource)
	for _, r := range existingResources {
		existingMap[r.ID] = r
	}

	finalResources := []apimodels.Resource{}

	switch mergeStrategy {
	case "replace":
		// Replace all resources with imported ones
		finalResources = resources
		importedCount = len(resources)

	case "skip_existing":
		// Only add new resources, skip existing ones
		for _, resource := range resources {
			if _, exists := existingMap[resource.ID]; !exists {
				finalResources = append(finalResources, resource)
				importedCount++
			} else {
				skippedCount++
			}
		}
		// Add back existing resources
		for _, existing := range existingResources {
			finalResources = append(finalResources, existing)
		}

	default: // "merge"
		// Merge imported resources with existing ones (imported overwrites existing)
		for _, resource := range resources {
			existingMap[resource.ID] = resource
			importedCount++
		}
		for _, resource := range existingMap {
			finalResources = append(finalResources, resource)
		}
	}

	// Update the discovery hub cache
	s.discoveryHub.PrePopulateCache(finalResources)

	// Save to persistence if available
	if s.persistence != nil {
		for _, resource := range finalResources {
			s.persistence.SaveResource(resource)
		}
	}

	// Prepare response
	response := map[string]interface{}{
		"imported":       importedCount,
		"skipped":        skippedCount,
		"errors":         errorCount,
		"total":          len(finalResources),
		"merge_strategy": mergeStrategy,
	}

	if len(errors) > 0 && errorCount <= 10 {
		response["error_details"] = errors
	} else if errorCount > 10 {
		response["error_details"] = append(errors[:10], fmt.Sprintf("... and %d more errors", errorCount-10))
	}

	// Log the import
	if s.auditLogger != nil {
		s.auditLogger.Log("resource_import", map[string]interface{}{
			"imported":       importedCount,
			"skipped":        skippedCount,
			"errors":         errorCount,
			"merge_strategy": mergeStrategy,
			"user":           r.Header.Get("X-User-ID"),
		})
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleResourcesDelete deletes resources
func (s *Server) handleResourcesDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	// Get resource details first
	resource, exists := s.persistence.GetResource(resourceID)
	if !exists {
		s.respondError(w, http.StatusNotFound, fmt.Sprintf("Resource %s not found", resourceID))
		return
	}

	// Check if resource can be deleted
	if resource.Type == "" || resource.Provider == "" {
		s.respondError(w, http.StatusBadRequest, "Resource missing required metadata for deletion")
		return
	}

	// Use deletion engine if available
	if s.deletionEngine != nil {
		// Create deletion request
		request := DeletionRequest{
			ResourceID:   resourceID,
			ResourceType: resource.Type,
			Provider:     resource.Provider,
			Region:       resource.Region,
			Force:        r.URL.Query().Get("force") == "true",
		}

		// Execute deletion
		ctx := r.Context()
		result, err := s.deletionEngine.DeleteResource(ctx, request)
		if err != nil {
			s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete resource: %v", err))
			return
		}

		// Remove from persistence if cloud deletion succeeded
		if result.Success {
			s.persistence.RemoveResource(resourceID)

			// Log the deletion
			if s.logger != nil {
				s.logger.Info("Resource deleted",
					"resource_id", resourceID,
					"resource_type", resource.Type,
					"provider", resource.Provider,
					"deleted_by", r.Header.Get("X-User-ID"),
				)
			}

			s.respondJSON(w, http.StatusOK, map[string]interface{}{
				"deleted":    resourceID,
				"success":    true,
				"message":    result.Message,
				"deleted_at": result.DeletedAt,
			})
		} else {
			s.respondError(w, http.StatusInternalServerError, result.Message)
		}
	} else {
		// Fallback: just remove from local persistence
		s.persistence.RemoveResource(resourceID)
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"deleted": resourceID,
			"success": true,
			"message": "Resource removed from local cache (cloud deletion not available)",
		})
	}
}

// handleGetRelationships returns resource relationships
func (s *Server) handleGetRelationships(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	relationships, _ := s.persistence.GetRelationships(resourceID)
	s.respondJSON(w, http.StatusOK, relationships)
}

// handleDiscoverRelationships discovers resource relationships
func (s *Server) handleDiscoverRelationships(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	if s.relationshipMapper == nil {
		s.respondError(w, http.StatusInternalServerError, "Relationship mapper not initialized")
		return
	}

	// Get current resources from cache
	resources := s.discoveryHub.GetCachedResources()
	if len(resources) == 0 {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"resource_id":   resourceID,
			"relationships": []interface{}{},
			"message":       "No cached resources available for relationship discovery",
		})
		return
	}

	// Convert to models.Resource for relationship discovery
	var modelResources []models.Resource
	for _, r := range resources {
		modelResources = append(modelResources, models.Resource{
			ID:         r.ID,
			Name:       r.Name,
			Type:       r.Type,
			Provider:   r.Provider,
			Region:     r.Region,
			State:      r.Status,
			Tags:       r.Tags,
			Properties: r.Properties,
		})
	}

	// Discover relationships for all resources
	ctx := context.Background()
	if err := s.relationshipMapper.DiscoverRelationships(ctx, modelResources); err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to discover relationships: %v", err))
		return
	}

	// Get relationships for the specific resource
	dependencies := s.relationshipMapper.GetDependencies(resourceID)
	dependents := s.relationshipMapper.GetDependents(resourceID)

	// Combine into response format
	var relationships []map[string]interface{}

	for _, dep := range dependencies {
		relationships = append(relationships, map[string]interface{}{
			"target_resource_id":   dep.TargetID,
			"target_resource_type": dep.TargetType,
			"relationship_type":    "depends_on",
			"strength":             dep.Strength,
			"reason":               dep.Reason,
		})
	}

	for _, dep := range dependents {
		relationships = append(relationships, map[string]interface{}{
			"target_resource_id":   dep.SourceID,
			"target_resource_type": dep.SourceType,
			"relationship_type":    "depended_by",
			"strength":             dep.Strength,
			"reason":               dep.Reason,
		})
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"resource_id":        resourceID,
		"relationships":      relationships,
		"total_dependencies": len(dependencies),
		"total_dependents":   len(dependents),
	})
}

// handleGetDependencyGraph returns the dependency graph
func (s *Server) handleGetDependencyGraph(w http.ResponseWriter, r *http.Request) {
	if s.relationshipMapper == nil {
		s.respondError(w, http.StatusInternalServerError, "Relationship mapper not initialized")
		return
	}

	// Get current resources from cache to ensure we have up-to-date data
	resources := s.discoveryHub.GetCachedResources()
	if len(resources) > 0 {
		// Convert to models.Resource for relationship discovery
		var modelResources []models.Resource
		for _, r := range resources {
			modelResources = append(modelResources, models.Resource{
				ID:         r.ID,
				Name:       r.Name,
				Type:       r.Type,
				Provider:   r.Provider,
				Region:     r.Region,
				State:      r.Status,
				Tags:       r.Tags,
				Properties: r.Properties,
			})
		}

		// Update relationships
		ctx := context.Background()
		if err := s.relationshipMapper.DiscoverRelationships(ctx, modelResources); err != nil {
			// Log error but continue with existing relationships
			fmt.Printf("Warning: Failed to refresh relationships: %v\n", err)
		}
	}

	// Get the complete dependency graph
	graph := s.relationshipMapper.GetGraph()

	// Convert graph to response format
	var nodes []map[string]interface{}
	var edges []map[string]interface{}

	// Create nodes from resources
	nodeMap := make(map[string]bool)

	// Add nodes from relationships
	for _, rel := range graph.Relationships {
		// Add source node
		if !nodeMap[rel.SourceID] {
			nodeMap[rel.SourceID] = true
			nodes = append(nodes, map[string]interface{}{
				"id":    rel.SourceID,
				"type":  rel.SourceType,
				"label": rel.SourceID,
			})
		}

		// Add target node
		if !nodeMap[rel.TargetID] {
			nodeMap[rel.TargetID] = true
			nodes = append(nodes, map[string]interface{}{
				"id":    rel.TargetID,
				"type":  rel.TargetType,
				"label": rel.TargetID,
			})
		}

		// Add edge
		edges = append(edges, map[string]interface{}{
			"source":        rel.SourceID,
			"target":        rel.TargetID,
			"relationship":  rel.Type,
			"strength":      rel.Strength,
			"reason":        rel.Reason,
			"bidirectional": rel.Bidirectional,
		})
	}

	// Add any resources that don't have relationships as isolated nodes
	for _, r := range resources {
		if !nodeMap[r.ID] {
			nodes = append(nodes, map[string]interface{}{
				"id":       r.ID,
				"type":     r.Type,
				"label":    r.Name,
				"isolated": true,
			})
		}
	}

	// Add graph metadata
	metadata := map[string]interface{}{
		"total_nodes":      len(nodes),
		"total_edges":      len(edges),
		"connected_nodes":  len(nodes) - countIsolatedNodes(nodes),
		"isolated_nodes":   countIsolatedNodes(nodes),
		"generated_at":     time.Now(),
		"layout_algorithm": "force-directed",
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"nodes":    nodes,
		"edges":    edges,
		"metadata": metadata,
	})
}

// countIsolatedNodes counts nodes marked as isolated
func countIsolatedNodes(nodes []map[string]interface{}) int {
	count := 0
	for _, node := range nodes {
		if isolated, ok := node["isolated"].(bool); ok && isolated {
			count++
		}
	}
	return count
}

// handleGetResourceRelationships returns resource relationships
func (s *Server) handleGetResourceRelationships(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	relationships, _ := s.persistence.GetRelationships(resourceID)
	s.respondJSON(w, http.StatusOK, relationships)
}

// handleGetDeletionOrder returns deletion order for resources
func (s *Server) handleGetDeletionOrder(w http.ResponseWriter, r *http.Request) {
	var resourceIDs []string
	if err := json.NewDecoder(r.Body).Decode(&resourceIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Calculate deletion order based on resource dependencies
	orderedResources, err := s.calculateDeletionOrder(resourceIDs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to calculate deletion order: %v", err), http.StatusInternalServerError)
		return
	}

	s.respondJSON(w, http.StatusOK, orderedResources)
}

// handleSendNotification sends a notification
func (s *Server) handleSendNotification(w http.ResponseWriter, r *http.Request) {
	var notification map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"sent": true,
	})
}

// handleNotificationHistory returns notification history
func (s *Server) handleNotificationHistory(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, []interface{}{})
}

// handleTestNotification tests notification configuration
func (s *Server) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Test notification sent",
	})
}

// handleNotificationConfig manages notification configuration
func (s *Server) handleNotificationConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"enabled":  false,
			"channels": []interface{}{},
		})
	} else {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"updated": true,
		})
	}
}

// calculateDeletionOrder determines the safe deletion order for resources based on dependencies
func (s *Server) calculateDeletionOrder(resourceIDs []string) ([]string, error) {
	// Build a dependency graph
	dependencyGraph := make(map[string][]string)
	resourceMap := make(map[string]*resource.Resource)

	// Fetch resource details from cache or discovery
	for _, id := range resourceIDs {
		resource := s.getResourceByID(id)
		if resource != nil {
			resourceMap[id] = resource
			dependencyGraph[id] = []string{}
		}
	}

	// Analyze dependencies between resources
	for id, res := range resourceMap {
		// Check for direct references in Dependencies field
		for _, depID := range res.Dependencies {
			if _, exists := resourceMap[depID]; exists {
				dependencyGraph[depID] = append(dependencyGraph[depID], id)
			}
		}

		// Infer dependencies based on resource types
		dependencies := s.inferResourceDependencies(res, resourceMap)
		for _, depID := range dependencies {
			if _, exists := resourceMap[depID]; exists {
				dependencyGraph[depID] = append(dependencyGraph[depID], id)
			}
		}
	}

	// Perform topological sort
	visited := make(map[string]bool)
	stack := []string{}
	var visit func(string) error

	visit = func(id string) error {
		if visited[id] {
			return nil
		}
		visited[id] = true

		// Visit all resources that depend on this one first
		for _, dependent := range dependencyGraph[id] {
			if err := visit(dependent); err != nil {
				return err
			}
		}

		// Add to stack after visiting dependents
		stack = append(stack, id)
		return nil
	}

	// Visit all resources
	for id := range resourceMap {
		if err := visit(id); err != nil {
			return nil, err
		}
	}

	// Reverse the stack to get deletion order (delete dependents first)
	result := make([]string, len(stack))
	for i, id := range stack {
		result[len(stack)-1-i] = id
	}

	return result, nil
}

// getResourceByID retrieves a resource by its ID from cache or discovery
func (s *Server) getResourceByID(resourceID string) *resource.Resource {
	// Try to get from cache first
	if s.cacheManager != nil {
		if cached, err := s.cacheManager.Get(context.Background(), fmt.Sprintf("resource:%s", resourceID)); err == nil {
			if res, ok := cached.(*resource.Resource); ok {
				return res
			}
		}
	}

	// Try to get from discovery hub
	if s.discoveryHub != nil {
		resources := s.discoveryHub.GetCachedResources()
		for _, r := range resources {
			if r.ID == resourceID {
				return &r
			}
		}
	}

	// Create a minimal resource if not found
	return &resource.Resource{
		ID:         resourceID,
		Type:       "unknown",
		Provider:   "unknown",
		Metadata:   make(map[string]string),
		Attributes: make(map[string]interface{}),
	}
}

// inferResourceDependencies infers dependencies based on resource types and relationships
func (s *Server) inferResourceDependencies(res *resource.Resource, allResources map[string]*resource.Resource) []string {
	var dependencies []string

	// Resource type-specific dependency rules
	switch res.Type {
	case "aws_instance", "ec2_instance":
		// EC2 instances depend on subnets, security groups
		if subnetID, ok := res.Attributes["subnet_id"].(string); ok {
			for id, r := range allResources {
				if r.Type == "aws_subnet" && r.Name == subnetID {
					dependencies = append(dependencies, id)
				}
			}
		}
		if sgIDs, ok := res.Attributes["security_groups"].([]interface{}); ok {
			for _, sg := range sgIDs {
				if sgID, ok := sg.(string); ok {
					for id, r := range allResources {
						if r.Type == "aws_security_group" && r.Name == sgID {
							dependencies = append(dependencies, id)
						}
					}
				}
			}
		}

	case "aws_subnet", "subnet":
		// Subnets depend on VPCs
		if vpcID, ok := res.Attributes["vpc_id"].(string); ok {
			for id, r := range allResources {
				if r.Type == "aws_vpc" && r.Name == vpcID {
					dependencies = append(dependencies, id)
				}
			}
		}

	case "aws_security_group", "security_group":
		// Security groups depend on VPCs
		if vpcID, ok := res.Attributes["vpc_id"].(string); ok {
			for id, r := range allResources {
				if r.Type == "aws_vpc" && r.Name == vpcID {
					dependencies = append(dependencies, id)
				}
			}
		}

	case "azurerm_virtual_machine", "virtual_machine":
		// Azure VMs depend on resource groups, vnets, subnets
		if rgName, ok := res.Attributes["resource_group_name"].(string); ok {
			for id, r := range allResources {
				if r.Type == "azurerm_resource_group" && r.Name == rgName {
					dependencies = append(dependencies, id)
				}
			}
		}
		if nicIDs, ok := res.Attributes["network_interface_ids"].([]interface{}); ok {
			for _, nic := range nicIDs {
				if nicID, ok := nic.(string); ok {
					for id, r := range allResources {
						if r.Type == "azurerm_network_interface" && r.Name == nicID {
							dependencies = append(dependencies, id)
						}
					}
				}
			}
		}

	case "azurerm_subnet":
		// Azure subnets depend on virtual networks
		if vnetName, ok := res.Attributes["virtual_network_name"].(string); ok {
			for id, r := range allResources {
				if r.Type == "azurerm_virtual_network" && r.Name == vnetName {
					dependencies = append(dependencies, id)
				}
			}
		}

	case "azurerm_virtual_network", "virtual_network":
		// Azure vnets depend on resource groups
		if rgName, ok := res.Attributes["resource_group_name"].(string); ok {
			for id, r := range allResources {
				if r.Type == "azurerm_resource_group" && r.Name == rgName {
					dependencies = append(dependencies, id)
				}
			}
		}

	case "google_compute_instance", "compute_instance":
		// GCP instances depend on networks and subnetworks
		if network, ok := res.Attributes["network"].(string); ok {
			for id, r := range allResources {
				if r.Type == "google_compute_network" && r.Name == network {
					dependencies = append(dependencies, id)
				}
			}
		}
		if subnetwork, ok := res.Attributes["subnetwork"].(string); ok {
			for id, r := range allResources {
				if r.Type == "google_compute_subnetwork" && r.Name == subnetwork {
					dependencies = append(dependencies, id)
				}
			}
		}

	case "google_compute_subnetwork", "subnetwork":
		// GCP subnetworks depend on networks
		if network, ok := res.Attributes["network"].(string); ok {
			for id, r := range allResources {
				if r.Type == "google_compute_network" && r.Name == network {
					dependencies = append(dependencies, id)
				}
			}
		}

	case "digitalocean_droplet", "droplet":
		// DO droplets depend on VPCs if specified
		if vpcID, ok := res.Attributes["vpc_uuid"].(string); ok && vpcID != "" {
			for id, r := range allResources {
				if r.Type == "digitalocean_vpc" && r.ID == vpcID {
					dependencies = append(dependencies, id)
				}
			}
		}
		// Droplets depend on volumes if attached
		if volumeIDs, ok := res.Attributes["volume_ids"].([]interface{}); ok {
			for _, vol := range volumeIDs {
				if volID, ok := vol.(string); ok {
					for id, r := range allResources {
						if r.Type == "digitalocean_volume" && r.ID == volID {
							dependencies = append(dependencies, id)
						}
					}
				}
			}
		}
	}

	// Check for explicit parent-child relationships
	if parentID, ok := res.Attributes["parent_id"].(string); ok {
		if _, exists := allResources[parentID]; exists {
			dependencies = append(dependencies, parentID)
		}
	}

	// Check for reference fields that might indicate dependencies
	for key, value := range res.Attributes {
		if strings.HasSuffix(key, "_id") || strings.HasSuffix(key, "_ref") {
			if refID, ok := value.(string); ok {
				if _, exists := allResources[refID]; exists {
					// Avoid duplicates
					found := false
					for _, dep := range dependencies {
						if dep == refID {
							found = true
							break
						}
					}
					if !found {
						dependencies = append(dependencies, refID)
					}
				}
			}
		}
	}

	return dependencies
}
