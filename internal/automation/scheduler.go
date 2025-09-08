package automation

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Scheduler manages scheduled automation jobs
type Scheduler struct {
	jobs     map[string]*ScheduledJob
	mu       sync.RWMutex
	
	config   *SchedulerConfig
	stopChan chan struct{}
	running  bool
}

// ScheduledJob represents a scheduled automation job
type ScheduledJob struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`     // workflow, script, command
	Schedule   string                 `json:"schedule"` // cron expression
	WorkflowID string                 `json:"workflow_id,omitempty"`
	Script     string                 `json:"script,omitempty"`
	Command    string                 `json:"command,omitempty"`
	Input      map[string]interface{} `json:"input"`
	Enabled    bool                   `json:"enabled"`
	LastRun    time.Time              `json:"last_run"`
	NextRun    time.Time              `json:"next_run"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// SchedulerConfig represents configuration for the scheduler
type SchedulerConfig struct {
	MaxJobs             int           `json:"max_jobs"`
	CheckInterval       time.Duration `json:"check_interval"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	AutoCleanup         bool          `json:"auto_cleanup"`
	NotificationEnabled bool          `json:"notification_enabled"`
	AuditLogging        bool          `json:"audit_logging"`
}

// NewScheduler creates a new scheduler
func NewScheduler() *Scheduler {
	config := &SchedulerConfig{
		MaxJobs:             1000,
		CheckInterval:       1 * time.Minute,
		RetentionPeriod:     30 * 24 * time.Hour,
		AutoCleanup:         true,
		NotificationEnabled: true,
		AuditLogging:        true,
	}

	return &Scheduler{
		jobs:     make(map[string]*ScheduledJob),
		eventBus: eventBus,
		config:   config,
		stopChan: make(chan struct{}),
		running:  false,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.running = true
	s.stopChan = make(chan struct{})

	// Start the scheduler loop
	go s.schedulerLoop(ctx)

	// Publish event
	if s.eventBus != nil {
		event := WorkflowEvent{
			Type:      "scheduler_started",
			Message:   "Scheduler started",
			Severity:  "info",
			Timestamp: time.Now(),
		}
		s.eventBus.PublishWorkflowEvent(event)
	}

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	s.running = false
	close(s.stopChan)

	// Publish event
	if s.eventBus != nil {
		event := WorkflowEvent{
			Type:      "scheduler_stopped",
			Message:   "Scheduler stopped",
			Severity:  "info",
			Timestamp: time.Now(),
		}
		s.eventBus.PublishWorkflowEvent(event)
	}

	return nil
}

// ScheduleJob schedules a new job
func (s *Scheduler) ScheduleJob(ctx context.Context, job *ScheduledJob) (*ScheduledJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check job limit
	if len(s.jobs) >= s.config.MaxJobs {
		return nil, fmt.Errorf("maximum number of jobs reached (%d)", s.config.MaxJobs)
	}

	// Validate job
	if err := s.validateJob(job); err != nil {
		return nil, fmt.Errorf("invalid job: %w", err)
	}

	// Set defaults
	if job.ID == "" {
		job.ID = fmt.Sprintf("job_%d", time.Now().Unix())
	}
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()

	// Calculate next run time
	nextRun, err := s.calculateNextRun(job.Schedule)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule: %w", err)
	}
	job.NextRun = nextRun

	// Store job
	s.jobs[job.ID] = job

	// Publish event
	if s.eventBus != nil {
		event := WorkflowEvent{
			Type:      "job_scheduled",
			Message:   fmt.Sprintf("Job '%s' scheduled", job.Name),
			Severity:  "info",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"job_name": job.Name,
				"job_type": job.Type,
				"schedule": job.Schedule,
				"next_run": job.NextRun,
			},
		}
		s.eventBus.PublishWorkflowEvent(event)
	}

	return job, nil
}

// GetJob retrieves a scheduled job
func (s *Scheduler) GetJob(ctx context.Context, jobID string) (*ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}

	return job, nil
}

// ListJobs lists all scheduled jobs
func (s *Scheduler) ListJobs(ctx context.Context) ([]*ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// UpdateJob updates an existing job
func (s *Scheduler) UpdateJob(ctx context.Context, jobID string, updates *ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	// Update fields
	if updates.Name != "" {
		job.Name = updates.Name
	}
	if updates.Schedule != "" {
		job.Schedule = updates.Schedule
		// Recalculate next run time
		nextRun, err := s.calculateNextRun(job.Schedule)
		if err != nil {
			return fmt.Errorf("invalid schedule: %w", err)
		}
		job.NextRun = nextRun
	}
	if updates.Input != nil {
		job.Input = updates.Input
	}
	job.UpdatedAt = time.Now()

	// Publish event
	if s.eventBus != nil {
		event := WorkflowEvent{
			Type:      "job_updated",
			Message:   fmt.Sprintf("Job '%s' updated", job.Name),
			Severity:  "info",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"job_name": job.Name,
				"job_type": job.Type,
			},
		}
		s.eventBus.PublishWorkflowEvent(event)
	}

	return nil
}

// DeleteJob deletes a scheduled job
func (s *Scheduler) DeleteJob(ctx context.Context, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	// Delete job
	delete(s.jobs, jobID)

	// Publish event
	if s.eventBus != nil {
		event := WorkflowEvent{
			Type:      "job_deleted",
			Message:   fmt.Sprintf("Job '%s' deleted", job.Name),
			Severity:  "info",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"job_name": job.Name,
			},
		}
		s.eventBus.PublishWorkflowEvent(event)
	}

	return nil
}

// EnableJob enables a scheduled job
func (s *Scheduler) EnableJob(ctx context.Context, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	job.Enabled = true
	job.UpdatedAt = time.Now()

	// Publish event
	if s.eventBus != nil {
		event := WorkflowEvent{
			Type:      "job_enabled",
			Message:   fmt.Sprintf("Job '%s' enabled", job.Name),
			Severity:  "info",
			Timestamp: time.Now(),
		}
		s.eventBus.PublishWorkflowEvent(event)
	}

	return nil
}

// DisableJob disables a scheduled job
func (s *Scheduler) DisableJob(ctx context.Context, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s not found", jobID)
	}

	job.Enabled = false
	job.UpdatedAt = time.Now()

	// Publish event
	if s.eventBus != nil {
		event := WorkflowEvent{
			Type:      "job_disabled",
			Message:   fmt.Sprintf("Job '%s' disabled", job.Name),
			Severity:  "info",
			Timestamp: time.Now(),
		}
		s.eventBus.PublishWorkflowEvent(event)
	}

	return nil
}

// Helper methods

// validateJob validates a scheduled job
func (s *Scheduler) validateJob(job *ScheduledJob) error {
	if job.Name == "" {
		return fmt.Errorf("job name is required")
	}
	if job.Type == "" {
		return fmt.Errorf("job type is required")
	}
	if job.Schedule == "" {
		return fmt.Errorf("job schedule is required")
	}

	// Validate job type specific fields
	switch job.Type {
	case "workflow":
		if job.WorkflowID == "" {
			return fmt.Errorf("workflow ID is required for workflow jobs")
		}
	case "script":
		if job.Script == "" {
			return fmt.Errorf("script is required for script jobs")
		}
	case "command":
		if job.Command == "" {
			return fmt.Errorf("command is required for command jobs")
		}
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}

	return nil
}

// calculateNextRun calculates the next run time for a job
func (s *Scheduler) calculateNextRun(schedule string) (time.Time, error) {
	// This is a simplified implementation
	// In a real system, you would use a proper cron parser

	now := time.Now()

	// Handle simple schedules
	switch schedule {
	case "@hourly":
		return now.Add(1 * time.Hour), nil
	case "@daily":
		return now.Add(24 * time.Hour), nil
	case "@weekly":
		return now.Add(7 * 24 * time.Hour), nil
	case "@monthly":
		return now.Add(30 * 24 * time.Hour), nil
	default:
		// For now, assume it's a simple interval in minutes
		// In reality, you'd parse cron expressions
		return now.Add(5 * time.Minute), nil
	}
}

// schedulerLoop runs the main scheduler loop
func (s *Scheduler) schedulerLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkAndRunJobs(ctx)
		}
	}
}

// checkAndRunJobs checks for jobs that need to be run
func (s *Scheduler) checkAndRunJobs(ctx context.Context) {
	s.mu.RLock()
	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		if job.Enabled && time.Now().After(job.NextRun) {
			jobs = append(jobs, job)
		}
	}
	s.mu.RUnlock()

	// Run jobs
	for _, job := range jobs {
		go s.runJob(ctx, job)
	}
}

// runJob runs a scheduled job
func (s *Scheduler) runJob(ctx context.Context, job *ScheduledJob) {
	// Update last run time
	s.mu.Lock()
	job.LastRun = time.Now()
	// Calculate next run time
	nextRun, nextRunErr := s.calculateNextRun(job.Schedule)
	if nextRunErr == nil {
		job.NextRun = nextRun
	}
	s.mu.Unlock()

	// Publish event
	if s.eventBus != nil {
		event := WorkflowEvent{
			Type:      "job_started",
			Message:   fmt.Sprintf("Job '%s' started", job.Name),
			Severity:  "info",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"job_name": job.Name,
				"job_type": job.Type,
			},
		}
		s.eventBus.PublishWorkflowEvent(event)
	}

	// Execute job based on type
	var err error
	switch job.Type {
	case "workflow":
		err = s.executeWorkflowJob(ctx, job)
	case "script":
		err = s.executeScriptJob(ctx, job)
	case "command":
		err = s.executeCommandJob(ctx, job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.Type)
	}

	// Publish completion event
	if s.eventBus != nil {
		event := WorkflowEvent{
			Type:      "job_completed",
			Message:   fmt.Sprintf("Job '%s' completed", job.Name),
			Severity:  "info",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"job_name": job.Name,
				"job_type": job.Type,
				"success":  err == nil,
			},
		}
		s.eventBus.PublishWorkflowEvent(event)
	}

	if err != nil {
		fmt.Printf("Job %s failed: %v\n", job.Name, err)
	}
}

// executeWorkflowJob executes a workflow job
func (s *Scheduler) executeWorkflowJob(ctx context.Context, job *ScheduledJob) error {
	// This would be handled by the automation service
	fmt.Printf("Executing workflow job: %s (workflow: %s)\n", job.Name, job.WorkflowID)
	return nil
}

// executeScriptJob executes a script job
func (s *Scheduler) executeScriptJob(ctx context.Context, job *ScheduledJob) error {
	// Placeholder for script execution
	fmt.Printf("Executing script job: %s\n", job.Name)
	return nil
}

// executeCommandJob executes a command job
func (s *Scheduler) executeCommandJob(ctx context.Context, job *ScheduledJob) error {
	// Placeholder for command execution
	fmt.Printf("Executing command job: %s (command: %s)\n", job.Name, job.Command)
	return nil
}

// SetConfig updates the scheduler configuration
func (s *Scheduler) SetConfig(config *SchedulerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

// GetConfig returns the current scheduler configuration
func (s *Scheduler) GetConfig() *SchedulerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}
