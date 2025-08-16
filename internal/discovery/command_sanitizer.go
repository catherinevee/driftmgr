package discovery

import (
	"fmt"
	"regexp"
	"strings"
)

// CommandSanitizer provides input validation and sanitization for CLI commands
type CommandSanitizer struct {
	allowedCommands map[string]bool
	allowedFlags    map[string]bool
	allowedValues   map[string]*regexp.Regexp
}

// NewCommandSanitizer creates a new command sanitizer
func NewCommandSanitizer() *CommandSanitizer {
	sanitizer := &CommandSanitizer{
		allowedCommands: make(map[string]bool),
		allowedFlags:    make(map[string]bool),
		allowedValues:   make(map[string]*regexp.Regexp),
	}

	// Initialize allowed commands
	sanitizer.initializeAllowedCommands()
	sanitizer.initializeAllowedFlags()
	sanitizer.initializeAllowedValues()

	return sanitizer
}

// initializeAllowedCommands sets up allowed CLI commands
func (cs *CommandSanitizer) initializeAllowedCommands() {
	// AWS CLI commands
	awsCommands := []string{
		"ec2", "describe-instances", "describe-vpcs", "describe-subnets", "describe-security-groups",
		"s3", "ls", "api", "list-buckets", "list-objects",
		"rds", "describe-db-instances", "describe-db-clusters",
		"lambda", "list-functions", "list-layers",
		"ecs", "list-clusters", "list-services", "list-tasks",
		"eks", "list-clusters", "list-nodegroups",
		"dynamodb", "list-tables", "describe-table",
		"iam", "list-users", "list-roles", "list-policies",
		"cloudwatch", "list-metrics", "describe-alarms",
		"autoscaling", "describe-auto-scaling-groups",
		"route53", "list-hosted-zones", "list-resource-record-sets",
		"sqs", "list-queues", "get-queue-attributes",
		"sns", "list-topics", "list-subscriptions",
		"elasticache", "describe-cache-clusters",
		"cloudformation", "list-stacks", "describe-stacks",
	}

	// Azure CLI commands
	azureCommands := []string{
		"vm", "list", "show", "get-instance-view",
		"network", "vnet", "list", "show", "subnet", "list",
		"storage", "account", "list", "show", "container", "list",
		"sql", "server", "list", "database", "list",
		"functionapp", "list", "show",
		"webapp", "list", "show",
		"aks", "list", "show",
		"cosmosdb", "list", "show",
		"monitor", "metrics", "list",
		"keyvault", "list", "show",
		"resource", "list", "show",
		"group", "list", "show",
	}

	// GCP CLI commands
	gcpCommands := []string{
		"compute", "instances", "list", "describe",
		"network", "networks", "list", "subnets", "list",
		"storage", "ls", "buckets", "list",
		"sql", "instances", "list", "databases", "list",
		"functions", "list", "describe",
		"run", "services", "list", "revisions", "list",
		"container", "clusters", "list", "describe",
		"bigquery", "datasets", "list", "tables", "list",
		"monitoring", "metrics", "list",
		"iam", "service-accounts", "list", "roles", "list",
		"projects", "list", "describe",
	}

	// Add all commands to allowed list
	for _, cmd := range awsCommands {
		cs.allowedCommands[cmd] = true
	}
	for _, cmd := range azureCommands {
		cs.allowedCommands[cmd] = true
	}
	for _, cmd := range gcpCommands {
		cs.allowedCommands[cmd] = true
	}
}

