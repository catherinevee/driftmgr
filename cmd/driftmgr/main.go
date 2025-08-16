package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

const (
	serverPort = "8080"
	serverURL  = "http://localhost:" + serverPort + "/health"
)

func main() {
	// Check for generic resource deletion command
	if len(os.Args) > 1 && os.Args[1] == "delete-resource" {
		handleResourceDeletion(os.Args[2:])
		return
	}

	// Get the directory where this executable is located
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}

	exeDir := filepath.Dir(exePath)

	// Check if server is running
	if !isServerRunning() {
		fmt.Println("DriftMgr server is not running. Starting server...")
		if err := startServer(exeDir); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
			fmt.Println("Continuing without server (some features may not work)...")
		} else {
			fmt.Println("Server started successfully!")
		}
	}

	// Path to the driftmgr-client executable
	clientPath := findClientExecutable(exeDir)

	// Create command to run the client
	cmd := exec.Command(clientPath, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the client
	err = cmd.Run()
	if err != nil {
		os.Exit(1)
	}
}

// isServerRunning checks if the DriftMgr server is running
func isServerRunning() bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(serverURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// startServer starts the DriftMgr server in the background
func startServer(exeDir string) error {
	// Determine the server executable name based on OS
	serverExe := "driftmgr-server.exe"
	if runtime.GOOS != "windows" {
		serverExe = "driftmgr-server"
	}

	// Try to find the server executable
	serverPath := filepath.Join(exeDir, "bin", serverExe)
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		// Try relative to current directory
		serverPath = filepath.Join("bin", serverExe)
		if _, err := os.Stat(serverPath); os.IsNotExist(err) {
			return fmt.Errorf("server executable not found: %s", serverExe)
		}
	}

	// Start the server in the background
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// On Windows, use start command to run in background
		cmd = exec.Command("cmd", "/C", "start", "/B", serverPath)
	} else {
		// On Unix systems, use nohup to run in background
		cmd = exec.Command("nohup", serverPath, "&")
	}

	// Set working directory to the bin directory
	cmd.Dir = filepath.Dir(serverPath)

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	// Wait a moment for the server to start
	time.Sleep(2 * time.Second)

	// Check if server is now running
	if !isServerRunning() {
		return fmt.Errorf("server started but health check failed")
	}

	return nil
}

// findClientExecutable finds the driftmgr-client executable
func findClientExecutable(exeDir string) string {
	// Determine the client executable name based on OS
	clientExe := "driftmgr-client.exe"
	if runtime.GOOS != "windows" {
		clientExe = "driftmgr-client"
	}

	// Path to the driftmgr-client executable
	clientPath := filepath.Join(exeDir, "bin", clientExe)

	// Check if the client executable exists
	if _, err := os.Stat(clientPath); os.IsNotExist(err) {
		// If not found, try relative to current directory
		clientPath = filepath.Join("bin", clientExe)
		if _, err := os.Stat(clientPath); os.IsNotExist(err) {
			// Last resort: try to find it in PATH
			clientPath = "driftmgr-client"
		}
	}

	return clientPath
}

