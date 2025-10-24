package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

// OAuth2Service handles OAuth2 authentication flows
type OAuth2Service struct {
	providers  map[string]*oauth2.Config
	userRepo   UserRepository
	jwtService *JWTService
}

// OAuth2UserInfo represents user information from OAuth2 providers
type OAuth2UserInfo struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	FirstName string `json:"given_name"`
	LastName  string `json:"family_name"`
	Picture   string `json:"picture"`
	Provider  string `json:"provider"`
}

// OAuth2Config represents OAuth2 provider configuration
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// NewOAuth2Service creates a new OAuth2 service
func NewOAuth2Service(userRepo UserRepository, jwtService *JWTService) *OAuth2Service {
	service := &OAuth2Service{
		providers:  make(map[string]*oauth2.Config),
		userRepo:   userRepo,
		jwtService: jwtService,
	}

	// Initialize providers from environment variables or config
	service.initializeProviders()

	return service
}

// initializeProviders sets up OAuth2 provider configurations
func (s *OAuth2Service) initializeProviders() {
	// Google OAuth2
	if clientID := getEnv("GOOGLE_CLIENT_ID", ""); clientID != "" {
		s.providers["google"] = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/oauth2/google/callback"),
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint:     google.Endpoint,
		}
	}

	// GitHub OAuth2
	if clientID := getEnv("GITHUB_CLIENT_ID", ""); clientID != "" {
		s.providers["github"] = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("GITHUB_REDIRECT_URL", "http://localhost:8080/auth/oauth2/github/callback"),
			Scopes:       []string{"user:email", "read:user"},
			Endpoint:     github.Endpoint,
		}
	}

	// Microsoft OAuth2
	if clientID := getEnv("MICROSOFT_CLIENT_ID", ""); clientID != "" {
		s.providers["microsoft"] = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: getEnv("MICROSOFT_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("MICROSOFT_REDIRECT_URL", "http://localhost:8080/auth/oauth2/microsoft/callback"),
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint:     microsoft.AzureADEndpoint("common"),
		}
	}
}

// GetAuthURL generates an OAuth2 authorization URL for the specified provider
func (s *OAuth2Service) GetAuthURL(provider string, state string) (string, error) {
	config, exists := s.providers[provider]
	if !exists {
		return "", fmt.Errorf("OAuth2 provider %s not configured", provider)
	}

	// Add state parameter for CSRF protection
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	return authURL, nil
}

// ExchangeCode exchanges an authorization code for tokens
func (s *OAuth2Service) ExchangeCode(ctx context.Context, provider, code string) (*oauth2.Token, error) {
	config, exists := s.providers[provider]
	if !exists {
		return nil, fmt.Errorf("OAuth2 provider %s not configured", provider)
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	return token, nil
}

// GetUserInfo retrieves user information from the OAuth2 provider
func (s *OAuth2Service) GetUserInfo(ctx context.Context, provider string, token *oauth2.Token) (*OAuth2UserInfo, error) {
	client := s.getHTTPClient(token)

	switch provider {
	case "google":
		return s.getGoogleUserInfo(ctx, client)
	case "github":
		return s.getGitHubUserInfo(ctx, client)
	case "microsoft":
		return s.getMicrosoftUserInfo(ctx, client)
	default:
		return nil, fmt.Errorf("unsupported OAuth2 provider: %s", provider)
	}
}

// getGoogleUserInfo retrieves user information from Google
func (s *OAuth2Service) getGoogleUserInfo(ctx context.Context, client *http.Client) (*OAuth2UserInfo, error) {
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get Google user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google API returned status %d", resp.StatusCode)
	}

	var userInfo OAuth2UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode Google user info: %w", err)
	}

	userInfo.Provider = "google"
	return &userInfo, nil
}

// getGitHubUserInfo retrieves user information from GitHub
func (s *OAuth2Service) getGitHubUserInfo(ctx context.Context, client *http.Client) (*OAuth2UserInfo, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var userInfo OAuth2UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub user info: %w", err)
	}

	userInfo.Provider = "github"

	// GitHub doesn't provide first/last name separately, so split the name
	if userInfo.Name != "" {
		parts := strings.SplitN(userInfo.Name, " ", 2)
		userInfo.FirstName = parts[0]
		if len(parts) > 1 {
			userInfo.LastName = parts[1]
		}
	}

	return &userInfo, nil
}

