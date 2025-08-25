package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/catherinevee/driftmgr/internal/core/color"
	"github.com/catherinevee/driftmgr/internal/core/progress"
	"github.com/catherinevee/driftmgr/internal/credentials"
)

// AccountSelector handles selection of cloud accounts/subscriptions/projects
type AccountSelector struct {
	detector *credentials.CredentialDetector
}

// NewAccountSelector creates a new account selector
func NewAccountSelector() *AccountSelector {
	return &AccountSelector{
		detector: credentials.NewCredentialDetector(),
	}
}

// SelectAccount allows users to select which account to work with
func (s *AccountSelector) SelectAccount(provider string) error {
	switch strings.ToLower(provider) {
	case "aws":
		return s.selectAWSProfile()
	case "azure":
		return s.selectAzureSubscription()
	case "gcp":
		return s.selectGCPProject()
	case "digitalocean":
		return s.selectDigitalOceanContext()
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

// selectAWSProfile allows selection of AWS profile
func (s *AccountSelector) selectAWSProfile() error {
	// Show spinner while detecting profiles
	spinner := progress.NewSpinner("Detecting AWS profiles and accounts")
	spinner.Start()

	// Get available profiles
	profiles := s.getAWSProfiles()
	if len(profiles) == 0 {
		spinner.Error("No AWS profiles found")
		return nil
	}

	// Get AWS accounts
	awsAccounts := s.detector.DetectAWSAccounts()
	spinner.Success(fmt.Sprintf("Found %d AWS profiles", len(profiles)))

	// If multiple accounts exist, show them grouped
	if len(awsAccounts) > 1 {
		fmt.Printf("\n%s\n", color.Header("Multiple AWS Accounts Detected:"))
		fmt.Println(color.DoubleDivider())

		accountNum := 1
		profileToNumber := make(map[string]int)

		for accountID, accountProfiles := range awsAccounts {
			fmt.Printf("\n%s %s\n", color.Label("Account:"), color.AWS(accountID))
			fmt.Println(color.Divider())
			for _, profile := range accountProfiles {
				accountInfo := s.getAWSAccountInfo(profile)
				fmt.Printf("%s %s %s %s\n",
					color.Count(accountNum),
					color.Label("Profile:"),
					color.Value(profile),
					color.Dim(accountInfo))
				profileToNumber[profile] = accountNum
				accountNum++
			}
		}

		// Get current profile
		currentProfile := os.Getenv("AWS_PROFILE")
		if currentProfile == "" {
			currentProfile = "default"
		}
		fmt.Printf("\nCurrent profile: %s\n", currentProfile)

		// Prompt for selection
		fmt.Print("\nSelect profile number (or press Enter to keep current): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Printf("Keeping current profile: %s\n", currentProfile)
			return nil
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > accountNum-1 {
			return fmt.Errorf("invalid selection")
		}

		// Find the profile for this number
		for profile, num := range profileToNumber {
			if num == choice {
				os.Setenv("AWS_PROFILE", profile)
				fmt.Printf("✓ Switched to AWS profile: %s\n", profile)
				return nil
			}
		}
	} else {
		// Single account, show profiles normally
		if len(profiles) == 1 {
			fmt.Printf("Only one AWS profile available: %s\n", profiles[0])
			return nil
		}

		// Display profiles
		fmt.Println("\nAvailable AWS Profiles:")
		fmt.Println("───────────────────────────")
		for i, profile := range profiles {
			// Get account info for each profile
			accountInfo := s.getAWSAccountInfo(profile)
			fmt.Printf("%d. %s %s\n", i+1, profile, accountInfo)
		}

		// Get current profile
		currentProfile := os.Getenv("AWS_PROFILE")
		if currentProfile == "" {
			currentProfile = "default"
		}
		fmt.Printf("\nCurrent profile: %s\n", currentProfile)

		// Prompt for selection
		fmt.Print("\nSelect profile number (or press Enter to keep current): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			fmt.Printf("Keeping current profile: %s\n", currentProfile)
			return nil
		}

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 1 || choice > len(profiles) {
			return fmt.Errorf("invalid selection")
		}

		selectedProfile := profiles[choice-1]

		// Set the profile
		os.Setenv("AWS_PROFILE", selectedProfile)
		fmt.Printf("✓ Switched to AWS profile: %s\n", selectedProfile)

		// Optionally, update AWS CLI default profile
		fmt.Print("Make this the default profile? (y/n): ")
		response, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(response)) == "y" {
			cmd := exec.Command("aws", "configure", "set", "profile.default.region",
				s.getAWSRegion(selectedProfile))
			cmd.Run()
			fmt.Println("✓ Set as default profile")
		}
	}

	return nil
}

// selectAzureSubscription allows selection of Azure subscription
func (s *AccountSelector) selectAzureSubscription() error {
	// Get available subscriptions
	cmd := exec.Command("az", "account", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list Azure subscriptions: %w", err)
	}

	// Parse subscriptions (simplified parsing)
	subs := s.parseAzureSubscriptions(string(output))
	if len(subs) == 0 {
		fmt.Println("No Azure subscriptions found.")
		return nil
	}

	if len(subs) == 1 {
		fmt.Printf("Only one Azure subscription available: %s\n", subs[0].Name)
		return nil
	}

	// Display subscriptions
	fmt.Println("\nAvailable Azure Subscriptions:")
	fmt.Println("───────────────────────────────")
	for i, sub := range subs {
		fmt.Printf("%d. %s (ID: %s)\n", i+1, sub.Name, sub.ID)
	}

	// Get current subscription
	currentCmd := exec.Command("az", "account", "show", "--query", "name", "-o", "tsv")
	currentOutput, _ := currentCmd.Output()
	currentSub := strings.TrimSpace(string(currentOutput))
	fmt.Printf("\nCurrent subscription: %s\n", currentSub)

	// Prompt for selection
	fmt.Print("\nSelect subscription number (or press Enter to keep current): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		fmt.Printf("Keeping current subscription: %s\n", currentSub)
		return nil
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(subs) {
		return fmt.Errorf("invalid selection")
	}

	selectedSub := subs[choice-1]

	// Set the subscription
	setCmd := exec.Command("az", "account", "set", "--subscription", selectedSub.ID)
	if err := setCmd.Run(); err != nil {
		return fmt.Errorf("failed to set subscription: %w", err)
	}

	fmt.Printf("✓ Switched to Azure subscription: %s\n", selectedSub.Name)
	return nil
}

// selectGCPProject allows selection of GCP project
func (s *AccountSelector) selectGCPProject() error {
	// Get available projects
	var cmd *exec.Cmd
	if os.Getenv("OS") == "Windows_NT" {
		cmd = exec.Command("cmd.exe", "/c", "gcloud", "projects", "list", "--format=value(projectId,name)")
	} else {
		cmd = exec.Command("gcloud", "projects", "list", "--format=value(projectId,name)")
	}

	output, err := cmd.Output()
	if err != nil {
		// Try to list configurations instead
		return s.selectGCPConfiguration()
	}

	projects := s.parseGCPProjects(string(output))
	if len(projects) == 0 {
		fmt.Println("No GCP projects found.")
		return nil
	}

	if len(projects) == 1 {
		fmt.Printf("Only one GCP project available: %s\n", projects[0].ID)
		return nil
	}

	// Display projects
	fmt.Println("\nAvailable GCP Projects:")
	fmt.Println("───────────────────────────")
	for i, project := range projects {
		fmt.Printf("%d. %s (%s)\n", i+1, project.ID, project.Name)
	}

	// Get current project
	var currentCmd *exec.Cmd
	if os.Getenv("OS") == "Windows_NT" {
		currentCmd = exec.Command("cmd.exe", "/c", "gcloud", "config", "get-value", "project")
	} else {
		currentCmd = exec.Command("gcloud", "config", "get-value", "project")
	}
	currentOutput, _ := currentCmd.Output()
	currentProject := strings.TrimSpace(string(currentOutput))
	fmt.Printf("\nCurrent project: %s\n", currentProject)

	// Prompt for selection
	fmt.Print("\nSelect project number (or press Enter to keep current): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		fmt.Printf("Keeping current project: %s\n", currentProject)
		return nil
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(projects) {
		return fmt.Errorf("invalid selection")
	}

	selectedProject := projects[choice-1]

	// Set the project
	var setCmd *exec.Cmd
	if os.Getenv("OS") == "Windows_NT" {
		setCmd = exec.Command("cmd.exe", "/c", "gcloud", "config", "set", "project", selectedProject.ID)
	} else {
		setCmd = exec.Command("gcloud", "config", "set", "project", selectedProject.ID)
	}

	if err := setCmd.Run(); err != nil {
		return fmt.Errorf("failed to set project: %w", err)
	}

	fmt.Printf("✓ Switched to GCP project: %s\n", selectedProject.ID)
	return nil
}

// selectGCPConfiguration allows selection of GCP configuration
func (s *AccountSelector) selectGCPConfiguration() error {
	fmt.Println("\nGCP configurations allow you to switch between different projects and settings.")

	var cmd *exec.Cmd
	if os.Getenv("OS") == "Windows_NT" {
		cmd = exec.Command("cmd.exe", "/c", "gcloud", "config", "configurations", "list")
	} else {
		cmd = exec.Command("gcloud", "config", "configurations", "list")
	}

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list GCP configurations: %w", err)
	}

	fmt.Println(string(output))

	fmt.Print("\nEnter configuration name to activate (or press Enter to skip): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return nil
	}

	var activateCmd *exec.Cmd
	if os.Getenv("OS") == "Windows_NT" {
		activateCmd = exec.Command("cmd.exe", "/c", "gcloud", "config", "configurations", "activate", input)
	} else {
		activateCmd = exec.Command("gcloud", "config", "configurations", "activate", input)
	}

	if err := activateCmd.Run(); err != nil {
		return fmt.Errorf("failed to activate configuration: %w", err)
	}

	fmt.Printf("✓ Activated GCP configuration: %s\n", input)
	return nil
}

// selectDigitalOceanContext allows selection of DigitalOcean context
func (s *AccountSelector) selectDigitalOceanContext() error {
	// Check if doctl is available
	cmd := exec.Command("doctl", "auth", "list")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("doctl CLI not found or not configured.")
		fmt.Println("Please install doctl and run: doctl auth init")
		return nil
	}

	fmt.Println("\nDigitalOcean Authentication Contexts:")
	fmt.Println("─────────────────────────────────────")
	fmt.Println(string(output))

	// Get account info for current context
	accountCmd := exec.Command("doctl", "account", "get")
	if accountOutput, err := accountCmd.Output(); err == nil {
		fmt.Println("\nCurrent Account Details:")
		fmt.Println(string(accountOutput))
	}

	fmt.Println("\nTo switch contexts, use: doctl auth switch --context <name>")
	fmt.Print("Enter context name to switch to (or press Enter to skip): ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return nil
	}

	switchCmd := exec.Command("doctl", "auth", "switch", "--context", input)
	if err := switchCmd.Run(); err != nil {
		return fmt.Errorf("failed to switch context: %w", err)
	}

	fmt.Printf("✓ Switched to DigitalOcean context: %s\n", input)
	return nil
}

// Helper methods

func (s *AccountSelector) getAWSProfiles() []string {
	var profiles []string

	// Check AWS config file
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".aws", "config")

	if content, err := os.ReadFile(configPath); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "[profile ") {
				profile := strings.TrimSuffix(strings.TrimPrefix(line, "[profile "), "]")
				profiles = append(profiles, profile)
			} else if line == "[default]" {
				profiles = append(profiles, "default")
			}
		}
	}

	return profiles
}

