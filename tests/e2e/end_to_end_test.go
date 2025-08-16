package e2e

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/analysis"
	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/remediation"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteWorkflow tests the complete drift detection and remediation workflow
func TestCompleteWorkflow(t *testing.T) {
	// Setup components
	discoveryEngine := discovery.NewEngine()
	analysisEngine := analysis.NewEngine()
	remediationEngine := remediation.NewEngine()
	auth := security.NewAuth("test-secret-key")

	// Create test user
	user := &security.User{
		Username: "testuser",
		Role:     "admin",
		Permissions: []string{
			"view_dashboard",
			"execute_discovery",
			"execute_analysis",
			"execute_remediation",
		},
	}

	// Generate token
	token, err := auth.GenerateToken(user)
	require.NoError(t, err)

	// Validate token
	claims, err := auth.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, user.Username, claims.Username)

	ctx := context.Background()

	// Step 1: Discover resources
	t.Log("Step 1: Discovering resources...")
	resources, err := discoveryEngine.DiscoverResources(ctx, "aws", "us-east-1", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, resources)

	// Step 2: Analyze for drift
	t.Log("Step 2: Analyzing drift...")
	driftResults, err := analysisEngine.AnalyzeDrift(ctx, resources, resources)
	require.NoError(t, err)
	assert.NotNil(t, driftResults)

	// Step 3: Generate remediation plan
	t.Log("Step 3: Generating remediation plan...")
	plan, err := remediationEngine.GeneratePlan(ctx, driftResults)
	require.NoError(t, err)
	assert.NotNil(t, plan)

	// Step 4: Execute remediation (dry run)
	t.Log("Step 4: Executing remediation (dry run)...")
	results, err := remediationEngine.ExecutePlan(ctx, plan, true) // dry run
	require.NoError(t, err)
	assert.NotNil(t, results)

	t.Log("Complete workflow test passed")
}

// TestMultiCloudWorkflow tests workflow across multiple cloud providers
func TestMultiCloudWorkflow(t *testing.T) {
	discoveryEngine := discovery.NewEngine()
	analysisEngine := analysis.NewEngine()

	providers := []string{"aws", "azure", "gcp"}
	regions := map[string][]string{
		"aws":   {"us-east-1", "us-west-2"},
		"azure": {"eastus", "westus2"},
		"gcp":   {"us-central1", "us-east1"},
	}

	ctx := context.Background()

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			providerRegions := regions[provider]
			for _, region := range providerRegions {
				t.Run(region, func(t *testing.T) {
					// Discover resources
					resources, err := discoveryEngine.DiscoverResources(ctx, provider, region, nil)
					if err != nil {
						t.Logf("Discovery failed for %s/%s: %v", provider, region, err)
						return // Skip if provider not configured
					}

					// Analyze drift
					driftResults, err := analysisEngine.AnalyzeDrift(ctx, resources, resources)
					require.NoError(t, err)
					assert.NotNil(t, driftResults)

					t.Logf("Successfully processed %s/%s with %d resources", provider, region, len(resources))
				})
			}
		})
	}
}

// TestConcurrentOperations tests system behavior under concurrent load
func TestConcurrentOperations(t *testing.T) {
	discoveryEngine := discovery.NewEngine()
	analysisEngine := analysis.NewEngine()

	ctx := context.Background()
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}

	// Test concurrent discovery
	t.Run("ConcurrentDiscovery", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make(chan error, len(regions))

		for _, region := range regions {
			wg.Add(1)
			go func(r string) {
				defer wg.Done()
				_, err := discoveryEngine.DiscoverResources(ctx, "aws", r, nil)
				results <- err
			}(region)
		}

		wg.Wait()
		close(results)

		for err := range results {
			if err != nil {
				t.Logf("Discovery error: %v", err)
			}
		}
	})

	// Test concurrent analysis
	t.Run("ConcurrentAnalysis", func(t *testing.T) {
		testResources := generateTestResources(100)

		var wg sync.WaitGroup
		results := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := analysisEngine.AnalyzeDrift(ctx, testResources, testResources)
				results <- err
			}()
		}

		wg.Wait()
		close(results)

		for err := range results {
			require.NoError(t, err)
		}
	})
}

