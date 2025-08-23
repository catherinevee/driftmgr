package performance

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// IncrementalDiscovery provides incremental resource discovery with change detection
type IncrementalDiscovery struct {
	mu               sync.RWMutex
	stateManager     *StateManager
	changeDetector   *ChangeDetector
	deltaProcessor   *DeltaProcessor
	comparisonEngine *ComparisonEngine
	config           *DiscoveryConfig
	metrics          *DiscoveryMetrics

	// State tracking
	lastDiscovery    time.Time
	currentSnapshot  *ResourceSnapshot
	previousSnapshot *ResourceSnapshot

	// Background operations
	ctx            context.Context
	cancel         context.CancelFunc
	snapshotTicker *time.Ticker
	cleanupTicker  *time.Ticker
}

// DiscoveryConfig holds configuration for incremental discovery
type DiscoveryConfig struct {
	// Snapshot settings
	SnapshotInterval time.Duration
	RetentionPeriod  time.Duration
	MaxSnapshots     int

	// Change detection settings
	HashAlgorithm     string
	IgnoreFields      []string
	SensitiveFields   []string
	ChangeSensitivity float64

	// Delta processing
	DeltaBatchSize  int
	DeltaTimeout    time.Duration
	MaxDeltaHistory int

	// Comparison settings
	DeepCompare        bool
	ParallelComparison bool
	ComparisonWorkers  int

	// Performance settings
	EnableCompression bool
	CompressionLevel  int
	CacheComparisons  bool
	OptimizeMemory    bool
}

// DiscoveryMetrics holds Prometheus metrics for incremental discovery
type DiscoveryMetrics struct {
	snapshotsCreated         prometheus.Counter
	comparisonsPerformed     prometheus.Counter
	changesDetected          prometheus.Counter
	deltaOperations          prometheus.Counter
	snapshotSize             prometheus.Histogram
	comparisonTime           prometheus.Histogram
	deltaProcessingTime      prometheus.Histogram
	resourcesAdded           prometheus.Counter
	resourcesRemoved         prometheus.Counter
	resourcesModified        prometheus.Counter
	resourcesUnchanged       prometheus.Counter
	stateSize                prometheus.Gauge
	snapshotCompressionRatio prometheus.Histogram
}

// StateManager manages resource state and snapshots
type StateManager struct {
	mu           sync.RWMutex
	snapshots    map[string]*ResourceSnapshot
	currentState *ResourceState
	storage      StateStorage
	compression  CompressionHandler
	serializer   Serializer
	metrics      *StateMetrics
}

// StateMetrics holds state management metrics
type StateMetrics struct {
	stateUpdates    prometheus.Counter
	snapshotsSaved  prometheus.Counter
	snapshotsLoaded prometheus.Counter
	stateSize       prometheus.Gauge
	storageErrors   prometheus.Counter
}

// ResourceSnapshot represents a point-in-time snapshot of resources
type ResourceSnapshot struct {
	ID         string                      `json:"id"`
	Timestamp  time.Time                   `json:"timestamp"`
	Resources  map[string]*models.Resource `json:"resources"`
	Metadata   map[string]interface{}      `json:"metadata"`
	Checksum   string                      `json:"checksum"`
	Version    int                         `json:"version"`
	Size       int64                       `json:"size"`
	Compressed bool                        `json:"compressed"`

	// Statistics
	TotalResources int            `json:"total_resources"`
	ByProvider     map[string]int `json:"by_provider"`
	ByRegion       map[string]int `json:"by_region"`
	ByType         map[string]int `json:"by_type"`
}

// ResourceState represents the current state of all resources
type ResourceState struct {
	mu          sync.RWMutex
	resources   map[string]*models.Resource
	lastUpdated time.Time
	version     int64
	checksum    string
}

// ChangeDetector detects changes between resource snapshots
type ChangeDetector struct {
	config      *ChangeDetectionConfig
	hashCache   map[string]string
	changeCache map[string]*ChangeSet
	mu          sync.RWMutex
	metrics     *ChangeDetectionMetrics
}

// ChangeDetectionConfig holds change detection configuration
type ChangeDetectionConfig struct {
	Algorithm         string
	IgnoreFields      []string
	SensitiveFields   []string
	Sensitivity       float64
	CacheResults      bool
	ParallelDetection bool
	Workers           int
}

