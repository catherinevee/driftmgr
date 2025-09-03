package resource

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/pkg/models"
)

// DiscoveryVisualizer provides visualization capabilities for discovery results
type DiscoveryVisualizer struct {
	mu              sync.RWMutex
	resources       []models.Resource
	relationships   map[string][]string
	groupedByType   map[string][]models.Resource
	groupedByRegion map[string][]models.Resource
	groupedByProvider map[string][]models.Resource
	timeline        []TimelineEvent
	stats           *DiscoveryStats
}

// TimelineEvent represents an event in the discovery timeline
type TimelineEvent struct {
	Timestamp   time.Time
	EventType   string
	Provider    string
	Region      string
	Service     string
	ResourceID  string
	Description string
}

// DiscoveryStats holds statistics about the discovery
type DiscoveryStats struct {
	TotalResources    int
	ResourcesByType   map[string]int
	ResourcesByRegion map[string]int
	ResourcesByProvider map[string]int
	DiscoveryDuration time.Duration
	ErrorCount        int
	WarningCount      int
	AverageDiscoveryTime time.Duration
}

// NewDiscoveryVisualizer creates a new discovery visualizer
func NewDiscoveryVisualizer() *DiscoveryVisualizer {
	return &DiscoveryVisualizer{
		resources:       make([]models.Resource, 0),
		relationships:   make(map[string][]string),
		groupedByType:   make(map[string][]models.Resource),
		groupedByRegion: make(map[string][]models.Resource),
		groupedByProvider: make(map[string][]models.Resource),
		timeline:        make([]TimelineEvent, 0),
		stats: &DiscoveryStats{
			ResourcesByType:   make(map[string]int),
			ResourcesByRegion: make(map[string]int),
			ResourcesByProvider: make(map[string]int),
		},
	}
}

// AddResource adds a resource to the visualizer
func (dv *DiscoveryVisualizer) AddResource(resource models.Resource) {
	dv.mu.Lock()
	defer dv.mu.Unlock()

	dv.resources = append(dv.resources, resource)
	
	// Update groupings
	dv.groupedByType[resource.Type] = append(dv.groupedByType[resource.Type], resource)
	dv.groupedByRegion[resource.Region] = append(dv.groupedByRegion[resource.Region], resource)
	dv.groupedByProvider[resource.Provider] = append(dv.groupedByProvider[resource.Provider], resource)
	
	// Update stats
	dv.stats.TotalResources++
	dv.stats.ResourcesByType[resource.Type]++
	dv.stats.ResourcesByRegion[resource.Region]++
	dv.stats.ResourcesByProvider[resource.Provider]++
	
	// Add timeline event
	dv.timeline = append(dv.timeline, TimelineEvent{
		Timestamp:   time.Now(),
		EventType:   "resource_discovered",
		Provider:    resource.Provider,
		Region:      resource.Region,
		Service:     resource.Type,
		ResourceID:  resource.ID,
		Description: fmt.Sprintf("Discovered %s resource: %s", resource.Type, resource.Name),
	})
}

// AddRelationship adds a relationship between resources
func (dv *DiscoveryVisualizer) AddRelationship(fromResourceID, toResourceID string) {
	dv.mu.Lock()
	defer dv.mu.Unlock()

	if dv.relationships[fromResourceID] == nil {
		dv.relationships[fromResourceID] = make([]string, 0)
	}
	dv.relationships[fromResourceID] = append(dv.relationships[fromResourceID], toResourceID)
}

// GenerateASCIIDiagram generates an ASCII diagram of resources
func (dv *DiscoveryVisualizer) GenerateASCIIDiagram() string {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	var buf bytes.Buffer
	buf.WriteString("=== Resource Discovery Diagram ===\n\n")
	
	// Group by provider
	for provider, resources := range dv.groupedByProvider {
		buf.WriteString(fmt.Sprintf("┌─ %s (%d resources) ─┐\n", provider, len(resources)))
		
		// Group by region within provider
		regionMap := make(map[string][]models.Resource)
		for _, r := range resources {
			regionMap[r.Region] = append(regionMap[r.Region], r)
		}
		
		for region, regionResources := range regionMap {
			buf.WriteString(fmt.Sprintf("│  ├─ %s (%d)\n", region, len(regionResources)))
			
			// Group by type within region
			typeMap := make(map[string]int)
			for _, r := range regionResources {
				typeMap[r.Type]++
			}
			
			for rType, count := range typeMap {
				buf.WriteString(fmt.Sprintf("│  │  └─ %s: %d\n", rType, count))
			}
		}
		buf.WriteString("└────────────────────────┘\n\n")
	}
	
	return buf.String()
}

