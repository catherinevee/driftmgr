package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGlobalCache(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")

	assert.NotNil(t, cache)
	assert.NotNil(t, cache.items)
	assert.Equal(t, int64(1024*1024), cache.maxSize)
	assert.Equal(t, 1*time.Hour, cache.defaultTTL)
	assert.Empty(t, cache.persistPath)
	assert.NotNil(t, cache.metrics)
	assert.NotNil(t, cache.stopCleaner)

	cache.Close()
}

func TestNewGlobalCache_WithPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	persistPath := filepath.Join(tmpDir, "cache.json")

	cache := NewGlobalCache(1024*1024, 1*time.Hour, persistPath)
	defer cache.Close()

	assert.Equal(t, persistPath, cache.persistPath)
}

func TestGlobalCache_Set(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// Set a value
	err := cache.Set("key1", "value1")
	assert.NoError(t, err)

	// Verify value was stored
	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Verify metrics
	metrics := cache.GetMetrics()
	assert.Equal(t, int64(1), metrics.Sets)
	assert.Equal(t, 1, metrics.ItemCount)
}

func TestGlobalCache_Set_WithCustomTTL(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// Set a value with custom TTL
	err := cache.Set("key1", "value1", 100*time.Millisecond)
	assert.NoError(t, err)

	// Verify value exists
	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Verify value is expired
	value, exists = cache.Get("key1")
	assert.False(t, exists)
	assert.Nil(t, value)
}

func TestGlobalCache_Set_Overwrite(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// Set initial value
	err := cache.Set("key1", "value1")
	assert.NoError(t, err)

	// Overwrite with new value
	err = cache.Set("key1", "value2")
	assert.NoError(t, err)

	// Verify new value
	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value2", value)

	// Verify metrics (should still be 1 item)
	metrics := cache.GetMetrics()
	assert.Equal(t, int64(2), metrics.Sets) // Two Set operations
	assert.Equal(t, 1, metrics.ItemCount)   // But only 1 item
}

