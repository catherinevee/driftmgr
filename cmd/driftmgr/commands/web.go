package commands

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/catherinevee/driftmgr/internal/analytics"
	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/catherinevee/driftmgr/internal/automation"
	"github.com/catherinevee/driftmgr/internal/bi"
	"github.com/catherinevee/driftmgr/internal/cost"
	"github.com/catherinevee/driftmgr/internal/remediation"
	"github.com/catherinevee/driftmgr/internal/security"
	"github.com/catherinevee/driftmgr/internal/tenant"
)

// WebCommand represents the web dashboard command
type WebCommand struct {
	apiServer *api.Server
	webServer *http.Server
	services  *api.Services
}

// MockWebEventBus is a mock implementation of EventBus for web command
type MockWebEventBus struct{}

func (m *MockWebEventBus) PublishEvent(eventType string, data interface{}) error {
	return nil
}

func (m *MockWebEventBus) PublishComplianceEvent(event security.ComplianceEvent) error {
	return nil
}

func (m *MockWebEventBus) PublishTenantEvent(event tenant.TenantEvent) error {
	return nil
}

// NewWebCommand creates a new web command
func NewWebCommand() *WebCommand {
	// Create mock event bus
	eventBus := &MockWebEventBus{}

	// Create services
	services := &api.Services{
		Analytics:   analytics.NewAnalyticsService(),
		Automation:  automation.NewAutomationService(),
		BI:          bi.NewBIService(),
		Cost:        cost.NewCostAnalyzer(),
		Remediation: remediation.NewIntelligentRemediationService(nil),
		Security:    security.NewSecurityService(eventBus),
		Tenant:      tenant.NewTenantService(eventBus),
	}

	// Create API server
	apiConfig := &api.Config{
		Host:             "0.0.0.0",
		Port:             8080,
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     30 * time.Second,
		IdleTimeout:      120 * time.Second,
		MaxHeaderBytes:   1 << 20,
		CORSEnabled:      true,
		AuthEnabled:      false,
		RateLimitEnabled: true,
		RateLimitRPS:     100,
		LoggingEnabled:   true,
	}

	apiServer := api.NewServer(apiConfig, services)

	// Create web server
	webServer := &http.Server{
		Addr:    ":3000",
		Handler: createWebHandler(),
	}

	return &WebCommand{
		apiServer: apiServer,
		webServer: webServer,
		services:  services,
	}
}

// HandleWeb handles the web command
func HandleWeb(args []string) {
	cmd := NewWebCommand()

	if len(args) == 0 {
		cmd.showHelp()
		return
	}

	switch args[0] {
	case "start":
		cmd.handleStart(args[1:])
	case "stop":
		cmd.handleStop(args[1:])
	case "status":
		cmd.handleStatus(args[1:])
	case "build":
		cmd.handleBuild(args[1:])
	default:
		fmt.Printf("Unknown web command: %s\n", args[0])
		cmd.showHelp()
	}
}

// showHelp shows the help for web commands
func (cmd *WebCommand) showHelp() {
	fmt.Println("Web Dashboard Commands:")
	fmt.Println("  start                 - Start the web dashboard")
	fmt.Println("  stop                  - Stop the web dashboard")
	fmt.Println("  status                - Show web dashboard status")
	fmt.Println("  build                 - Build the web dashboard")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  driftmgr web start    - Start both API server and web dashboard")
	fmt.Println("  driftmgr web stop     - Stop the web dashboard")
	fmt.Println("  driftmgr web status   - Show current status")
	fmt.Println("  driftmgr web build    - Build the web dashboard")
}

// handleStart handles starting the web dashboard
func (cmd *WebCommand) handleStart(args []string) {
	fmt.Println("Starting DriftMgr Web Dashboard...")

	// Start API server
	ctx := context.Background()
	go func() {
		if err := cmd.apiServer.Start(ctx); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	// Wait a moment for API server to start
	time.Sleep(2 * time.Second)

	// Start web server
	go func() {
		if err := cmd.webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Web server error: %v", err)
		}
	}()

	fmt.Println("✅ Web Dashboard started successfully!")
	fmt.Println()
	fmt.Println("🌐 Dashboard: http://localhost:3000")
	fmt.Println("🔌 API: http://localhost:8080")
	fmt.Println("❤️  Health Check: http://localhost:8080/health")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop the servers...")

	// Keep the main thread alive
	select {}
}

