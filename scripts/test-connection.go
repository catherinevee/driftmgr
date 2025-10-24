package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/providers"
)

func main() {
	fmt.Println("Testing Connection Testing Functionality...")

	// Create event bus
	eventBus := events.NewEventBus()

	// Create provider factory
	factory := providers.NewProviderFactory(map[string]interface{}{})

	// Create connection service
	connectionService := providers.NewConnectionService(factory, eventBus, 30*time.Second)

	// Test AWS connection
	fmt.Println("\n=== Testing AWS Connection ===")
	result, err := connectionService.TestProviderConnection(context.Background(), "aws", "us-east-1")
	if err != nil {
		log.Printf("AWS connection test failed: %v", err)
	} else {
		fmt.Printf("AWS Connection Test Result:\n")
		fmt.Printf("  Provider: %s\n", result.Provider)
		fmt.Printf("  Region: %s\n", result.Region)
		fmt.Printf("  Success: %t\n", result.Success)
		fmt.Printf("  Latency: %v\n", result.Latency)
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}
		if result.Details != nil {
			fmt.Printf("  Details: %+v\n", result.Details)
		}
	}

	// Test Azure connection
	fmt.Println("\n=== Testing Azure Connection ===")
	result, err = connectionService.TestProviderConnection(context.Background(), "azure", "eastus")
	if err != nil {
		log.Printf("Azure connection test failed: %v", err)
	} else {
		fmt.Printf("Azure Connection Test Result:\n")
		fmt.Printf("  Provider: %s\n", result.Provider)
		fmt.Printf("  Region: %s\n", result.Region)
		fmt.Printf("  Success: %t\n", result.Success)
		fmt.Printf("  Latency: %v\n", result.Latency)
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}
		if result.Details != nil {
			fmt.Printf("  Details: %+v\n", result.Details)
		}
	}

	// Test GCP connection
	fmt.Println("\n=== Testing GCP Connection ===")
	result, err = connectionService.TestProviderConnection(context.Background(), "gcp", "us-central1")
	if err != nil {
		log.Printf("GCP connection test failed: %v", err)
	} else {
		fmt.Printf("GCP Connection Test Result:\n")
		fmt.Printf("  Provider: %s\n", result.Provider)
		fmt.Printf("  Region: %s\n", result.Region)
		fmt.Printf("  Success: %t\n", result.Success)
		fmt.Printf("  Latency: %v\n", result.Latency)
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}
		if result.Details != nil {
			fmt.Printf("  Details: %+v\n", result.Details)
		}
	}

	// Test DigitalOcean connection
	fmt.Println("\n=== Testing DigitalOcean Connection ===")
	result, err = connectionService.TestProviderConnection(context.Background(), "digitalocean", "nyc1")
	if err != nil {
		log.Printf("DigitalOcean connection test failed: %v", err)
	} else {
		fmt.Printf("DigitalOcean Connection Test Result:\n")
		fmt.Printf("  Provider: %s\n", result.Provider)
		fmt.Printf("  Region: %s\n", result.Region)
		fmt.Printf("  Success: %t\n", result.Success)
		fmt.Printf("  Latency: %v\n", result.Latency)
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}
		if result.Details != nil {
			fmt.Printf("  Details: %+v\n", result.Details)
		}
	}

	// Test all providers
	fmt.Println("\n=== Testing All Providers ===")
	results, err := connectionService.TestAllProviders(context.Background(), "us-east-1")
	if err != nil {
		log.Printf("All providers test failed: %v", err)
	} else {
		fmt.Printf("All Providers Test Results:\n")
		for provider, result := range results {
			fmt.Printf("  %s: Success=%t, Latency=%v", provider, result.Success, result.Latency)
			if result.Error != "" {
				fmt.Printf(", Error=%s", result.Error)
			}
			fmt.Println()
		}
	}

	// Get connection summary
	fmt.Println("\n=== Connection Summary ===")
	summary := connectionService.GetConnectionSummary()
	for provider, providerSummary := range summary {
		fmt.Printf("  %s: %+v\n", provider, providerSummary)
	}

	fmt.Println("\nConnection testing completed!")
}
