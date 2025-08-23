package performance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DistributedCache provides multi-level caching with Redis support
type DistributedCache struct {
	mu               sync.RWMutex
	memoryCache      *MemoryCache
	distributedCache DistributedBackend
	config           *CacheConfig
	metrics          *CacheMetrics
	serializer       Serializer
	compression      CompressionHandler

	// Background operations
	ctx           context.Context
	cancel        context.CancelFunc
	cleanupTicker *time.Ticker
	statsTicker   *time.Ticker
}

// CacheConfig holds configuration for the distributed cache
type CacheConfig struct {
	// Memory cache settings
	MemoryMaxSize         int64
	MemoryTTL             time.Duration
	MemoryCleanupInterval time.Duration

	// Distributed cache settings
	DistributedTTL       time.Duration
	CompressionThreshold int

	// Behavior settings
	WriteThrough      bool
	WriteBack         bool
	ReadThrough       bool
	InvalidateOnWrite bool

	// Performance settings
	AsyncWrite    bool
	BatchSize     int
	FlushInterval time.Duration

	// Networking
	ConnectionTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	MaxConnections    int
}

// CacheMetrics holds Prometheus metrics for cache operations
type CacheMetrics struct {
	hits                   prometheus.Counter
	misses                 prometheus.Counter
	memoryCacheHits        prometheus.Counter
	memoryCacheMisses      prometheus.Counter
	distributedCacheHits   prometheus.Counter
	distributedCacheMisses prometheus.Counter
	evictions              prometheus.Counter
	errors                 prometheus.Counter
	setOperations          prometheus.Counter
	deleteOperations       prometheus.Counter
	cacheSize              prometheus.Gauge
	memoryCacheSize        prometheus.Gauge
	operationDuration      prometheus.Histogram
	compressionRatio       prometheus.Histogram
	networkLatency         prometheus.Histogram
}

// MemoryCache implements an in-memory LRU cache
type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]*CacheItem
	lruList  *LRUList
	maxSize  int64
	currSize int64
	ttl      time.Duration
}

// CacheItem represents a cached item
type CacheItem struct {
	Key        string
	Value      interface{}
	Expires    time.Time
	Size       int64
	Hits       int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Compressed bool
	Checksum   string
	Metadata   map[string]interface{}

	// LRU linked list pointers
	prev *CacheItem
	next *CacheItem
}

// LRUList maintains the least recently used order
type LRUList struct {
	head *CacheItem
	tail *CacheItem
	size int
}

// DistributedBackend defines the interface for distributed cache backends
type DistributedBackend interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error
	Close() error
}

// Serializer handles serialization and deserialization
type Serializer interface {
	Serialize(interface{}) ([]byte, error)
	Deserialize([]byte, interface{}) error
}

// CompressionHandler handles data compression
type CompressionHandler interface {
	Compress([]byte) ([]byte, error)
	Decompress([]byte) ([]byte, error)
	ShouldCompress([]byte) bool
}

// CacheInvalidationStrategy defines cache invalidation strategies
type CacheInvalidationStrategy interface {
	ShouldInvalidate(key string, oldValue, newValue interface{}) bool
	GetInvalidationKeys(key string, value interface{}) []string
}

// NewDistributedCache creates a new distributed cache instance
func NewDistributedCache(backend DistributedBackend, config *CacheConfig) *DistributedCache {
	ctx, cancel := context.WithCancel(context.Background())

	metrics := &CacheMetrics{
		hits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_cache_hits_total",
			Help: "Total number of cache hits",
		}),
		misses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_cache_misses_total",
			Help: "Total number of cache misses",
		}),
		memoryCacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_memory_cache_hits_total",
			Help: "Total number of memory cache hits",
		}),
		memoryCacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_memory_cache_misses_total",
			Help: "Total number of memory cache misses",
		}),
		distributedCacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_distributed_cache_hits_total",
			Help: "Total number of distributed cache hits",
		}),
		distributedCacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_distributed_cache_misses_total",
			Help: "Total number of distributed cache misses",
		}),
		evictions: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_cache_evictions_total",
			Help: "Total number of cache evictions",
		}),
		errors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_cache_errors_total",
			Help: "Total number of cache errors",
		}),
		setOperations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_cache_set_operations_total",
			Help: "Total number of cache set operations",
		}),
		deleteOperations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_cache_delete_operations_total",
			Help: "Total number of cache delete operations",
		}),
		cacheSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "driftmgr_cache_size_bytes",
			Help: "Current cache size in bytes",
		}),
		memoryCacheSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "driftmgr_memory_cache_size_bytes",
			Help: "Current memory cache size in bytes",
		}),
		operationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_cache_operation_duration_seconds",
			Help:    "Cache operation duration",
			Buckets: prometheus.DefBuckets,
		}),
		compressionRatio: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_cache_compression_ratio",
			Help:    "Cache compression ratio",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		}),
		networkLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_cache_network_latency_seconds",
			Help:    "Network latency for distributed cache operations",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		}),
	}

	memCache := &MemoryCache{
		items:   make(map[string]*CacheItem),
		lruList: &LRUList{},
		maxSize: config.MemoryMaxSize,
		ttl:     config.MemoryTTL,
	}

	cache := &DistributedCache{
		memoryCache:      memCache,
		distributedCache: backend,
		config:           config,
		metrics:          metrics,
		serializer:       &JSONSerializer{},
		compression:      &GzipCompression{threshold: config.CompressionThreshold},
		ctx:              ctx,
		cancel:           cancel,
	}

	// Start background operations
	cache.cleanupTicker = time.NewTicker(config.MemoryCleanupInterval)
	cache.statsTicker = time.NewTicker(time.Minute)

	go cache.backgroundCleanup()
	go cache.updateStats()

	return cache
}

