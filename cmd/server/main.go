package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/catherinevee/driftmgr/internal/config"
)

func main() {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered: %v", r)
		}
	}()

	var (
		port       = flag.String("port", "8080", "Server port")
		host       = flag.String("host", "0.0.0.0", "Server host")
		configPath = flag.String("config", "", "Path to configuration file")
		// tlsCert    = flag.String("tls-cert", "", "Path to TLS certificate") // unused for now
		// tlsKey     = flag.String("tls-key", "", "Path to TLS key") // unused for now
		// jwtSecret  = flag.String("jwt-secret", "", "JWT secret for authentication") // unused for now
		_ = flag.Bool("auth", false, "Enable authentication")
	)
	flag.Parse()

	fmt.Printf("Starting DriftMgr Server\n")
	fmt.Printf("Listening on %s:%s\n", *host, *port)

	// Create server configuration
	portInt, _ := strconv.Atoi(*port)
	apiConfig := &api.Config{
		Host:        *host,
		Port:        portInt,
		AuthEnabled: true, // Enable authentication by default
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
	log.Printf("Created API server with config: Host=%s, Port=%d", apiConfig.Host, apiConfig.Port)

	// Start server
	log.Println("Starting server...")
	if err := apiServer.Start(nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}

	// Give server time to start
	time.Sleep(2 * time.Second)
	fmt.Println("✅ DriftMgr API Server is running!")
	fmt.Printf("🌐 API: http://localhost:%s\n", *port)
	fmt.Printf("🔌 Health: http://localhost:%s/health\n", *port)
	fmt.Printf("📊 Dashboard: http://localhost:%s/dashboard\n", *port)
	fmt.Println("\nServer is running. Press Ctrl+C to stop.")

	// Keep server running indefinitely
	select {}
}
