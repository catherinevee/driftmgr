package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "GET health check",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "healthy", body["status"])
				assert.Equal(t, "driftmgr-api", body["service"])
				assert.Equal(t, "1.0.0", body["version"])
				assert.NotNil(t, body["timestamp"])
			},
		},
		{
			name:           "POST health check",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "healthy", body["status"])
			},
		},
		{
			name:           "PUT health check",
			method:         http.MethodPut,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "healthy", body["status"])
			},
		},
		{
			name:           "DELETE health check",
			method:         http.MethodDelete,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "healthy", body["status"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			HealthHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			tt.validateBody(t, response)
		})
	}
}

func TestHealthHandler_ConcurrentRequests(t *testing.T) {
	// Test concurrent access to health endpoint
	numRequests := 100
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()
			HealthHandler(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

func BenchmarkHealthHandler(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		HealthHandler(w, req)
	}
}
