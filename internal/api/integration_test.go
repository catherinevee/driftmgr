package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/websocket"
)

func TestAPIServerIntegration(t *testing.T) {
	// Create server configuration
	config := &Config{
		Host:        "localhost",
		Port:        8080,
		AuthEnabled: true,
	}

	// Create services
	services := &Services{
		WebSocket: websocket.NewService(),
	}

	// Create server
	server := NewServer(config, services)
	defer server.Stop(context.Background())

	// Test server creation
	if server == nil {
		t.Fatal("Server should not be nil")
	}

	if server.config == nil {
		t.Fatal("Server config should not be nil")
	}

	if server.services == nil {
		t.Fatal("Server services should not be nil")
	}
}

func TestHealthEndpoint(t *testing.T) {
	// Create server
	config := &Config{
		Host:        "localhost",
		Port:        8080,
		AuthEnabled: false,
	}
	services := &Services{}
	server := NewServer(config, services)
	defer server.Stop(context.Background())

	// Create request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Call handler
	server.handleHealth(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response body
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
}

func TestVersionEndpoint(t *testing.T) {
	// Create server
	config := &Config{
		Host:        "localhost",
		Port:        8080,
		AuthEnabled: false,
	}
	services := &Services{}
	server := NewServer(config, services)
	defer server.Stop(context.Background())

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/version", nil)
	w := httptest.NewRecorder()

	// Call handler
	server.handleVersion(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response body
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["version"] == nil {
		t.Error("Expected version field in response")
	}
}

func TestWebSocketStatsEndpoint(t *testing.T) {
	// Create server with WebSocket service
	config := &Config{
		Host:        "localhost",
		Port:        8080,
		AuthEnabled: false,
	}

	services := &Services{
		WebSocket: websocket.NewService(),
	}

	server := NewServer(config, services)
	defer server.Stop(context.Background())

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/ws/stats", nil)
	w := httptest.NewRecorder()

	// Call handler
	server.handleWebSocketStats(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response body
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !response["success"].(bool) {
		t.Error("Expected success to be true")
	}

	data := response["data"].(map[string]interface{})
	if data["total_connections"] == nil {
		t.Error("Expected total_connections field in response")
	}
}

func TestWebSocketStatsEndpointNoService(t *testing.T) {
	// Create server without WebSocket service
	config := &Config{
		Host:        "localhost",
		Port:        8080,
		AuthEnabled: false,
	}
	services := &Services{}
	server := NewServer(config, services)
	defer server.Stop(context.Background())

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/ws/stats", nil)
	w := httptest.NewRecorder()

	// Call handler
	server.handleWebSocketStats(w, req)

	// Check response
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	// Check response body
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["success"].(bool) {
		t.Error("Expected success to be false")
	}

	if response["error"] != "WebSocket service not available" {
		t.Errorf("Expected error message, got %v", response["error"])
	}
}

func TestBackendHandlersIntegration(t *testing.T) {
	// Create backend handlers
	handlers := NewBackendHandlers()

	// Test ListBackends
	req := httptest.NewRequest("GET", "/api/v1/backends/list", nil)
	w := httptest.NewRecorder()

	handlers.ListBackends(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test DiscoverBackends
	req = httptest.NewRequest("POST", "/api/v1/backends/discover", nil)
	w = httptest.NewRecorder()

	handlers.DiscoverBackends(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestStateHandlersIntegration(t *testing.T) {
	// Create state handlers
	handlers := NewStateHandlers()

	// Test ListStateFiles
	req := httptest.NewRequest("GET", "/api/v1/state/list", nil)
	w := httptest.NewRecorder()

	handlers.ListStateFiles(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test GetStateDetails
	req = httptest.NewRequest("GET", "/api/v1/state/details", nil)
	w = httptest.NewRecorder()

	handlers.GetStateDetails(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestResourceHandlersIntegration(t *testing.T) {
	// Create resource handlers
	handlers := NewResourceHandlers()

	// Test ListResources
	req := httptest.NewRequest("GET", "/api/v1/resources", nil)
	w := httptest.NewRecorder()

	handlers.ListResources(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test SearchResources
	req = httptest.NewRequest("GET", "/api/v1/resources/search?q=test", nil)
	w = httptest.NewRecorder()

	handlers.SearchResources(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestDriftHandlersIntegration(t *testing.T) {
	// Create drift handlers
	handlers := NewDriftHandlers()

	// Test DetectDrift
	req := httptest.NewRequest("POST", "/api/v1/drift/detect", nil)
	w := httptest.NewRecorder()

	handlers.DetectDrift(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test ListDriftResults
	req = httptest.NewRequest("GET", "/api/v1/drift/results", nil)
	w = httptest.NewRecorder()

	handlers.ListDriftResults(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAuthenticationIntegration(t *testing.T) {
	// Create server with authentication
	config := &Config{
		Host:        "localhost",
		Port:        8080,
		AuthEnabled: true,
	}

	services := &Services{}

	server := NewServer(config, services)
	defer server.Stop(context.Background())

	// Create a test server to handle the request
	testServer := httptest.NewServer(server)
	defer testServer.Close()

	// Make request to test server
	resp, err := http.Get(testServer.URL + "/api/v1/auth/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestCORSIntegration(t *testing.T) {
	// Create server with CORS enabled
	config := &Config{
		Host:        "localhost",
		Port:        8080,
		CORSEnabled: true,
	}

	server := NewServer(config, nil)
	defer server.Stop(context.Background())

	// Create OPTIONS request
	req := httptest.NewRequest("OPTIONS", "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	// Call server directly
	server.ServeHTTP(w, req)

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("Expected CORS headers to be set")
	}
}

func TestRateLimitingIntegration(t *testing.T) {
	// Create server with rate limiting
	config := &Config{
		Host:             "localhost",
		Port:             8080,
		RateLimitEnabled: true,
		RateLimitRPS:     10,
	}

	server := NewServer(config, nil)
	defer server.Stop(context.Background())

	// Make multiple requests quickly
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i, w.Code)
		}
	}
}

func TestServerLifecycle(t *testing.T) {
	// Create server
	config := &Config{
		Host:        "localhost",
		Port:        8080,
		AuthEnabled: false,
	}
	services := &Services{}
	server := NewServer(config, services)

	// Test start
	ctx := context.Background()
	err := server.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Test stop
	err = server.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
}

func TestRouterIntegration(t *testing.T) {
	// Create router
	router := &Router{
		routes: make(map[string]map[string]http.HandlerFunc),
	}

	// Test route registration
	router.GET("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	})

	// Test route handling
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "test" {
		t.Errorf("Expected body 'test', got %s", w.Body.String())
	}
}
