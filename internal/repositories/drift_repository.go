package repositories

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/services"
)

// MemoryDriftRepository is an in-memory implementation of DriftRepository
type MemoryDriftRepository struct {
	results map[string]*models.DriftResult
	events  map[string][]*models.DriftEvent
	mu      sync.RWMutex
}

// NewMemoryDriftRepository creates a new in-memory drift repository
func NewMemoryDriftRepository() *MemoryDriftRepository {
	return &MemoryDriftRepository{
		results: make(map[string]*models.DriftResult),
		events:  make(map[string][]*models.DriftEvent),
	}
}

// Create creates a new drift result
func (r *MemoryDriftRepository) Create(ctx context.Context, result *models.DriftResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if result already exists
	if _, exists := r.results[result.ID]; exists {
		return fmt.Errorf("drift result with ID %s already exists", result.ID)
	}

	// Set timestamps
	now := time.Now()
	result.CreatedAt = now
	result.UpdatedAt = now

	// Store result
	r.results[result.ID] = result

	return nil
}

// GetByID retrieves a drift result by ID
func (r *MemoryDriftRepository) GetByID(ctx context.Context, id string) (*models.DriftResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result, exists := r.results[id]
	if !exists {
		return nil, fmt.Errorf("drift result with ID %s not found", id)
	}

	// Return a copy to prevent external modifications
	resultCopy := *result
	return &resultCopy, nil
}

// GetAll retrieves all drift results with optional filtering
func (r *MemoryDriftRepository) GetAll(ctx context.Context, filters services.DriftFilters) ([]*models.DriftResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.DriftResult

	for _, result := range r.results {
		// Apply filters
		if filters.Provider != "" && result.Provider != filters.Provider {
			continue
		}
		if filters.Status != "" && result.Status != filters.Status {
			continue
		}
		if !filters.StartDate.IsZero() && result.Timestamp.Before(filters.StartDate) {
			continue
		}
		if !filters.EndDate.IsZero() && result.Timestamp.After(filters.EndDate) {
			continue
		}

		// Return a copy to prevent external modifications
		resultCopy := *result
		results = append(results, &resultCopy)
	}

	// Apply pagination
	if filters.Offset > 0 && filters.Offset < len(results) {
		results = results[filters.Offset:]
	}
	if filters.Limit > 0 && filters.Limit < len(results) {
		results = results[:filters.Limit]
	}

	return results, nil
}

// Update updates an existing drift result
func (r *MemoryDriftRepository) Update(ctx context.Context, result *models.DriftResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if result exists
	if _, exists := r.results[result.ID]; !exists {
		return fmt.Errorf("drift result with ID %s not found", result.ID)
	}

	// Update timestamp
	result.UpdatedAt = time.Now()

	// Store updated result
	r.results[result.ID] = result

	return nil
}

// Delete deletes a drift result
func (r *MemoryDriftRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if result exists
	if _, exists := r.results[id]; !exists {
		return fmt.Errorf("drift result with ID %s not found", id)
	}

	// Delete result
	delete(r.results, id)

	return nil
}

// GetDriftHistory retrieves drift history for a resource
func (r *MemoryDriftRepository) GetDriftHistory(ctx context.Context, resourceID string) ([]*models.DriftEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events, exists := r.events[resourceID]
	if !exists {
		// Return empty slice if no history exists
		return []*models.DriftEvent{}, nil
	}

	// Return a copy to prevent external modifications
	eventsCopy := make([]*models.DriftEvent, len(events))
	for i, event := range events {
		eventCopy := *event
		eventsCopy[i] = &eventCopy
	}

	return eventsCopy, nil
}

// GetDriftSummary retrieves a summary of drift detection results
func (r *MemoryDriftRepository) GetDriftSummary(ctx context.Context, filters services.DriftFilters) (*services.DriftSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var totalResources int
	var driftedResources int
	driftsByProvider := make(map[string]int)
	driftsBySeverity := make(map[string]int)
	var recentDrifts []*models.DriftResult

	// Count resources and drifts
	for _, result := range r.results {
		// Apply filters
		if filters.Provider != "" && result.Provider != filters.Provider {
			continue
		}
		if filters.Status != "" && result.Status != filters.Status {
			continue
		}
		if !filters.StartDate.IsZero() && result.Timestamp.Before(filters.StartDate) {
			continue
		}
		if !filters.EndDate.IsZero() && result.Timestamp.After(filters.EndDate) {
			continue
		}

		totalResources++

		if result.DriftCount > 0 {
			driftedResources++

			// Count by provider
			if result.Provider != "" {
				driftsByProvider[result.Provider]++
			}

			// Count by severity - DriftSummary is a struct, not a map
			// For now, we'll use a default severity since the Summary field is not a map
			driftsBySeverity["medium"]++
		}

		// Collect recent drifts (last 10)
		if len(recentDrifts) < 10 {
			resultCopy := *result
			recentDrifts = append(recentDrifts, &resultCopy)
		}
	}

	// Calculate drift percentage
	var driftPercentage float64
	if totalResources > 0 {
		driftPercentage = float64(driftedResources) / float64(totalResources) * 100
	}

	summary := &services.DriftSummary{
		TotalResources:   totalResources,
		DriftedResources: driftedResources,
		DriftPercentage:  driftPercentage,
		DriftsByProvider: driftsByProvider,
		DriftsBySeverity: driftsBySeverity,
		RecentDrifts:     recentDrifts,
		Summary: map[string]interface{}{
			"total_checks":    totalResources,
			"drift_detected":  driftedResources,
			"no_drift":        totalResources - driftedResources,
			"compliance_rate": 100 - driftPercentage,
		},
		GeneratedAt: time.Now(),
	}

	return summary, nil
}