// ChangeDetectionMetrics holds change detection metrics
type ChangeDetectionMetrics struct {
	changesDetected  prometheus.Counter
	hashCalculations prometheus.Counter
	cacheHits        prometheus.Counter
	cacheMisses      prometheus.Counter
	detectionTime    prometheus.Histogram
}

// ChangeSet represents a set of changes between snapshots
type ChangeSet struct {
	ID           string           `json:"id"`
	FromSnapshot string           `json:"from_snapshot"`
	ToSnapshot   string           `json:"to_snapshot"`
	Timestamp    time.Time        `json:"timestamp"`
	Changes      []ResourceChange `json:"changes"`
	Summary      ChangeSummary    `json:"summary"`

	// Performance data
	DetectionTime time.Duration `json:"detection_time"`
	ResourceCount int           `json:"resource_count"`
}

// ResourceChange represents a change to a resource
type ResourceChange struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	ChangeType   ChangeType             `json:"change_type"`
	Field        string                 `json:"field,omitempty"`
	OldValue     interface{}            `json:"old_value,omitempty"`
	NewValue     interface{}            `json:"new_value,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Severity     ChangeSeverity         `json:"severity"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Checksum     string                 `json:"checksum"`
}

// ChangeType represents the type of change
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeRemoved  ChangeType = "removed"
	ChangeTypeModified ChangeType = "modified"
	ChangeTypeMoved    ChangeType = "moved"
)

// ChangeSeverity represents the severity of a change
type ChangeSeverity string

const (
	SeverityLow      ChangeSeverity = "low"
	SeverityMedium   ChangeSeverity = "medium"
	SeverityHigh     ChangeSeverity = "high"
	SeverityCritical ChangeSeverity = "critical"
)

// ChangeSummary provides a summary of changes
type ChangeSummary struct {
	TotalChanges   int                    `json:"total_changes"`
	ByType         map[ChangeType]int     `json:"by_type"`
	BySeverity     map[ChangeSeverity]int `json:"by_severity"`
	ByResourceType map[string]int         `json:"by_resource_type"`
	ByProvider     map[string]int         `json:"by_provider"`
	ByRegion       map[string]int         `json:"by_region"`
}

// DeltaProcessor processes delta changes efficiently
type DeltaProcessor struct {
	mu        sync.RWMutex
	deltas    []ResourceChange
	batchSize int
	timeout   time.Duration
	lastFlush time.Time
	processor func([]ResourceChange) error
	metrics   *DeltaMetrics
}

// DeltaMetrics holds delta processing metrics
type DeltaMetrics struct {
	deltasProcessed  prometheus.Counter
	batchesProcessed prometheus.Counter
	processingTime   prometheus.Histogram
	batchSize        prometheus.Histogram
}

// ComparisonEngine performs optimized resource comparisons
type ComparisonEngine struct {
	config      *ComparisonConfig
	workers     int
	workQueue   chan ComparisonTask
	resultQueue chan ComparisonResult
	running     bool
	mu          sync.RWMutex
	metrics     *ComparisonMetrics
}

// ComparisonConfig holds comparison engine configuration
type ComparisonConfig struct {
	Workers      int
	QueueSize    int
	DeepCompare  bool
	IgnoreFields []string
	CacheResults bool
}

// ComparisonMetrics holds comparison metrics
type ComparisonMetrics struct {
	comparisonsPerformed prometheus.Counter
	comparisonTime       prometheus.Histogram
	cacheHits            prometheus.Counter
	cacheMisses          prometheus.Counter
}

// ComparisonTask represents a comparison task
type ComparisonTask struct {
	ID        string
	Resource1 *Resource
	Resource2 *Resource
	Options   ComparisonOptions
}

// ComparisonOptions holds options for resource comparison
type ComparisonOptions struct {
	IgnoreFields    []string
	DeepCompare     bool
	Sensitivity     float64
	CompareMetadata bool
}

// ComparisonResult represents the result of a resource comparison
type ComparisonResult struct {
	TaskID     string
	Equal      bool
	Changes    []ResourceChange
	Similarity float64
	Duration   time.Duration
	Error      error
}

