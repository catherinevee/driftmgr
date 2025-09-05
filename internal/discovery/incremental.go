package discovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/catherinevee/driftmgr/internal/providers"
)

// IncrementalDiscovery provides efficient incremental resource discovery
type IncrementalDiscovery struct {
	providers       map[string]providers.CloudProvider
	cache           *DiscoveryCache
	changeTracker   *ChangeTracker
	bloomFilter     *bloom.BloomFilter
	config          DiscoveryConfig
	mu              sync.RWMutex
}

// DiscoveryConfig configures incremental discovery
type DiscoveryConfig struct {
	CacheDuration      time.Duration
	BloomFilterSize    uint
	BloomFilterHashes  uint
	ParallelWorkers    int
	BatchSize          int
	UseCloudTrails     bool
	UseResourceTags    bool
	DifferentialSync   bool
}

// DiscoveryCache caches resource states with TTL
type DiscoveryCache struct {
	resources map[string]*CachedResource
	mu        sync.RWMutex
}

// CachedResource represents a cached resource
type CachedResource struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	Attributes   map[string]interface{} `json:"attributes"`
	Version      string                 `json:"version"`
	ETag         string                 `json:"etag,omitempty"`
	LastModified time.Time              `json:"last_modified"`
	LastChecked  time.Time              `json:"last_checked"`
	Checksum     string                 `json:"checksum"`
	TTL          time.Duration          `json:"-"`
}

// ChangeTracker tracks resource changes using various methods
type ChangeTracker struct {
	lastDiscovery   map[string]time.Time
	resourceETags   map[string]string
	changeLogReader ChangeLogReader
	mu              sync.RWMutex
}

// ChangeLogReader reads cloud provider change logs
type ChangeLogReader interface {
	GetChanges(ctx context.Context, since time.Time) ([]ResourceChange, error)
}

// ResourceChange represents a change to a resource
type ResourceChange struct {
	ResourceID string
	ChangeType string
	Timestamp  time.Time
	Details    map[string]interface{}
}

// DiscoveryResult represents the result of incremental discovery
type DiscoveryResult struct {
	NewResources     []interface{}
	UpdatedResources []interface{}
	DeletedResources []string
	UnchangedCount   int
	DiscoveryTime    time.Duration
	CacheHits        int
	CacheMisses      int
}

// NewIncrementalDiscovery creates a new incremental discovery engine
func NewIncrementalDiscovery(config DiscoveryConfig) *IncrementalDiscovery {
	if config.CacheDuration == 0 {
		config.CacheDuration = 5 * time.Minute
	}
	if config.BloomFilterSize == 0 {
		config.BloomFilterSize = 100000
	}
	if config.BloomFilterHashes == 0 {
		config.BloomFilterHashes = 5
	}
	if config.ParallelWorkers == 0 {
		config.ParallelWorkers = 10
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	
	return &IncrementalDiscovery{
		providers:     make(map[string]providers.CloudProvider),
		cache:         NewDiscoveryCache(),
		changeTracker: NewChangeTracker(),
		bloomFilter:   bloom.NewWithEstimates(config.BloomFilterSize, 0.01),
		config:        config,
	}
}

// RegisterProvider registers a cloud provider
func (d *IncrementalDiscovery) RegisterProvider(name string, provider providers.CloudProvider) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.providers[name] = provider
}

// DiscoverIncremental performs incremental discovery
func (d *IncrementalDiscovery) DiscoverIncremental(ctx context.Context) (*DiscoveryResult, error) {
	startTime := time.Now()
	result := &DiscoveryResult{}
	
	// Step 1: Get changed resources from cloud trails/audit logs
	changedResources := d.getChangedResources(ctx)
	
	// Step 2: Check bloom filter for potentially changed resources
	potentialChanges := d.checkBloomFilter(changedResources)
	
	// Step 3: Perform differential discovery
	if d.config.DifferentialSync {
		d.performDifferentialSync(ctx, potentialChanges, result)
	} else {
		d.performFullDiscovery(ctx, result)
	}
	
	// Step 4: Update cache and bloom filter
	d.updateCache(result)
	d.updateBloomFilter(result)
	
	result.DiscoveryTime = time.Since(startTime)
	return result, nil
}

