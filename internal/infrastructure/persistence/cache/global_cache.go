package cache

import (
	"sync"
	"time"
)

// GlobalCache provides a shared cache instance for the entire application
type GlobalCache struct {
	entries map[string]*CacheItem
	mu      sync.RWMutex
}

// CacheItem represents a cached item with metadata
type CacheItem struct {
	Value     interface{}
	CreatedAt time.Time
	TTL       time.Duration
}

var (
	globalCache *GlobalCache
	once        sync.Once
)

// GetGlobalCache returns the singleton global cache instance
func GetGlobalCache() *GlobalCache {
	once.Do(func() {
		globalCache = &GlobalCache{
			entries: make(map[string]*CacheItem),
		}
		// Start cleanup goroutine
		go globalCache.cleanupExpired()
	})
	return globalCache
}

// Set stores a value in the cache with a TTL
func (c *GlobalCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.entries[key] = &CacheItem{
		Value:     value,
		CreatedAt: time.Now(),
		TTL:       ttl,
	}
}

// Get retrieves a value from the cache
func (c *GlobalCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, exists := c.entries[key]
	if !exists {
		return nil, false
	}
	
	// Check if expired
	if time.Since(item.CreatedAt) > item.TTL {
		return nil, false
	}
	
	return item.Value, true
}

// GetWithAge retrieves a value from the cache along with its age
func (c *GlobalCache) GetWithAge(key string) (interface{}, bool, time.Duration) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, exists := c.entries[key]
	if !exists {
		return nil, false, 0
	}
	
	age := time.Since(item.CreatedAt)
	
	// Check if expired
	if age > item.TTL {
		return nil, false, 0
	}
	
	return item.Value, true, age
}

// Delete removes a value from the cache
func (c *GlobalCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	delete(c.entries, key)
	return nil
}

// Clear removes all entries from the cache
func (c *GlobalCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.entries = make(map[string]*CacheItem)
	return nil
}

// InvalidatePattern removes all cache entries matching a pattern
func (c *GlobalCache) InvalidatePattern(pattern string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Simple prefix matching for now
	for key := range c.entries {
		if len(key) >= len(pattern) && key[:len(pattern)] == pattern {
			delete(c.entries, key)
		}
	}
	return nil
}

// GetStats returns cache statistics
func (c *GlobalCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	total := len(c.entries)
	expired := 0
	
	for _, item := range c.entries {
		if time.Since(item.CreatedAt) > item.TTL {
			expired++
		}
	}
	
	return map[string]interface{}{
		"total_entries": total,
		"active_entries": total - expired,
		"expired_entries": expired,
	}
}

// cleanupExpired periodically removes expired entries
func (c *GlobalCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		for key, item := range c.entries {
			if time.Since(item.CreatedAt) > item.TTL {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}

// SetIfNotExists sets a value only if the key doesn't exist
func (c *GlobalCache) SetIfNotExists(key string, value interface{}, ttl time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.entries[key]; exists {
		// Check if expired
		if time.Since(c.entries[key].CreatedAt) <= c.entries[key].TTL {
			return false
		}
	}
	
	c.entries[key] = &CacheItem{
		Value:     value,
		CreatedAt: time.Now(),
		TTL:       ttl,
	}
	return true
}

// Touch updates the timestamp of a cache entry to extend its TTL
func (c *GlobalCache) Touch(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	item, exists := c.entries[key]
	if !exists {
		return false
	}
	
	item.CreatedAt = time.Now()
	return true
}

// Keys returns all cache keys matching a pattern
func (c *GlobalCache) Keys(pattern string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var keys []string
	for key := range c.entries {
		if pattern == "*" || pattern == "" {
			keys = append(keys, key)
		} else if matchPattern(key, pattern) {
			keys = append(keys, key)
		}
	}

	return keys
}

// matchPattern checks if a key matches a simple glob pattern
func matchPattern(key, pattern string) bool {
	// Simple pattern matching - supports * wildcard
	if pattern == "*" {
		return true
	}
	
	// Check for prefix match with *
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}
	
	// Check for suffix match with *
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(key) >= len(suffix) && key[len(key)-len(suffix):] == suffix
	}
	
	// Check for contains with *pattern*
	if len(pattern) > 1 && pattern[0] == '*' && pattern[len(pattern)-1] == '*' {
		contains := pattern[1:len(pattern)-1]
		for i := 0; i <= len(key)-len(contains); i++ {
			if key[i:i+len(contains)] == contains {
				return true
			}
		}
		return false
	}
	
	// Exact match
	return key == pattern
}