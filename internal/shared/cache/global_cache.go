package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheEntry represents a single cache entry
type CacheEntry struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	Expiration time.Time   `json:"expiration"`
	Created    time.Time   `json:"created"`
	LastAccess time.Time   `json:"last_access"`
	HitCount   int64       `json:"hit_count"`
	Size       int64       `json:"size"`
}

// GlobalCache provides thread-safe caching with TTL and persistence
type GlobalCache struct {
	mu          sync.RWMutex
	items       map[string]*CacheEntry
	maxSize     int64
	currentSize int64
	defaultTTL  time.Duration
	persistPath string
	metrics     *CacheMetrics
	stopCleaner chan struct{}
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	mu        sync.RWMutex
	Hits      int64
	Misses    int64
	Sets      int64
	Deletes   int64
	Evictions int64
	TotalSize int64
	ItemCount int
}

// NewGlobalCache creates a new global cache instance
func NewGlobalCache(maxSize int64, defaultTTL time.Duration, persistPath string) *GlobalCache {
	gc := &GlobalCache{
		items:       make(map[string]*CacheEntry),
		maxSize:     maxSize,
		defaultTTL:  defaultTTL,
		persistPath: persistPath,
		metrics:     &CacheMetrics{},
		stopCleaner: make(chan struct{}),
	}

	// Load persisted cache if available
	if persistPath != "" {
		_ = gc.loadFromDisk()
	}

	// Start cleanup goroutine
	go gc.cleanupExpired()

	return gc
}

// Set stores a value in the cache with the default TTL
func (gc *GlobalCache) Set(key string, value interface{}, ttl ...time.Duration) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	expiration := gc.defaultTTL
	if len(ttl) > 0 {
		expiration = ttl[0]
	}

	// Calculate size
	size := gc.calculateSize(value)

	// Check if we need to evict items
	if gc.currentSize+size > gc.maxSize {
		gc.evictLRU(size)
	}

	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Expiration: time.Now().Add(expiration),
		Created:    time.Now(),
		LastAccess: time.Now(),
		HitCount:   0,
		Size:       size,
	}

	// Update current size
	if existing, exists := gc.items[key]; exists {
		gc.currentSize -= existing.Size
	}
	gc.currentSize += size

	gc.items[key] = entry

	// Update metrics
	gc.metrics.mu.Lock()
	gc.metrics.Sets++
	gc.metrics.TotalSize = gc.currentSize
	gc.metrics.ItemCount = len(gc.items)
	gc.metrics.mu.Unlock()

	// Persist to disk if enabled
	if gc.persistPath != "" {
		go func() { _ = gc.saveToDisk() }()
	}

	return nil
}

// Get retrieves a value from the cache
func (gc *GlobalCache) Get(key string) (interface{}, bool) {
	gc.mu.RLock()
	entry, exists := gc.items[key]
	gc.mu.RUnlock()

	if !exists {
		gc.metrics.mu.Lock()
		gc.metrics.Misses++
		gc.metrics.mu.Unlock()
		return nil, false
	}

	// Check expiration
	if time.Now().After(entry.Expiration) {
		gc.Delete(key)
		gc.metrics.mu.Lock()
		gc.metrics.Misses++
		gc.metrics.mu.Unlock()
		return nil, false
	}

	// Update access time and hit count
	gc.mu.Lock()
	entry.LastAccess = time.Now()
	entry.HitCount++
	gc.mu.Unlock()

	gc.metrics.mu.Lock()
	gc.metrics.Hits++
	gc.metrics.mu.Unlock()

	return entry.Value, true
}

// GetWithAge returns a value and its age
func (gc *GlobalCache) GetWithAge(key string) (interface{}, bool, time.Duration) {
	gc.mu.RLock()
	entry, exists := gc.items[key]
	gc.mu.RUnlock()

	if !exists {
		gc.metrics.mu.Lock()
		gc.metrics.Misses++
		gc.metrics.mu.Unlock()
		return nil, false, 0
	}

	// Check expiration
	if time.Now().After(entry.Expiration) {
		gc.Delete(key)
		gc.metrics.mu.Lock()
		gc.metrics.Misses++
		gc.metrics.mu.Unlock()
		return nil, false, 0
	}

	age := time.Since(entry.Created)

	// Update access time and hit count
	gc.mu.Lock()
	entry.LastAccess = time.Now()
	entry.HitCount++
	gc.mu.Unlock()

	gc.metrics.mu.Lock()
	gc.metrics.Hits++
	gc.metrics.mu.Unlock()

	return entry.Value, true, age
}

// Delete removes an item from the cache
func (gc *GlobalCache) Delete(key string) bool {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	entry, exists := gc.items[key]
	if !exists {
		return false
	}

	gc.currentSize -= entry.Size
	delete(gc.items, key)

	gc.metrics.mu.Lock()
	gc.metrics.Deletes++
	gc.metrics.TotalSize = gc.currentSize
	gc.metrics.ItemCount = len(gc.items)
	gc.metrics.mu.Unlock()

	// Persist to disk if enabled
	if gc.persistPath != "" {
		go func() { _ = gc.saveToDisk() }()
	}

	return true
}

