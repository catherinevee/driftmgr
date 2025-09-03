package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	
	"github.com/catherinevee/driftmgr/pkg/models"
)

// NotFoundError represents a resource not found error
type NotFoundError struct {
	ResourceType string
	ResourceID   string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("resource %s of type %s not found", e.ResourceID, e.ResourceType)
}

// AWSProvider implements CloudProvider for AWS
type AWSProvider struct {
	region    string
	awsConfig aws.Config
	ec2Client *ec2.Client
	s3Client  *s3.Client
	rdsClient *rds.Client
	iamClient *iam.Client
	lambdaClient *lambda.Client
	dynamoClient *dynamodb.Client
}

// NewAWSProvider creates a new AWS provider
func NewAWSProvider(region string) *AWSProvider {
	return &AWSProvider{
		region: region,
	}
}

// Initialize initializes the AWS clients
func (p *AWSProvider) Initialize(ctx context.Context) error {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(p.region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	
	p.awsConfig = cfg
	p.ec2Client = ec2.NewFromConfig(cfg)
	p.s3Client = s3.NewFromConfig(cfg)
	p.rdsClient = rds.NewFromConfig(cfg)
	p.iamClient = iam.NewFromConfig(cfg)
	p.lambdaClient = lambda.NewFromConfig(cfg)
	p.dynamoClient = dynamodb.NewFromConfig(cfg)
	
	return nil
}

// GetResource retrieves a specific AWS resource
func (p *AWSProvider) GetResource(ctx context.Context, resourceType string, resourceID string) (*models.Resource, error) {
	// Initialize if not already done
	if p.ec2Client == nil {
		if err := p.Initialize(ctx); err != nil {
			return nil, err
		}
	}
	
	switch {
	case strings.HasPrefix(resourceType, "aws_instance"):
		return p.getEC2Instance(ctx, resourceID)
	case strings.HasPrefix(resourceType, "aws_s3_bucket"):
		return p.getS3Bucket(ctx, resourceID)
	case strings.HasPrefix(resourceType, "aws_db_instance"):
		return p.getRDSInstance(ctx, resourceID)
	case strings.HasPrefix(resourceType, "aws_iam_role"):
		return p.getIAMRole(ctx, resourceID)
	case strings.HasPrefix(resourceType, "aws_lambda_function"):
		return p.getLambdaFunction(ctx, resourceID)
	case strings.HasPrefix(resourceType, "aws_dynamodb_table"):
		return p.getDynamoDBTable(ctx, resourceID)
	case strings.HasPrefix(resourceType, "aws_security_group"):
		return p.getSecurityGroup(ctx, resourceID)
	case strings.HasPrefix(resourceType, "aws_vpc"):
		return p.getVPC(ctx, resourceID)
	case strings.HasPrefix(resourceType, "aws_subnet"):
		return p.getSubnet(ctx, resourceID)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// ListResources lists all AWS resources
func (p *AWSProvider) ListResources(ctx context.Context) ([]*models.Resource, error) {
	// Initialize if not already done
	if p.ec2Client == nil {
		if err := p.Initialize(ctx); err != nil {
			return nil, err
		}
	}
	
	resources := []*models.Resource{}
	
	// List EC2 instances
	instances, err := p.listEC2Instances(ctx)
	if err == nil {
		resources = append(resources, instances...)
	}
	
	// List S3 buckets
	buckets, err := p.listS3Buckets(ctx)
	if err == nil {
		resources = append(resources, buckets...)
	}
	
	// Add other resource types as needed
	
	return resources, nil
}

// GetProviderName returns the provider name
func (p *AWSProvider) GetProviderName() string {
	return "aws"
}

// EC2 Instance methods
func (p *AWSProvider) getEC2Instance(ctx context.Context, instanceID string) (*models.Resource, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}
	
	result, err := p.ec2Client.DescribeInstances(ctx, input)
	if err != nil {
		return nil, err
	}
	
	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, &NotFoundError{ResourceType: "aws_instance", ResourceID: instanceID}
	}
	
	instance := result.Reservations[0].Instances[0]
	
	// Convert to models.Resource
	resource := &models.Resource{
		ID:   *instance.InstanceId,
		Type: "aws_instance",
		Name: p.getTagValue(instance.Tags, "Name"),
		Region: p.region,
		Tags: p.convertTags(instance.Tags),
		Attributes: map[string]interface{}{
			"id":                *instance.InstanceId,
			"instance_type":     string(instance.InstanceType),
			"ami":              *instance.ImageId,
			"availability_zone": *instance.Placement.AvailabilityZone,
			"subnet_id":        *instance.SubnetId,
			"vpc_id":           *instance.VpcId,
			"state":            string(instance.State.Name),
			"private_ip":       *instance.PrivateIpAddress,
		},
	}
	
	if instance.PublicIpAddress != nil {
		resource.Attributes["public_ip"] = *instance.PublicIpAddress
	}
	
	if instance.KeyName != nil {
		resource.Attributes["key_name"] = *instance.KeyName
	}
	
	if len(instance.SecurityGroups) > 0 {
		sgIds := make([]string, len(instance.SecurityGroups))
		for i, sg := range instance.SecurityGroups {
			sgIds[i] = *sg.GroupId
		}
		resource.Attributes["security_groups"] = sgIds
	}
	
	return resource, nil
}

func (p *AWSProvider) listEC2Instances(ctx context.Context) ([]*models.Resource, error) {
	resources := []*models.Resource{}
	
	paginator := ec2.NewDescribeInstancesPaginator(p.ec2Client, &ec2.DescribeInstancesInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		
		for _, reservation := range page.Reservations {
			for _, instance := range reservation.Instances {
				if instance.State.Name == "terminated" {
					continue
				}
				
				resource := &models.Resource{
					ID:   *instance.InstanceId,
					Type: "aws_instance",
					Name: p.getTagValue(instance.Tags, "Name"),
					Region: p.region,
					Tags: p.convertTags(instance.Tags),
					Attributes: map[string]interface{}{
						"id":           *instance.InstanceId,
						"instance_type": string(instance.InstanceType),
						"state":        string(instance.State.Name),
					},
				}
				resources = append(resources, resource)
			}
		}
	}
	
	return resources, nil
}

// S3 Bucket methods
func (p *AWSProvider) getS3Bucket(ctx context.Context, bucketName string) (*models.Resource, error) {
	// Check if bucket exists
	_, err := p.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	
	if err != nil {
		return nil, &NotFoundError{ResourceType: "aws_s3_bucket", ResourceID: bucketName}
	}
	
	// Get bucket location
	location, err := p.s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})
	
	region := "us-east-1"
	if err == nil && location.LocationConstraint != "" {
		region = string(location.LocationConstraint)
	}
	
	// Get bucket tags
	tags := make(map[string]string)
	tagging, err := p.s3Client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		for _, tag := range tagging.TagSet {
			tags[*tag.Key] = *tag.Value
		}
	}
	
	resource := &models.Resource{
		ID:     bucketName,
		Type:   "aws_s3_bucket",
		Name:   bucketName,
		Region: region,
		Tags:   tags,
		Attributes: map[string]interface{}{
			"id":     bucketName,
			"bucket": bucketName,
			"region": region,
		},
	}
	
	// Get versioning status
	versioning, err := p.s3Client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && versioning.Status != "" {
		resource.Attributes["versioning"] = string(versioning.Status)
	}
	
	// Get encryption
	encryption, err := p.s3Client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && encryption.ServerSideEncryptionConfiguration != nil {
		resource.Attributes["encryption"] = true
	}
	
	return resource, nil
}

