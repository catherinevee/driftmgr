package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/go-redis/redis/v8"
)

type CacheStrategy string

const (
	StrategyLRU        CacheStrategy = "lru"
	StrategyLFU        CacheStrategy = "lfu"
	StrategyARC        CacheStrategy = "arc"
	StrategyPredictive CacheStrategy = "predictive"
)

type CacheEntry struct {
	Key          string                 `json:"key"`
	Value        interface{}            `json:"value"`
	Size         int64                  `json:"size"`
	TTL          time.Duration          `json:"ttl"`
	CreatedAt    time.Time              `json:"created_at"`
	LastAccessed time.Time              `json:"last_accessed"`
	AccessCount  int64                  `json:"access_count"`
	Priority     int                    `json:"priority"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type CacheStats struct {
	Hits            int64         `json:"hits"`
	Misses          int64         `json:"misses"`
	Evictions       int64         `json:"evictions"`
	TotalSize       int64         `json:"total_size"`
	EntryCount      int64         `json:"entry_count"`
	HitRate         float64       `json:"hit_rate"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	MemoryUsage     int64         `json:"memory_usage"`
}

type SmartCache struct {
	mu              sync.RWMutex
	strategy        CacheStrategy
	entries         map[string]*CacheEntry
	lruList         *DoublyLinkedList
	lfuHeap         *FrequencyHeap
	arcT1           *DoublyLinkedList
	arcT2           *DoublyLinkedList
	arcB1           *DoublyLinkedList
	arcB2           *DoublyLinkedList
	arcP            int
	maxSize         int64
	currentSize     int64
	stats           *CacheStats
	redis           *redis.Client
	warmupEnabled   bool
	predictiveModel *PredictiveModel
	compressionEnabled bool
	shardCount      int
	shards          []*CacheShard
}

type CacheShard struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	size    int64
}

type DoublyLinkedList struct {
	head *ListNode
	tail *ListNode
	size int
}

type ListNode struct {
	key  string
	prev *ListNode
	next *ListNode
}

type FrequencyHeap struct {
	items []*HeapItem
}

type HeapItem struct {
	key       string
	frequency int64
}

type PredictiveModel struct {
	accessPatterns map[string]*AccessPattern
	predictions    map[string]float64
}

type AccessPattern struct {
	Times      []time.Time
	Intervals  []time.Duration
	Prediction time.Time
}

func NewSmartCache(strategy CacheStrategy, maxSizeMB int64) *SmartCache {
	shardCount := runtime.NumCPU()
	shards := make([]*CacheShard, shardCount)
	for i := 0; i < shardCount; i++ {
		shards[i] = &CacheShard{
			entries: make(map[string]*CacheEntry),
			size:    0,
		}
	}

	cache := &SmartCache{
		strategy:        strategy,
		entries:         make(map[string]*CacheEntry),
		maxSize:         maxSizeMB * 1024 * 1024, // Convert to bytes
		currentSize:     0,
		stats:           &CacheStats{},
		warmupEnabled:   true,
		compressionEnabled: true,
		shardCount:      shardCount,
		shards:          shards,
		predictiveModel: &PredictiveModel{
			accessPatterns: make(map[string]*AccessPattern),
			predictions:    make(map[string]float64),
		},
	}

	// Initialize strategy-specific structures
	switch strategy {
	case StrategyLRU:
		cache.lruList = &DoublyLinkedList{}
	case StrategyLFU:
		cache.lfuHeap = &FrequencyHeap{}
	case StrategyARC:
		cache.arcT1 = &DoublyLinkedList{}
		cache.arcT2 = &DoublyLinkedList{}
		cache.arcB1 = &DoublyLinkedList{}
		cache.arcB2 = &DoublyLinkedList{}
		cache.arcP = 0
	}

	// Try to connect to Redis for distributed caching
	cache.redis = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// Start background tasks
	go cache.startEvictionWorker()
	go cache.startStatsCollector()
	go cache.startPredictivePreloader()

	return cache
}

