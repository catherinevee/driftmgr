package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Permission represents a specific action that can be performed
type Permission string

const (
	// Discovery permissions
	PermissionDiscoveryRead   Permission = "discovery:read"
	PermissionDiscoveryWrite  Permission = "discovery:write"
	PermissionDiscoveryDelete Permission = "discovery:delete"

	// Drift permissions
	PermissionDriftRead   Permission = "drift:read"
	PermissionDriftWrite  Permission = "drift:write"
	PermissionDriftDelete Permission = "drift:delete"

	// Remediation permissions
	PermissionRemediationRead    Permission = "remediation:read"
	PermissionRemediationExecute Permission = "remediation:execute"
	PermissionRemediationApprove Permission = "remediation:approve"

	// State management permissions
	PermissionStateRead   Permission = "state:read"
	PermissionStateWrite  Permission = "state:write"
	PermissionStateDelete Permission = "state:delete"

	// Admin permissions
	PermissionAdminUsers  Permission = "admin:users"
	PermissionAdminRoles  Permission = "admin:roles"
	PermissionAdminConfig Permission = "admin:config"
	PermissionAdminAudit  Permission = "admin:audit"
)

// Role represents a collection of permissions
type Role struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Permissions []Permission           `json:"permissions"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// User represents a user with assigned roles
type User struct {
	ID        string                 `json:"id"`
	Username  string                 `json:"username"`
	Email     string                 `json:"email"`
	Roles     []string               `json:"roles"` // Role IDs
	Active    bool                   `json:"active"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	LastLogin time.Time              `json:"last_login,omitempty"`
}

// Policy represents an access control policy
type Policy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Effect      Effect                 `json:"effect"`
	Resources   []string               `json:"resources"`
	Actions     []string               `json:"actions"`
	Conditions  map[string]interface{} `json:"conditions,omitempty"`
	Priority    int                    `json:"priority"`
	Description string                 `json:"description"`
}

// Effect represents whether a policy allows or denies access
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// Manager handles RBAC operations
type Manager struct {
	mu       sync.RWMutex
	users    map[string]*User
	roles    map[string]*Role
	policies map[string]*Policy
	store    Store
}

// Store interface for persistence
type Store interface {
	LoadUsers() (map[string]*User, error)
	SaveUsers(users map[string]*User) error
	LoadRoles() (map[string]*Role, error)
	SaveRoles(roles map[string]*Role) error
	LoadPolicies() (map[string]*Policy, error)
	SavePolicies(policies map[string]*Policy) error
}

// NewManager creates a new RBAC manager
func NewManager(store Store) (*Manager, error) {
	m := &Manager{
		users:    make(map[string]*User),
		roles:    make(map[string]*Role),
		policies: make(map[string]*Policy),
		store:    store,
	}

	// Load existing data
	if err := m.load(); err != nil {
		return nil, err
	}

	// Initialize default roles if none exist
	if len(m.roles) == 0 {
		m.initializeDefaultRoles()
	}

	return m, nil
}

// CreateUser creates a new user
func (m *Manager) CreateUser(user *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[user.ID]; exists {
		return fmt.Errorf("user %s already exists", user.ID)
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.Active = true

	m.users[user.ID] = user
	return m.store.SaveUsers(m.users)
}

// GetUser retrieves a user by ID
func (m *Manager) GetUser(userID string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	return user, nil
}

// AssignRole assigns a role to a user
func (m *Manager) AssignRole(userID, roleID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	if _, exists := m.roles[roleID]; !exists {
		return fmt.Errorf("role %s not found", roleID)
	}

	// Check if role already assigned
	for _, r := range user.Roles {
		if r == roleID {
			return nil
		}
	}

	user.Roles = append(user.Roles, roleID)
	user.UpdatedAt = time.Now()

	return m.store.SaveUsers(m.users)
}

// RemoveRole removes a role from a user
func (m *Manager) RemoveRole(userID, roleID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	newRoles := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		if r != roleID {
			newRoles = append(newRoles, r)
		}
	}

	user.Roles = newRoles
	user.UpdatedAt = time.Now()

	return m.store.SaveUsers(m.users)
}

// CheckPermission checks if a user has a specific permission
func (m *Manager) CheckPermission(ctx context.Context, userID string, permission Permission) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return false, fmt.Errorf("user %s not found", userID)
	}

	if !user.Active {
		return false, errors.New("user is not active")
	}

	// Check each role for the permission
	for _, roleID := range user.Roles {
		role, exists := m.roles[roleID]
		if !exists {
			continue
		}

		for _, perm := range role.Permissions {
			if perm == permission {
				return true, nil
			}
		}
	}

	return false, nil
}

// CheckAccess checks if a user can perform an action on a resource
func (m *Manager) CheckAccess(ctx context.Context, userID, resource, action string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return false, fmt.Errorf("user %s not found", userID)
	}

	// Evaluate policies
	var allowPolicies, denyPolicies []*Policy

	for _, policy := range m.policies {
		if m.policyMatches(policy, resource, action) {
			if policy.Effect == EffectAllow {
				allowPolicies = append(allowPolicies, policy)
			} else {
				denyPolicies = append(denyPolicies, policy)
			}
		}
	}

	// Deny takes precedence
	if len(denyPolicies) > 0 {
		return false, nil
	}

	// Check if any allow policy matches
	return len(allowPolicies) > 0, nil
}

// CreateRole creates a new role
func (m *Manager) CreateRole(role *Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[role.ID]; exists {
		return fmt.Errorf("role %s already exists", role.ID)
	}

	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()

	m.roles[role.ID] = role
	return m.store.SaveRoles(m.roles)
}

