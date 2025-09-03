// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/catherinevee/driftmgr/internal/providers"
	awsprovider "github.com/catherinevee/driftmgr/internal/providers/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LocalStack configuration
const (
	localstackEndpoint = "http://localhost:4566"
	testRegion         = "us-east-1"
	testBucket         = "test-drift-bucket"
	testVPCCIDR        = "10.0.0.0/16"
)

// TestWithLocalStack runs integration tests against LocalStack
func TestWithLocalStack(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TESTS=true to run.")
	}

	// Check if LocalStack is running
	if !isLocalStackRunning() {
		t.Skip("LocalStack is not running. Start it with: docker-compose up -d localstack")
	}

	// Create AWS configuration for LocalStack
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(testRegion),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           localstackEndpoint,
					SigningRegion: testRegion,
				}, nil
			})),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     "test",
				SecretAccessKey: "test",
				SessionToken:    "",
				Source:          "LocalStackTestCredentials",
			}, nil
		})),
	)
	require.NoError(t, err)

	// Run test suites
	t.Run("S3Operations", func(t *testing.T) {
		testS3Operations(t, cfg)
	})

	t.Run("EC2Operations", func(t *testing.T) {
		testEC2Operations(t, cfg)
	})

	t.Run("DriftDetection", func(t *testing.T) {
		testDriftDetection(t, cfg)
	})

	t.Run("StateFileOperations", func(t *testing.T) {
		testStateFileOperations(t, cfg)
	})
}

func testS3Operations(t *testing.T, cfg aws.Config) {
	ctx := context.Background()
	client := s3.NewFromConfig(cfg)

	// Create bucket
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(testBucket),
	})
	require.NoError(t, err)

	// List buckets
	result, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	require.NoError(t, err)
	
	found := false
	for _, bucket := range result.Buckets {
		if *bucket.Name == testBucket {
			found = true
			break
		}
	}
	assert.True(t, found, "Created bucket should be in list")

	// Upload test state file
	testStateContent := `{
		"version": 4,
		"terraform_version": "1.0.0",
		"serial": 1,
		"lineage": "test-lineage",
		"outputs": {},
		"resources": [
			{
				"mode": "managed",
				"type": "aws_s3_bucket",
				"name": "test",
				"provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
				"instances": [
					{
						"attributes": {
							"id": "test-drift-bucket",
							"bucket": "test-drift-bucket",
							"region": "us-east-1"
						}
					}
				]
			}
		]
	}`

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(testBucket),
		Key:    aws.String("terraform.tfstate"),
		Body:   strings.NewReader(testStateContent),
	})
	require.NoError(t, err)

	// Verify object exists
	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(testBucket),
		Key:    aws.String("terraform.tfstate"),
	})
	require.NoError(t, err)

	// Cleanup
	defer func() {
		// Delete object
		_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(testBucket),
			Key:    aws.String("terraform.tfstate"),
		})

		// Delete bucket
		_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{
			Bucket: aws.String(testBucket),
		})
	}()
}

func testEC2Operations(t *testing.T, cfg aws.Config) {
	ctx := context.Background()
	client := ec2.NewFromConfig(cfg)

	// Create VPC
	vpcResult, err := client.CreateVpc(ctx, &ec2.CreateVpcInput{
		CidrBlock: aws.String(testVPCCIDR),
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags: []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("test-vpc"),
					},
					{
						Key:   aws.String("ManagedBy"),
						Value: aws.String("DriftMgr"),
					},
				},
			},
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, vpcResult.Vpc)

	vpcID := *vpcResult.Vpc.VpcId

	// Create Security Group
	sgResult, err := client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String("test-sg"),
		Description: aws.String("Test security group for DriftMgr"),
		VpcId:       aws.String(vpcID),
	})
	require.NoError(t, err)
	assert.NotNil(t, sgResult.GroupId)

	sgID := *sgResult.GroupId

	// Add ingress rule
	_, err = client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(sgID),
		IpPermissions: []types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(80),
				ToPort:     aws.Int32(80),
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("Allow HTTP"),
					},
				},
			},
		},
	})
	require.NoError(t, err)

	// Verify resources exist
	vpcs, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	})
	require.NoError(t, err)
	assert.Len(t, vpcs.Vpcs, 1)

	sgs, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{sgID},
	})
	require.NoError(t, err)
	assert.Len(t, sgs.SecurityGroups, 1)
	assert.Len(t, sgs.SecurityGroups[0].IpPermissions, 1)

	// Cleanup
	defer func() {
		// Delete security group
		_, _ = client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(sgID),
		})

		// Delete VPC
		_, _ = client.DeleteVpc(ctx, &ec2.DeleteVpcInput{
			VpcId: aws.String(vpcID),
		})
	}()
}

