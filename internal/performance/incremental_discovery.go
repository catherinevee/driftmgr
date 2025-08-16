package performance

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// IncrementalDiscovery provides incremental resource discovery capabilities
type IncrementalDiscovery struct {
	stateManager    *DiscoveryStateManager
	changeDetector  *ChangeDetector
	deltaProcessor  *DeltaProcessor
	config          *IncrementalConfig
	mu              sync.RWMutex
}

// IncrementalConfig defines incremental discovery behavior
type IncrementalConfig struct {
	Enabled           bool          `yaml:"enabled"`
	StateFile         string        `yaml:"state_file"`
	ChangeThreshold   float64       `yaml:"change_threshold"`
	MaxDeltaSize      int           `yaml:"max_delta_size"`
	CompressionEnabled bool         `yaml:"compression_enabled"`
	BackupEnabled     bool          `yaml:"backup_enabled"`
	RetentionDays     int           `yaml:"retention_days"`
}

// DiscoveryState represents the state of discovered resources
type DiscoveryState struct {
	Version     string                     `json:"version"`
	Timestamp   time.Time                  `json:"timestamp"`
	Resources   map[string]*ResourceState  `json:"resources"`
	Metadata    map[string]interface{}     `json:"metadata"`
	Checksums   map[string]string          `json:"checksums"`
}

// ResourceState represents the state of a single resource
type ResourceState struct {
	Resource    *models.Resource `json:"resource"`
	LastSeen    time.Time        `json:"last_seen"`
	Checksum    string           `json:"checksum"`
	ChangeCount int              `json:"change_count"`
	Tags        map[string]string `json:"tags"`
}

// DiscoveryDelta represents changes in resources
type DiscoveryDelta struct {
	Added       []*models.Resource `json:"added"`
	Modified    []*models.Resource `json:"modified"`
	Deleted     []*models.Resource `json:"deleted"`
	Unchanged   []*models.Resource `json:"unchanged"`
	Timestamp   time.Time          `json:"timestamp"`
	ChangeCount int                `json:"change_count"`
}

// NewIncrementalDiscovery creates a new incremental discovery system
func NewIncrementalDiscovery(config *IncrementalConfig) *IncrementalDiscovery {
	if config == nil {
		config = &IncrementalConfig{
			Enabled:           true,
			StateFile:         "discovery-state.json",
			ChangeThreshold:   0.1,
			MaxDeltaSize:      1000,
			CompressionEnabled: true,
			BackupEnabled:     true,
			RetentionDays:     30,
		}
	}

	return &IncrementalDiscovery{
		stateManager:   NewDiscoveryStateManager(config),
		changeDetector: NewChangeDetector(config.ChangeThreshold),
		deltaProcessor: NewDeltaProcessor(config.MaxDeltaSize),
		config:         config,
	}
}

// DiscoverIncremental performs incremental resource discovery
func (id *IncrementalDiscovery) DiscoverIncremental(
	ctx context.Context,
	discoverer func(context.Context) ([]*models.Resource, error),
) (*DiscoveryDelta, error) {
	id.mu.Lock()
	defer id.mu.Unlock()

	// Load previous state
	previousState, err := id.stateManager.LoadState()
	if err != nil {
		// If no previous state, perform full discovery
		return id.performFullDiscovery(ctx, discoverer)
	}

	// Perform current discovery
	currentResources, err := discoverer(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover current resources: %w", err)
	}

	// Detect changes
	delta := id.changeDetector.DetectChanges(previousState, currentResources)

	// Process delta
	processedDelta := id.deltaProcessor.ProcessDelta(delta)

	// Update state
	newState := id.createNewState(currentResources, processedDelta)
	err = id.stateManager.SaveState(newState)
	if err != nil {
		return nil, fmt.Errorf("failed to save discovery state: %w", err)
	}

	return processedDelta, nil
}

// performFullDiscovery performs a full discovery when no previous state exists
func (id *IncrementalDiscovery) performFullDiscovery(
	ctx context.Context,
	discoverer func(context.Context) ([]*models.Resource, error),
) (*DiscoveryDelta, error) {
	resources, err := discoverer(ctx)
	if err != nil {
		return nil, err
	}

	// Create initial state
	state := id.createNewState(resources, nil)
	err = id.stateManager.SaveState(state)
	if err != nil {
		return nil, fmt.Errorf("failed to save initial state: %w", err)
	}

	// Return delta with all resources as added
	delta := &DiscoveryDelta{
		Added:       resources,
		Modified:    []*models.Resource{},
		Deleted:     []*models.Resource{},
		Unchanged:   []*models.Resource{},
		Timestamp:   time.Now(),
		ChangeCount: len(resources),
	}

	return delta, nil
}

