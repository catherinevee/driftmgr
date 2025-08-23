package aws

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/catherinevee/driftmgr/internal/core/models"
)

// AWSRoleAssumer handles cross-account role assumption for AWS
type AWSRoleAssumer struct {
	stsClient  *sts.Client
	baseConfig aws.Config
}

// NewAWSRoleAssumer creates a new AWS role assumer
func NewAWSRoleAssumer() (*AWSRoleAssumer, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSRoleAssumer{
		stsClient:  sts.NewFromConfig(cfg),
		baseConfig: cfg,
	}, nil
}

// AssumeRole assumes a role in the target account
func (ara *AWSRoleAssumer) AssumeRole(ctx context.Context, accountID, roleName string) (aws.Config, error) {
	if roleName == "" {
		// Use default Organization role name
		roleName = "OrganizationAccountAccessRole"
	}

	roleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
	sessionName := fmt.Sprintf("driftmgr-%d", time.Now().Unix())

	// Create STS assume role provider
	creds := stscreds.NewAssumeRoleProvider(ara.stsClient, roleARN, func(o *stscreds.AssumeRoleOptions) {
		o.RoleSessionName = sessionName
		o.Duration = 1 * time.Hour
	})

	// Create new config with assumed role credentials
	cfg := ara.baseConfig.Copy()
	cfg.Credentials = aws.NewCredentialsCache(creds)

	return cfg, nil
}

// GetAssumedRoleConfig returns an AWS config for the specified account with role assumption
func GetAssumedRoleConfig(ctx context.Context, accountID string, roleName string) (aws.Config, error) {
	// Check if we should use role assumption
	if os.Getenv("DRIFTMGR_ASSUME_ROLE") == "false" {
		// Use default config without role assumption
		return config.LoadDefaultConfig(ctx)
	}

	// Check for custom role name from environment
	if envRole := os.Getenv("DRIFTMGR_ASSUME_ROLE_NAME"); envRole != "" {
		roleName = envRole
	}

	// Default role name for AWS Organizations
	if roleName == "" {
		roleName = "OrganizationAccountAccessRole"
	}

	// Load base config
	baseCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// If accountID matches current account, no need to assume role
	stsClient := sts.NewFromConfig(baseCfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err == nil && identity.Account != nil && *identity.Account == accountID {
		// Same account, no role assumption needed
		return baseCfg, nil
	}

	// Assume role in target account
	roleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
	sessionName := fmt.Sprintf("driftmgr-%d", time.Now().Unix())

	creds := stscreds.NewAssumeRoleProvider(stsClient, roleARN, func(o *stscreds.AssumeRoleOptions) {
		o.RoleSessionName = sessionName
		o.Duration = 1 * time.Hour
	})

	// Create new config with assumed role credentials
	cfg := baseCfg.Copy()
	cfg.Credentials = aws.NewCredentialsCache(creds)

	// Verify the assumed role works
	testStsClient := sts.NewFromConfig(cfg)
	_, err = testStsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to verify assumed role: %w", err)
	}

	return cfg, nil
}

// DiscoverAWSWithRoleAssumption discovers AWS resources using role assumption
func DiscoverAWSWithRoleAssumption(ctx context.Context, accountID string, regions []string) ([]models.Resource, error) {
	// Get config with assumed role
	cfg, err := GetAssumedRoleConfig(ctx, accountID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to assume role for account %s: %w", accountID, err)
	}

	// Create a new AWS discoverer with the assumed role config
	// This would integrate with your existing AWS discovery logic

	var resources []models.Resource

	// Example: Discover EC2 instances with assumed role
	for _, region := range regions {
		regionalCfg := cfg.Copy()
		regionalCfg.Region = region

		// Here you would call your existing AWS discovery functions
		// using the regionalCfg with assumed role credentials

		// For now, return empty as this is a placeholder
		// In production, this would call your actual discovery functions
	}

	return resources, nil
}