func testDriftDetection(t *testing.T) {
	ctx := context.Background()

	// Create provider with LocalStack endpoint
	provider := awsprovider.NewAWSProvider(testRegion)
	
	// Override with LocalStack configuration
	os.Setenv("AWS_ENDPOINT_URL", localstackEndpoint)
	os.Setenv("AWS_ACCESS_KEY_ID", "test")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	defer func() {
		os.Unsetenv("AWS_ENDPOINT_URL")
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}()

	// Initialize provider
	err := provider.Initialize(ctx)
	require.NoError(t, err)

	// Discover resources
	resources, err := provider.DiscoverResources(ctx, map[string]interface{}{
		"resource_types": []string{"ec2", "vpc", "s3"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resources)

	// Create desired state (simulate Terraform state)
	desiredState := map[string]interface{}{
		"resources": []map[string]interface{}{
			{
				"type": "aws_vpc",
				"name": "test-vpc",
				"properties": map[string]interface{}{
					"cidr_block": testVPCCIDR,
					"tags": map[string]string{
						"Name":      "test-vpc",
						"ManagedBy": "Terraform", // Different from actual
					},
				},
			},
		},
	}

	// Detect drift
	detector := drift.NewDriftDetector(provider)
	drifts, err := detector.DetectDrift(ctx, desiredState)
	require.NoError(t, err)
	
	// Should detect tag drift
	assert.NotEmpty(t, drifts)
	
	hasDrift := false
	for _, d := range drifts {
		if d.ResourceType == "aws_vpc" {
			hasDrift = true
			assert.Equal(t, drift.ConfigurationDrift, d.DriftType)
			break
		}
	}
	assert.True(t, hasDrift, "Should detect VPC tag drift")
}

func testStateFileOperations(t *testing.T, cfg aws.Config) {
	ctx := context.Background()
	
	// Create state manager
	stateManager := state.NewS3StateManager(cfg, testBucket)

	// Test state file
	testState := &state.TerraformState{
		Version: 4,
		Serial:  1,
		Resources: []state.Resource{
			{
				Type: "aws_instance",
				Name: "test",
				Instances: []state.Instance{
					{
						ID: "i-12345",
						Attributes: map[string]interface{}{
							"instance_type": "t2.micro",
							"ami":           "ami-12345",
						},
					},
				},
			},
		},
	}

	// Push state
	err := stateManager.PushState(ctx, "test-env/terraform.tfstate", testState)
	require.NoError(t, err)

	// Pull state
	pulledState, err := stateManager.PullState(ctx, "test-env/terraform.tfstate")
	require.NoError(t, err)
	assert.Equal(t, testState.Serial, pulledState.Serial)
	assert.Len(t, pulledState.Resources, 1)

	// List states
	states, err := stateManager.ListStates(ctx, "test-env/")
	require.NoError(t, err)
	assert.Contains(t, states, "test-env/terraform.tfstate")

	// Lock state
	lockID, err := stateManager.LockState(ctx, "test-env/terraform.tfstate")
	require.NoError(t, err)
	assert.NotEmpty(t, lockID)

	// Try to lock again (should fail)
	_, err = stateManager.LockState(ctx, "test-env/terraform.tfstate")
	assert.Error(t, err, "Should not be able to lock already locked state")

	// Unlock state
	err = stateManager.UnlockState(ctx, "test-env/terraform.tfstate", lockID)
	require.NoError(t, err)

	// Cleanup
	err = stateManager.DeleteState(ctx, "test-env/terraform.tfstate")
	require.NoError(t, err)
}

// Helper function to check if LocalStack is running
func isLocalStackRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           localstackEndpoint,
					SigningRegion: testRegion,
				}, nil
			})),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     "test",
				SecretAccessKey: "test",
			}, nil
		})),
	)
	
	if err != nil {
		return false
	}

	client := s3.NewFromConfig(cfg)
	_, err = client.ListBuckets(ctx, &s3.ListBucketsInput{})
	return err == nil
}