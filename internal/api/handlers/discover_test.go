package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "GET discovery status",
			method:         http.MethodGet,
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "ready", body["status"])
				providers, ok := body["providers"].([]interface{})
				require.True(t, ok)
				assert.Contains(t, providers, "aws")
				assert.Contains(t, providers, "azure")
				assert.Contains(t, providers, "gcp")
				assert.Contains(t, providers, "digitalocean")
			},
		},
		{
			name:   "POST start discovery",
			method: http.MethodPost,
			body: map[string]interface{}{
				"provider": "aws",
				"regions":  []string{"us-east-1", "us-west-2"},
			},
			expectedStatus: http.StatusAccepted,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "accepted", body["status"])
				assert.NotNil(t, body["id"])
				assert.Contains(t, body["id"], "discovery-")
				assert.NotNil(t, body["request"])
			},
		},
		{
			name:           "POST with empty body",
			method:         http.MethodPost,
			body:           map[string]interface{}{},
			expectedStatus: http.StatusAccepted,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "accepted", body["status"])
				assert.NotNil(t, body["id"])
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
			name:           "PUT not allowed",
			method:         http.MethodPut,
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
			validateBody:   func(t *testing.T, body map[string]interface{}) {},
		},
		{
			name:           "DELETE not allowed",
			method:         http.MethodDelete,
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
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
				req = httptest.NewRequest(tt.method, "/discover", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/discover", nil)
			}

			w := httptest.NewRecorder()
			DiscoverHandler(w, req)

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

func TestDiscoverHandler_LargeRequest(t *testing.T) {
	// Test with a large request body
	regions := make([]string, 100)
	for i := range regions {
		regions[i] = "region-" + string(rune('0'+i))
	}

	body := map[string]interface{}{
		"provider": "aws",
		"regions":  regions,
		"options": map[string]interface{}{
			"includeAllResources": true,
			"maxConcurrency":      10,
			"timeout":             300,
		},
	}

	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/discover", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	DiscoverHandler(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "accepted", response["status"])
	assert.NotNil(t, response["request"])
}

func TestDiscoverHandler_MalformedJSON(t *testing.T) {
	malformedJSONs := []string{
		`{"provider": "aws"`,           // Missing closing brace
		`{"provider": aws}`,             // Unquoted value
		`{'provider': 'aws'}`,           // Single quotes
		`{"provider": "aws", "regions"`, // Incomplete
	}

	for i, malformed := range malformedJSONs {
		t.Run("malformed_json_"+strings.ReplaceAll(malformed, " ", "_"), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/discover", strings.NewReader(malformed))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			DiscoverHandler(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code, "Test case %d failed", i)
		})
	}
}

func BenchmarkDiscoverHandler_GET(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/discover", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		DiscoverHandler(w, req)
	}
}

func BenchmarkDiscoverHandler_POST(b *testing.B) {
	body := map[string]interface{}{
		"provider": "aws",
		"regions":  []string{"us-east-1"},
	}
	bodyBytes, _ := json.Marshal(body)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/discover", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		DiscoverHandler(w, req)
	}
}