// TestErrorHandling tests system behavior under various error conditions
func TestErrorHandling(t *testing.T) {
	discoveryEngine := discovery.NewEngine()
	analysisEngine := analysis.NewEngine()
	auth := security.NewAuth("test-secret-key")

	ctx := context.Background()

	// Test invalid provider
	t.Run("InvalidProvider", func(t *testing.T) {
		_, err := discoveryEngine.DiscoverResources(ctx, "invalid-provider", "us-east-1", nil)
		assert.Error(t, err)
	})

	// Test invalid region
	t.Run("InvalidRegion", func(t *testing.T) {
		_, err := discoveryEngine.DiscoverResources(ctx, "aws", "invalid-region", nil)
		assert.Error(t, err)
	})

	// Test invalid token
	t.Run("InvalidToken", func(t *testing.T) {
		_, err := auth.ValidateToken("invalid-token")
		assert.Error(t, err)
	})

	// Test expired token
	t.Run("ExpiredToken", func(t *testing.T) {
		user := &security.User{
			Username: "testuser",
			Role:     "admin",
		}

		// Create token with short expiration
		auth.TokenExpiration = 1 * time.Millisecond
		token, err := auth.GenerateToken(user)
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		_, err = auth.ValidateToken(token)
		assert.Error(t, err)
	})

	// Test empty resource lists
	t.Run("EmptyResources", func(t *testing.T) {
		emptyResources := []models.Resource{}
		_, err := analysisEngine.AnalyzeDrift(ctx, emptyResources, emptyResources)
		require.NoError(t, err) // Should handle empty lists gracefully
	})
}

// TestPerformanceUnderLoad tests system performance under sustained load
func TestPerformanceUnderLoad(t *testing.T) {
	discoveryEngine := discovery.NewEngine()
	analysisEngine := analysis.NewEngine()

	ctx := context.Background()
	testResources := generateTestResources(1000)

	// Test sustained analysis operations
	t.Run("SustainedAnalysis", func(t *testing.T) {
		start := time.Now()
		iterations := 100

		for i := 0; i < iterations; i++ {
			_, err := analysisEngine.AnalyzeDrift(ctx, testResources, testResources)
			require.NoError(t, err)
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		t.Logf("Completed %d analysis operations in %v (avg: %v per operation)",
			iterations, duration, avgTime)

		// Performance assertion: each operation should complete within reasonable time
		assert.Less(t, avgTime, 100*time.Millisecond, "Analysis operations taking too long")
	})

	// Test memory usage
	t.Run("MemoryUsage", func(t *testing.T) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		initialAlloc := m.Alloc

		// Perform operations
		for i := 0; i < 50; i++ {
			_, err := analysisEngine.AnalyzeDrift(ctx, testResources, testResources)
			require.NoError(t, err)
		}

		runtime.ReadMemStats(&m)
		finalAlloc := m.Alloc
		memoryIncrease := finalAlloc - initialAlloc

		t.Logf("Memory usage: initial=%d bytes, final=%d bytes, increase=%d bytes",
			initialAlloc, finalAlloc, memoryIncrease)

		// Memory assertion: should not leak significantly
		assert.Less(t, memoryIncrease, uint64(10*1024*1024), "Memory usage increased too much")
	})
}

