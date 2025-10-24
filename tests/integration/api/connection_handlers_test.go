package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/api"
	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionHandlers_Integration(t *testing.T) {
	// Skip if no credentials are available
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create event bus
	eventBus := events.NewEventBus(100)
	defer eventBus.Close()

	// Create provider factory
	factory := providers.NewProviderFactory()

	// Create connection service
	connectionService := providers.NewConnectionService(factory, eventBus, 30*time.Second)

	// Create connection handlers
	handlers := api.NewConnectionHandlers(connectionService)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register routes
	apiGroup := router.Group("/api/v1")
	{
		apiGroup.GET("/connection/test/:provider", handlers.TestProviderConnection)
		apiGroup.GET("/connection/test/:provider/service/:service", handlers.TestProviderService)
		apiGroup.GET("/connection/test/all", handlers.TestAllProviders)
		apiGroup.GET("/connection/test/:provider/regions", handlers.TestProviderAllRegions)
		apiGroup.GET("/connection/test/:provider/services", handlers.TestProviderAllServices)
		apiGroup.GET("/connection/results", handlers.GetConnectionResults)
		apiGroup.GET("/connection/results/:provider", handlers.GetConnectionResults)
		apiGroup.GET("/connection/summary", handlers.GetConnectionSummary)
		apiGroup.POST("/connection/health-check", handlers.RunHealthCheck)
		apiGroup.DELETE("/connection/results", handlers.ClearConnectionResults)
		apiGroup.DELETE("/connection/results/:provider", handlers.ClearConnectionResults)
	}

	t.Run("TestProviderConnection", func(t *testing.T) {
		// Skip if AWS credentials are not available
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			t.Skip("AWS credentials not available")
		}

		req, err := http.NewRequest("GET", "/api/v1/connection/test/aws?region=us-east-1", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "result")

		result := response["result"].(map[string]interface{})
		assert.Equal(t, "aws", result["provider"])
		assert.Equal(t, "us-east-1", result["region"])
	})

	t.Run("TestProviderService", func(t *testing.T) {
		// Skip if AWS credentials are not available
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			t.Skip("AWS credentials not available")
		}

		req, err := http.NewRequest("GET", "/api/v1/connection/test/aws/service/aws_s3_bucket?region=us-east-1", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "result")

		result := response["result"].(map[string]interface{})
		assert.Equal(t, "aws", result["provider"])
		assert.Equal(t, "us-east-1", result["region"])
		assert.Equal(t, "aws_s3_bucket", result["service"])
	})

	t.Run("TestAllProviders", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/connection/test/all?region=us-east-1", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "results")

		results := response["results"].(map[string]interface{})
		expectedProviders := []string{"aws", "azure", "gcp", "digitalocean"}
		for _, provider := range expectedProviders {
			assert.Contains(t, results, provider)
		}
	})

	t.Run("TestProviderAllRegions", func(t *testing.T) {
		// Skip if AWS credentials are not available
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			t.Skip("AWS credentials not available")
		}

		req, err := http.NewRequest("GET", "/api/v1/connection/test/aws/regions", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "results")

		results := response["results"].([]interface{})
		assert.Greater(t, len(results), 0)
	})

	t.Run("TestProviderAllServices", func(t *testing.T) {
		// Skip if AWS credentials are not available
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			t.Skip("AWS credentials not available")
		}

		req, err := http.NewRequest("GET", "/api/v1/connection/test/aws/services?region=us-east-1", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "results")

		results := response["results"].([]interface{})
		assert.Greater(t, len(results), 0)
	})

	t.Run("GetConnectionResults", func(t *testing.T) {
		// Test getting all results
		req, err := http.NewRequest("GET", "/api/v1/connection/results", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "results")

		// Test getting specific provider results
		req, err = http.NewRequest("GET", "/api/v1/connection/results/aws", nil)
		require.NoError(t, err)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "results")
	})

	t.Run("GetConnectionSummary", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/connection/summary", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "summary")
	})

	t.Run("RunHealthCheck", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/api/v1/connection/health-check?region=us-east-1", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "result")

		result := response["result"].(map[string]interface{})
		assert.Equal(t, "us-east-1", result["region"])
		assert.Contains(t, result, "duration")
		assert.Contains(t, result, "summary")
		assert.Contains(t, result, "results")
	})

	t.Run("ClearConnectionResults", func(t *testing.T) {
		// Test clearing all results
		req, err := http.NewRequest("DELETE", "/api/v1/connection/results", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Equal(t, "Connection results cleared", response["message"])

		// Test clearing specific provider results
		req, err = http.NewRequest("DELETE", "/api/v1/connection/results/aws", nil)
		require.NoError(t, err)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Equal(t, "Connection results cleared", response["message"])
	})

	t.Run("TestTimeoutParameter", func(t *testing.T) {
		// Skip if AWS credentials are not available
		if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
			t.Skip("AWS credentials not available")
		}

		req, err := http.NewRequest("GET", "/api/v1/connection/test/aws?region=us-east-1&timeout=10s", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		assert.Contains(t, response, "result")
	})

	t.Run("TestInvalidProvider", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/connection/test/invalid-provider?region=us-east-1", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.False(t, response["success"].(bool))
		assert.Contains(t, response, "error")
	})
}
