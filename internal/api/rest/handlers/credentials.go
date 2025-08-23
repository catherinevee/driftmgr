package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	// Removed cyclic import
)

// ProviderConfig represents provider configuration
type ProviderConfig struct {
	Provider    string                 `json:"provider"`
	Credentials map[string]interface{} `json:"credentials"`
	Regions     []string               `json:"regions,omitempty"`
	Services    []string               `json:"services,omitempty"`
}

// CredentialStatus represents the status of credentials for a provider
type CredentialStatus struct {
	Provider    string    `json:"provider"`
	Configured  bool      `json:"configured"`
	Valid       bool      `json:"valid"`
	LastChecked time.Time `json:"lastChecked"`
	Accounts    []string  `json:"accounts,omitempty"`
	Regions     []string  `json:"regions,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// CredentialTestRequest represents a credential test request
type CredentialTestRequest struct {
	Provider    string                 `json:"provider"`
	Credentials map[string]interface{} `json:"credentials"`
}

// CredentialConfigRequest represents a credential configuration request
type CredentialConfigRequest struct {
	Provider    string                 `json:"provider"`
	Credentials map[string]interface{} `json:"credentials"`
	Persist     bool                   `json:"persist"`
}

// ProviderInfo represents information about a cloud provider
type ProviderInfo struct {
	Name         string                 `json:"name"`
	DisplayName  string                 `json:"displayName"`
	Supported    bool                   `json:"supported"`
	Configured   bool                   `json:"configured"`
	RequiredKeys []string               `json:"requiredKeys"`
	OptionalKeys []string               `json:"optionalKeys"`
	Icon         string                 `json:"icon"`
	Regions      []string               `json:"regions,omitempty"`
	Services     []string               `json:"services,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

// getCredentialStatus retrieves the status of all configured credentials
func (s *EnhancedDashboardServer) getCredentialStatus(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	providers := []string{"aws", "azure", "gcp", "digitalocean"}
	statuses := make([]CredentialStatus, 0)

	for _, provider := range providers {
		status := CredentialStatus{
			Provider:    provider,
			LastChecked: time.Now(),
		}

		// Check if credentials are configured
		configured, err := s.credManager.IsConfigured(ctx, provider)
		status.Configured = configured

		if configured {
			// Validate credentials
			valid, validationErr := s.credManager.ValidateCredentials(ctx, provider)
			status.Valid = valid

			if validationErr != nil {
				status.Error = validationErr.Error()
			} else if valid {
				// Get account information
				accounts, _ := s.credManager.GetAccounts(ctx, provider)
				status.Accounts = accounts

				// Get configured regions
				regions, _ := s.credManager.GetRegions(ctx, provider)
				status.Regions = regions
			}
		} else if err != nil {
			status.Error = err.Error()
		}

		statuses = append(statuses, status)
	}

	// Store credential status
	statusInterfaces := make([]interface{}, len(statuses))
	for i, status := range statuses {
		statusInterfaces[i] = status
	}
	s.dataStore.SetCredentialStatus(statusInterfaces)

	// Broadcast credential status update
	s.broadcast <- map[string]interface{}{
		"type":     "credential_status_updated",
		"statuses": statuses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// testCredentials tests provided credentials without persisting them
func (s *EnhancedDashboardServer) testCredentials(w http.ResponseWriter, r *http.Request) {
	var req CredentialTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Validate required fields
	if req.Provider == "" {
		http.Error(w, "Provider is required", http.StatusBadRequest)
		return
	}

	// Test credentials
	result := map[string]interface{}{
		"provider": req.Provider,
		"valid":    false,
	}

	// Create temporary credential configuration
	tempConfig := &ProviderConfig{
		Provider:    req.Provider,
		Credentials: req.Credentials,
	}

	// Test the credentials
	err := s.credManager.TestProviderConfig(ctx, tempConfig)
	if err != nil {
		result["error"] = err.Error()
		result["message"] = "Credentials validation failed"
	} else {
		result["valid"] = true
		result["message"] = "Credentials are valid"

		// Get additional information if valid
		if accountInfo, err := s.credManager.GetAccountInfoWithConfig(ctx, tempConfig); err == nil {
			result["accountInfo"] = accountInfo
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// configureCredentials configures credentials for a provider
func (s *EnhancedDashboardServer) configureCredentials(w http.ResponseWriter, r *http.Request) {
	var req CredentialConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Validate required fields
	if req.Provider == "" {
		http.Error(w, "Provider is required", http.StatusBadRequest)
		return
	}

	// Create provider configuration
	config := &ProviderConfig{
		Provider:    req.Provider,
		Credentials: req.Credentials,
	}

	// Configure the provider
	err := s.credManager.ConfigureProvider(ctx, config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Persist if requested
	if req.Persist {
		if err := s.credManager.SaveProviderConfig(ctx, config); err != nil {
			// Log error but don't fail the request
			s.broadcast <- map[string]interface{}{
				"type":    "credential_persist_warning",
				"message": "Credentials configured but not persisted",
				"error":   err.Error(),
			}
		}
	}

	// Notify provider configuration success
	go func() {
		s.broadcast <- map[string]interface{}{
			"type":     "provider_configured",
			"provider": req.Provider,
		}
	}()

	response := map[string]interface{}{
		"status":   "configured",
		"provider": req.Provider,
		"message":  "Credentials configured successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// listProviders lists all available cloud providers
func (s *EnhancedDashboardServer) listProviders(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	providers := []ProviderInfo{
		{
			Name:         "aws",
			DisplayName:  "Amazon Web Services",
			Supported:    true,
			Icon:         "aws",
			RequiredKeys: []string{"access_key_id", "secret_access_key"},
			OptionalKeys: []string{"session_token", "region", "profile"},
			Services:     []string{"EC2", "S3", "RDS", "Lambda", "DynamoDB", "VPC"},
		},
		{
			Name:         "azure",
			DisplayName:  "Microsoft Azure",
			Supported:    true,
			Icon:         "azure",
			RequiredKeys: []string{"subscription_id", "tenant_id", "client_id", "client_secret"},
			OptionalKeys: []string{"resource_group", "location"},
			Services:     []string{"Virtual Machines", "Storage", "SQL Database", "Functions", "VNet"},
		},
		{
			Name:         "gcp",
			DisplayName:  "Google Cloud Platform",
			Supported:    true,
			Icon:         "gcp",
			RequiredKeys: []string{"project_id", "credentials_json"},
			OptionalKeys: []string{"region", "zone"},
			Services:     []string{"Compute Engine", "Cloud Storage", "Cloud SQL", "Cloud Functions", "VPC"},
		},
		{
			Name:         "digitalocean",
			DisplayName:  "DigitalOcean",
			Supported:    true,
			Icon:         "digitalocean",
			RequiredKeys: []string{"token"},
			OptionalKeys: []string{"region"},
			Services:     []string{"Droplets", "Spaces", "Databases", "Kubernetes", "VPC"},
		},
	}

	// Check configuration status for each provider
	for i := range providers {
		configured, _ := s.credManager.IsConfigured(ctx, providers[i].Name)
		providers[i].Configured = configured

		if configured {
			// Get regions if configured
			if regions, err := s.credManager.GetRegions(ctx, providers[i].Name); err == nil {
				providers[i].Regions = regions
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

// getProviderConfig retrieves the configuration for a specific provider
func (s *EnhancedDashboardServer) getProviderConfig(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		http.Error(w, "Provider parameter is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	config, err := s.credManager.GetProviderConfig(ctx, provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Mask sensitive information
	maskedConfig := s.maskSensitiveData(config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(maskedConfig)
}

// deleteProviderConfig removes the configuration for a provider
func (s *EnhancedDashboardServer) deleteProviderConfig(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		http.Error(w, "Provider parameter is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	err := s.credManager.DeleteProviderConfig(ctx, provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast deletion
	s.broadcast <- map[string]interface{}{
		"type":     "provider_config_deleted",
		"provider": provider,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "deleted",
		"provider": provider,
	})
}

// rotateCredentials rotates credentials for a provider
func (s *EnhancedDashboardServer) rotateCredentials(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string                 `json:"provider"`
		NewCreds map[string]interface{} `json:"newCredentials"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Test new credentials first
	tempConfig := &ProviderConfig{
		Provider:    req.Provider,
		Credentials: req.NewCreds,
	}

	if err := s.credManager.TestProviderConfig(ctx, tempConfig); err != nil {
		http.Error(w, "New credentials validation failed", http.StatusBadRequest)
		return
	}

	// Backup old credentials
	oldConfig, _ := s.credManager.GetProviderConfig(ctx, req.Provider)

	// Update to new credentials
	if err := s.credManager.ConfigureProvider(ctx, tempConfig); err != nil {
		// Restore old credentials
		if oldConfig != nil {
			s.credManager.ConfigureProvider(ctx, oldConfig)
		}
		http.Error(w, "Failed to rotate credentials", http.StatusInternalServerError)
		return
	}

	// Save new credentials
	if err := s.credManager.SaveProviderConfig(ctx, tempConfig); err != nil {
		// Log warning but continue
		s.broadcast <- map[string]interface{}{
			"type":    "credential_rotation_warning",
			"message": "Credentials rotated but not persisted",
		}
	}

	// Broadcast rotation success
	s.broadcast <- map[string]interface{}{
		"type":     "credentials_rotated",
		"provider": req.Provider,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "rotated",
		"provider": req.Provider,
		"message":  "Credentials rotated successfully",
	})
}

// Helper functions

func (s *EnhancedDashboardServer) maskSensitiveData(config *ProviderConfig) map[string]interface{} {
	masked := map[string]interface{}{
		"provider": config.Provider,
	}

	// Mask sensitive credential fields
	maskedCreds := make(map[string]interface{})
	for key, value := range config.Credentials {
		if isSensitiveField(key) {
			// Show only last 4 characters
			if str, ok := value.(string); ok && len(str) > 4 {
				maskedCreds[key] = "****" + str[len(str)-4:]
			} else {
				maskedCreds[key] = "****"
			}
		} else {
			maskedCreds[key] = value
		}
	}

	masked["credentials"] = maskedCreds
	return masked
}

func isSensitiveField(field string) bool {
	sensitiveFields := []string{
		"secret", "password", "token", "key", "credentials",
		"client_secret", "secret_access_key", "api_key",
	}

	fieldLower := strings.ToLower(field)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(fieldLower, sensitive) {
			return true
		}
	}
	return false
}
