package deletion

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/constants"
	"github.com/catherinevee/driftmgr/internal/models"
)

// DeletionEngine provides comprehensive resource deletion capabilities
type DeletionEngine struct {
	providers map[string]CloudProvider
	mu        sync.RWMutex
}

// CloudProvider interface for different cloud providers
type CloudProvider interface {
	DeleteResources(ctx context.Context, accountID string, options DeletionOptions) (*DeletionResult, error)
	DeleteResource(ctx context.Context, resource models.Resource) error
	ListResources(ctx context.Context, accountID string) ([]models.Resource, error)
	ValidateCredentials(ctx context.Context, accountID string) error
}

// DeletionOptions configures the deletion process
type DeletionOptions struct {
	DryRun           bool                 `json:"dry_run"`
	Force            bool                 `json:"force"`
	ResourceTypes    []string             `json:"resource_types,omitempty"`
	Regions          []string             `json:"regions,omitempty"`
	ExcludeResources []string             `json:"exclude_resources,omitempty"`
	IncludeResources []string             `json:"include_resources,omitempty"`
	Filters          map[string]string    `json:"filters,omitempty"`
	Timeout          time.Duration        `json:"timeout"`
	BatchSize        int                  `json:"batch_size"`
	MaxRetries       int                  `json:"max_retries"`
	RetryDelay       time.Duration        `json:"retry_delay"`
	SafetyChecks     []SafetyCheck        `json:"safety_checks,omitempty"`
	ProgressCallback func(ProgressUpdate) `json:"-"`
	ErrorCallback    func(DeletionError)  `json:"-"`
}

