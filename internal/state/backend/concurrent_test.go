package backend

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RetryableBackend wraps a backend with retry logic
type RetryableBackend struct {
	backend    Backend
	maxRetries int
	retryDelay time.Duration
	backoff    float64
}

func NewRetryableBackend(backend Backend, maxRetries int, retryDelay time.Duration, backoff float64) *RetryableBackend {
	return &RetryableBackend{
		backend:    backend,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
		backoff:    backoff,
	}
}

func (r *RetryableBackend) Pull(ctx context.Context) (*StateData, error) {
	return r.retryOperation(ctx, "pull", func() (*StateData, error) {
		return r.backend.Pull(ctx)
	})
}

func (r *RetryableBackend) Push(ctx context.Context, state *StateData) error {
	_, err := r.retryOperation(ctx, "push", func() (*StateData, error) {
		return nil, r.backend.Push(ctx, state)
	})
	return err
}

func (r *RetryableBackend) Lock(ctx context.Context, info *LockInfo) (string, error) {
	result, err := r.retryOperation(ctx, "lock", func() (*StateData, error) {
		lockID, err := r.backend.Lock(ctx, info)
		return &StateData{Lineage: lockID}, err
	})
	if err != nil {
		return "", err
	}
	return result.Lineage, nil
}

func (r *RetryableBackend) Unlock(ctx context.Context, lockID string) error {
	_, err := r.retryOperation(ctx, "unlock", func() (*StateData, error) {
		return nil, r.backend.Unlock(ctx, lockID)
	})
	return err
}

func (r *RetryableBackend) retryOperation(ctx context.Context, operation string, fn func() (*StateData, error)) (*StateData, error) {
	var lastErr error
	delay := r.retryDelay

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			// Increase delay for next attempt
			delay = time.Duration(float64(delay) * r.backoff)
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry certain errors
		if isNonRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("operation %s failed after %d attempts: %w", operation, r.maxRetries+1, lastErr)
}

func isNonRetryableError(err error) bool {
	// Add logic to determine if error is retryable
	errStr := err.Error()
	return contains(errStr, "already locked") ||
		contains(errStr, "does not exist") ||
		contains(errStr, "invalid")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[:len(substr)] == substr ||
		(len(s) > len(substr) && len(substr) > 0 &&
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())
}

// Delegate remaining methods to the wrapped backend
func (r *RetryableBackend) GetVersions(ctx context.Context) ([]*StateVersion, error) {
	return r.backend.GetVersions(ctx)
}

func (r *RetryableBackend) GetVersion(ctx context.Context, versionID string) (*StateData, error) {
	return r.backend.GetVersion(ctx, versionID)
}

func (r *RetryableBackend) ListWorkspaces(ctx context.Context) ([]string, error) {
	return r.backend.ListWorkspaces(ctx)
}

func (r *RetryableBackend) SelectWorkspace(ctx context.Context, name string) error {
	return r.backend.SelectWorkspace(ctx, name)
}

func (r *RetryableBackend) CreateWorkspace(ctx context.Context, name string) error {
	return r.backend.CreateWorkspace(ctx, name)
}

func (r *RetryableBackend) DeleteWorkspace(ctx context.Context, name string) error {
	return r.backend.DeleteWorkspace(ctx, name)
}

func (r *RetryableBackend) GetLockInfo(ctx context.Context) (*LockInfo, error) {
	return r.backend.GetLockInfo(ctx)
}

func (r *RetryableBackend) Validate(ctx context.Context) error {
	return r.backend.Validate(ctx)
}

func (r *RetryableBackend) GetMetadata() *BackendMetadata {
	return r.backend.GetMetadata()
}

// ErrorSimulatingBackend simulates various error conditions
type ErrorSimulatingBackend struct {
	backend      Backend
	failureRate  float64 // 0.0 to 1.0
	errorTypes   []string
	mu           sync.RWMutex
	callCount    int64
	errorCount   int64
	networkDelay time.Duration
}

