package main

import (
	"context"
	"log"
	"net/http"
	"os"
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

func main() {
	// Create services
	services := &api.Services{
		Analytics:   analytics.NewAnalyticsService(),
		Automation:  automation.NewAutomationService(nil),
		BI:          bi.NewBIService(),
		Cost:        cost.NewCostAnalyzer(),
		Remediation: remediation.NewIntelligentRemediationService(nil),
		Security:    security.NewSecurityService(nil),
		Tenant:      tenant.NewTenantService(nil),
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

	// Create web server for static files
	webServer := &http.Server{
		Addr:    ":3000",
		Handler: createWebHandler(),
	}

	// Start API server
	go func() {
		ctx := context.Background()
		if err := apiServer.Start(ctx); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	// Start web server
	log.Println("Starting DriftMgr Web Dashboard...")
	log.Println("Dashboard: http://localhost:3000")
	log.Println("API: http://localhost:8080")
	log.Println("Health Check: http://localhost:8080/health")

	if err := webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Web server error: %v", err)
	}
}

func createWebHandler() http.Handler {
	mux := http.NewServeMux()

	// Serve static files
	webDir := getWebDir()
	fs := http.FileServer(http.Dir(webDir))
	mux.Handle("/", fs)

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

	// Fallback to current directory
	return "."
}
