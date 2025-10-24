package simulation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/catherinevee/driftmgr/internal/state"
)

// AWSSimulator simulates drift in AWS resources
type AWSSimulator struct {
	ec2Client *ec2.Client
	s3Client  *s3.Client
	iamClient *iam.Client
	region    string
}

// NewAWSSimulator creates a new AWS drift simulator
func NewAWSSimulator() *AWSSimulator {
	return &AWSSimulator{
		region: "us-east-1", // Default region
	}
}

// Initialize sets up AWS clients
func (s *AWSSimulator) Initialize(ctx context.Context) error {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(s.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	s.ec2Client = ec2.NewFromConfig(cfg)
	s.s3Client = s3.NewFromConfig(cfg)
	s.iamClient = iam.NewFromConfig(cfg)

	return nil
}

// SimulateDrift creates drift in AWS resources
func (s *AWSSimulator) SimulateDrift(ctx context.Context, driftType DriftType, resourceID string, state *state.TerraformState) (*SimulationResult, error) {
	// Initialize clients if not already done
	if s.ec2Client == nil {
		if err := s.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	// Find the resource in state
	resource := s.findResource(resourceID, state)
	if resource == nil {
		return nil, fmt.Errorf("resource %s not found in state", resourceID)
	}

	// Execute drift based on type
	switch driftType {
	case DriftTypeTagChange:
		return s.simulateTagDrift(ctx, resource)
	case DriftTypeRuleAddition:
		return s.simulateSecurityGroupRuleDrift(ctx, resource)
	case DriftTypeResourceCreation:
		return s.simulateResourceCreation(ctx, resource, state)
	case DriftTypeAttributeChange:
		return s.simulateAttributeChange(ctx, resource)
	case DriftTypeResourceDeletion:
		return s.simulateResourceDeletion(ctx, resource)
	default:
		return nil, fmt.Errorf("drift type %s not implemented", driftType)
	}
}

// simulateTagDrift adds or modifies tags on a resource
func (s *AWSSimulator) simulateTagDrift(ctx context.Context, resource *state.Resource) (*SimulationResult, error) {
	result := &SimulationResult{
		Provider:     "aws",
		ResourceType: resource.Type,
		ResourceID:   resource.ID,
		DriftType:    DriftTypeTagChange,
		Changes:      make(map[string]interface{}),
		CostEstimate: "$0.00 (tags are free)",
	}

	// Generate drift tag
	driftTag := types.Tag{
		Key:   aws.String("DriftSimulation"),
		Value: aws.String(fmt.Sprintf("Created-%s", time.Now().Format("2006-01-02-15:04:05"))),
	}

	switch resource.Type {
	case "aws_instance":
		// Add tag to EC2 instance
		instanceID := s.extractInstanceID(resource)
		if instanceID == "" {
			return nil, fmt.Errorf("could not extract instance ID")
		}

		_, err := s.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
			Resources: []string{instanceID},
			Tags:      []types.Tag{driftTag},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add tag to instance: %w", err)
		}

		result.Changes["added_tag"] = map[string]string{
			*driftTag.Key: *driftTag.Value,
		}
		result.RollbackData = &RollbackData{
			Provider:     "aws",
			ResourceType: "aws_instance",
			ResourceID:   instanceID,
			Action:       "remove_tag",
			OriginalData: map[string]interface{}{
				"tag_key": *driftTag.Key,
			},
			Timestamp: time.Now(),
		}
		result.Success = true

	case "aws_s3_bucket":
		// Add tag to S3 bucket
		bucketName := s.extractBucketName(resource)
		if bucketName == "" {
			return nil, fmt.Errorf("could not extract bucket name")
		}

		// Get existing tags
		taggingOutput, err := s.s3Client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
			Bucket: aws.String(bucketName),
		})

		var existingTags []s3types.Tag
		if taggingOutput != nil {
			existingTags = taggingOutput.TagSet
		}

		// Add new drift tag
		newTags := append(existingTags, s3types.Tag{
			Key:   aws.String("DriftSimulation"),
			Value: aws.String(fmt.Sprintf("Created-%s", time.Now().Format("2006-01-02-15:04:05"))),
		})

		_, err = s.s3Client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
			Bucket: aws.String(bucketName),
			Tagging: &s3types.Tagging{
				TagSet: newTags,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add tag to S3 bucket: %w", err)
		}

		result.Changes["added_tag"] = map[string]string{
			"DriftSimulation": fmt.Sprintf("Created-%s", time.Now().Format("2006-01-02-15:04:05")),
		}
		result.RollbackData = &RollbackData{
			Provider:     "aws",
			ResourceType: "aws_s3_bucket",
			ResourceID:   bucketName,
			Action:       "restore_tags",
			OriginalData: map[string]interface{}{
				"original_tags": existingTags,
			},
			Timestamp: time.Now(),
		}
		result.Success = true

	case "aws_vpc", "aws_security_group", "aws_subnet":
		// Add tag to VPC resources
		resourceARN := s.extractResourceARN(resource)
		if resourceARN == "" {
			return nil, fmt.Errorf("could not extract resource ARN")
		}

		_, err := s.ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
			Resources: []string{resourceARN},
			Tags:      []types.Tag{driftTag},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add tag: %w", err)
		}

		result.Changes["added_tag"] = map[string]string{
			*driftTag.Key: *driftTag.Value,
		}
		result.RollbackData = &RollbackData{
			Provider:     "aws",
			ResourceType: resource.Type,
			ResourceID:   resourceARN,
			Action:       "remove_tag",
			OriginalData: map[string]interface{}{
				"tag_key": *driftTag.Key,
			},
			Timestamp: time.Now(),
		}
		result.Success = true

	default:
		return nil, fmt.Errorf("tag drift not implemented for resource type %s", resource.Type)
	}

	return result, nil
}

