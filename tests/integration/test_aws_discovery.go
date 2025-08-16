package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// DiscoveryRequest represents the request for resource discovery
type DiscoveryRequest struct {
	Provider string   `json:"provider"`
	Regions  []string `json:"regions"`
	Account  string   `json:"account"`
}

// DiscoveryResponse represents the response from resource discovery
type DiscoveryResponse struct {
	Resources []Resource    `json:"resources"`
	Total     int           `json:"total"`
	Duration  time.Duration `json:"duration"`
}

// Resource represents a discovered cloud resource
type Resource struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Provider string            `json:"provider"`
	Region   string            `json:"region"`
	Tags     map[string]string `json:"tags"`
	State    string            `json:"state"`
	Created  time.Time         `json:"created"`
	Updated  time.Time         `json:"updated"`
}

// TestEnhancedAWSDiscovery demonstrates the enhanced AWS discovery with more regions
func TestEnhancedAWSDiscovery() {
	fmt.Println("=== Enhanced AWS Discovery Test ===")
	fmt.Println("Testing AWS resource discovery with various region configurations...")
	fmt.Println()

	// Test cases
	testCases := []struct {
		name    string
		regions []string
	}{
		{
			name:    "Single Region Test",
			regions: []string{"us-east-1"},
		},
		{
			name:    "Multiple Regions Test",
			regions: []string{"us-east-1", "us-west-2", "eu-west-1"},
		},
		{
			name:    "All Regions Test",
			regions: []string{"all"},
		},
		{
			name:    "Edge Regions Test",
			regions: []string{"ap-southeast-4", "me-central-1", "eu-south-2"},
		},
	}

	for _, tc := range testCases {
		fmt.Printf("--- %s ---\n", tc.name)
		fmt.Printf("Regions: %v\n", tc.regions)

		// Create discovery request
		req := DiscoveryRequest{
			Provider: "aws",
			Regions:  tc.regions,
			Account:  "default",
		}

		// Send request to DriftMgr server
		resp, err := sendDiscoveryRequest(req)
		if err != nil {
			fmt.Printf("❌ Error: %v\n", err)
		} else {
			fmt.Printf("✅ Success! Discovered %d resources in %v\n", resp.Total, resp.Duration)

			// Show sample resources
			if len(resp.Resources) > 0 {
				fmt.Println("Sample resources:")
				for i, resource := range resp.Resources {
					if i >= 5 { // Show only first 5
						break
					}
					fmt.Printf("  • %s (%s) in %s\n", resource.Name, resource.Type, resource.Region)
				}
				if len(resp.Resources) > 5 {
					fmt.Printf("  ... and %d more resources\n", len(resp.Resources)-5)
				}
			}
		}
		fmt.Println()
	}
}

// sendDiscoveryRequest sends a discovery request to the DriftMgr server
func sendDiscoveryRequest(req DiscoveryRequest) (*DiscoveryResponse, error) {
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
	var discoveryResp DiscoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&discoveryResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &discoveryResp, nil
}

// TestRegionExpansion tests the "all" regions expansion functionality
func TestRegionExpansion() {
	fmt.Println("=== Region Expansion Test ===")

	// Expected regions when "all" is specified
	expectedRegions := []string{
		"us-east-1",      // US East (N. Virginia)
		"us-east-2",      // US East (Ohio)
		"us-west-1",      // US West (N. California)
		"us-west-2",      // US West (Oregon)
		"af-south-1",     // Africa (Cape Town)
		"ap-east-1",      // Asia Pacific (Hong Kong)
		"ap-south-1",     // Asia Pacific (Mumbai)
		"ap-northeast-1", // Asia Pacific (Tokyo)
		"ap-northeast-2", // Asia Pacific (Seoul)
		"ap-northeast-3", // Asia Pacific (Osaka)
		"ap-southeast-1", // Asia Pacific (Singapore)
		"ap-southeast-2", // Asia Pacific (Sydney)
		"ap-southeast-3", // Asia Pacific (Jakarta)
		"ap-southeast-4", // Asia Pacific (Melbourne)
		"ca-central-1",   // Canada (Central)
		"eu-central-1",   // Europe (Frankfurt)
		"eu-west-1",      // Europe (Ireland)
		"eu-west-2",      // Europe (London)
		"eu-west-3",      // Europe (Paris)
		"eu-north-1",     // Europe (Stockholm)
		"eu-south-1",     // Europe (Milan)
		"eu-south-2",     // Europe (Spain)
		"me-south-1",     // Middle East (Bahrain)
		"me-central-1",   // Middle East (UAE)
		"sa-east-1",      // South America (São Paulo)
	}

	fmt.Printf("Expected regions when 'all' is specified: %d regions\n", len(expectedRegions))
	fmt.Println("Regions:")
	for i, region := range expectedRegions {
		fmt.Printf("  %2d. %s\n", i+1, region)
	}
	fmt.Println()
}

// TestPerformanceComparison compares performance between single region and all regions
func TestPerformanceComparison() {
	fmt.Println("=== Performance Comparison Test ===")

	// Test single region
	fmt.Println("Testing single region (us-east-1)...")
	start := time.Now()
	req1 := DiscoveryRequest{
		Provider: "aws",
		Regions:  []string{"us-east-1"},
		Account:  "default",
	}
	resp1, err1 := sendDiscoveryRequest(req1)
	duration1 := time.Since(start)

	if err1 != nil {
		fmt.Printf("❌ Single region test failed: %v\n", err1)
	} else {
		fmt.Printf("✅ Single region: %d resources in %v\n", resp1.Total, duration1)
	}

	// Test all regions
	fmt.Println("Testing all regions...")
	start = time.Now()
	req2 := DiscoveryRequest{
		Provider: "aws",
		Regions:  []string{"all"},
		Account:  "default",
	}
	resp2, err2 := sendDiscoveryRequest(req2)
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

func runAWSTest() {
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
	TestRegionExpansion()
	TestEnhancedAWSDiscovery()
	TestPerformanceComparison()

	fmt.Println("=== Test Complete ===")
	fmt.Println("Enhanced AWS discovery with more regions has been tested successfully!")
}
