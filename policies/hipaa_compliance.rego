package hipaa.compliance

# Default deny
default allow = false

# Allow if no violations are found
allow {
    count(violations) == 0
}

# HIPAA Administrative Safeguards - Security Management Process
violations["security_management_process"] {
    input.action == "create"
    input.resource.type == "iam_policy"
    not input.resource.security_management_process
    violation := {
        "rule": "164.308(a)(1)",
        "message": "Security Management Process not implemented",
        "severity": "high",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Implement security management process for IAM policies"
    }
}

# HIPAA Technical Safeguards - Access Control
violations["access_control"] {
    input.action == "create"
    input.resource.type == "security_group"
    input.resource.rules[_].cidr_blocks[_] == "0.0.0.0/0"
    violation := {
        "rule": "164.312(a)(1)",
        "message": "Access control not properly implemented",
        "severity": "critical",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Implement proper access controls and restrict network access"
    }
}

# HIPAA Technical Safeguards - Encryption
violations["encryption_required_s3"] {
    input.action == "create"
    input.resource.type == "s3_bucket"
    not input.resource.encryption.enabled
    violation := {
        "rule": "164.312(a)(2)(iv)",
        "message": "Encryption required for S3 bucket PHI data storage",
        "severity": "critical",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Enable encryption for S3 bucket"
    }
}

violations["encryption_required_rds"] {
    input.action == "create"
    input.resource.type == "rds_instance"
    not input.resource.encryption.enabled
    violation := {
        "rule": "164.312(a)(2)(iv)",
        "message": "Encryption required for RDS instance PHI data storage",
        "severity": "critical",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Enable encryption for RDS instance"
    }
}

violations["encryption_required_ebs"] {
    input.action == "create"
    input.resource.type == "ebs_volume"
    not input.resource.encryption.enabled
    violation := {
        "rule": "164.312(a)(2)(iv)",
        "message": "Encryption required for EBS volume PHI data storage",
        "severity": "critical",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Enable encryption for EBS volume"
    }
}

# HIPAA Technical Safeguards - Audit Controls
violations["audit_controls"] {
    input.action == "create"
    input.resource.type == "cloudtrail"
    not input.resource.logging_enabled
    violation := {
        "rule": "164.312(b)",
        "message": "Audit controls not properly configured",
        "severity": "high",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Enable comprehensive audit logging"
    }
}

# HIPAA Physical Safeguards - Workstation Use
violations["workstation_security"] {
    input.action == "create"
    input.resource.type == "ec2_instance"
    count([sg | sg := input.resource.security_groups[_]; sg.name == "hipaa-secure"]) == 0
    violation := {
        "rule": "164.310(c)",
        "message": "Workstation security not properly configured",
        "severity": "medium",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Apply HIPAA-compliant security groups to workstations"
    }
}

# HIPAA Administrative Safeguards - Information Access Management
violations["information_access_management"] {
    input.action == "create"
    input.resource.type == "iam_user"
    not input.resource.mfa_enabled
    violation := {
        "rule": "164.308(a)(4)",
        "message": "Multi-factor authentication required for user access",
        "severity": "high",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Enable MFA for all user accounts"
    }
}

# HIPAA Technical Safeguards - Transmission Security
violations["transmission_security"] {
    input.action == "create"
    input.resource.type == "load_balancer"
    not input.resource.ssl_enabled
    violation := {
        "rule": "164.312(e)(1)",
        "message": "Transmission security not properly implemented",
        "severity": "high",
        "resource": sprintf("%v", [input.resource]),
        "remediation": "Enable SSL/TLS for all data transmission"
    }
}
