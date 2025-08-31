package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/cache"
	"github.com/catherinevee/driftmgr/internal/core/drift"
	"github.com/catherinevee/driftmgr/internal/events"
	"github.com/catherinevee/driftmgr/internal/jobs"
	"github.com/catherinevee/driftmgr/internal/providers/cloud"
	"github.com/google/uuid"
)

// DriftService provides unified drift detection operations
type DriftService struct {
	driftDetector    *drift.Detector
	discoveryService *DiscoveryService
	stateService     *StateService
	cache            cache.Cache
	eventBus         *events.EventBus
	jobQueue         *jobs.Queue
	mu               sync.RWMutex
}

// NewDriftService creates a new drift service
func NewDriftService(
	driftDetector *drift.Detector,
	discoveryService *DiscoveryService,
	stateService *StateService,
	cache cache.Cache,
	eventBus *events.EventBus,
	jobQueue *jobs.Queue,
) *DriftService {
	return &DriftService{
		driftDetector:    driftDetector,
		discoveryService: discoveryService,
		stateService:     stateService,
		cache:            cache,
		eventBus:         eventBus,
		jobQueue:         jobQueue,
	}
}

// DriftDetectionRequest represents a request to detect drift
type DriftDetectionRequest struct {
	Provider      string   `json:"provider,omitempty"`
	StateFileID   string   `json:"state_file_id,omitempty"`
	StateFilePath string   `json:"state_file_path,omitempty"`
	ResourceTypes []string `json:"resource_types,omitempty"`
	Regions       []string `json:"regions,omitempty"`
	AutoRemediate bool     `json:"auto_remediate"`
	Async         bool     `json:"async"`
}

// DriftDetectionResponse represents the response from drift detection
type DriftDetectionResponse struct {
	JobID      string       `json:"job_id,omitempty"`
	Status     string       `json:"status"`
	Progress   int          `json:"progress"`
	Message    string       `json:"message"`
	Report     *DriftReport `json:"report,omitempty"`
	StartedAt  time.Time    `json:"started_at"`
	EndedAt    *time.Time   `json:"ended_at,omitempty"`
}

// DriftReport represents a drift detection report
type DriftReport struct {
	ID               string         `json:"id"`
	Summary          DriftSummary   `json:"summary"`
	Drifts           []DriftItem    `json:"drifts"`
	GeneratedAt      time.Time      `json:"generated_at"`
	StateFileID      string         `json:"state_file_id,omitempty"`
	Provider         string         `json:"provider,omitempty"`
	ComplianceScore  float64        `json:"compliance_score"`
	Recommendations  []string       `json:"recommendations,omitempty"`
}

// DriftSummary provides a summary of drift detection
type DriftSummary struct {
	Total            int     `json:"total"`
	Drifted          int     `json:"drifted"`
	Missing          int     `json:"missing"`
	Unmanaged        int     `json:"unmanaged"`
	Compliant        int     `json:"compliant"`
	Remediable       int     `json:"remediable"`
	SecurityRelated  int     `json:"security_related"`
	CostImpact       float64 `json:"cost_impact"`
}

// DriftItem represents a single drift item
type DriftItem struct {
	ResourceID      string                 `json:"resource_id"`
	ResourceType    string                 `json:"resource_type"`
	ResourceName    string                 `json:"resource_name"`
	Provider        string                 `json:"provider"`
	Region          string                 `json:"region"`
	DriftType       string                 `json:"drift_type"`
	Severity        string                 `json:"severity"`
	StateDiff       map[string]interface{} `json:"state_diff"`
	ActualState     map[string]interface{} `json:"actual_state"`
	ExpectedState   map[string]interface{} `json:"expected_state"`
	Remediable      bool                   `json:"remediable"`
	RemediationPlan string                 `json:"remediation_plan,omitempty"`
	SecurityImpact  bool                   `json:"security_impact"`
	CostImpact      float64                `json:"cost_impact"`
	DetectedAt      time.Time              `json:"detected_at"`
}

// StartDriftDetection initiates drift detection
func (s *DriftService) StartDriftDetection(ctx context.Context, req DriftDetectionRequest) (*DriftDetectionResponse, error) {
	// Validate request
	if err := s.validateDriftRequest(req); err != nil {
		return nil, fmt.Errorf("invalid drift detection request: %w", err)
	}

	// Generate job ID
	jobID := uuid.New().String()

	// Emit drift detection started event
	s.eventBus.Publish(events.Event{
		Type:      events.DriftDetectionStarted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"job_id":        jobID,
			"provider":      req.Provider,
			"state_file_id": req.StateFileID,
		},
	})

	// If async, create a job and return immediately
	if req.Async {
		job := &jobs.Job{
			ID:        jobID,
			Type:      jobs.DriftDetection,
			Status:    jobs.StatusPending,
			CreatedAt: time.Now(),
			Data:      req,
		}

		if err := s.jobQueue.Enqueue(job); err != nil {
			return nil, fmt.Errorf("failed to enqueue drift detection job: %w", err)
		}

		// Start processing in background
		go s.processDriftJob(context.Background(), job)

		return &DriftDetectionResponse{
			JobID:     jobID,
			Status:    "running",
			Progress:  0,
			Message:   "Drift detection job started",
			StartedAt: time.Now(),
		}, nil
	}

	// Synchronous drift detection
	return s.executeDriftDetection(ctx, jobID, req)
}

