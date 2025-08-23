package drift

// PredefinedDriftPatterns returns a set of common drift patterns
func PredefinedDriftPatterns() []*DriftPattern {
	return []*DriftPattern{
		// Security-related patterns
		{
			ID:          "security_group_open",
			Name:        "Open Security Group",
			Description: "Security group with overly permissive rules",
			Confidence:  0.85,
			RiskLevel:   RiskLevelHigh,
			Weight:      0.3,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "equals", Value: "aws_security_group"},
			},
		},
		{
			ID:          "public_access",
			Name:        "Public Resource Access",
			Description: "Resource with public access enabled",
			Confidence:  0.9,
			RiskLevel:   RiskLevelCritical,
			Weight:      0.4,
			Conditions: []PatternCondition{
				{Field: "tags", Operator: "contains", Value: map[string]string{"public": "true"}},
			},
		},
		{
			ID:          "unencrypted_storage",
			Name:        "Unencrypted Storage",
			Description: "Storage resource without encryption enabled",
			Confidence:  0.8,
			RiskLevel:   RiskLevelHigh,
			Weight:      0.25,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "in", Value: []string{"aws_s3_bucket", "aws_ebs_volume", "azure_storage_account"}},
			},
		},

		// Cost optimization patterns
		{
			ID:          "unused_resource",
			Name:        "Unused Resource",
			Description: "Resource that appears to be unused",
			Confidence:  0.7,
			RiskLevel:   RiskLevelMedium,
			Weight:      0.2,
			Conditions: []PatternCondition{
				{Field: "tags", Operator: "contains", Value: map[string]string{"purpose": "test"}},
			},
		},
		{
			ID:          "oversized_instance",
			Name:        "Oversized Instance",
			Description: "Compute instance that may be oversized for its workload",
			Confidence:  0.6,
			RiskLevel:   RiskLevelMedium,
			Weight:      0.15,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "in", Value: []string{"aws_instance", "azure_virtual_machine", "google_compute_instance"}},
			},
		},
		{
			ID:          "idle_database",
			Name:        "Idle Database",
			Description: "Database instance with low activity",
			Confidence:  0.75,
			RiskLevel:   RiskLevelMedium,
			Weight:      0.2,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "in", Value: []string{"aws_rds_instance", "aws_dynamodb_table", "azure_sql_database"}},
			},
		},

		// Compliance patterns
		{
			ID:          "missing_tags",
			Name:        "Missing Required Tags",
			Description: "Resource missing required compliance tags",
			Confidence:  0.8,
			RiskLevel:   RiskLevelMedium,
			Weight:      0.25,
			Conditions: []PatternCondition{
				{Field: "tags", Operator: "missing", Value: []string{"environment", "owner", "cost-center"}},
			},
		},
		{
			ID:          "non_compliant_backup",
			Name:        "Non-Compliant Backup",
			Description: "Resource without proper backup configuration",
			Confidence:  0.7,
			RiskLevel:   RiskLevelHigh,
			Weight:      0.3,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "in", Value: []string{"aws_rds_instance", "aws_ebs_volume", "azure_virtual_machine"}},
			},
		},

		// Performance patterns
		{
			ID:          "single_az_deployment",
			Name:        "Single AZ Deployment",
			Description: "Critical resource deployed in single availability zone",
			Confidence:  0.8,
			RiskLevel:   RiskLevelHigh,
			Weight:      0.25,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "in", Value: []string{"aws_rds_instance", "aws_elasticache_cluster"}},
			},
		},
		{
			ID:          "no_auto_scaling",
			Name:        "No Auto Scaling",
			Description: "Application without auto-scaling configuration",
			Confidence:  0.6,
			RiskLevel:   RiskLevelMedium,
			Weight:      0.15,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "in", Value: []string{"aws_ecs_service", "aws_eks_node_group"}},
			},
		},

		// Availability patterns
		{
			ID:          "no_monitoring",
			Name:        "No Monitoring",
			Description: "Resource without proper monitoring configuration",
			Confidence:  0.7,
			RiskLevel:   RiskLevelMedium,
			Weight:      0.2,
			Conditions: []PatternCondition{
				{Field: "tags", Operator: "missing", Value: []string{"monitoring"}},
			},
		},
		{
			ID:          "outdated_ami",
			Name:        "Outdated AMI",
			Description: "Instance using outdated or unsupported AMI",
			Confidence:  0.8,
			RiskLevel:   RiskLevelHigh,
			Weight:      0.3,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "equals", Value: "aws_instance"},
			},
		},

		// Network patterns
		{
			ID:          "default_vpc",
			Name:        "Default VPC Usage",
			Description: "Resource deployed in default VPC",
			Confidence:  0.9,
			RiskLevel:   RiskLevelMedium,
			Weight:      0.2,
			Conditions: []PatternCondition{
				{Field: "tags", Operator: "contains", Value: map[string]string{"vpc": "default"}},
			},
		},
		{
			ID:          "no_waf",
			Name:        "No WAF Protection",
			Description: "Web application without WAF protection",
			Confidence:  0.7,
			RiskLevel:   RiskLevelHigh,
			Weight:      0.25,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "in", Value: []string{"aws_alb", "aws_nlb", "azure_application_gateway"}},
			},
		},

		// Data patterns
		{
			ID:          "no_versioning",
			Name:        "No Versioning",
			Description: "S3 bucket without versioning enabled",
			Confidence:  0.8,
			RiskLevel:   RiskLevelMedium,
			Weight:      0.2,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "equals", Value: "aws_s3_bucket"},
			},
		},
		{
			ID:          "no_lifecycle",
			Name:        "No Lifecycle Policy",
			Description: "Storage without lifecycle management",
			Confidence:  0.6,
			RiskLevel:   RiskLevelLow,
			Weight:      0.1,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "in", Value: []string{"aws_s3_bucket", "aws_glacier_vault"}},
			},
		},
	}
}

