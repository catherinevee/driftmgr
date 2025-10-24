package main

import (
	"fmt"
	"os"

	"github.com/emicklei/dot"
)

func main() {
	g := dot.NewGraph(dot.Directed)
	g.Attr("rankdir", "LR")
	g.Attr("label", "DriftMgr API Architecture (25+ Endpoints)")
	g.Attr("labelloc", "t")
	g.Attr("fontsize", "18")
	g.Attr("fontname", "Arial")

	// Client Layer
	clientCluster := g.Subgraph("Client Layer", dot.ClusterOption{})
	webDashboard := clientCluster.Node("webdashboard").Label("Web Dashboard\\n(HTML/CSS/JS)").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightblue")
	apiClient := clientCluster.Node("apiclient").Label("API Client\\n(Programmatic)").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightblue")

	// API Gateway Layer
	gatewayCluster := g.Subgraph("API Gateway Layer", dot.ClusterOption{})
	restAPI := gatewayCluster.Node("restapi").Label("REST API\\n(25+ Endpoints)\\nJWT Authentication").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	wsAPI := gatewayCluster.Node("wsapi").Label("WebSocket API\\n(Real-time)\\nLive Updates").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightgreen")

	// Authentication Endpoints
	authCluster := g.Subgraph("Authentication Endpoints", dot.ClusterOption{})
	login := authCluster.Node("login").Label("POST /auth/login\\nUser Authentication").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	register := authCluster.Node("register").Label("POST /auth/register\\nUser Registration").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	refresh := authCluster.Node("refresh").Label("POST /auth/refresh\\nToken Refresh").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	profile := authCluster.Node("profile").Label("GET /auth/profile\\nUser Profile").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	apikeys := authCluster.Node("apikeys").Label("POST /auth/api-keys\\nAPI Key Management").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcoral")

	// Backend Management Endpoints
	backendCluster := g.Subgraph("Backend Management", dot.ClusterOption{})
	backendList := backendCluster.Node("backendlist").Label("GET /backends/list\\nList Backends").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	backendDiscover := backendCluster.Node("backenddiscover").Label("POST /backends/discover\\nDiscover Backends").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	backendDetails := backendCluster.Node("backenddetails").Label("GET /backends/{id}\\nBackend Details").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	backendUpdate := backendCluster.Node("backendupdate").Label("PUT /backends/{id}\\nUpdate Backend").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	backendTest := backendCluster.Node("backendtest").Label("POST /backends/{id}/test\\nTest Connection").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightyellow")

	// State Management Endpoints
	stateCluster := g.Subgraph("State Management", dot.ClusterOption{})
	stateList := stateCluster.Node("statelist").Label("GET /state/list\\nList State Files").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightsteelblue")
	stateDetails := stateCluster.Node("statedetails").Label("GET /state/details\\nState Details").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightsteelblue")
	stateImport := stateCluster.Node("stateimport").Label("POST /state/import\\nImport Resource").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightsteelblue")
	stateRemove := stateCluster.Node("stateremove").Label("DELETE /state/resources/{id}\\nRemove Resource").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightsteelblue")
	stateMove := stateCluster.Node("statemove").Label("POST /state/move\\nMove Resource").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightsteelblue")
	stateLock := stateCluster.Node("statelock").Label("POST /state/lock\\nLock State").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightsteelblue")

	// Resource Management Endpoints
	resourceCluster := g.Subgraph("Resource Management", dot.ClusterOption{})
	resourceList := resourceCluster.Node("resourcelist").Label("GET /resources\\nList Resources").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightpink")
	resourceDetails := resourceCluster.Node("resourcedetails").Label("GET /resources/{id}\\nResource Details").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightpink")
	resourceSearch := resourceCluster.Node("resourcesearch").Label("GET /resources/search\\nSearch Resources").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightpink")
	resourceTags := resourceCluster.Node("resourcetags").Label("PUT /resources/{id}/tags\\nUpdate Tags").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightpink")
	resourceCost := resourceCluster.Node("resourcecost").Label("GET /resources/{id}/cost\\nResource Cost").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightpink")
	resourceCompliance := resourceCluster.Node("resourcecompliance").Label("GET /resources/{id}/compliance\\nCompliance Status").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightpink")

	// Drift Detection Endpoints
	driftCluster := g.Subgraph("Drift Detection", dot.ClusterOption{})
	driftDetect := driftCluster.Node("driftdetect").Label("POST /drift/detect\\nDetect Drift").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcyan")
	driftResults := driftCluster.Node("driftresults").Label("GET /drift/results\\nList Results").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcyan")
	driftDetails := driftCluster.Node("driftdetails").Label("GET /drift/results/{id}\\nResult Details").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcyan")
	driftDelete := driftCluster.Node("driftdelete").Label("DELETE /drift/results/{id}\\nDelete Result").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcyan")
	driftHistory := driftCluster.Node("drifthistory").Label("GET /drift/history\\nDrift History").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcyan")
	driftSummary := driftCluster.Node("driftsummary").Label("GET /drift/summary\\nDrift Summary").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightcyan")

	// WebSocket Endpoints
	wsCluster := g.Subgraph("WebSocket Endpoints", dot.ClusterOption{})
	wsConnection := wsCluster.Node("wsconnection").Label("GET /ws\\nWebSocket Connection").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgray")
	wsAPIEndpoint := wsCluster.Node("wsapiendpoint").Label("GET /api/v1/ws\\nAPI WebSocket").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgray")
	wsStats := wsCluster.Node("wsstats").Label("GET /api/v1/ws/stats\\nConnection Stats").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgray")

	// Create connections - Client to API Gateway
	g.Edge(webDashboard, restAPI).Attr("label", "HTTP/HTTPS")
	g.Edge(webDashboard, wsAPI).Attr("label", "WebSocket")
	g.Edge(apiClient, restAPI).Attr("label", "REST Calls")
	g.Edge(apiClient, wsAPI).Attr("label", "WebSocket")

	// API Gateway to endpoint groups
	g.Edge(restAPI, login).Attr("label", "Route")
	g.Edge(restAPI, register).Attr("label", "Route")
	g.Edge(restAPI, refresh).Attr("label", "Route")
	g.Edge(restAPI, profile).Attr("label", "Route")
	g.Edge(restAPI, apikeys).Attr("label", "Route")

	g.Edge(restAPI, backendList).Attr("label", "Route")
	g.Edge(restAPI, backendDiscover).Attr("label", "Route")
	g.Edge(restAPI, backendDetails).Attr("label", "Route")
	g.Edge(restAPI, backendUpdate).Attr("label", "Route")
	g.Edge(restAPI, backendTest).Attr("label", "Route")

	g.Edge(restAPI, stateList).Attr("label", "Route")
	g.Edge(restAPI, stateDetails).Attr("label", "Route")
	g.Edge(restAPI, stateImport).Attr("label", "Route")
	g.Edge(restAPI, stateRemove).Attr("label", "Route")
	g.Edge(restAPI, stateMove).Attr("label", "Route")
	g.Edge(restAPI, stateLock).Attr("label", "Route")

	g.Edge(restAPI, resourceList).Attr("label", "Route")
	g.Edge(restAPI, resourceDetails).Attr("label", "Route")
	g.Edge(restAPI, resourceSearch).Attr("label", "Route")
	g.Edge(restAPI, resourceTags).Attr("label", "Route")
	g.Edge(restAPI, resourceCost).Attr("label", "Route")
	g.Edge(restAPI, resourceCompliance).Attr("label", "Route")

	g.Edge(restAPI, driftDetect).Attr("label", "Route")
	g.Edge(restAPI, driftResults).Attr("label", "Route")
	g.Edge(restAPI, driftDetails).Attr("label", "Route")
	g.Edge(restAPI, driftDelete).Attr("label", "Route")
	g.Edge(restAPI, driftHistory).Attr("label", "Route")
	g.Edge(restAPI, driftSummary).Attr("label", "Route")

	g.Edge(wsAPI, wsConnection).Attr("label", "Route")
	g.Edge(wsAPI, wsAPIEndpoint).Attr("label", "Route")
	g.Edge(wsAPI, wsStats).Attr("label", "Route")

	// Write DOT file
	file, err := os.Create("output/driftmgr_api_architecture.dot")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	g.Write(file)
	fmt.Println("Generated output/driftmgr_api_architecture.dot")
	fmt.Println("To convert to PNG, run: dot -Tpng output/driftmgr_api_architecture.dot -o output/driftmgr_api_architecture.png")
}