// DeletionResult represents the result of a deletion operation
type DeletionResult struct {
	AccountID        string                 `json:"account_id"`
	Provider         string                 `json:"provider"`
	TotalResources   int                    `json:"total_resources"`
	DeletedResources int                    `json:"deleted_resources"`
	FailedResources  int                    `json:"failed_resources"`
	SkippedResources int                    `json:"skipped_resources"`
	RetriedResources int                    `json:"retried_resources"`
	StartTime        time.Time              `json:"start_time"`
	EndTime          time.Time              `json:"end_time"`
	Duration         time.Duration          `json:"duration"`
	Errors           []DeletionError        `json:"errors,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
	Details          map[string]interface{} `json:"details,omitempty"`
}

// DeletionError represents an error during deletion
type DeletionError struct {
	ResourceID   string    `json:"resource_id"`
	ResourceType string    `json:"resource_type"`
	Error        string    `json:"error"`
	RetryCount   int       `json:"retry_count"`
	Timestamp    time.Time `json:"timestamp"`
}

// ProgressUpdate represents real-time progress information
type ProgressUpdate struct {
	Type      string      `json:"type"`
	Message   string      `json:"message"`
	Progress  int         `json:"progress"`
	Total     int         `json:"total"`
	Current   string      `json:"current,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// SafetyCheck represents a safety validation
type SafetyCheck struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// ResourceDependency represents resource dependencies
type ResourceDependency struct {
	ResourceID   string   `json:"resource_id"`
	Dependencies []string `json:"dependencies"`
}

// NewDeletionEngine creates a new deletion engine
func NewDeletionEngine() *DeletionEngine {
	return &DeletionEngine{
		providers: make(map[string]CloudProvider),
	}
}

// RegisterProvider registers a cloud provider
func (de *DeletionEngine) RegisterProvider(name string, provider CloudProvider) {
	de.mu.Lock()
	defer de.mu.Unlock()
	de.providers[name] = provider
}

// GetProvider retrieves a cloud provider by name
func (de *DeletionEngine) GetProvider(name string) (CloudProvider, bool) {
	de.mu.RLock()
	defer de.mu.RUnlock()
	provider, exists := de.providers[name]
	return provider, exists
}

// DeleteAccountResources deletes all resources in a specific account with enhanced error handling
func (de *DeletionEngine) DeleteAccountResources(ctx context.Context, provider, accountID string, options DeletionOptions) (*DeletionResult, error) {
	de.mu.RLock()
	providerImpl, exists := de.providers[provider]
	de.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not supported", provider)
	}

	// Set default options
	if options.Timeout == 0 {
		options.Timeout = constants.DefaultDeletionTimeout
	}
	if options.BatchSize == 0 {
		options.BatchSize = constants.DefaultBatchSize
	}
	if options.MaxRetries == 0 {
		options.MaxRetries = constants.DefaultMaxRetries
	}
	if options.RetryDelay == 0 {
		options.RetryDelay = constants.DefaultRetryDelay
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()

	// Validate credentials
	if err := providerImpl.ValidateCredentials(ctx, accountID); err != nil {
		return nil, fmt.Errorf("credential validation failed: %w", err)
	}

	// Perform enhanced safety checks if not forced and not dry run
	if !options.Force && !options.DryRun {
		if err := de.performEnhancedSafetyChecks(ctx, providerImpl, accountID, options); err != nil {
			return nil, fmt.Errorf("safety checks failed: %w", err)
		}
	}

	// Execute deletion with retry logic
	return de.executeDeletionWithRetry(ctx, providerImpl, accountID, options)
}

// executeDeletionWithRetry executes deletion with retry logic for failed resources
func (de *DeletionEngine) executeDeletionWithRetry(ctx context.Context, provider CloudProvider, accountID string, options DeletionOptions) (*DeletionResult, error) {
	// First attempt
	result, err := provider.DeleteResources(ctx, accountID, options)
	if err != nil {
		return nil, err
	}

	// Retry failed resources
	if len(result.Errors) > 0 && options.MaxRetries > 0 {
		de.retryFailedResources(ctx, provider, accountID, options, result)
	}

	return result, nil
}

// retryFailedResources retries deletion of failed resources
func (de *DeletionEngine) retryFailedResources(ctx context.Context, provider CloudProvider, accountID string, options DeletionOptions, result *DeletionResult) {
	retryErrors := make([]DeletionError, 0)

	for _, err := range result.Errors {
		if err.RetryCount < options.MaxRetries {
			// Wait before retry
			time.Sleep(options.RetryDelay)

			// Retry the specific resource
			if retryErr := de.retrySingleResource(ctx, provider, accountID, options, err); retryErr != nil {
				err.RetryCount++
				err.Error = retryErr.Error()
				retryErrors = append(retryErrors, err)
			} else {
				// Success - update counters
				result.FailedResources--
				result.DeletedResources++
				result.RetriedResources++
			}
		} else {
			retryErrors = append(retryErrors, err)
		}
	}

	result.Errors = retryErrors
}

// retrySingleResource retries deletion of a single resource
func (de *DeletionEngine) retrySingleResource(ctx context.Context, provider CloudProvider, accountID string, options DeletionOptions, err DeletionError) error {
	maxRetries := constants.DefaultMaxRetries
	backoffMs := int(constants.DefaultRetryDelay.Milliseconds())

	log.Printf("Retrying deletion of resource %s (type: %s)", err.ResourceID, err.ResourceType)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Wait with exponential backoff
		time.Sleep(time.Duration(backoffMs*attempt) * time.Millisecond)

		// List resources to find the specific one
		resources, listErr := provider.ListResources(ctx, accountID)
		if listErr != nil {
			log.Printf("Retry %d/%d: Failed to list resources: %v", attempt, maxRetries, listErr)
			continue
		}

		// Find the resource that failed
		var targetResource *models.Resource
		for _, r := range resources {
			if r.ID == err.ResourceID {
				targetResource = &r
				break
			}
		}

		if targetResource == nil {
			// Resource might have been deleted already
			log.Printf("Resource %s not found - may have been deleted", err.ResourceID)
			return nil
		}

		// Attempt deletion again
		deleteErr := provider.DeleteResource(ctx, *targetResource)
		if deleteErr == nil {
			log.Printf("Retry %d/%d: Successfully deleted resource %s", attempt, maxRetries, err.ResourceID)
			return nil
		}

		// Check if error is retryable
		errStr := strings.ToLower(deleteErr.Error())
		isRetryable := strings.Contains(errStr, "throttl") ||
			strings.Contains(errStr, "rate limit") ||
			strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "temporary") ||
			strings.Contains(errStr, "try again")

		if !isRetryable {
			log.Printf("Retry %d/%d: Non-retryable error for resource %s: %v", attempt, maxRetries, err.ResourceID, deleteErr)
			return deleteErr
		}

		log.Printf("Retry %d/%d: Retryable error for resource %s: %v", attempt, maxRetries, err.ResourceID, deleteErr)
	}

	return fmt.Errorf("failed to delete resource %s after %d retries: %v", err.ResourceID, maxRetries, err.Error)
}

// performEnhancedSafetyChecks performs comprehensive safety validations
func (de *DeletionEngine) performEnhancedSafetyChecks(ctx context.Context, provider CloudProvider, accountID string, options DeletionOptions) error {
	// List resources to check for critical ones
	resources, err := provider.ListResources(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to list resources for safety check: %w", err)
	}

	// Check for critical resources that should not be deleted
	criticalResources := de.identifyCriticalResources(resources)
	if len(criticalResources) > 0 && !options.Force {
		return fmt.Errorf("critical resources found that would be deleted: %v", criticalResources)
	}

	// Check resource count with configurable threshold
	if len(resources) > constants.MaxResourcesWarning && !options.Force {
		return fmt.Errorf("large number of resources (%d) would be deleted. Use --force to override", len(resources))
	}

	// Check for production resources
	productionResources := de.identifyProductionResources(resources)
	if len(productionResources) > 0 && !options.Force {
		return fmt.Errorf("production resources found: %v. Use --force to override", productionResources)
	}

	// Check resource dependencies
	dependencyIssues := de.checkResourceDependencies(resources)
	if len(dependencyIssues) > 0 {
		return fmt.Errorf("resource dependency issues found: %v", dependencyIssues)
	}

	return nil
}

