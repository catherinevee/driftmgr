package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewGlobalCache(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 15*time.Minute, "/tmp/cache")

	assert.NotNil(t, cache)
	assert.NotNil(t, cache.items)
	assert.Equal(t, int64(1024*1024), cache.maxSize)
	assert.Equal(t, 15*time.Minute, cache.defaultTTL)
	assert.Equal(t, "/tmp/cache", cache.persistPath)
}

func TestGlobalCache_SetAndGet(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 15*time.Minute, "")

	// Test setting and getting a value
	err := cache.Set("key1", "value1", 1*time.Hour)
	assert.NoError(t, err)

	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Test non-existent key
	_, exists = cache.Get("nonexistent")
	assert.False(t, exists)
}

func TestGlobalCache_Expiration(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 15*time.Minute, "")

	// Set with short TTL
	err := cache.Set("expire", "value", 100*time.Millisecond)
	assert.NoError(t, err)

	// Should exist immediately
	value, exists := cache.Get("expire")
	assert.True(t, exists)
	assert.Equal(t, "value", value)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, exists = cache.Get("expire")
	assert.False(t, exists)
}

func TestGlobalCache_Delete(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 15*time.Minute, "")

	err := cache.Set("delete-me", "value", 1*time.Hour)
	assert.NoError(t, err)

	// Verify it exists
	_, exists := cache.Get("delete-me")
	assert.True(t, exists)

	// Delete it
	cache.Delete("delete-me")

	// Verify it's gone
	_, exists = cache.Get("delete-me")
	assert.False(t, exists)
}

func TestGlobalCache_Clear(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 15*time.Minute, "")

	// Add multiple items
	cache.Set("key1", "value1", 1*time.Hour)
	cache.Set("key2", "value2", 1*time.Hour)
	cache.Set("key3", "value3", 1*time.Hour)

	// Clear all
	cache.Clear()

	// Verify all are gone
	_, exists1 := cache.Get("key1")
	_, exists2 := cache.Get("key2")
	_, exists3 := cache.Get("key3")

	assert.False(t, exists1)
	assert.False(t, exists2)
	assert.False(t, exists3)
}

func TestGlobalCache_Stats(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 15*time.Minute, "")

	cache.Set("key1", "value1", 1*time.Hour)
	cache.Set("key2", "value2", 1*time.Hour)
	cache.Set("key3", "value3", 1*time.Hour)

	// Get one to increase hits
	cache.Get("key1")
	cache.Get("nonexistent") // Miss

	stats := cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, int64(3), stats.Sets)
	assert.Equal(t, 3, stats.ItemCount)
}

func TestGlobalCache_MaxSize(t *testing.T) {
	// Small cache for testing eviction
	cache := NewGlobalCache(100, 15*time.Minute, "")

	// Add a large item
	largeData := make([]byte, 60)
	err := cache.Set("large1", largeData, 1*time.Hour)
	assert.NoError(t, err)

	// Try to add another large item
	err = cache.Set("large2", largeData, 1*time.Hour)
	// Should evict the first item or refuse if over limit

	stats := cache.GetStats()
	assert.LessOrEqual(t, stats.TotalSize, int64(100))
}

func TestGlobalCache_SetDefault(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 10*time.Minute, "")

	// Set with default TTL
	err := cache.SetDefault("key", "value")
	assert.NoError(t, err)

	value, exists := cache.Get("key")
	assert.True(t, exists)
	assert.Equal(t, "value", value)
}

