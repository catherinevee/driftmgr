package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

// AuthHandlers handles authentication API endpoints
type AuthHandlers struct {
	authService   *Service
	oauth2Service *OAuth2Service
	validator     *validator.Validate
}

// NewAuthHandlers creates a new set of authentication handlers
func NewAuthHandlers(authService *Service, oauth2Service *OAuth2Service) *AuthHandlers {
	return &AuthHandlers{
		authService:   authService,
		oauth2Service: oauth2Service,
		validator:     validator.New(),
	}
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		writeValidationError(w, err.Error())
		return
	}

	// Get client information
	userAgent := r.Header.Get("User-Agent")
	ipAddress := getClientIP(r)

	// Authenticate user
	response, err := h.authService.Login(&req, userAgent, ipAddress)
	if err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, "AUTHENTICATION_FAILED", "Login failed", err.Error())
		return
	}

	// Set secure cookie for refresh token (optional)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    response.RefreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(response.ExpiresIn),
	})

	writeJSONResponse(w, http.StatusOK, response, nil)
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		writeValidationError(w, err.Error())
		return
	}

	// Register user
	response, err := h.authService.Register(&req)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "REGISTRATION_FAILED", "Registration failed", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusCreated, response, nil)
}

// RefreshToken handles POST /api/v1/auth/refresh
func (h *AuthHandlers) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Try to get refresh token from cookie first, then from request body
	var refreshToken string

	// Check cookie
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		refreshToken = cookie.Value
	} else {
		// Check request body
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload", err.Error())
			return
		}
		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_REFRESH_TOKEN", "Refresh token is required", "")
		return
	}

	// Refresh tokens
	response, err := h.authService.RefreshToken(refreshToken)
	if err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Invalid refresh token", err.Error())
		return
	}

	// Update refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    response.RefreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(response.ExpiresIn),
	})

	writeJSONResponse(w, http.StatusOK, response, nil)
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_TOKEN", "Authorization token is required", "")
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_TOKEN_FORMAT", "Invalid token format", "")
		return
	}

	// Logout user
	if err := h.authService.Logout(token); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "LOGOUT_FAILED", "Logout failed", err.Error())
		return
	}

	// Clear refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Logged out successfully"}, nil)
}

// GetProfile handles GET /api/v1/auth/profile
func (h *AuthHandlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "")
		return
	}

	// Get user profile
	profile, err := h.authService.GetUserProfile(userID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, profile, nil)
}

// UpdateProfile handles PUT /api/v1/auth/profile
func (h *AuthHandlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "")
		return
	}

	// Parse request body
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload", err.Error())
		return
	}

	// Update user profile
	profile, err := h.authService.UpdateUserProfile(userID, updates)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "UPDATE_FAILED", "Profile update failed", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, profile, nil)
}

// ChangePassword handles POST /api/v1/auth/change-password
func (h *AuthHandlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "")
		return
	}

	// Parse request
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		writeValidationError(w, err.Error())
		return
	}

	// Change password
	if err := h.authService.ChangePassword(userID, &req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "PASSWORD_CHANGE_FAILED", "Password change failed", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Password changed successfully"}, nil)
}

// CreateAPIKey handles POST /api/v1/auth/api-keys
func (h *AuthHandlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "")
		return
	}

	// Parse request
	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload", err.Error())
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		writeValidationError(w, err.Error())
		return
	}

	// Create API key
	apiKey, err := h.authService.CreateAPIKey(userID, &req)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "API_KEY_CREATION_FAILED", "API key creation failed", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusCreated, apiKey, nil)
}

// ListAPIKeys handles GET /api/v1/auth/api-keys
func (h *AuthHandlers) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "")
		return
	}

	// Get API keys for the user
	apiKeys, err := h.authService.GetUserAPIKeys(userID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "FAILED_TO_GET_API_KEYS", "Failed to get API keys", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, apiKeys, nil)
}

