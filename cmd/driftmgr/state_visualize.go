package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/catherinevee/driftmgr/internal/core/models"
	"github.com/catherinevee/driftmgr/internal/core/visualization"
	"github.com/catherinevee/driftmgr/internal/terraform/state"
)

// handleStateVisualize handles the "state visualize" command
func handleStateVisualize(args []string) {
	var statePath string
	var outputDir string = "./visualizations"
	var format string = "all"
	var showDependencies bool = true
	var theme string = "light"

	// Parse arguments
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--state", "-s":
			if i+1 < len(args) {
				statePath = args[i+1]
				i++
			}
		case "--output", "-o":
			if i+1 < len(args) {
				outputDir = args[i+1]
				i++
			}
		case "--format", "-f":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		case "--theme":
			if i+1 < len(args) {
				theme = args[i+1]
				i++
			}
		case "--no-dependencies":
			showDependencies = false
		case "--help", "-h":
			fmt.Println("Usage: driftmgr state visualize [flags]")
			fmt.Println()
			fmt.Println("Generate visual representations of Terraform state files")
			fmt.Println()
			fmt.Println("Flags:")
			fmt.Println("  --state, -s string    Path to state file (required)")
			fmt.Println("  --output, -o string   Output directory (default: ./visualizations)")
			fmt.Println("  --format, -f string   Output format: html, svg, ascii, mermaid, dot, all (default: all)")
			fmt.Println("  --theme string        Color theme: light, dark (default: light)")
			fmt.Println("  --no-dependencies     Don't show resource dependencies")
			fmt.Println()
			fmt.Println("Output Formats:")
			fmt.Println("  html       - Interactive HTML dashboard with D3.js")
			fmt.Println("  svg        - Scalable Vector Graphics diagram")
			fmt.Println("  ascii      - Terminal-friendly ASCII art diagram")
			fmt.Println("  mermaid    - Mermaid diagram (for documentation)")
			fmt.Println("  dot        - Graphviz DOT format")
			fmt.Println("  terravision - Terravision-style infrastructure diagram")
			fmt.Println("  all        - Generate all formats")
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  driftmgr state visualize --state terraform.tfstate")
			fmt.Println("  driftmgr state visualize --state azure.tfstate --format html --theme dark")
			fmt.Println("  driftmgr state visualize --state s3://bucket/key --output ./diagrams")
			return
		}
	}

	if statePath == "" {
		// Try to find state file in current directory
		if _, err := os.Stat("terraform.tfstate"); err == nil {
			statePath = "terraform.tfstate"
		} else {
			fmt.Println("Error: State file path required. Use --state flag")
			fmt.Println("Run 'driftmgr state visualize --help' for usage")
			os.Exit(1)
		}
	}

	// Load the state file
	fmt.Printf("Loading state file: %s\n", statePath)
	loader := state.NewStateLoader(statePath)

	ctx := context.Background()
	stateFile, err := loader.LoadStateFile(ctx, statePath, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading state file: %v\n", err)
		os.Exit(1)
	}

	// Convert state resources to visualization models
	var resources []models.Resource
	for _, r := range stateFile.Resources {
		resources = append(resources, models.Resource{
			ID:       r.Name,
			Name:     r.Name,
			Type:     r.Type,
			Provider: r.Provider,
		})
	}

	fmt.Printf("Found %d resources to visualize\n", len(resources))

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate visualizations based on format
	formats := []string{}
	if format == "all" {
		formats = []string{"html", "svg", "ascii", "mermaid", "dot", "terravision"}
	} else {
		formats = strings.Split(format, ",")
	}

	for _, fmt := range formats {
		// Create a temporary StateFile wrapper for visualization
		modelStateFile := &models.StateFile{
			Path: statePath,
		}
		generateVisualization(resources, modelStateFile, outputDir, fmt, theme, showDependencies)
	}

	fmt.Printf("\nVisualizations generated in: %s\n", outputDir)
}

func convertStateToResources(stateFile *models.StateFile) []models.Resource {
	var resources []models.Resource

	for _, tfResource := range stateFile.Resources {
		// Create a resource for visualization
		resource := models.Resource{
			ID:       tfResource.Name,
			Name:     tfResource.Name,
			Type:     tfResource.Type,
			Provider: extractProviderName(tfResource.Provider),
		}

		// Extract attributes from first instance if available
		if len(tfResource.Instances) > 0 && tfResource.Instances[0].Attributes != nil {
			attrs := tfResource.Instances[0].Attributes

			// Try to get common attributes
			if id, ok := attrs["id"].(string); ok {
				resource.ID = id
			}
			if region, ok := attrs["region"].(string); ok {
				resource.Region = region
			} else if location, ok := attrs["location"].(string); ok {
				resource.Region = location
			}

			// Store all attributes for detailed visualization
			resource.Attributes = attrs
		}

		resources = append(resources, resource)
	}

	return resources
}

