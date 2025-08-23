package discovery

import (
	"sync"
	"time"
)

// Cache provides caching for discovery results
type Cache struct {
	items       map[string]*cacheItem
	ttl         time.Duration
	mu          sync.RWMutex
	stopCleaner chan struct{}
}

// cacheItem represents a cached item
type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewCache creates a new cache with the specified TTL
func NewCache(ttl time.Duration) *Cache {
	c := &Cache{
		items:       make(map[string]*cacheItem),
		ttl:         ttl,
		stopCleaner: make(chan struct{}),
	}

	// Start cleanup goroutine
	go c.cleaner()

	return c
}

// Set adds or updates an item in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &cacheItem{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check if item has expired
	if time.Now().After(item.expiration) {
		return nil, false
	}

	return item.value, true
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
}

// Size returns the number of items in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// cleaner periodically removes expired items
func (c *Cache) cleaner() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stopCleaner:
			return
		}
	}
}

// removeExpired removes all expired items from the cache
func (c *Cache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiration) {
			delete(c.items, key)
		}
	}
}

// Stop stops the cache cleaner
func (c *Cache) Stop() {
	close(c.stopCleaner)
}

// SetTTL updates the TTL for new items
func (c *Cache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ttl = ttl
}

// GetTTL returns the current TTL
func (c *Cache) GetTTL() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.ttl
}

// Has checks if a key exists in the cache
func (c *Cache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return false
	}

	return time.Now().Before(item.expiration)
}

// Keys returns all non-expired keys in the cache
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.items))
	now := time.Now()

	for key, item := range c.items {
		if now.Before(item.expiration) {
			keys = append(keys, key)
		}
	}

	return keys
}

// GetWithExpiry returns an item and its expiration time
func (c *Cache) GetWithExpiry(key string) (interface{}, time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, time.Time{}, false
	}

	if time.Now().After(item.expiration) {
		return nil, time.Time{}, false
	}

	return item.value, item.expiration, true
}

// Refresh resets the expiration time for an item
func (c *Cache) Refresh(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		return false
	}

	item.expiration = time.Now().Add(c.ttl)
	return true
}
