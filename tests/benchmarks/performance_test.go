package benchmarks

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/visualization"
	"github.com/catherinevee/driftmgr/internal/infrastructure/config"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PerformanceTestSuite provides comprehensive performance testing
type PerformanceTestSuite struct {
	config       *config.Config
	discoverer   *discovery.EnhancedDiscoverer
	stateManager *state.RemoteStateManager
	visualizer   *visualization.DiagramGenerator
	tempDir      string
	testData     *PerformanceTestData
}

// PerformanceTestData contains test data for performance testing
type PerformanceTestData struct {
	SmallStateFile  string
	MediumStateFile string
	LargeStateFile  string
	HugeStateFile   string
	MockResources   []models.Resource
}

// MemoryStats tracks memory usage during tests
type MemoryStats struct {
	InitialHeap  uint64
	PeakHeap     uint64
	FinalHeap    uint64
	AllocatedMem uint64
	GCCycles     uint32
	StartTime    time.Time
	EndTime      time.Time
}

// PerformanceMetrics tracks various performance metrics
type PerformanceMetrics struct {
	Duration           time.Duration
	ResourcesProcessed int
	ThroughputRPS      float64
	Memory             MemoryStats
	CPUUsage           float64
	GoroutineCount     int
	Errors             int
}

// SetupPerformanceTest initializes the performance test environment
func SetupPerformanceTest(t *testing.T) *PerformanceTestSuite {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "driftmgr-perf-*")
	require.NoError(t, err)

	// Initialize configuration optimized for performance testing
	config := &config.Config{
		Discovery: config.DiscoveryConfig{
			EnableCaching:    true,
			CacheTimeout:     10 * time.Minute,
			MaxConcurrency:   runtime.NumCPU() * 2,
			EnableValidation: true,
			EnableMetrics:    true,
			Providers: []config.ProviderConfig{
				{
					Name:    "aws",
					Enabled: true,
					Regions: []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"},
				},
				{
					Name:    "azure",
					Enabled: true,
					Regions: []string{"eastus", "westus2", "westeurope", "southeastasia"},
				},
				{
					Name:    "gcp",
					Enabled: true,
					Regions: []string{"us-central1", "us-west1", "europe-west1", "asia-southeast1"},
				},
			},
		},
		Database: config.DatabaseConfig{
			Driver:      "sqlite",
			Host:        filepath.Join(tempDir, "perf_test.db"),
			MaxOpenConn: 100,
			MaxIdleConn: 10,
		},
		Server: config.ServerConfig{
			Port:           8080,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxRequestSize: 100 * 1024 * 1024, // 100MB
		},
	}

	// Initialize components
	suite := &PerformanceTestSuite{
		config:       config,
		discoverer:   discovery.NewEnhancedDiscoverer(config),
		stateManager: state.NewRemoteStateManager(config),
		visualizer:   visualization.NewDiagramGenerator(config),
		tempDir:      tempDir,
	}

	// Setup test data
	suite.setupPerformanceTestData(t)

	return suite
}

// Cleanup removes temporary test files
func (suite *PerformanceTestSuite) Cleanup() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// setupPerformanceTestData creates test data of various sizes
func (suite *PerformanceTestSuite) setupPerformanceTestData(t *testing.T) {
	suite.testData = &PerformanceTestData{}

	// Create state files of different sizes
	suite.testData.SmallStateFile = suite.createStateFile(t, "small", 10)    // 10 resources
	suite.testData.MediumStateFile = suite.createStateFile(t, "medium", 100) // 100 resources
	suite.testData.LargeStateFile = suite.createStateFile(t, "large", 1000)  // 1,000 resources
	suite.testData.HugeStateFile = suite.createStateFile(t, "huge", 10000)   // 10,000 resources

	// Create mock resources for discovery testing
	suite.testData.MockResources = suite.generateMockResources(1000)
}

