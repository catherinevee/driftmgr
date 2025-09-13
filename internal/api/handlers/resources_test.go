package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourcesHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		queryParams    map[string]string
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "GET all resources",
			method:         http.MethodGet,
			queryParams:    nil,
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				resources, ok := body["resources"].([]interface{})
				require.True(t, ok)
				assert.NotNil(t, resources)
			},
		},
		{
			name:   "GET resources with provider filter",
			method: http.MethodGet,
			queryParams: map[string]string{
				"provider": "aws",
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				resources, ok := body["resources"].([]interface{})
				require.True(t, ok)
				assert.NotNil(t, resources)
			},
		},
		{
			name:   "GET resources with region filter",
			method: http.MethodGet,
			queryParams: map[string]string{
				"region": "us-east-1",
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				resources, ok := body["resources"].([]interface{})
				require.True(t, ok)
				assert.NotNil(t, resources)
			},
		},
		{
			name:   "GET resources with type filter",
			method: http.MethodGet,
			queryParams: map[string]string{
				"type": "ec2_instance",
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				resources, ok := body["resources"].([]interface{})
				require.True(t, ok)
				assert.NotNil(t, resources)
			},
		},
		{
			name:   "GET resources with multiple filters",
			method: http.MethodGet,
			queryParams: map[string]string{
				"provider": "aws",
				"region":   "us-west-2",
				"type":     "s3_bucket",
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				resources, ok := body["resources"].([]interface{})
				require.True(t, ok)
				assert.NotNil(t, resources)
			},
		},
		{
			name:           "POST not allowed",
			method:         http.MethodPost,
			queryParams:    nil,
			expectedStatus: http.StatusMethodNotAllowed,
			validateBody:   func(t *testing.T, body map[string]interface{}) {},
		},
		{
			name:           "PUT not allowed",
			method:         http.MethodPut,
			queryParams:    nil,
			expectedStatus: http.StatusMethodNotAllowed,
			validateBody:   func(t *testing.T, body map[string]interface{}) {},
		},
		{
			name:           "DELETE not allowed",
			method:         http.MethodDelete,
			queryParams:    nil,
			expectedStatus: http.StatusMethodNotAllowed,
			validateBody:   func(t *testing.T, body map[string]interface{}) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := "/resources"
			if tt.queryParams != nil {
				values := url.Values{}
				for k, v := range tt.queryParams {
					values.Add(k, v)
				}
				reqURL += "?" + values.Encode()
			}

			req := httptest.NewRequest(tt.method, reqURL, nil)
			w := httptest.NewRecorder()

			ResourcesHandler(w, req)

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

func TestResourcesHandler_Pagination(t *testing.T) {
	tests := []struct {
		name        string
		queryParams map[string]string
	}{
		{
			name: "pagination with limit",
			queryParams: map[string]string{
				"limit": "10",
			},
		},
		{
			name: "pagination with offset",
			queryParams: map[string]string{
				"offset": "20",
			},
		},
		{
			name: "pagination with limit and offset",
			queryParams: map[string]string{
				"limit":  "10",
				"offset": "20",
			},
		},
		{
			name: "pagination with page",
			queryParams: map[string]string{
				"page":     "2",
				"per_page": "25",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{}
			for k, v := range tt.queryParams {
				values.Add(k, v)
			}
			reqURL := "/resources?" + values.Encode()

			req := httptest.NewRequest(http.MethodGet, reqURL, nil)
			w := httptest.NewRecorder()

			ResourcesHandler(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.NotNil(t, response["resources"])
		})
	}
}

func TestResourcesHandler_Sorting(t *testing.T) {
	sortFields := []string{"name", "type", "provider", "region", "created_at", "updated_at"}

	for _, field := range sortFields {
		for _, order := range []string{"asc", "desc"} {
			t.Run("sort_by_"+field+"_"+order, func(t *testing.T) {
				reqURL := "/resources?sort=" + field + "&order=" + order
				req := httptest.NewRequest(http.MethodGet, reqURL, nil)
				w := httptest.NewRecorder()

				ResourcesHandler(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotNil(t, response["resources"])
			})
		}
	}
}

func BenchmarkResourcesHandler(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/resources", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		ResourcesHandler(w, req)
	}
}

func BenchmarkResourcesHandler_WithFilters(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/resources?provider=aws&region=us-east-1&type=ec2_instance", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		ResourcesHandler(w, req)
	}
}
