package workflow

import "time"

// PredefinedWorkflows returns a set of common workflow templates
func PredefinedWorkflows() []*Workflow {
	return []*Workflow{
		// Security remediation workflow
		{
			ID:          "security_remediation",
			Name:        "Security Remediation",
			Description: "Automated workflow for security-related drift remediation",
			Timeout:     30 * time.Minute,
			Retries:     3,
			Steps: []WorkflowStep{
				{
					ID:       "backup_current_state",
					Name:     "Backup Current State",
					Action:   "backup_resource",
					Resource: "security_group",
					Required: true,
					Timeout:  5 * time.Minute,
					Retries:  2,
				},
				{
					ID:       "validate_changes",
					Name:     "Validate Changes",
					Action:   "validate_configuration",
					Resource: "security_rules",
					Required: true,
					Timeout:  2 * time.Minute,
					Retries:  1,
				},
				{
					ID:       "apply_security_fixes",
					Name:     "Apply Security Fixes",
					Action:   "terraform_apply",
					Resource: "security_group",
					Required: true,
					Timeout:  10 * time.Minute,
					Retries:  3,
				},
				{
					ID:       "verify_security",
					Name:     "Verify Security",
					Action:   "health_check",
					Resource: "security_scan",
					Required: true,
					Timeout:  5 * time.Minute,
					Retries:  2,
				},
				{
					ID:       "notify_completion",
					Name:     "Notify Completion",
					Action:   "notify",
					Resource: "security_team",
					Required: false,
					Timeout:  1 * time.Minute,
					Retries:  1,
				},
			},
			Rollback: &RollbackPlan{
				Auto: true,
				Steps: []WorkflowStep{
					{
						ID:       "restore_backup",
						Name:     "Restore Backup",
						Action:   "restore_resource",
						Resource: "security_group",
						Required: true,
						Timeout:  5 * time.Minute,
						Retries:  2,
					},
				},
			},
		},

		// Cost optimization workflow
		{
			ID:          "cost_optimization",
			Name:        "Cost Optimization",
			Description: "Automated workflow for cost optimization and resource cleanup",
			Timeout:     45 * time.Minute,
			Retries:     2,
			Steps: []WorkflowStep{
				{
					ID:       "analyze_usage",
					Name:     "Analyze Resource Usage",
					Action:   "validate_configuration",
					Resource: "cost_analysis",
					Required: true,
					Timeout:  10 * time.Minute,
					Retries:  1,
				},
				{
					ID:       "backup_resources",
					Name:     "Backup Resources",
					Action:   "backup_resource",
					Resource: "target_resources",
					Required: true,
					Timeout:  15 * time.Minute,
					Retries:  2,
				},
				{
					ID:       "downsize_instances",
					Name:     "Downsize Instances",
					Action:   "terraform_apply",
					Resource: "instance_sizing",
					Required: false,
					Timeout:  20 * time.Minute,
					Retries:  3,
				},
				{
					ID:       "cleanup_unused",
					Name:     "Cleanup Unused Resources",
					Action:   "terraform_destroy",
					Resource: "unused_resources",
					Required: false,
					Timeout:  10 * time.Minute,
					Retries:  2,
				},
				{
					ID:       "verify_optimization",
					Name:     "Verify Optimization",
					Action:   "health_check",
					Resource: "cost_verification",
					Required: true,
					Timeout:  5 * time.Minute,
					Retries:  2,
				},
			},
			Rollback: &RollbackPlan{
				Auto: true,
				Steps: []WorkflowStep{
					{
						ID:       "restore_resources",
						Name:     "Restore Resources",
						Action:   "restore_resource",
						Resource: "backup_resources",
						Required: true,
						Timeout:  15 * time.Minute,
						Retries:  2,
					},
				},
			},
		},

		// Compliance remediation workflow
		{
			ID:          "compliance_remediation",
			Name:        "Compliance Remediation",
			Description: "Automated workflow for compliance-related drift remediation",
			Timeout:     60 * time.Minute,
			Retries:     3,
			Steps: []WorkflowStep{
				{
					ID:       "audit_compliance",
					Name:     "Audit Compliance Status",
					Action:   "validate_configuration",
					Resource: "compliance_audit",
					Required: true,
					Timeout:  15 * time.Minute,
					Retries:  1,
				},
				{
					ID:       "backup_current",
					Name:     "Backup Current Configuration",
					Action:   "backup_resource",
					Resource: "compliance_resources",
					Required: true,
					Timeout:  10 * time.Minute,
					Retries:  2,
				},
				{
					ID:       "apply_compliance_fixes",
					Name:     "Apply Compliance Fixes",
					Action:   "terraform_apply",
					Resource: "compliance_configuration",
					Required: true,
					Timeout:  30 * time.Minute,
					Retries:  3,
				},
				{
					ID:       "verify_compliance",
					Name:     "Verify Compliance",
					Action:   "health_check",
					Resource: "compliance_verification",
					Required: true,
					Timeout:  10 * time.Minute,
					Retries:  2,
				},
				{
					ID:       "generate_report",
					Name:     "Generate Compliance Report",
					Action:   "notify",
					Resource: "compliance_report",
					Required: true,
					Timeout:  5 * time.Minute,
					Retries:  1,
				},
			},
			Rollback: &RollbackPlan{
				Auto: true,
				Steps: []WorkflowStep{
					{
						ID:       "restore_compliance",
						Name:     "Restore Compliance Configuration",
						Action:   "restore_resource",
						Resource: "compliance_backup",
						Required: true,
						Timeout:  15 * time.Minute,
						Retries:  2,
					},
				},
			},
		},

		// Performance optimization workflow
		{
			ID:          "performance_optimization",
			Name:        "Performance Optimization",
			Description: "Automated workflow for performance-related drift remediation",
			Timeout:     40 * time.Minute,
			Retries:     2,
			Steps: []WorkflowStep{
				{
					ID:       "analyze_performance",
					Name:     "Analyze Performance",
					Action:   "validate_configuration",
					Resource: "performance_metrics",
					Required: true,
					Timeout:  10 * time.Minute,
					Retries:  1,
				},
				{
					ID:       "backup_configuration",
					Name:     "Backup Configuration",
					Action:   "backup_resource",
					Resource: "performance_config",
					Required: true,
					Timeout:  5 * time.Minute,
					Retries:  2,
				},
				{
					ID:       "optimize_resources",
					Name:     "Optimize Resources",
					Action:   "terraform_apply",
					Resource: "performance_optimization",
					Required: true,
					Timeout:  20 * time.Minute,
					Retries:  3,
				},
				{
					ID:       "verify_performance",
					Name:     "Verify Performance",
					Action:   "health_check",
					Resource: "performance_verification",
					Required: true,
					Timeout:  10 * time.Minute,
					Retries:  2,
				},
			},
			Rollback: &RollbackPlan{
				Auto: true,
				Steps: []WorkflowStep{
					{
						ID:       "restore_performance",
						Name:     "Restore Performance Configuration",
						Action:   "restore_resource",
						Resource: "performance_backup",
						Required: true,
						Timeout:  10 * time.Minute,
						Retries:  2,
					},
				},
			},
		},

		// Emergency rollback workflow
		{
			ID:          "emergency_rollback",
			Name:        "Emergency Rollback",
			Description: "Emergency workflow for quick rollback of failed changes",
			Timeout:     15 * time.Minute,
			Retries:     1,
			Steps: []WorkflowStep{
				{
					ID:       "assess_damage",
					Name:     "Assess Damage",
					Action:   "validate_configuration",
					Resource: "system_health",
					Required: true,
					Timeout:  2 * time.Minute,
					Retries:  1,
				},
				{
					ID:       "emergency_restore",
					Name:     "Emergency Restore",
					Action:   "restore_resource",
					Resource: "latest_backup",
					Required: true,
					Timeout:  10 * time.Minute,
					Retries:  1,
				},
				{
					ID:       "verify_restore",
					Name:     "Verify Restore",
					Action:   "health_check",
					Resource: "system_verification",
					Required: true,
					Timeout:  3 * time.Minute,
					Retries:  1,
				},
				{
					ID:       "notify_emergency",
					Name:     "Notify Emergency",
					Action:   "notify",
					Resource: "emergency_team",
					Required: true,
					Timeout:  1 * time.Minute,
					Retries:  1,
				},
			},
		},
	}
}