// executeDriftDetection performs the actual drift detection
func (s *DriftService) executeDriftDetection(ctx context.Context, jobID string, req DriftDetectionRequest) (*DriftDetectionResponse, error) {
	startTime := time.Now()
	response := &DriftDetectionResponse{
		JobID:     jobID,
		Status:    "running",
		Progress:  0,
		Message:   "Starting drift detection",
		StartedAt: startTime,
	}

	report := &DriftReport{
		ID:          jobID,
		Drifts:      []DriftItem{},
		GeneratedAt: time.Now(),
		Provider:    req.Provider,
		StateFileID: req.StateFileID,
	}

	// Get state file if specified
	var stateFile *StateFile
	if req.StateFileID != "" {
		sf, err := s.stateService.GetStateFile(ctx, req.StateFileID)
		if err != nil {
			return nil, fmt.Errorf("failed to get state file: %w", err)
		}
		stateFile = sf
	} else if req.StateFilePath != "" {
		sf, err := s.stateService.ImportStateFile(ctx, req.StateFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to import state file: %w", err)
		}
		stateFile = sf
		report.StateFileID = sf.ID
	}

	// If we have a state file, perform state-based drift detection
	if stateFile != nil {
		drifts, err := s.detectStateFileDrift(ctx, stateFile, req)
		if err != nil {
			return nil, fmt.Errorf("failed to detect state file drift: %w", err)
		}
		report.Drifts = append(report.Drifts, drifts...)
	} else {
		// Otherwise, perform provider-based drift detection
		drifts, err := s.detectProviderDrift(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to detect provider drift: %w", err)
		}
		report.Drifts = append(report.Drifts, drifts...)
	}

	// Calculate summary
	report.Summary = s.calculateDriftSummary(report.Drifts)
	report.ComplianceScore = s.calculateComplianceScore(report.Summary)
	report.Recommendations = s.generateRecommendations(report)

	// Cache the report
	s.cache.Set(fmt.Sprintf("drift:report:%s", jobID), report, 1*time.Hour)

	// Final response
	endTime := time.Now()
	response.EndedAt = &endTime
	response.Status = "completed"
	response.Progress = 100
	response.Message = fmt.Sprintf("Drift detection completed. Found %d drifts", report.Summary.Drifted)
	response.Report = report

	// Emit drift detection completed event
	s.eventBus.Publish(events.Event{
		Type:      events.DriftDetectionCompleted,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"job_id":     jobID,
			"total":      report.Summary.Total,
			"drifted":    report.Summary.Drifted,
			"compliance": report.ComplianceScore,
		},
	})

	return response, nil
}

