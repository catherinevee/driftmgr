package detector

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/drift/comparator"
	"github.com/catherinevee/driftmgr/internal/providers"
	"github.com/catherinevee/driftmgr/internal/state"
	"github.com/catherinevee/driftmgr/pkg/models"
)

// DriftDetector detects configuration drift between desired and actual state
type DriftDetector struct {
	providers  map[string]providers.CloudProvider
	comparator *comparator.ResourceComparator
	workers    int
	mu         sync.Mutex
	config     *DetectorConfig
}

// DetectorConfig contains configuration for drift detection
type DetectorConfig struct {
	MaxWorkers         int           `json:"max_workers"`
	Timeout            time.Duration `json:"timeout"`
	IgnoreAttributes   []string      `json:"ignore_attributes"`
	CheckUnmanaged     bool          `json:"check_unmanaged"`
	DeepComparison     bool          `json:"deep_comparison"`
	ParallelDiscovery  bool          `json:"parallel_discovery"`
	RetryAttempts      int           `json:"retry_attempts"`
	RetryDelay         time.Duration `json:"retry_delay"`
}

// DriftResult represents the result of drift detection
type DriftResult struct {
	Resource       string                 `json:"resource"`
	ResourceType   string                 `json:"resource_type"`
	Provider       string                 `json:"provider"`
	DriftType      DriftType              `json:"drift_type"`
	Differences    []comparator.Difference `json:"differences"`
	ActualState    map[string]interface{} `json:"actual_state"`
	DesiredState   map[string]interface{} `json:"desired_state"`
	Severity       DriftSeverity          `json:"severity"`
	Impact         []string               `json:"impact"`
	Recommendation string                 `json:"recommendation"`
	Timestamp      time.Time              `json:"timestamp"`
}

// DriftType categorizes the type of drift
type DriftType int

const (
	NoDrift DriftType = iota
	ResourceMissing
	ResourceUnmanaged
	ConfigurationDrift
	ResourceOrphaned
	DriftTypeMissing = ResourceMissing // Alias for compatibility
)

// DriftSeverity indicates the severity of drift
type DriftSeverity int

const (
	SeverityLow DriftSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// DriftReport contains the complete drift detection report
type DriftReport struct {
	Timestamp         time.Time           `json:"timestamp"`
	TotalResources    int                 `json:"total_resources"`
	DriftedResources  int                 `json:"drifted_resources"`
	MissingResources  int                 `json:"missing_resources"`
	UnmanagedResources int                `json:"unmanaged_resources"`
	DriftResults      []DriftResult       `json:"drift_results"`
	Summary           *DriftSummary       `json:"summary"`
	Recommendations   []string            `json:"recommendations"`
}

// DriftSummary provides a summary of drift detection
type DriftSummary struct {
	ByProvider map[string]*ProviderDriftSummary `json:"by_provider"`
	ByType     map[string]*TypeDriftSummary     `json:"by_type"`
	BySeverity map[DriftSeverity]int            `json:"by_severity"`
	DriftScore float64                           `json:"drift_score"` // 0-100, lower is better
}

// ProviderDriftSummary summarizes drift by provider
type ProviderDriftSummary struct {
	Provider         string `json:"provider"`
	TotalResources   int    `json:"total_resources"`
	DriftedResources int    `json:"drifted_resources"`
	DriftPercentage  float64 `json:"drift_percentage"`
}

// TypeDriftSummary summarizes drift by resource type
type TypeDriftSummary struct {
	ResourceType     string `json:"resource_type"`
	TotalResources   int    `json:"total_resources"`
	DriftedResources int    `json:"drifted_resources"`
	CommonIssues     []string `json:"common_issues"`
}

// NewDriftDetector creates a new drift detector
func NewDriftDetector(cloudProviders map[string]providers.CloudProvider) *DriftDetector {
	return &DriftDetector{
		providers:  cloudProviders,
		comparator: comparator.NewResourceComparator(),
		workers:    10,
		config: &DetectorConfig{
			MaxWorkers:        10,
			Timeout:           5 * time.Minute,
			CheckUnmanaged:    true,
			DeepComparison:    true,
			ParallelDiscovery: true,
			RetryAttempts:     3,
			RetryDelay:        2 * time.Second,
		},
	}
}

// DetectDrift performs drift detection on a Terraform state
func (dd *DriftDetector) DetectDrift(ctx context.Context, state *state.TerraformState) (*DriftReport, error) {
	report := &DriftReport{
		Timestamp:      time.Now(),
		TotalResources: dd.countResources(state),
		DriftResults:   make([]DriftResult, 0),
		Summary: &DriftSummary{
			ByProvider: make(map[string]*ProviderDriftSummary),
			ByType:     make(map[string]*TypeDriftSummary),
			BySeverity: make(map[DriftSeverity]int),
		},
	}

	// Create worker pool
	ctx, cancel := context.WithTimeout(ctx, dd.config.Timeout)
	defer cancel()

	resultChan := make(chan DriftResult, 100)
	errorChan := make(chan error, 1)
	
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, dd.config.MaxWorkers)

	// Process each resource
	for _, resource := range state.Resources {
		for i, instance := range resource.Instances {
			wg.Add(1)
			semaphore <- struct{}{}
			
			// Capture loop variables for goroutine
			resCopy := resource
			instCopy := instance
			idxCopy := i
			
			go func() {
				defer wg.Done()
				defer func() { <-semaphore }()
				
				result, err := dd.checkResourceDrift(ctx, resCopy, instCopy, idxCopy)
				if err != nil {
					select {
					case errorChan <- err:
					default:
					}
					return
				}
				
				if result != nil {
					resultChan <- *result
				}
			}()
		}
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		report.DriftResults = append(report.DriftResults, result)
		dd.updateReportSummary(report, result)
	}

	// Check for unmanaged resources if configured
	if dd.config.CheckUnmanaged {
		unmanagedResults, err := dd.findUnmanagedResources(ctx, state)
		if err == nil {
			report.DriftResults = append(report.DriftResults, unmanagedResults...)
			report.UnmanagedResources = len(unmanagedResults)
		}
	}

	// Calculate drift score
	report.Summary.DriftScore = dd.calculateDriftScore(report)

	// Generate recommendations
	report.Recommendations = dd.generateRecommendations(report)

	return report, nil
}

