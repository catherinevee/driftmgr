package repositories

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/services"
)

// MemoryBackendRepository is an in-memory implementation of BackendRepository
type MemoryBackendRepository struct {
	backends map[string]*models.ProviderConfiguration
	mu       sync.RWMutex
}

// NewMemoryBackendRepository creates a new in-memory backend repository
func NewMemoryBackendRepository() *MemoryBackendRepository {
	return &MemoryBackendRepository{
		backends: make(map[string]*models.ProviderConfiguration),
	}
}

// Create creates a new backend configuration
func (r *MemoryBackendRepository) Create(ctx context.Context, backend *models.ProviderConfiguration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if backend already exists
	if _, exists := r.backends[backend.ID]; exists {
		return fmt.Errorf("backend with ID %s already exists", backend.ID)
	}

	// Set timestamps
	now := time.Now()
	backend.CreatedAt = now
	backend.UpdatedAt = now

	// Store backend
	r.backends[backend.ID] = backend

	return nil
}

// GetByID retrieves a backend configuration by ID
func (r *MemoryBackendRepository) GetByID(ctx context.Context, id string) (*models.ProviderConfiguration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	backend, exists := r.backends[id]
	if !exists {
		return nil, fmt.Errorf("backend with ID %s not found", id)
	}

	// Return a copy to prevent external modifications
	backendCopy := *backend
	return &backendCopy, nil
}

// GetAll retrieves all backend configurations with optional filtering
func (r *MemoryBackendRepository) GetAll(ctx context.Context, filters services.BackendFilters) ([]*models.ProviderConfiguration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.ProviderConfiguration

	for _, backend := range r.backends {
		// Apply filters
		if filters.Provider != "" && string(backend.Provider) != filters.Provider {
			continue
		}
		if filters.Region != "" && backend.Region != filters.Region {
			continue
		}
		if filters.IsActive != nil && backend.IsActive != *filters.IsActive {
			continue
		}
		if filters.IsDefault != nil && backend.IsDefault != *filters.IsDefault {
			continue
		}
		if filters.CreatedBy != "" && backend.CreatedBy != filters.CreatedBy {
			continue
		}

		// Return a copy to prevent external modifications
		backendCopy := *backend
		results = append(results, &backendCopy)
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

// Update updates an existing backend configuration
func (r *MemoryBackendRepository) Update(ctx context.Context, backend *models.ProviderConfiguration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if backend exists
	if _, exists := r.backends[backend.ID]; !exists {
		return fmt.Errorf("backend with ID %s not found", backend.ID)
	}

	// Update timestamp
	backend.UpdatedAt = time.Now()

	// Store updated backend
	r.backends[backend.ID] = backend

	return nil
}

// Delete deletes a backend configuration
func (r *MemoryBackendRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if backend exists
	if _, exists := r.backends[id]; !exists {
		return fmt.Errorf("backend with ID %s not found", id)
	}

	// Delete backend
	delete(r.backends, id)

	return nil
}

// TestConnection tests the connection to a backend
func (r *MemoryBackendRepository) TestConnection(ctx context.Context, backend *models.ProviderConfiguration) (*models.ProviderTestConnectionResponse, error) {
	// This would typically involve calling the actual provider's connection test
	// For now, we'll simulate a connection test

	// Simulate connection test based on provider
	var success bool
	var message string
	var details map[string]interface{}

	switch backend.Provider {
	case models.ProviderAWS:
		success = true
		message = "AWS connection test successful"
		details = map[string]interface{}{
			"provider":   "aws",
			"region":     backend.Region,
			"account_id": backend.AccountID,
			"tested_at":  time.Now(),
		}
	case models.ProviderAzure:
		success = false
		message = "Azure connection test not implemented"
		details = map[string]interface{}{
			"provider": "azure",
			"error":    "Azure provider not yet implemented",
		}
	case models.ProviderGCP:
		success = false
		message = "GCP connection test not implemented"
		details = map[string]interface{}{
			"provider": "gcp",
			"error":    "GCP provider not yet implemented",
		}
	case models.ProviderDigitalOcean:
		success = false
		message = "DigitalOcean connection test not implemented"
		details = map[string]interface{}{
			"provider": "digitalocean",
			"error":    "DigitalOcean provider not yet implemented",
		}
	default:
		success = false
		message = "Unknown provider"
		details = map[string]interface{}{
			"provider": string(backend.Provider),
			"error":    "Unknown provider type",
		}
	}

	response := &models.ProviderTestConnectionResponse{
		Success:  success,
		Message:  message,
		Details:  details,
		TestedAt: time.Now(),
	}

	return response, nil
}
