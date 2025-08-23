package dashboard

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/gorilla/websocket"
)

// DashboardServer provides an interactive web dashboard
type DashboardServer struct {
	server    *http.Server
	websocket *WebSocketManager
	charts    *ChartGenerator
	realtime  *RealtimeManager
	port      string
	mu        sync.RWMutex
	clients   map[string]*WebSocketClient
	events    chan DashboardEvent
	broadcast chan BroadcastMessage
}

// WebSocketManager handles WebSocket connections
type WebSocketManager struct {
	upgrader websocket.Upgrader
	clients  map[string]*WebSocketClient
	mu       sync.RWMutex
}

// WebSocketClient represents a connected client
type WebSocketClient struct {
	ID      string
	Conn    *websocket.Conn
	Send    chan []byte
	Filters map[string]bool
	mu      sync.Mutex
}

// ChartGenerator creates charts and visualizations
type ChartGenerator struct {
	templates map[string]*template.Template
}

// RealtimeManager handles real-time updates
type RealtimeManager struct {
	clients   map[string]*WebSocketClient
	events    chan DashboardEvent
	broadcast chan BroadcastMessage
	mu        sync.RWMutex
}

// DashboardEvent represents a dashboard event
type DashboardEvent struct {
	Type      string                 `json:"type"`
	Data      interface{}            `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	Event  DashboardEvent `json:"event"`
	Filter string         `json:"filter"`
}

// DashboardData represents the data for the dashboard
type DashboardData struct {
	DriftTimeline    []DriftEvent      `json:"drift_timeline"`
	ResourceMap      ResourceHierarchy `json:"resource_map"`
	CostAnalysis     CostData          `json:"cost_analysis"`
	SecurityScore    SecurityMetrics   `json:"security_score"`
	ComplianceStatus ComplianceData    `json:"compliance_status"`
	Performance      PerformanceData   `json:"performance"`
	Summary          DashboardSummary  `json:"summary"`
}

// DriftEvent represents a drift event
type DriftEvent struct {
	ID           string    `json:"id"`
	ResourceID   string    `json:"resource_id"`
	ResourceType string    `json:"resource_type"`
	Provider     string    `json:"provider"`
	Region       string    `json:"region"`
	DriftType    string    `json:"drift_type"`
	Severity     string    `json:"severity"`
	Description  string    `json:"description"`
	Timestamp    time.Time `json:"timestamp"`
	Status       string    `json:"status"`
}

// ResourceHierarchy represents the resource hierarchy
type ResourceHierarchy struct {
	Root       ResourceNode `json:"root"`
	TotalNodes int          `json:"total_nodes"`
	Levels     int          `json:"levels"`
}

// ResourceNode represents a node in the resource hierarchy
type ResourceNode struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Provider string                 `json:"provider"`
	Region   string                 `json:"region"`
	State    string                 `json:"state"`
	Children []ResourceNode         `json:"children"`
	Metadata map[string]interface{} `json:"metadata"`
}

// CostData represents cost analysis data
type CostData struct {
	CurrentCost   float64            `json:"current_cost"`
	ProjectedCost float64            `json:"projected_cost"`
	Optimizations []CostOptimization `json:"optimizations"`
	Savings       float64            `json:"savings"`
	Trends        []CostTrend        `json:"trends"`
}

// CostOptimization represents a cost optimization
type CostOptimization struct {
	ResourceID    string  `json:"resource_id"`
	ResourceType  string  `json:"resource_type"`
	CurrentCost   float64 `json:"current_cost"`
	OptimizedCost float64 `json:"optimized_cost"`
	Savings       float64 `json:"savings"`
	Description   string  `json:"description"`
	Priority      string  `json:"priority"`
}

// CostTrend represents a cost trend
type CostTrend struct {
	Date  time.Time `json:"date"`
	Cost  float64   `json:"cost"`
	Trend string    `json:"trend"`
}

// SecurityMetrics represents security metrics
type SecurityMetrics struct {
	OverallScore    int                      `json:"overall_score"`
	RiskLevel       string                   `json:"risk_level"`
	Vulnerabilities []SecurityVulnerability  `json:"vulnerabilities"`
	Recommendations []SecurityRecommendation `json:"recommendations"`
	Trends          []SecurityTrend          `json:"trends"`
}

// SecurityVulnerability represents a security vulnerability
type SecurityVulnerability struct {
	ID          string `json:"id"`
	ResourceID  string `json:"resource_id"`
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// SecurityRecommendation represents a security recommendation
type SecurityRecommendation struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// SecurityTrend represents a security trend
type SecurityTrend struct {
	Date  time.Time `json:"date"`
	Score int       `json:"score"`
	Trend string    `json:"trend"`
}

// ComplianceData represents compliance status
type ComplianceData struct {
	OverallStatus string                `json:"overall_status"`
	Frameworks    []ComplianceFramework `json:"frameworks"`
	Violations    []ComplianceViolation `json:"violations"`
	Trends        []ComplianceTrend     `json:"trends"`
}

// ComplianceFramework represents a compliance framework
type ComplianceFramework struct {
	Name         string  `json:"name"`
	Status       string  `json:"status"`
	Score        float64 `json:"score"`
	TotalChecks  int     `json:"total_checks"`
	PassedChecks int     `json:"passed_checks"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	ID          string `json:"id"`
	Framework   string `json:"framework"`
	Control     string `json:"control"`
	ResourceID  string `json:"resource_id"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// ComplianceTrend represents a compliance trend
type ComplianceTrend struct {
	Date   time.Time `json:"date"`
	Status string    `json:"status"`
	Score  float64   `json:"score"`
}

// PerformanceData represents performance data
type PerformanceData struct {
	OverallScore    int                         `json:"overall_score"`
	Issues          []PerformanceIssue          `json:"issues"`
	Recommendations []PerformanceRecommendation `json:"recommendations"`
	Trends          []PerformanceTrend          `json:"trends"`
}

// PerformanceIssue represents a performance issue
type PerformanceIssue struct {
	ID          string `json:"id"`
	ResourceID  string `json:"resource_id"`
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Impact      string `json:"impact"`
}

// PerformanceRecommendation represents a performance recommendation
type PerformanceRecommendation struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Priority    string `json:"priority"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// PerformanceTrend represents a performance trend
type PerformanceTrend struct {
	Date  time.Time `json:"date"`
	Score int       `json:"score"`
	Trend string    `json:"trend"`
}

// DashboardSummary represents dashboard summary
type DashboardSummary struct {
	TotalResources   int       `json:"total_resources"`
	DriftsFound      int       `json:"drifts_found"`
	CriticalDrifts   int       `json:"critical_drifts"`
	SecurityScore    int       `json:"security_score"`
	ComplianceScore  float64   `json:"compliance_score"`
	PerformanceScore int       `json:"performance_score"`
	CostSavings      float64   `json:"cost_savings"`
	LastUpdated      time.Time `json:"last_updated"`
}

// NewDashboardServer creates a new dashboard server
func NewDashboardServer(port string) *DashboardServer {
	ds := &DashboardServer{
		port:      port,
		clients:   make(map[string]*WebSocketClient),
		events:    make(chan DashboardEvent, 100),
		broadcast: make(chan BroadcastMessage, 100),
		websocket: &WebSocketManager{
			upgrader: websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool {
					return true // Allow all origins for development
				},
			},
			clients: make(map[string]*WebSocketClient),
		},
		charts: &ChartGenerator{
			templates: make(map[string]*template.Template),
		},
		realtime: &RealtimeManager{
			clients:   make(map[string]*WebSocketClient),
			events:    make(chan DashboardEvent, 100),
			broadcast: make(chan BroadcastMessage, 100),
		},
	}

	// Initialize templates
	ds.initializeTemplates()

	return ds
}

