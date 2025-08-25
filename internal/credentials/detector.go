package credentials

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Credential represents a cloud credential
type Credential struct {
	Provider string                 `json:"provider"`
	Status   string                 `json:"status"`
	Details  map[string]interface{} `json:"details"`
}

// CredentialDetector detects cloud credentials
type CredentialDetector struct{}

// NewCredentialDetector creates a new credential detector
func NewCredentialDetector() *CredentialDetector {
	return &CredentialDetector{}
}

// IsConfigured checks if a provider is configured
func (d *CredentialDetector) IsConfigured(provider string) bool {
	switch provider {
	case "aws":
		cred := d.detectAWS()
		return cred != nil && cred.Status == "configured"
	case "azure":
		cred := d.detectAzure()
		return cred != nil && cred.Status == "configured"
	case "gcp":
		cred := d.detectGCP()
		return cred != nil && cred.Status == "configured"
	case "digitalocean":
		cred := d.detectDigitalOcean()
		return cred != nil && cred.Status == "configured"
	default:
		return false
	}
}

// DetectAll detects all cloud credentials in parallel for speed
func (d *CredentialDetector) DetectAll() []Credential {
	type credResult struct {
		cred *Credential
		idx  int
	}

	// Run all detections in parallel
	resultChan := make(chan credResult, 4)

	go func() {
		resultChan <- credResult{cred: d.detectAWS(), idx: 0}
	}()
	go func() {
		resultChan <- credResult{cred: d.detectAzure(), idx: 1}
	}()
	go func() {
		resultChan <- credResult{cred: d.detectGCP(), idx: 2}
	}()
	go func() {
		resultChan <- credResult{cred: d.detectDigitalOcean(), idx: 3}
	}()

	// Collect results
	results := make([]*Credential, 4)
	for i := 0; i < 4; i++ {
		result := <-resultChan
		results[result.idx] = result.cred
	}
	close(resultChan)

	// Build final credential list
	var creds []Credential
	for _, cred := range results {
		if cred != nil {
			creds = append(creds, *cred)
		}
	}

	return creds
}

// DetectMultipleProfiles checks if multiple profiles/accounts are configured for AWS
func (d *CredentialDetector) DetectMultipleProfiles() map[string][]string {
	profiles := make(map[string][]string)

	// Check AWS profiles and get account info for each
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		awsConfigPath := filepath.Join(homeDir, ".aws", "config")
		if content, err := os.ReadFile(awsConfigPath); err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "[profile ") || line == "[default]" {
					profileName := strings.TrimSpace(strings.Trim(line, "[]"))
					profileName = strings.TrimPrefix(profileName, "profile ")
					profiles["AWS"] = append(profiles["AWS"], profileName)
				}
			}
		}
	}

	// Check Azure subscriptions
	cmd := exec.Command("az", "account", "list", "--output", "tsv", "--query", "[].name")
	if output, err := cmd.Output(); err == nil {
		subs := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, sub := range subs {
			if sub != "" {
				profiles["Azure"] = append(profiles["Azure"], sub)
			}
		}
	}

	// Check GCP projects (more useful than configurations)
	if os.Getenv("OS") == "Windows_NT" {
		cmd = exec.Command("cmd.exe", "/c", "gcloud", "projects", "list", "--format=value(projectId)")
	} else {
		cmd = exec.Command("gcloud", "projects", "list", "--format=value(projectId)")
	}
	if output, err := cmd.Output(); err == nil {
		projects := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, project := range projects {
			if project != "" && project != "(unset)" {
				profiles["GCP"] = append(profiles["GCP"], project)
			}
		}
	}

	return profiles
}

// DetectAWSAccounts detects all AWS accounts and their associated profiles
func (d *CredentialDetector) DetectAWSAccounts() map[string][]string {
	accountProfiles := make(map[string][]string)
	profileAccounts := make(map[string]string)

	// Get all profiles first
	var allProfiles []string
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		awsConfigPath := filepath.Join(homeDir, ".aws", "config")
		if content, err := os.ReadFile(awsConfigPath); err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "[profile ") || line == "[default]" {
					profileName := strings.TrimSpace(strings.Trim(line, "[]"))
					profileName = strings.TrimPrefix(profileName, "profile ")
					allProfiles = append(allProfiles, profileName)
				}
			}
		}
	}

	// For each profile, try to get the account ID
	for _, profile := range allProfiles {
		cmd := exec.Command("aws", "sts", "get-caller-identity", "--profile", profile, "--output", "text", "--query", "Account")
		output, err := cmd.Output()
		if err == nil {
			accountID := strings.TrimSpace(string(output))
			if accountID != "" && accountID != "None" {
				// Store the profile under this account
				accountProfiles[accountID] = append(accountProfiles[accountID], profile)
				profileAccounts[profile] = accountID
			}
		}
	}

	return accountProfiles
}

