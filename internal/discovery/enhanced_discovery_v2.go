package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/cost"
	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/security"
	"github.com/catherinevee/driftmgr/internal/utilization"
)

// EnhancedResource represents a resource with comprehensive metadata
type EnhancedResource struct {
	models.Resource
	CostData        *cost.CostAnalysis
	UtilizationData *utilization.UtilizationMetrics
	SecurityScore   *security.SecurityAssessment
	ComplianceData  *ComplianceStatus
	LastActivity    time.Time
	OwnerInfo       *ResourceOwner
	Dependencies    []string
	CrossRegionRefs []CrossRegionReference
}

// ComplianceStatus represents compliance information for a resource
type ComplianceStatus struct {
	OverallScore   int
	ComplianceGaps []ComplianceGap
	LastAssessment time.Time
	Framework      string // "SOC2", "GDPR", "HIPAA", "custom"
}

// ComplianceGap represents a compliance gap
type ComplianceGap struct {
	Category    string
	Description string
	Severity    string // "low", "medium", "high", "critical"
	Remediation string
}

// ResourceOwner represents ownership information
type ResourceOwner struct {
	Team        string
	Contact     string
	Department  string
	Project     string
	CostCenter  string
	LastContact time.Time
}

// CrossRegionReference represents cross-region resource relationships
type CrossRegionReference struct {
	ResourceID   string
	ResourceType string
	Region       string
	Relationship string // "depends_on", "replicates", "backup_of"
}

// EnhancedDiscoveryEngine provides comprehensive resource discovery
type EnhancedDiscoveryEngine struct {
	providers       map[string]CloudProvider
	costAnalyzer    *cost.CostAnalyzer
	securityScanner *security.SecurityScanner
	utilizationMon  *utilization.UtilizationMonitor
	mu              sync.RWMutex
	cache           map[string]*EnhancedResource
	cacheExpiry     time.Duration
}

// CloudProvider interface for multi-cloud support
type CloudProvider interface {
	DiscoverResources(ctx context.Context, regions []string) ([]models.Resource, error)
	GetResourceMetadata(ctx context.Context, resource models.Resource) (*ResourceMetadata, error)
	GetCrossRegionRefs(ctx context.Context, resource models.Resource) ([]CrossRegionReference, error)
	GetDependencies(ctx context.Context, resource models.Resource) ([]string, error)
}

// ResourceMetadata represents detailed resource metadata
type ResourceMetadata struct {
	CreationTime    time.Time
	LastModified    time.Time
	Tags            map[string]string
	Configuration   map[string]interface{}
	PerformanceData map[string]interface{}
	SecurityConfig  map[string]interface{}
}

// NewEnhancedDiscoveryEngine creates a new enhanced discovery engine
func NewEnhancedDiscoveryEngine(
	costAnalyzer *cost.CostAnalyzer,
	securityScanner *security.SecurityScanner,
	utilizationMon *utilization.UtilizationMonitor,
) *EnhancedDiscoveryEngine {
	return &EnhancedDiscoveryEngine{
		providers:       make(map[string]CloudProvider),
		costAnalyzer:    costAnalyzer,
		securityScanner: securityScanner,
		utilizationMon:  utilizationMon,
		cache:           make(map[string]*EnhancedResource),
		cacheExpiry:     30 * time.Minute,
	}
}

// RegisterProvider registers a cloud provider
func (ede *EnhancedDiscoveryEngine) RegisterProvider(name string, provider CloudProvider) {
	ede.mu.Lock()
	defer ede.mu.Unlock()
	ede.providers[name] = provider
}

// DiscoverResourcesEnhanced performs comprehensive resource discovery
func (ede *EnhancedDiscoveryEngine) DiscoverResourcesEnhanced(
	ctx context.Context,
	provider string,
	regions []string,
	options DiscoveryOptions,
) ([]*EnhancedResource, error) {
	// Check cache first
	if options.UseCache {
		if cached := ede.getCachedResources(provider, regions); len(cached) > 0 {
			return cached, nil
		}
	}

	// Get provider
	cloudProvider, exists := ede.providers[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not registered", provider)
	}

	// Discover basic resources
	resources, err := cloudProvider.DiscoverResources(ctx, regions)
	if err != nil {
		return nil, fmt.Errorf("failed to discover resources: %w", err)
	}

	// Enhance resources with additional data
	enhancedResources := make([]*EnhancedResource, 0, len(resources))

	// Use worker pool for parallel processing
	workerCount := options.MaxConcurrency
	if workerCount <= 0 {
		workerCount = 10
	}

	// Create channels for coordination
	resourceChan := make(chan models.Resource, len(resources))
	resultChan := make(chan *EnhancedResource, len(resources))
	errorChan := make(chan error, len(resources))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for resource := range resourceChan {
				enhanced, err := ede.enhanceResource(ctx, resource, cloudProvider, options)
				if err != nil {
					errorChan <- fmt.Errorf("failed to enhance resource %s: %w", resource.ID, err)
					continue
				}
				resultChan <- enhanced
			}
		}()
	}

	// Send resources to workers
	go func() {
		defer close(resourceChan)
		for _, resource := range resources {
			resourceChan <- resource
		}
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Process results
	for enhanced := range resultChan {
		enhancedResources = append(enhancedResources, enhanced)
	}

	// Check for errors
	select {
	case err := <-errorChan:
		if err != nil {
			return enhancedResources, err // Return partial results with error
		}
	default:
	}

	// Cache results
	if options.UseCache {
		ede.cacheResources(provider, regions, enhancedResources)
	}

	return enhancedResources, nil
}