func extractProviderName(provider string) string {
	// Extract provider name from format like "provider[\"registry.terraform.io/hashicorp/aws\"]"
	parts := strings.Split(provider, "/")
	if len(parts) > 0 {
		providerPart := parts[len(parts)-1]
		providerPart = strings.TrimSuffix(providerPart, "\"]")
		return providerPart
	}
	return provider
}

func generateVisualization(resources []models.Resource, stateFile *models.StateFile, outputDir, format, theme string, showDeps bool) {
	fmt.Printf("Generating %s visualization...\n", format)

	switch format {
	case "ascii":
		generateASCIIVisualization(resources, stateFile, outputDir)
	case "html":
		generateHTMLVisualization(resources, stateFile, outputDir, theme, showDeps)
	case "svg":
		generateSVGVisualization(resources, stateFile, outputDir, theme)
	case "mermaid":
		generateMermaidDiagram(resources, stateFile, outputDir)
	case "dot":
		generateDOTGraph(resources, stateFile, outputDir)
	case "terravision":
		generateTerravisionVisualization(stateFile, outputDir, theme)
	default:
		fmt.Printf("Warning: Unknown format %s\n", format)
	}
}

func generateASCIIVisualization(resources []models.Resource, stateFile *models.StateFile, outputDir string) {
	config := visualization.DefaultVisualizationConfig()
	visualizer := visualization.NewSimpleVisualizer(config)

	// Generate resource tree
	tree, _ := visualizer.GenerateASCII(resources)

	// Save to file
	outputPath := filepath.Join(outputDir, "state_diagram.txt")
	content := fmt.Sprintf("TERRAFORM STATE VISUALIZATION\n")
	content += fmt.Sprintf("State File: %s\n", stateFile.Path)
	content += fmt.Sprintf("Version: %d | Terraform: %s\n", stateFile.Version, stateFile.TerraformVersion)
	content += fmt.Sprintf("Resources: %d\n", len(resources))
	content += strings.Repeat("=", 80) + "\n\n"
	content += tree

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		fmt.Printf("Error writing ASCII visualization: %v\n", err)
	} else {
		fmt.Printf("  ✓ ASCII diagram saved to %s\n", outputPath)
	}
}

func generateHTMLVisualization(resources []models.Resource, stateFile *models.StateFile, outputDir, theme string, showDeps bool) {
	config := visualization.DefaultVisualizationConfig()
	config.Format = "html"
	config.ShowDetails = showDeps

	visualizer := visualization.NewEnhancedVisualizer(config)

	result, err := visualizer.GenerateMermaid(resources)
	if err != nil {
		fmt.Printf("Error generating HTML visualization: %v\n", err)
		return
	}

	outputPath := filepath.Join(outputDir, "state_interactive.html")
	fmt.Printf("  ✓ Interactive HTML saved to %s\n", outputPath)

	if result != "" {
		// Save the result to file
		if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
			fmt.Printf("Error writing HTML file: %v\n", err)
		} else {
			fmt.Printf("    Open in browser: file://%s\n", filepath.ToSlash(outputPath))
		}
	}
}

func generateSVGVisualization(resources []models.Resource, stateFile *models.StateFile, outputDir, theme string) {
	// Generate SVG using a simple box layout
	svg := generateSimpleSVG(resources, stateFile, theme)

	outputPath := filepath.Join(outputDir, "state_diagram.svg")
	if err := os.WriteFile(outputPath, []byte(svg), 0644); err != nil {
		fmt.Printf("Error writing SVG visualization: %v\n", err)
	} else {
		fmt.Printf("  ✓ SVG diagram saved to %s\n", outputPath)
	}
}

func generateMermaidDiagram(resources []models.Resource, stateFile *models.StateFile, outputDir string) {
	mermaid := "graph TB\n"
	mermaid += fmt.Sprintf("    State[State File: %s]\n", filepath.Base(stateFile.Path))

	// Group resources by provider
	providers := make(map[string][]models.Resource)
	for _, res := range resources {
		providers[res.Provider] = append(providers[res.Provider], res)
	}

	for provider, providerResources := range providers {
		providerID := strings.ReplaceAll(provider, "-", "_")
		mermaid += fmt.Sprintf("    State --> %s[%s Provider]\n", providerID, provider)

		// Group by resource type
		types := make(map[string][]models.Resource)
		for _, res := range providerResources {
			types[res.Type] = append(types[res.Type], res)
		}

		for resType, typeResources := range types {
			typeID := strings.ReplaceAll(resType, ".", "_")
			mermaid += fmt.Sprintf("    %s --> %s[%s]\n", providerID, typeID, resType)

			for i, res := range typeResources {
				resID := fmt.Sprintf("%s_%d", typeID, i)
				mermaid += fmt.Sprintf("    %s --> %s[%s]\n", typeID, resID, res.Name)
			}
		}
	}

	outputPath := filepath.Join(outputDir, "state_diagram.mermaid")
	if err := os.WriteFile(outputPath, []byte(mermaid), 0644); err != nil {
		fmt.Printf("Error writing Mermaid diagram: %v\n", err)
	} else {
		fmt.Printf("  ✓ Mermaid diagram saved to %s\n", outputPath)
		fmt.Printf("    View at: https://mermaid.live/\n")
	}
}

