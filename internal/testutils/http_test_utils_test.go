package testutils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockServer(t *testing.T) {
	// Define test cases
	tests := []struct {
		name           string
		setupConfigs   []MockServerConfig
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "GET request with JSON response",
			setupConfigs: []MockServerConfig{
				{
					Method: http.MethodGet,
					Path:   "/test",
					Status: http.StatusOK,
					ResponseBody: map[string]string{
						"message": "success",
					},
				},
			},
			method:         http.MethodGet,
			path:           "/test",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"success"}`,
		},
		{
			name: "POST request with custom status",
			setupConfigs: []MockServerConfig{
				{
					Method: http.MethodPost,
					Path:   "/create",
					Status: http.StatusCreated,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					ResponseBody: "created",
				},
			},
			method:         http.MethodPost,
			path:           "/create",
			expectedStatus: http.StatusCreated,
			expectedBody:   `created`,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			server, serverURL := SetupMockServer(t, tt.setupConfigs)
			defer server.Close()

			// Make request to the mock server
			url := fmt.Sprintf("%s%s", serverURL, tt.path)
			req, err := http.NewRequest(tt.method, url, nil)
			require.NoError(t, err)

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			t.Logf("Verifying response for %s %s", tt.method, tt.path)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode, "Unexpected status code")

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "Failed to read response body")

			t.Logf("Response body: %s", string(body))

			if tt.expectedBody != "" {
				t.Logf("Expected body: %s", tt.expectedBody)

				// For JSON responses, we need to compare the unmarshaled data
				// to handle potential formatting differences
				var expected, actual interface{}
				err = json.Unmarshal([]byte(tt.expectedBody), &expected)
				if err == nil {
					err = json.Unmarshal(body, &actual)
					if assert.NoError(t, err, "Failed to unmarshal actual response") {
						assert.Equal(t, expected, actual, "Response body does not match expected")
					}
				} else {
					// Not JSON, compare as string
					assert.Equal(t, tt.expectedBody, string(body), "Response body does not match expected string")
				}
			}
		})
	}
}

func TestCreateMockJSONResponse(t *testing.T) {
	// Test with a struct
	t.Run("with struct", func(t *testing.T) {
		type TestStruct struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}

		testData := TestStruct{ID: 1, Name: "test"}
		resp := CreateMockJSONResponse(t, http.StatusOK, testData)

		// Verify response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		// Read and unmarshal response body
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result TestStruct
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)
		assert.Equal(t, testData, result)
	})

	// Test with a map
	t.Run("with map", func(t *testing.T) {
		testData := map[string]interface{}{"key": "value"}
		resp := CreateMockJSONResponse(t, http.StatusCreated, testData)

		// Verify response
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Read and unmarshal response body
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		require.NoError(t, err)
		assert.Equal(t, testData["key"], result["key"])
	})
}