// GenerateHTMLReport generates an HTML report of the discovery
func (dv *DiscoveryVisualizer) GenerateHTMLReport() string {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	var buf bytes.Buffer
	
	buf.WriteString(`<!DOCTYPE html>
<html>
<head>
    <title>Discovery Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        .stats { background: #f5f5f5; padding: 15px; border-radius: 5px; }
        table { border-collapse: collapse; width: 100%; margin-top: 20px; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
        .chart { margin: 20px 0; }
        .bar { background: #4CAF50; color: white; padding: 5px; margin: 2px 0; }
    </style>
</head>
<body>
    <h1>Resource Discovery Report</h1>
    
    <div class="stats">
        <h2>Summary Statistics</h2>
        <p><strong>Total Resources:</strong> ` + fmt.Sprintf("%d", dv.stats.TotalResources) + `</p>
        <p><strong>Providers:</strong> ` + fmt.Sprintf("%d", len(dv.stats.ResourcesByProvider)) + `</p>
        <p><strong>Regions:</strong> ` + fmt.Sprintf("%d", len(dv.stats.ResourcesByRegion)) + `</p>
        <p><strong>Resource Types:</strong> ` + fmt.Sprintf("%d", len(dv.stats.ResourcesByType)) + `</p>
    </div>
    
    <div class="chart">
        <h2>Resources by Provider</h2>`)
	
	for provider, count := range dv.stats.ResourcesByProvider {
		percentage := float64(count) * 100 / float64(dv.stats.TotalResources)
		buf.WriteString(fmt.Sprintf(`
        <div class="bar" style="width: %.1f%%;">%s: %d (%.1f%%)</div>`,
			percentage, provider, count, percentage))
	}
	
	buf.WriteString(`
    </div>
    
    <h2>Resource Details</h2>
    <table>
        <tr>
            <th>Provider</th>
            <th>Region</th>
            <th>Type</th>
            <th>Name</th>
            <th>ID</th>
            <th>Status</th>
        </tr>`)
	
	for _, resource := range dv.resources {
		buf.WriteString(fmt.Sprintf(`
        <tr>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
            <td>%s</td>
        </tr>`,
			resource.Provider, resource.Region, resource.Type,
			resource.Name, resource.ID, resource.Status))
	}
	
	buf.WriteString(`
    </table>
</body>
</html>`)
	
	return buf.String()
}

// GenerateJSON generates a JSON representation of the discovery
func (dv *DiscoveryVisualizer) GenerateJSON() map[string]interface{} {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	return map[string]interface{}{
		"summary": map[string]interface{}{
			"total_resources": dv.stats.TotalResources,
			"providers":       len(dv.stats.ResourcesByProvider),
			"regions":         len(dv.stats.ResourcesByRegion),
			"resource_types":  len(dv.stats.ResourcesByType),
			"discovery_duration": dv.stats.DiscoveryDuration.String(),
		},
		"resources_by_provider": dv.stats.ResourcesByProvider,
		"resources_by_region":   dv.stats.ResourcesByRegion,
		"resources_by_type":     dv.stats.ResourcesByType,
		"resources":             dv.resources,
		"relationships":         dv.relationships,
		"timeline":              dv.timeline,
	}
}

// GenerateGraphviz generates a Graphviz DOT representation
func (dv *DiscoveryVisualizer) GenerateGraphviz() string {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	var buf bytes.Buffer
	buf.WriteString("digraph ResourceGraph {\n")
	buf.WriteString("  rankdir=LR;\n")
	buf.WriteString("  node [shape=box];\n\n")
	
	// Add nodes
	for _, resource := range dv.resources {
		label := fmt.Sprintf("%s\\n%s", resource.Type, resource.Name)
		color := dv.getColorForProvider(resource.Provider)
		buf.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\", color=\"%s\"];\n",
			resource.ID, label, color))
	}
	
	// Add edges for relationships
	buf.WriteString("\n")
	for from, tos := range dv.relationships {
		for _, to := range tos {
			buf.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", from, to))
		}
	}
	
	buf.WriteString("}\n")
	return buf.String()
}

