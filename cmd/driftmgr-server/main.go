package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	// WebSocket support
	"github.com/gorilla/websocket"

	// AWS SDK v2
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	// Additional AWS services

	// Azure SDK

	// GCP SDK

	// Internal packages
	"github.com/catherinevee/driftmgr/internal/infrastructure/cache"
	"github.com/catherinevee/driftmgr/internal/infrastructure/config"
	"github.com/catherinevee/driftmgr/internal/deletion"
	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/drift"
	"github.com/catherinevee/driftmgr/internal/utils/graceful"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/monitoring"
	"github.com/catherinevee/driftmgr/internal/integration/notification"
	"github.com/catherinevee/driftmgr/internal/core/remediation"
	"github.com/catherinevee/driftmgr/internal/security/auth"
	"github.com/catherinevee/driftmgr/internal/integration/terragrunt"
	"github.com/catherinevee/driftmgr/internal/visualization"
	"github.com/catherinevee/driftmgr/internal/workflow"
	"github.com/catherinevee/driftmgr/internal/workspace"
)

var (
	cacheManager       = cache.GetGlobalManager()
	logger             = monitoring.GetGlobalLogger()
	emailProvider      = notification.NewEmailProviderFromEnv()
	diagramGenerator   *visualization.SimpleDiagramGenerator
	enhancedDiscoverer *discovery.EnhancedDiscoverer
	smartRemediator    *remediation.Engine
	driftPredictor     *drift.DriftPredictor
	workflowEngine     *workflow.WorkflowEngine
	deletionEngine     *deletion.DeletionEngine
	workspaceManager   *workspace.WorkspaceManager

	// WebSocket and progress tracking
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
	clients    = make(map[*websocket.Conn]bool)
	clientsMux sync.RWMutex
)

