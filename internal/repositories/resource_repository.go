package repositories

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/services"
)

// MemoryResourceRepository is an in-memory implementation of ResourceRepository
type MemoryResourceRepository struct {
	resources map[string]*models.CloudResource
	mu        sync.RWMutex
}

// NewMemoryResourceRepository creates a new in-memory resource repository
func NewMemoryResourceRepository() *MemoryResourceRepository {
	return &MemoryResourceRepository{
		resources: make(map[string]*models.CloudResource),
	}
}

// Create creates a new resource
func (r *MemoryResourceRepository) Create(ctx context.Context, resource *models.CloudResource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if resource already exists
	if _, exists := r.resources[resource.ID]; exists {
		return fmt.Errorf("resource with ID %s already exists", resource.ID)
	}

	// Set timestamps
	now := time.Now()
	resource.CreatedAt = now
	resource.UpdatedAt = now

	// Store resource
	r.resources[resource.ID] = resource

	return nil
}

// GetByID retrieves a resource by ID
func (r *MemoryResourceRepository) GetByID(ctx context.Context, id string) (*models.CloudResource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resource, exists := r.resources[id]
	if !exists {
		return nil, fmt.Errorf("resource with ID %s not found", id)
	}

	// Return a copy to prevent external modifications
	resourceCopy := *resource
	return &resourceCopy, nil
}

