package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/catherinevee/driftmgr/internal/auth"
)

// WebSocketHandlers handles WebSocket connections and messages
type WebSocketHandlers struct {
	hub *Hub
}

// NewWebSocketHandlers creates a new WebSocket handlers instance
func NewWebSocketHandlers(hub *Hub) *WebSocketHandlers {
	return &WebSocketHandlers{
		hub: hub,
	}
}

// HandleWebSocket handles WebSocket connections
func (h *WebSocketHandlers) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Extract user information from context (if authenticated)
	userID, _ := auth.GetUserIDFromContext(r.Context())
	roles, _ := auth.GetRolesFromContext(r.Context())

	// Create client
	client := &Client{
		conn:        conn,
		send:        make(chan []byte, 256),
		hub:         h.hub,
		userID:      userID,
		roles:       roles,
		connectedAt: time.Now(),
	}

	// Register client with hub
	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()

	// Send welcome message
	welcomeMsg := NewMessage("connection_established", map[string]interface{}{
		"message": "Connected to DriftMgr WebSocket",
		"user_id": userID,
		"roles":   roles,
	})
	client.SendMessage(welcomeMsg)
}

// BroadcastDriftDetection sends a drift detection update to all connected clients
func (h *WebSocketHandlers) BroadcastDriftDetection(driftData interface{}) {
	message := NewMessage("drift_detection", driftData)
	data, _ := json.Marshal(message)
	h.hub.Broadcast(data)
}

// BroadcastRemediationUpdate sends a remediation update to all connected clients
func (h *WebSocketHandlers) BroadcastRemediationUpdate(remediationData interface{}) {
	message := NewMessage("remediation_update", remediationData)
	data, _ := json.Marshal(message)
	h.hub.Broadcast(data)
}

// BroadcastResourceUpdate sends a resource update to all connected clients
func (h *WebSocketHandlers) BroadcastResourceUpdate(resourceData interface{}) {
	message := NewMessage("resource_update", resourceData)
	data, _ := json.Marshal(message)
	h.hub.Broadcast(data)
}

// BroadcastStateUpdate sends a state management update to all connected clients
func (h *WebSocketHandlers) BroadcastStateUpdate(stateData interface{}) {
	message := NewMessage("state_update", stateData)
	data, _ := json.Marshal(message)
	h.hub.Broadcast(data)
}

// BroadcastBackendUpdate sends a backend discovery update to all connected clients
func (h *WebSocketHandlers) BroadcastBackendUpdate(backendData interface{}) {
	message := NewMessage("backend_update", backendData)
	data, _ := json.Marshal(message)
	h.hub.Broadcast(data)
}

// BroadcastSystemAlert sends a system alert to all connected clients
func (h *WebSocketHandlers) BroadcastSystemAlert(alertData interface{}) {
	message := NewMessage("system_alert", alertData)
	data, _ := json.Marshal(message)
	h.hub.Broadcast(data)
}

// SendToUser sends a message to a specific user
func (h *WebSocketHandlers) SendToUser(userID string, messageType string, data interface{}) {
	message := NewUserMessage(messageType, data, userID)
	messageData, _ := json.Marshal(message)
	h.hub.BroadcastToUser(userID, messageData)
}

// BroadcastToAdmins sends a message to all admin users
func (h *WebSocketHandlers) BroadcastToAdmins(messageType string, data interface{}) {
	message := NewMessage(messageType, data)
	messageData, _ := json.Marshal(message)

	// Send to all admin clients
	h.hub.mu.RLock()
	for client := range h.hub.clients {
		if client.IsAdmin() {
			select {
			case client.send <- messageData:
			default:
				close(client.send)
				delete(h.hub.clients, client)
			}
		}
	}
	h.hub.mu.RUnlock()
}

// GetConnectionStats returns statistics about WebSocket connections
func (h *WebSocketHandlers) GetConnectionStats() map[string]interface{} {
	h.hub.mu.RLock()
	defer h.hub.mu.RUnlock()

	stats := map[string]interface{}{
		"total_connections":     len(h.hub.clients),
		"admin_connections":     0,
		"user_connections":      0,
		"anonymous_connections": 0,
	}

	for client := range h.hub.clients {
		if client.userID == "" {
			stats["anonymous_connections"] = stats["anonymous_connections"].(int) + 1
		} else if client.IsAdmin() {
			stats["admin_connections"] = stats["admin_connections"].(int) + 1
		} else {
			stats["user_connections"] = stats["user_connections"].(int) + 1
		}
	}

	return stats
}