// UpdateRole updates an existing role
func (m *Manager) UpdateRole(role *Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[role.ID]; !exists {
		return fmt.Errorf("role %s not found", role.ID)
	}

	role.UpdatedAt = time.Now()
	m.roles[role.ID] = role

	return m.store.SaveRoles(m.roles)
}

// DeleteRole deletes a role
func (m *Manager) DeleteRole(roleID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.roles[roleID]; !exists {
		return fmt.Errorf("role %s not found", roleID)
	}

	// Remove role from all users
	for _, user := range m.users {
		newRoles := make([]string, 0, len(user.Roles))
		for _, r := range user.Roles {
			if r != roleID {
				newRoles = append(newRoles, r)
			}
		}
		user.Roles = newRoles
	}

	delete(m.roles, roleID)

	if err := m.store.SaveRoles(m.roles); err != nil {
		return err
	}

	return m.store.SaveUsers(m.users)
}

// GetUserPermissions returns all permissions for a user
func (m *Manager) GetUserPermissions(userID string) ([]Permission, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[userID]
	if !exists {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	permSet := make(map[Permission]bool)

	for _, roleID := range user.Roles {
		role, exists := m.roles[roleID]
		if !exists {
			continue
		}

		for _, perm := range role.Permissions {
			permSet[perm] = true
		}
	}

	permissions := make([]Permission, 0, len(permSet))
	for perm := range permSet {
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// Private methods

func (m *Manager) load() error {
	var err error

	m.users, err = m.store.LoadUsers()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	m.roles, err = m.store.LoadRoles()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	m.policies, err = m.store.LoadPolicies()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (m *Manager) initializeDefaultRoles() {
	// Admin role
	m.roles["admin"] = &Role{
		ID:          "admin",
		Name:        "Administrator",
		Description: "Full system access",
		Permissions: []Permission{
			PermissionDiscoveryRead, PermissionDiscoveryWrite, PermissionDiscoveryDelete,
			PermissionDriftRead, PermissionDriftWrite, PermissionDriftDelete,
			PermissionRemediationRead, PermissionRemediationExecute, PermissionRemediationApprove,
			PermissionStateRead, PermissionStateWrite, PermissionStateDelete,
			PermissionAdminUsers, PermissionAdminRoles, PermissionAdminConfig, PermissionAdminAudit,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Operator role
	m.roles["operator"] = &Role{
		ID:          "operator",
		Name:        "Operator",
		Description: "Manage drift detection and remediation",
		Permissions: []Permission{
			PermissionDiscoveryRead, PermissionDiscoveryWrite,
			PermissionDriftRead, PermissionDriftWrite,
			PermissionRemediationRead, PermissionRemediationExecute,
			PermissionStateRead, PermissionStateWrite,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Viewer role
	m.roles["viewer"] = &Role{
		ID:          "viewer",
		Name:        "Viewer",
		Description: "Read-only access",
		Permissions: []Permission{
			PermissionDiscoveryRead,
			PermissionDriftRead,
			PermissionRemediationRead,
			PermissionStateRead,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Approver role
	m.roles["approver"] = &Role{
		ID:          "approver",
		Name:        "Approver",
		Description: "Approve remediation actions",
		Permissions: []Permission{
			PermissionDiscoveryRead,
			PermissionDriftRead,
			PermissionRemediationRead,
			PermissionRemediationApprove,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.store.SaveRoles(m.roles)
}

func (m *Manager) policyMatches(policy *Policy, resource, action string) bool {
	// Check resource match
	resourceMatch := false
	for _, r := range policy.Resources {
		if r == "*" || r == resource {
			resourceMatch = true
			break
		}
	}

	if !resourceMatch {
		return false
	}

	// Check action match
	for _, a := range policy.Actions {
		if a == "*" || a == action {
			return true
		}
	}

	return false
}

// FileStore implements file-based storage for RBAC
type FileStore struct {
	basePath string
}

// NewFileStore creates a new file-based store
func NewFileStore(basePath string) (*FileStore, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, err
	}
	return &FileStore{basePath: basePath}, nil
}

// LoadUsers loads users from file
func (s *FileStore) LoadUsers() (map[string]*User, error) {
	path := filepath.Join(s.basePath, "users.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]*User), err
	}

	var users map[string]*User
	if err := json.Unmarshal(data, &users); err != nil {
		return make(map[string]*User), err
	}

	return users, nil
}

// SaveUsers saves users to file
func (s *FileStore) SaveUsers(users map[string]*User) error {
	path := filepath.Join(s.basePath, "users.json")
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadRoles loads roles from file
func (s *FileStore) LoadRoles() (map[string]*Role, error) {
	path := filepath.Join(s.basePath, "roles.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]*Role), err
	}

	var roles map[string]*Role
	if err := json.Unmarshal(data, &roles); err != nil {
		return make(map[string]*Role), err
	}

	return roles, nil
}

// SaveRoles saves roles to file
func (s *FileStore) SaveRoles(roles map[string]*Role) error {
	path := filepath.Join(s.basePath, "roles.json")
	data, err := json.MarshalIndent(roles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadPolicies loads policies from file
func (s *FileStore) LoadPolicies() (map[string]*Policy, error) {
	path := filepath.Join(s.basePath, "policies.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]*Policy), err
	}

	var policies map[string]*Policy
	if err := json.Unmarshal(data, &policies); err != nil {
		return make(map[string]*Policy), err
	}

	return policies, nil
}

// SavePolicies saves policies to file
func (s *FileStore) SavePolicies(policies map[string]*Policy) error {
	path := filepath.Join(s.basePath, "policies.json")
	data, err := json.MarshalIndent(policies, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}