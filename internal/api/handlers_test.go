package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerMethods tests the server helper methods
func TestServerMethods(t *testing.T) {
	server := &Server{}

	t.Run("handleCORS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		server.handleCORS(w, req)

		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
	})

	t.Run("handleCORS_OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()

		server.handleCORS(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("handleRateLimit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		allowed := server.handleRateLimit(w, req)
		assert.True(t, allowed)
	})

	t.Run("handleAuth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		authenticated := server.handleAuth(w, req)
		assert.True(t, authenticated)
		_ = w // Use w to avoid unused variable
	})

	t.Run("logRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		// Should not panic
		assert.NotPanics(t, func() {
			server.logRequest(req)
		})
		_ = w // Use w to avoid unused variable
	})

	t.Run("writeJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"test": "value"}

		server.writeJSON(w, http.StatusOK, data)

		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, http.StatusOK, w.Code)

		var result map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "value", result["test"])
	})

	t.Run("writeError", func(t *testing.T) {
		w := httptest.NewRecorder()

		server.writeError(w, http.StatusBadRequest, "test error")

		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var result map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "test error", result["error"])
	})

	t.Run("parseID", func(t *testing.T) {
		tests := []struct {
			path     string
			expected string
			hasError bool
		}{
			{"/api/v1/resources/123", "123", false},
			{"/api/v1/resources/abc", "abc", false},
			{"/api/v1/resources/", "", true},
			{"/api", "", true},
		}

		for _, tt := range tests {
			req := httptest.NewRequest("GET", tt.path, nil)
			id, err := server.parseID(req)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, id)
			}
		}
	})

	t.Run("parseQueryParams", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?param1=value1&param2=value2&param3=value3a&param3=value3b", nil)
		params := server.parseQueryParams(req)

		assert.Equal(t, "value1", params["param1"])
		assert.Equal(t, "value2", params["param2"])
		assert.Equal(t, "value3a", params["param3"]) // First value
	})

	t.Run("parsePagination", func(t *testing.T) {
		tests := []struct {
			url           string
			expectedPage  int
			expectedLimit int
		}{
			{"/test", 1, 10},
			{"/test?page=5", 5, 10},
			{"/test?limit=25", 1, 25},
			{"/test?page=3&limit=50", 3, 50},
			{"/test?page=0", 1, 10},    // Invalid page
			{"/test?limit=200", 1, 10}, // Limit too high
			{"/test?page=abc", 1, 10},  // Invalid page
			{"/test?limit=xyz", 1, 10}, // Invalid limit
		}

		for _, tt := range tests {
			req := httptest.NewRequest("GET", tt.url, nil)
			page, limit := server.parsePagination(req)

			assert.Equal(t, tt.expectedPage, page)
			assert.Equal(t, tt.expectedLimit, limit)
		}
	})
}

// TestHealthHandler tests the health check handler
func TestHealthHandler(t *testing.T) {
	handler := HealthHandler()

	t.Run("health_check", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "healthy", response["status"])
		assert.NotNil(t, response["timestamp"])
		assert.Equal(t, "1.0.0", response["version"])
	})
}

// TestDiscoverHandler tests the discovery handler
func TestDiscoverHandler(t *testing.T) {
	handler := DiscoverHandler()

	t.Run("discover_POST_success", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"providers": []string{"aws", "azure"},
			"regions":   []string{"us-east-1", "us-west-2"},
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/discover", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "discovery_started", response["status"])
		assert.NotNil(t, response["timestamp"])
		assert.Equal(t, []interface{}{"aws", "azure"}, response["providers"])
		assert.Equal(t, []interface{}{"us-east-1", "us-west-2"}, response["regions"])
	})

	t.Run("discover_wrong_method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/discover", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		assert.Contains(t, w.Body.String(), "Method not allowed")
	})

	t.Run("discover_invalid_json", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/discover", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid JSON")
	})
}

// TestDriftHandler tests the drift detection handler
func TestDriftHandler(t *testing.T) {
	handler := DriftHandler()

	t.Run("drift_POST_success", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"state_file": "terraform.tfstate",
			"provider":   "aws",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/drift", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "drift_detection_started", response["status"])
		assert.NotNil(t, response["timestamp"])
	})

	t.Run("drift_wrong_method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/drift", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("drift_invalid_json", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/drift", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestRemediationHandler tests the remediation handler
func TestRemediationHandler(t *testing.T) {
	handler := RemediationHandler()

	t.Run("remediate_POST_success", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"plan_file": "drift-plan.json",
			"apply":     false,
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/remediate", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "remediation_started", response["status"])
		assert.NotNil(t, response["timestamp"])
	})

	t.Run("remediate_wrong_method", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/remediate", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

// TestStateHandler tests the state management handler
func TestStateHandler(t *testing.T) {
	handler := StateHandler()

	t.Run("state_GET_list", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/state", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response["status"])
		assert.NotNil(t, response["states"])
	})

	t.Run("state_POST_push", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"state_file": "terraform.tfstate",
			"backend":    "s3",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/state", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response["status"])
		assert.Equal(t, "state_pushed", response["message"])
	})
}

// TestResourcesHandler tests the resources handler
func TestResourcesHandler(t *testing.T) {
	handler := ResourcesHandler()

	t.Run("resources_GET_list", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/resources", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response["status"])
		assert.NotNil(t, response["resources"])
	})
}