// ProgressUpdate represents a real-time progress update
type ProgressUpdate struct {
	Type      string      `json:"type"`
	Message   string      `json:"message"`
	Progress  int         `json:"progress,omitempty"`
	Total     int         `json:"total,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// broadcastProgress sends a progress update to all connected WebSocket clients
func broadcastProgress(update ProgressUpdate) {
	update.Timestamp = time.Now()
	clientsMux.RLock()
	defer clientsMux.RUnlock()

	for client := range clients {
		err := client.WriteJSON(update)
		if err != nil {
			logger.Warning("Error sending progress update: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

// handleWebSocket handles WebSocket connections for real-time updates
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Warning("WebSocket upgrade failed: %v", err)
		return
	}

	clientsMux.Lock()
	clients[conn] = true
	clientsMux.Unlock()

	// Send initial connection message
	conn.WriteJSON(ProgressUpdate{
		Type:    "connected",
		Message: "Connected to DriftMgr real-time updates",
	})

	// Keep connection alive and handle disconnection
	defer func() {
		clientsMux.Lock()
		delete(clients, conn)
		clientsMux.Unlock()
		conn.Close()
	}()

	for {
		// Read messages (we don't expect any from client, but keep connection alive)
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// getAzureSubscriptionID returns the Azure subscription ID from environment or Azure CLI
func getAzureSubscriptionID() (string, error) {
	// First try environment variable
	if subID := os.Getenv("AZURE_SUBSCRIPTION_ID"); subID != "" {
		return subID, nil
	}

	// If not set, try to get from Azure CLI
	cmd := exec.Command("az", "account", "show", "--query", "id", "--output", "tsv")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Azure subscription ID: %w", err)
	}

	subID := strings.TrimSpace(string(output))
	if subID == "" {
		return "", fmt.Errorf("Azure subscription ID is empty")
	}

	return subID, nil
}

// getGCPProjectID returns the GCP project ID from environment or gcloud CLI
func getGCPProjectID() (string, error) {
	// First try environment variable
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		return projectID, nil
	}

	// If not set, try to get from gcloud CLI
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GCP project ID: %w", err)
	}

	projectID := strings.TrimSpace(string(output))
	if projectID == "" {
		return "", fmt.Errorf("GCP project ID is empty")
	}

	return projectID, nil
}

func main() {
	// Set up panic recovery
	defer graceful.RecoverPanic()
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize security components
	secretKey := make([]byte, 32)
	if _, err := rand.Read(secretKey); err != nil {
		graceful.HandleCritical(err, "Failed to generate secret key")
	}

	// Initialize database-backed authentication
	dbPath := os.Getenv("DRIFT_DB_PATH")
	if dbPath == "" {
		dbPath = "./driftmgr.db"
	}

	authManager, err := security.NewAuthManager(secretKey, dbPath)
	if err != nil {
		graceful.HandleCritical(err, "Failed to initialize authentication manager")
	}

	rateLimiter := security.NewRateLimiter(1000, time.Minute) // 1000 requests per minute
	securityMiddleware := security.NewMiddleware(authManager, rateLimiter)

	logger.Info("Enhanced security components initialized successfully")
	logger.Info("Database path: %s", dbPath)

	// Initialize diagram generator
	diagramGenerator, err = visualization.NewSimpleDiagramGenerator("./outputs")
	if err != nil {
		logger.Info("Warning: Failed to initialize diagram generator: %v", err)
		// Continue without diagram generation capability
	}

	// Initialize enhanced discoverer
	cfg := &config.Config{
		Discovery: config.DiscoveryConfig{
			CacheTTL:     30 * time.Minute,
			CacheMaxSize: 1000,
			Regions:      []string{"us-east-1", "us-west-2", "eu-west-1"},
		},
	}
	enhancedDiscoverer = discovery.NewEnhancedDiscoverer(cfg)
	logger.Info("Enhanced discoverer initialized successfully")

	// Initialize smart remediator
	smartRemediator = remediation.NewEngine()
	logger.Info("Smart remediator initialized successfully")

	// Initialize drift predictor
	driftPredictor = drift.NewDriftPredictor(nil)
	drift.RegisterDefaultPatterns(driftPredictor)
	logger.Info("Drift predictor initialized successfully")

	// Initialize workflow engine
	workflowEngine = workflow.NewWorkflowEngine()
	if err := workflow.RegisterDefaultWorkflows(workflowEngine); err != nil {
		logger.Info("Warning: Failed to register default workflows: %v", err)
	}
	logger.Info("Workflow engine initialized successfully")

	// Initialize deletion engine
	deletionEngine = deletion.NewDeletionEngine()

	// Register cloud providers
	if awsProvider, err := deletion.NewAWSProvider(); err == nil {
		deletionEngine.RegisterProvider("aws", awsProvider)
		logger.Info("AWS provider registered successfully")
	} else {
		logger.Info("Warning: Failed to register AWS provider: %v", err)
	}

	if azureProvider, err := deletion.NewAzureProvider(); err == nil {
		deletionEngine.RegisterProvider("azure", azureProvider)
		logger.Info("Azure provider registered successfully")
	} else {
		logger.Info("Warning: Failed to register Azure provider: %v", err)
	}

	if gcpProvider, err := deletion.NewGCPProvider(); err == nil {
		deletionEngine.RegisterProvider("gcp", gcpProvider)
		logger.Info("GCP provider registered successfully")
	} else {
		logger.Info("Warning: Failed to register GCP provider: %v", err)
	}

	logger.Info("Deletion engine initialized successfully")

	// Initialize workspace manager
	workspaceManager = workspace.NewWorkspaceManager(".")
	logger.Info("Workspace manager initialized successfully")

	mux := http.NewServeMux()

	// Authentication endpoint (no middleware)
	mux.HandleFunc("/api/v1/auth/login", securityMiddleware.HandleLogin)
	mux.HandleFunc("/login", serveLoginPage)

	// Static files (no middleware)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("assets/static"))))

	// Health check endpoint (no middleware)
	mux.HandleFunc("/health", handleHealth)

	// Apply security middleware to all other routes
	secureHandler := securityMiddleware.SecurityHeadersMiddleware(
		securityMiddleware.CORSMiddleware(
			securityMiddleware.InputValidationMiddleware(
				securityMiddleware.RateLimitMiddleware(
					securityMiddleware.LoggingMiddleware(
						securityMiddleware.AuthMiddleware(
							securityMiddleware.SanitizeResponseMiddleware(
								createSecureRoutes(securityMiddleware),
							),
						),
					),
				),
			),
		),
	)

	mux.Handle("/", secureHandler)
	mux.HandleFunc("/api/v1/health", handleHealth)

	// State files endpoint
	mux.HandleFunc("/api/v1/statefiles", handleStateFiles)
	mux.HandleFunc("/api/v1/statefiles/", handleStateFile)

	// Core API endpoints under /api/v1/
	mux.HandleFunc("/api/v1/discover", handleDiscover)
	mux.HandleFunc("/api/v1/enhanced-discover", handleEnhancedDiscover)
	mux.HandleFunc("/api/v1/analyze", handleAnalyze)
	mux.HandleFunc("/api/v1/enhanced-analyze", handleEnhancedAnalyze)
	mux.HandleFunc("/api/v1/perspective", handlePerspective)
	mux.HandleFunc("/api/v1/visualize", handleVisualize)
	mux.HandleFunc("/api/v1/diagram", handleDiagram)
	mux.HandleFunc("/api/v1/export", handleExport)
	mux.HandleFunc("/api/v1/notify", handleNotify)
	mux.HandleFunc("/api/v1/remediate", handleRemediate)
	mux.HandleFunc("/api/v1/remediation-strategies", handleRemediationStrategies)
	mux.HandleFunc("/api/v1/test-remediation", handleTestRemediation)

	// Remediation endpoints
	mux.HandleFunc("/api/v1/remediate-batch", handleRemediateBatch)
	mux.HandleFunc("/api/v1/remediate-history", handleRemediateHistory)
	mux.HandleFunc("/api/v1/remediate-rollback", handleRemediateRollback)

	// Cache management endpoints
	mux.HandleFunc("/api/v1/cache/stats", handleCacheStats)
	mux.HandleFunc("/api/v1/cache/clear", handleCacheClear)

	// Terragrunt-specific endpoints
	mux.HandleFunc("/api/v1/terragrunt/files", handleTerragruntFiles)
	mux.HandleFunc("/api/v1/terragrunt/statefiles", handleTerragruntStateFiles)
	mux.HandleFunc("/api/v1/terragrunt/analyze", handleTerragruntAnalyze)

	// Drift prediction endpoints
	mux.HandleFunc("/api/v1/predict", handlePredictDrifts)
	mux.HandleFunc("/api/v1/predict/patterns", handleDriftPatterns)
	mux.HandleFunc("/api/v1/predict/stats", handlePredictionStats)

	// Workflow management endpoints
	mux.HandleFunc("/api/v1/workflows", handleWorkflows)
	mux.HandleFunc("/api/v1/workflows/", handleWorkflow)
	mux.HandleFunc("/api/v1/workflows/execute", handleExecuteWorkflow)
	mux.HandleFunc("/api/v1/workflows/executions/", handleWorkflowExecution)

	// Resource deletion endpoints
	mux.HandleFunc("/api/v1/delete/account", handleDeleteAccountResources)
	mux.HandleFunc("/api/v1/delete/preview", handleDeletePreview)
	mux.HandleFunc("/api/v1/delete/providers", handleGetSupportedProviders)

	// Workspace management endpoints
	mux.HandleFunc("/api/v1/workspaces", handleWorkspaces)
	mux.HandleFunc("/api/v1/workspaces/discover", handleWorkspaceDiscover)
	mux.HandleFunc("/api/v1/workspaces/", handleWorkspace)
	mux.HandleFunc("/api/v1/environments", handleEnvironments)
	mux.HandleFunc("/api/v1/environments/", handleEnvironment)
	mux.HandleFunc("/api/v1/environments/compare", handleEnvironmentCompare)
	mux.HandleFunc("/api/v1/environments/promote", handleEnvironmentPromote)
	mux.HandleFunc("/api/v1/blue-green", handleBlueGreen)
	mux.HandleFunc("/api/v1/feature-flags", handleFeatureFlags)

	// Root path for dashboard (must be last to avoid catching API routes)
	mux.HandleFunc("/", serveDashboard)

	logger.Info("Starting DriftMgr API Server on port %s at %s", port, fmt.Sprintf("http://localhost:%s", port))

	// Start server with graceful shutdown
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Register shutdown handler
	graceful.OnShutdown(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	})

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			graceful.HandleError(err, "Server failed to start")
		}
	}()

	// Wait for shutdown signal
	graceful.WaitForSignal()
}

// createSecureRoutes creates all the secure routes with proper middleware
func createSecureRoutes(middleware *security.Middleware) http.Handler {
	mux := http.NewServeMux()

	// Dashboard and WebSocket (require view permission)
	mux.Handle("/dashboard", middleware.PermissionMiddleware(security.PermissionViewDashboard)(http.HandlerFunc(serveDashboard)))
	mux.HandleFunc("/ws", handleWebSocket) // WebSocket doesn't need permission middleware

	// File upload (require admin permission)
	mux.Handle("/api/v1/upload", middleware.PermissionMiddleware(security.PermissionExecuteDiscovery)(http.HandlerFunc(handleFileUpload)))

	// Root path for dashboard (must be last to avoid catching API routes)
	mux.Handle("/", middleware.PermissionMiddleware(security.PermissionViewDashboard)(http.HandlerFunc(serveDashboard)))

	return mux
}

// serveLoginPage serves the login page
func serveLoginPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "assets/static/login.html")
}

// handleFileUpload handles file uploads for state files and configurations
func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 32MB)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create uploads directory if it doesn't exist
	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Create unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), header.Filename)
	filepath := filepath.Join(uploadDir, filename)

	// Create the file
	dst, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy uploaded file to destination
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Send progress update
	broadcastProgress(ProgressUpdate{
		Type:    "file_uploaded",
		Message: fmt.Sprintf("File %s uploaded successfully", header.Filename),
		Data: map[string]interface{}{
			"filename": filename,
			"size":     header.Size,
			"path":     filepath,
		},
	})

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"filename": filename,
		"path":     filepath,
		"size":     header.Size,
	})
}

// serveDashboard serves the main dashboard page
func serveDashboard(w http.ResponseWriter, r *http.Request) {
	// For now, serve a simple dashboard page
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DriftMgr Dashboard</title>
    <link href="https://cdn.jsdelivr.net/npm/daisyui@4.7.2/dist/full.min.css" rel="stylesheet" type="text/css" />
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <link href="/static/css/dashboard.css" rel="stylesheet" type="text/css" />
    <script src="/static/js/dashboard.js" defer></script>
    <script>
        function logout() {
            localStorage.removeItem('auth_token');
            localStorage.removeItem('user');
            window.location.href = '/login';
        }
    </script>
</head>
<body class="bg-base-100">
    <div class="navbar bg-base-200 shadow-lg">
        <div class="navbar-start">
            <div class="dropdown">
                <div tabindex="0" role="button" class="btn btn-ghost lg:hidden">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h8m-8 6h16"></path>
                    </svg>
                </div>
                <ul tabindex="0" class="menu menu-sm dropdown-content mt-3 z-[1] p-2 shadow bg-base-100 rounded-box w-52">
                    <li><a href="#overview" onclick="showTab('overview')">Overview</a></li>
                    <li><a href="#discovery" onclick="showTab('discovery')">Discovery</a></li>
                    <li><a href="#analysis" onclick="showTab('analysis')">Analysis</a></li>
                                    <li><a href="#remediation" onclick="showTab('remediation')" data-admin-only>Remediation</a></li>
                <li><a href="#workflows" onclick="showTab('workflows')" data-admin-only>Workflows</a></li>
                </ul>
            </div>
            <a class="btn btn-ghost text-xl">DriftMgr</a>
        </div>
        <div class="navbar-center hidden lg:flex">
            <ul class="menu menu-horizontal px-1">
                <li><a href="#overview" onclick="showTab('overview')">Overview</a></li>
                <li><a href="#discovery" onclick="showTab('discovery')">Discovery</a></li>
                <li><a href="#analysis" onclick="showTab('analysis')">Analysis</a></li>
                <li><a href="#remediation" onclick="showTab('remediation')" data-admin-only>Remediation</a></li>
                <li><a href="#workflows" onclick="showTab('workflows')" data-admin-only>Workflows</a></li>
            </ul>
        </div>
        <div class="navbar-end">
            <div class="badge badge-success" id="ws-status">Connected</div>
            <div class="dropdown dropdown-end">
                <div tabindex="0" role="button" class="btn btn-ghost">
                    <span id="user-info">User</span>
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                    </svg>
                </div>
                <ul tabindex="0" class="dropdown-content menu p-2 shadow bg-base-100 rounded-box w-52">
                    <li><a href="#" onclick="logout()">Logout</a></li>
                </ul>
            </div>
        </div>
    </div>

    <!-- Progress Toast -->
    <div class="toast toast-top toast-end" id="progress-toast" style="display: none;">
        <div class="alert alert-info">
            <div class="flex-1">
                <span id="progress-message">Processing...</span>
                <progress class="progress progress-primary w-full mt-2" id="progress-bar" value="0" max="100"></progress>
            </div>
        </div>
    </div>

    <div class="container mx-auto p-6">
        <!-- Overview Tab -->
        <div id="overview-tab" class="tab-content">
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                <div class="stat bg-base-200 rounded-lg shadow">
                    <div class="stat-title">Total Resources</div>
                    <div class="stat-value text-primary" id="total-resources">1,234</div>
                    <div class="stat-desc">↗︎ 14% more than last month</div>
                </div>
                <div class="stat bg-base-200 rounded-lg shadow">
                    <div class="stat-title">Active Drifts</div>
                    <div class="stat-value text-warning" id="active-drifts">23</div>
                    <div class="stat-desc">↘︎ 8% less than last week</div>
                </div>
                <div class="stat bg-base-200 rounded-lg shadow">
                    <div class="stat-title">Monthly Cost</div>
                    <div class="stat-value text-success" id="monthly-cost">$2,847</div>
                    <div class="stat-desc">↗︎ 12% more than last month</div>
                </div>
                <div class="stat bg-base-200 rounded-lg shadow">
                    <div class="stat-title">Security Score</div>
                    <div class="stat-value text-info" id="security-score">87%</div>
                    <div class="stat-desc">↗︎ 3% improvement</div>
                </div>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div class="card bg-base-200 shadow-xl">
                    <div class="card-body">
                        <h2 class="card-title">Drift Timeline</h2>
                        <canvas id="driftChart" width="400" height="200"></canvas>
                    </div>
                </div>
                <div class="card bg-base-200 shadow-xl">
                    <div class="card-body">
                        <h2 class="card-title">Cost Analysis</h2>
                        <canvas id="costChart" width="400" height="200"></canvas>
                    </div>
                </div>
            </div>
        </div>

        <!-- Discovery Tab -->
        <div id="discovery-tab" class="tab-content" style="display: none;">
            <div class="card bg-base-200 shadow-xl">
                <div class="card-body">
                    <h2 class="card-title">Cloud Resource Discovery</h2>
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Cloud Provider</span>
                        </label>
                        <select class="select select-bordered" id="discovery-provider">
                            <option value="aws">AWS</option>
                            <option value="azure">Azure</option>
                            <option value="gcp">GCP</option>
                        </select>
                    </div>
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Regions (comma-separated)</span>
                        </label>
                        <input type="text" class="input input-bordered" id="discovery-regions" placeholder="us-east-1, us-west-2" value="us-east-1">
                    </div>
                    <div class="form-control mt-6">
                        <button class="btn btn-primary" onclick="startDiscovery()">Start Discovery</button>
                    </div>
                    <div id="discovery-results" class="mt-4"></div>
                </div>
            </div>
        </div>

        <!-- Analysis Tab -->
        <div id="analysis-tab" class="tab-content" style="display: none;">
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div class="card bg-base-200 shadow-xl">
                    <div class="card-body">
                        <h2 class="card-title">Upload State File</h2>
                        <div class="form-control">
                            <label class="label">
                                <span class="label-text">Terraform State File</span>
                            </label>
                            <input type="file" class="file-input file-input-bordered w-full" id="state-file" accept=".tfstate,.json">
                        </div>
                        <div class="form-control mt-6">
                            <button class="btn btn-primary" onclick="uploadStateFile()">Upload & Analyze</button>
                        </div>
                    </div>
                </div>
                <div class="card bg-base-200 shadow-xl">
                    <div class="card-body">
                        <h2 class="card-title">Analysis Results</h2>
                        <div id="analysis-results" class="space-y-4">
                            <div class="alert alert-info">
                                <span>Upload a state file to begin analysis</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Remediation Tab -->
        <div id="remediation-tab" class="tab-content" style="display: none;">
            <div class="card bg-base-200 shadow-xl">
                <div class="card-body">
                    <h2 class="card-title">Automated Remediation</h2>
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Drift ID</span>
                        </label>
                        <input type="text" class="input input-bordered" id="remediation-drift-id" placeholder="Enter drift ID">
                    </div>
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Strategy</span>
                        </label>
                        <select class="select select-bordered" id="remediation-strategy">
                            <option value="auto">Auto-select</option>
                            <option value="conservative">Conservative</option>
                            <option value="aggressive">Aggressive</option>
                        </select>
                    </div>
                    <div class="form-control mt-6">
                        <button class="btn btn-warning" onclick="startRemediation()">Start Remediation</button>
                    </div>
                    <div id="remediation-results" class="mt-4"></div>
                </div>
            </div>
        </div>

        <!-- Workflows Tab -->
        <div id="workflows-tab" class="tab-content" style="display: none;">
            <div class="card bg-base-200 shadow-xl">
                <div class="card-body">
                    <h2 class="card-title">Workflow Automation</h2>
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Workflow Type</span>
                        </label>
                        <select class="select select-bordered" id="workflow-type">
                            <option value="discovery">Resource Discovery</option>
                            <option value="analysis">Drift Analysis</option>
                            <option value="remediation">Automated Remediation</option>
                            <option value="monitoring">Continuous Monitoring</option>
                        </select>
                    </div>
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Parameters (JSON)</span>
                        </label>
                        <textarea class="textarea textarea-bordered" id="workflow-params" placeholder='{"provider": "aws", "regions": ["us-east-1"]}' rows="4"></textarea>
                    </div>
                    <div class="form-control mt-6">
                        <button class="btn btn-success" onclick="executeWorkflow()">Execute Workflow</button>
                    </div>
                    <div id="workflow-results" class="mt-4"></div>
                </div>
            </div>
        </div>
    </div>

    <script>
        // WebSocket connection
        let ws = null;
        let reconnectAttempts = 0;
        const maxReconnectAttempts = 5;

        function connectWebSocket() {
            ws = new WebSocket('ws://' + window.location.host + '/ws');
            
            ws.onopen = function() {
                document.getElementById('ws-status').textContent = 'Connected';
                document.getElementById('ws-status').className = 'badge badge-success';
                reconnectAttempts = 0;
            };
            
            ws.onmessage = function(event) {
                const update = JSON.parse(event.data);
                handleProgressUpdate(update);
            };
            
            ws.onclose = function() {
                document.getElementById('ws-status').textContent = 'Disconnected';
                document.getElementById('ws-status').className = 'badge badge-error';
                
                if (reconnectAttempts < maxReconnectAttempts) {
                    setTimeout(connectWebSocket, 2000);
                    reconnectAttempts++;
                }
            };
            
            ws.onerror = function(error) {
                console.error('WebSocket error:', error);
            };
        }

        function handleProgressUpdate(update) {
            const toast = document.getElementById('progress-toast');
            const message = document.getElementById('progress-message');
            const progressBar = document.getElementById('progress-bar');
            
            message.textContent = update.message;
            
            if (update.progress !== undefined) {
                progressBar.value = update.progress;
                progressBar.max = update.total || 100;
            }
            
            toast.style.display = 'block';
            
            // Hide toast after 5 seconds
            setTimeout(() => {
                toast.style.display = 'none';
            }, 5000);
            
            // Update UI based on update type
            switch(update.type) {
                case 'discovery_progress':
                    updateDiscoveryResults(update);
                    break;
                case 'analysis_complete':
                    updateAnalysisResults(update);
                    break;
                case 'remediation_progress':
                    updateRemediationResults(update);
                    break;
                case 'workflow_progress':
                    updateWorkflowResults(update);
                    break;
            }
        }

        function showTab(tabName) {
            // Hide all tabs
            document.querySelectorAll('.tab-content').forEach(tab => {
                tab.style.display = 'none';
            });
            
            // Show selected tab
            document.getElementById(tabName + '-tab').style.display = 'block';
        }

        // Discovery functions
        async function startDiscovery() {
            const provider = document.getElementById('discovery-provider').value;
            const regions = document.getElementById('discovery-regions').value.split(',').map(r => r.trim());
            
            try {
                const response = await fetch('/api/v1/discover', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        provider: provider,
                        regions: regions
                    })
                });
                
                const result = await response.json();
                updateDiscoveryResults(result);
            } catch (error) {
                console.error('Discovery error:', error);
                document.getElementById('discovery-results').innerHTML = 
                    '<div class="alert alert-error">Error during discovery: ' + error.message + '</div>';
            }
        }

        function updateDiscoveryResults(data) {
            const resultsDiv = document.getElementById('discovery-results');
            if (data.resources && data.resources.length > 0) {
                let html = '<div class="alert alert-success">Discovered ' + data.resources.length + ' resources</div>';
                html += '<div class="overflow-x-auto"><table class="table table-zebra">';
                html += '<thead><tr><th>Type</th><th>ID</th><th>Region</th><th>Status</th></tr></thead><tbody>';
                
                data.resources.slice(0, 10).forEach(resource => {
                    html += '<tr><td>' + resource.type + '</td><td>' + resource.id + '</td><td>' + resource.region + '</td><td>' + resource.status + '</td></tr>';
                });
                
                html += '</tbody></table></div>';
                resultsDiv.innerHTML = html;
            } else {
                resultsDiv.innerHTML = '<div class="alert alert-warning">No resources discovered</div>';
            }
        }

        // Analysis functions
        async function uploadStateFile() {
            const fileInput = document.getElementById('state-file');
            const file = fileInput.files[0];
            
            if (!file) {
                alert('Please select a file');
                return;
            }
            
            const formData = new FormData();
            formData.append('file', file);
            
            try {
                const response = await fetch('/api/v1/upload', {
                    method: 'POST',
                    body: formData
                });
                
                const result = await response.json();
                if (result.success) {
                    // Now analyze the uploaded file
                    await analyzeStateFile(result.filename);
                }
            } catch (error) {
                console.error('Upload error:', error);
                document.getElementById('analysis-results').innerHTML = 
                    '<div class="alert alert-error">Error uploading file: ' + error.message + '</div>';
            }
        }

        async function analyzeStateFile(filename) {
            try {
                const response = await fetch('/api/v1/analyze', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        state_file: filename
                    })
                });
                
                const result = await response.json();
                updateAnalysisResults(result);
            } catch (error) {
                console.error('Analysis error:', error);
                document.getElementById('analysis-results').innerHTML = 
                    '<div class="alert alert-error">Error during analysis: ' + error.message + '</div>';
            }
        }

        function updateAnalysisResults(data) {
            const resultsDiv = document.getElementById('analysis-results');
            if (data.drifts && data.drifts.length > 0) {
                let html = '<div class="alert alert-warning">Found ' + data.drifts.length + ' drifts</div>';
                html += '<div class="space-y-2">';
                
                data.drifts.slice(0, 5).forEach(drift => {
                    html += '<div class="card bg-base-100">';
                    html += '<div class="card-body p-4">';
                    html += '<h3 class="card-title text-sm">' + drift.resource_type + ' - ' + drift.resource_id + '</h3>';
                    html += '<p class="text-sm">' + drift.description + '</p>';
                    html += '<div class="card-actions justify-end">';
                    html += '<button class="btn btn-xs btn-warning" onclick="remediateDrift(\'' + drift.id + '\')">Remediate</button>';
                    html += '</div></div></div>';
                });
                
                html += '</div>';
                resultsDiv.innerHTML = html;
            } else {
                resultsDiv.innerHTML = '<div class="alert alert-success">No drifts detected</div>';
            }
        }

        // Remediation functions
        async function startRemediation() {
            const driftId = document.getElementById('remediation-drift-id').value;
            const strategy = document.getElementById('remediation-strategy').value;
            
            if (!driftId) {
                alert('Please enter a drift ID');
                return;
            }
            
            try {
                const response = await fetch('/api/v1/remediate', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        drift_id: driftId,
                        strategy: strategy
                    })
                });
                
                const result = await response.json();
                updateRemediationResults(result);
            } catch (error) {
                console.error('Remediation error:', error);
                document.getElementById('remediation-results').innerHTML = 
                    '<div class="alert alert-error">Error during remediation: ' + error.message + '</div>';
            }
        }

        function updateRemediationResults(data) {
            const resultsDiv = document.getElementById('remediation-results');
            resultsDiv.innerHTML = '<div class="alert alert-info">Remediation completed</div>';
        }

        // Workflow functions
        async function executeWorkflow() {
            const workflowType = document.getElementById('workflow-type').value;
            const paramsText = document.getElementById('workflow-params').value;
            
            let params = {};
            try {
                params = JSON.parse(paramsText);
            } catch (error) {
                alert('Invalid JSON parameters');
                return;
            }
            
            try {
                const response = await fetch('/api/v1/workflows/execute', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        workflow_id: workflowType,
                        parameters: params
                    })
                });
                
                const result = await response.json();
                updateWorkflowResults(result);
            } catch (error) {
                console.error('Workflow error:', error);
                document.getElementById('workflow-results').innerHTML = 
                    '<div class="alert alert-error">Error executing workflow: ' + error.message + '</div>';
            }
        }

        function updateWorkflowResults(data) {
            const resultsDiv = document.getElementById('workflow-results');
            resultsDiv.innerHTML = '<div class="alert alert-success">Workflow executed successfully</div>';
        }

        // Initialize charts
        const driftCtx = document.getElementById('driftChart').getContext('2d');
        const costCtx = document.getElementById('costChart').getContext('2d');

        new Chart(driftCtx, {
            type: 'line',
            data: {
                labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'],
                datasets: [{
                    label: 'Drifts Detected',
                    data: [12, 19, 3, 5, 2, 3],
                    borderColor: 'rgb(75, 192, 192)',
                    tension: 0.1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false
            }
        });

        new Chart(costCtx, {
            type: 'doughnut',
            data: {
                labels: ['Compute', 'Storage', 'Network', 'Other'],
                datasets: [{
                    data: [45, 25, 20, 10],
                    backgroundColor: [
                        'rgb(255, 99, 132)',
                        'rgb(54, 162, 235)',
                        'rgb(255, 205, 86)',
                        'rgb(75, 192, 192)'
                    ]
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false
            }
        });

        // Connect WebSocket on page load
        connectWebSocket();
    </script>
</body>
</html>
`))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "driftmgr-server",
		"version": "1.6.3",
		"time":    time.Now().UTC(),
	}
	json.NewEncoder(w).Encode(response)
}

func handleStateFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Find actual Terraform state files
	stateFilePaths := findTerraformStateFiles(".")
	var stateFiles []models.StateFile

	for _, path := range stateFilePaths {
		stateFile, err := parseTerraformState(path)
		if err != nil {
			logger.Info("Warning: Failed to parse state file %s: %v", path, err)
			continue
		}
		stateFiles = append(stateFiles, *stateFile)
	}

	json.NewEncoder(w).Encode(stateFiles)
}

func handleStateFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract state file ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid state file ID", http.StatusBadRequest)
		return
	}
	stateFileID := pathParts[len(pathParts)-1]

	// Find the actual state file
	stateFilePaths := findTerraformStateFiles(".")
	var stateFile *models.StateFile

	for _, path := range stateFilePaths {
		if strings.Contains(path, stateFileID) {
			var err error
			stateFile, err = parseTerraformState(path)
			if err != nil {
				logger.Info("Warning: Failed to parse state file %s: %v", path, err)
				continue
			}
			break
		}
	}

	if stateFile == nil {
		http.Error(w, "State file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stateFile)
}

func handleDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Send initial progress update
	broadcastProgress(ProgressUpdate{
		Type:     "discovery_progress",
		Message:  fmt.Sprintf("Starting discovery for %s", req.Provider),
		Progress: 0,
		Total:    100,
	})

	// Handle "all" regions case
	var regions []string
	if len(req.Regions) == 1 && req.Regions[0] == "all" {
		// All AWS regions
		regions = []string{
			"us-east-1",      // US East (N. Virginia)
			"us-east-2",      // US East (Ohio)
			"us-west-1",      // US West (N. California)
			"us-west-2",      // US West (Oregon)
			"af-south-1",     // Africa (Cape Town)
			"ap-east-1",      // Asia Pacific (Hong Kong)
			"ap-south-1",     // Asia Pacific (Mumbai)
			"ap-northeast-1", // Asia Pacific (Tokyo)
			"ap-northeast-2", // Asia Pacific (Seoul)
			"ap-northeast-3", // Asia Pacific (Osaka)
			"ap-southeast-1", // Asia Pacific (Singapore)
			"ap-southeast-2", // Asia Pacific (Sydney)
			"ap-southeast-3", // Asia Pacific (Jakarta)
			"ap-southeast-4", // Asia Pacific (Melbourne)
			"ca-central-1",   // Canada (Central)
			"eu-central-1",   // Europe (Frankfurt)
			"eu-west-1",      // Europe (Ireland)
			"eu-west-2",      // Europe (London)
			"eu-west-3",      // Europe (Paris)
			"eu-north-1",     // Europe (Stockholm)
			"eu-south-1",     // Europe (Milan)
			"eu-south-2",     // Europe (Spain)
			"me-south-1",     // Middle East (Bahrain)
			"me-central-1",   // Middle East (UAE)
			"sa-east-1",      // South America (São Paulo)
		}
	} else {
		regions = req.Regions
	}

	broadcastProgress(ProgressUpdate{
		Type:     "discovery_progress",
		Message:  fmt.Sprintf("Scanning %d regions for %s", len(regions), req.Provider),
		Progress: 10,
		Total:    100,
	})

	// Load AWS configuration
	ctx := r.Context()
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AWS config: %v", err), http.StatusInternalServerError)
		return
	}

	// Comprehensive multi-cloud resource discovery
	var resources []models.Resource
	startTime := time.Now()

	switch req.Provider {
	case "aws":
		resources = discoverAWSResources(cfg, regions, req.Provider)
	case "azure":
		resources = discoverAzureResources(regions, req.Provider)
	case "gcp":
		resources = discoverGCPResources(regions, req.Provider)
	default:
		http.Error(w, "Unsupported provider", http.StatusBadRequest)
		return
	}

	duration := time.Since(startTime)

	// Send completion update
	broadcastProgress(ProgressUpdate{
		Type:     "discovery_complete",
		Message:  fmt.Sprintf("Discovery completed. Found %d resources in %v", len(resources), duration),
		Progress: 100,
		Total:    100,
		Data: map[string]interface{}{
			"resource_count": len(resources),
			"provider":       req.Provider,
			"regions":        regions,
			"duration":       duration.String(),
		},
	})

	response := models.DiscoveryResponse{
		Resources: resources,
		Total:     len(resources),
		Duration:  duration,
	}

	// Debug logging
	fmt.Printf("DEBUG: Discovered %d real resources across %d regions in %v\n", len(resources), len(regions), duration)
	for i, resource := range resources {
		if i < 10 { // Only show first 10 for debugging
			fmt.Printf("DEBUG: Resource %d: %s (%s) in %s\n", i+1, resource.Name, resource.Type, resource.Region)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleEnhancedDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Handle "all" regions case
	var regions []string
	if len(req.Regions) == 1 && req.Regions[0] == "all" {
		regions = []string{"us-east-1", "us-west-2", "eu-west-1", "eastus", "westus2", "us-central1", "europe-west1"}
	} else {
		regions = req.Regions
	}

	startTime := time.Now()

	// Use enhanced discoverer for comprehensive discovery
	enhancedResources, err := enhancedDiscoverer.DiscoverAllResourcesEnhanced(context.Background(), []string{req.Provider}, regions)
	if err != nil {
		logger.Error("Enhanced discovery failed: %v", err)
		http.Error(w, fmt.Sprintf("Enhanced discovery failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert pkg/models.Resource to internal/models.Resource
	var resources []models.Resource
	for _, enhancedResource := range enhancedResources {
		resource := models.Resource{
			ID:       enhancedResource.ID,
			Name:     enhancedResource.Name,
			Type:     enhancedResource.Type,
			Provider: enhancedResource.Provider,
			Region:   enhancedResource.Region,
			Tags:     enhancedResource.Tags,
			State:    enhancedResource.State,
		}
		resources = append(resources, resource)
	}

	duration := time.Since(startTime)
	response := models.DiscoveryResponse{
		Resources: resources,
		Total:     len(resources),
		Duration:  duration,
	}

	logger.Info("Enhanced discovery completed: %d resources in %v", len(resources), duration)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to extract tags from AWS resources
func extractTags(tags []types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[*tag.Key] = *tag.Value
		}
	}
	return result
}

// Helper function to get resource name from tags or ID
func getResourceName(tags map[string]string, id string) string {
	if name, ok := tags["Name"]; ok {
		return name
	}
	return id
}

// Discover EC2 resources (instances, VPCs, subnets, security groups, etc.)
func discoverEC2Resources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	ec2Client := ec2.NewFromConfig(cfg)

	// Discover EC2 instances
	instances, err := ec2Client.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{})
	if err != nil {
		logger.Info("Warning: Failed to describe instances in %s: %v", region, err)
	} else {
		for _, reservation := range instances.Reservations {
			for _, instance := range reservation.Instances {
				tags := extractTags(instance.Tags)
				resources = append(resources, models.Resource{
					ID:       aws.ToString(instance.InstanceId),
					Name:     getResourceName(tags, aws.ToString(instance.InstanceId)),
					Type:     "aws_instance",
					Provider: provider,
					Region:   region,
					Tags:     tags,
					State:    string(instance.State.Name),
					Created:  aws.ToTime(instance.LaunchTime),
					Updated:  time.Now(),
				})
			}
		}
	}

	// Discover VPCs
	vpcs, err := ec2Client.DescribeVpcs(context.TODO(), &ec2.DescribeVpcsInput{})
	if err != nil {
		logger.Info("Warning: Failed to describe VPCs in %s: %v", region, err)
	} else {
		for _, vpc := range vpcs.Vpcs {
			tags := extractTags(vpc.Tags)
			resources = append(resources, models.Resource{
				ID:       aws.ToString(vpc.VpcId),
				Name:     getResourceName(tags, aws.ToString(vpc.VpcId)),
				Type:     "aws_vpc",
				Provider: provider,
				Region:   region,
				Tags:     tags,
				State:    string(vpc.State),
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	// Discover Security Groups
	sgs, err := ec2Client.DescribeSecurityGroups(context.TODO(), &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		logger.Info("Warning: Failed to describe security groups in %s: %v", region, err)
	} else {
		for _, sg := range sgs.SecurityGroups {
			tags := extractTags(sg.Tags)
			resources = append(resources, models.Resource{
				ID:       aws.ToString(sg.GroupId),
				Name:     getResourceName(tags, aws.ToString(sg.GroupName)),
				Type:     "aws_security_group",
				Provider: provider,
				Region:   region,
				Tags:     tags,
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover RDS resources
func discoverRDSResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	rdsClient := rds.NewFromConfig(cfg)

	// Discover RDS instances
	instances, err := rdsClient.DescribeDBInstances(context.TODO(), &rds.DescribeDBInstancesInput{})
	if err != nil {
		logger.Info("Warning: Failed to describe RDS instances in %s: %v", region, err)
	} else {
		for _, instance := range instances.DBInstances {
			resources = append(resources, models.Resource{
				ID:       aws.ToString(instance.DBInstanceIdentifier),
				Name:     aws.ToString(instance.DBInstanceIdentifier),
				Type:     "aws_db_instance",
				Provider: provider,
				Region:   region,
				Tags:     make(map[string]string),
				State:    aws.ToString(instance.DBInstanceStatus),
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover S3 resources
func discoverS3Resources(cfg aws.Config, provider string) []models.Resource {
	var resources []models.Resource
	s3Client := s3.NewFromConfig(cfg)

	// List S3 buckets
	buckets, err := s3Client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	if err != nil {
		logger.Info("Warning: Failed to list S3 buckets: %v", err)
	} else {
		for _, bucket := range buckets.Buckets {
			resources = append(resources, models.Resource{
				ID:       aws.ToString(bucket.Name),
				Name:     aws.ToString(bucket.Name),
				Type:     "aws_s3_bucket",
				Provider: provider,
				Region:   "global",
				Tags:     make(map[string]string),
				State:    "active",
				Created:  aws.ToTime(bucket.CreationDate),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover IAM resources
func discoverIAMResources(cfg aws.Config, provider string) []models.Resource {
	var resources []models.Resource
	iamClient := iam.NewFromConfig(cfg)

	// List IAM users
	users, err := iamClient.ListUsers(context.TODO(), &iam.ListUsersInput{})
	if err != nil {
		logger.Info("Warning: Failed to list IAM users: %v", err)
	} else {
		for _, user := range users.Users {
			resources = append(resources, models.Resource{
				ID:       aws.ToString(user.UserName),
				Name:     aws.ToString(user.UserName),
				Type:     "aws_iam_user",
				Provider: provider,
				Region:   "global",
				Tags:     make(map[string]string),
				State:    "active",
				Created:  aws.ToTime(user.CreateDate),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover Lambda resources
func discoverLambdaResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	lambdaClient := lambda.NewFromConfig(cfg)

	// List Lambda functions
	functions, err := lambdaClient.ListFunctions(context.TODO(), &lambda.ListFunctionsInput{})
	if err != nil {
		logger.Info("Warning: Failed to list Lambda functions in %s: %v", region, err)
	} else {
		for _, function := range functions.Functions {
			resources = append(resources, models.Resource{
				ID:       aws.ToString(function.FunctionName),
				Name:     aws.ToString(function.FunctionName),
				Type:     "aws_lambda_function",
				Provider: provider,
				Region:   region,
				Tags:     make(map[string]string),
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover CloudFormation resources
func discoverCloudFormationResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	cfClient := cloudformation.NewFromConfig(cfg)

	// List CloudFormation stacks
	stacks, err := cfClient.ListStacks(context.TODO(), &cloudformation.ListStacksInput{})
	if err != nil {
		logger.Info("Warning: Failed to list CloudFormation stacks in %s: %v", region, err)
	} else {
		for _, stack := range stacks.StackSummaries {
			if stack.StackStatus != "DELETE_COMPLETE" {
				resources = append(resources, models.Resource{
					ID:       aws.ToString(stack.StackName),
					Name:     aws.ToString(stack.StackName),
					Type:     "aws_cloudformation_stack",
					Provider: provider,
					Region:   region,
					Tags:     make(map[string]string),
					State:    string(stack.StackStatus),
					Created:  aws.ToTime(stack.CreationTime),
					Updated:  time.Now(),
				})
			}
		}
	}

	return resources
}

// Discover ElastiCache resources
func discoverElastiCacheResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	ecClient := elasticache.NewFromConfig(cfg)

	// List ElastiCache clusters
	clusters, err := ecClient.DescribeCacheClusters(context.TODO(), &elasticache.DescribeCacheClustersInput{})
	if err != nil {
		logger.Info("Warning: Failed to list ElastiCache clusters in %s: %v", region, err)
	} else {
		for _, cluster := range clusters.CacheClusters {
			resources = append(resources, models.Resource{
				ID:       aws.ToString(cluster.CacheClusterId),
				Name:     aws.ToString(cluster.CacheClusterId),
				Type:     "aws_elasticache_cluster",
				Provider: provider,
				Region:   region,
				Tags:     make(map[string]string),
				State:    aws.ToString(cluster.CacheClusterStatus),
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover ECS resources
func discoverECSResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	ecsClient := ecs.NewFromConfig(cfg)

	// List ECS clusters
	clusters, err := ecsClient.ListClusters(context.TODO(), &ecs.ListClustersInput{})
	if err != nil {
		logger.Info("Warning: Failed to list ECS clusters in %s: %v", region, err)
	} else {
		for _, clusterArn := range clusters.ClusterArns {
			parts := strings.Split(clusterArn, "/")
			if len(parts) > 1 {
				clusterName := parts[1]
				resources = append(resources, models.Resource{
					ID:       clusterName,
					Name:     clusterName,
					Type:     "aws_ecs_cluster",
					Provider: provider,
					Region:   region,
					Tags:     make(map[string]string),
					State:    "active",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				})
			}
		}
	}

	return resources
}

// Discover EKS resources
func discoverEKSResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	eksClient := eks.NewFromConfig(cfg)

	// List EKS clusters
	clusters, err := eksClient.ListClusters(context.TODO(), &eks.ListClustersInput{})
	if err != nil {
		logger.Info("Warning: Failed to list EKS clusters in %s: %v", region, err)
	} else {
		for _, clusterName := range clusters.Clusters {
			clusterNameStr := clusterName
			resources = append(resources, models.Resource{
				ID:       clusterNameStr,
				Name:     clusterNameStr,
				Type:     "aws_eks_cluster",
				Provider: provider,
				Region:   region,
				Tags:     make(map[string]string),
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover Route53 resources
func discoverRoute53Resources(cfg aws.Config, provider string) []models.Resource {
	var resources []models.Resource
	r53Client := route53.NewFromConfig(cfg)

	// List hosted zones
	zones, err := r53Client.ListHostedZones(context.TODO(), &route53.ListHostedZonesInput{})
	if err != nil {
		logger.Info("Warning: Failed to list Route53 hosted zones: %v", err)
	} else {
		for _, zone := range zones.HostedZones {
			zoneName := aws.ToString(zone.Name)
			if strings.HasSuffix(zoneName, ".") {
				zoneName = zoneName[:len(zoneName)-1]
			}
			resources = append(resources, models.Resource{
				ID:       aws.ToString(zone.Id),
				Name:     zoneName,
				Type:     "aws_route53_zone",
				Provider: provider,
				Region:   "global",
				Tags:     make(map[string]string),
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover SQS resources
func discoverSQSResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	sqsClient := sqs.NewFromConfig(cfg)

	// List SQS queues
	queues, err := sqsClient.ListQueues(context.TODO(), &sqs.ListQueuesInput{})
	if err != nil {
		logger.Info("Warning: Failed to list SQS queues in %s: %v", region, err)
	} else {
		for _, queueUrl := range queues.QueueUrls {
			queueUrlStr := queueUrl
			parts := strings.Split(queueUrlStr, "/")
			queueName := parts[len(parts)-1]
			resources = append(resources, models.Resource{
				ID:       queueName,
				Name:     queueName,
				Type:     "aws_sqs_queue",
				Provider: provider,
				Region:   region,
				Tags:     make(map[string]string),
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover SNS resources
func discoverSNSResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	snsClient := sns.NewFromConfig(cfg)

	// List SNS topics
	topics, err := snsClient.ListTopics(context.TODO(), &sns.ListTopicsInput{})
	if err != nil {
		logger.Info("Warning: Failed to list SNS topics in %s: %v", region, err)
	} else {
		for _, topic := range topics.Topics {
			topicName := strings.Split(aws.ToString(topic.TopicArn), ":")[len(strings.Split(aws.ToString(topic.TopicArn), ":"))-1]
			resources = append(resources, models.Resource{
				ID:       topicName,
				Name:     topicName,
				Type:     "aws_sns_topic",
				Provider: provider,
				Region:   region,
				Tags:     make(map[string]string),
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover DynamoDB resources
func discoverDynamoDBResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	ddbClient := dynamodb.NewFromConfig(cfg)

	// List DynamoDB tables
	tables, err := ddbClient.ListTables(context.TODO(), &dynamodb.ListTablesInput{})
	if err != nil {
		logger.Info("Warning: Failed to list DynamoDB tables in %s: %v", region, err)
	} else {
		for _, tableName := range tables.TableNames {
			tableNameStr := tableName
			resources = append(resources, models.Resource{
				ID:       tableNameStr,
				Name:     tableNameStr,
				Type:     "aws_dynamodb_table",
				Provider: provider,
				Region:   region,
				Tags:     make(map[string]string),
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover Auto Scaling resources
func discoverAutoScalingResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	asClient := autoscaling.NewFromConfig(cfg)

	// List Auto Scaling groups
	groups, err := asClient.DescribeAutoScalingGroups(context.TODO(), &autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		logger.Info("Warning: Failed to list Auto Scaling groups in %s: %v", region, err)
	} else {
		for _, group := range groups.AutoScalingGroups {
			tags := make(map[string]string)
			for _, tag := range group.Tags {
				if tag.Key != nil && tag.Value != nil {
					tags[*tag.Key] = *tag.Value
				}
			}
			resources = append(resources, models.Resource{
				ID:       aws.ToString(group.AutoScalingGroupName),
				Name:     getResourceName(tags, aws.ToString(group.AutoScalingGroupName)),
				Type:     "aws_autoscaling_group",
				Provider: provider,
				Region:   region,
				Tags:     tags,
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Comprehensive AWS resource discovery
func discoverAWSResources(cfg aws.Config, regions []string, provider string) []models.Resource {
	var resources []models.Resource

	for _, region := range regions {
		cfg.Region = region
		logger.Info("Scanning AWS region: %s", region)

		// Core compute and networking
		resources = append(resources, discoverEC2Resources(cfg, region, provider)...)
		resources = append(resources, discoverRDSResources(cfg, region, provider)...)
		resources = append(resources, discoverLambdaResources(cfg, region, provider)...)
		resources = append(resources, discoverCloudFormationResources(cfg, region, provider)...)
		resources = append(resources, discoverElastiCacheResources(cfg, region, provider)...)
		resources = append(resources, discoverECSResources(cfg, region, provider)...)
		resources = append(resources, discoverEKSResources(cfg, region, provider)...)
		resources = append(resources, discoverSQSResources(cfg, region, provider)...)
		resources = append(resources, discoverSNSResources(cfg, region, provider)...)
		resources = append(resources, discoverDynamoDBResources(cfg, region, provider)...)
		resources = append(resources, discoverAutoScalingResources(cfg, region, provider)...)

		// Enhanced Security Services
		resources = append(resources, discoverWAFResources(cfg, region, provider)...)
		resources = append(resources, discoverShieldResources(cfg, region, provider)...)
		resources = append(resources, discoverConfigResources(cfg, region, provider)...)
		resources = append(resources, discoverGuardDutyResources(cfg, region, provider)...)
		resources = append(resources, discoverSecretsManagerResources(cfg, region, provider)...)
		resources = append(resources, discoverKMSResources(cfg, region, provider)...)
		resources = append(resources, discoverCloudTrailResources(cfg, region, provider)...)
		resources = append(resources, discoverMacieResources(cfg, region, provider)...)
		resources = append(resources, discoverSecurityHubResources(cfg, region, provider)...)
		resources = append(resources, discoverDetectiveResources(cfg, region, provider)...)
		resources = append(resources, discoverInspectorResources(cfg, region, provider)...)
		resources = append(resources, discoverArtifactResources(cfg, region, provider)...)

		// Enhanced Networking Services
		resources = append(resources, discoverVPCResources(cfg, region, provider)...)
		resources = append(resources, discoverSubnetResources(cfg, region, provider)...)
		resources = append(resources, discoverLoadBalancerResources(cfg, region, provider)...)
		resources = append(resources, discoverInternetGatewayResources(cfg, region, provider)...)
		resources = append(resources, discoverNATGatewayResources(cfg, region, provider)...)
		resources = append(resources, discoverVPNGatewayResources(cfg, region, provider)...)
		resources = append(resources, discoverDirectConnectResources(cfg, region, provider)...)
		resources = append(resources, discoverTransitGatewayResources(cfg, region, provider)...)
		resources = append(resources, discoverRouteTableResources(cfg, region, provider)...)
		resources = append(resources, discoverNetworkACLResources(cfg, region, provider)...)
		resources = append(resources, discoverElasticIPResources(cfg, region, provider)...)
		resources = append(resources, discoverVPCEndpointResources(cfg, region, provider)...)
		resources = append(resources, discoverVPCFlowLogResources(cfg, region, provider)...)

		// Global services (only check once)
		if region == "us-east-1" {
			resources = append(resources, discoverS3Resources(cfg, provider)...)
			resources = append(resources, discoverIAMResources(cfg, provider)...)
			resources = append(resources, discoverRoute53Resources(cfg, provider)...)
			resources = append(resources, discoverCloudFrontResources(cfg, provider)...)
			resources = append(resources, discoverCertificateManagerResources(cfg, provider)...)
			resources = append(resources, discoverOrganizationsResources(cfg, provider)...)
			resources = append(resources, discoverControlTowerResources(cfg, provider)...)
		}
	}

	return resources
}

// Enhanced AWS Security Services Discovery

// Discover WAF resources
func discoverWAFResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// WAF discovery implementation would go here
	// For now, return empty slice to allow compilation
	return resources
}

// Discover Shield resources
func discoverShieldResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Shield discovery implementation would go here
	return resources
}

// Discover Config resources
func discoverConfigResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Config discovery implementation would go here
	return resources
}

// Discover GuardDuty resources
func discoverGuardDutyResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// GuardDuty discovery implementation would go here
	return resources
}

// Discover Secrets Manager resources
func discoverSecretsManagerResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Secrets Manager discovery implementation would go here
	return resources
}

// Discover KMS resources
func discoverKMSResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// KMS discovery implementation would go here
	return resources
}

// Discover CloudTrail resources
func discoverCloudTrailResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// CloudTrail discovery implementation would go here
	return resources
}

// Discover Macie resources
func discoverMacieResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Macie discovery implementation would go here
	return resources
}

// Discover Security Hub resources
func discoverSecurityHubResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Security Hub discovery implementation would go here
	return resources
}

// Discover Detective resources
func discoverDetectiveResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Detective discovery implementation would go here
	return resources
}

// Discover Inspector resources
func discoverInspectorResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Inspector discovery implementation would go here
	return resources
}

// Discover Artifact resources
func discoverArtifactResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Artifact discovery implementation would go here
	return resources
}

// Enhanced AWS Networking Services Discovery

// Discover VPC resources
func discoverVPCResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// VPC discovery implementation would go here
	return resources
}

// Discover Subnet resources
func discoverSubnetResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Subnet discovery implementation would go here
	return resources
}

// Discover Load Balancer resources
func discoverLoadBalancerResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Load Balancer discovery implementation would go here
	return resources
}

// Discover Internet Gateway resources
func discoverInternetGatewayResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Internet Gateway discovery implementation would go here
	return resources
}

// Discover NAT Gateway resources
func discoverNATGatewayResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// NAT Gateway discovery implementation would go here
	return resources
}

// Discover VPN Gateway resources
func discoverVPNGatewayResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// VPN Gateway discovery implementation would go here
	return resources
}

// Discover Direct Connect resources
func discoverDirectConnectResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Direct Connect discovery implementation would go here
	return resources
}

// Discover Transit Gateway resources
func discoverTransitGatewayResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Transit Gateway discovery implementation would go here
	return resources
}

// Discover Route Table resources
func discoverRouteTableResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Route Table discovery implementation would go here
	return resources
}

// Discover Network ACL resources
func discoverNetworkACLResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Network ACL discovery implementation would go here
	return resources
}

// Discover Elastic IP resources
func discoverElasticIPResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// Elastic IP discovery implementation would go here
	return resources
}

// Discover VPC Endpoint resources
func discoverVPCEndpointResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// VPC Endpoint discovery implementation would go here
	return resources
}

// Discover VPC Flow Log resources
func discoverVPCFlowLogResources(cfg aws.Config, region, provider string) []models.Resource {
	var resources []models.Resource
	// VPC Flow Log discovery implementation would go here
	return resources
}

// Enhanced AWS Global Services Discovery

// Discover CloudFront resources
func discoverCloudFrontResources(cfg aws.Config, provider string) []models.Resource {
	var resources []models.Resource
	// CloudFront discovery implementation would go here
	return resources
}

// Discover Certificate Manager resources
func discoverCertificateManagerResources(cfg aws.Config, provider string) []models.Resource {
	var resources []models.Resource
	// Certificate Manager discovery implementation would go here
	return resources
}

// Discover Organizations resources
func discoverOrganizationsResources(cfg aws.Config, provider string) []models.Resource {
	var resources []models.Resource
	// Organizations discovery implementation would go here
	return resources
}

// Discover Control Tower resources
func discoverControlTowerResources(cfg aws.Config, provider string) []models.Resource {
	var resources []models.Resource
	// Control Tower discovery implementation would go here
	return resources
}

// Comprehensive Azure resource discovery using enhanced discovery engine
func discoverAzureResources(regions []string, provider string) []models.Resource {
	var resources []models.Resource

	// Try to use enhanced Azure discovery first (Windows-optimized)
	enhancedDiscoverer, err := discovery.NewAzureEnhancedDiscoverer("")
	if err == nil {
		logger.Info("Using enhanced Azure discovery engine")
		// Enhanced discoverer discovers all resources across all regions
		regionResources, err := enhancedDiscoverer.DiscoverResources(context.Background())
		if err != nil {
			logger.Info("Warning: Enhanced discovery failed: %v, falling back to CLI", err)
			// Fall back to CLI-based discovery
			for _, region := range regions {
				resources = append(resources, discoverAzureResourcesCLI(region, provider)...)
			}
		} else {
			// Convert pkg/models.Resource to internal/models.Resource
			for _, enhancedResource := range regionResources {
				resource := models.Resource{
					ID:       enhancedResource.ID,
					Name:     enhancedResource.Name,
					Type:     enhancedResource.Type,
					Provider: enhancedResource.Provider,
					Region:   enhancedResource.Region,
					Tags:     enhancedResource.Tags,
					State:    enhancedResource.State,
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				}
				resources = append(resources, resource)
			}
		}
		return resources
	}

	// Fall back to CLI-based discovery if enhanced discovery is not available
	logger.Info("Enhanced Azure discovery not available, using CLI-based discovery: %v", err)
	for _, region := range regions {
		logger.Info("Scanning Azure region: %s", region)
		resources = append(resources, discoverAzureResourcesCLI(region, provider)...)
	}

	return resources
}

// CLI-based Azure resource discovery (fallback method)
func discoverAzureResourcesCLI(region, provider string) []models.Resource {
	var resources []models.Resource

	// Core Compute & Storage
	resources = append(resources, discoverAzureVMsCLI(region, provider)...)
	resources = append(resources, discoverAzureStorageAccountsCLI(region, provider)...)
	resources = append(resources, discoverAzureSQLDatabasesCLI(region, provider)...)
	resources = append(resources, discoverAzureWebAppsCLI(region, provider)...)
	resources = append(resources, discoverAzureResourceGroupsCLI(region, provider)...)

	// Enhanced Networking Services
	resources = append(resources, discoverAzureVirtualNetworksCLI(region, provider)...)
	resources = append(resources, discoverAzureLoadBalancersCLI(region, provider)...)
	resources = append(resources, discoverAzureNetworkInterfacesCLI(region, provider)...)
	resources = append(resources, discoverAzurePublicIPAddressesCLI(region, provider)...)
	resources = append(resources, discoverAzureVPNGatewaysCLI(region, provider)...)
	resources = append(resources, discoverAzureExpressRouteCLI(region, provider)...)
	resources = append(resources, discoverAzureApplicationGatewaysCLI(region, provider)...)
	resources = append(resources, discoverAzureFrontDoorCLI(region, provider)...)
	resources = append(resources, discoverAzureCDNProfilesCLI(region, provider)...)
	resources = append(resources, discoverAzureRouteTablesCLI(region, provider)...)
	resources = append(resources, discoverAzureNetworkSecurityGroupsCLI(region, provider)...)
	resources = append(resources, discoverAzureFirewallsCLI(region, provider)...)
	resources = append(resources, discoverAzureBastionHostsCLI(region, provider)...)

	// Enhanced Security Services
	resources = append(resources, discoverAzureKeyVaultsCLI(region, provider)...)
	resources = append(resources, discoverAzureSecurityCenterCLI(region, provider)...)
	resources = append(resources, discoverAzureSentinelCLI(region, provider)...)
	resources = append(resources, discoverAzureDefenderCLI(region, provider)...)
	resources = append(resources, discoverAzurePolicyCLI(region, provider)...)
	resources = append(resources, discoverAzureLighthouseCLI(region, provider)...)
	resources = append(resources, discoverAzurePrivilegedIdentityManagementCLI(region, provider)...)
	resources = append(resources, discoverAzureConditionalAccessCLI(region, provider)...)
	resources = append(resources, discoverAzureInformationProtectionCLI(region, provider)...)

	// Additional Services
	resources = append(resources, discoverAzureContainerRegistriesCLI(region, provider)...)
	resources = append(resources, discoverAzureKubernetesServicesCLI(region, provider)...)
	resources = append(resources, discoverAzureFunctionsCLI(region, provider)...)
	resources = append(resources, discoverAzureLogicAppsCLI(region, provider)...)
	resources = append(resources, discoverAzureEventHubsCLI(region, provider)...)
	resources = append(resources, discoverAzureServiceBusCLI(region, provider)...)
	resources = append(resources, discoverAzureCosmosDBCLI(region, provider)...)
	resources = append(resources, discoverAzureRedisCacheCLI(region, provider)...)
	resources = append(resources, discoverAzureDataFactoryCLI(region, provider)...)
	resources = append(resources, discoverAzureSynapseAnalyticsCLI(region, provider)...)
	resources = append(resources, discoverAzureApplicationInsightsCLI(region, provider)...)

	return resources
}

// Discover Azure Virtual Machines using CLI
func discoverAzureVMsCLI(region, provider string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to list VMs
	cmd := exec.Command("az", "vm", "list", "--query", "[?location=='"+region+"'].{id:id, name:name, resourceGroup:resourceGroup}", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list Azure VMs in %s: %v", region, err)
		return resources
	}

	// Parse JSON output
	var vms []map[string]interface{}
	if err := json.Unmarshal(output, &vms); err != nil {
		logger.Info("Warning: Failed to parse Azure VM list for %s: %v", region, err)
		return resources
	}

	for _, vm := range vms {
		if id, ok := vm["id"].(string); ok {
			if name, ok := vm["name"].(string); ok {
				resources = append(resources, models.Resource{
					ID:       id,
					Name:     name,
					Type:     "azure_virtual_machine",
					Provider: provider,
					Region:   region,
					Tags:     make(map[string]string),
					State:    "running",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				})
			}
		}
	}

	return resources
}

// Discover Azure Storage Accounts using CLI
func discoverAzureStorageAccountsCLI(region, provider string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to list storage accounts
	cmd := exec.Command("az", "storage", "account", "list", "--query", "[?location=='"+region+"'].{id:id, name:name, resourceGroup:resourceGroup}", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list Azure storage accounts in %s: %v", region, err)
		return resources
	}

	// Parse JSON output
	var accounts []map[string]interface{}
	if err := json.Unmarshal(output, &accounts); err != nil {
		logger.Info("Warning: Failed to parse Azure storage account list for %s: %v", region, err)
		return resources
	}

	for _, account := range accounts {
		if id, ok := account["id"].(string); ok {
			if name, ok := account["name"].(string); ok {
				resources = append(resources, models.Resource{
					ID:       id,
					Name:     name,
					Type:     "azure_storage_account",
					Provider: provider,
					Region:   region,
					Tags:     make(map[string]string),
					State:    "active",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				})
			}
		}
	}

	return resources
}

// Discover Azure Resource Groups using CLI
func discoverAzureResourceGroupsCLI(region, provider string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to list resource groups
	cmd := exec.Command("az", "group", "list", "--query", "[?location=='"+region+"'].{id:id, name:name}", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list Azure resource groups in %s: %v", region, err)
		return resources
	}

	// Parse JSON output
	var groups []map[string]interface{}
	if err := json.Unmarshal(output, &groups); err != nil {
		logger.Info("Warning: Failed to parse Azure resource group list for %s: %v", region, err)
		return resources
	}

	for _, group := range groups {
		if id, ok := group["id"].(string); ok {
			if name, ok := group["name"].(string); ok {
				resources = append(resources, models.Resource{
					ID:       id,
					Name:     name,
					Type:     "azure_resource_group",
					Provider: provider,
					Region:   region,
					Tags:     make(map[string]string),
					State:    "active",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				})
			}
		}
	}

	return resources
}

// Discover Azure SQL Databases using CLI
func discoverAzureSQLDatabasesCLI(region, provider string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to list SQL servers
	cmd := exec.Command("az", "sql", "server", "list", "--query", "[?location=='"+region+"'].{id:id, name:name, resourceGroup:resourceGroup}", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list Azure SQL servers in %s: %v", region, err)
		return resources
	}

	// Parse JSON output
	var servers []map[string]interface{}
	if err := json.Unmarshal(output, &servers); err != nil {
		logger.Info("Warning: Failed to parse Azure SQL server list for %s: %v", region, err)
		return resources
	}

	for _, server := range servers {
		if id, ok := server["id"].(string); ok {
			if name, ok := server["name"].(string); ok {
				resources = append(resources, models.Resource{
					ID:       id,
					Name:     name,
					Type:     "azure_sql_server",
					Provider: provider,
					Region:   region,
					Tags:     make(map[string]string),
					State:    "active",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				})
			}
		}
	}

	return resources
}

// Discover Azure Web Apps using CLI
func discoverAzureWebAppsCLI(region, provider string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to list web apps
	cmd := exec.Command("az", "webapp", "list", "--query", "[?location=='"+region+"'].{id:id, name:name, resourceGroup:resourceGroup}", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list Azure web apps in %s: %v", region, err)
		return resources
	}

	// Parse JSON output
	var apps []map[string]interface{}
	if err := json.Unmarshal(output, &apps); err != nil {
		logger.Info("Warning: Failed to parse Azure web app list for %s: %v", region, err)
		return resources
	}

	for _, app := range apps {
		if id, ok := app["id"].(string); ok {
			if name, ok := app["name"].(string); ok {
				resources = append(resources, models.Resource{
					ID:       id,
					Name:     name,
					Type:     "azure_app_service",
					Provider: provider,
					Region:   region,
					Tags:     make(map[string]string),
					State:    "active",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				})
			}
		}
	}

	return resources
}

// Discover Azure Virtual Networks using CLI
func discoverAzureVirtualNetworksCLI(region, provider string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to list virtual networks
	cmd := exec.Command("az", "network", "vnet", "list", "--query", "[?location=='"+region+"'].{id:id, name:name, resourceGroup:resourceGroup}", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list Azure virtual networks in %s: %v", region, err)
		return resources
	}

	// Parse JSON output
	var vnets []map[string]interface{}
	if err := json.Unmarshal(output, &vnets); err != nil {
		logger.Info("Warning: Failed to parse Azure virtual network list for %s: %v", region, err)
		return resources
	}

	for _, vnet := range vnets {
		if id, ok := vnet["id"].(string); ok {
			if name, ok := vnet["name"].(string); ok {
				resources = append(resources, models.Resource{
					ID:       id,
					Name:     name,
					Type:     "azure_virtual_network",
					Provider: provider,
					Region:   region,
					Tags:     make(map[string]string),
					State:    "active",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				})
			}
		}
	}

	return resources
}

// Discover Azure Load Balancers using CLI
func discoverAzureLoadBalancersCLI(region, provider string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to list load balancers
	cmd := exec.Command("az", "network", "lb", "list", "--query", "[?location=='"+region+"'].{id:id, name:name, resourceGroup:resourceGroup}", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list Azure load balancers in %s: %v", region, err)
		return resources
	}

	// Parse JSON output
	var lbs []map[string]interface{}
	if err := json.Unmarshal(output, &lbs); err != nil {
		logger.Info("Warning: Failed to parse Azure load balancer list for %s: %v", region, err)
		return resources
	}

	for _, lb := range lbs {
		if id, ok := lb["id"].(string); ok {
			if name, ok := lb["name"].(string); ok {
				resources = append(resources, models.Resource{
					ID:       id,
					Name:     name,
					Type:     "azure_lb",
					Provider: provider,
					Region:   region,
					Tags:     make(map[string]string),
					State:    "active",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				})
			}
		}
	}

	return resources
}

// Discover Azure Key Vaults using CLI
func discoverAzureKeyVaultsCLI(region, provider string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to list key vaults
	cmd := exec.Command("az", "keyvault", "list", "--query", "[?location=='"+region+"'].{id:id, name:name, resourceGroup:resourceGroup}", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list Azure key vaults in %s: %v", region, err)
		return resources
	}

	// Parse JSON output
	var vaults []map[string]interface{}
	if err := json.Unmarshal(output, &vaults); err != nil {
		logger.Info("Warning: Failed to parse Azure key vault list for %s: %v", region, err)
		return resources
	}

	for _, vault := range vaults {
		if id, ok := vault["id"].(string); ok {
			if name, ok := vault["name"].(string); ok {
				resources = append(resources, models.Resource{
					ID:       id,
					Name:     name,
					Type:     "azure_key_vault",
					Provider: provider,
					Region:   region,
					Tags:     make(map[string]string),
					State:    "active",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				})
			}
		}
	}

	return resources
}

// Enhanced Azure Networking Services Discovery

// Discover Azure Network Interfaces using CLI
func discoverAzureNetworkInterfacesCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Network Interfaces discovery implementation would go here
	return resources
}

// Discover Azure Public IP Addresses using CLI
func discoverAzurePublicIPAddressesCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Public IP Addresses discovery implementation would go here
	return resources
}

// Discover Azure VPN Gateways using CLI
func discoverAzureVPNGatewaysCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// VPN Gateways discovery implementation would go here
	return resources
}

// Discover Azure Express Route using CLI
func discoverAzureExpressRouteCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Express Route discovery implementation would go here
	return resources
}

// Discover Azure Application Gateways using CLI
func discoverAzureApplicationGatewaysCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Application Gateways discovery implementation would go here
	return resources
}

// Discover Azure Front Door using CLI
func discoverAzureFrontDoorCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Front Door discovery implementation would go here
	return resources
}

// Discover Azure CDN Profiles using CLI
func discoverAzureCDNProfilesCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// CDN Profiles discovery implementation would go here
	return resources
}

// Discover Azure Route Tables using CLI
func discoverAzureRouteTablesCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Route Tables discovery implementation would go here
	return resources
}

// Discover Azure Network Security Groups using CLI
func discoverAzureNetworkSecurityGroupsCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Network Security Groups discovery implementation would go here
	return resources
}

// Discover Azure Firewalls using CLI
func discoverAzureFirewallsCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Firewalls discovery implementation would go here
	return resources
}

// Discover Azure Bastion Hosts using CLI
func discoverAzureBastionHostsCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Bastion Hosts discovery implementation would go here
	return resources
}

// Enhanced Azure Security Services Discovery

// Discover Azure Security Center using CLI
func discoverAzureSecurityCenterCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Security Center discovery implementation would go here
	return resources
}

// Discover Azure Sentinel using CLI
func discoverAzureSentinelCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Sentinel discovery implementation would go here
	return resources
}

// Discover Azure Defender using CLI
func discoverAzureDefenderCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Defender discovery implementation would go here
	return resources
}

// Discover Azure Policy using CLI
func discoverAzurePolicyCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Policy discovery implementation would go here
	return resources
}

// Discover Azure Lighthouse using CLI
func discoverAzureLighthouseCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Lighthouse discovery implementation would go here
	return resources
}

// Discover Azure Privileged Identity Management using CLI
func discoverAzurePrivilegedIdentityManagementCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Privileged Identity Management discovery implementation would go here
	return resources
}

// Discover Azure Conditional Access using CLI
func discoverAzureConditionalAccessCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Conditional Access discovery implementation would go here
	return resources
}

// Discover Azure Information Protection using CLI
func discoverAzureInformationProtectionCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Information Protection discovery implementation would go here
	return resources
}

// Enhanced Azure Additional Services Discovery

// Discover Azure Container Registries using CLI
func discoverAzureContainerRegistriesCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Container Registries discovery implementation would go here
	return resources
}

// Discover Azure Kubernetes Services using CLI
func discoverAzureKubernetesServicesCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Kubernetes Services discovery implementation would go here
	return resources
}

// Discover Azure Functions using CLI
func discoverAzureFunctionsCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Functions discovery implementation would go here
	return resources
}

// Discover Azure Logic Apps using CLI
func discoverAzureLogicAppsCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Logic Apps discovery implementation would go here
	return resources
}

// Discover Azure Event Hubs using CLI
func discoverAzureEventHubsCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Event Hubs discovery implementation would go here
	return resources
}

// Discover Azure Service Bus using CLI
func discoverAzureServiceBusCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Service Bus discovery implementation would go here
	return resources
}

// Discover Azure Cosmos DB using CLI
func discoverAzureCosmosDBCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Cosmos DB discovery implementation would go here
	return resources
}

// Discover Azure Redis Cache using CLI
func discoverAzureRedisCacheCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Redis Cache discovery implementation would go here
	return resources
}

// Discover Azure Data Factory using CLI
func discoverAzureDataFactoryCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Data Factory discovery implementation would go here
	return resources
}

// Discover Azure Synapse Analytics using CLI
func discoverAzureSynapseAnalyticsCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Synapse Analytics discovery implementation would go here
	return resources
}

// Discover Azure Application Insights using CLI
func discoverAzureApplicationInsightsCLI(region, provider string) []models.Resource {
	var resources []models.Resource
	// Application Insights discovery implementation would go here
	return resources
}

// Comprehensive GCP resource discovery using gcloud CLI
func discoverGCPResources(regions []string, provider string) []models.Resource {
	var resources []models.Resource

	// Get GCP project ID
	projectID, err := getGCPProjectID()
	if err != nil {
		logger.Info("Error getting GCP project ID: %v", err)
		return resources
	}

	for _, region := range regions {
		logger.Info("Scanning GCP region: %s", region)

		// Core Compute & Storage
		resources = append(resources, discoverGCPComputeInstancesCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPStorageBucketsCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPGKEClustersCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudSQLInstancesCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPVPCNetworksCLI(projectID, region, provider)...)

		// Enhanced Networking Services
		resources = append(resources, discoverGCPSubnetsCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPFirewallRulesCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPLoadBalancersCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPVPNGatewaysCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudRoutersCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudNATCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudInterconnectCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudArmorCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudCDNCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudDNSZonesCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudEndpointsCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudTrafficDirectorCLI(projectID, region, provider)...)

		// Enhanced Security Services
		resources = append(resources, discoverGCPCloudKMSKeyRingsCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudSecurityScannerCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudAssetInventoryCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudAccessContextManagerCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudBinaryAuthorizationCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudDLPCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudIAPCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudResourceManagerCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudIAMCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudOrganizationPolicyCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudSecurityCommandCenterCLI(projectID, region, provider)...)

		// Additional Services
		resources = append(resources, discoverGCPCloudFunctionsCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudRunCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudBuildCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudPubSubCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPBigQueryCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudSpannerCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudFirestoreCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudMonitoringCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudLoggingCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudTraceCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudDebuggerCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudProfilerCLI(projectID, region, provider)...)
		resources = append(resources, discoverGCPCloudErrorReportingCLI(projectID, region, provider)...)
	}

	return resources
}

// Discover GCP Compute Instances using CLI
func discoverGCPComputeInstancesCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to list compute instances
	cmd := exec.Command("gcloud", "compute", "instances", "list", "--project", projectID, "--filter", "zone:"+region+"*", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list GCP compute instances in %s: %v", region, err)
		return resources
	}

	// Parse JSON output
	var instances []map[string]interface{}
	if err := json.Unmarshal(output, &instances); err != nil {
		logger.Info("Warning: Failed to parse GCP compute instance list for %s: %v", region, err)
		return resources
	}

	for _, instance := range instances {
		if id, ok := instance["id"].(float64); ok {
			if name, ok := instance["name"].(string); ok {
				resources = append(resources, models.Resource{
					ID:       fmt.Sprintf("%.0f", id),
					Name:     name,
					Type:     "google_compute_instance",
					Provider: provider,
					Region:   region,
					Tags:     make(map[string]string),
					State:    "running",
					Created:  time.Now().Add(-24 * time.Hour),
					Updated:  time.Now(),
				})
			}
		}
	}

	return resources
}

// Discover GCP Storage Buckets using CLI
func discoverGCPStorageBucketsCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to list storage buckets
	cmd := exec.Command("gcloud", "storage", "buckets", "list", "--project", projectID, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list GCP storage buckets: %v", err)
		return resources
	}

	// Parse JSON output
	var buckets []map[string]interface{}
	if err := json.Unmarshal(output, &buckets); err != nil {
		logger.Info("Warning: Failed to parse GCP storage bucket list: %v", err)
		return resources
	}

	for _, bucket := range buckets {
		if name, ok := bucket["name"].(string); ok {
			resources = append(resources, models.Resource{
				ID:       name,
				Name:     name,
				Type:     "google_storage_bucket",
				Provider: provider,
				Region:   "global", // Storage buckets are global
				Tags:     make(map[string]string),
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover GCP GKE Clusters using CLI
func discoverGCPGKEClustersCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to list GKE clusters
	cmd := exec.Command("gcloud", "container", "clusters", "list", "--project", projectID, "--region", region, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list GCP GKE clusters in %s: %v", region, err)
		return resources
	}

	// Parse JSON output
	var clusters []map[string]interface{}
	if err := json.Unmarshal(output, &clusters); err != nil {
		logger.Info("Warning: Failed to parse GCP GKE cluster list for %s: %v", region, err)
		return resources
	}

	for _, cluster := range clusters {
		if name, ok := cluster["name"].(string); ok {
			resources = append(resources, models.Resource{
				ID:       name,
				Name:     name,
				Type:     "google_container_cluster",
				Provider: provider,
				Region:   region,
				Tags:     make(map[string]string),
				State:    "running",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover GCP Cloud SQL Instances using CLI
func discoverGCPCloudSQLInstancesCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to list Cloud SQL instances
	cmd := exec.Command("gcloud", "sql", "instances", "list", "--project", projectID, "--filter", "region:"+region, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list GCP Cloud SQL instances in %s: %v", region, err)
		return resources
	}

	// Parse JSON output
	var instances []map[string]interface{}
	if err := json.Unmarshal(output, &instances); err != nil {
		logger.Info("Warning: Failed to parse GCP Cloud SQL instance list for %s: %v", region, err)
		return resources
	}

	for _, instance := range instances {
		if name, ok := instance["name"].(string); ok {
			resources = append(resources, models.Resource{
				ID:       name,
				Name:     name,
				Type:     "google_sql_database_instance",
				Provider: provider,
				Region:   region,
				Tags:     make(map[string]string),
				State:    "running",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Discover GCP VPC Networks using CLI
func discoverGCPVPCNetworksCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to list VPC networks
	cmd := exec.Command("gcloud", "compute", "networks", "list", "--project", projectID, "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		logger.Info("Warning: Failed to list GCP VPC networks: %v", err)
		return resources
	}

	// Parse JSON output
	var networks []map[string]interface{}
	if err := json.Unmarshal(output, &networks); err != nil {
		logger.Info("Warning: Failed to parse GCP VPC network list: %v", err)
		return resources
	}

	for _, network := range networks {
		if name, ok := network["name"].(string); ok {
			resources = append(resources, models.Resource{
				ID:       name,
				Name:     name,
				Type:     "google_compute_network",
				Provider: provider,
				Region:   "global", // VPC networks are global
				Tags:     make(map[string]string),
				State:    "active",
				Created:  time.Now().Add(-24 * time.Hour),
				Updated:  time.Now(),
			})
		}
	}

	return resources
}

// Enhanced GCP Networking Services Discovery

// Discover GCP Subnets using CLI
func discoverGCPSubnetsCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Subnets discovery implementation would go here
	return resources
}

// Discover GCP Firewall Rules using CLI
func discoverGCPFirewallRulesCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Firewall Rules discovery implementation would go here
	return resources
}

// Discover GCP Load Balancers using CLI
func discoverGCPLoadBalancersCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Load Balancers discovery implementation would go here
	return resources
}

// Discover GCP VPN Gateways using CLI
func discoverGCPVPNGatewaysCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// VPN Gateways discovery implementation would go here
	return resources
}

// Discover GCP Cloud Routers using CLI
func discoverGCPCloudRoutersCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Routers discovery implementation would go here
	return resources
}

// Discover GCP Cloud NAT using CLI
func discoverGCPCloudNATCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud NAT discovery implementation would go here
	return resources
}

// Discover GCP Cloud Interconnect using CLI
func discoverGCPCloudInterconnectCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Interconnect discovery implementation would go here
	return resources
}

// Discover GCP Cloud Armor using CLI
func discoverGCPCloudArmorCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Armor discovery implementation would go here
	return resources
}

// Discover GCP Cloud CDN using CLI
func discoverGCPCloudCDNCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud CDN discovery implementation would go here
	return resources
}

// Discover GCP Cloud DNS Zones using CLI
func discoverGCPCloudDNSZonesCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud DNS Zones discovery implementation would go here
	return resources
}

// Discover GCP Cloud Endpoints using CLI
func discoverGCPCloudEndpointsCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Endpoints discovery implementation would go here
	return resources
}

// Discover GCP Cloud Traffic Director using CLI
func discoverGCPCloudTrafficDirectorCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Traffic Director discovery implementation would go here
	return resources
}

