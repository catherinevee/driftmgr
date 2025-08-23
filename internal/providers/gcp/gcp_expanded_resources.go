package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// GetAllGCPExpandedResourceTypes returns comprehensive list of GCP resource types matching AWS parity
func GetAllGCPExpandedResourceTypes() []string {
	return []string{
		// Compute - Expanded
		"compute_instance", "compute_instance_template", "compute_instance_group",
		"compute_instance_group_manager", "compute_autoscaler", "compute_region_autoscaler",
		"compute_disk", "compute_snapshot", "compute_image", "compute_machine_image",
		"gke_cluster", "gke_node_pool", "gke_workload_identity", "gke_autopilot",
		"gke_hub", "gke_fleet", "gke_backup", "gke_binary_authorization",
		"cloud_run_service", "cloud_run_job", "cloud_run_revision", "cloud_run_domain_mapping",
		"cloud_functions", "cloud_functions_gen2", "cloud_functions_trigger",
		"app_engine_application", "app_engine_service", "app_engine_version",
		"batch_job", "batch_task_group", "batch_compute_environment",
		"compute_sole_tenant_node", "compute_reservation", "compute_commitment",
		"anthos_cluster", "anthos_config", "anthos_service_mesh", "anthos_policy",
		"vm_migration", "vm_migration_source", "vm_migration_group",

		// Storage - Expanded
		"storage_bucket", "storage_object", "storage_bucket_iam", "storage_notification",
		"storage_hmac_key", "storage_transfer_job", "storage_transfer_agent",
		"filestore_instance", "filestore_backup", "filestore_snapshot",
		"persistent_disk", "regional_disk", "disk_resource_policy",
		"cloud_storage_for_firebase", "firebase_storage_bucket",
		"backup_dr_management", "backup_dr_backup_vault", "backup_dr_backup_plan",
		"cloud_storage_insights", "storage_lens", "storage_lifecycle_policy",
		"nearline_storage", "coldline_storage", "archive_storage",
		"cloud_cdn", "cdn_cache_invalidation", "cdn_signed_url",
		"parallelstore", "storage_pool", "storage_volume",

		// Database - Expanded
		"cloud_sql_instance", "cloud_sql_database", "cloud_sql_user", "cloud_sql_backup",
		"cloud_sql_replica", "cloud_sql_failover", "cloud_sql_ssl_cert",
		"cloud_spanner_instance", "cloud_spanner_database", "cloud_spanner_backup",
		"cloud_spanner_instance_config", "cloud_spanner_database_role",
		"bigtable_instance", "bigtable_cluster", "bigtable_table", "bigtable_backup",
		"bigtable_app_profile", "bigtable_gc_policy", "bigtable_iam",
		"firestore_database", "firestore_document", "firestore_index", "firestore_backup",
		"firestore_field", "firestore_collection_group",
		"datastore_entity", "datastore_index", "datastore_backup", "datastore_export",
		"memorystore_redis", "memorystore_redis_cluster", "memorystore_memcached",
		"alloydb_cluster", "alloydb_instance", "alloydb_backup", "alloydb_user",
		"database_migration_service", "database_migration_job", "database_migration_connection",
		"cloud_sql_insights", "query_insights", "database_observability",

		// Networking - Expanded
		"vpc_network", "vpc_subnet", "vpc_firewall_rule", "vpc_firewall_policy",
		"vpc_route", "vpc_router", "vpc_nat", "vpc_peering", "vpc_connector",
		"load_balancer", "backend_service", "backend_bucket", "url_map",
		"target_http_proxy", "target_https_proxy", "target_tcp_proxy", "target_ssl_proxy",
		"health_check", "http_health_check", "https_health_check", "tcp_health_check",
		"cloud_armor_policy", "cloud_armor_rule", "cloud_armor_security_policy",
		"cloud_cdn_policy", "cloud_cdn_backend", "cloud_cdn_cache_key",
		"cloud_vpn_gateway", "cloud_vpn_tunnel", "vpn_peer_gateway",
		"cloud_interconnect", "interconnect_attachment", "partner_interconnect",
		"cloud_router", "cloud_router_interface", "cloud_router_peer",
		"private_service_access", "private_service_connection", "service_networking",
		"network_endpoint_group", "global_network_endpoint_group", "regional_network_endpoint_group",
		"dns_managed_zone", "dns_record_set", "dns_policy", "dns_response_policy",
		"traffic_director", "service_mesh", "envoy_proxy_config",
		"network_connectivity_center", "network_hub", "network_spoke",
		"packet_mirroring", "flow_logs", "network_intelligence_center",

		// Security & Identity - Expanded
		"iam_role", "iam_binding", "iam_member", "iam_policy", "iam_audit_config",
		"service_account", "service_account_key", "service_account_iam",
		"identity_platform_tenant", "identity_platform_config", "identity_platform_provider",
		"cloud_kms_keyring", "cloud_kms_crypto_key", "cloud_kms_crypto_key_version",
		"cloud_kms_import_job", "cloud_kms_ekm_connection",
		"cloud_hsm", "cloud_hsm_cluster", "cloud_hsm_partition",
		"secret_manager_secret", "secret_manager_version", "secret_manager_iam",
		"security_command_center", "security_finding", "security_source",
		"security_health_analytics", "security_threat_detection", "security_posture",
		"certificate_authority", "certificate", "certificate_template", "certificate_revocation",
		"binary_authorization_policy", "binary_authorization_attestor", "container_analysis",
		"web_security_scanner", "security_scanner_config", "scan_finding",
		"cloud_dlp_job", "cloud_dlp_template", "cloud_dlp_stored_info", "cloud_dlp_trigger",
		"vpc_service_controls", "access_context_manager", "access_level", "access_policy",
		"assured_workloads", "sovereign_controls", "compliance_posture",
		"confidential_computing", "confidential_vm", "confidential_space",

		// Analytics - Expanded
		"bigquery_dataset", "bigquery_table", "bigquery_view", "bigquery_routine",
		"bigquery_job", "bigquery_scheduled_query", "bigquery_data_transfer",
		"bigquery_ml_model", "bigquery_external_table", "bigquery_materialized_view",
		"bigquery_connection", "bigquery_reservation", "bigquery_capacity_commitment",
		"bigquery_bi_engine", "bigquery_omni", "analytics_hub",
		"dataflow_job", "dataflow_template", "dataflow_flex_template", "dataflow_pipeline",
		"dataflow_shuffle", "dataflow_streaming_engine", "dataflow_prime",
		"cloud_composer_environment", "cloud_composer_dag", "cloud_composer_plugin",
		"dataproc_cluster", "dataproc_job", "dataproc_workflow", "dataproc_autoscaling",
		"dataproc_metastore", "dataproc_batch", "dataproc_session",
		"pub_sub_topic", "pub_sub_subscription", "pub_sub_schema", "pub_sub_snapshot",
		"pub_sub_lite_topic", "pub_sub_lite_subscription", "pub_sub_lite_reservation",
		"datastream", "datastream_connection", "datastream_stream", "datastream_route",
		"data_catalog", "data_catalog_entry", "data_catalog_tag", "data_catalog_taxonomy",
		"dataplex", "dataplex_lake", "dataplex_zone", "dataplex_asset",
		"looker", "looker_instance", "looker_model", "looker_dashboard",
		"data_fusion", "data_fusion_instance", "data_fusion_pipeline",

		// Application Integration - Expanded
		"cloud_tasks_queue", "cloud_tasks_task", "cloud_scheduler_job",
		"cloud_workflows", "cloud_workflows_execution", "cloud_workflows_revision",
		"eventarc_trigger", "eventarc_channel", "eventarc_provider",
		"cloud_functions_connector", "cloud_run_connector", "app_engine_connector",
		"api_gateway", "api_gateway_api", "api_gateway_config", "api_gateway_deployment",
		"apigee", "apigee_organization", "apigee_environment", "apigee_proxy",
		"apigee_product", "apigee_developer", "apigee_app",
		"cloud_endpoints", "endpoints_service", "endpoints_deployment",
		"service_directory_namespace", "service_directory_service", "service_directory_endpoint",
		"traffic_director_mesh", "traffic_director_gateway", "traffic_director_route",
		"integration_connectors", "application_integration_platform",

		// Developer Tools - Expanded
		"cloud_source_repository", "cloud_build_trigger", "cloud_build_worker_pool",
		"artifact_registry_repository", "artifact_registry_package", "artifact_registry_version",
		"container_registry", "container_image", "container_scan",
		"cloud_deploy_pipeline", "cloud_deploy_release", "cloud_deploy_rollout",
		"cloud_deploy_target", "cloud_deploy_automation",
		"cloud_shell", "cloud_shell_environment", "cloud_code",
		"cloud_debugger", "cloud_trace", "cloud_profiler",
		"error_reporting", "error_group", "error_event",
		"firebase_project", "firebase_app", "firebase_hosting", "firebase_database",
		"firebase_auth", "firebase_storage", "firebase_functions", "firebase_messaging",
		"firebase_crashlytics", "firebase_performance", "firebase_test_lab",
		"game_services_realm", "game_services_cluster", "game_services_deployment",

		// Management & Monitoring - Expanded
		"cloud_monitoring_dashboard", "cloud_monitoring_alert", "cloud_monitoring_uptime",
		"cloud_monitoring_slo", "cloud_monitoring_metric", "cloud_monitoring_channel",
		"cloud_logging_sink", "cloud_logging_metric", "cloud_logging_exclusion",
		"cloud_logging_bucket", "cloud_logging_view", "cloud_logging_link",
		"cloud_ops_agent", "ops_agent_policy", "monitoring_query_language",
		"resource_manager_folder", "resource_manager_project", "resource_manager_lien",
		"resource_manager_tag", "resource_manager_tag_binding", "resource_manager_tag_value",
		"organization_policy", "organization_constraint", "policy_intelligence",
		"cloud_billing_account", "cloud_billing_budget", "cloud_billing_export",
		"cost_insights", "committed_use_discount", "sustained_use_discount",
		"recommender_recommendation", "recommender_insight", "recommender_config",
		"cloud_asset_inventory", "cloud_asset_feed", "cloud_asset_export",
		"deployment_manager_deployment", "deployment_manager_manifest", "deployment_manager_type",
		"config_controller", "policy_controller", "config_management",

		// AI & Machine Learning - Expanded
		"vertex_ai_dataset", "vertex_ai_model", "vertex_ai_endpoint", "vertex_ai_pipeline",
		"vertex_ai_training_job", "vertex_ai_batch_prediction", "vertex_ai_online_prediction",
		"vertex_ai_feature_store", "vertex_ai_feature", "vertex_ai_entity",
		"vertex_ai_metadata_store", "vertex_ai_experiment", "vertex_ai_tensorboard",
		"vertex_ai_index", "vertex_ai_index_endpoint", "vertex_ai_matching_engine",
		"vertex_ai_workbench", "vertex_ai_notebook", "vertex_ai_managed_notebook",
		"automl_dataset", "automl_model", "automl_prediction",
		"ai_platform_job", "ai_platform_version", "ai_platform_prediction",
		"cloud_vision_api", "cloud_vision_product", "cloud_vision_product_set",
		"cloud_natural_language", "cloud_translation", "cloud_text_to_speech",
		"cloud_speech_to_text", "cloud_speech_adaptation", "cloud_speech_recognition",
		"dialogflow_agent", "dialogflow_intent", "dialogflow_entity", "dialogflow_cx",
		"contact_center_ai", "agent_assist", "conversation_insights",
		"document_ai_processor", "document_ai_version", "document_ai_dataset",
		"cloud_talent_solution", "talent_job", "talent_company", "talent_profile",
		"recommendations_ai", "retail_api", "retail_catalog", "retail_product",
		"media_translation", "media_intelligence", "video_intelligence",

		// IoT - Expanded
		"iot_core_registry", "iot_core_device", "iot_core_gateway", "iot_core_config",
		"iot_core_state", "iot_core_telemetry", "iot_core_command",
		"cloud_iot_edge", "edge_device", "edge_module", "edge_deployment",
		"iot_analytics", "iot_data_pipeline", "iot_rule_engine",

		// Healthcare & Life Sciences - Expanded
		"healthcare_dataset", "healthcare_fhir_store", "healthcare_dicom_store",
		"healthcare_hl7v2_store", "healthcare_consent_store", "healthcare_annotation_store",
		"life_sciences_pipeline", "genomics_dataset", "genomics_variant_set",
		"alphafold", "biomedical_data_platform", "clinical_trials",

		// Media & Gaming - Expanded
		"transcoder_job", "transcoder_template", "transcoder_job_template",
		"live_stream_channel", "live_stream_input", "live_stream_event",
		"video_stitcher_cdn_key", "video_stitcher_slate", "video_stitcher_live_session",
		"game_servers_realm", "game_servers_cluster", "game_servers_deployment",
		"game_servers_config", "game_servers_rollout",

		// Industry Solutions - Expanded
		"retail_search", "retail_recommendation", "retail_catalog_management",
		"financial_services_api", "payment_gateway", "banking_api",
		"manufacturing_insights", "supply_chain_twin", "factory_optimization",
		"telecom_automation", "network_function", "5g_edge_application",
		"automotive_platform", "connected_vehicle", "autonomous_driving",
		"energy_insights", "sustainability_api", "carbon_footprint",

		// Migration - Expanded
		"migrate_for_compute", "migrate_assessment", "migrate_wave", "migrate_group",
		"database_migration_service", "database_assessment", "schema_conversion",
		"transfer_appliance", "offline_transfer", "online_transfer",
		"vmware_engine", "vmware_private_cloud", "vmware_cluster", "vmware_node",
		"bare_metal_solution", "bare_metal_server", "bare_metal_volume",
	}
}

