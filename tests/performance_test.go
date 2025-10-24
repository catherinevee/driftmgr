package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/catherinevee/driftmgr/internal/websocket"
)

// PerformanceTestSuite represents the performance test suite
type PerformanceTestSuite struct {
	server  *api.Server
	baseURL string
	client  *http.Client
}

// NewPerformanceTestSuite creates a new performance test suite
func NewPerformanceTestSuite() *PerformanceTestSuite {
	// Create server configuration
	config := &api.Config{
		Host:        "localhost",
		Port:        8080,
		AuthEnabled: false, // Disable auth for performance tests
		CORSEnabled: true,
	}

	// Create services
	services := &api.Services{
		WebSocket: websocket.NewService(),
	}

	// Create server
	server := api.NewServer(config, services)

	// Create test server
	testServer := httptest.NewServer(server)

	return &PerformanceTestSuite{
		server:  server,
		baseURL: testServer.URL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func BenchmarkHealthEndpoint(b *testing.B) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := suite.client.Get(suite.baseURL + "/health")
			if err != nil {
				b.Fatalf("Failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
}

func BenchmarkAPIHealthEndpoint(b *testing.B) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := suite.client.Get(suite.baseURL + "/api/v1/health")
			if err != nil {
				b.Fatalf("Failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
}

func BenchmarkBackendListEndpoint(b *testing.B) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := suite.client.Get(suite.baseURL + "/api/v1/backends/list")
			if err != nil {
				b.Fatalf("Failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
}

func BenchmarkResourceListEndpoint(b *testing.B) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := suite.client.Get(suite.baseURL + "/api/v1/resources")
			if err != nil {
				b.Fatalf("Failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
}

func BenchmarkDriftResultsEndpoint(b *testing.B) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := suite.client.Get(suite.baseURL + "/api/v1/drift/results")
			if err != nil {
				b.Fatalf("Failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
}

func BenchmarkWebSocketStatsEndpoint(b *testing.B) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := suite.client.Get(suite.baseURL + "/api/v1/ws/stats")
			if err != nil {
				b.Fatalf("Failed to make request: %v", err)
			}
			resp.Body.Close()
		}
	})
}

func TestConcurrentHealthRequests(t *testing.T) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	concurrency := 100
	requestsPerGoroutine := 10
	totalRequests := concurrency * requestsPerGoroutine

	start := time.Now()
	var wg sync.WaitGroup
	successCount := int64(0)
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				resp, err := suite.client.Get(suite.baseURL + "/health")
				if err == nil && resp.StatusCode == http.StatusOK {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
				if resp != nil {
					resp.Body.Close()
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Completed %d requests in %v", totalRequests, duration)
	t.Logf("Success rate: %.2f%%", float64(successCount)/float64(totalRequests)*100)
	t.Logf("Requests per second: %.2f", float64(totalRequests)/duration.Seconds())

	if successCount != int64(totalRequests) {
		t.Errorf("Expected %d successful requests, got %d", totalRequests, successCount)
	}
}

func TestConcurrentAPIRequests(t *testing.T) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	endpoints := []string{
		"/api/v1/health",
		"/api/v1/version",
		"/api/v1/backends/list",
		"/api/v1/resources",
		"/api/v1/drift/results",
		"/api/v1/ws/stats",
	}

	concurrency := 50
	requestsPerGoroutine := 5
	totalRequests := concurrency * requestsPerGoroutine * len(endpoints)

	start := time.Now()
	var wg sync.WaitGroup
	successCount := int64(0)
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				for _, endpoint := range endpoints {
					resp, err := suite.client.Get(suite.baseURL + endpoint)
					if err == nil && resp.StatusCode == http.StatusOK {
						mu.Lock()
						successCount++
						mu.Unlock()
					}
					if resp != nil {
						resp.Body.Close()
					}
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Completed %d API requests in %v", totalRequests, duration)
	t.Logf("Success rate: %.2f%%", float64(successCount)/float64(totalRequests)*100)
	t.Logf("Requests per second: %.2f", float64(totalRequests)/duration.Seconds())

	if successCount != int64(totalRequests) {
		t.Errorf("Expected %d successful requests, got %d", totalRequests, successCount)
	}
}

func TestMemoryUsage(t *testing.T) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	// Make many requests to test memory usage
	numRequests := 1000
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		resp, err := suite.client.Get(suite.baseURL + "/health")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		resp.Body.Close()
	}

	duration := time.Since(start)
	t.Logf("Completed %d requests in %v", numRequests, duration)
	t.Logf("Average request time: %v", duration/time.Duration(numRequests))
}

func TestResponseTimeDistribution(t *testing.T) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	numRequests := 100
	responseTimes := make([]time.Duration, numRequests)

	for i := 0; i < numRequests; i++ {
		start := time.Now()
		resp, err := suite.client.Get(suite.baseURL + "/health")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		resp.Body.Close()
		responseTimes[i] = time.Since(start)
	}

	// Calculate statistics
	var total time.Duration
	var min, max time.Duration = responseTimes[0], responseTimes[0]

	for _, rt := range responseTimes {
		total += rt
		if rt < min {
			min = rt
		}
		if rt > max {
			max = rt
		}
	}

	avg := total / time.Duration(numRequests)

	t.Logf("Response time statistics:")
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)
	t.Logf("  Average: %v", avg)

	// Check that average response time is reasonable (less than 100ms)
	if avg > 100*time.Millisecond {
		t.Errorf("Average response time %v is too high", avg)
	}
}

