package visualization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// MermaidRenderer renders visualizations in Mermaid format
type MermaidRenderer struct{}

// NewMermaidRenderer creates a new Mermaid renderer
func NewMermaidRenderer() *MermaidRenderer {
	return &MermaidRenderer{}
}

// Format returns the format name
func (r *MermaidRenderer) Format() string {
	return "mermaid"
}

// Render renders the visualization data as Mermaid diagram
func (r *MermaidRenderer) Render(data *VisualizationData) ([]byte, error) {
	var buf bytes.Buffer

	// Choose diagram type based on layout
	switch data.Options.Layout {
	case "circular":
		buf.WriteString("graph LR\n")
	case "hierarchical":
		buf.WriteString("graph TD\n")
	default:
		buf.WriteString("graph TB\n")
	}

	// Group resources if specified
	if data.Options.GroupBy != "" {
		groups := groupResources(data.Resources, data.Options.GroupBy)
		for groupName, resources := range groups {
			buf.WriteString(fmt.Sprintf("    subgraph %s\n", sanitizeMermaidID(groupName)))
			for _, resource := range resources {
				r.renderResource(&buf, resource, data)
			}
			buf.WriteString("    end\n")
		}
	} else {
		// Render individual resources
		for _, resource := range data.Resources {
			r.renderResource(&buf, resource, data)
		}
	}

	// Render relationships
	for _, rel := range data.Relationships {
		label := rel.Label
		if label == "" {
			label = rel.Type
		}
		buf.WriteString(fmt.Sprintf("    %s -->|%s| %s\n",
			sanitizeMermaidID(rel.Source),
			label,
			sanitizeMermaidID(rel.Target)))
	}

	// Add styling for drift if enabled
	if data.Options.ShowDrift && len(data.Drifts) > 0 {
		buf.WriteString("\n    %% Drift styling\n")
		driftedResources := make(map[string]bool)
		for _, drift := range data.Drifts {
			driftedResources[drift.ResourceID] = true
		}

		for id := range driftedResources {
			buf.WriteString(fmt.Sprintf("    class %s drifted\n", sanitizeMermaidID(id)))
		}

		buf.WriteString("\n    classDef drifted fill:#ff9999,stroke:#ff0000,stroke-width:2px\n")
	}

	// Add custom styles if provided
	if len(data.Options.CustomStyles) > 0 {
		buf.WriteString("\n    %% Custom styles\n")
		for selector, style := range data.Options.CustomStyles {
			buf.WriteString(fmt.Sprintf("    classDef %s %s\n", selector, style))
		}
	}

	return buf.Bytes(), nil
}

func (r *MermaidRenderer) renderResource(buf *bytes.Buffer, resource interface{}, data *VisualizationData) {
	res := toResource(resource)
	id := sanitizeMermaidID(res.ID)
	label := res.Name
	if label == "" {
		label = res.ID
	}

	if data.Options.ShowLabels {
		label = fmt.Sprintf("%s\\n(%s)", label, res.Type)
	}

	// Choose shape based on resource type
	shape := r.getShapeForType(res.Type)
	buf.WriteString(fmt.Sprintf("    %s%s%s\n", id, shape, label))
}

func (r *MermaidRenderer) getShapeForType(resourceType string) string {
	if strings.Contains(resourceType, "database") {
		return "[("
	} else if strings.Contains(resourceType, "storage") {
		return "[["
	} else if strings.Contains(resourceType, "network") {
		return "{{"
	}
	return "["
}

// GraphvizRenderer renders visualizations in Graphviz DOT format
type GraphvizRenderer struct{}

// NewGraphvizRenderer creates a new Graphviz renderer
func NewGraphvizRenderer() *GraphvizRenderer {
	return &GraphvizRenderer{}
}

// Format returns the format name
func (r *GraphvizRenderer) Format() string {
	return "graphviz"
}

