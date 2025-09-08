package checkers

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// AWSChecker performs health checks for AWS resources
type AWSChecker struct {
	checkType string
}

// NewAWSChecker creates a new AWS health checker
func NewAWSChecker() *AWSChecker {
	return &AWSChecker{
		checkType: "aws",
	}
}

// Check performs health checks on AWS resources
func (ac *AWSChecker) Check(ctx context.Context, resource *models.Resource) (*HealthCheck, error) {
	check := &HealthCheck{
		ID:          fmt.Sprintf("aws-%s", resource.ID),
		Name:        "AWS Resource Health",
		Type:        ac.checkType,
		ResourceID:  resource.ID,
		LastChecked: time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	start := time.Now()
	defer func() {
		check.Duration = time.Since(start)
	}()

	// Perform resource-specific health checks
	switch {
	case ac.isEC2Instance(resource):
		return ac.checkEC2Instance(ctx, resource, check)
	case ac.isS3Bucket(resource):
		return ac.checkS3Bucket(ctx, resource, check)
	case ac.isRDSInstance(resource):
		return ac.checkRDSInstance(ctx, resource, check)
	case ac.isLambdaFunction(resource):
		return ac.checkLambdaFunction(ctx, resource, check)
	case ac.isLoadBalancer(resource):
		return ac.checkLoadBalancer(ctx, resource, check)
	default:
		return ac.checkGenericAWSResource(ctx, resource, check)
	}
}

// GetType returns the checker type
func (ac *AWSChecker) GetType() string {
	return ac.checkType
}

// GetDescription returns the checker description
func (ac *AWSChecker) GetDescription() string {
	return "AWS resource health checker"
}

// isEC2Instance checks if the resource is an EC2 instance
func (ac *AWSChecker) isEC2Instance(resource *models.Resource) bool {
	return resource.Type == "aws_instance" || resource.Type == "aws_ec2_instance"
}

// isS3Bucket checks if the resource is an S3 bucket
func (ac *AWSChecker) isS3Bucket(resource *models.Resource) bool {
	return resource.Type == "aws_s3_bucket"
}

// isRDSInstance checks if the resource is an RDS instance
func (ac *AWSChecker) isRDSInstance(resource *models.Resource) bool {
	return resource.Type == "aws_db_instance" || resource.Type == "aws_rds_cluster"
}

// isLambdaFunction checks if the resource is a Lambda function
func (ac *AWSChecker) isLambdaFunction(resource *models.Resource) bool {
	return resource.Type == "aws_lambda_function"
}

// isLoadBalancer checks if the resource is a load balancer
func (ac *AWSChecker) isLoadBalancer(resource *models.Resource) bool {
	return resource.Type == "aws_lb" || resource.Type == "aws_elb" || resource.Type == "aws_alb"
}

// checkEC2Instance performs health checks on EC2 instances
func (ac *AWSChecker) checkEC2Instance(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	// Check instance state
	if state, ok := resource.Attributes["state"].(string); ok {
		switch state {
		case "running":
			check.Status = HealthStatusHealthy
			check.Message = "Instance is running normally"
		case "stopped":
			check.Status = HealthStatusWarning
			check.Message = "Instance is stopped"
		case "stopping":
			check.Status = HealthStatusWarning
			check.Message = "Instance is stopping"
		case "pending":
			check.Status = HealthStatusWarning
			check.Message = "Instance is pending"
		case "terminated":
			check.Status = HealthStatusCritical
			check.Message = "Instance is terminated"
		default:
			check.Status = HealthStatusUnknown
			check.Message = fmt.Sprintf("Unknown instance state: %s", state)
		}
	} else {
		check.Status = HealthStatusUnknown
		check.Message = "Instance state not available"
	}

	// Check instance type and performance
	if instanceType, ok := resource.Attributes["instance_type"].(string); ok {
		check.Metadata["instance_type"] = instanceType

		// Check if instance type is deprecated or inefficient
		if ac.isDeprecatedInstanceType(instanceType) {
			check.Status = HealthStatusWarning
			check.Message += " - Instance type is deprecated"
		}
	}

	// Check security groups
	if securityGroups, ok := resource.Attributes["security_groups"].([]interface{}); ok {
		check.Metadata["security_groups"] = len(securityGroups)
		if len(securityGroups) == 0 {
			check.Status = HealthStatusCritical
			check.Message += " - No security groups attached"
		}
	}

	// Check tags
	if tags, ok := resource.Attributes["tags"].(map[string]interface{}); ok {
		check.Metadata["tags"] = len(tags)
		if len(tags) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No tags applied"
		}
	}

	return check, nil
}