func (s *AccountSelector) getAWSAccountInfo(profile string) string {
	// Try to get account info for the profile
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--profile", profile, "--output", "text")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	fields := strings.Fields(string(output))
	if len(fields) >= 1 {
		return fmt.Sprintf("(Account: %s)", fields[0])
	}
	return ""
}

func (s *AccountSelector) getAWSRegion(profile string) string {
	cmd := exec.Command("aws", "configure", "get", "region", "--profile", profile)
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

// Subscription represents an Azure subscription
type Subscription struct {
	ID   string
	Name string
}

func (s *AccountSelector) parseAzureSubscriptions(jsonOutput string) []Subscription {
	var subs []Subscription

	// Simple JSON parsing for subscriptions
	lines := strings.Split(jsonOutput, "\n")
	currentSub := Subscription{}

	for _, line := range lines {
		if strings.Contains(line, "\"id\":") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				currentSub.ID = parts[3]
			}
		} else if strings.Contains(line, "\"name\":") {
			parts := strings.Split(line, "\"")
			if len(parts) >= 4 {
				currentSub.Name = parts[3]
				if currentSub.ID != "" {
					subs = append(subs, currentSub)
					currentSub = Subscription{}
				}
			}
		}
	}

	return subs
}

// Project represents a GCP project
type Project struct {
	ID   string
	Name string
}