func TestGlobalCache_Persistence(t *testing.T) {
	tempFile := "/tmp/test_cache.json"
	cache := NewGlobalCache(1024*1024, 15*time.Minute, tempFile)

	// Add some data
	cache.Set("persist1", "value1", 1*time.Hour)
	cache.Set("persist2", "value2", 1*time.Hour)

	// Save to disk
	err := cache.SaveToDisk()
	assert.NoError(t, err)

	// Create new cache and load
	newCache := NewGlobalCache(1024*1024, 15*time.Minute, tempFile)
	err = newCache.LoadFromDisk()
	assert.NoError(t, err)

	// Verify data loaded
	value1, exists1 := newCache.Get("persist1")
	value2, exists2 := newCache.Get("persist2")

	assert.True(t, exists1)
	assert.Equal(t, "value1", value1)
	assert.True(t, exists2)
	assert.Equal(t, "value2", value2)
}

func TestGlobalCache_ConcurrentAccess(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 15*time.Minute, "")
	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", n)
			value := fmt.Sprintf("value%d", n)
			cache.Set(key, value, 1*time.Hour)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", n)
			cache.Get(key)
		}(i)
	}

	wg.Wait()

	// Verify some entries exist
	stats := cache.GetStats()
	assert.True(t, stats.ItemCount > 0)
}

func TestGlobalCache_CleanupExpired(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 15*time.Minute, "")

	// Add items with different TTLs
	cache.Set("expire1", "value1", 100*time.Millisecond)
	cache.Set("expire2", "value2", 100*time.Millisecond)
	cache.Set("keep", "value3", 1*time.Hour)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Access to trigger cleanup
	cache.Get("expire1")

	// Check expired items are gone
	_, exists1 := cache.Get("expire1")
	_, exists2 := cache.Get("expire2")
	assert.False(t, exists1)
	assert.False(t, exists2)

	// Check non-expired item remains
	value, exists := cache.Get("keep")
	assert.True(t, exists)
	assert.Equal(t, "value3", value)
}

func TestCacheEntry(t *testing.T) {
	entry := &CacheEntry{
		Key:        "test-key",
		Value:      "test-value",
		Expiration: time.Now().Add(1 * time.Hour),
		Created:    time.Now(),
		LastAccess: time.Now(),
		HitCount:   5,
		Size:       100,
	}

	assert.Equal(t, "test-key", entry.Key)
	assert.Equal(t, "test-value", entry.Value)
	assert.Equal(t, int64(5), entry.HitCount)
	assert.Equal(t, int64(100), entry.Size)
	assert.True(t, entry.Expiration.After(time.Now()))
}

func TestCacheMetrics(t *testing.T) {
	metrics := &CacheMetrics{
		Hits:      10,
		Misses:    5,
		Sets:      15,
		Deletes:   2,
		Evictions: 1,
		TotalSize: 1024,
		ItemCount: 8,
	}

	assert.Equal(t, int64(10), metrics.Hits)
	assert.Equal(t, int64(5), metrics.Misses)
	assert.Equal(t, int64(15), metrics.Sets)
	assert.Equal(t, int64(2), metrics.Deletes)
	assert.Equal(t, int64(1), metrics.Evictions)
	assert.Equal(t, int64(1024), metrics.TotalSize)
	assert.Equal(t, 8, metrics.ItemCount)

	// Test hit ratio
	hitRatio := float64(metrics.Hits) / float64(metrics.Hits+metrics.Misses)
	assert.InDelta(t, 0.667, hitRatio, 0.001)
}

func BenchmarkGlobalCache_Set(b *testing.B) {
	cache := NewGlobalCache(1024*1024*10, 15*time.Minute, "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i)
		cache.Set(key, i, 1*time.Hour)
	}
}

func BenchmarkGlobalCache_Get(b *testing.B) {
	cache := NewGlobalCache(1024*1024*10, 15*time.Minute, "")

	// Pre-populate
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		cache.Set(key, i, 1*time.Hour)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%1000)
		cache.Get(key)
	}
}

func BenchmarkGlobalCache_ConcurrentAccess(b *testing.B) {
	cache := NewGlobalCache(1024*1024*10, 15*time.Minute, "")

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i%1000)
			if i%2 == 0 {
				cache.Set(key, i, 1*time.Hour)
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}