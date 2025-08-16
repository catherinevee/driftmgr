package discovery

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"sort"

	"github.com/catherinevee/driftmgr/internal/models"
	"gopkg.in/yaml.v3"
)

// PluginConfig represents the configuration for discovery plugins
type PluginConfig struct {
	DiscoveryPlugins  map[string][]DiscoveryPlugin `yaml:"discovery_plugins"`
	DiscoverySettings DiscoverySettings            `yaml:"discovery_settings"`
}

// DiscoverySettings contains global discovery configuration
type DiscoverySettings struct {
	CacheTTL                 string               `yaml:"cache_ttl"`
	CacheMaxSize             int                  `yaml:"cache_max_size"`
	MaxConcurrentDiscoveries int                  `yaml:"max_concurrent_discoveries"`
	MaxConcurrentRegions     int                  `yaml:"max_concurrent_regions"`
	DiscoveryTimeout         string               `yaml:"discovery_timeout"`
	APITimeout               string               `yaml:"api_timeout"`
	MaxRetries               int                  `yaml:"max_retries"`
	RetryDelay               string               `yaml:"retry_delay"`
	DefaultFilters           DiscoveryFilters     `yaml:"default_filters"`
	QualityThresholds        QualityThresholds    `yaml:"quality_thresholds"`
	Notifications            NotificationSettings `yaml:"notifications"`
}

// DiscoveryFilters represents filtering configuration
type DiscoveryFilters struct {
	IncludeTags   map[string]string `yaml:"include_tags"`
	ExcludeTags   map[string]string `yaml:"exclude_tags"`
	AgeThreshold  string            `yaml:"age_threshold"`
	CostThreshold float64           `yaml:"cost_threshold"`
	SecurityScore int               `yaml:"security_score"`
}

// QualityThresholds represents quality metrics
type QualityThresholds struct {
	Completeness float64 `yaml:"completeness"`
	Accuracy     float64 `yaml:"accuracy"`
	Freshness    string  `yaml:"freshness"`
}

// NotificationSettings represents notification configuration
type NotificationSettings struct {
	OnDiscoveryComplete  bool `yaml:"on_discovery_complete"`
	OnDiscoveryError     bool `yaml:"on_discovery_error"`
	OnQualityDegradation bool `yaml:"on_quality_degradation"`
}

// PluginLoader handles loading and managing discovery plugins
type PluginLoader struct {
	config     *PluginConfig
	discoverer *EnhancedDiscoverer
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(discoverer *EnhancedDiscoverer) *PluginLoader {
	return &PluginLoader{
		discoverer: discoverer,
	}
}

// LoadPluginsFromFile loads plugins from a configuration file
func (pl *PluginLoader) LoadPluginsFromFile(configPath string) error {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read plugin config file: %w", err)
	}

	var config PluginConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse plugin config: %w", err)
	}

	pl.config = &config
	return pl.registerPlugins()
}

// LoadPluginsFromData loads plugins from configuration data
func (pl *PluginLoader) LoadPluginsFromData(data []byte) error {
	var config PluginConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse plugin config: %w", err)
	}

	pl.config = &config
	return pl.registerPlugins()
}

// registerPlugins registers all enabled plugins with the discoverer
func (pl *PluginLoader) registerPlugins() error {
	if pl.config == nil {
		return fmt.Errorf("no plugin configuration loaded")
	}

	// Register plugins by provider
	for provider, plugins := range pl.config.DiscoveryPlugins {
		log.Printf("Registering %d plugins for provider: %s", len(plugins), provider)

		// Sort plugins by priority (lower number = higher priority)
		sort.Slice(plugins, func(i, j int) bool {
			return plugins[i].Priority < plugins[j].Priority
		})

		for _, plugin := range plugins {
			if plugin.Enabled {
				// Assign discovery function based on plugin name
				plugin.DiscoveryFn = pl.getDiscoveryFunction(provider, plugin.Name)
				pl.discoverer.RegisterPlugin(&plugin)
				log.Printf("Registered plugin: %s (priority: %d)", plugin.Name, plugin.Priority)
			}
		}
	}

	return nil
}

// getDiscoveryFunction returns the appropriate discovery function for a plugin
func (pl *PluginLoader) getDiscoveryFunction(provider, pluginName string) func(context.Context, string, string) ([]models.Resource, error) {
	switch provider {
	case "aws":
		return pl.getAWSDiscoveryFunction(pluginName)
	case "azure":
		return pl.getAzureDiscoveryFunction(pluginName)
	case "gcp":
		return pl.getGCPDiscoveryFunction(pluginName)
	default:
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return []models.Resource{}, nil
		}
	}
}

