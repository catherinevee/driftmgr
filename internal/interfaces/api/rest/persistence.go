package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apimodels "github.com/catherinevee/driftmgr/internal/api/models"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/database"
	"github.com/google/uuid"
)

// PersistenceManager manages data persistence
type PersistenceManager struct {
	db *database.DB
}

// NewPersistenceManager creates a new persistence manager
func NewPersistenceManager(dbPath string) (*PersistenceManager, error) {
	config := &database.Config{
		Path:       dbPath,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}

	if dbPath == "" {
		config = database.DefaultConfig()
	}

	db, err := database.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &PersistenceManager{db: db}, nil
}

// Close closes the database connection
func (pm *PersistenceManager) Close() error {
	if pm.db != nil {
		return pm.db.Close()
	}
	return nil
}

// SaveDiscoveryResults persists discovery results to database
func (pm *PersistenceManager) SaveDiscoveryResults(jobID string, resources []apimodels.Resource) error {
	tx, err := pm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Save each resource
	for _, resource := range resources {
		resourceMap := map[string]interface{}{
			"id":       resource.ID,
			"name":     resource.Name,
			"type":     resource.Type,
			"provider": resource.Provider,
			"region":   resource.Region,
			"status":   resource.Status,
			"tags":     resource.Tags,
			"metadata": map[string]interface{}{
				"created_at": resource.CreatedAt,
				"managed":    resource.Managed,
				"account_id": resource.Account,
			},
		}

		if err := pm.db.SaveResource(resourceMap); err != nil {
			return err
		}
	}

	// Update discovery job
	if jobID != "" {
		_, err = tx.Exec(`
			UPDATE discovery_jobs 
			SET resources_found = ?, status = 'completed', completed_at = ?
			WHERE id = ?
		`, len(resources), time.Now(), jobID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveDrift persists a drift detection result
func (pm *PersistenceManager) SaveDrift(drift *models.DriftItem) error {
	driftID := fmt.Sprintf("drift-%s", uuid.New().String()[:8])
	
	driftMap := map[string]interface{}{
		"id":           driftID,
		"resource_id":  drift.ResourceID,
		"drift_type":   drift.DriftType,
		"severity":     drift.Severity,
		"before":       drift.Before,
		"after":        drift.After,
		"details":      drift.Details,
	}

	return pm.db.SaveDrift(driftMap)
}

// SaveRemediationJob persists a remediation job
func (pm *PersistenceManager) SaveRemediationJob(job *RemediationJob) error {
	tx, err := pm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	detailsJSON, _ := json.Marshal(job.Details)

	_, err = tx.Exec(`
		INSERT OR REPLACE INTO remediation_jobs 
		(id, resource_id, resource_type, provider, region, status, action, 
		 drift_type, created_at, started_at, completed_at, error, details, remediated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.ID, job.ResourceID, job.ResourceType, job.Provider, job.Region,
		job.Status, job.Action, job.DriftType, job.CreatedAt, job.StartedAt,
		job.CompletedAt, job.Error, string(detailsJSON), job.RemediatedBy)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// LoadRemediationJobs loads remediation jobs from database
func (pm *PersistenceManager) LoadRemediationJobs(limit int) ([]*RemediationJob, error) {
	query := `
		SELECT id, resource_id, resource_type, provider, region, status, action,
		       drift_type, created_at, started_at, completed_at, error, details
		FROM remediation_jobs
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := pm.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*RemediationJob
	for rows.Next() {
		job := &RemediationJob{}
		var detailsJSON string
		var startedAt, completedAt *time.Time

		err := rows.Scan(
			&job.ID, &job.ResourceID, &job.ResourceType, &job.Provider,
			&job.Region, &job.Status, &job.Action, &job.DriftType,
			&job.CreatedAt, &startedAt, &completedAt, &job.Error, &detailsJSON,
		)

		if err != nil {
			continue
		}

		job.StartedAt = startedAt
		job.CompletedAt = completedAt
		json.Unmarshal([]byte(detailsJSON), &job.Details)

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// SaveStateFile persists state file information
func (pm *PersistenceManager) SaveStateFile(stateFile *StateFileInfo) error {
	resourcesJSON, _ := json.Marshal(stateFile.Resources)
	outputsJSON, _ := json.Marshal(stateFile.Outputs)

	_, err := pm.db.Exec(`
		INSERT OR REPLACE INTO state_files 
		(path, name, workspace, backend, provider, size, resource_count,
		 version, serial, last_modified, status, error, resources, outputs)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, stateFile.Path, stateFile.Path, "default", stateFile.Backend,
		stateFile.Provider, stateFile.Size, stateFile.ResourceCount,
		stateFile.Version, stateFile.Serial, stateFile.LastModified,
		"discovered", stateFile.Error, string(resourcesJSON), string(outputsJSON))

	return err
}

// LoadStateFiles loads state files from database
func (pm *PersistenceManager) LoadStateFiles() ([]*StateFileInfo, error) {
	query := `
		SELECT path, backend, provider, size, resource_count, version, serial,
		       last_modified, status, error, resources, outputs
		FROM state_files
		ORDER BY last_modified DESC
	`

	rows, err := pm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stateFiles []*StateFileInfo
	for rows.Next() {
		sf := &StateFileInfo{}
		var resourcesJSON, outputsJSON string

		var status string // Temporary variable for status column if it exists
		err := rows.Scan(
			&sf.Path, &sf.Backend, &sf.Provider, &sf.Size,
			&sf.ResourceCount, &sf.Version, &sf.Serial,
			&sf.LastModified, &status, &sf.Error,
			&resourcesJSON, &outputsJSON,
		)

		if err != nil {
			continue
		}

		sf.FullPath = sf.Path
		json.Unmarshal([]byte(resourcesJSON), &sf.Resources)
		json.Unmarshal([]byte(outputsJSON), &sf.Outputs)

		stateFiles = append(stateFiles, sf)
	}

	return stateFiles, nil
}

// SaveAuditLog persists an audit log entry
func (pm *PersistenceManager) SaveAuditLog(ctx context.Context, eventType, user string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)

	_, err := pm.db.Exec(`
		INSERT INTO audit_logs (event_type, user, action, details)
		VALUES (?, ?, ?, ?)
	`, eventType, user, details["action"], string(detailsJSON))

	return err
}

// GetAuditLogs retrieves audit logs
func (pm *PersistenceManager) GetAuditLogs(limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT timestamp, event_type, severity, user, action, details
		FROM audit_logs
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := pm.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		log := make(map[string]interface{})
		var detailsJSON string
		var timestamp time.Time
		var eventType, severity, user, action string

		err := rows.Scan(
			&timestamp, &eventType, &severity,
			&user, &action, &detailsJSON,
		)

		if err != nil {
			continue
		}

		// Assign values to map
		log["timestamp"] = timestamp
		log["event_type"] = eventType
		log["severity"] = severity
		log["user"] = user
		log["action"] = action

		// Parse details JSON
		if detailsJSON != "" {
			var details map[string]interface{}
			json.Unmarshal([]byte(detailsJSON), &details)
			log["details"] = details
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// SaveConfiguration saves a configuration value
func (pm *PersistenceManager) SaveConfiguration(key, value, updatedBy string) error {
	_, err := pm.db.Exec(`
		INSERT OR REPLACE INTO configuration (key, value, updated_by, updated_at)
		VALUES (?, ?, ?, ?)
	`, key, value, updatedBy, time.Now())

	return err
}

// GetConfiguration retrieves a configuration value
func (pm *PersistenceManager) GetConfiguration(key string) (string, error) {
	var value string
	err := pm.db.QueryRow("SELECT value FROM configuration WHERE key = ?", key).Scan(&value)
	return value, err
}

// GetAllConfiguration retrieves all configuration values
func (pm *PersistenceManager) GetAllConfiguration() (map[string]string, error) {
	query := "SELECT key, value FROM configuration"
	rows, err := pm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	config := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		config[key] = value
	}

	return config, nil
}

// SaveResourceRelationship saves a relationship between resources
func (pm *PersistenceManager) SaveResourceRelationship(sourceID, targetID, relationshipType string, metadata map[string]interface{}) error {
	metadataJSON, _ := json.Marshal(metadata)

	_, err := pm.db.Exec(`
		INSERT OR REPLACE INTO resource_relationships 
		(source_id, target_id, relationship_type, metadata)
		VALUES (?, ?, ?, ?)
	`, sourceID, targetID, relationshipType, string(metadataJSON))

	return err
}

// GetResourceRelationships retrieves relationships for a resource
func (pm *PersistenceManager) GetResourceRelationships(resourceID string) ([]map[string]interface{}, error) {
	query := `
		SELECT source_id, target_id, relationship_type, metadata
		FROM resource_relationships
		WHERE source_id = ? OR target_id = ?
	`

	rows, err := pm.db.Query(query, resourceID, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []map[string]interface{}
	for rows.Next() {
		rel := make(map[string]interface{})
		var metadataJSON string
		var sourceID, targetID, relationshipType string

		err := rows.Scan(
			&sourceID, &targetID,
			&relationshipType, &metadataJSON,
		)

		if err != nil {
			continue
		}

		// Assign values to map
		rel["source_id"] = sourceID
		rel["target_id"] = targetID
		rel["relationship_type"] = relationshipType
		
		// Parse metadata JSON
		if metadataJSON != "" {
			var metadata map[string]interface{}
			json.Unmarshal([]byte(metadataJSON), &metadata)
			rel["metadata"] = metadata
		}
		relationships = append(relationships, rel)
	}

	return relationships, nil
}

// GetRelationships returns relationships for a resource
func (pm *PersistenceManager) GetRelationships(resourceID string) ([]map[string]interface{}, error) {
	query := `
		SELECT source_id, target_id, relationship_type, metadata
		FROM relationships
		WHERE source_id = ? OR target_id = ?
		ORDER BY created_at DESC
	`
	
	rows, err := pm.db.Query(query, resourceID, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var relationships []map[string]interface{}
	for rows.Next() {
		rel := make(map[string]interface{})
		var metadataJSON string
		var sourceID, targetID, relationshipType string
		
		err := rows.Scan(
			&sourceID, &targetID,
			&relationshipType, &metadataJSON,
		)
		
		if err != nil {
			continue
		}
		
		// Assign values to map
		rel["source_id"] = sourceID
		rel["target_id"] = targetID
		rel["relationship_type"] = relationshipType
		
		// Parse metadata JSON
		if metadataJSON != "" {
			var metadata map[string]interface{}
			json.Unmarshal([]byte(metadataJSON), &metadata)
			rel["metadata"] = metadata
		}
		relationships = append(relationships, rel)
	}
	
	return relationships, nil
}

// Cache Operations

// SetCache stores a value in cache
func (pm *PersistenceManager) SetCache(key string, value interface{}, ttl time.Duration) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(ttl)

	_, err = pm.db.Exec(`
		INSERT OR REPLACE INTO cache (key, value, expires_at)
		VALUES (?, ?, ?)
	`, key, string(valueJSON), expiresAt)

	return err
}

// GetCache retrieves a value from cache
func (pm *PersistenceManager) GetCache(key string) (interface{}, error) {
	var valueJSON string
	var expiresAt time.Time

	err := pm.db.QueryRow(`
		SELECT value, expires_at FROM cache WHERE key = ?
	`, key).Scan(&valueJSON, &expiresAt)

	if err != nil {
		return nil, err
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		// Delete expired entry
		pm.db.Exec("DELETE FROM cache WHERE key = ?", key)
		return nil, fmt.Errorf("cache entry expired")
	}

	var value interface{}
	err = json.Unmarshal([]byte(valueJSON), &value)
	return value, err
}

// ClearCache clears all cache entries
func (pm *PersistenceManager) ClearCache() error {
	_, err := pm.db.Exec("DELETE FROM cache")
	return err
}

// Cleanup runs cleanup operations
func (pm *PersistenceManager) Cleanup(retentionDays int) error {
	return pm.db.CleanupOldData(retentionDays)
}

// GetStats returns database statistics
func (pm *PersistenceManager) GetStats() (map[string]interface{}, error) {
	return pm.db.GetStats()
}

// Query and Exec are helper methods for direct database access

// Query executes a query and returns rows
func (pm *PersistenceManager) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := pm.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Prepare scan destinations
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Scan all rows
	var results []map[string]interface{}
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		result := make(map[string]interface{})
		for i, col := range columns {
			result[col] = values[i]
		}
		results = append(results, result)
	}

	return results, nil
}

// Exec executes a statement
func (pm *PersistenceManager) Exec(query string, args ...interface{}) error {
	_, err := pm.db.Exec(query, args...)
	return err
}