// initializeAllowedFlags sets up allowed CLI flags
func (cs *CommandSanitizer) initializeAllowedFlags() {
	// Common flags across providers
	commonFlags := []string{
		"--output", "--format", "--query", "--region", "--profile",
		"--verbose", "--debug", "--help", "--version",
		"--max-items", "--page-size", "--starting-token",
		"--filters", "--tags", "--name", "--id",
	}

	// AWS-specific flags
	awsFlags := []string{
		"--instance-ids", "--vpc-ids", "--subnet-ids", "--security-group-ids",
		"--bucket", "--prefix", "--delimiter",
		"--db-instance-identifier", "--db-cluster-identifier",
		"--function-name", "--runtime",
		"--cluster-name", "--service-name",
		"--table-name", "--index-name",
		"--user-name", "--role-name", "--policy-arn",
		"--metric-name", "--namespace",
		"--auto-scaling-group-name",
		"--hosted-zone-id",
		"--queue-url", "--topic-arn",
		"--cache-cluster-id",
		"--stack-name",
	}

	// Azure-specific flags
	azureFlags := []string{
		"--resource-group", "--name", "--location",
		"--subscription", "--tenant", "--client-id",
		"--vm-name", "--vnet-name", "--subnet-name",
		"--storage-account", "--container-name",
		"--server-name", "--database-name",
		"--function-app-name", "--web-app-name",
		"--cluster-name", "--cosmosdb-name",
		"--vault-name", "--key-name",
		"--resource-type", "--resource-id",
	}

	// GCP-specific flags
	gcpFlags := []string{
		"--project", "--zone", "--region",
		"--instance", "--network", "--subnet",
		"--bucket", "--prefix",
		"--instance-name", "--database",
		"--function", "--service",
		"--cluster", "--node-pool",
		"--dataset", "--table",
		"--metric", "--filter",
		"--service-account", "--role",
	}

	// Add all flags to allowed list
	for _, flag := range commonFlags {
		cs.allowedFlags[flag] = true
	}
	for _, flag := range awsFlags {
		cs.allowedFlags[flag] = true
	}
	for _, flag := range azureFlags {
		cs.allowedFlags[flag] = true
	}
	for _, flag := range gcpFlags {
		cs.allowedFlags[flag] = true
	}
}

// initializeAllowedValues sets up regex patterns for allowed values
func (cs *CommandSanitizer) initializeAllowedValues() {
	// Region patterns
	cs.allowedValues["region"] = regexp.MustCompile(`^[a-z0-9-]+$`)

	// Resource ID patterns
	cs.allowedValues["resource-id"] = regexp.MustCompile(`^[a-zA-Z0-9-_/]+$`)

	// Name patterns
	cs.allowedValues["name"] = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)

	// Tag patterns
	cs.allowedValues["tag"] = regexp.MustCompile(`^[a-zA-Z0-9-_=]+$`)

	// Filter patterns
	cs.allowedValues["filter"] = regexp.MustCompile(`^[a-zA-Z0-9-_=<>!]+$`)
}

// SanitizeCommand sanitizes a CLI command and its arguments
func (cs *CommandSanitizer) SanitizeCommand(command string, args []string) (string, []string, error) {
	// Validate command
	if !cs.isCommandAllowed(command) {
		return "", nil, fmt.Errorf("command not allowed: %s", command)
	}

	// Sanitize arguments
	sanitizedArgs := make([]string, 0, len(args))
	for i, arg := range args {
		sanitizedArg, err := cs.sanitizeArgument(arg, i)
		if err != nil {
			return "", nil, fmt.Errorf("argument %d not allowed: %s - %v", i, arg, err)
		}
		sanitizedArgs = append(sanitizedArgs, sanitizedArg)
	}

	return command, sanitizedArgs, nil
}

// isCommandAllowed checks if a command is in the allowed list
func (cs *CommandSanitizer) isCommandAllowed(command string) bool {
	// Remove any path prefix
	cmdName := command
	if idx := strings.LastIndex(command, "/"); idx != -1 {
		cmdName = command[idx+1:]
	}
	if idx := strings.LastIndex(command, "\\"); idx != -1 {
		cmdName = command[idx+1:]
	}

	return cs.allowedCommands[cmdName]
}