// identifyProductionResources identifies production resources based on tags and naming
func (de *DeletionEngine) identifyProductionResources(resources []models.Resource) []string {
	var production []string

	for _, resource := range resources {
		// Check for production tags
		tags := resource.GetTagsAsMap()
		if de.hasProductionTags(tags) {
			production = append(production, fmt.Sprintf("%s (production tags)", resource.Name))
		}

		// Check for production naming patterns
		if de.hasProductionNaming(resource.Name) {
			production = append(production, fmt.Sprintf("%s (production naming)", resource.Name))
		}
	}

	return production
}

// hasProductionTags checks if resource has production-related tags
func (de *DeletionEngine) hasProductionTags(tags map[string]string) bool {
	productionTagValues := map[string]bool{
		"production":    true,
		"prod":          true,
		"live":          true,
		"critical":      true,
		"protected":     true,
		"do-not-delete": true,
		"environment":   true,
	}

	for key, value := range tags {
		if productionTagValues[strings.ToLower(value)] || productionTagValues[strings.ToLower(key)] {
			return true
		}
	}

	return false
}

// hasProductionNaming checks if resource name suggests production environment
func (de *DeletionEngine) hasProductionNaming(name string) bool {
	productionPatterns := []string{
		"prod-", "production-", "live-", "critical-", "main-", "primary-",
		"-prod", "-production", "-live", "-critical", "-main", "-primary",
	}

	nameLower := strings.ToLower(name)
	for _, pattern := range productionPatterns {
		if strings.Contains(nameLower, pattern) {
			return true
		}
	}

	return false
}

// checkResourceDependencies checks for potential dependency issues
func (de *DeletionEngine) checkResourceDependencies(resources []models.Resource) []string {
	var issues []string

	// This is a simplified check - in a real implementation, you would:
	// 1. Build a dependency graph
	// 2. Check for circular dependencies
	// 3. Verify deletion order
	// 4. Check for shared resources

	// Example: Check for resources that might have dependencies
	for _, resource := range resources {
		if de.hasPotentialDependencies(resource) {
			issues = append(issues, fmt.Sprintf("%s may have dependencies", resource.Name))
		}
	}

	return issues
}

// hasPotentialDependencies checks if a resource might have dependencies
func (de *DeletionEngine) hasPotentialDependencies(resource models.Resource) bool {
	// Resources that typically have dependencies
	dependentTypes := map[string]bool{
		"ec2_instance":        true,
		"rds_instance":        true,
		"eks_cluster":         true,
		"elasticache_cluster": true,
		"load_balancer":       true,
		"virtual_machine":     true,
		"kubernetes_cluster":  true,
	}

	return dependentTypes[resource.Type]
}

// identifyCriticalResources identifies resources that should not be deleted
func (de *DeletionEngine) identifyCriticalResources(resources []models.Resource) []string {
	var critical []string

	for _, resource := range resources {
		// Check for critical resource types
		if de.isCriticalResourceType(resource.Type) {
			critical = append(critical, fmt.Sprintf("%s (%s)", resource.Name, resource.Type))
		}

		// Check for critical tags
		tags := resource.GetTagsAsMap()
		if de.hasCriticalTags(tags) {
			critical = append(critical, fmt.Sprintf("%s (critical tags)", resource.Name))
		}
	}

	return critical
}

// isCriticalResourceType checks if a resource type is critical
func (de *DeletionEngine) isCriticalResourceType(resourceType string) bool {
	criticalTypes := map[string]bool{
		"aws_iam_user":            true,
		"aws_iam_role":            true,
		"aws_iam_policy":          true,
		"aws_s3_bucket":           true,
		"aws_rds_cluster":         true,
		"aws_eks_cluster":         true,
		"azurerm_storage_account": true,
		"azurerm_key_vault":       true,
		"google_storage_bucket":   true,
		"google_kms_crypto_key":   true,
	}

	return criticalTypes[resourceType]
}

// hasCriticalTags checks if resource has critical tags
func (de *DeletionEngine) hasCriticalTags(tags map[string]string) bool {
	criticalTagValues := map[string]bool{
		"production":    true,
		"prod":          true,
		"critical":      true,
		"protected":     true,
		"do-not-delete": true,
	}

	for _, value := range tags {
		if criticalTagValues[value] {
			return true
		}
	}

	return false
}

// GetSupportedProviders returns list of supported providers
func (de *DeletionEngine) GetSupportedProviders() []string {
	de.mu.RLock()
	defer de.mu.RUnlock()

	providers := make([]string, 0, len(de.providers))
	for provider := range de.providers {
		providers = append(providers, provider)
	}

	return providers
}
