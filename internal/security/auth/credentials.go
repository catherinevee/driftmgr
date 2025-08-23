package security

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"
)

// Credential represents a cloud credential
type Credential struct {
	Provider string                 `json:"provider"`
	Status   string                 `json:"status"`
	Details  map[string]interface{} `json:"details"`
}

// CredentialDetector provides unified credential detection
type CredentialDetector struct {
	// Cache detected credentials
	cache map[string]*Credential
}

// NewCredentialDetector creates a new unified credential detector
func NewCredentialDetector() *CredentialDetector {
	return &CredentialDetector{
		cache: make(map[string]*Credential),
	}
}

// DetectAll detects all cloud credentials
func (d *CredentialDetector) DetectAll() []Credential {
	var creds []Credential

	// Check each provider
	if cred := d.DetectAWS(); cred != nil {
		creds = append(creds, *cred)
	}
	if cred := d.DetectAzure(); cred != nil {
		creds = append(creds, *cred)
	}
	if cred := d.DetectGCP(); cred != nil {
		creds = append(creds, *cred)
	}
	if cred := d.DetectDigitalOcean(); cred != nil {
		creds = append(creds, *cred)
	}

	return creds
}

// DetectAWS detects AWS credentials
func (d *CredentialDetector) DetectAWS() *Credential {
	// Check cache
	if cred, ok := d.cache["aws"]; ok {
		return cred
	}

	// Environment variables
	if os.Getenv("AWS_ACCESS_KEY_ID") != "" || os.Getenv("AWS_PROFILE") != "" {
		cred := &Credential{
			Provider: "AWS",
			Status:   "configured",
			Details: map[string]interface{}{
				"profile": os.Getenv("AWS_PROFILE"),
				"region":  os.Getenv("AWS_DEFAULT_REGION"),
				"source":  "environment",
			},
		}
		d.cache["aws"] = cred
		return cred
	}

	// AWS credentials file
	homeDir, _ := os.UserHomeDir()
	awsCredsFile := filepath.Join(homeDir, ".aws", "credentials")
	if _, err := os.Stat(awsCredsFile); err == nil {
		cred := &Credential{
			Provider: "AWS",
			Status:   "configured",
			Details: map[string]interface{}{
				"source": "credentials file",
			},
		}
		d.cache["aws"] = cred
		return cred
	}

	// AWS CLI
	if output, err := d.execCommand("aws", "sts", "get-caller-identity"); err == nil && len(output) > 0 {
		cred := &Credential{
			Provider: "AWS",
			Status:   "configured",
			Details: map[string]interface{}{
				"source": "aws cli",
			},
		}
		d.cache["aws"] = cred
		return cred
	}

	return nil
}

// DetectAzure detects Azure credentials
func (d *CredentialDetector) DetectAzure() *Credential {
	// Check cache
	if cred, ok := d.cache["azure"]; ok {
		return cred
	}

	// Environment variables
	if os.Getenv("AZURE_SUBSCRIPTION_ID") != "" {
		cred := &Credential{
			Provider: "Azure",
			Status:   "configured",
			Details: map[string]interface{}{
				"subscription": os.Getenv("AZURE_SUBSCRIPTION_ID"),
				"source":       "environment",
			},
		}
		d.cache["azure"] = cred
		return cred
	}

	// Azure CLI
	if output, err := d.execCommand("az", "account", "show"); err == nil && len(output) > 0 {
		cred := &Credential{
			Provider: "Azure",
			Status:   "configured",
			Details: map[string]interface{}{
				"source": "azure cli",
			},
		}
		d.cache["azure"] = cred
		return cred
	}

	return nil
}

// DetectGCP detects GCP credentials
func (d *CredentialDetector) DetectGCP() *Credential {
	// Check cache
	if cred, ok := d.cache["gcp"]; ok {
		return cred
	}

	// Environment variables
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		cred := &Credential{
			Provider: "GCP",
			Status:   "configured",
			Details: map[string]interface{}{
				"credentials_file": os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
				"source":           "environment",
			},
		}
		d.cache["gcp"] = cred
		return cred
	}

	// Application default credentials
	homeDir, _ := os.UserHomeDir()
	adcPath := filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json")
	if _, err := os.Stat(adcPath); err == nil {
		cred := &Credential{
			Provider: "GCP",
			Status:   "configured",
			Details: map[string]interface{}{
				"source": "application default credentials",
			},
		}
		d.cache["gcp"] = cred
		return cred
	}

	// gcloud CLI
	if output, err := d.execCommand("gcloud", "config", "get-value", "account"); err == nil &&
		len(strings.TrimSpace(string(output))) > 0 {
		cred := &Credential{
			Provider: "GCP",
			Status:   "configured",
			Details: map[string]interface{}{
				"source": "gcloud cli",
			},
		}
		d.cache["gcp"] = cred
		return cred
	}

	return nil
}

