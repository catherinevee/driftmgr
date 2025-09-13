//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultServerURL = "http://localhost:8080"
	testTimeout      = 30 * time.Second
)

func getServerURL() string {
	if url := os.Getenv("DRIFTMGR_SERVER_URL"); url != "" {
		return url
	}
	return defaultServerURL
}

func TestHealthEndpoints(t *testing.T) {
	serverURL := getServerURL()
	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("Liveness", func(t *testing.T) {
		resp, err := client.Get(fmt.Sprintf("%s/health/live", serverURL))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Readiness", func(t *testing.T) {
		resp, err := client.Get(fmt.Sprintf("%s/health/ready", serverURL))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestDiscoveryAPI(t *testing.T) {
	serverURL := getServerURL()
	client := &http.Client{Timeout: testTimeout}

	t.Run("StartDiscovery", func(t *testing.T) {
		payload := map[string]interface{}{
			"provider":       "aws",
			"regions":        []string{"us-east-1"},
			"resource_types": []string{"ec2_instance", "s3_bucket"},
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		resp, err := client.Post(
			fmt.Sprintf("%s/api/v1/discover", serverURL),
			"application/json",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "id")
		assert.Contains(t, result, "status")
	})
}

func TestDriftDetectionAPI(t *testing.T) {
	serverURL := getServerURL()
	client := &http.Client{Timeout: testTimeout}

	t.Run("DetectDrift", func(t *testing.T) {
		// Create test state file
		stateContent := `{
			"version": 4,
			"terraform_version": "1.0.0",
			"serial": 1,
			"resources": []
		}`

		payload := map[string]interface{}{
			"state":    stateContent,
			"mode":     "quick",
			"provider": "aws",
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		resp, err := client.Post(
			fmt.Sprintf("%s/api/v1/drift/detect", serverURL),
			"application/json",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "id")
		assert.Contains(t, result, "status")
	})
}

func TestStateManagementAPI(t *testing.T) {
	serverURL := getServerURL()
	client := &http.Client{Timeout: testTimeout}

	t.Run("ListStates", func(t *testing.T) {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/state", serverURL))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
	})

	t.Run("AnalyzeState", func(t *testing.T) {
		stateContent := `{
			"version": 4,
			"terraform_version": "1.0.0",
			"serial": 1,
			"resources": [
				{
					"mode": "managed",
					"type": "aws_instance",
					"name": "test",
					"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
					"instances": []
				}
			]
		}`

		payload := map[string]interface{}{
			"state": stateContent,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		resp, err := client.Post(
			fmt.Sprintf("%s/api/v1/state/analyze", serverURL),
			"application/json",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Contains(t, result, "resources")
		assert.Contains(t, result, "providers")
	})
}

func TestRemediationAPI(t *testing.T) {
	serverURL := getServerURL()
	client := &http.Client{Timeout: testTimeout}

	t.Run("CreateRemediationPlan", func(t *testing.T) {
		payload := map[string]interface{}{
			"drift_id": "test-drift-123",
			"strategy": "code_as_truth",
			"dry_run":  true,
		}

		body, err := json.Marshal(payload)
		require.NoError(t, err)

		resp, err := client.Post(
			fmt.Sprintf("%s/api/v1/remediate", serverURL),
			"application/json",
			bytes.NewReader(body),
		)
		require.NoError(t, err)
		defer resp.Body.Close()

		// May return 404 if drift ID doesn't exist, which is OK for this test
		assert.Contains(t, []int{http.StatusAccepted, http.StatusNotFound}, resp.StatusCode)
	})
}

func TestResourcesAPI(t *testing.T) {
	serverURL := getServerURL()
	client := &http.Client{Timeout: testTimeout}

	t.Run("ListResources", func(t *testing.T) {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/resources", serverURL))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
	})
}

func TestMetricsEndpoint(t *testing.T) {
	serverURL := getServerURL()
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(fmt.Sprintf("%s/metrics", serverURL))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Metrics should return Prometheus format
	var body bytes.Buffer
	_, err = body.ReadFrom(resp.Body)
	require.NoError(t, err)

	// Check for common Prometheus metrics
	content := body.String()
	assert.Contains(t, content, "# HELP")
	assert.Contains(t, content, "# TYPE")
}

func TestConcurrentRequests(t *testing.T) {
	serverURL := getServerURL()
	client := &http.Client{Timeout: 10 * time.Second}

	// Test server can handle concurrent requests
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			defer func() { done <- true }()

			resp, err := client.Get(fmt.Sprintf("%s/health/live", serverURL))
			assert.NoError(t, err)
			if resp != nil {
				resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

func TestServerTimeout(t *testing.T) {
	serverURL := getServerURL()

	// Create client with very short timeout
	client := &http.Client{Timeout: 1 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/health/live", serverURL), nil)
	require.NoError(t, err)

	// This should timeout
	_, err = client.Do(req)
	assert.Error(t, err)
}