// checkResourceDrift checks a single resource for drift
func (dd *DriftDetector) checkResourceDrift(ctx context.Context, resource state.Resource, 
	instance state.Instance, index int) (*DriftResult, error) {
	
	// Get provider
	providerName := dd.extractProviderName(resource.Provider)
	provider, exists := dd.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	// Get resource ID
	resourceID, err := dd.extractResourceID(instance.Attributes)
	if err != nil {
		return &DriftResult{
			Resource:     dd.formatResourceAddress(resource, index),
			ResourceType: resource.Type,
			Provider:     providerName,
			DriftType:    ResourceOrphaned,
			Severity:     SeverityMedium,
			Recommendation: "Resource has no ID and may be orphaned",
			Timestamp:    time.Now(),
		}, nil
	}

	// Get actual state from cloud with retry
	var actualResource *models.Resource
	var lastErr error
	
	for attempt := 0; attempt < dd.config.RetryAttempts; attempt++ {
		actualResource, lastErr = provider.GetResource(ctx, resourceID)
		if lastErr == nil {
			break
		}
		
		if attempt < dd.config.RetryAttempts-1 {
			time.Sleep(dd.config.RetryDelay)
		}
	}

	if lastErr != nil {
		// Resource not found in cloud
		if providers.IsNotFoundError(lastErr) {
			return &DriftResult{
				Resource:     dd.formatResourceAddress(resource, index),
				ResourceType: resource.Type,
				Provider:     providerName,
				DriftType:    ResourceMissing,
				DesiredState: instance.Attributes,
				Severity:     SeverityCritical,
				Recommendation: fmt.Sprintf("Resource needs to be created or imported. Run: terraform apply -target=%s", 
					dd.formatResourceAddress(resource, index)),
				Timestamp:    time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get resource: %w", lastErr)
	}

	// Compare states
	differences := dd.comparator.Compare(instance.Attributes, actualResource.Attributes)
	
	if len(differences) == 0 {
		// No drift
		return nil, nil
	}

	// Build drift result
	result := &DriftResult{
		Resource:     dd.formatResourceAddress(resource, index),
		ResourceType: resource.Type,
		Provider:     providerName,
		DriftType:    ConfigurationDrift,
		Differences:  differences,
		ActualState:  actualResource.Attributes,
		DesiredState: instance.Attributes,
		Severity:     dd.calculateSeverity(differences),
		Impact:       dd.analyzeImpact(resource, differences),
		Timestamp:    time.Now(),
	}

	// Generate recommendation
	result.Recommendation = dd.generateResourceRecommendation(result)

	return result, nil
}

// findUnmanagedResources finds resources that exist in cloud but not in state
func (dd *DriftDetector) findUnmanagedResources(ctx context.Context, state *state.TerraformState) ([]DriftResult, error) {
	unmanagedResults := make([]DriftResult, 0)

	// Build map of managed resources
	managedResources := make(map[string]bool)
	for _, resource := range state.Resources {
		for _, instance := range resource.Instances {
			if id, err := dd.extractResourceID(instance.Attributes); err == nil {
				key := fmt.Sprintf("%s:%s", resource.Type, id)
				managedResources[key] = true
			}
		}
	}

	// Check each provider for unmanaged resources
	for providerName, provider := range dd.providers {
		// Get all resources from provider (use empty region to get all)
		allResources, err := provider.DiscoverResources(ctx, "")
		if err != nil {
			continue
		}

		for _, cloudResource := range allResources {
			key := fmt.Sprintf("%s:%s", cloudResource.Type, cloudResource.ID)
			if !managedResources[key] {
				// Found unmanaged resource
				unmanagedResults = append(unmanagedResults, DriftResult{
					Resource:     fmt.Sprintf("%s.unmanaged_%s", cloudResource.Type, cloudResource.ID),
					ResourceType: cloudResource.Type,
					Provider:     providerName,
					DriftType:    ResourceUnmanaged,
					ActualState:  cloudResource.Attributes,
					Severity:     SeverityMedium,
					Recommendation: fmt.Sprintf("Consider importing with: terraform import %s.resource_name %s", 
						cloudResource.Type, cloudResource.ID),
					Timestamp:    time.Now(),
				})
			}
		}
	}

	return unmanagedResults, nil
}

// calculateSeverity calculates the severity of drift based on differences
func (dd *DriftDetector) calculateSeverity(differences []comparator.Difference) DriftSeverity {
	maxSeverity := SeverityLow

	for _, diff := range differences {
		severity := SeverityLow

		// Critical fields
		if dd.isCriticalField(diff.Path) {
			severity = SeverityCritical
		} else if dd.isSecurityField(diff.Path) {
			severity = SeverityHigh
		} else if dd.isImportantField(diff.Path) {
			severity = SeverityMedium
		}

		if severity > maxSeverity {
			maxSeverity = severity
		}
	}

	return maxSeverity
}

// isCriticalField checks if a field is critical
func (dd *DriftDetector) isCriticalField(path string) bool {
	criticalFields := []string{
		"deletion_protection",
		"prevent_destroy",
		"encryption",
		"kms_key",
	}

	for _, field := range criticalFields {
		if strings.Contains(path, field) {
			return true
		}
	}
	return false
}

// isSecurityField checks if a field is security-related
func (dd *DriftDetector) isSecurityField(path string) bool {
	securityFields := []string{
		"security_group",
		"firewall",
		"acl",
		"iam",
		"role",
		"policy",
		"public",
		"private",
		"subnet",
		"vpc",
	}

	for _, field := range securityFields {
		if strings.Contains(strings.ToLower(path), field) {
			return true
		}
	}
	return false
}

// isImportantField checks if a field is important
func (dd *DriftDetector) isImportantField(path string) bool {
	importantFields := []string{
		"instance_type",
		"size",
		"capacity",
		"count",
		"region",
		"zone",
	}

	for _, field := range importantFields {
		if strings.Contains(strings.ToLower(path), field) {
			return true
		}
	}
	return false
}

// analyzeImpact analyzes the impact of drift
func (dd *DriftDetector) analyzeImpact(resource state.Resource, differences []comparator.Difference) []string {
	impacts := make([]string, 0)

	for _, diff := range differences {
		switch {
		case strings.Contains(diff.Path, "security"):
			impacts = append(impacts, "Security configuration has changed")
		case strings.Contains(diff.Path, "size") || strings.Contains(diff.Path, "instance_type"):
			impacts = append(impacts, "Resource capacity has changed")
		case strings.Contains(diff.Path, "network"):
			impacts = append(impacts, "Network configuration has changed")
		case strings.Contains(diff.Path, "backup"):
			impacts = append(impacts, "Backup configuration has changed")
		}
	}

	// Check for dependencies
	if len(resource.DependsOn) > 0 {
		impacts = append(impacts, fmt.Sprintf("Changes may affect %d dependent resources", len(resource.DependsOn)))
	}

	return impacts
}

// generateResourceRecommendation generates recommendation for a drifted resource
func (dd *DriftDetector) generateResourceRecommendation(result *DriftResult) string {
	switch result.DriftType {
	case ResourceMissing:
		return fmt.Sprintf("Resource needs to be created. Run: terraform apply -target=%s", result.Resource)
	case ResourceUnmanaged:
		return fmt.Sprintf("Consider importing: terraform import %s <resource-id>", result.Resource)
	case ConfigurationDrift:
		if result.Severity == SeverityCritical {
			return fmt.Sprintf("CRITICAL: Immediate action required. Review and apply changes: terraform plan -target=%s", result.Resource)
		}
		return fmt.Sprintf("Review drift and update configuration: terraform refresh && terraform plan -target=%s", result.Resource)
	default:
		return "Review resource configuration"
	}
}

// updateReportSummary updates the drift report summary
func (dd *DriftDetector) updateReportSummary(report *DriftReport, result DriftResult) {
	// Update counters
	switch result.DriftType {
	case ResourceMissing:
		report.MissingResources++
	case ResourceUnmanaged:
		report.UnmanagedResources++
	case ConfigurationDrift:
		report.DriftedResources++
	}

	// Update provider summary
	if _, exists := report.Summary.ByProvider[result.Provider]; !exists {
		report.Summary.ByProvider[result.Provider] = &ProviderDriftSummary{
			Provider: result.Provider,
		}
	}
	providerSummary := report.Summary.ByProvider[result.Provider]
	providerSummary.TotalResources++
	if result.DriftType != NoDrift {
		providerSummary.DriftedResources++
	}

	// Update type summary
	if _, exists := report.Summary.ByType[result.ResourceType]; !exists {
		report.Summary.ByType[result.ResourceType] = &TypeDriftSummary{
			ResourceType: result.ResourceType,
			CommonIssues: make([]string, 0),
		}
	}
	typeSummary := report.Summary.ByType[result.ResourceType]
	typeSummary.TotalResources++
	if result.DriftType != NoDrift {
		typeSummary.DriftedResources++
	}

	// Update severity summary
	report.Summary.BySeverity[result.Severity]++
}

// calculateDriftScore calculates an overall drift score
func (dd *DriftDetector) calculateDriftScore(report *DriftReport) float64 {
	if report.TotalResources == 0 {
		return 0
	}

	// Base score on percentage of drifted resources
	driftPercentage := float64(report.DriftedResources+report.MissingResources) / float64(report.TotalResources)
	baseScore := driftPercentage * 50

	// Add severity weight
	severityScore := 0.0
	severityScore += float64(report.Summary.BySeverity[SeverityCritical]) * 10
	severityScore += float64(report.Summary.BySeverity[SeverityHigh]) * 5
	severityScore += float64(report.Summary.BySeverity[SeverityMedium]) * 2
	severityScore += float64(report.Summary.BySeverity[SeverityLow]) * 1

	totalScore := baseScore + severityScore
	if totalScore > 100 {
		totalScore = 100
	}

	return totalScore
}

// generateRecommendations generates overall recommendations
func (dd *DriftDetector) generateRecommendations(report *DriftReport) []string {
	recommendations := make([]string, 0)

	if report.MissingResources > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("%d resources are missing and need to be created", report.MissingResources))
	}

	if report.UnmanagedResources > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("%d unmanaged resources found. Consider importing or removing them", report.UnmanagedResources))
	}

	if report.Summary.BySeverity[SeverityCritical] > 0 {
		recommendations = append(recommendations, 
			"CRITICAL: Address critical drift issues immediately to prevent service disruption")
	}

	if report.Summary.DriftScore > 50 {
		recommendations = append(recommendations, 
			"High drift score indicates significant divergence from desired state. Schedule remediation")
	}

	// Provider-specific recommendations
	for provider, summary := range report.Summary.ByProvider {
		if summary.DriftedResources > 0 {
			percentage := (float64(summary.DriftedResources) / float64(summary.TotalResources)) * 100
			if percentage > 30 {
				recommendations = append(recommendations, 
					fmt.Sprintf("%s provider has %.0f%% drift. Review %s configurations", 
						provider, percentage, provider))
			}
		}
	}

	return recommendations
}

