package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/catherinevee/driftmgr/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	handler := HealthHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["status"])
	assert.NotNil(t, response["timestamp"])
}

func TestDiscoverHandler(t *testing.T) {
	handler := DiscoverHandler()

	tests := []struct {
		name       string
		method     string
		body       interface{}
		wantStatus int
	}{
		{
			name:       "GET request",
			method:     "GET",
			body:       nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST with valid body",
			method:     "POST",
			body:       map[string]string{"provider": "aws", "region": "us-east-1"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST with invalid body",
			method:     "POST",
			body:       "invalid json",
			wantStatus: http.StatusBadRequest,
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
				req = httptest.NewRequest(tt.method, "/api/v1/discover", bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/api/v1/discover", nil)
			}

			w := httptest.NewRecorder()
			handler(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestDriftHandler(t *testing.T) {
	handler := DriftHandler()

	req := httptest.NewRequest("GET", "/api/v1/drift", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
}

func TestStateHandler(t *testing.T) {
	handler := StateHandler()

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "GET state",
			method:     "GET",
			path:       "/api/v1/state",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST state",
			method:     "POST",
			path:       "/api/v1/state",
			wantStatus: http.StatusOK,
		},
		{
			name:       "PUT state",
			method:     "PUT",
			path:       "/api/v1/state",
			wantStatus: http.StatusOK,
		},
		{
			name:       "DELETE state",
			method:     "DELETE",
			path:       "/api/v1/state",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestRemediationHandler(t *testing.T) {
	handler := RemediationHandler()

	tests := []struct {
		name       string
		method     string
		body       interface{}
		wantStatus int
	}{
		{
			name:       "GET remediations",
			method:     "GET",
			body:       nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST create remediation",
			method:     "POST",
			body: map[string]interface{}{
				"resource_id": "i-123",
				"action":      "update",
				"parameters":  map[string]string{"instance_type": "t2.micro"},
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, "/api/v1/remediation", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/api/v1/remediation", nil)
			}

			w := httptest.NewRecorder()
			handler(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestResourcesHandler(t *testing.T) {
	handler := ResourcesHandler()

	req := httptest.NewRequest("GET", "/api/v1/resources", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.Resource
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestProvidersHandler(t *testing.T) {
	handler := ProvidersHandler()

	req := httptest.NewRequest("GET", "/api/v1/providers", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestConfigHandler(t *testing.T) {
	handler := ConfigHandler()

	tests := []struct {
		name       string
		method     string
		body       interface{}
		wantStatus int
	}{
		{
			name:       "GET config",
			method:     "GET",
			body:       nil,
			wantStatus: http.StatusOK,
		},
		{
			name:       "PUT update config",
			method:     "PUT",
			body: map[string]interface{}{
				"auto_discovery": true,
				"max_workers":    10,
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				req = httptest.NewRequest(tt.method, "/api/v1/config", bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/api/v1/config", nil)
			}

			w := httptest.NewRecorder()
			handler(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestErrorResponse(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		message  string
		wantCode int
	}{
		{
			name:     "Bad Request",
			status:   http.StatusBadRequest,
			message:  "Invalid input",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Internal Server Error",
			status:   http.StatusInternalServerError,
			message:  "Something went wrong",
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "Not Found",
			status:   http.StatusNotFound,
			message:  "Resource not found",
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeErrorResponse(w, tt.status, tt.message)

			assert.Equal(t, tt.wantCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, tt.message, response["error"])
		})
	}
}

func TestJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		wantCode int
	}{
		{
			name:     "Simple object",
			data:     map[string]string{"key": "value"},
			wantCode: http.StatusOK,
		},
		{
			name:     "Array",
			data:     []string{"item1", "item2", "item3"},
			wantCode: http.StatusOK,
		},
		{
			name:     "Complex object",
			data: models.Resource{
				ID:       "resource-1",
				Type:     "aws_instance",
				Provider: "aws",
				Region:   "us-east-1",
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSONResponse(w, tt.wantCode, tt.data)

			assert.Equal(t, tt.wantCode, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			// Verify JSON is valid
			var result interface{}
			err := json.Unmarshal(w.Body.Bytes(), &result)
			assert.NoError(t, err)
		})
	}
}

func TestPaginationParams(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantPage   int
		wantLimit  int
		wantOffset int
	}{
		{
			name:       "Default values",
			query:      "",
			wantPage:   1,
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "Custom page and limit",
			query:      "page=3&limit=50",
			wantPage:   3,
			wantLimit:  50,
			wantOffset: 100,
		},
		{
			name:       "Invalid values use defaults",
			query:      "page=invalid&limit=invalid",
			wantPage:   1,
			wantLimit:  20,
			wantOffset: 0,
		},
		{
			name:       "Negative values use defaults",
			query:      "page=-1&limit=-10",
			wantPage:   1,
			wantLimit:  20,
			wantOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test?"+tt.query, nil)
			page, limit, offset := getPaginationParams(req)

			assert.Equal(t, tt.wantPage, page)
			assert.Equal(t, tt.wantLimit, limit)
			assert.Equal(t, tt.wantOffset, offset)
		})
	}
}

func TestFilterParams(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantFilters map[string]string
	}{
		{
			name:        "No filters",
			query:       "",
			wantFilters: map[string]string{},
		},
		{
			name:  "Single filter",
			query: "provider=aws",
			wantFilters: map[string]string{
				"provider": "aws",
			},
		},
		{
			name:  "Multiple filters",
			query: "provider=aws&region=us-east-1&type=ec2",
			wantFilters: map[string]string{
				"provider": "aws",
				"region":   "us-east-1",
				"type":     "ec2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test?"+tt.query, nil)
			filters := getFilterParams(req)

			assert.Equal(t, len(tt.wantFilters), len(filters))
			for k, v := range tt.wantFilters {
				assert.Equal(t, v, filters[k])
			}
		})
	}
}

// Helper functions that might be needed
func writeErrorResponse(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func getPaginationParams(r *http.Request) (page, limit, offset int) {
	page = 1
	limit = 20

	// Parse query parameters
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := parseInt(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset = (page - 1) * limit
	return
}

func getFilterParams(r *http.Request) map[string]string {
	filters := make(map[string]string)
	for k, v := range r.URL.Query() {
		if k != "page" && k != "limit" && len(v) > 0 {
			filters[k] = v[0]
		}
	}
	return filters
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}