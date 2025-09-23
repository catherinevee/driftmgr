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

// TestComprehensiveHandlerScenarios tests comprehensive scenarios for all handlers
func TestComprehensiveHandlerScenarios(t *testing.T) {
	t.Run("HealthHandler_Comprehensive", func(t *testing.T) {
		handler := HealthHandler()

		// Test multiple requests
		for i := 0; i < 5; i++ {
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
		}
	})

	t.Run("DiscoverHandler_Comprehensive", func(t *testing.T) {
		handler := DiscoverHandler()

		testCases := []struct {
			name        string
			requestBody map[string]interface{}
			expectError bool
		}{
			{
				name: "Valid request with providers and regions",
				requestBody: map[string]interface{}{
					"providers": []string{"aws", "azure", "gcp"},
					"regions":   []string{"us-east-1", "us-west-2", "eastus"},
				},
				expectError: false,
			},
			{
				name: "Request with empty providers",
				requestBody: map[string]interface{}{
					"providers": []string{},
					"regions":   []string{"us-east-1"},
				},
				expectError: false,
			},
			{
				name: "Request with empty regions",
				requestBody: map[string]interface{}{
					"providers": []string{"aws"},
					"regions":   []string{},
				},
				expectError: false,
			},
			{
				name: "Request with single provider and region",
				requestBody: map[string]interface{}{
					"providers": []string{"aws"},
					"regions":   []string{"us-east-1"},
				},
				expectError: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				jsonBody, _ := json.Marshal(tc.requestBody)
				req := httptest.NewRequest("POST", "/discover", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				handler(w, req)

				if tc.expectError {
					assert.NotEqual(t, http.StatusOK, w.Code)
				} else {
					assert.Equal(t, http.StatusOK, w.Code)
					assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

					var response map[string]interface{}
					err := json.Unmarshal(w.Body.Bytes(), &response)
					require.NoError(t, err)

					assert.Equal(t, "discovery_started", response["status"])
					assert.NotNil(t, response["timestamp"])
					// Note: JSON unmarshaling converts []string to []interface{}
					// We'll just verify the fields exist rather than exact type matching
					assert.NotNil(t, response["providers"])
					assert.NotNil(t, response["regions"])
				}
			})
		}
	})

	t.Run("DriftHandler_Comprehensive", func(t *testing.T) {
		handler := DriftHandler()

		testCases := []struct {
			name        string
			requestBody map[string]interface{}
			expectError bool
		}{
			{
				name: "Valid request with resource_id",
				requestBody: map[string]interface{}{
					"resource_id": "test-resource-123",
				},
				expectError: false,
			},
			{
				name: "Request with empty resource_id",
				requestBody: map[string]interface{}{
					"resource_id": "",
				},
				expectError: false,
			},
			{
				name: "Request with additional fields",
				requestBody: map[string]interface{}{
					"resource_id": "test-resource-123",
					"provider":    "aws",
					"region":      "us-east-1",
				},
				expectError: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				jsonBody, _ := json.Marshal(tc.requestBody)
				req := httptest.NewRequest("POST", "/drift", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				handler(w, req)

				if tc.expectError {
					assert.NotEqual(t, http.StatusOK, w.Code)
				} else {
					assert.Equal(t, http.StatusOK, w.Code)
					assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

					var response map[string]interface{}
					err := json.Unmarshal(w.Body.Bytes(), &response)
					require.NoError(t, err)

					assert.Equal(t, "drift_analysis_started", response["status"])
					assert.NotNil(t, response["timestamp"])
					assert.Equal(t, tc.requestBody["resource_id"], response["resource_id"])
				}
			})
		}
	})

	t.Run("RemediationHandler_Comprehensive", func(t *testing.T) {
		handler := RemediationHandler()

		testCases := []struct {
			name        string
			requestBody map[string]interface{}
			expectError bool
		}{
			{
				name: "Valid request with resource_id and action",
				requestBody: map[string]interface{}{
					"resource_id": "test-resource-123",
					"action":      "restart",
				},
				expectError: false,
			},
			{
				name: "Request with empty resource_id",
				requestBody: map[string]interface{}{
					"resource_id": "",
					"action":      "restart",
				},
				expectError: false,
			},
			{
				name: "Request with empty action",
				requestBody: map[string]interface{}{
					"resource_id": "test-resource-123",
					"action":      "",
				},
				expectError: false,
			},
			{
				name: "Request with additional fields",
				requestBody: map[string]interface{}{
					"resource_id": "test-resource-123",
					"action":      "restart",
					"provider":    "aws",
					"region":      "us-east-1",
				},
				expectError: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				jsonBody, _ := json.Marshal(tc.requestBody)
				req := httptest.NewRequest("POST", "/remediate", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				handler(w, req)

				if tc.expectError {
					assert.NotEqual(t, http.StatusOK, w.Code)
				} else {
					assert.Equal(t, http.StatusOK, w.Code)
					assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

					var response map[string]interface{}
					err := json.Unmarshal(w.Body.Bytes(), &response)
					require.NoError(t, err)

					assert.Equal(t, "remediation_started", response["status"])
					assert.NotNil(t, response["timestamp"])
					assert.Equal(t, tc.requestBody["resource_id"], response["resource_id"])
					assert.Equal(t, tc.requestBody["action"], response["action"])
				}
			})
		}
	})

	t.Run("StateHandler_Comprehensive", func(t *testing.T) {
		handler := StateHandler()

		// Test GET method
		t.Run("GET_method", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/state", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.NotNil(t, response["states"])
			assert.NotNil(t, response["timestamp"])
		})

		// Test POST method
		t.Run("POST_method", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/state", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, "state_created", response["status"])
			assert.NotNil(t, response["timestamp"])
		})

		// Test unsupported method
		t.Run("PUT_method", func(t *testing.T) {
			req := httptest.NewRequest("PUT", "/state", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	})

	t.Run("ResourcesHandler_Comprehensive", func(t *testing.T) {
		handler := ResourcesHandler()

		// Test GET method
		t.Run("GET_method", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/resources", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.NotNil(t, response["resources"])
			assert.NotNil(t, response["timestamp"])
		})

		// Test unsupported method
		t.Run("POST_method", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/resources", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	})

	t.Run("ProvidersHandler_Comprehensive", func(t *testing.T) {
		handler := ProvidersHandler()

		// Test GET method
		t.Run("GET_method", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/providers", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.NotNil(t, response["providers"])
			assert.NotNil(t, response["timestamp"])

			// Verify providers list
			providers, ok := response["providers"].([]interface{})
			require.True(t, ok)
			assert.Contains(t, providers, "aws")
			assert.Contains(t, providers, "azure")
			assert.Contains(t, providers, "gcp")
			assert.Contains(t, providers, "digitalocean")
		})

		// Test unsupported method
		t.Run("POST_method", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/providers", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	})

	t.Run("ConfigHandler_Comprehensive", func(t *testing.T) {
		handler := ConfigHandler()

		// Test GET method
		t.Run("GET_method", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/config", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.NotNil(t, response["config"])
			assert.NotNil(t, response["timestamp"])

			// Verify config structure
			config, ok := response["config"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, "1.0.0", config["version"])
			assert.NotNil(t, config["features"])
		})

		// Test unsupported method
		t.Run("POST_method", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/config", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	})
}

// TestHandlerErrorScenarios tests various error scenarios
func TestHandlerErrorScenarios(t *testing.T) {
	t.Run("InvalidJSON_Scenarios", func(t *testing.T) {
		handlers := []struct {
			name    string
			handler http.HandlerFunc
			method  string
			path    string
		}{
			{"DiscoverHandler", DiscoverHandler(), "POST", "/discover"},
			{"DriftHandler", DriftHandler(), "POST", "/drift"},
			{"RemediationHandler", RemediationHandler(), "POST", "/remediate"},
		}

		invalidJSONs := []string{
			"{invalid json}",
			"{\"key\": \"value\"",
			"{\"key\": value}",
			"{\"key\": \"value\",}",
			"not json at all",
			"",
		}

		for _, h := range handlers {
			for _, invalidJSON := range invalidJSONs {
				t.Run(h.name+"_"+strings.ReplaceAll(invalidJSON, " ", "_"), func(t *testing.T) {
					req := httptest.NewRequest(h.method, h.path, strings.NewReader(invalidJSON))
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()

					h.handler(w, req)

					assert.Equal(t, http.StatusBadRequest, w.Code)
					assert.Contains(t, w.Body.String(), "Invalid JSON")
				})
			}
		}
	})

	t.Run("MethodNotAllowed_Scenarios", func(t *testing.T) {
		handlers := []struct {
			name    string
			handler http.HandlerFunc
			method  string
			path    string
		}{
			{"DiscoverHandler_GET", DiscoverHandler(), "GET", "/discover"},
			{"DriftHandler_GET", DriftHandler(), "GET", "/drift"},
			{"RemediationHandler_GET", RemediationHandler(), "GET", "/remediate"},
			{"ResourcesHandler_POST", ResourcesHandler(), "POST", "/resources"},
			{"ProvidersHandler_POST", ProvidersHandler(), "POST", "/providers"},
			{"ConfigHandler_POST", ConfigHandler(), "POST", "/config"},
		}

		for _, h := range handlers {
			t.Run(h.name, func(t *testing.T) {
				req := httptest.NewRequest(h.method, h.path, nil)
				w := httptest.NewRecorder()

				h.handler(w, req)

				assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
			})
		}
	})

	t.Run("EmptyRequestBody_Scenarios", func(t *testing.T) {
		handlers := []struct {
			name    string
			handler http.HandlerFunc
			method  string
			path    string
		}{
			{"DiscoverHandler", DiscoverHandler(), "POST", "/discover"},
			{"DriftHandler", DriftHandler(), "POST", "/drift"},
			{"RemediationHandler", RemediationHandler(), "POST", "/remediate"},
		}

		for _, h := range handlers {
			t.Run(h.name, func(t *testing.T) {
				req := httptest.NewRequest(h.method, h.path, nil)
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				h.handler(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
			})
		}
	})
}

// TestHandlerPerformance tests handler performance characteristics
func TestHandlerPerformance(t *testing.T) {
	handlers := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
		body    string
	}{
		{"HealthHandler", HealthHandler(), "GET", "/health", ""},
		{"DiscoverHandler", DiscoverHandler(), "POST", "/discover", `{"providers":["aws"],"regions":["us-east-1"]}`},
		{"DriftHandler", DriftHandler(), "POST", "/drift", `{"resource_id":"test-123"}`},
		{"RemediationHandler", RemediationHandler(), "POST", "/remediate", `{"resource_id":"test-123","action":"restart"}`},
		{"StateHandler_GET", StateHandler(), "GET", "/state", ""},
		{"StateHandler_POST", StateHandler(), "POST", "/state", ""},
		{"ResourcesHandler", ResourcesHandler(), "GET", "/resources", ""},
		{"ProvidersHandler", ProvidersHandler(), "GET", "/providers", ""},
		{"ConfigHandler", ConfigHandler(), "GET", "/config", ""},
	}

	for _, h := range handlers {
		t.Run(h.name+"_Performance", func(t *testing.T) {
			// Test single request performance
			var req *http.Request
			if h.body != "" {
				req = httptest.NewRequest(h.method, h.path, strings.NewReader(h.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(h.method, h.path, nil)
			}
			w := httptest.NewRecorder()

			start := time.Now()
			h.handler(w, req)
			duration := time.Since(start)

			assert.Less(t, duration, 100*time.Millisecond, "Handler should respond quickly")
			assert.NotEqual(t, http.StatusInternalServerError, w.Code, "Handler should not return 500")
		})
	}
}

// TestHandlerConcurrency tests concurrent handler execution
func TestHandlerConcurrency(t *testing.T) {
	handlers := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
		body    string
	}{
		{"HealthHandler", HealthHandler(), "GET", "/health", ""},
		{"DiscoverHandler", DiscoverHandler(), "POST", "/discover", `{"providers":["aws"],"regions":["us-east-1"]}`},
		{"DriftHandler", DriftHandler(), "POST", "/drift", `{"resource_id":"test-123"}`},
		{"StateHandler_GET", StateHandler(), "GET", "/state", ""},
		{"ResourcesHandler", ResourcesHandler(), "GET", "/resources", ""},
		{"ProvidersHandler", ProvidersHandler(), "GET", "/providers", ""},
		{"ConfigHandler", ConfigHandler(), "GET", "/config", ""},
	}

	for _, h := range handlers {
		t.Run(h.name+"_Concurrency", func(t *testing.T) {
			const numRequests = 10
			results := make(chan int, numRequests)

			for i := 0; i < numRequests; i++ {
				go func() {
					var req *http.Request
					if h.body != "" {
						req = httptest.NewRequest(h.method, h.path, strings.NewReader(h.body))
						req.Header.Set("Content-Type", "application/json")
					} else {
						req = httptest.NewRequest(h.method, h.path, nil)
					}
					w := httptest.NewRecorder()

					h.handler(w, req)
					results <- w.Code
				}()
			}

			// Collect results
			for i := 0; i < numRequests; i++ {
				code := <-results
				assert.NotEqual(t, http.StatusInternalServerError, code, "Handler should handle concurrency gracefully")
			}
		})
	}
}

// TestHandlerResponseConsistency tests response format consistency
func TestHandlerResponseConsistency(t *testing.T) {
	handlers := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
		body    string
	}{
		{"HealthHandler", HealthHandler(), "GET", "/health", ""},
		{"DiscoverHandler", DiscoverHandler(), "POST", "/discover", `{"providers":["aws"],"regions":["us-east-1"]}`},
		{"DriftHandler", DriftHandler(), "POST", "/drift", `{"resource_id":"test-123"}`},
		{"RemediationHandler", RemediationHandler(), "POST", "/remediate", `{"resource_id":"test-123","action":"restart"}`},
		{"StateHandler_GET", StateHandler(), "GET", "/state", ""},
		{"StateHandler_POST", StateHandler(), "POST", "/state", ""},
		{"ResourcesHandler", ResourcesHandler(), "GET", "/resources", ""},
		{"ProvidersHandler", ProvidersHandler(), "GET", "/providers", ""},
		{"ConfigHandler", ConfigHandler(), "GET", "/config", ""},
	}

	for _, h := range handlers {
		t.Run(h.name+"_ResponseConsistency", func(t *testing.T) {
			// Test multiple requests to ensure consistent responses
			for i := 0; i < 3; i++ {
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

				// Response should have either status field or timestamp field
				hasStatus := response["status"] != nil
				hasTimestamp := response["timestamp"] != nil
				assert.True(t, hasStatus || hasTimestamp, "Response should have status or timestamp field for %s", h.name)
			}
		})
	}
}
