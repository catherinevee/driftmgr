package providers

import (
	"context"
	"fmt"
	"time"
)

// ConnectionTestResult represents the result of a connection test
type ConnectionTestResult struct {
	Provider string                 `json:"provider"`
	Success  bool                   `json:"success"`
	Latency  time.Duration          `json:"latency"`
	Error    string                 `json:"error,omitempty"`
	Details  map[string]interface{} `json:"details"`
	TestedAt time.Time              `json:"tested_at"`
	Region   string                 `json:"region,omitempty"`
	Service  string                 `json:"service,omitempty"`
}

// ConnectionTester defines the interface for testing cloud provider connections
type ConnectionTester interface {
	TestConnection(ctx context.Context, provider CloudProvider, region string) (*ConnectionTestResult, error)
	TestServiceConnection(ctx context.Context, provider CloudProvider, region, service string) (*ConnectionTestResult, error)
	TestAllRegions(ctx context.Context, provider CloudProvider) ([]ConnectionTestResult, error)
	TestAllServices(ctx context.Context, provider CloudProvider, region string) ([]ConnectionTestResult, error)
}

// ConnectionTesterImpl implements the ConnectionTester interface
type ConnectionTesterImpl struct {
	timeout time.Duration
}

// NewConnectionTester creates a new connection tester
func NewConnectionTester(timeout time.Duration) *ConnectionTesterImpl {
	return &ConnectionTesterImpl{
		timeout: timeout,
	}
}

// TestConnection tests the basic connection to a cloud provider
func (ct *ConnectionTesterImpl) TestConnection(ctx context.Context, provider CloudProvider, region string) (*ConnectionTestResult, error) {
	start := time.Now()

	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, ct.timeout)
	defer cancel()

	result := &ConnectionTestResult{
		Provider: provider.Name(),
		Region:   region,
		Details:  make(map[string]interface{}),
		TestedAt: time.Now(),
	}

	// Test credentials validation
	err := provider.ValidateCredentials(testCtx)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("credential validation failed: %v", err)
		result.Latency = time.Since(start)
		return result, nil
	}

	// Test region listing
	regions, err := provider.ListRegions(testCtx)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("region listing failed: %v", err)
		result.Latency = time.Since(start)
		return result, nil
	}

	// Check if the specified region is available
	regionFound := false
	for _, r := range regions {
		if r == region {
			regionFound = true
			break
		}
	}

	if !regionFound {
		result.Success = false
		result.Error = fmt.Sprintf("region %s not available", region)
		result.Latency = time.Since(start)
		return result, nil
	}

	// Test resource discovery in the region
	resources, err := provider.DiscoverResources(testCtx, region)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("resource discovery failed: %v", err)
		result.Latency = time.Since(start)
		return result, nil
	}

	result.Success = true
	result.Latency = time.Since(start)
	result.Details = map[string]interface{}{
		"available_regions":        len(regions),
		"discovered_resources":     len(resources),
		"supported_resource_types": provider.SupportedResourceTypes(),
	}

	return result, nil
}

// TestServiceConnection tests connection to a specific service
func (ct *ConnectionTesterImpl) TestServiceConnection(ctx context.Context, provider CloudProvider, region, service string) (*ConnectionTestResult, error) {
	start := time.Now()

	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, ct.timeout)
	defer cancel()

	result := &ConnectionTestResult{
		Provider: provider.Name(),
		Region:   region,
		Service:  service,
		Details:  make(map[string]interface{}),
		TestedAt: time.Now(),
	}

	// Test basic connection first
	basicResult, err := ct.TestConnection(testCtx, provider, region)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("basic connection test failed: %v", err)
		result.Latency = time.Since(start)
		return result, nil
	}

	if !basicResult.Success {
		result.Success = false
		result.Error = basicResult.Error
		result.Latency = time.Since(start)
		return result, nil
	}

	// Test service-specific functionality
	serviceResult, err := ct.testSpecificService(testCtx, provider, region, service)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("service test failed: %v", err)
		result.Latency = time.Since(start)
		return result, nil
	}

	result.Success = serviceResult.Success
	result.Error = serviceResult.Error
	result.Latency = time.Since(start)
	result.Details = serviceResult.Details

	return result, nil
}