// Get retrieves a value from the cache
func (dc *DistributedCache) Get(ctx context.Context, key string) (interface{}, error) {
	start := time.Now()
	defer func() {
		dc.metrics.operationDuration.Observe(time.Since(start).Seconds())
	}()

	// Try memory cache first
	if value, found := dc.memoryCache.Get(key); found {
		dc.metrics.hits.Inc()
		dc.metrics.memoryCacheHits.Inc()
		return value, nil
	}
	dc.metrics.memoryCacheMisses.Inc()

	// Try distributed cache
	networkStart := time.Now()
	data, err := dc.distributedCache.Get(ctx, key)
	dc.metrics.networkLatency.Observe(time.Since(networkStart).Seconds())

	if err != nil {
		dc.metrics.misses.Inc()
		dc.metrics.distributedCacheMisses.Inc()
		return nil, err
	}

	if data == nil {
		dc.metrics.misses.Inc()
		dc.metrics.distributedCacheMisses.Inc()
		return nil, ErrCacheKeyNotFound
	}

	dc.metrics.hits.Inc()
	dc.metrics.distributedCacheHits.Inc()

	// Deserialize and decompress
	var value interface{}
	if err := dc.deserializeValue(data, &value); err != nil {
		dc.metrics.errors.Inc()
		return nil, fmt.Errorf("failed to deserialize cache value: %w", err)
	}

	// Store in memory cache for faster future access
	if dc.config.WriteThrough {
		dc.memoryCache.Set(key, value, dc.config.MemoryTTL)
	}

	return value, nil
}

// Set stores a value in the cache
func (dc *DistributedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		dc.metrics.operationDuration.Observe(time.Since(start).Seconds())
		dc.metrics.setOperations.Inc()
	}()

	// Serialize value
	data, err := dc.serializeValue(value)
	if err != nil {
		dc.metrics.errors.Inc()
		return fmt.Errorf("failed to serialize cache value: %w", err)
	}

	// Store in memory cache
	dc.memoryCache.Set(key, value, dc.config.MemoryTTL)

	// Store in distributed cache
	if dc.config.AsyncWrite {
		go func() {
			networkStart := time.Now()
			if err := dc.distributedCache.Set(context.Background(), key, data, ttl); err != nil {
				dc.metrics.errors.Inc()
				log.Printf("Failed to write to distributed cache: %v", err)
			}
			dc.metrics.networkLatency.Observe(time.Since(networkStart).Seconds())
		}()
	} else {
		networkStart := time.Now()
		err = dc.distributedCache.Set(ctx, key, data, ttl)
		dc.metrics.networkLatency.Observe(time.Since(networkStart).Seconds())
		if err != nil {
			dc.metrics.errors.Inc()
			return fmt.Errorf("failed to write to distributed cache: %w", err)
		}
	}

	return nil
}

// Delete removes a value from the cache
func (dc *DistributedCache) Delete(ctx context.Context, key string) error {
	start := time.Now()
	defer func() {
		dc.metrics.operationDuration.Observe(time.Since(start).Seconds())
		dc.metrics.deleteOperations.Inc()
	}()

	// Remove from memory cache
	dc.memoryCache.Delete(key)

	// Remove from distributed cache
	networkStart := time.Now()
	err := dc.distributedCache.Delete(ctx, key)
	dc.metrics.networkLatency.Observe(time.Since(networkStart).Seconds())

	if err != nil {
		dc.metrics.errors.Inc()
		return fmt.Errorf("failed to delete from distributed cache: %w", err)
	}

	return nil
}

