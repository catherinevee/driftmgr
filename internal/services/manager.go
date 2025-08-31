package services

import (
	"context"
	"fmt"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/drift"
	"github.com/catherinevee/driftmgr/internal/core/remediation"
	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/jobs"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// Manager coordinates all services
type Manager struct {
	Discovery    *DiscoveryService
	State        *StateService
	Drift        *DriftService
	Remediation  *RemediationService
	EventBus     *events.EventBus
	JobQueue     *jobs.Queue
	Cache        cache.Cache
}

// Config holds configuration for the service manager
type Config struct {
	CacheSize       int
	JobWorkers      int
	EventBufferSize int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		CacheSize:       1000,
		JobWorkers:      5,
		EventBufferSize: 100,
	}
}

// NewManager creates a new service manager
func NewManager(cfg *Config) (*Manager, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Initialize shared components
	cacheInstance := cache.GetGlobalCache()
	eventBus := events.NewEventBus(cfg.EventBufferSize)
	
	// Initialize job queue with in-memory persistence for now
	jobQueue := jobs.NewQueue(cfg.JobWorkers, nil)

	// Initialize discovery engine
	discoveryEngine := discovery.NewEngine()

	// Initialize state loader
	stateLoader := state.NewLoader()

	// Initialize drift detector
	driftDetector := drift.NewDetector()

	// Initialize remediation executor
	remediationExecutor := remediation.NewExecutor()

	// Create services
	discoveryService := NewDiscoveryService(
		discoveryEngine,
		cacheInstance,
		eventBus,
		jobQueue,
	)

	stateService := NewStateService(
		stateLoader,
		cacheInstance,
		eventBus,
		jobQueue,
	)

	driftService := NewDriftService(
		driftDetector,
		discoveryService,
		stateService,
		cacheInstance,
		eventBus,
		jobQueue,
	)

	remediationService := NewRemediationService(
		remediationExecutor,
		driftService,
		cacheInstance,
		eventBus,
		jobQueue,
	)

	// Register job handlers
	registerJobHandlers(jobQueue, discoveryService, stateService, driftService, remediationService)

	// Create manager
	manager := &Manager{
		Discovery:   discoveryService,
		State:       stateService,
		Drift:       driftService,
		Remediation: remediationService,
		EventBus:    eventBus,
		JobQueue:    jobQueue,
		Cache:       cacheInstance,
	}

	return manager, nil
}

// registerJobHandlers registers handlers for different job types
func registerJobHandlers(
	queue *jobs.Queue,
	discovery *DiscoveryService,
	state *StateService,
	drift *DriftService,
	remediation *RemediationService,
) {
	// Discovery job handler
	queue.RegisterHandler(jobs.DiscoveryJob, func(ctx context.Context, job *jobs.Job) error {
		req, ok := job.Data.(DiscoveryRequest)
		if !ok {
			return fmt.Errorf("invalid discovery request data")
		}
		
		response, err := discovery.executeDiscovery(ctx, job.ID, req)
		if err != nil {
			return err
		}
		
		job.Result = response
		job.Progress = 100
		job.Message = "Discovery completed"
		return nil
	})

	// Drift detection job handler
	queue.RegisterHandler(jobs.DriftDetection, func(ctx context.Context, job *jobs.Job) error {
		req, ok := job.Data.(DriftDetectionRequest)
		if !ok {
			return fmt.Errorf("invalid drift detection request data")
		}
		
		response, err := drift.executeDriftDetection(ctx, job.ID, req)
		if err != nil {
			return err
		}
		
		job.Result = response
		job.Progress = 100
		job.Message = "Drift detection completed"
		return nil
	})

	// State analysis job handler
	queue.RegisterHandler(jobs.StateAnalysis, func(ctx context.Context, job *jobs.Job) error {
		fileIDs, ok := job.Data.([]string)
		if !ok {
			return fmt.Errorf("invalid state analysis request data")
		}
		
		analysis, err := state.AnalyzeStateFiles(ctx, fileIDs)
		if err != nil {
			return err
		}
		
		job.Result = analysis
		job.Progress = 100
		job.Message = "State analysis completed"
		return nil
	})

	// Remediation job handler
	queue.RegisterHandler(jobs.Remediation, func(ctx context.Context, job *jobs.Job) error {
		data, ok := job.Data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid remediation request data")
		}
		
		req, _ := data["request"].(RemediationRequest)
		plan, _ := data["plan"].(*RemediationPlan)
		
		response, err := remediation.executeRemediation(ctx, job.ID, req, plan)
		if err != nil {
			return err
		}
		
		job.Result = response
		job.Progress = 100
		job.Message = "Remediation completed"
		return nil
	})
}

// Start starts all services
func (m *Manager) Start(ctx context.Context) error {
	// Services are already running via their goroutines
	// This method can be used for additional initialization if needed
	return nil
}

// Shutdown gracefully shuts down all services
func (m *Manager) Shutdown(ctx context.Context) error {
	// Shutdown job queue
	if err := m.JobQueue.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown job queue: %w", err)
	}

	// Clear event bus
	m.EventBus.Clear()

	// Clear cache
	if err := m.Cache.Clear(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	return nil
}

// GetStats returns statistics about the services
func (m *Manager) GetStats() map[string]interface{} {
	cacheStats := m.Cache.GetStats()
	cacheSize := 0
	if items, ok := cacheStats["items"].(int); ok {
		cacheSize = items
	} else if items, ok := cacheStats["total_entries"].(int); ok {
		cacheSize = items
	}
	
	return map[string]interface{}{
		"job_queue":   m.JobQueue.GetQueueStatus(),
		"event_bus":   m.EventBus.GetMetrics(),
		"cache_size":  cacheSize,
		"cache_stats": cacheStats,
	}
}