// simulateSecurityGroupRuleDrift adds a new rule to a security group
func (s *AWSSimulator) simulateSecurityGroupRuleDrift(ctx context.Context, resource *state.Resource) (*SimulationResult, error) {
	result := &SimulationResult{
		Provider:     "aws",
		ResourceType: resource.Type,
		ResourceID:   resource.ID,
		DriftType:    DriftTypeRuleAddition,
		Changes:      make(map[string]interface{}),
		CostEstimate: "$0.00 (security group rules are free)",
	}

	// Find a security group
	var sgID string
	if resource.Type == "aws_security_group" {
		sgID = s.extractSecurityGroupID(resource)
	} else if resource.Type == "aws_instance" {
		// Get security group from instance
		sgID = s.extractInstanceSecurityGroup(ctx, resource)
	} else {
		return nil, fmt.Errorf("cannot add security group rule to resource type %s", resource.Type)
	}

	if sgID == "" {
		return nil, fmt.Errorf("could not find security group ID")
	}

	// Add a harmless rule (allow HTTPS from drift test)
	rule := types.IpPermission{
		IpProtocol: aws.String("tcp"),
		FromPort:   aws.Int32(8443),
		ToPort:     aws.Int32(8443),
		IpRanges: []types.IpRange{
			{
				CidrIp:      aws.String("192.0.2.0/32"), // TEST-NET-1 (safe IP)
				Description: aws.String("DriftSimulation - Test rule"),
			},
		},
	}

	_, err := s.ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       aws.String(sgID),
		IpPermissions: []types.IpPermission{rule},
	})
	if err != nil {
		// If rule already exists, that's okay
		if !strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("failed to add security group rule: %w", err)
		}
	}

	result.Changes["added_rule"] = map[string]interface{}{
		"protocol":    "tcp",
		"port":        8443,
		"source":      "192.0.2.0/32",
		"description": "DriftSimulation - Test rule",
	}
	result.RollbackData = &RollbackData{
		Provider:     "aws",
		ResourceType: "aws_security_group",
		ResourceID:   sgID,
		Action:       "remove_rule",
		OriginalData: map[string]interface{}{
			"rule": rule,
		},
		Timestamp: time.Now(),
	}
	result.Success = true

	return result, nil
}