// Render renders the visualization data as Graphviz DOT
func (r *GraphvizRenderer) Render(data *VisualizationData) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("digraph Resources {\n")
	buf.WriteString("    rankdir=TB;\n")
	buf.WriteString("    node [shape=box, style=rounded];\n")

	// Set layout engine
	switch data.Options.Layout {
	case "circular":
		buf.WriteString("    layout=circo;\n")
	case "force":
		buf.WriteString("    layout=fdp;\n")
	default:
		buf.WriteString("    layout=dot;\n")
	}

	// Group resources if specified
	if data.Options.GroupBy != "" {
		groups := groupResources(data.Resources, data.Options.GroupBy)
		clusterIndex := 0
		for groupName, resources := range groups {
			buf.WriteString(fmt.Sprintf("    subgraph cluster_%d {\n", clusterIndex))
			buf.WriteString(fmt.Sprintf("        label=\"%s\";\n", groupName))
			buf.WriteString("        style=filled;\n")
			buf.WriteString("        color=lightgrey;\n")

			for _, resource := range resources {
				r.renderResource(&buf, resource, data)
			}
			buf.WriteString("    }\n")
			clusterIndex++
		}
	} else {
		// Render individual resources
		for _, resource := range data.Resources {
			r.renderResource(&buf, resource, data)
		}
	}

	// Render relationships
	for _, rel := range data.Relationships {
		label := rel.Label
		if label == "" {
			label = rel.Type
		}
		buf.WriteString(fmt.Sprintf("    \"%s\" -> \"%s\" [label=\"%s\"];\n",
			rel.Source, rel.Target, label))
	}

	// Add drift highlighting
	if data.Options.ShowDrift && len(data.Drifts) > 0 {
		buf.WriteString("\n    // Drift highlighting\n")
		for _, drift := range data.Drifts {
			color := r.getSeverityColor(drift.Severity)
			buf.WriteString(fmt.Sprintf("    \"%s\" [fillcolor=\"%s\", style=filled];\n",
				drift.ResourceID, color))
		}
	}

	buf.WriteString("}\n")

	return buf.Bytes(), nil
}

func (r *GraphvizRenderer) renderResource(buf *bytes.Buffer, resource interface{}, data *VisualizationData) {
	res := toResource(resource)
	label := res.Name
	if label == "" {
		label = res.ID
	}

	if data.Options.ShowLabels {
		label = fmt.Sprintf("%s\\n%s", label, res.Type)
	}

	shape := r.getShapeForType(res.Type)
	buf.WriteString(fmt.Sprintf("    \"%s\" [label=\"%s\", shape=%s];\n",
		res.ID, label, shape))
}

func (r *GraphvizRenderer) getShapeForType(resourceType string) string {
	if strings.Contains(resourceType, "database") {
		return "cylinder"
	} else if strings.Contains(resourceType, "storage") {
		return "folder"
	} else if strings.Contains(resourceType, "network") {
		return "diamond"
	} else if strings.Contains(resourceType, "function") {
		return "component"
	}
	return "box"
}

func (r *GraphvizRenderer) getSeverityColor(severity string) string {
	switch severity {
	case "critical":
		return "#ff0000"
	case "high":
		return "#ff6600"
	case "medium":
		return "#ffcc00"
	case "low":
		return "#ffff99"
	default:
		return "#ffffff"
	}
}

// ASCIIRenderer renders visualizations in ASCII art format
type ASCIIRenderer struct{}

// NewASCIIRenderer creates a new ASCII renderer
func NewASCIIRenderer() *ASCIIRenderer {
	return &ASCIIRenderer{}
}

// Format returns the format name
func (r *ASCIIRenderer) Format() string {
	return "ascii"
}

