package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	sessions map[string]*Session
	users    map[string]*User
	mu       sync.RWMutex
}

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Email        string    `json:"email,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login,omitempty"`
}

// Session represents an active user session
type Session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents successful login response
type LoginResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler() *AuthHandler {
	ah := &AuthHandler{
		sessions: make(map[string]*Session),
		users:    make(map[string]*User),
	}
	
	// Initialize default users
	ah.initDefaultUsers()
	
	return ah
}

// initDefaultUsers creates default system users
func (ah *AuthHandler) initDefaultUsers() {
	// Root user
	rootPassword, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	ah.users["root"] = &User{
		ID:           "1",
		Username:     "root",
		PasswordHash: string(rootPassword),
		Role:         "admin",
		Email:        "admin@driftmgr.local",
		CreatedAt:    time.Now(),
	}
	
	// Readonly user
	readonlyPassword, _ := bcrypt.GenerateFromPassword([]byte("readonly"), bcrypt.DefaultCost)
	ah.users["readonly"] = &User{
		ID:           "2",
		Username:     "readonly",
		PasswordHash: string(readonlyPassword),
		Role:         "viewer",
		Email:        "readonly@driftmgr.local",
		CreatedAt:    time.Now(),
	}
	
	// Operator user
	operatorPassword, _ := bcrypt.GenerateFromPassword([]byte("operator"), bcrypt.DefaultCost)
	ah.users["operator"] = &User{
		ID:           "3",
		Username:     "operator",
		PasswordHash: string(operatorPassword),
		Role:         "operator",
		Email:        "operator@driftmgr.local",
		CreatedAt:    time.Now(),
	}
}

// HandleLogin handles user login requests
func (ah *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		sendJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate credentials
	ah.mu.RLock()
	user, exists := ah.users[loginReq.Username]
	ah.mu.RUnlock()
	
	if !exists {
		sendJSONError(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	
	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(loginReq.Password)); err != nil {
		sendJSONError(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	
	// Create session
	session := ah.createSession(user)
	
	// Update last login
	ah.mu.Lock()
	user.LastLogin = time.Now()
	ah.mu.Unlock()
	
	// Send response
	response := LoginResponse{
		Token: session.Token,
		User:  user,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleLogout handles user logout requests
func (ah *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	token := ah.extractToken(r)
	if token == "" {
		sendJSONError(w, "No token provided", http.StatusUnauthorized)
		return
	}
	
	ah.mu.Lock()
	delete(ah.sessions, token)
	ah.mu.Unlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Logged out successfully"})
}

// HandleValidate validates a session token
func (ah *AuthHandler) HandleValidate(w http.ResponseWriter, r *http.Request) {
	token := ah.extractToken(r)
	if token == "" {
		sendJSONError(w, "No token provided", http.StatusUnauthorized)
		return
	}
	
	ah.mu.RLock()
	session, exists := ah.sessions[token]
	ah.mu.RUnlock()
	
	if !exists || session.ExpiresAt.Before(time.Now()) {
		sendJSONError(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// HandleRefresh refreshes a session token
func (ah *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	oldToken := ah.extractToken(r)
	if oldToken == "" {
		sendJSONError(w, "No token provided", http.StatusUnauthorized)
		return
	}
	
	ah.mu.Lock()
	defer ah.mu.Unlock()
	
	session, exists := ah.sessions[oldToken]
	if !exists || session.ExpiresAt.Before(time.Now()) {
		sendJSONError(w, "Invalid or expired token", http.StatusUnauthorized)
		return
	}
	
	// Create new token
	newToken := ah.generateToken()
	newSession := &Session{
		Token:     newToken,
		UserID:    session.UserID,
		Username:  session.Username,
		Role:      session.Role,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	
	// Replace old session
	delete(ah.sessions, oldToken)
	ah.sessions[newToken] = newSession
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": newToken})
}

// Middleware validates authentication for protected routes
func (ah *AuthHandler) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for login endpoint
		if strings.HasSuffix(r.URL.Path, "/auth/login") {
			next(w, r)
			return
		}
		
		token := ah.extractToken(r)
		if token == "" {
			sendJSONError(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		
		ah.mu.RLock()
		session, exists := ah.sessions[token]
		ah.mu.RUnlock()
		
		if !exists || session.ExpiresAt.Before(time.Now()) {
			sendJSONError(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}
		
		// Add user info to request context
		r.Header.Set("X-User-ID", session.UserID)
		r.Header.Set("X-Username", session.Username)
		r.Header.Set("X-User-Role", session.Role)
		
		next(w, r)
	}
}

// RequireRole middleware checks if user has required role
func (ah *AuthHandler) RequireRole(role string, next http.HandlerFunc) http.HandlerFunc {
	return ah.Middleware(func(w http.ResponseWriter, r *http.Request) {
		userRole := r.Header.Get("X-User-Role")
		
		if !ah.hasPermission(userRole, role) {
			sendJSONError(w, "Insufficient permissions", http.StatusForbidden)
			return
		}
		
		next(w, r)
	})
}

// createSession creates a new user session
func (ah *AuthHandler) createSession(user *User) *Session {
	token := ah.generateToken()
	session := &Session{
		Token:     token,
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	
	ah.mu.Lock()
	ah.sessions[token] = session
	ah.mu.Unlock()
	
	return session
}

// generateToken generates a secure random token
func (ah *AuthHandler) generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// extractToken extracts token from request
func (ah *AuthHandler) extractToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}
	
	// Check X-Auth-Token header
	if token := r.Header.Get("X-Auth-Token"); token != "" {
		return token
	}
	
	// Check query parameter
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	
	// Check cookie
	if cookie, err := r.Cookie("auth_token"); err == nil {
		return cookie.Value
	}
	
	return ""
}

// hasPermission checks if a role has permission for required role
func (ah *AuthHandler) hasPermission(userRole, requiredRole string) bool {
	// Role hierarchy: admin > operator > viewer
	roleLevel := map[string]int{
		"admin":    3,
		"operator": 2,
		"viewer":   1,
	}
	
	userLevel, ok1 := roleLevel[userRole]
	requiredLevel, ok2 := roleLevel[requiredRole]
	
	if !ok1 || !ok2 {
		return false
	}
	
	return userLevel >= requiredLevel
}

// sendJSONError sends a JSON error response
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

// GetUserByToken returns user information for a given token
func (ah *AuthHandler) GetUserByToken(token string) (*User, error) {
	ah.mu.RLock()
	defer ah.mu.RUnlock()
	
	session, exists := ah.sessions[token]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	if session.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("session expired")
	}
	
	for _, user := range ah.users {
		if user.ID == session.UserID {
			return user, nil
		}
	}
	
	return nil, fmt.Errorf("user not found for session")
}

// RegisterRoutes registers authentication routes
func (ah *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/auth/login", ah.HandleLogin)
	mux.HandleFunc("/api/v1/auth/logout", ah.HandleLogout)
	mux.HandleFunc("/api/v1/auth/validate", ah.HandleValidate)
	mux.HandleFunc("/api/v1/auth/refresh", ah.HandleRefresh)
}