func (c *SmartCache) Get(ctx context.Context, key string) (interface{}, error) {
	start := time.Now()
	defer func() {
		c.updateResponseTime(time.Since(start))
	}()

	// Try local cache first
	shard := c.getShard(key)
	shard.mu.RLock()
	entry, exists := shard.entries[key]
	shard.mu.RUnlock()

	if exists && !c.isExpired(entry) {
		atomic.AddInt64(&c.stats.Hits, 1)
		c.updateAccessStats(entry)
		return c.decompress(entry.Value), nil
	}

	// Try distributed cache
	if c.redis != nil {
		val, err := c.redis.Get(ctx, key).Result()
		if err == nil {
			atomic.AddInt64(&c.stats.Hits, 1)
			// Update local cache
			c.Put(ctx, key, val, 5*time.Minute)
			return val, nil
		}
	}

	atomic.AddInt64(&c.stats.Misses, 1)
	return nil, fmt.Errorf("cache miss for key: %s", key)
}

func (c *SmartCache) Put(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	size := c.calculateSize(value)
	
	// Check if we need to evict
	if c.currentSize+size > c.maxSize {
		c.evict(size)
	}

	compressed := c.compress(value)
	entry := &CacheEntry{
		Key:          key,
		Value:        compressed,
		Size:         size,
		TTL:          ttl,
		CreatedAt:    time.Now(),
		LastAccessed: time.Now(),
		AccessCount:  1,
		Priority:     c.calculatePriority(key, value),
	}

	shard := c.getShard(key)
	shard.mu.Lock()
	shard.entries[key] = entry
	shard.size += size
	shard.mu.Unlock()

	atomic.AddInt64(&c.currentSize, size)
	atomic.AddInt64(&c.stats.EntryCount, 1)

	// Update strategy-specific structures
	c.updateStrategy(key, entry)

	// Update distributed cache
	if c.redis != nil {
		data, _ := json.Marshal(value)
		c.redis.Set(ctx, key, data, ttl)
	}

	// Update predictive model
	c.updatePredictiveModel(key)

	return nil
}

func (c *SmartCache) Delete(ctx context.Context, key string) error {
	shard := c.getShard(key)
	shard.mu.Lock()
	entry, exists := shard.entries[key]
	if exists {
		delete(shard.entries, key)
		shard.size -= entry.Size
		atomic.AddInt64(&c.currentSize, -entry.Size)
		atomic.AddInt64(&c.stats.EntryCount, -1)
	}
	shard.mu.Unlock()

	// Remove from distributed cache
	if c.redis != nil {
		c.redis.Del(ctx, key)
	}

	return nil
}

func (c *SmartCache) evict(requiredSize int64) {
	atomic.AddInt64(&c.stats.Evictions, 1)

	switch c.strategy {
	case StrategyLRU:
		c.evictLRU(requiredSize)
	case StrategyLFU:
		c.evictLFU(requiredSize)
	case StrategyARC:
		c.evictARC(requiredSize)
	case StrategyPredictive:
		c.evictPredictive(requiredSize)
	}
}

func (c *SmartCache) evictLRU(requiredSize int64) {
	evicted := int64(0)
	
	for evicted < requiredSize && c.lruList.size > 0 {
		// Remove from tail (least recently used)
		key := c.lruList.removeTail()
		shard := c.getShard(key)
		
		shard.mu.Lock()
		if entry, exists := shard.entries[key]; exists {
			evicted += entry.Size
			delete(shard.entries, key)
			shard.size -= entry.Size
		}
		shard.mu.Unlock()
	}
}

func (c *SmartCache) evictLFU(requiredSize int64) {
	evicted := int64(0)
	
	for evicted < requiredSize && len(c.lfuHeap.items) > 0 {
		// Remove least frequently used
		item := c.lfuHeap.pop()
		shard := c.getShard(item.key)
		
		shard.mu.Lock()
		if entry, exists := shard.entries[item.key]; exists {
			evicted += entry.Size
			delete(shard.entries, item.key)
			shard.size -= entry.Size
		}
		shard.mu.Unlock()
	}
}

