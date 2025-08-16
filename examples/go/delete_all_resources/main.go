package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/deletion"
	"github.com/catherinevee/driftmgr/internal/models"
)

const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorCyan    = "\033[36m"
	ColorMagenta = "\033[35m"
	ColorBold    = "\033[1m"
	ColorDim     = "\033[2m"
)

// CLI flags
var (
	dryRun           = flag.Bool("dry-run", false, "Preview what would be deleted without actually deleting")
	force            = flag.Bool("force", false, "Skip safety checks and force deletion")
	resourceTypes    = flag.String("resource-types", "", "Comma-separated list of resource types to delete (e.g., ec2_instance,s3_bucket)")
	regions          = flag.String("regions", "", "Comma-separated list of regions to target")
	excludeResources = flag.String("exclude", "", "Comma-separated list of resource IDs to exclude")
	includeResources = flag.String("include", "", "Comma-separated list of resource IDs to include")
	timeout          = flag.Duration("timeout", 30*time.Minute, "Timeout for deletion operation")
	batchSize        = flag.Int("batch-size", 10, "Number of resources to delete in parallel")
	maxRetries       = flag.Int("max-retries", 3, "Maximum number of retry attempts for failed deletions")
	retryDelay       = flag.Duration("retry-delay", 5*time.Second, "Delay between retry attempts")
	logLevel         = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	exportLogs       = flag.String("export-logs", "", "Export logs in specified format (json, csv, summary)")
	providers        = flag.String("providers", "aws,azure,gcp", "Comma-separated list of cloud providers to target")
	noConfirm        = flag.Bool("no-confirm", false, "Skip confirmation prompts")
	showDependencies = flag.Bool("show-dependencies", false, "Show resource dependencies before deletion")
	verbose          = flag.Bool("verbose", false, "Enable verbose output")
)