func (s *AccountSelector) parseGCPProjects(output string) []Project {
	var projects []Project

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			project := Project{ID: fields[0]}
			if len(fields) >= 2 {
				project.Name = strings.Join(fields[1:], " ")
			}
			projects = append(projects, project)
		}
	}

	return projects
}

// ShowAllAccounts displays all available accounts across all providers
func (s *AccountSelector) ShowAllAccounts() {
	fmt.Println("\n=== All Available Cloud Accounts ===")
	fmt.Println()

	// AWS
	profiles := s.getAWSProfiles()
	if len(profiles) > 0 {
		fmt.Println("AWS Profiles:")
		for _, profile := range profiles {
			accountInfo := s.getAWSAccountInfo(profile)
			fmt.Printf("  • %s %s\n", profile, accountInfo)
		}
		fmt.Println()
	}

	// Azure
	cmd := exec.Command("az", "account", "list", "--output", "tsv", "--query", "[].name")
	if output, err := cmd.Output(); err == nil {
		subs := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(subs) > 0 && subs[0] != "" {
			fmt.Println("Azure Subscriptions:")
			for _, sub := range subs {
				if sub != "" {
					fmt.Printf("  • %s\n", sub)
				}
			}
			fmt.Println()
		}
	}

	// GCP
	var gcpCmd *exec.Cmd
	if os.Getenv("OS") == "Windows_NT" {
		gcpCmd = exec.Command("cmd.exe", "/c", "gcloud", "projects", "list", "--format=value(projectId)")
	} else {
		gcpCmd = exec.Command("gcloud", "projects", "list", "--format=value(projectId)")
	}

	if output, err := gcpCmd.Output(); err == nil {
		projects := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(projects) > 0 && projects[0] != "" {
			fmt.Println("GCP Projects:")
			for _, project := range projects {
				if project != "" {
					fmt.Printf("  • %s\n", project)
				}
			}
			fmt.Println()
		}
	}

	// DigitalOcean
	doCmd := exec.Command("doctl", "auth", "list", "--format", "Context")
	if output, err := doCmd.Output(); err == nil {
		contexts := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(contexts) > 1 { // Skip header
			fmt.Println("DigitalOcean Contexts:")
			for i := 1; i < len(contexts); i++ {
				if contexts[i] != "" {
					fmt.Printf("  • %s\n", contexts[i])
				}
			}
			fmt.Println()
		}
	}
}
