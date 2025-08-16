package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// UserManagementAPI provides HTTP handlers for user management
type UserManagementAPI struct {
	authManager *AuthManager
}

// NewUserManagementAPI creates a new user management API
func NewUserManagementAPI(authManager *AuthManager) *UserManagementAPI {
	return &UserManagementAPI{authManager: authManager}
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// EnableMFARequest represents an MFA enable request
type EnableMFARequest struct {
	Token string `json:"token"`
}

// HandleCreateUser handles user creation
func (api *UserManagementAPI) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Validate password strength
	if err := api.authManager.passwordValidator.ValidatePassword(req.Password); err != nil {
		http.Error(w, fmt.Sprintf("Password validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Determine role
	role := RoleReadOnly
	if req.Role == "root" {
		role = RoleRoot
	}

	// Create user
	user := &User{
		ID:        generateUserID(),
		Username:  req.Username,
		Password:  hashedPassword,
		Role:      role,
		Created:   time.Now(),
		Email:     req.Email,
	}

	if err := api.authManager.userDB.CreateUser(user); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
		return
	}

	// Log audit event
	api.authManager.userDB.LogAuditEvent(&AuditEvent{
		UserID:    getCurrentUserID(r),
		Action:    "create_user",
		Resource:  fmt.Sprintf("user:%s", user.Username),
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Details:   fmt.Sprintf("Created user %s with role %s", user.Username, user.Role),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User created successfully",
		"user_id": user.ID,
	})
}

// HandleChangePassword handles password changes
func (api *UserManagementAPI) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current user
	currentUser := getCurrentUser(r)
	if currentUser == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Verify current password
	if err := ComparePassword(currentUser.Password, req.CurrentPassword); err != nil {
		http.Error(w, "Current password is incorrect", http.StatusBadRequest)
		return
	}

	// Validate new password
	if err := api.authManager.passwordValidator.ValidatePassword(req.NewPassword); err != nil {
		http.Error(w, fmt.Sprintf("Password validation failed: %v", err), http.StatusBadRequest)
		return
	}

	// Hash new password
	hashedPassword, err := HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Update user
	currentUser.Password = hashedPassword
	currentUser.PasswordChangedAt = time.Now()

	if err := api.authManager.userDB.UpdateUser(currentUser); err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	// Log audit event
	api.authManager.userDB.LogAuditEvent(&AuditEvent{
		UserID:    currentUser.ID,
		Action:    "change_password",
		Resource:  fmt.Sprintf("user:%s", currentUser.Username),
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Details:   "Password changed successfully",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Password changed successfully",
	})
}

// HandleEnableMFA handles MFA enablement
func (api *UserManagementAPI) HandleEnableMFA(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EnableMFARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get current user
	currentUser := getCurrentUser(r)
	if currentUser == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Generate MFA secret if not already set
	if currentUser.MFASecret == "" {
		secret, err := GenerateMFASecret()
		if err != nil {
			http.Error(w, "Failed to generate MFA secret", http.StatusInternalServerError)
			return
		}
		currentUser.MFASecret = secret
	}

	// Validate token
	if !ValidateMFAToken(currentUser.MFASecret, req.Token) {
		http.Error(w, "Invalid MFA token", http.StatusBadRequest)
		return
	}

	// Enable MFA
	currentUser.MFAEnabled = true

	if err := api.authManager.userDB.UpdateUser(currentUser); err != nil {
		http.Error(w, "Failed to enable MFA", http.StatusInternalServerError)
		return
	}

	// Log audit event
	api.authManager.userDB.LogAuditEvent(&AuditEvent{
		UserID:    currentUser.ID,
		Action:    "enable_mfa",
		Resource:  fmt.Sprintf("user:%s", currentUser.Username),
		IPAddress: getClientIP(r),
		UserAgent: r.UserAgent(),
		Timestamp: time.Now(),
		Details:   "MFA enabled successfully",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "MFA enabled successfully",
		"secret":  currentUser.MFASecret, // In production, this should be shown only once
	})
}

// HandleGetUsers handles user listing (admin only)
func (api *UserManagementAPI) HandleGetUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if user has permission
	currentUser := getCurrentUser(r)
	if currentUser == nil || currentUser.Role != RoleRoot {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// TODO: Implement user listing from database
	// For now, return a simple response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User listing not yet implemented",
	})
}

// HandleGetAuditLogs handles audit log retrieval (admin only)
func (api *UserManagementAPI) HandleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if user has permission
	currentUser := getCurrentUser(r)
	if currentUser == nil || currentUser.Role != RoleRoot {
		http.Error(w, "Insufficient permissions", http.StatusForbidden)
		return
	}

	// TODO: Implement audit log retrieval from database
	// For now, return a simple response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Audit logs not yet implemented",
	})
}

// Helper functions

func generateUserID() string {
	return fmt.Sprintf("user_%d", time.Now().UnixNano())
}

func getCurrentUser(r *http.Request) *User {
	if user := r.Context().Value("user"); user != nil {
		return user.(*User)
	}
	return nil
}

func getCurrentUserID(r *http.Request) string {
	if user := getCurrentUser(r); user != nil {
		return user.ID
	}
	return "unknown"
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
