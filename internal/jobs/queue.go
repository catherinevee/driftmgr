package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// JobType represents the type of job
type JobType string

const (
	DiscoveryJob    JobType = "discovery"
	DriftDetection  JobType = "drift_detection"
	Remediation     JobType = "remediation"
	StateAnalysis   JobType = "state_analysis"
	BulkOperation   JobType = "bulk_operation"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusRunning    JobStatus = "running"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
	StatusCancelled  JobStatus = "cancelled"
)

// JobPriority represents job priority
type JobPriority int

const (
	PriorityLow    JobPriority = 0
	PriorityNormal JobPriority = 1
	PriorityHigh   JobPriority = 2
	PriorityUrgent JobPriority = 3
)

// Job represents a job in the queue
type Job struct {
	ID          string                 `json:"id"`
	Type        JobType                `json:"type"`
	Status      JobStatus              `json:"status"`
	Priority    JobPriority            `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Data        interface{}            `json:"data"`
	Result      interface{}            `json:"result,omitempty"`
	Error       error                  `json:"error,omitempty"`
	Progress    int                    `json:"progress"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	cancel      context.CancelFunc
}

// Queue manages job execution
type Queue struct {
	jobs       map[string]*Job
	pending    []*Job
	running    map[string]*Job
	completed  map[string]*Job
	mu         sync.RWMutex
	workers    int
	workerPool chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
	handlers   map[JobType]JobHandler
	persistence JobPersistence
	metrics    *QueueMetrics
}

// JobHandler processes a specific type of job
type JobHandler func(ctx context.Context, job *Job) error

// JobPersistence handles job persistence
type JobPersistence interface {
	Save(job *Job) error
	Load(id string) (*Job, error)
	LoadAll() ([]*Job, error)
	Delete(id string) error
}

// QueueMetrics tracks queue metrics
type QueueMetrics struct {
	TotalJobs       int64
	PendingJobs     int
	RunningJobs     int
	CompletedJobs   int
	FailedJobs      int
	AverageWaitTime time.Duration
	AverageRunTime  time.Duration
}

// NewQueue creates a new job queue
func NewQueue(workers int, persistence JobPersistence) *Queue {
	ctx, cancel := context.WithCancel(context.Background())
	
	q := &Queue{
		jobs:        make(map[string]*Job),
		pending:     make([]*Job, 0),
		running:     make(map[string]*Job),
		completed:   make(map[string]*Job),
		workers:     workers,
		workerPool:  make(chan struct{}, workers),
		ctx:         ctx,
		cancel:      cancel,
		handlers:    make(map[JobType]JobHandler),
		persistence: persistence,
		metrics:     &QueueMetrics{},
	}

	// Initialize worker pool
	for i := 0; i < workers; i++ {
		q.workerPool <- struct{}{}
	}

	// Load persisted jobs if available
	if persistence != nil {
		q.loadPersistedJobs()
	}

	// Start processing loop
	go q.processLoop()

	return q
}

// RegisterHandler registers a handler for a job type
func (q *Queue) RegisterHandler(jobType JobType, handler JobHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[jobType] = handler
}

// Enqueue adds a job to the queue
func (q *Queue) Enqueue(job *Job) error {
	if job.ID == "" {
		job.ID = generateJobID()
	}
	
	if job.Priority == 0 {
		job.Priority = PriorityNormal
	}

	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}

	job.Status = StatusPending
	job.CreatedAt = time.Now()

	q.mu.Lock()
	q.jobs[job.ID] = job
	q.pending = append(q.pending, job)
	q.sortPendingJobs()
	q.metrics.PendingJobs = len(q.pending)
	q.metrics.TotalJobs++
	q.mu.Unlock()

	// Persist job if persistence is enabled
	if q.persistence != nil {
		if err := q.persistence.Save(job); err != nil {
			return fmt.Errorf("failed to persist job: %w", err)
		}
	}

	return nil
}

// processLoop processes jobs from the queue
func (q *Queue) processLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			q.processNextJob()
		}
	}
}

// processNextJob processes the next available job
func (q *Queue) processNextJob() {
	// Check if worker is available
	select {
	case <-q.workerPool:
		// Worker available
	default:
		// No workers available
		return
	}

	// Get next job
	q.mu.Lock()
	if len(q.pending) == 0 {
		// Return worker to pool
		q.workerPool <- struct{}{}
		q.mu.Unlock()
		return
	}

	job := q.pending[0]
	q.pending = q.pending[1:]
	q.running[job.ID] = job
	q.metrics.PendingJobs = len(q.pending)
	q.metrics.RunningJobs = len(q.running)
	q.mu.Unlock()

	// Process job in goroutine
	go q.executeJob(job)
}

// executeJob executes a single job
func (q *Queue) executeJob(job *Job) {
	defer func() {
		// Return worker to pool
		q.workerPool <- struct{}{}
	}()

	// Create job context
	ctx, cancel := context.WithTimeout(q.ctx, 30*time.Minute)
	job.cancel = cancel
	defer cancel()

	// Update job status
	now := time.Now()
	job.StartedAt = &now
	job.Status = StatusRunning
	q.UpdateJob(job)

	// Get handler for job type
	q.mu.RLock()
	handler, exists := q.handlers[job.Type]
	q.mu.RUnlock()

	if !exists {
		job.Status = StatusFailed
		job.Error = fmt.Errorf("no handler registered for job type: %s", job.Type)
		q.completeJob(job)
		return
	}

	// Execute handler
	err := handler(ctx, job)

	if err != nil {
		job.RetryCount++
		if job.RetryCount < job.MaxRetries {
			// Retry job
			job.Status = StatusPending
			job.Error = nil
			q.mu.Lock()
			q.pending = append(q.pending, job)
			delete(q.running, job.ID)
			q.sortPendingJobs()
			q.mu.Unlock()
			return
		}
		
		job.Status = StatusFailed
		job.Error = err
	} else {
		job.Status = StatusCompleted
	}

	q.completeJob(job)
}

