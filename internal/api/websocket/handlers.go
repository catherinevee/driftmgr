package websocket

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// WebSocketCommand represents a command from a WebSocket client
type WebSocketCommand struct {
	Command string                 `json:"command"`
	Params  map[string]interface{} `json:"params"`
	ID      string                 `json:"id"`
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	conn          *websocket.Conn
	send          chan WebSocketMessage
	server        *EnhancedDashboardServer
	id            string
	subscriptions map[string]bool
}

// handleWebSocket handles WebSocket connections
func (s *EnhancedDashboardServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &WebSocketClient{
		conn:          conn,
		send:          make(chan WebSocketMessage, 256),
		server:        s,
		id:            uuid.New().String(),
		subscriptions: make(map[string]bool),
	}

	// Register client
	s.clientsMux.Lock()
	s.clients[client.id] = client
	s.clientsMux.Unlock()

	// Send welcome message
	welcomeMsg := WebSocketMessage{
		Type:      "connected",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"clientId": client.id,
			"message":  "Connected to DriftMgr Enhanced Dashboard",
			"version":  "2.0.0",
		},
	}

	if err := conn.WriteJSON(welcomeMsg); err != nil {
		log.Printf("Failed to send welcome message: %v", err)
		s.removeClient(client)
		return
	}

	// Start client handlers
	go client.writePump()
	go client.readPump()
}

// readPump handles incoming messages from the client
func (c *WebSocketClient) readPump() {
	defer func() {
		c.server.removeClient(c)
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var cmd WebSocketCommand
		err := c.conn.ReadJSON(&cmd)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process command
		c.processCommand(cmd)
	}
}

// writePump handles outgoing messages to the client
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				return
			}
			
			// Track message sent
			if c.server.apiServer != nil {
				c.server.apiServer.IncrementWSMessagesSent()
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// processCommand processes incoming WebSocket commands
func (c *WebSocketClient) processCommand(cmd WebSocketCommand) {
	response := WebSocketMessage{
		Type:      "response",
		ID:        cmd.ID,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}

	switch cmd.Command {
	case "subscribe":
		c.handleSubscribe(cmd, &response)
	case "unsubscribe":
		c.handleUnsubscribe(cmd, &response)
	case "getStatus":
		c.handleGetStatus(cmd, &response)
	case "getJobs":
		c.handleGetJobs(cmd, &response)
	case "getMetrics":
		c.handleGetMetrics(cmd, &response)
	case "ping":
		response.Type = "pong"
		response.Data["timestamp"] = time.Now().Unix()
	case "executeCommand":
		c.handleExecuteCommand(cmd, &response)
		return // Don't send response here as terminal output will be streamed
	default:
		response.Type = "error"
		response.Data["error"] = "Unknown command: " + cmd.Command
	}

	select {
	case c.send <- response:
	default:
		// Client's send channel is full
		c.server.removeClient(c)
	}
}

// handleSubscribe handles subscription requests
func (c *WebSocketClient) handleSubscribe(cmd WebSocketCommand, response *WebSocketMessage) {
	topics, ok := cmd.Params["topics"].([]interface{})
	if !ok {
		response.Type = "error"
		response.Data["error"] = "Invalid topics parameter"
		return
	}

	subscribed := []string{}
	for _, topic := range topics {
		if topicStr, ok := topic.(string); ok {
			c.subscriptions[topicStr] = true
			subscribed = append(subscribed, topicStr)
		}
	}

	response.Data["subscribed"] = subscribed
	response.Data["message"] = "Subscribed to topics"
}

// handleUnsubscribe handles unsubscription requests
func (c *WebSocketClient) handleUnsubscribe(cmd WebSocketCommand, response *WebSocketMessage) {
	topics, ok := cmd.Params["topics"].([]interface{})
	if !ok {
		response.Type = "error"
		response.Data["error"] = "Invalid topics parameter"
		return
	}

	unsubscribed := []string{}
	for _, topic := range topics {
		if topicStr, ok := topic.(string); ok {
			delete(c.subscriptions, topicStr)
			unsubscribed = append(unsubscribed, topicStr)
		}
	}

	response.Data["unsubscribed"] = unsubscribed
	response.Data["message"] = "Unsubscribed from topics"
}

// handleGetStatus handles status requests
func (c *WebSocketClient) handleGetStatus(cmd WebSocketCommand, response *WebSocketMessage) {
	response.Type = "status"

	// Get system status
	status := map[string]interface{}{
		"connected_clients": len(c.server.clients),
		"active_jobs":       c.server.jobManager.GetActiveJobCount(),
		"providers":         c.server.getProviderStatus(),
		"resources":         c.server.dataStore.GetResourceCount(),
		"drifts":            len(c.server.dataStore.GetDrifts()),
		"uptime":            time.Since(c.server.startTime).Seconds(),
	}

	response.Data["status"] = status
}

// handleGetJobs handles job list requests
func (c *WebSocketClient) handleGetJobs(cmd WebSocketCommand, response *WebSocketMessage) {
	response.Type = "jobs"

	// Get filter parameters
	jobType, _ := cmd.Params["type"].(string)
	status, _ := cmd.Params["status"].(string)
	limit := 50
	if l, ok := cmd.Params["limit"].(float64); ok {
		limit = int(l)
	}

	// Get filtered jobs
	jobs := c.server.jobManager.GetFilteredJobs(jobType, status, limit)

	response.Data["jobs"] = jobs
	response.Data["total"] = len(jobs)
}

// handleGetMetrics handles metrics requests
func (c *WebSocketClient) handleGetMetrics(cmd WebSocketCommand, response *WebSocketMessage) {
	response.Type = "metrics"

	period := "1h"
	if p, ok := cmd.Params["period"].(string); ok {
		period = p
	}

	metrics := c.server.getMetrics(period)
	response.Data["metrics"] = metrics
	response.Data["period"] = period
}

// Broadcast methods for real-time updates

// BroadcastDiscoveryUpdate broadcasts discovery updates
func (s *EnhancedDashboardServer) BroadcastDiscoveryUpdate(provider string, progress int, message string) {
	update := WebSocketMessage{
		Type:      "discovery_update",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"provider": provider,
			"progress": progress,
			"message":  message,
		},
	}

	s.broadcastToSubscribed("discovery", update)
}