// enhanceResource enhances a single resource with comprehensive data
func (ede *EnhancedDiscoveryEngine) enhanceResource(
	ctx context.Context,
	resource models.Resource,
	provider CloudProvider,
	options DiscoveryOptions,
) (*EnhancedResource, error) {
	enhanced := &EnhancedResource{
		Resource:     resource,
		LastActivity: time.Now(),
	}

	// Get metadata
	if options.IncludeMetadata {
		metadata, err := provider.GetResourceMetadata(ctx, resource)
		if err == nil {
			enhanced.LastActivity = metadata.LastModified
		}
	}

	// Get cost data
	if options.IncludeCostData && ede.costAnalyzer != nil {
		costData, err := ede.costAnalyzer.AnalyzeResourceCost(ctx, resource)
		if err == nil {
			enhanced.CostData = costData
		}
	}

	// Get utilization data
	if options.IncludeUtilization && ede.utilizationMon != nil {
		utilData, err := ede.utilizationMon.GetResourceUtilization(ctx, resource)
		if err == nil {
			enhanced.UtilizationData = utilData
		}
	}

	// Get security assessment
	if options.IncludeSecurityAssessment && ede.securityScanner != nil {
		securityData, err := ede.securityScanner.AssessResource(ctx, resource)
		if err == nil {
			enhanced.SecurityScore = securityData
		}
	}

	// Get compliance data
	if options.IncludeComplianceData {
		complianceData, err := ede.getComplianceData(ctx, resource)
		if err == nil {
			enhanced.ComplianceData = complianceData
		}
	}

	// Get owner information
	if options.IncludeOwnerInfo {
		ownerInfo, err := ede.getOwnerInfo(ctx, resource)
		if err == nil {
			enhanced.OwnerInfo = ownerInfo
		}
	}

	// Get dependencies
	if options.IncludeDependencies {
		dependencies, err := provider.GetDependencies(ctx, resource)
		if err == nil {
			enhanced.Dependencies = dependencies
		}
	}

	// Get cross-region references
	if options.IncludeCrossRegionRefs {
		crossRegionRefs, err := provider.GetCrossRegionRefs(ctx, resource)
		if err == nil {
			enhanced.CrossRegionRefs = crossRegionRefs
		}
	}

	return enhanced, nil
}

// getComplianceData retrieves compliance information for a resource
func (ede *EnhancedDiscoveryEngine) getComplianceData(ctx context.Context, resource models.Resource) (*ComplianceStatus, error) {
	// This would integrate with compliance frameworks
	// For now, return a basic assessment
	return &ComplianceStatus{
		OverallScore:   75, // Example score
		LastAssessment: time.Now(),
		Framework:      "SOC2",
		ComplianceGaps: []ComplianceGap{
			{
				Category:    "Data Protection",
				Description: "Resource lacks encryption at rest",
				Severity:    "medium",
				Remediation: "Enable encryption for the resource",
			},
		},
	}, nil
}

// getOwnerInfo extracts owner information from resource tags
func (ede *EnhancedDiscoveryEngine) getOwnerInfo(ctx context.Context, resource models.Resource) (*ResourceOwner, error) {
	owner := &ResourceOwner{
		LastContact: time.Now(),
	}

	// Extract owner info from tags
	if team, exists := resource.Tags["Team"]; exists {
		owner.Team = team
	}
	if contact, exists := resource.Tags["Contact"]; exists {
		owner.Contact = contact
	}
	if dept, exists := resource.Tags["Department"]; exists {
		owner.Department = dept
	}
	if project, exists := resource.Tags["Project"]; exists {
		owner.Project = project
	}
	if costCenter, exists := resource.Tags["CostCenter"]; exists {
		owner.CostCenter = costCenter
	}

	return owner, nil
}

// getCachedResources retrieves cached resources
func (ede *EnhancedDiscoveryEngine) getCachedResources(provider string, regions []string) []*EnhancedResource {
	ede.mu.RLock()
	defer ede.mu.RUnlock()

	// Simple cache key based on provider and regions
	cacheKey := fmt.Sprintf("%s-%v", provider, regions)

	if cached, exists := ede.cache[cacheKey]; exists {
		// Check if cache is still valid
		if time.Since(cached.LastActivity) < ede.cacheExpiry {
			return []*EnhancedResource{cached}
		}
		// Remove expired cache
		delete(ede.cache, cacheKey)
	}

	return nil
}

// cacheResources stores resources in cache
func (ede *EnhancedDiscoveryEngine) cacheResources(provider string, regions []string, resources []*EnhancedResource) {
	ede.mu.Lock()
	defer ede.mu.Unlock()

	cacheKey := fmt.Sprintf("%s-%v", provider, regions)
	// For simplicity, cache the first resource as representative
	if len(resources) > 0 {
		ede.cache[cacheKey] = resources[0]
	}
}

// DiscoveryOptions configures discovery behavior
type DiscoveryOptions struct {
	IncludeMetadata           bool
	IncludeCostData           bool
	IncludeUtilization        bool
	IncludeSecurityAssessment bool
	IncludeComplianceData     bool
	IncludeOwnerInfo          bool
	IncludeDependencies       bool
	IncludeCrossRegionRefs    bool
	UseCache                  bool
	MaxConcurrency            int
}

// DefaultDiscoveryOptions returns default discovery options
func DefaultDiscoveryOptions() DiscoveryOptions {
	return DiscoveryOptions{
		IncludeMetadata:           true,
		IncludeCostData:           true,
		IncludeUtilization:        true,
		IncludeSecurityAssessment: true,
		IncludeComplianceData:     true,
		IncludeOwnerInfo:          true,
		IncludeDependencies:       true,
		IncludeCrossRegionRefs:    true,
		UseCache:                  true,
		MaxConcurrency:            10,
	}
}