// detectStateFileDrift detects drift for resources in a state file
func (s *DriftService) detectStateFileDrift(ctx context.Context, stateFile *StateFile, req DriftDetectionRequest) ([]DriftItem, error) {
	var drifts []DriftItem
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Group resources by provider
	resourcesByProvider := make(map[string][]StateResource)
	for _, resource := range stateFile.Resources {
		provider := resource.Provider
		if provider == "" {
			provider = extractProviderFromType(resource.Type)
		}
		resourcesByProvider[provider] = append(resourcesByProvider[provider], resource)
	}

	// Detect drift for each provider
	for provider, resources := range resourcesByProvider {
		wg.Add(1)
		go func(p string, res []StateResource) {
			defer wg.Done()

			// Discover current resources
			discoveryReq := DiscoveryRequest{
				Provider: p,
				Regions:  req.Regions,
				Async:    false,
			}
			
			discoveryResp, err := s.discoveryService.StartDiscovery(ctx, discoveryReq)
			if err != nil {
				return
			}

			// Compare state resources with discovered resources
			currentResourcesMap := make(map[string]cloud.Resource)
			for _, r := range discoveryResp.Resources {
				key := fmt.Sprintf("%s:%s", r.Type, r.ID)
				currentResourcesMap[key] = r
			}

			// Check each state resource
			for _, stateResource := range res {
				key := fmt.Sprintf("%s:%s", stateResource.Type, stateResource.ID)
				
				if currentResource, exists := currentResourcesMap[key]; exists {
					// Resource exists - check for drift
					if diff := s.compareResources(stateResource, currentResource); diff != nil {
						drift := DriftItem{
							ResourceID:     stateResource.ID,
							ResourceType:   stateResource.Type,
							ResourceName:   stateResource.Name,
							Provider:       p,
							DriftType:      "modified",
							Severity:       s.calculateSeverity(diff),
							StateDiff:      diff,
							ExpectedState:  stateResource.Attributes,
							ActualState:    currentResource.Properties,
							Remediable:     s.isRemediable(stateResource.Type),
							SecurityImpact: s.hasSecurityImpact(diff),
							CostImpact:     s.calculateCostImpact(diff),
							DetectedAt:     time.Now(),
						}
						
						mu.Lock()
						drifts = append(drifts, drift)
						mu.Unlock()
					}
				} else {
					// Resource missing
					drift := DriftItem{
						ResourceID:     stateResource.ID,
						ResourceType:   stateResource.Type,
						ResourceName:   stateResource.Name,
						Provider:       p,
						DriftType:      "missing",
						Severity:       "high",
						ExpectedState:  stateResource.Attributes,
						Remediable:     true,
						SecurityImpact: true,
						DetectedAt:     time.Now(),
					}
					
					mu.Lock()
					drifts = append(drifts, drift)
					mu.Unlock()
				}
			}

			// Check for unmanaged resources (exist in cloud but not in state)
			for key, currentResource := range currentResourcesMap {
				found := false
				for _, stateResource := range res {
					stateKey := fmt.Sprintf("%s:%s", stateResource.Type, stateResource.ID)
					if stateKey == key {
						found = true
						break
					}
				}
				
				if !found {
					drift := DriftItem{
						ResourceID:   currentResource.ID,
						ResourceType: currentResource.Type,
						ResourceName: currentResource.Name,
						Provider:     p,
						DriftType:    "unmanaged",
						Severity:     "medium",
						ActualState:  currentResource.Properties,
						Remediable:   false,
						DetectedAt:   time.Now(),
					}
					
					mu.Lock()
					drifts = append(drifts, drift)
					mu.Unlock()
				}
			}
		}(provider, resources)
	}

	wg.Wait()
	return drifts, nil
}

// detectProviderDrift detects drift for a provider without state file
func (s *DriftService) detectProviderDrift(ctx context.Context, req DriftDetectionRequest) ([]DriftItem, error) {
	// This would typically compare against a baseline or policy
	// For now, we'll just mark all discovered resources as potential drift
	var drifts []DriftItem

	discoveryReq := DiscoveryRequest{
		Provider:      req.Provider,
		Regions:       req.Regions,
		ResourceTypes: req.ResourceTypes,
		Async:         false,
	}

	discoveryResp, err := s.discoveryService.StartDiscovery(ctx, discoveryReq)
	if err != nil {
		return nil, err
	}

	// Analyze discovered resources for potential issues
	for _, resource := range discoveryResp.Resources {
		// Check against policies or best practices
		if issues := s.analyzeResourceCompliance(resource); len(issues) > 0 {
			drift := DriftItem{
				ResourceID:     resource.ID,
				ResourceType:   resource.Type,
				ResourceName:   resource.Name,
				Provider:       req.Provider,
				Region:         resource.Region,
				DriftType:      "compliance",
				Severity:       "medium",
				ActualState:    resource.Properties,
				Remediable:     false,
				SecurityImpact: s.hasSecurityViolation(resource),
				DetectedAt:     time.Now(),
			}
			drifts = append(drifts, drift)
		}
	}

	return drifts, nil
}

// processDriftJob processes an async drift detection job
func (s *DriftService) processDriftJob(ctx context.Context, job *jobs.Job) {
	req, ok := job.Data.(DriftDetectionRequest)
	if !ok {
		job.Status = jobs.StatusFailed
		job.Error = fmt.Errorf("invalid job data")
		s.jobQueue.UpdateJob(job)
		return
	}

	job.Status = jobs.StatusRunning
	job.StartedAt = timePtr(time.Now())
	s.jobQueue.UpdateJob(job)

	response, err := s.executeDriftDetection(ctx, job.ID, req)
	
	if err != nil {
		job.Status = jobs.StatusFailed
		job.Error = err
		s.eventBus.Publish(events.Event{
			Type: events.DriftDetectionFailed,
			Data: map[string]interface{}{
				"job_id": job.ID,
				"error":  err.Error(),
			},
		})
	} else {
		job.Status = jobs.StatusCompleted
		job.Result = response
	}
	
	job.CompletedAt = timePtr(time.Now())
	s.jobQueue.UpdateJob(job)
}

// GetDriftReport retrieves a drift report
func (s *DriftService) GetDriftReport(ctx context.Context, reportID string) (*DriftReport, error) {
	cacheKey := fmt.Sprintf("drift:report:%s", reportID)
	if cached, found := s.cache.Get(cacheKey); found {
		if report, ok := cached.(*DriftReport); ok {
			return report, nil
		}
	}
	return nil, fmt.Errorf("drift report not found: %s", reportID)
}

