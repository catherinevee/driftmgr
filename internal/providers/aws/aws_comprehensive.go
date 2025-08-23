package discovery

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/catherinevee/driftmgr/internal/models"
)

// ComprehensiveAWSDiscoverer discovers ALL AWS resources
type ComprehensiveAWSDiscoverer struct {
	cfg      aws.Config
	regions  []string
	progress chan AWSDiscoveryProgress
}

// AWSDiscoveryProgress tracks discovery progress for AWS
type AWSDiscoveryProgress struct {
	Region       string
	Service      string
	ResourceType string
	Count        int
	Message      string
}

// NewComprehensiveAWSDiscoverer creates a new comprehensive AWS discoverer
func NewComprehensiveAWSDiscoverer() (*ComprehensiveAWSDiscoverer, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &ComprehensiveAWSDiscoverer{
		cfg:      cfg,
		progress: make(chan AWSDiscoveryProgress, 100),
	}, nil
}

// DiscoverAllAWSResources discovers all AWS resources across all regions
func (d *ComprehensiveAWSDiscoverer) DiscoverAllAWSResources(ctx context.Context, regions []string) ([]models.Resource, error) {
	d.regions = regions
	if len(d.regions) == 0 {
		d.regions = d.getDefaultRegions()
	}

	// Start progress reporter
	go d.reportProgress()

	var allResources []models.Resource
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Process regions in parallel
	semaphore := make(chan struct{}, 5) // Limit concurrent regions

	for _, region := range d.regions {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			resources := d.discoverRegionResources(ctx, r)
			mu.Lock()
			allResources = append(allResources, resources...)
			mu.Unlock()
		}(region)
	}

	wg.Wait()

	// Also discover global resources (before closing the channel)
	globalResources := d.discoverGlobalResources(ctx)
	allResources = append(allResources, globalResources...)

	// Close the progress channel after all discovery is done
	close(d.progress)

	log.Printf("Comprehensive AWS discovery completed: %d total resources found", len(allResources))
	return allResources, nil
}

// discoverRegionResources discovers all resources in a specific region
func (d *ComprehensiveAWSDiscoverer) discoverRegionResources(ctx context.Context, region string) []models.Resource {
	var resources []models.Resource

	// Configure region-specific client
	cfg := d.cfg.Copy()
	cfg.Region = region

	// EC2 Resources
	resources = append(resources, d.discoverEC2Resources(ctx, cfg, region)...)

	// RDS Resources
	resources = append(resources, d.discoverRDSResources(ctx, cfg, region)...)

	// Lambda Functions
	resources = append(resources, d.discoverLambdaFunctions(ctx, cfg, region)...)

	// ECS Resources
	resources = append(resources, d.discoverECSResources(ctx, cfg, region)...)

	// EKS Clusters
	resources = append(resources, d.discoverEKSClusters(ctx, cfg, region)...)

	// Load Balancers
	resources = append(resources, d.discoverLoadBalancers(ctx, cfg, region)...)

	// ElastiCache
	resources = append(resources, d.discoverElastiCacheResources(ctx, cfg, region)...)

	// Auto Scaling Groups
	resources = append(resources, d.discoverAutoScalingGroups(ctx, cfg, region)...)

	// DynamoDB Tables
	resources = append(resources, d.discoverDynamoDBTables(ctx, cfg, region)...)

	// SQS Queues
	resources = append(resources, d.discoverSQSQueues(ctx, cfg, region)...)

	// SNS Topics
	resources = append(resources, d.discoverSNSTopics(ctx, cfg, region)...)

	// API Gateways
	resources = append(resources, d.discoverAPIGateways(ctx, cfg, region)...)

	return resources
}

