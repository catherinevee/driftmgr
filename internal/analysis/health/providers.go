package health

import (
	"fmt"
	"strings"

	"github.com/catherinevee/driftmgr/internal/state/parser"
)

// AWSHealthChecker implements health checks for AWS resources
type AWSHealthChecker struct {
	requiredAttributes map[string][]string
	deprecatedAttributes map[string][]string
	securityRules map[string][]SecurityRule
}

// NewAWSHealthChecker creates a new AWS health checker
func NewAWSHealthChecker() *AWSHealthChecker {
	checker := &AWSHealthChecker{
		requiredAttributes: make(map[string][]string),
		deprecatedAttributes: make(map[string][]string),
		securityRules: make(map[string][]SecurityRule),
	}
	
	checker.initialize()
	return checker
}

func (c *AWSHealthChecker) initialize() {
	// Define required attributes for common AWS resources
	c.requiredAttributes["aws_instance"] = []string{"ami", "instance_type"}
	c.requiredAttributes["aws_s3_bucket"] = []string{"bucket"}
	c.requiredAttributes["aws_security_group"] = []string{"name"}
	c.requiredAttributes["aws_rds_instance"] = []string{"engine", "instance_class"}
	
	// Define deprecated attributes
	c.deprecatedAttributes["aws_s3_bucket"] = []string{"acl"}
	c.deprecatedAttributes["aws_instance"] = []string{"network_interface"}
	
	// Define security rules
	c.securityRules["aws_s3_bucket"] = []SecurityRule{
		{
			Name:        "public_access_block",
			Description: "S3 bucket should have public access blocked",
			Check: func(attrs map[string]interface{}) bool {
				if publicBlock, exists := attrs["public_access_block_configuration"]; exists {
					if config, ok := publicBlock.(map[string]interface{}); ok {
						return config["block_public_acls"] == true &&
							   config["block_public_policy"] == true &&
							   config["ignore_public_acls"] == true &&
							   config["restrict_public_buckets"] == true
					}
				}
				return false
			},
			Severity:    SeverityHigh,
			Remediation: "Enable all public access block settings on the S3 bucket",
		},
		{
			Name:        "encryption",
			Description: "S3 bucket should have encryption enabled",
			Check: func(attrs map[string]interface{}) bool {
				_, exists := attrs["server_side_encryption_configuration"]
				return exists
			},
			Severity:    SeverityHigh,
			Remediation: "Enable server-side encryption for the S3 bucket",
		},
	}
	
	c.securityRules["aws_instance"] = []SecurityRule{
		{
			Name:        "public_ip",
			Description: "EC2 instance should not have a public IP unless necessary",
			Check: func(attrs map[string]interface{}) bool {
				publicIP, exists := attrs["associate_public_ip_address"]
				if !exists {
					return true
				}
				return publicIP == false
			},
			Severity:    SeverityMedium,
			Remediation: "Consider removing public IP if not required",
		},
		{
			Name:        "monitoring",
			Description: "EC2 instance should have monitoring enabled",
			Check: func(attrs map[string]interface{}) bool {
				monitoring, exists := attrs["monitoring"]
				return exists && monitoring == true
			},
			Severity:    SeverityLow,
			Remediation: "Enable detailed monitoring for the EC2 instance",
		},
	}
	
	c.securityRules["aws_rds_instance"] = []SecurityRule{
		{
			Name:        "backup_retention",
			Description: "RDS instance should have adequate backup retention",
			Check: func(attrs map[string]interface{}) bool {
				retention, exists := attrs["backup_retention_period"]
				if !exists {
					return false
				}
				if days, ok := retention.(float64); ok {
					return days >= 7
				}
				return false
			},
			Severity:    SeverityMedium,
			Remediation: "Set backup retention period to at least 7 days",
		},
		{
			Name:        "encryption",
			Description: "RDS instance should have encryption enabled",
			Check: func(attrs map[string]interface{}) bool {
				encrypted, exists := attrs["storage_encrypted"]
				return exists && encrypted == true
			},
			Severity:    SeverityHigh,
			Remediation: "Enable storage encryption for the RDS instance",
		},
	}
}

func (c *AWSHealthChecker) CheckResource(resource *parser.Resource, instance *parser.Instance) *HealthReport {
	report := &HealthReport{
		Issues:      make([]HealthIssue, 0),
		Suggestions: make([]string, 0),
	}
	
	// AWS-specific checks
	switch resource.Type {
	case "aws_instance":
		c.checkEC2Instance(instance, report)
	case "aws_s3_bucket":
		c.checkS3Bucket(instance, report)
	case "aws_security_group":
		c.checkSecurityGroup(instance, report)
	case "aws_rds_instance":
		c.checkRDSInstance(instance, report)
	}
	
	return report
}

