package main

import (
	"fmt"
	"log"

	"github.com/catherinevee/driftmgr/internal/api"
)

func main() {
	fmt.Println("Starting quick test server...")

	// Create server without pre-discovery
	server, err := api.NewServer("8085")
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	fmt.Println("Server created, starting on port 8085")
	fmt.Println("Check http://localhost:8085/api/v1/resources/stats for cached resources")

	// Start server
	if err := server.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
