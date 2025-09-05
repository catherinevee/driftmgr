package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/state"
)

// DriftSimulator creates controlled drift for testing
type DriftSimulator struct {
	stateFile    string
	provider     string
	driftType    DriftType
	targetResource string
	rollbackData  *RollbackData
	awsSim       *AWSSimulator
	azureSim     *AzureSimulator
	gcpSim       *GCPSimulator
}

// DriftType represents the type of drift to simulate
type DriftType string

const (
	DriftTypeTagChange        DriftType = "tag-change"
	DriftTypeRuleAddition     DriftType = "rule-addition"
	DriftTypeResourceCreation DriftType = "resource-creation"
	DriftTypeAttributeChange  DriftType = "attribute-change"
	DriftTypeResourceDeletion DriftType = "resource-deletion"
	DriftTypeRandom           DriftType = "random"
)

// SimulationResult contains the result of a drift simulation
type SimulationResult struct {
	Success        bool                   `json:"success"`
	DriftType      DriftType              `json:"drift_type"`
	Provider       string                 `json:"provider"`
	ResourceType   string                 `json:"resource_type"`
	ResourceID     string                 `json:"resource_id"`
	Changes        map[string]interface{} `json:"changes"`
	RollbackData   *RollbackData          `json:"rollback_data,omitempty"`
	ErrorMessage   string                 `json:"error,omitempty"`
	CostEstimate   string                 `json:"cost_estimate"`
	DetectedDrift  []DriftItem            `json:"detected_drift,omitempty"`
}

// DriftItem represents a detected drift
type DriftItem struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	DriftType    string                 `json:"drift_type"`
	Before       map[string]interface{} `json:"before"`
	After        map[string]interface{} `json:"after"`
	Impact       string                 `json:"impact"`
}

