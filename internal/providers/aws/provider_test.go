package aws

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockEC2Client is a mock implementation of EC2 client
type MockEC2Client struct {
	mock.Mock
}

func (m *MockEC2Client) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeInstancesOutput), args.Error(1)
}

func (m *MockEC2Client) DescribeSecurityGroups(ctx context.Context, params *ec2.DescribeSecurityGroupsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeSecurityGroupsOutput), args.Error(1)
}

func (m *MockEC2Client) DescribeVpcs(ctx context.Context, params *ec2.DescribeVpcsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeVpcsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeVpcsOutput), args.Error(1)
}

func (m *MockEC2Client) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ec2.DescribeSubnetsOutput), args.Error(1)
}

func TestNewAWSProvider(t *testing.T) {
	tests := []struct {
		name   string
		region string
		want   string
	}{
		{
			name:   "with region",
			region: "us-west-2",
			want:   "us-west-2",
		},
		{
			name:   "without region uses default",
			region: "",
			want:   "us-east-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewAWSProvider(tt.region)
			assert.NotNil(t, provider)
			assert.Equal(t, tt.want, provider.region)
		})
	}
}

func TestAWSProvider_Initialize(t *testing.T) {
	provider := NewAWSProvider("us-east-1")

	// Test initialization without credentials
	ctx := context.Background()
	err := provider.Initialize(ctx)

	// This will fail if AWS credentials are not configured
	// In CI/CD, we'll use LocalStack or mocked clients
	if err != nil {
		t.Skipf("Skipping Initialize test - AWS credentials not configured: %v", err)
	}

	assert.NotNil(t, provider.ec2Client)
	assert.NotNil(t, provider.s3Client)
	assert.NotNil(t, provider.iamClient)
}

func TestAWSProvider_GetProviderName(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	assert.Equal(t, "aws", provider.GetProviderName())
}

func TestAWSProvider_DiscoverResources(t *testing.T) {
	ctx := context.Background()
	provider := &AWSProvider{
		region: "us-east-1",
	}

	// Create mock client
	mockEC2 := new(MockEC2Client)

	// Set up expected responses
	instanceOutput := &ec2.DescribeInstancesOutput{
		Reservations: []types.Reservation{
			{
				Instances: []types.Instance{
					{
						InstanceId:   aws.String("i-1234567890abcdef0"),
						InstanceType: types.InstanceTypeT2Micro,
						State: &types.InstanceState{
							Name: types.InstanceStateNameRunning,
						},
						Tags: []types.Tag{
							{
								Key:   aws.String("Name"),
								Value: aws.String("test-instance"),
							},
						},
					},
				},
			},
		},
	}

	sgOutput := &ec2.DescribeSecurityGroupsOutput{
		SecurityGroups: []types.SecurityGroup{
			{
				GroupId:     aws.String("sg-12345"),
				GroupName:   aws.String("test-sg"),
				Description: aws.String("Test security group"),
			},
		},
	}

	vpcOutput := &ec2.DescribeVpcsOutput{
		Vpcs: []types.Vpc{
			{
				VpcId:     aws.String("vpc-12345"),
				CidrBlock: aws.String("10.0.0.0/16"),
				State:     types.VpcStateAvailable,
			},
		},
	}

	subnetOutput := &ec2.DescribeSubnetsOutput{
		Subnets: []types.Subnet{
			{
				SubnetId:         aws.String("subnet-12345"),
				VpcId:            aws.String("vpc-12345"),
				CidrBlock:        aws.String("10.0.1.0/24"),
				AvailabilityZone: aws.String("us-east-1a"),
			},
		},
	}

	// Set up mock expectations
	mockEC2.On("DescribeInstances", ctx, mock.Anything).Return(instanceOutput, nil)
	mockEC2.On("DescribeSecurityGroups", ctx, mock.Anything).Return(sgOutput, nil)
	mockEC2.On("DescribeVpcs", ctx, mock.Anything).Return(vpcOutput, nil)
	mockEC2.On("DescribeSubnets", ctx, mock.Anything).Return(subnetOutput, nil)

	// Inject mock client
	provider.ec2Client = mockEC2

	// Test discovery
	resources, err := provider.DiscoverResources(ctx, map[string]interface{}{
		"resource_types": []string{"ec2", "vpc"},
	})

	// Since we need to implement the actual discovery logic,
	// for now we'll just verify the mock was called
	require.NoError(t, err)
	assert.NotNil(t, resources)

	// Verify all mocks were called
	mockEC2.AssertExpectations(t)
}

func TestAWSProvider_GetResource(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	// Test getting EC2 instance
	resource, err := provider.GetResource(ctx, "aws_instance", "i-1234567890abcdef0")

	// This will fail without proper AWS setup
	if err != nil {
		t.Skipf("Skipping GetResource test - AWS not configured: %v", err)
	}

	assert.NotNil(t, resource)
}

func TestAWSProvider_ValidateCredentials(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := provider.ValidateCredentials(ctx)

	// Skip if no credentials configured
	if err != nil {
		t.Skipf("Skipping ValidateCredentials test - AWS credentials not configured: %v", err)
	}
}

func TestAWSProvider_EstimateCost(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	tests := []struct {
		name         string
		resourceType string
		config       map[string]interface{}
		wantMin      float64
		wantMax      float64
	}{
		{
			name:         "t2.micro instance",
			resourceType: "aws_instance",
			config: map[string]interface{}{
				"instance_type": "t2.micro",
			},
			wantMin: 0.0,
			wantMax: 10.0,
		},
		{
			name:         "unknown resource",
			resourceType: "aws_unknown",
			config:       map[string]interface{}{},
			wantMin:      0.0,
			wantMax:      0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := provider.EstimateCost(ctx, tt.resourceType, tt.config)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, cost, tt.wantMin)
			assert.LessOrEqual(t, cost, tt.wantMax)
		})
	}
}

// Benchmark tests
func BenchmarkAWSProvider_DiscoverResources(b *testing.B) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	// Skip if AWS is not configured
	if err := provider.Initialize(ctx); err != nil {
		b.Skipf("AWS not configured: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.DiscoverResources(ctx, map[string]interface{}{
			"resource_types": []string{"ec2"},
			"max_results":    10,
		})
	}
}

func BenchmarkAWSProvider_ParallelDiscovery(b *testing.B) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	// Skip if AWS is not configured
	if err := provider.Initialize(ctx); err != nil {
		b.Skipf("AWS not configured: %v", err)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = provider.DiscoverResources(ctx, map[string]interface{}{
				"resource_types": []string{"vpc"},
				"max_results":    5,
			})
		}
	})
}