func (p *AWSProvider) listS3Buckets(ctx context.Context) ([]*models.Resource, error) {
	resources := []*models.Resource{}
	
	result, err := p.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	
	for _, bucket := range result.Buckets {
		resource := &models.Resource{
			ID:   *bucket.Name,
			Type: "aws_s3_bucket",
			Name: *bucket.Name,
			Attributes: map[string]interface{}{
				"id":           *bucket.Name,
				"bucket":       *bucket.Name,
				"creation_date": bucket.CreationDate.Format("2006-01-02T15:04:05Z"),
			},
		}
		resources = append(resources, resource)
	}
	
	return resources, nil
}

// RDS Instance methods
func (p *AWSProvider) getRDSInstance(ctx context.Context, dbInstanceID string) (*models.Resource, error) {
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbInstanceID),
	}
	
	result, err := p.rdsClient.DescribeDBInstances(ctx, input)
	if err != nil {
		return nil, err
	}
	
	if len(result.DBInstances) == 0 {
		return nil, &NotFoundError{ResourceType: "aws_db_instance", ResourceID: dbInstanceID}
	}
	
	dbInstance := result.DBInstances[0]
	
	resource := &models.Resource{
		ID:   *dbInstance.DBInstanceIdentifier,
		Type: "aws_db_instance",
		Name: *dbInstance.DBInstanceIdentifier,
		Region: p.region,
		Attributes: map[string]interface{}{
			"id":                    *dbInstance.DBInstanceIdentifier,
			"db_instance_class":     *dbInstance.DBInstanceClass,
			"engine":                *dbInstance.Engine,
			"engine_version":        *dbInstance.EngineVersion,
			"allocated_storage":     *dbInstance.AllocatedStorage,
			"availability_zone":     *dbInstance.AvailabilityZone,
			"status":                *dbInstance.DBInstanceStatus,
			"multi_az":              *dbInstance.MultiAZ,
			"publicly_accessible":   *dbInstance.PubliclyAccessible,
		},
	}
	
	if dbInstance.Endpoint != nil {
		resource.Attributes["endpoint"] = *dbInstance.Endpoint.Address
		resource.Attributes["port"] = *dbInstance.Endpoint.Port
	}
	
	return resource, nil
}

