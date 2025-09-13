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

func TestConfigHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "GET configuration",
			method:         http.MethodGet,
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.NotNil(t, body["config"])
			},
		},
		{
			name:   "POST update configuration",
			method: http.MethodPost,
			body: map[string]interface{}{
				"settings": map[string]interface{}{
					"auto_discovery":   true,
					"parallel_workers": 10,
					"cache_ttl":        "5m",
				},
			},
			expectedStatus: http.StatusAccepted,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "accepted", body["status"])
				assert.NotNil(t, body["config"])
			},
		},
		{
			name:   "PUT replace configuration",
			method: http.MethodPut,
			body: map[string]interface{}{
				"provider": "aws",
				"regions":  []string{"us-east-1"},
				"settings": map[string]interface{}{
					"auto_discovery": false,
				},
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "updated", body["status"])
			},
		},
		{
			name:           "DELETE reset configuration",
			method:         http.MethodDelete,
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "reset", body["status"])
			},
		},
		{
			name:           "POST with invalid JSON",
			method:         http.MethodPost,
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
			validateBody:   func(t *testing.T, body map[string]interface{}) {},
		},
		{
			name:           "PUT with invalid JSON",
			method:         http.MethodPut,
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
				req = httptest.NewRequest(tt.method, "/config", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/config", nil)
			}

			w := httptest.NewRecorder()
			ConfigHandler(w, req)

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

func TestConfigHandler_CompleteConfig(t *testing.T) {
	config := map[string]interface{}{
		"provider": "aws",
		"regions":  []string{"us-east-1", "us-west-2", "eu-west-1"},
		"credentials": map[string]string{
			"profile": "default",
		},
		"settings": map[string]interface{}{
			"auto_discovery":   true,
			"parallel_workers": 8,
			"cache_ttl":        "10m",
			"drift_detection": map[string]interface{}{
				"enabled":  true,
				"interval": "1h",
				"severity": "high",
			},
			"remediation": map[string]interface{}{
				"enabled":           true,
				"dry_run":           false,
				"approval_required": true,
				"max_retries":       3,
			},
			"database": map[string]interface{}{
				"enabled": true,
				"path":    "/var/lib/driftmgr/driftmgr.db",
				"backup":  true,
			},
			"logging": map[string]interface{}{
				"level":  "info",
				"file":   "/var/log/driftmgr/driftmgr.log",
				"format": "json",
			},
			"notifications": map[string]interface{}{
				"enabled":  true,
				"channels": []string{"email", "slack"},
				"email": map[string]interface{}{
					"enabled":   true,
					"smtp_host": "smtp.example.com",
					"smtp_port": 587,
					"from":      "driftmgr@example.com",
					"to":        []string{"ops@example.com"},
				},
				"slack": map[string]interface{}{
					"enabled":     true,
					"webhook_url": "https://hooks.slack.com/services/XXX",
					"channel":     "#alerts",
					"username":    "DriftMgr",
				},
			},
		},
		"providers": map[string]interface{}{
			"aws": map[string]interface{}{
				"enabled": true,
				"regions": []string{"us-east-1", "us-west-2"},
				"resource_types": []string{
					"ec2_instance",
					"s3_bucket",
					"rds_instance",
				},
			},
			"azure": map[string]interface{}{
				"enabled":         false,
				"subscription_id": "12345-67890",
			},
		},
	}

	// Test POST with complete config
	bodyBytes, _ := json.Marshal(config)
	req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ConfigHandler(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "accepted", response["status"])
	assert.NotNil(t, response["config"])
}

func TestConfigHandler_PartialUpdate(t *testing.T) {
	updates := []map[string]interface{}{
		{
			"settings": map[string]interface{}{
				"parallel_workers": 16,
			},
		},
		{
			"regions": []string{"ap-southeast-1", "ap-northeast-1"},
		},
		{
			"provider": "azure",
		},
		{
			"settings": map[string]interface{}{
				"drift_detection": map[string]interface{}{
					"interval": "30m",
				},
			},
		},
	}

	for i, update := range updates {
		t.Run("partial_update_"+string(rune('0'+i)), func(t *testing.T) {
			bodyBytes, _ := json.Marshal(update)
			req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			ConfigHandler(w, req)

			assert.Equal(t, http.StatusAccepted, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, "accepted", response["status"])
		})
	}
}

func TestConfigHandler_Validation(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedStatus int
	}{
		{
			name: "valid parallel_workers",
			config: map[string]interface{}{
				"settings": map[string]interface{}{
					"parallel_workers": 8,
				},
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name: "negative parallel_workers",
			config: map[string]interface{}{
				"settings": map[string]interface{}{
					"parallel_workers": -1,
				},
			},
			expectedStatus: http.StatusAccepted, // Should still accept but may use default
		},
		{
			name: "excessive parallel_workers",
			config: map[string]interface{}{
				"settings": map[string]interface{}{
					"parallel_workers": 1000,
				},
			},
			expectedStatus: http.StatusAccepted, // Should still accept but may cap value
		},
		{
			name: "invalid cache_ttl format",
			config: map[string]interface{}{
				"settings": map[string]interface{}{
					"cache_ttl": "invalid",
				},
			},
			expectedStatus: http.StatusAccepted, // Should still accept but may use default
		},
		{
			name: "empty regions",
			config: map[string]interface{}{
				"regions": []string{},
			},
			expectedStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.config)
			req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			ConfigHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func BenchmarkConfigHandler_GET(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/config", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		ConfigHandler(w, req)
	}
}

func BenchmarkConfigHandler_POST(b *testing.B) {
	config := map[string]interface{}{
		"settings": map[string]interface{}{
			"parallel_workers": 8,
			"cache_ttl":        "10m",
		},
	}
	bodyBytes, _ := json.Marshal(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		ConfigHandler(w, req)
	}
}
