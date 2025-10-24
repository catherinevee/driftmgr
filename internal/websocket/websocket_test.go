package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketService(t *testing.T) {
	// Create WebSocket service
	service := NewService()
	defer service.Stop(context.Background())

	// Test service creation
	if service == nil {
		t.Fatal("WebSocket service should not be nil")
	}

	if service.hub == nil {
		t.Fatal("WebSocket hub should not be nil")
	}

	if service.handlers == nil {
		t.Fatal("WebSocket handlers should not be nil")
	}
}

func TestWebSocketConnection(t *testing.T) {
	// Create WebSocket service
	service := NewService()
	defer service.Stop(context.Background())

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(service.GetHandlers().HandleWebSocket))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Test connection
	if conn == nil {
		t.Fatal("WebSocket connection should not be nil")
	}
}

func TestWebSocketMessageHandling(t *testing.T) {
	// Create WebSocket service
	service := NewService()
	defer service.Stop(context.Background())

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(service.GetHandlers().HandleWebSocket))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Test message sending
	testMessage := map[string]interface{}{
		"type": "test",
		"data": "test data",
	}

	err = conn.WriteJSON(testMessage)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Test message receiving
	var receivedMessage map[string]interface{}
	err = conn.ReadJSON(&receivedMessage)
	if err != nil {
		t.Fatalf("Failed to receive message: %v", err)
	}

	// Verify message type
	if receivedMessage["type"] != "connection_established" {
		t.Errorf("Expected connection_established, got %v", receivedMessage["type"])
	}
}

func TestWebSocketBroadcast(t *testing.T) {
	// Create WebSocket service
	service := NewService()
	defer service.Stop(context.Background())

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(service.GetHandlers().HandleWebSocket))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect first client
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect first client: %v", err)
	}
	defer conn1.Close()

	// Connect second client
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect second client: %v", err)
	}
	defer conn2.Close()

	// Wait for connections to be established
	time.Sleep(100 * time.Millisecond)

	// Broadcast a message
	testData := map[string]interface{}{
		"message": "test broadcast",
	}
	service.BroadcastDriftDetection(testData)

	// Read connection_established messages first
	var message1, message2 map[string]interface{}

	// Set read deadline
	conn1.SetReadDeadline(time.Now().Add(1 * time.Second))
	conn2.SetReadDeadline(time.Now().Add(1 * time.Second))

	// Read connection_established from first client
	err = conn1.ReadJSON(&message1)
	if err != nil {
		t.Fatalf("Failed to read from first client: %v", err)
	}

	// Read connection_established from second client
	err = conn2.ReadJSON(&message2)
	if err != nil {
		t.Fatalf("Failed to read from second client: %v", err)
	}

	// Now read the broadcast messages
	conn1.SetReadDeadline(time.Now().Add(1 * time.Second))
	conn2.SetReadDeadline(time.Now().Add(1 * time.Second))

	// Read broadcast from first client
	err = conn1.ReadJSON(&message1)
	if err != nil {
		t.Fatalf("Failed to read broadcast from first client: %v", err)
	}

	// Read broadcast from second client
	err = conn2.ReadJSON(&message2)
	if err != nil {
		t.Fatalf("Failed to read broadcast from second client: %v", err)
	}

	// Verify both clients received the broadcast
	if message1["type"] != "drift_detection" {
		t.Errorf("First client: Expected drift_detection, got %v", message1["type"])
	}

	if message2["type"] != "drift_detection" {
		t.Errorf("Second client: Expected drift_detection, got %v", message2["type"])
	}
}

func TestWebSocketConnectionStats(t *testing.T) {
	// Create WebSocket service
	service := NewService()
	defer service.Stop(context.Background())

	// Test initial stats
	stats := service.GetConnectionStats()
	if stats["total_connections"] != 0 {
		t.Errorf("Expected 0 total connections, got %v", stats["total_connections"])
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(service.GetHandlers().HandleWebSocket))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect client
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Wait for connection to be established
	time.Sleep(100 * time.Millisecond)

	// Test stats after connection
	stats = service.GetConnectionStats()
	if stats["total_connections"] != 1 {
		t.Errorf("Expected 1 total connection, got %v", stats["total_connections"])
	}
}