// IAM Role methods
func (p *AWSProvider) getIAMRole(ctx context.Context, roleName string) (*models.Resource, error) {
	input := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}
	
	result, err := p.iamClient.GetRole(ctx, input)
	if err != nil {
		return nil, &NotFoundError{ResourceType: "aws_iam_role", ResourceID: roleName}
	}
	
	role := result.Role
	
	// Get tags
	tags := make(map[string]string)
	tagging, err := p.iamClient.ListRoleTags(ctx, &iam.ListRoleTagsInput{
		RoleName: aws.String(roleName),
	})
	if err == nil {
		for _, tag := range tagging.Tags {
			tags[*tag.Key] = *tag.Value
		}
	}
	
	resource := &models.Resource{
		ID:   *role.RoleName,
		Type: "aws_iam_role",
		Name: *role.RoleName,
		Tags: tags,
		Attributes: map[string]interface{}{
			"id":                      *role.RoleName,
			"name":                    *role.RoleName,
			"arn":                     *role.Arn,
			"path":                    *role.Path,
			"assume_role_policy":      *role.AssumeRolePolicyDocument,
			"create_date":             role.CreateDate.Format("2006-01-02T15:04:05Z"),
			"max_session_duration":    *role.MaxSessionDuration,
		},
	}
	
	if role.Description != nil {
		resource.Attributes["description"] = *role.Description
	}
	
	return resource, nil
}

