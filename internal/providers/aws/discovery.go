package aws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/catherinevee/driftmgr/internal/models"
)

// DiscoveryEngine handles AWS resource discovery
type DiscoveryEngine struct {
	client *Client
	// Service clients
	ec2Client   *ec2.Client
	s3Client    *s3.Client
	rdsClient   *rds.Client
	lambdaClient *lambda.Client
	iamClient   *iam.Client
	cfClient    *cloudformation.Client
}

// NewDiscoveryEngine creates a new AWS discovery engine
func NewDiscoveryEngine(client *Client) *DiscoveryEngine {
	config := client.GetConfig()
	
	return &DiscoveryEngine{
		client:       client,
		ec2Client:    ec2.NewFromConfig(config),
		s3Client:     s3.NewFromConfig(config),
		rdsClient:    rds.NewFromConfig(config),
		lambdaClient: lambda.NewFromConfig(config),
		iamClient:    iam.NewFromConfig(config),
		cfClient:     cloudformation.NewFromConfig(config),
	}
}

// DiscoverResources discovers all AWS resources
func (d *DiscoveryEngine) DiscoverResources(ctx context.Context, job *models.DiscoveryJob) (*models.DiscoveryResults, error) {
	results := &models.DiscoveryResults{
		TotalDiscovered:   0,
		ResourcesByType:   make(map[string]int),
		ResourcesByRegion: make(map[string]int),
		NewResources:      []string{},
		UpdatedResources:  []string{},
		DeletedResources:  []string{},
		Errors:            []models.DiscoveryError{},
		Summary:           make(map[string]interface{}),
	}

	// Use channels and goroutines for concurrent discovery
	var wg sync.WaitGroup
	resourceChan := make(chan []models.CloudResource, 6)
	errorChan := make(chan models.DiscoveryError, 6)

	// Discover different resource types concurrently
	wg.Add(6)
	
	go func() {
		defer wg.Done()
		resources, err := d.DiscoverEC2Resources(ctx)
		if err != nil {
			errorChan <- models.DiscoveryError{
				ResourceType: "ec2",
				Error:        err.Error(),
				Timestamp:    time.Now(),
			}
			resourceChan <- []models.CloudResource{}
		} else {
			resourceChan <- resources
		}
	}()

	go func() {
		defer wg.Done()
		resources, err := d.DiscoverS3Resources(ctx)
		if err != nil {
			errorChan <- models.DiscoveryError{
				ResourceType: "s3",
				Error:        err.Error(),
				Timestamp:    time.Now(),
			}
			resourceChan <- []models.CloudResource{}
		} else {
			resourceChan <- resources
		}
	}()

	go func() {
		defer wg.Done()
		resources, err := d.DiscoverRDSResources(ctx)
		if err != nil {
			errorChan <- models.DiscoveryError{
				ResourceType: "rds",
				Error:        err.Error(),
				Timestamp:    time.Now(),
			}
			resourceChan <- []models.CloudResource{}
		} else {
			resourceChan <- resources
		}
	}()

	go func() {
		defer wg.Done()
		resources, err := d.DiscoverLambdaResources(ctx)
		if err != nil {
			errorChan <- models.DiscoveryError{
				ResourceType: "lambda",
				Error:        err.Error(),
				Timestamp:    time.Now(),
			}
			resourceChan <- []models.CloudResource{}
		} else {
			resourceChan <- resources
		}
	}()

	go func() {
		defer wg.Done()
		resources, err := d.DiscoverIAMResources(ctx)
		if err != nil {
			errorChan <- models.DiscoveryError{
				ResourceType: "iam",
				Error:        err.Error(),
				Timestamp:    time.Now(),
			}
			resourceChan <- []models.CloudResource{}
		} else {
			resourceChan <- resources
		}
	}()

	go func() {
		defer wg.Done()
		resources, err := d.DiscoverCloudFormationResources(ctx)
		if err != nil {
			errorChan <- models.DiscoveryError{
				ResourceType: "cloudformation",
				Error:        err.Error(),
				Timestamp:    time.Now(),
			}
			resourceChan <- []models.CloudResource{}
		} else {
			resourceChan <- resources
		}
	}()

	// Close channels when all goroutines are done
	go func() {
		wg.Wait()
		close(resourceChan)
		close(errorChan)
	}()

	// Collect results
	allResources := []models.CloudResource{}
	
	// Collect resources
	for resources := range resourceChan {
		allResources = append(allResources, resources...)
	}

	// Collect errors
	for err := range errorChan {
		results.Errors = append(results.Errors, err)
	}

	// Process results
	results.TotalDiscovered = len(allResources)
	
	for _, resource := range allResources {
		// Count by type
		results.ResourcesByType[resource.Type]++
		
		// Count by region
		results.ResourcesByRegion[resource.Region]++
	}

	// Add summary information
	results.Summary["discovery_time"] = time.Now()
	results.Summary["region"] = d.client.GetRegion()
	results.Summary["total_errors"] = len(results.Errors)

	return results, nil
}