// sanitizeArgument sanitizes a single command argument
func (cs *CommandSanitizer) sanitizeArgument(arg string, position int) (string, error) {
	// Check if it's a flag
	if strings.HasPrefix(arg, "--") {
		if !cs.isFlagAllowed(arg) {
			return "", fmt.Errorf("flag not allowed: %s", arg)
		}
		return arg, nil
	}

	// Check if it's a short flag
	if strings.HasPrefix(arg, "-") && len(arg) == 2 {
		// Allow common short flags
		allowedShortFlags := map[string]bool{
			"-h": true, "-v": true, "-q": true, "-r": true, "-p": true,
		}
		if !allowedShortFlags[arg] {
			return "", fmt.Errorf("short flag not allowed: %s", arg)
		}
		return arg, nil
	}

	// Sanitize value based on position and context
	return cs.sanitizeValue(arg, position)
}

// isFlagAllowed checks if a flag is in the allowed list
func (cs *CommandSanitizer) isFlagAllowed(flag string) bool {
	return cs.allowedFlags[flag]
}

// sanitizeValue sanitizes a command value
func (cs *CommandSanitizer) sanitizeValue(value string, position int) (string, error) {
	// Basic sanitization - remove any potentially dangerous characters
	sanitized := value

	// Remove command injection attempts
	dangerousPatterns := []string{
		";", "&&", "||", "|", ">", "<", "`", "$(", "$[",
		"$((", "eval", "exec", "system", "subprocess",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(sanitized, pattern) {
			return "", fmt.Errorf("dangerous pattern detected: %s", pattern)
		}
	}

	// Apply regex validation based on expected value type
	if cs.allowedValues["region"].MatchString(sanitized) {
		return sanitized, nil
	}

	if cs.allowedValues["resource-id"].MatchString(sanitized) {
		return sanitized, nil
	}

	if cs.allowedValues["name"].MatchString(sanitized) {
		return sanitized, nil
	}

	if cs.allowedValues["tag"].MatchString(sanitized) {
		return sanitized, nil
	}

	if cs.allowedValues["filter"].MatchString(sanitized) {
		return sanitized, nil
	}

	// If no specific pattern matches, apply basic alphanumeric check
	if regexp.MustCompile(`^[a-zA-Z0-9-_./]+$`).MatchString(sanitized) {
		return sanitized, nil
	}

	return "", fmt.Errorf("value contains invalid characters: %s", value)
}

// ValidateProviderCommand validates a provider-specific command
func (cs *CommandSanitizer) ValidateProviderCommand(provider, command string, args []string) error {
	// Provider-specific validation
	switch provider {
	case "aws":
		return cs.validateAWSCommand(command, args)
	case "azure":
		return cs.validateAzureCommand(command, args)
	case "gcp":
		return cs.validateGCPCommand(command, args)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

// validateAWSCommand validates AWS CLI commands
func (cs *CommandSanitizer) validateAWSCommand(command string, args []string) error {
	// AWS-specific validation logic
	if !strings.HasPrefix(command, "aws") {
		return fmt.Errorf("invalid AWS command: %s", command)
	}

	// Additional AWS-specific checks can be added here
	return nil
}

// validateAzureCommand validates Azure CLI commands
func (cs *CommandSanitizer) validateAzureCommand(command string, args []string) error {
	// Azure-specific validation logic
	if !strings.HasPrefix(command, "az") {
		return fmt.Errorf("invalid Azure command: %s", command)
	}

	// Additional Azure-specific checks can be added here
	return nil
}

// validateGCPCommand validates GCP CLI commands
func (cs *CommandSanitizer) validateGCPCommand(command string, args []string) error {
	// GCP-specific validation logic
	if !strings.HasPrefix(command, "gcloud") && !strings.HasPrefix(command, "gsutil") {
		return fmt.Errorf("invalid GCP command: %s", command)
	}

	// Additional GCP-specific checks can be added here
	return nil
}
