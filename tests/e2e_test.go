package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/catherinevee/driftmgr/internal/websocket"
)

// E2ETestSuite represents the end-to-end test suite
type E2ETestSuite struct {
	server    *api.Server
	baseURL   string
	client    *http.Client
	authToken string
}

// NewE2ETestSuite creates a new E2E test suite
func NewE2ETestSuite() *E2ETestSuite {
	// Create server configuration
	config := &api.Config{
		Host:        "localhost",
		Port:        8080,
		AuthEnabled: true,
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
	defer testServer.Close()

	return &E2ETestSuite{
		server:  server,
		baseURL: testServer.URL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func TestE2EHealthCheck(t *testing.T) {
	suite := NewE2ETestSuite()

	// Test health endpoint
	resp, err := suite.client.Get(suite.baseURL + "/health")
	if err != nil {
		t.Fatalf("Failed to make health check request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test API health endpoint
	resp, err = suite.client.Get(suite.baseURL + "/api/v1/health")
	if err != nil {
		t.Fatalf("Failed to make API health check request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestE2EUserRegistrationAndLogin(t *testing.T) {
	suite := NewE2ETestSuite()

	// Test user registration
	registerData := map[string]interface{}{
		"username": "testuser",
		"email":    "test@example.com",
		"password": "password123",
	}

	jsonData, _ := json.Marshal(registerData)
	resp, err := suite.client.Post(
		suite.baseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Test user login
	loginData := map[string]interface{}{
		"username": "testuser",
		"password": "password123",
	}

	jsonData, _ = json.Marshal(loginData)
	resp, err = suite.client.Post(
		suite.baseURL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		t.Fatalf("Failed to login user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse login response
	var loginResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}

	// Extract token
	data := loginResp["data"].(map[string]interface{})
	suite.authToken = data["access_token"].(string)

	if suite.authToken == "" {
		t.Error("Access token should not be empty")
	}
}

func TestE2EProtectedEndpoints(t *testing.T) {
	suite := NewE2ETestSuite()

	// First register and login to get token
	suite.registerAndLogin(t)

	// Test protected endpoint without token
	resp, err := suite.client.Get(suite.baseURL + "/api/v1/auth/profile")
	if err != nil {
		t.Fatalf("Failed to make profile request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}

	// Test protected endpoint with token
	req, _ := http.NewRequest("GET", suite.baseURL+"/api/v1/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+suite.authToken)

	resp, err = suite.client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make authenticated profile request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestE2EBackendManagement(t *testing.T) {
	suite := NewE2ETestSuite()

	// Test list backends
	resp, err := suite.client.Get(suite.baseURL + "/api/v1/backends/list")
	if err != nil {
		t.Fatalf("Failed to list backends: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test discover backends
	resp, err = suite.client.Post(
		suite.baseURL+"/api/v1/backends/discover",
		"application/json",
		bytes.NewBuffer([]byte("{}")),
	)
	if err != nil {
		t.Fatalf("Failed to discover backends: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestE2EStateManagement(t *testing.T) {
	suite := NewE2ETestSuite()

	// Test list state files
	resp, err := suite.client.Get(suite.baseURL + "/api/v1/state/list")
	if err != nil {
		t.Fatalf("Failed to list state files: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test get state details
	resp, err = suite.client.Get(suite.baseURL + "/api/v1/state/details")
	if err != nil {
		t.Fatalf("Failed to get state details: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestE2EResourceManagement(t *testing.T) {
	suite := NewE2ETestSuite()

	// Test list resources
	resp, err := suite.client.Get(suite.baseURL + "/api/v1/resources")
	if err != nil {
		t.Fatalf("Failed to list resources: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test search resources
	resp, err = suite.client.Get(suite.baseURL + "/api/v1/resources/search?q=test")
	if err != nil {
		t.Fatalf("Failed to search resources: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestE2EDriftDetection(t *testing.T) {
	suite := NewE2ETestSuite()

	// Test detect drift
	resp, err := suite.client.Post(
		suite.baseURL+"/api/v1/drift/detect",
		"application/json",
		bytes.NewBuffer([]byte("{}")),
	)
	if err != nil {
		t.Fatalf("Failed to detect drift: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test list drift results
	resp, err = suite.client.Get(suite.baseURL + "/api/v1/drift/results")
	if err != nil {
		t.Fatalf("Failed to list drift results: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestE2EWebSocketConnection(t *testing.T) {
	suite := NewE2ETestSuite()

	// Test WebSocket stats endpoint
	resp, err := suite.client.Get(suite.baseURL + "/api/v1/ws/stats")
	if err != nil {
		t.Fatalf("Failed to get WebSocket stats: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse response
	var statsResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&statsResp)
	if err != nil {
		t.Fatalf("Failed to decode stats response: %v", err)
	}

	if !statsResp["success"].(bool) {
		t.Error("Expected success to be true")
	}
}

func TestE2EDashboardAccess(t *testing.T) {
	suite := NewE2ETestSuite()

	// Test dashboard access
	resp, err := suite.client.Get(suite.baseURL + "/dashboard")
	if err != nil {
		t.Fatalf("Failed to access dashboard: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test login page access
	resp, err = suite.client.Get(suite.baseURL + "/login")
	if err != nil {
		t.Fatalf("Failed to access login page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestE2EAPIVersion(t *testing.T) {
	suite := NewE2ETestSuite()

	// Test API version endpoint
	resp, err := suite.client.Get(suite.baseURL + "/api/v1/version")
	if err != nil {
		t.Fatalf("Failed to get API version: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse response
	var versionResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&versionResp)
	if err != nil {
		t.Fatalf("Failed to decode version response: %v", err)
	}

	if versionResp["version"] == nil {
		t.Error("Expected version field in response")
	}
}

func TestE2ECORSHeaders(t *testing.T) {
	suite := NewE2ETestSuite()

	// Test CORS preflight request
	req, _ := http.NewRequest("OPTIONS", suite.baseURL+"/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")

	resp, err := suite.client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make CORS preflight request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check CORS headers
	if resp.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Error("Expected CORS headers to be set")
	}
}

func TestE2EConcurrentRequests(t *testing.T) {
	suite := NewE2ETestSuite()

	// Make multiple concurrent requests
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			resp, err := suite.client.Get(suite.baseURL + "/health")
			if err != nil {
				t.Errorf("Failed to make concurrent request: %v", err)
				done <- false
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
				done <- false
				return
			}

			done <- true
		}()
	}

	// Wait for all requests to complete
	successCount := 0
	for i := 0; i < concurrency; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != concurrency {
		t.Errorf("Expected %d successful requests, got %d", concurrency, successCount)
	}
}

// Helper method to register and login a user
func (suite *E2ETestSuite) registerAndLogin(t *testing.T) {
	// Register user
	registerData := map[string]interface{}{
		"username": "testuser",
		"email":    "test@example.com",
		"password": "password123",
	}

	jsonData, _ := json.Marshal(registerData)
	resp, err := suite.client.Post(
		suite.baseURL+"/api/v1/auth/register",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}
	resp.Body.Close()

	// Login user
	loginData := map[string]interface{}{
		"username": "testuser",
		"password": "password123",
	}

	jsonData, _ = json.Marshal(loginData)
	resp, err = suite.client.Post(
		suite.baseURL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		t.Fatalf("Failed to login user: %v", err)
	}
	defer resp.Body.Close()

	// Parse login response
	var loginResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}

	// Extract token
	data := loginResp["data"].(map[string]interface{})
	suite.authToken = data["access_token"].(string)
}
