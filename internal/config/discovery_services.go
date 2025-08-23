package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// ServiceConfig represents configuration for a discovery service
type ServiceConfig struct {
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`
	Priority    int               `json:"priority"`
	Regions     []string          `json:"regions"`
	Parameters  map[string]string `json:"parameters"`
	Description string            `json:"description"`
}

// ProviderServices represents services for a cloud provider
type ProviderServices struct {
	Provider string                    `json:"provider"`
	Services map[string]*ServiceConfig `json:"services"`
}

// DiscoveryServicesConfig manages service configurations
type DiscoveryServicesConfig struct {
	Providers  map[string]*ProviderServices `json:"providers"`
	mu         sync.RWMutex
	configPath string
}

// NewDiscoveryServicesConfig creates a new service configuration manager
func NewDiscoveryServicesConfig(configPath string) *DiscoveryServicesConfig {
	config := &DiscoveryServicesConfig{
		Providers:  make(map[string]*ProviderServices),
		configPath: configPath,
	}

	// Load default configuration
	config.loadDefaults()

	// Load from file if exists
	if err := config.LoadFromFile(); err != nil {
		fmt.Printf("Warning: Could not load service config from %s: %v\n", configPath, err)
	}

	return config
}

// loadDefaults loads default service configurations
func (dsc *DiscoveryServicesConfig) loadDefaults() {
	dsc.mu.Lock()
	defer dsc.mu.Unlock()

	// AWS Services - Core Infrastructure
	awsServices := &ProviderServices{
		Provider: "aws",
		Services: map[string]*ServiceConfig{
			"ec2": {
				Name:        "ec2",
				Enabled:     true,
				Priority:    1,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Elastic Compute Cloud instances",
			},
			"vpc": {
				Name:        "vpc",
				Enabled:     true,
				Priority:    2,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Virtual Private Cloud",
			},
			"s3": {
				Name:        "s3",
				Enabled:     true,
				Priority:    3,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Simple Storage Service",
			},
			"rds": {
				Name:        "rds",
				Enabled:     true,
				Priority:    4,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Relational Database Service",
			},
			"lambda": {
				Name:        "lambda",
				Enabled:     true,
				Priority:    5,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Lambda functions",
			},
			"ecs": {
				Name:        "ecs",
				Enabled:     true,
				Priority:    6,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Elastic Container Service",
			},
			"eks": {
				Name:        "eks",
				Enabled:     true,
				Priority:    7,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Elastic Kubernetes Service",
			},
			"dynamodb": {
				Name:        "dynamodb",
				Enabled:     true,
				Priority:    8,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon DynamoDB",
			},
			"cloudwatch": {
				Name:        "cloudwatch",
				Enabled:     true,
				Priority:    9,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon CloudWatch",
			},
			"iam": {
				Name:        "iam",
				Enabled:     true,
				Priority:    10,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Identity and Access Management",
			},
			// Security & Compliance Services
			"waf": {
				Name:        "waf",
				Enabled:     true,
				Priority:    11,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Web Application Firewall",
			},
			"shield": {
				Name:        "shield",
				Enabled:     true,
				Priority:    12,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Shield DDoS Protection",
			},
			"config": {
				Name:        "config",
				Enabled:     true,
				Priority:    13,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Config",
			},
			"guardduty": {
				Name:        "guardduty",
				Enabled:     true,
				Priority:    14,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS GuardDuty",
			},
			"cloudtrail": {
				Name:        "cloudtrail",
				Enabled:     true,
				Priority:    15,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS CloudTrail",
			},
			"secretsmanager": {
				Name:        "secretsmanager",
				Enabled:     true,
				Priority:    16,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Secrets Manager",
			},
			"kms": {
				Name:        "kms",
				Enabled:     true,
				Priority:    17,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Key Management Service",
			},
			// Data & Analytics Services
			"glue": {
				Name:        "glue",
				Enabled:     true,
				Priority:    18,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Glue",
			},
			"redshift": {
				Name:        "redshift",
				Enabled:     true,
				Priority:    19,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Redshift",
			},
			"elasticsearch": {
				Name:        "elasticsearch",
				Enabled:     true,
				Priority:    20,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Elasticsearch Service",
			},
			"athena": {
				Name:        "athena",
				Enabled:     true,
				Priority:    21,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Athena",
			},
			"kinesis": {
				Name:        "kinesis",
				Enabled:     true,
				Priority:    22,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Kinesis",
			},
			// CI/CD Services
			"codebuild": {
				Name:        "codebuild",
				Enabled:     true,
				Priority:    23,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS CodeBuild",
			},
			"codepipeline": {
				Name:        "codepipeline",
				Enabled:     true,
				Priority:    24,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS CodePipeline",
			},
			"codedeploy": {
				Name:        "codedeploy",
				Enabled:     true,
				Priority:    25,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS CodeDeploy",
			},
			// Additional Services
			"batch": {
				Name:        "batch",
				Enabled:     true,
				Priority:    26,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Batch",
			},
			"fargate": {
				Name:        "fargate",
				Enabled:     true,
				Priority:    27,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Fargate",
			},
			"emr": {
				Name:        "emr",
				Enabled:     true,
				Priority:    28,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon EMR",
			},
			"neptune": {
				Name:        "neptune",
				Enabled:     true,
				Priority:    29,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Neptune",
			},
			"documentdb": {
				Name:        "documentdb",
				Enabled:     true,
				Priority:    30,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon DocumentDB",
			},
			"msk": {
				Name:        "msk",
				Enabled:     true,
				Priority:    31,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon MSK",
			},
			"mq": {
				Name:        "mq",
				Enabled:     true,
				Priority:    32,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon MQ",
			},
			"transfer": {
				Name:        "transfer",
				Enabled:     true,
				Priority:    33,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Transfer",
			},
			"directconnect": {
				Name:        "directconnect",
				Enabled:     true,
				Priority:    34,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Direct Connect",
			},
			"vpn": {
				Name:        "vpn",
				Enabled:     true,
				Priority:    35,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS VPN",
			},
			"transitgateway": {
				Name:        "transitgateway",
				Enabled:     true,
				Priority:    36,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Transit Gateway",
			},
			"appmesh": {
				Name:        "appmesh",
				Enabled:     true,
				Priority:    37,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS App Mesh",
			},
			"xray": {
				Name:        "xray",
				Enabled:     true,
				Priority:    38,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS X-Ray",
			},
			"cloud9": {
				Name:        "cloud9",
				Enabled:     true,
				Priority:    39,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Cloud9",
			},
			"codestar": {
				Name:        "codestar",
				Enabled:     true,
				Priority:    40,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS CodeStar",
			},
			"amplify": {
				Name:        "amplify",
				Enabled:     true,
				Priority:    41,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Amplify",
			},
			"quicksight": {
				Name:        "quicksight",
				Enabled:     true,
				Priority:    42,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon QuickSight",
			},
			"datasync": {
				Name:        "datasync",
				Enabled:     true,
				Priority:    43,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS DataSync",
			},
			"storagegateway": {
				Name:        "storagegateway",
				Enabled:     true,
				Priority:    44,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Storage Gateway",
			},
			"backup": {
				Name:        "backup",
				Enabled:     true,
				Priority:    45,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Backup",
			},
			"fsx": {
				Name:        "fsx",
				Enabled:     true,
				Priority:    46,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon FSx",
			},
			"workspaces": {
				Name:        "workspaces",
				Enabled:     true,
				Priority:    47,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon WorkSpaces",
			},
			"appstream": {
				Name:        "appstream",
				Enabled:     true,
				Priority:    48,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon AppStream",
			},
			"route53": {
				Name:        "route53",
				Enabled:     true,
				Priority:    49,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon Route 53",
			},
			"cloudformation": {
				Name:        "cloudformation",
				Enabled:     true,
				Priority:    50,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS CloudFormation",
			},
			"elastiCache": {
				Name:        "elastiCache",
				Enabled:     true,
				Priority:    51,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon ElastiCache",
			},
			"sqs": {
				Name:        "sqs",
				Enabled:     true,
				Priority:    52,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon SQS",
			},
			"sns": {
				Name:        "sns",
				Enabled:     true,
				Priority:    53,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "Amazon SNS",
			},
			"autoscaling": {
				Name:        "autoscaling",
				Enabled:     true,
				Priority:    54,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Auto Scaling",
			},
			"stepfunctions": {
				Name:        "stepfunctions",
				Enabled:     true,
				Priority:    55,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Step Functions",
			},
			"systemsmanager": {
				Name:        "systemsmanager",
				Enabled:     true,
				Priority:    56,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Systems Manager",
			},
			// Additional AWS Services - Security & Compliance
			"macie": {
				Name:        "macie",
				Enabled:     true,
				Priority:    57,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Macie",
			},
			"securityhub": {
				Name:        "securityhub",
				Enabled:     true,
				Priority:    58,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Security Hub",
			},
			"detective": {
				Name:        "detective",
				Enabled:     true,
				Priority:    59,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Detective",
			},
			"inspector": {
				Name:        "inspector",
				Enabled:     true,
				Priority:    60,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Inspector",
			},
			"artifact": {
				Name:        "artifact",
				Enabled:     true,
				Priority:    61,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Artifact",
			},
			// Additional AWS Services - Networking
			"subnets": {
				Name:        "subnets",
				Enabled:     true,
				Priority:    62,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Subnets",
			},
			"securitygroups": {
				Name:        "securitygroups",
				Enabled:     true,
				Priority:    63,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Security Groups",
			},
			"internetgateway": {
				Name:        "internetgateway",
				Enabled:     true,
				Priority:    64,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Internet Gateway",
			},
			"natgateway": {
				Name:        "natgateway",
				Enabled:     true,
				Priority:    65,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS NAT Gateway",
			},
			"vpngateway": {
				Name:        "vpngateway",
				Enabled:     true,
				Priority:    66,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS VPN Gateway",
			},
			"routetables": {
				Name:        "routetables",
				Enabled:     true,
				Priority:    67,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Route Tables",
			},
			"networkacls": {
				Name:        "networkacls",
				Enabled:     true,
				Priority:    68,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Network ACLs",
			},
			"elasticip": {
				Name:        "elasticip",
				Enabled:     true,
				Priority:    69,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Elastic IPs",
			},
			"vpcendpoints": {
				Name:        "vpcendpoints",
				Enabled:     true,
				Priority:    70,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS VPC Endpoints",
			},
			"vpcflowlogs": {
				Name:        "vpcflowlogs",
				Enabled:     true,
				Priority:    71,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS VPC Flow Logs",
			},
			// Additional AWS Services - CDN & Content Delivery
			"cloudfront": {
				Name:        "cloudfront",
				Enabled:     true,
				Priority:    72,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS CloudFront",
			},
			"certificatemanager": {
				Name:        "certificatemanager",
				Enabled:     true,
				Priority:    73,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Certificate Manager",
			},
			// Additional AWS Services - Organization & Governance
			"organizations": {
				Name:        "organizations",
				Enabled:     true,
				Priority:    74,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Organizations",
			},
			"controltower": {
				Name:        "controltower",
				Enabled:     true,
				Priority:    75,
				Regions:     []string{"us-east-1", "us-west-2", "eu-west-1"},
				Parameters:  map[string]string{},
				Description: "AWS Control Tower",
			},
		},
	}

	dsc.Providers["aws"] = awsServices

	// Azure Services
	azureServices := &ProviderServices{
		Provider: "azure",
		Services: map[string]*ServiceConfig{
			"vm": {
				Name:        "vm",
				Enabled:     true,
				Priority:    1,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Virtual Machines",
			},
			"vnet": {
				Name:        "vnet",
				Enabled:     true,
				Priority:    2,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Virtual Networks",
			},
			"storage": {
				Name:        "storage",
				Enabled:     true,
				Priority:    3,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Storage Accounts",
			},
			"sql": {
				Name:        "sql",
				Enabled:     true,
				Priority:    4,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure SQL Database",
			},
			"function": {
				Name:        "function",
				Enabled:     true,
				Priority:    5,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Functions",
			},
			"webapp": {
				Name:        "webapp",
				Enabled:     true,
				Priority:    6,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Web Apps",
			},
			"aks": {
				Name:        "aks",
				Enabled:     true,
				Priority:    7,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Kubernetes Service",
			},
			"cosmosdb": {
				Name:        "cosmosdb",
				Enabled:     true,
				Priority:    8,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Cosmos DB",
			},
			"monitor": {
				Name:        "monitor",
				Enabled:     true,
				Priority:    9,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Monitor",
			},
			"keyvault": {
				Name:        "keyvault",
				Enabled:     true,
				Priority:    10,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Key Vault",
			},
			// Additional Azure Services
			"containerinstances": {
				Name:        "containerinstances",
				Enabled:     true,
				Priority:    11,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Container Instances",
			},
			"containerregistry": {
				Name:        "containerregistry",
				Enabled:     true,
				Priority:    12,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Container Registry",
			},
			"servicefabric": {
				Name:        "servicefabric",
				Enabled:     true,
				Priority:    13,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Service Fabric",
			},
			"springcloud": {
				Name:        "springcloud",
				Enabled:     true,
				Priority:    14,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Spring Cloud",
			},
			"apimanagement": {
				Name:        "apimanagement",
				Enabled:     true,
				Priority:    15,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure API Management",
			},
			"eventgrid": {
				Name:        "eventgrid",
				Enabled:     true,
				Priority:    16,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Event Grid",
			},
			"streamanalytics": {
				Name:        "streamanalytics",
				Enabled:     true,
				Priority:    17,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Stream Analytics",
			},
			"datalakestorage": {
				Name:        "datalakestorage",
				Enabled:     true,
				Priority:    18,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Data Lake Storage",
			},
			"hdinsight": {
				Name:        "hdinsight",
				Enabled:     true,
				Priority:    19,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure HDInsight",
			},
			"databricks": {
				Name:        "databricks",
				Enabled:     true,
				Priority:    20,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Databricks",
			},
			"machinelearning": {
				Name:        "machinelearning",
				Enabled:     true,
				Priority:    21,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Machine Learning",
			},
			"cognitiveservices": {
				Name:        "cognitiveservices",
				Enabled:     true,
				Priority:    22,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Cognitive Services",
			},
			"botservice": {
				Name:        "botservice",
				Enabled:     true,
				Priority:    23,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Bot Service",
			},
			"signalr": {
				Name:        "signalr",
				Enabled:     true,
				Priority:    24,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure SignalR",
			},
			"mediaservices": {
				Name:        "mediaservices",
				Enabled:     true,
				Priority:    25,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Media Services",
			},
			"videoindexer": {
				Name:        "videoindexer",
				Enabled:     true,
				Priority:    26,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Video Indexer",
			},
			"maps": {
				Name:        "maps",
				Enabled:     true,
				Priority:    27,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Maps",
			},
			"timeseriesinsights": {
				Name:        "timeseriesinsights",
				Enabled:     true,
				Priority:    28,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Time Series Insights",
			},
			"digitaltwins": {
				Name:        "digitaltwins",
				Enabled:     true,
				Priority:    29,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Digital Twins",
			},
			"dataexplorer": {
				Name:        "dataexplorer",
				Enabled:     true,
				Priority:    30,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Data Explorer",
			},
			"datashare": {
				Name:        "datashare",
				Enabled:     true,
				Priority:    31,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Data Share",
			},
			"purview": {
				Name:        "purview",
				Enabled:     true,
				Priority:    32,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Purview",
			},
			"datafactoryv2": {
				Name:        "datafactoryv2",
				Enabled:     true,
				Priority:    33,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Data Factory V2",
			},
			"datalakeanalytics": {
				Name:        "datalakeanalytics",
				Enabled:     true,
				Priority:    34,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Data Lake Analytics",
			},
			"datalakestore": {
				Name:        "datalakestore",
				Enabled:     true,
				Priority:    35,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Data Lake Store",
			},
			"datacatalog": {
				Name:        "datacatalog",
				Enabled:     true,
				Priority:    36,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Data Catalog",
			},
			"databox": {
				Name:        "databox",
				Enabled:     true,
				Priority:    37,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Data Box",
			},
			"logicapps": {
				Name:        "logicapps",
				Enabled:     true,
				Priority:    38,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Logic Apps",
			},
			"eventhubs": {
				Name:        "eventhubs",
				Enabled:     true,
				Priority:    39,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Event Hubs",
			},
			"servicebus": {
				Name:        "servicebus",
				Enabled:     true,
				Priority:    40,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Service Bus",
			},
			"datafactory": {
				Name:        "datafactory",
				Enabled:     true,
				Priority:    41,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Data Factory",
			},
			"synapseanalytics": {
				Name:        "synapseanalytics",
				Enabled:     true,
				Priority:    42,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Synapse Analytics",
			},
			"applicationinsights": {
				Name:        "applicationinsights",
				Enabled:     true,
				Priority:    43,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Application Insights",
			},
			"policy": {
				Name:        "policy",
				Enabled:     true,
				Priority:    44,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Policy",
			},
			"bastion": {
				Name:        "bastion",
				Enabled:     true,
				Priority:    45,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Bastion",
			},
			"loadbalancer": {
				Name:        "loadbalancer",
				Enabled:     true,
				Priority:    46,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Load Balancer",
			},
			"resourcegroup": {
				Name:        "resourcegroup",
				Enabled:     true,
				Priority:    47,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Resource Groups",
			},
			// Additional Azure Services - Networking
			"networkinterfaces": {
				Name:        "networkinterfaces",
				Enabled:     true,
				Priority:    48,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Network Interfaces",
			},
			"publicipaddresses": {
				Name:        "publicipaddresses",
				Enabled:     true,
				Priority:    49,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Public IP Addresses",
			},
			"vpngateways": {
				Name:        "vpngateways",
				Enabled:     true,
				Priority:    50,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure VPN Gateways",
			},
			"expressroute": {
				Name:        "expressroute",
				Enabled:     true,
				Priority:    51,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure ExpressRoute",
			},
			"applicationgateways": {
				Name:        "applicationgateways",
				Enabled:     true,
				Priority:    52,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Application Gateways",
			},
			"frontdoor": {
				Name:        "frontdoor",
				Enabled:     true,
				Priority:    53,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Front Door",
			},
			"cdnprofiles": {
				Name:        "cdnprofiles",
				Enabled:     true,
				Priority:    54,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure CDN Profiles",
			},
			"routetables": {
				Name:        "routetables",
				Enabled:     true,
				Priority:    55,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Route Tables",
			},
			"networksecuritygroups": {
				Name:        "networksecuritygroups",
				Enabled:     true,
				Priority:    56,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Network Security Groups",
			},
			"firewalls": {
				Name:        "firewalls",
				Enabled:     true,
				Priority:    57,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Firewalls",
			},
			"bastionhosts": {
				Name:        "bastionhosts",
				Enabled:     true,
				Priority:    58,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Bastion Hosts",
			},
			// Additional Azure Services - Security & Compliance
			"securitycenter": {
				Name:        "securitycenter",
				Enabled:     true,
				Priority:    59,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Security Center",
			},
			"sentinel": {
				Name:        "sentinel",
				Enabled:     true,
				Priority:    60,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Sentinel",
			},
			"defender": {
				Name:        "defender",
				Enabled:     true,
				Priority:    61,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Defender",
			},
			"lighthouse": {
				Name:        "lighthouse",
				Enabled:     true,
				Priority:    62,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Lighthouse",
			},
			"privilegedidentitymanagement": {
				Name:        "privilegedidentitymanagement",
				Enabled:     true,
				Priority:    63,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Privileged Identity Management",
			},
			"conditionalaccess": {
				Name:        "conditionalaccess",
				Enabled:     true,
				Priority:    64,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Conditional Access",
			},
			"informationprotection": {
				Name:        "informationprotection",
				Enabled:     true,
				Priority:    65,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Information Protection",
			},
			// Additional Azure Services - Data & Analytics
			"rediscache": {
				Name:        "rediscache",
				Enabled:     true,
				Priority:    66,
				Regions:     []string{}, // Discovered dynamically
				Parameters:  map[string]string{},
				Description: "Azure Redis Cache",
			},
		},
	}

	dsc.Providers["azure"] = azureServices

	// GCP Services
	gcpServices := &ProviderServices{
		Provider: "gcp",
		Services: map[string]*ServiceConfig{
			"compute": {
				Name:        "compute",
				Enabled:     true,
				Priority:    1,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Compute Engine",
			},
			"network": {
				Name:        "network",
				Enabled:     true,
				Priority:    2,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Networking",
			},
			"storage": {
				Name:        "storage",
				Enabled:     true,
				Priority:    3,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Storage",
			},
			"sql": {
				Name:        "sql",
				Enabled:     true,
				Priority:    4,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud SQL",
			},
			"function": {
				Name:        "function",
				Enabled:     true,
				Priority:    5,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Functions",
			},
			"run": {
				Name:        "run",
				Enabled:     true,
				Priority:    6,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Run",
			},
			"gke": {
				Name:        "gke",
				Enabled:     true,
				Priority:    7,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Kubernetes Engine",
			},
			"bigquery": {
				Name:        "bigquery",
				Enabled:     true,
				Priority:    8,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google BigQuery",
			},
			"monitoring": {
				Name:        "monitoring",
				Enabled:     true,
				Priority:    9,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Monitoring",
			},
			"iam": {
				Name:        "iam",
				Enabled:     true,
				Priority:    10,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud IAM",
			},
			// Additional GCP Services
			"cloudbuild": {
				Name:        "cloudbuild",
				Enabled:     true,
				Priority:    11,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Build",
			},
			"pubsub": {
				Name:        "pubsub",
				Enabled:     true,
				Priority:    12,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Pub/Sub",
			},
			"spanner": {
				Name:        "spanner",
				Enabled:     true,
				Priority:    13,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Spanner",
			},
			"firestore": {
				Name:        "firestore",
				Enabled:     true,
				Priority:    14,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Firestore",
			},
			"armor": {
				Name:        "armor",
				Enabled:     true,
				Priority:    15,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Armor",
			},
			"logging": {
				Name:        "logging",
				Enabled:     true,
				Priority:    16,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Logging",
			},
			"tasks": {
				Name:        "tasks",
				Enabled:     true,
				Priority:    17,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Tasks",
			},
			"scheduler": {
				Name:        "scheduler",
				Enabled:     true,
				Priority:    18,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Scheduler",
			},
			"dns": {
				Name:        "dns",
				Enabled:     true,
				Priority:    19,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud DNS",
			},
			"cdn": {
				Name:        "cdn",
				Enabled:     true,
				Priority:    20,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud CDN",
			},
			"loadbalancing": {
				Name:        "loadbalancing",
				Enabled:     true,
				Priority:    21,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Load Balancing",
			},
			"nat": {
				Name:        "nat",
				Enabled:     true,
				Priority:    22,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud NAT",
			},
			"router": {
				Name:        "router",
				Enabled:     true,
				Priority:    23,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Router",
			},
			"vpn": {
				Name:        "vpn",
				Enabled:     true,
				Priority:    24,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud VPN",
			},
			"interconnect": {
				Name:        "interconnect",
				Enabled:     true,
				Priority:    25,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Interconnect",
			},
			"kms": {
				Name:        "kms",
				Enabled:     true,
				Priority:    26,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud KMS",
			},
			"resourcemanager": {
				Name:        "resourcemanager",
				Enabled:     true,
				Priority:    27,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Resource Manager",
			},
			"billing": {
				Name:        "billing",
				Enabled:     true,
				Priority:    28,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Billing",
			},
			"trace": {
				Name:        "trace",
				Enabled:     true,
				Priority:    29,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Trace",
			},
			"debugger": {
				Name:        "debugger",
				Enabled:     true,
				Priority:    30,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Debugger",
			},
			"profiler": {
				Name:        "profiler",
				Enabled:     true,
				Priority:    31,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Profiler",
			},
			"errorreporting": {
				Name:        "errorreporting",
				Enabled:     true,
				Priority:    32,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Error Reporting",
			},
			"dataflow": {
				Name:        "dataflow",
				Enabled:     true,
				Priority:    33,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Dataflow",
			},
			"dataproc": {
				Name:        "dataproc",
				Enabled:     true,
				Priority:    34,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Dataproc",
			},
			"composer": {
				Name:        "composer",
				Enabled:     true,
				Priority:    35,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Composer",
			},
			"datacatalog": {
				Name:        "datacatalog",
				Enabled:     true,
				Priority:    36,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Data Catalog",
			},
			"datafusion": {
				Name:        "datafusion",
				Enabled:     true,
				Priority:    37,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Data Fusion",
			},
			"datalabeling": {
				Name:        "datalabeling",
				Enabled:     true,
				Priority:    38,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Data Labeling",
			},
			"automl": {
				Name:        "automl",
				Enabled:     true,
				Priority:    39,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud AutoML",
			},
			"vertexai": {
				Name:        "vertexai",
				Enabled:     true,
				Priority:    40,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Vertex AI",
			},
			"clouddeploy": {
				Name:        "clouddeploy",
				Enabled:     true,
				Priority:    41,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Deploy",
			},
			// Additional GCP Services - Networking
			"subnets": {
				Name:        "subnets",
				Enabled:     true,
				Priority:    42,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Subnets",
			},
			"firewallrules": {
				Name:        "firewallrules",
				Enabled:     true,
				Priority:    43,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Firewall Rules",
			},
			"loadbalancers": {
				Name:        "loadbalancers",
				Enabled:     true,
				Priority:    44,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Load Balancers",
			},
			"vpngateways": {
				Name:        "vpngateways",
				Enabled:     true,
				Priority:    45,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud VPN Gateways",
			},
			"cloudrouters": {
				Name:        "cloudrouters",
				Enabled:     true,
				Priority:    46,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud Routers",
			},
			"cloudnat": {
				Name:        "cloudnat",
				Enabled:     true,
				Priority:    47,
				Regions:     []string{"us-central1", "us-west1", "europe-west1"},
				Parameters:  map[string]string{},
				Description: "Google Cloud NAT",
			},
		},
	}

	// DigitalOcean Services
	digitalOceanServices := &ProviderServices{
		Provider: "digitalocean",
		Services: map[string]*ServiceConfig{
			"droplets": {
				Name:        "droplets",
				Enabled:     true,
				Priority:    1,
				Regions:     []string{"nyc1", "sfo2", "fra1"},
				Parameters:  map[string]string{},
				Description: "DigitalOcean Droplets",
			},
			"vpcs": {
				Name:        "vpcs",
				Enabled:     true,
				Priority:    2,
				Regions:     []string{"nyc1", "sfo2", "fra1"},
				Parameters:  map[string]string{},
				Description: "DigitalOcean VPCs",
			},
			"spaces": {
				Name:        "spaces",
				Enabled:     true,
				Priority:    3,
				Regions:     []string{"nyc1", "sfo2", "fra1"},
				Parameters:  map[string]string{},
				Description: "DigitalOcean Spaces",
			},
			"loadbalancers": {
				Name:        "loadbalancers",
				Enabled:     true,
				Priority:    4,
				Regions:     []string{"nyc1", "sfo2", "fra1"},
				Parameters:  map[string]string{},
				Description: "DigitalOcean Load Balancers",
			},
			"databases": {
				Name:        "databases",
				Enabled:     true,
				Priority:    5,
				Regions:     []string{"nyc1", "sfo2", "fra1"},
				Parameters:  map[string]string{},
				Description: "DigitalOcean Databases",
			},
			"kubernetes": {
				Name:        "kubernetes",
				Enabled:     true,
				Priority:    6,
				Regions:     []string{"nyc1", "sfo2", "fra1"},
				Parameters:  map[string]string{},
				Description: "DigitalOcean Kubernetes",
			},
			"containerregistry": {
				Name:        "containerregistry",
				Enabled:     true,
				Priority:    7,
				Regions:     []string{"nyc1", "sfo2", "fra1"},
				Parameters:  map[string]string{},
				Description: "DigitalOcean Container Registry",
			},
			"cdn": {
				Name:        "cdn",
				Enabled:     true,
				Priority:    8,
				Regions:     []string{"nyc1", "sfo2", "fra1"},
				Parameters:  map[string]string{},
				Description: "DigitalOcean CDN",
			},
			"monitoring": {
				Name:        "monitoring",
				Enabled:     true,
				Priority:    9,
				Regions:     []string{"nyc1", "sfo2", "fra1"},
				Parameters:  map[string]string{},
				Description: "DigitalOcean Monitoring",
			},
			"firewalls": {
				Name:        "firewalls",
				Enabled:     true,
				Priority:    10,
				Regions:     []string{"nyc1", "sfo2", "fra1"},
				Parameters:  map[string]string{},
				Description: "DigitalOcean Firewalls",
			},
		},
	}

	dsc.Providers["gcp"] = gcpServices
	dsc.Providers["digitalocean"] = digitalOceanServices
}

// LoadFromFile loads service configuration from file
func (dsc *DiscoveryServicesConfig) LoadFromFile() error {
	if dsc.configPath == "" {
		return fmt.Errorf("no config path specified")
	}

	data, err := os.ReadFile(dsc.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	dsc.mu.Lock()
	defer dsc.mu.Unlock()

	if err := json.Unmarshal(data, &dsc.Providers); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// SaveToFile saves service configuration to file
func (dsc *DiscoveryServicesConfig) SaveToFile() error {
	if dsc.configPath == "" {
		return fmt.Errorf("no config path specified")
	}

	dsc.mu.RLock()
	defer dsc.mu.RUnlock()

	data, err := json.MarshalIndent(dsc.Providers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(dsc.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetEnabledServices returns enabled services for a provider
func (dsc *DiscoveryServicesConfig) GetEnabledServices(provider string) []string {
	dsc.mu.RLock()
	defer dsc.mu.RUnlock()

	providerServices, exists := dsc.Providers[provider]
	if !exists {
		return []string{}
	}

	var enabledServices []string
	for serviceName, serviceConfig := range providerServices.Services {
		if serviceConfig.Enabled {
			enabledServices = append(enabledServices, serviceName)
		}
	}

	return enabledServices
}

// GetServiceConfig returns configuration for a specific service
func (dsc *DiscoveryServicesConfig) GetServiceConfig(provider, service string) (*ServiceConfig, error) {
	dsc.mu.RLock()
	defer dsc.mu.RUnlock()

	providerServices, exists := dsc.Providers[provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", provider)
	}

	serviceConfig, exists := providerServices.Services[service]
	if !exists {
		return nil, fmt.Errorf("service %s not found for provider %s", service, provider)
	}

	return serviceConfig, nil
}

// EnableService enables a service for discovery
func (dsc *DiscoveryServicesConfig) EnableService(provider, service string) error {
	dsc.mu.Lock()
	defer dsc.mu.Unlock()

	providerServices, exists := dsc.Providers[provider]
	if !exists {
		return fmt.Errorf("provider %s not found", provider)
	}

	serviceConfig, exists := providerServices.Services[service]
	if !exists {
		return fmt.Errorf("service %s not found for provider %s", service, provider)
	}

	serviceConfig.Enabled = true
	return nil
}

// DisableService disables a service for discovery
func (dsc *DiscoveryServicesConfig) DisableService(provider, service string) error {
	dsc.mu.Lock()
	defer dsc.mu.Unlock()

	providerServices, exists := dsc.Providers[provider]
	if !exists {
		return fmt.Errorf("provider %s not found", provider)
	}

	serviceConfig, exists := providerServices.Services[service]
	if !exists {
		return fmt.Errorf("service %s not found for provider %s", service, provider)
	}

	serviceConfig.Enabled = false
	return nil
}

// GetServicesMap returns a map of provider to enabled services
func (dsc *DiscoveryServicesConfig) GetServicesMap() map[string][]string {
	dsc.mu.RLock()
	defer dsc.mu.RUnlock()

	servicesMap := make(map[string][]string)

	for provider, providerServices := range dsc.Providers {
		var enabledServices []string
		for serviceName, serviceConfig := range providerServices.Services {
			if serviceConfig.Enabled {
				enabledServices = append(enabledServices, serviceName)
			}
		}
		servicesMap[provider] = enabledServices
	}

	return servicesMap
}