// checkS3Bucket performs health checks on S3 buckets
func (ac *AWSChecker) checkS3Bucket(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "S3 bucket is accessible"

	// Check bucket versioning
	if versioning, ok := resource.Attributes["versioning"].([]interface{}); ok {
		if len(versioning) > 0 {
			check.Metadata["versioning_enabled"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - Versioning not enabled"
		}
	}

	// Check bucket encryption
	if serverSideEncryption, ok := resource.Attributes["server_side_encryption_configuration"].([]interface{}); ok {
		if len(serverSideEncryption) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No encryption configured"
		} else {
			check.Metadata["encryption_enabled"] = true
		}
	}

	// Check public access block
	if publicAccessBlock, ok := resource.Attributes["public_access_block"].([]interface{}); ok {
		if len(publicAccessBlock) == 0 {
			check.Status = HealthStatusCritical
			check.Message += " - Public access not blocked"
		} else {
			check.Metadata["public_access_blocked"] = true
		}
	}

	// Check lifecycle configuration
	if lifecycle, ok := resource.Attributes["lifecycle_rule"].([]interface{}); ok {
		check.Metadata["lifecycle_rules"] = len(lifecycle)
		if len(lifecycle) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No lifecycle rules configured"
		}
	}

	return check, nil
}

// checkRDSInstance performs health checks on RDS instances
func (ac *AWSChecker) checkRDSInstance(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	// Check instance status
	if status, ok := resource.Attributes["status"].(string); ok {
		switch status {
		case "available":
			check.Status = HealthStatusHealthy
			check.Message = "RDS instance is available"
		case "backing-up":
			check.Status = HealthStatusHealthy
			check.Message = "RDS instance is backing up"
		case "creating":
			check.Status = HealthStatusWarning
			check.Message = "RDS instance is being created"
		case "deleting":
			check.Status = HealthStatusCritical
			check.Message = "RDS instance is being deleted"
		case "failed":
			check.Status = HealthStatusCritical
			check.Message = "RDS instance has failed"
		default:
			check.Status = HealthStatusUnknown
			check.Message = fmt.Sprintf("Unknown RDS status: %s", status)
		}
	} else {
		check.Status = HealthStatusUnknown
		check.Message = "RDS status not available"
	}

	// Check encryption
	if storageEncrypted, ok := resource.Attributes["storage_encrypted"].(bool); ok {
		if !storageEncrypted {
			check.Status = HealthStatusCritical
			check.Message += " - Storage not encrypted"
		} else {
			check.Metadata["encryption_enabled"] = true
		}
	}

	// Check backup retention
	if backupRetentionPeriod, ok := resource.Attributes["backup_retention_period"].(int); ok {
		check.Metadata["backup_retention_days"] = backupRetentionPeriod
		if backupRetentionPeriod == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No backup retention configured"
		}
	}

	// Check multi-AZ deployment
	if multiAz, ok := resource.Attributes["multi_az"].(bool); ok {
		check.Metadata["multi_az"] = multiAz
		if !multiAz {
			check.Status = HealthStatusWarning
			check.Message += " - Not deployed in multiple AZs"
		}
	}

	return check, nil
}

// checkLambdaFunction performs health checks on Lambda functions
func (ac *AWSChecker) checkLambdaFunction(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Lambda function is configured"

	// Check runtime
	if runtime, ok := resource.Attributes["runtime"].(string); ok {
		check.Metadata["runtime"] = runtime
		if ac.isDeprecatedRuntime(runtime) {
			check.Status = HealthStatusWarning
			check.Message += " - Runtime is deprecated"
		}
	}

	// Check timeout
	if timeout, ok := resource.Attributes["timeout"].(int); ok {
		check.Metadata["timeout_seconds"] = timeout
		if timeout > 300 { // 5 minutes
			check.Status = HealthStatusWarning
			check.Message += " - Timeout is very high"
		}
	}

	// Check memory
	if memory, ok := resource.Attributes["memory_size"].(int); ok {
		check.Metadata["memory_mb"] = memory
		if memory < 128 {
			check.Status = HealthStatusWarning
			check.Message += " - Memory allocation is very low"
		}
	}

	// Check VPC configuration
	if vpcConfig, ok := resource.Attributes["vpc_config"].([]interface{}); ok {
		if len(vpcConfig) > 0 {
			check.Metadata["vpc_configured"] = true
		} else {
			check.Status = HealthStatusWarning
			check.Message += " - No VPC configuration"
		}
	}

	return check, nil
}

