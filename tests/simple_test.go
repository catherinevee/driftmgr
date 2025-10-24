package tests

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/api"
)

func main() {
	fmt.Println("Testing server startup...")

	// Create a simple config
	config := &api.Config{
		Host:        "0.0.0.0",
		Port:        8080,
		AuthEnabled: false,
	}

	// Create services
	services := &api.Services{}

	// Create server
	server := api.NewServer(config, services)
	fmt.Println("Server created successfully")

	// Try to start server
	fmt.Println("Starting server...")

	// Start server in goroutine
	go func() {
		if err := server.Start(nil); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait a bit
	time.Sleep(3 * time.Second)

	// Test if server is running
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		log.Printf("Failed to connect to server: %v", err)
	} else {
		fmt.Printf("Server is running! Status: %s\n", resp.Status)
		resp.Body.Close()
	}

	fmt.Println("Test completed")
}
