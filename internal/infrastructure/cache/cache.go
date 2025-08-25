package cache

import (
	"sync"
	"time"
)

// CacheManager provides caching functionality
type CacheManager struct {
	cache   map[string]*CacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
	maxSize int
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
}

var globalManager *CacheManager
var once sync.Once

// GetGlobalManager returns the global cache manager instance
func GetGlobalManager() *CacheManager {
	once.Do(func() {
		globalManager = NewCacheManager(15*time.Minute, 1000)
	})
	return globalManager
}

// NewCacheManager creates a new cache manager
func NewCacheManager(ttl time.Duration, maxSize int) *CacheManager {
	return &CacheManager{
		cache:   make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get retrieves a value from cache
func (c *CacheManager) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry.Value, true
}

// Set stores a value in cache
func (c *CacheManager) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value in cache with custom TTL
func (c *CacheManager) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict old entries if cache is full
	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}

	c.cache[key] = &CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a value from cache
func (c *CacheManager) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
}

// Clear removes all values from cache
func (c *CacheManager) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*CacheEntry)
	return nil
}

// GetStats returns cache statistics
func (c *CacheManager) GetStats() *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	validEntries := 0
	expiredEntries := 0
	now := time.Now()

	for _, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			expiredEntries++
		} else {
			validEntries++
		}
	}

	return &CacheStats{
		TotalEntries:   len(c.cache),
		ValidEntries:   validEntries,
		ExpiredEntries: expiredEntries,
		MaxSize:        c.maxSize,
		TTL:            c.ttl,
	}
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalEntries   int           `json:"total_entries"`
	ValidEntries   int           `json:"valid_entries"`
	ExpiredEntries int           `json:"expired_entries"`
	MaxSize        int           `json:"max_size"`
	TTL            time.Duration `json:"ttl"`
}

// evictOldest removes the oldest entry from cache
func (c *CacheManager) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.cache {
		if oldestKey == "" || entry.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

// Size returns the current cache size
func (c *CacheManager) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// CleanExpired removes expired entries
func (c *CacheManager) CleanExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			delete(c.cache, key)
		}
	}
}

// DiscoveryCache provides caching for discovery operations
type DiscoveryCache struct {
	manager *CacheManager
}

// NewDiscoveryCache creates a new discovery cache
func NewDiscoveryCache() *DiscoveryCache {
	return &DiscoveryCache{
		manager: NewCacheManager(30*time.Minute, 500),
	}
}

// GetResources retrieves cached resources
func (dc *DiscoveryCache) GetResources(key string) (interface{}, bool) {
	return dc.manager.Get(key)
}

// Get retrieves a value from cache
func (dc *DiscoveryCache) Get(key string) (interface{}, bool) {
	return dc.manager.Get(key)
}

// SetResources stores resources in cache
func (dc *DiscoveryCache) SetResources(key string, resources interface{}) {
	dc.manager.Set(key, resources)
}

// Set stores a value in cache
func (dc *DiscoveryCache) Set(key string, value interface{}) {
	dc.manager.Set(key, value)
}

// Clear clears the discovery cache
func (dc *DiscoveryCache) Clear() {
	dc.manager.Clear()
}
