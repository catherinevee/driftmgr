package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// MetadataManager manages resource metadata
type MetadataManager struct {
	metadata map[string]*models.ResourceMetadata
	mu       sync.RWMutex
}

// NewMetadataManager creates a new metadata manager
func NewMetadataManager() *MetadataManager {
	return &MetadataManager{
		metadata: make(map[string]*models.ResourceMetadata),
	}
}

// Update updates metadata for a resource
func (m *MetadataManager) Update(ctx context.Context, resource *models.CloudResource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create metadata
	meta := m.metadata[resource.ID]
	if meta == nil {
		meta = &models.ResourceMetadata{
			ID:         generateMetadataID(resource.ID),
			ResourceID: resource.ID,
			CreatedAt:  time.Now(),
		}
	}

	// Update metadata based on resource type
	m.updateMetadataFromResource(meta, resource)
	meta.UpdatedAt = time.Now()

	m.metadata[resource.ID] = meta
	return nil
}

// Enrich enriches a resource with metadata
func (m *MetadataManager) Enrich(ctx context.Context, resource *models.CloudResource) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	meta := m.metadata[resource.ID]
	if meta == nil {
		return nil // No metadata available
	}

	// Create a copy of the metadata to avoid race conditions
	resourceMetadata := *meta
	resource.Metadata = map[string]interface{}{
		"category":           resourceMetadata.Category,
		"status":             resourceMetadata.Status,
		"size":               resourceMetadata.Size,
		"instance_type":      resourceMetadata.InstanceType,
		"operating_system":   resourceMetadata.OperatingSystem,
		"architecture":       resourceMetadata.Architecture,
		"public_ip":          resourceMetadata.PublicIP,
		"private_ip":         resourceMetadata.PrivateIP,
		"ports":              resourceMetadata.Ports,
		"protocols":          resourceMetadata.Protocols,
		"endpoints":          resourceMetadata.Endpoints,
		"version":            resourceMetadata.Version,
		"runtime":            resourceMetadata.Runtime,
		"framework":          resourceMetadata.Framework,
		"database":           resourceMetadata.Database,
		"engine":             resourceMetadata.Engine,
		"storage_type":       resourceMetadata.StorageType,
		"storage_size":       resourceMetadata.StorageSize,
		"encryption":         resourceMetadata.Encryption,
		"backup_enabled":     resourceMetadata.BackupEnabled,
		"monitoring_enabled": resourceMetadata.MonitoringEnabled,
		"logging_enabled":    resourceMetadata.LoggingEnabled,
		"custom_fields":      resourceMetadata.CustomFields,
	}

	return nil
}

// Delete deletes metadata for a resource
func (m *MetadataManager) Delete(ctx context.Context, resourceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.metadata, resourceID)
	return nil
}

// Get retrieves metadata for a resource
func (m *MetadataManager) Get(ctx context.Context, resourceID string) (*models.ResourceMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	meta := m.metadata[resourceID]
	if meta == nil {
		return nil, fmt.Errorf("metadata for resource %s not found", resourceID)
	}

	return meta, nil
}

// GetStatistics returns metadata statistics
func (m *MetadataManager) GetStatistics(ctx context.Context) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"total_metadata":  len(m.metadata),
		"by_category":     make(map[string]int),
		"by_status":       make(map[string]int),
		"by_architecture": make(map[string]int),
		"by_os":           make(map[string]int),
	}

	// Count by category
	for _, meta := range m.metadata {
		stats["by_category"].(map[string]int)[meta.Category.String()]++
		stats["by_status"].(map[string]int)[meta.Status.String()]++
		if meta.Architecture != "" {
			stats["by_architecture"].(map[string]int)[meta.Architecture]++
		}
		if meta.OperatingSystem != "" {
			stats["by_os"].(map[string]int)[meta.OperatingSystem]++
		}
	}

	return stats
}

// Health checks the health of the metadata manager
func (m *MetadataManager) Health(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Basic health check
	if len(m.metadata) < 0 {
		return fmt.Errorf("metadata manager has negative count")
	}

	return nil
}

// updateMetadataFromResource updates metadata based on resource information
func (m *MetadataManager) updateMetadataFromResource(meta *models.ResourceMetadata, resource *models.CloudResource) {
	// Set category based on resource type
	meta.Category = models.GetCategoryForResourceType(models.ResourceType(resource.Type))

	// Extract metadata from resource configuration
	if config := resource.Configuration; config != nil {
		// Extract instance type
		if instanceType, ok := config["instance_type"].(string); ok {
			meta.InstanceType = instanceType
		}

		// Extract architecture
		if architecture, ok := config["architecture"].(string); ok {
			meta.Architecture = architecture
		}

		// Extract operating system
		if platform, ok := config["platform"].(string); ok {
			meta.OperatingSystem = platform
		}

		// Extract runtime
		if runtime, ok := config["runtime"].(string); ok {
			meta.Runtime = runtime
		}

		// Extract engine
		if engine, ok := config["engine"].(string); ok {
			meta.Engine = engine
		}

		// Extract storage type
		if storageType, ok := config["storage_type"].(string); ok {
			meta.StorageType = storageType
		}

		// Extract encryption status
		if encrypted, ok := config["encrypted"].(bool); ok {
			meta.Encryption = encrypted
		}

		// Extract backup status
		if backupEnabled, ok := config["backup_enabled"].(bool); ok {
			meta.BackupEnabled = backupEnabled
		}

		// Extract monitoring status
		if monitoringEnabled, ok := config["monitoring_enabled"].(bool); ok {
			meta.MonitoringEnabled = monitoringEnabled
		}

		// Extract logging status
		if loggingEnabled, ok := config["logging_enabled"].(bool); ok {
			meta.LoggingEnabled = loggingEnabled
		}
	}

	// Extract metadata from resource metadata
	if resourceMetadata := resource.Metadata; resourceMetadata != nil {
		// Extract public IP
		if publicIP, ok := resourceMetadata["public_ip"].(string); ok {
			meta.PublicIP = publicIP
		}

		// Extract private IP
		if privateIP, ok := resourceMetadata["private_ip"].(string); ok {
			meta.PrivateIP = privateIP
		}

		// Extract size
		if size, ok := resourceMetadata["size"].(string); ok {
			meta.Size = size
		}

		// Extract version
		if version, ok := resourceMetadata["version"].(string); ok {
			meta.Version = version
		}

		// Extract framework
		if framework, ok := resourceMetadata["framework"].(string); ok {
			meta.Framework = framework
		}

		// Extract database
		if database, ok := resourceMetadata["database"].(string); ok {
			meta.Database = database
		}

		// Extract storage size
		if storageSize, ok := resourceMetadata["storage_size"].(string); ok {
			meta.StorageSize = storageSize
		}

		// Extract ports
		if ports, ok := resourceMetadata["ports"].([]int); ok {
			meta.Ports = ports
		}

		// Extract protocols
		if protocols, ok := resourceMetadata["protocols"].([]string); ok {
			meta.Protocols = protocols
		}

		// Extract endpoints
		if endpoints, ok := resourceMetadata["endpoints"].([]string); ok {
			meta.Endpoints = endpoints
		}

		// Extract custom fields
		if customFields, ok := resourceMetadata["custom_fields"].(map[string]interface{}); ok {
			meta.CustomFields = customFields
		}
	}

	// Set default status based on resource type
	if meta.Status == "" {
		meta.Status = models.ResourceStatusUnknown
	}
}

// generateMetadataID generates a metadata ID
func generateMetadataID(resourceID string) string {
	return fmt.Sprintf("meta-%s", resourceID)
}