func (c *SmartCache) evictARC(requiredSize int64) {
	// Adaptive Replacement Cache algorithm
	evicted := int64(0)
	
	// Evict from T1 or T2 based on adaptive parameter
	if c.arcT1.size > c.arcP {
		// Evict from T1
		for evicted < requiredSize && c.arcT1.size > 0 {
			key := c.arcT1.removeTail()
			c.arcB1.addHead(key)
			
			shard := c.getShard(key)
			shard.mu.Lock()
			if entry, exists := shard.entries[key]; exists {
				evicted += entry.Size
				delete(shard.entries, key)
				shard.size -= entry.Size
			}
			shard.mu.Unlock()
		}
	} else {
		// Evict from T2
		for evicted < requiredSize && c.arcT2.size > 0 {
			key := c.arcT2.removeTail()
			c.arcB2.addHead(key)
			
			shard := c.getShard(key)
			shard.mu.Lock()
			if entry, exists := shard.entries[key]; exists {
				evicted += entry.Size
				delete(shard.entries, key)
				shard.size -= entry.Size
			}
			shard.mu.Unlock()
		}
	}
}

func (c *SmartCache) evictPredictive(requiredSize int64) {
	// Evict based on predicted future access patterns
	evicted := int64(0)
	
	// Sort entries by predicted next access time
	predictions := make([]*PredictionEntry, 0)
	for key := range c.entries {
		if pred, exists := c.predictiveModel.predictions[key]; exists {
			predictions = append(predictions, &PredictionEntry{
				Key:        key,
				Prediction: pred,
			})
		}
	}
	
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].Prediction > predictions[j].Prediction
	})
	
	// Evict entries predicted to be accessed furthest in the future
	for _, pred := range predictions {
		if evicted >= requiredSize {
			break
		}
		
		shard := c.getShard(pred.Key)
		shard.mu.Lock()
		if entry, exists := shard.entries[pred.Key]; exists {
			evicted += entry.Size
			delete(shard.entries, pred.Key)
			shard.size -= entry.Size
		}
		shard.mu.Unlock()
	}
}

type PredictionEntry struct {
	Key        string
	Prediction float64
}

func (c *SmartCache) updateStrategy(key string, entry *CacheEntry) {
	switch c.strategy {
	case StrategyLRU:
		c.lruList.addHead(key)
	case StrategyLFU:
		c.lfuHeap.push(&HeapItem{
			key:       key,
			frequency: entry.AccessCount,
		})
	case StrategyARC:
		c.arcT1.addHead(key)
	}
}

func (c *SmartCache) updateAccessStats(entry *CacheEntry) {
	entry.LastAccessed = time.Now()
	atomic.AddInt64(&entry.AccessCount, 1)
	
	// Update strategy-specific access tracking
	switch c.strategy {
	case StrategyLRU:
		c.lruList.moveToHead(entry.Key)
	case StrategyLFU:
		c.lfuHeap.update(entry.Key, entry.AccessCount)
	case StrategyARC:
		// Move between T1 and T2 based on access pattern
		if c.arcT1.contains(entry.Key) {
			c.arcT1.remove(entry.Key)
			c.arcT2.addHead(entry.Key)
		}
	}
}

func (c *SmartCache) updatePredictiveModel(key string) {
	pattern, exists := c.predictiveModel.accessPatterns[key]
	if !exists {
		pattern = &AccessPattern{
			Times:     []time.Time{},
			Intervals: []time.Duration{},
		}
		c.predictiveModel.accessPatterns[key] = pattern
	}
	
	now := time.Now()
	pattern.Times = append(pattern.Times, now)
	
	if len(pattern.Times) > 1 {
		interval := now.Sub(pattern.Times[len(pattern.Times)-2])
		pattern.Intervals = append(pattern.Intervals, interval)
		
		// Predict next access time
		if len(pattern.Intervals) >= 3 {
			avgInterval := c.calculateAverageInterval(pattern.Intervals)
			pattern.Prediction = now.Add(avgInterval)
			c.predictiveModel.predictions[key] = avgInterval.Seconds()
		}
	}
	
	// Keep only recent history
	if len(pattern.Times) > 100 {
		pattern.Times = pattern.Times[50:]
		pattern.Intervals = pattern.Intervals[50:]
	}
}

func (c *SmartCache) calculateAverageInterval(intervals []time.Duration) time.Duration {
	if len(intervals) == 0 {
		return time.Hour
	}
	
	var total time.Duration
	for _, interval := range intervals {
		total += interval
	}
	
	return total / time.Duration(len(intervals))
}

func (c *SmartCache) startEvictionWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		// Check memory pressure
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		
		if m.Alloc > uint64(c.maxSize)*8/10 { // 80% threshold
			c.evict(c.maxSize / 10) // Free 10%
		}
		
		// Clean expired entries
		c.cleanExpired()
	}
}

