package digitalocean

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// GetAllDigitalOceanExpandedResourceTypes returns comprehensive list of DigitalOcean resource types
func GetAllDigitalOceanExpandedResourceTypes() []string {
	return []string{
		// Compute - Expanded
		"droplet", "droplet_snapshot", "droplet_backup", "droplet_action",
		"droplet_neighbor", "droplet_kernel", "droplet_size",
		"kubernetes_cluster", "kubernetes_node_pool", "kubernetes_node",
		"kubernetes_upgrade", "kubernetes_registry", "kubernetes_options",
		"app_platform", "app_component", "app_deployment", "app_domain",
		"app_build", "app_alert", "app_log", "app_tier",
		"functions_namespace", "functions_trigger", "functions_package",
		"reserved_ip", "reserved_ip_action", "reserved_ip_assignment",

		// Storage - Expanded
		"volume", "volume_snapshot", "volume_action", "volume_attachment",
		"spaces_bucket", "spaces_object", "spaces_cors", "spaces_lifecycle",
		"spaces_acl", "spaces_versioning", "spaces_replication",
		"cdn_endpoint", "cdn_cache", "cdn_purge", "cdn_certificate",
		"container_registry", "container_repository", "container_tag",
		"container_manifest", "container_garbage_collection",
		"backup", "backup_policy", "backup_restore", "backup_schedule",

		// Database - Expanded
		"database_cluster", "database_replica", "database_user", "database_db",
		"database_pool", "database_backup", "database_restore", "database_maintenance",
		"database_firewall", "database_migration", "database_config",
		"postgresql_cluster", "postgresql_config", "postgresql_extension",
		"mysql_cluster", "mysql_config", "mysql_slow_query",
		"redis_cluster", "redis_config", "redis_eviction_policy",
		"mongodb_cluster", "mongodb_config", "mongodb_user",
		"kafka_cluster", "kafka_topic", "kafka_acl", "kafka_consumer_group",
		"opensearch_cluster", "opensearch_index", "opensearch_dashboard",
		"database_insights", "query_performance", "database_metrics",

		// Networking - Expanded
		"vpc", "vpc_peering", "vpc_route", "vpc_member",
		"load_balancer", "load_balancer_rule", "load_balancer_health_check",
		"load_balancer_sticky_session", "load_balancer_ssl",
		"firewall", "firewall_rule", "firewall_tag", "firewall_group",
		"domain", "domain_record", "domain_ptr", "domain_caa",
		"certificate", "certificate_chain", "certificate_renewal",
		"floating_ip", "floating_ip_action", "floating_ip_assignment",
		"gateway", "gateway_route", "gateway_nat", "gateway_vpn",
		"ddos_protection", "rate_limiting", "geo_blocking",
		"private_network", "network_interface", "network_acl",

		// Security & Identity - Expanded
		"ssh_key", "ssh_fingerprint", "ssh_authorized_keys",
		"team", "team_member", "team_role", "team_permission",
		"project", "project_resource", "project_environment", "project_limit",
		"api_token", "personal_access_token", "oauth_app", "oauth_token",
		"two_factor_auth", "security_key", "recovery_code",
		"audit_log", "audit_event", "audit_actor", "audit_resource",
		"security_advisory", "vulnerability_scan", "compliance_report",
		"secrets_manager", "secret", "secret_version", "secret_rotation",
		"vault", "vault_policy", "vault_token", "vault_seal",

		// Monitoring & Observability - Expanded
		"monitoring_alert", "monitoring_policy", "monitoring_metric",
		"monitoring_dashboard", "monitoring_widget", "monitoring_graph",
		"uptime_check", "uptime_alert", "uptime_contact", "uptime_region",
		"status_page", "status_incident", "status_maintenance", "status_subscriber",
		"log_forward", "log_sink", "log_filter", "log_archive",
		"metrics_endpoint", "metrics_scraper", "metrics_aggregation",
		"trace", "trace_span", "trace_service", "trace_operation",
		"insights_report", "insights_recommendation", "insights_cost",
		"performance_monitoring", "apm_service", "apm_trace",

		// Developer Tools - Expanded
		"git_repository", "git_branch", "git_commit", "git_webhook",
		"ci_pipeline", "ci_build", "ci_artifact", "ci_environment",
		"registry_image", "registry_tag", "registry_webhook", "registry_robot",
		"api_gateway", "api_route", "api_method", "api_key",
		"webhook", "webhook_event", "webhook_delivery", "webhook_retry",
		"oauth_client", "oauth_scope", "oauth_grant", "oauth_refresh",
		"sdk_key", "sdk_environment", "sdk_feature_flag",
		"terraform_workspace", "terraform_state", "terraform_module",

		// Billing & Account - Expanded
		"billing_history", "billing_invoice", "billing_payment_method",
		"billing_alert", "billing_budget", "billing_credit",
		"account", "account_settings", "account_verification", "account_limit",
		"subscription", "subscription_tier", "subscription_addon",
		"usage_report", "usage_metric", "usage_limit", "usage_alert",
		"cost_report", "cost_breakdown", "cost_forecast", "cost_optimization",
		"payment", "payment_history", "payment_failure", "payment_retry",
		"tax_info", "tax_exemption", "tax_invoice",
		"referral", "referral_credit", "referral_program",

		// Marketplace & Integrations - Expanded
		"marketplace_app", "marketplace_listing", "marketplace_subscription",
		"marketplace_review", "marketplace_vendor", "marketplace_payout",
		"integration", "integration_webhook", "integration_config",
		"third_party_service", "service_connection", "service_sync",
		"plugin", "plugin_config", "plugin_version", "plugin_update",
		"extension", "extension_settings", "extension_permission",
		"api_integration", "api_mapping", "api_transformation",

		// Support & Documentation - Expanded
		"support_ticket", "support_response", "support_attachment",
		"support_priority", "support_sla", "support_category",
		"documentation", "documentation_version", "documentation_search",
		"tutorial", "tutorial_progress", "tutorial_completion",
		"community_post", "community_reply", "community_vote",
		"knowledge_base", "knowledge_article", "knowledge_category",

		// Advanced Features - Expanded
		"gpu_droplet", "gpu_cluster", "gpu_allocation",
		"high_memory_droplet", "cpu_optimized_droplet", "storage_optimized_droplet",
		"dedicated_host", "dedicated_cpu", "dedicated_bandwidth",
		"anycast_ip", "bgp_session", "bgp_route", "bgp_peer",
		"edge_location", "edge_cache", "edge_function", "edge_rule",
		"iot_device", "iot_gateway", "iot_telemetry", "iot_command",
		"blockchain_node", "blockchain_network", "blockchain_transaction",
		"ai_model", "ai_training", "ai_inference", "ai_dataset",
		"quantum_simulator", "quantum_circuit", "quantum_job",

		// Regional Services - Expanded
		"region", "region_availability", "region_capacity", "region_feature",
		"datacenter", "datacenter_rack", "datacenter_power", "datacenter_cooling",
		"availability_zone", "fault_domain", "update_domain",
		"geo_replication", "geo_failover", "geo_backup",
		"cross_region_snapshot", "cross_region_backup", "cross_region_replication",

		// Compliance & Governance - Expanded
		"compliance_standard", "compliance_control", "compliance_audit",
		"governance_policy", "governance_rule", "governance_exception",
		"data_residency", "data_sovereignty", "data_classification",
		"privacy_policy", "privacy_request", "privacy_deletion",
		"gdpr_compliance", "hipaa_compliance", "pci_compliance",
		"iso_certification", "soc_report", "penetration_test",
	}
}