// MGet retrieves multiple values from the cache
func (dc *DistributedCache) MGet(ctx context.Context, keys []string) (map[string]interface{}, error) {
	start := time.Now()
	defer func() {
		dc.metrics.operationDuration.Observe(time.Since(start).Seconds())
	}()

	result := make(map[string]interface{})
	missingKeys := make([]string, 0, len(keys))

	// Check memory cache first
	for _, key := range keys {
		if value, found := dc.memoryCache.Get(key); found {
			result[key] = value
			dc.metrics.memoryCacheHits.Inc()
		} else {
			missingKeys = append(missingKeys, key)
			dc.metrics.memoryCacheMisses.Inc()
		}
	}

	// Get missing keys from distributed cache
	if len(missingKeys) > 0 {
		networkStart := time.Now()
		distributedData, err := dc.distributedCache.MGet(ctx, missingKeys)
		dc.metrics.networkLatency.Observe(time.Since(networkStart).Seconds())

		if err != nil {
			dc.metrics.errors.Inc()
			return result, fmt.Errorf("failed to get from distributed cache: %w", err)
		}

		for key, data := range distributedData {
			if data != nil {
				var value interface{}
				if err := dc.deserializeValue(data, &value); err != nil {
					dc.metrics.errors.Inc()
					log.Printf("Failed to deserialize value for key %s: %v", key, err)
					continue
				}

				result[key] = value
				dc.metrics.distributedCacheHits.Inc()

				// Store in memory cache
				if dc.config.WriteThrough {
					dc.memoryCache.Set(key, value, dc.config.MemoryTTL)
				}
			} else {
				dc.metrics.distributedCacheMisses.Inc()
			}
		}
	}

	// Update hit/miss metrics
	hitCount := len(result)
	missCount := len(keys) - hitCount

	for i := 0; i < hitCount; i++ {
		dc.metrics.hits.Inc()
	}
	for i := 0; i < missCount; i++ {
		dc.metrics.misses.Inc()
	}

	return result, nil
}

// Invalidate removes cached data based on patterns or tags
func (dc *DistributedCache) Invalidate(ctx context.Context, pattern string) error {
	// This is a simplified implementation
	// In a real system, you'd need to implement pattern matching
	// and potentially use Redis SCAN or similar functionality

	return dc.Delete(ctx, pattern)
}

// Exists checks if a key exists in the cache
func (dc *DistributedCache) Exists(ctx context.Context, key string) (bool, error) {
	// Check memory cache first
	if dc.memoryCache.Exists(key) {
		return true, nil
	}

	// Check distributed cache
	networkStart := time.Now()
	exists, err := dc.distributedCache.Exists(ctx, key)
	dc.metrics.networkLatency.Observe(time.Since(networkStart).Seconds())

	return exists, err
}

// serializeValue serializes and optionally compresses a value
func (dc *DistributedCache) serializeValue(value interface{}) ([]byte, error) {
	// Serialize
	data, err := dc.serializer.Serialize(value)
	if err != nil {
		return nil, err
	}

	// Compress if needed
	if dc.compression.ShouldCompress(data) {
		originalSize := len(data)
		compressed, err := dc.compression.Compress(data)
		if err != nil {
			return data, nil // Fall back to uncompressed
		}

		compressionRatio := float64(len(compressed)) / float64(originalSize)
		dc.metrics.compressionRatio.Observe(compressionRatio)

		return compressed, nil
	}

	return data, nil
}

// deserializeValue deserializes and optionally decompresses a value
func (dc *DistributedCache) deserializeValue(data []byte, value interface{}) error {
	// Try to decompress first (compression libraries usually have magic bytes)
	if decompressed, err := dc.compression.Decompress(data); err == nil {
		data = decompressed
	}

	return dc.serializer.Deserialize(data, value)
}

// Get retrieves a value from memory cache
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	item, exists := mc.items[key]
	if !exists {
		return nil, false
	}

	// Check expiration
	if time.Now().After(item.Expires) {
		delete(mc.items, key)
		mc.currSize -= item.Size
		mc.lruList.Remove(item)
		return nil, false
	}

	// Update LRU
	mc.lruList.MoveToFront(item)
	item.Hits++
	item.UpdatedAt = time.Now()

	return item.Value, true
}

// Set stores a value in memory cache
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()

	// Calculate size (simplified)
	data, _ := json.Marshal(value)
	size := int64(len(data))

	// Check if key already exists
	if existing, exists := mc.items[key]; exists {
		mc.currSize -= existing.Size
		mc.lruList.Remove(existing)
	}

	// Ensure we have space
	for mc.currSize+size > mc.maxSize && mc.lruList.tail != nil {
		oldest := mc.lruList.tail
		delete(mc.items, oldest.Key)
		mc.currSize -= oldest.Size
		mc.lruList.Remove(oldest)
	}

	// Create new item
	item := &CacheItem{
		Key:       key,
		Value:     value,
		Expires:   now.Add(ttl),
		Size:      size,
		CreatedAt: now,
		UpdatedAt: now,
		Checksum:  generateChecksum(data),
	}

	mc.items[key] = item
	mc.currSize += size
	mc.lruList.AddToFront(item)
}