// Start starts the dashboard server
func (ds *DashboardServer) Start() error {
	mux := http.NewServeMux()

	// Dashboard routes
	mux.HandleFunc("/", ds.serveDashboard)
	mux.HandleFunc("/dashboard", ds.serveDashboard)
	mux.HandleFunc("/api/drift", ds.getDriftData)
	mux.HandleFunc("/api/costs", ds.getCostData)
	mux.HandleFunc("/api/performance", ds.getPerformanceData)
	mux.HandleFunc("/api/security", ds.getSecurityData)
	mux.HandleFunc("/api/compliance", ds.getComplianceData)
	mux.HandleFunc("/ws", ds.handleWebSocket)

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	ds.server = &http.Server{
		Addr:    ":" + ds.port,
		Handler: mux,
	}

	// Start real-time manager
	go ds.realtimeManager()

	fmt.Printf("🚀 Dashboard server starting on port %s\n", ds.port)
	return ds.server.ListenAndServe()
}

// serveDashboard serves the main dashboard page
func (ds *DashboardServer) serveDashboard(w http.ResponseWriter, r *http.Request) {
	// Generate sample dashboard data
	data := ds.generateSampleData()

	// Parse and execute template
	tmpl, err := template.New("dashboard").Parse(dashboardHTML)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, data)
}

// getDriftData returns drift data as JSON
func (ds *DashboardServer) getDriftData(w http.ResponseWriter, r *http.Request) {
	data := ds.generateSampleData()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data.DriftTimeline)
}

// getCostData returns cost data as JSON
func (ds *DashboardServer) getCostData(w http.ResponseWriter, r *http.Request) {
	data := ds.generateSampleData()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data.CostAnalysis)
}

