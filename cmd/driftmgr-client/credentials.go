package main

import (
	"fmt"
	"strings"

	"github.com/catherinevee/driftmgr/internal/credentials"
)

// handleCredentials handles the credentials command
func handleCredentials(args []string) error {
	if len(args) == 0 {
		return showCredentialsHelp()
	}

	subCommand := strings.ToLower(args[0])

	switch subCommand {
	case "setup":
		return handleCredentialsSetup()
	case "list":
		return handleCredentialsList()
	case "validate":
		return handleCredentialsValidate(args[1:])
	case "help", "?":
		return showCredentialsHelp()
	default:
		return fmt.Errorf("unknown credentials command: %s. Use 'credentials ?' for help", subCommand)
	}
}

// handleCredentialsSetup handles the credentials setup command
func handleCredentialsSetup() error {
	cm, err := credentials.NewCredentialManager()
	if err != nil {
		return fmt.Errorf("failed to create credential manager: %w", err)
	}

	fmt.Println("Cloud Credentials Setup")
	fmt.Println("=======================")
	fmt.Println("This will help you configure credentials for cloud providers.")
	fmt.Println()

	return cm.SetupCredentials()
}

// handleCredentialsList handles the credentials list command
func handleCredentialsList() error {
	cm, err := credentials.NewCredentialManager()
	if err != nil {
		return fmt.Errorf("failed to create credential manager: %w", err)
	}

	return cm.ListConfiguredProviders()
}

// handleCredentialsValidate handles the credentials validate command
func handleCredentialsValidate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("provider required. Usage: credentials validate <provider>")
	}

	provider := strings.ToLower(args[0])
	cloudProvider := credentials.CloudProvider(provider)

	cm, err := credentials.NewCredentialManager()
	if err != nil {
		return fmt.Errorf("failed to create credential manager: %w", err)
	}

	fmt.Printf("Validating %s credentials...\n", strings.ToUpper(provider))
	fmt.Println()

	return cm.ValidateCredentials(cloudProvider)
}

// showCredentialsHelp shows help for the credentials command
func showCredentialsHelp() error {
	fmt.Println("Credentials Management")
	fmt.Println("======================")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  setup                    - Interactive setup for cloud provider credentials")
	fmt.Println("  list                     - List configured cloud providers")
	fmt.Println("  validate <provider>      - Validate credentials for a specific provider")
	fmt.Println("  help, ?                  - Show this help message")
	fmt.Println()
	fmt.Println("Supported providers:")
	fmt.Println("  aws                      - Amazon Web Services")
	fmt.Println("  azure                    - Microsoft Azure")
	fmt.Println("  gcp                      - Google Cloud Platform")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  credentials setup        - Start interactive credential setup")
	fmt.Println("  credentials list         - Check which providers are configured")
	fmt.Println("  credentials validate aws - Validate AWS credentials")
	fmt.Println("  credentials ?            - Show this help")
	fmt.Println()

	return nil
}

// getCredentialsHelp returns the help text for the credentials command
func getCredentialsHelp() string {
	return `Credentials Management

Commands:
  setup                    - Interactive setup for cloud provider credentials
  list                     - List configured cloud providers
  validate <provider>      - Validate credentials for a specific provider
  help, ?                  - Show this help message

Supported providers:
  aws                      - Amazon Web Services
  azure                    - Microsoft Azure
  gcp                      - Google Cloud Platform

Examples:
  credentials setup        - Start interactive credential setup
  credentials list         - Check which providers are configured
  credentials validate aws - Validate AWS credentials
  credentials ?            - Show this help`
}