// Delete removes a value from memory cache
func (mc *MemoryCache) Delete(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if item, exists := mc.items[key]; exists {
		delete(mc.items, key)
		mc.currSize -= item.Size
		mc.lruList.Remove(item)
	}
}

// Exists checks if a key exists in memory cache
func (mc *MemoryCache) Exists(key string) bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	item, exists := mc.items[key]
	if !exists {
		return false
	}

	return time.Now().Before(item.Expires)
}

// LRU list operations
func (lru *LRUList) AddToFront(item *CacheItem) {
	if lru.head == nil {
		lru.head = item
		lru.tail = item
	} else {
		item.next = lru.head
		lru.head.prev = item
		lru.head = item
	}
	lru.size++
}

func (lru *LRUList) Remove(item *CacheItem) {
	if item.prev != nil {
		item.prev.next = item.next
	} else {
		lru.head = item.next
	}

	if item.next != nil {
		item.next.prev = item.prev
	} else {
		lru.tail = item.prev
	}

	item.prev = nil
	item.next = nil
	lru.size--
}

func (lru *LRUList) MoveToFront(item *CacheItem) {
	if item == lru.head {
		return
	}

	lru.Remove(item)
	lru.AddToFront(item)
}

// Background operations
func (dc *DistributedCache) backgroundCleanup() {
	for {
		select {
		case <-dc.ctx.Done():
			return
		case <-dc.cleanupTicker.C:
			dc.cleanupExpiredItems()
		}
	}
}

func (dc *DistributedCache) updateStats() {
	for {
		select {
		case <-dc.ctx.Done():
			return
		case <-dc.statsTicker.C:
			dc.updateCacheMetrics()
		}
	}
}

func (dc *DistributedCache) cleanupExpiredItems() {
	dc.memoryCache.mu.Lock()
	defer dc.memoryCache.mu.Unlock()

	now := time.Now()
	for key, item := range dc.memoryCache.items {
		if now.After(item.Expires) {
			delete(dc.memoryCache.items, key)
			dc.memoryCache.currSize -= item.Size
			dc.memoryCache.lruList.Remove(item)
			dc.metrics.evictions.Inc()
		}
	}
}

func (dc *DistributedCache) updateCacheMetrics() {
	dc.memoryCache.mu.RLock()
	defer dc.memoryCache.mu.RUnlock()

	dc.metrics.memoryCacheSize.Set(float64(dc.memoryCache.currSize))
	dc.metrics.cacheSize.Set(float64(dc.memoryCache.currSize))
}

// GetStats returns cache statistics
func (dc *DistributedCache) GetStats() CacheStats {
	dc.memoryCache.mu.RLock()
	defer dc.memoryCache.mu.RUnlock()

	return CacheStats{
		MemoryCacheSize:    dc.memoryCache.currSize,
		MemoryCacheItems:   len(dc.memoryCache.items),
		MemoryCacheMaxSize: dc.memoryCache.maxSize,
		LRUListSize:        dc.memoryCache.lruList.size,
	}
}

// CacheStats holds cache statistics
type CacheStats struct {
	MemoryCacheSize    int64
	MemoryCacheItems   int
	MemoryCacheMaxSize int64
	LRUListSize        int
}

// Close shuts down the cache
func (dc *DistributedCache) Close() error {
	dc.cancel()
	dc.cleanupTicker.Stop()
	dc.statsTicker.Stop()

	if dc.distributedCache != nil {
		return dc.distributedCache.Close()
	}

	return nil
}

// Utility functions
func generateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// JSONSerializer implements JSON serialization
type JSONSerializer struct{}

func (j *JSONSerializer) Serialize(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (j *JSONSerializer) Deserialize(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// GzipCompression implements gzip compression
type GzipCompression struct {
	threshold int
}

func (g *GzipCompression) Compress(data []byte) ([]byte, error) {
	// Simplified - in production use gzip.Writer
	return data, nil
}

func (g *GzipCompression) Decompress(data []byte) ([]byte, error) {
	// Simplified - in production use gzip.Reader
	return data, nil
}

func (g *GzipCompression) ShouldCompress(data []byte) bool {
	return len(data) > g.threshold
}

// Error definitions
var (
	ErrCacheKeyNotFound = fmt.Errorf("cache key not found")
)