// DeleteAPIKey handles DELETE /api/v1/auth/api-keys/{id}
func (h *AuthHandlers) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", "")
		return
	}

	// Extract API key ID from URL
	apiKeyID := r.URL.Query().Get("id")
	if apiKeyID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_API_KEY_ID", "API key ID is required", "")
		return
	}

	// Delete the API key
	if err := h.authService.DeleteAPIKey(userID, apiKeyID); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "FAILED_TO_DELETE_API_KEY", "Failed to delete API key", err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "API key deleted successfully"}, nil)
}

// OAuth2Callback handles OAuth2 provider callbacks
func (h *AuthHandlers) OAuth2Callback(w http.ResponseWriter, r *http.Request) {
	// Extract provider from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_PATH", "Invalid OAuth2 callback path", "")
		return
	}
	provider := pathParts[len(pathParts)-2] // Get provider from path like /auth/oauth2/google/callback

	// Get authorization code from query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_CODE", "Authorization code is required", "")
		return
	}

	// Validate state parameter (CSRF protection)
	if state == "" {
		writeErrorResponse(w, http.StatusBadRequest, "MISSING_STATE", "State parameter is required", "")
		return
	}

	// Exchange code for token
	token, err := h.oauth2Service.ExchangeCode(r.Context(), provider, code)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "TOKEN_EXCHANGE_FAILED", "Failed to exchange code for token", err.Error())
		return
	}

	// Get user information from provider
	userInfo, err := h.oauth2Service.GetUserInfo(r.Context(), provider, token)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "USER_INFO_FAILED", "Failed to get user information", err.Error())
		return
	}

	// Create or update user
	user, err := h.oauth2Service.CreateOrUpdateUser(userInfo)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "USER_CREATION_FAILED", "Failed to create or update user", err.Error())
		return
	}

	// Generate JWT tokens
	authResponse, err := h.oauth2Service.GenerateTokens(user)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "TOKEN_GENERATION_FAILED", "Failed to generate tokens", err.Error())
		return
	}

	// Set secure cookie for refresh token
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    authResponse.RefreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   authResponse.ExpiresIn,
	})

	writeJSONResponse(w, http.StatusOK, authResponse, nil)
}

// GetOAuth2Providers handles GET /api/v1/auth/oauth2/providers
func (h *AuthHandlers) GetOAuth2Providers(w http.ResponseWriter, r *http.Request) {
	// Get available OAuth2 providers from the service
	providers := h.oauth2Service.GetAvailableProviders()
	writeJSONResponse(w, http.StatusOK, providers, nil)
}

// OAuth2Auth handles GET /api/v1/auth/oauth2/{provider}
func (h *AuthHandlers) OAuth2Auth(w http.ResponseWriter, r *http.Request) {
	// Extract provider from URL path
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		writeErrorResponse(w, http.StatusBadRequest, "INVALID_PATH", "Invalid OAuth2 auth path", "")
		return
	}
	provider := pathParts[len(pathParts)-1] // Get provider from path like /auth/oauth2/google

	// Generate state parameter for CSRF protection
	state := generateRandomState()

	// Get authorization URL
	authURL, err := h.oauth2Service.GetAuthURL(provider, state)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "PROVIDER_NOT_CONFIGURED", "OAuth2 provider not configured", err.Error())
		return
	}

	// Redirect to OAuth2 provider
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// Helper functions

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

// writeJSONResponse writes a JSON response
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}, error *ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"success": error == nil,
		"data":    data,
	}

	if error != nil {
		response["error"] = error
	}

	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse writes an error response
func writeErrorResponse(w http.ResponseWriter, statusCode int, code, message, details string) {
	writeJSONResponse(w, statusCode, nil, &ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	})
}

// writeValidationError writes a validation error response
func writeValidationError(w http.ResponseWriter, message string) {
	writeErrorResponse(w, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", message)
}

// generateRandomState generates a random state parameter for OAuth2
func generateRandomState() string {
	// Generate a random 32-character string
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 32)
	for i := range b {
		// Use a simple random number generator for demo purposes
		// In production, use crypto/rand
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