// RollbackData stores information needed to undo the drift
type RollbackData struct {
	Provider     string                 `json:"provider"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Action       string                 `json:"action"`
	OriginalData map[string]interface{} `json:"original_data"`
	Timestamp    time.Time              `json:"timestamp"`
}

// SimulatorConfig configures the drift simulator
type SimulatorConfig struct {
	StateFile      string
	Provider       string
	DriftType      DriftType
	TargetResource string // Optional: specific resource to target
	AutoRollback   bool
	DryRun         bool
}

// NewDriftSimulator creates a new drift simulator
func NewDriftSimulator(config SimulatorConfig) (*DriftSimulator, error) {
	sim := &DriftSimulator{
		stateFile:    config.StateFile,
		provider:     config.Provider,
		driftType:    config.DriftType,
		targetResource: config.TargetResource,
	}

	// Initialize provider-specific simulators
	switch config.Provider {
	case "aws":
		sim.awsSim = NewAWSSimulator()
	case "azure":
		sim.azureSim = NewAzureSimulator()
	case "gcp":
		sim.gcpSim = NewGCPSimulator()
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	// If drift type is random, select one
	if config.DriftType == DriftTypeRandom {
		sim.driftType = sim.selectRandomDriftType()
	}

	return sim, nil
}

// SimulateDrift creates drift in the cloud provider
func (s *DriftSimulator) SimulateDrift(ctx context.Context) (*SimulationResult, error) {
	// Parse state file to understand managed resources
	state, err := s.parseStateFile()
	if err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Select a target resource if not specified
	if s.targetResource == "" {
		resource := s.selectTargetResource(state)
		if resource == nil {
			return nil, fmt.Errorf("no suitable resources found in state file")
		}
		s.targetResource = resource.ID
	}

	// Execute the drift simulation based on provider
	var result *SimulationResult
	switch s.provider {
	case "aws":
		result, err = s.awsSim.SimulateDrift(ctx, s.driftType, s.targetResource, state)
	case "azure":
		result, err = s.azureSim.SimulateDrift(ctx, s.driftType, s.targetResource, state)
	case "gcp":
		result, err = s.gcpSim.SimulateDrift(ctx, s.driftType, s.targetResource, state)
	default:
		return nil, fmt.Errorf("provider %s not implemented", s.provider)
	}

	if err != nil {
		return nil, err
	}

	// Store rollback data
	s.rollbackData = result.RollbackData

	return result, nil
}

// DetectDrift runs drift detection after simulation
func (s *DriftSimulator) DetectDrift(ctx context.Context) ([]DriftItem, error) {
	// This would integrate with the existing drift detection engine
	// For now, we'll implement a simple detection

	state, err := s.parseStateFile()
	if err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	var drifts []DriftItem

	// Get current resource state from provider
	switch s.provider {
	case "aws":
		drifts, err = s.awsSim.DetectDrift(ctx, state)
	case "azure":
		drifts, err = s.azureSim.DetectDrift(ctx, state)
	case "gcp":
		drifts, err = s.gcpSim.DetectDrift(ctx, state)
	}

	return drifts, err
}

// Rollback undoes the simulated drift
func (s *DriftSimulator) Rollback(ctx context.Context) error {
	if s.rollbackData == nil {
		return fmt.Errorf("no rollback data available")
	}

	fmt.Printf("Rolling back drift simulation...\n")
	fmt.Printf("  Provider: %s\n", s.rollbackData.Provider)
	fmt.Printf("  Resource: %s (%s)\n", s.rollbackData.ResourceID, s.rollbackData.ResourceType)
	fmt.Printf("  Action: %s\n", s.rollbackData.Action)

	var err error
	switch s.provider {
	case "aws":
		err = s.awsSim.Rollback(ctx, s.rollbackData)
	case "azure":
		err = s.azureSim.Rollback(ctx, s.rollbackData)
	case "gcp":
		err = s.gcpSim.Rollback(ctx, s.rollbackData)
	}

	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Println("✓ Rollback completed successfully")
	return nil
}

// parseStateFile reads and parses the Terraform state file
func (s *DriftSimulator) parseStateFile() (*state.TerraformState, error) {
	data, err := ioutil.ReadFile(s.stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state state.TerraformState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// selectTargetResource selects a resource from the state to target
func (s *DriftSimulator) selectTargetResource(tfState *state.TerraformState) *state.Resource {
	if len(tfState.Resources) == 0 {
		return nil
	}

	// Filter resources by provider
	providerResources := make([]*state.Resource, 0)
	for i := range tfState.Resources {
		if strings.HasPrefix(tfState.Resources[i].Type, s.provider) {
			providerResources = append(providerResources, &tfState.Resources[i])
		}
	}

	if len(providerResources) == 0 {
		return nil
	}

	// Prefer certain resource types for drift simulation
	preferredTypes := s.getPreferredResourceTypes()
	for _, preferred := range preferredTypes {
		for _, resource := range providerResources {
			if resource.Type == preferred {
				return resource
			}
		}
	}

	// Return a random resource if no preferred type found
	rand.Seed(time.Now().UnixNano())
	return providerResources[rand.Intn(len(providerResources))]
}

// getPreferredResourceTypes returns resource types that are good for drift simulation
func (s *DriftSimulator) getPreferredResourceTypes() []string {
	switch s.provider {
	case "aws":
		return []string{
			"aws_instance",
			"aws_security_group",
			"aws_s3_bucket",
			"aws_iam_role",
			"aws_vpc",
		}
	case "azure":
		return []string{
			"azurerm_virtual_machine",
			"azurerm_network_security_group",
			"azurerm_storage_account",
			"azurerm_resource_group",
		}
	case "gcp":
		return []string{
			"google_compute_instance",
			"google_storage_bucket",
			"google_compute_firewall",
			"google_project_iam_member",
		}
	default:
		return []string{}
	}
}

// selectRandomDriftType selects a random drift type
func (s *DriftSimulator) selectRandomDriftType() DriftType {
	types := []DriftType{
		DriftTypeTagChange,
		DriftTypeRuleAddition,
		DriftTypeResourceCreation,
		DriftTypeAttributeChange,
	}
	
	rand.Seed(time.Now().UnixNano())
	return types[rand.Intn(len(types))]
}

// GenerateReport generates a detailed drift simulation report
func (s *DriftSimulator) GenerateReport(result *SimulationResult, drifts []DriftItem) string {
	var report strings.Builder

	report.WriteString("=== Drift Simulation Report ===\n\n")
	report.WriteString(fmt.Sprintf("State File: %s\n", s.stateFile))
	report.WriteString(fmt.Sprintf("Provider: %s\n", result.Provider))
	report.WriteString(fmt.Sprintf("Drift Type: %s\n", result.DriftType))
	report.WriteString(fmt.Sprintf("Resource: %s (%s)\n", result.ResourceID, result.ResourceType))
	report.WriteString(fmt.Sprintf("Cost Estimate: %s\n\n", result.CostEstimate))

	report.WriteString("Changes Applied:\n")
	for key, value := range result.Changes {
		report.WriteString(fmt.Sprintf("  - %s: %v\n", key, value))
	}

	if len(drifts) > 0 {
		report.WriteString("\nDetected Drift:\n")
		for _, drift := range drifts {
			report.WriteString(fmt.Sprintf("\n  Resource: %s\n", drift.ResourceID))
			report.WriteString(fmt.Sprintf("  Type: %s\n", drift.DriftType))
			report.WriteString(fmt.Sprintf("  Impact: %s\n", drift.Impact))
			
			if len(drift.Before) > 0 {
				report.WriteString("  Before:\n")
				for k, v := range drift.Before {
					report.WriteString(fmt.Sprintf("    %s: %v\n", k, v))
				}
			}
			
			if len(drift.After) > 0 {
				report.WriteString("  After:\n")
				for k, v := range drift.After {
					report.WriteString(fmt.Sprintf("    %s: %v\n", k, v))
				}
			}
		}
	}

	if result.RollbackData != nil {
		report.WriteString("\nRollback Information:\n")
		report.WriteString(fmt.Sprintf("  Can rollback: Yes\n"))
		report.WriteString(fmt.Sprintf("  Rollback command: driftmgr simulate-drift --rollback\n"))
	}

	return report.String()
}