// handleResourceDeletion handles the delete-resource command with generic dependency management
func handleResourceDeletion(args []string) {
	if len(args) < 1 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Println("Usage: driftmgr delete-resource [<resource-type> <resource-name>] [options]")
		fmt.Println()
		fmt.Println("Description:")
		fmt.Println("  Delete a cloud resource with automatic dependency management.")
		fmt.Println("  This command ensures proper deletion order and validates resource state.")
		fmt.Println("  If no resource type/name is provided, will discover and let you select resources.")
		fmt.Println()
		fmt.Println("Arguments:")
		fmt.Println("  <resource-type>    Type of resource to delete (e.g., eks_cluster, ecs_cluster, rds_instance)")
		fmt.Println("  <resource-name>    Name of the resource to delete")
		fmt.Println("                     (If omitted, will discover and show available resources)")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --region <region>     AWS region (default: us-east-1)")
		fmt.Println("  --force               Skip validation and force deletion")
		fmt.Println("  --dry-run             Show what would be deleted without actually deleting")
		fmt.Println("  --include-deps        Include dependent resources")
		fmt.Println("  --wait                Wait for deletion to complete")
		fmt.Println("  --discover            Force resource discovery and selection")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  driftmgr delete-resource eks_cluster prod-use1-eks-main")
		fmt.Println("  driftmgr delete-resource rds_instance my-database --region us-east-1 --dry-run")
		fmt.Println("  driftmgr delete-resource ecs_cluster my-cluster --include-deps --force")
		fmt.Println("  driftmgr delete-resource --discover  # Interactive resource selection")
		fmt.Println()
		fmt.Println("Supported Resource Types:")
		fmt.Println()
		fmt.Println("Complex Resources (with dependencies):")
		fmt.Println("  - eks_cluster: EKS clusters (handles nodegroups)")
		fmt.Println("  - ecs_cluster: ECS clusters (handles services)")
		fmt.Println("  - rds_instance: RDS instances (handles snapshots)")
		fmt.Println("  - vpc: VPCs (handles gateways, route tables)")
		fmt.Println("  - ec2_instance: EC2 instances (handles volumes, security groups)")
		fmt.Println("  - elasticache_cluster: ElastiCache clusters")
		fmt.Println("  - load_balancer: Load balancers")
		fmt.Println("  - lambda_function: Lambda functions (handles IAM roles)")
		fmt.Println("  - api_gateway: API Gateway (handles integrations)")
		fmt.Println("  - cloudfront_distribution: CloudFront distributions")
		fmt.Println("  - elasticsearch_domain: OpenSearch/Elasticsearch domains")
		fmt.Println("  - redshift_cluster: Redshift clusters")
		fmt.Println("  - emr_cluster: EMR clusters")
		fmt.Println("  - msk_cluster: MSK (Kafka) clusters")
		fmt.Println("  - neptune_cluster: Neptune graph databases")
		fmt.Println("  - docdb_cluster: DocumentDB clusters")
		fmt.Println("  - aurora_cluster: Aurora clusters")
		fmt.Println("  - elastic_beanstalk_environment: Elastic Beanstalk environments")
		fmt.Println("  - sagemaker_notebook_instance: SageMaker notebook instances")
		fmt.Println("  - transit_gateway: Transit Gateways")
		fmt.Println()
		fmt.Println("Simple Resources (no dependencies):")
		fmt.Println("  - s3_bucket: S3 buckets")
		fmt.Println("  - dynamodb_table: DynamoDB tables")
		fmt.Println("  - sqs_queue: SQS queues")
		fmt.Println("  - sns_topic: SNS topics")
		fmt.Println("  - cloudwatch_log_group: CloudWatch log groups")
		fmt.Println("  - cloudwatch_alarm: CloudWatch alarms")
		fmt.Println("  - kms_key: KMS keys")
		fmt.Println("  - secretsmanager_secret: Secrets Manager secrets")
		fmt.Println("  - ssm_parameter: Systems Manager parameters")
		fmt.Println("  - ecr_repository: ECR repositories")
		fmt.Println("  - codecommit_repository: CodeCommit repositories")
		fmt.Println("  - route53_zone: Route53 hosted zones")
		fmt.Println("  - route53_record: Route53 records")
		fmt.Println("  - acm_certificate: ACM certificates")
		fmt.Println("  - waf_web_acl: WAF web ACLs")
		fmt.Println("  - guardduty_detector: GuardDuty detectors")
		fmt.Println("  - backup_vault: Backup vaults")
		fmt.Println("  - glue_job: Glue jobs")
		fmt.Println("  - athena_workgroup: Athena workgroups")
		fmt.Println("  - quicksight_dashboard: QuickSight dashboards")
		fmt.Println("  - cognito_user_pool: Cognito user pools")
		fmt.Println("  - amplify_app: Amplify applications")
		fmt.Println("  - pinpoint_app: Pinpoint applications")
		fmt.Println("  - s3_object: S3 objects")
		fmt.Println("  - ebs_volume: EBS volumes")
		fmt.Println("  - ebs_snapshot: EBS snapshots")
		fmt.Println("  - ami: Amazon Machine Images")
		fmt.Println("  - elastic_ip: Elastic IP addresses")
		fmt.Println("  - key_pair: EC2 key pairs")
		fmt.Println("  - customer_gateway: Customer gateways")
		fmt.Println("  - dhcp_options: DHCP option sets")
		fmt.Println("  - flow_log: VPC flow logs")
		fmt.Println("  - network_acl: Network ACLs")
		fmt.Println("  - peering_connection: VPC peering connections")
		fmt.Println()
		fmt.Println("And many more... (see full list in documentation)")
		fmt.Println()
		fmt.Println("Safety Features:")
		fmt.Println("  - Validates resource state before deletion")
		fmt.Println("  - Checks for production/critical indicators")
		fmt.Println("  - Handles dependencies in correct order")
		fmt.Println("  - Waits for deletion completion")
		os.Exit(0)
	}

	// Check if we should discover resources
	shouldDiscover := false
	forceDiscover := false
	
	// Parse options first to check for --discover flag
	for i := 0; i < len(args); i++ {
		if args[i] == "--discover" {
			forceDiscover = true
			// Remove the --discover flag from args
			args = append(args[:i], args[i+1:]...)
			break
		}
	}

	// If no resource type/name provided or --discover flag used, discover resources
	if len(args) < 2 || forceDiscover {
		shouldDiscover = true
	}

	if shouldDiscover {
		handleInteractiveResourceSelection(args)
		return
	}

	resourceType := args[0]
	resourceName := args[1]
	region := "us-east-1"
	force := false
	dryRun := false
	includeDeps := false
	wait := false

	// Parse options
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--region":
			if i+1 < len(args) {
				region = args[i+1]
				i++
			}
		case "--force":
			force = true
		case "--dry-run":
			dryRun = true
		case "--include-deps":
			includeDeps = true
		case "--wait":
			wait = true
		}
	}

	fmt.Printf("=== Resource Deletion with Dependency Management ===\n")
	fmt.Printf("Resource Type: %s\n", resourceType)
	fmt.Printf("Resource Name: %s\n", resourceName)
	fmt.Printf("Region: %s\n", region)
	fmt.Printf("Force: %v\n", force)
	fmt.Printf("Dry Run: %v\n", dryRun)
	fmt.Printf("Include Dependencies: %v\n", includeDeps)
	fmt.Printf("Wait for Completion: %v\n", wait)
	fmt.Println()

	// Use the enhanced deletion tool
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}

	exeDir := filepath.Dir(exePath)
	deleteToolPath := filepath.Join(exeDir, "delete_all_resources.exe")

	// Build command arguments
	cmdArgs := []string{
		"--resource-types", resourceType,
		"--regions", region,
		"--include", resourceName,
	}

	if dryRun {
		cmdArgs = append(cmdArgs, "--dry-run")
	}

	if force {
		cmdArgs = append(cmdArgs, "--force")
	}

	if includeDeps {
		// Get common dependencies for the resource type
		deps := getCommonDependencies(resourceType)
		if len(deps) > 0 {
			cmdArgs = append(cmdArgs, "--resource-types", strings.Join(deps, ","))
		}
	}

	// Run the deletion tool
	cmd := exec.Command(deleteToolPath, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Executing: %s %s\n", deleteToolPath, strings.Join(cmdArgs, " "))
	fmt.Println()

	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Resource deletion failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nResource deletion completed successfully!\n")
}