func main() {
	flag.Parse()

	fmt.Printf("%s=== DriftMgr Enhanced Multi-Cloud Resource Deletion Tool ===%s\n", ColorBold, ColorReset)
	fmt.Printf("%sVersion: 2.0 - Enhanced with dependency management, retry logic, and comprehensive logging%s\n", ColorDim, ColorReset)
	fmt.Println()

	// Parse CLI flags
	options := parseDeletionOptions()

	// Show help if requested
	if len(flag.Args()) > 0 && flag.Args()[0] == "help" {
		showHelp()
		return
	}

	// Initialize the deletion engine
	fmt.Printf("%sInitializing enhanced deletion engine...%s\n", ColorCyan, ColorReset)
	deletionEngine := deletion.NewDeletionEngine()

	// Register cloud providers based on CLI flags
	enabledProviders := strings.Split(*providers, ",")
	registeredProviders := registerProviders(deletionEngine, enabledProviders)

	if len(registeredProviders) == 0 {
		fmt.Printf("%sNo cloud providers could be registered. Check your credentials.%s\n", ColorRed, ColorReset)
		os.Exit(1)
	}

	// Get account information
	accountInfo := getAccountInformation(registeredProviders)

	// Initialize logger
	logger, err := deletion.NewDeletionLogger(accountInfo["aws"], "multi-cloud")
	if err != nil {
		fmt.Printf("%sWarning: Failed to initialize logger: %v%s\n", ColorYellow, err, ColorReset)
	} else {
		defer logger.Close()
		logger.SetLogLevel(parseLogLevel(*logLevel))
	}

	// First, do a dry run to see what would be deleted
	fmt.Printf("\n%s=== Step 1: Enhanced Preview Deletion ===%s\n", ColorBold+ColorBlue, ColorReset)

	allResources := []models.Resource{}
	totalResources := 0

	// Preview resources for each provider
	for provider, accountID := range accountInfo {
		if accountID == "" {
			continue
		}

		fmt.Printf("\n%s  %s Resources:%s\n", ColorCyan+ColorBold, strings.ToUpper(provider), ColorReset)
		providerResources, err := previewProviderResources(deletionEngine, provider, accountID, options)
		if err != nil {
			fmt.Printf("%s    Error: %v%s\n", ColorRed, err, ColorReset)
			if logger != nil {
				logger.LogError("Provider preview failed", "", "", err.Error(), map[string]interface{}{"provider": provider})
			}
		} else {
			allResources = append(allResources, providerResources...)
			totalResources += len(providerResources)
			fmt.Printf("%s    Found %d resources%s\n", ColorGreen, len(providerResources), ColorReset)

			if logger != nil {
				logger.LogResourceDiscovery(provider, len(providerResources), groupResourcesByType(providerResources))
			}
		}
	}

	fmt.Printf("\n%sTotal resources found across all providers: %d%s\n", ColorBold+ColorYellow, totalResources, ColorReset)

	if totalResources == 0 {
		fmt.Printf("%sNo resources found to delete.%s\n", ColorGreen, ColorReset)
		return
	}

	// Show dependencies if requested
	if *showDependencies {
		showResourceDependencies(allResources)
	}

	// Get detailed resource information for preview
	fmt.Printf("\n%s=== Detailed Resource Preview ===%s\n", ColorBold+ColorBlue, ColorReset)
	printDetailedResourcePreview(allResources)

	// Safety checks
	if !*force {
		performSafetyChecks(allResources)
	}

	// Final confirmation
	if !*noConfirm {
		confirmDeletion(totalResources)
	}

	// Perform actual deletion
	fmt.Printf("\n%s=== Step 2: Enhanced Deletion with Retry Logic ===%s\n", ColorBold+ColorBlue, ColorReset)

	if logger != nil {
		logger.LogDeletionStart(options, totalResources)
	}

	startTime := time.Now()
	successCount := 0
	failureCount := 0

	// Delete resources for each provider
	for provider, accountID := range accountInfo {
		if accountID == "" {
			continue
		}

		providerResources := getProviderResources(allResources, provider)
		if len(providerResources) == 0 {
			continue
		}

		fmt.Printf("\n%s  Deleting %s Resources:%s\n", ColorCyan+ColorBold, strings.ToUpper(provider), ColorReset)
		result, err := deleteProviderResources(deletionEngine, provider, accountID, options, logger)
		if err != nil {
			fmt.Printf("%s    Error: %v%s\n", ColorRed, err, ColorReset)
			failureCount += len(providerResources)
			if logger != nil {
				logger.LogError("Provider deletion failed", "", "", err.Error(), map[string]interface{}{"provider": provider})
			}
		} else {
			successCount += result.DeletedResources
			failureCount += result.FailedResources
			fmt.Printf("%s    Deleted: %d, Failed: %d, Retried: %d%s\n",
				ColorGreen, result.DeletedResources, result.FailedResources, result.RetriedResources, ColorReset)
		}
	}

	duration := time.Since(startTime)

	// Final summary
	fmt.Printf("\n%s=== Enhanced Deletion Summary ===%s\n", ColorBold+ColorBlue, ColorReset)
	fmt.Printf("%s  Total resources processed: %d%s\n", ColorCyan, totalResources, ColorReset)
	fmt.Printf("%s  Successfully deleted: %d%s\n", ColorGreen, successCount, ColorReset)
	fmt.Printf("%s  Failed to delete: %d%s\n", ColorRed, failureCount, ColorReset)
	fmt.Printf("%s  Duration: %v%s\n", ColorCyan, duration, ColorReset)
	fmt.Printf("%s  Success rate: %.2f%%%s\n", ColorCyan, float64(successCount)/float64(totalResources)*100, ColorReset)

	if logger != nil {
		result := &deletion.DeletionResult{
			TotalResources:   totalResources,
			DeletedResources: successCount,
			FailedResources:  failureCount,
		}
		logger.LogDeletionComplete(result)
	}

	// Export logs if requested
	if *exportLogs != "" && logger != nil {
		exportLogsToFile(logger, *exportLogs)
	}

	if failureCount == 0 {
		fmt.Printf("\n%sAll resources were successfully deleted!%s\n", ColorGreen+ColorBold, ColorReset)
	} else {
		fmt.Printf("\n%sDeletion completed with %d failures. Check the logs for details.%s\n", ColorYellow+ColorBold, failureCount, ColorReset)
	}
}