// getAWSDiscoveryFunction returns AWS-specific discovery functions
func (pl *PluginLoader) getAWSDiscoveryFunction(pluginName string) func(context.Context, string, string) ([]models.Resource, error) {
	switch pluginName {
	case "ec2_instances":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSEC2(ctx, region), nil
		}
	case "rds_databases":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSRDS(ctx, region), nil
		}
	case "lambda_functions":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSLambda(ctx, region), nil
		}
	case "cloudformation_stacks":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSCloudFormation(ctx, region), nil
		}
	case "waf_web_acls":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSWAF(ctx, region), nil
		}
	case "shield_protection":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSShield(ctx, region), nil
		}
	case "config_rules":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSConfig(ctx, region), nil
		}
	case "guardduty_detectors":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSGuardDuty(ctx, region), nil
		}
	case "cloudfront_distributions":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSCloudFront(ctx, region), nil
		}
	case "api_gateway_apis":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSAPIGateway(ctx, region), nil
		}
	case "glue_databases":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSGlue(ctx, region), nil
		}
	case "redshift_clusters":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSRedshift(ctx, region), nil
		}
	case "elasticsearch_domains":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSElasticsearch(ctx, region), nil
		}
	case "cloudwatch_alarms":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSCloudWatch(ctx, region), nil
		}
	case "systems_manager_parameters":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSSystemsManager(ctx, region), nil
		}
	case "step_functions_state_machines":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSStepFunctions(ctx, region), nil
		}
	case "s3_buckets":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSS3(ctx), nil
		}
	case "iam_roles":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSIAM(ctx), nil
		}
	case "route53_zones":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAWSRoute53(ctx), nil
		}
	default:
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return []models.Resource{}, nil
		}
	}
}

// getAzureDiscoveryFunction returns Azure-specific discovery functions
func (pl *PluginLoader) getAzureDiscoveryFunction(pluginName string) func(context.Context, string, string) ([]models.Resource, error) {
	switch pluginName {
	case "virtual_machines":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureVMs(ctx, region), nil
		}
	case "storage_accounts":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureStorageAccounts(ctx, region), nil
		}
	case "sql_databases":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureSQLDatabases(ctx, region), nil
		}
	case "web_apps":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureWebApps(ctx, region), nil
		}
	case "virtual_networks":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureVirtualNetworks(ctx, region), nil
		}
	case "load_balancers":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureLoadBalancers(ctx, region), nil
		}
	case "key_vaults":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureKeyVaults(ctx, region), nil
		}
	case "resource_groups":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureResourceGroups(ctx, region), nil
		}
	case "function_apps":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureFunctions(ctx, region), nil
		}
	case "logic_apps":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureLogicApps(ctx, region), nil
		}
	case "event_hubs":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureEventHubs(ctx, region), nil
		}
	case "service_bus":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureServiceBus(ctx, region), nil
		}
	case "cosmos_db_accounts":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureCosmosDB(ctx, region), nil
		}
	case "data_factories":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureDataFactory(ctx, region), nil
		}
	case "synapse_workspaces":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureSynapseAnalytics(ctx, region), nil
		}
	case "application_insights":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureApplicationInsights(ctx, region), nil
		}
	case "policy_assignments":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzurePolicy(ctx, region), nil
		}
	case "bastion_hosts":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverAzureBastion(ctx, region), nil
		}
	default:
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return []models.Resource{}, nil
		}
	}
}

// getGCPDiscoveryFunction returns GCP-specific discovery functions
func (pl *PluginLoader) getGCPDiscoveryFunction(pluginName string) func(context.Context, string, string) ([]models.Resource, error) {
	switch pluginName {
	case "compute_instances":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPComputeInstances(ctx, region), nil
		}
	case "storage_buckets":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPStorageBuckets(ctx, region), nil
		}
	case "gke_clusters":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPGKEClusters(ctx, region), nil
		}
	case "cloud_sql_instances":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPCloudSQL(ctx, region), nil
		}
	case "vpc_networks":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPVPCNetworks(ctx, region), nil
		}
	case "cloud_functions":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPCloudFunctions(ctx, region), nil
		}
	case "cloud_run_services":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPCloudRun(ctx, region), nil
		}
	case "cloud_build_triggers":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPCloudBuild(ctx, region), nil
		}
	case "pubsub_topics":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPCloudPubSub(ctx, region), nil
		}
	case "bigquery_datasets":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPBigQuery(ctx, region), nil
		}
	case "spanner_instances":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPCloudSpanner(ctx, region), nil
		}
	case "firestore_databases":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPCloudFirestore(ctx, region), nil
		}
	case "cloud_armor_policies":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPCloudArmor(ctx, region), nil
		}
	case "monitoring_workspaces":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPCloudMonitoring(ctx, region), nil
		}
	case "logging_sinks":
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return pl.discoverer.discoverGCPCloudLogging(ctx, region), nil
		}
	default:
		return func(ctx context.Context, region, provider string) ([]models.Resource, error) {
			return []models.Resource{}, nil
		}
	}
}

// GetPluginConfig returns the loaded plugin configuration
func (pl *PluginLoader) GetPluginConfig() *PluginConfig {
	return pl.config
}

// GetEnabledPlugins returns all enabled plugins for a provider
func (pl *PluginLoader) GetEnabledPlugins(provider string) []DiscoveryPlugin {
	if pl.config == nil {
		return []DiscoveryPlugin{}
	}

	var enabled []DiscoveryPlugin
	if plugins, exists := pl.config.DiscoveryPlugins[provider]; exists {
		for _, plugin := range plugins {
			if plugin.Enabled {
				enabled = append(enabled, plugin)
			}
		}
	}

	return enabled
}

// GetPluginByName returns a specific plugin by name and provider
func (pl *PluginLoader) GetPluginByName(provider, name string) *DiscoveryPlugin {
	if pl.config == nil {
		return nil
	}

	if plugins, exists := pl.config.DiscoveryPlugins[provider]; exists {
		for _, plugin := range plugins {
			if plugin.Name == name {
				return &plugin
			}
		}
	}

	return nil
}
