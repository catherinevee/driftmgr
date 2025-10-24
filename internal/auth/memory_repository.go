package auth

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MemoryUserRepository is an in-memory implementation of UserRepository
type MemoryUserRepository struct {
	users map[string]*User
	mu    sync.RWMutex
}

// NewMemoryUserRepository creates a new in-memory user repository
func NewMemoryUserRepository() *MemoryUserRepository {
	repo := &MemoryUserRepository{
		users: make(map[string]*User),
	}

	// Create default admin user
	adminUser := &User{
		ID:           "admin-user-id",
		Username:     "admin",
		Email:        "admin@driftmgr.com",
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=2$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG", // "admin123"
		FirstName:    "Admin",
		LastName:     "User",
		IsActive:     true,
		IsAdmin:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	repo.users[adminUser.ID] = adminUser
	repo.users[adminUser.Username] = adminUser
	repo.users[adminUser.Email] = adminUser

	return repo
}

// Create creates a new user
func (r *MemoryUserRepository) Create(user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if username already exists
	for _, u := range r.users {
		if u.Username == user.Username {
			return errors.New("username already exists")
		}
		if u.Email == user.Email {
			return errors.New("email already exists")
		}
	}

	r.users[user.ID] = user
	r.users[user.Username] = user
	r.users[user.Email] = user
	return nil
}

// GetByID retrieves a user by ID
func (r *MemoryUserRepository) GetByID(id string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// GetByUsername retrieves a user by username
func (r *MemoryUserRepository) GetByUsername(username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[username]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// GetByEmail retrieves a user by email
func (r *MemoryUserRepository) GetByEmail(email string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[email]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// Update updates a user
func (r *MemoryUserRepository) Update(user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.ID]; !exists {
		return errors.New("user not found")
	}

	user.UpdatedAt = time.Now()
	r.users[user.ID] = user
	r.users[user.Username] = user
	r.users[user.Email] = user
	return nil
}

// Delete deletes a user
func (r *MemoryUserRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[id]
	if !exists {
		return errors.New("user not found")
	}

	delete(r.users, user.ID)
	delete(r.users, user.Username)
	delete(r.users, user.Email)
	return nil
}

// List returns a list of users
func (r *MemoryUserRepository) List(limit, offset int) ([]*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*User, 0)
	count := 0
	for _, user := range r.users {
		if count >= offset && len(users) < limit {
			users = append(users, user)
		}
		count++
	}
	return users, nil
}

// Count returns the total number of users
func (r *MemoryUserRepository) Count() (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Count unique users (not duplicates by username/email)
	uniqueUsers := make(map[string]bool)
	for _, user := range r.users {
		uniqueUsers[user.ID] = true
	}
	return len(uniqueUsers), nil
}

// Search searches for users
func (r *MemoryUserRepository) Search(query string, limit, offset int) ([]*User, error) {
	// Simple implementation - in production, you'd use proper search
	return r.List(limit, offset)
}

// GetByRole returns users with a specific role
func (r *MemoryUserRepository) GetByRole(roleID string, limit, offset int) ([]*User, error) {
	// Simple implementation
	return r.List(limit, offset)
}

// GetActiveUsers returns active users
func (r *MemoryUserRepository) GetActiveUsers(limit, offset int) ([]*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*User, 0)
	count := 0
	for _, user := range r.users {
		if user.IsActive && count >= offset && len(users) < limit {
			users = append(users, user)
		}
		count++
	}
	return users, nil
}

// GetInactiveUsers returns inactive users
func (r *MemoryUserRepository) GetInactiveUsers(limit, offset int) ([]*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	users := make([]*User, 0)
	count := 0
	for _, user := range r.users {
		if !user.IsActive && count >= offset && len(users) < limit {
			users = append(users, user)
		}
		count++
	}
	return users, nil
}

// BulkUpdate performs bulk updates
func (r *MemoryUserRepository) BulkUpdate(updates map[string]interface{}) error {
	// Simple implementation
	return nil
}

// BulkDelete performs bulk deletes
func (r *MemoryUserRepository) BulkDelete(ids []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, id := range ids {
		if user, exists := r.users[id]; exists {
			delete(r.users, user.ID)
			delete(r.users, user.Username)
			delete(r.users, user.Email)
		}
	}
	return nil
}

// Context-aware operations (simple implementations)
func (r *MemoryUserRepository) CreateWithContext(ctx context.Context, user *User) error {
	return r.Create(user)
}

func (r *MemoryUserRepository) GetByIDWithContext(ctx context.Context, id string) (*User, error) {
	return r.GetByID(id)
}

func (r *MemoryUserRepository) UpdateWithContext(ctx context.Context, user *User) error {
	return r.Update(user)
}

func (r *MemoryUserRepository) DeleteWithContext(ctx context.Context, id string) error {
	return r.Delete(id)
}

// MemoryRoleRepository is an in-memory implementation of RoleRepository
type MemoryRoleRepository struct {
	roles     map[string]*Role
	userRoles map[string][]string // userID -> roleIDs
	mu        sync.RWMutex
}

// NewMemoryRoleRepository creates a new in-memory role repository
func NewMemoryRoleRepository() *MemoryRoleRepository {
	repo := &MemoryRoleRepository{
		roles:     make(map[string]*Role),
		userRoles: make(map[string][]string),
	}

	// Create default roles
	defaultRoles := []*Role{
		{
			ID:          "admin-role-id",
			Name:        RoleAdmin,
			Description: "Administrator with full access",
			Permissions: DefaultRoles[RoleAdmin],
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "operator-role-id",
			Name:        RoleOperator,
			Description: "Operator with read/write access",
			Permissions: DefaultRoles[RoleOperator],
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "viewer-role-id",
			Name:        RoleViewer,
			Description: "Viewer with read-only access",
			Permissions: DefaultRoles[RoleViewer],
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "auditor-role-id",
			Name:        RoleAuditor,
			Description: "Auditor with read access and audit capabilities",
			Permissions: DefaultRoles[RoleAuditor],
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, role := range defaultRoles {
		repo.roles[role.ID] = role
		repo.roles[role.Name] = role
	}

	// Assign admin role to admin user
	repo.userRoles["admin-user-id"] = []string{"admin-role-id"}

	return repo
}

// Create creates a new role
func (r *MemoryRoleRepository) Create(role *Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.roles[role.ID] = role
	r.roles[role.Name] = role
	return nil
}

// GetByID retrieves a role by ID
func (r *MemoryRoleRepository) GetByID(id string) (*Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	role, exists := r.roles[id]
	if !exists {
		return nil, errors.New("role not found")
	}
	return role, nil
}

// GetByName retrieves a role by name
func (r *MemoryRoleRepository) GetByName(name string) (*Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	role, exists := r.roles[name]
	if !exists {
		return nil, errors.New("role not found")
	}
	return role, nil
}

// Update updates a role
func (r *MemoryRoleRepository) Update(role *Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.roles[role.ID]; !exists {
		return errors.New("role not found")
	}

	role.UpdatedAt = time.Now()
	r.roles[role.ID] = role
	r.roles[role.Name] = role
	return nil
}

// Delete deletes a role
func (r *MemoryRoleRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	role, exists := r.roles[id]
	if !exists {
		return errors.New("role not found")
	}

	delete(r.roles, role.ID)
	delete(r.roles, role.Name)
	return nil
}

// List returns a list of roles
func (r *MemoryRoleRepository) List(limit, offset int) ([]*Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles := make([]*Role, 0)
	count := 0
	for _, role := range r.roles {
		if count >= offset && len(roles) < limit {
			roles = append(roles, role)
		}
		count++
	}
	return roles, nil
}

// Count returns the total number of roles
func (r *MemoryRoleRepository) Count() (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Count unique roles (not duplicates by name)
	uniqueRoles := make(map[string]bool)
	for _, role := range r.roles {
		uniqueRoles[role.ID] = true
	}
	return len(uniqueRoles), nil
}

// AssignRole assigns a role to a user
func (r *MemoryRoleRepository) AssignRole(userRole *UserRole) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.userRoles[userRole.UserID] = append(r.userRoles[userRole.UserID], userRole.RoleID)
	return nil
}

// RemoveRole removes a role from a user
func (r *MemoryRoleRepository) RemoveRole(userID, roleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	roles := r.userRoles[userID]
	for i, rid := range roles {
		if rid == roleID {
			r.userRoles[userID] = append(roles[:i], roles[i+1:]...)
			break
		}
	}
	return nil
}

// GetByUserID returns roles for a user
func (r *MemoryRoleRepository) GetByUserID(userID string) ([]Role, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roleIDs := r.userRoles[userID]
	roles := make([]Role, 0, len(roleIDs))

	for _, roleID := range roleIDs {
		if role, exists := r.roles[roleID]; exists {
			roles = append(roles, *role)
		}
	}
	return roles, nil
}

// GetUsersByRole returns users with a specific role
func (r *MemoryRoleRepository) GetUsersByRole(roleID string, limit, offset int) ([]*User, error) {
	// Simple implementation
	return []*User{}, nil
}

// AddPermission adds a permission to a role
func (r *MemoryRoleRepository) AddPermission(roleID, permission string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	role, exists := r.roles[roleID]
	if !exists {
		return errors.New("role not found")
	}

	// Check if permission already exists
	for _, p := range role.Permissions {
		if p == permission {
			return nil // Already exists
		}
	}

	role.Permissions = append(role.Permissions, permission)
	role.UpdatedAt = time.Now()
	return nil
}

// RemovePermission removes a permission from a role
func (r *MemoryRoleRepository) RemovePermission(roleID, permission string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	role, exists := r.roles[roleID]
	if !exists {
		return errors.New("role not found")
	}

	for i, p := range role.Permissions {
		if p == permission {
			role.Permissions = append(role.Permissions[:i], role.Permissions[i+1:]...)
			role.UpdatedAt = time.Now()
			break
		}
	}
	return nil
}

// GetPermissions returns permissions for a role
func (r *MemoryRoleRepository) GetPermissions(roleID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	role, exists := r.roles[roleID]
	if !exists {
		return nil, errors.New("role not found")
	}
	return role.Permissions, nil
}

// Context-aware operations
func (r *MemoryRoleRepository) CreateWithContext(ctx context.Context, role *Role) error {
	return r.Create(role)
}

func (r *MemoryRoleRepository) GetByIDWithContext(ctx context.Context, id string) (*Role, error) {
	return r.GetByID(id)
}

func (r *MemoryRoleRepository) UpdateWithContext(ctx context.Context, role *Role) error {
	return r.Update(role)
}

func (r *MemoryRoleRepository) DeleteWithContext(ctx context.Context, id string) error {
	return r.Delete(id)
}

// MemorySessionRepository is an in-memory implementation of SessionRepository
type MemorySessionRepository struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewMemorySessionRepository creates a new in-memory session repository
func NewMemorySessionRepository() *MemorySessionRepository {
	return &MemorySessionRepository{
		sessions: make(map[string]*Session),
	}
}

// Create creates a new session
func (r *MemorySessionRepository) Create(session *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[session.ID] = session
	r.sessions[session.Token] = session
	if session.RefreshToken != "" {
		r.sessions[session.RefreshToken] = session
	}
	return nil
}

// GetByID retrieves a session by ID
func (r *MemorySessionRepository) GetByID(id string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[id]
	if !exists {
		return nil, errors.New("session not found")
	}
	return session, nil
}

// GetByToken retrieves a session by token
func (r *MemorySessionRepository) GetByToken(token string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[token]
	if !exists {
		return nil, errors.New("session not found")
	}
	return session, nil
}

// GetByRefreshToken retrieves a session by refresh token
func (r *MemorySessionRepository) GetByRefreshToken(refreshToken string) (*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[refreshToken]
	if !exists {
		return nil, errors.New("session not found")
	}
	return session, nil
}

// Update updates a session
func (r *MemorySessionRepository) Update(session *Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[session.ID]; !exists {
		return errors.New("session not found")
	}

	r.sessions[session.ID] = session
	r.sessions[session.Token] = session
	if session.RefreshToken != "" {
		r.sessions[session.RefreshToken] = session
	}
	return nil
}

// Delete deletes a session
func (r *MemorySessionRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[id]
	if !exists {
		return errors.New("session not found")
	}

	delete(r.sessions, session.ID)
	delete(r.sessions, session.Token)
	if session.RefreshToken != "" {
		delete(r.sessions, session.RefreshToken)
	}
	return nil
}

// DeleteByToken deletes a session by token
func (r *MemorySessionRepository) DeleteByToken(token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[token]
	if !exists {
		return errors.New("session not found")
	}

	delete(r.sessions, session.ID)
	delete(r.sessions, session.Token)
	if session.RefreshToken != "" {
		delete(r.sessions, session.RefreshToken)
	}
	return nil
}

// GetByUserID returns sessions for a user
func (r *MemorySessionRepository) GetByUserID(userID string, limit, offset int) ([]*Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessions := make([]*Session, 0)
	count := 0
	for _, session := range r.sessions {
		if session.UserID == userID && count >= offset && len(sessions) < limit {
			sessions = append(sessions, session)
		}
		count++
	}
	return sessions, nil
}

// DeleteByUserID deletes all sessions for a user
func (r *MemorySessionRepository) DeleteByUserID(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, session := range r.sessions {
		if session.UserID == userID {
			delete(r.sessions, id)
			delete(r.sessions, session.Token)
			if session.RefreshToken != "" {
				delete(r.sessions, session.RefreshToken)
			}
		}
	}
	return nil
}

// DeleteExpired deletes expired sessions
func (r *MemorySessionRepository) DeleteExpired() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for id, session := range r.sessions {
		if session.ExpiresAt.Before(now) {
			delete(r.sessions, id)
			delete(r.sessions, session.Token)
			if session.RefreshToken != "" {
				delete(r.sessions, session.RefreshToken)
			}
		}
	}
	return nil
}

// DeleteInactive deletes inactive sessions
func (r *MemorySessionRepository) DeleteInactive(olderThan time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, session := range r.sessions {
		if session.LastUsedAt.Before(olderThan) {
			delete(r.sessions, id)
			delete(r.sessions, session.Token)
			if session.RefreshToken != "" {
				delete(r.sessions, session.RefreshToken)
			}
		}
	}
	return nil
}

// IsValid checks if a session is valid
func (r *MemorySessionRepository) IsValid(token string) (bool, error) {
	session, err := r.GetByToken(token)
	if err != nil {
		return false, nil
	}
	return session.IsValid(), nil
}

// Refresh refreshes a session
func (r *MemorySessionRepository) Refresh(session *Session) error {
	return r.Update(session)
}

// Context-aware operations
func (r *MemorySessionRepository) CreateWithContext(ctx context.Context, session *Session) error {
	return r.Create(session)
}

func (r *MemorySessionRepository) GetByTokenWithContext(ctx context.Context, token string) (*Session, error) {
	return r.GetByToken(token)
}

func (r *MemorySessionRepository) UpdateWithContext(ctx context.Context, session *Session) error {
	return r.Update(session)
}

func (r *MemorySessionRepository) DeleteWithContext(ctx context.Context, id string) error {
	return r.Delete(id)
}

// MemoryAPIKeyRepository is an in-memory implementation of APIKeyRepository
type MemoryAPIKeyRepository struct {
	apiKeys map[string]*APIKey
	mu      sync.RWMutex
}

// NewMemoryAPIKeyRepository creates a new in-memory API key repository
func NewMemoryAPIKeyRepository() *MemoryAPIKeyRepository {
	return &MemoryAPIKeyRepository{
		apiKeys: make(map[string]*APIKey),
	}
}

// Create creates a new API key
func (r *MemoryAPIKeyRepository) Create(apiKey *APIKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.apiKeys[apiKey.ID] = apiKey
	return nil
}

// GetByID retrieves an API key by ID
func (r *MemoryAPIKeyRepository) GetByID(id string) (*APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	apiKey, exists := r.apiKeys[id]
	if !exists {
		return nil, errors.New("API key not found")
	}
	return apiKey, nil
}

// GetByUserID retrieves API keys for a user
func (r *MemoryAPIKeyRepository) GetByUserID(userID string) ([]*APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	apiKeys := make([]*APIKey, 0)
	for _, apiKey := range r.apiKeys {
		if apiKey.UserID == userID {
			apiKeys = append(apiKeys, apiKey)
		}
	}
	return apiKeys, nil
}

// Update updates an API key
func (r *MemoryAPIKeyRepository) Update(apiKey *APIKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.apiKeys[apiKey.ID]; !exists {
		return errors.New("API key not found")
	}

	apiKey.UpdatedAt = time.Now()
	r.apiKeys[apiKey.ID] = apiKey
	return nil
}

// Delete deletes an API key
func (r *MemoryAPIKeyRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.apiKeys[id]; !exists {
		return errors.New("API key not found")
	}

	delete(r.apiKeys, id)
	return nil
}

// List returns a list of API keys
func (r *MemoryAPIKeyRepository) List(limit, offset int) ([]*APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	apiKeys := make([]*APIKey, 0)
	count := 0
	for _, apiKey := range r.apiKeys {
		if count >= offset && len(apiKeys) < limit {
			apiKeys = append(apiKeys, apiKey)
		}
		count++
	}
	return apiKeys, nil
}

// Count returns the total number of API keys
func (r *MemoryAPIKeyRepository) Count() (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.apiKeys), nil
}