// getPerformanceData returns performance data as JSON
func (ds *DashboardServer) getPerformanceData(w http.ResponseWriter, r *http.Request) {
	data := ds.generateSampleData()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data.Performance)
}

// getSecurityData returns security data as JSON
func (ds *DashboardServer) getSecurityData(w http.ResponseWriter, r *http.Request) {
	data := ds.generateSampleData()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data.SecurityScore)
}

// getComplianceData returns compliance data as JSON
func (ds *DashboardServer) getComplianceData(w http.ResponseWriter, r *http.Request) {
	data := ds.generateSampleData()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data.ComplianceStatus)
}

// handleWebSocket handles WebSocket connections
func (ds *DashboardServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ds.websocket.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade failed: %v\n", err)
		return
	}

	client := &WebSocketClient{
		ID:      fmt.Sprintf("client_%d", time.Now().UnixNano()),
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Filters: make(map[string]bool),
	}

	// Register client
	ds.websocket.mu.Lock()
	ds.websocket.clients[client.ID] = client
	ds.websocket.mu.Unlock()

	// Start client handlers
	go ds.handleClient(client)
}

// handleClient handles a WebSocket client
func (ds *DashboardServer) handleClient(client *WebSocketClient) {
	defer func() {
		client.Conn.Close()
		ds.websocket.mu.Lock()
		delete(ds.websocket.clients, client.ID)
		ds.websocket.mu.Unlock()
	}()

	// Send welcome message
	welcome := DashboardEvent{
		Type:      "welcome",
		Data:      map[string]string{"client_id": client.ID},
		Timestamp: time.Now(),
	}
	client.sendEvent(welcome)

	// Handle incoming messages
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}

		// Parse message
		var event DashboardEvent
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		// Handle event
		ds.handleClientEvent(client, event)
	}
}

// handleClientEvent handles client events
func (ds *DashboardServer) handleClientEvent(client *WebSocketClient, event DashboardEvent) {
	switch event.Type {
	case "subscribe":
		// Handle subscription
		if filters, ok := event.Data.(map[string]interface{}); ok {
			for filter, enabled := range filters {
				if enabled, ok := enabled.(bool); ok {
					client.Filters[filter] = enabled
				}
			}
		}
	case "request_data":
		// Send requested data
		ds.sendRequestedData(client, event)
	}
}

// sendRequestedData sends requested data to client
func (ds *DashboardServer) sendRequestedData(client *WebSocketClient, event DashboardEvent) {
	if dataType, ok := event.Data.(string); ok {
		var data interface{}

		switch dataType {
		case "drift":
			data = ds.generateSampleData().DriftTimeline
		case "costs":
			data = ds.generateSampleData().CostAnalysis
		case "security":
			data = ds.generateSampleData().SecurityScore
		case "compliance":
			data = ds.generateSampleData().ComplianceStatus
		case "performance":
			data = ds.generateSampleData().Performance
		}

		if data != nil {
			response := DashboardEvent{
				Type:      "data_response",
				Data:      data,
				Timestamp: time.Now(),
				Metadata:  map[string]interface{}{"type": dataType},
			}
			client.sendEvent(response)
		}
	}
}

// sendEvent sends an event to a client
func (client *WebSocketClient) sendEvent(event DashboardEvent) {
	client.mu.Lock()
	defer client.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	select {
	case client.Send <- data:
	default:
		// Channel is full, close connection
		client.Conn.Close()
	}
}

// BroadcastDrift broadcasts a drift event to all clients
func (ds *DashboardServer) BroadcastDrift(drift models.DriftAnalysis) {
	event := DashboardEvent{
		Type: "drift_detected",
		Data: DriftEvent{
			ID:           drift.ResourceID,
			ResourceID:   drift.ResourceID,
			ResourceType: drift.ResourceType,
			Provider:     drift.Provider,
			Region:       drift.Region,
			DriftType:    drift.DriftType,
			Severity:     drift.Severity,
			Description:  "Drift detected",
			Timestamp:    drift.Timestamp,
			Status:       "new",
		},
		Timestamp: time.Now(),
	}

	ds.broadcast <- BroadcastMessage{
		Event:  event,
		Filter: "drift",
	}
}

// realtimeManager manages real-time updates
func (ds *DashboardServer) realtimeManager() {
	for {
		select {
		case msg := <-ds.broadcast:
			ds.broadcastToClients(msg)
		}
	}
}

// broadcastToClients broadcasts a message to all clients
func (ds *DashboardServer) broadcastToClients(msg BroadcastMessage) {
	ds.websocket.mu.RLock()
	defer ds.websocket.mu.RUnlock()

	for _, client := range ds.websocket.clients {
		// Check if client is subscribed to this filter
		if client.Filters[msg.Filter] {
			client.sendEvent(msg.Event)
		}
	}
}

