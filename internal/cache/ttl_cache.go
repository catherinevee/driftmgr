package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/logging"
)

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Value      interface{}
	ExpiresAt  time.Time
	CreatedAt  time.Time
	AccessedAt time.Time
	HitCount   int64
}

// TTLCache provides thread-safe caching with TTL support
type TTLCache struct {
	items      map[string]*CacheEntry
	mu         sync.RWMutex
	defaultTTL time.Duration
	maxSize    int
	stats      *CacheStats
	logger     *logging.Logger
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits      int64
	Misses    int64
	Evictions int64
	Expired   int64
	TotalSets int64
	TotalGets int64
	mu        sync.RWMutex
}

// NewTTLCache creates a new TTL cache
func NewTTLCache(defaultTTL time.Duration, maxSize int) *TTLCache {
	cache := &TTLCache{
		items:      make(map[string]*CacheEntry),
		defaultTTL: defaultTTL,
		maxSize:    maxSize,
		stats:      &CacheStats{},
		logger:     logging.GetLogger(),
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Set adds or updates an item in the cache
func (c *TTLCache) Set(key string, value interface{}, ttl ...time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiration := c.defaultTTL
	if len(ttl) > 0 {
		expiration = ttl[0]
	}

	// Check if we need to evict items
	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = &CacheEntry{
		Value:      value,
		ExpiresAt:  time.Now().Add(expiration),
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		HitCount:   0,
	}

	c.stats.mu.Lock()
	c.stats.TotalSets++
	c.stats.mu.Unlock()

	c.logger.Debug("Cache set", map[string]interface{}{
		"key": key,
		"ttl": expiration.String(),
	})
}

// Get retrieves an item from the cache
func (c *TTLCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	entry, exists := c.items[key]
	c.mu.RUnlock()

	c.stats.mu.Lock()
	c.stats.TotalGets++
	c.stats.mu.Unlock()

	if !exists {
		c.stats.mu.Lock()
		c.stats.Misses++
		c.stats.mu.Unlock()
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()

		c.stats.mu.Lock()
		c.stats.Expired++
		c.stats.Misses++
		c.stats.mu.Unlock()

		return nil, false
	}

	// Update access time and hit count
	c.mu.Lock()
	entry.AccessedAt = time.Now()
	entry.HitCount++
	c.mu.Unlock()

	c.stats.mu.Lock()
	c.stats.Hits++
	c.stats.mu.Unlock()

	return entry.Value, true
}

// GetWithLoader retrieves from cache or loads if missing
func (c *TTLCache) GetWithLoader(key string, loader func() (interface{}, error), ttl ...time.Duration) (interface{}, error) {
	// Try to get from cache first
	if val, ok := c.Get(key); ok {
		return val, nil
	}

	// Load the value
	val, err := loader()
	if err != nil {
		return nil, err
	}

	// Cache the loaded value
	c.Set(key, val, ttl...)

	return val, nil
}

// Delete removes an item from the cache
func (c *TTLCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *TTLCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheEntry)

	c.logger.Info("Cache cleared")
}

// Size returns the number of items in the cache
func (c *TTLCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// GetStats returns cache statistics
func (c *TTLCache) GetStats() CacheStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	return *c.stats
}

// evictOldest removes the least recently accessed item
func (c *TTLCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.items {
		if oldestKey == "" || entry.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.AccessedAt
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)

		c.stats.mu.Lock()
		c.stats.Evictions++
		c.stats.mu.Unlock()

		c.logger.Debug("Cache eviction", map[string]interface{}{
			"key": oldestKey,
		})
	}
}

// cleanupExpired periodically removes expired items
func (c *TTLCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		expiredKeys := []string{}

		for key, entry := range c.items {
			if now.After(entry.ExpiresAt) {
				expiredKeys = append(expiredKeys, key)
			}
		}

		for _, key := range expiredKeys {
			delete(c.items, key)

			c.stats.mu.Lock()
			c.stats.Expired++
			c.stats.mu.Unlock()
		}

		c.mu.Unlock()

		if len(expiredKeys) > 0 {
			c.logger.Debug("Expired cache entries cleaned", map[string]interface{}{
				"count": len(expiredKeys),
			})
		}
	}
}

