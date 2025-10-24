package websocket

import (
	"context"
	"encoding/json"
	"log"
	"time"
)

// Service manages WebSocket connections and real-time events
type Service struct {
	hub      *Hub
	handlers *WebSocketHandlers
}

// NewService creates a new WebSocket service
func NewService() *Service {
	hub := NewHub()
	handlers := NewWebSocketHandlers(hub)

	service := &Service{
		hub:      hub,
		handlers: handlers,
	}

	// Start the hub
	go hub.Run()

	return service
}

// GetHandlers returns the WebSocket handlers
func (s *Service) GetHandlers() *WebSocketHandlers {
	return s.handlers
}

// GetHub returns the WebSocket hub
func (s *Service) GetHub() *Hub {
	return s.hub
}

// Start starts the WebSocket service
func (s *Service) Start(ctx context.Context) error {
	log.Println("WebSocket service started")

	// Start background tasks
	go s.startHeartbeat(ctx)
	go s.startStatsBroadcast(ctx)

	return nil
}

// Stop stops the WebSocket service
func (s *Service) Stop(ctx context.Context) error {
	log.Println("WebSocket service stopped")
	return nil
}

// startHeartbeat sends periodic heartbeat messages to keep connections alive
func (s *Service) startHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			heartbeat := NewMessage("heartbeat", map[string]interface{}{
				"timestamp":   time.Now().Unix(),
				"server_time": time.Now().Format(time.RFC3339),
			})

			data, err := json.Marshal(heartbeat)
			if err != nil {
				log.Printf("Failed to marshal heartbeat: %v", err)
				continue
			}

			s.hub.Broadcast(data)
		}
	}
}

// startStatsBroadcast sends periodic connection statistics
func (s *Service) startStatsBroadcast(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := s.handlers.GetConnectionStats()
			statsMsg := NewMessage("connection_stats", stats)

			data, err := json.Marshal(statsMsg)
			if err != nil {
				log.Printf("Failed to marshal stats: %v", err)
				continue
			}

			s.hub.Broadcast(data)
		}
	}
}

// BroadcastDriftDetection broadcasts drift detection results
func (s *Service) BroadcastDriftDetection(driftData interface{}) {
	s.handlers.BroadcastDriftDetection(driftData)
}

// BroadcastRemediationUpdate broadcasts remediation updates
func (s *Service) BroadcastRemediationUpdate(remediationData interface{}) {
	s.handlers.BroadcastRemediationUpdate(remediationData)
}

// BroadcastResourceUpdate broadcasts resource updates
func (s *Service) BroadcastResourceUpdate(resourceData interface{}) {
	s.handlers.BroadcastResourceUpdate(resourceData)
}

// BroadcastStateUpdate broadcasts state management updates
func (s *Service) BroadcastStateUpdate(stateData interface{}) {
	s.handlers.BroadcastStateUpdate(stateData)
}

// BroadcastBackendUpdate broadcasts backend discovery updates
func (s *Service) BroadcastBackendUpdate(backendData interface{}) {
	s.handlers.BroadcastBackendUpdate(backendData)
}

// BroadcastSystemAlert broadcasts system alerts
func (s *Service) BroadcastSystemAlert(alertData interface{}) {
	s.handlers.BroadcastSystemAlert(alertData)
}

// SendToUser sends a message to a specific user
func (s *Service) SendToUser(userID string, messageType string, data interface{}) {
	s.handlers.SendToUser(userID, messageType, data)
}

// BroadcastToAdmins broadcasts a message to all admin users
func (s *Service) BroadcastToAdmins(messageType string, data interface{}) {
	s.handlers.BroadcastToAdmins(messageType, data)
}

// GetConnectionStats returns current connection statistics
func (s *Service) GetConnectionStats() map[string]interface{} {
	return s.handlers.GetConnectionStats()
}
