package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/catherinevee/driftmgr/internal/api"
)

func main() {
	var (
		port       = flag.String("port", "8080", "Server port")
		host       = flag.String("host", "0.0.0.0", "Server host")
		configPath = flag.String("config", "", "Path to configuration file")
		tlsCert    = flag.String("tls-cert", "", "Path to TLS certificate")
		tlsKey     = flag.String("tls-key", "", "Path to TLS key")
		jwtSecret  = flag.String("jwt-secret", "", "JWT secret for authentication")
		enableAuth = flag.Bool("auth", false, "Enable authentication")
	)
	flag.Parse()

	fmt.Printf("Starting DriftMgr Server v3.0.0\n")
	fmt.Printf("Listening on %s:%s\n", *host, *port)

	// Create server configuration
	config := api.ServerConfig{
		Host:       *host,
		Port:       *port,
		EnableAuth: *enableAuth,
		JWTSecret:  *jwtSecret,
		TLSCert:    *tlsCert,
		TLSKey:     *tlsKey,
	}

	// Load configuration file if provided
	if *configPath != "" {
		// TODO: Load configuration from file
		log.Printf("Loading configuration from %s", *configPath)
	}

	// Create API server
	apiServer := api.NewServer(config)

	// Setup routes
	setupRoutes(apiServer)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", *host, *port),
		Handler:      apiServer.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		var err error
		if *tlsCert != "" && *tlsKey != "" {
			fmt.Println("Starting HTTPS server...")
			err = srv.ListenAndServeTLS(*tlsCert, *tlsKey)
		} else {
			fmt.Println("Starting HTTP server...")
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server stopped")
}

func setupRoutes(server *api.Server) {
	// Health check endpoints
	server.RegisterHealthChecks()

	// API v1 routes
	v1 := "/api/v1"

	// Discovery endpoints
	server.RegisterRoute("POST", v1+"/discover", server.HandleDiscovery)
	server.RegisterRoute("GET", v1+"/discover/:id", server.HandleDiscoveryStatus)

	// Drift detection endpoints
	server.RegisterRoute("POST", v1+"/drift/detect", server.HandleDriftDetection)
	server.RegisterRoute("GET", v1+"/drift/:id", server.HandleDriftResults)

	// State management endpoints
	server.RegisterRoute("GET", v1+"/state", server.HandleListStates)
	server.RegisterRoute("POST", v1+"/state/analyze", server.HandleStateAnalysis)
	server.RegisterRoute("POST", v1+"/state/push", server.HandleStatePush)
	server.RegisterRoute("POST", v1+"/state/pull", server.HandleStatePull)

	// Remediation endpoints
	server.RegisterRoute("POST", v1+"/remediate", server.HandleRemediation)
	server.RegisterRoute("GET", v1+"/remediate/:id", server.HandleRemediationStatus)

	// Resource endpoints
	server.RegisterRoute("GET", v1+"/resources", server.HandleListResources)
	server.RegisterRoute("GET", v1+"/resources/:id", server.HandleGetResource)

	// Metrics endpoint
	server.RegisterRoute("GET", "/metrics", server.HandleMetrics)

	// WebSocket for real-time updates
	server.RegisterRoute("GET", "/ws", server.HandleWebSocket)
}