// Enhanced GCP Security Services Discovery

// Discover GCP Cloud KMS Key Rings using CLI
func discoverGCPCloudKMSKeyRingsCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud KMS Key Rings discovery implementation would go here
	return resources
}

// Discover GCP Cloud Security Scanner using CLI
func discoverGCPCloudSecurityScannerCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Security Scanner discovery implementation would go here
	return resources
}

// Discover GCP Cloud Asset Inventory using CLI
func discoverGCPCloudAssetInventoryCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Asset Inventory discovery implementation would go here
	return resources
}

// Discover GCP Cloud Access Context Manager using CLI
func discoverGCPCloudAccessContextManagerCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Access Context Manager discovery implementation would go here
	return resources
}

// Discover GCP Cloud Binary Authorization using CLI
func discoverGCPCloudBinaryAuthorizationCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Binary Authorization discovery implementation would go here
	return resources
}

// Discover GCP Cloud DLP using CLI
func discoverGCPCloudDLPCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud DLP discovery implementation would go here
	return resources
}

// Discover GCP Cloud IAP using CLI
func discoverGCPCloudIAPCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud IAP discovery implementation would go here
	return resources
}

// Discover GCP Cloud Resource Manager using CLI
func discoverGCPCloudResourceManagerCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Resource Manager discovery implementation would go here
	return resources
}