func TestWebSocketConnectionPerformance(t *testing.T) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	// Test WebSocket stats endpoint performance
	numRequests := 100
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		resp, err := suite.client.Get(suite.baseURL + "/api/v1/ws/stats")
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}

		// Parse response to ensure it's valid
		var statsResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&statsResp)
		if err != nil {
			t.Fatalf("Failed to decode response %d: %v", i, err)
		}

		if !statsResp["success"].(bool) {
			t.Errorf("Request %d returned unsuccessful response", i)
		}

		resp.Body.Close()
	}

	duration := time.Since(start)
	t.Logf("Completed %d WebSocket stats requests in %v", numRequests, duration)
	t.Logf("Average request time: %v", duration/time.Duration(numRequests))
}

func TestConcurrentPOSTRequests(t *testing.T) {
	suite := NewPerformanceTestSuite()
	defer suite.server.Stop(context.Background())

	concurrency := 20
	requestsPerGoroutine := 5
	totalRequests := concurrency * requestsPerGoroutine

	start := time.Now()
	var wg sync.WaitGroup
	successCount := int64(0)
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				// Test drift detection endpoint
				resp, err := suite.client.Post(
					suite.baseURL+"/api/v1/drift/detect",
					"application/json",
					bytes.NewBuffer([]byte("{}")),
				)
				if err == nil && resp.StatusCode == http.StatusOK {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
				if resp != nil {
					resp.Body.Close()
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Completed %d POST requests in %v", totalRequests, duration)
	t.Logf("Success rate: %.2f%%", float64(successCount)/float64(totalRequests)*100)
	t.Logf("Requests per second: %.2f", float64(totalRequests)/duration.Seconds())

	if successCount != int64(totalRequests) {
		t.Errorf("Expected %d successful requests, got %d", totalRequests, successCount)
	}
}

func TestServerStartupTime(t *testing.T) {
	start := time.Now()

	// Create server configuration
	config := &api.Config{
		Host:        "localhost",
		Port:        8080,
		AuthEnabled: false,
		CORSEnabled: true,
	}

	// Create services
	services := &api.Services{
		WebSocket: websocket.NewService(),
	}

	// Create server
	server := api.NewServer(config, services)
	defer server.Stop(context.Background())

	// Start server
	err := server.Start(context.Background())
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	startupTime := time.Since(start)
	t.Logf("Server startup time: %v", startupTime)

	// Check that startup time is reasonable (less than 1 second)
	if startupTime > 1*time.Second {
		t.Errorf("Server startup time %v is too high", startupTime)
	}
}

func TestServerShutdownTime(t *testing.T) {
	// Create server
	config := &api.Config{
		Host:        "localhost",
		Port:        8080,
		AuthEnabled: false,
		CORSEnabled: true,
	}

	services := &api.Services{
		WebSocket: websocket.NewService(),
	}

	server := api.NewServer(config, services)

	// Start server
	err := server.Start(context.Background())
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Test shutdown time
	start := time.Now()
	err = server.Stop(context.Background())
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	shutdownTime := time.Since(start)
	t.Logf("Server shutdown time: %v", shutdownTime)

	// Check that shutdown time is reasonable (less than 5 seconds)
	if shutdownTime > 5*time.Second {
		t.Errorf("Server shutdown time %v is too high", shutdownTime)
	}
}