// RegisterDefaultPatterns registers all predefined patterns with a predictor
func RegisterDefaultPatterns(predictor *DriftPredictor) {
	patterns := PredefinedDriftPatterns()
	for _, pattern := range patterns {
		predictor.RegisterPattern(pattern)
	}
}

// CreateCustomPattern creates a custom drift pattern
func CreateCustomPattern(id, name, description string, confidence float64, riskLevel RiskLevel, conditions []PatternCondition) *DriftPattern {
	return &DriftPattern{
		ID:          id,
		Name:        name,
		Description: description,
		Confidence:  confidence,
		RiskLevel:   riskLevel,
		Weight:      0.2, // Default weight
		Conditions:  conditions,
	}
}

// PatternTemplates provides templates for common pattern types
func PatternTemplates() map[string]*DriftPattern {
	return map[string]*DriftPattern{
		"security_violation": {
			ID:          "security_violation",
			Name:        "Security Violation",
			Description: "Template for security-related drift patterns",
			Confidence:  0.8,
			RiskLevel:   RiskLevelHigh,
			Weight:      0.3,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "equals", Value: "template"},
			},
		},
		"cost_optimization": {
			ID:          "cost_optimization",
			Name:        "Cost Optimization",
			Description: "Template for cost optimization patterns",
			Confidence:  0.7,
			RiskLevel:   RiskLevelMedium,
			Weight:      0.2,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "equals", Value: "template"},
			},
		},
		"compliance_check": {
			ID:          "compliance_check",
			Name:        "Compliance Check",
			Description: "Template for compliance-related patterns",
			Confidence:  0.8,
			RiskLevel:   RiskLevelMedium,
			Weight:      0.25,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "equals", Value: "template"},
			},
		},
		"performance_issue": {
			ID:          "performance_issue",
			Name:        "Performance Issue",
			Description: "Template for performance-related patterns",
			Confidence:  0.7,
			RiskLevel:   RiskLevelMedium,
			Weight:      0.2,
			Conditions: []PatternCondition{
				{Field: "type", Operator: "equals", Value: "template"},
			},
		},
	}
}
