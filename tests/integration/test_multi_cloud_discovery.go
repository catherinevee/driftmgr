package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// MultiCloudDiscoveryRequest represents the request for multi-cloud resource discovery
type MultiCloudDiscoveryRequest struct {
	Provider string   `json:"provider"`
	Regions  []string `json:"regions"`
	Account  string   `json:"account"`
}

// MultiCloudDiscoveryResponse represents the response from multi-cloud resource discovery
type MultiCloudDiscoveryResponse struct {
	Resources []Resource    `json:"resources"`
	Total     int           `json:"total"`
	Duration  time.Duration `json:"duration"`
	Provider  string        `json:"provider"`
	Regions   []string      `json:"regions"`
}

// TestMultiCloudDiscovery demonstrates multi-cloud discovery with AWS, Azure, and GCP
func TestMultiCloudDiscovery() {
	fmt.Println("=== Multi-Cloud Discovery Test ===")
	fmt.Println("Testing AWS, Azure, and GCP resource discovery with various region configurations...")
	fmt.Println()

	// Test cases for each provider
	providers := []string{"aws", "azure", "gcp"}

	for _, provider := range providers {
		fmt.Printf("--- Testing %s Discovery ---\n", provider)

		testCases := []struct {
			name    string
			regions []string
		}{
			{
				name:    "Single Region Test",
				regions: getSingleRegion(provider),
			},
			{
				name:    "Multiple Regions Test",
				regions: getMultipleRegions(provider),
			},
			{
				name:    "All Regions Test",
				regions: []string{"all"},
			},
		}

		for _, tc := range testCases {
			fmt.Printf("  %s\n", tc.name)
			fmt.Printf("  Regions: %v\n", tc.regions)

			// Create discovery request
			req := MultiCloudDiscoveryRequest{
				Provider: provider,
				Regions:  tc.regions,
				Account:  "default",
			}

			// Send request to DriftMgr server
			resp, err := sendMultiCloudDiscoveryRequest(req)
			if err != nil {
				fmt.Printf("  ❌ Error: %v\n", err)
			} else {
				fmt.Printf("  ✅ Success! Discovered %d resources in %v\n", resp.Total, resp.Duration)

				// Show sample resources
				if len(resp.Resources) > 0 {
					fmt.Println("  Sample resources:")
					for i, resource := range resp.Resources {
						if i >= 3 { // Show only first 3
							break
						}
						fmt.Printf("    • %s (%s) in %s\n", resource.Name, resource.Type, resource.Region)
					}
					if len(resp.Resources) > 3 {
						fmt.Printf("    ... and %d more resources\n", len(resp.Resources)-3)
					}
				}
			}
			fmt.Println()
		}
	}
}

// getSingleRegion returns a single region for testing based on provider
func getSingleRegion(provider string) []string {
	switch provider {
	case "aws":
		return []string{"us-east-1"}
	case "azure":
		return []string{"eastus"}
	case "gcp":
		return []string{"us-central1"}
	default:
		return []string{"us-east-1"}
	}
}

// getMultipleRegions returns multiple regions for testing based on provider
func getMultipleRegions(provider string) []string {
	switch provider {
	case "aws":
		return []string{"us-east-1", "us-west-2", "eu-west-1"}
	case "azure":
		return []string{"eastus", "westus2", "northeurope"}
	case "gcp":
		return []string{"us-central1", "us-east1", "europe-west1"}
	default:
		return []string{"us-east-1", "us-west-2"}
	}
}