// StateStorage defines the interface for state storage backends
type StateStorage interface {
	SaveSnapshot(snapshot *ResourceSnapshot) error
	LoadSnapshot(id string) (*ResourceSnapshot, error)
	ListSnapshots() ([]string, error)
	DeleteSnapshot(id string) error
	GetSnapshotMetadata(id string) (*SnapshotMetadata, error)
}

// SnapshotMetadata holds metadata about a snapshot
type SnapshotMetadata struct {
	ID            string
	Timestamp     time.Time
	Size          int64
	ResourceCount int
	Compressed    bool
	Checksum      string
}

// NewIncrementalDiscovery creates a new incremental discovery instance
func NewIncrementalDiscovery(storage StateStorage, config *DiscoveryConfig) *IncrementalDiscovery {
	ctx, cancel := context.WithCancel(context.Background())

	metrics := &DiscoveryMetrics{
		snapshotsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_snapshots_created_total",
			Help: "Total number of snapshots created",
		}),
		comparisonsPerformed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_comparisons_performed_total",
			Help: "Total number of resource comparisons performed",
		}),
		changesDetected: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_changes_detected_total",
			Help: "Total number of changes detected",
		}),
		deltaOperations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_delta_operations_total",
			Help: "Total number of delta operations",
		}),
		snapshotSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_snapshot_size_bytes",
			Help:    "Snapshot size in bytes",
			Buckets: []float64{1024, 10240, 102400, 1048576, 10485760, 104857600},
		}),
		comparisonTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_comparison_time_seconds",
			Help:    "Resource comparison time",
			Buckets: prometheus.DefBuckets,
		}),
		deltaProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_delta_processing_time_seconds",
			Help:    "Delta processing time",
			Buckets: prometheus.DefBuckets,
		}),
		resourcesAdded: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_resources_added_total",
			Help: "Total number of resources added",
		}),
		resourcesRemoved: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_resources_removed_total",
			Help: "Total number of resources removed",
		}),
		resourcesModified: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_resources_modified_total",
			Help: "Total number of resources modified",
		}),
		resourcesUnchanged: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_resources_unchanged_total",
			Help: "Total number of resources unchanged",
		}),
		stateSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "driftmgr_state_size_bytes",
			Help: "Current state size in bytes",
		}),
		snapshotCompressionRatio: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_snapshot_compression_ratio",
			Help:    "Snapshot compression ratio",
			Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		}),
	}

	stateMetrics := &StateMetrics{
		stateUpdates: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_state_updates_total",
			Help: "Total number of state updates",
		}),
		snapshotsSaved: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_snapshots_saved_total",
			Help: "Total number of snapshots saved",
		}),
		snapshotsLoaded: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_snapshots_loaded_total",
			Help: "Total number of snapshots loaded",
		}),
		stateSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "driftmgr_current_state_size_bytes",
			Help: "Current state size in bytes",
		}),
		storageErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_storage_errors_total",
			Help: "Total number of storage errors",
		}),
	}

	changeMetrics := &ChangeDetectionMetrics{
		changesDetected: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_change_detection_changes_total",
			Help: "Total number of changes detected by change detection",
		}),
		hashCalculations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_hash_calculations_total",
			Help: "Total number of hash calculations",
		}),
		cacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_change_cache_hits_total",
			Help: "Total number of change detection cache hits",
		}),
		cacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_change_cache_misses_total",
			Help: "Total number of change detection cache misses",
		}),
		detectionTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_change_detection_time_seconds",
			Help:    "Change detection time",
			Buckets: prometheus.DefBuckets,
		}),
	}

	deltaMetrics := &DeltaMetrics{
		deltasProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_deltas_processed_total",
			Help: "Total number of deltas processed",
		}),
		batchesProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_delta_batches_processed_total",
			Help: "Total number of delta batches processed",
		}),
		processingTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_delta_batch_processing_time_seconds",
			Help:    "Delta batch processing time",
			Buckets: prometheus.DefBuckets,
		}),
		batchSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_delta_batch_size",
			Help:    "Delta batch size",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),
	}

	comparisonMetrics := &ComparisonMetrics{
		comparisonsPerformed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_resource_comparisons_total",
			Help: "Total number of resource comparisons performed",
		}),
		comparisonTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "driftmgr_resource_comparison_time_seconds",
			Help:    "Resource comparison time",
			Buckets: prometheus.DefBuckets,
		}),
		cacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_comparison_cache_hits_total",
			Help: "Total number of comparison cache hits",
		}),
		cacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "driftmgr_comparison_cache_misses_total",
			Help: "Total number of comparison cache misses",
		}),
	}

	discovery := &IncrementalDiscovery{
		config:  config,
		metrics: metrics,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Initialize state manager
	discovery.stateManager = &StateManager{
		snapshots: make(map[string]*ResourceSnapshot),
		currentState: &ResourceState{
			resources: make(map[string]*models.Resource),
		},
		storage:     storage,
		compression: &GzipCompression{threshold: 1024},
		serializer:  &JSONSerializer{},
		metrics:     stateMetrics,
	}

	// Initialize change detector
	discovery.changeDetector = &ChangeDetector{
		config: &ChangeDetectionConfig{
			Algorithm:         config.HashAlgorithm,
			IgnoreFields:      config.IgnoreFields,
			SensitiveFields:   config.SensitiveFields,
			Sensitivity:       config.ChangeSensitivity,
			CacheResults:      true,
			ParallelDetection: true,
			Workers:           4,
		},
		hashCache:   make(map[string]string),
		changeCache: make(map[string]*ChangeSet),
		metrics:     changeMetrics,
	}

	// Initialize delta processor
	discovery.deltaProcessor = &DeltaProcessor{
		batchSize: config.DeltaBatchSize,
		timeout:   config.DeltaTimeout,
		metrics:   deltaMetrics,
	}

	// Initialize comparison engine
	discovery.comparisonEngine = &ComparisonEngine{
		config: &ComparisonConfig{
			Workers:      config.ComparisonWorkers,
			QueueSize:    1000,
			DeepCompare:  config.DeepCompare,
			IgnoreFields: config.IgnoreFields,
			CacheResults: config.CacheComparisons,
		},
		workers:     config.ComparisonWorkers,
		workQueue:   make(chan ComparisonTask, 1000),
		resultQueue: make(chan ComparisonResult, 1000),
		metrics:     comparisonMetrics,
	}

	// Start background operations
	discovery.snapshotTicker = time.NewTicker(config.SnapshotInterval)
	discovery.cleanupTicker = time.NewTicker(time.Hour)

	go discovery.backgroundSnapshots()
	go discovery.backgroundCleanup()

	return discovery
}