// detectAWS detects AWS credentials
func (d *CredentialDetector) detectAWS() *Credential {
	details := make(map[string]interface{})

	// Check environment variables
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		details["profile"] = profile
	}
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		details["region"] = region
	}

	// Try to get account ID and alias
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--output", "json")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		// Parse the JSON output to get account ID
		if strings.Contains(string(output), "Account") {
			// Extract account ID from JSON (simple parsing)
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Account") {
					parts := strings.Split(line, "\"")
					if len(parts) >= 4 {
						accountID := parts[3]
						details["account_id"] = accountID
						details["account"] = "AWS Account " + accountID
					}
				}
			}
		}

		// Try to get account alias
		aliasCmd := exec.Command("aws", "iam", "list-account-aliases", "--output", "text")
		aliasOutput, _ := aliasCmd.Output()
		if len(aliasOutput) > 0 {
			aliases := strings.TrimSpace(string(aliasOutput))
			if aliases != "" && !strings.Contains(aliases, "ACCOUNTALIASES") {
				details["account_alias"] = aliases
				if accountID, ok := details["account_id"]; ok {
					details["account"] = fmt.Sprintf("%s (%s)", aliases, accountID)
				}
			}
		}

		return &Credential{
			Provider: "AWS",
			Status:   "configured",
			Details:  details,
		}
	}

	// Check if credentials exist even if STS call failed
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_PROFILE") != "" {
		return &Credential{
			Provider: "AWS",
			Status:   "configured",
			Details:  details,
		}
	}

	return nil
}

// detectAzure detects Azure credentials
func (d *CredentialDetector) detectAzure() *Credential {
	cmd := exec.Command("az", "account", "show", "--output", "json")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		details := make(map[string]interface{})

		// Parse subscription information
		if strings.Contains(string(output), "name") {
			// Extract subscription name and ID
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "\"name\":") {
					parts := strings.Split(line, "\"")
					if len(parts) >= 4 {
						subName := parts[3]
						details["subscription_name"] = subName
						details["account"] = "Azure: " + subName
					}
				} else if strings.Contains(line, "\"id\":") && !strings.Contains(line, "tenantId") {
					parts := strings.Split(line, "\"")
					if len(parts) >= 4 {
						details["subscription_id"] = parts[3]
					}
				} else if strings.Contains(line, "\"tenantId\":") {
					parts := strings.Split(line, "\"")
					if len(parts) >= 4 {
						details["tenant_id"] = parts[3]
					}
				}
			}
		}

		return &Credential{
			Provider: "Azure",
			Status:   "configured",
			Details:  details,
		}
	}
	return nil
}

// detectGCP detects GCP credentials
func (d *CredentialDetector) detectGCP() *Credential {
	details := make(map[string]interface{})

	// Check environment variables
	if credFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credFile != "" {
		details["credentials_file"] = credFile
	}
	if projectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); projectID != "" {
		details["project_id"] = projectID
		details["account"] = "GCP Project: " + projectID
	}

	// Check for application default credentials
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		adcPath := filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json")
		if _, err := os.Stat(adcPath); err == nil {
			details["source"] = "application default credentials"
		}
	}

	// Try gcloud CLI to get project and account
	var projectCmd, accountCmd *exec.Cmd
	if os.Getenv("OS") == "Windows_NT" {
		projectCmd = exec.Command("cmd.exe", "/c", "gcloud", "config", "get-value", "project")
		accountCmd = exec.Command("cmd.exe", "/c", "gcloud", "config", "get-value", "account")
	} else {
		projectCmd = exec.Command("gcloud", "config", "get-value", "project")
		accountCmd = exec.Command("gcloud", "config", "get-value", "account")
	}

	projectOutput, err := projectCmd.Output()
	if err == nil && len(strings.TrimSpace(string(projectOutput))) > 0 {
		projectID := strings.TrimSpace(string(projectOutput))
		if projectID != "" && projectID != "(unset)" {
			details["project_id"] = projectID
			details["account"] = "GCP Project: " + projectID
		}
	}

	accountOutput, err := accountCmd.Output()
	if err == nil && len(strings.TrimSpace(string(accountOutput))) > 0 {
		account := strings.TrimSpace(string(accountOutput))
		if account != "" && account != "(unset)" {
			details["email"] = account
		}

		return &Credential{
			Provider: "GCP",
			Status:   "configured",
			Details:  details,
		}
	}

	// Check if credentials exist even if gcloud command failed
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" || details["source"] != nil {
		return &Credential{
			Provider: "GCP",
			Status:   "configured",
			Details:  details,
		}
	}

	return nil
}

