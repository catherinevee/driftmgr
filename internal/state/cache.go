package state

import (
	"sync"
	"time"
)

// StateCache provides caching for parsed Terraform states
type StateCache struct {
	mu      sync.RWMutex
	items   map[string]*CacheItem
	ttl     time.Duration
	maxSize int
}

// CacheItem represents a cached state
type CacheItem struct {
	State     *TerraformState
	ExpiresAt time.Time
	AccessedAt time.Time
	Size      int64
}

// NewStateCache creates a new state cache
func NewStateCache(ttl time.Duration) *StateCache {
	cache := &StateCache{
		items:   make(map[string]*CacheItem),
		ttl:     ttl,
		maxSize: 100, // Maximum 100 states in cache
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Get retrieves a state from the cache
func (c *StateCache) Get(key string) *TerraformState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(item.ExpiresAt) {
		return nil
	}

	// Update access time
	item.AccessedAt = time.Now()

	return item.State
}

// Set stores a state in the cache
func (c *StateCache) Set(key string, state *TerraformState) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check cache size limit
	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = &CacheItem{
		State:      state,
		ExpiresAt:  time.Now().Add(c.ttl),
		AccessedAt: time.Now(),
		Size:       c.calculateSize(state),
	}
}

// Delete removes a state from the cache
func (c *StateCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *StateCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheItem)
}

// Size returns the number of items in the cache
func (c *StateCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}

// GetStats returns cache statistics
func (c *StateCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		ItemCount:  len(c.items),
		TotalSize:  0,
		OldestItem: time.Now(),
		NewestItem: time.Time{},
	}

	for _, item := range c.items {
		stats.TotalSize += item.Size
		
		if item.AccessedAt.Before(stats.OldestItem) {
			stats.OldestItem = item.AccessedAt
		}
		
		if item.AccessedAt.After(stats.NewestItem) {
			stats.NewestItem = item.AccessedAt
		}
	}

	return stats
}

// CacheStats contains cache statistics
type CacheStats struct {
	ItemCount  int       `json:"item_count"`
	TotalSize  int64     `json:"total_size"`
	OldestItem time.Time `json:"oldest_item"`
	NewestItem time.Time `json:"newest_item"`
}

// cleanupLoop runs periodic cleanup of expired items
func (c *StateCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired items
func (c *StateCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
		}
	}
}

// evictOldest removes the least recently accessed item
func (c *StateCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.items {
		if oldestKey == "" || item.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.AccessedAt
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// calculateSize estimates the size of a state in bytes
func (c *StateCache) calculateSize(state *TerraformState) int64 {
	// Simple estimation based on resource count
	// In production, you might want to use actual serialization size
	baseSize := int64(1024) // Base overhead
	resourceSize := int64(512) // Estimated size per resource
	
	resourceCount := int64(len(state.Resources))
	for _, resource := range state.Resources {
		resourceCount += int64(len(resource.Instances))
	}
	
	return baseSize + (resourceCount * resourceSize)
}

// SetTTL updates the TTL for the cache
func (c *StateCache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.ttl = ttl
}

// SetMaxSize updates the maximum cache size
func (c *StateCache) SetMaxSize(size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.maxSize = size
	
	// Evict items if necessary
	for len(c.items) > c.maxSize {
		c.evictOldest()
	}
}

// Has checks if a key exists in the cache
func (c *StateCache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	item, exists := c.items[key]
	if !exists {
		return false
	}
	
	// Check if expired
	return !time.Now().After(item.ExpiresAt)
}

// GetKeys returns all cache keys
func (c *StateCache) GetKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	keys := make([]string, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}
	
	return keys
}

// Touch updates the access time of a cached item
func (c *StateCache) Touch(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if item, exists := c.items[key]; exists {
		item.AccessedAt = time.Now()
		item.ExpiresAt = time.Now().Add(c.ttl)
	}
}