// getChangedResources retrieves changed resources from audit logs
func (d *IncrementalDiscovery) getChangedResources(ctx context.Context) []string {
	if !d.config.UseCloudTrails {
		return nil
	}
	
	d.changeTracker.mu.RLock()
	lastCheck := d.changeTracker.lastDiscovery["global"]
	d.changeTracker.mu.RUnlock()
	
	if d.changeTracker.changeLogReader != nil {
		changes, err := d.changeTracker.changeLogReader.GetChanges(ctx, lastCheck)
		if err == nil {
			var resourceIDs []string
			for _, change := range changes {
				resourceIDs = append(resourceIDs, change.ResourceID)
			}
			return resourceIDs
		}
	}
	
	return nil
}

// checkBloomFilter checks for potentially changed resources
func (d *IncrementalDiscovery) checkBloomFilter(knownChanges []string) []string {
	var potentialChanges []string
	
	// Add known changes
	potentialChanges = append(potentialChanges, knownChanges...)
	
	// Check cached resources for potential changes
	d.cache.mu.RLock()
	for id, resource := range d.cache.resources {
		// Check if resource might have changed
		if !d.bloomFilter.Test([]byte(resource.Checksum)) {
			potentialChanges = append(potentialChanges, id)
		}
	}
	d.cache.mu.RUnlock()
	
	return potentialChanges
}

// performDifferentialSync performs differential synchronization
func (d *IncrementalDiscovery) performDifferentialSync(ctx context.Context, targetResources []string, result *DiscoveryResult) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Create worker pool
	workerChan := make(chan string, len(targetResources))
	resultChan := make(chan interface{}, len(targetResources))
	
	var wg sync.WaitGroup
	for i := 0; i < d.config.ParallelWorkers; i++ {
		wg.Add(1)
		go d.discoveryWorker(ctx, workerChan, resultChan, &wg)
	}
	
	// Queue resources for discovery
	for _, resourceID := range targetResources {
		workerChan <- resourceID
	}
	close(workerChan)
	
	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results
	for resource := range resultChan {
		if resource != nil {
			// Check if resource is new or updated
			if cached := d.cache.Get(getResourceID(resource)); cached != nil {
				if !d.resourcesEqual(cached, resource) {
					result.UpdatedResources = append(result.UpdatedResources, resource)
				} else {
					result.UnchangedCount++
					result.CacheHits++
				}
			} else {
				result.NewResources = append(result.NewResources, resource)
				result.CacheMisses++
			}
		}
	}
}

// performFullDiscovery performs full resource discovery
func (d *IncrementalDiscovery) performFullDiscovery(ctx context.Context, result *DiscoveryResult) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	currentResources := make(map[string]interface{})
	
	// Discover from all providers
	for name, provider := range d.providers {
		resources, err := provider.DiscoverResources(ctx, "")
		if err != nil {
			fmt.Printf("Error discovering from %s: %v\n", name, err)
			continue
		}
		
		for _, resource := range resources {
			id := getResourceID(resource)
			currentResources[id] = resource
			
			// Check cache
			if cached := d.cache.Get(id); cached != nil {
				if !d.resourcesEqual(cached, resource) {
					result.UpdatedResources = append(result.UpdatedResources, resource)
				} else {
					result.UnchangedCount++
					result.CacheHits++
				}
			} else {
				result.NewResources = append(result.NewResources, resource)
				result.CacheMisses++
			}
		}
	}
	
	// Detect deletions
	d.cache.mu.RLock()
	for id := range d.cache.resources {
		if _, exists := currentResources[id]; !exists {
			result.DeletedResources = append(result.DeletedResources, id)
		}
	}
	d.cache.mu.RUnlock()
}

// discoveryWorker is a worker for parallel resource discovery
func (d *IncrementalDiscovery) discoveryWorker(ctx context.Context, workerChan chan string, resultChan chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for resourceID := range workerChan {
		// Discover specific resource
		resource := d.discoverResource(ctx, resourceID)
		resultChan <- resource
	}
}

// discoverResource discovers a specific resource
func (d *IncrementalDiscovery) discoverResource(ctx context.Context, resourceID string) interface{} {
	// Determine provider and discover resource
	// This is simplified - in reality would parse resource ID to determine provider
	for _, provider := range d.providers {
		// Try to discover resource from provider
		// This would call a provider-specific method
		_ = provider
	}
	return nil
}