func (c *SmartCache) cleanExpired() {
	for i := 0; i < c.shardCount; i++ {
		shard := c.shards[i]
		shard.mu.Lock()
		
		for key, entry := range shard.entries {
			if c.isExpired(entry) {
				delete(shard.entries, key)
				atomic.AddInt64(&c.currentSize, -entry.Size)
				atomic.AddInt64(&c.stats.EntryCount, -1)
			}
		}
		
		shard.mu.Unlock()
	}
}

func (c *SmartCache) startStatsCollector() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		hits := atomic.LoadInt64(&c.stats.Hits)
		misses := atomic.LoadInt64(&c.stats.Misses)
		total := hits + misses
		
		if total > 0 {
			c.stats.HitRate = float64(hits) / float64(total)
		}
		
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		c.stats.MemoryUsage = int64(m.Alloc)
		c.stats.TotalSize = atomic.LoadInt64(&c.currentSize)
	}
}

func (c *SmartCache) startPredictivePreloader() {
	if !c.warmupEnabled {
		return
	}
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		// Preload entries predicted to be accessed soon
		now := time.Now()
		
		for key, pattern := range c.predictiveModel.accessPatterns {
			if pattern.Prediction.After(now) && pattern.Prediction.Before(now.Add(10*time.Minute)) {
				// This entry is likely to be accessed soon
				shard := c.getShard(key)
				shard.mu.RLock()
				_, exists := shard.entries[key]
				shard.mu.RUnlock()
				
				if !exists {
					// Trigger preload (would need actual data source)
					c.preloadEntry(key)
				}
			}
		}
	}
}

func (c *SmartCache) preloadEntry(key string) {
	// This would fetch from the actual data source
	// For now, just a placeholder
}

func (c *SmartCache) getShard(key string) *CacheShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return c.shards[h.Sum32()%uint32(c.shardCount)]
}

func (c *SmartCache) isExpired(entry *CacheEntry) bool {
	if entry.TTL == 0 {
		return false
	}
	return time.Since(entry.CreatedAt) > entry.TTL
}

func (c *SmartCache) calculateSize(value interface{}) int64 {
	// Simplified size calculation
	data, _ := json.Marshal(value)
	return int64(len(data))
}

func (c *SmartCache) calculatePriority(key string, value interface{}) int {
	// Calculate priority based on various factors
	priority := 5 // Default medium priority
	
	// Increase priority for frequently accessed patterns
	if pattern, exists := c.predictiveModel.accessPatterns[key]; exists {
		if len(pattern.Times) > 10 {
			priority += 2
		}
	}
	
	// Increase priority for small items (keep more in cache)
	size := c.calculateSize(value)
	if size < 1024 { // Less than 1KB
		priority += 1
	}
	
	return priority
}

func (c *SmartCache) compress(value interface{}) interface{} {
	if !c.compressionEnabled {
		return value
	}
	// Simplified - would use actual compression
	return value
}

func (c *SmartCache) decompress(value interface{}) interface{} {
	if !c.compressionEnabled {
		return value
	}
	// Simplified - would use actual decompression
	return value
}

func (c *SmartCache) updateResponseTime(duration time.Duration) {
	// Exponential moving average
	alpha := 0.1
	current := c.stats.AvgResponseTime.Nanoseconds()
	new := duration.Nanoseconds()
	avg := int64(float64(current)*(1-alpha) + float64(new)*alpha)
	c.stats.AvgResponseTime = time.Duration(avg)
}

func (c *SmartCache) GetStats() *CacheStats {
	return &CacheStats{
		Hits:            atomic.LoadInt64(&c.stats.Hits),
		Misses:          atomic.LoadInt64(&c.stats.Misses),
		Evictions:       atomic.LoadInt64(&c.stats.Evictions),
		TotalSize:       atomic.LoadInt64(&c.stats.TotalSize),
		EntryCount:      atomic.LoadInt64(&c.stats.EntryCount),
		HitRate:         c.stats.HitRate,
		AvgResponseTime: c.stats.AvgResponseTime,
		MemoryUsage:     c.stats.MemoryUsage,
	}
}

