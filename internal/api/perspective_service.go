package api

import (
	"context"
	"fmt"
	"time"

	apimodels "github.com/catherinevee/driftmgr/internal/api/models"
	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// PerspectiveService provides perspective-related functionality
type PerspectiveService struct {
	stateAnalyzer *state.StateAnalyzer
}

// NewPerspectiveService creates a new perspective service
func NewPerspectiveService() *PerspectiveService {
	return &PerspectiveService{
		stateAnalyzer: state.NewStateAnalyzer(),
	}
}

// GeneratePerspective generates a perspective from a state file
func (ps *PerspectiveService) GeneratePerspective(ctx context.Context, stateFile *state.State, cloudResources []models.Resource) (*apimodels.Perspective, error) {
	// Convert State to StateFile for analyzer
	stateFileData := &state.StateFile{
		ID:        fmt.Sprintf("state-%d", time.Now().Unix()),
		Path:      stateFile.Path,
		Version:   stateFile.Version,
		Resources: stateFile.Resources,
	}

	// Convert models.Resource to interface{} for analyzer
	var cloudResourcesInterface []interface{}
	for _, res := range cloudResources {
		cloudResourcesInterface = append(cloudResourcesInterface, res)
	}

	// Analyze the perspective
	statePerspective, err := ps.stateAnalyzer.AnalyzePerspective(ctx, stateFileData, cloudResourcesInterface)
	if err != nil {
		return nil, err
	}

	// Convert to API perspective
	perspective := &apimodels.Perspective{
		ID:               statePerspective.StateFileID,
		Name:             fmt.Sprintf("Perspective-%s", statePerspective.StateFileID),
		StateFilePath:    statePerspective.StateFilePath,
		ManagedResources: []models.Resource{},
		OutOfBand:        []models.Resource{},
		Coverage:         statePerspective.Statistics.CoveragePercentage,
		DriftPercentage:  statePerspective.Statistics.DriftPercentage,
		Timestamp:        statePerspective.Timestamp,
	}

	// Convert managed resources
	for _, managed := range statePerspective.ManagedResources {
		resource := models.Resource{
			ID:       managed.ID,
			Name:     managed.Name,
			Type:     managed.Type,
			Provider: managed.Provider,
			Status:   managed.Status,
		}
		perspective.ManagedResources = append(perspective.ManagedResources, resource)
	}

	// Convert out-of-band resources
	for _, oob := range statePerspective.OutOfBand {
		resource := models.Resource{
			ID:       oob.ID,
			Name:     oob.Name,
			Type:     oob.Type,
			Provider: oob.Provider,
			Region:   oob.Region,
			Tags:     oob.Tags,
		}
		perspective.OutOfBand = append(perspective.OutOfBand, resource)
	}

	return perspective, nil
}

// GetPerspective retrieves a cached perspective
func (ps *PerspectiveService) GetPerspective(id string) (*apimodels.Perspective, bool) {
	statePerspective, exists := ps.stateAnalyzer.GetPerspective(id)
	if !exists {
		return nil, false
	}

	// Convert to API perspective
	perspective := &apimodels.Perspective{
		ID:              statePerspective.StateFileID,
		Name:            fmt.Sprintf("Perspective-%s", statePerspective.StateFileID),
		StateFilePath:   statePerspective.StateFilePath,
		Coverage:        statePerspective.Statistics.CoveragePercentage,
		DriftPercentage: statePerspective.Statistics.DriftPercentage,
		Timestamp:       statePerspective.Timestamp,
	}

	return perspective, true
}

// ComparePerspectives compares two perspectives
func (ps *PerspectiveService) ComparePerspectives(p1, p2 *apimodels.Perspective) (*PerspectiveComparison, error) {
	// This would require converting back to state perspectives
	// For now, return a basic comparison
	comparison := &PerspectiveComparison{
		Perspective1ID: p1.ID,
		Perspective2ID: p2.ID,
		Timestamp:      time.Now(),
	}

	// Compare managed resources
	p1Map := make(map[string]bool)
	for _, res := range p1.ManagedResources {
		p1Map[res.ID] = true
	}

	p2Map := make(map[string]bool)
	for _, res := range p2.ManagedResources {
		p2Map[res.ID] = true
	}

	// Find differences
	for id := range p1Map {
		if !p2Map[id] {
			comparison.OnlyInFirst = append(comparison.OnlyInFirst, id)
		}
	}

	for id := range p2Map {
		if !p1Map[id] {
			comparison.OnlyInSecond = append(comparison.OnlyInSecond, id)
		}
	}

	return comparison, nil
}

// PerspectiveComparison represents a comparison between perspectives
type PerspectiveComparison struct {
	Perspective1ID string    `json:"perspective_1_id"`
	Perspective2ID string    `json:"perspective_2_id"`
	Timestamp      time.Time `json:"timestamp"`
	OnlyInFirst    []string  `json:"only_in_first"`
	OnlyInSecond   []string  `json:"only_in_second"`
}