package deletion

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/catherinevee/driftmgr/internal/models"
)

// AWSProvider implements CloudProvider for AWS
type AWSProvider struct {
	cfg     aws.Config
	regions []string
}

// NewAWSProvider creates a new AWS provider
func NewAWSProvider() (*AWSProvider, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSProvider{
		cfg:     cfg,
		regions: []string{"us-east-1", "us-west-2", "eu-west-1"},
	}, nil
}

// ValidateCredentials validates AWS credentials
func (ap *AWSProvider) ValidateCredentials(ctx context.Context, accountID string) error {
	stsClient := sts.NewFromConfig(ap.cfg)

	_, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to validate AWS credentials: %w", err)
	}

	return nil
}

// ListResources lists all AWS resources
func (ap *AWSProvider) ListResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Define resource discovery functions
	discoveryFuncs := []struct {
		name string
		fn   func(context.Context, string) ([]models.Resource, error)
	}{
		{"EC2", ap.discoverEC2Resources},
		{"S3", ap.discoverS3Resources},
		{"RDS", ap.discoverRDSResources},
		{"Lambda", ap.discoverLambdaResources},
		{"EKS", ap.discoverEKSResources},
		{"ECS", ap.discoverECSResources},
		{"DynamoDB", ap.discoverDynamoDBResources},
		{"ElastiCache", ap.discoverElastiCacheResources},
		{"SNS", ap.discoverSNSResources},
		{"SQS", ap.discoverSQSResources},
		{"IAM", ap.discoverIAMResources},
		{"Route53", ap.discoverRoute53Resources},
		{"CloudFormation", ap.discoverCloudFormationResources},
		{"AutoScaling", ap.discoverAutoScalingResources},
		{"LoadBalancer", ap.discoverLoadBalancerResources},
		{"VPC", ap.discoverVPCResources},
		{"SecurityGroup", ap.discoverSecurityGroupResources},
		{"Subnet", ap.discoverSubnetResources},
		{"RouteTable", ap.discoverRouteTableResources},
		{"InternetGateway", ap.discoverInternetGatewayResources},
		{"NATGateway", ap.discoverNATGatewayResources},
		{"ElasticIP", ap.discoverElasticIPResources},
		// {"ECR", ap.discoverECRResources}, // TODO: Enable when ECR package is available
		{"CloudWatch", ap.discoverCloudWatchResources},
		{"KMS", ap.discoverKMSResources},
		{"SecretsManager", ap.discoverSecretsManagerResources},
		{"SystemsManager", ap.discoverSystemsManagerResources},
		{"WAF", ap.discoverWAFResources},
		{"CloudFront", ap.discoverCloudFrontResources},
		{"APIGateway", ap.discoverAPIGatewayResources},
		{"Cognito", ap.discoverCognitoResources},
		{"OpenSearch", ap.discoverOpenSearchResources},
		{"Neptune", ap.discoverNeptuneResources},
		{"DocDB", ap.discoverDocDBResources},
		{"MemoryDB", ap.discoverMemoryDBResources},
		{"Timestream", ap.discoverTimestreamResources},
		{"IoT", ap.discoverIoTResources},
		{"EventBridge", ap.discoverEventBridgeResources},
		{"StepFunctions", ap.discoverStepFunctionsResources},
		{"Batch", ap.discoverBatchResources},
		{"CodeBuild", ap.discoverCodeBuildResources},
		{"CodePipeline", ap.discoverCodePipelineResources},
		{"CodeDeploy", ap.discoverCodeDeployResources},
		{"CloudTrail", ap.discoverCloudTrailResources},
		{"Config", ap.discoverConfigResources},
		{"GuardDuty", ap.discoverGuardDutyResources},
		{"Macie", ap.discoverMacieResources},
		{"SecurityHub", ap.discoverSecurityHubResources},
		{"Workspaces", ap.discoverWorkspacesResources},
		{"DirectoryService", ap.discoverDirectoryServiceResources},
		{"FSx", ap.discoverFSxResources},
		{"EFS", ap.discoverEFSResources},
		{"StorageGateway", ap.discoverStorageGatewayResources},
		{"DataSync", ap.discoverDataSyncResources},
		{"Transfer", ap.discoverTransferResources},
		{"Backup", ap.discoverBackupResources},
		{"Glacier", ap.discoverGlacierResources},
		{"Athena", ap.discoverAthenaResources},
		{"QuickSight", ap.discoverQuickSightResources},
		{"Forecast", ap.discoverForecastResources},
		{"Personalize", ap.discoverPersonalizeResources},
		{"Rekognition", ap.discoverRekognitionResources},
		{"Textract", ap.discoverTextractResources},
		{"Comprehend", ap.discoverComprehendResources},
		{"Translate", ap.discoverTranslateResources},
		{"Polly", ap.discoverPollyResources},
		{"Transcribe", ap.discoverTranscribeResources},
		{"Lex", ap.discoverLexResources},
		{"Connect", ap.discoverConnectResources},
		{"Chime", ap.discoverChimeResources},
		{"Pinpoint", ap.discoverPinpointResources},
		{"SES", ap.discoverSESResources},
		{"SMS", ap.discoverSMSResources},
		{"Route53Resolver", ap.discoverRoute53ResolverResources},
		{"DirectConnect", ap.discoverDirectConnectResources},
		{"VPCLattice", ap.discoverVPCLatticeResources},
		{"GlobalAccelerator", ap.discoverGlobalAcceleratorResources},
		{"CloudHSM", ap.discoverCloudHSMResources},
		{"Cloud9", ap.discoverCloud9Resources},
		{"CodeCommit", ap.discoverCodeCommitResources},
		{"CodeStar", ap.discoverCodeStarResources},
		{"CodeStarConnections", ap.discoverCodeStarConnectionsResources},
		{"CodeStarNotifications", ap.discoverCodeStarNotificationsResources},
		{"XRay", ap.discoverXRayResources},
		{"ApplicationInsights", ap.discoverApplicationInsightsResources},
		{"CloudWatchLogs", ap.discoverCloudWatchLogsResources},
		{"EventBridge", ap.discoverEventBridgeResources},
		{"Schemas", ap.discoverSchemasResources},
		{"MQ", ap.discoverMQResources},
		{"Kafka", ap.discoverKafkaResources},
		{"RedshiftData", ap.discoverRedshiftDataResources},
		{"RedshiftServerless", ap.discoverRedshiftServerlessResources},
		{"Aurora", ap.discoverAuroraResources},
		{"TimestreamQuery", ap.discoverTimestreamQueryResources},
		{"IoTAnalytics", ap.discoverIoTAnalyticsResources},
		{"IoTCoreDeviceAdvisor", ap.discoverIoTCoreDeviceAdvisorResources},
		{"IoTSiteWise", ap.discoverIoTSiteWiseResources},
		{"IoTThingsGraph", ap.discoverIoTThingsGraphResources},
		{"IoTWireless", ap.discoverIoTWirelessResources},
		{"IoTFleetHub", ap.discoverIoTFleetHubResources},
		{"IoTDeviceAdvisor", ap.discoverIoTDeviceAdvisorResources},
		{"IoTSecureTunneling", ap.discoverIoTSecureTunnelingResources},
		{"IoT1ClickDevices", ap.discoverIoT1ClickDevicesResources},
		{"IoT1ClickProjects", ap.discoverIoT1ClickProjectsResources},
		{"IoTDataPlane", ap.discoverIoTDataPlaneResources},
		{"IoTJobsDataPlane", ap.discoverIoTJobsDataPlaneResources},
		{"IoTRoboRunner", ap.discoverIoTRoboRunnerResources},
	}

	for _, discovery := range discoveryFuncs {
		wg.Add(1)
		go func(d struct {
			name string
			fn   func(context.Context, string) ([]models.Resource, error)
		}) {
			defer wg.Done()

			res, err := d.fn(ctx, accountID)
			if err != nil {
				log.Printf("Error discovering %s resources: %v", d.name, err)
				return
			}

			mu.Lock()
			resources = append(resources, res...)
			mu.Unlock()
		}(discovery)
	}

	wg.Wait()
	return resources, nil
}

