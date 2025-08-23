package credentials

import (
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

// DetectAll detects all cloud credentials
func (d *CredentialDetector) DetectAll() []Credential {
	var creds []Credential

	// Check AWS
	if awsCred := d.detectAWS(); awsCred != nil {
		creds = append(creds, *awsCred)
	}

	// Check Azure
	if azureCred := d.detectAzure(); azureCred != nil {
		creds = append(creds, *azureCred)
	}

	// Check GCP
	if gcpCred := d.detectGCP(); gcpCred != nil {
		creds = append(creds, *gcpCred)
	}

	// Check DigitalOcean
	if doCred := d.detectDigitalOcean(); doCred != nil {
		creds = append(creds, *doCred)
	}

	return creds
}

// detectAWS detects AWS credentials
func (d *CredentialDetector) detectAWS() *Credential {
	// Check environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_PROFILE") != "" {
		return &Credential{
			Provider: "AWS",
			Status:   "configured",
			Details: map[string]interface{}{
				"profile": os.Getenv("AWS_PROFILE"),
				"region":  os.Getenv("AWS_DEFAULT_REGION"),
			},
		}
	}

	// Check AWS CLI
	cmd := exec.Command("aws", "sts", "get-caller-identity")
	if err := cmd.Run(); err == nil {
		return &Credential{
			Provider: "AWS",
			Status:   "configured",
			Details:  map[string]interface{}{},
		}
	}

	return nil
}

// detectAzure detects Azure credentials
func (d *CredentialDetector) detectAzure() *Credential {
	cmd := exec.Command("az", "account", "show")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return &Credential{
			Provider: "Azure",
			Status:   "configured",
			Details:  map[string]interface{}{},
		}
	}
	return nil
}

// detectGCP detects GCP credentials
func (d *CredentialDetector) detectGCP() *Credential {
	// Check environment variables
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		return &Credential{
			Provider: "GCP",
			Status:   "configured",
			Details: map[string]interface{}{
				"credentials_file": os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
			},
		}
	}

	// Check for application default credentials
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		adcPath := filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json")
		if _, err := os.Stat(adcPath); err == nil {
			return &Credential{
				Provider: "GCP",
				Status:   "configured",
				Details: map[string]interface{}{
					"source": "application default credentials",
				},
			}
		}
	}

	// Try gcloud CLI (may not be in PATH on Windows)
	var cmd *exec.Cmd
	if os.Getenv("OS") == "Windows_NT" {
		cmd = exec.Command("cmd.exe", "/c", "gcloud", "config", "get-value", "account")
	} else {
		cmd = exec.Command("gcloud", "config", "get-value", "account")
	}
	output, err := cmd.Output()
	if err == nil && len(strings.TrimSpace(string(output))) > 0 {
		return &Credential{
			Provider: "GCP",
			Status:   "configured",
			Details:  map[string]interface{}{},
		}
	}

	return nil
}

// detectDigitalOcean detects DigitalOcean credentials
func (d *CredentialDetector) detectDigitalOcean() *Credential {
	// Check various common DigitalOcean token environment variables
	tokenVars := []string{
		"DIGITALOCEAN_ACCESS_TOKEN",
		"DIGITALOCEAN_TOKEN",
		"DO_TOKEN",
		"DO_ACCESS_TOKEN",
		"DIGITAL_OCEAN_TOKEN",
	}

	for _, envVar := range tokenVars {
		if token := os.Getenv(envVar); token != "" {
			return &Credential{
				Provider: "DigitalOcean",
				Status:   "configured",
				Details: map[string]interface{}{
					"token_var": envVar,
				},
			}
		}
	}

	// Check for doctl CLI configuration
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
				return &Credential{
					Provider: "DigitalOcean",
					Status:   "configured",
					Details: map[string]interface{}{
						"source": "doctl",
					},
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