// DiscoverGCPExpandedResources discovers all expanded GCP resource types
func DiscoverGCPExpandedResources(ctx context.Context, projectID string) ([]models.Resource, error) {
	var allResources []models.Resource
	resourceTypes := GetAllGCPExpandedResourceTypes()

	for _, resourceType := range resourceTypes {
		resources, err := discoverGCPResourceType(ctx, projectID, resourceType)
		if err != nil {
			// Log error but continue with other resource types
			fmt.Printf("Warning: Failed to discover GCP %s resources: %v\n", resourceType, err)
			continue
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

// discoverGCPResourceType discovers a specific GCP resource type
func discoverGCPResourceType(ctx context.Context, projectID, resourceType string) ([]models.Resource, error) {
	// Map internal resource type to GCP API format
	gcpService, gcpType := mapToGCPResourceType(resourceType)

	// Use gcloud commands to discover resources
	var cmd *exec.Cmd
	switch gcpService {
	case "compute":
		cmd = exec.CommandContext(ctx, "gcloud", "compute", gcpType, "list",
			"--project", projectID, "--format", "json")
	case "storage":
		if gcpType == "buckets" {
			cmd = exec.CommandContext(ctx, "gsutil", "ls", "-L", "-b", "-p", projectID)
		} else {
			cmd = exec.CommandContext(ctx, "gcloud", "storage", gcpType, "list",
				"--project", projectID, "--format", "json")
		}
	case "sql":
		cmd = exec.CommandContext(ctx, "gcloud", "sql", gcpType, "list",
			"--project", projectID, "--format", "json")
	case "container":
		cmd = exec.CommandContext(ctx, "gcloud", "container", gcpType, "list",
			"--project", projectID, "--format", "json")
	default:
		cmd = exec.CommandContext(ctx, "gcloud", gcpService, gcpType, "list",
			"--project", projectID, "--format", "json")
	}

	output, err := cmd.Output()
	if err != nil {
		// Some resource types might not exist or be enabled
		return []models.Resource{}, nil
	}

	var gcpResources []map[string]interface{}
	if err := json.Unmarshal(output, &gcpResources); err != nil {
		// Try parsing as single object
		var singleResource map[string]interface{}
		if err := json.Unmarshal(output, &singleResource); err != nil {
			return nil, fmt.Errorf("failed to parse GCP response: %w", err)
		}
		gcpResources = []map[string]interface{}{singleResource}
	}

	var resources []models.Resource
	for _, gcpResource := range gcpResources {
		resource := models.Resource{
			ID:         getGCPResourceID(gcpResource),
			Name:       getGCPResourceName(gcpResource),
			Type:       resourceType,
			Provider:   "gcp",
			Region:     getGCPResourceLocation(gcpResource),
			Properties: gcpResource,
			Tags:       extractGCPLabels(gcpResource),
			CreatedAt:  parseGCPTimestamp(getGCPStringValue(gcpResource, "creationTimestamp")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// mapToGCPResourceType maps internal type names to GCP service and type
func mapToGCPResourceType(resourceType string) (string, string) {
	// Map of internal names to GCP service/type format
	typeMap := map[string]struct{ service, resourceType string }{
		"compute_instance":      {"compute", "instances"},
		"compute_disk":          {"compute", "disks"},
		"gke_cluster":           {"container", "clusters"},
		"cloud_sql_instance":    {"sql", "instances"},
		"storage_bucket":        {"storage", "buckets"},
		"vpc_network":           {"compute", "networks"},
		"load_balancer":         {"compute", "forwarding-rules"},
		"cloud_functions":       {"functions", "functions"},
		"bigquery_dataset":      {"bigquery", "datasets"},
		"pub_sub_topic":         {"pubsub", "topics"},
		"cloud_run_service":     {"run", "services"},
		"vertex_ai_model":       {"ai-platform", "models"},
		"iot_core_registry":     {"cloudiot", "registries"},
		"secret_manager_secret": {"secrets", "secrets"},
		"cloud_kms_keyring":     {"kms", "keyrings"},
		// Add more mappings as needed
	}

	if mapping, ok := typeMap[resourceType]; ok {
		return mapping.service, mapping.resourceType
	}

	// Default: try to parse from resource type
	parts := strings.Split(resourceType, "_")
	if len(parts) >= 2 {
		return parts[0], strings.Join(parts[1:], "-") + "s"
	}

	return "compute", resourceType
}

// getGCPResourceID extracts the resource ID from GCP resource
func getGCPResourceID(resource map[string]interface{}) string {
	// Try different common ID fields
	idFields := []string{"id", "name", "selfLink", "resourceName"}
	for _, field := range idFields {
		if val, ok := resource[field].(string); ok && val != "" {
			return val
		}
	}
	return ""
}

// getGCPResourceName extracts the resource name from GCP resource
func getGCPResourceName(resource map[string]interface{}) string {
	// Try different common name fields
	nameFields := []string{"name", "displayName", "title"}
	for _, field := range nameFields {
		if val, ok := resource[field].(string); ok && val != "" {
			return val
		}
	}
	return getGCPResourceID(resource)
}

// getGCPResourceLocation extracts the location/region from GCP resource
func getGCPResourceLocation(resource map[string]interface{}) string {
	// Try different location fields
	locationFields := []string{"location", "region", "zone", "locations"}
	for _, field := range locationFields {
		if val, ok := resource[field].(string); ok && val != "" {
			// Extract region from zone (e.g., us-central1-a -> us-central1)
			if strings.Contains(val, "-") && strings.Count(val, "-") >= 2 {
				parts := strings.Split(val, "-")
				if len(parts) >= 3 {
					return strings.Join(parts[:2], "-")
				}
			}
			return val
		}
	}
	return "global"
}

// extractGCPLabels extracts labels from GCP resource
func extractGCPLabels(resource map[string]interface{}) map[string]string {
	labels := make(map[string]string)

	// Try both "labels" and "tags" fields
	for _, field := range []string{"labels", "tags"} {
		if labelData, ok := resource[field].(map[string]interface{}); ok {
			for key, value := range labelData {
				if strValue, ok := value.(string); ok {
					labels[key] = strValue
				}
			}
		}
	}

	return labels
}

// getGCPStringValue safely extracts string value from map
func getGCPStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// parseGCPTimestamp parses GCP timestamp string
func parseGCPTimestamp(timestamp string) time.Time {
	if timestamp == "" {
		return time.Time{}
	}

	// Try different timestamp formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestamp); err == nil {
			return t
		}
	}

	return time.Time{}
}