// GenerateMarkdownReport generates a Markdown report
func (dv *DiscoveryVisualizer) GenerateMarkdownReport() string {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	var buf bytes.Buffer
	
	buf.WriteString("# Resource Discovery Report\n\n")
	buf.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))
	
	buf.WriteString("## Summary\n\n")
	buf.WriteString(fmt.Sprintf("- **Total Resources:** %d\n", dv.stats.TotalResources))
	buf.WriteString(fmt.Sprintf("- **Providers:** %d\n", len(dv.stats.ResourcesByProvider)))
	buf.WriteString(fmt.Sprintf("- **Regions:** %d\n", len(dv.stats.ResourcesByRegion)))
	buf.WriteString(fmt.Sprintf("- **Resource Types:** %d\n\n", len(dv.stats.ResourcesByType)))
	
	buf.WriteString("## Resources by Provider\n\n")
	buf.WriteString("| Provider | Count | Percentage |\n")
	buf.WriteString("|----------|-------|------------|\n")
	
	for provider, count := range dv.stats.ResourcesByProvider {
		percentage := float64(count) * 100 / float64(dv.stats.TotalResources)
		buf.WriteString(fmt.Sprintf("| %s | %d | %.1f%% |\n", provider, count, percentage))
	}
	
	buf.WriteString("\n## Resources by Region\n\n")
	buf.WriteString("| Region | Count |\n")
	buf.WriteString("|--------|-------|\n")
	
	for region, count := range dv.stats.ResourcesByRegion {
		buf.WriteString(fmt.Sprintf("| %s | %d |\n", region, count))
	}
	
	buf.WriteString("\n## Resources by Type\n\n")
	buf.WriteString("| Type | Count |\n")
	buf.WriteString("|------|-------|\n")
	
	// Sort types for consistent output
	types := make([]string, 0, len(dv.stats.ResourcesByType))
	for t := range dv.stats.ResourcesByType {
		types = append(types, t)
	}
	sort.Strings(types)
	
	for _, t := range types {
		buf.WriteString(fmt.Sprintf("| %s | %d |\n", t, dv.stats.ResourcesByType[t]))
	}
	
	return buf.String()
}

// GenerateCSV generates a CSV export of resources
func (dv *DiscoveryVisualizer) GenerateCSV() string {
	dv.mu.RLock()
	defer dv.mu.RUnlock()

	var buf bytes.Buffer
	
	// Header
	buf.WriteString("Provider,Region,Type,Name,ID,Status,Tags\n")
	
	// Resources
	for _, resource := range dv.resources {
		tags := dv.formatTags(resource.Tags)
		buf.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,\"%s\"\n",
			resource.Provider, resource.Region, resource.Type,
			resource.Name, resource.ID, resource.Status, tags))
	}
	
	return buf.String()
}

// WriteReport writes a report to an io.Writer
func (dv *DiscoveryVisualizer) WriteReport(w io.Writer, format string) error {
	var content string
	
	switch strings.ToLower(format) {
	case "ascii":
		content = dv.GenerateASCIIDiagram()
	case "html":
		content = dv.GenerateHTMLReport()
	case "markdown", "md":
		content = dv.GenerateMarkdownReport()
	case "graphviz", "dot":
		content = dv.GenerateGraphviz()
	case "csv":
		content = dv.GenerateCSV()
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	_, err := w.Write([]byte(content))
	return err
}

// GetStats returns discovery statistics
func (dv *DiscoveryVisualizer) GetStats() *DiscoveryStats {
	dv.mu.RLock()
	defer dv.mu.RUnlock()
	return dv.stats
}

// GetTimeline returns the discovery timeline
func (dv *DiscoveryVisualizer) GetTimeline() []TimelineEvent {
	dv.mu.RLock()
	defer dv.mu.RUnlock()
	
	timeline := make([]TimelineEvent, len(dv.timeline))
	copy(timeline, dv.timeline)
	return timeline
}

// Reset clears all visualization data
func (dv *DiscoveryVisualizer) Reset() {
	dv.mu.Lock()
	defer dv.mu.Unlock()
	
	dv.resources = make([]models.Resource, 0)
	dv.relationships = make(map[string][]string)
	dv.groupedByType = make(map[string][]models.Resource)
	dv.groupedByRegion = make(map[string][]models.Resource)
	dv.groupedByProvider = make(map[string][]models.Resource)
	dv.timeline = make([]TimelineEvent, 0)
	dv.stats = &DiscoveryStats{
		ResourcesByType:   make(map[string]int),
		ResourcesByRegion: make(map[string]int),
		ResourcesByProvider: make(map[string]int),
	}
}

// Helper functions

func (dv *DiscoveryVisualizer) getColorForProvider(provider string) string {
	colors := map[string]string{
		"aws":          "orange",
		"azure":        "blue",
		"gcp":          "green",
		"digitalocean": "darkblue",
	}
	
	if color, exists := colors[strings.ToLower(provider)]; exists {
		return color
	}
	return "black"
}

func (dv *DiscoveryVisualizer) formatTags(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	
	pairs := make([]string, 0, len(tags))
	for k, v := range tags {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}