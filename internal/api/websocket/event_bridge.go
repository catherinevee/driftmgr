package websocket

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/events"
)

// EventBridge connects the event bus to WebSocket clients
type EventBridge struct {
	eventBus    *events.EventBus
	wsServer    *EnhancedDashboardServer
	mu          sync.RWMutex
	active      bool
	subscribers []*events.Subscription
}

// NewEventBridge creates a new event bridge
func NewEventBridge(eventBus *events.EventBus, wsServer *EnhancedDashboardServer) *EventBridge {
	return &EventBridge{
		eventBus:    eventBus,
		wsServer:    wsServer,
		subscribers: make([]*events.Subscription, 0),
	}
}

// Start begins bridging events to WebSocket clients
func (eb *EventBridge) Start() error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.active {
		return fmt.Errorf("event bridge already active")
	}

	// Subscribe to all discovery events
	discoveryEvents := []events.EventType{
		events.DiscoveryStarted,
		events.DiscoveryProgress,
		events.DiscoveryCompleted,
		events.DiscoveryFailed,
	}

	discoverySub := eb.eventBus.SubscribeToTypes(discoveryEvents, func(event events.Event) {
		eb.handleDiscoveryEvent(event)
	})
	eb.subscribers = append(eb.subscribers, discoverySub)

	// Subscribe to drift detection events
	driftEvents := []events.EventType{
		events.DriftDetectionStarted,
		events.DriftDetectionCompleted,
		events.DriftDetectionFailed,
	}

	driftSub := eb.eventBus.SubscribeToTypes(driftEvents, func(event events.Event) {
		eb.handleDriftEvent(event)
	})
	eb.subscribers = append(eb.subscribers, driftSub)

	// Subscribe to remediation events
	remediationEvents := []events.EventType{
		events.RemediationStarted,
		events.RemediationCompleted,
		events.RemediationFailed,
	}

	remediationSub := eb.eventBus.SubscribeToTypes(remediationEvents, func(event events.Event) {
		eb.handleRemediationEvent(event)
	})
	eb.subscribers = append(eb.subscribers, remediationSub)

	// Subscribe to job events
	jobEvents := []events.EventType{
		events.JobCreated,
		events.JobStarted,
		events.JobCompleted,
		events.JobFailed,
	}

	jobSub := eb.eventBus.SubscribeToTypes(jobEvents, func(event events.Event) {
		eb.handleJobEvent(event)
	})
	eb.subscribers = append(eb.subscribers, jobSub)

	// Subscribe to resource events
	resourceEvents := []events.EventType{
		events.ResourceCreated,
		events.ResourceUpdated,
		events.ResourceDeleted,
	}

	resourceSub := eb.eventBus.SubscribeToTypes(resourceEvents, func(event events.Event) {
		eb.handleResourceEvent(event)
	})
	eb.subscribers = append(eb.subscribers, resourceSub)

	// Subscribe to state events
	stateEvents := []events.EventType{
		events.StateImported,
		events.StateAnalyzed,
		events.StateDeleted,
	}

	stateSub := eb.eventBus.SubscribeToTypes(stateEvents, func(event events.Event) {
		eb.handleStateEvent(event)
	})
	eb.subscribers = append(eb.subscribers, stateSub)

	eb.active = true
	return nil
}

// Stop stops the event bridge
func (eb *EventBridge) Stop() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if !eb.active {
		return
	}

	// Unsubscribe from all events
	for _, sub := range eb.subscribers {
		eb.eventBus.Unsubscribe(sub)
	}

	eb.subscribers = nil
	eb.active = false
}

// handleDiscoveryEvent processes discovery events
func (eb *EventBridge) handleDiscoveryEvent(event events.Event) {
	wsMessage := map[string]interface{}{
		"type":      "discovery_update",
		"timestamp": event.Timestamp,
		"data": map[string]interface{}{
			"event":    string(event.Type),
			"provider": event.Data["provider"],
			"region":   event.Data["region"],
			"progress": event.Data["progress"],
			"message":  event.Data["message"],
			"stats":    event.Data["stats"],
		},
	}

	// Update job if job ID is present
	if jobID, ok := event.Data["job_id"].(string); ok {
		if job, exists := eb.wsServer.jobManager.GetJob(jobID); exists {
			updates := map[string]interface{}{}

			switch event.Type {
			case events.DiscoveryStarted:
				updates["status"] = "running"
				updates["progress"] = 0
			case events.DiscoveryProgress:
				if progress, ok := event.Data["progress"].(int); ok {
					updates["progress"] = progress
				}
			case events.DiscoveryCompleted:
				updates["status"] = "completed"
				updates["progress"] = 100
				updates["result"] = event.Data["resources"]
			case events.DiscoveryFailed:
				updates["status"] = "failed"
				updates["error"] = event.Data["error"]
			}

			eb.wsServer.jobManager.UpdateJob(jobID, updates)
			wsMessage["job"] = job
		}
	}

	// Broadcast to all connected clients
	eb.wsServer.broadcast <- wsMessage
}

