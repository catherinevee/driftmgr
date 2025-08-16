package security

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// UserDB represents the database interface for user management
type UserDB struct {
	db *sql.DB
}

// NewUserDB creates a new database connection for user management
func NewUserDB(dbPath string) (*UserDB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	userDB := &UserDB{db: db}
	if err := userDB.initializeTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return userDB, nil
}

// initializeTables creates the necessary database tables
func (udb *UserDB) initializeTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			last_login DATETIME,
			password_changed_at DATETIME NOT NULL,
			failed_login_attempts INTEGER DEFAULT 0,
			locked_until DATETIME,
			email TEXT,
			mfa_enabled BOOLEAN DEFAULT FALSE,
			mfa_secret TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS user_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token_hash TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT,
			action TEXT NOT NULL,
			resource TEXT,
			ip_address TEXT,
			user_agent TEXT,
			timestamp DATETIME NOT NULL,
			details TEXT,
			FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS password_policies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			min_length INTEGER DEFAULT 8,
			require_uppercase BOOLEAN DEFAULT TRUE,
			require_lowercase BOOLEAN DEFAULT TRUE,
			require_numbers BOOLEAN DEFAULT TRUE,
			require_special_chars BOOLEAN DEFAULT TRUE,
			max_age_days INTEGER DEFAULT 90,
			prevent_reuse_count INTEGER DEFAULT 5,
			lockout_threshold INTEGER DEFAULT 5,
			lockout_duration_minutes INTEGER DEFAULT 30
		)`,
	}

	for _, query := range queries {
		if _, err := udb.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	// Initialize default password policy if not exists
	if err := udb.initializeDefaultPasswordPolicy(); err != nil {
		return fmt.Errorf("failed to initialize password policy: %w", err)
	}

	return nil
}

// initializeDefaultPasswordPolicy creates default password policy
func (udb *UserDB) initializeDefaultPasswordPolicy() error {
	_, err := udb.db.Exec(`
		INSERT OR IGNORE INTO password_policies 
		(min_length, require_uppercase, require_lowercase, require_numbers, require_special_chars, max_age_days, prevent_reuse_count, lockout_threshold, lockout_duration_minutes)
		VALUES (8, TRUE, TRUE, TRUE, TRUE, 90, 5, 5, 30)
	`)
	return err
}

// CreateUser creates a new user in the database
func (udb *UserDB) CreateUser(user *User) error {
	query := `
		INSERT INTO users (id, username, password_hash, role, created_at, password_changed_at, email)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := udb.db.Exec(query, 
		user.ID, 
		user.Username, 
		user.Password, 
		user.Role, 
		user.Created, 
		time.Now(),
		user.Email,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	
	return nil
}

// GetUserByUsername retrieves a user by username
func (udb *UserDB) GetUserByUsername(username string) (*User, error) {
	query := `
		SELECT id, username, password_hash, role, created_at, last_login, email, 
		       failed_login_attempts, locked_until, mfa_enabled
		FROM users WHERE username = ?
	`
	
	var user User
	var lastLogin, lockedUntil sql.NullTime
	
	err := udb.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Password, &user.Role, &user.Created,
		&lastLogin, &user.Email, &user.FailedLoginAttempts, &lockedUntil, &user.MFAEnabled,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	if lastLogin.Valid {
		user.LastLogin = lastLogin.Time
	}
	
	if lockedUntil.Valid {
		user.LockedUntil = &lockedUntil.Time
	}
	
	return &user, nil
}

