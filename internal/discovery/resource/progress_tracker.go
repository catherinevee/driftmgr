package resource

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ProgressTracker tracks discovery progress across providers, regions, and services
type ProgressTracker struct {
	mu               sync.RWMutex
	providers        []string
	regions          []string
	services         []string
	totalResources   int64
	processedResources int64
	startTime        time.Time
	providerProgress map[string]*ProviderProgress
	regionProgress   map[string]*RegionProgress
	serviceProgress  map[string]*ServiceProgress
	errors           []error
	completed        bool
}

// ProviderProgress tracks progress for a specific provider
type ProviderProgress struct {
	Name            string
	TotalRegions    int
	CompletedRegions int
	TotalResources  int64
	DiscoveredResources int64
	StartTime       time.Time
	EndTime         *time.Time
	Errors          []error
}

// RegionProgress tracks progress for a specific region
type RegionProgress struct {
	Provider        string
	Region          string
	TotalServices   int
	CompletedServices int
	TotalResources  int64
	DiscoveredResources int64
	StartTime       time.Time
	EndTime         *time.Time
	Errors          []error
}

// ServiceProgress tracks progress for a specific service
type ServiceProgress struct {
	Provider        string
	Region          string
	Service         string
	TotalResources  int64
	DiscoveredResources int64
	StartTime       time.Time
	EndTime         *time.Time
	Errors          []error
	LastActivity    time.Time
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(providers, regions, services []string) *ProgressTracker {
	pt := &ProgressTracker{
		providers:        providers,
		regions:          regions,
		services:         services,
		startTime:        time.Now(),
		providerProgress: make(map[string]*ProviderProgress),
		regionProgress:   make(map[string]*RegionProgress),
		serviceProgress:  make(map[string]*ServiceProgress),
		errors:           make([]error, 0),
	}

	// Initialize provider progress
	for _, provider := range providers {
		pt.providerProgress[provider] = &ProviderProgress{
			Name:         provider,
			TotalRegions: len(regions),
			StartTime:    time.Now(),
			Errors:       make([]error, 0),
		}
	}

	// Initialize region progress
	for _, provider := range providers {
		for _, region := range regions {
			key := fmt.Sprintf("%s:%s", provider, region)
			pt.regionProgress[key] = &RegionProgress{
				Provider:      provider,
				Region:        region,
				TotalServices: len(services),
				StartTime:     time.Now(),
				Errors:        make([]error, 0),
			}
		}
	}

	// Initialize service progress
	for _, provider := range providers {
		for _, region := range regions {
			for _, service := range services {
				key := fmt.Sprintf("%s:%s:%s", provider, region, service)
				pt.serviceProgress[key] = &ServiceProgress{
					Provider:     provider,
					Region:       region,
					Service:      service,
					StartTime:    time.Now(),
					Errors:       make([]error, 0),
					LastActivity: time.Now(),
				}
			}
		}
	}

	return pt
}

// UpdateProviderProgress updates progress for a provider
func (pt *ProgressTracker) UpdateProviderProgress(provider string, completedRegions int, resources int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if progress, exists := pt.providerProgress[provider]; exists {
		progress.CompletedRegions = completedRegions
		progress.DiscoveredResources = resources
		if completedRegions >= progress.TotalRegions {
			now := time.Now()
			progress.EndTime = &now
		}
	}
}

// UpdateRegionProgress updates progress for a region
func (pt *ProgressTracker) UpdateRegionProgress(provider, region string, completedServices int, resources int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	key := fmt.Sprintf("%s:%s", provider, region)
	if progress, exists := pt.regionProgress[key]; exists {
		progress.CompletedServices = completedServices
		progress.DiscoveredResources = resources
		if completedServices >= progress.TotalServices {
			now := time.Now()
			progress.EndTime = &now
		}
	}
}

// UpdateServiceProgress updates progress for a service
func (pt *ProgressTracker) UpdateServiceProgress(provider, region, service string, resources int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", provider, region, service)
	if progress, exists := pt.serviceProgress[key]; exists {
		progress.DiscoveredResources = resources
		progress.LastActivity = time.Now()
	}
}

// IncrementResources increments the resource counters
func (pt *ProgressTracker) IncrementResources(count int64) {
	atomic.AddInt64(&pt.processedResources, count)
}

// AddError adds an error to the tracker
func (pt *ProgressTracker) AddError(err error) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.errors = append(pt.errors, err)
}