// generateSampleData generates sample dashboard data
func (ds *DashboardServer) generateSampleData() DashboardData {
	return DashboardData{
		DriftTimeline: []DriftEvent{
			{
				ID:           "drift_1",
				ResourceID:   "sg-12345678",
				ResourceType: "aws_security_group",
				Provider:     "aws",
				Region:       "us-east-1",
				DriftType:    "security_group",
				Severity:     "high",
				Description:  "Security group has open port 22",
				Timestamp:    time.Now().Add(-1 * time.Hour),
				Status:       "new",
			},
			{
				ID:           "drift_2",
				ResourceID:   "i-87654321",
				ResourceType: "aws_instance",
				Provider:     "aws",
				Region:       "us-west-2",
				DriftType:    "instance",
				Severity:     "medium",
				Description:  "Instance type changed from t3.micro to t3.small",
				Timestamp:    time.Now().Add(-2 * time.Hour),
				Status:       "remediated",
			},
		},
		ResourceMap: ResourceHierarchy{
			Root: ResourceNode{
				ID:       "root",
				Name:     "Infrastructure",
				Type:     "root",
				Provider: "multi",
				Region:   "global",
				State:    "active",
				Children: []ResourceNode{
					{
						ID:       "vpc-1",
						Name:     "Main VPC",
						Type:     "aws_vpc",
						Provider: "aws",
						Region:   "us-east-1",
						State:    "active",
						Children: []ResourceNode{
							{
								ID:       "sg-12345678",
								Name:     "Web Security Group",
								Type:     "aws_security_group",
								Provider: "aws",
								Region:   "us-east-1",
								State:    "active",
							},
						},
					},
				},
			},
			TotalNodes: 3,
			Levels:     3,
		},
		CostAnalysis: CostData{
			CurrentCost:   1250.50,
			ProjectedCost: 1350.75,
			Optimizations: []CostOptimization{
				{
					ResourceID:    "i-87654321",
					ResourceType:  "aws_instance",
					CurrentCost:   150.00,
					OptimizedCost: 75.00,
					Savings:       75.00,
					Description:   "Downsize instance from t3.small to t3.micro",
					Priority:      "high",
				},
			},
			Savings: 75.00,
			Trends: []CostTrend{
				{
					Date:  time.Now().AddDate(0, -1, 0),
					Cost:  1200.00,
					Trend: "increasing",
				},
				{
					Date:  time.Now(),
					Cost:  1250.50,
					Trend: "increasing",
				},
			},
		},
		SecurityScore: SecurityMetrics{
			OverallScore: 85,
			RiskLevel:    "medium",
			Vulnerabilities: []SecurityVulnerability{
				{
					ID:          "vuln_1",
					ResourceID:  "sg-12345678",
					Type:        "open_port",
					Severity:    "high",
					Description: "Port 22 is open to 0.0.0.0/0",
					Remediation: "Restrict port 22 to specific IP ranges",
				},
			},
			Recommendations: []SecurityRecommendation{
				{
					ID:          "rec_1",
					Type:        "security_group",
					Priority:    "high",
					Description: "Close unnecessary ports",
					Action:      "Update security group rules",
				},
			},
		},
		ComplianceStatus: ComplianceData{
			OverallStatus: "compliant",
			Frameworks: []ComplianceFramework{
				{
					Name:         "SOC2",
					Status:       "compliant",
					Score:        95.5,
					TotalChecks:  100,
					PassedChecks: 95,
				},
			},
		},
		Performance: PerformanceData{
			OverallScore: 90,
			Issues: []PerformanceIssue{
				{
					ID:          "perf_1",
					ResourceID:  "i-87654321",
					Type:        "high_cpu",
					Severity:    "medium",
					Description: "CPU utilization is consistently high",
					Impact:      "Potential performance degradation",
				},
			},
		},
		Summary: DashboardSummary{
			TotalResources:   150,
			DriftsFound:      2,
			CriticalDrifts:   1,
			SecurityScore:    85,
			ComplianceScore:  95.5,
			PerformanceScore: 90,
			CostSavings:      75.00,
			LastUpdated:      time.Now(),
		},
	}
}

// initializeTemplates initializes HTML templates
func (ds *DashboardServer) initializeTemplates() {
	// This would load actual HTML templates
	// For now, we'll use the embedded HTML
}

// Stop stops the dashboard server
func (ds *DashboardServer) Stop() error {
	if ds.server != nil {
		return ds.server.Close()
	}
	return nil
}

