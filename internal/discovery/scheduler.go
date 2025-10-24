package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// Scheduler represents a discovery job scheduler
type Scheduler struct {
	jobs      map[string]*models.DiscoveryJob
	schedules map[string]*models.DiscoverySchedule
	running   bool
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
}

// NewScheduler creates a new discovery scheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		jobs:      make(map[string]*models.DiscoveryJob),
		schedules: make(map[string]*models.DiscoverySchedule),
		running:   false,
	}
}

// Schedule schedules a discovery job
func (s *Scheduler) Schedule(job *models.DiscoveryJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.jobs[job.ID] != nil {
		return fmt.Errorf("job %s already exists", job.ID)
	}

	s.jobs[job.ID] = job
	return nil
}

// ScheduleRecurring schedules a recurring discovery job
func (s *Scheduler) ScheduleRecurring(schedule *models.DiscoverySchedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.schedules[schedule.ID] != nil {
		return fmt.Errorf("schedule %s already exists", schedule.ID)
	}

	s.schedules[schedule.ID] = schedule
	return nil
}

// Cancel cancels a scheduled discovery job
func (s *Scheduler) Cancel(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.jobs[jobID] == nil {
		return fmt.Errorf("job %s not found", jobID)
	}

	// Update job status
	s.jobs[jobID].SetStatus(models.JobStatusCancelled)
	return nil
}

// CancelRecurring cancels a recurring discovery schedule
func (s *Scheduler) CancelRecurring(scheduleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.schedules[scheduleID] == nil {
		return fmt.Errorf("schedule %s not found", scheduleID)
	}

	// Mark schedule as inactive
	s.schedules[scheduleID].IsActive = false
	return nil
}

// GetJobs returns all scheduled discovery jobs
func (s *Scheduler) GetJobs() []*models.DiscoveryJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var jobs []*models.DiscoveryJob
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetSchedules returns all discovery schedules
func (s *Scheduler) GetSchedules() []*models.DiscoverySchedule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var schedules []*models.DiscoverySchedule
	for _, schedule := range s.schedules {
		schedules = append(schedules, schedule)
	}
	return schedules
}

// GetJob returns a specific discovery job
func (s *Scheduler) GetJob(jobID string) (*models.DiscoveryJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}

	return job, nil
}

// GetSchedule returns a specific discovery schedule
func (s *Scheduler) GetSchedule(scheduleID string) (*models.DiscoverySchedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return nil, fmt.Errorf("schedule %s not found", scheduleID)
	}

	return schedule, nil
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true

	// Start the scheduling loop
	go s.schedulingLoop()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
}

// schedulingLoop runs the main scheduling loop
func (s *Scheduler) schedulingLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.processSchedules()
		}
	}
}

// processSchedules processes all active schedules
func (s *Scheduler) processSchedules() {
	s.mu.RLock()
	schedules := make([]*models.DiscoverySchedule, 0, len(s.schedules))
	for _, schedule := range s.schedules {
		if schedule.IsActive {
			schedules = append(schedules, schedule)
		}
	}
	s.mu.RUnlock()

	now := time.Now()
	for _, schedule := range schedules {
		// Check if it's time to run the schedule
		if schedule.NextRun != nil && now.After(*schedule.NextRun) {
			s.executeSchedule(schedule)
		}
	}
}

// executeSchedule executes a discovery schedule
func (s *Scheduler) executeSchedule(schedule *models.DiscoverySchedule) {
	// Create a new discovery job from the schedule
	job := &models.DiscoveryJob{
		ID:            fmt.Sprintf("%s-%d", schedule.ID, time.Now().Unix()),
		Provider:      schedule.Provider,
		AccountID:     schedule.AccountID,
		Region:        schedule.Region,
		ResourceTypes: schedule.ResourceTypes,
		Status:        models.JobStatusPending,
		Configuration: schedule.Configuration,
		CreatedBy:     schedule.CreatedBy,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Schedule the job
	s.Schedule(job)

	// Update schedule
	s.mu.Lock()
	schedule.LastRun = &time.Time{}
	*schedule.LastRun = time.Now()

	// Calculate next run time (simplified - in production, use a proper cron parser)
	nextRun := schedule.LastRun.Add(1 * time.Hour) // Default to 1 hour
	schedule.NextRun = &nextRun
	s.mu.Unlock()
}

// GetNextRunTime returns the next run time for a schedule
func (s *Scheduler) GetNextRunTime(scheduleID string) (*time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return nil, fmt.Errorf("schedule %s not found", scheduleID)
	}

	return schedule.NextRun, nil
}

// GetLastRunTime returns the last run time for a schedule
func (s *Scheduler) GetLastRunTime(scheduleID string) (*time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedule, exists := s.schedules[scheduleID]
	if !exists {
		return nil, fmt.Errorf("schedule %s not found", scheduleID)
	}

	return schedule.LastRun, nil
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetStatistics returns scheduler statistics
func (s *Scheduler) GetStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeJobs := 0
	completedJobs := 0
	failedJobs := 0
	activeSchedules := 0

	for _, job := range s.jobs {
		switch job.Status {
		case models.JobStatusRunning:
			activeJobs++
		case models.JobStatusCompleted:
			completedJobs++
		case models.JobStatusFailed:
			failedJobs++
		}
	}

	for _, schedule := range s.schedules {
		if schedule.IsActive {
			activeSchedules++
		}
	}

	return map[string]interface{}{
		"total_jobs":       len(s.jobs),
		"active_jobs":      activeJobs,
		"completed_jobs":   completedJobs,
		"failed_jobs":      failedJobs,
		"total_schedules":  len(s.schedules),
		"active_schedules": activeSchedules,
		"is_running":       s.running,
	}
}