// TestProvidersHandler tests the providers handler
func TestProvidersHandler(t *testing.T) {
	handler := ProvidersHandler()

	t.Run("providers_GET_list", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/providers", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response["status"])
		assert.NotNil(t, response["providers"])
	})
}

// TestConfigHandler tests the configuration handler
func TestConfigHandler(t *testing.T) {
	handler := ConfigHandler()

	t.Run("config_GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response["status"])
		assert.NotNil(t, response["config"])
	})

	t.Run("config_POST", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"setting1": "value1",
			"setting2": "value2",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/config", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response["status"])
		assert.Equal(t, "config_updated", response["message"])
	})
}

// TestErrorHandling tests error handling scenarios
func TestErrorHandling(t *testing.T) {
	t.Run("malformed_json", func(t *testing.T) {
		handler := DiscoverHandler()
		req := httptest.NewRequest("POST", "/discover", strings.NewReader("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid JSON")
	})

	t.Run("empty_request_body", func(t *testing.T) {
		handler := DiscoverHandler()
		req := httptest.NewRequest("POST", "/discover", nil)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unsupported_content_type", func(t *testing.T) {
		handler := DiscoverHandler()
		req := httptest.NewRequest("POST", "/discover", strings.NewReader("test"))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestConcurrency tests concurrent handler execution
func TestConcurrency(t *testing.T) {
	handler := HealthHandler()

	t.Run("concurrent_requests", func(t *testing.T) {
		const numRequests = 10
		results := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				req := httptest.NewRequest("GET", "/health", nil)
				w := httptest.NewRecorder()
				handler(w, req)
				results <- w.Code
			}()
		}

		// Collect results
		for i := 0; i < numRequests; i++ {
			code := <-results
			assert.Equal(t, http.StatusOK, code)
		}
	})
}

// TestPerformance tests handler performance
func TestPerformance(t *testing.T) {
	handler := HealthHandler()

	t.Run("response_time", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		start := time.Now()
		handler(w, req)
		duration := time.Since(start)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Less(t, duration, 100*time.Millisecond, "Handler should respond quickly")
	})
}

// TestMiddlewareIntegration tests middleware integration
func TestMiddlewareIntegration(t *testing.T) {
	server := &Server{}

	t.Run("cors_and_auth_chain", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()

		// Test CORS handling
		server.handleCORS(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Test auth handling
		req = httptest.NewRequest("GET", "/test", nil)
		w = httptest.NewRecorder()
		authenticated := server.handleAuth(w, req)
		assert.True(t, authenticated)
	})
}

// TestDataValidation tests data validation
func TestDataValidation(t *testing.T) {
	t.Run("discover_request_validation", func(t *testing.T) {
		handler := DiscoverHandler()

		// Test with empty providers
		requestBody := map[string]interface{}{
			"providers": []string{},
			"regions":   []string{"us-east-1"},
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/discover", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code) // Should still work with empty providers
	})

	t.Run("drift_request_validation", func(t *testing.T) {
		handler := DriftHandler()

		// Test with missing required fields
		requestBody := map[string]interface{}{
			"state_file": "terraform.tfstate",
			// Missing provider
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/drift", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		assert.Equal(t, http.StatusOK, w.Code) // Should handle missing fields gracefully
	})
}

// TestResponseFormat tests response format consistency
func TestResponseFormat(t *testing.T) {
	handlers := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
		body    string
	}{
		{"HealthHandler", HealthHandler(), "GET", "/health", ""},
		{"DiscoverHandler", DiscoverHandler(), "POST", "/discover", `{"providers":["aws"],"regions":["us-east-1"]}`},
		{"DriftHandler", DriftHandler(), "POST", "/drift", `{"state_file":"test.tfstate","provider":"aws"}`},
		{"RemediationHandler", RemediationHandler(), "POST", "/remediation", `{"plan_file":"test.json"}`},
		{"StateHandler", StateHandler(), "GET", "/state", ""},
		{"ResourcesHandler", ResourcesHandler(), "GET", "/resources", ""},
		{"ProvidersHandler", ProvidersHandler(), "GET", "/providers", ""},
		{"ConfigHandler", ConfigHandler(), "GET", "/config", ""},
	}

	for _, h := range handlers {
		t.Run(h.name, func(t *testing.T) {
			var req *http.Request
			if h.body != "" {
				req = httptest.NewRequest(h.method, h.path, strings.NewReader(h.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(h.method, h.path, nil)
			}
			w := httptest.NewRecorder()

			h.handler(w, req)

			// All handlers should return JSON
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Response should be valid JSON
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err, "Response should be valid JSON for %s", h.name)

			// Response should have status field
			assert.Contains(t, response, "status", "Response should have status field for %s", h.name)
		})
	}
}