// Helper functions

func (s *DriftService) validateDriftRequest(req DriftDetectionRequest) error {
	if req.Provider == "" && req.StateFileID == "" && req.StateFilePath == "" {
		return fmt.Errorf("provider, state_file_id, or state_file_path required")
	}
	return nil
}

func (s *DriftService) calculateDriftSummary(drifts []DriftItem) DriftSummary {
	summary := DriftSummary{
		Total: len(drifts),
	}

	for _, drift := range drifts {
		switch drift.DriftType {
		case "modified":
			summary.Drifted++
		case "missing":
			summary.Missing++
		case "unmanaged":
			summary.Unmanaged++
		case "compliant":
			summary.Compliant++
		}

		if drift.Remediable {
			summary.Remediable++
		}
		if drift.SecurityImpact {
			summary.SecurityRelated++
		}
		summary.CostImpact += drift.CostImpact
	}

	return summary
}

func (s *DriftService) calculateComplianceScore(summary DriftSummary) float64 {
	if summary.Total == 0 {
		return 100.0
	}
	compliant := float64(summary.Total - summary.Drifted - summary.Missing)
	return (compliant / float64(summary.Total)) * 100
}

func (s *DriftService) generateRecommendations(report *DriftReport) []string {
	var recommendations []string

	if report.Summary.SecurityRelated > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("Address %d security-related drifts immediately", report.Summary.SecurityRelated))
	}

	if report.Summary.Remediable > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("%d drifts can be auto-remediated", report.Summary.Remediable))
	}

	if report.Summary.Unmanaged > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Import %d unmanaged resources into Terraform state", report.Summary.Unmanaged))
	}

	if report.ComplianceScore < 80 {
		recommendations = append(recommendations,
			"Compliance score below 80% - immediate action required")
	}

	return recommendations
}

func (s *DriftService) compareResources(state StateResource, current cloud.Resource) map[string]interface{} {
	// Simple comparison - can be enhanced
	diff := make(map[string]interface{})
	
	for key, stateValue := range state.Attributes {
		if currentValue, exists := current.Properties[key]; exists {
			if stateValue != currentValue {
				diff[key] = map[string]interface{}{
					"expected": stateValue,
					"actual":   currentValue,
				}
			}
		}
	}

	return diff
}

func (s *DriftService) calculateSeverity(diff map[string]interface{}) string {
	// Determine severity based on the type of changes
	if len(diff) > 5 {
		return "high"
	} else if len(diff) > 2 {
		return "medium"
	}
	return "low"
}

func (s *DriftService) isRemediable(resourceType string) bool {
	// Define which resource types can be auto-remediated
	remediableTypes := map[string]bool{
		"aws_instance":        true,
		"aws_security_group":  true,
		"azure_virtual_machine": true,
		"google_compute_instance": true,
	}
	return remediableTypes[resourceType]
}

func (s *DriftService) hasSecurityImpact(diff map[string]interface{}) bool {
	// Check if changes affect security-related attributes
	securityAttributes := []string{
		"security_groups",
		"iam_role",
		"encryption",
		"public_ip",
		"firewall_rules",
	}
	
	for _, attr := range securityAttributes {
		if _, exists := diff[attr]; exists {
			return true
		}
	}
	return false
}

func (s *DriftService) calculateCostImpact(diff map[string]interface{}) float64 {
	// Simplified cost impact calculation
	costAttributes := map[string]float64{
		"instance_type": 50.0,
		"size":          30.0,
		"storage":       10.0,
	}
	
	impact := 0.0
	for attr, cost := range costAttributes {
		if _, exists := diff[attr]; exists {
			impact += cost
		}
	}
	return impact
}

func (s *DriftService) analyzeResourceCompliance(resource cloud.Resource) []string {
	var issues []string
	
	// Check for common compliance issues
	if public, ok := resource.Properties["public_ip"].(bool); ok && public {
		issues = append(issues, "Resource has public IP")
	}
	
	if encrypted, ok := resource.Properties["encrypted"].(bool); ok && !encrypted {
		issues = append(issues, "Resource is not encrypted")
	}
	
	return issues
}

func (s *DriftService) hasSecurityViolation(resource cloud.Resource) bool {
	// Check for security violations
	return len(s.analyzeResourceCompliance(resource)) > 0
}

func extractProviderFromType(resourceType string) string {
	if strings.HasPrefix(resourceType, "aws_") {
		return "aws"
	} else if strings.HasPrefix(resourceType, "azure") {
		return "azure"
	} else if strings.HasPrefix(resourceType, "google_") {
		return "gcp"
	} else if strings.HasPrefix(resourceType, "digitalocean_") {
		return "digitalocean"
	}
	return "unknown"
}