// BroadcastDriftDetected broadcasts drift detection events
func (s *EnhancedDashboardServer) BroadcastDriftDetected(driftCount int, severity string) {
	update := WebSocketMessage{
		Type:      "drift_detected",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"count":    driftCount,
			"severity": severity,
			"alert":    driftCount > 10,
		},
	}

	s.broadcastToSubscribed("drift", update)
}

// BroadcastRemediationUpdate broadcasts remediation updates
func (s *EnhancedDashboardServer) BroadcastRemediationUpdate(planID string, status string, progress int) {
	update := WebSocketMessage{
		Type:      "remediation_update",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"planId":   planID,
			"status":   status,
			"progress": progress,
		},
	}

	s.broadcastToSubscribed("remediation", update)
}

// BroadcastResourceChange broadcasts resource changes
func (s *EnhancedDashboardServer) BroadcastResourceChange(changeType string, resourceID string, provider string) {
	update := WebSocketMessage{
		Type:      "resource_change",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"changeType": changeType,
			"resourceId": resourceID,
			"provider":   provider,
		},
	}

	s.broadcastToSubscribed("resources", update)
}

// broadcastToSubscribed sends message to clients subscribed to a topic
func (s *EnhancedDashboardServer) broadcastToSubscribed(topic string, message WebSocketMessage) {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()

	for _, client := range s.clients {
		// In production, would check client subscriptions
		go func(c *WebSocketClient) {
			if err := c.conn.WriteJSON(message); err != nil {
				s.removeClient(c)
			}
		}(client)
	}
}

// Helper methods for JobManager

// GetActiveJobCount returns the count of active jobs
func (jm *JobManager) GetActiveJobCount() int {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	count := 0
	for _, job := range jm.jobs {
		if job.Status == "running" || job.Status == "pending" {
			count++
		}
	}
	return count
}

