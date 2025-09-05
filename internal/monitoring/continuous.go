package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/shared/errors"
	"github.com/catherinevee/driftmgr/internal/providers"
)

// ContinuousMonitor provides real-time infrastructure monitoring
type ContinuousMonitor struct {
	providers      map[string]providers.CloudProvider
	webhookServer  *WebhookServer
	eventProcessor *EventProcessor
	changeDetector *ChangeDetector
	config         MonitorConfig
	mu             sync.RWMutex
	stopChan       chan struct{}
	wg             sync.WaitGroup
}

// MonitorConfig configures continuous monitoring
type MonitorConfig struct {
	WebhookPort       int
	PollingInterval   time.Duration
	AdaptivePolling   bool
	MinPollInterval   time.Duration
	MaxPollInterval   time.Duration
	EventBuffer       int
	EnableWebhooks    bool
	EnablePolling     bool
}

// CloudEvent represents a cloud provider event
type CloudEvent struct {
	ID           string                 `json:"id"`
	Source       string                 `json:"source"`
	Type         string                 `json:"type"`
	Time         time.Time              `json:"time"`
	Region       string                 `json:"region,omitempty"`
	Account      string                 `json:"account,omitempty"`
	Resource     string                 `json:"resource,omitempty"`
	Action       string                 `json:"action,omitempty"`
	Principal    string                 `json:"principal,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	ChangeType   ChangeType             `json:"change_type,omitempty"`
}

// ChangeType represents the type of change detected
type ChangeType string

const (
	ChangeTypeCreate ChangeType = "create"
	ChangeTypeUpdate ChangeType = "update"
	ChangeTypeDelete ChangeType = "delete"
	ChangeTypeDrift  ChangeType = "drift"
)

// NewContinuousMonitor creates a new continuous monitor
func NewContinuousMonitor(config MonitorConfig) *ContinuousMonitor {
	if config.PollingInterval == 0 {
		config.PollingInterval = 5 * time.Minute
	}
	if config.MinPollInterval == 0 {
		config.MinPollInterval = 1 * time.Minute
	}
	if config.MaxPollInterval == 0 {
		config.MaxPollInterval = 30 * time.Minute
	}
	if config.EventBuffer == 0 {
		config.EventBuffer = 1000
	}
	
	monitor := &ContinuousMonitor{
		providers:      make(map[string]providers.CloudProvider),
		config:         config,
		stopChan:       make(chan struct{}),
		eventProcessor: NewEventProcessor(config.EventBuffer),
		changeDetector: NewChangeDetector(),
	}
	
	if config.EnableWebhooks {
		monitor.webhookServer = NewWebhookServer(config.WebhookPort, monitor.handleWebhook)
	}
	
	return monitor
}

// Start begins continuous monitoring
func (m *ContinuousMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Start webhook server if enabled
	if m.config.EnableWebhooks && m.webhookServer != nil {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if err := m.webhookServer.Start(ctx); err != nil {
				fmt.Printf("Webhook server error: %v\n", err)
			}
		}()
	}
	
	// Start polling if enabled
	if m.config.EnablePolling {
		m.wg.Add(1)
		go m.pollingWorker(ctx)
	}
	
	// Start event processor
	m.wg.Add(1)
	go m.eventProcessor.Start(ctx)
	
	return nil
}

// Stop stops continuous monitoring
func (m *ContinuousMonitor) Stop() {
	close(m.stopChan)
	if m.webhookServer != nil {
		m.webhookServer.Stop()
	}
	m.eventProcessor.Stop()
	m.wg.Wait()
}

// RegisterProvider registers a cloud provider for monitoring
func (m *ContinuousMonitor) RegisterProvider(name string, provider providers.CloudProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[name] = provider
}

// pollingWorker performs periodic polling with adaptive intervals
func (m *ContinuousMonitor) pollingWorker(ctx context.Context) {
	defer m.wg.Done()
	
	currentInterval := m.config.PollingInterval
	lastChangeTime := time.Now()
	
	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			changes := m.pollProviders(ctx)
			
			if m.config.AdaptivePolling {
				// Adjust polling interval based on change frequency
				if len(changes) > 0 {
					lastChangeTime = time.Now()
					// Decrease interval when changes are detected
					currentInterval = m.decreaseInterval(currentInterval)
				} else if time.Since(lastChangeTime) > 30*time.Minute {
					// Increase interval when no changes for a while
					currentInterval = m.increaseInterval(currentInterval)
				}
				
				ticker.Reset(currentInterval)
			}
			
			// Process detected changes
			for _, event := range changes {
				m.eventProcessor.ProcessEvent(event)
			}
		}
	}
}

// pollProviders polls all registered providers for changes
func (m *ContinuousMonitor) pollProviders(ctx context.Context) []CloudEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var allEvents []CloudEvent
	
	for name, provider := range m.providers {
		events := m.pollProvider(ctx, name, provider)
		allEvents = append(allEvents, events...)
	}
	
	return allEvents
}

// pollProvider polls a single provider for changes
func (m *ContinuousMonitor) pollProvider(ctx context.Context, name string, provider providers.CloudProvider) []CloudEvent {
	// Get current state
	resources, err := provider.DiscoverResources(ctx)
	if err != nil {
		fmt.Printf("Error polling %s: %v\n", name, err)
		return nil
	}
	
	// Detect changes
	changes := m.changeDetector.DetectChanges(name, resources)
	
	// Convert to events
	var events []CloudEvent
	for _, change := range changes {
		events = append(events, CloudEvent{
			ID:         fmt.Sprintf("%s-%d", name, time.Now().UnixNano()),
			Source:     name,
			Type:       fmt.Sprintf("provider.%s.change", name),
			Time:       time.Now(),
			Resource:   change.ResourceID,
			ChangeType: change.Type,
			Details:    change.Details,
		})
	}
	
	return events
}

// handleWebhook processes incoming webhook events
func (m *ContinuousMonitor) handleWebhook(event CloudEvent) {
	// Validate and enrich event
	event.Time = time.Now()
	
	// Process event
	m.eventProcessor.ProcessEvent(event)
}

// decreaseInterval decreases polling interval for more frequent checks
func (m *ContinuousMonitor) decreaseInterval(current time.Duration) time.Duration {
	newInterval := time.Duration(float64(current) * 0.75)
	if newInterval < m.config.MinPollInterval {
		return m.config.MinPollInterval
	}
	return newInterval
}

// increaseInterval increases polling interval for less frequent checks
func (m *ContinuousMonitor) increaseInterval(current time.Duration) time.Duration {
	newInterval := time.Duration(float64(current) * 1.25)
	if newInterval > m.config.MaxPollInterval {
		return m.config.MaxPollInterval
	}
	return newInterval
}

// WebhookServer handles incoming webhook events
type WebhookServer struct {
	port     int
	handler  func(CloudEvent)
	server   *http.Server
	stopChan chan struct{}
}

// NewWebhookServer creates a new webhook server
func NewWebhookServer(port int, handler func(CloudEvent)) *WebhookServer {
	return &WebhookServer{
		port:     port,
		handler:  handler,
		stopChan: make(chan struct{}),
	}
}

// Start starts the webhook server
func (s *WebhookServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	
	// AWS EventBridge webhook
	mux.HandleFunc("/webhooks/aws/eventbridge", s.handleAWSEventBridge)
	
	// Azure Event Grid webhook
	mux.HandleFunc("/webhooks/azure/eventgrid", s.handleAzureEventGrid)
	
	// GCP Pub/Sub webhook
	mux.HandleFunc("/webhooks/gcp/pubsub", s.handleGCPPubSub)
	
	// Generic webhook endpoint
	mux.HandleFunc("/webhooks/generic", s.handleGeneric)
	
	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}
	
	fmt.Printf("Webhook server listening on port %d\n", s.port)
	
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Webhook server error: %v\n", err)
		}
	}()
	
	<-s.stopChan
	return s.server.Shutdown(ctx)
}

// Stop stops the webhook server
func (s *WebhookServer) Stop() {
	close(s.stopChan)
}

// handleAWSEventBridge handles AWS EventBridge webhooks
func (s *WebhookServer) handleAWSEventBridge(w http.ResponseWriter, r *http.Request) {
	var event struct {
		Version    string                 `json:"version"`
		ID         string                 `json:"id"`
		DetailType string                 `json:"detail-type"`
		Source     string                 `json:"source"`
		Account    string                 `json:"account"`
		Time       string                 `json:"time"`
		Region     string                 `json:"region"`
		Detail     map[string]interface{} `json:"detail"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Convert to CloudEvent
	eventTime, _ := time.Parse(time.RFC3339, event.Time)
	cloudEvent := CloudEvent{
		ID:       event.ID,
		Source:   "aws",
		Type:     event.DetailType,
		Time:     eventTime,
		Region:   event.Region,
		Account:  event.Account,
		Details:  event.Detail,
	}
	
	// Determine change type from detail
	if action, ok := event.Detail["eventName"].(string); ok {
		cloudEvent.Action = action
		cloudEvent.ChangeType = s.mapAWSActionToChangeType(action)
	}
	
	s.handler(cloudEvent)
	w.WriteHeader(http.StatusOK)
}