func generateDOTGraph(resources []models.Resource, stateFile *models.StateFile, outputDir string) {
	dot := "digraph TerraformState {\n"
	dot += "    rankdir=TB;\n"
	dot += "    node [shape=box];\n"
	dot += fmt.Sprintf("    \"State\" [label=\"%s\\nVersion: %d\", shape=folder];\n",
		filepath.Base(stateFile.Path), stateFile.Version)

	// Group resources by provider
	providers := make(map[string][]models.Resource)
	for _, res := range resources {
		providers[res.Provider] = append(providers[res.Provider], res)
	}

	for provider, providerResources := range providers {
		providerNode := fmt.Sprintf("provider_%s", provider)
		dot += fmt.Sprintf("    \"%s\" [label=\"%s\", shape=component, style=filled, fillcolor=lightblue];\n",
			providerNode, provider)
		dot += fmt.Sprintf("    \"State\" -> \"%s\";\n", providerNode)

		for i, res := range providerResources {
			resNode := fmt.Sprintf("%s_%s_%d", provider, res.Type, i)
			label := fmt.Sprintf("%s\\n%s", res.Type, res.Name)
			if res.Region != "" {
				label += fmt.Sprintf("\\n[%s]", res.Region)
			}
			dot += fmt.Sprintf("    \"%s\" [label=\"%s\"];\n", resNode, label)
			dot += fmt.Sprintf("    \"%s\" -> \"%s\";\n", providerNode, resNode)
		}
	}

	dot += "}\n"

	outputPath := filepath.Join(outputDir, "state_graph.dot")
	if err := os.WriteFile(outputPath, []byte(dot), 0644); err != nil {
		fmt.Printf("Error writing DOT graph: %v\n", err)
	} else {
		fmt.Printf("  ✓ Graphviz DOT saved to %s\n", outputPath)
		fmt.Printf("    Generate PNG: dot -Tpng %s -o state_graph.png\n", outputPath)
	}
}

func generateTerravisionVisualization(stateFile *models.StateFile, outputDir, theme string) {
	options := visualization.DefaultTerravisionOptions()
	options.OutputPath = filepath.Join(outputDir, "terravision.html")

	visualizer := visualization.NewTerravisionVisualizer(options)

	fmt.Printf("  ℹ Generating Terravision visualization...\n")

	// Generate Terravision diagram
	if err := visualizer.GenerateTerravision(stateFile); err != nil {
		fmt.Printf("Error generating Terravision diagram: %v\n", err)
	} else {
		fmt.Printf("  ✓ Terravision diagram saved to %s\n", options.OutputPath)
	}
}

func generateSimpleSVG(resources []models.Resource, stateFile *models.StateFile, theme string) string {
	width := 800
	height := 600

	// SVG header
	svg := fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">`, width, height)

	// Background
	bgColor := "#ffffff"
	textColor := "#000000"
	if theme == "dark" {
		bgColor = "#1a1a1a"
		textColor = "#ffffff"
	}

	svg += fmt.Sprintf(`<rect width="%d" height="%d" fill="%s"/>`, width, height, bgColor)

	// Title
	svg += fmt.Sprintf(`<text x="%d" y="30" text-anchor="middle" font-size="20" fill="%s">%s</text>`,
		width/2, textColor, filepath.Base(stateFile.Path))

	// Group resources by provider
	providers := make(map[string][]models.Resource)
	for _, res := range resources {
		providers[res.Provider] = append(providers[res.Provider], res)
	}

	// Draw provider boxes
	y := 80
	providerColors := map[string]string{
		"aws":          "#FF9900",
		"azure":        "#0078D4",
		"azurerm":      "#0078D4",
		"gcp":          "#4285F4",
		"google":       "#4285F4",
		"digitalocean": "#0080FF",
	}

	for provider, providerResources := range providers {
		color := providerColors[provider]
		if color == "" {
			color = "#666666"
		}

		// Provider box
		svg += fmt.Sprintf(`<rect x="50" y="%d" width="700" height="%d" fill="%s" opacity="0.1" stroke="%s" stroke-width="2"/>`,
			y, 50+len(providerResources)*30, color, color)

		// Provider label
		svg += fmt.Sprintf(`<text x="70" y="%d" font-size="16" font-weight="bold" fill="%s">%s (%d resources)</text>`,
			y+25, textColor, strings.ToUpper(provider), len(providerResources))

		// Resource list
		resourceY := y + 40
		for _, res := range providerResources {
			svg += fmt.Sprintf(`<text x="100" y="%d" font-size="12" fill="%s">• %s.%s</text>`,
				resourceY, textColor, res.Type, res.Name)
			resourceY += 20
		}

		y += 60 + len(providerResources)*20
	}

	svg += "</svg>"
	return svg
}