// discoverEC2Resources discovers all EC2 resources
func (d *ComprehensiveAWSDiscoverer) discoverEC2Resources(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := ec2.NewFromConfig(cfg)

	// Discover EC2 Instances
	instancesResp, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err == nil {
		for _, reservation := range instancesResp.Reservations {
			for _, instance := range reservation.Instances {
				if instance.InstanceId != nil {
					name := getEC2Tag(instance.Tags, "Name")
					if name == "" && instance.InstanceId != nil {
						name = *instance.InstanceId
					}

					resources = append(resources, models.Resource{
						ID:       *instance.InstanceId,
						Name:     name,
						Type:     "aws_instance",
						Provider: "aws",
						Region:   region,
						State:    string(instance.State.Name),
						Tags:     convertEC2Tags(instance.Tags),
						Properties: map[string]interface{}{
							"instance_type": string(instance.InstanceType),
							"state":         string(instance.State.Name),
							"launch_time":   instance.LaunchTime,
							"public_ip":     safeString(instance.PublicIpAddress),
							"private_ip":    safeString(instance.PrivateIpAddress),
						},
					})
				}
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "EC2", ResourceType: "Instances", Count: len(instancesResp.Reservations)}
	}

	// Discover VPCs
	vpcsResp, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err == nil {
		for _, vpc := range vpcsResp.Vpcs {
			if vpc.VpcId != nil {
				name := getEC2Tag(vpc.Tags, "Name")
				if name == "" {
					name = *vpc.VpcId
				}

				resources = append(resources, models.Resource{
					ID:       *vpc.VpcId,
					Name:     name,
					Type:     "aws_vpc",
					Provider: "aws",
					Region:   region,
					State:    string(vpc.State),
					Tags:     convertEC2Tags(vpc.Tags),
					Properties: map[string]interface{}{
						"cidr_block": safeString(vpc.CidrBlock),
						"state":      string(vpc.State),
						"is_default": safeBool(vpc.IsDefault),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "EC2", ResourceType: "VPCs", Count: len(vpcsResp.Vpcs)}
	}

	// Discover Security Groups
	sgsResp, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
	if err == nil {
		for _, sg := range sgsResp.SecurityGroups {
			if sg.GroupId != nil {
				resources = append(resources, models.Resource{
					ID:       *sg.GroupId,
					Name:     safeString(sg.GroupName),
					Type:     "aws_security_group",
					Provider: "aws",
					Region:   region,
					State:    "available",
					Tags:     convertEC2Tags(sg.Tags),
					Properties: map[string]interface{}{
						"description": safeString(sg.Description),
						"vpc_id":      safeString(sg.VpcId),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "EC2", ResourceType: "Security Groups", Count: len(sgsResp.SecurityGroups)}
	}

	// Discover Subnets
	subnetsResp, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{})
	if err == nil {
		for _, subnet := range subnetsResp.Subnets {
			if subnet.SubnetId != nil {
				name := getEC2Tag(subnet.Tags, "Name")
				if name == "" {
					name = *subnet.SubnetId
				}

				resources = append(resources, models.Resource{
					ID:       *subnet.SubnetId,
					Name:     name,
					Type:     "aws_subnet",
					Provider: "aws",
					Region:   region,
					State:    string(subnet.State),
					Tags:     convertEC2Tags(subnet.Tags),
					Properties: map[string]interface{}{
						"vpc_id":            safeString(subnet.VpcId),
						"cidr_block":        safeString(subnet.CidrBlock),
						"availability_zone": safeString(subnet.AvailabilityZone),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "EC2", ResourceType: "Subnets", Count: len(subnetsResp.Subnets)}
	}

	// Discover EBS Volumes
	volumesResp, err := client.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{})
	if err == nil {
		for _, volume := range volumesResp.Volumes {
			if volume.VolumeId != nil {
				name := getEC2Tag(volume.Tags, "Name")
				if name == "" {
					name = *volume.VolumeId
				}

				resources = append(resources, models.Resource{
					ID:       *volume.VolumeId,
					Name:     name,
					Type:     "aws_ebs_volume",
					Provider: "aws",
					Region:   region,
					State:    string(volume.State),
					Tags:     convertEC2Tags(volume.Tags),
					Properties: map[string]interface{}{
						"size":              volume.Size,
						"volume_type":       string(volume.VolumeType),
						"availability_zone": safeString(volume.AvailabilityZone),
						"encrypted":         safeBool(volume.Encrypted),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "EC2", ResourceType: "EBS Volumes", Count: len(volumesResp.Volumes)}
	}

	return resources
}

// discoverRDSResources discovers RDS databases
func (d *ComprehensiveAWSDiscoverer) discoverRDSResources(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := rds.NewFromConfig(cfg)

	// Discover RDS Instances
	resp, err := client.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err == nil {
		for _, db := range resp.DBInstances {
			if db.DBInstanceIdentifier != nil {
				resources = append(resources, models.Resource{
					ID:       *db.DBInstanceIdentifier,
					Name:     *db.DBInstanceIdentifier,
					Type:     "aws_db_instance",
					Provider: "aws",
					Region:   region,
					State:    safeString(db.DBInstanceStatus),
					Properties: map[string]interface{}{
						"engine":         safeString(db.Engine),
						"engine_version": safeString(db.EngineVersion),
						"instance_class": safeString(db.DBInstanceClass),
						"storage":        db.AllocatedStorage,
						"multi_az":       safeBool(db.MultiAZ),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "RDS", ResourceType: "DB Instances", Count: len(resp.DBInstances)}
	}

	// Discover RDS Clusters
	clustersResp, err := client.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{})
	if err == nil {
		for _, cluster := range clustersResp.DBClusters {
			if cluster.DBClusterIdentifier != nil {
				resources = append(resources, models.Resource{
					ID:       *cluster.DBClusterIdentifier,
					Name:     *cluster.DBClusterIdentifier,
					Type:     "aws_rds_cluster",
					Provider: "aws",
					Region:   region,
					State:    safeString(cluster.Status),
					Properties: map[string]interface{}{
						"engine":         safeString(cluster.Engine),
						"engine_version": safeString(cluster.EngineVersion),
						"multi_az":       safeBool(cluster.MultiAZ),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "RDS", ResourceType: "DB Clusters", Count: len(clustersResp.DBClusters)}
	}

	return resources
}

// discoverLambdaFunctions discovers Lambda functions
func (d *ComprehensiveAWSDiscoverer) discoverLambdaFunctions(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := lambda.NewFromConfig(cfg)

	resp, err := client.ListFunctions(ctx, &lambda.ListFunctionsInput{})
	if err == nil {
		for _, function := range resp.Functions {
			if function.FunctionName != nil {
				resources = append(resources, models.Resource{
					ID:       *function.FunctionArn,
					Name:     *function.FunctionName,
					Type:     "aws_lambda_function",
					Provider: "aws",
					Region:   region,
					State:    "available",
					Properties: map[string]interface{}{
						"runtime":     string(function.Runtime),
						"handler":     safeString(function.Handler),
						"memory_size": function.MemorySize,
						"timeout":     function.Timeout,
						"state":       string(function.State),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "Lambda", ResourceType: "Functions", Count: len(resp.Functions)}
	}

	return resources
}

// discoverECSResources discovers ECS clusters and services
func (d *ComprehensiveAWSDiscoverer) discoverECSResources(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := ecs.NewFromConfig(cfg)

	// List clusters
	clustersResp, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
	if err == nil && len(clustersResp.ClusterArns) > 0 {
		// Describe clusters
		descResp, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
			Clusters: clustersResp.ClusterArns,
		})
		if err == nil {
			for _, cluster := range descResp.Clusters {
				if cluster.ClusterArn != nil {
					resources = append(resources, models.Resource{
						ID:       *cluster.ClusterArn,
						Name:     safeString(cluster.ClusterName),
						Type:     "aws_ecs_cluster",
						Provider: "aws",
						Region:   region,
						State:    safeString(cluster.Status),
						Properties: map[string]interface{}{
							"status":                safeString(cluster.Status),
							"running_tasks_count":   cluster.RunningTasksCount,
							"pending_tasks_count":   cluster.PendingTasksCount,
							"active_services_count": cluster.ActiveServicesCount,
							"registered_instances":  cluster.RegisteredContainerInstancesCount,
						},
					})
				}
			}
			d.progress <- AWSDiscoveryProgress{Region: region, Service: "ECS", ResourceType: "Clusters", Count: len(descResp.Clusters)}
		}
	}

	return resources
}

// discoverEKSClusters discovers EKS clusters
func (d *ComprehensiveAWSDiscoverer) discoverEKSClusters(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := eks.NewFromConfig(cfg)

	resp, err := client.ListClusters(ctx, &eks.ListClustersInput{})
	if err == nil {
		for _, clusterName := range resp.Clusters {
			// Describe each cluster for details
			descResp, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
				Name: &clusterName,
			})
			if err == nil && descResp.Cluster != nil {
				cluster := descResp.Cluster
				resources = append(resources, models.Resource{
					ID:       safeString(cluster.Arn),
					Name:     safeString(cluster.Name),
					Type:     "aws_eks_cluster",
					Provider: "aws",
					Region:   region,
					State:    string(cluster.Status),
					Tags:     cluster.Tags,
					Properties: map[string]interface{}{
						"version":          safeString(cluster.Version),
						"status":           string(cluster.Status),
						"platform_version": safeString(cluster.PlatformVersion),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "EKS", ResourceType: "Clusters", Count: len(resp.Clusters)}
	}

	return resources
}

// discoverLoadBalancers discovers ALBs and NLBs
func (d *ComprehensiveAWSDiscoverer) discoverLoadBalancers(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := elasticloadbalancingv2.NewFromConfig(cfg)

	resp, err := client.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	if err == nil {
		for _, lb := range resp.LoadBalancers {
			if lb.LoadBalancerArn != nil {
				resources = append(resources, models.Resource{
					ID:       *lb.LoadBalancerArn,
					Name:     safeString(lb.LoadBalancerName),
					Type:     "aws_lb",
					Provider: "aws",
					Region:   region,
					State:    string(lb.State.Code),
					Properties: map[string]interface{}{
						"type":     string(lb.Type),
						"scheme":   string(lb.Scheme),
						"dns_name": safeString(lb.DNSName),
						"vpc_id":   safeString(lb.VpcId),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "ELB", ResourceType: "Load Balancers", Count: len(resp.LoadBalancers)}
	}

	return resources
}

// discoverElastiCacheResources discovers ElastiCache clusters
func (d *ComprehensiveAWSDiscoverer) discoverElastiCacheResources(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := elasticache.NewFromConfig(cfg)

	resp, err := client.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{})
	if err == nil {
		for _, cluster := range resp.CacheClusters {
			if cluster.CacheClusterId != nil {
				resources = append(resources, models.Resource{
					ID:       *cluster.CacheClusterId,
					Name:     *cluster.CacheClusterId,
					Type:     "aws_elasticache_cluster",
					Provider: "aws",
					Region:   region,
					State:    safeString(cluster.CacheClusterStatus),
					Properties: map[string]interface{}{
						"engine":         safeString(cluster.Engine),
						"engine_version": safeString(cluster.EngineVersion),
						"node_type":      safeString(cluster.CacheNodeType),
						"num_nodes":      cluster.NumCacheNodes,
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "ElastiCache", ResourceType: "Clusters", Count: len(resp.CacheClusters)}
	}

	return resources
}

// discoverAutoScalingGroups discovers Auto Scaling Groups
func (d *ComprehensiveAWSDiscoverer) discoverAutoScalingGroups(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := autoscaling.NewFromConfig(cfg)

	resp, err := client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{})
	if err == nil {
		for _, asg := range resp.AutoScalingGroups {
			if asg.AutoScalingGroupName != nil {
				resources = append(resources, models.Resource{
					ID:       *asg.AutoScalingGroupARN,
					Name:     *asg.AutoScalingGroupName,
					Type:     "aws_autoscaling_group",
					Provider: "aws",
					Region:   region,
					State:    "active",
					Properties: map[string]interface{}{
						"min_size":         asg.MinSize,
						"max_size":         asg.MaxSize,
						"desired_capacity": asg.DesiredCapacity,
						"instances":        len(asg.Instances),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "Auto Scaling", ResourceType: "Groups", Count: len(resp.AutoScalingGroups)}
	}

	return resources
}

// discoverDynamoDBTables discovers DynamoDB tables
func (d *ComprehensiveAWSDiscoverer) discoverDynamoDBTables(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := dynamodb.NewFromConfig(cfg)

	resp, err := client.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err == nil {
		for _, tableName := range resp.TableNames {
			// Describe each table for details
			descResp, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
				TableName: &tableName,
			})
			if err == nil && descResp.Table != nil {
				table := descResp.Table
				resources = append(resources, models.Resource{
					ID:       *table.TableArn,
					Name:     *table.TableName,
					Type:     "aws_dynamodb_table",
					Provider: "aws",
					Region:   region,
					State:    string(table.TableStatus),
					Properties: map[string]interface{}{
						"status":     string(table.TableStatus),
						"item_count": table.ItemCount,
						"size_bytes": table.TableSizeBytes,
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "DynamoDB", ResourceType: "Tables", Count: len(resp.TableNames)}
	}

	return resources
}

// discoverSQSQueues discovers SQS queues
func (d *ComprehensiveAWSDiscoverer) discoverSQSQueues(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := sqs.NewFromConfig(cfg)

	resp, err := client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err == nil && resp.QueueUrls != nil {
		for _, queueURL := range resp.QueueUrls {
			// Extract queue name from URL
			parts := strings.Split(queueURL, "/")
			queueName := parts[len(parts)-1]

			resources = append(resources, models.Resource{
				ID:       queueURL,
				Name:     queueName,
				Type:     "aws_sqs_queue",
				Provider: "aws",
				Region:   region,
				State:    "available",
				Properties: map[string]interface{}{
					"url": queueURL,
				},
			})
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "SQS", ResourceType: "Queues", Count: len(resp.QueueUrls)}
	}

	return resources
}

// discoverSNSTopics discovers SNS topics
func (d *ComprehensiveAWSDiscoverer) discoverSNSTopics(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := sns.NewFromConfig(cfg)

	resp, err := client.ListTopics(ctx, &sns.ListTopicsInput{})
	if err == nil && resp.Topics != nil {
		for _, topic := range resp.Topics {
			if topic.TopicArn != nil {
				// Extract topic name from ARN
				parts := strings.Split(*topic.TopicArn, ":")
				topicName := parts[len(parts)-1]

				resources = append(resources, models.Resource{
					ID:       *topic.TopicArn,
					Name:     topicName,
					Type:     "aws_sns_topic",
					Provider: "aws",
					Region:   region,
					State:    "available",
					Properties: map[string]interface{}{
						"arn": *topic.TopicArn,
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "SNS", ResourceType: "Topics", Count: len(resp.Topics)}
	}

	return resources
}

// discoverAPIGateways discovers API Gateway REST APIs
func (d *ComprehensiveAWSDiscoverer) discoverAPIGateways(ctx context.Context, cfg aws.Config, region string) []models.Resource {
	var resources []models.Resource
	client := apigateway.NewFromConfig(cfg)

	resp, err := client.GetRestApis(ctx, &apigateway.GetRestApisInput{})
	if err == nil && resp.Items != nil {
		for _, api := range resp.Items {
			if api.Id != nil {
				resources = append(resources, models.Resource{
					ID:       *api.Id,
					Name:     safeString(api.Name),
					Type:     "aws_api_gateway_rest_api",
					Provider: "aws",
					Region:   region,
					State:    "available",
					Properties: map[string]interface{}{
						"description": safeString(api.Description),
						"created":     api.CreatedDate,
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Region: region, Service: "API Gateway", ResourceType: "REST APIs", Count: len(resp.Items)}
	}

	return resources
}

// discoverGlobalResources discovers resources that are not region-specific
func (d *ComprehensiveAWSDiscoverer) discoverGlobalResources(ctx context.Context) []models.Resource {
	var resources []models.Resource
	cfg := d.cfg.Copy()
	cfg.Region = "us-east-1" // Global services use us-east-1

	// S3 Buckets (global)
	s3Client := s3.NewFromConfig(cfg)
	bucketsResp, err := s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err == nil && bucketsResp.Buckets != nil {
		for _, bucket := range bucketsResp.Buckets {
			if bucket.Name != nil {
				// Get bucket location
				location := "us-east-1"
				locResp, err := s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
					Bucket: bucket.Name,
				})
				if err == nil && locResp.LocationConstraint != "" {
					location = string(locResp.LocationConstraint)
				}

				resources = append(resources, models.Resource{
					ID:       *bucket.Name,
					Name:     *bucket.Name,
					Type:     "aws_s3_bucket",
					Provider: "aws",
					Region:   location,
					State:    "available",
					Properties: map[string]interface{}{
						"creation_date": bucket.CreationDate,
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Service: "S3", ResourceType: "Buckets", Count: len(bucketsResp.Buckets)}
	}

	// IAM Resources (global)
	iamClient := iam.NewFromConfig(cfg)

	// IAM Users
	usersResp, err := iamClient.ListUsers(ctx, &iam.ListUsersInput{})
	if err == nil && usersResp.Users != nil {
		for _, user := range usersResp.Users {
			if user.UserName != nil {
				resources = append(resources, models.Resource{
					ID:       *user.Arn,
					Name:     *user.UserName,
					Type:     "aws_iam_user",
					Provider: "aws",
					Region:   "global",
					State:    "active",
					Properties: map[string]interface{}{
						"created": user.CreateDate,
						"path":    safeString(user.Path),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Service: "IAM", ResourceType: "Users", Count: len(usersResp.Users)}
	}

	// IAM Roles
	rolesResp, err := iamClient.ListRoles(ctx, &iam.ListRolesInput{})
	if err == nil && rolesResp.Roles != nil {
		for _, role := range rolesResp.Roles {
			if role.RoleName != nil {
				resources = append(resources, models.Resource{
					ID:       *role.Arn,
					Name:     *role.RoleName,
					Type:     "aws_iam_role",
					Provider: "aws",
					Region:   "global",
					State:    "active",
					Properties: map[string]interface{}{
						"created": role.CreateDate,
						"path":    safeString(role.Path),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Service: "IAM", ResourceType: "Roles", Count: len(rolesResp.Roles)}
	}

	// CloudFront Distributions (global)
	cfClient := cloudfront.NewFromConfig(cfg)
	distResp, err := cfClient.ListDistributions(ctx, &cloudfront.ListDistributionsInput{})
	if err == nil && distResp.DistributionList != nil && distResp.DistributionList.Items != nil {
		for _, dist := range distResp.DistributionList.Items {
			if dist.Id != nil {
				resources = append(resources, models.Resource{
					ID:       *dist.Id,
					Name:     *dist.Id,
					Type:     "aws_cloudfront_distribution",
					Provider: "aws",
					Region:   "global",
					State:    safeString(dist.Status),
					Properties: map[string]interface{}{
						"domain_name": safeString(dist.DomainName),
						"enabled":     safeBool(dist.Enabled),
						"status":      safeString(dist.Status),
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Service: "CloudFront", ResourceType: "Distributions", Count: len(distResp.DistributionList.Items)}
	}

	// Route53 Hosted Zones (global)
	r53Client := route53.NewFromConfig(cfg)
	zonesResp, err := r53Client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
	if err == nil && zonesResp.HostedZones != nil {
		for _, zone := range zonesResp.HostedZones {
			if zone.Id != nil {
				// Clean up zone ID
				zoneID := strings.TrimPrefix(*zone.Id, "/hostedzone/")
				resources = append(resources, models.Resource{
					ID:       zoneID,
					Name:     *zone.Name,
					Type:     "aws_route53_zone",
					Provider: "aws",
					Region:   "global",
					State:    "available",
					Properties: map[string]interface{}{
						"private": zone.Config != nil && zone.Config.PrivateZone,
						"comment": zone.Config != nil && zone.Config.Comment != nil,
					},
				})
			}
		}
		d.progress <- AWSDiscoveryProgress{Service: "Route53", ResourceType: "Hosted Zones", Count: len(zonesResp.HostedZones)}
	}

	return resources
}

// Helper functions
func (d *ComprehensiveAWSDiscoverer) getDefaultRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1",
		"eu-north-1", "ap-south-1", "ap-southeast-1", "ap-southeast-2",
		"ap-northeast-1", "ap-northeast-2", "ap-northeast-3",
		"ca-central-1", "sa-east-1",
	}
}

func (d *ComprehensiveAWSDiscoverer) reportProgress() {
	for progress := range d.progress {
		if progress.Region != "" {
			log.Printf("[%s] %s: Discovered %d %s", progress.Region, progress.Service, progress.Count, progress.ResourceType)
		} else {
			log.Printf("[Global] %s: Discovered %d %s", progress.Service, progress.Count, progress.ResourceType)
		}
	}
}

func getEC2Tag(tags []ec2types.Tag, key string) string {
	for _, tag := range tags {
		if tag.Key != nil && *tag.Key == key && tag.Value != nil {
			return *tag.Value
		}
	}
	return ""
}

func convertEC2Tags(tags []ec2types.Tag) map[string]string {
	result := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			result[*tag.Key] = *tag.Value
		}
	}
	return result
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func safeBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
