package discovery

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"google.golang.org/api/iterator"
)

// CloudStateFile represents a discovered tfstate file in the cloud
type CloudStateFile struct {
	Provider     string            `json:"provider"`
	Region       string            `json:"region"`
	Bucket       string            `json:"bucket"`
	Key          string            `json:"key"`
	Size         int64             `json:"size"`
	LastModified string            `json:"last_modified"`
	URL          string            `json:"url"`
	Metadata     map[string]string `json:"metadata,omitempty"`

	// Terragrunt-specific fields
	IsTerragrunt bool   `json:"is_terragrunt,omitempty"`
	Environment  string `json:"environment,omitempty"`
	Stack        string `json:"stack,omitempty"`
	Component    string `json:"component,omitempty"`
	DeployRegion string `json:"deploy_region,omitempty"`
}

// CloudStateDiscovery discovers tfstate files across cloud providers
type CloudStateDiscovery struct {
	awsRegions []string
	mu         sync.Mutex
	results    []CloudStateFile
}

// NewCloudStateDiscovery creates a new cloud state discovery instance
func NewCloudStateDiscovery() *CloudStateDiscovery {
	return &CloudStateDiscovery{
		awsRegions: []string{
			"us-east-1", "us-east-2", "us-west-1", "us-west-2",
			"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
			"ap-south-1", "ap-northeast-1", "ap-northeast-2", "ap-northeast-3",
			"ap-southeast-1", "ap-southeast-2",
			"ca-central-1", "sa-east-1",
		},
		results: []CloudStateFile{},
	}
}

// DiscoverAll discovers tfstate files across all cloud providers
func (d *CloudStateDiscovery) DiscoverAll(ctx context.Context) ([]CloudStateFile, error) {
	var wg sync.WaitGroup
	errors := make(chan error, 3)

	// Discover AWS S3 state files
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := d.discoverAWSStates(ctx); err != nil {
			errors <- fmt.Errorf("AWS discovery error: %w", err)
		}
	}()

	// Discover Azure Blob state files
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := d.discoverAzureStates(ctx); err != nil {
			errors <- fmt.Errorf("Azure discovery error: %w", err)
		}
	}()

	// Discover GCS state files
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := d.discoverGCSStates(ctx); err != nil {
			errors <- fmt.Errorf("GCS discovery error: %w", err)
		}
	}()

	// Wait for all discoveries to complete
	wg.Wait()
	close(errors)

	// Collect any errors
	var errs []string
	for err := range errors {
		if err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		fmt.Printf("Discovery completed with errors: %s\n", strings.Join(errs, "; "))
	}

	return d.results, nil
}

// discoverAWSStates discovers tfstate files in S3 across all regions
func (d *CloudStateDiscovery) discoverAWSStates(ctx context.Context) error {
	fmt.Println("Scanning AWS S3 for tfstate files across all regions...")

	// Create initial session to list buckets
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %w", err)
	}

	svc := s3.New(sess)

	// List all buckets
	bucketsResp, err := svc.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("failed to list S3 buckets: %w", err)
	}

	fmt.Printf("Found %d S3 buckets to scan\n", len(bucketsResp.Buckets))

	// Scan each bucket for tfstate files
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Limit concurrent operations

	for _, bucket := range bucketsResp.Buckets {
		wg.Add(1)
		go func(bucketName string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			d.scanS3Bucket(ctx, bucketName)
		}(*bucket.Name)
	}

	wg.Wait()
	return nil
}

// scanS3Bucket scans a single S3 bucket for tfstate files
func (d *CloudStateDiscovery) scanS3Bucket(ctx context.Context, bucketName string) {
	// Get bucket region
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})
	svc := s3.New(sess)

	// Get bucket location
	locResp, err := svc.GetBucketLocation(&s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// Skip buckets we can't access
		return
	}

	region := "us-east-1"
	if locResp.LocationConstraint != nil && *locResp.LocationConstraint != "" {
		region = *locResp.LocationConstraint
	}

	// Create new session for the bucket's region
	sess, err = session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return
	}
	svc = s3.New(sess)

	// List objects in bucket looking for tfstate files
	err = svc.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			if obj.Key != nil && (strings.HasSuffix(*obj.Key, ".tfstate") ||
				strings.HasSuffix(*obj.Key, ".tfstate.backup")) {

				// Skip only obvious test files
				lowerKey := strings.ToLower(*obj.Key)
				if strings.Contains(lowerKey, "test-") || strings.Contains(lowerKey, "/tmp/") ||
					strings.Contains(lowerKey, "temp/") || strings.Contains(lowerKey, ".test.tfstate") {
					continue
				}

				// Parse Terragrunt patterns
				isTerragrunt, env, stack, component, deployRegion := parseTerragruntPath(*obj.Key)

				stateFile := CloudStateFile{
					Provider:     "aws",
					Region:       region,
					Bucket:       bucketName,
					Key:          *obj.Key,
					Size:         *obj.Size,
					LastModified: obj.LastModified.String(),
					URL:          fmt.Sprintf("s3://%s/%s", bucketName, *obj.Key),
					IsTerragrunt: isTerragrunt,
					Environment:  env,
					Stack:        stack,
					Component:    component,
					DeployRegion: deployRegion,
					Metadata: map[string]string{
						"storage_class": *obj.StorageClass,
					},
				}

				d.mu.Lock()
				d.results = append(d.results, stateFile)
				d.mu.Unlock()

				fmt.Printf("  Found: %s (%d bytes)\n", stateFile.URL, stateFile.Size)
			}
		}
		return !lastPage
	})
}