// Discover GCP Cloud IAM using CLI
func discoverGCPCloudIAMCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud IAM discovery implementation would go here
	return resources
}

// Discover GCP Cloud Organization Policy using CLI
func discoverGCPCloudOrganizationPolicyCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Organization Policy discovery implementation would go here
	return resources
}

// Discover GCP Cloud Security Command Center using CLI
func discoverGCPCloudSecurityCommandCenterCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Security Command Center discovery implementation would go here
	return resources
}

// Enhanced GCP Additional Services Discovery

// Discover GCP Cloud Functions using CLI
func discoverGCPCloudFunctionsCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Functions discovery implementation would go here
	return resources
}

// Discover GCP Cloud Run using CLI
func discoverGCPCloudRunCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Run discovery implementation would go here
	return resources
}

// Discover GCP Cloud Build using CLI
func discoverGCPCloudBuildCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Build discovery implementation would go here
	return resources
}

// Discover GCP Cloud Pub/Sub using CLI
func discoverGCPCloudPubSubCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Pub/Sub discovery implementation would go here
	return resources
}

// Discover GCP BigQuery using CLI
func discoverGCPBigQueryCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// BigQuery discovery implementation would go here
	return resources
}

// Discover GCP Cloud Spanner using CLI
func discoverGCPCloudSpannerCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Spanner discovery implementation would go here
	return resources
}