// Validate validates an API key
func (r *MemoryAPIKeyRepository) Validate(key string) (*APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Iterate through all API keys to find a match
	for _, apiKey := range r.apiKeys {
		if apiKey.IsValid() {
			// In a real implementation, we would use a password service to verify the key
			// For now, we'll implement a simple comparison (this should be replaced with proper hashing)
			// The actual verification should be done in the service layer using PasswordService.VerifyAPIKey
			if apiKey.KeyHash != "" {
				// This is a placeholder - real implementation would hash the provided key
				// and compare with the stored hash using constant time comparison
				return apiKey, nil
			}
		}
	}

	return nil, errors.New("invalid API key")
}

// GetByPrefix retrieves an API key by prefix
func (r *MemoryAPIKeyRepository) GetByPrefix(prefix string) (*APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, apiKey := range r.apiKeys {
		if apiKey.KeyPrefix == prefix {
			return apiKey, nil
		}
	}
	return nil, errors.New("API key not found")
}

// GetActiveByUserID returns active API keys for a user
func (r *MemoryAPIKeyRepository) GetActiveByUserID(userID string) ([]*APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	apiKeys := make([]*APIKey, 0)
	for _, apiKey := range r.apiKeys {
		if apiKey.UserID == userID && apiKey.IsValid() {
			apiKeys = append(apiKeys, apiKey)
		}
	}
	return apiKeys, nil
}

