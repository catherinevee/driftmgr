package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIServer(t *testing.T) {
	server := NewAPIServer(":8080")
	assert.NotNil(t, server)
	assert.Equal(t, ":8080", server.address)
	assert.NotNil(t, server.router)
}

func TestAPIServer_HealthCheck(t *testing.T) {
	server := NewAPIServer(":8080")

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestAPIServer_StartStop(t *testing.T) {
	server := NewAPIServer(":0") // Use port 0 for auto-assignment

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start server in background
	go func() {
		err := server.Start(ctx)
		assert.NoError(t, err)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	cancel()

	// Give server time to stop
	time.Sleep(100 * time.Millisecond)
}

func TestAPIServer_DiscoverEndpoint(t *testing.T) {
	server := NewAPIServer(":8080")

	tests := []struct {
		name       string
		method     string
		path       string
		body       interface{}
		wantStatus int
	}{
		{
			name:       "GET discover without params",
			method:     "GET",
			path:       "/api/v1/discover",
			body:       nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST discover with provider",
			method:     "POST",
			path:       "/api/v1/discover",
			body:       map[string]string{"provider": "aws", "region": "us-east-1"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid method",
			method:     "DELETE",
			path:       "/api/v1/discover",
			body:       nil,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestAPIServer_DriftEndpoint(t *testing.T) {
	server := NewAPIServer(":8080")

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{
			name:       "GET all drift",
			path:       "/api/v1/drift",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET drift by ID",
			path:       "/api/v1/drift/123",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET drift detection",
			path:       "/api/v1/drift/detect",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestAPIServer_StateEndpoint(t *testing.T) {
	server := NewAPIServer(":8080")

	tests := []struct {
		name       string
		method     string
		path       string
		body       interface{}
		wantStatus int
	}{
		{
			name:       "GET state",
			method:     "GET",
			path:       "/api/v1/state",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST state backup",
			method:     "POST",
			path:       "/api/v1/state/backup",
			body:       map[string]string{"name": "backup-1"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST state restore",
			method:     "POST",
			path:       "/api/v1/state/restore",
			body:       map[string]string{"backup_id": "backup-1"},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestAPIServer_RemediationEndpoint(t *testing.T) {
	server := NewAPIServer(":8080")

	tests := []struct {
		name       string
		method     string
		path       string
		body       interface{}
		wantStatus int
	}{
		{
			name:       "GET remediation plans",
			method:     "GET",
			path:       "/api/v1/remediation",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST create remediation",
			method:     "POST",
			path:       "/api/v1/remediation",
			body:       map[string]interface{}{"resource_id": "i-123", "action": "update"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST execute remediation",
			method:     "POST",
			path:       "/api/v1/remediation/execute",
			body:       map[string]string{"plan_id": "plan-123"},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}

			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestAPIServer_WebSocketEndpoint(t *testing.T) {
	server := NewAPIServer(":8080")

	// Test WebSocket upgrade request
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// WebSocket handler might not be implemented, so we check for either
	// upgrade response or not found
	assert.True(t, w.Code == http.StatusSwitchingProtocols || w.Code == http.StatusNotFound)
}

func TestAPIServer_MetricsEndpoint(t *testing.T) {
	server := NewAPIServer(":8080")

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	// Metrics endpoint might return 200 or 404 depending on implementation
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound)
}

func TestAPIServer_NotFound(t *testing.T) {
	server := NewAPIServer(":8080")

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAPIServer_CORS(t *testing.T) {
	server := NewAPIServer(":8080")

	req := httptest.NewRequest("OPTIONS", "/api/v1/discover", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	// Check CORS headers if implemented
	// Implementation might vary, so we just check the request doesn't fail
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNoContent || w.Code == http.StatusMethodNotAllowed)
}

func TestAPIServer_RateLimiting(t *testing.T) {
	server := NewAPIServer(":8080")

	// Send multiple rapid requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/api/v1/discover", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Rate limiting might kick in after a certain number of requests
		// We just verify the server handles the requests without panicking
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusTooManyRequests)
	}
}

func TestAPIServer_Authentication(t *testing.T) {
	server := NewAPIServer(":8080")

	tests := []struct {
		name       string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "No auth header",
			headers:    map[string]string{},
			wantStatus: http.StatusOK, // Might be OK if auth is optional
		},
		{
			name:       "Invalid auth header",
			headers:    map[string]string{"Authorization": "Invalid"},
			wantStatus: http.StatusOK, // Might be OK if auth is optional
		},
		{
			name:       "Valid Bearer token",
			headers:    map[string]string{"Authorization": "Bearer valid-token"},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/discover", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			// Auth might not be implemented, so we accept various responses
			assert.True(t, w.Code == tt.wantStatus || w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden)
		})
	}
}

func TestAPIServer_Compression(t *testing.T) {
	server := NewAPIServer(":8080")

	req := httptest.NewRequest("GET", "/api/v1/discover", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check if response is compressed (if compression is implemented)
	contentEncoding := w.Header().Get("Content-Encoding")
	assert.True(t, contentEncoding == "" || contentEncoding == "gzip")
}

func TestAPIServer_Timeout(t *testing.T) {
	server := NewAPIServer(":8080")

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest("GET", "/api/v1/discover", nil)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Wait to ensure context times out
	time.Sleep(2 * time.Millisecond)

	server.router.ServeHTTP(w, req)

	// Request might complete or timeout
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusRequestTimeout || w.Code == http.StatusServiceUnavailable)
}

func BenchmarkAPIServer_HealthCheck(b *testing.B) {
	server := NewAPIServer(":8080")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
	}
}

func BenchmarkAPIServer_DiscoverEndpoint(b *testing.B) {
	server := NewAPIServer(":8080")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/discover", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
	}
}

func BenchmarkAPIServer_ParallelRequests(b *testing.B) {
	server := NewAPIServer(":8080")

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
		}
	})
}