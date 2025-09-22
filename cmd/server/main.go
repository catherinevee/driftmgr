package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/catherinevee/driftmgr/internal/config"
)

func main() {
	var (
		port       = flag.String("port", "8080", "Server port")
		host       = flag.String("host", "0.0.0.0", "Server host")
		configPath = flag.String("config", "", "Path to configuration file")
		// tlsCert    = flag.String("tls-cert", "", "Path to TLS certificate") // unused for now
		// tlsKey     = flag.String("tls-key", "", "Path to TLS key") // unused for now
		// jwtSecret  = flag.String("jwt-secret", "", "JWT secret for authentication") // unused for now
		enableAuth = flag.Bool("auth", false, "Enable authentication")
	)
	flag.Parse()

	fmt.Printf("Starting DriftMgr Server v3.0.0\n")
	fmt.Printf("Listening on %s:%s\n", *host, *port)

	// Create server configuration
	portInt, _ := strconv.Atoi(*port)
	apiConfig := &api.Config{
		Host:        *host,
		Port:        portInt,
		AuthEnabled: *enableAuth,
	}

	// Load configuration file if provided
	if *configPath != "" {
		log.Printf("Loading configuration from %s", *configPath)

		// Load configuration using the config manager
		loadedConfig, err := config.LoadConfigFromFile(*configPath)
		if err != nil {
			log.Fatalf("Failed to load configuration: %v", err)
		}

		// Update the config with loaded values
		apiConfig.Host = loadedConfig.Server.Host
		apiConfig.Port = loadedConfig.Server.Port
		apiConfig.AuthEnabled = loadedConfig.Server.AuthEnabled

		log.Printf("Configuration loaded successfully")
	}

	// Create services (empty for now)
	services := &api.Services{}

	// Create API server
	apiServer := api.NewServer(apiConfig, services)

	// Setup routes
	setupRoutes(apiServer)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", *host, *port),
		Handler:      apiServer,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		fmt.Println("Starting HTTP server...")
		err := srv.ListenAndServe()
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
	// Routes are configured in the server itself
	// This is a placeholder for future route customization
	log.Println("Server routes configured")
}
