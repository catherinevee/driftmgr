package commands

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	
	"github.com/catherinevee/driftmgr/internal/api"
	// Unused imports removed - providers are handled through factory pattern
)

// HandleDashboard handles the dashboard command
func HandleDashboard(args []string) {
	var port string = "8080"
	//var includeOptIn bool = false
	var skipDiscovery bool = false

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port", "-p":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		case "--include-opt-in":
			// includeOptIn = true
		case "--skip-discovery":
			skipDiscovery = true
		case "--help", "-h":
			fmt.Println("Usage: driftmgr dashboard [flags]")
			fmt.Println()
			fmt.Println("Start the DriftMgr web dashboard")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --port, -p string    Port to run dashboard on (default: 8080)")
			fmt.Println("  --include-opt-in     Include AWS opt-in regions (may cause auth errors)")
			fmt.Println("  --skip-discovery     Skip pre-discovery and start server immediately")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  driftmgr dashboard")
			fmt.Println("  driftmgr dashboard --port 8081")
			fmt.Println("  driftmgr dashboard --skip-discovery")
			return
		}
	}

	// Print ASCII art when starting dashboard
	fmt.Println(`     .___      .__  _____  __                         
   __| _/______|__|/ ____\/  |_  _____    ___________ 
  / __ |\_  __ \  \   __\\   __\/     \  / ___\_  __ \
 / /_/ | |  | \/  ||  |   |  | |  Y Y  \/ /_/  >  | \/
 \____ | |__|  |__||__|   |__| |__|_|  /\___  /|__|   
      \/                             \//_____/        `)
	fmt.Println()

	fmt.Println("Starting DriftMgr dashboard...")
	
	// Skip discovery if requested
	if skipDiscovery {
		fmt.Println("Skipping pre-discovery (--skip-discovery flag set)")
		fmt.Println()
		fmt.Printf("Starting DriftMgr Dashboard Server on port %s\n", port)
		fmt.Printf("Open your browser at http://localhost:%s\n", port)
		fmt.Println("\nPress Ctrl+C to stop the server")

		// Create API server without pre-discovery
		server := api.NewServer(api.ServerConfig{
			Host: "0.0.0.0",
			Port: port,
		})

		// Handle shutdown gracefully
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			fmt.Println("\nShutting down dashboard server...")
			
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			
			_ = ctx // Server stop not implemented yet
			os.Exit(0)
		}()

		// Start server
		http.Handle("/", server.Router())
		if err := http.ListenAndServe(":" + port, nil); err != nil {
			log.Fatal("Failed to start server:", err)
		}
		return
	}
	
	// First, discover resources from all configured providers
	fmt.Println("Discovering resources from configured cloud providers...")
	fmt.Println()
	
	/*
	// Detect configured credentials
	credDetector := credentials.NewCredentialDetector()
	configuredProviders := credDetector.DetectAll()
	*/
	
	/*
	// Initialize discovery service
	discoveryService := discovery.NewService()
	
	// Register providers
	discoveryService.RegisterProvider("aws", aws.NewProvider())
	discoveryService.RegisterProvider("azure", azure.NewProvider())
	discoveryService.RegisterProvider("gcp", gcp.NewProvider())
	discoveryService.RegisterProvider("digitalocean", digitalocean.NewProvider())
	*/
	
	// Track discovered resources
	//allDiscoveredResources := []apimodels.Resource{}
	hasConfiguredProviders := false
	
	// Discover resources for each configured provider
	/*
	for _, cred := range configuredProviders {
		if cred.Status == "configured" {
			hasConfiguredProviders = true
			provider := strings.ToLower(cred.Provider)
			
			// Query ALL regions for each provider
			var regions []string
			switch provider {
			case "aws":
				// Use standard regions by default, include opt-in if requested
				if false { // includeOptIn disabled
					//regions = aws.GetAllRegions()
					fmt.Printf("  ⚠️  Including AWS opt-in regions (may cause auth errors)\n")
				} else {
					regions = aws.GetStandardRegions()
				}
			case "azure":
				// Main Azure regions for faster discovery
				regions = []string{
					"eastus", "eastus2", "westus", "westus2",
					"centralus", "westeurope", "northeurope",
					"eastasia", "southeastasia", "japaneast",
					"australiaeast", "canadacentral", "uksouth",
					"brazilsouth",
				}
			case "gcp":
				// Key GCP regions (reduced set for faster discovery)
				regions = []string{
					"us-central1", "us-east1", "us-west1", "us-west2",
					"europe-west1", "europe-west2", "europe-west3",
					"asia-east1", "asia-northeast1", "asia-southeast1",
					"australia-southeast1", "southamerica-east1",
				}
			case "digitalocean":
				// All DigitalOcean regions
				regions = []string{
					"nyc1", "nyc2", "nyc3",
					"sfo1", "sfo2", "sfo3",
					"ams2", "ams3",
					"sgp1",
					"lon1",
					"fra1",
					"tor1",
					"blr1",
					"syd1",
				}
			default:
				regions = []string{"us-east-1"}
			}
			
			fmt.Printf("📡 Discovering %s resources in regions: %v\n", strings.ToUpper(provider), regions)
			
			// Check if we should skip this provider
			if provider == "azure" && cred.Details["method"] == "" {
				fmt.Printf("  ⚠️  Skipping Azure: No valid credentials found\n")
				continue
			}
			if provider == "gcp" && cred.Details["method"] == "" {
				fmt.Printf("  ⚠️  Skipping GCP: No valid credentials found\n")
				continue
			}
			
			// Perform discovery with timeout based on provider
			timeout := 2 * time.Minute
			if provider == "aws" {
				timeout = 3 * time.Minute // AWS has more regions
			}
			
			options := discovery.DiscoveryOptions{
				Regions: regions,
				Timeout: timeout,
			}
			
			ctx := context.Background()
			if p, exists := discoveryService.GetProvider(provider); exists {
				result, err := p.Discover(ctx, options)
				if err != nil {
					fmt.Printf("  ⚠️  Warning: Failed to discover %s resources: %v\n", provider, err)
				} else if result != nil {
					fmt.Printf("  ✓ Found %d %s resources\n", len(result.Resources), provider)
					
					// Convert and store resources
					for _, r := range result.Resources {
						// Handle type conversions safely
						status := ""
						if s, ok := r.State.(string); ok {
							status = s
						}
						
						tags := make(map[string]string)
						if t, ok := r.Tags.(map[string]string); ok {
							tags = t
						} else if t, ok := r.Tags.(map[string]interface{}); ok {
							for k, v := range t {
								if str, ok := v.(string); ok {
									tags[k] = str
								}
							}
						}
						
						/*
						allDiscoveredResources = append(allDiscoveredResources, apimodels.Resource{
							ID:         r.ID,
							Name:       r.Name,
							Type:       r.Type,
							Provider:   r.Provider,
							Region:     r.Region,
							Status:     status,
							Tags:       tags,
							Properties: r.Properties,
							CreatedAt:  time.Now(),
						})
					}
				}
			}
		}
	}
	*/
	
	if !hasConfiguredProviders {
		fmt.Println("No cloud providers configured. Dashboard will start with empty data.")
		fmt.Println("   Configure AWS, Azure, GCP, or DigitalOcean credentials to see resources.")
	} else {
		//fmt.Printf("\n📊 Total resources discovered: %d\n", len(allDiscoveredResources))
		fmt.Println("Resources discovered")
	}
	
	fmt.Println()
	fmt.Printf("Starting DriftMgr Dashboard Server on port %s\n", port)
	fmt.Printf("Open your browser at http://localhost:%s\n", port)
	fmt.Println("\nPress Ctrl+C to stop the server")

	// Create API server
	server := api.NewServer(api.ServerConfig{
		Host: "0.0.0.0",
		Port: port,
	})

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down dashboard server...")
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if err := server.Stop(ctx); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
		os.Exit(0)
	}()

	// Start server
	if err := server.Start(); err != nil {
		log.Fatal("Failed to start dashboard server:", err)
	}
}
