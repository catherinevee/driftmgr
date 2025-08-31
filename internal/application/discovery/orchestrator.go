package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/infrastructure/cache"
	"github.com/catherinevee/driftmgr/internal/infrastructure/config"
)

// EnhancedDiscoverer provides advanced cloud discovery capabilities
type EnhancedDiscoverer struct {
	config              *config.Config
	cache               *cache.DiscoveryCache
	plugins             map[string]*DiscoveryPlugin
	hierarchy           *ResourceHierarchy
	filters             *DiscoveryFilter
	progressTracker     *ProgressTracker
	visualizer          *DiscoveryVisualizer
	// errorHandler    *ErrorHandler          // TODO: Define ErrorHandler type
	// errorReporting  *EnhancedErrorReporting // TODO: Define EnhancedErrorReporting type
	advancedQuery       *AdvancedQuery
	realTimeMonitor     *RealTimeMonitor
	sdkIntegration      *SDKIntegration
	discoveredResources []models.Resource
	metrics             map[string]interface{}
	lastDiscoveryTime   time.Time
	mu                  sync.RWMutex
}

// DiscoveryPlugin represents a configurable discovery plugin
type DiscoveryPlugin struct {
	Name         string            `yaml:"name"`
	Enabled      bool              `yaml:"enabled"`
	Priority     int               `yaml:"priority"`
	Dependencies []string          `yaml:"dependencies"`
	Config       map[string]string `yaml:"config"`
	DiscoveryFn  func(context.Context, string, string) ([]models.Resource, error)
}

// ResourceHierarchy represents hierarchical resource relationships
type ResourceHierarchy struct {
	Parent       *models.Resource
	Children     []*ResourceHierarchy
	Dependencies []string
	Level        int
}

// DiscoveryFilter provides intelligent resource filtering
type DiscoveryFilter struct {
	IncludeTags   map[string]string `yaml:"include_tags"`
	ExcludeTags   map[string]string `yaml:"exclude_tags"`
	ResourceTypes []string          `yaml:"resource_types"`
	AgeThreshold  time.Duration     `yaml:"age_threshold"`
	UsagePatterns []string          `yaml:"usage_patterns"`
	CostThreshold float64           `yaml:"cost_threshold"`
	SecurityScore int               `yaml:"security_score"`
	Environment   string            `yaml:"environment"`
}

// DiscoveryQuality represents discovery quality metrics
type DiscoveryQuality struct {
	Completeness float64            `json:"completeness"`
	Accuracy     float64            `json:"accuracy"`
	Freshness    time.Duration      `json:"freshness"`
	Coverage     map[string]float64 `json:"coverage"`
	Errors       []string           `json:"errors"`
}

// NewEnhancedDiscoverer creates a new enhanced discoverer
func NewEnhancedDiscoverer(cfg *config.Config) *EnhancedDiscoverer {
	// Initialize service configuration
	serviceConfig := config.NewDiscoveryServicesConfig("discovery-services.json")
	services := serviceConfig.GetServicesMap()

	// Default regions if not specified in config
	defaultRegions := []string{"us-east-1", "us-west-2", "eu-west-1"}
	if cfg.Discovery.Regions != nil && len(cfg.Discovery.Regions) > 0 {
		defaultRegions = cfg.Discovery.Regions
	}

	return &EnhancedDiscoverer{
		config:              cfg,
		cache:               cache.NewDiscoveryCache(),
		plugins:             make(map[string]*DiscoveryPlugin),
		hierarchy:           &ResourceHierarchy{},
		filters:             &DiscoveryFilter{},
		progressTracker:     NewProgressTracker([]string{"aws", "azure", "gcp"}, defaultRegions, services),
		// errorHandler:    NewErrorHandler(nil), // Use default retry config
		// errorReporting:  NewEnhancedErrorReporting(),
		sdkIntegration:      NewSDKIntegration(),
		discoveredResources: []models.Resource{},
		metrics:             make(map[string]interface{}),
		lastDiscoveryTime:   time.Now(),
	}
}

// DiscoverResources performs resource discovery across all configured providers
func (ed *EnhancedDiscoverer) DiscoverResources(ctx context.Context) ([]models.Resource, error) {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	var allResources []models.Resource
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, 10)

	// Check cache first
	if ed.cache != nil {
		if cached, found := ed.cache.Get("all_resources"); found {
			if resources, ok := cached.([]models.Resource); ok {
				log.Printf("Returning %d cached resources", len(resources))
				return resources, nil
			}
		}
	}

	providers := []string{"aws", "azure", "gcp"}
	regions := ed.config.Discovery.Regions
	if len(regions) == 0 {
		regions = []string{"us-east-1", "us-west-2", "eu-west-1"}
	}

	// Discover resources for each provider in parallel
	for _, provider := range providers {
		for _, region := range regions {
			wg.Add(1)
			go func(p, r string) {
				defer wg.Done()

				// Use context with timeout
				discoverCtx, cancel := context.WithTimeout(ctx, ed.config.Discovery.Timeout)
				defer cancel()

				resources, err := ed.discoverProviderResources(discoverCtx, p, r)
				if err != nil {
					select {
					case errChan <- fmt.Errorf("discovery failed for %s/%s: %w", p, r, err):
					default:
					}
					return
				}

				mu.Lock()
				allResources = append(allResources, resources...)
				mu.Unlock()
			}(provider, region)
		}
	}

	// Wait for all discoveries to complete
	wg.Wait()
	close(errChan)

	// Collect errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 && len(allResources) == 0 {
		return nil, fmt.Errorf("discovery failed: %v", errors)
	}

	// Cache results
	if ed.cache != nil {
		ed.cache.Set("all_resources", allResources)
	}

	log.Printf("Discovered %d total resources across all providers", len(allResources))
	return allResources, nil
}

