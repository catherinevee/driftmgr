package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var phase = flag.String("phase", "all", "Phase to check (1, 2, 3, 4, 5, 6, or all)")
	flag.Parse()

	fmt.Printf("Checking API documentation for phase: %s\n", *phase)

	// Define expected endpoints for each phase
	phaseEndpoints := map[string][]string{
		"1": {
			"GET /api/v1/drift/results/{id}",
			"GET /api/v1/drift/history",
			"GET /api/v1/drift/summary",
			"GET /api/v1/drift/results",
			"DELETE /api/v1/drift/results/{id}",
		},
		"2": {
			"POST /api/v1/remediation/apply",
			"POST /api/v1/remediation/preview",
			"GET /api/v1/remediation/status/{id}",
			"GET /api/v1/remediation/history",
			"POST /api/v1/remediation/cancel/{id}",
			"GET /api/v1/remediation/strategies",
		},
		"3": {
			"POST /api/v1/state/import",
			"POST /api/v1/state/remove",
			"POST /api/v1/state/move",
			"GET /api/v1/state/validate",
			"POST /api/v1/state/backup",
			"GET /api/v1/state/backups",
			"POST /api/v1/state/restore",
			"GET /api/v1/state/locks",
			"POST /api/v1/state/unlock",
		},
		"4": {
			"POST /api/v1/discover/scan",
			"GET /api/v1/discover/status/{id}",
			"GET /api/v1/discover/results/{id}",
			"POST /api/v1/discover/verify",
			"GET /api/v1/providers/status",
			"POST /api/v1/providers/{provider}/scan",
			"GET /api/v1/providers/{provider}/resources",
			"GET /api/v1/discover/history",
		},
		"5": {
			"GET /api/v1/config",
			"PUT /api/v1/config",
			"GET /api/v1/config/providers",
			"PUT /api/v1/config/providers",
			"POST /api/v1/config/providers/test",
			"GET /api/v1/config/environments",
			"PUT /api/v1/config/environments",
		},
		"6": {
			"GET /api/v1/metrics",
			"GET /api/v1/health/detailed",
			"GET /api/v1/status",
			"GET /api/v1/logs",
			"GET /api/v1/events",
			"POST /api/v1/alerts",
			"GET /api/v1/alerts",
			"PUT /api/v1/alerts/{id}",
			"DELETE /api/v1/alerts/{id}",
		},
	}

	var endpointsToCheck []string
	if *phase == "all" {
		for _, phaseEndpoints := range phaseEndpoints {
			endpointsToCheck = append(endpointsToCheck, phaseEndpoints...)
		}
	} else {
		if endpoints, exists := phaseEndpoints[*phase]; exists {
			endpointsToCheck = endpoints
		} else {
			fmt.Printf("Error: Invalid phase '%s'. Valid phases are: 1, 2, 3, 4, 5, 6, all\n", *phase)
			os.Exit(1)
		}
	}

	// Check if documentation exists for each endpoint
	docsDir := "docs/api"
	missingDocs := []string{}

	for _, endpoint := range endpointsToCheck {
		// Convert endpoint to filename
		filename := strings.ReplaceAll(endpoint, " ", "_")
		filename = strings.ReplaceAll(filename, "/", "_")
		filename = strings.ReplaceAll(filename, "{", "")
		filename = strings.ReplaceAll(filename, "}", "")
		filename = strings.ToLower(filename) + ".md"

		docPath := filepath.Join(docsDir, filename)
		if _, err := os.Stat(docPath); os.IsNotExist(err) {
			missingDocs = append(missingDocs, endpoint)
		}
	}

	if len(missingDocs) > 0 {
		fmt.Printf("❌ Missing documentation for %d endpoints:\n", len(missingDocs))
		for _, endpoint := range missingDocs {
			fmt.Printf("  - %s\n", endpoint)
		}
		os.Exit(1)
	}

	fmt.Printf("✅ All API endpoints have documentation\n")
}