// GetExpiredByUserID returns expired API keys for a user
func (r *MemoryAPIKeyRepository) GetExpiredByUserID(userID string) ([]*APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	apiKeys := make([]*APIKey, 0)
	for _, apiKey := range r.apiKeys {
		if apiKey.UserID == userID && apiKey.IsExpired() {
			apiKeys = append(apiKeys, apiKey)
		}
	}
	return apiKeys, nil
}

// DeleteByUserID deletes all API keys for a user
func (r *MemoryAPIKeyRepository) DeleteByUserID(userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, apiKey := range r.apiKeys {
		if apiKey.UserID == userID {
			delete(r.apiKeys, id)
		}
	}
	return nil
}

// DeleteExpired deletes expired API keys
func (r *MemoryAPIKeyRepository) DeleteExpired() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, apiKey := range r.apiKeys {
		if apiKey.IsExpired() {
			delete(r.apiKeys, id)
		}
	}
	return nil
}

// UpdatePermissions updates permissions for an API key
func (r *MemoryAPIKeyRepository) UpdatePermissions(id string, permissions []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	apiKey, exists := r.apiKeys[id]
	if !exists {
		return errors.New("API key not found")
	}

	apiKey.Permissions = permissions
	apiKey.UpdatedAt = time.Now()
	return nil
}

// GetByPermission returns API keys with a specific permission
func (r *MemoryAPIKeyRepository) GetByPermission(permission string, limit, offset int) ([]*APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	apiKeys := make([]*APIKey, 0)
	count := 0
	for _, apiKey := range r.apiKeys {
		for _, p := range apiKey.Permissions {
			if p == permission && count >= offset && len(apiKeys) < limit {
				apiKeys = append(apiKeys, apiKey)
				count++
				break
			}
		}
	}
	return apiKeys, nil
}

// Context-aware operations
func (r *MemoryAPIKeyRepository) CreateWithContext(ctx context.Context, apiKey *APIKey) error {
	return r.Create(apiKey)
}

func (r *MemoryAPIKeyRepository) GetByIDWithContext(ctx context.Context, id string) (*APIKey, error) {
	return r.GetByID(id)
}

func (r *MemoryAPIKeyRepository) UpdateWithContext(ctx context.Context, apiKey *APIKey) error {
	return r.Update(apiKey)
}

func (r *MemoryAPIKeyRepository) DeleteWithContext(ctx context.Context, id string) error {
	return r.Delete(id)
}
