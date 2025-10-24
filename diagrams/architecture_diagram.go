package main

import (
	"fmt"
	"os"

	"github.com/emicklei/dot"
)

func main() {
	g := dot.NewGraph(dot.Directed)
	g.Attr("rankdir", "TB")
	g.Attr("label", "DriftMgr Production Architecture")
	g.Attr("labelloc", "t")
	g.Attr("fontsize", "18")
	g.Attr("fontname", "Arial")

	// User Interface Layer
	uiCluster := g.Subgraph("User Interface Layer", dot.ClusterOption{})
	webUI := uiCluster.Node("webui").Label("Web Dashboard\\n(HTML/CSS/JS)\\nReal-time Updates").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightblue")
	restAPI := uiCluster.Node("restapi").Label("REST API\\n(25+ Endpoints)\\nJWT Auth").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightblue")
	wsAPI := uiCluster.Node("wsapi").Label("WebSocket API\\n(Real-time)\\nLive Updates").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightblue")

	// Authentication & Security Layer
	authCluster := g.Subgraph("Authentication & Security", dot.ClusterOption{})
	authService := authCluster.Node("auth").Label("Authentication\\nService\\nJWT + OAuth2").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	rbac := authCluster.Node("rbac").Label("Role-Based\\nAccess Control\\n(RBAC)").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	apiKeys := authCluster.Node("apikeys").Label("API Key\\nManagement").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcoral")

	// Core Business Logic Layer
	coreCluster := g.Subgraph("Core Business Logic", dot.ClusterOption{})
	driftDetector := coreCluster.Node("drift").Label("Drift Detection\\nEngine\\nSmart Prioritization").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	stateManager := coreCluster.Node("state").Label("State\\nManager\\nMulti-Backend").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	remediation := coreCluster.Node("remediation").Label("Remediation\\nEngine\\nAuto + Manual").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	discovery := coreCluster.Node("discovery").Label("Resource\\nDiscovery\\nMulti-Cloud").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	wsService := coreCluster.Node("wsservice").Label("WebSocket\\nService\\nReal-time").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")

	// Analytics & Intelligence Layer
	analysisCluster := g.Subgraph("Analytics & Intelligence", dot.ClusterOption{})
	analytics := analysisCluster.Node("analytics").Label("Analytics\\n& ML\\nPredictive").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	automation := analysisCluster.Node("automation").Label("Intelligent\\nAutomation\\nEngine").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	alerting := analysisCluster.Node("alerting").Label("Alerting\\nSystem\\nReal-time").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	monitoring := analysisCluster.Node("monitoring").Label("Monitoring\\n& Observability").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")

	// Data Layer
	dataCluster := g.Subgraph("Data Layer", dot.ClusterOption{})
	postgres := dataCluster.Node("postgres").Label("PostgreSQL\\nDatabase\\nConnection Pooling").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightsteelblue")
	redis := dataCluster.Node("redis").Label("Redis\\nCache\\nSession Store").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightsteelblue")

	// Cloud Providers
	cloudCluster := g.Subgraph("Cloud Providers", dot.ClusterOption{})
	awsProvider := cloudCluster.Node("aws").Label("AWS\\nProvider\\nEC2, S3, RDS").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")
	azureProvider := cloudCluster.Node("azure").Label("Azure\\nProvider\\nVMs, Storage").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")
	gcpProvider := cloudCluster.Node("gcp").Label("GCP\\nProvider\\nCompute, Storage").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")
	doProvider := cloudCluster.Node("do").Label("DigitalOcean\\nProvider\\nDroplets").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")

	// Backend Storage
	storageCluster := g.Subgraph("Terraform Backend Storage", dot.ClusterOption{})
	s3Backend := storageCluster.Node("s3").Label("S3\\nBackend").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	azureStorage := storageCluster.Node("azurestorage").Label("Azure Blob\\nStorage").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	gcsStorage := storageCluster.Node("gcs").Label("GCS\\nStorage").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	localBackend := storageCluster.Node("local").Label("Local\\nBackend").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightcoral")

	// Terraform State Files
	tfState := g.Node("tfstate").Label("Terraform\\nState Files").Attr("shape", "folder").Attr("style", "filled").Attr("fillcolor", "lightgray")

	// Create connections - User Interface Layer
	g.Edge(webUI, restAPI).Attr("label", "HTTP/HTTPS")
	g.Edge(webUI, wsAPI).Attr("label", "WebSocket")
	g.Edge(restAPI, wsAPI).Attr("label", "Real-time")

	// Authentication connections
	g.Edge(restAPI, authService).Attr("label", "JWT Auth")
	g.Edge(wsAPI, authService).Attr("label", "Token Validation")
	g.Edge(authService, rbac).Attr("label", "Permissions")
	g.Edge(authService, apiKeys).Attr("label", "API Keys")

	// Core Services connections
	g.Edge(restAPI, driftDetector).Attr("label", "API Calls")
	g.Edge(restAPI, stateManager).Attr("label", "State Ops")
	g.Edge(restAPI, remediation).Attr("label", "Remediation")
	g.Edge(restAPI, discovery).Attr("label", "Discovery")
	g.Edge(wsAPI, wsService).Attr("label", "Real-time")

	// Core Services interconnections
	g.Edge(driftDetector, discovery).Attr("label", "Resource Scan")
	g.Edge(driftDetector, stateManager).Attr("dir", "both").Attr("label", "State Compare")
	g.Edge(stateManager, tfState).Attr("dir", "both").Attr("label", "State Files")
	g.Edge(remediation, stateManager).Attr("dir", "both").Attr("label", "State Updates")
	g.Edge(wsService, driftDetector).Attr("label", "Live Updates")

	// Analytics & Intelligence connections
	g.Edge(driftDetector, analytics).Attr("label", "Drift Data")
	g.Edge(analytics, automation).Attr("label", "ML Insights")
	g.Edge(automation, remediation).Attr("label", "Auto Actions")
	g.Edge(driftDetector, alerting).Attr("label", "Alerts")
	g.Edge(alerting, wsService).Attr("label", "Notifications")
	g.Edge(monitoring, wsService).Attr("label", "Metrics")

	// Data Layer connections
	g.Edge(authService, postgres).Attr("label", "User Data")
	g.Edge(driftDetector, postgres).Attr("label", "Drift Results")
	g.Edge(stateManager, postgres).Attr("label", "State History")
	g.Edge(analytics, postgres).Attr("label", "Analytics Data")
	g.Edge(wsService, redis).Attr("label", "Sessions")
	g.Edge(authService, redis).Attr("label", "Token Cache")

	// Cloud provider connections
	g.Edge(discovery, awsProvider).Attr("label", "AWS API")
	g.Edge(discovery, azureProvider).Attr("label", "Azure API")
	g.Edge(discovery, gcpProvider).Attr("label", "GCP API")
	g.Edge(discovery, doProvider).Attr("label", "DO API")

	// Backend storage connections
	g.Edge(stateManager, s3Backend).Attr("dir", "both").Attr("label", "S3 State")
	g.Edge(stateManager, azureStorage).Attr("dir", "both").Attr("label", "Azure State")
	g.Edge(stateManager, gcsStorage).Attr("dir", "both").Attr("label", "GCS State")
	g.Edge(stateManager, localBackend).Attr("dir", "both").Attr("label", "Local State")

	// State file connections to backends
	g.Edge(tfState, s3Backend).Attr("dir", "both").Attr("label", "Remote State")
	g.Edge(tfState, azureStorage).Attr("dir", "both").Attr("label", "Remote State")
	g.Edge(tfState, gcsStorage).Attr("dir", "both").Attr("label", "Remote State")
	g.Edge(tfState, localBackend).Attr("dir", "both").Attr("label", "Local State")

	// Write DOT file
	file, err := os.Create("output/driftmgr_production_architecture.dot")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	g.Write(file)
	fmt.Println("Generated output/driftmgr_production_architecture.dot")
	fmt.Println("To convert to PNG, run: dot -Tpng output/driftmgr_production_architecture.dot -o output/driftmgr_production_architecture.png")
}