// SubscribeToEvents allows subscribing to specific event types
func (m *Manager) SubscribeToEvents(eventTypes []events.EventType, handler events.Handler) *events.Subscription {
	return m.EventBus.SubscribeToTypes(eventTypes, handler)
}

// ExecuteWorkflow executes a predefined workflow
func (m *Manager) ExecuteWorkflow(ctx context.Context, workflowType string, params map[string]interface{}) error {
	switch workflowType {
	case "terraform_drift":
		return m.executeTerraformDriftWorkflow(ctx, params)
	case "cleanup_unmanaged":
		return m.executeCleanupUnmanagedWorkflow(ctx, params)
	case "state_migration":
		return m.executeStateMigrationWorkflow(ctx, params)
	default:
		return fmt.Errorf("unknown workflow type: %s", workflowType)
	}
}

// executeTerraformDriftWorkflow executes a complete Terraform drift workflow
func (m *Manager) executeTerraformDriftWorkflow(ctx context.Context, params map[string]interface{}) error {
	// 1. Discover state files
	paths, _ := params["paths"].([]string)
	stateFiles, err := m.State.DiscoverStateFiles(ctx, paths)
	if err != nil {
		return fmt.Errorf("failed to discover state files: %w", err)
	}

	// 2. Analyze state files
	var fileIDs []string
	for _, sf := range stateFiles {
		fileIDs = append(fileIDs, sf.ID)
	}
	
	analysis, err := m.State.AnalyzeStateFiles(ctx, fileIDs)
	if err != nil {
		return fmt.Errorf("failed to analyze state files: %w", err)
	}

	// 3. Detect drift for each state file
	for _, sf := range stateFiles {
		driftReq := DriftDetectionRequest{
			StateFileID: sf.ID,
			Async:       false,
		}
		
		driftResp, err := m.Drift.StartDriftDetection(ctx, driftReq)
		if err != nil {
			continue
		}

		// 4. Auto-remediate if configured
		if autoRemediate, ok := params["auto_remediate"].(bool); ok && autoRemediate {
			if driftResp.Report != nil && driftResp.Report.Summary.Remediable > 0 {
				remediationReq := RemediationRequest{
					DriftReportID: driftResp.Report.ID,
					DryRun:        false,
					Async:         false,
				}
				
				_, err := m.Remediation.StartRemediation(ctx, remediationReq)
				if err != nil {
					return fmt.Errorf("failed to remediate: %w", err)
				}
			}
		}
	}

	// Emit workflow completed event
	m.EventBus.Publish(events.Event{
		Type: events.JobCompleted,
		Data: map[string]interface{}{
			"workflow":     "terraform_drift",
			"state_files":  len(stateFiles),
			"analysis":     analysis,
		},
	})

	return nil
}

// executeCleanupUnmanagedWorkflow executes a cleanup workflow for unmanaged resources
func (m *Manager) executeCleanupUnmanagedWorkflow(ctx context.Context, params map[string]interface{}) error {
	provider, _ := params["provider"].(string)
	regions, _ := params["regions"].([]string)
	
	// 1. Discover all resources
	discoveryReq := DiscoveryRequest{
		Provider: provider,
		Regions:  regions,
		Async:    false,
	}
	
	discoveryResp, err := m.Discovery.StartDiscovery(ctx, discoveryReq)
	if err != nil {
		return fmt.Errorf("failed to discover resources: %w", err)
	}

	// 2. Detect drift to identify unmanaged resources
	driftReq := DriftDetectionRequest{
		Provider: provider,
		Regions:  regions,
		Async:    false,
	}
	
	driftResp, err := m.Drift.StartDriftDetection(ctx, driftReq)
	if err != nil {
		return fmt.Errorf("failed to detect drift: %w", err)
	}

	// 3. Filter unmanaged resources
	var unmanagedCount int
	if driftResp.Report != nil {
		for _, drift := range driftResp.Report.Drifts {
			if drift.DriftType == "unmanaged" {
				unmanagedCount++
			}
		}
	}

	// Emit workflow completed event
	m.EventBus.Publish(events.Event{
		Type: events.JobCompleted,
		Data: map[string]interface{}{
			"workflow":           "cleanup_unmanaged",
			"total_resources":    len(discoveryResp.Resources),
			"unmanaged_resources": unmanagedCount,
		},
	})

	return nil
}

// executeStateMigrationWorkflow executes a state migration workflow
func (m *Manager) executeStateMigrationWorkflow(ctx context.Context, params map[string]interface{}) error {
	sourcePath, _ := params["source_path"].(string)
	targetPath, _ := params["target_path"].(string)
	
	// 1. Import source state file
	sourceState, err := m.State.ImportStateFile(ctx, sourcePath)
	if err != nil {
		return fmt.Errorf("failed to import source state: %w", err)
	}

	// 2. Import target state file if exists
	var targetState *StateFile
	if targetPath != "" {
		targetState, _ = m.State.ImportStateFile(ctx, targetPath)
	}

	// 3. Compare states if both exist
	if targetState != nil {
		comparison, err := m.State.CompareStateFiles(ctx, sourceState.ID, targetState.ID)
		if err != nil {
			return fmt.Errorf("failed to compare states: %w", err)
		}

		// Emit comparison results
		m.EventBus.Publish(events.Event{
			Type: events.StateAnalyzed,
			Data: map[string]interface{}{
				"workflow":  "state_migration",
				"added":     len(comparison.AddedResources),
				"removed":   len(comparison.RemovedResources),
				"modified":  len(comparison.ModifiedResources),
				"unchanged": comparison.UnchangedCount,
			},
		})
	}

	return nil
}