// GetProgress returns current progress statistics
func (pt *ProgressTracker) GetProgress() map[string]interface{} {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	elapsed := time.Since(pt.startTime)
	resourcesPerSecond := float64(pt.processedResources) / elapsed.Seconds()

	// Calculate completion percentages
	totalProviderProgress := 0.0
	activeProviders := 0
	for _, progress := range pt.providerProgress {
		if progress.TotalRegions > 0 {
			activeProviders++
			totalProviderProgress += float64(progress.CompletedRegions) / float64(progress.TotalRegions)
		}
	}

	overallProgress := 0.0
	if activeProviders > 0 {
		overallProgress = (totalProviderProgress / float64(activeProviders)) * 100
	}

	return map[string]interface{}{
		"elapsed_time":        elapsed.String(),
		"total_resources":     pt.totalResources,
		"processed_resources": pt.processedResources,
		"resources_per_second": resourcesPerSecond,
		"overall_progress":    overallProgress,
		"errors_count":        len(pt.errors),
		"providers_active":    activeProviders,
		"completed":           pt.completed,
	}
}

// GetProviderProgress returns progress for a specific provider
func (pt *ProgressTracker) GetProviderProgress(provider string) *ProviderProgress {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.providerProgress[provider]
}

// GetRegionProgress returns progress for a specific region
func (pt *ProgressTracker) GetRegionProgress(provider, region string) *RegionProgress {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", provider, region)
	return pt.regionProgress[key]
}

// GetServiceProgress returns progress for a specific service
func (pt *ProgressTracker) GetServiceProgress(provider, region, service string) *ServiceProgress {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	key := fmt.Sprintf("%s:%s:%s", provider, region, service)
	return pt.serviceProgress[key]
}

// MarkCompleted marks the discovery as completed
func (pt *ProgressTracker) MarkCompleted() {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.completed = true
}

// IsCompleted returns whether discovery is completed
func (pt *ProgressTracker) IsCompleted() bool {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.completed
}

// GetErrors returns all tracked errors
func (pt *ProgressTracker) GetErrors() []error {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	errorsCopy := make([]error, len(pt.errors))
	copy(errorsCopy, pt.errors)
	return errorsCopy
}

// Reset resets the progress tracker
func (pt *ProgressTracker) Reset() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.totalResources = 0
	pt.processedResources = 0
	pt.startTime = time.Now()
	pt.errors = make([]error, 0)
	pt.completed = false

	// Reset all progress tracking
	for _, progress := range pt.providerProgress {
		progress.CompletedRegions = 0
		progress.DiscoveredResources = 0
		progress.StartTime = time.Now()
		progress.EndTime = nil
		progress.Errors = make([]error, 0)
	}

	for _, progress := range pt.regionProgress {
		progress.CompletedServices = 0
		progress.DiscoveredResources = 0
		progress.StartTime = time.Now()
		progress.EndTime = nil
		progress.Errors = make([]error, 0)
	}

	for _, progress := range pt.serviceProgress {
		progress.DiscoveredResources = 0
		progress.StartTime = time.Now()
		progress.EndTime = nil
		progress.Errors = make([]error, 0)
		progress.LastActivity = time.Now()
	}
}

// GetSummary returns a summary of the discovery progress
func (pt *ProgressTracker) GetSummary() string {
	progress := pt.GetProgress()
	return fmt.Sprintf(
		"Discovery Progress: %.1f%% | Resources: %d | Rate: %.1f/sec | Elapsed: %s | Errors: %d",
		progress["overall_progress"].(float64),
		progress["processed_resources"].(int64),
		progress["resources_per_second"].(float64),
		progress["elapsed_time"].(string),
		progress["errors_count"].(int),
	)
}