// createStateFile generates a Terraform state file with specified number of resources
func (suite *PerformanceTestSuite) createStateFile(t *testing.T, size string, resourceCount int) string {
	filename := fmt.Sprintf("terraform_%s.tfstate", size)
	filepath := filepath.Join(suite.tempDir, filename)

	state := map[string]interface{}{
		"version":           4,
		"terraform_version": "1.0.0",
		"serial":            1,
		"lineage":           fmt.Sprintf("test-lineage-%s", size),
		"outputs":           map[string]interface{}{},
		"resources":         suite.generateStateResources(resourceCount),
	}

	data, err := json.MarshalIndent(state, "", "  ")
	require.NoError(t, err)

	err = os.WriteFile(filepath, data, 0644)
	require.NoError(t, err)

	return filepath
}

// generateStateResources creates mock Terraform resources
func (suite *PerformanceTestSuite) generateStateResources(count int) []map[string]interface{} {
	resources := make([]map[string]interface{}, count)

	resourceTypes := []string{
		"aws_instance", "aws_ebs_volume", "aws_vpc", "aws_subnet", "aws_security_group",
		"azurerm_virtual_machine", "azurerm_storage_account", "azurerm_virtual_network",
		"google_compute_instance", "google_storage_bucket", "google_compute_network",
	}

	regions := []string{
		"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1",
		"eastus", "westus2", "westeurope", "southeastasia",
		"us-central1", "us-west1", "europe-west1", "asia-southeast1",
	}

	for i := 0; i < count; i++ {
		resourceType := resourceTypes[i%len(resourceTypes)]
		region := regions[i%len(regions)]

		resource := map[string]interface{}{
			"mode":     "managed",
			"type":     resourceType,
			"name":     fmt.Sprintf("resource_%d", i),
			"provider": fmt.Sprintf("provider[\"%s\"]", getProviderForType(resourceType)),
			"instances": []map[string]interface{}{
				{
					"schema_version": 1,
					"attributes": map[string]interface{}{
						"id":     fmt.Sprintf("%s-%d", resourceType, i),
						"name":   fmt.Sprintf("test-resource-%d", i),
						"region": region,
						"tags": map[string]string{
							"Name":        fmt.Sprintf("test-resource-%d", i),
							"Environment": []string{"production", "staging", "development"}[i%3],
							"Team":        []string{"backend", "frontend", "devops", "data"}[i%4],
							"Project":     fmt.Sprintf("project-%d", i%10),
						},
					},
				},
			},
		}

		resources[i] = resource
	}

	return resources
}

// generateMockResources creates mock cloud resources for discovery testing
func (suite *PerformanceTestSuite) generateMockResources(count int) []models.Resource {
	resources := make([]models.Resource, count)

	providers := []string{"aws", "azure", "gcp"}
	resourceTypes := map[string][]string{
		"aws":   {"aws_instance", "aws_ebs_volume", "aws_s3_bucket", "aws_vpc", "aws_subnet"},
		"azure": {"azurerm_virtual_machine", "azurerm_storage_account", "azurerm_virtual_network"},
		"gcp":   {"google_compute_instance", "google_storage_bucket", "google_compute_network"},
	}

	regions := map[string][]string{
		"aws":   {"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"},
		"azure": {"eastus", "westus2", "westeurope", "southeastasia"},
		"gcp":   {"us-central1", "us-west1", "europe-west1", "asia-southeast1"},
	}

	for i := 0; i < count; i++ {
		provider := providers[i%len(providers)]
		types := resourceTypes[provider]
		regionList := regions[provider]

		resource := models.Resource{
			ID:       fmt.Sprintf("%s-%d", provider, i),
			Name:     fmt.Sprintf("resource-%d", i),
			Type:     types[i%len(types)],
			Provider: provider,
			Region:   regionList[i%len(regionList)],
			State:    []string{"running", "stopped", "pending"}[i%3],
			Tags: map[string]string{
				"Name":        fmt.Sprintf("resource-%d", i),
				"Environment": []string{"production", "staging", "development"}[i%3],
				"Team":        []string{"backend", "frontend", "devops", "data"}[i%4],
			},
			Created: time.Now().Add(-time.Duration(rand.Intn(720)) * time.Hour), // Random time in last 30 days
			Updated: time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour),  // Random time in last day
		}

		resources[i] = resource
	}

	return resources
}