// TestAllRegions tests connection to all available regions
func (ct *ConnectionTesterImpl) TestAllRegions(ctx context.Context, provider CloudProvider) ([]ConnectionTestResult, error) {
	// Get all available regions
	regions, err := provider.ListRegions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	var results []ConnectionTestResult

	// Test each region
	for _, region := range regions {
		result, err := ct.TestConnection(ctx, provider, region)
		if err != nil {
			// Continue with other regions even if one fails
			results = append(results, ConnectionTestResult{
				Provider: provider.Name(),
				Region:   region,
				Success:  false,
				Error:    fmt.Sprintf("test failed: %v", err),
				TestedAt: time.Now(),
			})
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// TestAllServices tests connection to all supported services in a region
func (ct *ConnectionTesterImpl) TestAllServices(ctx context.Context, provider CloudProvider, region string) ([]ConnectionTestResult, error) {
	// Get supported resource types (which represent services)
	services := provider.SupportedResourceTypes()

	var results []ConnectionTestResult

	// Test each service
	for _, service := range services {
		result, err := ct.TestServiceConnection(ctx, provider, region, service)
		if err != nil {
			// Continue with other services even if one fails
			results = append(results, ConnectionTestResult{
				Provider: provider.Name(),
				Region:   region,
				Service:  service,
				Success:  false,
				Error:    fmt.Sprintf("test failed: %v", err),
				TestedAt: time.Now(),
			})
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

// testSpecificService tests a specific service based on provider and service type
func (ct *ConnectionTesterImpl) testSpecificService(ctx context.Context, provider CloudProvider, region, service string) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{
		Provider: provider.Name(),
		Region:   region,
		Service:  service,
		Details:  make(map[string]interface{}),
		TestedAt: time.Now(),
	}

	// Test service-specific functionality based on provider
	switch provider.Name() {
	case "aws":
		return ct.testAWSService(ctx, provider, region, service)
	case "azure":
		return ct.testAzureService(ctx, provider, region, service)
	case "gcp":
		return ct.testGCPService(ctx, provider, region, service)
	case "digitalocean":
		return ct.testDigitalOceanService(ctx, provider, region, service)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unsupported provider: %s", provider.Name())
		return result, nil
	}
}

// testAWSService tests AWS-specific services
func (ct *ConnectionTesterImpl) testAWSService(ctx context.Context, provider CloudProvider, region, service string) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{
		Provider: provider.Name(),
		Region:   region,
		Service:  service,
		Details:  make(map[string]interface{}),
		TestedAt: time.Now(),
	}

	// Test service-specific functionality
	switch service {
	case "aws_s3_bucket":
		// Test S3 access by trying to list buckets
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("S3 bucket discovery failed: %v", err)
			return result, nil
		}

		s3Count := 0
		for _, resource := range resources {
			if resource.Type == "aws_s3_bucket" {
				s3Count++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"s3_buckets_found": s3Count,
		}

	case "aws_ec2_instance":
		// Test EC2 access by trying to discover instances
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("EC2 instance discovery failed: %v", err)
			return result, nil
		}

		ec2Count := 0
		for _, resource := range resources {
			if resource.Type == "aws_ec2_instance" {
				ec2Count++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"ec2_instances_found": ec2Count,
		}

	default:
		// For other services, just test basic resource discovery
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("resource discovery failed: %v", err)
			return result, nil
		}

		serviceCount := 0
		for _, resource := range resources {
			if resource.Type == service {
				serviceCount++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"resources_found": serviceCount,
		}
	}

	return result, nil
}

// testAzureService tests Azure-specific services
func (ct *ConnectionTesterImpl) testAzureService(ctx context.Context, provider CloudProvider, region, service string) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{
		Provider: provider.Name(),
		Region:   region,
		Service:  service,
		Details:  make(map[string]interface{}),
		TestedAt: time.Now(),
	}

	// Test service-specific functionality
	switch service {
	case "azurerm_storage_account":
		// Test storage account access
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("storage account discovery failed: %v", err)
			return result, nil
		}

		storageCount := 0
		for _, resource := range resources {
			if resource.Type == "azurerm_storage_account" {
				storageCount++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"storage_accounts_found": storageCount,
		}

	case "azurerm_virtual_machine":
		// Test VM access
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("VM discovery failed: %v", err)
			return result, nil
		}

		vmCount := 0
		for _, resource := range resources {
			if resource.Type == "azurerm_virtual_machine" {
				vmCount++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"virtual_machines_found": vmCount,
		}

	default:
		// For other services, just test basic resource discovery
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("resource discovery failed: %v", err)
			return result, nil
		}

		serviceCount := 0
		for _, resource := range resources {
			if resource.Type == service {
				serviceCount++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"resources_found": serviceCount,
		}
	}

	return result, nil
}

// testGCPService tests GCP-specific services
func (ct *ConnectionTesterImpl) testGCPService(ctx context.Context, provider CloudProvider, region, service string) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{
		Provider: provider.Name(),
		Region:   region,
		Service:  service,
		Details:  make(map[string]interface{}),
		TestedAt: time.Now(),
	}

	// Test service-specific functionality
	switch service {
	case "google_storage_bucket":
		// Test storage bucket access
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("storage bucket discovery failed: %v", err)
			return result, nil
		}

		bucketCount := 0
		for _, resource := range resources {
			if resource.Type == "google_storage_bucket" {
				bucketCount++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"storage_buckets_found": bucketCount,
		}

	case "google_compute_instance":
		// Test compute instance access
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("compute instance discovery failed: %v", err)
			return result, nil
		}

		instanceCount := 0
		for _, resource := range resources {
			if resource.Type == "google_compute_instance" {
				instanceCount++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"compute_instances_found": instanceCount,
		}

	default:
		// For other services, just test basic resource discovery
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("resource discovery failed: %v", err)
			return result, nil
		}

		serviceCount := 0
		for _, resource := range resources {
			if resource.Type == service {
				serviceCount++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"resources_found": serviceCount,
		}
	}

	return result, nil
}

// testDigitalOceanService tests DigitalOcean-specific services
func (ct *ConnectionTesterImpl) testDigitalOceanService(ctx context.Context, provider CloudProvider, region, service string) (*ConnectionTestResult, error) {
	result := &ConnectionTestResult{
		Provider: provider.Name(),
		Region:   region,
		Service:  service,
		Details:  make(map[string]interface{}),
		TestedAt: time.Now(),
	}

	// Test service-specific functionality
	switch service {
	case "digitalocean_droplet":
		// Test droplet access
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("droplet discovery failed: %v", err)
			return result, nil
		}

		dropletCount := 0
		for _, resource := range resources {
			if resource.Type == "digitalocean_droplet" {
				dropletCount++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"droplets_found": dropletCount,
		}

	case "digitalocean_volume":
		// Test volume access
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("volume discovery failed: %v", err)
			return result, nil
		}

		volumeCount := 0
		for _, resource := range resources {
			if resource.Type == "digitalocean_volume" {
				volumeCount++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"volumes_found": volumeCount,
		}

	default:
		// For other services, just test basic resource discovery
		resources, err := provider.DiscoverResources(ctx, region)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("resource discovery failed: %v", err)
			return result, nil
		}

		serviceCount := 0
		for _, resource := range resources {
			if resource.Type == service {
				serviceCount++
			}
		}

		result.Success = true
		result.Details = map[string]interface{}{
			"resources_found": serviceCount,
		}
	}

	return result, nil
}