// DiscoverChanges performs incremental discovery and returns detected changes
func (id *IncrementalDiscovery) DiscoverChanges(ctx context.Context, resources []*models.Resource) (*ChangeSet, error) {
	start := time.Now()

	// Create new snapshot
	snapshot, err := id.createSnapshot(resources)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Get previous snapshot for comparison
	var previousSnapshot *ResourceSnapshot
	if id.previousSnapshot != nil {
		previousSnapshot = id.previousSnapshot
	} else {
		// Try to load the latest snapshot from storage
		snapshots, err := id.stateManager.storage.ListSnapshots()
		if err == nil && len(snapshots) > 0 {
			// Sort snapshots by timestamp (newest first)
			sort.Strings(snapshots)
			if len(snapshots) > 0 {
				previousSnapshot, _ = id.stateManager.storage.LoadSnapshot(snapshots[len(snapshots)-1])
			}
		}
	}

	var changeSet *ChangeSet
	if previousSnapshot != nil {
		// Detect changes between snapshots
		changeSet, err = id.changeDetector.DetectChanges(previousSnapshot, snapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to detect changes: %w", err)
		}
	} else {
		// First discovery - all resources are new
		changeSet = &ChangeSet{
			ID:            generateChangeSetID(),
			ToSnapshot:    snapshot.ID,
			Timestamp:     time.Now(),
			Changes:       make([]ResourceChange, 0, len(resources)),
			DetectionTime: time.Since(start),
			ResourceCount: len(resources),
		}

		for _, resource := range resources {
			changeSet.Changes = append(changeSet.Changes, ResourceChange{
				ResourceID:   resource.ID,
				ResourceType: resource.Type,
				ChangeType:   ChangeTypeAdded,
				Timestamp:    time.Now(),
				Severity:     SeverityLow,
				Checksum:     id.calculateResourceHash(resource),
			})
		}
	}

	// Update state
	id.mu.Lock()
	id.previousSnapshot = id.currentSnapshot
	id.currentSnapshot = snapshot
	id.lastDiscovery = time.Now()
	id.mu.Unlock()

	// Save snapshot
	if err := id.stateManager.SaveSnapshot(snapshot); err != nil {
		// Log error but don't fail the discovery
		fmt.Printf("Warning: failed to save snapshot: %v\n", err)
	}

	// Process deltas
	if len(changeSet.Changes) > 0 {
		go id.deltaProcessor.ProcessDeltas(changeSet.Changes)
	}

	// Update metrics
	id.updateMetrics(changeSet)

	return changeSet, nil
}

