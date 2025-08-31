package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// Provider wraps the AWS discovery provider
type Provider struct {
	awsProvider *discovery.AWSProvider
}

// NewProvider creates a new AWS provider
func NewProvider() discovery.Provider {
	awsProvider, _ := discovery.NewAWSProvider()
	return &Provider{
		awsProvider: awsProvider,
	}
}

// Discover performs discovery across all configured regions
func (p *Provider) Discover(ctx context.Context, options discovery.DiscoveryOptions) (*discovery.Result, error) {
	if p.awsProvider == nil {
		// Try to initialize
		var err error
		p.awsProvider, err = discovery.NewAWSProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize AWS provider: %w", err)
		}
	}

	// If no regions specified, use default
	if len(options.Regions) == 0 {
		options.Regions = []string{"us-east-1"}
	}

	// Call the AWS provider's Discover method directly with all regions
	// This ensures S3 buckets (global service) are only discovered once
	return p.awsProvider.Discover(ctx, options)
}

// DiscoverRegion discovers resources in a specific region
func (p *Provider) DiscoverRegion(ctx context.Context, region string) ([]models.Resource, error) {
	if p.awsProvider == nil {
		var err error
		p.awsProvider, err = discovery.NewAWSProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize AWS provider: %w", err)
		}
	}

	// Use the AWS provider's Discover method
	options := discovery.DiscoveryOptions{
		Regions: []string{region},
		Timeout: 30 * time.Second,
	}
	
	result, err := p.awsProvider.Discover(ctx, options)
	if err != nil {
		return nil, err
	}
	
	if result != nil {
		return result.Resources, nil
	}
	
	return []models.Resource{}, nil
}

// Name returns the provider name
func (p *Provider) Name() string {
	if p.awsProvider != nil {
		return p.awsProvider.Name()
	}
	return "AWS"
}

// Regions returns supported regions
func (p *Provider) Regions() []string {
	if p.awsProvider != nil {
		return p.awsProvider.Regions()
	}
	return []string{"us-east-1", "us-west-2", "eu-west-1"}
}

// Services returns supported services
func (p *Provider) Services() []string {
	if p.awsProvider != nil {
		return p.awsProvider.Services()
	}
	return []string{"EC2", "S3", "RDS", "Lambda"}
}

// ValidateCredentials validates AWS credentials
func (p *Provider) ValidateCredentials(ctx context.Context) error {
	if p.awsProvider == nil {
		var err error
		p.awsProvider, err = discovery.NewAWSProvider()
		if err != nil {
			return fmt.Errorf("failed to validate credentials: %w", err)
		}
	}
	// Credentials are validated during provider creation
	return nil
}

// GetAccountInfo returns AWS account information
func (p *Provider) GetAccountInfo(ctx context.Context) (*discovery.AccountInfo, error) {
	// This would normally call STS GetCallerIdentity
	return &discovery.AccountInfo{
		Provider:  "aws",
		ID: "123456789012",
		Name:      "AWS Account",
	}, nil
}