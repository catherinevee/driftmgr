package account

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// AccountInfo represents account information for a cloud provider
type AccountInfo struct {
	Provider    string
	AccountID   string
	AccountName string
	UserID      string
	ARN         string
	Region      string
	Error       error
}

// GetAccountInfo retrieves account information for the specified provider
func GetAccountInfo(provider string) *AccountInfo {
	switch provider {
	case "aws":
		return getAWSAccountInfo()
	case "azure":
		return getAzureAccountInfo()
	case "gcp":
		return getGCPAccountInfo()
	case "digitalocean":
		return getDigitalOceanAccountInfo()
	default:
		return &AccountInfo{
			Provider: provider,
			Error:    fmt.Errorf("unsupported provider: %s", provider),
		}
	}
}

// getAWSAccountInfo retrieves AWS account information using STS
func getAWSAccountInfo() *AccountInfo {
	info := &AccountInfo{
		Provider: "aws",
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		info.Error = fmt.Errorf("AWS credentials not configured. Please configure AWS credentials using 'aws configure' or set environment variables")
		return info
	}

	// Create STS client
	stsClient := sts.NewFromConfig(cfg)

	// Get caller identity
	result, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		if strings.Contains(err.Error(), "NoCredentialProviders") {
			info.Error = fmt.Errorf("AWS credentials not found. Please configure AWS credentials using 'aws configure' or set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY")
		} else if strings.Contains(err.Error(), "ExpiredTokenException") {
			info.Error = fmt.Errorf("AWS credentials expired. Please refresh your AWS credentials")
		} else {
			info.Error = fmt.Errorf("failed to get AWS caller identity: %w", err)
		}
		return info
	}

	info.AccountID = *result.Account
	info.UserID = *result.UserId
	info.ARN = *result.Arn

	// Try to get account name from ARN
	if strings.Contains(info.ARN, "assumed-role") {
		parts := strings.Split(info.ARN, "/")
		if len(parts) >= 2 {
			info.AccountName = parts[len(parts)-1]
		}
	} else {
		info.AccountName = "AWS Account"
	}

	return info
}

// getAzureAccountInfo retrieves Azure account information using Azure CLI
func getAzureAccountInfo() *AccountInfo {
	info := &AccountInfo{
		Provider: "azure",
	}

	// Check if Azure CLI is installed
	if _, err := exec.LookPath("az"); err != nil {
		info.Error = fmt.Errorf("Azure CLI not found. Please install Azure CLI and run 'az login' to authenticate")
		return info
	}

	// Use Azure CLI to get account information
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get account name
	cmd := exec.CommandContext(ctx, "az", "account", "show", "--query", "name", "--output", "tsv")
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			info.Error = fmt.Errorf("not authenticated with Azure. Please run 'az login' to authenticate")
		} else {
			info.Error = fmt.Errorf("failed to get Azure account information: %w", err)
		}
		return info
	}
	info.AccountName = strings.TrimSpace(string(output))

	// Get subscription ID
	cmd = exec.CommandContext(ctx, "az", "account", "show", "--query", "id", "--output", "tsv")
	output, err = cmd.Output()
	if err != nil {
		info.Error = fmt.Errorf("failed to get Azure subscription ID: %w", err)
		return info
	}
	info.AccountID = strings.TrimSpace(string(output))

	// Get user ID
	cmd = exec.CommandContext(ctx, "az", "account", "show", "--query", "user.name", "--output", "tsv")
	output, err = cmd.Output()
	if err != nil {
		// User ID is optional, don't fail if we can't get it
		info.UserID = "Unknown"
	} else {
		info.UserID = strings.TrimSpace(string(output))
	}

	return info
}

// getGCPAccountInfo retrieves GCP account information using gcloud CLI
func getGCPAccountInfo() *AccountInfo {
	info := &AccountInfo{
		Provider: "gcp",
	}

	// Check if gcloud CLI is installed
	if _, err := exec.LookPath("gcloud"); err != nil {
		info.Error = fmt.Errorf("Google Cloud SDK not found. Please install Google Cloud SDK and run 'gcloud auth login' to authenticate")
		return info
	}

	// Use gcloud CLI to get account information
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get project ID
	cmd := exec.CommandContext(ctx, "gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			info.Error = fmt.Errorf("not authenticated with Google Cloud. Please run 'gcloud auth login' to authenticate")
		} else {
			info.Error = fmt.Errorf("failed to get GCP project ID: %w", err)
		}
		return info
	}
	info.AccountID = strings.TrimSpace(string(output))

	// Get account name
	cmd = exec.CommandContext(ctx, "gcloud", "config", "get-value", "account")
	output, err = cmd.Output()
	if err != nil {
		info.Error = fmt.Errorf("failed to get GCP account: %w", err)
		return info
	}
	info.AccountName = strings.TrimSpace(string(output))

	// Get user ID (same as account for GCP)
	info.UserID = info.AccountName

	return info
}

// getDigitalOceanAccountInfo retrieves DigitalOcean account information using doctl CLI
func getDigitalOceanAccountInfo() *AccountInfo {
	info := &AccountInfo{
		Provider: "digitalocean",
	}

	// Check if doctl CLI is installed
	if _, err := exec.LookPath("doctl"); err != nil {
		info.Error = fmt.Errorf("DigitalOcean CLI (doctl) not found. Please install doctl and run 'doctl auth init' to authenticate")
		return info
	}

	// Use doctl CLI to get account information
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get account information
	cmd := exec.CommandContext(ctx, "doctl", "account", "get", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			info.Error = fmt.Errorf("not authenticated with DigitalOcean. Please run 'doctl auth init' to authenticate")
		} else {
			info.Error = fmt.Errorf("failed to get DigitalOcean account info: %w", err)
		}
		return info
	}

	// For simplicity, we'll just extract basic info
	// In a real implementation, you'd parse the JSON output
	outputStr := strings.TrimSpace(string(output))
	if strings.Contains(outputStr, "uuid") {
		// Extract UUID from JSON (simplified)
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if strings.Contains(line, "uuid") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					info.AccountID = strings.Trim(strings.TrimSpace(parts[1]), `",`)
					break
				}
			}
		}
	}

	info.AccountName = "DigitalOcean Account"
	info.UserID = "Unknown"

	return info
}

// FormatAccountInfo formats account information for display
func FormatAccountInfo(info *AccountInfo) string {
	if info.Error != nil {
		return fmt.Sprintf("❌ %s Account: Error - %v", strings.ToUpper(info.Provider), info.Error)
	}

	switch info.Provider {
	case "aws":
		return fmt.Sprintf("☁️  AWS Account: %s (%s) | User: %s", info.AccountID, info.AccountName, info.UserID)
	case "azure":
		return fmt.Sprintf("☁️  Azure Account: %s (%s) | User: %s", info.AccountID, info.AccountName, info.UserID)
	case "gcp":
		return fmt.Sprintf("☁️  GCP Project: %s | Account: %s", info.AccountID, info.AccountName)
	case "digitalocean":
		return fmt.Sprintf("☁️  DigitalOcean Account: %s (%s)", info.AccountID, info.AccountName)
	default:
		return fmt.Sprintf("☁️  %s Account: %s", strings.ToUpper(info.Provider), info.AccountID)
	}
}
