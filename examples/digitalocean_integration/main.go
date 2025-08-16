package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/catherinevee/driftmgr/internal/discovery"
)

// Example of how to integrate DigitalOcean support into driftmgr
func main() {
	fmt.Println("DigitalOcean Integration Example")
	fmt.Println("================================")

	// Set up DigitalOcean API token
	apiToken := os.Getenv("DIGITALOCEAN_TOKEN")
	if apiToken == "" {
		fmt.Println("Please set DIGITALOCEAN_TOKEN environment variable")
		fmt.Println("Example: export DIGITALOCEAN_TOKEN=your_api_token_here")
		return
	}

	// Create DigitalOcean integration
	doIntegration := discovery.NewDigitalOceanIntegration(apiToken, "nyc1")

	// Validate credentials
	fmt.Println("Validating DigitalOcean credentials...")
	if err := doIntegration.ValidateCredentials(context.Background()); err != nil {
		log.Fatalf("Credential validation failed: %v", err)
	}
	fmt.Println("✓ Credentials validated successfully")

	// Show supported regions
	fmt.Println("\nSupported DigitalOcean regions:")
	regions := doIntegration.GetSupportedRegions()
	for _, region := range regions {
		fmt.Printf("  - %s\n", region)
	}

	// Show supported resource types
	fmt.Println("\nSupported DigitalOcean resource types:")
	resourceTypes := doIntegration.GetResourceTypes()
	for _, resourceType := range resourceTypes {
		fmt.Printf("  - %s\n", resourceType)
	}

	// Discover resources in specific regions
	fmt.Println("\nDiscovering DigitalOcean resources...")
	regionsToScan := []string{"nyc1", "sfo2", "lon1"}
	resources, err := doIntegration.DiscoverResources(context.Background(), regionsToScan, "digitalocean")
	if err != nil {
		log.Fatalf("Discovery failed: %v", err)
	}

	fmt.Printf("✓ Discovery complete. Found %d resources\n", len(resources))

	// Display discovered resources
	fmt.Println("\nDiscovered resources:")
	for i, resource := range resources {
		fmt.Printf("%d. %s (%s) - %s - %s\n", 
			i+1, 
			resource.Name, 
			resource.Type, 
			resource.Region, 
			resource.Status)
	}
}

// Example CLI integration function
func handleDigitalOceanDiscovery(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: driftmgr discover digitalocean [regions...]")
		fmt.Println("Example: driftmgr discover digitalocean nyc1 sfo2 lon1")
		return
	}

	// Get regions from arguments
	regions := args[1:]
	if len(regions) == 0 {
		regions = []string{"nyc1", "sfo2", "lon1", "fra1", "sgp1", "tor1", "ams3", "blr1"}
	}

	// Create integration
	doIntegration := discovery.NewDigitalOceanIntegration("", "")

	// Discover resources
	resources, err := doIntegration.DiscoverResources(context.Background(), regions, "digitalocean")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Display results
	fmt.Printf("Discovered %d DigitalOcean resources:\n", len(resources))
	for _, resource := range resources {
		fmt.Printf("  - %s (%s) in %s\n", resource.Name, resource.Type, resource.Region)
	}
}
