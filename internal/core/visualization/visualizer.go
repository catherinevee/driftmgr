package visualization

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// Visualizer provides unified visualization capabilities
type Visualizer struct {
	renderers map[string]Renderer
	exporters map[string]Exporter
}

// Renderer defines the interface for visualization renderers
type Renderer interface {
	Render(data *VisualizationData) ([]byte, error)
	Format() string
}

// Exporter defines the interface for visualization exporters
type Exporter interface {
	Export(data *VisualizationData, writer io.Writer) error
	Format() string
}

// VisualizationData contains data for visualization
type VisualizationData struct {
	Resources     []models.Resource      `json:"resources"`
	Drifts        []models.DriftItem     `json:"drifts,omitempty"`
	Relationships []Relationship         `json:"relationships,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Options       VisualizationOptions   `json:"options"`
}

// Relationship represents a relationship between resources
type Relationship struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
	Label  string `json:"label,omitempty"`
}

// VisualizationOptions configures visualization
type VisualizationOptions struct {
	Format       string                 `json:"format"` // "mermaid", "graphviz", "d3", "ascii"
	Layout       string                 `json:"layout"` // "hierarchical", "circular", "force"
	ShowLabels   bool                   `json:"show_labels"`
	ShowDrift    bool                   `json:"show_drift"`
	GroupBy      string                 `json:"group_by"` // "provider", "region", "type", "service"
	FilterEmpty  bool                   `json:"filter_empty"`
	ColorScheme  string                 `json:"color_scheme"`
	CustomStyles map[string]string      `json:"custom_styles"`
	Extra        map[string]interface{} `json:"extra"`
}

// NewVisualizer creates a new visualizer
func NewVisualizer() *Visualizer {
	v := &Visualizer{
		renderers: make(map[string]Renderer),
		exporters: make(map[string]Exporter),
	}

	// Register built-in renderers
	v.RegisterRenderer("mermaid", NewMermaidRenderer())
	v.RegisterRenderer("graphviz", NewGraphvizRenderer())
	v.RegisterRenderer("ascii", NewASCIIRenderer())
	v.RegisterRenderer("json", NewJSONRenderer())

	// Register built-in exporters
	v.RegisterExporter("html", NewHTMLExporter())
	v.RegisterExporter("svg", NewSVGExporter())
	v.RegisterExporter("png", NewPNGExporter())

	return v
}

// RegisterRenderer registers a visualization renderer
func (v *Visualizer) RegisterRenderer(format string, renderer Renderer) {
	v.renderers[format] = renderer
}

// RegisterExporter registers a visualization exporter
func (v *Visualizer) RegisterExporter(format string, exporter Exporter) {
	v.exporters[format] = exporter
}

// Visualize creates a visualization of resources
func (v *Visualizer) Visualize(resources []models.Resource, options VisualizationOptions) ([]byte, error) {
	// Build visualization data
	data := &VisualizationData{
		Resources:     resources,
		Relationships: v.buildRelationships(resources),
		Options:       options,
		Metadata: map[string]interface{}{
			"total_resources": len(resources),
			"providers":       v.getProviders(resources),
			"regions":         v.getRegions(resources),
		},
	}

	// Get appropriate renderer
	renderer, exists := v.renderers[options.Format]
	if !exists {
		renderer = v.renderers["mermaid"] // Default to mermaid
	}

	return renderer.Render(data)
}

// VisualizeDrift creates a visualization highlighting drift
func (v *Visualizer) VisualizeDrift(resources []models.Resource, drifts []models.DriftItem, options VisualizationOptions) ([]byte, error) {
	options.ShowDrift = true

	data := &VisualizationData{
		Resources:     resources,
		Drifts:        drifts,
		Relationships: v.buildRelationships(resources),
		Options:       options,
		Metadata: map[string]interface{}{
			"total_resources": len(resources),
			"total_drifts":    len(drifts),
			"drift_rate":      float64(len(drifts)) / float64(len(resources)) * 100,
		},
	}

	renderer, exists := v.renderers[options.Format]
	if !exists {
		renderer = v.renderers["mermaid"]
	}

	return renderer.Render(data)
}

// Export exports visualization to a specific format
func (v *Visualizer) Export(data *VisualizationData, format string, writer io.Writer) error {
	exporter, exists := v.exporters[format]
	if !exists {
		return fmt.Errorf("unsupported export format: %s", format)
	}

	return exporter.Export(data, writer)
}

// buildRelationships builds relationships between resources
func (v *Visualizer) buildRelationships(resources []models.Resource) []Relationship {
	relationships := []Relationship{}
	resourceMap := make(map[string]models.Resource)

	for _, r := range resources {
		resourceMap[r.ID] = r
	}

	for _, resource := range resources {
		// Extract dependencies from properties
		if deps, ok := resource.Properties["dependencies"].([]string); ok {
			for _, dep := range deps {
				if _, exists := resourceMap[dep]; exists {
					relationships = append(relationships, Relationship{
						Source: resource.ID,
						Target: dep,
						Type:   "depends_on",
						Label:  "depends on",
					})
				}
			}
		}

		// Infer relationships based on resource types
		relationships = append(relationships, v.inferRelationships(resource, resources)...)
	}

	return relationships
}

// inferRelationships infers relationships based on resource types and properties
func (v *Visualizer) inferRelationships(resource models.Resource, allResources []models.Resource) []Relationship {
	relationships := []Relationship{}

	// Example: EC2 instances depend on VPCs
	if strings.Contains(resource.Type, "instance") {
		if vpcID, ok := resource.Properties["vpc_id"].(string); ok {
			for _, r := range allResources {
				if r.ID == vpcID || (r.Type == "vpc" && r.Properties["id"] == vpcID) {
					relationships = append(relationships, Relationship{
						Source: resource.ID,
						Target: r.ID,
						Type:   "network",
						Label:  "in vpc",
					})
					break
				}
			}
		}
	}

	// Example: Databases depend on subnets
	if strings.Contains(resource.Type, "database") || strings.Contains(resource.Type, "rds") {
		if subnetID, ok := resource.Properties["subnet_id"].(string); ok {
			for _, r := range allResources {
				if r.ID == subnetID || r.Type == "subnet" {
					relationships = append(relationships, Relationship{
						Source: resource.ID,
						Target: r.ID,
						Type:   "network",
						Label:  "in subnet",
					})
					break
				}
			}
		}
	}

	return relationships
}

// getProviders extracts unique providers from resources
func (v *Visualizer) getProviders(resources []models.Resource) []string {
	providerMap := make(map[string]bool)
	for _, r := range resources {
		providerMap[r.Provider] = true
	}

	providers := make([]string, 0, len(providerMap))
	for p := range providerMap {
		providers = append(providers, p)
	}
	sort.Strings(providers)
	return providers
}

// getRegions extracts unique regions from resources
func (v *Visualizer) getRegions(resources []models.Resource) []string {
	regionMap := make(map[string]bool)
	for _, r := range resources {
		if r.Region != "" {
			regionMap[r.Region] = true
		}
	}

	regions := make([]string, 0, len(regionMap))
	for r := range regionMap {
		regions = append(regions, r)
	}
	sort.Strings(regions)
	return regions
}

// GroupResources groups resources by specified criteria
func (v *Visualizer) GroupResources(resources []models.Resource, groupBy string) map[string][]models.Resource {
	groups := make(map[string][]models.Resource)

	for _, resource := range resources {
		var key string
		switch groupBy {
		case "provider":
			key = resource.Provider
		case "region":
			key = resource.Region
		case "type":
			key = resource.Type
		case "service":
			// Extract service from resource type (e.g., "aws_ec2_instance" -> "ec2")
			parts := strings.Split(resource.Type, "_")
			if len(parts) > 1 {
				key = parts[1]
			} else {
				key = "other"
			}
		default:
			key = "default"
		}

		if key == "" {
			key = "unknown"
		}

		groups[key] = append(groups[key], resource)
	}

	return groups
}

// GenerateSummary generates a summary of visualization data
func (v *Visualizer) GenerateSummary(data *VisualizationData) map[string]interface{} {
	summary := map[string]interface{}{
		"total_resources":     len(data.Resources),
		"total_relationships": len(data.Relationships),
		"total_drifts":        len(data.Drifts),
	}

	// Count resources by provider
	byProvider := make(map[string]int)
	for _, r := range data.Resources {
		byProvider[r.Provider]++
	}
	summary["by_provider"] = byProvider

	// Count resources by type
	byType := make(map[string]int)
	for _, r := range data.Resources {
		byType[r.Type]++
	}
	summary["by_type"] = byType

	// Count relationships by type
	byRelType := make(map[string]int)
	for _, rel := range data.Relationships {
		byRelType[rel.Type]++
	}
	summary["by_relationship_type"] = byRelType

	// Drift summary if present
	if len(data.Drifts) > 0 {
		bySeverity := make(map[string]int)
		for _, d := range data.Drifts {
			bySeverity[d.Severity]++
		}
		summary["drifts_by_severity"] = bySeverity
	}

	return summary
}
