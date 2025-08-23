package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// GetAllAzureExpandedResourceTypes returns comprehensive list of Azure resource types matching AWS parity
func GetAllAzureExpandedResourceTypes() []string {
	return []string{
		// Compute - Expanded
		"virtual_machine", "vm_scale_set", "vm_availability_set", "vm_image", "vm_snapshot",
		"container_instance", "container_group", "container_registry", "container_registry_webhook",
		"aks_cluster", "aks_node_pool", "aks_managed_identity",
		"service_fabric_cluster", "service_fabric_application", "service_fabric_service",
		"batch_account", "batch_pool", "batch_job", "batch_task",
		"cloud_service", "cloud_service_role", "cloud_service_deployment",
		"dedicated_host", "dedicated_host_group", "proximity_placement_group",
		"virtual_desktop", "virtual_desktop_host_pool", "virtual_desktop_workspace",
		"arc_enabled_server", "arc_enabled_kubernetes", "arc_data_controller",

		// Storage - Expanded
		"storage_account", "storage_container", "storage_blob", "storage_queue", "storage_table",
		"storage_file_share", "storage_sync_service", "storage_sync_group",
		"managed_disk", "disk_snapshot", "disk_encryption_set", "disk_access",
		"netapp_account", "netapp_pool", "netapp_volume", "netapp_snapshot",
		"data_lake_store", "data_lake_analytics", "data_lake_gen2",
		"backup_vault", "backup_policy", "backup_protected_item", "recovery_services_vault",
		"site_recovery_fabric", "site_recovery_protection", "site_recovery_plan",
		"data_box", "data_box_edge", "data_box_gateway",
		"storage_mover", "storage_cache", "storage_target",

		// Database - Expanded
		"sql_server", "sql_database", "sql_elastic_pool", "sql_managed_instance",
		"sql_virtual_machine", "sql_availability_group", "sql_failover_group",
		"cosmos_db_account", "cosmos_db_database", "cosmos_db_container", "cosmos_db_cassandra",
		"cosmos_db_gremlin", "cosmos_db_mongo", "cosmos_db_table",
		"postgresql_server", "postgresql_flexible_server", "postgresql_database",
		"mysql_server", "mysql_flexible_server", "mysql_database",
		"mariadb_server", "mariadb_database",
		"redis_cache", "redis_enterprise", "redis_enterprise_database",
		"data_factory", "data_factory_pipeline", "data_factory_dataset", "data_factory_trigger",
		"synapse_workspace", "synapse_sql_pool", "synapse_spark_pool", "synapse_pipeline",
		"purview_account", "purview_scan", "purview_classification",
		"database_migration_service", "database_migration_project",

		// Networking - Expanded
		"virtual_network", "subnet", "network_security_group", "network_security_rule",
		"application_security_group", "network_interface", "public_ip", "public_ip_prefix",
		"load_balancer", "application_gateway", "application_gateway_waf",
		"traffic_manager", "traffic_manager_profile", "traffic_manager_endpoint",
		"front_door", "front_door_waf_policy", "front_door_rules_engine",
		"cdn_profile", "cdn_endpoint", "cdn_origin", "cdn_custom_domain",
		"firewall", "firewall_policy", "firewall_application_rule", "firewall_network_rule",
		"nat_gateway", "bastion_host", "virtual_wan", "virtual_hub", "vpn_gateway",
		"vpn_connection", "vpn_site", "express_route_circuit", "express_route_gateway",
		"express_route_connection", "express_route_port", "express_route_peering",
		"private_endpoint", "private_link_service", "private_dns_zone",
		"dns_zone", "dns_record", "dns_forwarding_ruleset",
		"network_watcher", "network_watcher_flow_log", "connection_monitor",
		"ddos_protection_plan", "web_application_firewall",

		// Security & Identity - Expanded
		"managed_identity", "user_assigned_identity", "system_assigned_identity",
		"key_vault", "key_vault_secret", "key_vault_key", "key_vault_certificate",
		"managed_hsm", "dedicated_hsm", "payment_hsm",
		"security_center", "security_assessment", "security_policy", "security_contact",
		"sentinel_workspace", "sentinel_alert_rule", "sentinel_incident", "sentinel_playbook",
		"defender_for_cloud", "defender_assessment", "defender_recommendation",
		"azure_policy", "policy_definition", "policy_assignment", "policy_initiative",
		"blueprint", "blueprint_assignment", "blueprint_artifact",
		"management_group", "subscription", "resource_group", "resource_lock",
		"role_definition", "role_assignment", "privileged_identity_management",
		"attestation_provider", "confidential_ledger",

		// Analytics - Expanded
		"log_analytics_workspace", "log_analytics_query", "log_analytics_solution",
		"application_insights", "application_insights_component", "application_insights_test",
		"event_hub_namespace", "event_hub", "event_hub_consumer_group", "event_hub_authorization",
		"stream_analytics_job", "stream_analytics_input", "stream_analytics_output",
		"stream_analytics_function", "stream_analytics_transformation",
		"databricks_workspace", "databricks_cluster", "databricks_job", "databricks_notebook",
		"hdinsight_cluster", "hdinsight_application", "hdinsight_script",
		"power_bi_workspace", "power_bi_dataset", "power_bi_report", "power_bi_dashboard",
		"analysis_services", "analysis_services_server", "analysis_services_model",
		"data_explorer_cluster", "data_explorer_database", "data_explorer_table",
		"time_series_insights", "time_series_environment", "time_series_event_source",
		"azure_quantum_workspace", "quantum_job", "quantum_provider",

		// Application Integration - Expanded
		"service_bus_namespace", "service_bus_queue", "service_bus_topic", "service_bus_subscription",
		"logic_app", "logic_app_workflow", "logic_app_integration_account", "logic_app_connector",
		"api_management", "api_management_api", "api_management_product", "api_management_subscription",
		"event_grid_domain", "event_grid_topic", "event_grid_subscription", "event_grid_partner",
		"notification_hub_namespace", "notification_hub", "notification_hub_authorization",
		"relay_namespace", "relay_hybrid_connection", "relay_wcf_relay",
		"integration_service_environment", "integration_runtime",
		"azure_stack_edge", "azure_stack_hub", "azure_stack_hci",

		// App Services - Expanded
		"app_service_plan", "web_app", "function_app", "logic_app_standard",
		"static_web_app", "app_service_environment", "app_service_domain",
		"app_service_certificate", "app_service_managed_certificate",
		"mobile_app", "api_app", "web_app_slot", "function_app_slot",
		"container_app", "container_app_environment", "container_app_job",
		"spring_cloud_service", "spring_cloud_app", "spring_cloud_deployment",

		// Developer Tools - Expanded
		"devops_organization", "devops_project", "devops_repository", "devops_pipeline",
		"devops_build", "devops_release", "devops_artifact", "devops_test_plan",
		"dev_test_lab", "dev_test_vm", "dev_test_artifact", "dev_test_environment",
		"app_configuration", "app_configuration_key", "app_configuration_feature_flag",
		"visual_studio_account", "visual_studio_project", "visual_studio_extension",
		"load_test", "load_test_resource", "performance_test",
		"chaos_studio", "chaos_experiment", "chaos_target",

		// Management & Monitoring - Expanded
		"monitor_action_group", "monitor_alert_rule", "monitor_metric_alert", "monitor_log_alert",
		"monitor_autoscale", "monitor_diagnostic_setting", "monitor_private_link",
		"automation_account", "automation_runbook", "automation_job", "automation_schedule",
		"automation_variable", "automation_credential", "automation_certificate",
		"scheduler_job", "scheduler_job_collection",
		"resource_graph_query", "resource_graph_chart", "resource_graph_dashboard",
		"cost_management_budget", "cost_management_alert", "cost_management_export",
		"advisor_recommendation", "advisor_suppression", "advisor_configuration",
		"service_health_alert", "service_health_incident", "service_health_event",
		"activity_log_alert", "activity_log_diagnostic",
		"update_management", "update_deployment", "update_configuration",

		// AI & Machine Learning - Expanded
		"machine_learning_workspace", "machine_learning_compute", "machine_learning_datastore",
		"machine_learning_dataset", "machine_learning_experiment", "machine_learning_pipeline",
		"machine_learning_model", "machine_learning_endpoint", "machine_learning_deployment",
		"cognitive_services_account", "cognitive_services_custom_vision", "cognitive_services_face",
		"cognitive_services_text_analytics", "cognitive_services_translator", "cognitive_services_speech",
		"cognitive_services_luis", "cognitive_services_qna_maker", "cognitive_services_personalizer",
		"cognitive_services_anomaly_detector", "cognitive_services_form_recognizer",
		"cognitive_services_metrics_advisor", "cognitive_services_immersive_reader",
		"bot_service", "bot_channel", "bot_connection",
		"azure_openai", "openai_deployment", "openai_model",
		"video_analyzer", "video_analyzer_edge", "video_analyzer_pipeline",

		// IoT - Expanded
		"iot_hub", "iot_hub_device", "iot_hub_module", "iot_hub_route", "iot_hub_endpoint",
		"iot_central_app", "iot_central_device", "iot_central_device_template",
		"iot_edge_device", "iot_edge_module", "iot_edge_deployment",
		"digital_twins_instance", "digital_twins_model", "digital_twins_twin",
		"digital_twins_endpoint", "digital_twins_route",
		"device_provisioning_service", "device_enrollment", "device_registration",
		"time_series_insights_gen2", "time_series_model", "time_series_hierarchy",
		"azure_sphere_catalog", "azure_sphere_product", "azure_sphere_device",
		"azure_rtos", "azure_percept", "azure_kinect",

		// Media Services - Expanded
		"media_services_account", "media_services_asset", "media_services_job",
		"media_services_transform", "media_services_streaming_locator", "media_services_streaming_endpoint",
		"media_services_live_event", "media_services_live_output", "media_services_content_key",
		"video_indexer_account", "video_indexer_project", "video_indexer_model",
		"azure_communication_service", "communication_identity", "communication_phone",
		"communication_chat", "communication_sms", "communication_email",
		"content_delivery_network", "cdn_security_policy", "cdn_web_application_firewall",

		// Gaming - Expanded
		"playfab_title", "playfab_player", "playfab_catalog", "playfab_inventory",
		"game_server", "game_session", "game_matchmaking",
		"xbox_live_service", "xbox_achievement", "xbox_leaderboard",

		// Blockchain - Expanded
		"blockchain_member", "blockchain_consortium", "blockchain_node",
		"blockchain_transaction_node", "blockchain_watcher",
		"azure_confidential_computing", "confidential_vm", "confidential_container",

		// Mixed Reality - Expanded
		"spatial_anchors_account", "spatial_anchor", "spatial_map",
		"remote_rendering_account", "remote_rendering_session", "remote_rendering_asset",
		"object_anchors_account", "object_model", "object_instance",
		"hololens_device", "hololens_app", "hololens_spatial_mapping",

		// Quantum - Expanded
		"quantum_workspace", "quantum_job", "quantum_provider", "quantum_target",
		"quantum_circuit", "quantum_algorithm", "quantum_simulator",

		// Industry Specific - Expanded
		"azure_health_data_services", "fhir_service", "dicom_service", "medtech_service",
		"azure_farmbeats", "farmbeats_hub", "farmbeats_device", "farmbeats_sensor",
		"azure_industrial_iot", "industrial_edge", "industrial_module",
		"azure_automotive", "connected_vehicle", "vehicle_telemetry",
		"azure_retail", "retail_recommendation", "retail_inventory",

		// Migration Services - Expanded
		"migrate_project", "migrate_assessment", "migrate_appliance",
		"database_migration_service", "database_migration_task",
		"data_box_family", "import_export_job",
		"azure_file_sync", "storage_sync_server", "sync_group",
	}
}