// simulateResourceCreation creates a new resource not in state
func (s *AWSSimulator) simulateResourceCreation(ctx context.Context, resource *state.Resource, state *state.TerraformState) (*SimulationResult, error) {
	result := &SimulationResult{
		Provider:     "aws",
		DriftType:    DriftTypeResourceCreation,
		Changes:      make(map[string]interface{}),
		CostEstimate: "$0.00 (using free tier resources)",
	}

	// Create a small S3 bucket (free tier)
	bucketName := fmt.Sprintf("drift-simulation-%d", time.Now().Unix())

	_, err := s.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		CreateBucketConfiguration: &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(s.region),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 bucket: %w", err)
	}

	// Add lifecycle rule to auto-delete after 1 day
	_, err = s.s3Client.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket: aws.String(bucketName),
		LifecycleConfiguration: &s3types.BucketLifecycleConfiguration{
			Rules: []s3types.LifecycleRule{
				{
					ID:     aws.String("auto-cleanup"),
					Status: s3types.ExpirationStatusEnabled,
					Expiration: &s3types.LifecycleExpiration{
						Days: aws.Int32(1),
					},
				},
			},
		},
	})

	result.ResourceType = "aws_s3_bucket"
	result.ResourceID = bucketName
	result.Changes["created_resource"] = map[string]interface{}{
		"type":   "aws_s3_bucket",
		"name":   bucketName,
		"region": s.region,
	}
	result.RollbackData = &RollbackData{
		Provider:     "aws",
		ResourceType: "aws_s3_bucket",
		ResourceID:   bucketName,
		Action:       "delete_resource",
		Timestamp:    time.Now(),
	}
	result.Success = true

	return result, nil
}

// simulateAttributeChange modifies a resource attribute
func (s *AWSSimulator) simulateAttributeChange(ctx context.Context, resource *state.Resource) (*SimulationResult, error) {
	result := &SimulationResult{
		Provider:     "aws",
		ResourceType: resource.Type,
		ResourceID:   resource.ID,
		DriftType:    DriftTypeAttributeChange,
		Changes:      make(map[string]interface{}),
		CostEstimate: "$0.00",
	}

	switch resource.Type {
	case "aws_s3_bucket":
		// Change versioning status
		bucketName := s.extractBucketName(resource)
		if bucketName == "" {
			return nil, fmt.Errorf("could not extract bucket name")
		}

		// Get current versioning status
		versioningOutput, err := s.s3Client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get bucket versioning: %w", err)
		}

		originalStatus := string(versioningOutput.Status)
		newStatus := s3types.BucketVersioningStatusSuspended
		if originalStatus == "" || originalStatus == string(s3types.BucketVersioningStatusSuspended) {
			newStatus = s3types.BucketVersioningStatusEnabled
		}

		// Change versioning
		_, err = s.s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
			Bucket: aws.String(bucketName),
			VersioningConfiguration: &s3types.VersioningConfiguration{
				Status: newStatus,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to change bucket versioning: %w", err)
		}

		result.Changes["versioning"] = map[string]string{
			"before": originalStatus,
			"after":  string(newStatus),
		}
		result.RollbackData = &RollbackData{
			Provider:     "aws",
			ResourceType: "aws_s3_bucket",
			ResourceID:   bucketName,
			Action:       "restore_versioning",
			OriginalData: map[string]interface{}{
				"versioning_status": originalStatus,
			},
			Timestamp: time.Now(),
		}
		result.Success = true

	case "aws_instance":
		// For EC2 instances, we'll just simulate by adding a tag
		// (changing instance type would cost money and require stop/start)
		return s.simulateTagDrift(ctx, resource)

	default:
		return nil, fmt.Errorf("attribute change not implemented for resource type %s", resource.Type)
	}

	return result, nil
}

