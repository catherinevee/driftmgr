package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimiter(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		rl := NewRateLimiter(nil)
		assert.NotNil(t, rl)
		assert.NotNil(t, rl.config)
		assert.Equal(t, 10, rl.config.RequestsPerSecond)
	})
	
	t.Run("with custom config", func(t *testing.T) {
		config := &Config{
			RequestsPerSecond: 5,
			BurstSize:         10,
		}
		rl := NewRateLimiter(config)
		assert.NotNil(t, rl)
		assert.Equal(t, 5, rl.config.RequestsPerSecond)
		assert.Equal(t, 10, rl.config.BurstSize)
	})
}

func TestRateLimiterAllow(t *testing.T) {
	config := &Config{
		RequestsPerSecond:       2,
		BurstSize:               4,
		GlobalRequestsPerSecond: 100,
		GlobalBurstSize:         200,
		MaxViolations:           3,
		BanDuration:             1 * time.Second,
		EnableDDoSProtection:    false,
	}
	
	rl := NewRateLimiter(config)
	identifier := "test-user"
	
	t.Run("allows requests within limit", func(t *testing.T) {
		// Should allow burst
		for i := 0; i < 4; i++ {
			assert.True(t, rl.Allow(identifier), "request %d should be allowed", i)
		}
		
		// Should be rate limited
		assert.False(t, rl.Allow(identifier), "5th request should be denied")
	})
	
	t.Run("bans after max violations", func(t *testing.T) {
		identifier := "bad-user"
		
		// Use up burst
		for i := 0; i < 4; i++ {
			rl.Allow(identifier)
		}
		
		// Exceed limit multiple times
		for i := 0; i < 3; i++ {
			assert.False(t, rl.Allow(identifier))
		}
		
		// Should be banned now
		assert.True(t, rl.isBanned(identifier))
		
		// Wait for ban to expire
		time.Sleep(1100 * time.Millisecond)
		assert.False(t, rl.isBanned(identifier))
	})
}

func TestRateLimiterAllowN(t *testing.T) {
	config := &Config{
		RequestsPerSecond:       10,
		BurstSize:               20,
		GlobalRequestsPerSecond: 100,
		GlobalBurstSize:         200,
	}
	
	rl := NewRateLimiter(config)
	identifier := "test-user"
	
	t.Run("allows N requests within burst", func(t *testing.T) {
		assert.True(t, rl.AllowN(identifier, 10))
		assert.True(t, rl.AllowN(identifier, 10))
		assert.False(t, rl.AllowN(identifier, 10)) // Exceeds burst
	})
	
	t.Run("denies when banned", func(t *testing.T) {
		rl.ban("banned-user")
		assert.False(t, rl.AllowN("banned-user", 1))
	})
}

func TestRateLimiterWait(t *testing.T) {
	config := &Config{
		RequestsPerSecond:       10,
		BurstSize:               10,
		GlobalRequestsPerSecond: 100,
		GlobalBurstSize:         100,
	}
	
	rl := NewRateLimiter(config)
	
	t.Run("waits for available token", func(t *testing.T) {
		ctx := context.Background()
		identifier := "wait-user"
		
		// Use up burst
		for i := 0; i < 10; i++ {
			require.True(t, rl.Allow(identifier))
		}
		
		// Should wait for next token
		start := time.Now()
		err := rl.Wait(ctx, identifier)
		elapsed := time.Since(start)
		
		assert.NoError(t, err)
		assert.True(t, elapsed >= 90*time.Millisecond) // ~100ms for 10 req/s
	})
	
	t.Run("returns error when banned", func(t *testing.T) {
		ctx := context.Background()
		rl.ban("banned-wait-user")
		
		err := rl.Wait(ctx, "banned-wait-user")
		assert.Error(t, err)
	})
	
	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		identifier := "cancel-user"
		
		// Use up burst
		for i := 0; i < 10; i++ {
			rl.Allow(identifier)
		}
		
		// Cancel context immediately
		cancel()
		
		err := rl.Wait(ctx, identifier)
		assert.Error(t, err)
	})
}