func TestGlobalCache_Get(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// Test non-existent key
	value, exists := cache.Get("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, value)

	// Set a value
	err := cache.Set("key1", "value1")
	assert.NoError(t, err)

	// Test existing key
	value, exists = cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Verify metrics
	metrics := cache.GetMetrics()
	assert.Equal(t, int64(1), metrics.Hits)
	assert.Equal(t, int64(1), metrics.Misses)
}

func TestGlobalCache_GetWithAge(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// Set a value
	err := cache.Set("key1", "value1")
	assert.NoError(t, err)

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Get with age
	value, exists, age := cache.GetWithAge("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)
	assert.Greater(t, age, time.Duration(0))
	assert.Less(t, age, 100*time.Millisecond)
}

func TestGlobalCache_GetWithAge_NonExistent(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	value, exists, age := cache.GetWithAge("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, value)
	assert.Equal(t, time.Duration(0), age)
}

func TestGlobalCache_Delete(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// Set a value
	err := cache.Set("key1", "value1")
	assert.NoError(t, err)

	// Delete the value
	deleted := cache.Delete("key1")
	assert.True(t, deleted)

	// Verify value is gone
	value, exists := cache.Get("key1")
	assert.False(t, exists)
	assert.Nil(t, value)

	// Verify metrics
	metrics := cache.GetMetrics()
	assert.Equal(t, int64(1), metrics.Deletes)
	assert.Equal(t, 0, metrics.ItemCount)
}

func TestGlobalCache_Delete_NonExistent(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	deleted := cache.Delete("nonexistent")
	assert.False(t, deleted)
}

func TestGlobalCache_Clear(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// Set some values
	err := cache.Set("key1", "value1")
	assert.NoError(t, err)
	err = cache.Set("key2", "value2")
	assert.NoError(t, err)

	// Verify values exist
	value1, exists1 := cache.Get("key1")
	assert.True(t, exists1)
	assert.Equal(t, "value1", value1)

	value2, exists2 := cache.Get("key2")
	assert.True(t, exists2)
	assert.Equal(t, "value2", value2)

	// Clear cache
	cache.Clear()

	// Verify values are gone
	value1, exists1 = cache.Get("key1")
	assert.False(t, exists1)
	assert.Nil(t, value1)

	value2, exists2 = cache.Get("key2")
	assert.False(t, exists2)
	assert.Nil(t, value2)

	// Verify metrics
	metrics := cache.GetMetrics()
	assert.Equal(t, 0, metrics.ItemCount)
	assert.Equal(t, int64(0), metrics.TotalSize)
}

func TestGlobalCache_GetMetrics(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// Initial metrics
	metrics := cache.GetMetrics()
	assert.Equal(t, int64(0), metrics.Hits)
	assert.Equal(t, int64(0), metrics.Misses)
	assert.Equal(t, int64(0), metrics.Sets)
	assert.Equal(t, int64(0), metrics.Deletes)
	assert.Equal(t, int64(0), metrics.Evictions)
	assert.Equal(t, int64(0), metrics.TotalSize)
	assert.Equal(t, 0, metrics.ItemCount)

	// Perform operations
	cache.Set("key1", "value1")
	cache.Get("key1")
	cache.Get("nonexistent")
	cache.Delete("key1")

	// Check updated metrics
	metrics = cache.GetMetrics()
	assert.Equal(t, int64(1), metrics.Hits)
	assert.Equal(t, int64(1), metrics.Misses)
	assert.Equal(t, int64(1), metrics.Sets)
	assert.Equal(t, int64(1), metrics.Deletes)
	assert.Equal(t, int64(0), metrics.Evictions)
	assert.Equal(t, int64(0), metrics.TotalSize)
	assert.Equal(t, 0, metrics.ItemCount)
}

func TestGlobalCache_HitRate(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// No operations yet
	hitRate := cache.HitRate()
	assert.Equal(t, 0.0, hitRate)

	// Set and get a value
	cache.Set("key1", "value1")
	cache.Get("key1")
	cache.Get("nonexistent")

	// 1 hit, 1 miss = 50% hit rate
	hitRate = cache.HitRate()
	assert.Equal(t, 50.0, hitRate)

	// Get the same value again
	cache.Get("key1")

	// 2 hits, 1 miss = 66.67% hit rate
	hitRate = cache.HitRate()
	assert.Equal(t, 66.66666666666666, hitRate)
}

func TestGlobalCache_Eviction(t *testing.T) {
	// Create cache with small size
	cache := NewGlobalCache(50, 1*time.Hour, "")
	defer cache.Close()

	// Set a value that fits
	err := cache.Set("key1", "small")
	assert.NoError(t, err)

	// Set a value that requires eviction
	err = cache.Set("key2", "this is a much larger value that should cause eviction")
	assert.NoError(t, err)

	// First value should be evicted
	value1, exists1 := cache.Get("key1")
	assert.False(t, exists1)
	assert.Nil(t, value1)

	// Second value should exist
	value2, exists2 := cache.Get("key2")
	assert.True(t, exists2)
	assert.Equal(t, "this is a much larger value that should cause eviction", value2)

	// Verify eviction metrics
	metrics := cache.GetMetrics()
	assert.Greater(t, metrics.Evictions, int64(0))
}

func TestGlobalCache_Expiration(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 100*time.Millisecond, "")
	defer cache.Close()

	// Set a value
	err := cache.Set("key1", "value1")
	assert.NoError(t, err)

	// Verify value exists
	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Verify value is expired
	value, exists = cache.Get("key1")
	assert.False(t, exists)
	assert.Nil(t, value)
}

func TestGlobalCache_ConcurrentAccess(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer func() { done <- true }()
			key := fmt.Sprintf("key%d", i)
			value := fmt.Sprintf("value%d", i)
			cache.Set(key, value)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all values were stored
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		expectedValue := fmt.Sprintf("value%d", i)
		value, exists := cache.Get(key)
		assert.True(t, exists)
		assert.Equal(t, expectedValue, value)
	}
}

func TestGlobalCache_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	persistPath := filepath.Join(tmpDir, "cache.json")

	// Create cache with persistence
	cache1 := NewGlobalCache(1024*1024, 1*time.Hour, persistPath)

	// Set some values
	err := cache1.Set("key1", "value1")
	assert.NoError(t, err)
	err = cache1.Set("key2", "value2")
	assert.NoError(t, err)

	// Close cache
	cache1.Close()

	// Wait a bit for async save to complete
	time.Sleep(100 * time.Millisecond)

	// Create new cache with same persistence path
	cache2 := NewGlobalCache(1024*1024, 1*time.Hour, persistPath)
	defer cache2.Close()

	// Verify values were persisted
	value1, exists1 := cache2.Get("key1")
	assert.True(t, exists1)
	assert.Equal(t, "value1", value1)

	value2, exists2 := cache2.Get("key2")
	assert.True(t, exists2)
	assert.Equal(t, "value2", value2)
}

func TestGlobalCache_Persistence_ExpiredItems(t *testing.T) {
	tmpDir := t.TempDir()
	persistPath := filepath.Join(tmpDir, "cache.json")

	// Create cache with persistence
	cache1 := NewGlobalCache(1024*1024, 100*time.Millisecond, persistPath)

	// Set a value that will expire
	err := cache1.Set("key1", "value1")
	assert.NoError(t, err)

	// Close cache
	cache1.Close()

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Create new cache with same persistence path
	cache2 := NewGlobalCache(1024*1024, 1*time.Hour, persistPath)
	defer cache2.Close()

	// Verify expired value was not loaded
	value1, exists1 := cache2.Get("key1")
	assert.False(t, exists1)
	assert.Nil(t, value1)
}