func (c *AWSHealthChecker) checkEC2Instance(instance *parser.Instance, report *HealthReport) {
	if instance.Attributes == nil {
		return
	}
	
	// Check instance type for previous generation
	if instanceType, exists := instance.Attributes["instance_type"]; exists {
		if typeStr, ok := instanceType.(string); ok {
			if strings.HasPrefix(typeStr, "t2.") || strings.HasPrefix(typeStr, "m3.") || 
			   strings.HasPrefix(typeStr, "c3.") || strings.HasPrefix(typeStr, "r3.") {
				report.Issues = append(report.Issues, HealthIssue{
					Type:     IssueTypePerformance,
					Severity: SeverityLow,
					Message:  fmt.Sprintf("Instance type %s is previous generation", typeStr),
					Field:    "instance_type",
					CurrentValue: typeStr,
				})
				report.Suggestions = append(report.Suggestions, 
					"Consider upgrading to current generation instance types for better performance and cost")
			}
		}
	}
}

func (c *AWSHealthChecker) checkS3Bucket(instance *parser.Instance, report *HealthReport) {
	if instance.Attributes == nil {
		return
	}
	
	// Check versioning
	if _, exists := instance.Attributes["versioning"]; !exists {
		report.Issues = append(report.Issues, HealthIssue{
			Type:     IssueTypeConfiguration,
			Severity: SeverityMedium,
			Message:  "S3 bucket versioning is not configured",
			Field:    "versioning",
		})
		report.Suggestions = append(report.Suggestions, "Enable versioning for data protection")
	}
	
	// Check lifecycle rules
	if _, exists := instance.Attributes["lifecycle_rule"]; !exists {
		report.Suggestions = append(report.Suggestions, 
			"Consider adding lifecycle rules to manage storage costs")
	}
}

func (c *AWSHealthChecker) checkSecurityGroup(instance *parser.Instance, report *HealthReport) {
	if instance.Attributes == nil {
		return
	}
	
	// Check for overly permissive ingress rules
	if ingress, exists := instance.Attributes["ingress"]; exists {
		if rules, ok := ingress.([]interface{}); ok {
			for _, rule := range rules {
				if ruleMap, ok := rule.(map[string]interface{}); ok {
					// Check for 0.0.0.0/0
					if cidr, exists := ruleMap["cidr_blocks"]; exists {
						if blocks, ok := cidr.([]interface{}); ok {
							for _, block := range blocks {
								if blockStr, ok := block.(string); ok && blockStr == "0.0.0.0/0" {
									fromPort := ruleMap["from_port"]
									toPort := ruleMap["to_port"]
									report.Issues = append(report.Issues, HealthIssue{
										Type:     IssueTypeSecurity,
										Severity: SeverityHigh,
										Message:  fmt.Sprintf("Security group allows unrestricted access from 0.0.0.0/0 on ports %v-%v", 
											fromPort, toPort),
										Field:    "ingress",
									})
								}
							}
						}
					}
				}
			}
		}
	}
}

func (c *AWSHealthChecker) checkRDSInstance(instance *parser.Instance, report *HealthReport) {
	if instance.Attributes == nil {
		return
	}
	
	// Check multi-AZ deployment
	if multiAZ, exists := instance.Attributes["multi_az"]; exists {
		if enabled, ok := multiAZ.(bool); ok && !enabled {
			report.Issues = append(report.Issues, HealthIssue{
				Type:     IssueTypeConfiguration,
				Severity: SeverityMedium,
				Message:  "RDS instance is not configured for Multi-AZ deployment",
				Field:    "multi_az",
				CurrentValue: false,
				ExpectedValue: true,
			})
			report.Suggestions = append(report.Suggestions, 
				"Enable Multi-AZ for high availability")
		}
	}
}

func (c *AWSHealthChecker) GetRequiredAttributes(resourceType string) []string {
	if attrs, exists := c.requiredAttributes[resourceType]; exists {
		return attrs
	}
	return []string{}
}

func (c *AWSHealthChecker) GetDeprecatedAttributes(resourceType string) []string {
	if attrs, exists := c.deprecatedAttributes[resourceType]; exists {
		return attrs
	}
	return []string{}
}

