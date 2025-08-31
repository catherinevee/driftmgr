package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

// Config represents database configuration
type Config struct {
	Path        string
	MaxRetries  int
	RetryDelay  time.Duration
}

// DefaultConfig returns default database configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		Path:       filepath.Join(homeDir, ".driftmgr", "driftmgr.db"),
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// New creates a new database connection
func New(config *Config) (*DB, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure directory exists
	dir := filepath.Dir(config.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	conn, err := sql.Open("sqlite3", config.Path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	db := &DB{conn: conn}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// initSchema creates the database schema
func (db *DB) initSchema() error {
	schema := `
	-- Resources table
	CREATE TABLE IF NOT EXISTS resources (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		provider TEXT NOT NULL,
		region TEXT,
		status TEXT,
		tags TEXT,
		metadata TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		discovered_at TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_resources_provider ON resources(provider);
	CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(type);
	CREATE INDEX IF NOT EXISTS idx_resources_region ON resources(region);
	CREATE INDEX IF NOT EXISTS idx_resources_status ON resources(status);

	-- Drifts table
	CREATE TABLE IF NOT EXISTS drifts (
		id TEXT PRIMARY KEY,
		resource_id TEXT NOT NULL,
		drift_type TEXT NOT NULL,
		severity TEXT,
		state_value TEXT,
		actual_value TEXT,
		details TEXT,
		detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		resolved_at TIMESTAMP,
		resolved_by TEXT,
		FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_drifts_resource ON drifts(resource_id);
	CREATE INDEX IF NOT EXISTS idx_drifts_severity ON drifts(severity);
	CREATE INDEX IF NOT EXISTS idx_drifts_detected ON drifts(detected_at);

	-- Remediation Jobs table
	CREATE TABLE IF NOT EXISTS remediation_jobs (
		id TEXT PRIMARY KEY,
		resource_id TEXT NOT NULL,
		resource_type TEXT,
		provider TEXT,
		region TEXT,
		status TEXT NOT NULL,
		action TEXT,
		drift_type TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		started_at TIMESTAMP,
		completed_at TIMESTAMP,
		error TEXT,
		details TEXT,
		remediated_by TEXT,
		FOREIGN KEY (resource_id) REFERENCES resources(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_remediation_status ON remediation_jobs(status);
	CREATE INDEX IF NOT EXISTS idx_remediation_created ON remediation_jobs(created_at);

	-- Discovery Jobs table
	CREATE TABLE IF NOT EXISTS discovery_jobs (
		id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		providers TEXT,
		regions TEXT,
		progress INTEGER DEFAULT 0,
		message TEXT,
		started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		error TEXT,
		resources_found INTEGER DEFAULT 0,
		summary TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_discovery_status ON discovery_jobs(status);
	CREATE INDEX IF NOT EXISTS idx_discovery_started ON discovery_jobs(started_at);

	-- State Files table
	CREATE TABLE IF NOT EXISTS state_files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		path TEXT UNIQUE NOT NULL,
		name TEXT,
		workspace TEXT DEFAULT 'default',
		backend TEXT DEFAULT 'local',
		provider TEXT,
		size INTEGER,
		resource_count INTEGER DEFAULT 0,
		version INTEGER,
		serial INTEGER,
		last_modified TIMESTAMP,
		status TEXT,
		error TEXT,
		resources TEXT,
		outputs TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_state_files_path ON state_files(path);
	CREATE INDEX IF NOT EXISTS idx_state_files_provider ON state_files(provider);

	-- Audit Logs table
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		event_type TEXT NOT NULL,
		severity TEXT,
		user TEXT,
		resource_id TEXT,
		action TEXT,
		details TEXT,
		ip_address TEXT,
		user_agent TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user);
	CREATE INDEX IF NOT EXISTS idx_audit_event ON audit_logs(event_type);

	-- Credentials table (encrypted)
	CREATE TABLE IF NOT EXISTS credentials (
		id TEXT PRIMARY KEY,
		provider TEXT NOT NULL UNIQUE,
		encrypted_data TEXT NOT NULL,
		status TEXT,
		last_validated TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Cache table
	CREATE TABLE IF NOT EXISTS cache (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		expires_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_cache_expires ON cache(expires_at);

	-- Resource Relationships table
	CREATE TABLE IF NOT EXISTS resource_relationships (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		source_id TEXT NOT NULL,
		target_id TEXT NOT NULL,
		relationship_type TEXT NOT NULL,
		metadata TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (source_id) REFERENCES resources(id) ON DELETE CASCADE,
		FOREIGN KEY (target_id) REFERENCES resources(id) ON DELETE CASCADE,
		UNIQUE(source_id, target_id, relationship_type)
	);
	CREATE INDEX IF NOT EXISTS idx_relationships_source ON resource_relationships(source_id);
	CREATE INDEX IF NOT EXISTS idx_relationships_target ON resource_relationships(target_id);

	-- Notifications table
	CREATE TABLE IF NOT EXISTS notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		severity TEXT,
		title TEXT NOT NULL,
		message TEXT,
		metadata TEXT,
		channel TEXT,
		status TEXT DEFAULT 'pending',
		sent_at TIMESTAMP,
		error TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
	CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications(created_at);

	-- Configuration table
	CREATE TABLE IF NOT EXISTS configuration (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		description TEXT,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_by TEXT
	);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Begin starts a new transaction
func (db *DB) Begin() (*sql.Tx, error) {
	return db.conn.Begin()
}

// Resource Operations

// SaveResource saves a resource to the database
func (db *DB) SaveResource(resource map[string]interface{}) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	tagsJSON, _ := json.Marshal(resource["tags"])
	metadataJSON, _ := json.Marshal(resource["metadata"])

	query := `
		INSERT OR REPLACE INTO resources 
		(id, name, type, provider, region, status, tags, metadata, discovered_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err := db.conn.Exec(query,
		resource["id"],
		resource["name"],
		resource["type"],
		resource["provider"],
		resource["region"],
		resource["status"],
		string(tagsJSON),
		string(metadataJSON),
		time.Now(),
	)

	return err
}

// GetResource retrieves a resource by ID
func (db *DB) GetResource(id string) (map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `
		SELECT id, name, type, provider, region, status, tags, metadata, 
		       created_at, updated_at, discovered_at
		FROM resources WHERE id = ?
	`

	resource := make(map[string]interface{})
	var tagsJSON, metadataJSON string
	var createdAt, updatedAt, discoveredAt sql.NullTime
	var id_, name, type_, provider, region, status string

	row := db.conn.QueryRow(query, id)
	err := row.Scan(
		&id_,
		&name,
		&type_,
		&provider,
		&region,
		&status,
		&tagsJSON,
		&metadataJSON,
		&createdAt,
		&updatedAt,
		&discoveredAt,
	)

	if err != nil {
		return nil, err
	}

	// Assign values to map
	resource["id"] = id_
	resource["name"] = name
	resource["type"] = type_
	resource["provider"] = provider
	resource["region"] = region
	resource["status"] = status

	// Parse JSON fields
	var tags, metadata interface{}
	json.Unmarshal([]byte(tagsJSON), &tags)
	json.Unmarshal([]byte(metadataJSON), &metadata)
	resource["tags"] = tags
	resource["metadata"] = metadata

	if createdAt.Valid {
		resource["created_at"] = createdAt.Time
	}
	if updatedAt.Valid {
		resource["updated_at"] = updatedAt.Time
	}
	if discoveredAt.Valid {
		resource["discovered_at"] = discoveredAt.Time
	}

	return resource, nil
}

// ListResources lists resources with optional filters
func (db *DB) ListResources(filters map[string]interface{}) ([]map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	query := `
		SELECT id, name, type, provider, region, status, tags, metadata,
		       created_at, updated_at, discovered_at
		FROM resources WHERE 1=1
	`

	args := []interface{}{}

	if provider, ok := filters["provider"].(string); ok && provider != "" {
		query += " AND provider = ?"
		args = append(args, provider)
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		query += " AND region = ?"
		args = append(args, region)
	}

	if resourceType, ok := filters["type"].(string); ok && resourceType != "" {
		query += " AND type = ?"
		args = append(args, resourceType)
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY updated_at DESC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	resources := []map[string]interface{}{}
	for rows.Next() {
		resource := make(map[string]interface{})
		var tagsJSON, metadataJSON string
		var createdAt, updatedAt, discoveredAt sql.NullTime
		var id, name, type_, provider, region, status string

		err := rows.Scan(
			&id,
			&name,
			&type_,
			&provider,
			&region,
			&status,
			&tagsJSON,
			&metadataJSON,
			&createdAt,
			&updatedAt,
			&discoveredAt,
		)

		if err != nil {
			continue
		}

		// Assign values to map
		resource["id"] = id
		resource["name"] = name
		resource["type"] = type_
		resource["provider"] = provider
		resource["region"] = region
		resource["status"] = status

		// Parse JSON fields
		var tags, metadata interface{}
		json.Unmarshal([]byte(tagsJSON), &tags)
		json.Unmarshal([]byte(metadataJSON), &metadata)
		resource["tags"] = tags
		resource["metadata"] = metadata

		if createdAt.Valid {
			resource["created_at"] = createdAt.Time
		}
		if updatedAt.Valid {
			resource["updated_at"] = updatedAt.Time
		}
		if discoveredAt.Valid {
			resource["discovered_at"] = discoveredAt.Time
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// DeleteResource deletes a resource by ID
func (db *DB) DeleteResource(id string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, err := db.conn.Exec("DELETE FROM resources WHERE id = ?", id)
	return err
}

// Drift Operations

// SaveDrift saves a drift record
func (db *DB) SaveDrift(drift map[string]interface{}) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	detailsJSON, _ := json.Marshal(drift["details"])

	query := `
		INSERT INTO drifts 
		(id, resource_id, drift_type, severity, state_value, actual_value, details, detected_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.Exec(query,
		drift["id"],
		drift["resource_id"],
		drift["drift_type"],
		drift["severity"],
		drift["state_value"],
		drift["actual_value"],
		string(detailsJSON),
		time.Now(),
	)

	return err
}

// ResolveDrift marks a drift as resolved
func (db *DB) ResolveDrift(id string, resolvedBy string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	query := `
		UPDATE drifts 
		SET resolved_at = ?, resolved_by = ?
		WHERE id = ?
	`

	_, err := db.conn.Exec(query, time.Now(), resolvedBy, id)
	return err
}

// Cleanup Operations

// CleanupOldData removes old data based on retention policy
func (db *DB) CleanupOldData(retentionDays int) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	// Cleanup old audit logs
	_, err := db.conn.Exec("DELETE FROM audit_logs WHERE timestamp < ?", cutoff)
	if err != nil {
		return err
	}

	// Cleanup old completed jobs
	_, err = db.conn.Exec(`
		DELETE FROM discovery_jobs 
		WHERE completed_at < ? AND status IN ('completed', 'failed')
	`, cutoff)
	if err != nil {
		return err
	}

	_, err = db.conn.Exec(`
		DELETE FROM remediation_jobs 
		WHERE completed_at < ? AND status IN ('completed', 'failed')
	`, cutoff)
	if err != nil {
		return err
	}

	// Cleanup expired cache
	_, err = db.conn.Exec("DELETE FROM cache WHERE expires_at < ?", time.Now())

	return err
}

// GetStats returns database statistics
func (db *DB) GetStats() (map[string]interface{}, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	stats := make(map[string]interface{})

	// Count resources
	var resourceCount int
	db.conn.QueryRow("SELECT COUNT(*) FROM resources").Scan(&resourceCount)
	stats["resources"] = resourceCount

	// Count drifts
	var driftCount, unresolvedDriftCount int
	db.conn.QueryRow("SELECT COUNT(*) FROM drifts").Scan(&driftCount)
	db.conn.QueryRow("SELECT COUNT(*) FROM drifts WHERE resolved_at IS NULL").Scan(&unresolvedDriftCount)
	stats["drifts"] = driftCount
	stats["unresolved_drifts"] = unresolvedDriftCount

	// Count jobs
	var discoveryJobs, remediationJobs int
	db.conn.QueryRow("SELECT COUNT(*) FROM discovery_jobs").Scan(&discoveryJobs)
	db.conn.QueryRow("SELECT COUNT(*) FROM remediation_jobs").Scan(&remediationJobs)
	stats["discovery_jobs"] = discoveryJobs
	stats["remediation_jobs"] = remediationJobs

	return stats, nil
}

// Query executes a query and returns rows
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.conn.Query(query, args...)
}

// QueryRow executes a query that returns a single row
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.conn.QueryRow(query, args...)
}

// Exec executes a statement
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.conn.Exec(query, args...)
}