// Lambda Function methods
func (p *AWSProvider) getLambdaFunction(ctx context.Context, functionName string) (*models.Resource, error) {
	input := &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	}
	
	result, err := p.lambdaClient.GetFunction(ctx, input)
	if err != nil {
		return nil, &NotFoundError{ResourceType: "aws_lambda_function", ResourceID: functionName}
	}
	
	config := result.Configuration
	
	// Get tags
	tags := make(map[string]string)
	if result.Tags != nil {
		tags = result.Tags
	}
	
	resource := &models.Resource{
		ID:     *config.FunctionName,
		Type:   "aws_lambda_function",
		Name:   *config.FunctionName,
		Region: p.region,
		Tags:   tags,
		Attributes: map[string]interface{}{
			"id":            *config.FunctionName,
			"function_name": *config.FunctionName,
			"arn":          *config.FunctionArn,
			"runtime":      string(config.Runtime),
			"handler":      *config.Handler,
			"role":         *config.Role,
			"timeout":      *config.Timeout,
			"memory_size":  *config.MemorySize,
			"last_modified": *config.LastModified,
		},
	}
	
	if config.Description != nil {
		resource.Attributes["description"] = *config.Description
	}
	
	if config.Environment != nil && len(config.Environment.Variables) > 0 {
		resource.Attributes["environment_variables"] = config.Environment.Variables
	}
	
	return resource, nil
}

// DynamoDB Table methods
func (p *AWSProvider) getDynamoDBTable(ctx context.Context, tableName string) (*models.Resource, error) {
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}
	
	result, err := p.dynamoClient.DescribeTable(ctx, input)
	if err != nil {
		return nil, &NotFoundError{ResourceType: "aws_dynamodb_table", ResourceID: tableName}
	}
	
	table := result.Table
	
	// Get tags
	tags := make(map[string]string)
	tagging, err := p.dynamoClient.ListTagsOfResource(ctx, &dynamodb.ListTagsOfResourceInput{
		ResourceArn: table.TableArn,
	})
	if err == nil {
		for _, tag := range tagging.Tags {
			tags[*tag.Key] = *tag.Value
		}
	}
	
	resource := &models.Resource{
		ID:     *table.TableName,
		Type:   "aws_dynamodb_table",
		Name:   *table.TableName,
		Region: p.region,
		Tags:   tags,
		Attributes: map[string]interface{}{
			"id":          *table.TableName,
			"name":        *table.TableName,
			"arn":         *table.TableArn,
			"status":      string(table.TableStatus),
			"item_count":  *table.ItemCount,
			"table_size":  *table.TableSizeBytes,
			"creation_time": table.CreationDateTime.Format("2006-01-02T15:04:05Z"),
		},
	}
	
	// Add billing mode
	if table.BillingModeSummary != nil {
		resource.Attributes["billing_mode"] = string(table.BillingModeSummary.BillingMode)
	}
	
	return resource, nil
}

// Security Group methods
func (p *AWSProvider) getSecurityGroup(ctx context.Context, groupID string) (*models.Resource, error) {
	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{groupID},
	}
	
	result, err := p.ec2Client.DescribeSecurityGroups(ctx, input)
	if err != nil {
		return nil, err
	}
	
	if len(result.SecurityGroups) == 0 {
		return nil, &NotFoundError{ResourceType: "aws_security_group", ResourceID: groupID}
	}
	
	sg := result.SecurityGroups[0]
	
	resource := &models.Resource{
		ID:   *sg.GroupId,
		Type: "aws_security_group",
		Name: *sg.GroupName,
		Region: p.region,
		Tags: p.convertTags(sg.Tags),
		Attributes: map[string]interface{}{
			"id":          *sg.GroupId,
			"name":        *sg.GroupName,
			"description": *sg.Description,
			"vpc_id":      *sg.VpcId,
		},
	}
	
	// Add ingress rules
	if len(sg.IpPermissions) > 0 {
		ingress := make([]map[string]interface{}, len(sg.IpPermissions))
		for i, rule := range sg.IpPermissions {
			ingress[i] = p.convertSecurityGroupRule(rule)
		}
		resource.Attributes["ingress"] = ingress
	}
	
	// Add egress rules
	if len(sg.IpPermissionsEgress) > 0 {
		egress := make([]map[string]interface{}, len(sg.IpPermissionsEgress))
		for i, rule := range sg.IpPermissionsEgress {
			egress[i] = p.convertSecurityGroupRule(rule)
		}
		resource.Attributes["egress"] = egress
	}
	
	return resource, nil
}