// getProviderForType returns the provider name for a given resource type
func getProviderForType(resourceType string) string {
	if len(resourceType) > 3 {
		switch resourceType[:3] {
		case "aws":
			return "registry.terraform.io/hashicorp/aws"
		case "azu":
			return "registry.terraform.io/hashicorp/azurerm"
		case "goo":
			return "registry.terraform.io/hashicorp/google"
		}
	}
	return "registry.terraform.io/hashicorp/aws"
}

// startMemoryTracking begins tracking memory usage
func startMemoryTracking() MemoryStats {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)

	return MemoryStats{
		InitialHeap: m.HeapAlloc,
		PeakHeap:    m.HeapAlloc,
		StartTime:   time.Now(),
		GCCycles:    m.NumGC,
	}
}

// updateMemoryTracking updates peak memory usage
func updateMemoryTracking(stats *MemoryStats) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if m.HeapAlloc > stats.PeakHeap {
		stats.PeakHeap = m.HeapAlloc
	}
}

// finishMemoryTracking completes memory tracking
func finishMemoryTracking(stats *MemoryStats) {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)

	stats.FinalHeap = m.HeapAlloc
	stats.AllocatedMem = m.TotalAlloc
	stats.GCCycles = m.NumGC - stats.GCCycles
	stats.EndTime = time.Now()
}

// BENCHMARK TESTS

// BenchmarkSmallStateFileProcessing tests processing of small state files
func BenchmarkSmallStateFileProcessing(b *testing.B) {
	suite := SetupPerformanceTest(b)
	defer suite.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := suite.stateManager.LoadStateFile(suite.testData.SmallStateFile)
		if err != nil {
			b.Fatalf("Failed to load small state file: %v", err)
		}
	}
}

// BenchmarkMediumStateFileProcessing tests processing of medium state files
func BenchmarkMediumStateFileProcessing(b *testing.B) {
	suite := SetupPerformanceTest(b)
	defer suite.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := suite.stateManager.LoadStateFile(suite.testData.MediumStateFile)
		if err != nil {
			b.Fatalf("Failed to load medium state file: %v", err)
		}
	}
}

// BenchmarkLargeStateFileProcessing tests processing of large state files
func BenchmarkLargeStateFileProcessing(b *testing.B) {
	suite := SetupPerformanceTest(b)
	defer suite.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := suite.stateManager.LoadStateFile(suite.testData.LargeStateFile)
		if err != nil {
			b.Fatalf("Failed to load large state file: %v", err)
		}
	}
}

// BenchmarkHugeStateFileProcessing tests processing of huge state files
func BenchmarkHugeStateFileProcessing(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping huge state file test in short mode")
	}

	suite := SetupPerformanceTest(b)
	defer suite.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := suite.stateManager.LoadStateFile(suite.testData.HugeStateFile)
		if err != nil {
			b.Fatalf("Failed to load huge state file: %v", err)
		}
	}
}

// BenchmarkDiscoverySequential tests sequential resource discovery
func BenchmarkDiscoverySequential(b *testing.B) {
	suite := SetupPerformanceTest(b)
	defer suite.Cleanup()

	ctx := context.Background()
	req := &models.DiscoveryRequest{
		Provider: "aws",
		Regions:  []string{"us-east-1"},
		Account:  "123456789012",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := suite.discoverer.DiscoverResources(ctx, req)
		if err != nil && !isCredentialError(err) {
			b.Fatalf("Discovery failed: %v", err)
		}
	}
}

// BenchmarkDiscoveryParallel tests parallel resource discovery
func BenchmarkDiscoveryParallel(b *testing.B) {
	suite := SetupPerformanceTest(b)
	defer suite.Cleanup()

	ctx := context.Background()
	providers := []string{"aws", "azure", "gcp"}
	regions := map[string][]string{
		"aws":   {"us-east-1", "us-west-2"},
		"azure": {"eastus", "westus2"},
		"gcp":   {"us-central1", "us-west1"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for _, provider := range providers {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				req := &models.DiscoveryRequest{
					Provider: p,
					Regions:  regions[p],
					Account:  fmt.Sprintf("test-%s", p),
				}
				_, err := suite.discoverer.DiscoverResources(ctx, req)
				if err != nil && !isCredentialError(err) {
					b.Errorf("Discovery failed for %s: %v", p, err)
				}
			}(provider)
		}
		wg.Wait()
	}
}