func (c *AWSHealthChecker) GetSecurityRules(resourceType string) []SecurityRule {
	if rules, exists := c.securityRules[resourceType]; exists {
		return rules
	}
	return []SecurityRule{}
}

// AzureHealthChecker implements health checks for Azure resources
type AzureHealthChecker struct {
	requiredAttributes map[string][]string
	deprecatedAttributes map[string][]string
	securityRules map[string][]SecurityRule
}

// NewAzureHealthChecker creates a new Azure health checker
func NewAzureHealthChecker() *AzureHealthChecker {
	checker := &AzureHealthChecker{
		requiredAttributes: make(map[string][]string),
		deprecatedAttributes: make(map[string][]string),
		securityRules: make(map[string][]SecurityRule),
	}
	
	checker.initialize()
	return checker
}

func (c *AzureHealthChecker) initialize() {
	// Define required attributes for common Azure resources
	c.requiredAttributes["azurerm_virtual_machine"] = []string{"name", "location", "resource_group_name", "vm_size"}
	c.requiredAttributes["azurerm_storage_account"] = []string{"name", "resource_group_name", "location", "account_tier"}
	c.requiredAttributes["azurerm_network_security_group"] = []string{"name", "location", "resource_group_name"}
	
	// Define security rules
	c.securityRules["azurerm_storage_account"] = []SecurityRule{
		{
			Name:        "https_only",
			Description: "Storage account should enforce HTTPS",
			Check: func(attrs map[string]interface{}) bool {
				https, exists := attrs["enable_https_traffic_only"]
				return exists && https == true
			},
			Severity:    SeverityHigh,
			Remediation: "Set enable_https_traffic_only to true",
		},
		{
			Name:        "min_tls_version",
			Description: "Storage account should use minimum TLS version 1.2",
			Check: func(attrs map[string]interface{}) bool {
				tls, exists := attrs["min_tls_version"]
				return exists && tls == "TLS1_2"
			},
			Severity:    SeverityMedium,
			Remediation: "Set min_tls_version to TLS1_2",
		},
	}
}

func (c *AzureHealthChecker) CheckResource(resource *parser.Resource, instance *parser.Instance) *HealthReport {
	report := &HealthReport{
		Issues:      make([]HealthIssue, 0),
		Suggestions: make([]string, 0),
	}
	
	// Azure-specific checks
	switch resource.Type {
	case "azurerm_virtual_machine":
		c.checkVirtualMachine(instance, report)
	case "azurerm_storage_account":
		c.checkStorageAccount(instance, report)
	}
	
	return report
}

func (c *AzureHealthChecker) checkVirtualMachine(instance *parser.Instance, report *HealthReport) {
	if instance.Attributes == nil {
		return
	}
	
	// Check for managed disks
	if _, exists := instance.Attributes["storage_os_disk"]; exists {
		report.Issues = append(report.Issues, HealthIssue{
			Type:     IssueTypeDeprecated,
			Severity: SeverityMedium,
			Message:  "VM is using unmanaged disks",
			Field:    "storage_os_disk",
		})
		report.Suggestions = append(report.Suggestions, 
			"Migrate to managed disks for better reliability and features")
	}
}

func (c *AzureHealthChecker) checkStorageAccount(instance *parser.Instance, report *HealthReport) {
	if instance.Attributes == nil {
		return
	}
	
	// Check replication type
	if replication, exists := instance.Attributes["account_replication_type"]; exists {
		if replType, ok := replication.(string); ok && replType == "LRS" {
			report.Issues = append(report.Issues, HealthIssue{
				Type:     IssueTypeConfiguration,
				Severity: SeverityLow,
				Message:  "Storage account uses locally redundant storage only",
				Field:    "account_replication_type",
				CurrentValue: "LRS",
			})
			report.Suggestions = append(report.Suggestions, 
				"Consider using GRS or ZRS for better redundancy")
		}
	}
}

func (c *AzureHealthChecker) GetRequiredAttributes(resourceType string) []string {
	if attrs, exists := c.requiredAttributes[resourceType]; exists {
		return attrs
	}
	return []string{}
}

func (c *AzureHealthChecker) GetDeprecatedAttributes(resourceType string) []string {
	if attrs, exists := c.deprecatedAttributes[resourceType]; exists {
		return attrs
	}
	return []string{}
}

func (c *AzureHealthChecker) GetSecurityRules(resourceType string) []SecurityRule {
	if rules, exists := c.securityRules[resourceType]; exists {
		return rules
	}
	return []SecurityRule{}
}

