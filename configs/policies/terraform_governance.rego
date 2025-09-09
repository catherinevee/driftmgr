package terraform.governance

import future.keywords.contains
import future.keywords.if
import future.keywords.in

# Default deny
default allow = false

# Allow if no violations
allow if {
    count(violations) == 0
}

# Production resource protection
violations contains msg if {
    input.action == "delete"
    input.tags.Environment == "production"
    msg := "Cannot delete production resources without approval"
}

violations contains msg if {
    input.action == "modify"
    input.tags.Environment == "production"
    not input.context.approved_by
    msg := "Production modifications require approval"
}

# Required tags enforcement
required_tags := ["Environment", "Owner", "CostCenter", "Project"]

violations contains msg if {
    some tag in required_tags
    not input.tags[tag]
    msg := sprintf("Missing required tag: %s", [tag])
}

# Cost controls
violations contains msg if {
    input.provider == "aws"
    input.resource.type == "aws_instance"
    instance_type := input.resource.attributes.instance_type
    expensive_types := ["p3.16xlarge", "x1e.32xlarge", "i3en.24xlarge"]
    instance_type in expensive_types
    not input.context.cost_approved
    msg := sprintf("Expensive instance type %s requires cost approval", [instance_type])
}

# Security policies
violations contains msg if {
    input.provider == "aws"
    input.resource.type == "aws_security_group_rule"
    input.resource.attributes.cidr_blocks[_] == "0.0.0.0/0"
    input.resource.attributes.from_port == 22
    msg := "SSH access from 0.0.0.0/0 is not allowed"
}

violations contains msg if {
    input.provider == "aws"
    input.resource.type == "aws_s3_bucket"
    not input.resource.attributes.server_side_encryption_configuration
    msg := "S3 buckets must have encryption enabled"
}

violations contains msg if {
    input.provider == "aws"
    input.resource.type == "aws_s3_bucket"
    input.resource.attributes.acl == "public-read"
    msg := "S3 buckets cannot be publicly readable"
}

# Compliance policies
violations contains msg if {
    input.tags.Compliance == "HIPAA"
    input.resource.type == "aws_db_instance"
    not input.resource.attributes.storage_encrypted
    msg := "HIPAA compliance requires database encryption"
}

violations contains msg if {
    input.tags.Compliance == "PCI-DSS"
    input.resource.type == "aws_instance"
    not input.resource.attributes.monitoring
    msg := "PCI-DSS compliance requires detailed monitoring"
}

# Region restrictions
allowed_regions := ["us-east-1", "us-west-2", "eu-west-1"]

violations contains msg if {
    not input.region in allowed_regions
    msg := sprintf("Region %s is not allowed. Use one of: %v", [input.region, allowed_regions])
}

# Backup requirements
violations contains msg if {
    input.resource.type in ["aws_db_instance", "aws_ebs_volume"]
    input.tags.Environment == "production"
    not input.resource.attributes.backup_retention_period
    msg := "Production databases and volumes must have backup configured"
}

# Network isolation
violations contains msg if {
    input.tags.Environment == "production"
    input.tags.NetworkIsolation != "true"
    input.resource.type in ["aws_instance", "aws_db_instance"]
    msg := "Production resources must have network isolation"
}

# Generate suggestions based on violations
suggestions[msg] if {
    "Missing required tag: Owner" in violations
    msg := "Run: terraform apply -var='tags={Owner=\"team-name\"}'"
}

suggestions[msg] if {
    some violation in violations
    contains(violation, "encryption")
    msg := "Enable encryption by adding 'encrypted = true' to your resource configuration"
}

# Risk scoring
risk_score := score if {
    score := sum([
        5 | violations[_]; contains(violations[_], "production")
        3 | violations[_]; contains(violations[_], "security")
        2 | violations[_]; contains(violations[_], "tag")
        1 | violations[_]
    ])
}

# Remediation recommendations
remediation[action] if {
    violations[_]
    contains(violations[_], "production")
    action := {
        "type": "approval_required",
        "approvers": ["platform-team", "security-team"],
        "sla": "4h"
    }
}

remediation[action] if {
    violations[_]
    contains(violations[_], "encryption")
    action := {
        "type": "automatic",
        "script": "enable_encryption.sh",
        "parameters": {"resource_id": input.resource.id}
    }
}