// parseDeletionOptions parses CLI flags into DeletionOptions
func parseDeletionOptions() deletion.DeletionOptions {
	var resourceTypesList []string
	if *resourceTypes != "" {
		resourceTypesList = strings.Split(*resourceTypes, ",")
	}

	var regionsList []string
	if *regions != "" {
		regionsList = strings.Split(*regions, ",")
	}

	var excludeList []string
	if *excludeResources != "" {
		excludeList = strings.Split(*excludeResources, ",")
	}

	var includeList []string
	if *includeResources != "" {
		includeList = strings.Split(*includeResources, ",")
	}

	return deletion.DeletionOptions{
		DryRun:           *dryRun,
		Force:            *force,
		ResourceTypes:    resourceTypesList,
		Regions:          regionsList,
		ExcludeResources: excludeList,
		IncludeResources: includeList,
		Timeout:          *timeout,
		BatchSize:        *batchSize,
		MaxRetries:       *maxRetries,
		RetryDelay:       *retryDelay,
		ProgressCallback: func(update deletion.ProgressUpdate) {
			if *verbose {
				fmt.Printf("%s    Progress: %s - %d/%d - %s%s\n",
					ColorDim, update.Type, update.Progress, update.Total, update.Message, ColorReset)
			}
		},
	}
}

// registerProviders registers cloud providers based on CLI flags
func registerProviders(deletionEngine *deletion.DeletionEngine, providers []string) map[string]string {
	registeredProviders := make(map[string]string)

	for _, provider := range providers {
		provider = strings.TrimSpace(provider)
		switch provider {
		case "aws":
			if awsProvider, err := deletion.NewAWSProvider(); err == nil {
				deletionEngine.RegisterProvider("aws", awsProvider)
				registeredProviders["aws"] = "local"
				fmt.Printf("%s  ✓ AWS provider registered successfully%s\n", ColorGreen, ColorReset)
			} else {
				fmt.Printf("%s  ✗ AWS provider registration failed: %v%s\n", ColorRed, err, ColorReset)
			}
		case "azure":
			if azureProvider, err := deletion.NewAzureProvider(); err == nil {
				deletionEngine.RegisterProvider("azure", azureProvider)
				registeredProviders["azure"] = os.Getenv("AZURE_SUBSCRIPTION_ID")
				fmt.Printf("%s  ✓ Azure provider registered successfully%s\n", ColorGreen, ColorReset)
			} else {
				fmt.Printf("%s  ✗ Azure provider registration failed: %v%s\n", ColorRed, err, ColorReset)
			}
		case "gcp":
			if gcpProvider, err := deletion.NewGCPProvider(); err == nil {
				deletionEngine.RegisterProvider("gcp", gcpProvider)
				registeredProviders["gcp"] = os.Getenv("GCP_PROJECT_ID")
				fmt.Printf("%s  ✓ GCP provider registered successfully%s\n", ColorGreen, ColorReset)
			} else {
				fmt.Printf("%s  ✗ GCP provider registration failed: %v%s\n", ColorRed, err, ColorReset)
			}
		}
	}

	return registeredProviders
}

// getAccountInformation gets account information for each provider
func getAccountInformation(registeredProviders map[string]string) map[string]string {
	accountInfo := make(map[string]string)

	for provider, accountID := range registeredProviders {
		if accountID == "" {
			accountID = "local"
			fmt.Printf("%sUsing local %s account (will be auto-detected)%s\n", ColorDim, provider, ColorReset)
		} else {
			fmt.Printf("%sUsing %s account: %s%s\n", ColorDim, provider, accountID, ColorReset)
		}
		accountInfo[provider] = accountID
	}

	return accountInfo
}

