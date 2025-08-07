package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/catherinevee/driftmgr/internal/models"
)

// AWSProvider implements the Provider interface for AWS
type AWSProvider struct {
	cfg aws.Config
	ctx context.Context
}

// NewAWSProvider creates a new AWS provider
func NewAWSProvider() (*AWSProvider, error) {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSProvider{
		cfg: cfg,
		ctx: ctx,
	}, nil
}

// Name returns the provider name
func (p *AWSProvider) Name() string {
	return "Amazon Web Services"
}

// SupportedRegions returns the list of supported AWS regions
func (p *AWSProvider) SupportedRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2",
		"ap-south-1", "sa-east-1", "ca-central-1",
	}
}

// SupportedResourceTypes returns the list of supported AWS resource types
func (p *AWSProvider) SupportedResourceTypes() []string {
	return []string{
		"aws_instance",
		"aws_vpc",
		"aws_subnet",
		"aws_security_group",
		"aws_s3_bucket",
		"aws_db_instance",
		"aws_lb",
		"aws_iam_role",
		"aws_iam_policy",
		"aws_route_table",
		"aws_internet_gateway",
		"aws_nat_gateway",
	}
}

// Discover discovers AWS resources
func (p *AWSProvider) Discover(config Config) ([]models.Resource, error) {
	fmt.Println("  [AWS] Discovering resources using AWS SDK...")

	var allResources []models.Resource

	// If specific regions are requested, use them; otherwise use current region
	regions := config.Regions
	if len(regions) == 0 {
		regions = []string{p.cfg.Region}
	}

	for _, region := range regions {
		fmt.Printf("  [AWS] Scanning region: %s\n", region)

		// Create region-specific config
		regionalCfg := p.cfg.Copy()
		regionalCfg.Region = region

		// Discover EC2 instances
		if config.ResourceType == "" || config.ResourceType == "aws_instance" {
			instances, err := p.discoverEC2Instances(regionalCfg, region)
			if err != nil {
				fmt.Printf("  [AWS] Warning: Failed to discover EC2 instances in %s: %v\n", region, err)
			} else {
				allResources = append(allResources, instances...)
			}
		}

		// Discover VPCs
		if config.ResourceType == "" || config.ResourceType == "aws_vpc" {
			vpcs, err := p.discoverVPCs(regionalCfg, region)
			if err != nil {
				fmt.Printf("  [AWS] Warning: Failed to discover VPCs in %s: %v\n", region, err)
			} else {
				allResources = append(allResources, vpcs...)
			}
		}

		// Discover Security Groups
		if config.ResourceType == "" || config.ResourceType == "aws_security_group" {
			sgs, err := p.discoverSecurityGroups(regionalCfg, region)
			if err != nil {
				fmt.Printf("  [AWS] Warning: Failed to discover Security Groups in %s: %v\n", region, err)
			} else {
				allResources = append(allResources, sgs...)
			}
		}
	}

	// Discover S3 buckets (global service)
	if config.ResourceType == "" || config.ResourceType == "aws_s3_bucket" {
		buckets, err := p.discoverS3Buckets()
		if err != nil {
			fmt.Printf("  [AWS] Warning: Failed to discover S3 buckets: %v\n", err)
		} else {
			allResources = append(allResources, buckets...)
		}
	}

	fmt.Printf("  [AWS] Found %d resources\n", len(allResources))
	return allResources, nil
}