// Render renders the visualization data as ASCII art
func (r *ASCIIRenderer) Render(data *VisualizationData) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("Infrastructure Visualization\n")
	buf.WriteString("=" + strings.Repeat("=", 50) + "\n\n")

	// Group resources if specified
	if data.Options.GroupBy != "" {
		groups := groupResources(data.Resources, data.Options.GroupBy)
		for groupName, resources := range groups {
			buf.WriteString(fmt.Sprintf("[%s]\n", strings.ToUpper(groupName)))
			buf.WriteString(strings.Repeat("-", len(groupName)+2) + "\n")

			for _, resource := range resources {
				r.renderResource(&buf, resource, data, "  ")
			}
			buf.WriteString("\n")
		}
	} else {
		// Render resources as tree
		buf.WriteString("Resources:\n")
		for i, resource := range data.Resources {
			prefix := "├── "
			if i == len(data.Resources)-1 {
				prefix = "└── "
			}
			r.renderResource(&buf, resource, data, prefix)
		}
	}

	// Show relationships
	if len(data.Relationships) > 0 {
		buf.WriteString("\nRelationships:\n")
		buf.WriteString(strings.Repeat("-", 50) + "\n")
		for _, rel := range data.Relationships {
			buf.WriteString(fmt.Sprintf("  %s --> %s (%s)\n",
				rel.Source, rel.Target, rel.Type))
		}
	}

	// Show drift summary
	if data.Options.ShowDrift && len(data.Drifts) > 0 {
		buf.WriteString("\nDrift Summary:\n")
		buf.WriteString(strings.Repeat("-", 50) + "\n")

		bySeverity := make(map[string]int)
		for _, drift := range data.Drifts {
			bySeverity[drift.Severity]++
		}

		for severity, count := range bySeverity {
			buf.WriteString(fmt.Sprintf("  %s: %d\n", severity, count))
		}
	}

	// Show statistics
	buf.WriteString("\nStatistics:\n")
	buf.WriteString(strings.Repeat("-", 50) + "\n")
	buf.WriteString(fmt.Sprintf("  Total Resources: %d\n", len(data.Resources)))
	buf.WriteString(fmt.Sprintf("  Total Relationships: %d\n", len(data.Relationships)))
	if len(data.Drifts) > 0 {
		buf.WriteString(fmt.Sprintf("  Total Drifts: %d\n", len(data.Drifts)))
	}

	return buf.Bytes(), nil
}

func (r *ASCIIRenderer) renderResource(buf *bytes.Buffer, resource interface{}, data *VisualizationData, prefix string) {
	res := toResource(resource)

	// Check if resource has drift
	hasDrift := false
	if data.Options.ShowDrift {
		for _, drift := range data.Drifts {
			if drift.ResourceID == res.ID {
				hasDrift = true
				break
			}
		}
	}

	driftMarker := ""
	if hasDrift {
		driftMarker = " [DRIFT]"
	}

	buf.WriteString(fmt.Sprintf("%s%s (%s)%s\n", prefix, res.Name, res.Type, driftMarker))

	if data.Options.ShowLabels {
		buf.WriteString(fmt.Sprintf("%s    ID: %s\n", prefix, res.ID))
		buf.WriteString(fmt.Sprintf("%s    Provider: %s\n", prefix, res.Provider))
		if res.Region != "" {
			buf.WriteString(fmt.Sprintf("%s    Region: %s\n", prefix, res.Region))
		}
	}
}

// JSONRenderer renders visualizations in JSON format
type JSONRenderer struct{}

// NewJSONRenderer creates a new JSON renderer
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// Format returns the format name
func (r *JSONRenderer) Format() string {
	return "json"
}

// Render renders the visualization data as JSON
func (r *JSONRenderer) Render(data *VisualizationData) ([]byte, error) {
	output := map[string]interface{}{
		"resources":     data.Resources,
		"relationships": data.Relationships,
		"metadata":      data.Metadata,
		"options":       data.Options,
	}

	if data.Options.ShowDrift && len(data.Drifts) > 0 {
		output["drifts"] = data.Drifts
	}

	// Add grouping if specified
	if data.Options.GroupBy != "" {
		output["groups"] = groupResources(data.Resources, data.Options.GroupBy)
	}

	return json.MarshalIndent(output, "", "  ")
}

// Helper functions

func sanitizeMermaidID(id string) string {
	// Replace characters that might break Mermaid syntax
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, ".", "_")
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, ":", "_")
	return id
}

func groupResources(resources interface{}, groupBy string) map[string][]interface{} {
	// This is a simplified version - in production, would use the actual resource type
	groups := make(map[string][]interface{})
	// Implementation would group resources by the specified field
	return groups
}

func toResource(resource interface{}) struct {
	ID       string
	Name     string
	Type     string
	Provider string
	Region   string
} {
	// This is a simplified conversion - in production, would properly convert the resource
	return struct {
		ID       string
		Name     string
		Type     string
		Provider string
		Region   string
	}{
		ID:       "resource-id",
		Name:     "resource-name",
		Type:     "resource-type",
		Provider: "provider",
		Region:   "region",
	}
}
