package auth

import (
	"context"
	"time"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// Basic CRUD operations
	Create(user *User) error
	GetByID(id string) (*User, error)
	GetByUsername(username string) (*User, error)
	GetByEmail(email string) (*User, error)
	Update(user *User) error
	Delete(id string) error
	List(limit, offset int) ([]*User, error)
	Count() (int, error)

	// Search and filtering
	Search(query string, limit, offset int) ([]*User, error)
	GetByRole(roleID string, limit, offset int) ([]*User, error)
	GetActiveUsers(limit, offset int) ([]*User, error)
	GetInactiveUsers(limit, offset int) ([]*User, error)

	// Bulk operations
	BulkUpdate(updates map[string]interface{}) error
	BulkDelete(ids []string) error

	// Context-aware operations
	CreateWithContext(ctx context.Context, user *User) error
	GetByIDWithContext(ctx context.Context, id string) (*User, error)
	UpdateWithContext(ctx context.Context, user *User) error
	DeleteWithContext(ctx context.Context, id string) error
}

// RoleRepository defines the interface for role data operations
type RoleRepository interface {
	// Basic CRUD operations
	Create(role *Role) error
	GetByID(id string) (*Role, error)
	GetByName(name string) (*Role, error)
	Update(role *Role) error
	Delete(id string) error
	List(limit, offset int) ([]*Role, error)
	Count() (int, error)

	// User-role relationships
	AssignRole(userRole *UserRole) error
	RemoveRole(userID, roleID string) error
	GetByUserID(userID string) ([]Role, error)
	GetUsersByRole(roleID string, limit, offset int) ([]*User, error)

	// Permission management
	AddPermission(roleID, permission string) error
	RemovePermission(roleID, permission string) error
	GetPermissions(roleID string) ([]string, error)

	// Context-aware operations
	CreateWithContext(ctx context.Context, role *Role) error
	GetByIDWithContext(ctx context.Context, id string) (*Role, error)
	UpdateWithContext(ctx context.Context, role *Role) error
	DeleteWithContext(ctx context.Context, id string) error
}

// SessionRepository defines the interface for session data operations
type SessionRepository interface {
	// Basic CRUD operations
	Create(session *Session) error
	GetByID(id string) (*Session, error)
	GetByToken(token string) (*Session, error)
	GetByRefreshToken(refreshToken string) (*Session, error)
	Update(session *Session) error
	Delete(id string) error
	DeleteByToken(token string) error

	// User session management
	GetByUserID(userID string, limit, offset int) ([]*Session, error)
	DeleteByUserID(userID string) error
	DeleteExpired() error
	DeleteInactive(olderThan time.Time) error

	// Session validation
	IsValid(token string) (bool, error)
	Refresh(session *Session) error

	// Context-aware operations
	CreateWithContext(ctx context.Context, session *Session) error
	GetByTokenWithContext(ctx context.Context, token string) (*Session, error)
	UpdateWithContext(ctx context.Context, session *Session) error
	DeleteWithContext(ctx context.Context, id string) error
}

// APIKeyRepository defines the interface for API key data operations
type APIKeyRepository interface {
	// Basic CRUD operations
	Create(apiKey *APIKey) error
	GetByID(id string) (*APIKey, error)
	GetByUserID(userID string) ([]*APIKey, error)
	Update(apiKey *APIKey) error
	Delete(id string) error
	List(limit, offset int) ([]*APIKey, error)
	Count() (int, error)

	// API key validation
	Validate(key string) (*APIKey, error)
	GetByPrefix(prefix string) (*APIKey, error)

	// User API key management
	GetActiveByUserID(userID string) ([]*APIKey, error)
	GetExpiredByUserID(userID string) ([]*APIKey, error)
	DeleteByUserID(userID string) error
	DeleteExpired() error

	// Permission management
	UpdatePermissions(id string, permissions []string) error
	GetByPermission(permission string, limit, offset int) ([]*APIKey, error)

	// Context-aware operations
	CreateWithContext(ctx context.Context, apiKey *APIKey) error
	GetByIDWithContext(ctx context.Context, id string) (*APIKey, error)
	UpdateWithContext(ctx context.Context, apiKey *APIKey) error
	DeleteWithContext(ctx context.Context, id string) error
}

// PermissionRepository defines the interface for permission data operations
type PermissionRepository interface {
	// Basic CRUD operations
	Create(permission *Permission) error
	GetByID(id string) (*Permission, error)
	GetByName(name string) (*Permission, error)
	Update(permission *Permission) error
	Delete(id string) error
	List(limit, offset int) ([]*Permission, error)
	Count() (int, error)

	// Resource and action filtering
	GetByResource(resource string) ([]*Permission, error)
	GetByAction(action string) ([]*Permission, error)
	GetByResourceAndAction(resource, action string) (*Permission, error)

	// Role permission management
	GetByRole(roleID string) ([]*Permission, error)
	AddToRole(roleID, permissionID string) error
	RemoveFromRole(roleID, permissionID string) error

	// Context-aware operations
	CreateWithContext(ctx context.Context, permission *Permission) error
	GetByIDWithContext(ctx context.Context, id string) (*Permission, error)
	UpdateWithContext(ctx context.Context, permission *Permission) error
	DeleteWithContext(ctx context.Context, id string) error
}

// OAuth2ProviderRepository defines the interface for OAuth2 provider data operations
type OAuth2ProviderRepository interface {
	// Basic CRUD operations
	Create(provider *OAuth2Provider) error
	GetByID(id string) (*OAuth2Provider, error)
	GetByName(name string) (*OAuth2Provider, error)
	Update(provider *OAuth2Provider) error
	Delete(id string) error
	List(limit, offset int) ([]*OAuth2Provider, error)
	Count() (int, error)

	// Active provider management
	GetActive() ([]*OAuth2Provider, error)
	GetInactive() ([]*OAuth2Provider, error)
	SetActive(id string, active bool) error

	// Context-aware operations
	CreateWithContext(ctx context.Context, provider *OAuth2Provider) error
	GetByIDWithContext(ctx context.Context, id string) (*OAuth2Provider, error)
	UpdateWithContext(ctx context.Context, provider *OAuth2Provider) error
	DeleteWithContext(ctx context.Context, id string) error
}

// TransactionRepository defines the interface for database transactions
type TransactionRepository interface {
	// Transaction management
	Begin() (Transaction, error)
	BeginWithContext(ctx context.Context) (Transaction, error)
}

// Transaction defines the interface for database transactions
type Transaction interface {
	// Transaction control
	Commit() error
	Rollback() error

	// Repository access within transaction
	UserRepository() UserRepository
	RoleRepository() RoleRepository
	SessionRepository() SessionRepository
	APIKeyRepository() APIKeyRepository
	PermissionRepository() PermissionRepository
	OAuth2ProviderRepository() OAuth2ProviderRepository
}

// RepositoryManager provides access to all repositories
type RepositoryManager interface {
	// Repository access
	Users() UserRepository
	Roles() RoleRepository
	Sessions() SessionRepository
	APIKeys() APIKeyRepository
	Permissions() PermissionRepository
	OAuth2Providers() OAuth2ProviderRepository
	Transactions() TransactionRepository

	// Health check
	Health() error
	Close() error
}
