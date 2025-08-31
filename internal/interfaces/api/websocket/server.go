package websocket

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// EnhancedDashboardServer represents the WebSocket server
type EnhancedDashboardServer struct {
	clients    map[string]*WebSocketClient
	broadcast  chan interface{}
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	wsUpgrader websocket.Upgrader
	jobManager *JobManager
	dataStore  *DataStore
	clientsMux sync.RWMutex
	startTime  time.Time
	mu         sync.RWMutex
}

// NewServer creates a new WebSocket server
func NewServer() *EnhancedDashboardServer {
	return &EnhancedDashboardServer{
		clients:    make(map[string]*WebSocketClient),
		broadcast:  make(chan interface{}),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		jobManager: NewJobManager(),
		dataStore:  NewDataStore(),
		startTime:  time.Now(),
	}
}

// JobManager manages async jobs
type JobManager struct {
	jobs map[string]*Job
	mu   sync.RWMutex
}

// NewJobManager creates a new job manager
func NewJobManager() *JobManager {
	return &JobManager{
		jobs: make(map[string]*Job),
	}
}

// Job represents an async job
type Job struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	Progress  int                    `json:"progress"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// CreateJob creates a new job
func (jm *JobManager) CreateJob(jobType string) *Job {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	job := &Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Status:    "pending",
		Progress:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	jm.jobs[job.ID] = job
	return job
}

// GetJob retrieves a job by ID
func (jm *JobManager) GetJob(id string) (*Job, bool) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	job, exists := jm.jobs[id]
	return job, exists
}

// UpdateJob updates a job
func (jm *JobManager) UpdateJob(id string, updates map[string]interface{}) error {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	job, exists := jm.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	if status, ok := updates["status"].(string); ok {
		job.Status = status
	}
	if progress, ok := updates["progress"].(int); ok {
		job.Progress = progress
	}
	if result, ok := updates["result"]; ok {
		job.Result = result
	}
	if errMsg, ok := updates["error"].(string); ok {
		job.Error = errMsg
	}

	job.UpdatedAt = time.Now()
	return nil
}

// DataStore stores server data
type DataStore struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewDataStore creates a new data store
func NewDataStore() *DataStore {
	return &DataStore{
		data: make(map[string]interface{}),
	}
}

// Get retrieves data
func (ds *DataStore) Get(key string) (interface{}, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	val, exists := ds.data[key]
	return val, exists
}

// Set stores data
func (ds *DataStore) Set(key string, value interface{}) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.data[key] = value
}

// SetCredentialStatus stores credential status
func (ds *DataStore) SetCredentialStatus(status interface{}) {
	ds.Set("credential_status", status)
}

// GetResourceCount retrieves resource count
func (ds *DataStore) GetResourceCount() int {
	if val, exists := ds.Get("resource_count"); exists {
		if count, ok := val.(int); ok {
			return count
		}
	}
	return 0
}

// GetDrifts retrieves drifts
func (ds *DataStore) GetDrifts() []interface{} {
	if val, exists := ds.Get("drifts"); exists {
		if drifts, ok := val.([]interface{}); ok {
			return drifts
		}
	}
	return []interface{}{}
}

// removeClient removes a WebSocket client
func (s *EnhancedDashboardServer) removeClient(client *WebSocketClient) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()

	if _, exists := s.clients[client.id]; exists {
		delete(s.clients, client.id)
		close(client.send)
	}
}