// BenchmarkMemoryUsage tests memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	suite := SetupPerformanceTest(b)
	defer suite.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memStats := startMemoryTracking()

		// Process multiple state files
		for _, stateFile := range []string{
			suite.testData.SmallStateFile,
			suite.testData.MediumStateFile,
			suite.testData.LargeStateFile,
		} {
			_, err := suite.stateManager.LoadStateFile(stateFile)
			if err != nil {
				b.Fatalf("Failed to load state file: %v", err)
			}
			updateMemoryTracking(&memStats)
		}

		finishMemoryTracking(&memStats)

		// Report memory usage if needed
		if i == 0 {
			b.Logf("Memory usage - Initial: %d KB, Peak: %d KB, Final: %d KB",
				memStats.InitialHeap/1024, memStats.PeakHeap/1024, memStats.FinalHeap/1024)
		}
	}
}

// BenchmarkConcurrentStateProcessing tests concurrent state file processing
func BenchmarkConcurrentStateProcessing(b *testing.B) {
	suite := SetupPerformanceTest(b)
	defer suite.Cleanup()

	stateFiles := []string{
		suite.testData.SmallStateFile,
		suite.testData.MediumStateFile,
		suite.testData.LargeStateFile,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for _, stateFile := range stateFiles {
			wg.Add(1)
			go func(file string) {
				defer wg.Done()
				_, err := suite.stateManager.LoadStateFile(file)
				if err != nil {
					b.Errorf("Failed to load state file %s: %v", file, err)
				}
			}(stateFile)
		}
		wg.Wait()
	}
}

// PERFORMANCE TESTS

// TestStateFileProcessingPerformance measures state file processing performance
func TestStateFileProcessingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	suite := SetupPerformanceTest(t)
	defer suite.Cleanup()

	testCases := []struct {
		name      string
		stateFile string
		maxTime   time.Duration
	}{
		{"Small State File", suite.testData.SmallStateFile, 100 * time.Millisecond},
		{"Medium State File", suite.testData.MediumStateFile, 500 * time.Millisecond},
		{"Large State File", suite.testData.LargeStateFile, 2 * time.Second},
		{"Huge State File", suite.testData.HugeStateFile, 10 * time.Second},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			memStats := startMemoryTracking()
			start := time.Now()

			result, err := suite.stateManager.LoadStateFile(tc.stateFile)

			duration := time.Since(start)
			finishMemoryTracking(&memStats)

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Less(t, duration, tc.maxTime, "Processing took too long")

			t.Logf("%s - Duration: %v, Resources: %d, Memory Peak: %d KB",
				tc.name, duration, len(result.Resources), memStats.PeakHeap/1024)
		})
	}
}

// TestDiscoveryPerformance measures discovery performance across providers
func TestDiscoveryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping discovery performance test in short mode")
	}

	suite := SetupPerformanceTest(t)
	defer suite.Cleanup()

	ctx := context.Background()
	providers := []string{"aws", "azure", "gcp"}
	maxDuration := 30 * time.Second

	for _, provider := range providers {
		t.Run(fmt.Sprintf("Discovery_%s", provider), func(t *testing.T) {
			req := &models.DiscoveryRequest{
				Provider: provider,
				Regions:  []string{"us-east-1", "eastus", "us-central1"}[:1], // One region each
				Account:  fmt.Sprintf("test-%s", provider),
			}

			memStats := startMemoryTracking()
			start := time.Now()

			result, err := suite.discoverer.DiscoverResources(ctx, req)

			duration := time.Since(start)
			finishMemoryTracking(&memStats)

			if err != nil {
				if isCredentialError(err) {
					t.Skip("Credentials not available for", provider)
					return
				}
				require.NoError(t, err)
			}

			assert.NotNil(t, result)
			assert.Less(t, duration, maxDuration, "Discovery took too long")

			t.Logf("%s Discovery - Duration: %v, Resources: %d, Memory Peak: %d KB",
				provider, duration, len(result.Resources), memStats.PeakHeap/1024)
		})
	}
}

