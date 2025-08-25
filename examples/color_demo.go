package main

import (
	"fmt"
	"github.com/catherinevee/driftmgr/internal/core/color"
)

func main() {
	fmt.Println(color.Header("DriftMgr Color Demo"))
	fmt.Println(color.DoubleDivider())

	// Provider colors
	fmt.Println(color.Subheader("Provider Colors:"))
	fmt.Printf("  %s\n", color.AWS("AWS Provider"))
	fmt.Printf("  %s\n", color.Azure("Azure Provider"))
	fmt.Printf("  %s\n", color.GCP("GCP Provider"))
	fmt.Printf("  %s\n", color.DigitalOcean("DigitalOcean Provider"))
	fmt.Println()

	// Status colors
	fmt.Println(color.Subheader("Status Colors:"))
	fmt.Printf("  %s %s\n", color.CheckMark(), color.Success("Success message"))
	fmt.Printf("  %s %s\n", color.CrossMark(), color.Error("Error message"))
	fmt.Printf("  %s %s\n", color.StatusIcon("warning"), color.Warning("Warning message"))
	fmt.Printf("  %s %s\n", color.Arrow(), color.Info("Info message"))
	fmt.Printf("  %s %s\n", color.Bullet(), color.Dim("Dimmed text"))
	fmt.Println()

	// Semantic colors
	fmt.Println(color.Subheader("Semantic Elements:"))
	fmt.Printf("  %s %s\n", color.Label("Label:"), color.Value("Value"))
	fmt.Printf("  %s\n", color.Command("driftmgr discover"))
	fmt.Printf("  %s %s\n", color.Flag("--all-accounts"), color.Dim("Include all accounts"))
	fmt.Printf("  %s\n", color.Path("/path/to/config.yaml"))
	fmt.Println()

	// Severity colors
	fmt.Println(color.Subheader("Drift Severity:"))
	fmt.Printf("  %s\n", color.Critical("Critical - Immediate action required"))
	fmt.Printf("  %s\n", color.High("High - Significant drift detected"))
	fmt.Printf("  %s\n", color.Medium("Medium - Moderate drift detected"))
	fmt.Printf("  %s\n", color.Low("Low - Minor drift detected"))
	fmt.Println()

	// Resource counts
	fmt.Println(color.Subheader("Resource Counts:"))
	fmt.Printf("  Small count: %s resources\n", color.Count(5))
	fmt.Printf("  Medium count: %s resources\n", color.Count(150))
	fmt.Printf("  Large count: %s resources\n", color.Count(750))
	fmt.Printf("  Zero count: %s resources\n", color.Count(0))
	fmt.Println()

	// Formatting
	fmt.Println(color.Subheader("Text Formatting:"))
	fmt.Printf("  %s\n", color.Bold("Bold text"))
	fmt.Printf("  %s\n", color.Underline("Underlined text"))
	fmt.Printf("  %s\n", color.DimText("Dimmed text"))
	fmt.Println()

	// Dividers
	fmt.Println(color.Subheader("Dividers:"))
	fmt.Println(color.Divider())
	fmt.Println("Content between dividers")
	fmt.Println(color.DoubleDivider())
	fmt.Println()

	// Combined example
	fmt.Println(color.Header("Complete Example"))
	fmt.Println(color.Divider())

	fmt.Printf("%s %s\n", color.AWS("AWS:"), color.Success(color.CheckMark()+" Configured"))
	fmt.Printf("  %s %s\n", color.Label("Account:"), color.Value("123456789012"))
	fmt.Printf("  %s %s\n", color.Label("Region:"), color.Value("us-west-2"))
	fmt.Printf("  %s\n", color.Subheader("Available profiles:"))
	fmt.Printf("    %s default %s\n", color.Bullet(), color.Success("(current)"))
	fmt.Printf("    %s production\n", color.Bullet())
	fmt.Printf("    %s staging\n", color.Bullet())

	fmt.Println(color.Divider())

	fmt.Printf("\n%s Discovered %s resources\n", color.Info("Discovery complete:"), color.Count(342))
	fmt.Printf("%s %s drift items detected\n", color.Warning("Warning:"), color.Count(15))
	fmt.Printf("  %s %s\n", color.Arrow(), color.Critical("3 critical"))
	fmt.Printf("  %s %s\n", color.Arrow(), color.High("5 high"))
	fmt.Printf("  %s %s\n", color.Arrow(), color.Medium("7 medium"))
}
