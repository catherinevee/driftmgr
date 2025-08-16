package web

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/remediation"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Dashboard provides a web-based dashboard for driftmgr
type Dashboard struct {
	router           *mux.Router
	discoveryEngine  *discovery.EnhancedDiscoveryEngine
	remediationEngine *remediation.AdvancedRemediationEngine
	upgrader         websocket.Upgrader
	clients          map[*websocket.Conn]bool
	clientsMutex     sync.RWMutex
	config           *DashboardConfig
}

// DashboardConfig configures the dashboard
type DashboardConfig struct {
	Port            string
	StaticDir       string
	TemplateDir     string
	RefreshInterval time.Duration
	MaxConnections  int
}

// NewDashboard creates a new dashboard
func NewDashboard(
	discoveryEngine *discovery.EnhancedDiscoveryEngine,
	remediationEngine *remediation.AdvancedRemediationEngine,
	config *DashboardConfig,
) *Dashboard {
	if config == nil {
		config = &DashboardConfig{
			Port:            "8080",
			StaticDir:       "web/static",
			TemplateDir:     "web/templates",
			RefreshInterval: 30 * time.Second,
			MaxConnections:  100,
		}
	}

	dashboard := &Dashboard{
		router:           mux.NewRouter(),
		discoveryEngine:  discoveryEngine,
		remediationEngine: remediationEngine,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		clients: make(map[*websocket.Conn]bool),
		config:  config,
	}

	dashboard.setupRoutes()
	return dashboard
}

// setupRoutes sets up the dashboard routes
func (d *Dashboard) setupRoutes() {
	// Static files
	d.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(d.config.StaticDir))))

	// API routes
	api := d.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/resources", d.getResources).Methods("GET")
	api.HandleFunc("/drift", d.getDrift).Methods("GET")
	api.HandleFunc("/remediate", d.remediateDrift).Methods("POST")
	api.HandleFunc("/costs", d.getCosts).Methods("GET")
	api.HandleFunc("/security", d.getSecurity).Methods("GET")
	api.HandleFunc("/compliance", d.getCompliance).Methods("GET")
	api.HandleFunc("/metrics", d.getMetrics).Methods("GET")

	// WebSocket for real-time updates
	d.router.HandleFunc("/ws", d.handleWebSocket)

	// Dashboard pages
	d.router.HandleFunc("/", d.dashboardPage)
	d.router.HandleFunc("/resources", d.resourcesPage)
	d.router.HandleFunc("/drift", d.driftPage)
	d.router.HandleFunc("/remediation", d.remediationPage)
	d.router.HandleFunc("/costs", d.costsPage)
	d.router.HandleFunc("/security", d.securityPage)
	d.router.HandleFunc("/compliance", d.compliancePage)
}

// Start starts the dashboard server
func (d *Dashboard) Start() error {
	// Start real-time update goroutine
	go d.startRealTimeUpdates()

	fmt.Printf("Dashboard starting on port %s\n", d.config.Port)
	return http.ListenAndServe(":"+d.config.Port, d.router)
}

// startRealTimeUpdates starts the real-time update loop
func (d *Dashboard) startRealTimeUpdates() {
	ticker := time.NewTicker(d.config.RefreshInterval)
	defer ticker.Stop()

	for range ticker.C {
		d.broadcastUpdate()
	}
}

// broadcastUpdate broadcasts updates to all connected clients
func (d *Dashboard) broadcastUpdate() {
	d.clientsMutex.RLock()
	defer d.clientsMutex.RUnlock()

	update := DashboardUpdate{
		Timestamp: time.Now(),
		Type:      "update",
		Data:      d.getDashboardData(),
	}

	updateJSON, err := json.Marshal(update)
	if err != nil {
		fmt.Printf("Error marshaling update: %v\n", err)
		return
	}

	for client := range d.clients {
		err := client.WriteMessage(websocket.TextMessage, updateJSON)
		if err != nil {
			fmt.Printf("Error sending update to client: %v\n", err)
			client.Close()
			delete(d.clients, client)
		}
	}
}

// getDashboardData gets current dashboard data
func (d *Dashboard) getDashboardData() map[string]interface{} {
	// This would fetch real data from the discovery and remediation engines
	// For now, return mock data
	return map[string]interface{}{
		"resources": map[string]interface{}{
			"total":      150,
			"by_type":    map[string]int{"ec2": 50, "rds": 20, "s3": 30, "eks": 10, "lambda": 40},
			"by_region":  map[string]int{"us-east-1": 80, "us-west-2": 70},
			"by_status":  map[string]int{"active": 140, "inactive": 10},
		},
		"drift": map[string]interface{}{
			"total":      25,
			"by_severity": map[string]int{"low": 10, "medium": 10, "high": 3, "critical": 2},
			"by_type":    map[string]int{"configuration": 15, "security": 5, "compliance": 5},
		},
		"costs": map[string]interface{}{
			"monthly":    15000.0,
			"daily":      500.0,
			"trend":      "increasing",
			"percentage": 5.2,
		},
		"security": map[string]interface{}{
			"overall_score": 75,
			"risks":         8,
			"recommendations": 12,
		},
		"compliance": map[string]interface{}{
			"overall_score": 85,
			"violations":    3,
			"frameworks":    []string{"SOC2", "GDPR", "HIPAA"},
		},
	}
}

