package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/catherinevee/driftmgr/quality"
)

func main() {
	var (
		reportPath = flag.String("report", "", "Path to quality report JSON")
		strict     = flag.Bool("strict", false, "Use strict quality gates")
	)
	flag.Parse()

	if *reportPath == "" {
		log.Fatal("Report path is required")
	}

	// Load report
	data, err := os.ReadFile(*reportPath)
	if err != nil {
		log.Fatalf("Failed to read report: %v", err)
	}

	var report map[string]interface{}
	if err := json.Unmarshal(data, &report); err != nil {
		log.Fatalf("Failed to parse report: %v", err)
	}

	// Check quality gates
	gates := quality.NewQualityGates(*strict)
	projectPath := "."
	if p, ok := report["project"].(string); ok {
		projectPath = p
	}

	passed, violations := gates.CheckAll(projectPath)

	// Output results
	if !passed {
		fmt.Println("Quality gates FAILED:")
		for _, v := range violations {
			fmt.Printf("  ❌ %s\n", v)
		}
		os.Exit(1)
	}

	fmt.Println("✅ All quality gates passed")

	// Check summary metrics if available
	if summary, ok := report["summary"].(map[string]interface{}); ok {
		if score, ok := summary["quality_score"].(float64); ok {
			fmt.Printf("Quality Score: %.1f/100\n", score)
			if *strict && score < 85 {
				fmt.Printf("❌ Quality score %.1f is below strict threshold of 85\n", score)
				os.Exit(1)
			}
		}
	}
}