// RegisterDefaultWorkflows registers all predefined workflows with the engine
func RegisterDefaultWorkflows(engine *WorkflowEngine) error {
	workflows := PredefinedWorkflows()
	for _, workflow := range workflows {
		if err := engine.RegisterWorkflow(workflow); err != nil {
			return err
		}
	}
	return nil
}

// CreateCustomWorkflow creates a custom workflow from a template
func CreateCustomWorkflow(id, name, description string, steps []WorkflowStep, rollback *RollbackPlan) *Workflow {
	return &Workflow{
		ID:          id,
		Name:        name,
		Description: description,
		Steps:       steps,
		Rollback:    rollback,
		Timeout:     30 * time.Minute,
		Retries:     3,
	}
}

// WorkflowTemplates provides templates for common workflow types
func WorkflowTemplates() map[string]*Workflow {
	return map[string]*Workflow{
		"simple_remediation": {
			ID:          "simple_remediation",
			Name:        "Simple Remediation",
			Description: "Template for simple remediation workflows",
			Timeout:     20 * time.Minute,
			Retries:     2,
			Steps: []WorkflowStep{
				{
					ID:       "backup",
					Name:     "Backup",
					Action:   "backup_resource",
					Resource: "template",
					Required: true,
					Timeout:  5 * time.Minute,
					Retries:  1,
				},
				{
					ID:       "apply",
					Name:     "Apply Changes",
					Action:   "terraform_apply",
					Resource: "template",
					Required: true,
					Timeout:  10 * time.Minute,
					Retries:  2,
				},
				{
					ID:       "verify",
					Name:     "Verify",
					Action:   "health_check",
					Resource: "template",
					Required: true,
					Timeout:  5 * time.Minute,
					Retries:  1,
				},
			},
		},
		"complex_remediation": {
			ID:          "complex_remediation",
			Name:        "Complex Remediation",
			Description: "Template for complex remediation workflows",
			Timeout:     60 * time.Minute,
			Retries:     3,
			Steps: []WorkflowStep{
				{
					ID:       "analysis",
					Name:     "Analysis",
					Action:   "validate_configuration",
					Resource: "template",
					Required: true,
					Timeout:  10 * time.Minute,
					Retries:  1,
				},
				{
					ID:       "backup",
					Name:     "Backup",
					Action:   "backup_resource",
					Resource: "template",
					Required: true,
					Timeout:  10 * time.Minute,
					Retries:  2,
				},
				{
					ID:       "apply_changes",
					Name:     "Apply Changes",
					Action:   "terraform_apply",
					Resource: "template",
					Required: true,
					Timeout:  30 * time.Minute,
					Retries:  3,
				},
				{
					ID:       "verify_changes",
					Name:     "Verify Changes",
					Action:   "health_check",
					Resource: "template",
					Required: true,
					Timeout:  10 * time.Minute,
					Retries:  2,
				},
				{
					ID:       "notify",
					Name:     "Notify",
					Action:   "notify",
					Resource: "template",
					Required: false,
					Timeout:  1 * time.Minute,
					Retries:  1,
				},
			},
		},
	}
}