// UpdateUser updates user information
func (udb *UserDB) UpdateUser(user *User) error {
	query := `
		UPDATE users 
		SET password_hash = ?, role = ?, last_login = ?, failed_login_attempts = ?, 
		    locked_until = ?, mfa_enabled = ?, mfa_secret = ?
		WHERE id = ?
	`
	
	var lockedUntil interface{}
	if user.LockedUntil != nil {
		lockedUntil = user.LockedUntil
	}
	
	_, err := udb.db.Exec(query,
		user.Password, user.Role, user.LastLogin, user.FailedLoginAttempts,
		lockedUntil, user.MFAEnabled, user.MFASecret, user.ID,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	
	return nil
}

// CreateSession creates a new user session
func (udb *UserDB) CreateSession(session *UserSession) error {
	query := `
		INSERT INTO user_sessions (id, user_id, token_hash, created_at, expires_at, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := udb.db.Exec(query,
		session.ID, session.UserID, session.TokenHash, session.CreatedAt,
		session.ExpiresAt, session.IPAddress, session.UserAgent,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	
	return nil
}

// GetSession retrieves a session by token hash
func (udb *UserDB) GetSession(tokenHash string) (*UserSession, error) {
	query := `
		SELECT id, user_id, token_hash, created_at, expires_at, ip_address, user_agent
		FROM user_sessions WHERE token_hash = ?
	`
	
	var session UserSession
	err := udb.db.QueryRow(query, tokenHash).Scan(
		&session.ID, &session.UserID, &session.TokenHash, &session.CreatedAt,
		&session.ExpiresAt, &session.IPAddress, &session.UserAgent,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	
	return &session, nil
}

// DeleteSession deletes a session
func (udb *UserDB) DeleteSession(sessionID string) error {
	query := `DELETE FROM user_sessions WHERE id = ?`
	
	_, err := udb.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	
	return nil
}

// CleanupExpiredSessions removes expired sessions
func (udb *UserDB) CleanupExpiredSessions() error {
	query := `DELETE FROM user_sessions WHERE expires_at < ?`
	
	_, err := udb.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	
	return nil
}

// LogAuditEvent logs an audit event
func (udb *UserDB) LogAuditEvent(event *AuditEvent) error {
	query := `
		INSERT INTO audit_logs (user_id, action, resource, ip_address, user_agent, timestamp, details)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := udb.db.Exec(query,
		event.UserID, event.Action, event.Resource, event.IPAddress,
		event.UserAgent, event.Timestamp, event.Details,
	)
	
	if err != nil {
		return fmt.Errorf("failed to log audit event: %w", err)
	}
	
	return nil
}

// GetPasswordPolicy retrieves the current password policy
func (udb *UserDB) GetPasswordPolicy() (*PasswordPolicy, error) {
	query := `
		SELECT min_length, require_uppercase, require_lowercase, require_numbers, 
		       require_special_chars, max_age_days, prevent_reuse_count, lockout_threshold, lockout_duration_minutes
		FROM password_policies LIMIT 1
	`
	
	var policy PasswordPolicy
	err := udb.db.QueryRow(query).Scan(
		&policy.MinLength, &policy.RequireUppercase, &policy.RequireLowercase,
		&policy.RequireNumbers, &policy.RequireSpecialChars, &policy.MaxAgeDays,
		&policy.PreventReuseCount, &policy.LockoutThreshold, &policy.LockoutDurationMinutes,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get password policy: %w", err)
	}
	
	return &policy, nil
}

// Close closes the database connection
func (udb *UserDB) Close() error {
	return udb.db.Close()
}

// UserSession represents a user session in the database
type UserSession struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	TokenHash  string    `json:"token_hash"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
}

// AuditEvent represents an audit log entry
type AuditEvent struct {
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details"`
}

// PasswordPolicy represents password policy settings
type PasswordPolicy struct {
	MinLength              int `json:"min_length"`
	RequireUppercase       bool `json:"require_uppercase"`
	RequireLowercase       bool `json:"require_lowercase"`
	RequireNumbers         bool `json:"require_numbers"`
	RequireSpecialChars    bool `json:"require_special_chars"`
	MaxAgeDays             int `json:"max_age_days"`
	PreventReuseCount      int `json:"prevent_reuse_count"`
	LockoutThreshold       int `json:"lockout_threshold"`
	LockoutDurationMinutes int `json:"lockout_duration_minutes"`
}