// createNewState creates a new discovery state from current resources
func (id *IncrementalDiscovery) createNewState(
	resources []*models.Resource,
	delta *DiscoveryDelta,
) *DiscoveryState {
	state := &DiscoveryState{
		Version:   "1.0",
		Timestamp: time.Now(),
		Resources: make(map[string]*ResourceState),
		Metadata:  make(map[string]interface{}),
		Checksums: make(map[string]string),
	}

	// Add metadata
	state.Metadata["total_resources"] = len(resources)
	state.Metadata["delta_size"] = 0
	if delta != nil {
		state.Metadata["delta_size"] = delta.ChangeCount
	}

	// Process each resource
	for _, resource := range resources {
		key := fmt.Sprintf("%s:%s:%s", resource.Provider, resource.Region, resource.ID)
		checksum := id.calculateResourceChecksum(resource)

		resourceState := &ResourceState{
			Resource:    resource,
			LastSeen:    time.Now(),
			Checksum:    checksum,
			ChangeCount: 0,
			Tags:        resource.Tags,
		}

		// Update change count if this was a modified resource
		if delta != nil {
			for _, modified := range delta.Modified {
				if modified.ID == resource.ID && modified.Provider == resource.Provider {
					resourceState.ChangeCount++
					break
				}
			}
		}

		state.Resources[key] = resourceState
		state.Checksums[key] = checksum
	}

	return state
}

