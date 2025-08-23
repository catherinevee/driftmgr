package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/discovery"
	"github.com/catherinevee/driftmgr/internal/core/drift"
	"github.com/catherinevee/driftmgr/internal/core/remediation"
	"github.com/catherinevee/driftmgr/internal/infrastructure/storage"
)

// EnhancedDashboardServer represents the REST API server
type EnhancedDashboardServer struct {
	discoveryService  *discovery.Service
	driftDetector     *drift.Detector
	remediationEngine *remediation.Engine
	credManager       *CredentialManager
	jobManager        *JobManager
	dataStore         *DataStore
	storage           storage.Storage
	logger            *log.Logger
	broadcast         chan map[string]interface{}
	mu                sync.RWMutex
}

// CredentialManager manages cloud credentials
type CredentialManager struct {
	credentials map[string]map[string]interface{}
	mu          sync.RWMutex
}

// JobManager manages async jobs
type JobManager struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

// Job represents an async job
type Job struct {
	ID        string
	Type      string
	Status    string
	Progress  float64
	Result    interface{}
	Error     error
	StartTime time.Time
	EndTime   time.Time
}

// DataStore stores data for the dashboard
type DataStore struct {
	resources          []interface{}
	drifts             []interface{}
	credentialStatus   []interface{}
	remediationHistory []interface{}
	mu                 sync.RWMutex
}

// GetRemediationHistory returns remediation history
func (ds *DataStore) GetRemediationHistory() []interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.remediationHistory
}

// AddRemediationHistory adds a remediation history item
func (ds *DataStore) AddRemediationHistory(item interface{}) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.remediationHistory = append(ds.remediationHistory, item)
}

// GetDrifts returns drift items
func (ds *DataStore) GetDrifts() []interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.drifts
}

// NewServer creates a new server instance
func NewServer() *EnhancedDashboardServer {
	return &EnhancedDashboardServer{
		discoveryService:  discovery.NewService(),
		driftDetector:     drift.NewDetector(),
		remediationEngine: remediation.NewEngine(),
		credManager:       &CredentialManager{credentials: make(map[string]map[string]interface{})},
		jobManager:        &JobManager{jobs: make(map[string]*Job)},
		dataStore:         &DataStore{},
		storage:           storage.NewMemoryStorage(),
		logger:            log.New(os.Stdout, "[API] ", log.LstdFlags),
		broadcast:         make(chan map[string]interface{}, 100),
	}
}

// Helper methods for JobManager
func (jm *JobManager) CreateJob(jobType string) *Job {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	job := &Job{
		ID:        generateJobID(),
		Type:      jobType,
		Status:    "pending",
		StartTime: time.Now(),
	}
	jm.jobs[job.ID] = job
	return job
}

func (jm *JobManager) GetJob(id string) (*Job, bool) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	job, ok := jm.jobs[id]
	return job, ok
}

func (jm *JobManager) UpdateJob(id string, status string, progress float64) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	if job, ok := jm.jobs[id]; ok {
		job.Status = status
		job.Progress = progress
		if status == "completed" || status == "failed" {
			job.EndTime = time.Now()
		}
	}
}

func (jm *JobManager) ListJobs() []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	jobs := make([]*Job, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// Helper methods for DataStore
func (ds *DataStore) SetResources(resources []interface{}) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.resources = resources
}

func (ds *DataStore) GetResources() []interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.resources
}

func (ds *DataStore) SetDrifts(drifts []interface{}) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.drifts = drifts
}

func (ds *DataStore) SetCredentialStatus(status []interface{}) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.credentialStatus = status
}

func (ds *DataStore) GetCredentialStatus() []interface{} {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.credentialStatus
}

func (ds *DataStore) SetRemediationHistory(history []interface{}) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.remediationHistory = history
}

func (ds *DataStore) GetBaseline() []interface{} {
	// Return empty baseline for now
	return []interface{}{}
}

func (ds *DataStore) SetConfig(key string, value interface{}) {
	// Store config - simplified implementation
}