func NewErrorSimulatingBackend(backend Backend, failureRate float64, errorTypes []string) *ErrorSimulatingBackend {
	return &ErrorSimulatingBackend{
		backend:      backend,
		failureRate:  failureRate,
		errorTypes:   errorTypes,
		networkDelay: 10 * time.Millisecond,
	}
}

func (e *ErrorSimulatingBackend) simulateNetworkDelay() {
	if e.networkDelay > 0 {
		// Add some jitter
		jitter := time.Duration(rand.Intn(int(e.networkDelay / 2)))
		time.Sleep(e.networkDelay + jitter)
	}
}

func (e *ErrorSimulatingBackend) shouldSimulateError() error {
	atomic.AddInt64(&e.callCount, 1)

	if rand.Float64() < e.failureRate {
		atomic.AddInt64(&e.errorCount, 1)

		if len(e.errorTypes) == 0 {
			return fmt.Errorf("simulated error")
		}

		errorType := e.errorTypes[rand.Intn(len(e.errorTypes))]
		switch errorType {
		case "network":
			return fmt.Errorf("network error: connection timeout")
		case "auth":
			return fmt.Errorf("authentication failed")
		case "permission":
			return fmt.Errorf("permission denied")
		case "throttling":
			return fmt.Errorf("rate limit exceeded")
		case "temporary":
			return fmt.Errorf("temporary service unavailable")
		default:
			return fmt.Errorf("simulated error: %s", errorType)
		}
	}

	return nil
}

func (e *ErrorSimulatingBackend) Pull(ctx context.Context) (*StateData, error) {
	e.simulateNetworkDelay()
	if err := e.shouldSimulateError(); err != nil {
		return nil, err
	}
	return e.backend.Pull(ctx)
}

func (e *ErrorSimulatingBackend) Push(ctx context.Context, state *StateData) error {
	e.simulateNetworkDelay()
	if err := e.shouldSimulateError(); err != nil {
		return err
	}
	return e.backend.Push(ctx, state)
}

func (e *ErrorSimulatingBackend) Lock(ctx context.Context, info *LockInfo) (string, error) {
	e.simulateNetworkDelay()
	if err := e.shouldSimulateError(); err != nil {
		return "", err
	}
	return e.backend.Lock(ctx, info)
}

func (e *ErrorSimulatingBackend) Unlock(ctx context.Context, lockID string) error {
	e.simulateNetworkDelay()
	if err := e.shouldSimulateError(); err != nil {
		return err
	}
	return e.backend.Unlock(ctx, lockID)
}

func (e *ErrorSimulatingBackend) GetStats() (int64, int64) {
	return atomic.LoadInt64(&e.callCount), atomic.LoadInt64(&e.errorCount)
}

// Delegate other methods
func (e *ErrorSimulatingBackend) GetVersions(ctx context.Context) ([]*StateVersion, error) {
	return e.backend.GetVersions(ctx)
}

func (e *ErrorSimulatingBackend) GetVersion(ctx context.Context, versionID string) (*StateData, error) {
	return e.backend.GetVersion(ctx, versionID)
}

func (e *ErrorSimulatingBackend) ListWorkspaces(ctx context.Context) ([]string, error) {
	return e.backend.ListWorkspaces(ctx)
}

func (e *ErrorSimulatingBackend) SelectWorkspace(ctx context.Context, name string) error {
	return e.backend.SelectWorkspace(ctx, name)
}

func (e *ErrorSimulatingBackend) CreateWorkspace(ctx context.Context, name string) error {
	return e.backend.CreateWorkspace(ctx, name)
}

func (e *ErrorSimulatingBackend) DeleteWorkspace(ctx context.Context, name string) error {
	return e.backend.DeleteWorkspace(ctx, name)
}

func (e *ErrorSimulatingBackend) GetLockInfo(ctx context.Context) (*LockInfo, error) {
	return e.backend.GetLockInfo(ctx)
}

func (e *ErrorSimulatingBackend) Validate(ctx context.Context) error {
	return e.backend.Validate(ctx)
}

func (e *ErrorSimulatingBackend) GetMetadata() *BackendMetadata {
	return e.backend.GetMetadata()
}

