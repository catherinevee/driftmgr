package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/catherinevee/driftmgr/internal/models"
)

// Client represents an AWS client with real AWS SDK integration
type Client struct {
	config      aws.Config
	region      string
	stsClient   *sts.Client
	ec2Client   *ec2.Client
	credentials *models.ProviderCredentials
}

// NewClient creates a new AWS client with real AWS SDK integration
func NewClient(region string, creds *models.ProviderCredentials) (*Client, error) {
	// Create AWS config with credentials
	var cfg aws.Config
	var err error

	if creds != nil && creds.AccessKey != "" && creds.SecretKey != "" {
		// Use provided credentials
		cfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				creds.AccessKey,
				creds.SecretKey,
				creds.Token,
			)),
		)
	} else {
		// Use default credential chain (environment, IAM role, etc.)
		cfg, err = config.LoadDefaultConfig(context.Background(),
			config.WithRegion(region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create service clients
	stsClient := sts.NewFromConfig(cfg)
	ec2Client := ec2.NewFromConfig(cfg)

	return &Client{
		config:      cfg,
		region:      region,
		stsClient:   stsClient,
		ec2Client:   ec2Client,
		credentials: creds,
	}, nil
}

// TestConnection tests the AWS connection using STS GetCallerIdentity
func (c *Client) TestConnection(ctx context.Context) error {
	// Use STS GetCallerIdentity to test the connection
	_, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("failed to test AWS connection: %w", err)
	}
	return nil
}

// GetRegion returns the AWS region
func (c *Client) GetRegion() string {
	return c.region
}

// SetRegion sets the AWS region and updates clients
func (c *Client) SetRegion(region string) {
	c.region = region
	c.config.Region = region
	// Update clients with new region
	c.stsClient = sts.NewFromConfig(c.config)
	c.ec2Client = ec2.NewFromConfig(c.config)
}

// GetAccountID returns the AWS account ID using STS GetCallerIdentity
func (c *Client) GetAccountID(ctx context.Context) (string, error) {
	result, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to get AWS account ID: %w", err)
	}
	return *result.Account, nil
}

// GetAvailableRegions returns available AWS regions using EC2 DescribeRegions
func (c *Client) GetAvailableRegions(ctx context.Context) ([]string, error) {
	result, err := c.ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false), // Only return regions that are enabled
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get available regions: %w", err)
	}

	regions := make([]string, 0, len(result.Regions))
	for _, region := range result.Regions {
		if region.RegionName != nil {
			regions = append(regions, *region.RegionName)
		}
	}

	return regions, nil
}

// ValidateCredentials validates AWS credentials using STS GetCallerIdentity
func (c *Client) ValidateCredentials(ctx context.Context) error {
	// Use STS GetCallerIdentity to validate credentials
	result, err := c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("invalid AWS credentials: %w", err)
	}

	// Additional validation - check if we got a valid response
	if result.Account == nil || *result.Account == "" {
		return fmt.Errorf("invalid AWS credentials: no account ID returned")
	}

	return nil
}

// GetCallerIdentity returns detailed caller identity information
func (c *Client) GetCallerIdentity(ctx context.Context) (*sts.GetCallerIdentityOutput, error) {
	return c.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
}

// GetConfig returns the AWS config for use by other services
func (c *Client) GetConfig() aws.Config {
	return c.config
}

// GetCredentials returns the credentials used by this client
func (c *Client) GetCredentials() *models.ProviderCredentials {
	return c.credentials
}

// IsUsingDefaultCredentials returns true if using default credential chain
func (c *Client) IsUsingDefaultCredentials() bool {
	return c.credentials == nil || c.credentials.AccessKey == ""
}