// completeJob marks a job as completed
func (q *Queue) completeJob(job *Job) {
	now := time.Now()
	job.CompletedAt = &now

	q.mu.Lock()
	delete(q.running, job.ID)
	q.completed[job.ID] = job
	
	if job.Status == StatusCompleted {
		q.metrics.CompletedJobs++
	} else {
		q.metrics.FailedJobs++
	}
	q.metrics.RunningJobs = len(q.running)
	
	// Calculate average run time
	if job.StartedAt != nil {
		runTime := job.CompletedAt.Sub(*job.StartedAt)
		q.metrics.AverageRunTime = (q.metrics.AverageRunTime + runTime) / 2
	}
	q.mu.Unlock()

	// Persist updated job
	if q.persistence != nil {
		q.persistence.Save(job)
	}
}

// GetJob returns a job by ID
func (q *Queue) GetJob(id string) (*Job, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	job, exists := q.jobs[id]
	if !exists {
		// Try loading from persistence
		if q.persistence != nil {
			return q.persistence.Load(id)
		}
		return nil, fmt.Errorf("job not found: %s", id)
	}

	return job, nil
}

// UpdateJob updates a job's status and progress
func (q *Queue) UpdateJob(job *Job) error {
	q.mu.Lock()
	q.jobs[job.ID] = job
	q.mu.Unlock()

	// Persist updated job
	if q.persistence != nil {
		return q.persistence.Save(job)
	}

	return nil
}

// CancelJob cancels a running job
func (q *Queue) CancelJob(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	job, exists := q.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	if job.Status != StatusRunning && job.Status != StatusPending {
		return fmt.Errorf("job cannot be cancelled, status: %s", job.Status)
	}

	// Cancel job context if running
	if job.cancel != nil {
		job.cancel()
	}

	job.Status = StatusCancelled
	now := time.Now()
	job.CompletedAt = &now

	// Remove from pending or running
	if job.Status == StatusPending {
		for i, j := range q.pending {
			if j.ID == id {
				q.pending = append(q.pending[:i], q.pending[i+1:]...)
				break
			}
		}
	} else {
		delete(q.running, job.ID)
	}

	q.completed[job.ID] = job

	// Persist cancelled job
	if q.persistence != nil {
		q.persistence.Save(job)
	}

	return nil
}

// GetQueueStatus returns the current queue status
func (q *Queue) GetQueueStatus() map[string]interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return map[string]interface{}{
		"pending":   len(q.pending),
		"running":   len(q.running),
		"completed": len(q.completed),
		"workers":   q.workers,
		"metrics":   q.metrics,
	}
}

// GetPendingJobs returns all pending jobs
func (q *Queue) GetPendingJobs() []*Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	jobs := make([]*Job, len(q.pending))
	copy(jobs, q.pending)
	return jobs
}

// GetRunningJobs returns all running jobs
func (q *Queue) GetRunningJobs() []*Job {
	q.mu.RLock()
	defer q.mu.RUnlock()

	jobs := make([]*Job, 0, len(q.running))
	for _, job := range q.running {
		jobs = append(jobs, job)
	}
	return jobs
}

// sortPendingJobs sorts pending jobs by priority and creation time
func (q *Queue) sortPendingJobs() {
	// Simple priority queue implementation
	// Higher priority jobs run first, then by creation time
	for i := 0; i < len(q.pending)-1; i++ {
		for j := i + 1; j < len(q.pending); j++ {
			if q.pending[j].Priority > q.pending[i].Priority ||
				(q.pending[j].Priority == q.pending[i].Priority && 
				 q.pending[j].CreatedAt.Before(q.pending[i].CreatedAt)) {
				q.pending[i], q.pending[j] = q.pending[j], q.pending[i]
			}
		}
	}
}

// loadPersistedJobs loads jobs from persistence
func (q *Queue) loadPersistedJobs() {
	jobs, err := q.persistence.LoadAll()
	if err != nil {
		return
	}

	for _, job := range jobs {
		q.jobs[job.ID] = job
		
		switch job.Status {
		case StatusPending:
			q.pending = append(q.pending, job)
		case StatusRunning:
			// Reset running jobs to pending
			job.Status = StatusPending
			q.pending = append(q.pending, job)
		case StatusCompleted, StatusFailed, StatusCancelled:
			q.completed[job.ID] = job
		}
	}

	q.sortPendingJobs()
}

// Shutdown gracefully shuts down the queue
func (q *Queue) Shutdown(ctx context.Context) error {
	q.cancel()

	// Wait for running jobs to complete or timeout
	timeout := time.NewTimer(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer timeout.Stop()
	defer ticker.Stop()

	for {
		select {
		case <-timeout.C:
			return fmt.Errorf("shutdown timeout")
		case <-ticker.C:
			q.mu.RLock()
			runningCount := len(q.running)
			q.mu.RUnlock()
			
			if runningCount == 0 {
				return nil
			}
		}
	}
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job-%d-%s", time.Now().Unix(), generateRandomString(8))
}

// generateRandomString generates a random string
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}