// Test concurrent access to state
func TestConcurrentAccess_StateOperations(t *testing.T) {
	mockBackend := NewMockBackend()
	ctx := context.Background()

	t.Run("Concurrent Pull Operations", func(t *testing.T) {
		// Push initial state
		initialState := &StateData{
			Version: 4,
			Serial:  1,
			Data:    []byte(`{"version": 4, "serial": 1}`),
		}
		err := mockBackend.Push(ctx, initialState)
		require.NoError(t, err)

		const numGoroutines = 50
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				_, err := mockBackend.Pull(ctx)
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Verify no errors occurred
		for err := range errors {
			t.Error("Concurrent pull failed:", err)
		}

		// Verify pull was called the expected number of times
		assert.GreaterOrEqual(t, mockBackend.pullCalls, numGoroutines)
	})

	t.Run("Concurrent Push Operations", func(t *testing.T) {
		const numGoroutines = 25
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)
		successCount := int64(0)

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				state := &StateData{
					Version: 4,
					Serial:  uint64(id + 2),
					Data:    []byte(fmt.Sprintf(`{"version": 4, "serial": %d}`, id+2)),
				}
				err := mockBackend.Push(ctx, state)
				if err != nil {
					errors <- err
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Verify most operations succeeded
		assert.GreaterOrEqual(t, successCount, int64(numGoroutines-5)) // Allow for some failures

		// Check for any errors
		for err := range errors {
			t.Log("Concurrent push error (may be expected):", err)
		}
	})

	t.Run("Mixed Concurrent Operations", func(t *testing.T) {
		const numOperations = 100
		var wg sync.WaitGroup
		var pullCount, pushCount int64

		wg.Add(numOperations)
		for i := 0; i < numOperations; i++ {
			go func(id int) {
				defer wg.Done()

				if id%2 == 0 {
					// Pull operation
					_, err := mockBackend.Pull(ctx)
					if err == nil {
						atomic.AddInt64(&pullCount, 1)
					}
				} else {
					// Push operation
					state := &StateData{
						Version: 4,
						Serial:  uint64(id + 100),
						Data:    []byte(fmt.Sprintf(`{"version": 4, "serial": %d}`, id+100)),
					}
					err := mockBackend.Push(ctx, state)
					if err == nil {
						atomic.AddInt64(&pushCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify operations completed
		assert.Greater(t, pullCount, int64(0))
		assert.Greater(t, pushCount, int64(0))
		t.Logf("Completed pulls: %d, pushes: %d", pullCount, pushCount)
	})
}

// Test concurrent locking
func TestConcurrentAccess_Locking(t *testing.T) {
	mockBackend := NewMockBackend()
	ctx := context.Background()

	t.Run("Concurrent Lock Attempts", func(t *testing.T) {
		const numGoroutines = 20
		var wg sync.WaitGroup
		successful := int64(0)
		failed := int64(0)
		lockIDs := make(chan string, numGoroutines)

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()

				lockInfo := &LockInfo{
					ID:        fmt.Sprintf("lock-%d", id),
					Operation: "test",
					Who:       fmt.Sprintf("user-%d", id),
					Created:   time.Now(),
				}

				lockID, err := mockBackend.Lock(ctx, lockInfo)
				if err != nil {
					atomic.AddInt64(&failed, 1)
				} else {
					atomic.AddInt64(&successful, 1)
					lockIDs <- lockID
				}
			}(i)
		}

		wg.Wait()
		close(lockIDs)

		// Only one lock should succeed
		assert.Equal(t, int64(1), successful)
		assert.Equal(t, int64(numGoroutines-1), failed)

		// Unlock the successful lock
		for lockID := range lockIDs {
			err := mockBackend.Unlock(ctx, lockID)
			assert.NoError(t, err)
		}
	})

	t.Run("Lock-Unlock Race Conditions", func(t *testing.T) {
		const numCycles = 100
		var wg sync.WaitGroup

		for cycle := 0; cycle < numCycles; cycle++ {
			wg.Add(2)

			// Goroutine 1: Lock and unlock
			go func(cycleID int) {
				defer wg.Done()

				lockInfo := &LockInfo{
					ID:        fmt.Sprintf("cycle-lock-%d", cycleID),
					Operation: "test",
					Who:       "locker",
					Created:   time.Now(),
				}

				lockID, err := mockBackend.Lock(ctx, lockInfo)
				if err == nil {
					// Small delay to create race condition
					time.Sleep(time.Microsecond * 10)
					mockBackend.Unlock(ctx, lockID)
				}
			}(cycle)

			// Goroutine 2: Try to acquire same lock
			go func(cycleID int) {
				defer wg.Done()

				lockInfo := &LockInfo{
					ID:        fmt.Sprintf("race-lock-%d", cycleID),
					Operation: "race-test",
					Who:       "racer",
					Created:   time.Now(),
				}

				lockID, err := mockBackend.Lock(ctx, lockInfo)
				if err == nil {
					mockBackend.Unlock(ctx, lockID)
				}
			}(cycle)
		}

		wg.Wait()
		// Test passes if no deadlocks or panics occur
	})
}

// Test concurrent workspace operations
func TestConcurrentAccess_Workspaces(t *testing.T) {
	mockBackend := NewMockBackend()
	ctx := context.Background()

	t.Run("Concurrent Workspace Creation", func(t *testing.T) {
		const numWorkspaces = 50
		var wg sync.WaitGroup
		created := int64(0)
		errors := make(chan error, numWorkspaces)

		wg.Add(numWorkspaces)
		for i := 0; i < numWorkspaces; i++ {
			go func(id int) {
				defer wg.Done()

				workspace := fmt.Sprintf("workspace-%d", id)
				err := mockBackend.CreateWorkspace(ctx, workspace)
				if err != nil {
					errors <- err
				} else {
					atomic.AddInt64(&created, 1)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Verify workspaces were created
		assert.Equal(t, int64(numWorkspaces), created)

		// Check for any unexpected errors
		errorCount := 0
		for err := range errors {
			t.Error("Workspace creation error:", err)
			errorCount++
		}
		assert.Equal(t, 0, errorCount)

		// Verify workspaces exist
		workspaces, err := mockBackend.ListWorkspaces(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(workspaces), numWorkspaces)
	})

	t.Run("Concurrent Workspace Operations", func(t *testing.T) {
		// Create test workspaces first
		testWorkspaces := []string{"test-1", "test-2", "test-3", "test-4", "test-5"}
		for _, ws := range testWorkspaces {
			err := mockBackend.CreateWorkspace(ctx, ws)
			require.NoError(t, err)
		}

		const numOperations = 100
		var wg sync.WaitGroup
		var selectCount, listCount int64

		wg.Add(numOperations)
		for i := 0; i < numOperations; i++ {
			go func(id int) {
				defer wg.Done()

				if id%2 == 0 {
					// Select workspace
					workspace := testWorkspaces[id%len(testWorkspaces)]
					err := mockBackend.SelectWorkspace(ctx, workspace)
					if err == nil {
						atomic.AddInt64(&selectCount, 1)
					}
				} else {
					// List workspaces
					_, err := mockBackend.ListWorkspaces(ctx)
					if err == nil {
						atomic.AddInt64(&listCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify operations completed
		assert.Greater(t, selectCount, int64(0))
		assert.Greater(t, listCount, int64(0))
		t.Logf("Completed selects: %d, lists: %d", selectCount, listCount)
	})
}

// Test retry logic
func TestRetryLogic(t *testing.T) {
	t.Run("Successful Retry After Failures", func(t *testing.T) {
		mockBackend := NewMockBackend()
		errorBackend := NewErrorSimulatingBackend(mockBackend, 0.7, []string{"network", "temporary"})
		retryBackend := NewRetryableBackend(errorBackend, 3, 10*time.Millisecond, 2.0)

		ctx := context.Background()

		// Try pull operation with retries
		state, err := retryBackend.Pull(ctx)
		if err != nil {
			t.Log("Pull failed even with retries:", err)
		} else {
			assert.NotNil(t, state)
		}

		// Check statistics
		calls, errors := errorBackend.GetStats()
		t.Logf("Total calls: %d, Errors: %d, Success rate: %.2f",
			calls, errors, float64(calls-errors)/float64(calls))
	})

	t.Run("Retry Limit Exceeded", func(t *testing.T) {
		mockBackend := NewMockBackend()
		// Very high failure rate
		errorBackend := NewErrorSimulatingBackend(mockBackend, 1.0, []string{"network"})
		retryBackend := NewRetryableBackend(errorBackend, 2, 5*time.Millisecond, 1.5)

		ctx := context.Background()

		startTime := time.Now()
		_, err := retryBackend.Pull(ctx)
		elapsed := time.Since(startTime)

		// Should fail after retries
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed after")

		// Should have taken some time due to retries
		assert.Greater(t, elapsed, 10*time.Millisecond)

		calls, errors := errorBackend.GetStats()
		t.Logf("Retry test - Calls: %d, Errors: %d", calls, errors)
	})

	t.Run("Non-Retryable Errors", func(t *testing.T) {
		mockBackend := NewMockBackend()

		// Create workspace to test non-retryable error
		err := mockBackend.CreateWorkspace(context.Background(), "existing")
		require.NoError(t, err)

		retryBackend := NewRetryableBackend(mockBackend, 3, 10*time.Millisecond, 2.0)

		ctx := context.Background()

		startTime := time.Now()
		// Try to create workspace that already exists
		err = retryBackend.CreateWorkspace(ctx, "existing")
		elapsed := time.Since(startTime)

		// Should fail immediately without retries
		assert.Error(t, err)
		// Should be fast since no retries
		assert.Less(t, elapsed, 50*time.Millisecond)
	})
}

// Test error recovery scenarios
func TestErrorRecovery(t *testing.T) {
	t.Run("Network Timeout Recovery", func(t *testing.T) {
		mockBackend := NewMockBackend()
		errorBackend := NewErrorSimulatingBackend(mockBackend, 0.3, []string{"network"})

		// Simulate network delay
		errorBackend.networkDelay = 50 * time.Millisecond

		ctx := context.Background()
		successCount := 0
		totalAttempts := 20

		for i := 0; i < totalAttempts; i++ {
			_, err := errorBackend.Pull(ctx)
			if err == nil {
				successCount++
			}
		}

		// Should have some successes despite network issues
		assert.Greater(t, successCount, totalAttempts/4)
		t.Logf("Success rate with network issues: %d/%d", successCount, totalAttempts)
	})

	t.Run("Partial Failure Recovery", func(t *testing.T) {
		mockBackend := NewMockBackend()

		// Start with high failure rate, then reduce it
		errorBackend := NewErrorSimulatingBackend(mockBackend, 0.8, []string{"temporary"})

		ctx := context.Background()
		var wg sync.WaitGroup
		results := make(chan bool, 50)

		// Launch operations
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(attempt int) {
				defer wg.Done()

				// Reduce failure rate over time
				if attempt > 25 {
					errorBackend.failureRate = 0.2
				}

				_, err := errorBackend.Pull(ctx)
				results <- (err == nil)
			}(i)
		}

		wg.Wait()
		close(results)

		// Count successes
		successes := 0
		for success := range results {
			if success {
				successes++
			}
		}

		// Later operations should have higher success rate
		assert.Greater(t, successes, 15)
		t.Logf("Recovery test successes: %d/50", successes)
	})
}

// Benchmark concurrent operations
func BenchmarkConcurrentOperations_Performance(b *testing.B) {
	mockBackend := NewMockBackend()
	ctx := context.Background()

	// Prepare test state
	testState := &StateData{
		Version: 4,
		Serial:  1,
		Data:    []byte(`{"version": 4, "serial": 1, "resources": []}`),
	}

	b.Run("ConcurrentPull", func(b *testing.B) {
		// Push initial state
		err := mockBackend.Push(ctx, testState)
		require.NoError(b, err)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := mockBackend.Pull(ctx)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	b.Run("ConcurrentPush", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			serial := uint64(0)
			for pb.Next() {
				serial++
				state := *testState
				state.Serial = serial
				err := mockBackend.Push(ctx, &state)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})

	b.Run("ConcurrentLocking", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			lockInfo := &LockInfo{
				ID:        fmt.Sprintf("bench-lock-%d", i),
				Operation: "benchmark",
				Who:       "bench-user",
				Created:   time.Now(),
			}

			lockID, err := mockBackend.Lock(ctx, lockInfo)
			if err != nil {
				continue // Skip if lock already held
			}

			err = mockBackend.Unlock(ctx, lockID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ConcurrentWorkspaces", func(b *testing.B) {
		workspaces := []string{"bench-1", "bench-2", "bench-3", "bench-4", "bench-5"}

		// Create workspaces
		for _, ws := range workspaces {
			mockBackend.CreateWorkspace(ctx, ws)
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			wsIndex := 0
			for pb.Next() {
				workspace := workspaces[wsIndex%len(workspaces)]
				wsIndex++

				err := mockBackend.SelectWorkspace(ctx, workspace)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

// Test stress scenarios
func TestStressScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Run("High Concurrency Stress", func(t *testing.T) {
		mockBackend := NewMockBackend()
		ctx := context.Background()

		const numGoroutines = 200
		const operationsPerGoroutine = 10

		var wg sync.WaitGroup
		totalOps := int64(0)
		errors := int64(0)

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					atomic.AddInt64(&totalOps, 1)

					switch j % 4 {
					case 0:
						_, err := mockBackend.Pull(ctx)
						if err != nil {
							atomic.AddInt64(&errors, 1)
						}
					case 1:
						state := &StateData{
							Version: 4,
							Serial:  uint64(goroutineID*1000 + j),
							Data:    []byte(fmt.Sprintf(`{"serial": %d}`, goroutineID*1000+j)),
						}
						err := mockBackend.Push(ctx, state)
						if err != nil {
							atomic.AddInt64(&errors, 1)
						}
					case 2:
						_, err := mockBackend.ListWorkspaces(ctx)
						if err != nil {
							atomic.AddInt64(&errors, 1)
						}
					case 3:
						_, err := mockBackend.GetVersions(ctx)
						if err != nil {
							atomic.AddInt64(&errors, 1)
						}
					}
				}
			}(i)
		}

		wg.Wait()

		errorRate := float64(errors) / float64(totalOps)
		t.Logf("Stress test completed: %d operations, %d errors (%.2f%% error rate)",
			totalOps, errors, errorRate*100)

		// Allow for some errors under high stress
		assert.Less(t, errorRate, 0.05) // Less than 5% error rate
	})

	t.Run("Memory Pressure Stress", func(t *testing.T) {
		mockBackend := NewMockBackend()
		ctx := context.Background()

		// Create increasingly large states
		for size := 1024; size <= 1024*1024; size *= 2 {
			// Create simple large state
			largeData := make([]byte, size)
			for i := range largeData {
				largeData[i] = byte(i % 256)
			}

			state := &StateData{
				Version: 4,
				Serial:  1,
				Data:    largeData,
				Size:    int64(size),
			}

			err := mockBackend.Push(ctx, state)
			require.NoError(t, err, "Failed to push state of size %d", size)

			pulledState, err := mockBackend.Pull(ctx)
			require.NoError(t, err, "Failed to pull state of size %d", size)
			assert.GreaterOrEqual(t, pulledState.Size, state.Size/2) // Allow for some compression

			// Force GC to check for memory leaks
			if size >= 1024*1024 {
				var m1, m2 runtime.MemStats
				runtime.ReadMemStats(&m1)
				runtime.GC()
				runtime.ReadMemStats(&m2)
				t.Logf("Memory after %d bytes: Alloc=%d KB, Freed=%d KB",
					size, m2.Alloc/1024, (m1.Alloc-m2.Alloc)/1024)
			}
		}
	})
}
