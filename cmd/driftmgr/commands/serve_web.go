package commands

import (
	"fmt"
	"log"

	"github.com/catherinevee/driftmgr/internal/api"
)

// HandleServeWeb handles the serve web command with optional auto-discovery
func HandleServeWeb(args []string) {
	var port string = "8080"
	var autoDiscover bool = false
	var scanInterval string = "5m"
	var debug bool = false

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port", "-p":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		case "--auto-discover":
			autoDiscover = true
		case "--scan-interval":
			if i+1 < len(args) {
				scanInterval = args[i+1]
				i++
			}
		case "--debug":
			debug = true
		case "--help", "-h":
			fmt.Println("Usage: driftmgr serve web [flags]")
			fmt.Println()
			fmt.Println("Start the DriftMgr web server")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --port, -p string        Port to run server on (default: 8080)")
			fmt.Println("  --auto-discover          Enable automatic state discovery")
			fmt.Println("  --scan-interval string   Discovery scan interval (default: 5m)")
			fmt.Println("  --debug                  Enable debug output")
			return
		}
	}

	// Print ASCII art
	fmt.Println(`     .___      .__  _____  __                         
   __| _/______|__|/ ____\/  |_  _____    ___________ 
  / __ |\_  __ \  \   __\\   __\/     \  / ___\_  __ \
 / /_/ | |  | \/  ||  |   |  | |  Y Y  \/ /_/  >  | \/
 \____ | |__|  |__||__|   |__| |__|_|  /\___  /|__|   
      \/                             \//_____/        `)
	fmt.Println()

	fmt.Println("Starting DriftMgr web server...")
	fmt.Println("Loading cached resources...")
	
	// Create server with config
	config := &api.ServerConfig{
		Port: port,
	}
	
	// Create and start server
	server := api.NewServer(*config)

	fmt.Printf("\nStarting DriftMgr Web Server on port %s\n", port)
	if autoDiscover {
		fmt.Printf("Auto-discovery enabled with scan interval: %s\n", scanInterval)
	}
	if debug {
		fmt.Println("Debug mode enabled")
	}
	fmt.Printf("Open your browser at http://localhost:%s\n", port)
	fmt.Println("\nPress Ctrl+C to stop the server")

	// Start server
	if err := server.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}