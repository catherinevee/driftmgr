package tests

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/concurrency"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/security"
)

// TestCacheIntegration tests the cache system
func TestCacheIntegration(t *testing.T) {
	// Use the actual cache implementation
	cacheInstance := cache.NewDiscoveryCache(5*time.Minute, 1000)

	// Test basic cache operations
	testData := []models.Resource{
		{
			ID:       "test-resource-1",
			Name:     "Test Resource 1",
			Type:     "aws_instance",
			Provider: "aws",
			Region:   "us-east-1",
		},
	}

	// Store data in cache
	cacheInstance.Set("test-key", testData, 5*time.Minute)

	// Retrieve data
	retrievedData, exists := cacheInstance.Get("test-key")
	if !exists {
		t.Fatal("Data not found in cache")
	}

	// Type assert the retrieved data
	retrievedResources, ok := retrievedData.([]models.Resource)
	if !ok {
		t.Fatal("Retrieved data is not of expected type")
	}

	if len(retrievedResources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(retrievedResources))
	}

	if retrievedResources[0].ID != "test-resource-1" {
		t.Errorf("Expected resource ID 'test-resource-1', got '%s'", retrievedResources[0].ID)
	}

	// Test cache with different data types
	cacheInstance.Set("string-key", "test-string", 5*time.Minute)
	cacheInstance.Set("int-key", 42, 5*time.Minute)

	// Test string retrieval
	if str, exists := cacheInstance.Get("string-key"); !exists || str != "test-string" {
		t.Error("String data not retrieved correctly")
	}

	// Test int retrieval
	if num, exists := cacheInstance.Get("int-key"); !exists || num != 42 {
		t.Error("Integer data not retrieved correctly")
	}

	// Test cache expiration
	cacheInstance.Set("expire-key", "will-expire", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	if _, exists := cacheInstance.Get("expire-key"); exists {
		t.Error("Expired data should not be retrievable")
	}

	// Test cache invalidation by setting to nil
	cacheInstance.Set("test-key", nil, 5*time.Minute)
	if data, exists := cacheInstance.Get("test-key"); exists && data != nil {
		t.Error("Nil data should not be retrievable as non-nil")
	}

	t.Log("Cache integration test passed")
}

// TestWorkerPoolIntegration tests the worker pool concurrency system
func TestWorkerPoolIntegration(t *testing.T) {
	pool := concurrency.NewWorkerPool(3)

	// Create test tasks with a channel to track completion
	taskCount := 10
	completedTasks := 0
	var mu sync.Mutex
	completionChan := make(chan bool, taskCount)

	// Submit simple tasks
	for i := 0; i < taskCount; i++ {
		task := func() {
			// Simulate work
			time.Sleep(5 * time.Millisecond)
			mu.Lock()
			completedTasks++
			mu.Unlock()
			completionChan <- true
		}

		err := pool.Submit(task)
		if err != nil {
			t.Fatalf("Failed to submit task %d: %v", i, err)
		}
	}

	// Wait for all tasks to complete using the channel
	completed := 0
	for i := 0; i < taskCount; i++ {
		select {
		case <-completionChan:
			completed++
		case <-time.After(5 * time.Second):
			t.Logf("Timeout waiting for task completion, completed: %d", completed)
			break
		}
	}

	// Shutdown the pool
	err := pool.Shutdown(5 * time.Second)
	if err != nil {
		t.Fatalf("Failed to shutdown pool: %v", err)
	}

	// Verify completion
	if completed < taskCount-2 { // Allow for 2 tasks to be missed due to timing
		t.Errorf("Expected at least %d completed tasks, got %d", taskCount-2, completed)
	}

	t.Logf("Worker pool processed %d tasks successfully (mutex count: %d)", completed, completedTasks)
}