// DiscoverAzureExpandedResources discovers all expanded Azure resource types
func DiscoverAzureExpandedResources(ctx context.Context, subscriptionID string) ([]models.Resource, error) {
	var allResources []models.Resource
	resourceTypes := GetAllAzureExpandedResourceTypes()

	for _, resourceType := range resourceTypes {
		resources, err := discoverAzureResourceType(ctx, subscriptionID, resourceType)
		if err != nil {
			// Log error but continue with other resource types
			fmt.Printf("Warning: Failed to discover Azure %s resources: %v\n", resourceType, err)
			continue
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

// discoverAzureResourceType discovers a specific Azure resource type
func discoverAzureResourceType(ctx context.Context, subscriptionID, resourceType string) ([]models.Resource, error) {
	// Map internal resource type to Azure API provider/type format
	azureType := mapToAzureResourceType(resourceType)

	cmd := exec.CommandContext(ctx, "az", "resource", "list",
		"--subscription", subscriptionID,
		"--resource-type", azureType,
		"--output", "json")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list %s resources: %w", resourceType, err)
	}

	var azureResources []map[string]interface{}
	if err := json.Unmarshal(output, &azureResources); err != nil {
		return nil, fmt.Errorf("failed to parse Azure response: %w", err)
	}

	var resources []models.Resource
	for _, azResource := range azureResources {
		resource := models.Resource{
			ID:         getAzureStringValue(azResource, "id"),
			Name:       getAzureStringValue(azResource, "name"),
			Type:       resourceType,
			Provider:   "azure",
			Region:     getAzureStringValue(azResource, "location"),
			Properties: azResource,
			Tags:       extractAzureTags(azResource),
			CreatedAt:  parseAzureTimestamp(getAzureStringValue(azResource, "createdTime")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// mapToAzureResourceType maps internal type names to Azure API format
func mapToAzureResourceType(resourceType string) string {
	// Map of internal names to Azure provider/type format
	typeMap := map[string]string{
		"virtual_machine":            "Microsoft.Compute/virtualMachines",
		"storage_account":            "Microsoft.Storage/storageAccounts",
		"sql_database":               "Microsoft.Sql/servers/databases",
		"cosmos_db_account":          "Microsoft.DocumentDB/databaseAccounts",
		"aks_cluster":                "Microsoft.ContainerService/managedClusters",
		"key_vault":                  "Microsoft.KeyVault/vaults",
		"virtual_network":            "Microsoft.Network/virtualNetworks",
		"app_service_plan":           "Microsoft.Web/serverfarms",
		"web_app":                    "Microsoft.Web/sites",
		"function_app":               "Microsoft.Web/sites",
		"cognitive_services_account": "Microsoft.CognitiveServices/accounts",
		"iot_hub":                    "Microsoft.Devices/IotHubs",
		"event_hub":                  "Microsoft.EventHub/namespaces/eventhubs",
		"service_bus_namespace":      "Microsoft.ServiceBus/namespaces",
		"machine_learning_workspace": "Microsoft.MachineLearningServices/workspaces",
		// Add more mappings as needed
	}

	if azureType, ok := typeMap[resourceType]; ok {
		return azureType
	}

	// Default: try to construct from resource type
	// Convert snake_case to PascalCase and guess provider
	parts := strings.Split(resourceType, "_")
	if len(parts) > 0 {
		provider := guessAzureProvider(parts[0])
		resourceName := toPascalCase(strings.Join(parts, ""))
		return fmt.Sprintf("%s/%s", provider, resourceName)
	}

	return resourceType
}

// guessAzureProvider attempts to determine the Azure provider from resource type
func guessAzureProvider(prefix string) string {
	providerMap := map[string]string{
		"vm":        "Microsoft.Compute",
		"virtual":   "Microsoft.Compute",
		"storage":   "Microsoft.Storage",
		"sql":       "Microsoft.Sql",
		"cosmos":    "Microsoft.DocumentDB",
		"aks":       "Microsoft.ContainerService",
		"key":       "Microsoft.KeyVault",
		"network":   "Microsoft.Network",
		"app":       "Microsoft.Web",
		"web":       "Microsoft.Web",
		"function":  "Microsoft.Web",
		"cognitive": "Microsoft.CognitiveServices",
		"iot":       "Microsoft.Devices",
		"event":     "Microsoft.EventHub",
		"service":   "Microsoft.ServiceBus",
		"machine":   "Microsoft.MachineLearningServices",
		"media":     "Microsoft.Media",
		"monitor":   "Microsoft.Insights",
		"security":  "Microsoft.Security",
		"identity":  "Microsoft.ManagedIdentity",
	}

	for key, provider := range providerMap {
		if strings.HasPrefix(strings.ToLower(prefix), key) {
			return provider
		}
	}

	return "Microsoft.Resources"
}

// toPascalCase converts snake_case to PascalCase
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

// extractAzureTags extracts tags from Azure resource
func extractAzureTags(resource map[string]interface{}) map[string]string {
	tags := make(map[string]string)
	if tagData, ok := resource["tags"].(map[string]interface{}); ok {
		for key, value := range tagData {
			if strValue, ok := value.(string); ok {
				tags[key] = strValue
			}
		}
	}
	return tags
}

// parseAzureTimestamp parses Azure timestamp string
func parseAzureTimestamp(timestamp string) time.Time {
	if timestamp == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, timestamp)
	return t
}

// getAzureStringValue safely extracts string value from map
func getAzureStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
