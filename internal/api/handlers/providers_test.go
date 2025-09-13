package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvidersHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "GET all providers",
			method:         http.MethodGet,
			path:           "/providers",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				providers, ok := body["providers"].([]interface{})
				require.True(t, ok)
				assert.NotEmpty(t, providers)
			},
		},
		{
			name:           "GET specific provider",
			method:         http.MethodGet,
			path:           "/providers/aws",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				providers, ok := body["providers"].([]interface{})
				require.True(t, ok)
				assert.NotEmpty(t, providers)
			},
		},
		{
			name:   "POST configure provider",
			method: http.MethodPost,
			path:   "/providers/aws",
			body: map[string]interface{}{
				"region":      "us-east-1",
				"credentials": map[string]string{"profile": "default"},
			},
			expectedStatus: http.StatusAccepted,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "accepted", body["status"])
				assert.NotNil(t, body["provider"])
			},
		},
		{
			name:   "PUT update provider",
			method: http.MethodPut,
			path:   "/providers/aws",
			body: map[string]interface{}{
				"enabled": true,
				"regions": []string{"us-east-1", "us-west-2"},
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "updated", body["status"])
			},
		},
		{
			name:           "DELETE disable provider",
			method:         http.MethodDelete,
			path:           "/providers/aws",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "disabled", body["status"])
			},
		},
		{
			name:           "GET non-existent provider",
			method:         http.MethodGet,
			path:           "/providers/nonexistent",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				providers, ok := body["providers"].([]interface{})
				require.True(t, ok)
				assert.NotNil(t, providers)
			},
		},
		{
			name:           "POST with invalid JSON",
			method:         http.MethodPost,
			path:           "/providers/aws",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
			validateBody:   func(t *testing.T, body map[string]interface{}) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				var bodyBytes []byte
				if str, ok := tt.body.(string); ok {
					bodyBytes = []byte(str)
				} else {
					bodyBytes, _ = json.Marshal(tt.body)
				}
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			ProvidersHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus < 400 {
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.validateBody(t, response)
			}
		})
	}
}

func TestProvidersHandler_AllProviders(t *testing.T) {
	providers := []string{"aws", "azure", "gcp", "digitalocean"}

	for _, provider := range providers {
		t.Run("provider_"+provider, func(t *testing.T) {
			// Test GET
			req := httptest.NewRequest(http.MethodGet, "/providers/"+provider, nil)
			w := httptest.NewRecorder()
			ProvidersHandler(w, req)
			assert.Equal(t, http.StatusOK, w.Code)

			// Test POST
			body := map[string]interface{}{
				"enabled": true,
			}
			bodyBytes, _ := json.Marshal(body)
			req = httptest.NewRequest(http.MethodPost, "/providers/"+provider, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()
			ProvidersHandler(w, req)
			assert.Equal(t, http.StatusAccepted, w.Code)

			// Test PUT
			req = httptest.NewRequest(http.MethodPut, "/providers/"+provider, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w = httptest.NewRecorder()
			ProvidersHandler(w, req)
			assert.Equal(t, http.StatusOK, w.Code)

			// Test DELETE
			req = httptest.NewRequest(http.MethodDelete, "/providers/"+provider, nil)
			w = httptest.NewRecorder()
			ProvidersHandler(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestProvidersHandler_ConfigValidation(t *testing.T) {
	tests := []struct {
		name           string
		provider       string
		config         map[string]interface{}
		expectedStatus int
	}{
		{
			name:     "AWS valid config",
			provider: "aws",
			config: map[string]interface{}{
				"region":      "us-east-1",
				"credentials": map[string]string{"profile": "default"},
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:     "Azure valid config",
			provider: "azure",
			config: map[string]interface{}{
				"subscription_id": "12345-67890",
				"tenant_id":       "abcdef-12345",
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:     "GCP valid config",
			provider: "gcp",
			config: map[string]interface{}{
				"project_id":   "my-project",
				"credentials":  map[string]string{"type": "service_account"},
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:     "DigitalOcean valid config",
			provider: "digitalocean",
			config: map[string]interface{}{
				"token": "do_token_12345",
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:     "Empty config",
			provider: "aws",
			config:   map[string]interface{}{},
			expectedStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.config)
			req := httptest.NewRequest(http.MethodPost, "/providers/"+tt.provider, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			ProvidersHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func BenchmarkProvidersHandler_GET(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/providers", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		ProvidersHandler(w, req)
	}
}

func BenchmarkProvidersHandler_POST(b *testing.B) {
	body := map[string]interface{}{
		"region": "us-east-1",
		"enabled": true,
	}
	bodyBytes, _ := json.Marshal(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/providers/aws", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		ProvidersHandler(w, req)
	}
}