// showResourceDependencies shows resource dependencies
func showResourceDependencies(resources []models.Resource) {
	fmt.Printf("\n%s=== Resource Dependencies Analysis ===%s\n", ColorBold+ColorBlue, ColorReset)

	dependencyManager := deletion.NewDependencyManager()
	orderedResources, err := dependencyManager.GetDeletionOrder(resources)
	if err != nil {
		fmt.Printf("%sError analyzing dependencies: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	fmt.Printf("%sDeletion order (dependencies resolved):%s\n", ColorCyan, ColorReset)
	for i, resource := range orderedResources {
		fmt.Printf("%s  %d. %s (%s) - %s%s\n",
			ColorDim, i+1, resource.Name, resource.Type, resource.Region, ColorReset)
	}
}

// performSafetyChecks performs enhanced safety checks
func performSafetyChecks(resources []models.Resource) {
	fmt.Printf("\n%s=== Enhanced Safety Checks ===%s\n", ColorBold+ColorBlue, ColorReset)

	// Check for production resources
	productionCount := 0
	for _, resource := range resources {
		if hasProductionIndicators(resource) {
			productionCount++
		}
	}

	if productionCount > 0 {
		fmt.Printf("%s⚠️  Found %d resources with production indicators%s\n", ColorYellow, productionCount, ColorReset)
		fmt.Printf("%s  Review these resources carefully before proceeding%s\n", ColorYellow, ColorReset)
	}

	// Check for critical resources
	criticalCount := 0
	for _, resource := range resources {
		if isCriticalResource(resource) {
			criticalCount++
		}
	}

	if criticalCount > 0 {
		fmt.Printf("%s⚠️  Found %d critical resources%s\n", ColorRed, criticalCount, ColorReset)
		fmt.Printf("%s  These resources may require special handling%s\n", ColorRed, ColorReset)
	}
}

// hasProductionIndicators checks if a resource has production indicators
func hasProductionIndicators(resource models.Resource) bool {
	name := strings.ToLower(resource.Name)
	productionPatterns := []string{"prod-", "production-", "live-", "critical-", "main-", "primary-"}

	for _, pattern := range productionPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}

	// Check tags
	for key, value := range resource.Tags {
		if strings.Contains(strings.ToLower(key), "environment") &&
			(strings.Contains(strings.ToLower(value), "prod") || strings.Contains(strings.ToLower(value), "production")) {
			return true
		}
	}

	return false
}

// isCriticalResource checks if a resource is critical
func isCriticalResource(resource models.Resource) bool {
	criticalTypes := map[string]bool{
		"iam_user":    true,
		"iam_role":    true,
		"iam_policy":  true,
		"s3_bucket":   true,
		"rds_cluster": true,
		"eks_cluster": true,
		"key_vault":   true,
	}

	return criticalTypes[resource.Type]
}

// confirmDeletion asks for user confirmation
func confirmDeletion(totalResources int) {
	fmt.Printf("\n%sFINAL WARNING: This will delete %d resources across all cloud providers!%s\n",
		ColorRed+ColorBold, totalResources, ColorReset)
	fmt.Printf("%sType 'DELETE' to proceed with actual deletion: %s", ColorYellow, ColorReset)

	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "DELETE" {
		fmt.Printf("%sDeletion cancelled.%s\n", ColorRed, ColorReset)
		os.Exit(0)
	}
}

// exportLogsToFile exports logs in the specified format
func exportLogsToFile(logger *deletion.DeletionLogger, format string) {
	fmt.Printf("\n%s=== Exporting Logs ===%s\n", ColorBold+ColorBlue, ColorReset)

	logData, err := logger.ExportLogs(format)
	if err != nil {
		fmt.Printf("%sError exporting logs: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	filename := fmt.Sprintf("deletion_logs_%s.%s", time.Now().Format("20060102_150405"), format)
	if err := os.WriteFile(filename, logData, 0644); err != nil {
		fmt.Printf("%sError writing log file: %v%s\n", ColorRed, err, ColorReset)
		return
	}

	fmt.Printf("%sLogs exported to: %s%s\n", ColorGreen, filename, ColorReset)
}

// Helper functions
func parseLogLevel(level string) deletion.LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return deletion.DEBUG
	case "warn":
		return deletion.WARN
	case "error":
		return deletion.ERROR
	default:
		return deletion.INFO
	}
}

func groupResourcesByType(resources []models.Resource) map[string]int {
	groups := make(map[string]int)
	for _, resource := range resources {
		groups[resource.Type]++
	}
	return groups
}

func showHelp() {
	fmt.Printf("%sDriftMgr Enhanced Resource Deletion Tool%s\n", ColorBold, ColorReset)
	fmt.Println()
	fmt.Println("Usage: delete_all_resources.exe [options]")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  delete_all_resources.exe --dry-run")
	fmt.Println("  delete_all_resources.exe --resource-types ec2_instance,s3_bucket --regions us-east-1")
	fmt.Println("  delete_all_resources.exe --providers aws,azure --force --no-confirm")
	fmt.Println("  delete_all_resources.exe --max-retries 5 --retry-delay 10s --verbose")
}

// previewProviderResources previews resources for a specific provider
func previewProviderResources(deletionEngine *deletion.DeletionEngine, provider, accountID string, options deletion.DeletionOptions) ([]models.Resource, error) {
	previewOptions := deletion.DeletionOptions{
		DryRun:        true,
		Force:         false,
		ResourceTypes: []string{},
		Regions:       []string{},
		Timeout:       5 * time.Minute,
		BatchSize:     10,
	}

	_, err := deletionEngine.DeleteAccountResources(context.Background(), provider, accountID, previewOptions)
	if err != nil {
		return nil, err
	}

	// Get detailed resources for preview
	providerImpl, exists := deletionEngine.GetProvider(provider)
	if !exists {
		return nil, fmt.Errorf("provider %s not found", provider)
	}

	return providerImpl.ListResources(context.Background(), accountID)
}

// deleteProviderResources deletes resources for a specific provider
func deleteProviderResources(deletionEngine *deletion.DeletionEngine, provider, accountID string, options deletion.DeletionOptions, logger *deletion.DeletionLogger) (*deletion.DeletionResult, error) {
	deletionOptions := deletion.DeletionOptions{
		DryRun:        false,
		Force:         true,
		ResourceTypes: []string{},
		Regions:       []string{},
		Timeout:       15 * time.Minute,
		BatchSize:     10,
		MaxRetries:    options.MaxRetries,
		RetryDelay:    options.RetryDelay,
		ProgressCallback: func(update deletion.ProgressUpdate) {
			fmt.Printf("%s    Progress: %s - %d/%d - %s%s\n", ColorDim, update.Type, update.Progress, update.Total, update.Message, ColorReset)
		},
	}

	return deletionEngine.DeleteAccountResources(context.Background(), provider, accountID, deletionOptions)
}

// getProviderResources filters resources by provider
func getProviderResources(resources []models.Resource, provider string) []models.Resource {
	var filtered []models.Resource
	for _, resource := range resources {
		if resource.Provider == provider {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

// printDetailedResourcePreview prints detailed information about resources
func printDetailedResourcePreview(resources []models.Resource) {
	if len(resources) == 0 {
		fmt.Printf("%sNo resources found.%s\n", ColorYellow, ColorReset)
		return
	}

	// Group resources by provider and type
	providerGroups := make(map[string]map[string][]models.Resource)
	for _, resource := range resources {
		if providerGroups[resource.Provider] == nil {
			providerGroups[resource.Provider] = make(map[string][]models.Resource)
		}
		providerGroups[resource.Provider][resource.Type] = append(providerGroups[resource.Provider][resource.Type], resource)
	}

	// Print summary by provider and resource type
	fmt.Printf("%sResource Summary by Provider:%s\n", ColorBold+ColorCyan, ColorReset)
	for provider, typeGroups := range providerGroups {
		fmt.Printf("\n%s  %s:%s\n", ColorBold, strings.ToUpper(provider), ColorReset)
		for resourceType, resources := range typeGroups {
			fmt.Printf("%s    - %s: %d resources%s\n", ColorDim, formatResourceType(resourceType), len(resources), ColorReset)
		}
	}

	fmt.Printf("\n%sAffected Resources:%s\n", ColorBold+ColorCyan, ColorReset)

	// Check for Terraform/Terragrunt state files
	terraformStateFiles := []models.Resource{}
	terragruntStateFiles := []models.Resource{}

	// Group resources by type for more concise display
	resourceGroups := make(map[string][]models.Resource)
	for _, resource := range resources {
		resourceGroups[resource.Type] = append(resourceGroups[resource.Type], resource)

		// Check if this is a Terraform/Terragrunt state file
		if isTerraformStateFile(resource) {
			terraformStateFiles = append(terraformStateFiles, resource)
		}
		if isTerragruntStateFile(resource) {
			terragruntStateFiles = append(terragruntStateFiles, resource)
		}
	}

	// Display resources grouped by type
	for resourceType, typeResources := range resourceGroups {
		fmt.Printf("%s  %s (%d):%s\n", ColorBold, formatResourceType(resourceType), len(typeResources), ColorReset)
		for _, resource := range typeResources {
			fmt.Printf("%s    - %s [%s]%s\n", ColorDim, resource.Name, resource.Region, ColorReset)
		}
	}

	// Special warnings for Terraform/Terragrunt state files
	if len(terraformStateFiles) > 0 {
		fmt.Printf("\n%sWARNING: Terraform State Files Detected (%d)%s\n", ColorRed+ColorBold, len(terraformStateFiles), ColorReset)
		for _, resource := range terraformStateFiles {
			fmt.Printf("%s  - %s [%s]%s\n", ColorYellow, resource.Name, resource.Region, ColorReset)
		}
	}

	if len(terragruntStateFiles) > 0 {
		fmt.Printf("\n%sWARNING: Terragrunt State Files Detected (%d)%s\n", ColorRed+ColorBold, len(terragruntStateFiles), ColorReset)
		for _, resource := range terragruntStateFiles {
			fmt.Printf("%s  - %s [%s]%s\n", ColorYellow, resource.Name, resource.Region, ColorReset)
		}
	}

	// Print cost implications
	fmt.Printf("\n%sNote:%s Deleting these resources will stop all associated charges.%s\n", ColorCyan, ColorReset, ColorDim)
}

// isTerraformStateFile checks if a resource is a Terraform state file
func isTerraformStateFile(resource models.Resource) bool {
	name := strings.ToLower(resource.Name)
	return strings.Contains(name, "terraform") &&
		(strings.Contains(name, "state") || strings.Contains(name, "tfstate"))
}

// isTerragruntStateFile checks if a resource is a Terragrunt state file
func isTerragruntStateFile(resource models.Resource) bool {
	name := strings.ToLower(resource.Name)
	return strings.Contains(name, "terragrunt") &&
		(strings.Contains(name, "state") || strings.Contains(name, "tfstate"))
}

// formatResourceType formats resource type for display
func formatResourceType(resourceType string) string {
	switch resourceType {
	case "ec2_instance":
		return "EC2 Instance"
	case "s3_bucket":
		return "S3 Bucket"
	case "rds_instance":
		return "RDS Instance"
	case "lambda_function":
		return "Lambda Function"
	case "eks_cluster":
		return "EKS Cluster"
	case "ecs_cluster":
		return "ECS Cluster"
	case "dynamodb_table":
		return "DynamoDB Table"
	case "elasticache_cluster":
		return "ElastiCache Cluster"
	case "sns_topic":
		return "SNS Topic"
	case "sqs_queue":
		return "SQS Queue"
	case "iam_role":
		return "IAM Role"
	case "iam_policy":
		return "IAM Policy"
	case "iam_user":
		return "IAM User"
	case "route53_hosted_zone":
		return "Route53 Hosted Zone"
	case "cloudformation_stack":
		return "CloudFormation Stack"
	case "virtual_machine":
		return "Virtual Machine"
	case "storage_account":
		return "Storage Account"
	case "virtual_network":
		return "Virtual Network"
	case "load_balancer":
		return "Load Balancer"
	case "app_service":
		return "App Service"
	case "key_vault":
		return "Key Vault"
	case "compute_instance":
		return "Compute Instance"
	case "storage_bucket":
		return "Storage Bucket"
	case "kubernetes_cluster":
		return "Kubernetes Cluster"
	case "sql_instance":
		return "SQL Instance"
	case "pubsub_topic":
		return "Pub/Sub Topic"
	case "cloud_function":
		return "Cloud Function"
	default:
		return strings.Title(strings.ReplaceAll(resourceType, "_", " "))
	}
}

// formatTags formats resource tags for display
func formatTags(tags map[string]string) string {
	if len(tags) == 0 {
		return "none"
	}

	var tagPairs []string
	for key, value := range tags {
		tagPairs = append(tagPairs, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(tagPairs, ", ")
}