// DetectDrift detects drift in AWS resources
func (s *AWSSimulator) DetectDrift(ctx context.Context, state *state.TerraformState) ([]DriftItem, error) {
	var drifts []DriftItem

	// Initialize clients if needed
	if s.ec2Client == nil {
		if err := s.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	// Check each resource in state
	for _, resource := range state.Resources {
		if !strings.HasPrefix(resource.Type, "aws_") {
			continue
		}

		drift := s.checkResourceDrift(ctx, &resource)
		if drift != nil {
			drifts = append(drifts, *drift)
		}
	}

	// Also check for unmanaged resources (like our drift simulation bucket)
	unmanagedDrifts := s.checkUnmanagedResources(ctx, state)
	drifts = append(drifts, unmanagedDrifts...)

	return drifts, nil
}

// checkResourceDrift checks a single resource for drift
func (s *AWSSimulator) checkResourceDrift(ctx context.Context, resource *state.Resource) *DriftItem {
	switch resource.Type {
	case "aws_instance":
		return s.checkInstanceDrift(ctx, resource)
	case "aws_s3_bucket":
		return s.checkBucketDrift(ctx, resource)
	case "aws_security_group":
		return s.checkSecurityGroupDrift(ctx, resource)
	default:
		return nil
	}
}

// checkInstanceDrift checks EC2 instance for drift
func (s *AWSSimulator) checkInstanceDrift(ctx context.Context, resource *state.Resource) *DriftItem {
	instanceID := s.extractInstanceID(resource)
	if instanceID == "" {
		return nil
	}

	// Get current instance state
	output, err := s.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil || len(output.Reservations) == 0 {
		return nil
	}

	instance := output.Reservations[0].Instances[0]

	// Check for drift simulation tags
	for _, tag := range instance.Tags {
		if tag.Key != nil && *tag.Key == "DriftSimulation" {
			return &DriftItem{
				ResourceID:   instanceID,
				ResourceType: "aws_instance",
				DriftType:    "tag_addition",
				Before: map[string]interface{}{
					"tags": s.extractResourceTags(resource),
				},
				After: map[string]interface{}{
					"tags": s.convertEC2Tags(instance.Tags),
				},
				Impact: "Low - Tag addition detected",
			}
		}
	}

	return nil
}

// checkBucketDrift checks S3 bucket for drift
func (s *AWSSimulator) checkBucketDrift(ctx context.Context, resource *state.Resource) *DriftItem {
	bucketName := s.extractBucketName(resource)
	if bucketName == "" {
		return nil
	}

	// Check tags
	taggingOutput, err := s.s3Client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && taggingOutput != nil {
		for _, tag := range taggingOutput.TagSet {
			if tag.Key != nil && *tag.Key == "DriftSimulation" {
				return &DriftItem{
					ResourceID:   bucketName,
					ResourceType: "aws_s3_bucket",
					DriftType:    "tag_addition",
					Before: map[string]interface{}{
						"tags": s.extractResourceTags(resource),
					},
					After: map[string]interface{}{
						"tags": s.convertS3Tags(taggingOutput.TagSet),
					},
					Impact: "Low - Tag addition detected",
				}
			}
		}
	}

	// Check versioning
	versioningOutput, err := s.s3Client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && versioningOutput != nil {
		stateVersioning := s.extractBucketVersioning(resource)
		currentVersioning := string(versioningOutput.Status)

		if stateVersioning != currentVersioning {
			return &DriftItem{
				ResourceID:   bucketName,
				ResourceType: "aws_s3_bucket",
				DriftType:    "attribute_change",
				Before: map[string]interface{}{
					"versioning": stateVersioning,
				},
				After: map[string]interface{}{
					"versioning": currentVersioning,
				},
				Impact: "Medium - Versioning configuration changed",
			}
		}
	}

	return nil
}

// checkSecurityGroupDrift checks security group for drift
func (s *AWSSimulator) checkSecurityGroupDrift(ctx context.Context, resource *state.Resource) *DriftItem {
	sgID := s.extractSecurityGroupID(resource)
	if sgID == "" {
		return nil
	}

	// Get current security group state
	output, err := s.ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{sgID},
	})
	if err != nil || len(output.SecurityGroups) == 0 {
		return nil
	}

	sg := output.SecurityGroups[0]

	// Check for drift simulation rule
	for _, rule := range sg.IpPermissions {
		for _, ipRange := range rule.IpRanges {
			if ipRange.Description != nil && strings.Contains(*ipRange.Description, "DriftSimulation") {
				return &DriftItem{
					ResourceID:   sgID,
					ResourceType: "aws_security_group",
					DriftType:    "rule_addition",
					Before: map[string]interface{}{
						"ingress_rules": s.extractSecurityGroupRules(resource),
					},
					After: map[string]interface{}{
						"ingress_rules": s.convertSecurityGroupRules(sg.IpPermissions),
					},
					Impact: "High - Security group rule added",
				}
			}
		}
	}

	return nil
}

