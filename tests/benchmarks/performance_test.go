package benchmarks

import (
	"fmt"
	"testing"
	"time"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/security"
)

func BenchmarkCachePerformance(b *testing.B) {
	// Create cache
	c := cache.NewDiscoveryCache(5*time.Minute, 1000)
	
	// Benchmark cache operations
	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i)
			value := fmt.Sprintf("value-%d", i)
			c.Set(key, value, 5*time.Minute)
		}
	})

	b.Run("Get", func(b *testing.B) {
		// Pre-populate cache
		for i := 0; i < 100; i++ {
			key := fmt.Sprintf("key-%d", i)
			value := fmt.Sprintf("value-%d", i)
			c.Set(key, value, 5*time.Minute)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := fmt.Sprintf("key-%d", i%100)
			c.Get(key)
		}
	})
}

func BenchmarkSecurityOperations(b *testing.B) {
	secretKey := []byte("test-secret-key")
	tm := security.NewTokenManager(secretKey)

	b.Run("TokenGeneration", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			userID := fmt.Sprintf("user-%d", i)
			tm.GenerateToken(userID, 1*time.Hour)
		}
	})

	b.Run("TokenValidation", func(b *testing.B) {
		// Pre-generate tokens
		tokens := make([]string, 100)
		for i := 0; i < 100; i++ {
			userID := fmt.Sprintf("user-%d", i)
			token, _ := tm.GenerateToken(userID, 1*time.Hour)
			tokens[i] = token
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tm.ValidateToken(tokens[i%100])
		}
	})
}

func BenchmarkRateLimiter(b *testing.B) {
	rl := security.NewRateLimiter(1000, 1*time.Minute)
	clientIP := "192.168.1.1"

	b.Run("AllowRequests", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rl.Allow(clientIP)
		}
	})
}

func BenchmarkPasswordOperations(b *testing.B) {
	policy := &security.PasswordPolicy{
		MinLength:          8,
		RequireUppercase:   true,
		RequireLowercase:   true,
		RequireNumbers:     true,
		RequireSpecialChars: true,
	}
	validator := security.NewPasswordValidator(policy)

	b.Run("PasswordValidation", func(b *testing.B) {
		password := "TestPass123!"
		for i := 0; i < b.N; i++ {
			validator.ValidatePassword(password)
		}
	})

	b.Run("PasswordHashing", func(b *testing.B) {
		password := "test-password-123"
		for i := 0; i < b.N; i++ {
			security.HashPassword(password)
		}
	})

	b.Run("PasswordGeneration", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			security.GenerateSecurePassword(16)
		}
	})
}

func BenchmarkConcurrentOperations(b *testing.B) {
	c := cache.NewDiscoveryCache(5*time.Minute, 1000)
	
	b.Run("ConcurrentCacheAccess", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key-%d", i)
				value := fmt.Sprintf("value-%d", i)
				c.Set(key, value, 5*time.Minute)
				c.Get(key)
				i++
			}
		})
	})
}

func BenchmarkMemoryUsage(b *testing.B) {
	// Test memory usage for large datasets
	b.Run("LargeCache", func(b *testing.B) {
		c := cache.NewDiscoveryCache(5*time.Minute, 10000)
		
		for i := 0; i < b.N; i++ {
			// Add 1000 items
			for j := 0; j < 1000; j++ {
				key := fmt.Sprintf("large-key-%d-%d", i, j)
				value := fmt.Sprintf("large-value-%d-%d", i, j)
				c.Set(key, value, 5*time.Minute)
			}
		}
	})
}

// Helper function to generate test data
func generateTestData(count int) map[string]interface{} {
	data := make(map[string]interface{})
	for i := 0; i < count; i++ {
		key := fmt.Sprintf("test-key-%d", i)
		value := map[string]interface{}{
			"id":       fmt.Sprintf("resource-%d", i),
			"name":     fmt.Sprintf("Resource %d", i),
			"type":     "test_resource",
			"provider": "test",
			"region":   "us-east-1",
		}
		data[key] = value
	}
	return data
}