// Clear removes all items from the cache
func (gc *GlobalCache) Clear() {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	gc.items = make(map[string]*CacheEntry)
	gc.currentSize = 0

	gc.metrics.mu.Lock()
	gc.metrics.TotalSize = 0
	gc.metrics.ItemCount = 0
	gc.metrics.mu.Unlock()

	// Clear persisted cache
	if gc.persistPath != "" {
		os.Remove(gc.persistPath)
	}
}

// GetMetrics returns cache metrics
func (gc *GlobalCache) GetMetrics() CacheMetrics {
	gc.metrics.mu.RLock()
	defer gc.metrics.mu.RUnlock()

	return CacheMetrics{
		Hits:      gc.metrics.Hits,
		Misses:    gc.metrics.Misses,
		Sets:      gc.metrics.Sets,
		Deletes:   gc.metrics.Deletes,
		Evictions: gc.metrics.Evictions,
		TotalSize: gc.metrics.TotalSize,
		ItemCount: gc.metrics.ItemCount,
	}
}

// HitRate returns the cache hit rate
func (gc *GlobalCache) HitRate() float64 {
	gc.metrics.mu.RLock()
	defer gc.metrics.mu.RUnlock()

	total := gc.metrics.Hits + gc.metrics.Misses
	if total == 0 {
		return 0
	}

	return float64(gc.metrics.Hits) / float64(total) * 100
}

// Close stops the cleanup goroutine and saves cache to disk
func (gc *GlobalCache) Close() {
	close(gc.stopCleaner)

	if gc.persistPath != "" {
		_ = gc.saveToDisk()
	}
}

// cleanupExpired removes expired entries periodically
func (gc *GlobalCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gc.removeExpired()
		case <-gc.stopCleaner:
			return
		}
	}
}

// removeExpired removes all expired entries
func (gc *GlobalCache) removeExpired() {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	now := time.Now()
	for key, entry := range gc.items {
		if now.After(entry.Expiration) {
			gc.currentSize -= entry.Size
			delete(gc.items, key)

			gc.metrics.mu.Lock()
			gc.metrics.Evictions++
			gc.metrics.mu.Unlock()
		}
	}

	gc.metrics.mu.Lock()
	gc.metrics.TotalSize = gc.currentSize
	gc.metrics.ItemCount = len(gc.items)
	gc.metrics.mu.Unlock()
}

// evictLRU removes least recently used items to make space
func (gc *GlobalCache) evictLRU(requiredSize int64) {
	// Find LRU items
	type lruItem struct {
		key        string
		lastAccess time.Time
		size       int64
	}

	items := make([]lruItem, 0, len(gc.items))
	for key, entry := range gc.items {
		items = append(items, lruItem{
			key:        key,
			lastAccess: entry.LastAccess,
			size:       entry.Size,
		})
	}

	// Sort by last access time (oldest first)
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].lastAccess.After(items[j].lastAccess) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// Evict until we have enough space
	freedSpace := int64(0)
	for _, item := range items {
		if gc.currentSize+requiredSize-freedSpace <= gc.maxSize {
			break
		}

		gc.currentSize -= item.size
		freedSpace += item.size
		delete(gc.items, item.key)

		gc.metrics.mu.Lock()
		gc.metrics.Evictions++
		gc.metrics.mu.Unlock()
	}
}

// calculateSize estimates the size of a value
func (gc *GlobalCache) calculateSize(value interface{}) int64 {
	// Simple size calculation - can be improved
	data, err := json.Marshal(value)
	if err != nil {
		return 1024 // Default size
	}
	return int64(len(data))
}

// saveToDisk persists cache to disk
func (gc *GlobalCache) saveToDisk() error {
	gc.mu.RLock()
	data, err := json.Marshal(gc.items)
	gc.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(gc.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Write to temp file first
	tempFile := gc.persistPath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	// Rename to final location
	if err := os.Rename(tempFile, gc.persistPath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	return nil
}

// loadFromDisk loads cache from disk
func (gc *GlobalCache) loadFromDisk() error {
	data, err := os.ReadFile(gc.persistPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file, that's okay
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var items map[string]*CacheEntry
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	gc.mu.Lock()
	defer gc.mu.Unlock()

	// Load only non-expired items
	now := time.Now()
	for key, entry := range items {
		if now.Before(entry.Expiration) {
			gc.items[key] = entry
			gc.currentSize += entry.Size
		}
	}

	gc.metrics.mu.Lock()
	gc.metrics.TotalSize = gc.currentSize
	gc.metrics.ItemCount = len(gc.items)
	gc.metrics.mu.Unlock()

	return nil
}