// handleInteractiveResourceSelection provides an interactive interface for resource discovery and selection
func handleInteractiveResourceSelection(args []string) {
	fmt.Println("=== Interactive Resource Discovery and Selection ===")
	fmt.Println()

	// Parse options for discovery
	region := "us-east-1"
	provider := "aws"
	
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--region":
			if i+1 < len(args) {
				region = args[i+1]
				i++
			}
		}
	}

	fmt.Printf("Discovering resources in %s region for %s provider...\n", region, provider)
	fmt.Println()

	// Discover resources using the driftmgr client
	resources, err := discoverResources(provider, []string{region})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to discover resources: %v\n", err)
		fmt.Println()
		fmt.Println("Troubleshooting:")
		fmt.Println("1. Make sure the driftmgr server is running")
		fmt.Println("2. Try starting the server: driftmgr-server")
		fmt.Println("3. Or use direct resource deletion: driftmgr delete-resource <type> <name>")
		fmt.Println("4. Check if the server supports enhanced discovery")
		fmt.Println()
		
		// Offer fallback to direct deletion
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Would you like to proceed with direct resource deletion? (y/N): ")
		fallbackInput, _ := reader.ReadString('\n')
		fallback := strings.ToLower(strings.TrimSpace(fallbackInput)) == "y"
		
		if fallback {
			fmt.Println()
			fmt.Println("Please provide the resource type and name:")
			fmt.Print("Resource type (e.g., eks_cluster, rds_instance): ")
			resourceTypeInput, _ := reader.ReadString('\n')
			resourceType := strings.TrimSpace(resourceTypeInput)
			
			fmt.Print("Resource name: ")
			resourceNameInput, _ := reader.ReadString('\n')
			resourceName := strings.TrimSpace(resourceNameInput)
			
			if resourceType != "" && resourceName != "" {
				// Build direct deletion arguments
				deleteArgs := []string{resourceType, resourceName, "--region", region}
				handleResourceDeletion(deleteArgs)
				return
			}
		}
		
		fmt.Println("Operation cancelled.")
		os.Exit(1)
	}

	if len(resources) == 0 {
		fmt.Println("No resources found in the specified region.")
		return
	}

	// Group resources by type
	resourceGroups := groupResourcesByType(resources)
	
	// Display available resources
	fmt.Printf("Found %d resources across %d types:\n\n", len(resources), len(resourceGroups))
	
	resourceMap := make(map[int]models.Resource)
	counter := 1

	for resourceType, typeResources := range resourceGroups {
		complexity := getResourceComplexity(resourceType)
		fmt.Printf("=== %s (%d resources) [%s] ===\n", strings.ToUpper(resourceType), len(typeResources), complexity)
		for _, resource := range typeResources {
			fmt.Printf("%3d. %s (%s)\n", counter, resource.Name, resource.ID)
			resourceMap[counter] = resource
			counter++
		}
		fmt.Println()
	}

	// Get user selection
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter the number of the resource to delete (or 'q' to quit): ")
	selection, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	selection = strings.TrimSpace(selection)
	if selection == "q" || selection == "quit" {
		fmt.Println("Operation cancelled.")
		return
	}

	resourceNum, err := strconv.Atoi(selection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid selection: %s\n", selection)
		os.Exit(1)
	}

	selectedResource, exists := resourceMap[resourceNum]
	if !exists {
		fmt.Fprintf(os.Stderr, "Invalid resource number: %d\n", resourceNum)
		os.Exit(1)
	}

	complexity := getResourceComplexity(selectedResource.Type)
	fmt.Printf("\nSelected resource: %s (%s)\n", selectedResource.Name, selectedResource.Type)
	fmt.Printf("Resource ID: %s\n", selectedResource.ID)
	fmt.Printf("Region: %s\n", selectedResource.Region)
	fmt.Printf("Complexity: %s\n", complexity)
	fmt.Println()

	// Get deletion options
	includeDeps := false
	if complexity == "Complex" {
		fmt.Print("Include dependencies? (y/N): ")
		includeDepsInput, _ := reader.ReadString('\n')
		includeDeps = strings.ToLower(strings.TrimSpace(includeDepsInput)) == "y"
	} else {
		fmt.Println("Simple resource - no dependencies to include")
	}

	fmt.Print("Force deletion (skip validation)? (y/N): ")
	forceInput, _ := reader.ReadString('\n')
	force := strings.ToLower(strings.TrimSpace(forceInput)) == "y"

	fmt.Print("Dry run (show what would be deleted)? (y/N): ")
	dryRunInput, _ := reader.ReadString('\n')
	dryRun := strings.ToLower(strings.TrimSpace(dryRunInput)) == "y"

	fmt.Print("Wait for completion? (Y/n): ")
	waitInput, _ := reader.ReadString('\n')
	wait := strings.ToLower(strings.TrimSpace(waitInput)) != "n"

	fmt.Println()

	// Confirm deletion
	fmt.Printf("About to delete: %s (%s)\n", selectedResource.Name, selectedResource.Type)
	if includeDeps {
		fmt.Println("Will include dependent resources")
	}
	if force {
		fmt.Println("Force deletion enabled (skipping validation)")
	}
	if dryRun {
		fmt.Println("DRY RUN MODE - No actual deletion will occur")
	}
	fmt.Print("Proceed? (y/N): ")

	confirmInput, _ := reader.ReadString('\n')
	confirm := strings.ToLower(strings.TrimSpace(confirmInput)) == "y"

	if !confirm {
		fmt.Println("Deletion cancelled.")
		return
	}

	// Build deletion arguments
	deleteArgs := []string{selectedResource.Type, selectedResource.Name}
	
	if includeDeps {
		deleteArgs = append(deleteArgs, "--include-deps")
	}
	if force {
		deleteArgs = append(deleteArgs, "--force")
	}
	if dryRun {
		deleteArgs = append(deleteArgs, "--dry-run")
	}
	if wait {
		deleteArgs = append(deleteArgs, "--wait")
	}
	deleteArgs = append(deleteArgs, "--region", selectedResource.Region)

	// Call the deletion function
	handleResourceDeletion(deleteArgs)
}