// GetFilteredJobs returns filtered jobs
func (jm *JobManager) GetFilteredJobs(jobType, status string, limit int) []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	filtered := make([]*Job, 0)
	for _, job := range jm.jobs {
		if jobType != "" && job.Type != jobType {
			continue
		}
		if status != "" && job.Status != status {
			continue
		}
		filtered = append(filtered, job)
		if len(filtered) >= limit {
			break
		}
	}
	return filtered
}

// getProviderStatus returns the status of all providers
func (s *EnhancedDashboardServer) getProviderStatus() map[string]bool {
	status := make(map[string]bool)
	providers := []string{"aws", "azure", "gcp", "digitalocean"}

	for _, provider := range providers {
		// Check environment variables for simple configuration check
		configured := false
		switch provider {
		case "aws":
			configured = os.Getenv("AWS_ACCESS_KEY_ID") != ""
		case "azure":
			configured = os.Getenv("AZURE_SUBSCRIPTION_ID") != ""
		case "gcp":
			configured = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != ""
		case "digitalocean":
			configured = os.Getenv("DIGITALOCEAN_TOKEN") != ""
		}
		status[provider] = configured
	}

	return status
}

// getMetrics returns system metrics for the specified period
func (s *EnhancedDashboardServer) getMetrics(period string) map[string]interface{} {
	// Simplified metrics - in production would query actual metrics store
	return map[string]interface{}{
		"discovery_runs":    42,
		"drifts_detected":   156,
		"remediations":      23,
		"resources_managed": s.dataStore.GetResourceCount(),
		"avg_response_time": 125.5,
		"error_rate":        0.02,
	}
}

// handleExecuteCommand handles command execution and streams output via WebSocket
func (c *WebSocketClient) handleExecuteCommand(cmd WebSocketCommand, response *WebSocketMessage) {
	commandStr, ok := cmd.Params["command"].(string)
	if !ok {
		response.Type = "error"
		response.Data["error"] = "Invalid command parameter"
		c.send <- *response
		return
	}
	
	jobID, _ := cmd.Params["jobID"].(string)
	if jobID == "" {
		jobID = uuid.New().String()
	}
	
	// Send initial terminal status
	statusMsg := WebSocketMessage{
		Type:      "terminal_status",
		ID:        jobID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status":  "running",
			"command": commandStr,
		},
	}
	c.send <- statusMsg
	
	// Stream terminal output (simulated for now)
	go c.streamTerminalOutput(jobID, commandStr)
}

// streamTerminalOutput simulates streaming terminal output
func (c *WebSocketClient) streamTerminalOutput(jobID string, command string) {
	// In a real implementation, this would execute the command and stream its output
	// For now, we'll simulate output based on the command
	
	outputs := []struct {
		text       string
		outputType string
		delay      time.Duration
	}{
		{"Initializing discovery process...", "info", 100 * time.Millisecond},
		{"Connecting to cloud provider...", "info", 200 * time.Millisecond},
		{"Authentication successful", "success", 150 * time.Millisecond},
		{"Scanning regions: us-east-1, us-west-2", "info", 300 * time.Millisecond},
		{"Found 5 VPCs", "info", 200 * time.Millisecond},
		{"Found 12 EC2 instances", "info", 200 * time.Millisecond},
		{"Found 8 S3 buckets", "info", 200 * time.Millisecond},
		{"Found 3 RDS databases", "info", 200 * time.Millisecond},
		{"Analyzing resource configurations...", "info", 500 * time.Millisecond},
		{"Discovery completed successfully", "success", 100 * time.Millisecond},
	}
	
	for _, output := range outputs {
		time.Sleep(output.delay)
		
		msg := WebSocketMessage{
			Type:      "terminal_output",
			ID:        jobID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"text":        output.text,
				"output_type": output.outputType,
			},
		}
		
		select {
		case c.send <- msg:
		default:
			return // Client disconnected
		}
	}
	
	// Send completion status
	time.Sleep(200 * time.Millisecond)
	statusMsg := WebSocketMessage{
		Type:      "terminal_status",
		ID:        jobID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"status": "completed",
		},
	}
	c.send <- statusMsg
}