// TestSecurityFeatures tests security-related functionality
func TestSecurityFeatures(t *testing.T) {
	auth := security.NewAuth("test-secret-key")
	rbac := security.NewRBAC()

	// Test user authentication
	t.Run("UserAuthentication", func(t *testing.T) {
		user := &security.User{
			Username: "testuser",
			Role:     "admin",
			Permissions: []string{
				"view_dashboard",
				"execute_discovery",
			},
		}

		token, err := auth.GenerateToken(user)
		require.NoError(t, err)

		claims, err := auth.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, user.Username, claims.Username)
		assert.Equal(t, user.Role, claims.Role)
	})

	// Test role-based access control
	t.Run("RoleBasedAccess", func(t *testing.T) {
		adminUser := &security.User{
			Username: "admin",
			Role:     "root",
			Permissions: []string{
				"view_dashboard",
				"execute_discovery",
				"execute_remediation",
				"view_sensitive",
			},
		}

		readOnlyUser := &security.User{
			Username: "readonly",
			Role:     "readonly",
			Permissions: []string{
				"view_dashboard",
				"view_resources",
			},
		}

		// Test admin permissions
		assert.True(t, rbac.HasPermission(adminUser, "execute_remediation"))
		assert.True(t, rbac.HasPermission(adminUser, "view_sensitive"))

		// Test read-only permissions
		assert.True(t, rbac.HasPermission(readOnlyUser, "view_dashboard"))
		assert.False(t, rbac.HasPermission(readOnlyUser, "execute_remediation"))
		assert.False(t, rbac.HasPermission(readOnlyUser, "view_sensitive"))
	})

	// Test rate limiting
	t.Run("RateLimiting", func(t *testing.T) {
		limiter := security.NewRateLimiter(5, time.Minute) // 5 requests per minute

		// Test within limit
		for i := 0; i < 5; i++ {
			allowed := limiter.Allow("test-ip")
			assert.True(t, allowed, "Request %d should be allowed", i+1)
		}

		// Test exceeding limit
		allowed := limiter.Allow("test-ip")
		assert.False(t, allowed, "Request should be blocked after exceeding limit")
	})
}

// TestDataIntegrity tests data consistency and integrity
func TestDataIntegrity(t *testing.T) {
	analysisEngine := analysis.NewEngine()

	ctx := context.Background()

	// Test data consistency across operations
	t.Run("DataConsistency", func(t *testing.T) {
		originalResources := generateTestResources(100)

		// Perform multiple analyses with same data
		results := make([]*models.DriftResult, 5)
		for i := 0; i < 5; i++ {
			result, err := analysisEngine.AnalyzeDrift(ctx, originalResources, originalResources)
			require.NoError(t, err)
			results[i] = result
		}

		// All results should be identical
		for i := 1; i < len(results); i++ {
			assert.Equal(t, results[0].TotalDrifts, results[i].TotalDrifts)
			assert.Equal(t, results[0].DriftSummary, results[i].DriftSummary)
		}
	})

	// Test data validation
	t.Run("DataValidation", func(t *testing.T) {
		// Test with invalid resource data
		invalidResources := []models.Resource{
			{
				ID:       "", // Invalid: empty ID
				Name:     "Test Resource",
				Type:     "aws_instance",
				Provider: "aws",
				Region:   "us-east-1",
			},
		}

		_, err := analysisEngine.AnalyzeDrift(ctx, invalidResources, invalidResources)
		// Should either handle gracefully or return validation error
		if err != nil {
			t.Logf("Validation error as expected: %v", err)
		}
	})
}

// Helper function to generate test resources
func generateTestResources(count int) []models.Resource {
	resources := make([]models.Resource, count)

	for i := 0; i < count; i++ {
		resources[i] = models.Resource{
			ID:       fmt.Sprintf("test-resource-%d", i),
			Name:     fmt.Sprintf("Test Resource %d", i),
			Type:     "aws_instance",
			Provider: "aws",
			Region:   "us-east-1",
			Tags: map[string]string{
				"Environment": "test",
				"Project":     "driftmgr",
				"Index":       fmt.Sprintf("%d", i),
			},
			Properties: map[string]interface{}{
				"instance_type": "t3.micro",
				"ami":           "ami-12345678",
				"subnet_id":     "subnet-12345678",
			},
		}
	}

	return resources
}