func TestWebSocketMessageTypes(t *testing.T) {
	// Create WebSocket service
	service := NewService()
	defer service.Stop(context.Background())

	// Test different message types
	testData := map[string]interface{}{
		"test": "data",
	}

	// Test drift detection broadcast
	service.BroadcastDriftDetection(testData)

	// Test remediation update broadcast
	service.BroadcastRemediationUpdate(testData)

	// Test resource update broadcast
	service.BroadcastResourceUpdate(testData)

	// Test state update broadcast
	service.BroadcastStateUpdate(testData)

	// Test backend update broadcast
	service.BroadcastBackendUpdate(testData)

	// Test system alert broadcast
	service.BroadcastSystemAlert(testData)

	// Test user-specific message
	service.SendToUser("test-user", "test-message", testData)

	// Test admin broadcast
	service.BroadcastToAdmins("admin-message", testData)

	// All methods should execute without error
}

func TestWebSocketHub(t *testing.T) {
	// Create hub
	hub := NewHub()

	// Test hub creation
	if hub == nil {
		t.Fatal("Hub should not be nil")
	}

	// Test initial client count
	if hub.GetClientCount() != 0 {
		t.Errorf("Expected 0 clients, got %d", hub.GetClientCount())
	}

	// Start hub in goroutine
	go hub.Run()

	// Wait a bit for hub to start
	time.Sleep(10 * time.Millisecond)

	// Test broadcast to empty hub (should not block)
	select {
	case hub.broadcast <- []byte("test message"):
		// Message sent successfully
	case <-time.After(100 * time.Millisecond):
		// Timeout - this is expected for empty hub
	}

	// Test broadcast to user (should not block)
	select {
	case hub.broadcast <- []byte("test message"):
		// Message sent successfully
	case <-time.After(100 * time.Millisecond):
		// Timeout - this is expected for empty hub
	}

	// All operations should complete without error
}

func TestWebSocketClient(t *testing.T) {
	// Create hub
	hub := NewHub()

	// Create mock client
	client := &Client{
		send:        make(chan []byte, 256),
		hub:         hub,
		userID:      "test-user",
		roles:       []string{"user"},
		connectedAt: time.Now(),
	}

	// Test client creation
	if client == nil {
		t.Fatal("Client should not be nil")
	}

	// Test role checking
	if !client.HasRole("user") {
		t.Error("Client should have user role")
	}

	if client.HasRole("admin") {
		t.Error("Client should not have admin role")
	}

	if client.IsAdmin() {
		t.Error("Client should not be admin")
	}

	// Test connection duration
	duration := client.GetConnectionDuration()
	if duration < 0 {
		t.Error("Connection duration should be positive")
	}

	// Test message sending
	message := NewMessage("test", "test data")
	err := client.SendMessage(message)
	if err != nil {
		t.Errorf("Failed to send message: %v", err)
	}
}

func TestWebSocketMessage(t *testing.T) {
	// Test message creation
	message := NewMessage("test-type", "test-data")
	if message.Type != "test-type" {
		t.Errorf("Expected test-type, got %s", message.Type)
	}

	if message.Data != "test-data" {
		t.Errorf("Expected test-data, got %v", message.Data)
	}

	// Test user message creation
	userMessage := NewUserMessage("user-type", "user-data", "user-id")
	if userMessage.Type != "user-type" {
		t.Errorf("Expected user-type, got %s", userMessage.Type)
	}

	if userMessage.UserID != "user-id" {
		t.Errorf("Expected user-id, got %s", userMessage.UserID)
	}

	// Test message marshaling
	data, err := json.Marshal(message)
	if err != nil {
		t.Errorf("Failed to marshal message: %v", err)
	}

	if len(data) == 0 {
		t.Error("Marshaled message should not be empty")
	}
}
