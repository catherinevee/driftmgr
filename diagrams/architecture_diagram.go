package main

import (
	"fmt"
	"os"

	"github.com/emicklei/dot"
)

func main() {
	g := dot.NewGraph(dot.Directed)
	g.Attr("rankdir", "TB")
	g.Attr("label", "DriftMgr Architecture")
	g.Attr("labelloc", "t")
	g.Attr("fontsize", "16")
	g.Attr("fontname", "Arial")

	// User Interface Layer
	uiCluster := g.Subgraph("User Interface", dot.ClusterOption{})
	cli := uiCluster.Node("cli").Label("CLI\\n(driftmgr)").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightblue")
	webUI := uiCluster.Node("webui").Label("Web UI\\n(Dashboard)").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightblue")
	api := uiCluster.Node("api").Label("REST API\\n(Server)").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightblue")

	// Core Services Layer
	coreCluster := g.Subgraph("Core Services", dot.ClusterOption{})
	driftDetector := coreCluster.Node("drift").Label("Drift Detection\\nEngine").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	stateManager := coreCluster.Node("state").Label("State\\nManager").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	remediation := coreCluster.Node("remediation").Label("Remediation\\nEngine").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	discovery := coreCluster.Node("discovery").Label("Resource\\nDiscovery").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")

	// Analysis & Intelligence Layer
	analysisCluster := g.Subgraph("Analysis & Intelligence", dot.ClusterOption{})
	analytics := analysisCluster.Node("analytics").Label("Analytics\\n& BI").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	costAnalyzer := analysisCluster.Node("cost").Label("Cost\\nAnalyzer").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	compliance := analysisCluster.Node("compliance").Label("Compliance\\n& Policy").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	automation := analysisCluster.Node("automation").Label("Automation\\nEngine").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")

	// Cloud Providers
	cloudCluster := g.Subgraph("Cloud Providers", dot.ClusterOption{})
	awsProvider := cloudCluster.Node("aws").Label("AWS\\nProvider").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")
	azureProvider := cloudCluster.Node("azure").Label("Azure\\nProvider").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")
	gcpProvider := cloudCluster.Node("gcp").Label("GCP\\nProvider").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")
	doProvider := cloudCluster.Node("do").Label("DigitalOcean\\nProvider").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")

	// Backend Storage
	storageCluster := g.Subgraph("Backend Storage", dot.ClusterOption{})
	s3Backend := storageCluster.Node("s3").Label("S3\\nBackend").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	azureStorage := storageCluster.Node("azurestorage").Label("Azure Blob\\nStorage").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	gcsStorage := storageCluster.Node("gcs").Label("GCS\\nStorage").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	localBackend := storageCluster.Node("local").Label("Local\\nBackend").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightcoral")

	// Terraform State Files
	tfState := g.Node("tfstate").Label("Terraform\\nState Files").Attr("shape", "folder").Attr("style", "filled").Attr("fillcolor", "lightgray")

	// Create connections - User Interface to Core Services
	g.Edge(cli, driftDetector)
	g.Edge(webUI, api)
	g.Edge(api, driftDetector)
	g.Edge(api, stateManager)
	g.Edge(api, remediation)

	// Core Services interconnections
	g.Edge(driftDetector, discovery)
	g.Edge(driftDetector, stateManager).Attr("dir", "both")
	g.Edge(stateManager, tfState).Attr("dir", "both")
	g.Edge(remediation, stateManager).Attr("dir", "both")

	// Analysis layer connections
	g.Edge(driftDetector, analytics)
	g.Edge(driftDetector, costAnalyzer)
	g.Edge(analytics, automation)
	g.Edge(compliance, remediation)

	// Cloud provider connections
	g.Edge(discovery, awsProvider)
	g.Edge(discovery, azureProvider)
	g.Edge(discovery, gcpProvider)
	g.Edge(discovery, doProvider)

	// Backend storage connections
	g.Edge(stateManager, s3Backend).Attr("dir", "both")
	g.Edge(stateManager, azureStorage).Attr("dir", "both")
	g.Edge(stateManager, gcsStorage).Attr("dir", "both")
	g.Edge(stateManager, localBackend).Attr("dir", "both")

	// State file connections to backends
	g.Edge(tfState, s3Backend).Attr("dir", "both")
	g.Edge(tfState, azureStorage).Attr("dir", "both")
	g.Edge(tfState, gcsStorage).Attr("dir", "both")
	g.Edge(tfState, localBackend).Attr("dir", "both")

	// Write DOT file
	file, err := os.Create("driftmgr_architecture.dot")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	g.Write(file)
	fmt.Println("Generated driftmgr_architecture.dot")
	fmt.Println("To convert to PNG, run: dot -Tpng driftmgr_architecture.dot -o driftmgr_architecture.png")
}