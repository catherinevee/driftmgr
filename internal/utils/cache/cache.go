package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Cache provides a thread-safe caching mechanism with TTL
type Cache struct {
	mu       sync.RWMutex
	items    map[string]*Item
	ttl      time.Duration
	basePath string
}

// Item represents a cached item
type Item struct {
	Value      interface{}
	Expiration time.Time
}

// New creates a new cache with specified TTL
func New(ttl time.Duration) *Cache {
	homeDir, _ := os.UserHomeDir()
	cachePath := filepath.Join(homeDir, ".driftmgr", "cache")
	os.MkdirAll(cachePath, 0755)

	return &Cache{
		items:    make(map[string]*Item),
		ttl:      ttl,
		basePath: cachePath,
	}
}

// NewWithPath creates a cache with custom storage path
func NewWithPath(ttl time.Duration, path string) *Cache {
	os.MkdirAll(path, 0755)
	return &Cache{
		items:    make(map[string]*Item),
		ttl:      ttl,
		basePath: path,
	}
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &Item{
		Value:      value,
		Expiration: time.Now().Add(c.ttl),
	}

	// Also persist to disk for recovery
	c.persistToDisk(key, value)
}

// SetWithTTL stores a value with custom TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &Item{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}

	c.persistToDisk(key, value)
}

// Get retrieves a value from cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		// Try to load from disk
		if value, err := c.loadFromDisk(key); err == nil {
			return value, true
		}
		return nil, false
	}

	// Check if expired
	if time.Now().After(item.Expiration) {
		delete(c.items, key)
		c.removeFromDisk(key)
		return nil, false
	}

	return item.Value, true
}

// GetOrSet retrieves from cache or sets new value if not present
func (c *Cache) GetOrSet(key string, fn func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	if value, ok := c.Get(key); ok {
		return value, nil
	}

	// Generate new value
	value, err := fn()
	if err != nil {
		return nil, err
	}

	// Store in cache
	c.Set(key, value)
	return value, nil
}

// Delete removes an item from cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	c.removeFromDisk(key)
}

// Clear removes all items from cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*Item)

	// Clear disk cache
	files, _ := filepath.Glob(filepath.Join(c.basePath, "*.cache"))
	for _, file := range files {
		os.Remove(file)
	}
}

// CleanExpired removes expired items
func (c *Cache) CleanExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.Expiration) {
			delete(c.items, key)
			c.removeFromDisk(key)
		}
	}
}

// StartCleanupTask starts a background task to clean expired items
func (c *Cache) StartCleanupTask(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			c.CleanExpired()
		}
	}()
}

// persistToDisk saves cache item to disk
func (c *Cache) persistToDisk(key string, value interface{}) {
	filename := filepath.Join(c.basePath, fmt.Sprintf("%s.cache", sanitizeKey(key)))

	data := &diskItem{
		Value:      value,
		Expiration: time.Now().Add(c.ttl),
	}

	jsonData, err := json.Marshal(data)
	if err == nil {
		os.WriteFile(filename, jsonData, 0644)
	}
}

// loadFromDisk loads cache item from disk
func (c *Cache) loadFromDisk(key string) (interface{}, error) {
	filename := filepath.Join(c.basePath, fmt.Sprintf("%s.cache", sanitizeKey(key)))

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var item diskItem
	if err := json.Unmarshal(data, &item); err != nil {
		return nil, err
	}

	// Check if expired
	if time.Now().After(item.Expiration) {
		os.Remove(filename)
		return nil, fmt.Errorf("cache expired")
	}

	// Restore to memory cache
	c.mu.Lock()
	c.items[key] = &Item{
		Value:      item.Value,
		Expiration: item.Expiration,
	}
	c.mu.Unlock()

	return item.Value, nil
}

// removeFromDisk removes cache file from disk
func (c *Cache) removeFromDisk(key string) {
	filename := filepath.Join(c.basePath, fmt.Sprintf("%s.cache", sanitizeKey(key)))
	os.Remove(filename)
}

// diskItem represents a cache item stored on disk
type diskItem struct {
	Value      interface{} `json:"value"`
	Expiration time.Time   `json:"expiration"`
}

// sanitizeKey sanitizes cache key for filesystem
func sanitizeKey(key string) string {
	// Replace invalid filename characters
	replacer := map[rune]rune{
		'/':  '-',
		'\\': '-',
		':':  '-',
		'*':  '-',
		'?':  '-',
		'"':  '-',
		'<':  '-',
		'>':  '-',
		'|':  '-',
	}

	result := make([]rune, 0, len(key))
	for _, r := range key {
		if replacement, ok := replacer[r]; ok {
			result = append(result, replacement)
		} else {
			result = append(result, r)
		}
	}

	return string(result)
}

// DiscoveryCache provides specialized caching for discovery results
type DiscoveryCache struct {
	*Cache
}

// NewDiscoveryCache creates a cache optimized for discovery results
func NewDiscoveryCache() *DiscoveryCache {
	return &DiscoveryCache{
		Cache: New(15 * time.Minute), // 15 minute TTL for discovery results
	}
}

// GetDiscoveryKey generates a cache key for discovery operations
func GetDiscoveryKey(provider, region, resourceType string) string {
	return fmt.Sprintf("discovery:%s:%s:%s", provider, region, resourceType)
}

// GetStateKey generates a cache key for state operations
func GetStateKey(stateFile string) string {
	return fmt.Sprintf("state:%s", filepath.Base(stateFile))
}

// GetDriftKey generates a cache key for drift detection
func GetDriftKey(provider, environment string) string {
	return fmt.Sprintf("drift:%s:%s", provider, environment)
}