// VPC methods
func (p *AWSProvider) getVPC(ctx context.Context, vpcID string) (*models.Resource, error) {
	input := &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	}
	
	result, err := p.ec2Client.DescribeVpcs(ctx, input)
	if err != nil {
		return nil, err
	}
	
	if len(result.Vpcs) == 0 {
		return nil, &NotFoundError{ResourceType: "aws_vpc", ResourceID: vpcID}
	}
	
	vpc := result.Vpcs[0]
	
	resource := &models.Resource{
		ID:   *vpc.VpcId,
		Type: "aws_vpc",
		Name: p.getTagValue(vpc.Tags, "Name"),
		Region: p.region,
		Tags: p.convertTags(vpc.Tags),
		Attributes: map[string]interface{}{
			"id":         *vpc.VpcId,
			"cidr_block": *vpc.CidrBlock,
			"state":      string(vpc.State),
			"is_default": *vpc.IsDefault,
		},
	}
	
	if vpc.DhcpOptionsId != nil {
		resource.Attributes["dhcp_options_id"] = *vpc.DhcpOptionsId
	}
	
	return resource, nil
}

// Subnet methods
func (p *AWSProvider) getSubnet(ctx context.Context, subnetID string) (*models.Resource, error) {
	input := &ec2.DescribeSubnetsInput{
		SubnetIds: []string{subnetID},
	}
	
	result, err := p.ec2Client.DescribeSubnets(ctx, input)
	if err != nil {
		return nil, err
	}
	
	if len(result.Subnets) == 0 {
		return nil, &NotFoundError{ResourceType: "aws_subnet", ResourceID: subnetID}
	}
	
	subnet := result.Subnets[0]
	
	resource := &models.Resource{
		ID:   *subnet.SubnetId,
		Type: "aws_subnet",
		Name: p.getTagValue(subnet.Tags, "Name"),
		Region: p.region,
		Tags: p.convertTags(subnet.Tags),
		Attributes: map[string]interface{}{
			"id":                       *subnet.SubnetId,
			"vpc_id":                   *subnet.VpcId,
			"cidr_block":               *subnet.CidrBlock,
			"availability_zone":        *subnet.AvailabilityZone,
			"available_ip_address_count": *subnet.AvailableIpAddressCount,
			"map_public_ip_on_launch":   *subnet.MapPublicIpOnLaunch,
			"state":                    string(subnet.State),
		},
	}
	
	return resource, nil
}

// Helper methods
func (p *AWSProvider) getTagValue(tags []ec2types.Tag, key string) string {
	for _, tag := range tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}
	return ""
}

func (p *AWSProvider) convertTags(tags []ec2types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[*tag.Key] = *tag.Value
	}
	return result
}

func (p *AWSProvider) convertSecurityGroupRule(rule ec2types.IpPermission) map[string]interface{} {
	result := map[string]interface{}{}
	
	if rule.FromPort != nil {
		result["from_port"] = *rule.FromPort
	}
	if rule.ToPort != nil {
		result["to_port"] = *rule.ToPort
	}
	if rule.IpProtocol != nil {
		result["protocol"] = *rule.IpProtocol
	}
	
	// Add CIDR blocks
	if len(rule.IpRanges) > 0 {
		cidrs := make([]string, len(rule.IpRanges))
		for i, r := range rule.IpRanges {
			cidrs[i] = *r.CidrIp
		}
		result["cidr_blocks"] = cidrs
	}
	
	// Add security groups
	if len(rule.UserIdGroupPairs) > 0 {
		sgs := make([]string, len(rule.UserIdGroupPairs))
		for i, sg := range rule.UserIdGroupPairs {
			sgs[i] = *sg.GroupId
		}
		result["security_groups"] = sgs
	}
	
	return result
}