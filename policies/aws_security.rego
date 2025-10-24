package aws.security

# Default deny
default allow = false

# Allow if no violations are found
allow {
    count(violations) == 0
}

# Check for production resource deletion
violations["production_deletion"] {
    input.action == "delete"
    input.tags.Environment == "production"
    violation := {
        "rule": "no_delete_production",
        "message": "Cannot delete production resources",
        "severity": "high",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Remove production tag or use staging environment"
    }
}

# Check for required tags
violations["missing_owner_tag"] {
    not input.tags.Owner
    violation := {
        "rule": "required_tags",
        "message": "Missing required tag: Owner",
        "severity": "medium",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Add Owner tag to the resource"
    }
}

violations["missing_environment_tag"] {
    not input.tags.Environment
    violation := {
        "rule": "required_tags",
        "message": "Missing required tag: Environment",
        "severity": "medium",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Add Environment tag to the resource"
    }
}

violations["missing_cost_center_tag"] {
    not input.tags.CostCenter
    violation := {
        "rule": "required_tags",
        "message": "Missing required tag: CostCenter",
        "severity": "medium",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Add CostCenter tag to the resource"
    }
}

# Check for encryption requirements
violations["unencrypted_s3_bucket"] {
    input.resource.type == "s3_bucket"
    not input.resource.encryption.enabled
    violation := {
        "rule": "encryption_required",
        "message": "S3 bucket must be encrypted",
        "severity": "high",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Enable encryption on the S3 bucket"
    }
}

violations["unencrypted_rds_instance"] {
    input.resource.type == "rds_instance"
    not input.resource.encryption.enabled
    violation := {
        "rule": "encryption_required",
        "message": "RDS instance must be encrypted",
        "severity": "high",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Enable encryption on the RDS instance"
    }
}

# Check for public access
violations["public_s3_bucket"] {
    input.resource.type == "s3_bucket"
    input.resource.public_access
    violation := {
        "rule": "no_public_access",
        "message": "S3 bucket should not have public access",
        "severity": "critical",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Remove public access from the S3 bucket"
    }
}

# Check for security group rules
violations["open_security_group"] {
    input.resource.type == "security_group"
    input.resource.rules[_].cidr_blocks[_] == "0.0.0.0/0"
    violation := {
        "rule": "restrictive_security_groups",
        "message": "Security group allows access from anywhere",
        "severity": "high",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Restrict security group to specific IP ranges"
    }
}