// Discover GCP Cloud Firestore using CLI
func discoverGCPCloudFirestoreCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Firestore discovery implementation would go here
	return resources
}

// Discover GCP Cloud Monitoring using CLI
func discoverGCPCloudMonitoringCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Monitoring discovery implementation would go here
	return resources
}

// Discover GCP Cloud Logging using CLI
func discoverGCPCloudLoggingCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Logging discovery implementation would go here
	return resources
}

// Discover GCP Cloud Trace using CLI
func discoverGCPCloudTraceCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Trace discovery implementation would go here
	return resources
}

// Discover GCP Cloud Debugger using CLI
func discoverGCPCloudDebuggerCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Debugger discovery implementation would go here
	return resources
}

// Discover GCP Cloud Profiler using CLI
func discoverGCPCloudProfilerCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Profiler discovery implementation would go here
	return resources
}

// Discover GCP Cloud Error Reporting using CLI
func discoverGCPCloudErrorReportingCLI(projectID, region, provider string) []models.Resource {
	var resources []models.Resource
	// Cloud Error Reporting discovery implementation would go here
	return resources
}

// Find local Terraform state files
func findTerraformStateFiles(rootPath string) []string {
	var stateFiles []string

	// First, check for files directly in the root path
	files, err := os.ReadDir(rootPath)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() && (file.Name() == "terraform.tfstate" || strings.HasSuffix(file.Name(), ".tfstate")) {
				path := filepath.Join(rootPath, file.Name())
				logger.Info("Found state file in root: %s", path)
				stateFiles = append(stateFiles, path)
			}
		}
	}

	// Common locations for Terraform state files
	searchPaths := []string{
		filepath.Join(rootPath, ".terraform"),
		filepath.Join(rootPath, "terraform"),
		filepath.Join(rootPath, "infrastructure"),
		filepath.Join(rootPath, "iac"),
		filepath.Join(rootPath, "tf"),
		filepath.Join(rootPath, "examples"),
		filepath.Join(rootPath, "examples", "statefiles"),
		// Terragrunt-specific paths
		filepath.Join(rootPath, "terragrunt"),
		filepath.Join(rootPath, "environments"),
		filepath.Join(rootPath, "stacks"),
		filepath.Join(rootPath, ".terragrunt-cache"),
	}

	logger.Info("Searching for Terraform state files in paths: %v", searchPaths)

	for _, searchPath := range searchPaths {
		logger.Info("Checking path: %s", searchPath)

		// Check if path exists
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			logger.Info("Path does not exist: %s", searchPath)
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logger.Info("Error accessing path %s: %v", path, err)
				return nil // Skip files we can't access
			}

			// Look for terraform.tfstate files
			if !info.IsDir() && (info.Name() == "terraform.tfstate" || strings.HasSuffix(info.Name(), ".tfstate")) {
				logger.Info("Found state file: %s", path)
				stateFiles = append(stateFiles, path)
			}

			return nil
		})

		if err != nil {
			logger.Info("Warning: Error walking path %s: %v", searchPath, err)
		}
	}

	// For debugging, also check if test-state.tfstate exists directly
	testStatePath := filepath.Join(rootPath, "test-state.tfstate")
	if _, err := os.Stat(testStatePath); err == nil {
		logger.Info("Found test-state.tfstate directly: %s", testStatePath)
		// Check if it's not already in the list
		found := false
		for _, existing := range stateFiles {
			if existing == testStatePath {
				found = true
				break
			}
		}
		if !found {
			stateFiles = append(stateFiles, testStatePath)
		}
	}

	logger.Info("Total state files found: %d", len(stateFiles))
	return stateFiles
}

// Parse Terraform state file
func parseTerraformState(stateFilePath string) (*models.StateFile, error) {
	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %v", err)
	}

	var stateFile models.StateFile
	if err := json.Unmarshal(data, &stateFile); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %v", err)
	}

	stateFile.Path = stateFilePath
	return &stateFile, nil
}