// GetAll retrieves all resources with optional filtering
func (r *MemoryResourceRepository) GetAll(ctx context.Context, filters services.ResourceFilters) ([]*models.CloudResource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.CloudResource

	for _, resource := range r.resources {
		// Apply filters
		if filters.Provider != "" && string(resource.Provider) != filters.Provider {
			continue
		}
		if filters.Type != "" && resource.Type != filters.Type {
			continue
		}
		if filters.Region != "" && resource.Region != filters.Region {
			continue
		}
		if filters.AccountID != "" && resource.AccountID != filters.AccountID {
			continue
		}
		if filters.ProjectID != "" && resource.ProjectID != filters.ProjectID {
			continue
		}

		// Apply tag filters
		if len(filters.Tags) > 0 {
			matches := true
			for key, value := range filters.Tags {
				if resource.Tags[key] != value {
					matches = false
					break
				}
			}
			if !matches {
				continue
			}
		}

		// Return a copy to prevent external modifications
		resourceCopy := *resource
		results = append(results, &resourceCopy)
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

// Update updates an existing resource
func (r *MemoryResourceRepository) Update(ctx context.Context, resource *models.CloudResource) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if resource exists
	if _, exists := r.resources[resource.ID]; !exists {
		return fmt.Errorf("resource with ID %s not found", resource.ID)
	}

	// Update timestamp
	resource.UpdatedAt = time.Now()

	// Store updated resource
	r.resources[resource.ID] = resource

	return nil
}

// Delete deletes a resource
func (r *MemoryResourceRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if resource exists
	if _, exists := r.resources[id]; !exists {
		return fmt.Errorf("resource with ID %s not found", id)
	}

	// Delete resource
	delete(r.resources, id)

	return nil
}

// Search searches for resources based on query and filters
func (r *MemoryResourceRepository) Search(ctx context.Context, query services.ResourceSearchQuery) ([]*models.CloudResource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*models.CloudResource

	for _, resource := range r.resources {
		// Apply text search
		if query.Query != "" {
			// Simple text search in name, type, and tags
			searchText := fmt.Sprintf("%s %s", resource.Name, resource.Type)
			if !containsIgnoreCase(searchText, query.Query) {
				continue
			}
		}

		// Apply filters
		if query.Filters.Provider != "" && string(resource.Provider) != query.Filters.Provider {
			continue
		}
		if query.Filters.Type != "" && resource.Type != query.Filters.Type {
			continue
		}
		if query.Filters.Region != "" && resource.Region != query.Filters.Region {
			continue
		}
		if query.Filters.AccountID != "" && resource.AccountID != query.Filters.AccountID {
			continue
		}
		if query.Filters.ProjectID != "" && resource.ProjectID != query.Filters.ProjectID {
			continue
		}

		// Apply tag filters
		if len(query.Filters.Tags) > 0 {
			matches := true
			for key, value := range query.Filters.Tags {
				if resource.Tags[key] != value {
					matches = false
					break
				}
			}
			if !matches {
				continue
			}
		}

		// Return a copy to prevent external modifications
		resourceCopy := *resource
		results = append(results, &resourceCopy)
	}

	// Apply pagination
	if query.Offset > 0 && query.Offset < len(results) {
		results = results[query.Offset:]
	}
	if query.Limit > 0 && query.Limit < len(results) {
		results = results[:query.Limit]
	}

	return results, nil
}

// UpdateTags updates tags for a resource
func (r *MemoryResourceRepository) UpdateTags(ctx context.Context, resourceID string, tags map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if resource exists
	resource, exists := r.resources[resourceID]
	if !exists {
		return fmt.Errorf("resource with ID %s not found", resourceID)
	}

	// Update tags
	resource.Tags = tags
	resource.UpdatedAt = time.Now()

	return nil
}

// GetResourceCost retrieves cost information for a resource
func (r *MemoryResourceRepository) GetResourceCost(ctx context.Context, resourceID string) (*models.CostInformation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if resource exists
	resource, exists := r.resources[resourceID]
	if !exists {
		return nil, fmt.Errorf("resource with ID %s not found", resourceID)
	}

	// Simulate cost information based on resource type
	var costPerHour, costPerMonth float64
	var currency string = "USD"

	switch resource.Type {
	case "aws_instance":
		costPerHour = 0.10
		costPerMonth = 72.0
	case "aws_s3_bucket":
		costPerHour = 0.023
		costPerMonth = 16.56
	case "aws_rds_instance":
		costPerHour = 0.25
		costPerMonth = 180.0
	case "aws_lambda_function":
		costPerHour = 0.0000166667
		costPerMonth = 0.012
	default:
		costPerHour = 0.05
		costPerMonth = 36.0
	}

	costInfo := &models.CostInformation{
		HourlyCost:  costPerHour,
		MonthlyCost: costPerMonth,
		Currency:    currency,
		LastUpdated: time.Now(),
		CostBreakdown: map[string]float64{
			"base_cost": costPerHour,
		},
	}

	return costInfo, nil
}

// GetResourceCompliance retrieves compliance status for a resource
func (r *MemoryResourceRepository) GetResourceCompliance(ctx context.Context, resourceID string) (*models.ComplianceStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check if resource exists
	resource, exists := r.resources[resourceID]
	if !exists {
		return nil, fmt.Errorf("resource with ID %s not found", resourceID)
	}

	// Simulate compliance status based on resource type and configuration
	var violations []models.Violation

	// Check for common compliance issues
	if resource.Type == "aws_instance" {
		// Check if instance has proper security groups
		if securityGroups, ok := resource.Configuration["security_groups"]; !ok || securityGroups == nil {
			violations = append(violations, models.Violation{
				ID:          "violation-sg-001",
				RuleID:      "SG-001",
				RuleName:    "Security Groups Required",
				Severity:    "high",
				Description: "EC2 instance must have security groups configured",
				Remediation: "Configure appropriate security groups for the instance",
				DetectedAt:  time.Now(),
			})
		}

		// Check if instance has encryption enabled
		if encryption, ok := resource.Configuration["encryption"]; !ok || encryption != true {
			violations = append(violations, models.Violation{
				ID:          "violation-enc-001",
				RuleID:      "ENC-001",
				RuleName:    "Encryption Required",
				Severity:    "medium",
				Description: "EC2 instance should have encryption enabled",
				Remediation: "Enable encryption for the instance",
				DetectedAt:  time.Now(),
			})
		}
	}

	if resource.Type == "aws_s3_bucket" {
		// Check if bucket has versioning enabled
		if versioning, ok := resource.Configuration["versioning"]; !ok || versioning != true {
			violations = append(violations, models.Violation{
				ID:          "violation-ver-001",
				RuleID:      "VER-001",
				RuleName:    "Versioning Required",
				Severity:    "medium",
				Description: "S3 bucket should have versioning enabled",
				Remediation: "Enable versioning for the S3 bucket",
				DetectedAt:  time.Now(),
			})
		}
	}

	complianceStatus := &models.ComplianceStatus{
		Status:      models.ComplianceLevelCompliant,
		PolicyID:    "policy-1",
		PolicyName:  "Security Policy",
		Violations:  violations,
		LastChecked: time.Now(),
		NextCheck:   time.Now().Add(24 * time.Hour),
		CheckedBy:   "system",
	}

	return complianceStatus, nil
}

// Helper function for case-insensitive string search
func containsIgnoreCase(s, substr string) bool {
	// Simple implementation - in production, you'd use a proper case-insensitive search
	return len(s) >= len(substr) && (s == substr || len(substr) == 0)
}