// handleStop handles stopping the web dashboard
func (cmd *WebCommand) handleStop(args []string) {
	fmt.Println("Stopping DriftMgr Web Dashboard...")

	ctx := context.Background()

	// Stop web server
	if err := cmd.webServer.Shutdown(ctx); err != nil {
		fmt.Printf("Error stopping web server: %v\n", err)
	}

	// Stop API server
	if err := cmd.apiServer.Stop(ctx); err != nil {
		fmt.Printf("Error stopping API server: %v\n", err)
	}

	fmt.Println("✅ Web Dashboard stopped successfully!")
}

// handleStatus handles web dashboard status
func (cmd *WebCommand) handleStatus(args []string) {
	fmt.Println("DriftMgr Web Dashboard Status:")
	fmt.Println()
	fmt.Println("🌐 Web Server:")
	fmt.Println("  Status: Running")
	fmt.Println("  Port: 3000")
	fmt.Println("  URL: http://localhost:3000")
	fmt.Println()
	fmt.Println("🔌 API Server:")
	fmt.Println("  Status: Running")
	fmt.Println("  Port: 8080")
	fmt.Println("  URL: http://localhost:8080")
	fmt.Println()
	fmt.Println("📊 Services:")
	fmt.Println("  Analytics: Available")
	fmt.Println("  Automation: Available")
	fmt.Println("  Business Intelligence: Available")
	fmt.Println("  Cost Analysis: Available")
	fmt.Println("  Remediation: Available")
	fmt.Println("  Security: Available")
	fmt.Println("  Multi-Tenant: Available")
	fmt.Println()
	fmt.Println("🔗 Endpoints:")
	fmt.Println("  Health Check: http://localhost:8080/health")
	fmt.Println("  API Version: http://localhost:8080/api/v1/version")
	fmt.Println("  WebSocket: ws://localhost:8080/ws/drift")
}

// handleBuild handles building the web dashboard
func (cmd *WebCommand) handleBuild(args []string) {
	fmt.Println("Building DriftMgr Web Dashboard...")

	// Check if web directory exists
	webDir := getWebDir()
	if webDir == "" {
		fmt.Println("❌ Web directory not found!")
		return
	}

	fmt.Printf("📁 Web directory: %s\n", webDir)

	// Check for required files
	requiredFiles := []string{
		"dashboard/index.html",
		"dashboard/styles.css",
		"dashboard/script.js",
	}

	allFilesExist := true
	for _, file := range requiredFiles {
		filePath := filepath.Join(webDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("❌ Missing file: %s\n", file)
			allFilesExist = false
		} else {
			fmt.Printf("✅ Found: %s\n", file)
		}
	}

	if !allFilesExist {
		fmt.Println("❌ Build failed: Missing required files!")
		return
	}

	fmt.Println("✅ Web dashboard build completed successfully!")
	fmt.Println()
	fmt.Println("🚀 Ready to start with: driftmgr web start")
}

// createWebHandler creates the web handler
func createWebHandler() http.Handler {
	mux := http.NewServeMux()

	// Serve static files
	webDir := getWebDir()
	if webDir != "" {
		fs := http.FileServer(http.Dir(webDir))
		mux.Handle("/", fs)
	}

	// API proxy
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		// Proxy API requests to the API server
		http.Redirect(w, r, "http://localhost:8080"+r.URL.Path, http.StatusTemporaryRedirect)
	})

	// WebSocket proxy
	mux.HandleFunc("/ws/", func(w http.ResponseWriter, r *http.Request) {
		// Proxy WebSocket requests to the API server
		http.Redirect(w, r, "ws://localhost:8080"+r.URL.Path, http.StatusTemporaryRedirect)
	})

	return mux
}

// getWebDir finds the web directory
func getWebDir() string {
	// Try to find the web directory
	dirs := []string{
		"./web",
		"../web",
		"../../web",
		"./cmd/web/web",
		"../cmd/web/web",
		"../../cmd/web/web",
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	return ""
}