func TestCacheEntry_Struct(t *testing.T) {
	now := time.Now()
	entry := &CacheEntry{
		Key:        "test-key",
		Value:      "test-value",
		Expiration: now.Add(1 * time.Hour),
		Created:    now,
		LastAccess: now,
		HitCount:   5,
		Size:       1024,
	}

	assert.Equal(t, "test-key", entry.Key)
	assert.Equal(t, "test-value", entry.Value)
	assert.Equal(t, now.Add(1*time.Hour), entry.Expiration)
	assert.Equal(t, now, entry.Created)
	assert.Equal(t, now, entry.LastAccess)
	assert.Equal(t, int64(5), entry.HitCount)
	assert.Equal(t, int64(1024), entry.Size)
}

func TestCacheMetrics_Struct(t *testing.T) {
	metrics := CacheMetrics{
		Hits:      10,
		Misses:    5,
		Sets:      15,
		Deletes:   3,
		Evictions: 2,
		TotalSize: 1024,
		ItemCount: 12,
	}

	assert.Equal(t, int64(10), metrics.Hits)
	assert.Equal(t, int64(5), metrics.Misses)
	assert.Equal(t, int64(15), metrics.Sets)
	assert.Equal(t, int64(3), metrics.Deletes)
	assert.Equal(t, int64(2), metrics.Evictions)
	assert.Equal(t, int64(1024), metrics.TotalSize)
	assert.Equal(t, 12, metrics.ItemCount)
}

func TestGlobalCache_CalculateSize(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// Test different value types
	tests := []struct {
		value       interface{}
		expectedMin int64
	}{
		{"small", 5},
		{"this is a longer string", 20},
		{map[string]interface{}{"key": "value"}, 15}, // JSON object
		{[]string{"item1", "item2", "item3"}, 25},    // JSON array
		{123, 3},  // JSON number
		{true, 4}, // JSON boolean
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.value), func(t *testing.T) {
			size := cache.calculateSize(tt.value)
			assert.GreaterOrEqual(t, size, tt.expectedMin)
		})
	}
}

func TestGlobalCache_CalculateSize_Error(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "")
	defer cache.Close()

	// Test with value that can't be marshaled to JSON
	// (This is hard to achieve in practice, but we can test the error path)
	size := cache.calculateSize(make(chan int))
	assert.Equal(t, int64(1024), size) // Should return default size
}

func TestGlobalCache_CleanupExpired(t *testing.T) {
	cache := NewGlobalCache(1024*1024, 100*time.Millisecond, "")
	defer cache.Close()

	// Set a value that will expire
	err := cache.Set("key1", "value1")
	assert.NoError(t, err)

	// Verify value exists
	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Manually trigger cleanup
	cache.removeExpired()

	// Verify value is gone
	value, exists = cache.Get("key1")
	assert.False(t, exists)
	assert.Nil(t, value)

	// Verify eviction metrics
	metrics := cache.GetMetrics()
	assert.Greater(t, metrics.Evictions, int64(0))
}

func TestGlobalCache_EvictLRU(t *testing.T) {
	// Create cache with very small size
	cache := NewGlobalCache(50, 1*time.Hour, "")
	defer cache.Close()

	// Set a value that fits
	cache.Set("key1", "small")

	// Set a large value that should evict key1
	cache.Set("key2", "this is a very large value that should cause eviction")

	// key1 should be evicted
	value1, exists1 := cache.Get("key1")
	assert.False(t, exists1)
	assert.Nil(t, value1)

	// key2 should exist
	value2, exists2 := cache.Get("key2")
	assert.True(t, exists2)
	assert.Equal(t, "this is a very large value that should cause eviction", value2)
}

func TestGlobalCache_SaveToDisk_Error(t *testing.T) {
	// Create cache with invalid path
	cache := NewGlobalCache(1024*1024, 1*time.Hour, "/invalid/path/cache.json")
	defer cache.Close()

	// Set a value
	err := cache.Set("key1", "value1")
	assert.NoError(t, err) // Set should not error even if persistence fails

	// The saveToDisk is called asynchronously, so we can't easily test the error
	// But we can verify the cache still works
	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)
}

func TestGlobalCache_LoadFromDisk_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	persistPath := filepath.Join(tmpDir, "nonexistent.json")

	cache := NewGlobalCache(1024*1024, 1*time.Hour, persistPath)
	defer cache.Close()

	// Should not error when file doesn't exist
	// Cache should be empty
	metrics := cache.GetMetrics()
	assert.Equal(t, 0, metrics.ItemCount)
}

func TestGlobalCache_LoadFromDisk_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	persistPath := filepath.Join(tmpDir, "cache.json")

	// Write invalid JSON
	err := os.WriteFile(persistPath, []byte("invalid json"), 0644)
	require.NoError(t, err)

	cache := NewGlobalCache(1024*1024, 1*time.Hour, persistPath)
	defer cache.Close()

	// Should not error when JSON is invalid
	// Cache should be empty
	metrics := cache.GetMetrics()
	assert.Equal(t, 0, metrics.ItemCount)
}
