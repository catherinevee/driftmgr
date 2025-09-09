package main

import (
	"fmt"
	"os"

	"github.com/emicklei/dot"
)

func main() {
	g := dot.NewGraph(dot.Directed)
	g.Attr("rankdir", "TB")
	g.Attr("label", "DriftMgr: Remediation Workflow")
	g.Attr("labelloc", "t")
	g.Attr("fontsize", "16")
	g.Attr("fontname", "Arial")

	// Input: Detected Drift
	driftInput := g.Node("drift").Label("Detected\\nDrift").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightcoral")

	// Decision: Remediation Strategy
	strategyDecision := g.Node("strategy").Label("Strategy\\nSelection").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")

	// Three remediation strategies
	strategyCluster := g.Subgraph("Remediation Strategies", dot.ClusterOption{})
	codeAsTruth := strategyCluster.Node("code").Label("Code-as-Truth\\n(Apply Terraform)").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	cloudAsTruth := strategyCluster.Node("cloud").Label("Cloud-as-Truth\\n(Update Code)").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	manualReview := strategyCluster.Node("manual").Label("Manual Review\\n(Generate Plan)").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightgreen")

	// Backup Creation
	backup := g.Node("backup").Label("Create\\nBackup").Attr("shape", "cylinder").Attr("style", "filled").Attr("fillcolor", "lightblue")

	// Execution Steps
	executionCluster := g.Subgraph("Execution Pipeline", dot.ClusterOption{})
	planGeneration := executionCluster.Node("plan").Label("Generate\\nPlan").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "orange")
	approval := executionCluster.Node("approval").Label("Approval\\nWorkflow").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "orange")
	execution := executionCluster.Node("execute").Label("Execute\\nRemediation").Attr("shape", "ellipse").Attr("style", "filled").Attr("fillcolor", "orange")

	// Verification
	verification := g.Node("verify").Label("Verify\\nChanges").Attr("shape", "diamond").Attr("style", "filled").Attr("fillcolor", "lightyellow")

	// Outcomes
	outcomeCluster := g.Subgraph("Outcomes", dot.ClusterOption{})
	success := outcomeCluster.Node("success").Label("Success\\n✓").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightgreen")
	rollback := outcomeCluster.Node("rollback").Label("Rollback\\nto Backup").Attr("shape", "box").Attr("style", "filled").Attr("fillcolor", "lightcoral")

	// Reporting
	reportingCluster := g.Subgraph("Reporting", dot.ClusterOption{})
	auditLog := reportingCluster.Node("audit").Label("Audit\\nLogging").Attr("shape", "note").Attr("style", "filled").Attr("fillcolor", "lightgray")
	notification := reportingCluster.Node("notify").Label("Send\\nNotification").Attr("shape", "note").Attr("style", "filled").Attr("fillcolor", "lightgray")

	// Main flow
	g.Edge(driftInput, strategyDecision)

	// Strategy branches
	g.Edge(strategyDecision, codeAsTruth).Label("Code-First")
	g.Edge(strategyDecision, cloudAsTruth).Label("Cloud-First")
	g.Edge(strategyDecision, manualReview).Label("Manual")

	// All strategies lead to backup
	g.Edge(codeAsTruth, backup)
	g.Edge(cloudAsTruth, backup)
	g.Edge(manualReview, backup)

	// Execution pipeline
	g.Edge(backup, planGeneration)
	g.Edge(planGeneration, approval)
	g.Edge(approval, execution)
	g.Edge(execution, verification)

	// Verification outcomes
	g.Edge(verification, success).Label("Pass")
	g.Edge(verification, rollback).Label("Fail")

	// Logging and notification
	g.Edge(success, auditLog)
	g.Edge(rollback, auditLog)
	g.Edge(auditLog, notification)

	// Write DOT file
	file, err := os.Create("remediation_workflow.dot")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	g.Write(file)
	fmt.Println("Generated remediation_workflow.dot")
	fmt.Println("To convert to PNG, run: dot -Tpng remediation_workflow.dot -o remediation_workflow.png")
}
