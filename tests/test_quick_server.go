package main

import (
	"context"
	"fmt"
	"log"

	"github.com/catherinevee/driftmgr/internal/api"
)

func main() {
	fmt.Println("Starting quick test server...")

	// Create server without pre-discovery
	config := &api.Config{
		Port: 8085,
	}
	services := &api.Services{}
	server := api.NewServer(config, services)

	fmt.Println("Server created, starting on port 8085")
	fmt.Println("Check http://localhost:8085/api/v1/resources/stats for cached resources")

	// Start server
	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