// handleWebSocket handles WebSocket connections
func (d *Dashboard) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := d.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Error upgrading connection: %v\n", err)
		return
	}

	d.clientsMutex.Lock()
	if len(d.clients) >= d.config.MaxConnections {
		d.clientsMutex.Unlock()
		conn.Close()
		return
	}
	d.clients[conn] = true
	d.clientsMutex.Unlock()

	// Send initial data
	initialData := DashboardUpdate{
		Timestamp: time.Now(),
		Type:      "initial",
		Data:      d.getDashboardData(),
	}

	initialJSON, err := json.Marshal(initialData)
	if err == nil {
		conn.WriteMessage(websocket.TextMessage, initialJSON)
	}

	// Handle client messages
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			d.clientsMutex.Lock()
			delete(d.clients, conn)
			d.clientsMutex.Unlock()
			conn.Close()
			break
		}

		// Handle client message (e.g., filter changes, refresh requests)
		d.handleClientMessage(conn, message)
	}
}

// handleClientMessage handles messages from WebSocket clients
func (d *Dashboard) handleClientMessage(conn *websocket.Conn, message []byte) {
	var clientMessage ClientMessage
	if err := json.Unmarshal(message, &clientMessage); err != nil {
		return
	}

	switch clientMessage.Type {
	case "refresh":
		// Send immediate update
		update := DashboardUpdate{
			Timestamp: time.Now(),
			Type:      "refresh",
			Data:      d.getDashboardData(),
		}
		updateJSON, _ := json.Marshal(update)
		conn.WriteMessage(websocket.TextMessage, updateJSON)
	case "filter":
		// Handle filter changes
		d.handleFilterChange(conn, clientMessage.Data)
	}
}

// handleFilterChange handles filter changes from clients
func (d *Dashboard) handleFilterChange(conn *websocket.Conn, filterData interface{}) {
	// Apply filters and send filtered data
	filteredData := d.getFilteredData(filterData)
	update := DashboardUpdate{
		Timestamp: time.Now(),
		Type:      "filtered",
		Data:      filteredData,
	}
	updateJSON, _ := json.Marshal(update)
	conn.WriteMessage(websocket.TextMessage, updateJSON)
}

// getFilteredData gets data filtered by client criteria
func (d *Dashboard) getFilteredData(filterData interface{}) map[string]interface{} {
	// Implementation would apply filters to real data
	// For now, return mock filtered data
	return map[string]interface{}{
		"filtered_resources": map[string]interface{}{
			"count": 75,
			"data":  []interface{}{},
		},
	}
}

// API handlers
func (d *Dashboard) getResources(w http.ResponseWriter, r *http.Request) {
	// Get resources from discovery engine
	resources, err := d.discoveryEngine.DiscoverResourcesEnhanced(
		context.Background(),
		"aws",
		[]string{"us-east-1", "us-west-2"},
		discovery.DefaultDiscoveryOptions(),
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := APIResponse{
		Success: true,
		Data:    resources,
	}

	json.NewEncoder(w).Encode(response)
}

func (d *Dashboard) getDrift(w http.ResponseWriter, r *http.Request) {
	// Get drift information
	// This would integrate with drift detection
	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"drifts": []interface{}{},
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (d *Dashboard) remediateDrift(w http.ResponseWriter, r *http.Request) {
	var request RemediationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Perform remediation
	// This would integrate with remediation engine
	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"remediation_id": "remediation-123",
			"status":         "in_progress",
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (d *Dashboard) getCosts(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"costs": map[string]interface{}{
				"monthly":    15000.0,
				"daily":      500.0,
				"trend":      "increasing",
				"percentage": 5.2,
			},
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (d *Dashboard) getSecurity(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"security": map[string]interface{}{
				"overall_score": 75,
				"risks":         8,
				"recommendations": 12,
			},
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (d *Dashboard) getCompliance(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"compliance": map[string]interface{}{
				"overall_score": 85,
				"violations":    3,
				"frameworks":    []string{"SOC2", "GDPR", "HIPAA"},
			},
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (d *Dashboard) getMetrics(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"metrics": map[string]interface{}{
				"discovery_accuracy": 99.5,
				"drift_detection_precision": 95.2,
				"remediation_success_rate": 92.1,
				"api_response_time": 245,
			},
		},
	}

	json.NewEncoder(w).Encode(response)
}

// Page handlers
func (d *Dashboard) dashboardPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(d.config.TemplateDir + "/dashboard.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := DashboardPageData{
		Title: "DriftMgr Dashboard",
		Data:  d.getDashboardData(),
	}

	tmpl.Execute(w, data)
}

func (d *Dashboard) resourcesPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(d.config.TemplateDir + "/resources.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title: "Resources",
	}

	tmpl.Execute(w, data)
}

func (d *Dashboard) driftPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(d.config.TemplateDir + "/drift.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title: "Drift Detection",
	}

	tmpl.Execute(w, data)
}

func (d *Dashboard) remediationPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(d.config.TemplateDir + "/remediation.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title: "Remediation",
	}

	tmpl.Execute(w, data)
}

func (d *Dashboard) costsPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(d.config.TemplateDir + "/costs.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title: "Cost Analysis",
	}

	tmpl.Execute(w, data)
}

func (d *Dashboard) securityPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(d.config.TemplateDir + "/security.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title: "Security Assessment",
	}

	tmpl.Execute(w, data)
}

func (d *Dashboard) compliancePage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(d.config.TemplateDir + "/compliance.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title: "Compliance",
	}

	tmpl.Execute(w, data)
}

// Data structures
type DashboardUpdate struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
}

type ClientMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

type RemediationRequest struct {
	ResourceID string `json:"resource_id"`
	Strategy   string `json:"strategy"`
	Force      bool   `json:"force"`
}

type DashboardPageData struct {
	Title string                 `json:"title"`
	Data  map[string]interface{} `json:"data"`
}

type PageData struct {
	Title string `json:"title"`
}
