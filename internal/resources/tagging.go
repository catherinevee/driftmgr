package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// TaggingManager manages resource tags
type TaggingManager struct {
	tags map[string][]models.ResourceTag
	mu   sync.RWMutex
}

// NewTaggingManager creates a new tagging manager
func NewTaggingManager() *TaggingManager {
	return &TaggingManager{
		tags: make(map[string][]models.ResourceTag),
	}
}

// UpdateTags updates tags for a resource
func (m *TaggingManager) UpdateTags(ctx context.Context, resourceID string, tags map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert map to slice of ResourceTag
	var resourceTags []models.ResourceTag
	for key, value := range tags {
		tag := models.ResourceTag{
			ID:         generateTagID(resourceID, key),
			ResourceID: resourceID,
			Key:        key,
			Value:      value,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		resourceTags = append(resourceTags, tag)
	}

	m.tags[resourceID] = resourceTags
	return nil
}

// GetTags retrieves tags for a resource
func (m *TaggingManager) GetTags(ctx context.Context, resourceID string) ([]models.ResourceTag, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tags := m.tags[resourceID]
	if tags == nil {
		return []models.ResourceTag{}, nil
	}

	return tags, nil
}

// DeleteTags deletes tags for a resource
func (m *TaggingManager) DeleteTags(ctx context.Context, resourceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tags, resourceID)
	return nil
}

// GetResourcesByTag retrieves resources that have a specific tag
func (m *TaggingManager) GetResourcesByTag(ctx context.Context, key, value string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var resourceIDs []string
	for resourceID, tags := range m.tags {
		for _, tag := range tags {
			if tag.Key == key && tag.Value == value {
				resourceIDs = append(resourceIDs, resourceID)
				break
			}
		}
	}

	return resourceIDs, nil
}

// GetTagStatistics returns tag statistics
func (m *TaggingManager) GetTagStatistics(ctx context.Context) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]interface{}{
		"total_resources_with_tags": len(m.tags),
		"total_tags":                0,
		"unique_keys":               make(map[string]int),
		"most_common_tags":          make(map[string]int),
	}

	keyCounts := make(map[string]int)
	tagCounts := make(map[string]int)

	for _, tags := range m.tags {
		stats["total_tags"] = stats["total_tags"].(int) + len(tags)
		for _, tag := range tags {
			keyCounts[tag.Key]++
			tagCounts[fmt.Sprintf("%s:%s", tag.Key, tag.Value)]++
		}
	}

	stats["unique_keys"] = keyCounts
	stats["most_common_tags"] = tagCounts

	return stats
}

// Health checks the health of the tagging manager
func (m *TaggingManager) Health(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Basic health check
	if len(m.tags) < 0 {
		return fmt.Errorf("tagging manager has negative count")
	}

	return nil
}

// generateTagID generates a tag ID
func generateTagID(resourceID, key string) string {
	return fmt.Sprintf("tag-%s-%s", resourceID, key)
}