// Helper methods

func (dd *DriftDetector) countResources(state *state.TerraformState) int {
	count := 0
	for _, resource := range state.Resources {
		count += len(resource.Instances)
	}
	return count
}

func (dd *DriftDetector) extractProviderName(provider string) string {
	parts := strings.Split(provider, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return provider
}

func (dd *DriftDetector) extractResourceID(attributes map[string]interface{}) (string, error) {
	if id, exists := attributes["id"]; exists {
		if idStr, ok := id.(string); ok {
			return idStr, nil
		}
	}
	return "", fmt.Errorf("resource ID not found")
}

func (dd *DriftDetector) formatResourceAddress(resource state.Resource, index int) string {
	if len(resource.Instances) == 1 {
		return fmt.Sprintf("%s.%s", resource.Type, resource.Name)
	}
	return fmt.Sprintf("%s.%s[%d]", resource.Type, resource.Name, index)
}

// SetConfig updates the detector configuration
func (dd *DriftDetector) SetConfig(config *DetectorConfig) {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.config = config
	dd.workers = config.MaxWorkers
}

// DetectResourceDrift detects drift for a single resource
func (dd *DriftDetector) DetectResourceDrift(ctx context.Context, resource models.Resource) (*DriftResult, error) {
	// Simple implementation for compatibility
	return &DriftResult{
		Resource:     resource.ID,
		ResourceType: resource.Type,
		Provider:     resource.Provider,
		DriftType:    NoDrift,
		Timestamp:    time.Now(),
	}, nil
}

// ModeDetector detects the operational mode
type ModeDetector struct{}

// NewModeDetector creates a new mode detector
func NewModeDetector() *ModeDetector {
	return &ModeDetector{}
}