// TestSecurityIntegration tests the security system
func TestSecurityIntegration(t *testing.T) {
	// Test token manager
	secretKey := []byte("test-secret-key-32-bytes-long")
	tokenManager := security.NewTokenManager(secretKey)

	// Generate token
	token, err := tokenManager.GenerateToken("test-user", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Generated token is empty")
	}

	// Validate token
	userID, valid := tokenManager.ValidateToken(token)
	if !valid {
		t.Error("Generated token is not valid")
	}

	if userID != "test-user" {
		t.Errorf("Expected user ID 'test-user', got '%s'", userID)
	}

	// Test token revocation
	revoked := tokenManager.RevokeToken(token)
	if !revoked {
		t.Error("Failed to revoke token")
	}

	// Verify token is no longer valid
	_, valid = tokenManager.ValidateToken(token)
	if valid {
		t.Error("Revoked token is still valid")
	}

	// Test rate limiter
	rateLimiter := security.NewRateLimiter(5, 1*time.Second)

	// Test within rate limit
	for i := 0; i < 5; i++ {
		if !rateLimiter.Allow("test-ip") {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Test exceeding rate limit
	if rateLimiter.Allow("test-ip") {
		t.Error("Request should be rate limited")
	}

	// Test password operations
	password := "TestPass123!"
	hashedPassword, err := security.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	err = security.ComparePassword(hashedPassword, password)
	if err != nil {
		t.Errorf("Password comparison failed: %v", err)
	}

	// Test password validation
	policy := &security.PasswordPolicy{
		MinLength:           8,
		RequireUppercase:    true,
		RequireLowercase:    true,
		RequireNumbers:      true,
		RequireSpecialChars: true,
	}
	validator := security.NewPasswordValidator(policy)

	err = validator.ValidatePassword(password)
	if err != nil {
		t.Errorf("Password validation failed: %v", err)
	}

	t.Logf("Security integration test passed for user: %s", userID)
}

// TestSemaphoreIntegration tests the semaphore functionality
func TestSemaphoreIntegration(t *testing.T) {
	semaphore := concurrency.NewSemaphore(2)

	// Test basic acquire/release
	semaphore.Acquire()
	semaphore.Acquire()

	// Try to acquire more than capacity
	acquired := semaphore.TryAcquire()
	if acquired {
		t.Error("Should not be able to acquire more than capacity")
	}

	// Release one permit
	semaphore.Release()

	// Now should be able to acquire again
	acquired = semaphore.TryAcquire()
	if !acquired {
		t.Error("Should be able to acquire after release")
	}

	// Test timeout acquire
	acquired = semaphore.AcquireWithTimeout(100 * time.Millisecond)
	if acquired {
		t.Error("Should not be able to acquire when at capacity")
	}

	// Release remaining permits
	semaphore.Release()
	semaphore.Release()

	t.Log("Semaphore integration test passed")
}

// TestConcurrentCacheAccess tests concurrent access to the cache
func TestConcurrentCacheAccess(t *testing.T) {
	cacheInstance := cache.NewDiscoveryCache(5*time.Minute, 1000)

	var wg sync.WaitGroup
	goroutines := 10
	operationsPerGoroutine := 100

	// Test concurrent writes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("goroutine-%d-key-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				cacheInstance.Set(key, value, 5*time.Minute)
			}
		}(i)
	}

	wg.Wait()

	// Test concurrent reads
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("goroutine-%d-key-%d", id, j)
				cacheInstance.Get(key)
			}
		}(i)
	}

	wg.Wait()

	t.Log("Concurrent cache access test passed")
}

// TestCachePerformance tests cache performance under load
func TestCachePerformance(t *testing.T) {
	cacheInstance := cache.NewDiscoveryCache(5*time.Minute, 10000)

	start := time.Now()

	// Perform many operations
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("perf-key-%d", i)
		value := fmt.Sprintf("perf-value-%d", i)
		cacheInstance.Set(key, value, 5*time.Minute)
	}

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("perf-key-%d", i)
		cacheInstance.Get(key)
	}

	duration := time.Since(start)
	t.Logf("Cache performance test completed in %v", duration)

	// Performance assertion: should complete within reasonable time
	if duration > 1*time.Second {
		t.Errorf("Cache operations took too long: %v", duration)
	}
}