// checkUnmanagedResources checks for resources not in state
func (s *AWSSimulator) checkUnmanagedResources(ctx context.Context, state *state.TerraformState) []DriftItem {
	var drifts []DriftItem

	// Check for drift simulation buckets
	output, err := s.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err == nil && output != nil {
		for _, bucket := range output.Buckets {
			if bucket.Name != nil && strings.HasPrefix(*bucket.Name, "drift-simulation-") {
				// Check if this bucket is in state
				found := false
				for _, resource := range state.Resources {
					if resource.Type == "aws_s3_bucket" {
						bucketName := s.extractBucketName(&resource)
						if bucketName == *bucket.Name {
							found = true
							break
						}
					}
				}

				if !found {
					drifts = append(drifts, DriftItem{
						ResourceID:   *bucket.Name,
						ResourceType: "aws_s3_bucket",
						DriftType:    "unmanaged_resource",
						After: map[string]interface{}{
							"name":          *bucket.Name,
							"creation_date": bucket.CreationDate,
						},
						Impact: "High - Unmanaged S3 bucket detected",
					})
				}
			}
		}
	}

	return drifts
}

// Rollback undoes the simulated drift
func (s *AWSSimulator) Rollback(ctx context.Context, data *RollbackData) error {
	// Initialize clients if needed
	if s.ec2Client == nil {
		if err := s.Initialize(ctx); err != nil {
			return err
		}
	}

	switch data.Action {
	case "remove_tag":
		return s.rollbackTagRemoval(ctx, data)
	case "restore_tags":
		return s.rollbackTagRestore(ctx, data)
	case "remove_rule":
		return s.rollbackRuleRemoval(ctx, data)
	case "delete_resource":
		return s.rollbackResourceDeletion(ctx, data)
	case "restore_versioning":
		return s.rollbackVersioningRestore(ctx, data)
	default:
		return fmt.Errorf("unknown rollback action: %s", data.Action)
	}
}

// Helper functions

func (s *AWSSimulator) findResource(resourceID string, state *state.TerraformState) *state.Resource {
	for _, resource := range state.Resources {
		if resource.ID == resourceID || resource.Name == resourceID {
			return &resource
		}
	}
	return nil
}

func (s *AWSSimulator) extractInstanceID(resource *state.Resource) string {
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if id, ok := resource.Instances[0].Attributes["id"].(string); ok {
			return id
		}
	}
	return ""
}

func (s *AWSSimulator) extractBucketName(resource *state.Resource) string {
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if name, ok := resource.Instances[0].Attributes["bucket"].(string); ok {
			return name
		}
		if id, ok := resource.Instances[0].Attributes["id"].(string); ok {
			return id
		}
	}
	return ""
}

func (s *AWSSimulator) extractSecurityGroupID(resource *state.Resource) string {
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if id, ok := resource.Instances[0].Attributes["id"].(string); ok {
			return id
		}
	}
	return ""
}

func (s *AWSSimulator) extractResourceARN(resource *state.Resource) string {
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if arn, ok := resource.Instances[0].Attributes["arn"].(string); ok {
			return arn
		}
		if id, ok := resource.Instances[0].Attributes["id"].(string); ok {
			return id
		}
	}
	return ""
}

func (s *AWSSimulator) extractInstanceSecurityGroup(ctx context.Context, resource *state.Resource) string {
	instanceID := s.extractInstanceID(resource)
	if instanceID == "" {
		return ""
	}

	output, err := s.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil || len(output.Reservations) == 0 || len(output.Reservations[0].Instances) == 0 {
		return ""
	}

	instance := output.Reservations[0].Instances[0]
	if len(instance.SecurityGroups) > 0 && instance.SecurityGroups[0].GroupId != nil {
		return *instance.SecurityGroups[0].GroupId
	}

	return ""
}