func TestRateLimiterConcurrency(t *testing.T) {
	config := &Config{
		RequestsPerSecond:       100,
		BurstSize:               200,
		GlobalRequestsPerSecond: 1000,
		GlobalBurstSize:         2000,
	}
	
	rl := NewRateLimiter(config)
	
	t.Run("handles concurrent requests", func(t *testing.T) {
		var wg sync.WaitGroup
		successCount := 0
		mu := sync.Mutex{}
		
		// Launch 100 goroutines
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				identifier := "concurrent-user"
				
				if rl.Allow(identifier) {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}
		
		wg.Wait()
		
		// Should allow up to burst size
		assert.LessOrEqual(t, successCount, 200)
		assert.Greater(t, successCount, 0)
	})
}

func TestRateLimiterCleanup(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 10,
		BurstSize:         20,
		CleanupInterval:   100 * time.Millisecond,
		LimiterTTL:        200 * time.Millisecond,
	}
	
	rl := NewRateLimiter(config)
	
	// Create limiter
	rl.Allow("cleanup-user")
	
	// Verify limiter exists
	rl.mu.RLock()
	_, exists := rl.limiters["cleanup-user"]
	rl.mu.RUnlock()
	assert.True(t, exists)
	
	// Wait for cleanup
	time.Sleep(350 * time.Millisecond)
	
	// Verify limiter was cleaned up
	rl.mu.RLock()
	_, exists = rl.limiters["cleanup-user"]
	rl.mu.RUnlock()
	assert.False(t, exists)
}

func TestRateLimiterDDoSProtection(t *testing.T) {
	config := &Config{
		RequestsPerSecond:          10,
		BurstSize:                  20,
		EnableDDoSProtection:       true,
		SuspiciousRequestThreshold: 50,
		WindowSize:                 1 * time.Second,
		BanDuration:                1 * time.Second,
		MaxViolations:              5,
	}
	
	rl := NewRateLimiter(config)
	identifier := "ddos-attacker"
	
	// Simulate DDoS attack
	for i := 0; i < 100; i++ {
		rl.Allow(identifier)
	}
	
	// Should be banned
	assert.True(t, rl.isBanned(identifier))
}

func TestRateLimiterMiddleware(t *testing.T) {
	config := &Config{
		RequestsPerSecond: 2,
		BurstSize:         4,
	}
	
	rl := NewRateLimiter(config)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	middleware := rl.Middleware(handler)
	
	t.Run("allows requests within limit", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			rec := httptest.NewRecorder()
			
			middleware(rec, req)
			
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
	
	t.Run("blocks requests over limit", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()
		
		middleware(rec, req)
		
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})
	
	t.Run("adds rate limit headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.168.1.2:1234"
		rec := httptest.NewRecorder()
		
		middleware(rec, req)
		
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
	})
}

func TestRateLimiterGetStats(t *testing.T) {
	rl := NewRateLimiter(nil)
	
	// Create some limiters
	rl.Allow("user1")
	rl.Allow("user2")
	rl.ban("banned-user")
	
	stats := rl.GetStats()
	
	assert.Equal(t, 2, stats["active_limiters"])
	assert.Equal(t, 1, stats["banned_count"])
	assert.NotNil(t, stats["config"])
}

func TestRateLimiterReset(t *testing.T) {
	rl := NewRateLimiter(nil)
	identifier := "reset-user"
	
	// Create limiter and ban
	rl.Allow(identifier)
	rl.ban(identifier)
	
	// Verify exists
	assert.True(t, rl.isBanned(identifier))
	
	// Reset
	rl.Reset(identifier)
	
	// Verify cleared
	assert.False(t, rl.isBanned(identifier))
	
	rl.mu.RLock()
	_, exists := rl.limiters[identifier]
	rl.mu.RUnlock()
	assert.False(t, exists)
}

func BenchmarkRateLimiterAllow(b *testing.B) {
	rl := NewRateLimiter(&Config{
		RequestsPerSecond: 1000,
		BurstSize:         2000,
	})
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		identifier := "bench-user"
		for pb.Next() {
			rl.Allow(identifier)
		}
	})
}

func BenchmarkRateLimiterConcurrent(b *testing.B) {
	rl := NewRateLimiter(&Config{
		RequestsPerSecond:       1000,
		BurstSize:               2000,
		GlobalRequestsPerSecond: 10000,
		GlobalBurstSize:         20000,
	})
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			identifier := string(rune(i % 100))
			rl.Allow(identifier)
			i++
		}
	})
}