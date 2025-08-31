package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Import mappings from old to new paths
var importMappings = map[string]string{
	"github.com/catherinevee/driftmgr/internal/domain/resource":                      "github.com/catherinevee/driftmgr/internal/domain/resource",
	"github.com/catherinevee/driftmgr/internal/domain/resource":                      "github.com/catherinevee/driftmgr/internal/domain/resource",
	"github.com/catherinevee/driftmgr/internal/domain/resource":                        "github.com/catherinevee/driftmgr/internal/domain/resource",
	"github.com/catherinevee/driftmgr/internal/domain/drift":                       "github.com/catherinevee/driftmgr/internal/domain/drift",
	"github.com/catherinevee/driftmgr/internal/infrastructure/terraform/state":                  "github.com/catherinevee/driftmgr/internal/infrastructure/terraform/state",
	"github.com/catherinevee/driftmgr/internal/domain/state":                 "github.com/catherinevee/driftmgr/internal/domain/state",
	"github.com/catherinevee/driftmgr/internal/application/analysis":                          "github.com/catherinevee/driftmgr/internal/application/analysis",
	"github.com/catherinevee/driftmgr/internal/domain/remediation":            "github.com/catherinevee/driftmgr/internal/domain/remediation",
	"github.com/catherinevee/driftmgr/internal/domain/remediation":                         "github.com/catherinevee/driftmgr/internal/domain/remediation",
	"github.com/catherinevee/driftmgr/internal/domain/remediation":                            "github.com/catherinevee/driftmgr/internal/domain/remediation",
	"github.com/catherinevee/driftmgr/internal/application/discovery":                   "github.com/catherinevee/driftmgr/internal/application/discovery",
	"github.com/catherinevee/driftmgr/internal/application/discovery":                  "github.com/catherinevee/driftmgr/internal/application/discovery",
	"github.com/catherinevee/driftmgr/internal/infrastructure/persistence/cache":                            "github.com/catherinevee/driftmgr/internal/infrastructure/persistence/cache",
	"github.com/catherinevee/driftmgr/internal/application/monitoring":                       "github.com/catherinevee/driftmgr/internal/application/monitoring",
	"github.com/catherinevee/driftmgr/internal/application/monitoring":                         "github.com/catherinevee/driftmgr/internal/application/monitoring",
	"github.com/catherinevee/driftmgr/internal/application/analysis":                            "github.com/catherinevee/driftmgr/internal/application/analysis",
	"github.com/catherinevee/driftmgr/internal/infrastructure/persistence/cache":                      "github.com/catherinevee/driftmgr/internal/infrastructure/persistence/cache",
	"github.com/catherinevee/driftmgr/internal/infrastructure/cloud/aws":                    "github.com/catherinevee/driftmgr/internal/infrastructure/cloud/aws",
	"github.com/catherinevee/driftmgr/internal/infrastructure/cloud/azure":                  "github.com/catherinevee/driftmgr/internal/infrastructure/cloud/azure",
	"github.com/catherinevee/driftmgr/internal/infrastructure/cloud/gcp":                    "github.com/catherinevee/driftmgr/internal/infrastructure/cloud/gcp",
	"github.com/catherinevee/driftmgr/internal/infrastructure/cloud/digitalocean":           "github.com/catherinevee/driftmgr/internal/infrastructure/cloud/digitalocean",
	"github.com/catherinevee/driftmgr/internal/infrastructure/cloud":                         "github.com/catherinevee/driftmgr/internal/infrastructure/cloud",
	"github.com/catherinevee/driftmgr/internal/infrastructure/persistence/database":                         "github.com/catherinevee/driftmgr/internal/infrastructure/persistence/database",
	"github.com/catherinevee/driftmgr/internal/infrastructure/notifications":                    "github.com/catherinevee/driftmgr/internal/infrastructure/notifications",
	"github.com/catherinevee/driftmgr/internal/interfaces/api/rest":                              "github.com/catherinevee/driftmgr/internal/interfaces/api/rest",
	"github.com/catherinevee/driftmgr/internal/interfaces/api/rest/websocket":                    "github.com/catherinevee/driftmgr/internal/interfaces/api/websocket",
	"github.com/catherinevee/driftmgr/internal/shared/config":                           "github.com/catherinevee/driftmgr/internal/shared/config",
	"github.com/catherinevee/driftmgr/internal/shared/credentials":                      "github.com/catherinevee/driftmgr/internal/shared/credentials",
	"github.com/catherinevee/driftmgr/internal/shared/resilience":                       "github.com/catherinevee/driftmgr/internal/shared/resilience",
	"github.com/catherinevee/driftmgr/internal/domain/resource":                                "github.com/catherinevee/driftmgr/internal/domain/resource",
}

func main() {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-Go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip vendor and .git directories
		if strings.Contains(path, "vendor") || strings.Contains(path, ".git") {
			return nil
		}

		// Read file
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		originalContent := string(content)
		newContent := originalContent

		// Apply all import mappings
		for oldImport, newImport := range importMappings {
			newContent = strings.ReplaceAll(newContent, oldImport, newImport)
		}

		// Also update package references in code
		// Update resource.Resource to resource.Resource
		newContent = strings.ReplaceAll(newContent, "resource.Resource", "resource.Resource")
		newContent = strings.ReplaceAll(newContent, "resource.DriftResult", "resource.DriftResult")
		newContent = strings.ReplaceAll(newContent, "resource.CostEstimate", "resource.CostEstimate")
		
		// Update fingerprint references
		newContent = strings.ReplaceAll(newContent, "resource.ResourceFingerprinter", "resource.ResourceFingerprinter")
		newContent = strings.ReplaceAll(newContent, "resource.NewResourceFingerprinter", "resource.NewResourceFingerprinter")
		
		// Update discovery references
		newContent = strings.ReplaceAll(newContent, "resource.ResourceCategorizer", "resource.ResourceCategorizer")
		newContent = strings.ReplaceAll(newContent, "resource.ResourceCategory", "resource.ResourceCategory")
		newContent = strings.ReplaceAll(newContent, "resource.ImportCandidate", "resource.ImportCandidate")
		
		// Update aggregator references
		newContent = strings.ReplaceAll(newContent, "state.StateAggregator", "state.StateAggregator")
		newContent = strings.ReplaceAll(newContent, "state.NewStateAggregator", "state.NewStateAggregator")

		// Only write if content changed
		if newContent != originalContent {
			err = ioutil.WriteFile(path, []byte(newContent), info.Mode())
			if err != nil {
				return err
			}
			fmt.Printf("Updated imports in: %s\n", path)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Import updates completed successfully!")
}