func (ds *DataStore) GetDriftHistory() []interface{} {
	// Return historical drifts - simplified
	return ds.GetDrifts()
}

// generateJobID generates a unique job ID
func generateJobID() string {
	return "job-" + time.Now().Format("20060102-150405")
}

// Credential Manager methods
func (cm *CredentialManager) IsConfigured(ctx context.Context, provider string) (bool, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	_, ok := cm.credentials[provider]
	return ok, nil
}

func (cm *CredentialManager) ValidateCredentials(ctx context.Context, provider string) (bool, error) {
	// Simple validation - just check if credentials exist
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	_, ok := cm.credentials[provider]
	return ok, nil
}

func (cm *CredentialManager) GetAccounts(ctx context.Context, provider string) ([]string, error) {
	// Return mock accounts
	return []string{"default-account"}, nil
}

func (cm *CredentialManager) GetRegions(ctx context.Context, provider string) ([]string, error) {
	// Return default regions
	switch provider {
	case "aws":
		return []string{"us-east-1", "us-west-2"}, nil
	case "azure":
		return []string{"eastus", "westus"}, nil
	default:
		return []string{}, nil
	}
}

func (cm *CredentialManager) TestProviderConfig(ctx context.Context, config *ProviderConfig) error {
	// Simple test - just check if credentials are provided
	if len(config.Credentials) == 0 {
		return fmt.Errorf("no credentials provided")
	}
	return nil
}

func (cm *CredentialManager) GetAccountInfoWithConfig(ctx context.Context, config *ProviderConfig) (map[string]interface{}, error) {
	// Get actual account info based on provider
	accountInfo := make(map[string]interface{})
	
	switch config.Provider {
	case "aws":
		// Get AWS account ID from STS or environment
		accountInfo["account"] = os.Getenv("AWS_ACCOUNT_ID")
		if accountInfo["account"] == "" {
			accountInfo["account"] = "aws-account"
		}
		if len(config.Regions) > 0 {
			accountInfo["region"] = config.Regions[0]
		} else {
			accountInfo["region"] = os.Getenv("AWS_DEFAULT_REGION")
			if accountInfo["region"] == "" {
				accountInfo["region"] = "us-east-1"
			}
		}
	case "azure":
		accountInfo["account"] = os.Getenv("AZURE_SUBSCRIPTION_ID")
		if accountInfo["account"] == "" {
			accountInfo["account"] = "azure-subscription"
		}
		if len(config.Regions) > 0 {
			accountInfo["region"] = config.Regions[0]
		} else {
			accountInfo["region"] = "eastus"
		}
	case "gcp":
		accountInfo["account"] = os.Getenv("GOOGLE_CLOUD_PROJECT")
		if accountInfo["account"] == "" {
			accountInfo["account"] = "gcp-project"
		}
		if len(config.Regions) > 0 {
			accountInfo["region"] = config.Regions[0]
		} else {
			accountInfo["region"] = "us-central1"
		}
	default:
		accountInfo["account"] = config.Provider + "-account"
		if len(config.Regions) > 0 {
			accountInfo["region"] = config.Regions[0]
		} else {
			accountInfo["region"] = "default-region"
		}
	}
	
	return accountInfo, nil
}

func (cm *CredentialManager) ConfigureProvider(ctx context.Context, config *ProviderConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.credentials[config.Provider] = config.Credentials
	return nil
}

func (cm *CredentialManager) SaveProviderConfig(ctx context.Context, config *ProviderConfig) error {
	// In production would persist to file/database
	return nil
}

func (cm *CredentialManager) GetProviderConfig(ctx context.Context, provider string) (*ProviderConfig, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	if creds, ok := cm.credentials[provider]; ok {
		return &ProviderConfig{
			Provider:    provider,
			Credentials: creds,
		}, nil
	}
	return nil, fmt.Errorf("provider not configured")
}

func (cm *CredentialManager) DeleteProviderConfig(ctx context.Context, provider string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.credentials, provider)
	return nil
}
