package aws

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
			name:   "without region",
			region: "",
			want:   "", // Empty region when not provided
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
	provider := NewAWSProvider("us-east-1")

	// Skip if AWS credentials are not configured
	if err := provider.Initialize(ctx); err != nil {
		t.Skipf("Skipping DiscoverResources test - AWS not configured: %v", err)
	}

	// Test discovery
	resources, err := provider.DiscoverResources(ctx, "us-east-1")

	// The test will pass even if no resources are found
	// as long as no error occurs
	assert.NoError(t, err)
	assert.NotNil(t, resources)
}

func TestAWSProvider_GetResource(t *testing.T) {
	provider := NewAWSProvider("us-east-1")
	ctx := context.Background()

	// Test getting EC2 instance
	resource, err := provider.GetResource(ctx, "i-1234567890abcdef0")

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

// EstimateCost test removed as the method doesn't exist in the provider

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
		_, _ = provider.DiscoverResources(ctx, "us-east-1")
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
			_, _ = provider.DiscoverResources(ctx, "us-east-1")
		}
	})
}