// discoverAzureStates discovers tfstate files in Azure Storage
func (d *CloudStateDiscovery) discoverAzureStates(ctx context.Context) error {
	fmt.Println("Scanning Azure Storage for tfstate files...")

	// Get Azure credentials
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return fmt.Errorf("failed to get Azure credentials: %w", err)
	}

	// Get subscription ID from environment or use default
	subscriptionID := getAzureSubscriptionIDForCloudState()
	if subscriptionID == "" {
		fmt.Println("No Azure subscription found, skipping Azure scan")
		return nil
	}

	// Create storage client
	clientFactory, err := armstorage.NewClientFactory(subscriptionID, cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create Azure storage client: %w", err)
	}

	accountsClient := clientFactory.NewAccountsClient()

	// List all storage accounts
	pager := accountsClient.NewListPager(nil)
	accountCount := 0

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list storage accounts: %w", err)
		}

		for _, account := range page.Value {
			accountCount++
			d.scanAzureStorageAccount(ctx, *account.Name, *account.Location)
		}
	}

	fmt.Printf("Scanned %d Azure storage accounts\n", accountCount)
	return nil
}

// scanAzureStorageAccount scans an Azure storage account for tfstate files
func (d *CloudStateDiscovery) scanAzureStorageAccount(ctx context.Context, accountName, location string) {
	// This would require storage keys which we'll skip for now in the scan
	// In production, you'd enumerate containers and blobs
	fmt.Printf("  Would scan storage account: %s in %s\n", accountName, location)
}

// discoverGCSStates discovers tfstate files in Google Cloud Storage
func (d *CloudStateDiscovery) discoverGCSStates(ctx context.Context) error {
	fmt.Println("Scanning Google Cloud Storage for tfstate files...")

	// Create GCS client
	client, err := storage.NewClient(ctx)
	if err != nil {
		// GCS not configured, skip
		fmt.Println("GCS not configured, skipping GCS scan")
		return nil
	}
	defer client.Close()

	// Get project ID from environment or gcloud config
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT")
	}
	if projectID == "" {
		// Try to get from gcloud config
		cmd := exec.Command("gcloud", "config", "get-value", "project")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			projectID = strings.TrimSpace(string(output))
		}
	}

	if projectID == "" {
		fmt.Println("GCP project ID not found, skipping GCS scan")
		return nil
	}

	// List all buckets
	it := client.Buckets(ctx, projectID)
	bucketCount := 0

	for {
		bucket, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list GCS buckets: %w", err)
		}

		bucketCount++
		d.scanGCSBucket(ctx, client, bucket.Name)
	}

	fmt.Printf("Scanned %d GCS buckets\n", bucketCount)
	return nil
}

// scanGCSBucket scans a GCS bucket for tfstate files
func (d *CloudStateDiscovery) scanGCSBucket(ctx context.Context, client *storage.Client, bucketName string) {
	bucket := client.Bucket(bucketName)
	it := bucket.Objects(ctx, nil)

	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Skip buckets we can't access
			return
		}

		if strings.HasSuffix(obj.Name, ".tfstate") || strings.HasSuffix(obj.Name, ".tfstate.backup") {
			// Skip only obvious test files
			lowerName := strings.ToLower(obj.Name)
			if strings.Contains(lowerName, "test-") || strings.Contains(lowerName, "/tmp/") ||
				strings.Contains(lowerName, "temp/") || strings.Contains(lowerName, ".test.tfstate") {
				continue
			}

			// Parse Terragrunt patterns
			isTerragrunt, env, stack, component, deployRegion := parseTerragruntPath(obj.Name)

			stateFile := CloudStateFile{
				Provider:     "gcp",
				Region:       obj.Metadata["region"],
				Bucket:       bucketName,
				Key:          obj.Name,
				Size:         obj.Size,
				LastModified: obj.Updated.String(),
				URL:          fmt.Sprintf("gs://%s/%s", bucketName, obj.Name),
				IsTerragrunt: isTerragrunt,
				Environment:  env,
				Stack:        stack,
				Component:    component,
				DeployRegion: deployRegion,
			}

			d.mu.Lock()
			d.results = append(d.results, stateFile)
			d.mu.Unlock()

			fmt.Printf("  Found: %s (%d bytes)\n", stateFile.URL, stateFile.Size)
		}
	}
}

// Helper function to get Azure subscription ID for cloud state discovery
func getAzureSubscriptionIDForCloudState() string {
	// Try environment variable first
	subID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subID != "" {
		return subID
	}

	// Try Azure CLI
	cmd := exec.Command("az", "account", "show", "--query", "id", "-o", "tsv")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output))
	}

	return ""
}

