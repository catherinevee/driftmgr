package main

import (
	"fmt"
	"os"

	"github.com/emicklei/dot"
)

func main() {
	g := dot.NewGraph(dot.Directed)
	g.Attr("rankdir", "LR")
	g.Attr("label", "DriftMgr: Drift Detection Workflow")
	g.Attr("labelloc", "t")
	g.Attr("fontsize", "16")
	g.Attr("fontname", "Arial")

	// Start: User initiation
	user := g.Node("user").Label("User").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightblue")
	
	// Step 1: Backend Discovery
	backendDiscovery := g.Node("backend").Label("Backend\\nDiscovery").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	
	// Step 2: State Retrieval
	stateRetrieval := g.Node("retrieval").Label("State File\\nRetrieval").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	
	// Step 3: State Parsing & Validation
	stateParsing := g.Node("parsing").Label("State Parsing\\n& Validation").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	
	// Step 4: Cloud Resource Discovery (Parallel)
	discoveryCluster := g.Subgraph("Cloud Discovery", dot.ClusterOption{})
	awsDiscovery := discoveryCluster.Node("aws").Label("AWS Resource\\nDiscovery").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")
	azureDiscovery := discoveryCluster.Node("azure").Label("Azure Resource\\nDiscovery").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")
	gcpDiscovery := discoveryCluster.Node("gcp").Label("GCP Resource\\nDiscovery").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")
	doDiscovery := discoveryCluster.Node("do").Label("DO Resource\\nDiscovery").Attr("shape", "box3d").Attr("style", "filled").Attr("fillcolor", "orange")
	
	// Step 5: Comparison Engine
	comparison := g.Node("comparison").Label("Resource\\nComparison").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	
	// Step 6: Drift Classification
	classification := g.Node("classification").Label("Drift\\nClassification").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	
	// Step 7: Severity Scoring
	scoring := g.Node("scoring").Label("Severity\\nScoring").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")
	
	// Step 8: Report Generation
	reporting := g.Node("reporting").Label("Drift Report\\nGeneration").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	
	// Step 9: Output formats
	outputCluster := g.Subgraph("Output Formats", dot.ClusterOption{})
	jsonOutput := outputCluster.Node("json").Label("JSON\\nOutput").Attr("shape", "note").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	htmlReport := outputCluster.Node("html").Label("HTML\\nReport").Attr("shape", "note").Attr("style", "filled").Attr("fillcolor", "lightcoral")
	dashboard := outputCluster.Node("dashboard").Label("Web\\nDashboard").Attr("shape", "note").Attr("style", "filled").Attr("fillcolor", "lightcoral")

	// Create the main workflow flow
	g.Edge(user, backendDiscovery)
	g.Edge(backendDiscovery, stateRetrieval)
	g.Edge(stateRetrieval, stateParsing)
	
	// Parallel cloud discovery
	g.Edge(stateParsing, awsDiscovery)
	g.Edge(stateParsing, azureDiscovery)
	g.Edge(stateParsing, gcpDiscovery)
	g.Edge(stateParsing, doDiscovery)
	
	// Convergence to comparison
	g.Edge(awsDiscovery, comparison)
	g.Edge(azureDiscovery, comparison)
	g.Edge(gcpDiscovery, comparison)
	g.Edge(doDiscovery, comparison)
	
	// Analysis pipeline
	g.Edge(comparison, classification)
	g.Edge(classification, scoring)
	g.Edge(scoring, reporting)
	
	// Output formats
	g.Edge(reporting, jsonOutput)
	g.Edge(reporting, htmlReport)
	g.Edge(reporting, dashboard)

	// Write DOT file
	file, err := os.Create("drift_detection_flow.dot")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	g.Write(file)
	fmt.Println("Generated drift_detection_flow.dot")
	fmt.Println("To convert to PNG, run: dot -Tpng drift_detection_flow.dot -o drift_detection_flow.png")
}