// calculateResourceChecksum calculates a checksum for a resource
func (id *IncrementalDiscovery) calculateResourceChecksum(resource *models.Resource) string {
	// Create a stable representation of the resource
	data := map[string]interface{}{
		"id":       resource.ID,
		"type":     resource.Type,
		"provider": resource.Provider,
		"region":   resource.Region,
		"tags":     resource.Tags,
		"status":   resource.Status,
		"created":  resource.CreatedAt,
		"modified": resource.LastModified,
	}

	// Convert to JSON for consistent hashing
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

// GetChangeStatistics returns statistics about changes
func (id *IncrementalDiscovery) GetChangeStatistics() map[string]interface{} {
	id.mu.RLock()
	defer id.mu.RUnlock()

	state, err := id.stateManager.LoadState()
	if err != nil {
		return map[string]interface{}{
			"error": "Failed to load state",
		}
	}

	stats := map[string]interface{}{
		"total_resources": len(state.Resources),
		"last_update":     state.Timestamp,
		"version":         state.Version,
	}

	// Calculate change statistics
	totalChanges := 0
	recentChanges := 0
	last24h := time.Now().Add(-24 * time.Hour)

	for _, resourceState := range state.Resources {
		totalChanges += resourceState.ChangeCount
		if resourceState.LastSeen.After(last24h) {
			recentChanges++
		}
	}

	stats["total_changes"] = totalChanges
	stats["recent_changes"] = recentChanges
	stats["change_rate"] = float64(totalChanges) / float64(len(state.Resources))

	return stats
}

// CleanupOldStates cleans up old discovery states
func (id *IncrementalDiscovery) CleanupOldStates() error {
	return id.stateManager.CleanupOldStates(id.config.RetentionDays)
}

// DiscoveryStateManager manages discovery state persistence
type DiscoveryStateManager struct {
	config     *IncrementalConfig
	compressor *CompressionManager
}

// NewDiscoveryStateManager creates a new state manager
func NewDiscoveryStateManager(config *IncrementalConfig) *DiscoveryStateManager {
	return &DiscoveryStateManager{
		config:     config,
		compressor: NewCompressionManager(),
	}
}

// LoadState loads the discovery state from storage
func (dsm *DiscoveryStateManager) LoadState() (*DiscoveryState, error) {
	// Implementation would read from file/database
	// For now, return empty state
	return &DiscoveryState{
		Version:   "1.0",
		Timestamp: time.Now(),
		Resources: make(map[string]*ResourceState),
		Metadata:  make(map[string]interface{}),
		Checksums: make(map[string]string),
	}, nil
}

// SaveState saves the discovery state to storage
func (dsm *DiscoveryStateManager) SaveState(state *DiscoveryState) error {
	// Implementation would write to file/database
	// For now, just return success
	return nil
}

// CleanupOldStates removes old discovery states
func (dsm *DiscoveryStateManager) CleanupOldStates(retentionDays int) error {
	// Implementation would remove old state files
	// For now, just return success
	return nil
}

// ChangeDetector detects changes between resource states
type ChangeDetector struct {
	threshold float64
}

// NewChangeDetector creates a new change detector
func NewChangeDetector(threshold float64) *ChangeDetector {
	return &ChangeDetector{
		threshold: threshold,
	}
}

// DetectChanges detects changes between previous and current resources
func (cd *ChangeDetector) DetectChanges(
	previousState *DiscoveryState,
	currentResources []*models.Resource,
) *DiscoveryDelta {
	delta := &DiscoveryDelta{
		Added:       []*models.Resource{},
		Modified:    []*models.Resource{},
		Deleted:     []*models.Resource{},
		Unchanged:   []*models.Resource{},
		Timestamp:   time.Now(),
		ChangeCount: 0,
	}

	// Create map of current resources for efficient lookup
	currentMap := make(map[string]*models.Resource)
	for _, resource := range currentResources {
		key := fmt.Sprintf("%s:%s:%s", resource.Provider, resource.Region, resource.ID)
		currentMap[key] = resource
	}

	// Check for deleted and modified resources
	for key, resourceState := range previousState.Resources {
		if currentResource, exists := currentMap[key]; exists {
			// Resource still exists, check if modified
			currentChecksum := cd.calculateResourceChecksum(currentResource)
			if currentChecksum != resourceState.Checksum {
				delta.Modified = append(delta.Modified, currentResource)
				delta.ChangeCount++
			} else {
				delta.Unchanged = append(delta.Unchanged, currentResource)
			}
			delete(currentMap, key) // Remove from current map
		} else {
			// Resource was deleted
			delta.Deleted = append(delta.Deleted, resourceState.Resource)
			delta.ChangeCount++
		}
	}

	// Remaining resources in currentMap are new
	for _, resource := range currentMap {
		delta.Added = append(delta.Added, resource)
		delta.ChangeCount++
	}

	return delta
}

// calculateResourceChecksum calculates checksum for change detection
func (cd *ChangeDetector) calculateResourceChecksum(resource *models.Resource) string {
	data := map[string]interface{}{
		"id":       resource.ID,
		"type":     resource.Type,
		"provider": resource.Provider,
		"region":   resource.Region,
		"tags":     resource.Tags,
		"status":   resource.Status,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

// DeltaProcessor processes discovery deltas
type DeltaProcessor struct {
	maxDeltaSize int
}

// NewDeltaProcessor creates a new delta processor
func NewDeltaProcessor(maxDeltaSize int) *DeltaProcessor {
	return &DeltaProcessor{
		maxDeltaSize: maxDeltaSize,
	}
}

// ProcessDelta processes a discovery delta
func (dp *DeltaProcessor) ProcessDelta(delta *DiscoveryDelta) *DiscoveryDelta {
	// Limit delta size if necessary
	if delta.ChangeCount > dp.maxDeltaSize {
		// Prioritize changes: added > modified > deleted
		delta = dp.limitDeltaSize(delta)
	}

	return delta
}

// limitDeltaSize limits the delta size by prioritizing changes
func (dp *DeltaProcessor) limitDeltaSize(delta *DiscoveryDelta) *DiscoveryDelta {
	limitedDelta := &DiscoveryDelta{
		Timestamp:   delta.Timestamp,
		ChangeCount: 0,
	}

	// Add resources in priority order
	remaining := dp.maxDeltaSize

	// First, add all added resources
	if len(delta.Added) <= remaining {
		limitedDelta.Added = delta.Added
		remaining -= len(delta.Added)
	} else {
		limitedDelta.Added = delta.Added[:remaining]
		remaining = 0
	}

	// Then, add modified resources
	if remaining > 0 && len(delta.Modified) > 0 {
		if len(delta.Modified) <= remaining {
			limitedDelta.Modified = delta.Modified
			remaining -= len(delta.Modified)
		} else {
			limitedDelta.Modified = delta.Modified[:remaining]
			remaining = 0
		}
	}

	// Finally, add deleted resources
	if remaining > 0 && len(delta.Deleted) > 0 {
		if len(delta.Deleted) <= remaining {
			limitedDelta.Deleted = delta.Deleted
		} else {
			limitedDelta.Deleted = delta.Deleted[:remaining]
		}
	}

	limitedDelta.ChangeCount = len(limitedDelta.Added) + len(limitedDelta.Modified) + len(limitedDelta.Deleted)
	return limitedDelta
}