// parseTerragruntPath analyzes the state file path to extract Terragrunt metadata
func parseTerragruntPath(key string) (isTerragrunt bool, env, stack, component, region string) {
	// Common Terragrunt patterns:
	// 1. {ENV}/{REGION}/{RESOURCE}.tfstate (e.g., prod/us-east-1/vpc.tfstate)
	// 2. {REGION}/{ENV}/{RESOURCE}.tfstate (e.g., us-east-1/prod/vpc.tfstate)
	// 3. env/region/stack/terraform.tfstate (e.g., prod/us-east-1/vpc/terraform.tfstate)
	// 4. env/region/component.tfstate (e.g., staging/eu-west-1/rds.tfstate)
	// 5. terragrunt/env/region/resource/terraform.tfstate

	parts := strings.Split(key, "/")

	// Check if path contains common Terragrunt indicators
	lowerKey := strings.ToLower(key)
	hasTerragruntIndicator := strings.Contains(lowerKey, "terragrunt") ||
		strings.Contains(lowerKey, "live/") ||
		strings.Contains(lowerKey, "environments/")

	// Check for environment names
	envNames := []string{"prod", "production", "staging", "stage", "dev", "development",
		"qa", "uat", "test", "demo", "sandbox", "preprod", "pre-prod"}

	// Check for AWS region pattern
	isAWSRegion := func(s string) bool {
		// More comprehensive region check
		return strings.Contains(s, "-") &&
			(strings.HasPrefix(s, "us-") || strings.HasPrefix(s, "eu-") ||
				strings.HasPrefix(s, "ap-") || strings.HasPrefix(s, "ca-") ||
				strings.HasPrefix(s, "sa-") || strings.HasPrefix(s, "af-") ||
				strings.HasPrefix(s, "me-"))
	}

	// Helper to check if string is an environment
	isEnvironment := func(s string) bool {
		sLower := strings.ToLower(s)
		for _, envName := range envNames {
			if sLower == envName || strings.HasPrefix(sLower, envName+"-") {
				return true
			}
		}
		return false
	}

	// Pattern 1: {ENV}/{REGION}/{RESOURCE}.tfstate (3 parts, ends with .tfstate)
	if len(parts) == 3 && strings.HasSuffix(parts[2], ".tfstate") {
		if isEnvironment(parts[0]) && isAWSRegion(parts[1]) {
			env = parts[0]
			region = parts[1]
			component = strings.TrimSuffix(parts[2], ".tfstate")
			isTerragrunt = true
			return
		}
	}

	// Pattern 2: {REGION}/{ENV}/{RESOURCE}.tfstate (3 parts, ends with .tfstate)
	if len(parts) == 3 && strings.HasSuffix(parts[2], ".tfstate") {
		if isAWSRegion(parts[0]) && isEnvironment(parts[1]) {
			region = parts[0]
			env = parts[1]
			component = strings.TrimSuffix(parts[2], ".tfstate")
			isTerragrunt = true
			return
		}
	}

	// Pattern detection based on path structure for other formats
	if len(parts) >= 3 {

		// Try to identify components
		for i, part := range parts {
			partLower := strings.ToLower(part)

			// Check for environment
			for _, envName := range envNames {
				if partLower == envName || strings.HasPrefix(partLower, envName+"-") {
					env = part
					isTerragrunt = true
					break
				}
			}

			// Check for AWS region
			if isAWSRegion(part) {
				region = part
				isTerragrunt = true
			}

			// Extract component/stack name (usually the directory before terraform.tfstate)
			if i < len(parts)-1 && strings.HasSuffix(parts[len(parts)-1], ".tfstate") {
				if i == len(parts)-2 {
					// Last directory before tfstate file
					if part != "terraform" && !isAWSRegion(part) {
						component = part
					}
				}
			}
		}

		// Additional pattern: env/region/stack/terraform.tfstate
		if len(parts) == 4 && parts[3] == "terraform.tfstate" {
			if env == "" && len(parts[0]) > 0 {
				env = parts[0]
			}
			if region == "" && isAWSRegion(parts[1]) {
				region = parts[1]
			}
			if component == "" && parts[2] != "" {
				component = parts[2]
			}
			isTerragrunt = true
		}

		// Pattern: env/region/component.tfstate
		if len(parts) == 3 && strings.HasSuffix(parts[2], ".tfstate") {
			if env == "" && len(parts[0]) > 0 {
				env = parts[0]
			}
			if region == "" && isAWSRegion(parts[1]) {
				region = parts[1]
			}
			if component == "" {
				component = strings.TrimSuffix(parts[2], ".tfstate")
			}
			isTerragrunt = true
		}
	}

	// If we found environment or region patterns, likely Terragrunt
	if env != "" || region != "" || hasTerragruntIndicator {
		isTerragrunt = true
	}

	// Set stack as component if not set
	if stack == "" && component != "" {
		stack = component
	}

	return isTerragrunt, env, stack, component, region
}