// getMicrosoftUserInfo retrieves user information from Microsoft
func (s *OAuth2Service) getMicrosoftUserInfo(ctx context.Context, client *http.Client) (*OAuth2UserInfo, error) {
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return nil, fmt.Errorf("failed to get Microsoft user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Microsoft API returned status %d", resp.StatusCode)
	}

	var userInfo OAuth2UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode Microsoft user info: %w", err)
	}

	userInfo.Provider = "microsoft"
	return &userInfo, nil
}

// CreateOrUpdateUser creates a new user or updates an existing one from OAuth2 info
func (s *OAuth2Service) CreateOrUpdateUser(userInfo *OAuth2UserInfo) (*User, error) {
	// Try to find existing user by email
	user, err := s.userRepo.GetByEmail(userInfo.Email)
	if err != nil {
		// User doesn't exist, create new one
		user = &User{
			ID:        uuid.New().String(),
			Username:  s.generateUsername(userInfo.Email),
			Email:     userInfo.Email,
			FirstName: userInfo.FirstName,
			LastName:  userInfo.LastName,
			IsActive:  true,
			IsAdmin:   false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// For OAuth2 users, we don't set a password hash
		// They can only authenticate via OAuth2

		if err := s.userRepo.Create(user); err != nil {
			return nil, fmt.Errorf("failed to create OAuth2 user: %w", err)
		}
	} else {
		// Update existing user with OAuth2 info
		user.FirstName = userInfo.FirstName
		user.LastName = userInfo.LastName
		user.UpdatedAt = time.Now()

		if err := s.userRepo.Update(user); err != nil {
			return nil, fmt.Errorf("failed to update OAuth2 user: %w", err)
		}
	}

	return user, nil
}

// GenerateTokens generates JWT tokens for an OAuth2 user
func (s *OAuth2Service) GenerateTokens(user *User) (*AuthResponse, error) {
	// For now, assign default roles (this should be improved)
	// In a real implementation, we would get roles from the role repository
	roleNames := []string{RoleViewer}
	if user.IsAdmin {
		roleNames = []string{RoleAdmin}
	}

	// Generate tokens
	accessToken, err := s.jwtService.GenerateAccessToken(user, roleNames)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.jwtService.GetTokenExpiry().Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// GetAvailableProviders returns a list of configured OAuth2 providers
func (s *OAuth2Service) GetAvailableProviders() []map[string]interface{} {
	providers := make([]map[string]interface{}, 0, len(s.providers))

	for name := range s.providers {
		providers = append(providers, map[string]interface{}{
			"name":      strings.Title(name),
			"id":        name,
			"auth_url":  fmt.Sprintf("/api/v1/auth/oauth2/%s", name),
			"is_active": true,
		})
	}

	return providers
}

// Helper methods

// getHTTPClient creates an HTTP client with OAuth2 token
func (s *OAuth2Service) getHTTPClient(token *oauth2.Token) *http.Client {
	// This would use the OAuth2 config to create a client
	// For now, return a basic client
	return &http.Client{}
}

// generateUsername creates a username from email
func (s *OAuth2Service) generateUsername(email string) string {
	// Extract username part from email
	parts := strings.Split(email, "@")
	username := parts[0]

	// Clean up username (remove special characters, etc.)
	username = strings.ReplaceAll(username, ".", "")
	username = strings.ReplaceAll(username, "+", "")
	username = strings.ToLower(username)

	// Ensure uniqueness by appending random suffix if needed
	// This is a simplified approach - in production, you'd check for uniqueness
	return username + "-" + uuid.New().String()[:8]
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	// In a real implementation, this would use os.Getenv
	// For now, return default values for testing
	switch key {
	case "GOOGLE_CLIENT_ID":
		return "your-google-client-id"
	case "GOOGLE_CLIENT_SECRET":
		return "your-google-client-secret"
	case "GOOGLE_REDIRECT_URL":
		return "http://localhost:8080/auth/oauth2/google/callback"
	case "GITHUB_CLIENT_ID":
		return "your-github-client-id"
	case "GITHUB_CLIENT_SECRET":
		return "your-github-client-secret"
	case "GITHUB_REDIRECT_URL":
		return "http://localhost:8080/auth/oauth2/github/callback"
	case "MICROSOFT_CLIENT_ID":
		return "your-microsoft-client-id"
	case "MICROSOFT_CLIENT_SECRET":
		return "your-microsoft-client-secret"
	case "MICROSOFT_REDIRECT_URL":
		return "http://localhost:8080/auth/oauth2/microsoft/callback"
	default:
		return defaultValue
	}
}