// handleDriftEvent processes drift detection events
func (eb *EventBridge) handleDriftEvent(event events.Event) {
	wsMessage := map[string]interface{}{
		"type":      "drift_update",
		"timestamp": event.Timestamp,
		"data": map[string]interface{}{
			"event":       string(event.Type),
			"drift_count": event.Data["drift_count"],
			"resources":   event.Data["resources"],
			"severity":    event.Data["severity"],
			"message":     event.Data["message"],
		},
	}

	// Store drifts in data store for later retrieval
	if drifts, ok := event.Data["drifts"]; ok {
		eb.wsServer.dataStore.Set("drifts", drifts)
		eb.wsServer.dataStore.Set("last_drift_check", time.Now())
	}

	// Update drift statistics
	if event.Type == events.DriftDetectionCompleted {
		stats := map[string]interface{}{
			"total_drifts":    event.Data["drift_count"],
			"critical_drifts": event.Data["critical_count"],
			"warning_drifts":  event.Data["warning_count"],
			"info_drifts":     event.Data["info_count"],
			"last_check":      time.Now(),
		}
		eb.wsServer.dataStore.Set("drift_stats", stats)
		wsMessage["stats"] = stats
	}

	eb.wsServer.broadcast <- wsMessage
}

// handleRemediationEvent processes remediation events
func (eb *EventBridge) handleRemediationEvent(event events.Event) {
	wsMessage := map[string]interface{}{
		"type":      "remediation_update",
		"timestamp": event.Timestamp,
		"data": map[string]interface{}{
			"event":       string(event.Type),
			"plan_id":     event.Data["plan_id"],
			"action":      event.Data["action"],
			"resource_id": event.Data["resource_id"],
			"progress":    event.Data["progress"],
			"status":      event.Data["status"],
			"message":     event.Data["message"],
		},
	}

	// Track remediation progress
	if planID, ok := event.Data["plan_id"].(string); ok {
		remediationKey := fmt.Sprintf("remediation_%s", planID)

		switch event.Type {
		case events.RemediationStarted:
			eb.wsServer.dataStore.Set(remediationKey, map[string]interface{}{
				"status":     "running",
				"started_at": time.Now(),
				"progress":   0,
			})
		case events.RemediationCompleted:
			if existing, ok := eb.wsServer.dataStore.Get(remediationKey); ok {
				if data, ok := existing.(map[string]interface{}); ok {
					data["status"] = "completed"
					data["completed_at"] = time.Now()
					data["progress"] = 100
					data["result"] = event.Data["result"]
					eb.wsServer.dataStore.Set(remediationKey, data)
				}
			}
		case events.RemediationFailed:
			if existing, ok := eb.wsServer.dataStore.Get(remediationKey); ok {
				if data, ok := existing.(map[string]interface{}); ok {
					data["status"] = "failed"
					data["failed_at"] = time.Now()
					data["error"] = event.Data["error"]
					eb.wsServer.dataStore.Set(remediationKey, data)
				}
			}
		}

		wsMessage["remediation_status"] = eb.wsServer.dataStore.data[remediationKey]
	}

	eb.wsServer.broadcast <- wsMessage
}

// handleJobEvent processes job events
func (eb *EventBridge) handleJobEvent(event events.Event) {
	jobID, _ := event.Data["job_id"].(string)

	wsMessage := map[string]interface{}{
		"type":      "job_update",
		"timestamp": event.Timestamp,
		"data": map[string]interface{}{
			"event":    string(event.Type),
			"job_id":   jobID,
			"job_type": event.Data["job_type"],
			"status":   event.Data["status"],
			"progress": event.Data["progress"],
			"message":  event.Data["message"],
		},
	}

	// Get full job details
	if jobID != "" {
		if job, exists := eb.wsServer.jobManager.GetJob(jobID); exists {
			wsMessage["job"] = job
		}
	}

	// Update job queue statistics
	jobStats := map[string]interface{}{
		"total_jobs":     len(eb.wsServer.jobManager.jobs),
		"pending_jobs":   0,
		"running_jobs":   0,
		"completed_jobs": 0,
		"failed_jobs":    0,
	}

	for _, job := range eb.wsServer.jobManager.jobs {
		switch job.Status {
		case "pending":
			jobStats["pending_jobs"] = jobStats["pending_jobs"].(int) + 1
		case "running":
			jobStats["running_jobs"] = jobStats["running_jobs"].(int) + 1
		case "completed":
			jobStats["completed_jobs"] = jobStats["completed_jobs"].(int) + 1
		case "failed":
			jobStats["failed_jobs"] = jobStats["failed_jobs"].(int) + 1
		}
	}

	wsMessage["job_stats"] = jobStats
	eb.wsServer.dataStore.Set("job_stats", jobStats)

	eb.wsServer.broadcast <- wsMessage
}