// GCPHealthChecker implements health checks for GCP resources
type GCPHealthChecker struct {
	requiredAttributes map[string][]string
	deprecatedAttributes map[string][]string
	securityRules map[string][]SecurityRule
}

// NewGCPHealthChecker creates a new GCP health checker
func NewGCPHealthChecker() *GCPHealthChecker {
	checker := &GCPHealthChecker{
		requiredAttributes: make(map[string][]string),
		deprecatedAttributes: make(map[string][]string),
		securityRules: make(map[string][]SecurityRule),
	}
	
	checker.initialize()
	return checker
}

func (c *GCPHealthChecker) initialize() {
	// Define required attributes for common GCP resources
	c.requiredAttributes["google_compute_instance"] = []string{"name", "machine_type", "zone"}
	c.requiredAttributes["google_storage_bucket"] = []string{"name", "location"}
	c.requiredAttributes["google_sql_database_instance"] = []string{"database_version", "settings"}
	
	// Define security rules
	c.securityRules["google_storage_bucket"] = []SecurityRule{
		{
			Name:        "uniform_access",
			Description: "Storage bucket should use uniform bucket-level access",
			Check: func(attrs map[string]interface{}) bool {
				uniform, exists := attrs["uniform_bucket_level_access"]
				if exists {
					if config, ok := uniform.(map[string]interface{}); ok {
						return config["enabled"] == true
					}
				}
				return false
			},
			Severity:    SeverityMedium,
			Remediation: "Enable uniform bucket-level access",
		},
	}
	
	c.securityRules["google_compute_instance"] = []SecurityRule{
		{
			Name:        "shielded_vm",
			Description: "Compute instance should use Shielded VM features",
			Check: func(attrs map[string]interface{}) bool {
				_, exists := attrs["shielded_instance_config"]
				return exists
			},
			Severity:    SeverityMedium,
			Remediation: "Enable Shielded VM features for better security",
		},
	}
}

func (c *GCPHealthChecker) CheckResource(resource *parser.Resource, instance *parser.Instance) *HealthReport {
	report := &HealthReport{
		Issues:      make([]HealthIssue, 0),
		Suggestions: make([]string, 0),
	}
	
	// GCP-specific checks
	switch resource.Type {
	case "google_compute_instance":
		c.checkComputeInstance(instance, report)
	case "google_storage_bucket":
		c.checkStorageBucket(instance, report)
	}
	
	return report
}

func (c *GCPHealthChecker) checkComputeInstance(instance *parser.Instance, report *HealthReport) {
	if instance.Attributes == nil {
		return
	}
	
	// Check for preemptible instances in production
	if preemptible, exists := instance.Attributes["scheduling"]; exists {
		if sched, ok := preemptible.(map[string]interface{}); ok {
			if sched["preemptible"] == true {
				report.Issues = append(report.Issues, HealthIssue{
					Type:     IssueTypeConfiguration,
					Severity: SeverityMedium,
					Message:  "Instance is configured as preemptible",
					Field:    "scheduling.preemptible",
				})
				report.Suggestions = append(report.Suggestions, 
					"Consider using standard instances for production workloads")
			}
		}
	}
}

func (c *GCPHealthChecker) checkStorageBucket(instance *parser.Instance, report *HealthReport) {
	if instance.Attributes == nil {
		return
	}
	
	// Check lifecycle rules
	if _, exists := instance.Attributes["lifecycle_rule"]; !exists {
		report.Suggestions = append(report.Suggestions, 
			"Consider adding lifecycle rules to manage storage costs")
	}
	
	// Check versioning
	if versioning, exists := instance.Attributes["versioning"]; exists {
		if ver, ok := versioning.(map[string]interface{}); ok {
			if ver["enabled"] != true {
				report.Issues = append(report.Issues, HealthIssue{
					Type:     IssueTypeConfiguration,
					Severity: SeverityLow,
					Message:  "Bucket versioning is not enabled",
					Field:    "versioning.enabled",
				})
			}
		}
	}
}

func (c *GCPHealthChecker) GetRequiredAttributes(resourceType string) []string {
	if attrs, exists := c.requiredAttributes[resourceType]; exists {
		return attrs
	}
	return []string{}
}

func (c *GCPHealthChecker) GetDeprecatedAttributes(resourceType string) []string {
	if attrs, exists := c.deprecatedAttributes[resourceType]; exists {
		return attrs
	}
	return []string{}
}

func (c *GCPHealthChecker) GetSecurityRules(resourceType string) []SecurityRule {
	if rules, exists := c.securityRules[resourceType]; exists {
		return rules
	}
	return []SecurityRule{}
}