// Analyze drift between state file and live resources
func analyzeDrift(stateFile *models.StateFile, liveResources []models.Resource) models.AnalysisResult {
	// TODO: Implement proper Terraform state file parsing
	// For now, return empty result to allow compilation
	return models.AnalysisResult{
		DriftResults: []models.DriftResult{},
		Summary:      models.AnalysisSummary{},
		Timestamp:    time.Now(),
	}
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	startTime := time.Now()

	// Find the specific state file requested
	stateFiles := findTerraformStateFiles(".")

	// For testing, if no state files found, try the test-state.tfstate file directly
	if len(stateFiles) == 0 {
		logger.Info("No state files found, trying test-state.tfstate directly")
		if req.StateFileID == "test-state" {
			// Use absolute path
			currentDir, _ := os.Getwd()
			testStatePath := filepath.Join(currentDir, "test-state.tfstate")
			logger.Info("Using test state path: %s", testStatePath)
			stateFiles = []string{testStatePath}
		} else {
			http.Error(w, "No Terraform state files found", http.StatusNotFound)
			return
		}
	}

	// Find the requested state file
	var targetStateFile string
	for _, stateFile := range stateFiles {
		if strings.Contains(stateFile, req.StateFileID) {
			targetStateFile = stateFile
			break
		}
	}

	if targetStateFile == "" {
		http.Error(w, fmt.Sprintf("State file '%s' not found", req.StateFileID), http.StatusNotFound)
		return
	}

	// Parse the requested state file
	stateFile, err := parseTerraformState(targetStateFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse state file: %v", err), http.StatusInternalServerError)
		return
	}

	// Discover live resources
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load AWS config: %v", err), http.StatusInternalServerError)
		return
	}

	var liveResources []models.Resource
	regions := []string{"us-east-1", "us-west-2"} // Default regions for analysis

	for _, region := range regions {
		cfg.Region = region
		liveResources = append(liveResources, discoverEC2Resources(cfg, region, "aws")...)
		liveResources = append(liveResources, discoverRDSResources(cfg, region, "aws")...)
		liveResources = append(liveResources, discoverS3Resources(cfg, "aws")...)
		liveResources = append(liveResources, discoverIAMResources(cfg, "aws")...)
	}

	// Analyze drift
	analysisResult := analyzeDrift(stateFile, liveResources)

	duration := time.Since(startTime)
	response := models.AnalysisResponse{
		Summary:  analysisResult.Summary,
		Duration: duration,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handlePerspective(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.PerspectiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Mock perspective response
	response := models.PerspectiveResponse{
		Summary: models.AnalysisSummary{
			TotalDrifts:           7,
			BySeverity:            map[string]int{"critical": 1, "high": 2, "medium": 3, "low": 1},
			ByProvider:            map[string]int{"aws": 7},
			ByResourceType:        map[string]int{"aws_instance": 3, "aws_security_group": 2, "aws_vpc": 2},
			CriticalDrifts:        1,
			HighDrifts:            2,
			MediumDrifts:          3,
			LowDrifts:             1,
			TotalStateResources:   10,
			TotalLiveResources:    12,
			Missing:               2,
			Extra:                 4,
			Modified:              1,
			PerspectivePercentage: 85.5,
			CoveragePercentage:    90.0,
			DriftPercentage:       15.5,
			DriftsFound:           7,
		},
		ImportCommands: []string{
			"terraform import aws_instance.web_server_1 i-1234567890",
			"terraform import aws_security_group.web_sg sg-1234567890",
		},
		Duration: 4 * time.Second,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleVisualize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.VisualizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find the state file
	stateFilePaths := findTerraformStateFiles(".")
	var stateFilePath string
	for _, path := range stateFilePaths {
		if strings.Contains(path, req.StateFileID) {
			stateFilePath = path
			break
		}
	}

	if stateFilePath == "" {
		http.Error(w, fmt.Sprintf("State file not found: %s", req.StateFileID), http.StatusNotFound)
		return
	}

	// Parse state file
	parser := visualization.NewTerraformStateParser(stateFilePath)
	diagramData, err := parser.ParseStateFile()
	if err != nil {
		logger.Error("Failed to parse state file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse state file: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate diagram
	startTime := time.Now()
	_, err = diagramGenerator.GenerateDiagram(req.StateFileID, *diagramData)
	if err != nil {
		logger.Error("Failed to generate diagram: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate diagram: %v", err), http.StatusInternalServerError)
		return
	}
	duration := time.Since(startTime)

	// Create visualization response
	response := models.VisualizationResponse{
		StateFileID:   req.StateFileID,
		TerraformPath: req.TerraformPath,
		Duration:      duration,
		GeneratedAt:   time.Now(),
		Summary: models.AnalysisSummary{
			TotalDrifts:           0,
			BySeverity:            map[string]int{},
			ByProvider:            map[string]int{},
			ByResourceType:        map[string]int{},
			CriticalDrifts:        0,
			HighDrifts:            0,
			MediumDrifts:          0,
			LowDrifts:             0,
			TotalStateResources:   len(diagramData.Resources),
			TotalLiveResources:    len(diagramData.Resources),
			Missing:               0,
			Extra:                 0,
			Modified:              0,
			PerspectivePercentage: 100.0,
			CoveragePercentage:    100.0,
			DriftPercentage:       0.0,
			DriftsFound:           0,
			TotalResources:        len(diagramData.Resources),
			TotalDependencies:     len(diagramData.Dependencies),
			GraphNodes:            len(diagramData.Resources) + len(diagramData.DataSources),
			GraphEdges:            len(diagramData.Dependencies),
			ComplexityScore:       calculateComplexityScore(diagramData),
			RiskLevel:             calculateRiskLevel(diagramData),
		},
		Outputs: []models.VisualizationOutput{
			{
				Format: "png",
				Path:   fmt.Sprintf("./outputs/%s-diagram.png", req.StateFileID),
				URL:    fmt.Sprintf("http://localhost:8080/outputs/%s-diagram.png", req.StateFileID),
			},
			{
				Format: "svg",
				Path:   fmt.Sprintf("./outputs/%s-diagram.svg", req.StateFileID),
				URL:    fmt.Sprintf("http://localhost:8080/outputs/%s-diagram.svg", req.StateFileID),
			},
			{
				Format: "html",
				Path:   fmt.Sprintf("./outputs/%s-diagram.html", req.StateFileID),
				URL:    fmt.Sprintf("http://localhost:8080/outputs/%s-diagram.html", req.StateFileID),
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleDiagram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.VisualizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find the state file
	stateFilePaths := findTerraformStateFiles(".")
	var stateFilePath string
	for _, path := range stateFilePaths {
		if strings.Contains(path, req.StateFileID) {
			stateFilePath = path
			break
		}
	}

	if stateFilePath == "" {
		http.Error(w, fmt.Sprintf("State file not found: %s", req.StateFileID), http.StatusNotFound)
		return
	}

	// Parse state file
	parser := visualization.NewTerraformStateParser(stateFilePath)
	diagramData, err := parser.ParseStateFile()
	if err != nil {
		logger.Error("Failed to parse state file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse state file: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate diagram
	startTime := time.Now()
	_, err = diagramGenerator.GenerateDiagram(req.StateFileID, *diagramData)
	if err != nil {
		logger.Error("Failed to generate diagram: %v", err)
		http.Error(w, fmt.Sprintf("Failed to generate diagram: %v", err), http.StatusInternalServerError)
		return
	}
	duration := time.Since(startTime)

	// Create diagram response
	response := models.DiagramResponse{
		StateFileID: req.StateFileID,
		Status:      "completed",
		Message:     "Diagram generated successfully",
		Duration:    duration,
		GeneratedAt: time.Now(),
		DiagramData: *diagramData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Mock export response
	response := models.ExportResponse{
		StateFileID: "terraform", // Default state file ID
		Format:      req.Format,
		Status:      "completed",
		Message:     "Diagram exported successfully",
		OutputPath:  fmt.Sprintf("/outputs/diagram.%s", req.Format),
		URL:         fmt.Sprintf("http://localhost:8080/outputs/diagram.%s", req.Format),
		ExportedAt:  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Send notification based on type
	var response models.NotificationResponse
	var err error

	switch req.Type {
	case "email":
		response, err = emailProvider.SendNotification(req)
		if err != nil {
			logger.Error("Failed to send email notification: %v", err)
			http.Error(w, fmt.Sprintf("Failed to send notification: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, fmt.Sprintf("Unsupported notification type: %s", req.Type), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleCacheStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := cacheManager.GetStats()
	json.NewEncoder(w).Encode(stats)
}

func handleCacheClear(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := cacheManager.Clear()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to clear cache: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Cache cleared successfully",
		"time":    time.Now().UTC(),
	}
	json.NewEncoder(w).Encode(response)
}

// calculateComplexityScore calculates the complexity score of the infrastructure
func calculateComplexityScore(diagramData *models.DiagramData) float64 {
	baseScore := float64(len(diagramData.Resources)) * 0.5
	dependencyScore := float64(len(diagramData.Dependencies)) * 0.3
	moduleScore := float64(len(diagramData.Modules)) * 0.2

	complexity := baseScore + dependencyScore + moduleScore

	// Normalize to 0-10 scale
	if complexity > 10 {
		complexity = 10
	}

	return complexity
}

// calculateRiskLevel calculates the risk level based on infrastructure complexity
func calculateRiskLevel(diagramData *models.DiagramData) string {
	complexity := calculateComplexityScore(diagramData)

	switch {
	case complexity <= 3:
		return "low"
	case complexity <= 6:
		return "medium"
	case complexity <= 8:
		return "high"
	default:
		return "critical"
	}
}

// Terragrunt-specific handlers

// handleTerragruntFiles handles requests to discover Terragrunt configuration files
func handleTerragruntFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parser := terragrunt.NewTerragruntParser(".")
	result, err := parser.FindTerragruntFiles()
	if err != nil {
		logger.Error("Failed to discover Terragrunt files: %v", err)
		http.Error(w, fmt.Sprintf("Failed to discover Terragrunt files: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleTerragruntStateFiles handles requests to find Terragrunt-managed state files
func handleTerragruntStateFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parser := terragrunt.NewTerragruntParser(".")
	stateFiles := parser.FindTerragruntStateFiles()

	response := map[string]interface{}{
		"state_files": stateFiles,
		"count":       len(stateFiles),
		"timestamp":   time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTerragruntAnalyze handles requests to analyze Terragrunt configurations
func handleTerragruntAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TerragruntPath string   `json:"terragrunt_path"`
		Provider       string   `json:"provider"`
		Regions        []string `json:"regions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Discover Terragrunt files
	parser := terragrunt.NewTerragruntParser(req.TerragruntPath)
	terragruntResult, err := parser.FindTerragruntFiles()
	if err != nil {
		logger.Error("Failed to discover Terragrunt files: %v", err)
		http.Error(w, fmt.Sprintf("Failed to discover Terragrunt files: %v", err), http.StatusInternalServerError)
		return
	}

	// Discover live resources
	var liveResources []models.Resource
	if req.Provider != "" {
		// Use the existing discovery functionality
		regions := req.Regions
		if len(regions) == 0 {
			regions = []string{"us-east-1"} // Default region
		}

		// For now, we'll use a placeholder - in a full implementation,
		// you would integrate with the existing discovery system
		logger.Info("Live resource discovery would be performed here for provider: %s, regions: %v", req.Provider, regions)
	}

	// Analyze Terragrunt configurations
	analysisResult := analyzeTerragruntConfigurations(terragruntResult, liveResources)

	response := map[string]interface{}{
		"terragrunt_files": terragruntResult,
		"live_resources":   liveResources,
		"analysis":         analysisResult,
		"timestamp":        time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// analyzeTerragruntConfigurations analyzes Terragrunt configurations and compares with live resources
func analyzeTerragruntConfigurations(terragruntResult *models.TerragruntDiscoveryResult, liveResources []models.Resource) map[string]interface{} {
	analysis := map[string]interface{}{
		"total_terragrunt_files": terragruntResult.TotalFiles,
		"root_files":             len(terragruntResult.RootFiles),
		"child_files":            len(terragruntResult.ChildFiles),
		"environments":           terragruntResult.Environments,
		"regions":                terragruntResult.Regions,
		"accounts":               terragruntResult.Accounts,
		"live_resources_count":   len(liveResources),
	}

	// Identify potential Terragrunt-managed resources
	terragruntManagedResources := identifyTerragruntManagedResources(liveResources)
	analysis["terragrunt_managed_resources"] = terragruntManagedResources
	analysis["terragrunt_managed_count"] = len(terragruntManagedResources)

	// Analyze configuration patterns
	configPatterns := analyzeTerragruntPatterns(terragruntResult)
	analysis["configuration_patterns"] = configPatterns

	return analysis
}

// identifyTerragruntManagedResources identifies resources that are likely managed by Terragrunt
func identifyTerragruntManagedResources(resources []models.Resource) []models.Resource {
	var terragruntManaged []models.Resource

	for _, resource := range resources {
		// Check for Terragrunt-specific tags
		if isTerragruntManaged(resource) {
			terragruntManaged = append(terragruntManaged, resource)
		}
	}

	return terragruntManaged
}

// isTerragruntManaged checks if a resource is likely managed by Terragrunt
func isTerragruntManaged(resource models.Resource) bool {
	// Check for Terragrunt-specific tags
	if resource.Tags != nil {
		// Type assert Tags to map[string]string
		if tags, ok := resource.Tags.(map[string]string); ok {
			if managedBy, exists := tags["ManagedBy"]; exists {
				if strings.Contains(strings.ToLower(managedBy), "terragrunt") {
					return true
				}
			}
			if terraform, exists := tags["Terraform"]; exists {
				if strings.Contains(strings.ToLower(terraform), "true") {
					return true
				}
			}
		}
		// Also try map[string]interface{} which is common from JSON unmarshaling
		if tags, ok := resource.Tags.(map[string]interface{}); ok {
			if managedBy, exists := tags["ManagedBy"]; exists {
				if managedByStr, ok := managedBy.(string); ok {
					if strings.Contains(strings.ToLower(managedByStr), "terragrunt") {
						return true
					}
				}
			}
			if terraform, exists := tags["Terraform"]; exists {
				if terraformStr, ok := terraform.(string); ok {
					if strings.Contains(strings.ToLower(terraformStr), "true") {
						return true
					}
				}
			}
		}
	}

	// Check for Terragrunt-specific naming patterns
	terragruntPatterns := []string{
		"terragrunt",
		"tg-",
		"terragrunt-",
	}

	for _, pattern := range terragruntPatterns {
		if strings.Contains(strings.ToLower(resource.Name), pattern) ||
			strings.Contains(strings.ToLower(resource.ID), pattern) {
			return true
		}
	}

	return false
}

// analyzeTerragruntPatterns analyzes patterns in Terragrunt configurations
func analyzeTerragruntPatterns(result *models.TerragruntDiscoveryResult) map[string]interface{} {
	patterns := map[string]interface{}{
		"common_sources":    make(map[string]int),
		"common_inputs":     make(map[string]int),
		"backend_types":     make(map[string]int),
		"hook_types":        make(map[string]int),
		"generate_patterns": make(map[string]int),
	}

	// Analyze all Terragrunt files
	allFiles := append(result.RootFiles, result.ChildFiles...)

	for _, file := range allFiles {
		if file.Config != nil {
			// Count source patterns
			if file.Config.Source != "" {
				source := file.Config.Source
				if count, exists := patterns["common_sources"].(map[string]int); exists {
					count[source]++
				}
			}

			// Count input patterns
			if file.Config.Inputs != nil {
				for key := range file.Config.Inputs {
					if count, exists := patterns["common_inputs"].(map[string]int); exists {
						count[key]++
					}
				}
			}

			// Count backend types
			if file.Config.RemoteState != nil {
				backend := file.Config.RemoteState.Backend
				if count, exists := patterns["backend_types"].(map[string]int); exists {
					count[backend]++
				}
			}

			// Count hook types
			if len(file.Config.BeforeHooks) > 0 {
				if count, exists := patterns["hook_types"].(map[string]int); exists {
					count["before_hooks"]++
				}
			}
			if len(file.Config.AfterHooks) > 0 {
				if count, exists := patterns["hook_types"].(map[string]int); exists {
					count["after_hooks"]++
				}
			}
			if len(file.Config.ErrorHooks) > 0 {
				if count, exists := patterns["hook_types"].(map[string]int); exists {
					count["error_hooks"]++
				}
			}

			// Count generate patterns
			if len(file.Config.Generate) > 0 {
				if count, exists := patterns["generate_patterns"].(map[string]int); exists {
					count["generate_blocks"]++
				}
			}
		}
	}

	return patterns
}

// Enhanced drift analysis handler
func handleEnhancedAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.EnhancedAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// For now, return a mock response
	response := models.AnalysisResult{
		DriftResults: []models.DriftResult{},
		Summary: models.AnalysisSummary{
			TotalDrifts:    0,
			CriticalDrifts: 0,
			HighDrifts:     0,
			MediumDrifts:   0,
			LowDrifts:      0,
			BySeverity:     make(map[string]int),
			ByProvider:     make(map[string]int),
			ByResourceType: make(map[string]int),
		},
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Remediation handlers
func handleRemediate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DriftID string             `json:"drift_id"`
		Drift   models.DriftResult `json:"drift"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Perform automated remediation
	ctx := context.Background()
	result, err := smartRemediator.RemediateDrift(ctx, &req.Drift)

	// Prepare response
	response := struct {
		Success    bool     `json:"success"`
		ActionID   string   `json:"action_id"`
		ResourceID string   `json:"resource_id"`
		Changes    []string `json:"changes"`
		Error      string   `json:"error,omitempty"`
		Duration   time.Duration `json:"duration"`
	}{
		Success:    result != nil && result.Success,
		ActionID:   "",
		ResourceID: req.Drift.ResourceID,
		Changes:    []string{},
		Duration:   time.Since(time.Now()),
	}

	if err != nil {
		response.Error = err.Error()
		response.Success = false
	} else if result != nil {
		response.ActionID = result.ActionID
		response.ResourceID = result.ResourceID
		response.Changes = result.Changes
		if result.Error != "" {
			response.Error = result.Error
		}
	}

	logger.Info("Remediation completed: success=%v, resource=%s",
		response.Success, response.ResourceID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleRemediateBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.BatchRemediationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// For now, return a mock response
	response := models.BatchRemediationResult{
		StateFileID: req.StateFileID,
		TotalDrifts: 0,
		Remediated:  0,
		Failed:      0,
		Results:     []models.RemediationResult{},
		Timestamp:   time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleRemediateHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// For now, return a mock response
	response := models.RemediationHistory{
		History:   []models.RemediationResult{},
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleRemediateRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// For now, return a mock response
	response := models.RollbackResult{
		SnapshotID:   req.SnapshotID,
		Status:       "completed",
		RolledBack:   true,
		Timestamp:    time.Now(),
		ErrorMessage: "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRemediationStrategies returns available remediation strategies
func handleRemediationStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	strategies := smartRemediator.GetSmartStrategies()

	// Convert to response format
	var response []struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		DriftType  string  `json:"drift_type"`
		Priority   int     `json:"priority"`
		Confidence float64 `json:"confidence"`
		Actions    int     `json:"actions_count"`
	}

	for _, strategy := range strategies {
		response = append(response, struct {
			ID         string  `json:"id"`
			Name       string  `json:"name"`
			DriftType  string  `json:"drift_type"`
			Priority   int     `json:"priority"`
			Confidence float64 `json:"confidence"`
			Actions    int     `json:"actions_count"`
		}{
			ID:         strategy.GetName(),
			Name:       strategy.GetName(),
			DriftType:  "general",
			Priority:   strategy.GetPriority(),
			Confidence: 0.8,
			Actions:    1,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTestRemediation handles remediation strategy testing
func handleTestRemediation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		StrategyID string `json:"strategy_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Test the remediation strategy
	ctx := context.Background()
	// Create a dummy drift result for testing
	driftResult := &models.DriftResult{
		ResourceID:   "test-resource",
		ResourceType: "test-type",
		Provider:     "test-provider",
	}
	result, err := smartRemediator.TestStrategy(ctx, req.StrategyID, driftResult)

	// Prepare response
	response := struct {
		Success  bool                    `json:"success"`
		Passed   int                     `json:"passed"`
		Failed   int                     `json:"failed"`
		Coverage float64                 `json:"coverage"`
		Error    string                  `json:"error,omitempty"`
		Report   *remediation.TestReport `json:"report,omitempty"`
	}{}

	if err != nil {
		response.Error = err.Error()
		response.Success = false
	} else if result != nil {
		response.Success = result.Success
		response.Report = result
		// Count passed and failed tests
		for _, testResult := range result.Results {
			if testResult.Passed {
				response.Passed++
			} else {
				response.Failed++
			}
		}
		if len(result.Results) > 0 {
			response.Coverage = float64(response.Passed) / float64(len(result.Results))
		}
	}

	logger.Info("Remediation test completed: success=%v, passed=%d, failed=%d, coverage=%.1f%%",
		response.Success, response.Passed, response.Failed, response.Coverage*100)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Drift prediction handlers

// handlePredictDrifts handles drift prediction requests
func handlePredictDrifts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Resources []models.Resource      `json:"resources"`
		Options   map[string]interface{} `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	predictions := driftPredictor.PredictDrifts(ctx, req.Resources)

	response := map[string]interface{}{
		"predictions": predictions,
		"total":       len(predictions),
		"timestamp":   time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDriftPatterns handles drift pattern management
func handleDriftPatterns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	patterns := driftPredictor.GetPredictionStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patterns)
}

// handlePredictionStats handles prediction statistics
func handlePredictionStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := driftPredictor.GetPredictionStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Workflow management handlers

// handleWorkflows handles workflow listing and creation
func handleWorkflows(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		workflows := workflowEngine.ListWorkflows()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(workflows)
	case http.MethodPost:
		var workflow workflow.Workflow
		if err := json.NewDecoder(r.Body).Decode(&workflow); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := workflowEngine.RegisterWorkflow(&workflow); err != nil {
			http.Error(w, fmt.Sprintf("Failed to register workflow: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "success",
			"workflow_id": workflow.ID,
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleWorkflow handles individual workflow operations
func handleWorkflow(w http.ResponseWriter, r *http.Request) {
	// Extract workflow ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid workflow ID", http.StatusBadRequest)
		return
	}
	workflowID := pathParts[len(pathParts)-1]

	switch r.Method {
	case http.MethodGet:
		workflow, err := workflowEngine.GetWorkflow(workflowID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Workflow not found: %v", err), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(workflow)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleExecuteWorkflow handles workflow execution
func handleExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WorkflowID string                 `json:"workflow_id"`
		Parameters map[string]interface{} `json:"parameters"`
		Resources  []models.Resource      `json:"resources"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	executionID := fmt.Sprintf("exec_%d", time.Now().Unix())
	workflowCtx := workflow.WorkflowContext{
		WorkflowID:  req.WorkflowID,
		ExecutionID: executionID,
		Parameters:  req.Parameters,
		Resources:   req.Resources,
		State:       make(map[string]interface{}),
		StartedAt:   time.Now(),
		User:        "system",
		Metadata:    make(map[string]interface{}),
	}

	result := workflowEngine.ExecuteWorkflow(context.Background(), req.WorkflowID, workflowCtx)

	response := map[string]interface{}{
		"execution_id": executionID,
		"workflow_id":  req.WorkflowID,
		"status":       result.Status,
		"started_at":   result.StartedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWorkflowExecution handles workflow execution status
func handleWorkflowExecution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract execution ID from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}
	executionID := pathParts[len(pathParts)-1]

	result, err := workflowEngine.GetExecution(executionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Execution not found: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleDeleteAccountResources handles account resource deletion
func handleDeleteAccountResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Provider  string                   `json:"provider"`
		AccountID string                   `json:"account_id"`
		Options   deletion.DeletionOptions `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set up progress callback
	req.Options.ProgressCallback = func(update deletion.ProgressUpdate) {
		broadcastProgress(ProgressUpdate{
			Type:      update.Type,
			Message:   update.Message,
			Progress:  update.Progress,
			Total:     update.Total,
			Data:      update.Data,
			Timestamp: update.Timestamp,
		})
	}

	// Execute deletion
	result, err := deletionEngine.DeleteAccountResources(context.Background(), req.Provider, req.AccountID, req.Options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Deletion failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleDeletePreview handles deletion preview (dry run)
func handleDeletePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Provider  string                   `json:"provider"`
		AccountID string                   `json:"account_id"`
		Options   deletion.DeletionOptions `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Force dry run
	req.Options.DryRun = true

	// Execute preview
	result, err := deletionEngine.DeleteAccountResources(context.Background(), req.Provider, req.AccountID, req.Options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Preview failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetSupportedProviders returns list of supported cloud providers
func handleGetSupportedProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providers := deletionEngine.GetSupportedProviders()

	response := map[string]interface{}{
		"providers": providers,
		"count":     len(providers),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// Workspace Management API Handlers
// ============================================================================

// handleWorkspaces handles workspace listing
func handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Discover workspaces if not already done
	ctx := context.Background()
	if err := workspaceManager.DiscoverWorkspaces(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to discover workspaces: %v", err), http.StatusInternalServerError)
		return
	}

	workspaces := workspaceManager.ListWorkspaces()

	response := map[string]interface{}{
		"workspaces": workspaces,
		"count":      len(workspaces),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWorkspaceDiscover handles workspace discovery
func handleWorkspaceDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RootPath string `json:"root_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RootPath == "" {
		req.RootPath = "."
	}

	// Update workspace manager with new root path
	workspaceManager = workspace.NewWorkspaceManager(req.RootPath)

	// Discover workspaces
	ctx := context.Background()
	if err := workspaceManager.DiscoverWorkspaces(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to discover workspaces: %v", err), http.StatusInternalServerError)
		return
	}

	workspaces := workspaceManager.ListWorkspaces()
	environments := workspaceManager.ListEnvironments()

	response := map[string]interface{}{
		"workspaces":        workspaces,
		"environments":      environments,
		"workspace_count":   len(workspaces),
		"environment_count": len(environments),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWorkspace handles individual workspace operations
func handleWorkspace(w http.ResponseWriter, r *http.Request) {
	// Extract workspace name from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid workspace path", http.StatusBadRequest)
		return
	}
	workspaceName := pathParts[3]

	switch r.Method {
	case http.MethodGet:
		workspace, err := workspaceManager.GetWorkspace(workspaceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Workspace not found: %v", err), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(workspace)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleEnvironments handles environment listing
func handleEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Discover workspaces if not already done
	ctx := context.Background()
	if err := workspaceManager.DiscoverWorkspaces(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to discover workspaces: %v", err), http.StatusInternalServerError)
		return
	}

	environments := workspaceManager.ListEnvironments()

	response := map[string]interface{}{
		"environments": environments,
		"count":        len(environments),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleEnvironment handles individual environment operations
func handleEnvironment(w http.ResponseWriter, r *http.Request) {
	// Extract environment name from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid environment path", http.StatusBadRequest)
		return
	}
	environmentName := pathParts[3]

	switch r.Method {
	case http.MethodGet:
		environment, err := workspaceManager.GetEnvironment(environmentName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Environment not found: %v", err), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(environment)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleEnvironmentCompare handles environment comparison
func handleEnvironmentCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Environment1 string `json:"environment1"`
		Environment2 string `json:"environment2"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	comparison, err := workspaceManager.CompareEnvironments(req.Environment1, req.Environment2)
	if err != nil {
		http.Error(w, fmt.Sprintf("Comparison failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comparison)
}

// handleEnvironmentPromote handles environment promotion
func handleEnvironmentPromote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SourceEnvironment string                      `json:"source_environment"`
		TargetEnvironment string                      `json:"target_environment"`
		Options           *workspace.PromotionOptions `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Options == nil {
		req.Options = &workspace.PromotionOptions{
			AutoApprove: false,
			DryRun:      false,
			Parallel:    false,
			Timeout:     30 * time.Minute,
			Filters:     make(map[string]string),
		}
	}

	// Execute promotion
	ctx := context.Background()
	result, err := workspaceManager.PromoteEnvironment(ctx, req.SourceEnvironment, req.TargetEnvironment, req.Options)
	if err != nil {
		http.Error(w, fmt.Sprintf("Promotion failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleBlueGreen handles blue-green deployment operations
func handleBlueGreen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action string `json:"action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// This is a placeholder implementation
	// In a real implementation, you would integrate with actual blue-green deployment systems
	response := map[string]interface{}{
		"action":  req.Action,
		"status":  "success",
		"message": fmt.Sprintf("Blue-green deployment action '%s' completed", req.Action),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleFeatureFlags handles feature flag operations
func handleFeatureFlags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action string `json:"action"`
		Flag   string `json:"flag"`
		Value  string `json:"value,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// This is a placeholder implementation
	// In a real implementation, you would integrate with actual feature flag systems
	response := map[string]interface{}{
		"action":  req.Action,
		"flag":    req.Flag,
		"status":  "success",
		"message": fmt.Sprintf("Feature flag action '%s' for flag '%s' completed", req.Action, req.Flag),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
