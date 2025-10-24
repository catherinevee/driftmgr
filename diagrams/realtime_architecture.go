package main

import (
	"fmt"
	"os"

	"github.com/emicklei/dot"
)

func main() {
	g := dot.NewGraph(dot.Directed)
	g.Attr("rankdir", "TB")
	g.Attr("label", "DriftMgr Real-time Architecture")
	g.Attr("labelloc", "t")
	g.Attr("fontsize", "18")
	g.Attr("fontname", "Arial")

	// Client Layer
	clientCluster := g.Subgraph("Client Layer", dot.ClusterOption{})
	webClient := clientCluster.Node("webclient").Label("Web Dashboard\\n(HTML/CSS/JS)\\nReal-time UI").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightblue")
	apiClient := clientCluster.Node("apiclient").Label("API Client\\n(REST + WebSocket)\\nProgrammatic").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightblue")

	// WebSocket Layer
	wsCluster := g.Subgraph("WebSocket Layer", dot.ClusterOption{})
	wsHub := wsCluster.Node("wshub").Label("WebSocket Hub\\nConnection Manager\\nMessage Broadcasting").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	wsClient1 := wsCluster.Node("wsclient1").Label("WebSocket\\nClient 1").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightcyan")
	wsClient2 := wsCluster.Node("wsclient2").Label("WebSocket\\nClient 2").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightcyan")
	wsClientN := wsCluster.Node("wsclientn").Label("WebSocket\\nClient N").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightcyan")

	// Real-time Services
	realtimeCluster := g.Subgraph("Real-time Services", dot.ClusterOption{})
	driftService := realtimeCluster.Node("driftservice").Label("Drift Detection\\nService\\nLive Monitoring").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	alertService := realtimeCluster.Node("alertservice").Label("Alerting\\nService\\nInstant Notifications").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	monitorService := realtimeCluster.Node("monitorservice").Label("Monitoring\\nService\\nMetrics & Stats").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	authService := realtimeCluster.Node("authservice").Label("Authentication\\nService\\nToken Validation").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightyellow")

	// Message Types
	messageCluster := g.Subgraph("Message Types", dot.ClusterOption{})
	driftMsg := messageCluster.Node("driftmsg").Label("Drift Detection\\nMessages").Attr("shape", "note").Attr("style", "filled").Attr("fillcolor", "lightpink")
	alertMsg := messageCluster.Node("alertmsg").Label("Alert\\nMessages").Attr("shape", "note").Attr("style", "filled").Attr("fillcolor", "lightpink")
	statusMsg := messageCluster.Node("statusmsg").Label("Status\\nUpdates").Attr("shape", "note").Attr("style", "filled").Attr("fillcolor", "lightpink")
	heartbeatMsg := messageCluster.Node("heartbeatmsg").Label("Heartbeat\\nMessages").Attr("shape", "note").Attr("style", "filled").Attr("fillcolor", "lightpink")

	// Data Sources
	dataCluster := g.Subgraph("Data Sources", dot.ClusterOption{})
	cloudAPIs := dataCluster.Node("cloudapis").Label("Cloud APIs\\n(AWS, Azure, GCP, DO)\\nResource Discovery").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")
	stateFiles := dataCluster.Node("statefiles").Label("Terraform\\nState Files\\nInfrastructure State").Attr("shape", "folder").Attr("style", "filled").Attr("fillcolor", "lightgray")
	database := dataCluster.Node("database").Label("PostgreSQL\\nDatabase\\nHistorical Data").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightsteelblue")

	// Create connections - Client to WebSocket
	g.Edge(webClient, wsClient1).Attr("label", "WebSocket\\nConnection")
	g.Edge(apiClient, wsClient2).Attr("label", "WebSocket\\nConnection")

	// WebSocket Hub connections
	g.Edge(wsClient1, wsHub).Attr("label", "Register\\nConnection")
	g.Edge(wsClient2, wsHub).Attr("label", "Register\\nConnection")
	g.Edge(wsClientN, wsHub).Attr("label", "Register\\nConnection")

	// Real-time service connections
	g.Edge(wsHub, driftService).Attr("label", "Broadcast\\nDrift Events")
	g.Edge(wsHub, alertService).Attr("label", "Broadcast\\nAlerts")
	g.Edge(wsHub, monitorService).Attr("label", "Broadcast\\nMetrics")
	g.Edge(wsHub, authService).Attr("label", "Validate\\nTokens")

	// Message flow connections
	g.Edge(driftService, driftMsg).Attr("label", "Generate")
	g.Edge(alertService, alertMsg).Attr("label", "Generate")
	g.Edge(monitorService, statusMsg).Attr("label", "Generate")
	g.Edge(wsHub, heartbeatMsg).Attr("label", "Periodic")

	// Data source connections
	g.Edge(driftService, cloudAPIs).Attr("label", "Query\\nResources")
	g.Edge(driftService, stateFiles).Attr("label", "Read\\nState")
	g.Edge(alertService, database).Attr("label", "Store\\nAlerts")
	g.Edge(monitorService, database).Attr("label", "Store\\nMetrics")

	// Message broadcasting
	g.Edge(driftMsg, wsClient1).Attr("label", "Real-time\\nUpdate")
	g.Edge(driftMsg, wsClient2).Attr("label", "Real-time\\nUpdate")
	g.Edge(driftMsg, wsClientN).Attr("label", "Real-time\\nUpdate")
	g.Edge(alertMsg, wsClient1).Attr("label", "Instant\\nAlert")
	g.Edge(alertMsg, wsClient2).Attr("label", "Instant\\nAlert")
	g.Edge(statusMsg, wsClient1).Attr("label", "Status\\nUpdate")
	g.Edge(heartbeatMsg, wsClient1).Attr("label", "Keep\\nAlive")

	// Write DOT file
	file, err := os.Create("output/driftmgr_realtime_architecture.dot")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	g.Write(file)
	fmt.Println("Generated output/driftmgr_realtime_architecture.dot")
	fmt.Println("To convert to PNG, run: dot -Tpng output/driftmgr_realtime_architecture.dot -o output/driftmgr_realtime_architecture.png")
}