// detectDigitalOcean detects DigitalOcean credentials
func (d *CredentialDetector) detectDigitalOcean() *Credential {
	details := make(map[string]interface{})

	// Check various common DigitalOcean token environment variables
	tokenVars := []string{
		"DIGITALOCEAN_ACCESS_TOKEN",
		"DIGITALOCEAN_TOKEN",
		"DO_TOKEN",
		"DO_ACCESS_TOKEN",
		"DIGITAL_OCEAN_TOKEN",
	}

	tokenFound := false
	for _, envVar := range tokenVars {
		if token := os.Getenv(envVar); token != "" {
			details["token_var"] = envVar
			tokenFound = true

			// Try to get account info using doctl if available
			// This requires the token to be configured in doctl
			cmd := exec.Command("doctl", "account", "get", "--format", "Email,UUID,Status")
			if output, err := cmd.Output(); err == nil && len(output) > 0 {
				lines := strings.Split(strings.TrimSpace(string(output)), "\n")
				if len(lines) > 1 {
					// Parse the output (skip header)
					fields := strings.Fields(lines[1])
					if len(fields) >= 2 {
						details["email"] = fields[0]
						details["account_id"] = fields[1]
						details["account"] = fmt.Sprintf("DO Account: %s", fields[0])
					}
				}
			}

			if tokenFound {
				return &Credential{
					Provider: "DigitalOcean",
					Status:   "configured",
					Details:  details,
				}
			}
		}
	}

	// Check for doctl CLI configuration if no token found
	if !tokenFound {
		// Try multiple ways to find doctl
		doctlPaths := []string{
			"doctl",
			"C:\\Program Files\\doctl\\doctl.exe",
			"C:\\Users\\" + os.Getenv("USERNAME") + "\\AppData\\Local\\doctl\\doctl.exe",
		}

		for _, doctlPath := range doctlPaths {
			cmd := exec.Command(doctlPath, "auth", "list")
			if output, err := cmd.Output(); err == nil && len(output) > 0 {
				// Check if output contains "default" or any authentication
				outputStr := string(output)
				if strings.Contains(outputStr, "default") || strings.Contains(outputStr, "(current)") {
					details["source"] = "doctl"

					// Try to get account info
					accountCmd := exec.Command(doctlPath, "account", "get", "--format", "Email,UUID,Status")
					if accountOutput, err := accountCmd.Output(); err == nil && len(accountOutput) > 0 {
						lines := strings.Split(strings.TrimSpace(string(accountOutput)), "\n")
						if len(lines) > 1 {
							fields := strings.Fields(lines[1])
							if len(fields) >= 2 {
								details["email"] = fields[0]
								details["account_id"] = fields[1]
								details["account"] = fmt.Sprintf("DO Account: %s", fields[0])
							}
						}
					}

					return &Credential{
						Provider: "DigitalOcean",
						Status:   "configured",
						Details:  details,
					}
				}
			}
		}
	}

	// Check for doctl config file
	homeDir, _ := os.UserHomeDir()
	doctlConfigPath := filepath.Join(homeDir, ".config", "doctl", "config.yaml")
	if _, err := os.Stat(doctlConfigPath); err == nil {
		return &Credential{
			Provider: "DigitalOcean",
			Status:   "configured",
			Details: map[string]interface{}{
				"source": "doctl config file",
			},
		}
	}

	return nil
}