// ProviderCache provides caching specific to cloud providers
type ProviderCache struct {
	resourceCache   *TTLCache
	credentialCache *TTLCache
	discoveryCache  *TTLCache
	stateCache      *TTLCache
	logger          *logging.Logger
}

// NewProviderCache creates a provider-specific cache
func NewProviderCache() *ProviderCache {
	return &ProviderCache{
		resourceCache:   NewTTLCache(5*time.Minute, 1000), // Resources cached for 5 minutes
		credentialCache: NewTTLCache(30*time.Minute, 100), // Credentials cached for 30 minutes
		discoveryCache:  NewTTLCache(2*time.Minute, 500),  // Discovery results cached for 2 minutes
		stateCache:      NewTTLCache(10*time.Minute, 200), // State cached for 10 minutes
		logger:          logging.GetLogger(),
	}
}

// GetResources retrieves cached resources for a provider
func (p *ProviderCache) GetResources(provider string, region string) ([]interface{}, bool) {
	key := fmt.Sprintf("resources:%s:%s", provider, region)
	if val, ok := p.resourceCache.Get(key); ok {
		if resources, ok := val.([]interface{}); ok {
			return resources, true
		}
	}
	return nil, false
}

// SetResources caches resources for a provider
func (p *ProviderCache) SetResources(provider string, region string, resources []interface{}) {
	key := fmt.Sprintf("resources:%s:%s", provider, region)
	p.resourceCache.Set(key, resources)

	p.logger.Info("Resources cached", map[string]interface{}{
		"provider": provider,
		"region":   region,
		"count":    len(resources),
	})
}

// GetCredentials retrieves cached credentials
func (p *ProviderCache) GetCredentials(provider string) (interface{}, bool) {
	key := fmt.Sprintf("credentials:%s", provider)
	return p.credentialCache.Get(key)
}

// SetCredentials caches credentials
func (p *ProviderCache) SetCredentials(provider string, credentials interface{}) {
	key := fmt.Sprintf("credentials:%s", provider)
	p.credentialCache.Set(key, credentials)
}

// GetDiscoveryResult retrieves cached discovery results
func (p *ProviderCache) GetDiscoveryResult(provider string, account string) (interface{}, bool) {
	key := fmt.Sprintf("discovery:%s:%s", provider, account)
	return p.discoveryCache.Get(key)
}

// SetDiscoveryResult caches discovery results
func (p *ProviderCache) SetDiscoveryResult(provider string, account string, result interface{}) {
	key := fmt.Sprintf("discovery:%s:%s", provider, account)
	p.discoveryCache.Set(key, result)
}

// InvalidateProvider clears all caches for a provider
func (p *ProviderCache) InvalidateProvider(provider string) {
	// Clear all caches related to this provider
	p.resourceCache.Clear()
	p.discoveryCache.Clear()

	p.logger.Info("Provider cache invalidated", map[string]interface{}{
		"provider": provider,
	})
}

// GetCacheStats returns aggregated cache statistics
func (p *ProviderCache) GetCacheStats() map[string]CacheStats {
	return map[string]CacheStats{
		"resources":   p.resourceCache.GetStats(),
		"credentials": p.credentialCache.GetStats(),
		"discovery":   p.discoveryCache.GetStats(),
		"state":       p.stateCache.GetStats(),
	}
}

// SerializeCache serializes cache stats to JSON
func (p *ProviderCache) SerializeStats() (string, error) {
	stats := p.GetCacheStats()
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Global provider cache instance
var globalProviderCache *ProviderCache
var cacheOnce sync.Once

// GetProviderCache returns the global provider cache instance
func GetProviderCache() *ProviderCache {
	cacheOnce.Do(func() {
		globalProviderCache = NewProviderCache()
	})
	return globalProviderCache
}
