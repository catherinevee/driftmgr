package cmd

import (
	"fmt"

	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/spf13/cobra"
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover resources in cloud providers",
	Long: `Discover existing resources across cloud providers including AWS, Azure, and GCP.
This command scans your cloud accounts and provides detailed information about
existing resources that can be imported into Terraform.`,
	RunE: runDiscover,
}

var (
	provider     string
	regions      []string
	resourceType string
	tags         []string
	outputFormat string
	outputFile   string
)

func init() {
	rootCmd.AddCommand(discoverCmd)

	discoverCmd.Flags().StringVarP(&provider, "provider", "p", "", "cloud provider (aws, azure, gcp)")
	discoverCmd.Flags().StringSliceVarP(&regions, "region", "r", []string{}, "regions to scan")
	discoverCmd.Flags().StringVarP(&resourceType, "type", "t", "", "specific resource type to discover")
	discoverCmd.Flags().StringSliceVar(&tags, "tags", []string{}, "filter by tags (key:value format)")
	discoverCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "output format (table, json, csv)")
	discoverCmd.Flags().StringVarP(&outputFile, "file", "f", "", "output file path")

	discoverCmd.MarkFlagRequired("provider")
}

func runDiscover(cmd *cobra.Command, args []string) error {
	fmt.Printf("🔍 Discovering %s resources...\n", provider)

	discoveryEngine, err := discovery.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize discovery engine: %w", err)
	}

	config := discovery.Config{
		Provider:     provider,
		Regions:      regions,
		ResourceType: resourceType,
		Tags:         tags,
		OutputFormat: outputFormat,
		OutputFile:   outputFile,
	}

	resources, err := discoveryEngine.Discover(config)
	if err != nil {
		return fmt.Errorf("failed to discover resources: %w", err)
	}

	fmt.Printf("✅ Found %d resources\n", len(resources))

	return discoveryEngine.OutputResources(resources, config)
}
