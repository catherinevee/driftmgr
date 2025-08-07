package cmd

import (
	"fmt"

	"github.com/catherinevee/driftmgr/internal/importer"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import resources into Terraform state",
	Long: `Import discovered resources into Terraform state files. Supports bulk imports
with parallel processing and includes comprehensive error handling and rollback capabilities.`,
	RunE: runImport,
}

var (
	inputFile      string
	parallelism    int
	dryRun         bool
	generateConfig bool
	validateAfter  bool
)

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&inputFile, "file", "f", "", "input file with resources to import (CSV/JSON)")
	importCmd.Flags().IntVarP(&parallelism, "parallel", "p", 5, "number of parallel import operations")
	importCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview imports without executing them")
	importCmd.Flags().BoolVar(&generateConfig, "generate-config", true, "generate Terraform configuration blocks")
	importCmd.Flags().BoolVar(&validateAfter, "validate", true, "validate state after import")
}

func runImport(cmd *cobra.Command, args []string) error {
	if inputFile == "" {
		return fmt.Errorf("input file is required for import operation")
	}

	fmt.Printf("📦 Starting import process from %s...\n", inputFile)

	importEngine := importer.NewEngine()

	config := importer.Config{
		InputFile:      inputFile,
		Parallelism:    parallelism,
		DryRun:         dryRun,
		GenerateConfig: generateConfig,
		ValidateAfter:  validateAfter,
	}

	if dryRun {
		fmt.Println("🔍 Running in dry-run mode - no actual imports will be performed")
	}

	result, err := importEngine.Import(config)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	fmt.Printf("✅ Import completed: %d successful, %d failed\n",
		result.Successful, result.Failed)

	return nil
}