// createSnapshot creates a new resource snapshot
func (id *IncrementalDiscovery) createSnapshot(resources []*models.Resource) (*ResourceSnapshot, error) {
	snapshot := &ResourceSnapshot{
		ID:             generateSnapshotID(),
		Timestamp:      time.Now(),
		Resources:      make(map[string]*models.Resource),
		Metadata:       make(map[string]interface{}),
		Version:        1,
		TotalResources: len(resources),
		ByProvider:     make(map[string]int),
		ByRegion:       make(map[string]int),
		ByType:         make(map[string]int),
	}

	// Add resources to snapshot
	for _, resource := range resources {
		snapshot.Resources[resource.ID] = resource
		snapshot.ByProvider[resource.Provider]++
		snapshot.ByRegion[resource.Region]++
		snapshot.ByType[resource.Type]++
	}

	// Calculate checksum
	snapshot.Checksum = id.calculateSnapshotHash(snapshot)

	// Calculate size
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize snapshot: %w", err)
	}
	snapshot.Size = int64(len(data))

	id.metrics.snapshotsCreated.Inc()
	id.metrics.snapshotSize.Observe(float64(snapshot.Size))

	return snapshot, nil
}

// DetectChanges detects changes between two snapshots
func (cd *ChangeDetector) DetectChanges(oldSnapshot, newSnapshot *ResourceSnapshot) (*ChangeSet, error) {
	start := time.Now()
	defer func() {
		cd.metrics.detectionTime.Observe(time.Since(start).Seconds())
	}()

	changeSet := &ChangeSet{
		ID:            generateChangeSetID(),
		FromSnapshot:  oldSnapshot.ID,
		ToSnapshot:    newSnapshot.ID,
		Timestamp:     time.Now(),
		Changes:       make([]ResourceChange, 0),
		DetectionTime: time.Since(start),
		ResourceCount: len(newSnapshot.Resources),
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s-%s", oldSnapshot.ID, newSnapshot.ID)
	cd.mu.RLock()
	if cached, exists := cd.changeCache[cacheKey]; exists {
		cd.mu.RUnlock()
		cd.metrics.cacheHits.Inc()
		return cached, nil
	}
	cd.mu.RUnlock()
	cd.metrics.cacheMisses.Inc()

	// Detect added resources
	for resourceID, resource := range newSnapshot.Resources {
		if _, exists := oldSnapshot.Resources[resourceID]; !exists {
			changeSet.Changes = append(changeSet.Changes, ResourceChange{
				ResourceID:   resourceID,
				ResourceType: resource.Type,
				ChangeType:   ChangeTypeAdded,
				Timestamp:    time.Now(),
				Severity:     cd.calculateChangeSeverity(ChangeTypeAdded, nil, resource),
				Checksum:     cd.calculateResourceHash(resource),
			})
		}
	}

	// Detect removed resources
	for resourceID, resource := range oldSnapshot.Resources {
		if _, exists := newSnapshot.Resources[resourceID]; !exists {
			changeSet.Changes = append(changeSet.Changes, ResourceChange{
				ResourceID:   resourceID,
				ResourceType: resource.Type,
				ChangeType:   ChangeTypeRemoved,
				Timestamp:    time.Now(),
				Severity:     cd.calculateChangeSeverity(ChangeTypeRemoved, resource, nil),
				Checksum:     cd.calculateResourceHash(resource),
			})
		}
	}

	// Detect modified resources
	for resourceID, newResource := range newSnapshot.Resources {
		if oldResource, exists := oldSnapshot.Resources[resourceID]; exists {
			if changes := cd.compareResources(oldResource, newResource); len(changes) > 0 {
				for _, change := range changes {
					change.ResourceID = resourceID
					change.ResourceType = newResource.Type
					change.Timestamp = time.Now()
					changeSet.Changes = append(changeSet.Changes, change)
				}
			}
		}
	}

	// Calculate summary
	changeSet.Summary = cd.calculateChangeSummary(changeSet.Changes)

	// Cache result
	if cd.config.CacheResults {
		cd.mu.Lock()
		cd.changeCache[cacheKey] = changeSet
		cd.mu.Unlock()
	}

	cd.metrics.changesDetected.Add(float64(len(changeSet.Changes)))

	return changeSet, nil
}

// compareResources compares two resources and returns detected changes
func (cd *ChangeDetector) compareResources(oldResource, newResource *models.Resource) []ResourceChange {
	var changes []ResourceChange

	// Compare basic fields
	if oldResource.Name != newResource.Name {
		changes = append(changes, ResourceChange{
			ChangeType: ChangeTypeModified,
			Field:      "name",
			OldValue:   oldResource.Name,
			NewValue:   newResource.Name,
			Severity:   SeverityMedium,
		})
	}

	if oldResource.State != newResource.State {
		changes = append(changes, ResourceChange{
			ChangeType: ChangeTypeModified,
			Field:      "state",
			OldValue:   oldResource.State,
			NewValue:   newResource.State,
			Severity:   SeverityHigh,
		})
	}

	if oldResource.Status != newResource.Status {
		changes = append(changes, ResourceChange{
			ChangeType: ChangeTypeModified,
			Field:      "status",
			OldValue:   oldResource.Status,
			NewValue:   newResource.Status,
			Severity:   SeverityMedium,
		})
	}

	// Compare tags
	oldTags := make(map[string]interface{})
	newTags := make(map[string]interface{})
	for k, v := range oldResource.Tags {
		oldTags[k] = v
	}
	for k, v := range newResource.Tags {
		newTags[k] = v
	}
	tagChanges := cd.compareMaps(oldTags, newTags, "tags")
	changes = append(changes, tagChanges...)

	// Compare attributes
	attrChanges := cd.compareMaps(oldResource.Attributes, newResource.Attributes, "attributes")
	changes = append(changes, attrChanges...)

	// Compare properties
	propChanges := cd.compareMaps(oldResource.Properties, newResource.Properties, "properties")
	changes = append(changes, propChanges...)

	return changes
}

// compareMaps compares two maps and returns detected changes
func (cd *ChangeDetector) compareMaps(oldMap, newMap map[string]interface{}, fieldPrefix string) []ResourceChange {
	var changes []ResourceChange

	// Check for added/modified keys
	for key, newValue := range newMap {
		fieldName := fmt.Sprintf("%s.%s", fieldPrefix, key)

		// Skip ignored fields
		if cd.isIgnoredField(fieldName) {
			continue
		}

		if oldValue, exists := oldMap[key]; exists {
			// Key exists, check if value changed
			if !cd.valuesEqual(oldValue, newValue) {
				severity := SeverityLow
				if cd.isSensitiveField(fieldName) {
					severity = SeverityHigh
				}

				changes = append(changes, ResourceChange{
					ChangeType: ChangeTypeModified,
					Field:      fieldName,
					OldValue:   oldValue,
					NewValue:   newValue,
					Severity:   severity,
				})
			}
		} else {
			// Key added
			changes = append(changes, ResourceChange{
				ChangeType: ChangeTypeAdded,
				Field:      fieldName,
				NewValue:   newValue,
				Severity:   SeverityLow,
			})
		}
	}

	// Check for removed keys
	for key, oldValue := range oldMap {
		fieldName := fmt.Sprintf("%s.%s", fieldPrefix, key)

		// Skip ignored fields
		if cd.isIgnoredField(fieldName) {
			continue
		}

		if _, exists := newMap[key]; !exists {
			// Key removed
			changes = append(changes, ResourceChange{
				ChangeType: ChangeTypeRemoved,
				Field:      fieldName,
				OldValue:   oldValue,
				Severity:   SeverityMedium,
			})
		}
	}

	return changes
}

// ProcessDeltas processes a batch of resource changes
func (dp *DeltaProcessor) ProcessDeltas(deltas []ResourceChange) error {
	start := time.Now()
	defer func() {
		dp.metrics.processingTime.Observe(time.Since(start).Seconds())
		dp.metrics.batchesProcessed.Inc()
		dp.metrics.batchSize.Observe(float64(len(deltas)))
	}()

	dp.mu.Lock()
	defer dp.mu.Unlock()

	// Add deltas to pending batch
	dp.deltas = append(dp.deltas, deltas...)
	dp.metrics.deltasProcessed.Add(float64(len(deltas)))

	// Process if batch is full or timeout reached
	if len(dp.deltas) >= dp.batchSize || time.Since(dp.lastFlush) > dp.timeout {
		return dp.flushDeltas()
	}

	return nil
}

// flushDeltas processes all pending deltas
func (dp *DeltaProcessor) flushDeltas() error {
	if len(dp.deltas) == 0 {
		return nil
	}

	// Process the batch
	if dp.processor != nil {
		if err := dp.processor(dp.deltas); err != nil {
			return fmt.Errorf("failed to process delta batch: %w", err)
		}
	}

	// Clear batch
	dp.deltas = dp.deltas[:0]
	dp.lastFlush = time.Now()

	return nil
}

// SaveSnapshot saves a snapshot to storage
func (sm *StateManager) SaveSnapshot(snapshot *ResourceSnapshot) error {
	start := time.Now()
	defer sm.metrics.snapshotsSaved.Inc()

	// Compress if enabled
	if sm.compression != nil {
		data, err := sm.serializer.Serialize(snapshot)
		if err != nil {
			sm.metrics.storageErrors.Inc()
			return fmt.Errorf("failed to serialize snapshot: %w", err)
		}

		if sm.compression.ShouldCompress(data) {
			compressed, err := sm.compression.Compress(data)
			if err == nil {
				snapshot.Compressed = true
				// Update size with compressed size
				snapshot.Size = int64(len(compressed))
			}
		}
	}

	if err := sm.storage.SaveSnapshot(snapshot); err != nil {
		sm.metrics.storageErrors.Inc()
		return fmt.Errorf("failed to save snapshot to storage: %w", err)
	}

	// Cache snapshot
	sm.mu.Lock()
	sm.snapshots[snapshot.ID] = snapshot
	sm.mu.Unlock()

	fmt.Printf("Snapshot %s saved in %v\n", snapshot.ID, time.Since(start))
	return nil
}

// Background operations
func (id *IncrementalDiscovery) backgroundSnapshots() {
	for {
		select {
		case <-id.ctx.Done():
			return
		case <-id.snapshotTicker.C:
			// Periodic snapshot creation would be implemented here
			// This would trigger discovery of current resources
		}
	}
}

func (id *IncrementalDiscovery) backgroundCleanup() {
	for {
		select {
		case <-id.ctx.Done():
			return
		case <-id.cleanupTicker.C:
			id.cleanupOldSnapshots()
		}
	}
}

func (id *IncrementalDiscovery) cleanupOldSnapshots() {
	// Implementation would clean up old snapshots based on retention policy
	snapshots, err := id.stateManager.storage.ListSnapshots()
	if err != nil {
		return
	}

	// Keep only the most recent snapshots
	if len(snapshots) > id.config.MaxSnapshots {
		toDelete := snapshots[:len(snapshots)-id.config.MaxSnapshots]
		for _, snapshotID := range toDelete {
			id.stateManager.storage.DeleteSnapshot(snapshotID)
		}
	}
}

// Utility functions
func (id *IncrementalDiscovery) calculateResourceHash(resource *models.Resource) string {
	data, _ := json.Marshal(resource)
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

func (id *IncrementalDiscovery) calculateSnapshotHash(snapshot *ResourceSnapshot) string {
	// Create a consistent hash of the snapshot content
	var resourceHashes []string
	for _, resource := range snapshot.Resources {
		resourceHashes = append(resourceHashes, id.calculateResourceHash(resource))
	}
	sort.Strings(resourceHashes)

	data := fmt.Sprintf("%v", resourceHashes)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (cd *ChangeDetector) calculateResourceHash(resource *models.Resource) string {
	data, _ := json.Marshal(resource)
	hash := md5.Sum(data)
	cd.metrics.hashCalculations.Inc()
	return hex.EncodeToString(hash[:])
}

func (cd *ChangeDetector) calculateChangeSeverity(changeType ChangeType, oldResource, newResource *models.Resource) ChangeSeverity {
	switch changeType {
	case ChangeTypeAdded:
		return SeverityLow
	case ChangeTypeRemoved:
		return SeverityHigh
	case ChangeTypeModified:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

func (cd *ChangeDetector) calculateChangeSummary(changes []ResourceChange) ChangeSummary {
	summary := ChangeSummary{
		TotalChanges:   len(changes),
		ByType:         make(map[ChangeType]int),
		BySeverity:     make(map[ChangeSeverity]int),
		ByResourceType: make(map[string]int),
		ByProvider:     make(map[string]int),
		ByRegion:       make(map[string]int),
	}

	for _, change := range changes {
		summary.ByType[change.ChangeType]++
		summary.BySeverity[change.Severity]++
		summary.ByResourceType[change.ResourceType]++
	}

	return summary
}

func (cd *ChangeDetector) isIgnoredField(field string) bool {
	for _, ignored := range cd.config.IgnoreFields {
		if field == ignored {
			return true
		}
	}
	return false
}

func (cd *ChangeDetector) isSensitiveField(field string) bool {
	for _, sensitive := range cd.config.SensitiveFields {
		if field == sensitive {
			return true
		}
	}
	return false
}

func (cd *ChangeDetector) valuesEqual(a, b interface{}) bool {
	// Simplified comparison - in production, use deep comparison
	aData, _ := json.Marshal(a)
	bData, _ := json.Marshal(b)
	return string(aData) == string(bData)
}

func (id *IncrementalDiscovery) updateMetrics(changeSet *ChangeSet) {
	for _, change := range changeSet.Changes {
		switch change.ChangeType {
		case ChangeTypeAdded:
			id.metrics.resourcesAdded.Inc()
		case ChangeTypeRemoved:
			id.metrics.resourcesRemoved.Inc()
		case ChangeTypeModified:
			id.metrics.resourcesModified.Inc()
		}
	}

	id.metrics.changesDetected.Add(float64(len(changeSet.Changes)))
	id.metrics.deltaOperations.Inc()
}

// GetStats returns discovery statistics
func (id *IncrementalDiscovery) GetStats() DiscoveryStats {
	id.mu.RLock()
	defer id.mu.RUnlock()

	stats := DiscoveryStats{
		LastDiscovery:    id.lastDiscovery,
		SnapshotCount:    len(id.stateManager.snapshots),
		CurrentResources: 0,
	}

	if id.currentSnapshot != nil {
		stats.CurrentResources = id.currentSnapshot.TotalResources
		stats.CurrentSnapshot = id.currentSnapshot.ID
	}

	if id.previousSnapshot != nil {
		stats.PreviousSnapshot = id.previousSnapshot.ID
	}

	return stats
}

// DiscoveryStats holds discovery statistics
type DiscoveryStats struct {
	LastDiscovery    time.Time
	SnapshotCount    int
	CurrentResources int
	CurrentSnapshot  string
	PreviousSnapshot string
}

// Close shuts down the incremental discovery
func (id *IncrementalDiscovery) Close() error {
	id.cancel()
	id.snapshotTicker.Stop()
	id.cleanupTicker.Stop()

	// Flush any pending deltas
	return id.deltaProcessor.flushDeltas()
}

// Utility functions for ID generation
func generateSnapshotID() string {
	return fmt.Sprintf("snapshot_%d", time.Now().UnixNano())
}

func generateChangeSetID() string {
	return fmt.Sprintf("changeset_%d", time.Now().UnixNano())
}