// DiscoverEC2Resources discovers EC2 resources
func (d *DiscoveryEngine) DiscoverEC2Resources(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Discover EC2 Instances
	instances, err := d.discoverEC2Instances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover EC2 instances: %w", err)
	}
	resources = append(resources, instances...)

	// Discover Security Groups
	securityGroups, err := d.discoverSecurityGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover security groups: %w", err)
	}
	resources = append(resources, securityGroups...)

	// Discover Volumes
	volumes, err := d.discoverVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover volumes: %w", err)
	}
	resources = append(resources, volumes...)

	// Discover VPCs
	vpcs, err := d.discoverVPCs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover VPCs: %w", err)
	}
	resources = append(resources, vpcs...)

	return resources, nil
}

// discoverEC2Instances discovers EC2 instances
func (d *DiscoveryEngine) discoverEC2Instances(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	paginator := ec2.NewDescribeInstancesPaginator(d.ec2Client, &ec2.DescribeInstancesInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, reservation := range page.Reservations {
			for _, instance := range reservation.Instances {
				if instance.InstanceId == nil {
					continue
				}

				// Convert tags to map
				tags := make(map[string]string)
				for _, tag := range instance.Tags {
					if tag.Key != nil && tag.Value != nil {
						tags[*tag.Key] = *tag.Value
					}
				}

				// Get instance state
				state := "unknown"
				if instance.State != nil {
					state = string(instance.State.Name)
				}

				// Get instance type
				instanceType := string(instance.InstanceType)

				// Get creation time
				createdAt := time.Now()
				if instance.LaunchTime != nil {
					createdAt = *instance.LaunchTime
				}

				resource := models.CloudResource{
					ID:          *instance.InstanceId,
					Type:        "aws_instance",
					Name:        d.getResourceName(tags, *instance.InstanceId),
					Provider:    "aws",
					Region:      d.client.GetRegion(),
					AccountID:   "", // Will be set later from client
					Tags:        tags,
					Metadata: map[string]interface{}{
						"instance_type": instanceType,
						"state":         state,
						"launch_time":   instance.LaunchTime,
						"public_ip":     instance.PublicIpAddress,
						"private_ip":    instance.PrivateIpAddress,
						"vpc_id":        instance.VpcId,
						"subnet_id":     instance.SubnetId,
						"security_groups": instance.SecurityGroups,
					},
					LastDiscovered: time.Now(),
					CreatedAt:      createdAt,
					UpdatedAt:      time.Now(),
				}

				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

// discoverSecurityGroups discovers security groups
func (d *DiscoveryEngine) discoverSecurityGroups(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	paginator := ec2.NewDescribeSecurityGroupsPaginator(d.ec2Client, &ec2.DescribeSecurityGroupsInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, sg := range page.SecurityGroups {
			if sg.GroupId == nil {
				continue
			}

			// Convert tags to map
			tags := make(map[string]string)
			for _, tag := range sg.Tags {
				if tag.Key != nil && tag.Value != nil {
					tags[*tag.Key] = *tag.Value
				}
			}

			resource := models.CloudResource{
				ID:             *sg.GroupId,
				Type:           "aws_security_group",
				Name:           d.getResourceName(tags, *sg.GroupId),
				Provider:       "aws",
				Region:         d.client.GetRegion(),
				AccountID:      "", // Will be set later from client
				Tags:           tags,
				Metadata: map[string]interface{}{
					"group_name":        sg.GroupName,
					"description":       sg.Description,
					"vpc_id":           sg.VpcId,
					"ingress_rules":    sg.IpPermissions,
					"egress_rules":     sg.IpPermissionsEgress,
					"state":            "active",
				},
				LastDiscovered: time.Now(),
				CreatedAt:      time.Now(), // Security groups don't have creation time
				UpdatedAt:      time.Now(),
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// discoverVolumes discovers EBS volumes
func (d *DiscoveryEngine) discoverVolumes(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	paginator := ec2.NewDescribeVolumesPaginator(d.ec2Client, &ec2.DescribeVolumesInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, volume := range page.Volumes {
			if volume.VolumeId == nil {
				continue
			}

			// Convert tags to map
			tags := make(map[string]string)
			for _, tag := range volume.Tags {
				if tag.Key != nil && tag.Value != nil {
					tags[*tag.Key] = *tag.Value
				}
			}

				// Get volume state
				state := string(volume.State)

			// Get creation time
			createdAt := time.Now()
			if volume.CreateTime != nil {
				createdAt = *volume.CreateTime
			}

			resource := models.CloudResource{
				ID:             *volume.VolumeId,
				Type:           "aws_ebs_volume",
				Name:           d.getResourceName(tags, *volume.VolumeId),
				Provider:       "aws",
				Region:         d.client.GetRegion(),
				AccountID:      "", // Will be set later from client
				Tags:           tags,
				Metadata: map[string]interface{}{
					"size":           volume.Size,
					"volume_type":    volume.VolumeType,
					"encrypted":      volume.Encrypted,
					"availability_zone": volume.AvailabilityZone,
					"attachments":    volume.Attachments,
					"state":          state,
				},
				LastDiscovered: time.Now(),
				CreatedAt:      createdAt,
				UpdatedAt:      time.Now(),
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// discoverVPCs discovers VPCs
func (d *DiscoveryEngine) discoverVPCs(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	paginator := ec2.NewDescribeVpcsPaginator(d.ec2Client, &ec2.DescribeVpcsInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, vpc := range page.Vpcs {
			if vpc.VpcId == nil {
				continue
			}

			// Convert tags to map
			tags := make(map[string]string)
			for _, tag := range vpc.Tags {
				if tag.Key != nil && tag.Value != nil {
					tags[*tag.Key] = *tag.Value
				}
			}

				// Get VPC state
				state := string(vpc.State)

			resource := models.CloudResource{
				ID:             *vpc.VpcId,
				Type:           "aws_vpc",
				Name:           d.getResourceName(tags, *vpc.VpcId),
				Provider:       "aws",
				Region:         d.client.GetRegion(),
				AccountID:      "", // Will be set later from client
				Tags:           tags,
				Metadata: map[string]interface{}{
					"cidr_block":       vpc.CidrBlock,
					"is_default":       vpc.IsDefault,
					"dhcp_options_id":  vpc.DhcpOptionsId,
					"instance_tenancy": vpc.InstanceTenancy,
					"state":            state,
				},
				LastDiscovered: time.Now(),
				CreatedAt:      time.Now(), // VPCs don't have creation time
				UpdatedAt:      time.Now(),
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// getResourceName extracts name from tags or uses ID as fallback
func (d *DiscoveryEngine) getResourceName(tags map[string]string, id string) string {
	if name, exists := tags["Name"]; exists && name != "" {
		return name
	}
	return id
}

// DiscoverS3Resources discovers S3 resources
func (d *DiscoveryEngine) DiscoverS3Resources(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// List all S3 buckets
	result, err := d.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 buckets: %w", err)
	}

	for _, bucket := range result.Buckets {
		if bucket.Name == nil {
			continue
		}

		// Get bucket location
		locationResult, err := d.s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: bucket.Name,
		})
		if err != nil {
			// Continue with other buckets if we can't get location
			continue
		}

		// Get bucket tags
		tags := make(map[string]string)
		tagResult, err := d.s3Client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
			Bucket: bucket.Name,
		})
		if err == nil {
			for _, tag := range tagResult.TagSet {
				if tag.Key != nil && tag.Value != nil {
					tags[*tag.Key] = *tag.Value
				}
			}
		}

		// Get bucket versioning
		versioningResult, err := d.s3Client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
			Bucket: bucket.Name,
		})
		versioning := "Disabled"
		if err == nil {
			versioning = string(versioningResult.Status)
		}

		// Get bucket encryption
		encryptionResult, err := d.s3Client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
			Bucket: bucket.Name,
		})
		encrypted := false
		if err == nil && encryptionResult.ServerSideEncryptionConfiguration != nil {
			encrypted = true
		}

		// Get creation time
		createdAt := time.Now()
		if bucket.CreationDate != nil {
			createdAt = *bucket.CreationDate
		}

		resource := models.CloudResource{
			ID:             *bucket.Name,
			Type:           "aws_s3_bucket",
			Name:           d.getResourceName(tags, *bucket.Name),
			Provider:       "aws",
			Region:         string(locationResult.LocationConstraint),
			AccountID:      "", // Will be set later from client
			Tags:           tags,
			Metadata: map[string]interface{}{
				"creation_date": bucket.CreationDate,
				"versioning":    versioning,
				"encrypted":     encrypted,
				"state":         "active",
			},
			LastDiscovered: time.Now(),
			CreatedAt:      createdAt,
			UpdatedAt:      time.Now(),
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// DiscoverRDSResources discovers RDS resources
func (d *DiscoveryEngine) DiscoverRDSResources(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Discover RDS DB Instances
	instances, err := d.discoverRDSInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover RDS instances: %w", err)
	}
	resources = append(resources, instances...)

	// Discover RDS DB Clusters
	clusters, err := d.discoverRDSClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover RDS clusters: %w", err)
	}
	resources = append(resources, clusters...)

	return resources, nil
}

// discoverRDSInstances discovers RDS DB instances
func (d *DiscoveryEngine) discoverRDSInstances(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	paginator := rds.NewDescribeDBInstancesPaginator(d.rdsClient, &rds.DescribeDBInstancesInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, instance := range page.DBInstances {
			if instance.DBInstanceIdentifier == nil {
				continue
			}

			// Convert tags to map
			tags := make(map[string]string)
			// Note: RDS tags are retrieved separately via ListTagsForResource
			tagResult, err := d.rdsClient.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
				ResourceName: instance.DBInstanceArn,
			})
			if err == nil {
				for _, tag := range tagResult.TagList {
					if tag.Key != nil && tag.Value != nil {
						tags[*tag.Key] = *tag.Value
					}
				}
			}


			resource := models.CloudResource{
				ID:       *instance.DBInstanceIdentifier,
				Type:     "aws_db_instance",
				Name:     d.getResourceName(tags, *instance.DBInstanceIdentifier),
				Provider: "aws",
				Region:   d.client.GetRegion(),
				Tags:     tags,
				Metadata: map[string]interface{}{
					"engine":                instance.Engine,
					"engine_version":        instance.EngineVersion,
					"instance_class":        instance.DBInstanceClass,
					"allocated_storage":     instance.AllocatedStorage,
					"storage_type":          instance.StorageType,
					"multi_az":             instance.MultiAZ,
					"publicly_accessible":  instance.PubliclyAccessible,
					"vpc_security_groups":  instance.VpcSecurityGroups,
					"db_subnet_group":      instance.DBSubnetGroup,
					"availability_zone":    instance.AvailabilityZone,
					"backup_retention":     instance.BackupRetentionPeriod,
					"encrypted":            instance.StorageEncrypted,
				},
				CreatedAt: *instance.InstanceCreateTime,
				UpdatedAt: time.Now(),
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// discoverRDSClusters discovers RDS DB clusters
func (d *DiscoveryEngine) discoverRDSClusters(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	paginator := rds.NewDescribeDBClustersPaginator(d.rdsClient, &rds.DescribeDBClustersInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, cluster := range page.DBClusters {
			if cluster.DBClusterIdentifier == nil {
				continue
			}

			// Convert tags to map
			tags := make(map[string]string)
			tagResult, err := d.rdsClient.ListTagsForResource(ctx, &rds.ListTagsForResourceInput{
				ResourceName: cluster.DBClusterArn,
			})
			if err == nil {
				for _, tag := range tagResult.TagList {
					if tag.Key != nil && tag.Value != nil {
						tags[*tag.Key] = *tag.Value
					}
				}
			}


			resource := models.CloudResource{
				ID:       *cluster.DBClusterIdentifier,
				Type:     "aws_rds_cluster",
				Name:     d.getResourceName(tags, *cluster.DBClusterIdentifier),
				Provider: "aws",
				Region:   d.client.GetRegion(),
				Tags:     tags,
				Metadata: map[string]interface{}{
					"engine":                cluster.Engine,
					"engine_version":        cluster.EngineVersion,
					"database_name":         cluster.DatabaseName,
					"master_username":       cluster.MasterUsername,
					"allocated_storage":     cluster.AllocatedStorage,
					"storage_encrypted":     cluster.StorageEncrypted,
					"backup_retention":      cluster.BackupRetentionPeriod,
					"vpc_security_groups":   cluster.VpcSecurityGroups,
					"db_subnet_group":       cluster.DBSubnetGroup,
					"availability_zones":    cluster.AvailabilityZones,
					"multi_az":             cluster.MultiAZ,
					"port":                 cluster.Port,
				},
				CreatedAt: *cluster.ClusterCreateTime,
				UpdatedAt: time.Now(),
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// DiscoverLambdaResources discovers Lambda resources
func (d *DiscoveryEngine) DiscoverLambdaResources(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	paginator := lambda.NewListFunctionsPaginator(d.lambdaClient, &lambda.ListFunctionsInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, function := range page.Functions {
			if function.FunctionName == nil {
				continue
			}

			// Get function tags
			tags := make(map[string]string)
			tagResult, err := d.lambdaClient.ListTags(ctx, &lambda.ListTagsInput{
				Resource: function.FunctionArn,
			})
			if err == nil {
				for key, value := range tagResult.Tags {
					tags[key] = value
				}
			}


			resource := models.CloudResource{
				ID:       *function.FunctionName,
				Type:     "aws_lambda_function",
				Name:     d.getResourceName(tags, *function.FunctionName),
				Provider: "aws",
				Region:   d.client.GetRegion(),
				Tags:     tags,
				Metadata: map[string]interface{}{
					"function_arn":      function.FunctionArn,
					"runtime":          function.Runtime,
					"handler":          function.Handler,
					"code_size":        function.CodeSize,
					"description":      function.Description,
					"timeout":          function.Timeout,
					"memory_size":      function.MemorySize,
					"last_modified":    function.LastModified,
					"version":          function.Version,
					"environment":      function.Environment,
					"vpc_config":       function.VpcConfig,
					"dead_letter_config": function.DeadLetterConfig,
				},
				CreatedAt: time.Now(), // Lambda doesn't provide creation time
				UpdatedAt: time.Now(),
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// DiscoverIAMResources discovers IAM resources
func (d *DiscoveryEngine) DiscoverIAMResources(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Discover IAM Users
	users, err := d.discoverIAMUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover IAM users: %w", err)
	}
	resources = append(resources, users...)

	// Discover IAM Roles
	roles, err := d.discoverIAMRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover IAM roles: %w", err)
	}
	resources = append(resources, roles...)

	// Discover IAM Policies
	policies, err := d.discoverIAMPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover IAM policies: %w", err)
	}
	resources = append(resources, policies...)

	return resources, nil
}

// discoverIAMUsers discovers IAM users
func (d *DiscoveryEngine) discoverIAMUsers(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	paginator := iam.NewListUsersPaginator(d.iamClient, &iam.ListUsersInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, user := range page.Users {
			if user.UserName == nil {
				continue
			}

			// Get user tags
			tags := make(map[string]string)
			tagResult, err := d.iamClient.ListUserTags(ctx, &iam.ListUserTagsInput{
				UserName: user.UserName,
			})
			if err == nil {
				for _, tag := range tagResult.Tags {
					if tag.Key != nil && tag.Value != nil {
						tags[*tag.Key] = *tag.Value
					}
				}
			}

			resource := models.CloudResource{
				ID:       *user.UserName,
				Type:     "aws_iam_user",
				Name:     d.getResourceName(tags, *user.UserName),
				Provider: "aws",
				Region:   "global", // IAM is global
				Tags:     tags,
				Metadata: map[string]interface{}{
					"user_id":       user.UserId,
					"arn":          user.Arn,
					"path":         user.Path,
					"create_date":  user.CreateDate,
					"password_last_used": user.PasswordLastUsed,
				},
				CreatedAt: *user.CreateDate,
				UpdatedAt: time.Now(),
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// discoverIAMRoles discovers IAM roles
func (d *DiscoveryEngine) discoverIAMRoles(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	paginator := iam.NewListRolesPaginator(d.iamClient, &iam.ListRolesInput{})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, role := range page.Roles {
			if role.RoleName == nil {
				continue
			}

			// Get role tags
			tags := make(map[string]string)
			tagResult, err := d.iamClient.ListRoleTags(ctx, &iam.ListRoleTagsInput{
				RoleName: role.RoleName,
			})
			if err == nil {
				for _, tag := range tagResult.Tags {
					if tag.Key != nil && tag.Value != nil {
						tags[*tag.Key] = *tag.Value
					}
				}
			}

			resource := models.CloudResource{
				ID:       *role.RoleName,
				Type:     "aws_iam_role",
				Name:     d.getResourceName(tags, *role.RoleName),
				Provider: "aws",
				Region:   "global", // IAM is global
				Tags:     tags,
				Metadata: map[string]interface{}{
					"role_id":      role.RoleId,
					"arn":         role.Arn,
					"path":        role.Path,
					"create_date": role.CreateDate,
					"assume_role_policy_document": role.AssumeRolePolicyDocument,
					"description": role.Description,
					"max_session_duration": role.MaxSessionDuration,
				},
				CreatedAt: *role.CreateDate,
				UpdatedAt: time.Now(),
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// discoverIAMPolicies discovers IAM policies
func (d *DiscoveryEngine) discoverIAMPolicies(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	// Discover customer managed policies
	paginator := iam.NewListPoliciesPaginator(d.iamClient, &iam.ListPoliciesInput{
		Scope: "Local", // Only customer managed policies
	})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, policy := range page.Policies {
			if policy.PolicyName == nil {
				continue
			}

			// Get policy tags
			tags := make(map[string]string)
			tagResult, err := d.iamClient.ListPolicyTags(ctx, &iam.ListPolicyTagsInput{
				PolicyArn: policy.Arn,
			})
			if err == nil {
				for _, tag := range tagResult.Tags {
					if tag.Key != nil && tag.Value != nil {
						tags[*tag.Key] = *tag.Value
					}
				}
			}

			resource := models.CloudResource{
				ID:       *policy.PolicyName,
				Type:     "aws_iam_policy",
				Name:     d.getResourceName(tags, *policy.PolicyName),
				Provider: "aws",
				Region:   "global", // IAM is global
				Tags:     tags,
				Metadata: map[string]interface{}{
					"policy_id":   policy.PolicyId,
					"arn":        policy.Arn,
					"path":       policy.Path,
					"create_date": policy.CreateDate,
					"update_date": policy.UpdateDate,
					"description": policy.Description,
					"attachment_count": policy.AttachmentCount,
				},
				CreatedAt: *policy.CreateDate,
				UpdatedAt: *policy.UpdateDate,
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// DiscoverCloudFormationResources discovers CloudFormation resources
func (d *DiscoveryEngine) DiscoverCloudFormationResources(ctx context.Context) ([]models.CloudResource, error) {
	var resources []models.CloudResource

	paginator := cloudformation.NewListStacksPaginator(d.cfClient, &cloudformation.ListStacksInput{
		StackStatusFilter: []types.StackStatus{
			types.StackStatusCreateComplete,
			types.StackStatusUpdateComplete,
			types.StackStatusUpdateRollbackComplete,
			types.StackStatusImportComplete,
		},
	})
	
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, stackSummary := range page.StackSummaries {
			if stackSummary.StackName == nil {
				continue
			}

			// Get detailed stack information
			stackResult, err := d.cfClient.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
				StackName: stackSummary.StackName,
			})
			if err != nil {
				continue // Skip this stack if we can't get details
			}

			if len(stackResult.Stacks) == 0 {
				continue
			}

			stack := stackResult.Stacks[0]

			// Convert tags to map
			tags := make(map[string]string)
			for _, tag := range stack.Tags {
				if tag.Key != nil && tag.Value != nil {
					tags[*tag.Key] = *tag.Value
				}
			}


			resource := models.CloudResource{
				ID:       *stack.StackName,
				Type:     "aws_cloudformation_stack",
				Name:     d.getResourceName(tags, *stack.StackName),
				Provider: "aws",
				Region:   d.client.GetRegion(),
				Tags:     tags,
				Metadata: map[string]interface{}{
					"stack_id":           stack.StackId,
					"stack_status":       stack.StackStatus,
					"stack_status_reason": stack.StackStatusReason,
					"creation_time":      stack.CreationTime,
					"last_updated_time":  stack.LastUpdatedTime,
					"template_description": stack.Description,
					"capabilities":       stack.Capabilities,
					"outputs":           stack.Outputs,
					"parameters":        stack.Parameters,
					"role_arn":          stack.RoleARN,
					"timeout_in_minutes": stack.TimeoutInMinutes,
				},
				CreatedAt: *stack.CreationTime,
				UpdatedAt: time.Now(),
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// GetResourceDetails gets detailed information about a specific resource
func (d *DiscoveryEngine) GetResourceDetails(ctx context.Context, resourceID string) (*models.CloudResource, error) {
	// This is a simplified implementation that would need to be enhanced
	// to determine resource type and call appropriate service methods
	// For now, we'll return an error indicating this needs to be implemented
	// based on the specific resource type and ID format
	
	// In a real implementation, you would:
	// 1. Parse the resourceID to determine the resource type
	// 2. Call the appropriate AWS service method to get details
	// 3. Convert the response to a CloudResource model
	
	return nil, fmt.Errorf("GetResourceDetails not fully implemented - requires resource type detection")
}

// ValidateResource validates a resource configuration
func (d *DiscoveryEngine) ValidateResource(ctx context.Context, resource *models.CloudResource) error {
	// Basic validation - check required fields
	if resource.ID == "" {
		return fmt.Errorf("resource ID is required")
	}
	
	if resource.Type == "" {
		return fmt.Errorf("resource type is required")
	}
	
	if resource.Provider != "aws" {
		return fmt.Errorf("invalid provider for AWS discovery engine")
	}
	
	// Additional validation could include:
	// - Checking if resource exists in AWS
	// - Validating resource configuration against AWS best practices
	// - Checking for security issues
	
	return nil
}