func (s *AWSSimulator) extractResourceTags(resource *state.Resource) map[string]string {
	tags := make(map[string]string)
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if t, ok := resource.Instances[0].Attributes["tags"].(map[string]interface{}); ok {
			for k, v := range t {
				tags[k] = fmt.Sprintf("%v", v)
			}
		}
	}
	return tags
}

func (s *AWSSimulator) extractBucketVersioning(resource *state.Resource) string {
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if v, ok := resource.Instances[0].Attributes["versioning"].(map[string]interface{}); ok {
			if enabled, ok := v["enabled"].(bool); ok && enabled {
				return "Enabled"
			}
		}
	}
	return ""
}

func (s *AWSSimulator) extractSecurityGroupRules(resource *state.Resource) []map[string]interface{} {
	var rules []map[string]interface{}
	if resource.Instances != nil && len(resource.Instances) > 0 {
		if ingress, ok := resource.Instances[0].Attributes["ingress"].([]interface{}); ok {
			for _, r := range ingress {
				if rule, ok := r.(map[string]interface{}); ok {
					rules = append(rules, rule)
				}
			}
		}
	}
	return rules
}

func (s *AWSSimulator) convertEC2Tags(tags []types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[*tag.Key] = *tag.Value
		}
	}
	return result
}

func (s *AWSSimulator) convertS3Tags(tags []s3types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[*tag.Key] = *tag.Value
		}
	}
	return result
}

func (s *AWSSimulator) convertSecurityGroupRules(permissions []types.IpPermission) []map[string]interface{} {
	var rules []map[string]interface{}
	for _, perm := range permissions {
		rule := make(map[string]interface{})
		if perm.IpProtocol != nil {
			rule["protocol"] = *perm.IpProtocol
		}
		if perm.FromPort != nil {
			rule["from_port"] = *perm.FromPort
		}
		if perm.ToPort != nil {
			rule["to_port"] = *perm.ToPort
		}
		if len(perm.IpRanges) > 0 {
			var cidrs []string
			for _, r := range perm.IpRanges {
				if r.CidrIp != nil {
					cidrs = append(cidrs, *r.CidrIp)
				}
			}
			rule["cidr_blocks"] = cidrs
		}
		rules = append(rules, rule)
	}
	return rules
}

// Rollback helper functions

func (s *AWSSimulator) rollbackTagRemoval(ctx context.Context, data *RollbackData) error {
	tagKey := data.OriginalData["tag_key"].(string)

	switch data.ResourceType {
	case "aws_instance", "aws_vpc", "aws_security_group", "aws_subnet":
		_, err := s.ec2Client.DeleteTags(ctx, &ec2.DeleteTagsInput{
			Resources: []string{data.ResourceID},
			Tags: []types.Tag{
				{Key: aws.String(tagKey)},
			},
		})
		return err
	}
	return nil
}

func (s *AWSSimulator) rollbackTagRestore(ctx context.Context, data *RollbackData) error {
	if data.ResourceType == "aws_s3_bucket" {
		originalTags := data.OriginalData["original_tags"].([]s3types.Tag)
		_, err := s.s3Client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{
			Bucket: aws.String(data.ResourceID),
			Tagging: &s3types.Tagging{
				TagSet: originalTags,
			},
		})
		return err
	}
	return nil
}

func (s *AWSSimulator) rollbackRuleRemoval(ctx context.Context, data *RollbackData) error {
	rule := data.OriginalData["rule"].(types.IpPermission)
	_, err := s.ec2Client.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
		GroupId:       aws.String(data.ResourceID),
		IpPermissions: []types.IpPermission{rule},
	})
	return err
}

func (s *AWSSimulator) rollbackResourceDeletion(ctx context.Context, data *RollbackData) error {
	if data.ResourceType == "aws_s3_bucket" {
		// First, delete all objects
		listOutput, err := s.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: aws.String(data.ResourceID),
		})
		if err == nil && listOutput != nil && len(listOutput.Contents) > 0 {
			var objects []s3types.ObjectIdentifier
			for _, obj := range listOutput.Contents {
				objects = append(objects, s3types.ObjectIdentifier{
					Key: obj.Key,
				})
			}
			_, _ = s.s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(data.ResourceID),
				Delete: &s3types.Delete{
					Objects: objects,
				},
			})
		}

		// Delete the bucket
		_, err = s.s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: aws.String(data.ResourceID),
		})
		return err
	}
	return nil
}