// handleAzureEventGrid handles Azure Event Grid webhooks
func (s *WebhookServer) handleAzureEventGrid(w http.ResponseWriter, r *http.Request) {
	var events []struct {
		ID          string                 `json:"id"`
		Topic       string                 `json:"topic"`
		Subject     string                 `json:"subject"`
		EventType   string                 `json:"eventType"`
		EventTime   string                 `json:"eventTime"`
		Data        map[string]interface{} `json:"data"`
		DataVersion string                 `json:"dataVersion"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Handle validation request
	if len(events) == 1 && events[0].EventType == "Microsoft.EventGrid.SubscriptionValidationEvent" {
		validationCode := events[0].Data["validationCode"]
		response := map[string]interface{}{
			"validationResponse": validationCode,
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Process events
	for _, event := range events {
		eventTime, _ := time.Parse(time.RFC3339, event.EventTime)
		cloudEvent := CloudEvent{
			ID:       event.ID,
			Source:   "azure",
			Type:     event.EventType,
			Time:     eventTime,
			Resource: event.Subject,
			Details:  event.Data,
		}
		
		cloudEvent.ChangeType = s.mapAzureEventToChangeType(event.EventType)
		s.handler(cloudEvent)
	}
	
	w.WriteHeader(http.StatusOK)
}

// handleGCPPubSub handles GCP Pub/Sub push webhooks
func (s *WebhookServer) handleGCPPubSub(w http.ResponseWriter, r *http.Request) {
	var message struct {
		Message struct {
			Data       string            `json:"data"`
			Attributes map[string]string `json:"attributes"`
			MessageID  string            `json:"messageId"`
		} `json:"message"`
		Subscription string `json:"subscription"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Decode message data (base64 encoded)
	var eventData map[string]interface{}
	if err := json.Unmarshal([]byte(message.Message.Data), &eventData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	cloudEvent := CloudEvent{
		ID:      message.Message.MessageID,
		Source:  "gcp",
		Type:    message.Message.Attributes["eventType"],
		Time:    time.Now(),
		Details: eventData,
	}
	
	s.handler(cloudEvent)
	w.WriteHeader(http.StatusOK)
}

// handleGeneric handles generic webhook events
func (s *WebhookServer) handleGeneric(w http.ResponseWriter, r *http.Request) {
	var event CloudEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	s.handler(event)
	w.WriteHeader(http.StatusOK)
}

// mapAWSActionToChangeType maps AWS API actions to change types
func (s *WebhookServer) mapAWSActionToChangeType(action string) ChangeType {
	switch {
	case contains(action, "Create"), contains(action, "Put"):
		return ChangeTypeCreate
	case contains(action, "Update"), contains(action, "Modify"):
		return ChangeTypeUpdate
	case contains(action, "Delete"), contains(action, "Remove"):
		return ChangeTypeDelete
	default:
		return ChangeTypeDrift
	}
}

// mapAzureEventToChangeType maps Azure events to change types
func (s *WebhookServer) mapAzureEventToChangeType(eventType string) ChangeType {
	switch {
	case contains(eventType, "Created"):
		return ChangeTypeCreate
	case contains(eventType, "Updated"):
		return ChangeTypeUpdate
	case contains(eventType, "Deleted"):
		return ChangeTypeDelete
	default:
		return ChangeTypeDrift
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}