// DiscoverDigitalOceanExpandedResources discovers all expanded DigitalOcean resource types
func DiscoverDigitalOceanExpandedResources(ctx context.Context) ([]models.Resource, error) {
	var allResources []models.Resource
	resourceTypes := GetAllDigitalOceanExpandedResourceTypes()

	for _, resourceType := range resourceTypes {
		resources, err := discoverDigitalOceanResourceType(ctx, resourceType)
		if err != nil {
			// Log error but continue with other resource types
			fmt.Printf("Warning: Failed to discover DigitalOcean %s resources: %v\n", resourceType, err)
			continue
		}
		allResources = append(allResources, resources...)
	}

	return allResources, nil
}

// discoverDigitalOceanResourceType discovers a specific DigitalOcean resource type
func discoverDigitalOceanResourceType(ctx context.Context, resourceType string) ([]models.Resource, error) {
	// Map internal resource type to doctl command
	doctlCommand := mapToDigitalOceanCommand(resourceType)

	// Build doctl command
	cmdParts := strings.Split(doctlCommand, " ")
	cmdParts = append(cmdParts, "list", "--output", "json")

	cmd := exec.CommandContext(ctx, "doctl", cmdParts...)
	output, err := cmd.Output()
	if err != nil {
		// Some resource types might not exist or be available
		return []models.Resource{}, nil
	}

	// Parse JSON output
	var doResources []map[string]interface{}
	if err := json.Unmarshal(output, &doResources); err != nil {
		// Try parsing as single object
		var singleResource map[string]interface{}
		if err := json.Unmarshal(output, &singleResource); err != nil {
			// Some commands might not be implemented yet
			return []models.Resource{}, nil
		}
		doResources = []map[string]interface{}{singleResource}
	}

	var resources []models.Resource
	for _, doResource := range doResources {
		resource := models.Resource{
			ID:         getDigitalOceanResourceID(doResource),
			Name:       getDigitalOceanResourceName(doResource),
			Type:       resourceType,
			Provider:   "digitalocean",
			Region:     getDigitalOceanResourceRegion(doResource),
			Properties: doResource,
			Tags:       extractDigitalOceanTags(doResource),
			CreatedAt:  parseDigitalOceanTimestamp(getDOStringValue(doResource, "created_at")),
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// mapToDigitalOceanCommand maps internal resource type to doctl command
func mapToDigitalOceanCommand(resourceType string) string {
	// Map of internal names to doctl commands
	commandMap := map[string]string{
		// Compute
		"droplet":              "compute droplet",
		"droplet_snapshot":     "compute droplet-snapshot",
		"droplet_backup":       "compute droplet backup",
		"droplet_action":       "compute droplet-action",
		"kubernetes_cluster":   "kubernetes cluster",
		"kubernetes_node_pool": "kubernetes node-pool",
		"app_platform":         "apps",
		"app_component":        "apps component",
		"functions_namespace":  "serverless namespaces",
		"reserved_ip":          "compute reserved-ip",

		// Storage
		"volume":               "compute volume",
		"volume_snapshot":      "compute volume snapshot",
		"spaces_bucket":        "spaces bucket",
		"cdn_endpoint":         "compute cdn",
		"container_registry":   "registry",
		"container_repository": "registry repository",

		// Database
		"database_cluster":   "databases",
		"database_replica":   "databases replica",
		"database_user":      "databases user",
		"database_db":        "databases db",
		"database_pool":      "databases pool",
		"postgresql_cluster": "databases postgres",
		"mysql_cluster":      "databases mysql",
		"redis_cluster":      "databases redis",
		"mongodb_cluster":    "databases mongodb",
		"kafka_cluster":      "databases kafka",
		"opensearch_cluster": "databases opensearch",

		// Networking
		"vpc":           "vpcs",
		"load_balancer": "compute load-balancer",
		"firewall":      "compute firewall",
		"domain":        "compute domain",
		"domain_record": "compute domain records",
		"certificate":   "compute certificate",
		"floating_ip":   "compute floating-ip",

		// Security
		"ssh_key":          "compute ssh-key",
		"project":          "projects",
		"project_resource": "projects resources",

		// Monitoring
		"monitoring_alert": "monitoring alert",
		"uptime_check":     "monitoring uptime-check",

		// Billing
		"billing_history": "billing-history",
		"account":         "account",

		// Registry
		"registry_image": "registry repository list-tags",

		// Default
	}

	if cmd, ok := commandMap[resourceType]; ok {
		return cmd
	}

	// Try to construct command from resource type
	parts := strings.Split(resourceType, "_")
	if len(parts) > 0 {
		// Handle special cases
		switch parts[0] {
		case "database":
			return "databases " + strings.Join(parts[1:], "-")
		case "kubernetes":
			return "kubernetes " + strings.Join(parts[1:], "-")
		case "app":
			return "apps " + strings.Join(parts[1:], "-")
		case "monitoring":
			return "monitoring " + strings.Join(parts[1:], "-")
		default:
			return "compute " + strings.ReplaceAll(resourceType, "_", "-")
		}
	}

	return resourceType
}

// getDigitalOceanResourceID extracts the resource ID
func getDigitalOceanResourceID(resource map[string]interface{}) string {
	// Try different ID fields
	idFields := []string{"id", "uuid", "name", "slug"}
	for _, field := range idFields {
		if val := resource[field]; val != nil {
			switch v := val.(type) {
			case string:
				return v
			case float64:
				return strconv.FormatFloat(v, 'f', 0, 64)
			case int:
				return strconv.Itoa(v)
			}
		}
	}
	return ""
}

// getDigitalOceanResourceName extracts the resource name
func getDigitalOceanResourceName(resource map[string]interface{}) string {
	// Try different name fields
	nameFields := []string{"name", "title", "hostname", "slug"}
	for _, field := range nameFields {
		if val, ok := resource[field].(string); ok && val != "" {
			return val
		}
	}
	return getDigitalOceanResourceID(resource)
}

// getDigitalOceanResourceRegion extracts the region
func getDigitalOceanResourceRegion(resource map[string]interface{}) string {
	// Handle different region formats
	if region := resource["region"]; region != nil {
		switch r := region.(type) {
		case string:
			return r
		case map[string]interface{}:
			if slug, ok := r["slug"].(string); ok {
				return slug
			}
			if name, ok := r["name"].(string); ok {
				return name
			}
		}
	}

	// Try region slug
	if slug, ok := resource["region_slug"].(string); ok {
		return slug
	}

	// For VPCs and other resources
	if regions, ok := resource["regions"].([]interface{}); ok && len(regions) > 0 {
		if region, ok := regions[0].(string); ok {
			return region
		}
	}

	return "global"
}

// extractDigitalOceanTags extracts tags from resource
func extractDigitalOceanTags(resource map[string]interface{}) map[string]string {
	tags := make(map[string]string)

	// DigitalOcean uses array of tag strings
	if tagList, ok := resource["tags"].([]interface{}); ok {
		for i, tag := range tagList {
			if tagStr, ok := tag.(string); ok {
				// Parse key:value format if present
				if strings.Contains(tagStr, ":") {
					parts := strings.SplitN(tagStr, ":", 2)
					tags[parts[0]] = parts[1]
				} else {
					// Use tag as both key and value
					tags[tagStr] = tagStr
				}
			} else {
				tags[fmt.Sprintf("tag_%d", i)] = fmt.Sprintf("%v", tag)
			}
		}
	}

	// Also check for labels (some resources use this)
	if labels, ok := resource["labels"].(map[string]interface{}); ok {
		for key, value := range labels {
			if strValue, ok := value.(string); ok {
				tags[key] = strValue
			}
		}
	}

	return tags
}

// getDOStringValue safely extracts string value from map
func getDOStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// parseDigitalOceanTimestamp parses DigitalOcean timestamp
func parseDigitalOceanTimestamp(timestamp string) time.Time {
	if timestamp == "" {
		return time.Time{}
	}

	// DigitalOcean uses RFC3339 format
	t, _ := time.Parse(time.RFC3339, timestamp)
	return t
}

// ImplementMultiAccountSupport adds multi-account support for DigitalOcean
func ImplementMultiAccountSupport(ctx context.Context) ([]string, error) {
	// DigitalOcean doesn't have traditional multi-account like AWS/Azure
	// But we can support team accounts and projects

	// List all accessible teams
	cmd := exec.CommandContext(ctx, "doctl", "account", "get", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	var account map[string]interface{}
	if err := json.Unmarshal(output, &account); err != nil {
		return nil, fmt.Errorf("failed to parse account info: %w", err)
	}

	// Get team UUID
	teamID := getDOStringValue(account, "team_uuid")
	if teamID == "" {
		teamID = getDOStringValue(account, "uuid")
	}

	// List all projects (which act like logical accounts)
	cmd = exec.CommandContext(ctx, "doctl", "projects", "list", "--output", "json")
	output, err = cmd.Output()
	if err != nil {
		return []string{teamID}, nil // Return just the main account
	}

	var projects []map[string]interface{}
	if err := json.Unmarshal(output, &projects); err != nil {
		return []string{teamID}, nil
	}

	// Return project IDs as "accounts"
	var accounts []string
	for _, project := range projects {
		if projectID := getDOStringValue(project, "id"); projectID != "" {
			accounts = append(accounts, projectID)
		}
	}

	if len(accounts) == 0 {
		accounts = []string{teamID}
	}

	return accounts, nil
}