func (c *SmartCache) WarmUp(ctx context.Context, resources []models.Resource) error {
	// Preload cache with commonly accessed resources
	for _, resource := range resources {
		key := fmt.Sprintf("%s/%s/%s", resource.Provider, resource.Type, resource.ID)
		c.Put(ctx, key, resource, 1*time.Hour)
	}
	return nil
}

func (c *SmartCache) Clear() {
	for i := 0; i < c.shardCount; i++ {
		shard := c.shards[i]
		shard.mu.Lock()
		shard.entries = make(map[string]*CacheEntry)
		shard.size = 0
		shard.mu.Unlock()
	}
	
	atomic.StoreInt64(&c.currentSize, 0)
	atomic.StoreInt64(&c.stats.EntryCount, 0)
	
	// Clear strategy-specific structures
	switch c.strategy {
	case StrategyLRU:
		c.lruList = &DoublyLinkedList{}
	case StrategyLFU:
		c.lfuHeap = &FrequencyHeap{}
	case StrategyARC:
		c.arcT1 = &DoublyLinkedList{}
		c.arcT2 = &DoublyLinkedList{}
		c.arcB1 = &DoublyLinkedList{}
		c.arcB2 = &DoublyLinkedList{}
		c.arcP = 0
	}
}

// DoublyLinkedList methods
func (dll *DoublyLinkedList) addHead(key string) {
	node := &ListNode{key: key}
	if dll.head == nil {
		dll.head = node
		dll.tail = node
	} else {
		node.next = dll.head
		dll.head.prev = node
		dll.head = node
	}
	dll.size++
}

func (dll *DoublyLinkedList) removeTail() string {
	if dll.tail == nil {
		return ""
	}
	key := dll.tail.key
	if dll.tail.prev != nil {
		dll.tail = dll.tail.prev
		dll.tail.next = nil
	} else {
		dll.head = nil
		dll.tail = nil
	}
	dll.size--
	return key
}

func (dll *DoublyLinkedList) remove(key string) {
	node := dll.head
	for node != nil {
		if node.key == key {
			if node.prev != nil {
				node.prev.next = node.next
			} else {
				dll.head = node.next
			}
			if node.next != nil {
				node.next.prev = node.prev
			} else {
				dll.tail = node.prev
			}
			dll.size--
			return
		}
		node = node.next
	}
}

func (dll *DoublyLinkedList) moveToHead(key string) {
	dll.remove(key)
	dll.addHead(key)
}

func (dll *DoublyLinkedList) contains(key string) bool {
	node := dll.head
	for node != nil {
		if node.key == key {
			return true
		}
		node = node.next
	}
	return false
}

// FrequencyHeap methods
func (fh *FrequencyHeap) push(item *HeapItem) {
	fh.items = append(fh.items, item)
	fh.bubbleUp(len(fh.items) - 1)
}

func (fh *FrequencyHeap) pop() *HeapItem {
	if len(fh.items) == 0 {
		return nil
	}
	item := fh.items[0]
	fh.items[0] = fh.items[len(fh.items)-1]
	fh.items = fh.items[:len(fh.items)-1]
	if len(fh.items) > 0 {
		fh.bubbleDown(0)
	}
	return item
}

func (fh *FrequencyHeap) update(key string, frequency int64) {
	for i, item := range fh.items {
		if item.key == key {
			item.frequency = frequency
			fh.bubbleUp(i)
			fh.bubbleDown(i)
			return
		}
	}
}

func (fh *FrequencyHeap) bubbleUp(index int) {
	for index > 0 {
		parent := (index - 1) / 2
		if fh.items[index].frequency < fh.items[parent].frequency {
			fh.items[index], fh.items[parent] = fh.items[parent], fh.items[index]
			index = parent
		} else {
			break
		}
	}
}

func (fh *FrequencyHeap) bubbleDown(index int) {
	for {
		left := 2*index + 1
		right := 2*index + 2
		smallest := index
		
		if left < len(fh.items) && fh.items[left].frequency < fh.items[smallest].frequency {
			smallest = left
		}
		if right < len(fh.items) && fh.items[right].frequency < fh.items[smallest].frequency {
			smallest = right
		}
		
		if smallest != index {
			fh.items[index], fh.items[smallest] = fh.items[smallest], fh.items[index]
			index = smallest
		} else {
			break
		}
	}
}