// discoverEC2Instances discovers EC2 instances in a region
func (p *AWSProvider) discoverEC2Instances(cfg aws.Config, region string) ([]models.Resource, error) {
	client := ec2.NewFromConfig(cfg)

	result, err := client.DescribeInstances(p.ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, err
	}

	var resources []models.Resource
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.State.Name == types.InstanceStateNameTerminated {
				continue // Skip terminated instances
			}

			resource := models.Resource{
				ID:            aws.ToString(instance.InstanceId),
				Name:          p.getInstanceName(instance.Tags),
				Type:          "aws_instance",
				TerraformType: "aws_instance",
				Provider:      "aws",
				Region:        region,
				Tags:          p.convertEC2Tags(instance.Tags),
				ImportID:      aws.ToString(instance.InstanceId),
				CreatedAt:     *instance.LaunchTime,
				Metadata: map[string]interface{}{
					"instance_type": string(instance.InstanceType),
					"state":         string(instance.State.Name),
					"vpc_id":        aws.ToString(instance.VpcId),
					"subnet_id":     aws.ToString(instance.SubnetId),
					"public_ip":     aws.ToString(instance.PublicIpAddress),
					"private_ip":    aws.ToString(instance.PrivateIpAddress),
				},
			}
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// discoverVPCs discovers VPCs in a region
func (p *AWSProvider) discoverVPCs(cfg aws.Config, region string) ([]models.Resource, error) {
	client := ec2.NewFromConfig(cfg)

	result, err := client.DescribeVpcs(p.ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, err
	}

	var resources []models.Resource
	for _, vpc := range result.Vpcs {
		resource := models.Resource{
			ID:            aws.ToString(vpc.VpcId),
			Name:          p.getVPCName(vpc.Tags),
			Type:          "aws_vpc",
			TerraformType: "aws_vpc",
			Provider:      "aws",
			Region:        region,
			Tags:          p.convertEC2Tags(vpc.Tags),
			ImportID:      aws.ToString(vpc.VpcId),
			CreatedAt:     time.Now(), // VPCs don't have creation time in API
			Metadata: map[string]interface{}{
				"cidr_block":       aws.ToString(vpc.CidrBlock),
				"state":            string(vpc.State),
				"is_default":       aws.ToBool(vpc.IsDefault),
				"instance_tenancy": string(vpc.InstanceTenancy),
			},
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverSecurityGroups discovers Security Groups in a region
func (p *AWSProvider) discoverSecurityGroups(cfg aws.Config, region string) ([]models.Resource, error) {
	client := ec2.NewFromConfig(cfg)

	result, err := client.DescribeSecurityGroups(p.ctx, &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, err
	}

	var resources []models.Resource
	for _, sg := range result.SecurityGroups {
		resource := models.Resource{
			ID:            aws.ToString(sg.GroupId),
			Name:          aws.ToString(sg.GroupName),
			Type:          "aws_security_group",
			TerraformType: "aws_security_group",
			Provider:      "aws",
			Region:        region,
			Tags:          p.convertEC2Tags(sg.Tags),
			ImportID:      aws.ToString(sg.GroupId),
			CreatedAt:     time.Now(),
			Metadata: map[string]interface{}{
				"description": aws.ToString(sg.Description),
				"vpc_id":      aws.ToString(sg.VpcId),
			},
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// discoverS3Buckets discovers S3 buckets (global service)
func (p *AWSProvider) discoverS3Buckets() ([]models.Resource, error) {
	client := s3.NewFromConfig(p.cfg)

	result, err := client.ListBuckets(p.ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	var resources []models.Resource
	for _, bucket := range result.Buckets {
		resource := models.Resource{
			ID:            aws.ToString(bucket.Name),
			Name:          aws.ToString(bucket.Name),
			Type:          "aws_s3_bucket",
			TerraformType: "aws_s3_bucket",
			Provider:      "aws",
			Region:        "global",
			Tags:          make(map[string]string), // S3 bucket tags require separate API call
			ImportID:      aws.ToString(bucket.Name),
			CreatedAt:     *bucket.CreationDate,
			Metadata:      map[string]interface{}{},
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// Helper functions
func (p *AWSProvider) getInstanceName(tags []types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func (p *AWSProvider) getVPCName(tags []types.Tag) string {
	for _, tag := range tags {
		if aws.ToString(tag.Key) == "Name" {
			return aws.ToString(tag.Value)
		}
	}
	return ""
}

func (p *AWSProvider) convertEC2Tags(tags []types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		key := aws.ToString(tag.Key)
		value := aws.ToString(tag.Value)
		if key != "" {
			result[key] = value
		}
	}
	return result
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