// handleResourceEvent processes resource events
func (eb *EventBridge) handleResourceEvent(event events.Event) {
	wsMessage := map[string]interface{}{
		"type":      "resource_update",
		"timestamp": event.Timestamp,
		"data": map[string]interface{}{
			"event":         string(event.Type),
			"resource_id":   event.Data["resource_id"],
			"resource_type": event.Data["resource_type"],
			"provider":      event.Data["provider"],
			"region":        event.Data["region"],
			"action":        event.Data["action"],
			"changes":       event.Data["changes"],
		},
	}

	// Update resource count
	resourceCount := eb.wsServer.dataStore.GetResourceCount()
	switch event.Type {
	case events.ResourceCreated:
		resourceCount++
	case events.ResourceDeleted:
		resourceCount--
	}
	eb.wsServer.dataStore.Set("resource_count", resourceCount)
	wsMessage["resource_count"] = resourceCount

	// Track resource changes for audit
	auditEntry := map[string]interface{}{
		"timestamp":     event.Timestamp,
		"event":         string(event.Type),
		"resource_id":   event.Data["resource_id"],
		"resource_type": event.Data["resource_type"],
		"user":          event.Data["user"],
		"source":        event.Source,
	}

	auditLog := []interface{}{}
	if existing, ok := eb.wsServer.dataStore.Get("audit_log"); ok {
		if log, ok := existing.([]interface{}); ok {
			auditLog = log
		}
	}
	auditLog = append(auditLog, auditEntry)

	// Keep only last 1000 entries
	if len(auditLog) > 1000 {
		auditLog = auditLog[len(auditLog)-1000:]
	}
	eb.wsServer.dataStore.Set("audit_log", auditLog)

	eb.wsServer.broadcast <- wsMessage
}

// handleStateEvent processes state management events
func (eb *EventBridge) handleStateEvent(event events.Event) {
	wsMessage := map[string]interface{}{
		"type":      "state_update",
		"timestamp": event.Timestamp,
		"data": map[string]interface{}{
			"event":      string(event.Type),
			"state_id":   event.Data["state_id"],
			"state_path": event.Data["state_path"],
			"provider":   event.Data["provider"],
			"resources":  event.Data["resource_count"],
			"message":    event.Data["message"],
		},
	}

	// Update state inventory
	if event.Type == events.StateImported {
		stateInfo := map[string]interface{}{
			"id":             event.Data["state_id"],
			"path":           event.Data["state_path"],
			"provider":       event.Data["provider"],
			"resource_count": event.Data["resource_count"],
			"imported_at":    time.Now(),
		}

		states := []interface{}{}
		if existing, ok := eb.wsServer.dataStore.Get("states"); ok {
			if s, ok := existing.([]interface{}); ok {
				states = s
			}
		}
		states = append(states, stateInfo)
		eb.wsServer.dataStore.Set("states", states)
		wsMessage["states"] = states
	}

	eb.wsServer.broadcast <- wsMessage
}

// BroadcastCustomMessage sends a custom message to all WebSocket clients
func (eb *EventBridge) BroadcastCustomMessage(messageType string, data interface{}) {
	message := map[string]interface{}{
		"type":      messageType,
		"timestamp": time.Now(),
		"data":      data,
	}

	if jsonData, err := json.Marshal(message); err == nil {
		eb.wsServer.broadcast <- jsonData
	}
}

// GetConnectedClients returns the number of connected WebSocket clients
func (eb *EventBridge) GetConnectedClients() int {
	eb.wsServer.clientsMux.RLock()
	defer eb.wsServer.clientsMux.RUnlock()
	return len(eb.wsServer.clients)
}

// IsActive returns whether the event bridge is active
func (eb *EventBridge) IsActive() bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return eb.active
}