// discoverResources calls the driftmgr discovery API to find resources
func discoverResources(provider string, regions []string) ([]models.Resource, error) {
	// First check if server is running
	if !isServerRunning() {
		return nil, fmt.Errorf("driftmgr server is not running. Please start the server first or use direct resource deletion")
	}

	// Create discovery request
	discoveryReq := map[string]interface{}{
		"provider": provider,
		"regions":  regions,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(discoveryReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal discovery request: %v", err)
	}

	// Make HTTP request to discovery API
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(
		"http://localhost:8080/api/v1/enhanced-discover",
		"application/json",
		strings.NewReader(string(jsonData)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call discovery API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("discovery API endpoint not found (404). The server may not support enhanced discovery")
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery API returned status %d", resp.StatusCode)
	}

	// Parse response
	var discoveryResp struct {
		Resources []models.Resource `json:"resources"`
		Error     string            `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&discoveryResp); err != nil {
		return nil, fmt.Errorf("failed to decode discovery response: %v", err)
	}

	if discoveryResp.Error != "" {
		return nil, fmt.Errorf("discovery error: %s", discoveryResp.Error)
	}

	return discoveryResp.Resources, nil
}

// groupResourcesByType groups resources by their type for better organization
func groupResourcesByType(resources []models.Resource) map[string][]models.Resource {
	groups := make(map[string][]models.Resource)
	
	for _, resource := range resources {
		groups[resource.Type] = append(groups[resource.Type], resource)
	}
	
	return groups
}

// getResourceComplexity returns whether a resource type is complex (has dependencies) or simple
func getResourceComplexity(resourceType string) string {
	complexResources := map[string]bool{
		"eks_cluster": true, "ecs_cluster": true, "rds_instance": true, "vpc": true,
		"ec2_instance": true, "elasticache_cluster": true, "load_balancer": true,
		"lambda_function": true, "api_gateway": true, "cloudfront_distribution": true,
		"elasticsearch_domain": true, "redshift_cluster": true, "emr_cluster": true,
		"msk_cluster": true, "opensearch_domain": true, "neptune_cluster": true,
		"docdb_cluster": true, "aurora_cluster": true, "elastic_beanstalk_environment": true,
		"ecs_service": true, "ecs_task_definition": true, "autoscaling_group": true,
		"launch_template": true, "target_group": true, "nat_gateway": true,
		"internet_gateway": true, "route_table": true, "subnet": true, "security_group": true,
		"iam_role": true, "iam_policy": true, "iam_user": true, "iam_group": true,
		"sns_topic": true, "sns_subscription": true, "codepipeline": true,
		"codebuild_project": true, "codedeploy_application": true, "codedeploy_deployment_group": true,
		"route53_zone": true, "route53_record": true, "backup_vault": true, "backup_plan": true,
		"glue_job": true, "glue_crawler": true, "sagemaker_notebook_instance": true,
		"sagemaker_endpoint": true, "sagemaker_model": true, "transit_gateway": true,
		"transit_gateway_attachment": true, "vpn_connection": true, "vpn_gateway": true,
		"appsync_graphql_api": true, "cognito_identity_pool": true, "endpoint": true,
		"network_acl": true, "transit_gateway_route_table": true, "transit_gateway_vpc_attachment": true,
	}
	
	if complexResources[resourceType] {
		return "Complex"
	}
	return "Simple"
}

// getCommonDependencies returns common dependency types for a resource type
func getCommonDependencies(resourceType string) []string {
	dependencyMap := map[string][]string{
		// Complex resources with dependencies
		"eks_cluster":        {"eks_cluster", "autoscaling_group", "iam_role", "security_group", "subnet", "vpc"},
		"ecs_cluster":        {"ecs_cluster", "ecs_service", "iam_role", "security_group", "subnet", "vpc"},
		"rds_instance":       {"rds_instance", "rds_snapshot", "security_group", "subnet", "vpc"},
		"vpc":               {"vpc", "nat_gateway", "internet_gateway", "route_table", "subnet", "security_group"},
		"ec2_instance":      {"ec2_instance", "ebs_volume", "security_group", "iam_role"},
		"elasticache_cluster": {"elasticache_cluster", "security_group", "subnet", "vpc"},
		"load_balancer":     {"load_balancer", "target_group", "security_group", "subnet", "vpc"},
		"lambda_function":   {"lambda_function", "iam_role", "cloudwatch_log_group"},
		"api_gateway":       {"api_gateway", "lambda_function", "iam_role", "cloudwatch_log_group"},
		"cloudfront_distribution": {"cloudfront_distribution", "s3_bucket", "iam_role"},
		"elasticsearch_domain": {"elasticsearch_domain", "security_group", "subnet", "vpc"},
		"redshift_cluster":  {"redshift_cluster", "security_group", "subnet", "vpc", "iam_role"},
		"emr_cluster":       {"emr_cluster", "ec2_instance", "security_group", "subnet", "vpc", "iam_role"},
		"msk_cluster":       {"msk_cluster", "security_group", "subnet", "vpc"},
		"opensearch_domain": {"opensearch_domain", "security_group", "subnet", "vpc"},
		"neptune_cluster":   {"neptune_cluster", "security_group", "subnet", "vpc"},
		"docdb_cluster":     {"docdb_cluster", "security_group", "subnet", "vpc"},
		"aurora_cluster":    {"aurora_cluster", "rds_instance", "security_group", "subnet", "vpc"},
		"elastic_beanstalk_environment": {"elastic_beanstalk_environment", "ec2_instance", "security_group", "subnet", "vpc", "iam_role"},
		"ecs_service":       {"ecs_service", "ecs_task_definition", "iam_role", "security_group"},
		"ecs_task_definition": {"ecs_task_definition", "iam_role"},
		"autoscaling_group": {"autoscaling_group", "launch_template", "iam_role"},
		"launch_template":   {"launch_template", "iam_role"},
		"target_group":      {"target_group", "load_balancer"},
		"nat_gateway":       {"nat_gateway", "subnet", "vpc"},
		"internet_gateway":  {"internet_gateway", "vpc"},
		"route_table":       {"route_table", "vpc"},
		"subnet":            {"subnet", "vpc"},
		"security_group":    {"security_group", "vpc"},
		"iam_role":          {"iam_role", "iam_policy"},
		"iam_policy":        {"iam_policy"},
		"iam_user":          {"iam_user", "iam_access_key"},
		"iam_group":         {"iam_group"},
		"cloudwatch_log_group": {"cloudwatch_log_group"},
		"cloudwatch_alarm":  {"cloudwatch_alarm"},
		"cloudwatch_dashboard": {"cloudwatch_dashboard"},
		"sns_topic":         {"sns_topic", "sns_subscription"},
		"sns_subscription":  {"sns_subscription"},
		"sqs_queue":         {"sqs_queue"},
		"dynamodb_table":    {"dynamodb_table"},
		"s3_bucket":         {"s3_bucket"},
		"kms_key":           {"kms_key"},
		"secretsmanager_secret": {"secretsmanager_secret"},
		"ssm_parameter":     {"ssm_parameter"},
		"ecr_repository":    {"ecr_repository"},
		"ecr_image":         {"ecr_image"},
		"codecommit_repository": {"codecommit_repository"},
		"codepipeline":      {"codepipeline", "codebuild_project"},
		"codebuild_project": {"codebuild_project", "iam_role"},
		"codedeploy_application": {"codedeploy_application", "codedeploy_deployment_group"},
		"codedeploy_deployment_group": {"codedeploy_deployment_group"},
		"cloudformation_stack": {"cloudformation_stack"},
		"route53_zone":      {"route53_zone", "route53_record"},
		"route53_record":    {"route53_record"},
		"acm_certificate":   {"acm_certificate"},
		"waf_web_acl":       {"waf_web_acl"},
		"wafv2_web_acl":     {"wafv2_web_acl"},
		"shield_protection": {"shield_protection"},
		"guardduty_detector": {"guardduty_detector"},
		"config_recorder":   {"config_recorder"},
		"config_rule":       {"config_rule"},
		"backup_vault":      {"backup_vault", "backup_plan"},
		"backup_plan":       {"backup_plan"},
		"glue_job":          {"glue_job", "iam_role"},
		"glue_crawler":      {"glue_crawler", "iam_role"},
		"athena_workgroup":  {"athena_workgroup"},
		"quicksight_dashboard": {"quicksight_dashboard"},
		"sagemaker_notebook_instance": {"sagemaker_notebook_instance", "iam_role", "security_group", "subnet", "vpc"},
		"sagemaker_endpoint": {"sagemaker_endpoint", "sagemaker_model"},
		"sagemaker_model":   {"sagemaker_model"},
		"transit_gateway":   {"transit_gateway", "transit_gateway_attachment"},
		"transit_gateway_attachment": {"transit_gateway_attachment"},
		"vpn_connection":    {"vpn_connection", "vpn_gateway"},
		"vpn_gateway":       {"vpn_gateway"},
		"direct_connect_connection": {"direct_connect_connection"},
		"direct_connect_gateway": {"direct_connect_gateway"},
		"appsync_graphql_api": {"appsync_graphql_api", "iam_role"},
		"amplify_app":       {"amplify_app"},
		"cognito_user_pool": {"cognito_user_pool"},
		"cognito_identity_pool": {"cognito_identity_pool", "iam_role"},
		"pinpoint_app":      {"pinpoint_app"},
		"s3_object":         {"s3_object"},
		"ebs_volume":        {"ebs_volume"},
		"ebs_snapshot":      {"ebs_snapshot"},
		"ami":               {"ami"},
		"elastic_ip":        {"elastic_ip"},
		"network_interface": {"network_interface"},
		"placement_group":   {"placement_group"},
		"key_pair":          {"key_pair"},
		"customer_gateway":  {"customer_gateway"},
		"dhcp_options":      {"dhcp_options"},
		"endpoint":          {"endpoint", "vpc"},
		"flow_log":          {"flow_log"},
		"network_acl":       {"network_acl", "vpc"},
		"peering_connection": {"peering_connection"},
		"transit_gateway_route_table": {"transit_gateway_route_table"},
		"transit_gateway_vpc_attachment": {"transit_gateway_vpc_attachment"},
	}

	if deps, exists := dependencyMap[resourceType]; exists {
		return deps
	}
	return []string{resourceType}
}
