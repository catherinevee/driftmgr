package discovery

import (
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// CacheEntry represents a cache entry
type CacheEntry struct {
	Results   *models.DiscoveryResults
	Timestamp time.Time
	TTL       time.Duration
}

// Cache represents a discovery cache
type Cache struct {
	entries map[string]*CacheEntry
	stats   *CacheStats
	mu      sync.RWMutex
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits    int64
	Misses  int64
	Sets    int64
	Deletes int64
}

// NewCache creates a new discovery cache
func NewCache() *Cache {
	return &Cache{
		entries: make(map[string]*CacheEntry),
		stats:   &CacheStats{},
	}
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	// Check if entry has expired
	if entry.TTL > 0 && time.Since(entry.Timestamp) > entry.TTL {
		c.stats.Misses++
		return nil, false
	}

	c.stats.Hits++
	return entry, true
}

// Set stores a value in the cache
func (c *Cache) Set(key string, entry *CacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Set default TTL if not specified
	if entry.TTL == 0 {
		entry.TTL = 30 * time.Minute
	}

	c.entries[key] = entry
	c.stats.Sets++
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; exists {
		delete(c.entries, key)
		c.stats.Deletes++
	}
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.stats = &CacheStats{}
}

// Size returns the number of entries in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// HitRate returns the cache hit rate
func (c *Cache) HitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0.0
	}
	return float64(c.stats.Hits) / float64(total)
}

// MissRate returns the cache miss rate
func (c *Cache) MissRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0.0
	}
	return float64(c.stats.Misses) / float64(total)
}

// GetStats returns cache statistics
func (c *Cache) GetStats() *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &CacheStats{
		Hits:    c.stats.Hits,
		Misses:  c.stats.Misses,
		Sets:    c.stats.Sets,
		Deletes: c.stats.Deletes,
	}
}

// CleanupExpired removes expired entries from the cache
func (c *Cache) CleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if entry.TTL > 0 && now.Sub(entry.Timestamp) > entry.TTL {
			delete(c.entries, key)
		}
	}
}

// StartCleanup starts a background cleanup routine
func (c *Cache) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			c.CleanupExpired()
		}
	}()
}