// updateCache updates the discovery cache
func (d *IncrementalDiscovery) updateCache(result *DiscoveryResult) {
	// Update cache with new and updated resources
	for _, resource := range result.NewResources {
		d.cache.Put(resource)
	}
	for _, resource := range result.UpdatedResources {
		d.cache.Put(resource)
	}
	
	// Remove deleted resources from cache
	for _, id := range result.DeletedResources {
		d.cache.Delete(id)
	}
}

// updateBloomFilter updates the bloom filter
func (d *IncrementalDiscovery) updateBloomFilter(result *DiscoveryResult) {
	// Add checksums of all current resources to bloom filter
	for _, resource := range result.NewResources {
		checksum := d.calculateChecksum(resource)
		d.bloomFilter.Add([]byte(checksum))
	}
	for _, resource := range result.UpdatedResources {
		checksum := d.calculateChecksum(resource)
		d.bloomFilter.Add([]byte(checksum))
	}
}

// resourcesEqual compares two resources for equality
func (d *IncrementalDiscovery) resourcesEqual(cached *CachedResource, current interface{}) bool {
	// Use ETag if available
	if cached.ETag != "" {
		if etag := getResourceETag(current); etag != "" {
			return cached.ETag == etag
		}
	}
	
	// Use checksum comparison
	currentChecksum := d.calculateChecksum(current)
	return cached.Checksum == currentChecksum
}

// calculateChecksum calculates a checksum for a resource
func (d *IncrementalDiscovery) calculateChecksum(resource interface{}) string {
	data, _ := json.Marshal(resource)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// DiscoveryCache implementation

// NewDiscoveryCache creates a new discovery cache
func NewDiscoveryCache() *DiscoveryCache {
	return &DiscoveryCache{
		resources: make(map[string]*CachedResource),
	}
}

// Get retrieves a resource from cache
func (c *DiscoveryCache) Get(id string) *CachedResource {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	resource, exists := c.resources[id]
	if !exists {
		return nil
	}
	
	// Check if cached resource has expired
	if time.Since(resource.LastChecked) > resource.TTL {
		return nil
	}
	
	return resource
}

// Put adds or updates a resource in cache
func (c *DiscoveryCache) Put(resource interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	id := getResourceID(resource)
	cached := &CachedResource{
		ID:          id,
		Type:        getResourceType(resource),
		LastChecked: time.Now(),
		TTL:         5 * time.Minute,
	}
	
	// Extract additional metadata
	if resMap, ok := resource.(map[string]interface{}); ok {
		cached.Attributes = resMap
		if etag, ok := resMap["etag"].(string); ok {
			cached.ETag = etag
		}
		if modified, ok := resMap["last_modified"].(time.Time); ok {
			cached.LastModified = modified
		}
	}
	
	c.resources[id] = cached
}

// Delete removes a resource from cache
func (c *DiscoveryCache) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.resources, id)
}

// Clear clears the entire cache
func (c *DiscoveryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resources = make(map[string]*CachedResource)
}

// ChangeTracker implementation

// NewChangeTracker creates a new change tracker
func NewChangeTracker() *ChangeTracker {
	return &ChangeTracker{
		lastDiscovery: make(map[string]time.Time),
		resourceETags: make(map[string]string),
	}
}

// SetChangeLogReader sets the change log reader
func (t *ChangeTracker) SetChangeLogReader(reader ChangeLogReader) {
	t.changeLogReader = reader
}

// UpdateLastDiscovery updates the last discovery time
func (t *ChangeTracker) UpdateLastDiscovery(provider string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastDiscovery[provider] = time.Now()
}

// UpdateETag updates a resource's ETag
func (t *ChangeTracker) UpdateETag(resourceID, etag string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.resourceETags[resourceID] = etag
}

// GetETag gets a resource's ETag
func (t *ChangeTracker) GetETag(resourceID string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.resourceETags[resourceID]
}

// Helper functions

func getResourceID(resource interface{}) string {
	if resMap, ok := resource.(map[string]interface{}); ok {
		if id, ok := resMap["id"].(string); ok {
			return id
		}
	}
	return ""
}

func getResourceType(resource interface{}) string {
	if resMap, ok := resource.(map[string]interface{}); ok {
		if typ, ok := resMap["type"].(string); ok {
			return typ
		}
	}
	return ""
}

func getResourceETag(resource interface{}) string {
	if resMap, ok := resource.(map[string]interface{}); ok {
		if etag, ok := resMap["etag"].(string); ok {
			return etag
		}
	}
	return ""
}