// TestConcurrencyPerformance tests performance under high concurrency
func TestConcurrencyPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency performance test in short mode")
	}

	suite := SetupPerformanceTest(t)
	defer suite.Cleanup()

	concurrencyLevels := []int{1, 5, 10, 20, 50}

	for _, concurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(t *testing.T) {
			memStats := startMemoryTracking()
			start := time.Now()

			var wg sync.WaitGroup
			errorChan := make(chan error, concurrency)

			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()

					// Each goroutine processes a state file
					stateFile := suite.testData.MediumStateFile
					_, err := suite.stateManager.LoadStateFile(stateFile)
					if err != nil {
						errorChan <- fmt.Errorf("goroutine %d failed: %v", id, err)
					}
				}(i)
			}

			wg.Wait()
			close(errorChan)

			duration := time.Since(start)
			finishMemoryTracking(&memStats)

			// Check for errors
			var errors []error
			for err := range errorChan {
				errors = append(errors, err)
			}

			assert.Empty(t, errors, "Concurrent processing should not have errors")

			throughput := float64(concurrency) / duration.Seconds()

			t.Logf("Concurrency %d - Duration: %v, Throughput: %.2f ops/sec, Memory Peak: %d KB",
				concurrency, duration, throughput, memStats.PeakHeap/1024)
		})
	}
}

// TestMemoryLeakDetection checks for memory leaks
func TestMemoryLeakDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	suite := SetupPerformanceTest(t)
	defer suite.Cleanup()

	iterations := 100
	memoryReadings := make([]uint64, iterations)

	// Perform operations and measure memory
	for i := 0; i < iterations; i++ {
		_, err := suite.stateManager.LoadStateFile(suite.testData.MediumStateFile)
		require.NoError(t, err)

		if i%10 == 0 {
			runtime.GC()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			memoryReadings[i] = m.HeapAlloc
		}
	}

	// Check for consistent memory growth (indicating a leak)
	nonZeroReadings := []uint64{}
	for _, reading := range memoryReadings {
		if reading > 0 {
			nonZeroReadings = append(nonZeroReadings, reading)
		}
	}

	if len(nonZeroReadings) >= 3 {
		first := nonZeroReadings[0]
		last := nonZeroReadings[len(nonZeroReadings)-1]
		growthRatio := float64(last) / float64(first)

		// Memory shouldn't grow more than 50% over the test
		assert.Less(t, growthRatio, 1.5, "Potential memory leak detected")

		t.Logf("Memory readings - First: %d KB, Last: %d KB, Growth: %.2fx",
			first/1024, last/1024, growthRatio)
	}
}

// TestResourceProcessingThroughput measures resource processing throughput
func TestResourceProcessingThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput test in short mode")
	}

	suite := SetupPerformanceTest(t)
	defer suite.Cleanup()

	resourceCounts := []int{100, 500, 1000, 5000}

	for _, count := range resourceCounts {
		t.Run(fmt.Sprintf("Resources_%d", count), func(t *testing.T) {
			resources := suite.testData.MockResources[:count]

			memStats := startMemoryTracking()
			start := time.Now()

			// Simulate processing resources
			processed := 0
			for _, resource := range resources {
				// Simulate processing time
				_ = resource.ID + resource.Name + resource.Type
				processed++
			}

			duration := time.Since(start)
			finishMemoryTracking(&memStats)

			throughput := float64(processed) / duration.Seconds()

			assert.Equal(t, count, processed)

			t.Logf("Resources %d - Duration: %v, Throughput: %.0f resources/sec, Memory: %d KB",
				count, duration, throughput, memStats.PeakHeap/1024)
		})
	}
}