// checkLoadBalancer performs health checks on load balancers
func (ac *AWSChecker) checkLoadBalancer(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "Load balancer is configured"

	// Check load balancer type
	if lbType, ok := resource.Attributes["load_balancer_type"].(string); ok {
		check.Metadata["load_balancer_type"] = lbType
	}

	// Check scheme
	if scheme, ok := resource.Attributes["scheme"].(string); ok {
		check.Metadata["scheme"] = scheme
		if scheme == "internal" {
			check.Status = HealthStatusWarning
			check.Message += " - Internal load balancer"
		}
	}

	// Check security groups
	if securityGroups, ok := resource.Attributes["security_groups"].([]interface{}); ok {
		check.Metadata["security_groups"] = len(securityGroups)
		if len(securityGroups) == 0 {
			check.Status = HealthStatusCritical
			check.Message += " - No security groups attached"
		}
	}

	// Check SSL certificate
	if certificateArn, ok := resource.Attributes["certificate_arn"].(string); ok {
		if certificateArn == "" {
			check.Status = HealthStatusWarning
			check.Message += " - No SSL certificate configured"
		} else {
			check.Metadata["ssl_certificate"] = true
		}
	}

	return check, nil
}

// checkGenericAWSResource performs generic health checks for AWS resources
func (ac *AWSChecker) checkGenericAWSResource(ctx context.Context, resource *models.Resource, check *HealthCheck) (*HealthCheck, error) {
	check.Status = HealthStatusHealthy
	check.Message = "AWS resource is configured"

	// Check if resource has tags
	if tags, ok := resource.Attributes["tags"].(map[string]interface{}); ok {
		check.Metadata["tags"] = len(tags)
		if len(tags) == 0 {
			check.Status = HealthStatusWarning
			check.Message += " - No tags applied"
		}
	}

	// Check if resource is in a VPC
	if vpcId, ok := resource.Attributes["vpc_id"].(string); ok {
		if vpcId != "" {
			check.Metadata["vpc_id"] = vpcId
		}
	}

	return check, nil
}

// isDeprecatedInstanceType checks if an instance type is deprecated
func (ac *AWSChecker) isDeprecatedInstanceType(instanceType string) bool {
	deprecatedTypes := []string{
		"t1.micro", "m1.small", "m1.medium", "m1.large", "m1.xlarge",
		"m2.xlarge", "m2.2xlarge", "m2.4xlarge", "c1.medium", "c1.xlarge",
		"cc1.4xlarge", "cc2.8xlarge", "cg1.4xlarge", "cr1.8xlarge",
		"hi1.4xlarge", "hs1.8xlarge", "m3.medium", "m3.large", "m3.xlarge",
		"m3.2xlarge", "c3.large", "c3.xlarge", "c3.2xlarge", "c3.4xlarge",
		"c3.8xlarge", "g2.2xlarge", "g2.8xlarge", "r3.large", "r3.xlarge",
		"r3.2xlarge", "r3.4xlarge", "r3.8xlarge", "i2.xlarge", "i2.2xlarge",
		"i2.4xlarge", "i2.8xlarge", "d2.xlarge", "d2.2xlarge", "d2.4xlarge",
		"d2.8xlarge",
	}

	for _, deprecated := range deprecatedTypes {
		if instanceType == deprecated {
			return true
		}
	}
	return false
}

// isDeprecatedRuntime checks if a Lambda runtime is deprecated
func (ac *AWSChecker) isDeprecatedRuntime(runtime string) bool {
	deprecatedRuntimes := []string{
		"nodejs10.x", "nodejs8.10", "nodejs6.10", "nodejs4.3", "nodejs4.3-edge",
		"python2.7", "python3.6", "ruby2.5", "dotnetcore2.0", "dotnetcore2.1",
		"java8", "go1.x",
	}

	for _, deprecated := range deprecatedRuntimes {
		if runtime == deprecated {
			return true
		}
	}
	return false
}