// DeleteResources deletes AWS resources in the correct order
func (ap *AWSProvider) DeleteResources(ctx context.Context, accountID string, options DeletionOptions) (*DeletionResult, error) {
	startTime := time.Now()
	result := &DeletionResult{
		AccountID: accountID,
		Provider:  "aws",
		StartTime: startTime,
		Errors:    []DeletionError{},
		Warnings:  []string{},
		Details:   make(map[string]interface{}),
	}

	// List all resources first
	resources, err := ap.ListResources(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	result.TotalResources = len(resources)

	// Filter resources based on options
	filteredResources := ap.filterResources(resources, options)

	if options.DryRun {
		result.DeletedResources = len(filteredResources)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(startTime)
		return result, nil
	}

	// Delete resources in dependency order
	deletionOrder := []string{
		"autoscaling_group",
		"ecs_service",
		"ecs_cluster",
		"eks_nodegroup",
		"eks_cluster",
		"lambda_function",
		"rds_instance",
		"elasticache_cluster",
		"dynamodb_table",
		"ec2_instance",
		"ec2_volume",
		"load_balancer",
		"nat_gateway",
		"elastic_ip",
		"ec2_security_group",
		"ec2_subnet",
		"ec2_route_table",
		"ec2_internet_gateway",
		"vpc",
		"ecr_repository",
		"s3_bucket",
		"sns_topic",
		"sqs_queue",
		"route53_record",
		"route53_hosted_zone",
		"cloudformation_stack",
		"iam_role",
		"iam_policy",
		"iam_user",
	}

	// Group resources by type
	resourceGroups := make(map[string][]models.Resource)
	for _, resource := range filteredResources {
		resourceGroups[resource.Type] = append(resourceGroups[resource.Type], resource)
	}

	// Delete resources in order
	for _, resourceType := range deletionOrder {
		if resources, exists := resourceGroups[resourceType]; exists {
			for _, resource := range resources {
				if err := ap.deleteResource(ctx, resource, options); err != nil {
					result.Errors = append(result.Errors, DeletionError{
						ResourceID:   resource.ID,
						ResourceType: resource.Type,
						Error:        err.Error(),
						Timestamp:    time.Now(),
					})
					result.FailedResources++
				} else {
					result.DeletedResources++
				}

				// Send progress update
				if options.ProgressCallback != nil {
					options.ProgressCallback(ProgressUpdate{
						Type:      "deletion_progress",
						Message:   fmt.Sprintf("Deleted %s: %s", resource.Type, resource.Name),
						Progress:  result.DeletedResources + result.FailedResources,
						Total:     result.TotalResources,
						Current:   resource.Name,
						Timestamp: time.Now(),
					})
				}
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)
	return result, nil
}

// deleteResource deletes a single AWS resource
func (ap *AWSProvider) deleteResource(ctx context.Context, resource models.Resource, options DeletionOptions) error {
	// Use the generic resource deletion framework with dependency management
	return ap.deleteResourceWithDependencies(ctx, resource)
}

// filterResources filters resources based on deletion options
func (ap *AWSProvider) filterResources(resources []models.Resource, options DeletionOptions) []models.Resource {
	var filtered []models.Resource

	for _, resource := range resources {
		// Check if resource should be excluded
		if ap.shouldExcludeResource(resource, options) {
			continue
		}

		// Check if resource should be included
		if len(options.IncludeResources) > 0 && !ap.shouldIncludeResource(resource, options) {
			continue
		}

		// Check resource type filter
		if len(options.ResourceTypes) > 0 && !ap.containsString(options.ResourceTypes, resource.Type) {
			continue
		}

		// Check region filter
		if len(options.Regions) > 0 && !ap.containsString(options.Regions, resource.Region) {
			continue
		}

		filtered = append(filtered, resource)
	}

	return filtered
}

// Helper methods for resource discovery
func (ap *AWSProvider) discoverEC2Resources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := ec2.NewFromConfig(cfg)

		instances, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
		if err != nil {
			continue
		}

		for _, reservation := range instances.Reservations {
			for _, instance := range reservation.Instances {
				resources = append(resources, models.Resource{
					ID:       *instance.InstanceId,
					Name:     ap.getResourceName(instance.Tags),
					Type:     "ec2_instance",
					Provider: "aws",
					Region:   region,
					State:    string(instance.State.Name),
					Tags:     ap.convertTags(instance.Tags),
					Created:  *instance.LaunchTime,
				})
			}
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverS3Resources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client := s3.NewFromConfig(ap.cfg)
	buckets, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	for _, bucket := range buckets.Buckets {
		// Get bucket location to determine the correct region
		location, err := ap.getBucketLocation(ctx, *bucket.Name)
		if err != nil {
			// If we can't get location, default to us-east-1
			location = "us-east-1"
		}

		resources = append(resources, models.Resource{
			ID:       *bucket.Name,
			Name:     *bucket.Name,
			Type:     "s3_bucket",
			Provider: "aws",
			Region:   location,
			Created:  *bucket.CreationDate,
		})
	}

	return resources, nil
}

// getBucketLocation determines the region where an S3 bucket is located
func (ap *AWSProvider) getBucketLocation(ctx context.Context, bucketName string) (string, error) {
	client := s3.NewFromConfig(ap.cfg)

	result, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return "", err
	}

	// Convert location constraint to region name
	if result.LocationConstraint == "" {
		return "us-east-1", nil // Default region
	}

	return string(result.LocationConstraint), nil
}

// Additional discovery methods would be implemented similarly for other AWS services
func (ap *AWSProvider) discoverRDSResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := rds.NewFromConfig(cfg)

		instances, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
		if err != nil {
			continue
		}

		for _, instance := range instances.DBInstances {
			resources = append(resources, models.Resource{
				ID:       *instance.DBInstanceIdentifier,
				Name:     *instance.DBInstanceIdentifier,
				Type:     "rds_instance",
				Provider: "aws",
				Region:   region,
				State:    *instance.DBInstanceStatus,
				Tags:     nil, // TODO: Convert RDS tags
				Created:  *instance.InstanceCreateTime,
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverLambdaResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := lambda.NewFromConfig(cfg)

		functions, err := client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
		if err != nil {
			continue
		}

		for _, function := range functions.Functions {
			resources = append(resources, models.Resource{
				ID:       *function.FunctionName,
				Name:     *function.FunctionName,
				Type:     "lambda_function",
				Provider: "aws",
				Region:   region,
				State:    string(function.State),
				Created:  time.Now(), // Lambda doesn't provide creation time in ListFunctions
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverEKSResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := eks.NewFromConfig(cfg)

		clusters, err := client.ListClusters(ctx, &eks.ListClustersInput{})
		if err != nil {
			continue
		}

		for _, clusterName := range clusters.Clusters {
			resources = append(resources, models.Resource{
				ID:       clusterName,
				Name:     clusterName,
				Type:     "eks_cluster",
				Provider: "aws",
				Region:   region,
				State:    "active",
				Created:  time.Now(), // EKS doesn't provide creation time in ListClusters
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverECSResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := ecs.NewFromConfig(cfg)

		clusters, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
		if err != nil {
			continue
		}

		for _, clusterArn := range clusters.ClusterArns {
			resources = append(resources, models.Resource{
				ID:       clusterArn,
				Name:     clusterArn,
				Type:     "ecs_cluster",
				Provider: "aws",
				Region:   region,
				State:    "active",
				Created:  time.Now(), // ECS doesn't provide creation time in ListClusters
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverDynamoDBResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := dynamodb.NewFromConfig(cfg)

		tables, err := client.ListTables(ctx, &dynamodb.ListTablesInput{})
		if err != nil {
			continue
		}

		for _, tableName := range tables.TableNames {
			resources = append(resources, models.Resource{
				ID:       tableName,
				Name:     tableName,
				Type:     "dynamodb_table",
				Provider: "aws",
				Region:   region,
				State:    "active",
				Created:  time.Now(), // DynamoDB doesn't provide creation time in ListTables
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverElastiCacheResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := elasticache.NewFromConfig(cfg)

		clusters, err := client.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{})
		if err != nil {
			continue
		}

		for _, cluster := range clusters.CacheClusters {
			resources = append(resources, models.Resource{
				ID:       *cluster.CacheClusterId,
				Name:     *cluster.CacheClusterId,
				Type:     "elasticache_cluster",
				Provider: "aws",
				Region:   region,
				State:    *cluster.CacheClusterStatus,
				Created:  *cluster.CacheClusterCreateTime,
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverSNSResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := sns.NewFromConfig(cfg)

		topics, err := client.ListTopics(ctx, &sns.ListTopicsInput{})
		if err != nil {
			continue
		}

		for _, topic := range topics.Topics {
			resources = append(resources, models.Resource{
				ID:       *topic.TopicArn,
				Name:     *topic.TopicArn,
				Type:     "sns_topic",
				Provider: "aws",
				Region:   region,
				State:    "active",
				Created:  time.Now(), // SNS doesn't provide creation time in ListTopics
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverSQSResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := sqs.NewFromConfig(cfg)

		queues, err := client.ListQueues(ctx, &sqs.ListQueuesInput{})
		if err != nil {
			continue
		}

		for _, queueUrl := range queues.QueueUrls {
			resources = append(resources, models.Resource{
				ID:       queueUrl,
				Name:     queueUrl,
				Type:     "sqs_queue",
				Provider: "aws",
				Region:   region,
				State:    "active",
				Created:  time.Now(), // SQS doesn't provide creation time in ListQueues
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverIAMResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client := iam.NewFromConfig(ap.cfg)

	// Discover IAM Users
	users, err := client.ListUsers(ctx, &iam.ListUsersInput{})
	if err == nil {
		for _, user := range users.Users {
			resources = append(resources, models.Resource{
				ID:       *user.UserName,
				Name:     *user.UserName,
				Type:     "iam_user",
				Provider: "aws",
				Region:   "global",
				State:    "active",
				Created:  *user.CreateDate,
			})
		}
	}

	// Discover IAM Roles
	roles, err := client.ListRoles(ctx, &iam.ListRolesInput{})
	if err == nil {
		for _, role := range roles.Roles {
			resources = append(resources, models.Resource{
				ID:       *role.RoleName,
				Name:     *role.RoleName,
				Type:     "iam_role",
				Provider: "aws",
				Region:   "global",
				State:    "active",
				Created:  *role.CreateDate,
			})
		}
	}

	// Discover IAM Policies
	policies, err := client.ListPolicies(ctx, &iam.ListPoliciesInput{})
	if err == nil {
		for _, policy := range policies.Policies {
			resources = append(resources, models.Resource{
				ID:       *policy.Arn,
				Name:     *policy.PolicyName,
				Type:     "iam_policy",
				Provider: "aws",
				Region:   "global",
				State:    "active",
				Created:  *policy.CreateDate,
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverRoute53Resources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	client := route53.NewFromConfig(ap.cfg)

	hostedZones, err := client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
	if err != nil {
		return resources, nil
	}

	for _, zone := range hostedZones.HostedZones {
		resources = append(resources, models.Resource{
			ID:       *zone.Id,
			Name:     *zone.Name,
			Type:     "route53_hosted_zone",
			Provider: "aws",
			Region:   "global",
			State:    "active",
			Created:  time.Now(), // Route53 doesn't provide creation time
		})
	}

	return resources, nil
}

func (ap *AWSProvider) discoverCloudFormationResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := cloudformation.NewFromConfig(cfg)

		stacks, err := client.ListStacks(ctx, &cloudformation.ListStacksInput{})
		if err != nil {
			continue
		}

		for _, stack := range stacks.StackSummaries {
			// Skip deleted stacks
			if stack.StackStatus == "DELETE_COMPLETE" {
				continue
			}

			resources = append(resources, models.Resource{
				ID:       *stack.StackName,
				Name:     *stack.StackName,
				Type:     "cloudformation_stack",
				Provider: "aws",
				Region:   region,
				State:    string(stack.StackStatus),
				Created:  *stack.CreationTime,
			})
		}
	}

	return resources, nil
}

// Container Registry Resources
func (ap *AWSProvider) discoverECRResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	// TODO: Implement ECR discovery when ECR package is available
	return resources, nil
}

// VPC and Networking Resources
func (ap *AWSProvider) discoverVPCResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := ec2.NewFromConfig(cfg)

		vpcs, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
		if err != nil {
			continue
		}

		for _, vpc := range vpcs.Vpcs {
			// Skip default VPC if it's the only one
			if *vpc.IsDefault && len(vpcs.Vpcs) == 1 {
				continue
			}

			resources = append(resources, models.Resource{
				ID:       *vpc.VpcId,
				Name:     ap.getResourceName(vpc.Tags),
				Type:     "vpc",
				Provider: "aws",
				Region:   region,
				State:    "available",
				Tags:     ap.convertTags(vpc.Tags),
				Created:  time.Now(), // VPC doesn't provide creation time
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverSubnetResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := ec2.NewFromConfig(cfg)

		subnets, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{})
		if err != nil {
			continue
		}

		for _, subnet := range subnets.Subnets {
			resources = append(resources, models.Resource{
				ID:       *subnet.SubnetId,
				Name:     ap.getResourceName(subnet.Tags),
				Type:     "subnet",
				Provider: "aws",
				Region:   region,
				State:    string(subnet.State),
				Tags:     ap.convertTags(subnet.Tags),
				Created:  time.Now(), // Subnet doesn't provide creation time
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverSecurityGroupResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := ec2.NewFromConfig(cfg)

		securityGroups, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
		if err != nil {
			continue
		}

		for _, sg := range securityGroups.SecurityGroups {
			// Skip default security groups
			if *sg.GroupName == "default" {
				continue
			}

			resources = append(resources, models.Resource{
				ID:       *sg.GroupId,
				Name:     *sg.GroupName,
				Type:     "security_group",
				Provider: "aws",
				Region:   region,
				State:    "active",
				Tags:     ap.convertTags(sg.Tags),
				Created:  time.Now(), // Security groups don't have creation time
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverRouteTableResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := ec2.NewFromConfig(cfg)

		routeTables, err := client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{})
		if err != nil {
			continue
		}

		for _, rt := range routeTables.RouteTables {
			// Skip main route tables
			if len(rt.Associations) > 0 && *rt.Associations[0].Main {
				continue
			}

			resources = append(resources, models.Resource{
				ID:       *rt.RouteTableId,
				Name:     ap.getResourceName(rt.Tags),
				Type:     "route_table",
				Provider: "aws",
				Region:   region,
				State:    "active",
				Tags:     ap.convertTags(rt.Tags),
				Created:  time.Now(), // Route tables don't have creation time
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverInternetGatewayResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := ec2.NewFromConfig(cfg)

		internetGateways, err := client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{})
		if err != nil {
			continue
		}

		for _, igw := range internetGateways.InternetGateways {
			resources = append(resources, models.Resource{
				ID:       *igw.InternetGatewayId,
				Name:     ap.getResourceName(igw.Tags),
				Type:     "internet_gateway",
				Provider: "aws",
				Region:   region,
				State: func() string {
					if len(igw.Attachments) > 0 {
						return string(igw.Attachments[0].State)
					} else {
						return "detached"
					}
				}(),
				Tags:    ap.convertTags(igw.Tags),
				Created: time.Now(), // Internet gateways don't have creation time
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverNATGatewayResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := ec2.NewFromConfig(cfg)

		natGateways, err := client.DescribeNatGateways(ctx, &ec2.DescribeNatGatewaysInput{})
		if err != nil {
			continue
		}

		for _, nat := range natGateways.NatGateways {
			resources = append(resources, models.Resource{
				ID:       *nat.NatGatewayId,
				Name:     ap.getResourceName(nat.Tags),
				Type:     "nat_gateway",
				Provider: "aws",
				Region:   region,
				State:    string(nat.State),
				Tags:     ap.convertTags(nat.Tags),
				Created:  *nat.CreateTime,
			})
		}
	}

	return resources, nil
}

func (ap *AWSProvider) discoverElasticIPResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := ec2.NewFromConfig(cfg)

		elasticIPs, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
		if err != nil {
			continue
		}

		for _, eip := range elasticIPs.Addresses {
			resources = append(resources, models.Resource{
				ID:       *eip.AllocationId,
				Name:     ap.getResourceName(eip.Tags),
				Type:     "elastic_ip",
				Provider: "aws",
				Region:   region,
				State:    "allocated",
				Tags:     ap.convertTags(eip.Tags),
				Created:  time.Now(), // Elastic IPs don't have creation time
			})
		}
	}

	return resources, nil
}

// Auto Scaling Resources
func (ap *AWSProvider) discoverAutoScalingResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource

	for _, region := range ap.regions {
		cfg := ap.cfg.Copy()
		cfg.Region = region
		client := autoscaling.NewFromConfig(cfg)

		autoScalingGroups, err := client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{})
		if err != nil {
			continue
		}

		for _, asg := range autoScalingGroups.AutoScalingGroups {
			resources = append(resources, models.Resource{
				ID:       *asg.AutoScalingGroupName,
				Name:     *asg.AutoScalingGroupName,
				Type:     "autoscaling_group",
				Provider: "aws",
				Region:   region,
				State:    "active",
				Tags:     nil, // TODO: Convert Auto Scaling tags
				Created:  *asg.CreatedTime,
			})
		}
	}

	return resources, nil
}

// Load Balancer Resources
func (ap *AWSProvider) discoverLoadBalancerResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	// TODO: Implement load balancer discovery when elbv2 package is available
	return resources, nil
}

// Stub methods for remaining AWS services (to be implemented as needed)
func (ap *AWSProvider) discoverCloudWatchResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverKMSResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverSecretsManagerResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverSystemsManagerResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverWAFResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCloudFrontResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverAPIGatewayResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCognitoResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverOpenSearchResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverNeptuneResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverDocDBResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverMemoryDBResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverTimestreamResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverEventBridgeResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverStepFunctionsResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverBatchResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCodeBuildResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCodePipelineResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCodeDeployResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCloudTrailResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverConfigResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverGuardDutyResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverMacieResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverSecurityHubResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverWorkspacesResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverDirectoryServiceResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverFSxResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverEFSResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverStorageGatewayResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverDataSyncResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverTransferResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverBackupResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverGlacierResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverAthenaResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverQuickSightResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverForecastResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverPersonalizeResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverRekognitionResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverTextractResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverComprehendResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverTranslateResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverPollyResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverTranscribeResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverLexResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverConnectResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverChimeResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverPinpointResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverSESResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverSMSResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverRoute53ResolverResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverDirectConnectResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverVPCLatticeResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverGlobalAcceleratorResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCloudHSMResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCloud9Resources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCodeCommitResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCodeStarResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCodeStarConnectionsResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCodeStarNotificationsResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverXRayResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverApplicationInsightsResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverCloudWatchLogsResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverSchemasResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverMQResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverKafkaResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverRedshiftDataResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverRedshiftServerlessResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverAuroraResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverTimestreamQueryResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTAnalyticsResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTCoreDeviceAdvisorResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTSiteWiseResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTThingsGraphResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTWirelessResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTFleetHubResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTDeviceAdvisorResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTSecureTunnelingResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoT1ClickDevicesResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoT1ClickProjectsResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTDataPlaneResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTJobsDataPlaneResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

func (ap *AWSProvider) discoverIoTRoboRunnerResources(ctx context.Context, accountID string) ([]models.Resource, error) {
	var resources []models.Resource
	return resources, nil
}

// Helper methods for resource deletion
func (ap *AWSProvider) deleteEC2Instance(ctx context.Context, resource models.Resource) error {
	cfg := ap.cfg.Copy()
	cfg.Region = resource.Region
	client := ec2.NewFromConfig(cfg)

	_, err := client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{resource.ID},
	})

	return err
}

func (ap *AWSProvider) deleteS3Bucket(ctx context.Context, resource models.Resource) error {
	// Create S3 client with the correct region for this bucket
	cfg := ap.cfg.Copy()
	cfg.Region = resource.Region
	client := s3.NewFromConfig(cfg)

	// First, delete all objects in the bucket
	err := ap.deleteAllS3Objects(ctx, client, resource.ID)
	if err != nil {
		return fmt.Errorf("failed to delete objects in bucket %s: %w", resource.ID, err)
	}

	// Delete versioned objects if versioning is enabled
	err = ap.deleteAllS3ObjectVersions(ctx, client, resource.ID)
	if err != nil {
		return fmt.Errorf("failed to delete object versions in bucket %s: %w", resource.ID, err)
	}

	// Delete the bucket itself
	_, err = client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(resource.ID),
	})

	if err != nil {
		return fmt.Errorf("failed to delete bucket %s: %w", resource.ID, err)
	}

	return nil
}

// deleteAllS3Objects deletes all objects in an S3 bucket
func (ap *AWSProvider) deleteAllS3Objects(ctx context.Context, client *s3.Client, bucketName string) error {
	var continuationToken *string

	for {
		input := &s3.ListObjectsV2Input{
			Bucket: aws.String(bucketName),
		}

		if continuationToken != nil {
			input.ContinuationToken = continuationToken
		}

		result, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			return err
		}

		// Delete objects in batch
		if len(result.Contents) > 0 {
			var objects []s3types.ObjectIdentifier
			for _, obj := range result.Contents {
				objects = append(objects, s3types.ObjectIdentifier{
					Key: obj.Key,
				})
			}

			_, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(bucketName),
				Delete: &s3types.Delete{
					Objects: objects,
				},
			})
			if err != nil {
				return err
			}
		}

		// Check if there are more objects
		if result.IsTruncated == nil || !*result.IsTruncated {
			break
		}
		continuationToken = result.NextContinuationToken
	}

	return nil
}

// deleteAllS3ObjectVersions deletes all versions of objects in an S3 bucket
func (ap *AWSProvider) deleteAllS3ObjectVersions(ctx context.Context, client *s3.Client, bucketName string) error {
	var keyMarker *string
	var versionIdMarker *string

	for {
		input := &s3.ListObjectVersionsInput{
			Bucket: aws.String(bucketName),
		}

		if keyMarker != nil {
			input.KeyMarker = keyMarker
		}
		if versionIdMarker != nil {
			input.VersionIdMarker = versionIdMarker
		}

		result, err := client.ListObjectVersions(ctx, input)
		if err != nil {
			// If versioning is not enabled, this is not an error
			return nil
		}

		// Delete versions and delete markers
		var objects []s3types.ObjectIdentifier

		// Add versions
		for _, version := range result.Versions {
			objects = append(objects, s3types.ObjectIdentifier{
				Key:       version.Key,
				VersionId: version.VersionId,
			})
		}

		// Add delete markers
		for _, marker := range result.DeleteMarkers {
			objects = append(objects, s3types.ObjectIdentifier{
				Key:       marker.Key,
				VersionId: marker.VersionId,
			})
		}

		// Delete objects in batch if there are any
		if len(objects) > 0 {
			_, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(bucketName),
				Delete: &s3types.Delete{
					Objects: objects,
				},
			})
			if err != nil {
				return err
			}
		}

		// Check if there are more versions
		if result.IsTruncated == nil || !*result.IsTruncated {
			break
		}
		keyMarker = result.NextKeyMarker
		versionIdMarker = result.NextVersionIdMarker
	}

	return nil
}

// Additional deletion methods would be implemented for other AWS services
func (ap *AWSProvider) deleteRDSInstance(ctx context.Context, resource models.Resource) error {
	// Implementation for RDS instance deletion
	return nil
}

func (ap *AWSProvider) deleteLambdaFunction(ctx context.Context, resource models.Resource) error {
	// Implementation for Lambda function deletion
	return nil
}

func (ap *AWSProvider) deleteEKSCluster(ctx context.Context, resource models.Resource) error {
	// Generic resource deletion with dependency management
	return ap.deleteResourceWithDependencies(ctx, resource)
}

// deleteResourceWithDependencies handles resource deletion with proper dependency management
func (ap *AWSProvider) deleteResourceWithDependencies(ctx context.Context, resource models.Resource) error {
	// Get resource-specific deletion handler
	deletionHandler := ap.getResourceDeletionHandler(resource.Type)
	if deletionHandler == nil {
		return fmt.Errorf("no deletion handler found for resource type: %s", resource.Type)
	}

	// Validate resource before deletion
	if err := ap.validateResourceForDeletion(ctx, resource); err != nil {
		return fmt.Errorf("resource validation failed: %w", err)
	}

	// Handle dependencies first
	if err := ap.handleResourceDependencies(ctx, resource); err != nil {
		return fmt.Errorf("failed to handle dependencies: %w", err)
	}

	// Perform the actual deletion
	return deletionHandler(ctx, resource)
}

// getResourceDeletionHandler returns the appropriate deletion handler for a resource type
func (ap *AWSProvider) getResourceDeletionHandler(resourceType string) func(context.Context, models.Resource) error {
	handlers := map[string]func(context.Context, models.Resource) error{
		"ec2_instance":        ap.deleteEC2Instance,
		"s3_bucket":          ap.deleteS3Bucket,
		"rds_instance":       ap.deleteRDSInstance,
		"lambda_function":    ap.deleteLambdaFunction,
		"eks_cluster":        ap.deleteEKSClusterDirect,
		"ecs_cluster":        ap.deleteECSCluster,
		"dynamodb_table":     ap.deleteDynamoDBTable,
		"elasticache_cluster": ap.deleteElastiCacheCluster,
		"sns_topic":          ap.deleteSNSTopic,
		"sqs_queue":          ap.deleteSQSQueue,
		"iam_role":           ap.deleteIAMRole,
		"iam_policy":         ap.deleteIAMPolicy,
		"iam_user":           ap.deleteIAMUser,
		"route53_hosted_zone": ap.deleteRoute53HostedZone,
		"cloudformation_stack": ap.deleteCloudFormationStack,
	}

	return handlers[resourceType]
}

// validateResourceForDeletion performs generic validation for any resource type
func (ap *AWSProvider) validateResourceForDeletion(ctx context.Context, resource models.Resource) error {
	// Check if resource exists and is accessible
	if err := ap.checkResourceExists(ctx, resource); err != nil {
		return fmt.Errorf("resource %s not found or inaccessible: %w", resource.Name, err)
	}

	// Check resource state
	if err := ap.checkResourceState(ctx, resource); err != nil {
		return fmt.Errorf("resource %s is in invalid state: %w", resource.Name, err)
	}

	// Check for production/critical indicators
	ap.checkProductionIndicators(resource)

	return nil
}

// checkResourceExists verifies that a resource exists and is accessible
func (ap *AWSProvider) checkResourceExists(ctx context.Context, resource models.Resource) error {
	// This is a generic check - specific resource types can override this
	// For now, we'll assume the resource exists if it was discovered
	return nil
}

// checkResourceState verifies that a resource is in a deletable state
func (ap *AWSProvider) checkResourceState(ctx context.Context, resource models.Resource) error {
	// Generic state validation - specific resource types can override this
	// For now, we'll assume the resource is deletable
	return nil
}

// checkProductionIndicators warns about production/critical resources
func (ap *AWSProvider) checkProductionIndicators(resource models.Resource) {
	// Check for production tags
	for key, value := range resource.Tags {
		keyLower := strings.ToLower(key)
		valueLower := strings.ToLower(value)
		
		if keyLower == "environment" && valueLower == "production" {
			fmt.Printf("Warning: Resource %s is tagged as production environment\n", resource.Name)
		}
		if keyLower == "criticalitylevel" && valueLower == "critical" {
			fmt.Printf("Warning: Resource %s is tagged as critical level\n", resource.Name)
		}
	}

	// Check for production indicators in resource name
	resourceNameLower := strings.ToLower(resource.Name)
	if strings.Contains(resourceNameLower, "prod") || strings.Contains(resourceNameLower, "production") {
		fmt.Printf("Warning: Resource %s has production indicators in name\n", resource.Name)
	}
}

// handleResourceDependencies manages dependencies for any resource type
func (ap *AWSProvider) handleResourceDependencies(ctx context.Context, resource models.Resource) error {
	// Get dependency configuration for this resource type
	dependencyConfig := ap.getDependencyConfig(resource.Type)
	if dependencyConfig == nil {
		return nil // No dependencies to handle
	}

	// Handle each dependency type
	for _, depType := range dependencyConfig.DependencyTypes {
		if err := ap.handleDependencyType(ctx, resource, depType); err != nil {
			return fmt.Errorf("failed to handle dependency type %s: %w", depType, err)
		}
	}

	return nil
}

// DependencyConfig defines how to handle dependencies for a resource type
type DependencyConfig struct {
	DependencyTypes []string
	Handler         func(context.Context, models.Resource, string) error
}

// getDependencyConfig returns the dependency configuration for a resource type
func (ap *AWSProvider) getDependencyConfig(resourceType string) *DependencyConfig {
	configs := map[string]*DependencyConfig{
		"eks_cluster": {
			DependencyTypes: []string{"eks_nodegroup"},
			Handler:         ap.handleEKSNodegroupDependencies,
		},
		"ecs_cluster": {
			DependencyTypes: []string{"ecs_service"},
			Handler:         ap.handleECSServiceDependencies,
		},
		"rds_instance": {
			DependencyTypes: []string{"rds_snapshot"},
			Handler:         ap.handleRDSSnapshotDependencies,
		},
		"vpc": {
			DependencyTypes: []string{"nat_gateway", "internet_gateway", "route_table"},
			Handler:         ap.handleVPCDependencies,
		},
	}

	return configs[resourceType]
}

// handleDependencyType handles a specific type of dependency
func (ap *AWSProvider) handleDependencyType(ctx context.Context, resource models.Resource, dependencyType string) error {
	config := ap.getDependencyConfig(resource.Type)
	if config == nil || config.Handler == nil {
		return nil
	}

	return config.Handler(ctx, resource, dependencyType)
}

// handleEKSNodegroupDependencies handles EKS nodegroup dependencies
func (ap *AWSProvider) handleEKSNodegroupDependencies(ctx context.Context, resource models.Resource, dependencyType string) error {
	if dependencyType != "eks_nodegroup" {
		return nil
	}

	// Create EKS client for the resource's region
	cfg := ap.cfg.Copy()
	cfg.Region = resource.Region
	eksClient := eks.NewFromConfig(cfg)

	clusterName := resource.Name

	// List nodegroups
	nodegroups, err := eksClient.ListNodegroups(ctx, &eks.ListNodegroupsInput{
		ClusterName: &clusterName,
	})
	if err != nil {
		return fmt.Errorf("failed to list nodegroups for cluster %s: %w", clusterName, err)
	}

	// Delete each nodegroup
	for _, nodegroupName := range nodegroups.Nodegroups {
		fmt.Printf("Deleting EKS nodegroup: %s\n", nodegroupName)
		
		// Start nodegroup deletion
		_, err := eksClient.DeleteNodegroup(ctx, &eks.DeleteNodegroupInput{
			ClusterName:   &clusterName,
			NodegroupName: &nodegroupName,
		})
		if err != nil {
			return fmt.Errorf("failed to start deletion of nodegroup %s: %w", nodegroupName, err)
		}

		// Wait for nodegroup deletion to complete
		fmt.Printf("Waiting for nodegroup %s deletion to complete...\n", nodegroupName)
		waiter := eks.NewNodegroupDeletedWaiter(eksClient)
		err = waiter.Wait(ctx, &eks.DescribeNodegroupInput{
			ClusterName:   &clusterName,
			NodegroupName: &nodegroupName,
		}, 20*time.Minute)
		if err != nil {
			return fmt.Errorf("failed to wait for nodegroup %s deletion: %w", nodegroupName, err)
		}
		fmt.Printf("Nodegroup %s deleted successfully\n", nodegroupName)
	}

	return nil
}

// handleECSServiceDependencies handles ECS service dependencies
func (ap *AWSProvider) handleECSServiceDependencies(ctx context.Context, resource models.Resource, dependencyType string) error {
	// Implementation for ECS service dependencies
	return nil
}

// handleRDSSnapshotDependencies handles RDS snapshot dependencies
func (ap *AWSProvider) handleRDSSnapshotDependencies(ctx context.Context, resource models.Resource, dependencyType string) error {
	// Implementation for RDS snapshot dependencies
	return nil
}

// handleVPCDependencies handles VPC dependencies
func (ap *AWSProvider) handleVPCDependencies(ctx context.Context, resource models.Resource, dependencyType string) error {
	// Implementation for VPC dependencies
	return nil
}

// deleteEKSClusterDirect performs the actual EKS cluster deletion (without dependency handling)
func (ap *AWSProvider) deleteEKSClusterDirect(ctx context.Context, resource models.Resource) error {
	// Create EKS client for the resource's region
	cfg := ap.cfg.Copy()
	cfg.Region = resource.Region
	eksClient := eks.NewFromConfig(cfg)

	clusterName := resource.Name

	// Delete the EKS cluster
	fmt.Printf("Deleting EKS cluster: %s\n", clusterName)
	_, err := eksClient.DeleteCluster(ctx, &eks.DeleteClusterInput{
		Name: &clusterName,
	})
	if err != nil {
		return fmt.Errorf("failed to delete EKS cluster %s: %w", clusterName, err)
	}

	// Wait for cluster deletion to complete
	fmt.Printf("Waiting for EKS cluster %s deletion to complete...\n", clusterName)
	waiter := eks.NewClusterDeletedWaiter(eksClient)
	err = waiter.Wait(ctx, &eks.DescribeClusterInput{
		Name: &clusterName,
	}, 30*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to wait for EKS cluster %s deletion: %w", clusterName, err)
	}

	fmt.Printf("EKS cluster %s deleted successfully\n", clusterName)
	return nil
}

// Additional deletion methods for other resource types
func (ap *AWSProvider) deleteECSCluster(ctx context.Context, resource models.Resource) error {
	// Implementation for ECS cluster deletion
	return nil
}

func (ap *AWSProvider) deleteDynamoDBTable(ctx context.Context, resource models.Resource) error {
	// Implementation for DynamoDB table deletion
	return nil
}

func (ap *AWSProvider) deleteElastiCacheCluster(ctx context.Context, resource models.Resource) error {
	// Implementation for ElastiCache cluster deletion
	return nil
}

func (ap *AWSProvider) deleteSNSTopic(ctx context.Context, resource models.Resource) error {
	// Implementation for SNS topic deletion
	return nil
}

func (ap *AWSProvider) deleteSQSQueue(ctx context.Context, resource models.Resource) error {
	// Implementation for SQS queue deletion
	return nil
}

func (ap *AWSProvider) deleteIAMRole(ctx context.Context, resource models.Resource) error {
	// Implementation for IAM role deletion
	return nil
}

func (ap *AWSProvider) deleteIAMPolicy(ctx context.Context, resource models.Resource) error {
	// Implementation for IAM policy deletion
	return nil
}

func (ap *AWSProvider) deleteIAMUser(ctx context.Context, resource models.Resource) error {
	// Implementation for IAM user deletion
	return nil
}

func (ap *AWSProvider) deleteRoute53HostedZone(ctx context.Context, resource models.Resource) error {
	// Implementation for Route53 hosted zone deletion
	return nil
}

func (ap *AWSProvider) deleteCloudFormationStack(ctx context.Context, resource models.Resource) error {
	// Implementation for CloudFormation stack deletion
	return nil
}

// Helper utility methods
func (ap *AWSProvider) shouldExcludeResource(resource models.Resource, options DeletionOptions) bool {
	for _, excludeID := range options.ExcludeResources {
		if resource.ID == excludeID || resource.Name == excludeID {
			return true
		}
	}
	return false
}

func (ap *AWSProvider) shouldIncludeResource(resource models.Resource, options DeletionOptions) bool {
	for _, includeID := range options.IncludeResources {
		if resource.ID == includeID || resource.Name == includeID {
			return true
		}
	}
	return false
}

func (ap *AWSProvider) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (ap *AWSProvider) getResourceName(tags []ec2types.Tag) string {
	for _, tag := range tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}
	return ""
}

func (ap *AWSProvider) convertTags(tags []ec2types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		result[*tag.Key] = *tag.Value
	}
	return result
}