// discoverProviderResources discovers resources for a specific provider and region
func (ed *EnhancedDiscoverer) discoverProviderResources(ctx context.Context, provider, region string) ([]models.Resource, error) {
	// Check if we have a plugin for this provider
	if plugin, exists := ed.plugins[provider]; exists && plugin.Enabled {
		return plugin.DiscoveryFn(ctx, provider, region)
	}

	// Fallback to built-in discovery
	switch provider {
	case "aws":
		return ed.discoverAWSResourcesBuiltin(ctx, region), nil
	case "azure":
		return ed.discoverAzureResourcesBuiltin(ctx, region), nil
	case "gcp":
		return ed.discoverGCPResourcesBuiltin(ctx, region), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// RegisterPlugin registers a discovery plugin
func (ed *EnhancedDiscoverer) RegisterPlugin(plugin *DiscoveryPlugin) {
	ed.mu.Lock()
	defer ed.mu.Unlock()
	ed.plugins[plugin.Name] = plugin
}

// SetFilter sets the discovery filter
func (ed *EnhancedDiscoverer) SetFilter(filter *DiscoveryFilter) {
	ed.mu.Lock()
	defer ed.mu.Unlock()
	ed.filters = filter
}

// DiscoverAllResourcesEnhanced performs comprehensive resource discovery
func (ed *EnhancedDiscoverer) DiscoverAllResourcesEnhanced(ctx context.Context, providers []string, regions []string) ([]models.Resource, error) {
	start := time.Now()
	log.Printf("Starting enhanced discovery for providers: %v, regions: %v", providers, regions)

	var allResources []models.Resource
	var discoveryErrors []error

	// Check cache first
	cacheKey := fmt.Sprintf("discovery:%s:%s", providers[0], regions[0])
	if cached, found := ed.cache.Get(cacheKey); found {
		log.Printf("Using cached discovery results")
		return cached.([]models.Resource), nil
	}

	// Discover resources by provider
	for _, provider := range providers {
		for _, region := range regions {
			resources, err := ed.discoverProviderRegionEnhanced(ctx, provider, region)
			if err != nil {
				discoveryErrors = append(discoveryErrors, fmt.Errorf("discovery failed for %s/%s: %w", provider, region, err))
				continue
			}
			allResources = append(allResources, resources...)
		}
	}

	// Apply filters
	filteredResources := ed.applyFilters(allResources)

	// Build hierarchy
	ed.buildResourceHierarchy(filteredResources)

	// Cache results
	ed.cache.Set(cacheKey, filteredResources)

	log.Printf("Enhanced discovery completed in %v. Found %d resources", time.Since(start), len(filteredResources))

	return filteredResources, nil
}

// discoverProviderRegionEnhanced discovers resources for a specific provider and region
func (ed *EnhancedDiscoverer) discoverProviderRegionEnhanced(ctx context.Context, provider, region string) ([]models.Resource, error) {
	switch provider {
	case "aws":
		return ed.discoverAWSEnhanced(ctx, region)
	case "azure":
		return ed.discoverAzureEnhanced(ctx, region)
	case "gcp":
		return ed.discoverGCPEnhanced(ctx, region)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// discoverAWSEnhanced performs comprehensive AWS discovery
func (ed *EnhancedDiscoverer) discoverAWSEnhanced(ctx context.Context, region string) ([]models.Resource, error) {
	var resources []models.Resource

	// Core compute and networking (existing)
	resources = append(resources, ed.discoverAWSEC2(ctx, region)...)
	resources = append(resources, ed.discoverAWSRDS(ctx, region)...)
	resources = append(resources, ed.discoverAWSLambda(ctx, region)...)
	resources = append(resources, ed.discoverAWSCloudFormation(ctx, region)...)
	resources = append(resources, ed.discoverAWSElastiCache(ctx, region)...)
	resources = append(resources, ed.discoverAWSECS(ctx, region)...)
	resources = append(resources, ed.discoverAWSEKS(ctx, region)...)
	resources = append(resources, ed.discoverAWSSQS(ctx, region)...)
	resources = append(resources, ed.discoverAWSSNS(ctx, region)...)
	resources = append(resources, ed.discoverAWSDynamoDB(ctx, region)...)
	resources = append(resources, ed.discoverAWSAutoScaling(ctx, region)...)

	// NEW: Security services
	resources = append(resources, ed.discoverAWSWAF(ctx, region)...)
	resources = append(resources, ed.discoverAWSShield(ctx, region)...)
	resources = append(resources, ed.discoverAWSConfig(ctx, region)...)
	resources = append(resources, ed.discoverAWSGuardDuty(ctx, region)...)

	// NEW: CDN and API services
	resources = append(resources, ed.discoverAWSCloudFront(ctx, region)...)
	resources = append(resources, ed.discoverAWSAPIGateway(ctx, region)...)

	// NEW: Data and analytics services
	resources = append(resources, ed.discoverAWSGlue(ctx, region)...)
	resources = append(resources, ed.discoverAWSRedshift(ctx, region)...)
	resources = append(resources, ed.discoverAWSElasticsearch(ctx, region)...)

	// NEW: Monitoring and operations
	resources = append(resources, ed.discoverAWSCloudWatch(ctx, region)...)
	resources = append(resources, ed.discoverAWSSystemsManager(ctx, region)...)

	// NEW: Workflow and orchestration
	resources = append(resources, ed.discoverAWSStepFunctions(ctx, region)...)

	// Global services (only check once)
	if region == "us-east-1" {
		resources = append(resources, ed.discoverAWSS3(ctx)...)
		resources = append(resources, ed.discoverAWSIAM(ctx)...)
		resources = append(resources, ed.discoverAWSRoute53(ctx)...)
	}

	return resources, nil
}

// discoverAzureEnhanced performs comprehensive Azure discovery
func (ed *EnhancedDiscoverer) discoverAzureEnhanced(ctx context.Context, region string) ([]models.Resource, error) {
	var resources []models.Resource

	// Core services (existing)
	resources = append(resources, ed.discoverAzureVMs(ctx, region)...)
	resources = append(resources, ed.discoverAzureStorageAccounts(ctx, region)...)
	resources = append(resources, ed.discoverAzureSQLDatabases(ctx, region)...)
	resources = append(resources, ed.discoverAzureWebApps(ctx, region)...)
	resources = append(resources, ed.discoverAzureVirtualNetworks(ctx, region)...)
	resources = append(resources, ed.discoverAzureLoadBalancers(ctx, region)...)
	resources = append(resources, ed.discoverAzureKeyVaults(ctx, region)...)
	resources = append(resources, ed.discoverAzureResourceGroups(ctx, region)...)

	// NEW: Serverless and workflow services
	resources = append(resources, ed.discoverAzureFunctions(ctx, region)...)
	resources = append(resources, ed.discoverAzureLogicApps(ctx, region)...)

	// NEW: Messaging services
	resources = append(resources, ed.discoverAzureEventHubs(ctx, region)...)
	resources = append(resources, ed.discoverAzureServiceBus(ctx, region)...)

	// NEW: Data services
	resources = append(resources, ed.discoverAzureCosmosDB(ctx, region)...)
	resources = append(resources, ed.discoverAzureDataFactory(ctx, region)...)
	resources = append(resources, ed.discoverAzureSynapseAnalytics(ctx, region)...)

	// NEW: Monitoring and governance
	resources = append(resources, ed.discoverAzureApplicationInsights(ctx, region)...)
	resources = append(resources, ed.discoverAzurePolicy(ctx, region)...)

	// NEW: Security services
	resources = append(resources, ed.discoverAzureBastion(ctx, region)...)

	return resources, nil
}

// discoverGCPEnhanced performs comprehensive GCP discovery
func (ed *EnhancedDiscoverer) discoverGCPEnhanced(ctx context.Context, region string) ([]models.Resource, error) {
	var resources []models.Resource

	// Core services (existing)
	resources = append(resources, ed.discoverGCPComputeInstances(ctx, region)...)
	resources = append(resources, ed.discoverGCPStorageBuckets(ctx, region)...)
	resources = append(resources, ed.discoverGCPGKEClusters(ctx, region)...)
	resources = append(resources, ed.discoverGCPCloudSQL(ctx, region)...)
	resources = append(resources, ed.discoverGCPVPCNetworks(ctx, region)...)

	// NEW: Serverless and container services
	resources = append(resources, ed.discoverGCPCloudFunctions(ctx, region)...)
	resources = append(resources, ed.discoverGCPCloudRun(ctx, region)...)

	// NEW: CI/CD and messaging
	resources = append(resources, ed.discoverGCPCloudBuild(ctx, region)...)
	resources = append(resources, ed.discoverGCPCloudPubSub(ctx, region)...)

	// NEW: Data services
	resources = append(resources, ed.discoverGCPBigQuery(ctx, region)...)
	resources = append(resources, ed.discoverGCPCloudSpanner(ctx, region)...)
	resources = append(resources, ed.discoverGCPCloudFirestore(ctx, region)...)

	// NEW: Security and monitoring
	resources = append(resources, ed.discoverGCPCloudArmor(ctx, region)...)
	resources = append(resources, ed.discoverGCPCloudMonitoring(ctx, region)...)
	resources = append(resources, ed.discoverGCPCloudLogging(ctx, region)...)

	return resources, nil
}

// applyFilters applies intelligent filtering to resources
func (ed *EnhancedDiscoverer) applyFilters(resources []models.Resource) []models.Resource {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	if ed.filters == nil {
		return resources
	}

	var filtered []models.Resource

	for _, resource := range resources {
		if ed.shouldIncludeResource(resource) {
			filtered = append(filtered, resource)
		}
	}

	return filtered
}

// shouldIncludeResource determines if a resource should be included based on filters
func (ed *EnhancedDiscoverer) shouldIncludeResource(resource models.Resource) bool {
	filter := ed.filters

	// Check resource type filter
	if len(filter.ResourceTypes) > 0 {
		found := false
		for _, resourceType := range filter.ResourceTypes {
			if resource.Type == resourceType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check include tags
	if len(filter.IncludeTags) > 0 {
		resourceTags := resource.GetTagsAsMap()
		for key, value := range filter.IncludeTags {
			if resourceValue, exists := resourceTags[key]; !exists || resourceValue != value {
				return false
			}
		}
	}

	// Check exclude tags
	if len(filter.ExcludeTags) > 0 {
		resourceTags := resource.GetTagsAsMap()
		for key, value := range filter.ExcludeTags {
			if resourceValue, exists := resourceTags[key]; exists && resourceValue == value {
				return false
			}
		}
	}

	// Check age threshold
	if filter.AgeThreshold > 0 {
		if time.Since(resource.CreatedAt) < filter.AgeThreshold {
			return false
		}
	}

	// Check environment filter
	if filter.Environment != "" {
		resourceTags := resource.GetTagsAsMap()
		if envValue, exists := resourceTags["Environment"]; !exists || envValue != filter.Environment {
			return false
		}
	}

	return true
}

// buildResourceHierarchy builds hierarchical relationships between resources
func (ed *EnhancedDiscoverer) buildResourceHierarchy(resources []models.Resource) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	// Group resources by type
	resourceMap := make(map[string][]models.Resource)
	for _, resource := range resources {
		resourceMap[resource.Type] = append(resourceMap[resource.Type], resource)
	}

	// Build hierarchy based on dependencies
	ed.hierarchy = &ResourceHierarchy{
		Level: 0,
	}

	// Example hierarchy: VPC -> Subnets -> EC2 instances
	if vpcs, exists := resourceMap["aws_vpc"]; exists {
		for _, vpc := range vpcs {
			vpcHierarchy := &ResourceHierarchy{
				Parent: &vpc,
				Level:  1,
			}

			// Find subnets in this VPC
			if subnets, exists := resourceMap["aws_subnet"]; exists {
				for _, subnet := range subnets {
					subnetTags := subnet.GetTagsAsMap()
					if vpcTag, exists := subnetTags["VPC"]; exists && vpcTag == vpc.ID {
						vpcHierarchy.Children = append(vpcHierarchy.Children, &ResourceHierarchy{Parent: &subnet, Level: 2})
					}
				}
			}

			// Find EC2 instances in this VPC
			if instances, exists := resourceMap["aws_instance"]; exists {
				for _, instance := range instances {
					instanceTags := instance.GetTagsAsMap()
					if vpcTag, exists := instanceTags["VPC"]; exists && vpcTag == vpc.ID {
						vpcHierarchy.Children = append(vpcHierarchy.Children, &ResourceHierarchy{Parent: &instance, Level: 2})
					}
				}
			}

			ed.hierarchy.Children = append(ed.hierarchy.Children, &ResourceHierarchy{Parent: &vpc})
		}
	}
}

// GetDiscoveryQuality returns quality metrics for the discovery process
func (ed *EnhancedDiscoverer) GetDiscoveryQuality() DiscoveryQuality {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	// Calculate completeness based on discovered vs cached resources
	var totalDiscovered, totalCached int
	if ed.cache != nil {
		totalCached = ed.cache.GetSize()
	}
	totalDiscovered = len(ed.discoveredResources)
	
	completeness := 0.0
	if totalCached > 0 {
		completeness = float64(totalDiscovered) / float64(totalCached)
		if completeness > 1.0 {
			completeness = 1.0
		}
	} else if totalDiscovered > 0 {
		completeness = 1.0 // If we have discoveries but no cache, assume complete
	}

	// Calculate accuracy based on validation success rate
	var validationSuccess, validationTotal int
	// Since metrics is map[string]interface{}, we need type assertions
	if valAttempts, ok := ed.metrics["validation_attempts"].(int); ok {
		validationTotal = valAttempts
	}
	if valSuccess, ok := ed.metrics["validation_success"].(int); ok {
		validationSuccess = valSuccess
	}
	
	accuracy := 0.0
	if validationTotal > 0 {
		accuracy = float64(validationSuccess) / float64(validationTotal)
	} else {
		// No validations performed yet, use discovery success rate
		var discoverySuccess, discoveryTotal int
		if discTotal, ok := ed.metrics["discovery_total"].(int); ok {
			discoveryTotal = discTotal
		}
		if discSuccess, ok := ed.metrics["discovery_success"].(int); ok {
			discoverySuccess = discSuccess
		}
		if discoveryTotal > 0 {
			accuracy = float64(discoverySuccess) / float64(discoveryTotal)
		}
	}

	// Calculate freshness based on last discovery time
	freshness := time.Since(ed.lastDiscoveryTime)
	if ed.lastDiscoveryTime.IsZero() {
		freshness = time.Duration(0) // No discovery yet
	}

	// Calculate coverage by provider
	coverage := make(map[string]float64)
	providerCounts := make(map[string]int)
	providerExpected := map[string]int{
		"aws":          15, // Expected number of AWS resource types
		"azure":        12, // Expected number of Azure resource types
		"gcp":          10, // Expected number of GCP resource types
		"digitalocean": 8,  // Expected number of DO resource types
	}

	// Count discovered resource types per provider
	resourceTypes := make(map[string]map[string]bool)
	for _, resource := range ed.discoveredResources {
		if resourceTypes[resource.Provider] == nil {
			resourceTypes[resource.Provider] = make(map[string]bool)
		}
		resourceTypes[resource.Provider][resource.Type] = true
	}

	// Calculate coverage percentage
	for provider, expected := range providerExpected {
		if types, exists := resourceTypes[provider]; exists {
			providerCounts[provider] = len(types)
			coverage[provider] = float64(len(types)) / float64(expected)
			if coverage[provider] > 1.0 {
				coverage[provider] = 1.0
			}
		} else {
			coverage[provider] = 0.0
		}
	}

	// Collect errors from metrics
	var errors []string
	if lastErr, ok := ed.metrics["last_error"].(string); ok && lastErr != "" {
		errors = append(errors, lastErr)
	}

	return DiscoveryQuality{
		Completeness: completeness,
		Accuracy:     accuracy,
		Freshness:    freshness,
		Coverage:     coverage,
		Errors:       errors,
	}
}

// Actual discovery methods for new AWS services
func (ed *EnhancedDiscoverer) discoverAWSWAF(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover WAF Web ACLs
	cmd := exec.CommandContext(ctx, "aws", "wafv2", "list-web-acls",
		"--region", region,
		"--scope", "REGIONAL",
		"--query", "WebACLs[*].[Id,Name,Description,ARN]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering WAF Web ACLs in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for Web ACL
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_wafv2_web_acl",
					Name:       parts[1],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"description": parts[2], "arn": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSShield(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Shield protections
	cmd := exec.CommandContext(ctx, "aws", "shield", "list-protections",
		"--region", region,
		"--query", "Protections[*].[Id,Name,ResourceArn,ProtectionArn]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Shield protections in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for protection
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_shield_protection",
					Name:       parts[1],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"resource_arn": parts[2], "protection_arn": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSConfig(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Config recorders
	cmd := exec.CommandContext(ctx, "aws", "configservice", "describe-configuration-recorders",
		"--region", region,
		"--query", "ConfigurationRecorders[*].[name,roleARN,recordingGroup.allSupported,recordingGroup.includeGlobalResources]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Config recorders in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for recorder
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_config_configuration_recorder",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"role_arn": parts[1], "all_supported": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSGuardDuty(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover GuardDuty detectors
	cmd := exec.CommandContext(ctx, "aws", "guardduty", "list-detectors",
		"--region", region,
		"--query", "DetectorIds",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GuardDuty detectors in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for detector ID
			detectorId := strings.Trim(line, "\"")
			resource := models.Resource{
				ID:         detectorId,
				Type:       "aws_guardduty_detector",
				Name:       detectorId,
				Region:     region,
				Provider:   "aws",
				CreatedAt:  time.Now(),
				Tags:       map[string]string{},
				Properties: map[string]interface{}{},
			}
			resources = append(resources, resource)
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSCloudFront(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover CloudFront distributions
	cmd := exec.CommandContext(ctx, "aws", "cloudfront", "list-distributions",
		"--query", "DistributionList.Items[*].[Id,DomainName,Status,LastModifiedTime,Origins.Items[0].DomainName]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering CloudFront distributions: %v", err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for distribution
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_cloudfront_distribution",
					Name:       parts[0],
					Region:     "global", // CloudFront is global
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"domain_name": parts[1], "status": parts[2], "last_modified": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSAPIGateway(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover API Gateway REST APIs
	cmd := exec.CommandContext(ctx, "aws", "apigateway", "get-rest-apis",
		"--region", region,
		"--query", "items[*].[id,name,description,createdDate,version]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering API Gateway REST APIs in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for API
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_api_gateway_rest_api",
					Name:       parts[1],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"description": parts[2], "created_date": parts[3], "version": parts[4]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSGlue(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Glue databases
	cmd := exec.CommandContext(ctx, "aws", "glue", "get-databases",
		"--region", region,
		"--query", "DatabaseList[*].[Name,Description,CatalogId,CreateTime]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Glue databases in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for database
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_glue_catalog_database",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"description": parts[1], "catalog_id": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSRedshift(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Redshift clusters
	cmd := exec.CommandContext(ctx, "aws", "redshift", "describe-clusters",
		"--region", region,
		"--query", "Clusters[*].[ClusterIdentifier,NodeType,ClusterStatus,ClusterCreateTime,VpcId]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Redshift clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for cluster
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_redshift_cluster",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"VPC": parts[4]},
					Properties: map[string]interface{}{"node_type": parts[1], "status": parts[2], "create_time": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSElasticsearch(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover OpenSearch domains (formerly Elasticsearch)
	cmd := exec.CommandContext(ctx, "aws", "opensearch", "list-domain-names",
		"--region", region,
		"--query", "DomainNames[*].[DomainName,EngineType]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering OpenSearch domains in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for domain
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_opensearch_domain",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"engine_type": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSCloudWatch(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover CloudWatch log groups
	cmd := exec.CommandContext(ctx, "aws", "logs", "describe-log-groups",
		"--region", region,
		"--query", "logGroups[*].[logGroupName,creationTime,storedBytes,metricFilterCount]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering CloudWatch log groups in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for log group
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_cloudwatch_log_group",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"creation_time": parts[1], "stored_bytes": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSSystemsManager(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Systems Manager parameters
	cmd := exec.CommandContext(ctx, "aws", "ssm", "describe-parameters",
		"--region", region,
		"--query", "Parameters[*].[Name,Type,Description,LastModifiedDate]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Systems Manager parameters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for parameter
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_ssm_parameter",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"type": parts[1], "description": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSStepFunctions(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Step Functions state machines
	cmd := exec.CommandContext(ctx, "aws", "stepfunctions", "list-state-machines",
		"--region", region,
		"--query", "stateMachines[*].[name,stateMachineArn,type,creationDate]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Step Functions state machines in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for state machine
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_sfn_state_machine",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"arn": parts[1], "type": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// discoverAWSResourcesBuiltin performs built-in AWS discovery without plugins
func (ed *EnhancedDiscoverer) discoverAWSResourcesBuiltin(ctx context.Context, region string) []models.Resource {
	var allResources []models.Resource

	// Discover regional AWS resource types
	resourceTypes := []func(context.Context, string) []models.Resource{
		ed.discoverAWSEC2,
		ed.discoverAWSRDS,
		ed.discoverAWSLambda,
		ed.discoverAWSVPC,
		ed.discoverAWSECS,
		ed.discoverAWSEKS,
		ed.discoverAWSWAF,
		ed.discoverAWSShield,
		ed.discoverAWSConfig,
		ed.discoverAWSCloudWatch,
		ed.discoverAWSGuardDuty,
		ed.discoverAWSStepFunctions,
	}

	for _, discoverFunc := range resourceTypes {
		resources := discoverFunc(ctx, region)
		allResources = append(allResources, resources...)
	}

	// Discover global AWS services (don't need region)
	globalResources := ed.discoverAWSS3(ctx)
	allResources = append(allResources, globalResources...)
	
	iamResources := ed.discoverAWSIAM(ctx)
	allResources = append(allResources, iamResources...)

	return allResources
}

// discoverAzureResourcesBuiltin performs built-in Azure discovery without plugins
func (ed *EnhancedDiscoverer) discoverAzureResourcesBuiltin(ctx context.Context, region string) []models.Resource {
	var allResources []models.Resource

	// Discover various Azure resource types using Azure CLI
	resourceTypes := []struct {
		resourceType string
		cliCommand   []string
	}{
		{"vm", []string{"az", "vm", "list", "--output", "json"}},
		{"storage", []string{"az", "storage", "account", "list", "--output", "json"}},
		{"network", []string{"az", "network", "vnet", "list", "--output", "json"}},
		{"sql", []string{"az", "sql", "server", "list", "--output", "json"}},
		{"webapp", []string{"az", "webapp", "list", "--output", "json"}},
		{"keyvault", []string{"az", "keyvault", "list", "--output", "json"}},
	}

	for _, rt := range resourceTypes {
		cmd := exec.CommandContext(ctx, rt.cliCommand[0], rt.cliCommand[1:]...)
		output, err := cmd.Output()
		if err != nil {
			log.Printf("Error discovering Azure %s resources: %v", rt.resourceType, err)
			continue
		}

		// Parse JSON output
		var resources []map[string]interface{}
		if err := json.Unmarshal(output, &resources); err == nil {
			for _, res := range resources {
				if id, ok := res["id"].(string); ok {
					name := ""
					if n, ok := res["name"].(string); ok {
						name = n
					}
					location := region
					if loc, ok := res["location"].(string); ok {
						location = loc
					}

					resource := models.Resource{
						ID:         id,
						Type:       fmt.Sprintf("azure_%s", rt.resourceType),
						Name:       name,
						Region:     location,
						Provider:   "azure",
						CreatedAt:  time.Now(),
						Tags:       make(map[string]string),
						Properties: res,
					}
					allResources = append(allResources, resource)
				}
			}
		}
	}

	return allResources
}

// discoverGCPResourcesBuiltin performs built-in GCP discovery without plugins
func (ed *EnhancedDiscoverer) discoverGCPResourcesBuiltin(ctx context.Context, region string) []models.Resource {
	var allResources []models.Resource

	// Discover various GCP resource types using gcloud CLI
	resourceTypes := []struct {
		resourceType string
		cliCommand   []string
	}{
		{"compute_instance", []string{"gcloud", "compute", "instances", "list", "--format=json"}},
		{"storage_bucket", []string{"gcloud", "storage", "buckets", "list", "--format=json"}},
		{"sql_instance", []string{"gcloud", "sql", "instances", "list", "--format=json"}},
		{"container_cluster", []string{"gcloud", "container", "clusters", "list", "--format=json"}},
		{"function", []string{"gcloud", "functions", "list", "--format=json"}},
		{"vpc_network", []string{"gcloud", "compute", "networks", "list", "--format=json"}},
	}

	for _, rt := range resourceTypes {
		cmd := exec.CommandContext(ctx, rt.cliCommand[0], rt.cliCommand[1:]...)
		if region != "" && region != "global" {
			cmd.Args = append(cmd.Args, "--region", region)
		}

		output, err := cmd.Output()
		if err != nil {
			log.Printf("Error discovering GCP %s resources: %v", rt.resourceType, err)
			continue
		}

		// Parse JSON output
		var resources []map[string]interface{}
		if err := json.Unmarshal(output, &resources); err == nil {
			for _, res := range resources {
				id := ""
				if selfLink, ok := res["selfLink"].(string); ok {
					id = selfLink
				} else if n, ok := res["name"].(string); ok {
					id = n
				}

				name := ""
				if n, ok := res["name"].(string); ok {
					name = n
				}

				location := region
				if zone, ok := res["zone"].(string); ok {
					location = zone
				} else if loc, ok := res["location"].(string); ok {
					location = loc
				}

				resource := models.Resource{
					ID:         id,
					Type:       fmt.Sprintf("gcp_%s", rt.resourceType),
					Name:       name,
					Region:     location,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       make(map[string]string),
					Properties: res,
				}
				allResources = append(allResources, resource)
			}
		}
	}

	return allResources
}

// Actual discovery methods for AWS services using CLI
func (ed *EnhancedDiscoverer) discoverAWSEC2(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover EC2 instances
	cmd := exec.CommandContext(ctx, "aws", "ec2", "describe-instances",
		"--region", region,
		"--query", "Reservations[*].Instances[*].[InstanceId,InstanceType,State.Name,LaunchTime,Tags[?Key==`Name`].Value|[0],VpcId,SubnetId]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering EC2 instances in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	// This is a simplified implementation - in production, you'd use proper JSON parsing
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "i-") { // Basic check for instance ID
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_instance",
					Name:       parts[4],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(), // Would parse actual launch time
					Tags:       map[string]string{"VPC": parts[5], "Subnet": parts[6]},
					Properties: map[string]interface{}{"instance_type": parts[1], "state": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSRDS(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover RDS instances
	cmd := exec.CommandContext(ctx, "aws", "rds", "describe-db-instances",
		"--region", region,
		"--query", "DBInstances[*].[DBInstanceIdentifier,DBInstanceClass,Engine,DBInstanceStatus,InstanceCreateTime,DBSubnetGroup.VpcId]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering RDS instances in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "db-") || strings.Contains(line, "rds-") { // Basic check for RDS instance
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_db_instance",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(), // Would parse actual creation time
					Tags:       map[string]string{"VPC": parts[5]},
					Properties: map[string]interface{}{"instance_class": parts[1], "engine": parts[2], "status": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSLambda(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Lambda functions
	cmd := exec.CommandContext(ctx, "aws", "lambda", "list-functions",
		"--region", region,
		"--query", "Functions[*].[FunctionName,Runtime,CodeSize,LastModified,Description]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Lambda functions in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for function name
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_lambda_function",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(), // Would parse actual creation time
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"runtime": parts[1], "code_size": parts[2], "last_modified": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSS3(ctx context.Context) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover S3 buckets
	cmd := exec.CommandContext(ctx, "aws", "s3api", "list-buckets",
		"--query", "Buckets[*].[Name,CreationDate]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering S3 buckets: %v", err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for bucket name
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_s3_bucket",
					Name:       parts[0],
					Region:     "global", // S3 is global
					Provider:   "aws",
					CreatedAt:  time.Now(), // Would parse actual creation time
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"creation_date": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSIAM(ctx context.Context) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover IAM users
	cmd := exec.CommandContext(ctx, "aws", "iam", "list-users",
		"--query", "Users[*].[UserName,CreateDate,PasswordLastUsed]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering IAM users: %v", err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for username
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_iam_user",
					Name:       parts[0],
					Region:     "global", // IAM is global
					Provider:   "aws",
					CreatedAt:  time.Now(), // Would parse actual creation time
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"create_date": parts[1], "password_last_used": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSRoute53(ctx context.Context) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Route53 hosted zones
	cmd := exec.CommandContext(ctx, "aws", "route53", "list-hosted-zones",
		"--query", "HostedZones[*].[Id,Name,CallerReference]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Route53 hosted zones: %v", err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/hostedzone/") { // Basic check for hosted zone ID
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_route53_zone",
					Name:       parts[1],
					Region:     "global", // Route53 is global
					Provider:   "aws",
					CreatedAt:  time.Now(), // Would parse actual creation time
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"caller_reference": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSCloudFormation(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover CloudFormation stacks
	cmd := exec.CommandContext(ctx, "aws", "cloudformation", "list-stacks",
		"--region", region,
		"--stack-status-filter", "CREATE_COMPLETE", "UPDATE_COMPLETE",
		"--query", "StackSummaries[*].[StackName,StackStatus,CreationTime,LastUpdatedTime]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering CloudFormation stacks in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for stack name
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_cloudformation_stack",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(), // Would parse actual creation time
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"status": parts[1], "creation_time": parts[2], "last_updated": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSElastiCache(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover ElastiCache clusters
	cmd := exec.CommandContext(ctx, "aws", "elasticache", "describe-cache-clusters",
		"--region", region,
		"--query", "CacheClusters[*].[CacheClusterId,Engine,CacheNodeType,CacheClusterStatus,CacheClusterCreateTime]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering ElastiCache clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for cluster ID
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_elasticache_cluster",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(), // Would parse actual creation time
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"engine": parts[1], "node_type": parts[2], "status": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSECS(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover ECS clusters
	cmd := exec.CommandContext(ctx, "aws", "ecs", "list-clusters",
		"--region", region,
		"--query", "clusterArns",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering ECS clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "cluster/") { // Basic check for cluster ARN
			clusterName := strings.Split(line, "/")[1]
			resource := models.Resource{
				ID:         clusterName,
				Type:       "aws_ecs_cluster",
				Name:       clusterName,
				Region:     region,
				Provider:   "aws",
				CreatedAt:  time.Now(), // Would parse actual creation time
				Tags:       map[string]string{},
				Properties: map[string]interface{}{"arn": line},
			}
			resources = append(resources, resource)
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSEKS(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover EKS clusters
	cmd := exec.CommandContext(ctx, "aws", "eks", "list-clusters",
		"--region", region,
		"--query", "clusters",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering EKS clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for cluster name
			clusterName := strings.Trim(line, "\"")
			resource := models.Resource{
				ID:         clusterName,
				Type:       "aws_eks_cluster",
				Name:       clusterName,
				Region:     region,
				Provider:   "aws",
				CreatedAt:  time.Now(), // Would parse actual creation time
				Tags:       map[string]string{},
				Properties: map[string]interface{}{},
			}
			resources = append(resources, resource)
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSSQS(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover SQS queues
	cmd := exec.CommandContext(ctx, "aws", "sqs", "list-queues",
		"--region", region,
		"--query", "QueueUrls",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering SQS queues in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "sqs.") { // Basic check for SQS URL
			queueName := strings.Split(line, "/")[4]
			resource := models.Resource{
				ID:         queueName,
				Type:       "aws_sqs_queue",
				Name:       queueName,
				Region:     region,
				Provider:   "aws",
				CreatedAt:  time.Now(), // Would parse actual creation time
				Tags:       map[string]string{},
				Properties: map[string]interface{}{"url": line},
			}
			resources = append(resources, resource)
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSSNS(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover SNS topics
	cmd := exec.CommandContext(ctx, "aws", "sns", "list-topics",
		"--region", region,
		"--query", "Topics[*].TopicArn",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering SNS topics in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "sns.") { // Basic check for SNS ARN
			topicName := strings.Split(line, ":")[5]
			resource := models.Resource{
				ID:         topicName,
				Type:       "aws_sns_topic",
				Name:       topicName,
				Region:     region,
				Provider:   "aws",
				CreatedAt:  time.Now(), // Would parse actual creation time
				Tags:       map[string]string{},
				Properties: map[string]interface{}{"arn": line},
			}
			resources = append(resources, resource)
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSDynamoDB(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover DynamoDB tables
	cmd := exec.CommandContext(ctx, "aws", "dynamodb", "list-tables",
		"--region", region,
		"--query", "TableNames",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering DynamoDB tables in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for table name
			tableName := strings.Trim(line, "\"")
			resource := models.Resource{
				ID:         tableName,
				Type:       "aws_dynamodb_table",
				Name:       tableName,
				Region:     region,
				Provider:   "aws",
				CreatedAt:  time.Now(), // Would parse actual creation time
				Tags:       map[string]string{},
				Properties: map[string]interface{}{},
			}
			resources = append(resources, resource)
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSAutoScaling(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Auto Scaling groups
	cmd := exec.CommandContext(ctx, "aws", "autoscaling", "describe-auto-scaling-groups",
		"--region", region,
		"--query", "AutoScalingGroups[*].[AutoScalingGroupName,MinSize,MaxSize,DesiredCapacity,LaunchConfigurationName]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Auto Scaling groups in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for ASG name
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_autoscaling_group",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(), // Would parse actual creation time
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"min_size": parts[1], "max_size": parts[2], "desired_capacity": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Actual discovery methods for Azure services using CLI
func (ed *EnhancedDiscoverer) discoverAzureVMs(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Virtual Machines
	cmd := exec.CommandContext(ctx, "az", "vm", "list",
		"--query", "[?location=='"+region+"'].[id,name,vmSize,powerState,provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure VMs in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/virtualMachines/") { // Basic check for VM ID
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				vmName := strings.Split(parts[0], "/")[8] // Extract VM name from ID
				resource := models.Resource{
					ID:         vmName,
					Type:       "azurerm_virtual_machine",
					Name:       vmName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[4]},
					Properties: map[string]interface{}{"vm_size": parts[1], "power_state": parts[2], "provisioning_state": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureStorageAccounts(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Storage Accounts
	cmd := exec.CommandContext(ctx, "az", "storage", "account", "list",
		"--query", "[?location=='"+region+"'].[id,name,sku.name,statusOfPrimary,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Storage Accounts in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/storageAccounts/") { // Basic check for storage account ID
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				accountName := strings.Split(parts[0], "/")[8] // Extract account name from ID
				resource := models.Resource{
					ID:         accountName,
					Type:       "azurerm_storage_account",
					Name:       accountName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[3]},
					Properties: map[string]interface{}{"sku": parts[1], "status": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureSQLDatabases(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover SQL Databases
	cmd := exec.CommandContext(ctx, "az", "sql", "db", "list",
		"--query", "[?location=='"+region+"'].[id,name,edition,status,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure SQL Databases in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/databases/") { // Basic check for database ID
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				dbName := strings.Split(parts[0], "/")[10] // Extract database name from ID
				resource := models.Resource{
					ID:         dbName,
					Type:       "azurerm_sql_database",
					Name:       dbName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[3]},
					Properties: map[string]interface{}{"edition": parts[1], "status": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureWebApps(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Web Apps
	cmd := exec.CommandContext(ctx, "az", "webapp", "list",
		"--query", "[?location=='"+region+"'].[id,name,state,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Web Apps in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/sites/") { // Basic check for web app ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				appName := strings.Split(parts[0], "/")[8] // Extract app name from ID
				resource := models.Resource{
					ID:         appName,
					Type:       "azurerm_app_service",
					Name:       appName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureVirtualNetworks(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Virtual Networks
	cmd := exec.CommandContext(ctx, "az", "network", "vnet", "list",
		"--query", "[?location=='"+region+"'].[id,name,addressSpace.addressPrefixes[0],resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Virtual Networks in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/virtualNetworks/") { // Basic check for VNet ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				vnetName := strings.Split(parts[0], "/")[8] // Extract VNet name from ID
				resource := models.Resource{
					ID:         vnetName,
					Type:       "azurerm_virtual_network",
					Name:       vnetName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"address_prefix": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureLoadBalancers(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Load Balancers
	cmd := exec.CommandContext(ctx, "az", "network", "lb", "list",
		"--query", "[?location=='"+region+"'].[id,name,sku.name,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Load Balancers in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/loadBalancers/") { // Basic check for LB ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				lbName := strings.Split(parts[0], "/")[8] // Extract LB name from ID
				resource := models.Resource{
					ID:         lbName,
					Type:       "azurerm_lb",
					Name:       lbName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"sku": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureKeyVaults(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Key Vaults
	cmd := exec.CommandContext(ctx, "az", "keyvault", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.enabledForDeployment,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Key Vaults in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/vaults/") { // Basic check for Key Vault ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				vaultName := strings.Split(parts[0], "/")[8] // Extract vault name from ID
				resource := models.Resource{
					ID:         vaultName,
					Type:       "azurerm_key_vault",
					Name:       vaultName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"enabled_for_deployment": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureResourceGroups(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Resource Groups
	cmd := exec.CommandContext(ctx, "az", "group", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Resource Groups in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/resourceGroups/") { // Basic check for RG ID
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				rgName := strings.Split(parts[0], "/")[4] // Extract RG name from ID
				resource := models.Resource{
					ID:         rgName,
					Type:       "azurerm_resource_group",
					Name:       rgName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Actual discovery methods for new Azure services using CLI
func (ed *EnhancedDiscoverer) discoverAzureFunctions(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Function Apps
	cmd := exec.CommandContext(ctx, "az", "functionapp", "list",
		"--query", "[?location=='"+region+"'].[id,name,kind,state,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Functions in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/sites/") { // Basic check for function app ID
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				appName := strings.Split(parts[0], "/")[8] // Extract app name from ID
				resource := models.Resource{
					ID:         appName,
					Type:       "azurerm_function_app",
					Name:       appName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[3]},
					Properties: map[string]interface{}{"kind": parts[1], "state": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureLogicApps(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Logic Apps
	cmd := exec.CommandContext(ctx, "az", "logic", "workflow", "list",
		"--query", "[?location=='"+region+"'].[id,name,state,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Logic Apps in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/workflows/") { // Basic check for logic app ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				appName := strings.Split(parts[0], "/")[8] // Extract app name from ID
				resource := models.Resource{
					ID:         appName,
					Type:       "azurerm_logic_app_workflow",
					Name:       appName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureEventHubs(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Event Hubs Namespaces
	cmd := exec.CommandContext(ctx, "az", "eventhubs", "namespace", "list",
		"--query", "[?location=='"+region+"'].[id,name,sku.name,status,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Event Hubs in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/namespaces/") { // Basic check for namespace ID
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				namespaceName := strings.Split(parts[0], "/")[8] // Extract namespace name from ID
				resource := models.Resource{
					ID:         namespaceName,
					Type:       "azurerm_eventhub_namespace",
					Name:       namespaceName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[3]},
					Properties: map[string]interface{}{"sku": parts[1], "status": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureServiceBus(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Service Bus Namespaces
	cmd := exec.CommandContext(ctx, "az", "servicebus", "namespace", "list",
		"--query", "[?location=='"+region+"'].[id,name,sku.name,status,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Service Bus in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/namespaces/") { // Basic check for namespace ID
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				namespaceName := strings.Split(parts[0], "/")[8] // Extract namespace name from ID
				resource := models.Resource{
					ID:         namespaceName,
					Type:       "azurerm_servicebus_namespace",
					Name:       namespaceName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[3]},
					Properties: map[string]interface{}{"sku": parts[1], "status": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureCosmosDB(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Cosmos DB Accounts
	cmd := exec.CommandContext(ctx, "az", "cosmosdb", "list",
		"--query", "[?location=='"+region+"'].[id,name,kind,provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Cosmos DB in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/databaseAccounts/") { // Basic check for account ID
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				accountName := strings.Split(parts[0], "/")[8] // Extract account name from ID
				resource := models.Resource{
					ID:         accountName,
					Type:       "azurerm_cosmosdb_account",
					Name:       accountName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[3]},
					Properties: map[string]interface{}{"kind": parts[1], "provisioning_state": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureDataFactory(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Data Factories
	cmd := exec.CommandContext(ctx, "az", "datafactory", "factory", "list",
		"--query", "[?location=='"+region+"'].[id,name,provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Data Factory in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/factories/") { // Basic check for factory ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				factoryName := strings.Split(parts[0], "/")[8] // Extract factory name from ID
				resource := models.Resource{
					ID:         factoryName,
					Type:       "azurerm_data_factory",
					Name:       factoryName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureSynapseAnalytics(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Synapse Workspaces
	cmd := exec.CommandContext(ctx, "az", "synapse", "workspace", "list",
		"--query", "[?location=='"+region+"'].[id,name,provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Synapse Analytics in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/workspaces/") { // Basic check for workspace ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				workspaceName := strings.Split(parts[0], "/")[8] // Extract workspace name from ID
				resource := models.Resource{
					ID:         workspaceName,
					Type:       "azurerm_synapse_workspace",
					Name:       workspaceName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureApplicationInsights(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Application Insights
	cmd := exec.CommandContext(ctx, "az", "monitor", "app-insights", "component", "list",
		"--query", "[?location=='"+region+"'].[id,name,kind,provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Application Insights in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/components/") { // Basic check for component ID
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				componentName := strings.Split(parts[0], "/")[8] // Extract component name from ID
				resource := models.Resource{
					ID:         componentName,
					Type:       "azurerm_application_insights",
					Name:       componentName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[3]},
					Properties: map[string]interface{}{"kind": parts[1], "provisioning_state": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzurePolicy(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Policy Assignments
	cmd := exec.CommandContext(ctx, "az", "policy", "assignment", "list",
		"--query", "[?location=='"+region+"'].[id,name,displayName,enforcementMode]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Policy in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/policyAssignments/") { // Basic check for assignment ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				assignmentName := strings.Split(parts[0], "/")[8] // Extract assignment name from ID
				resource := models.Resource{
					ID:         assignmentName,
					Type:       "azurerm_policy_assignment",
					Name:       assignmentName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"display_name": parts[1], "enforcement_mode": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureBastion(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Bastion Hosts
	cmd := exec.CommandContext(ctx, "az", "network", "bastion", "list",
		"--query", "[?location=='"+region+"'].[id,name,provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Bastion in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/bastionHosts/") { // Basic check for bastion ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				bastionName := strings.Split(parts[0], "/")[8] // Extract bastion name from ID
				resource := models.Resource{
					ID:         bastionName,
					Type:       "azurerm_bastion_host",
					Name:       bastionName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Actual discovery methods for new GCP services using CLI
func (ed *EnhancedDiscoverer) discoverGCPCloudFunctions(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Functions
	cmd := exec.CommandContext(ctx, "gcloud", "functions", "list",
		"--region", region,
		"--format", "value(name,runtime,status,entryPoint)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Functions in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_cloudfunctions_function",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"runtime": parts[1], "status": parts[2], "entry_point": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudRun(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Run services
	cmd := exec.CommandContext(ctx, "gcloud", "run", "services", "list",
		"--region", region,
		"--format", "value(name,status.url,status.conditions[0].status)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Run in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_cloud_run_service",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"url": parts[1], "status": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudBuild(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Build triggers
	cmd := exec.CommandContext(ctx, "gcloud", "builds", "triggers", "list",
		"--region", region,
		"--format", "value(name,createTime,status)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Build in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_cloudbuild_trigger",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"create_time": parts[1], "status": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudPubSub(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Pub/Sub topics
	cmd := exec.CommandContext(ctx, "gcloud", "pubsub", "topics", "list",
		"--format", "value(name,messageRetentionDuration)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Pub/Sub in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_pubsub_topic",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"message_retention_duration": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPBigQuery(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover BigQuery datasets
	cmd := exec.CommandContext(ctx, "bq", "ls",
		"--format", "value(datasetReference.datasetId,creationTime,lastModifiedTime)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP BigQuery in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_bigquery_dataset",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"creation_time": parts[1], "last_modified_time": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudSpanner(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Spanner instances
	cmd := exec.CommandContext(ctx, "gcloud", "spanner", "instances", "list",
		"--format", "value(name,config,nodeCount,state)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Spanner in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_spanner_instance",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"config": parts[1], "node_count": parts[2], "state": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudFirestore(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Firestore databases
	cmd := exec.CommandContext(ctx, "gcloud", "firestore", "databases", "list",
		"--format", "value(name,type,locationId)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Firestore in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_firestore_database",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"type": parts[1], "location_id": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudArmor(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Armor security policies
	cmd := exec.CommandContext(ctx, "gcloud", "compute", "security-policies", "list",
		"--format", "value(name,type,adaptiveProtectionConfig.layer7DdosRuleConfig.enable)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Armor in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_compute_security_policy",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"type": parts[1], "ddos_protection": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudMonitoring(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Monitoring workspaces
	cmd := exec.CommandContext(ctx, "gcloud", "monitoring", "workspaces", "list",
		"--format", "value(name,displayName,createTime)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Monitoring in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_monitoring_workspace",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"display_name": parts[1], "create_time": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudLogging(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Logging sinks
	cmd := exec.CommandContext(ctx, "gcloud", "logging", "sinks", "list",
		"--format", "value(name,destination,filter)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Logging in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_logging_project_sink",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"destination": parts[1], "filter": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Actual discovery methods for existing GCP services using CLI
func (ed *EnhancedDiscoverer) discoverGCPComputeInstances(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Compute Engine instances
	cmd := exec.CommandContext(ctx, "gcloud", "compute", "instances", "list",
		"--filter", "zone:"+region+"*",
		"--format", "value(name,machineType,status,zone)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Compute Instances in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_compute_instance",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"machine_type": parts[1], "status": parts[2], "zone": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPStorageBuckets(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Storage buckets
	cmd := exec.CommandContext(ctx, "gsutil", "ls",
		"-L")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Storage Buckets in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "gs://") {
			bucketName := strings.TrimPrefix(strings.TrimSpace(line), "gs://")
			resource := models.Resource{
				ID:         bucketName,
				Type:       "google_storage_bucket",
				Name:       bucketName,
				Region:     region,
				Provider:   "gcp",
				CreatedAt:  time.Now(),
				Tags:       map[string]string{},
				Properties: map[string]interface{}{"location": region},
			}
			resources = append(resources, resource)
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPGKEClusters(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover GKE clusters
	cmd := exec.CommandContext(ctx, "gcloud", "container", "clusters", "list",
		"--region", region,
		"--format", "value(name,location,status,currentMasterVersion)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP GKE Clusters in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_container_cluster",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"location": parts[1], "status": parts[2], "master_version": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudSQL(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud SQL instances
	cmd := exec.CommandContext(ctx, "gcloud", "sql", "instances", "list",
		"--format", "value(name,region,databaseVersion,state)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud SQL in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_sql_database_instance",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"region": parts[1], "database_version": parts[2], "state": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPVPCNetworks(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover VPC networks
	cmd := exec.CommandContext(ctx, "gcloud", "compute", "networks", "list",
		"--format", "value(name,network,subnetworks[0])")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP VPC Networks in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_compute_network",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"network": parts[1], "subnetworks": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional AWS service discovery methods
func (ed *EnhancedDiscoverer) discoverAWSVPC(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover VPCs
	cmd := exec.CommandContext(ctx, "aws", "ec2", "describe-vpcs",
		"--region", region,
		"--query", "Vpcs[*].[VpcId,CidrBlock,State,Tags[?Key==`Name`].Value|[0]]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS VPCs in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "vpc-") { // Basic check for VPC ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_vpc",
					Name:       parts[3],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"cidr_block": parts[1], "state": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSSubnets(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Subnets
	cmd := exec.CommandContext(ctx, "aws", "ec2", "describe-subnets",
		"--region", region,
		"--query", "Subnets[*].[SubnetId,CidrBlock,AvailabilityZone,State,Tags[?Key==`Name`].Value|[0]]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Subnets in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "subnet-") { // Basic check for subnet ID
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_subnet",
					Name:       parts[4],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"cidr_block": parts[1], "availability_zone": parts[2], "state": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSLoadBalancers(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Application Load Balancers
	cmd := exec.CommandContext(ctx, "aws", "elbv2", "describe-load-balancers",
		"--region", region,
		"--query", "LoadBalancers[*].[LoadBalancerArn,LoadBalancerName,Type,State.Code,Scheme]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Load Balancers in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "arn:aws:elasticloadbalancing") { // Basic check for ALB ARN
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_lb",
					Name:       parts[1],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"type": parts[2], "state": parts[3], "scheme": parts[4]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSSecretsManager(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Secrets Manager secrets
	cmd := exec.CommandContext(ctx, "aws", "secretsmanager", "list-secrets",
		"--region", region,
		"--query", "SecretList[*].[ARN,Name,Description,LastChangedDate]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Secrets Manager in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "arn:aws:secretsmanager") { // Basic check for secret ARN
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_secretsmanager_secret",
					Name:       parts[1],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"description": parts[2], "last_changed_date": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSKMS(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover KMS keys
	cmd := exec.CommandContext(ctx, "aws", "kms", "list-keys",
		"--region", region,
		"--query", "Keys[*].[KeyId,KeyArn]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS KMS in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "arn:aws:kms") { // Basic check for KMS ARN
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_kms_key",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"key_arn": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSCloudTrail(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover CloudTrail trails
	cmd := exec.CommandContext(ctx, "aws", "cloudtrail", "list-trails",
		"--region", region,
		"--query", "Trails[*].[Name,TrailARN,HomeRegion]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS CloudTrail in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "arn:aws:cloudtrail") { // Basic check for trail ARN
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_cloudtrail",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"trail_arn": parts[1], "home_region": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSCloudFormationStacks(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover CloudFormation stacks
	cmd := exec.CommandContext(ctx, "aws", "cloudformation", "list-stacks",
		"--region", region,
		"--stack-status-filter", "CREATE_COMPLETE", "UPDATE_COMPLETE",
		"--query", "StackSummaries[*].[StackName,StackStatus,CreationTime]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS CloudFormation Stacks in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.Contains(line, "StackSummaries") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_cloudformation_stack",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"stack_status": parts[1], "creation_time": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSSecurityGroups(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Security Groups
	cmd := exec.CommandContext(ctx, "aws", "ec2", "describe-security-groups",
		"--region", region,
		"--query", "SecurityGroups[*].[GroupId,GroupName,Description,VpcId]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Security Groups in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "sg-") { // Basic check for security group ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_security_group",
					Name:       parts[1],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"description": parts[2], "vpc_id": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSInternetGateway(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Internet Gateways
	cmd := exec.CommandContext(ctx, "aws", "ec2", "describe-internet-gateways",
		"--region", region,
		"--query", "InternetGateways[*].[InternetGatewayId,Attachments[0].State,Attachments[0].VpcId]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Internet Gateways in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "igw-") { // Basic check for internet gateway ID
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_internet_gateway",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"state": parts[1], "vpc_id": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSNATGateway(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover NAT Gateways
	cmd := exec.CommandContext(ctx, "aws", "ec2", "describe-nat-gateways",
		"--region", region,
		"--query", "NatGateways[*].[NatGatewayId,State,SubnetId,VpcId]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS NAT Gateways in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "nat-") { // Basic check for NAT gateway ID
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "aws_nat_gateway",
					Name:       parts[0],
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"state": parts[1], "subnet_id": parts[2], "vpc_id": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional AWS services - Development & CI/CD
func (ed *EnhancedDiscoverer) discoverAWSCodeBuild(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover CodeBuild projects
	cmd := exec.CommandContext(ctx, "aws", "codebuild", "list-projects",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS CodeBuild projects in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for project name
			projectName := strings.Trim(strings.TrimSpace(line), "\"")
			if projectName != "" && projectName != "projects" {
				resource := models.Resource{
					ID:         projectName,
					Type:       "aws_codebuild_project",
					Name:       projectName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "codebuild"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSCodePipeline(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover CodePipeline pipelines
	cmd := exec.CommandContext(ctx, "aws", "codepipeline", "list-pipelines",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS CodePipeline pipelines in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for pipeline name
			pipelineName := strings.Trim(strings.TrimSpace(line), "\"")
			if pipelineName != "" && pipelineName != "pipelines" {
				resource := models.Resource{
					ID:         pipelineName,
					Type:       "aws_codepipeline",
					Name:       pipelineName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "codepipeline"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSCodeDeploy(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover CodeDeploy applications
	cmd := exec.CommandContext(ctx, "aws", "deploy", "list-applications",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS CodeDeploy applications in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for application name
			appName := strings.Trim(strings.TrimSpace(line), "\"")
			if appName != "" && appName != "applications" {
				resource := models.Resource{
					ID:         appName,
					Type:       "aws_codedeploy_app",
					Name:       appName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "codedeploy"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional AWS services - Compute & Batch Processing
func (ed *EnhancedDiscoverer) discoverAWSBatch(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Batch job queues
	cmd := exec.CommandContext(ctx, "aws", "batch", "describe-job-queues",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Batch job queues in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "jobQueueArn") { // Basic check for job queue
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				queueName := strings.Split(parts[1], "/")[1] // Extract name from ARN
				resource := models.Resource{
					ID:         queueName,
					Type:       "aws_batch_job_queue",
					Name:       queueName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "batch"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSFargate(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Fargate task definitions
	cmd := exec.CommandContext(ctx, "aws", "ecs", "list-task-definitions",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Fargate task definitions in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "taskDefinitionArn") { // Basic check for task definition
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				taskDefName := strings.Split(parts[1], "/")[1] // Extract name from ARN
				resource := models.Resource{
					ID:         taskDefName,
					Type:       "aws_ecs_task_definition",
					Name:       taskDefName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "fargate"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSEMR(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover EMR clusters
	cmd := exec.CommandContext(ctx, "aws", "emr", "list-clusters",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS EMR clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "j-") { // Basic check for cluster ID
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				clusterId := parts[0]
				clusterName := parts[1]
				resource := models.Resource{
					ID:         clusterId,
					Type:       "aws_emr_cluster",
					Name:       clusterName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "emr"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional AWS services - Database & Analytics
func (ed *EnhancedDiscoverer) discoverAWSNeptune(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Neptune clusters
	cmd := exec.CommandContext(ctx, "aws", "neptune", "describe-db-clusters",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Neptune clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "DBClusterIdentifier") { // Basic check for cluster
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				clusterName := strings.Trim(parts[1], "\"")
				resource := models.Resource{
					ID:         clusterName,
					Type:       "aws_neptune_cluster",
					Name:       clusterName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "neptune"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSDocumentDB(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover DocumentDB clusters
	cmd := exec.CommandContext(ctx, "aws", "docdb", "describe-db-clusters",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS DocumentDB clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "DBClusterIdentifier") { // Basic check for cluster
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				clusterName := strings.Trim(parts[1], "\"")
				resource := models.Resource{
					ID:         clusterName,
					Type:       "aws_docdb_cluster",
					Name:       clusterName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "documentdb"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSMSK(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover MSK clusters
	cmd := exec.CommandContext(ctx, "aws", "kafka", "list-clusters",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS MSK clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "ClusterArn") { // Basic check for cluster
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				clusterName := strings.Split(parts[1], "/")[1] // Extract name from ARN
				resource := models.Resource{
					ID:         clusterName,
					Type:       "aws_msk_cluster",
					Name:       clusterName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "msk"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSMQ(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover MQ brokers
	cmd := exec.CommandContext(ctx, "aws", "mq", "list-brokers",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS MQ brokers in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "BrokerId") { // Basic check for broker
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				brokerId := strings.Trim(parts[1], "\"")
				resource := models.Resource{
					ID:         brokerId,
					Type:       "aws_mq_broker",
					Name:       brokerId,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "mq"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional AWS services - Networking & Transfer
func (ed *EnhancedDiscoverer) discoverAWSTransfer(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Transfer servers
	cmd := exec.CommandContext(ctx, "aws", "transfer", "list-servers",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Transfer servers in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "ServerId") { // Basic check for server
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				serverId := strings.Trim(parts[1], "\"")
				resource := models.Resource{
					ID:         serverId,
					Type:       "aws_transfer_server",
					Name:       serverId,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "transfer"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSDirectConnect(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Direct Connect connections
	cmd := exec.CommandContext(ctx, "aws", "directconnect", "describe-connections",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Direct Connect connections in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "dxcon-") { // Basic check for connection ID
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				connectionId := parts[0]
				connectionName := parts[1]
				resource := models.Resource{
					ID:         connectionId,
					Type:       "aws_dx_connection",
					Name:       connectionName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "directconnect"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSVPN(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover VPN connections
	cmd := exec.CommandContext(ctx, "aws", "ec2", "describe-vpn-connections",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS VPN connections in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "vpn-") { // Basic check for VPN connection ID
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				vpnId := parts[0]
				vpnState := parts[1]
				resource := models.Resource{
					ID:         vpnId,
					Type:       "aws_vpn_connection",
					Name:       vpnId,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"state": vpnState, "service": "vpn"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSTransitGateway(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Transit Gateways
	cmd := exec.CommandContext(ctx, "aws", "ec2", "describe-transit-gateways",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Transit Gateways in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "tgw-") { // Basic check for transit gateway ID
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				tgwId := parts[0]
				tgwState := parts[1]
				resource := models.Resource{
					ID:         tgwId,
					Type:       "aws_ec2_transit_gateway",
					Name:       tgwId,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"state": tgwState, "service": "transitgateway"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional AWS services - Observability & Monitoring
func (ed *EnhancedDiscoverer) discoverAWSAppMesh(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover App Mesh meshes
	cmd := exec.CommandContext(ctx, "aws", "appmesh", "list-meshes",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS App Mesh meshes in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "meshName") { // Basic check for mesh
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				meshName := strings.Trim(parts[1], "\"")
				resource := models.Resource{
					ID:         meshName,
					Type:       "aws_appmesh_mesh",
					Name:       meshName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "appmesh"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSXRay(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover X-Ray groups
	cmd := exec.CommandContext(ctx, "aws", "xray", "get-groups",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS X-Ray groups in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "GroupName") { // Basic check for group
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				groupName := strings.Trim(parts[1], "\"")
				resource := models.Resource{
					ID:         groupName,
					Type:       "aws_xray_group",
					Name:       groupName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "xray"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional AWS services - Development Tools
func (ed *EnhancedDiscoverer) discoverAWSCloud9(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Cloud9 environments
	cmd := exec.CommandContext(ctx, "aws", "cloud9", "list-environments",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Cloud9 environments in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "environmentId") { // Basic check for environment
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				envId := strings.Trim(parts[1], "\"")
				resource := models.Resource{
					ID:         envId,
					Type:       "aws_cloud9_environment",
					Name:       envId,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "cloud9"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSCodeStar(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover CodeStar projects
	cmd := exec.CommandContext(ctx, "aws", "codestar", "list-projects",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS CodeStar projects in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "projectId") { // Basic check for project
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				projectId := strings.Trim(parts[1], "\"")
				resource := models.Resource{
					ID:         projectId,
					Type:       "aws_codestar_project",
					Name:       projectId,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "codestar"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSAmplify(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Amplify apps
	cmd := exec.CommandContext(ctx, "aws", "amplify", "list-apps",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Amplify apps in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "appId") { // Basic check for app
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				appId := strings.Trim(parts[1], "\"")
				resource := models.Resource{
					ID:         appId,
					Type:       "aws_amplify_app",
					Name:       appId,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "amplify"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional Azure services - Container & Orchestration
func (ed *EnhancedDiscoverer) discoverAzureContainerInstances(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Container Instances
	cmd := exec.CommandContext(ctx, "az", "container", "list",
		"--query", "[?location=='"+region+"'].[id,name,provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Container Instances in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/containerGroups/") { // Basic check for container group
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				containerName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         containerName,
					Type:       "azurerm_container_group",
					Name:       containerName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureContainerRegistry(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Container Registries
	cmd := exec.CommandContext(ctx, "az", "acr", "list",
		"--query", "[?location=='"+region+"'].[id,name,loginServer,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Container Registries in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/registries/") { // Basic check for registry
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				registryName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         registryName,
					Type:       "azurerm_container_registry",
					Name:       registryName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"login_server": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureAKS(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover AKS clusters
	cmd := exec.CommandContext(ctx, "az", "aks", "list",
		"--query", "[?location=='"+region+"'].[id,name,provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure AKS clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/managedClusters/") { // Basic check for AKS cluster
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				clusterName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         clusterName,
					Type:       "azurerm_kubernetes_cluster",
					Name:       clusterName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureServiceFabric(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Service Fabric clusters
	cmd := exec.CommandContext(ctx, "az", "sf", "cluster", "list",
		"--query", "[?location=='"+region+"'].[id,name,provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Service Fabric clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/clusters/") { // Basic check for Service Fabric cluster
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				clusterName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         clusterName,
					Type:       "azurerm_service_fabric_cluster",
					Name:       clusterName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureSpringCloud(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Spring Cloud services
	cmd := exec.CommandContext(ctx, "az", "spring-cloud", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Spring Cloud services in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/Spring/") { // Basic check for Spring Cloud service
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				serviceName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         serviceName,
					Type:       "azurerm_spring_cloud_service",
					Name:       serviceName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional Azure services - API & Integration
func (ed *EnhancedDiscoverer) discoverAzureAPIManagement(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover API Management services
	cmd := exec.CommandContext(ctx, "az", "apim", "list",
		"--query", "[?location=='"+region+"'].[id,name,provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure API Management services in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/service/") { // Basic check for API Management service
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				serviceName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         serviceName,
					Type:       "azurerm_api_management",
					Name:       serviceName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureEventGrid(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Event Grid topics
	cmd := exec.CommandContext(ctx, "az", "eventgrid", "topic", "list",
		"--query", "[?location=='"+region+"'].[id,name,provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Event Grid topics in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/topics/") { // Basic check for Event Grid topic
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				topicName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         topicName,
					Type:       "azurerm_eventgrid_topic",
					Name:       topicName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureStreamAnalytics(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Stream Analytics jobs
	cmd := exec.CommandContext(ctx, "az", "stream-analytics", "job", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.jobState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Stream Analytics jobs in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/streamingjobs/") { // Basic check for Stream Analytics job
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				jobName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         jobName,
					Type:       "azurerm_stream_analytics_job",
					Name:       jobName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"job_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional Azure services - Data & Analytics
func (ed *EnhancedDiscoverer) discoverAzureDataLakeStorage(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Data Lake Storage accounts
	cmd := exec.CommandContext(ctx, "az", "storage", "account", "list",
		"--query", "[?location=='"+region+"' && kind=='StorageV2'].[id,name,statusOfPrimary,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Data Lake Storage accounts in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/storageAccounts/") { // Basic check for storage account
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				accountName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         accountName,
					Type:       "azurerm_storage_data_lake_gen2_filesystem",
					Name:       accountName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"status": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureHDInsight(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover HDInsight clusters
	cmd := exec.CommandContext(ctx, "az", "hdinsight", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.clusterState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure HDInsight clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/clusters/") { // Basic check for HDInsight cluster
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				clusterName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         clusterName,
					Type:       "azurerm_hdinsight_cluster",
					Name:       clusterName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"cluster_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureDatabricks(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Databricks workspaces
	cmd := exec.CommandContext(ctx, "az", "databricks", "workspace", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Databricks workspaces in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/workspaces/") { // Basic check for Databricks workspace
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				workspaceName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         workspaceName,
					Type:       "azurerm_databricks_workspace",
					Name:       workspaceName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureMachineLearning(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Machine Learning workspaces
	cmd := exec.CommandContext(ctx, "az", "ml", "workspace", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Machine Learning workspaces in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/workspaces/") { // Basic check for ML workspace
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				workspaceName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         workspaceName,
					Type:       "azurerm_machine_learning_workspace",
					Name:       workspaceName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional Azure services - AI & Cognitive Services
func (ed *EnhancedDiscoverer) discoverAzureCognitiveServices(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Cognitive Services accounts
	cmd := exec.CommandContext(ctx, "az", "cognitiveservices", "account", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Cognitive Services accounts in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/cognitiveServices/") { // Basic check for Cognitive Services account
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				accountName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         accountName,
					Type:       "azurerm_cognitive_account",
					Name:       accountName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureBotService(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Bot Services
	cmd := exec.CommandContext(ctx, "az", "bot", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Bot Services in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/botServices/") { // Basic check for Bot Service
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				botName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         botName,
					Type:       "azurerm_bot_service",
					Name:       botName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional Azure services - Communication & Media
func (ed *EnhancedDiscoverer) discoverAzureSignalR(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover SignalR services
	cmd := exec.CommandContext(ctx, "az", "signalr", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure SignalR services in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/signalR/") { // Basic check for SignalR service
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				serviceName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         serviceName,
					Type:       "azurerm_signalr_service",
					Name:       serviceName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureMediaServices(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Media Services accounts
	cmd := exec.CommandContext(ctx, "az", "ams", "account", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Media Services accounts in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/mediaservices/") { // Basic check for Media Services account
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				accountName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         accountName,
					Type:       "azurerm_media_services_account",
					Name:       accountName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureVideoIndexer(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Video Indexer accounts
	cmd := exec.CommandContext(ctx, "az", "video-indexer", "account", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Video Indexer accounts in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/videoIndexer/") { // Basic check for Video Indexer account
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				accountName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         accountName,
					Type:       "azurerm_video_indexer_account",
					Name:       accountName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional Azure services - IoT & Edge
func (ed *EnhancedDiscoverer) discoverAzureMaps(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Maps accounts
	cmd := exec.CommandContext(ctx, "az", "maps", "account", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Maps accounts in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/accounts/") { // Basic check for Maps account
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				accountName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         accountName,
					Type:       "azurerm_maps_account",
					Name:       accountName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureTimeSeriesInsights(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Time Series Insights environments
	cmd := exec.CommandContext(ctx, "az", "tsi", "environment", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Time Series Insights environments in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/environments/") { // Basic check for TSI environment
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				envName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         envName,
					Type:       "azurerm_time_series_insights_environment",
					Name:       envName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureDigitalTwins(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Digital Twins instances
	cmd := exec.CommandContext(ctx, "az", "dt", "list",
		"--query", "[?location=='"+region+"'].[id,name,properties.provisioningState,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Digital Twins instances in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/digitalTwinsInstances/") { // Basic check for Digital Twins instance
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				instanceName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         instanceName,
					Type:       "azurerm_digital_twins_instance",
					Name:       instanceName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"provisioning_state": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional GCP services - Task & Scheduling
func (ed *EnhancedDiscoverer) discoverGCPCloudTasks(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Tasks queues
	cmd := exec.CommandContext(ctx, "gcloud", "tasks", "queues", "list",
		"--location", region,
		"--format", "value(name,state,type)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Tasks queues in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_cloud_tasks_queue",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"state": parts[1], "type": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudScheduler(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Scheduler jobs
	cmd := exec.CommandContext(ctx, "gcloud", "scheduler", "jobs", "list",
		"--location", region,
		"--format", "value(name,state,schedule)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Scheduler jobs in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_cloud_scheduler_job",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"state": parts[1], "schedule": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional GCP services - Networking
func (ed *EnhancedDiscoverer) discoverGCPCloudDNS(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud DNS managed zones
	cmd := exec.CommandContext(ctx, "gcloud", "dns", "managed-zones", "list",
		"--format", "value(name,dnsName,visibility)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud DNS managed zones: %v", err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_dns_managed_zone",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"dns_name": parts[1], "visibility": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudCDN(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud CDN backends
	cmd := exec.CommandContext(ctx, "gcloud", "compute", "backend-services", "list",
		"--filter", "enableCDN=true",
		"--format", "value(name,loadBalancingScheme,protocol)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud CDN backends: %v", err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_compute_backend_service",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"load_balancing_scheme": parts[1], "protocol": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudLoadBalancing(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Load Balancers
	cmd := exec.CommandContext(ctx, "gcloud", "compute", "url-maps", "list",
		"--format", "value(name,defaultService,loadBalancingScheme)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Load Balancers: %v", err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_compute_url_map",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"default_service": parts[1], "load_balancing_scheme": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudNAT(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud NAT gateways
	cmd := exec.CommandContext(ctx, "gcloud", "compute", "routers", "nats", "list",
		"--router", "default",
		"--region", region,
		"--format", "value(name,sourceSubnetworkIpRangesToNat,natIps)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud NAT gateways in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_compute_router_nat",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"source_subnetwork_ip_ranges": parts[1], "nat_ips": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudRouter(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Routers
	cmd := exec.CommandContext(ctx, "gcloud", "compute", "routers", "list",
		"--region", region,
		"--format", "value(name,network,asn)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Routers in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_compute_router",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"network": parts[1], "asn": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudVPN(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud VPN gateways
	cmd := exec.CommandContext(ctx, "gcloud", "compute", "vpn-gateways", "list",
		"--region", region,
		"--format", "value(name,network,vpnInterfaces)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud VPN gateways in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_compute_vpn_gateway",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"network": parts[1], "vpn_interfaces": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudInterconnect(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Interconnect attachments
	cmd := exec.CommandContext(ctx, "gcloud", "compute", "interconnects", "attachments", "list",
		"--region", region,
		"--format", "value(name,interconnect,router,operationalStatus)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Interconnect attachments in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_compute_interconnect_attachment",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"interconnect": parts[1], "router": parts[2], "operational_status": parts[3]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional GCP services - Security & IAM
func (ed *EnhancedDiscoverer) discoverGCPCloudKMS(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud KMS keyrings
	cmd := exec.CommandContext(ctx, "gcloud", "kms", "keyrings", "list",
		"--location", region,
		"--format", "value(name,createTime)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud KMS keyrings in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_kms_key_ring",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"create_time": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudIAM(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover IAM service accounts
	cmd := exec.CommandContext(ctx, "gcloud", "iam", "service-accounts", "list",
		"--format", "value(email,displayName,disabled)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP IAM service accounts: %v", err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_service_account",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"display_name": parts[1], "disabled": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudResourceManager(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover projects
	cmd := exec.CommandContext(ctx, "gcloud", "projects", "list",
		"--format", "value(projectId,name,projectNumber)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP projects: %v", err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_project",
					Name:       parts[1],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"project_number": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudBilling(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover billing accounts
	cmd := exec.CommandContext(ctx, "gcloud", "billing", "accounts", "list",
		"--format", "value(accountId,name,open)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP billing accounts: %v", err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_billing_account",
					Name:       parts[1],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"open": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional GCP services - Observability & Monitoring
func (ed *EnhancedDiscoverer) discoverGCPCloudTrace(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Trace traces
	cmd := exec.CommandContext(ctx, "gcloud", "trace", "traces", "list",
		"--limit", "10",
		"--format", "value(traceId,startTime)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Trace traces: %v", err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_trace_trace",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"start_time": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudDebugger(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Debugger debuggees
	cmd := exec.CommandContext(ctx, "gcloud", "debug", "targets", "list",
		"--format", "value(targetId,project,displayName)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Debugger targets: %v", err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_debug_target",
					Name:       parts[2],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"project": parts[1]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudProfiler(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Profiler profiles
	cmd := exec.CommandContext(ctx, "gcloud", "profiler", "profiles", "list",
		"--format", "value(profileType,deployment.target,deployment.labels)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Profiler profiles: %v", err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_profiler_profile",
					Name:       parts[1],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"profile_type": parts[0], "labels": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudErrorReporting(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Error Reporting services
	cmd := exec.CommandContext(ctx, "gcloud", "error-reporting", "services", "list",
		"--format", "value(serviceName,displayName)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Error Reporting services: %v", err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_error_reporting_service",
					Name:       parts[1],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service_name": parts[0]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional AWS services - Analytics & Data Processing
func (ed *EnhancedDiscoverer) discoverAWSAthena(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Athena workgroups
	cmd := exec.CommandContext(ctx, "aws", "athena", "list-work-groups",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Athena workgroups in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for workgroup name
			workgroupName := strings.Trim(strings.TrimSpace(line), "\"")
			if workgroupName != "" && workgroupName != "WorkGroups" {
				resource := models.Resource{
					ID:         workgroupName,
					Type:       "aws_athena_workgroup",
					Name:       workgroupName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "athena"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSKinesis(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Kinesis streams
	cmd := exec.CommandContext(ctx, "aws", "kinesis", "list-streams",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Kinesis streams in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for stream name
			streamName := strings.Trim(strings.TrimSpace(line), "\"")
			if streamName != "" && streamName != "StreamNames" {
				resource := models.Resource{
					ID:         streamName,
					Type:       "aws_kinesis_stream",
					Name:       streamName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "kinesis"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSDataPipeline(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Data Pipeline pipelines
	cmd := exec.CommandContext(ctx, "aws", "datapipeline", "list-pipelines",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Data Pipeline pipelines in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for pipeline name
			pipelineName := strings.Trim(strings.TrimSpace(line), "\"")
			if pipelineName != "" && pipelineName != "pipelineIdList" {
				resource := models.Resource{
					ID:         pipelineName,
					Type:       "aws_datapipeline_pipeline",
					Name:       pipelineName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "datapipeline"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSQuickSight(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover QuickSight dashboards
	cmd := exec.CommandContext(ctx, "aws", "quicksight", "list-dashboards",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS QuickSight dashboards in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for dashboard name
			dashboardName := strings.Trim(strings.TrimSpace(line), "\"")
			if dashboardName != "" && dashboardName != "DashboardSummaryList" {
				resource := models.Resource{
					ID:         dashboardName,
					Type:       "aws_quicksight_dashboard",
					Name:       dashboardName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "quicksight"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSDataSync(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover DataSync tasks
	cmd := exec.CommandContext(ctx, "aws", "datasync", "list-tasks",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS DataSync tasks in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for task name
			taskName := strings.Trim(strings.TrimSpace(line), "\"")
			if taskName != "" && taskName != "Tasks" {
				resource := models.Resource{
					ID:         taskName,
					Type:       "aws_datasync_task",
					Name:       taskName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "datasync"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSStorageGateway(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Storage Gateway gateways
	cmd := exec.CommandContext(ctx, "aws", "storagegateway", "list-gateways",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Storage Gateway gateways in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for gateway name
			gatewayName := strings.Trim(strings.TrimSpace(line), "\"")
			if gatewayName != "" && gatewayName != "Gateways" {
				resource := models.Resource{
					ID:         gatewayName,
					Type:       "aws_storagegateway_gateway",
					Name:       gatewayName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "storagegateway"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSBackup(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover Backup vaults
	cmd := exec.CommandContext(ctx, "aws", "backup", "list-backup-vaults",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS Backup vaults in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for vault name
			vaultName := strings.Trim(strings.TrimSpace(line), "\"")
			if vaultName != "" && vaultName != "BackupVaultList" {
				resource := models.Resource{
					ID:         vaultName,
					Type:       "aws_backup_vault",
					Name:       vaultName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "backup"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSFSx(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover FSx file systems
	cmd := exec.CommandContext(ctx, "aws", "fsx", "describe-file-systems",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS FSx file systems in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for file system name
			fsName := strings.Trim(strings.TrimSpace(line), "\"")
			if fsName != "" && fsName != "FileSystems" {
				resource := models.Resource{
					ID:         fsName,
					Type:       "aws_fsx_file_system",
					Name:       fsName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "fsx"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSWorkSpaces(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover WorkSpaces
	cmd := exec.CommandContext(ctx, "aws", "workspaces", "describe-workspaces",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS WorkSpaces in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for workspace name
			workspaceName := strings.Trim(strings.TrimSpace(line), "\"")
			if workspaceName != "" && workspaceName != "Workspaces" {
				resource := models.Resource{
					ID:         workspaceName,
					Type:       "aws_workspaces_workspace",
					Name:       workspaceName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "workspaces"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAWSAppStream(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use AWS CLI to discover AppStream fleets
	cmd := exec.CommandContext(ctx, "aws", "appstream", "describe-fleets",
		"--region", region,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering AWS AppStream fleets in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"") { // Basic check for fleet name
			fleetName := strings.Trim(strings.TrimSpace(line), "\"")
			if fleetName != "" && fleetName != "Fleets" {
				resource := models.Resource{
					ID:         fleetName,
					Type:       "aws_appstream_fleet",
					Name:       fleetName,
					Region:     region,
					Provider:   "aws",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"service": "appstream"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional Azure services - Analytics & Data Processing
func (ed *EnhancedDiscoverer) discoverAzureDataExplorer(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Data Explorer clusters
	cmd := exec.CommandContext(ctx, "az", "kusto", "cluster", "list",
		"--query", "[?location=='"+region+"'].[id,name,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Data Explorer clusters in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/clusters/") { // Basic check for cluster
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				clusterName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         clusterName,
					Type:       "azurerm_kusto_cluster",
					Name:       clusterName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"service": "dataexplorer"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureDataShare(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Data Share accounts
	cmd := exec.CommandContext(ctx, "az", "datashare", "account", "list",
		"--query", "[?location=='"+region+"'].[id,name,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Data Share accounts in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/dataShareAccounts/") { // Basic check for account
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				accountName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         accountName,
					Type:       "azurerm_data_share_account",
					Name:       accountName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"service": "datashare"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureDataBricks(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Databricks workspaces
	cmd := exec.CommandContext(ctx, "az", "databricks", "workspace", "list",
		"--query", "[?location=='"+region+"'].[id,name,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Databricks workspaces in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/workspaces/") { // Basic check for workspace
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				workspaceName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         workspaceName,
					Type:       "azurerm_databricks_workspace",
					Name:       workspaceName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"service": "databricks"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzurePurview(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Purview accounts
	cmd := exec.CommandContext(ctx, "az", "purview", "account", "list",
		"--query", "[?location=='"+region+"'].[id,name,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Purview accounts in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/accounts/") { // Basic check for account
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				accountName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         accountName,
					Type:       "azurerm_purview_account",
					Name:       accountName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"service": "purview"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureDataFactoryV2(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Data Factory V2 instances
	cmd := exec.CommandContext(ctx, "az", "datafactory", "factory", "list",
		"--query", "[?location=='"+region+"'].[id,name,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Data Factory V2 instances in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/factories/") { // Basic check for factory
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				factoryName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         factoryName,
					Type:       "azurerm_data_factory",
					Name:       factoryName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"service": "datafactory"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureDataLakeAnalytics(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Data Lake Analytics accounts
	cmd := exec.CommandContext(ctx, "az", "dla", "account", "list",
		"--query", "[?location=='"+region+"'].[id,name,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Data Lake Analytics accounts in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/accounts/") { // Basic check for account
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				accountName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         accountName,
					Type:       "azurerm_data_lake_analytics_account",
					Name:       accountName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"service": "datalakeanalytics"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureDataLakeStore(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Data Lake Store accounts
	cmd := exec.CommandContext(ctx, "az", "dls", "account", "list",
		"--query", "[?location=='"+region+"'].[id,name,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Data Lake Store accounts in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/accounts/") { // Basic check for account
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				accountName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         accountName,
					Type:       "azurerm_data_lake_store",
					Name:       accountName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"service": "datalakestore"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureDataCatalog(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Data Catalog instances
	cmd := exec.CommandContext(ctx, "az", "datacatalog", "catalog", "list",
		"--query", "[?location=='"+region+"'].[id,name,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Data Catalog instances in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/catalogs/") { // Basic check for catalog
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				catalogName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         catalogName,
					Type:       "azurerm_data_catalog",
					Name:       catalogName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"service": "datacatalog"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverAzureDataBox(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use Azure CLI to discover Data Box orders
	cmd := exec.CommandContext(ctx, "az", "databox", "job", "list",
		"--query", "[?location=='"+region+"'].[id,name,resourceGroup]",
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering Azure Data Box orders in %s: %v", region, err)
		return resources
	}

	// Parse the JSON output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "/jobs/") { // Basic check for job
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				jobName := strings.Split(parts[0], "/")[8] // Extract name from ID
				resource := models.Resource{
					ID:         jobName,
					Type:       "azurerm_databox_job",
					Name:       jobName,
					Region:     region,
					Provider:   "azure",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{"ResourceGroup": parts[2]},
					Properties: map[string]interface{}{"service": "databox"},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

// Additional GCP services - Analytics & Data Processing
func (ed *EnhancedDiscoverer) discoverGCPDataflow(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Dataflow jobs
	cmd := exec.CommandContext(ctx, "gcloud", "dataflow", "jobs", "list",
		"--region", region,
		"--format", "value(id,name,state)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Dataflow jobs in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_dataflow_job",
					Name:       parts[1],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"state": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPDataproc(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Dataproc clusters
	cmd := exec.CommandContext(ctx, "gcloud", "dataproc", "clusters", "list",
		"--region", region,
		"--format", "value(name,status.state,config.workerConfig.numInstances)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Dataproc clusters in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_dataproc_cluster",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"state": parts[1], "worker_instances": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPComposer(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Composer environments
	cmd := exec.CommandContext(ctx, "gcloud", "composer", "environments", "list",
		"--locations", region,
		"--format", "value(name,state,config.softwareConfig.imageVersion)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Composer environments in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_composer_environment",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"state": parts[1], "image_version": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPDataCatalog(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Data Catalog entries
	cmd := exec.CommandContext(ctx, "gcloud", "data-catalog", "entries", "list",
		"--location", region,
		"--format", "value(name,linkedResource,type)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Data Catalog entries in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_data_catalog_entry",
					Name:       parts[1],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"linked_resource": parts[1], "entry_type": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPDataFusion(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Data Fusion instances
	cmd := exec.CommandContext(ctx, "gcloud", "data-fusion", "instances", "list",
		"--location", region,
		"--format", "value(name,type,state)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Data Fusion instances in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_data_fusion_instance",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"instance_type": parts[1], "state": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPDataLabeling(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Data Labeling datasets
	cmd := exec.CommandContext(ctx, "gcloud", "ai", "platform", "datasets", "list",
		"--region", region,
		"--format", "value(name,displayName,metadataSchemaUri)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Data Labeling datasets in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_ai_platform_dataset",
					Name:       parts[1],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"display_name": parts[1], "schema_uri": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPAutoML(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover AutoML models
	cmd := exec.CommandContext(ctx, "gcloud", "ai", "platform", "models", "list",
		"--region", region,
		"--format", "value(name,displayName,defaultVersion.name)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP AutoML models in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_ai_platform_model",
					Name:       parts[1],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"display_name": parts[1], "default_version": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPVertexAI(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Vertex AI models
	cmd := exec.CommandContext(ctx, "gcloud", "ai", "models", "list",
		"--region", region,
		"--format", "value(name,displayName,versionId)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Vertex AI models in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_vertex_ai_model",
					Name:       parts[1],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"display_name": parts[1], "version_id": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func (ed *EnhancedDiscoverer) discoverGCPCloudDeploy(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Use gcloud CLI to discover Cloud Deploy delivery pipelines
	cmd := exec.CommandContext(ctx, "gcloud", "deploy", "delivery-pipelines", "list",
		"--region", region,
		"--format", "value(name,description,serialPipeline.stages)")

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error discovering GCP Cloud Deploy pipelines in %s: %v", region, err)
		return resources
	}

	// Parse the output and convert to resources
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				resource := models.Resource{
					ID:         parts[0],
					Type:       "google_clouddeploy_delivery_pipeline",
					Name:       parts[0],
					Region:     region,
					Provider:   "gcp",
					CreatedAt:  time.Now(),
					Tags:       map[string]string{},
					Properties: map[string]interface{}{"description": parts[1], "stages": parts[2]},
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}