// dashboardHTML contains the main dashboard template with DaisyUI
const dashboardHTML = `<!DOCTYPE html>
<html lang="en" data-theme="light">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DriftMgr Dashboard</title>
    <link href="https://cdn.jsdelivr.net/npm/daisyui@4.7.2/dist/full.min.css" rel="stylesheet" type="text/css" />
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <link href="/static/css/dashboard.css" rel="stylesheet" type="text/css" />
    <script src="/static/js/dashboard.js" defer></script>
    <style>
        .gradient-bg {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }
        .card-hover:hover {
            transform: translateY(-2px);
            transition: transform 0.2s ease-in-out;
        }
        .animate-pulse-slow {
            animation: pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite;
        }
    </style>
</head>
<body class="bg-base-100">
    <!-- Navigation -->
    <div class="navbar bg-base-100 shadow-lg border-b">
        <div class="navbar-start">
            <div class="dropdown">
                <div tabindex="0" role="button" class="btn btn-ghost lg:hidden">
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h8m-8 6h16"></path>
                    </svg>
                </div>
                <ul tabindex="0" class="menu menu-sm dropdown-content mt-3 z-[1] p-2 shadow bg-base-100 rounded-box w-52">
                    <li><a href="#overview">Overview</a></li>
                    <li><a href="#drifts">Drifts</a></li>
                    <li><a href="#resources">Resources</a></li>
                    <li><a href="#costs">Costs</a></li>
                    <li><a href="#security">Security</a></li>
                    <li><a href="#compliance">Compliance</a></li>
                </ul>
            </div>
            <a class="btn btn-ghost text-xl font-bold">
                <svg class="w-8 h-8 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"></path>
                </svg>
                DriftMgr
            </a>
        </div>
        <div class="navbar-center hidden lg:flex">
            <ul class="menu menu-horizontal px-1">
                <li><a href="#overview" class="font-semibold">Overview</a></li>
                <li><a href="#drifts" class="font-semibold">Drifts</a></li>
                <li><a href="#resources" class="font-semibold">Resources</a></li>
                <li><a href="#costs" class="font-semibold">Costs</a></li>
                <li><a href="#security" class="font-semibold">Security</a></li>
                <li><a href="#compliance" class="font-semibold">Compliance</a></li>
            </ul>
        </div>
        <div class="navbar-end">
            <div class="dropdown dropdown-end">
                <div tabindex="0" role="button" class="btn btn-ghost btn-circle">
                    <div class="indicator">
                        <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-5 5v-5z"></path>
                        </svg>
                        <span class="badge badge-xs badge-primary indicator-item"></span>
                    </div>
                </div>
                <div tabindex="0" class="mt-3 z-[1] card card-compact dropdown-content w-52 bg-base-100 shadow">
                    <div class="card-body">
                        <span class="font-bold text-lg">Notifications</span>
                        <span class="text-info">New drift detected!</span>
                    </div>
                </div>
            </div>
            <div class="dropdown dropdown-end ml-2">
                <div tabindex="0" role="button" class="btn btn-ghost btn-circle avatar">
                    <div class="w-10 rounded-full">
                        <img alt="Avatar" src="https://daisyui.com/images/stock/photo-1534528741775-53994a69daeb.jpg" />
                    </div>
                </div>
                <ul tabindex="0" class="menu menu-sm dropdown-content mt-3 z-[1] p-2 shadow bg-base-100 rounded-box w-52">
                    <li><a>Profile</a></li>
                    <li><a>Settings</a></li>
                    <li><a>Logout</a></li>
                </ul>
            </div>
        </div>
    </div>

    <!-- Main Content -->
    <div class="container mx-auto px-4 py-6">
        <!-- Overview Section -->
        <div id="overview" class="mb-8">
            <h2 class="text-3xl font-bold mb-6">Infrastructure Overview</h2>
            
            <!-- Summary Cards -->
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                <div class="stat bg-base-100 shadow-xl rounded-box card-hover">
                    <div class="stat-figure text-primary">
                        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
                        </svg>
                    </div>
                    <div class="stat-title">Total Resources</div>
                    <div class="stat-value text-primary">{{.Summary.TotalResources}}</div>
                    <div class="stat-desc">Across all providers</div>
                </div>

                <div class="stat bg-base-100 shadow-xl rounded-box card-hover">
                    <div class="stat-figure text-secondary">
                        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
                        </svg>
                    </div>
                    <div class="stat-title">Drifts Found</div>
                    <div class="stat-value text-secondary">{{.Summary.DriftsFound}}</div>
                    <div class="stat-desc">{{.Summary.CriticalDrifts}} critical</div>
                </div>

                <div class="stat bg-base-100 shadow-xl rounded-box card-hover">
                    <div class="stat-figure text-accent">
                        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                    </div>
                    <div class="stat-title">Security Score</div>
                    <div class="stat-value text-accent">{{.Summary.SecurityScore}}%</div>
                    <div class="stat-desc">Overall security rating</div>
                </div>

                <div class="stat bg-base-100 shadow-xl rounded-box card-hover">
                    <div class="stat-figure text-info">
                        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1"></path>
                        </svg>
                    </div>
                    <div class="stat-title">Cost Savings</div>
                    <div class="stat-value text-info">${{printf "%.2f" .Summary.CostSavings}}</div>
                    <div class="stat-desc">Potential monthly savings</div>
                </div>
            </div>

            <!-- Charts Row -->
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
                <div class="card bg-base-100 shadow-xl">
                    <div class="card-body">
                        <h3 class="card-title">Drift Timeline</h3>
                        <canvas id="driftChart" width="400" height="200"></canvas>
                    </div>
                </div>

                <div class="card bg-base-100 shadow-xl">
                    <div class="card-body">
                        <h3 class="card-title">Cost Trends</h3>
                        <canvas id="costChart" width="400" height="200"></canvas>
                    </div>
                </div>
            </div>
        </div>

        <!-- Drifts Section -->
        <div id="drifts" class="mb-8">
            <div class="flex justify-between items-center mb-6">
                <h2 class="text-3xl font-bold">Recent Drifts</h2>
                <button class="btn btn-primary">
                    <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                    </svg>
                    New Scan
                </button>
            </div>

            <div class="overflow-x-auto">
                <table class="table table-zebra w-full">
                    <thead>
                        <tr>
                            <th>Resource</th>
                            <th>Type</th>
                            <th>Provider</th>
                            <th>Severity</th>
                            <th>Status</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .DriftTimeline}}
                        <tr>
                            <td>
                                <div class="flex items-center space-x-3">
                                    <div class="avatar">
                                        <div class="mask mask-squircle w-12 h-12">
                                            <div class="bg-{{if eq .Severity "high"}}error{{else if eq .Severity "medium"}}warning{{else}}info{{end}} text-white flex items-center justify-center">
                                                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
                                                </svg>
                                            </div>
                                        </div>
                                    </div>
                                    <div>
                                        <div class="font-bold">{{.ResourceID}}</div>
                                        <div class="text-sm opacity-50">{{.ResourceType}}</div>
                                    </div>
                                </div>
                            </td>
                            <td>{{.DriftType}}</td>
                            <td>
                                <div class="badge badge-outline">{{.Provider}}</div>
                            </td>
                            <td>
                                <div class="badge badge-{{if eq .Severity "high"}}error{{else if eq .Severity "medium"}}warning{{else}}info{{end}} gap-1">
                                    <svg class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                                        <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd"></path>
                                    </svg>
                                    {{.Severity}}
                                </div>
                            </td>
                            <td>
                                <div class="badge badge-{{if eq .Status "new"}}error{{else if eq .Status "remediated"}}success{{else}}warning{{end}} gap-1">
                                    {{.Status}}
                                </div>
                            </td>
                            <td>
                                <div class="dropdown dropdown-left">
                                    <div tabindex="0" role="button" class="btn btn-ghost btn-xs">
                                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"></path>
                                        </svg>
                                    </div>
                                    <ul tabindex="0" class="dropdown-content menu p-2 shadow bg-base-100 rounded-box w-52">
                                        <li><a>View Details</a></li>
                                        <li><a>Remediate</a></li>
                                        <li><a>Ignore</a></li>
                                    </ul>
                                </div>
                            </td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Resources Section -->
        <div id="resources" class="mb-8">
            <h2 class="text-3xl font-bold mb-6">Resource Hierarchy</h2>
            
            <div class="card bg-base-100 shadow-xl">
                <div class="card-body">
                    <div class="flex justify-between items-center mb-4">
                        <h3 class="card-title">Infrastructure Tree</h3>
                        <div class="stats stats-horizontal shadow">
                            <div class="stat">
                                <div class="stat-title">Total Nodes</div>
                                <div class="stat-value text-primary">{{.ResourceMap.TotalNodes}}</div>
                            </div>
                            <div class="stat">
                                <div class="stat-title">Levels</div>
                                <div class="stat-value text-secondary">{{.ResourceMap.Levels}}</div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="mockup-code">
                        <pre data-prefix="$"><code>Infrastructure</code></pre>
                        {{range .ResourceMap.Root.Children}}
                        <pre data-prefix="├─"><code>{{.Name}} ({{.Type}})</code></pre>
                        {{range .Children}}
                        <pre data-prefix="│  └─"><code>{{.Name}} ({{.Type}})</code></pre>
                        {{end}}
                        {{end}}
                    </div>
                </div>
            </div>
        </div>

        <!-- Costs Section -->
        <div id="costs" class="mb-8">
            <h2 class="text-3xl font-bold mb-6">Cost Analysis</h2>
            
            <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
                <div class="card bg-base-100 shadow-xl">
                    <div class="card-body">
                        <h3 class="card-title">Current Costs</h3>
                        <div class="stat">
                            <div class="stat-value text-primary">${{printf "%.2f" .CostAnalysis.CurrentCost}}</div>
                            <div class="stat-desc">Monthly</div>
                        </div>
                    </div>
                </div>

                <div class="card bg-base-100 shadow-xl">
                    <div class="card-body">
                        <h3 class="card-title">Projected Costs</h3>
                        <div class="stat">
                            <div class="stat-value text-secondary">${{printf "%.2f" .CostAnalysis.ProjectedCost}}</div>
                            <div class="stat-desc">Next month</div>
                        </div>
                    </div>
                </div>

                <div class="card bg-base-100 shadow-xl">
                    <div class="card-body">
                        <h3 class="card-title">Potential Savings</h3>
                        <div class="stat">
                            <div class="stat-value text-accent">${{printf "%.2f" .CostAnalysis.Savings}}</div>
                            <div class="stat-desc">Optimization opportunities</div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="card bg-base-100 shadow-xl mt-6">
                <div class="card-body">
                    <h3 class="card-title">Cost Optimizations</h3>
                    <div class="overflow-x-auto">
                        <table class="table table-zebra w-full">
                            <thead>
                                <tr>
                                    <th>Resource</th>
                                    <th>Current Cost</th>
                                    <th>Optimized Cost</th>
                                    <th>Savings</th>
                                    <th>Priority</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range .CostAnalysis.Optimizations}}
                                <tr>
                                    <td>
                                        <div class="font-bold">{{.ResourceID}}</div>
                                        <div class="text-sm opacity-50">{{.ResourceType}}</div>
                                    </td>
                                    <td>${{printf "%.2f" .CurrentCost}}</td>
                                    <td>${{printf "%.2f" .OptimizedCost}}</td>
                                    <td class="text-success font-bold">${{printf "%.2f" .Savings}}</td>
                                    <td>
                                        <div class="badge badge-{{if eq .Priority "high"}}error{{else if eq .Priority "medium"}}warning{{else}}info{{end}}">
                                            {{.Priority}}
                                        </div>
                                    </td>
                                    <td>
                                        <button class="btn btn-primary btn-sm">Apply</button>
                                    </td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>

        <!-- Security Section -->
        <div id="security" class="mb-8">
            <h2 class="text-3xl font-bold mb-6">Security & Compliance</h2>
            
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div class="card bg-base-100 shadow-xl">
                    <div class="card-body">
                        <h3 class="card-title">Security Score</h3>
                        <div class="radial-progress text-primary" style="--value:{{.SecurityScore.OverallScore}}; --size:12rem; --thickness: 2px;">
                            {{.SecurityScore.OverallScore}}%
                        </div>
                        <div class="mt-4">
                            <div class="badge badge-{{if eq .SecurityScore.RiskLevel "high"}}error{{else if eq .SecurityScore.RiskLevel "medium"}}warning{{else}}success{{end}} gap-1">
                                Risk Level: {{.SecurityScore.RiskLevel}}
                            </div>
                        </div>
                    </div>
                </div>

                <div class="card bg-base-100 shadow-xl">
                    <div class="card-body">
                        <h3 class="card-title">Compliance Status</h3>
                        {{range .ComplianceStatus.Frameworks}}
                        <div class="flex justify-between items-center mb-4">
                            <div>
                                <div class="font-bold">{{.Name}}</div>
                                <div class="text-sm opacity-50">{{.PassedChecks}}/{{.TotalChecks}} checks passed</div>
                            </div>
                            <div class="text-right">
                                <div class="text-2xl font-bold text-{{if ge .Score 90}}success{{else if ge .Score 70}}warning{{else}}error{{end}}">{{printf "%.1f" .Score}}%</div>
                                <div class="badge badge-{{if eq .Status "compliant"}}success{{else}}error{{end}}">{{.Status}}</div>
                            </div>
                        </div>
                        {{end}}
                    </div>
                </div>
            </div>

            <div class="card bg-base-100 shadow-xl mt-6">
                <div class="card-body">
                    <h3 class="card-title">Security Vulnerabilities</h3>
                    <div class="overflow-x-auto">
                        <table class="table table-zebra w-full">
                            <thead>
                                <tr>
                                    <th>Resource</th>
                                    <th>Type</th>
                                    <th>Severity</th>
                                    <th>Description</th>
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {{range .SecurityScore.Vulnerabilities}}
                                <tr>
                                    <td>{{.ResourceID}}</td>
                                    <td>{{.Type}}</td>
                                    <td>
                                        <div class="badge badge-{{if eq .Severity "high"}}error{{else if eq .Severity "medium"}}warning{{else}}info{{end}}">
                                            {{.Severity}}
                                        </div>
                                    </td>
                                    <td>{{.Description}}</td>
                                    <td>
                                        <button class="btn btn-primary btn-sm">Remediate</button>
                                    </td>
                                </tr>
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Footer -->
    <footer class="footer footer-center p-10 bg-base-200 text-base-content rounded">
        <nav class="grid grid-flow-col gap-4">
            <a class="link link-hover">About us</a>
            <a class="link link-hover">Contact</a>
            <a class="link link-hover">Privacy Policy</a>
            <a class="link link-hover">Terms of Service</a>
        </nav> 
        <nav>
            <div class="grid grid-flow-col gap-4">
                <a><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" class="fill-current"><path d="M24 4.557c-.883.392-1.832.656-2.828.775 1.017-.609 1.798-1.574 2.165-2.724-.951.564-2.005.974-3.127 1.195-.897-.957-2.178-1.555-3.594-1.555-3.179 0-5.515 2.966-4.797 6.045-4.091-.205-7.719-2.165-10.148-5.144-1.29 2.213-.669 5.108 1.523 6.574-.806-.026-1.566-.247-2.229-.616-.054 2.281 1.581 4.415 3.949 4.89-.693.188-1.452.232-2.224.084.626 1.956 2.444 3.379 4.6 3.419-2.07 1.623-4.678 2.348-7.29 2.04 2.179 1.397 4.768 2.212 7.548 2.212 9.142 0 14.307-7.721 13.995-14.646.962-.695 1.797-1.562 2.457-2.549z"></path></svg></a>
                <a><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" class="fill-current"><path d="M19.615 3.184c-3.604-.246-11.631-.245-15.23 0-3.897.266-4.356 2.62-4.385 8.816.029 6.185.484 8.549 4.385 8.816 3.6.245 11.626.246 15.23 0 3.897-.266 4.356-2.62 4.385-8.816-.029-6.185-.484-8.549-4.385-8.816zm-10.615 12.816v-8l8 3.993-8 4.007z"></path></svg></a>
                <a><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" class="fill-current"><path d="M9 8h-3v4h3v12h5v-12h3.642l.358-4h-4v-1.667c0-.955.192-1.333 1.115-1.333h2.885v-5h-3.808c-3.596 0-5.192 1.583-5.192 4.615v3.385z"></path></svg></a>
            </div>
        </nav> 
        <aside>
            <p>Copyright © 2024 - All rights reserved by DriftMgr</p>
        </aside>
    </footer>

    <script>
        // Initialize charts
        document.addEventListener('DOMContentLoaded', function() {
            // Drift Chart
            const driftCtx = document.getElementById('driftChart').getContext('2d');
            new Chart(driftCtx, {
                type: 'line',
                data: {
                    labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'],
                    datasets: [{
                        label: 'Drifts Detected',
                        data: [12, 19, 3, 5, 2, 3],
                        borderColor: 'rgb(75, 192, 192)',
                        tension: 0.1
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        legend: {
                            position: 'top',
                        }
                    }
                }
            });

            // Cost Chart
            const costCtx = document.getElementById('costChart').getContext('2d');
            new Chart(costCtx, {
                type: 'bar',
                data: {
                    labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'],
                    datasets: [{
                        label: 'Monthly Cost ($)',
                        data: [1200, 1250, 1300, 1280, 1250, 1350],
                        backgroundColor: 'rgba(54, 162, 235, 0.2)',
                        borderColor: 'rgb(54, 162, 235)',
                        borderWidth: 1
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        legend: {
                            position: 'top',
                        }
                    }
                }
            });
        });

        // WebSocket connection for real-time updates
        const ws = new WebSocket('ws://' + window.location.host + '/ws');
        
        ws.onopen = function() {
            console.log('WebSocket connected');
        };
        
        ws.onmessage = function(event) {
            const data = JSON.parse(event.data);
            console.log('Received:', data);
            
            // Handle different event types
            switch(data.type) {
                case 'drift_detected':
                    showNotification('New drift detected!', 'warning');
                    break;
                case 'drift_remediated':
                    showNotification('Drift remediated successfully!', 'success');
                    break;
                default:
                    console.log('Unknown event type:', data.type);
            }
        };
        
        ws.onerror = function(error) {
            console.error('WebSocket error:', error);
        };
        
        ws.onclose = function() {
            console.log('WebSocket disconnected');
        };

        // Notification function
        function showNotification(message, type = 'info') {
            // Create toast notification
            const toast = document.createElement('div');
            toast.className = 'alert alert-' + type + ' fixed top-4 right-4 z-50 max-w-sm';
            toast.innerHTML = '<svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg><span>' + message + '</span>';
            
            document.body.appendChild(toast);
            
            // Remove after 5 seconds
            setTimeout(function() {
                toast.remove();
            }, 5000);
        }

        // Smooth scrolling for navigation links
        document.querySelectorAll('a[href^="#"]').forEach(anchor => {
            anchor.addEventListener('click', function (e) {
                e.preventDefault();
                const target = document.querySelector(this.getAttribute('href'));
                if (target) {
                    target.scrollIntoView({
                        behavior: 'smooth',
                        block: 'start'
                    });
                }
            });
        });
    </script>
</body>
</html>`