func (s *AWSSimulator) rollbackVersioningRestore(ctx context.Context, data *RollbackData) error {
	if data.ResourceType == "aws_s3_bucket" {
		originalStatus := data.OriginalData["versioning_status"].(string)
		status := s3types.BucketVersioningStatusSuspended
		if originalStatus == "Enabled" {
			status = s3types.BucketVersioningStatusEnabled
		}

		_, err := s.s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
			Bucket: aws.String(data.ResourceID),
			VersioningConfiguration: &s3types.VersioningConfiguration{
				Status: status,
			},
		})
		return err
	}
	return nil
}

// simulateResourceDeletion simulates the deletion of a resource
func (s *AWSSimulator) simulateResourceDeletion(ctx context.Context, resource *state.Resource) (*SimulationResult, error) {
	result := &SimulationResult{
		Provider:     "aws",
		ResourceType: resource.Type,
		ResourceID:   resource.ID,
		DriftType:    DriftTypeResourceDeletion,
		Changes:      make(map[string]interface{}),
		CostEstimate: "$0.00 (simulation only)",
	}

	// Store original resource data for rollback
	rollbackData := &RollbackData{
		ResourceID:   resource.ID,
		ResourceType: resource.Type,
		OriginalData: make(map[string]interface{}),
	}

	// Simulate different deletion scenarios based on resource type
	switch resource.Type {
	case "aws_instance":
		// Simulate EC2 instance deletion
		instanceID := s.extractInstanceID(resource)
		if instanceID == "" {
			return nil, fmt.Errorf("could not extract instance ID")
		}

		// Store original instance data
		rollbackData.OriginalData["instance_id"] = instanceID
		rollbackData.OriginalData["resource_type"] = "aws_instance"

		// Simulate deletion by creating a drift record
		result.Changes["deletion_simulated"] = true
		result.Changes["instance_id"] = instanceID
		result.Changes["deletion_time"] = time.Now().Format(time.RFC3339)

	case "aws_s3_bucket":
		// Simulate S3 bucket deletion
		bucketName := s.extractBucketName(resource)
		if bucketName == "" {
			return nil, fmt.Errorf("could not extract bucket name")
		}

		// Store original bucket data
		rollbackData.OriginalData["bucket_name"] = bucketName
		rollbackData.OriginalData["resource_type"] = "aws_s3_bucket"

		// Simulate deletion by creating a drift record
		result.Changes["deletion_simulated"] = true
		result.Changes["bucket_name"] = bucketName
		result.Changes["deletion_time"] = time.Now().Format(time.RFC3339)

	case "aws_security_group":
		// Simulate security group deletion
		sgID := s.extractSecurityGroupID(resource)
		if sgID == "" {
			return nil, fmt.Errorf("could not extract security group ID")
		}

		// Store original security group data
		rollbackData.OriginalData["security_group_id"] = sgID
		rollbackData.OriginalData["resource_type"] = "aws_security_group"

		// Simulate deletion by creating a drift record
		result.Changes["deletion_simulated"] = true
		result.Changes["security_group_id"] = sgID
		result.Changes["deletion_time"] = time.Now().Format(time.RFC3339)

	default:
		// Generic resource deletion simulation
		rollbackData.OriginalData["resource_id"] = resource.ID
		rollbackData.OriginalData["resource_type"] = resource.Type

		result.Changes["deletion_simulated"] = true
		result.Changes["resource_id"] = resource.ID
		result.Changes["deletion_time"] = time.Now().Format(time.RFC3339)
	}

	// Add drift detection record
	result.DetectedDrift = append(result.DetectedDrift, DriftItem{
		ResourceID:   resource.ID,
		ResourceType: resource.Type,
		DriftType:    "resource_deletion",
		Before: map[string]interface{}{
			"exists": true,
			"type":   resource.Type,
		},
		After: map[string]interface{}{
			"exists": false,
			"type":   resource.Type,
		},
		Impact: "High - Resource has been deleted",
	})

	result.RollbackData = rollbackData
	result.Success = true

	return result, nil
}