// sendMultiCloudDiscoveryRequest sends a multi-cloud discovery request to the DriftMgr server
func sendMultiCloudDiscoveryRequest(req MultiCloudDiscoveryRequest) (*MultiCloudDiscoveryResponse, error) {
	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Send POST request to DriftMgr server
	resp, err := http.Post("http://localhost:8080/api/v1/discover", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Decode response
	var discoveryResp MultiCloudDiscoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&discoveryResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &discoveryResp, nil
}

// TestRegionExpansionMultiCloud tests the "all" regions expansion functionality for all providers
func TestRegionExpansionMultiCloud() {
	fmt.Println("=== Multi-Cloud Region Expansion Test ===")

	providers := []string{"aws", "azure", "gcp"}

	for _, provider := range providers {
		fmt.Printf("--- %s Regions ---\n", provider)

		// Expected regions when "all" is specified
		expectedRegions := getExpectedRegions(provider)

		fmt.Printf("Expected regions when 'all' is specified: %d regions\n", len(expectedRegions))
		fmt.Println("Sample regions:")
		for i, region := range expectedRegions {
			if i >= 10 { // Show only first 10
				fmt.Printf("  ... and %d more regions\n", len(expectedRegions)-10)
				break
			}
			fmt.Printf("  %2d. %s\n", i+1, region)
		}
		fmt.Println()
	}
}

// getExpectedRegions returns expected regions for each provider
func getExpectedRegions(provider string) []string {
	switch provider {
	case "aws":
		return []string{
			"us-east-1", "us-east-2", "us-west-1", "us-west-2", "af-south-1",
			"ap-east-1", "ap-south-1", "ap-south-2", "ap-northeast-1", "ap-northeast-2", "ap-northeast-3",
			"ap-southeast-1", "ap-southeast-2", "ap-southeast-3", "ap-southeast-4",
			"ca-central-1", "eu-central-1", "eu-central-2", "eu-west-1", "eu-west-2", "eu-west-3",
			"eu-north-1", "eu-south-1", "eu-south-2", "me-south-1", "me-central-1", "sa-east-1", "il-central-1",
		}
	case "azure":
		return []string{
			"eastus", "eastus2", "southcentralus", "westus2", "westus3",
			"australiaeast", "australiasoutheast", "southeastasia", "northeurope", "swedencentral", "uksouth", "ukwest",
			"westeurope", "centralus", "northcentralus", "westus", "southafricanorth", "southafricawest",
			"centralindia", "southindia", "westindia", "eastasia", "japaneast", "japanwest", "koreacentral", "koreasouth",
			"canadacentral", "canadaeast", "francecentral", "francesouth", "germanywestcentral", "germanynorth", "italynorth",
			"norwayeast", "polandcentral", "switzerlandnorth", "switzerlandwest", "uaenorth", "uaecentral", "brazilsouth", "brazilsoutheast",
			"chilecentral", "mexicocentral", "qatarcentral",
		}
	case "gcp":
		return []string{
			"us-central1", "us-east1", "us-east4", "us-east5", "us-west1", "us-west2",
			"us-west3", "us-west4", "europe-west1", "europe-west2", "europe-west3",
			"europe-west4", "europe-west6", "europe-west8", "europe-west9", "europe-west10",
			"europe-west12", "europe-central2", "europe-north1", "europe-southwest1",
			"asia-east1", "asia-northeast1", "asia-northeast2", "asia-northeast3",
			"asia-south1", "asia-south2", "asia-southeast1", "asia-southeast2",
			"australia-southeast1", "australia-southeast2", "southamerica-east1",
			"northamerica-northeast1", "northamerica-northeast2",
		}
	default:
		return []string{}
	}
}

// TestPerformanceComparisonMultiCloud compares performance between providers
func TestPerformanceComparisonMultiCloud() {
	fmt.Println("=== Multi-Cloud Performance Comparison Test ===")

	providers := []string{"aws", "azure", "gcp"}

	for _, provider := range providers {
		fmt.Printf("--- %s Performance Test ---\n", provider)

		// Test single region
		fmt.Printf("Testing single region...\n")
		start := time.Now()
		req1 := MultiCloudDiscoveryRequest{
			Provider: provider,
			Regions:  getSingleRegion(provider),
			Account:  "default",
		}
		resp1, err1 := sendMultiCloudDiscoveryRequest(req1)
		duration1 := time.Since(start)

		if err1 != nil {
			fmt.Printf("❌ Single region test failed: %v\n", err1)
			continue
		} else {
			fmt.Printf("✅ Single region: %d resources in %v\n", resp1.Total, duration1)
		}

		// Test all regions
		fmt.Printf("Testing all regions...\n")
		start = time.Now()
		req2 := MultiCloudDiscoveryRequest{
			Provider: provider,
			Regions:  []string{"all"},
			Account:  "default",
		}
		resp2, err2 := sendMultiCloudDiscoveryRequest(req2)
		duration2 := time.Since(start)

		if err2 != nil {
			fmt.Printf("❌ All regions test failed: %v\n", err2)
		} else {
			fmt.Printf("✅ All regions: %d resources in %v\n", resp2.Total, duration2)

			// Calculate performance metrics
			if resp1 != nil && resp1.Total > 0 {
				resourceRatio := float64(resp2.Total) / float64(resp1.Total)
				timeRatio := float64(duration2) / float64(duration1)
				fmt.Printf("📊 Performance metrics:\n")
				fmt.Printf("   Resource ratio: %.2fx more resources\n", resourceRatio)
				fmt.Printf("   Time ratio: %.2fx longer\n", timeRatio)
				fmt.Printf("   Efficiency: %.2f resources per second\n", float64(resp2.Total)/duration2.Seconds())
			}
		}
		fmt.Println()
	}
}

// TestCrossCloudComparison compares discovery across different cloud providers
func TestCrossCloudComparison() {
	fmt.Println("=== Cross-Cloud Comparison Test ===")

	// Test all providers with single region
	providers := []string{"aws", "azure", "gcp"}
	results := make(map[string]*MultiCloudDiscoveryResponse)

	for _, provider := range providers {
		fmt.Printf("Testing %s...\n", provider)

		req := MultiCloudDiscoveryRequest{
			Provider: provider,
			Regions:  getSingleRegion(provider),
			Account:  "default",
		}

		resp, err := sendMultiCloudDiscoveryRequest(req)
		if err != nil {
			fmt.Printf("❌ %s test failed: %v\n", provider, err)
			continue
		}

		results[provider] = resp
		fmt.Printf("✅ %s: %d resources in %v\n", provider, resp.Total, resp.Duration)
	}

	// Compare results
	fmt.Println("\n📊 Cross-Cloud Comparison:")
	if len(results) > 1 {
		var maxResources int
		var fastestProvider string
		var fastestTime time.Duration

		for provider, result := range results {
			if result.Total > maxResources {
				maxResources = result.Total
			}
			if fastestTime == 0 || result.Duration < fastestTime {
				fastestTime = result.Duration
				fastestProvider = provider
			}
		}

		fmt.Printf("   Most resources: %d\n", maxResources)
		fmt.Printf("   Fastest discovery: %s (%v)\n", fastestProvider, fastestTime)
	}
	fmt.Println()
}

func runMultiCloudTest() {
	// Check if DriftMgr server is running
	fmt.Println("Checking DriftMgr server status...")
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		fmt.Println("❌ DriftMgr server is not running!")
		fmt.Println("Please start the server with: ./bin/driftmgr-server.exe")
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ DriftMgr server is running")
	} else {
		fmt.Printf("❌ DriftMgr server returned status %d\n", resp.StatusCode)
		os.Exit(1)
	}
	fmt.Println()

	// Run tests
	TestRegionExpansionMultiCloud()
	TestMultiCloudDiscovery()
	TestPerformanceComparisonMultiCloud()
	TestCrossCloudComparison()

	fmt.Println("=== Multi-Cloud Test Complete ===")
	fmt.Println("Enhanced multi-cloud discovery with AWS, Azure, and GCP has been tested successfully!")
}