// DetectDigitalOcean detects DigitalOcean credentials
func (d *CredentialDetector) DetectDigitalOcean() *Credential {
	// Check cache
	if cred, ok := d.cache["digitalocean"]; ok {
		return cred
	}

	// Environment variables - check multiple common variants
	tokenVars := []string{
		"DIGITALOCEAN_ACCESS_TOKEN",
		"DIGITALOCEAN_TOKEN",
		"DO_TOKEN",
		"DO_ACCESS_TOKEN",
		"DIGITAL_OCEAN_TOKEN",
	}

	for _, envVar := range tokenVars {
		if os.Getenv(envVar) != "" {
			cred := &Credential{
				Provider: "DigitalOcean",
				Status:   "configured",
				Details: map[string]interface{}{
					"token_var": envVar,
					"source":    "environment",
				},
			}
			d.cache["digitalocean"] = cred
			return cred
		}
	}

	// doctl config file
	homeDir, _ := os.UserHomeDir()
	doctlConfigPath := filepath.Join(homeDir, "AppData", "Roaming", "doctl", "config.yaml")
	if runtime.GOOS != "windows" {
		doctlConfigPath = filepath.Join(homeDir, ".config", "doctl", "config.yaml")
	}

	if data, err := os.ReadFile(doctlConfigPath); err == nil {
		var config map[string]interface{}
		if err := yaml.Unmarshal(data, &config); err == nil {
			if _, ok := config["access-token"]; ok {
				cred := &Credential{
					Provider: "DigitalOcean",
					Status:   "configured",
					Details: map[string]interface{}{
						"source": "doctl config",
					},
				}
				d.cache["digitalocean"] = cred
				return cred
			}
		}
	}

	return nil
}

// GetCredential gets credential for a specific provider
func (d *CredentialDetector) GetCredential(provider string) *Credential {
	switch strings.ToLower(provider) {
	case "aws":
		return d.DetectAWS()
	case "azure":
		return d.DetectAzure()
	case "gcp", "google":
		return d.DetectGCP()
	case "digitalocean", "do":
		return d.DetectDigitalOcean()
	default:
		return nil
	}
}

// GetToken extracts the actual token/credential for a provider
func (d *CredentialDetector) GetToken(provider string) (string, error) {
	switch strings.ToLower(provider) {
	case "digitalocean", "do":
		return d.getDigitalOceanToken()
	default:
		return "", fmt.Errorf("token extraction not implemented for %s", provider)
	}
}

// getDigitalOceanToken extracts DigitalOcean API token
func (d *CredentialDetector) getDigitalOceanToken() (string, error) {
	// Check environment variables
	tokenVars := []string{
		"DIGITALOCEAN_ACCESS_TOKEN",
		"DIGITALOCEAN_TOKEN",
		"DO_TOKEN",
		"DO_ACCESS_TOKEN",
		"DIGITAL_OCEAN_TOKEN",
	}

	for _, envVar := range tokenVars {
		if token := os.Getenv(envVar); token != "" {
			return token, nil
		}
	}

	// Check doctl config
	homeDir, _ := os.UserHomeDir()
	doctlConfigPath := filepath.Join(homeDir, "AppData", "Roaming", "doctl", "config.yaml")
	if runtime.GOOS != "windows" {
		doctlConfigPath = filepath.Join(homeDir, ".config", "doctl", "config.yaml")
	}

	if data, err := os.ReadFile(doctlConfigPath); err == nil {
		var config map[string]interface{}
		if err := yaml.Unmarshal(data, &config); err == nil {
			if token, ok := config["access-token"].(string); ok {
				return token, nil
			}
		}
	}

	return "", fmt.Errorf("DigitalOcean token not found")
}

// GetProjectID gets the project ID for GCP
func (d *CredentialDetector) GetProjectID() string {
	// Check environment variables
	envVars := []string{"GOOGLE_CLOUD_PROJECT", "GCP_PROJECT", "GCLOUD_PROJECT"}
	for _, env := range envVars {
		if projectID := os.Getenv(env); projectID != "" {
			return projectID
		}
	}

	// Check application default credentials
	homeDir, _ := os.UserHomeDir()
	adcPath := filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json")
	if data, err := os.ReadFile(adcPath); err == nil {
		var creds map[string]interface{}
		if err := json.Unmarshal(data, &creds); err == nil {
			if projectID, ok := creds["quota_project_id"].(string); ok && projectID != "" {
				return projectID
			}
		}
	}

	// Check gcloud config
	configPath := filepath.Join(homeDir, ".config", "gcloud", "configurations", "config_default")
	if data, err := os.ReadFile(configPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "project = ") {
				return strings.TrimSpace(strings.TrimPrefix(line, "project = "))
			}
		}
	}

	return ""
}

// ClearCache clears the credential cache
func (d *CredentialDetector) ClearCache() {
	d.cache = make(map[string]*Credential)
}

// execCommand executes a command with proper OS handling
func (d *CredentialDetector) execCommand(name string, args ...string) ([]byte, error) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// On Windows, try to use cmd.exe for commands that might not be in PATH
		cmdArgs := append([]string{"/c", name}, args...)
		cmd = exec.Command("cmd.exe", cmdArgs...)
	} else {
		cmd = exec.Command(name, args...)
	}

	return cmd.Output()
}

// IsConfigured checks if a provider has credentials configured
func (d *CredentialDetector) IsConfigured(provider string) bool {
	cred := d.GetCredential(provider)
	return cred != nil && cred.Status == "configured"
}

// GetConfiguredProviders returns list of configured providers
func (d *CredentialDetector) GetConfiguredProviders() []string {
	var providers []string

	for _, provider := range []string{"aws", "azure", "gcp", "digitalocean"} {
		if d.IsConfigured(provider) {
			providers = append(providers, provider)
		}
	}

	return providers
}