// LOAD TESTS

// TestHighVolumeStateFiles tests processing of many state files
func TestHighVolumeStateFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high volume test in short mode")
	}

	suite := SetupPerformanceTest(t)
	defer suite.Cleanup()

	// Create multiple state files
	stateFiles := make([]string, 50)
	for i := 0; i < 50; i++ {
		stateFiles[i] = suite.createStateFile(t, fmt.Sprintf("volume_%d", i), 50)
	}

	memStats := startMemoryTracking()
	start := time.Now()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrency
	errors := make(chan error, len(stateFiles))

	for _, stateFile := range stateFiles {
		wg.Add(1)
		go func(file string) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			_, err := suite.stateManager.LoadStateFile(file)
			if err != nil {
				errors <- err
			}
		}(stateFile)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)
	finishMemoryTracking(&memStats)

	// Check for errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	assert.Empty(t, errorList, "High volume processing should not have errors")

	throughput := float64(len(stateFiles)) / duration.Seconds()

	t.Logf("High Volume - Files: %d, Duration: %v, Throughput: %.2f files/sec, Memory Peak: %d KB",
		len(stateFiles), duration, throughput, memStats.PeakHeap/1024)
}

// TestStressDiscovery tests discovery under stress conditions
func TestStressDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	suite := SetupPerformanceTest(t)
	defer suite.Cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	providers := []string{"aws", "azure", "gcp"}
	regions := map[string][]string{
		"aws":   {"us-east-1", "us-west-2", "eu-west-1"},
		"azure": {"eastus", "westus2", "westeurope"},
		"gcp":   {"us-central1", "us-west1", "europe-west1"},
	}

	iterations := 20
	memStats := startMemoryTracking()
	start := time.Now()

	var wg sync.WaitGroup
	errorChan := make(chan error, iterations*len(providers))

	for i := 0; i < iterations; i++ {
		for _, provider := range providers {
			wg.Add(1)
			go func(p string, iter int) {
				defer wg.Done()

				req := &models.DiscoveryRequest{
					Provider: p,
					Regions:  regions[p][:1], // Use first region to limit scope
					Account:  fmt.Sprintf("test-%s-%d", p, iter),
				}

				_, err := suite.discoverer.DiscoverResources(ctx, req)
				if err != nil && !isCredentialError(err) {
					errorChan <- fmt.Errorf("iteration %d, provider %s: %v", iter, p, err)
				}
			}(provider, i)
		}
	}

	wg.Wait()
	close(errorChan)

	duration := time.Since(start)
	finishMemoryTracking(&memStats)

	// Count errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	totalOperations := iterations * len(providers)
	successRate := float64(totalOperations-len(errors)) / float64(totalOperations) * 100

	t.Logf("Stress Test - Operations: %d, Duration: %v, Success Rate: %.1f%%, Memory Peak: %d KB",
		totalOperations, duration, successRate, memStats.PeakHeap/1024)

	// Allow some failures due to credentials, but expect majority to succeed
	assert.GreaterOrEqual(t, successRate, 50.0, "Success rate should be at least 50%")
}

// Helper function to check for credential errors
func isCredentialError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	credentialErrors := []string{
		"credential", "authentication", "access denied", "unauthorized",
		"no valid credential", "unable to load credentials", "invalid credentials",
		"token", "login", "permission denied",
	}

	for _, credErr := range credentialErrors {
		if contains(errMsg, credErr) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Example usage test
func TestPerformanceTestSuiteUsage(t *testing.T) {
	suite := SetupPerformanceTest(t)
	defer suite.Cleanup()

	// Verify test data was created
	assert.FileExists(t, suite.testData.SmallStateFile)
	assert.FileExists(t, suite.testData.MediumStateFile)
	assert.FileExists(t, suite.testData.LargeStateFile)
	assert.FileExists(t, suite.testData.HugeStateFile)

	assert.Len(t, suite.testData.MockResources, 1000)

	// Test basic functionality
	result, err := suite.stateManager.LoadStateFile(suite.testData.SmallStateFile)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 10, len(result.Resources))
}
