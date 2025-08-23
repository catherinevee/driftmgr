package visualization

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/catherinevee/driftmgr/internal/core/models"
)

// EnhancedVisualization provides advanced visualization capabilities
type EnhancedVisualization struct {
	templates    map[string]*template.Template
	outputDir    string
	mu           sync.RWMutex
	realTimeData map[string]interface{}
	subscribers  map[string][]chan interface{}
}

// VisualizationType represents the type of visualization
type VisualizationType string

const (
	// VisualizationTypeNetwork represents network topology visualization
	VisualizationTypeNetwork VisualizationType = "network"
	// VisualizationTypeResource represents resource dependency visualization
	VisualizationTypeResource VisualizationType = "resource"
	// VisualizationTypeDrift represents drift analysis visualization
	VisualizationTypeDrift VisualizationType = "drift"
	// VisualizationTypeTimeline represents timeline visualization
	VisualizationTypeTimeline VisualizationType = "timeline"
	// VisualizationTypeMetrics represents metrics visualization
	VisualizationTypeMetrics VisualizationType = "metrics"
	// VisualizationTypeHeatmap represents heatmap visualization
	VisualizationTypeHeatmap VisualizationType = "heatmap"
)

// VisualizationOptions represents options for visualization generation
type VisualizationOptions struct {
	Type        VisualizationType `json:"type"`
	Format      string            `json:"format"` // html, svg, png, json
	Theme       string            `json:"theme"`  // light, dark, custom
	Interactive bool              `json:"interactive"`
	RealTime    bool              `json:"real_time"`
	Width       int               `json:"width"`
	Height      int               `json:"height"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	CustomCSS   string            `json:"custom_css"`
	CustomJS    string            `json:"custom_js"`
}

// VisualizationData represents data for visualization
type VisualizationData struct {
	Resources    []models.Resource      `json:"resources"`
	DriftResults []models.DriftResult   `json:"drift_results"`
	Metrics      map[string]interface{} `json:"metrics"`
	Timeline     []TimelineEvent        `json:"timeline"`
	Network      NetworkData            `json:"network"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// TimelineEvent represents a timeline event
type TimelineEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Data        map[string]interface{} `json:"data"`
}

// NetworkData represents network topology data
type NetworkData struct {
	Nodes []NetworkNode `json:"nodes"`
	Edges []NetworkEdge `json:"edges"`
}

// NetworkNode represents a network node
type NetworkNode struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Type     string                 `json:"type"`
	Group    string                 `json:"group"`
	Position Position               `json:"position"`
	Data     map[string]interface{} `json:"data"`
}

// NetworkEdge represents a network edge
type NetworkEdge struct {
	From  string                 `json:"from"`
	To    string                 `json:"to"`
	Label string                 `json:"label"`
	Type  string                 `json:"type"`
	Data  map[string]interface{} `json:"data"`
}

// Position represents a 2D position
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// NewEnhancedVisualization creates a new enhanced visualization system
func NewEnhancedVisualization(outputDir string) (*EnhancedVisualization, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	ev := &EnhancedVisualization{
		outputDir:    outputDir,
		templates:    make(map[string]*template.Template),
		realTimeData: make(map[string]interface{}),
		subscribers:  make(map[string][]chan interface{}),
	}

	// Load templates
	if err := ev.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return ev, nil
}

// GenerateVisualization generates a visualization based on the provided data and options
func (ev *EnhancedVisualization) GenerateVisualization(ctx context.Context, data *VisualizationData, options *VisualizationOptions) (string, error) {
	switch options.Type {
	case VisualizationTypeNetwork:
		return ev.generateNetworkVisualization(ctx, data, options)
	case VisualizationTypeResource:
		return ev.generateResourceVisualization(ctx, data, options)
	case VisualizationTypeDrift:
		return ev.generateDriftVisualization(ctx, data, options)
	case VisualizationTypeTimeline:
		return ev.generateTimelineVisualization(ctx, data, options)
	case VisualizationTypeMetrics:
		return ev.generateMetricsVisualization(ctx, data, options)
	case VisualizationTypeHeatmap:
		return ev.generateHeatmapVisualization(ctx, data, options)
	default:
		return "", fmt.Errorf("unsupported visualization type: %s", options.Type)
	}
}

// generateNetworkVisualization generates a network topology visualization
func (ev *EnhancedVisualization) generateNetworkVisualization(ctx context.Context, data *VisualizationData, options *VisualizationOptions) (string, error) {
	// Build network data from resources
	networkData := ev.buildNetworkData(data.Resources)

	// Generate HTML with interactive network visualization
	html, err := ev.generateNetworkHTML(networkData, options)
	if err != nil {
		return "", fmt.Errorf("failed to generate network HTML: %w", err)
	}

	// Save to file
	filename := fmt.Sprintf("network_visualization_%s.html", time.Now().Format("20060102_150405"))
	filepath := filepath.Join(ev.outputDir, filename)

	if err := os.WriteFile(filepath, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("failed to write network visualization file: %w", err)
	}

	return filepath, nil
}

// generateResourceVisualization generates a resource dependency visualization
func (ev *EnhancedVisualization) generateResourceVisualization(ctx context.Context, data *VisualizationData, options *VisualizationOptions) (string, error) {
	// Build resource dependency graph
	dependencyData := ev.buildDependencyData(data.Resources)

	// Generate HTML with interactive resource visualization
	html, err := ev.generateResourceHTML(dependencyData, options)
	if err != nil {
		return "", fmt.Errorf("failed to generate resource HTML: %w", err)
	}

	// Save to file
	filename := fmt.Sprintf("resource_visualization_%s.html", time.Now().Format("20060102_150405"))
	filepath := filepath.Join(ev.outputDir, filename)

	if err := os.WriteFile(filepath, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("failed to write resource visualization file: %w", err)
	}

	return filepath, nil
}

// generateDriftVisualization generates a drift analysis visualization
func (ev *EnhancedVisualization) generateDriftVisualization(ctx context.Context, data *VisualizationData, options *VisualizationOptions) (string, error) {
	// Build drift analysis data
	driftData := ev.buildDriftData(data.DriftResults)

	// Generate HTML with interactive drift visualization
	html, err := ev.generateDriftHTML(driftData, options)
	if err != nil {
		return "", fmt.Errorf("failed to generate drift HTML: %w", err)
	}

	// Save to file
	filename := fmt.Sprintf("drift_visualization_%s.html", time.Now().Format("20060102_150405"))
	filepath := filepath.Join(ev.outputDir, filename)

	if err := os.WriteFile(filepath, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("failed to write drift visualization file: %w", err)
	}

	return filepath, nil
}

// generateTimelineVisualization generates a timeline visualization
func (ev *EnhancedVisualization) generateTimelineVisualization(ctx context.Context, data *VisualizationData, options *VisualizationOptions) (string, error) {
	// Generate HTML with interactive timeline visualization
	html, err := ev.generateTimelineHTML(data.Timeline, options)
	if err != nil {
		return "", fmt.Errorf("failed to generate timeline HTML: %w", err)
	}

	// Save to file
	filename := fmt.Sprintf("timeline_visualization_%s.html", time.Now().Format("20060102_150405"))
	filepath := filepath.Join(ev.outputDir, filename)

	if err := os.WriteFile(filepath, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("failed to write timeline visualization file: %w", err)
	}

	return filepath, nil
}

// generateMetricsVisualization generates a metrics visualization
func (ev *EnhancedVisualization) generateMetricsVisualization(ctx context.Context, data *VisualizationData, options *VisualizationOptions) (string, error) {
	// Generate HTML with interactive metrics visualization
	html, err := ev.generateMetricsHTML(data.Metrics, options)
	if err != nil {
		return "", fmt.Errorf("failed to generate metrics HTML: %w", err)
	}

	// Save to file
	filename := fmt.Sprintf("metrics_visualization_%s.html", time.Now().Format("20060102_150405"))
	filepath := filepath.Join(ev.outputDir, filename)

	if err := os.WriteFile(filepath, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("failed to write metrics visualization file: %w", err)
	}

	return filepath, nil
}

// generateHeatmapVisualization generates a heatmap visualization
func (ev *EnhancedVisualization) generateHeatmapVisualization(ctx context.Context, data *VisualizationData, options *VisualizationOptions) (string, error) {
	// Generate HTML with interactive heatmap visualization
	html, err := ev.generateHeatmapHTML(data.Metrics, options)
	if err != nil {
		return "", fmt.Errorf("failed to generate heatmap HTML: %w", err)
	}

	// Save to file
	filename := fmt.Sprintf("heatmap_visualization_%s.html", time.Now().Format("20060102_150405"))
	filepath := filepath.Join(ev.outputDir, filename)

	if err := os.WriteFile(filepath, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("failed to write heatmap visualization file: %w", err)
	}

	return filepath, nil
}

// buildNetworkData builds network topology data from resources
func (ev *EnhancedVisualization) buildNetworkData(resources []models.Resource) *NetworkData {
	nodes := make([]NetworkNode, 0)
	edges := make([]NetworkEdge, 0)
	nodeMap := make(map[string]NetworkNode)

	// Create nodes from resources
	for i, resource := range resources {
		node := NetworkNode{
			ID:    resource.ID,
			Label: resource.Name,
			Type:  resource.Type,
			Group: resource.Provider,
			Position: Position{
				X: float64(i * 100),
				Y: float64(i * 100),
			},
			Data: map[string]interface{}{
				"region":  resource.Region,
				"state":   resource.State,
				"tags":    resource.Tags,
				"created": resource.Created,
				"updated": resource.Updated,
			},
		}
		nodes = append(nodes, node)
		nodeMap[resource.ID] = node
	}

	// Create edges based on resource relationships
	// This is a simplified example - in a real implementation, you would
	// analyze resource dependencies and create appropriate edges
	for i, resource := range resources {
		if i > 0 {
			edge := NetworkEdge{
				From:  resources[i-1].ID,
				To:    resource.ID,
				Label: "depends_on",
				Type:  "dependency",
				Data: map[string]interface{}{
					"relationship": "depends_on",
				},
			}
			edges = append(edges, edge)
		}
	}

	return &NetworkData{
		Nodes: nodes,
		Edges: edges,
	}
}

// buildDependencyData builds resource dependency data
func (ev *EnhancedVisualization) buildDependencyData(resources []models.Resource) map[string]interface{} {
	dependencies := make(map[string][]string)
	resourceMap := make(map[string]models.Resource)

	// Build resource map
	for _, resource := range resources {
		resourceMap[resource.ID] = resource
	}

	// Analyze dependencies (simplified example)
	for _, resource := range resources {
		deps := make([]string, 0)

		// Add dependencies based on resource type and tags
		for _, other := range resources {
			if resource.ID != other.ID {
				// Example dependency logic
				if resource.Provider == other.Provider && resource.Region == other.Region {
					deps = append(deps, other.ID)
				}
			}
		}

		dependencies[resource.ID] = deps
	}

	return map[string]interface{}{
		"resources":    resources,
		"dependencies": dependencies,
		"resource_map": resourceMap,
	}
}

// buildDriftData builds drift analysis data
func (ev *EnhancedVisualization) buildDriftData(driftResults []models.DriftResult) map[string]interface{} {
	driftBySeverity := make(map[string][]models.DriftResult)
	driftByType := make(map[string][]models.DriftResult)
	driftByProvider := make(map[string][]models.DriftResult)

	for _, drift := range driftResults {
		// Group by severity
		driftBySeverity[drift.Severity] = append(driftBySeverity[drift.Severity], drift)

		// Group by type
		driftByType[drift.DriftType] = append(driftByType[drift.DriftType], drift)

		// Group by provider
		driftByProvider[drift.Provider] = append(driftByProvider[drift.Provider], drift)
	}

	return map[string]interface{}{
		"drift_results":     driftResults,
		"drift_by_severity": driftBySeverity,
		"drift_by_type":     driftByType,
		"drift_by_provider": driftByProvider,
		"total_drifts":      len(driftResults),
	}
}

// generateNetworkHTML generates HTML for network visualization
func (ev *EnhancedVisualization) generateNetworkHTML(networkData *NetworkData, options *VisualizationOptions) (string, error) {
	const networkTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <script src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: {{if eq .Theme "dark"}}#1a1a1a{{else}}#ffffff{{end}}; color: {{if eq .Theme "dark"}}#ffffff{{else}}#000000{{end}}; }
        .container { width: 100%; height: 80vh; border: 1px solid #ccc; }
        .controls { margin-bottom: 20px; }
        .info { margin-top: 20px; padding: 10px; background-color: #f5f5f5; border-radius: 5px; }
        {{.CustomCSS}}
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>{{.Description}}</p>
    
    <div class="controls">
        <button onclick="fitNetwork()">Fit Network</button>
        <button onclick="stabilize()">Stabilize</button>
        <button onclick="exportData()">Export Data</button>
    </div>
    
    <div id="network" class="container"></div>
    
    <div class="info">
        <h3>Network Information</h3>
        <p>Nodes: <span id="nodeCount">{{len .NetworkData.Nodes}}</span></p>
        <p>Edges: <span id="edgeCount">{{len .NetworkData.Edges}}</span></p>
        <div id="nodeInfo"></div>
    </div>

    <script>
        // Network data
        const nodes = new vis.DataSet({{.NodesJSON}});
        const edges = new vis.DataSet({{.EdgesJSON}});
        
        // Network options
        const options = {
            nodes: {
                shape: 'dot',
                size: 16,
                font: {
                    size: 12,
                    color: '{{if eq .Theme "dark"}}#ffffff{{else}}#000000{{end}}'
                },
                borderWidth: 2,
                shadow: true
            },
            edges: {
                width: 2,
                shadow: true,
                smooth: {
                    type: 'continuous'
                }
            },
            physics: {
                stabilization: false,
                barnesHut: {
                    gravitationalConstant: -80000,
                    springConstant: 0.001,
                    springLength: 200
                }
            },
            interaction: {
                navigationButtons: true,
                keyboard: true
            }
        };
        
        // Create network
        const container = document.getElementById('network');
        const network = new vis.Network(container, {nodes, edges}, options);
        
        // Network events
        network.on('click', function(params) {
            if (params.nodes.length > 0) {
                const nodeId = params.nodes[0];
                const node = nodes.get(nodeId);
                document.getElementById('nodeInfo').innerHTML = 
                    '<strong>Selected Node:</strong><br>' +
                    'ID: ' + node.id + '<br>' +
                    'Label: ' + node.label + '<br>' +
                    'Type: ' + node.type + '<br>' +
                    'Group: ' + node.group;
            }
        });
        
        // Control functions
        function fitNetwork() {
            network.fit();
        }
        
        function stabilize() {
            network.stabilize();
        }
        
        function exportData() {
            const data = {
                nodes: nodes.get(),
                edges: edges.get()
            };
            const blob = new Blob([JSON.stringify(data, null, 2)], {type: 'application/json'});
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'network_data.json';
            a.click();
        }
        
        {{.CustomJS}}
    </script>
</body>
</html>`

	data := struct {
		Title       string
		Description string
		Theme       string
		NetworkData *NetworkData
		NodesJSON   string
		EdgesJSON   string
		CustomCSS   template.CSS
		CustomJS    template.JS
	}{
		Title:       options.Title,
		Description: options.Description,
		Theme:       options.Theme,
		NetworkData: networkData,
		CustomCSS:   template.CSS(options.CustomCSS),
		CustomJS:    template.JS(options.CustomJS),
	}

	// Convert network data to JSON
	nodesJSON, err := json.Marshal(networkData.Nodes)
	if err != nil {
		return "", err
	}
	data.NodesJSON = string(nodesJSON)

	edgesJSON, err := json.Marshal(networkData.Edges)
	if err != nil {
		return "", err
	}
	data.EdgesJSON = string(edgesJSON)

	// Parse and execute template
	tmpl, err := template.New("network").Parse(networkTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateResourceHTML generates HTML for resource visualization
func (ev *EnhancedVisualization) generateResourceHTML(dependencyData map[string]interface{}, options *VisualizationOptions) (string, error) {
	const resourceTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <script src="https://d3js.org/d3.v7.min.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: {{if eq .Theme "dark"}}#1a1a1a{{else}}#ffffff{{end}}; color: {{if eq .Theme "dark"}}#ffffff{{else}}#000000{{end}}; }
        .container { width: 100%; height: 80vh; border: 1px solid #ccc; }
        .controls { margin-bottom: 20px; }
        .info { margin-top: 20px; padding: 10px; background-color: #f5f5f5; border-radius: 5px; }
        {{.CustomCSS}}
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>{{.Description}}</p>
    
    <div class="controls">
        <button onclick="resetZoom()">Reset Zoom</button>
        <button onclick="exportData()">Export Data</button>
    </div>
    
    <div id="resource-graph" class="container"></div>
    
    <div class="info">
        <h3>Resource Information</h3>
        <p>Total Resources: <span id="resourceCount">{{len .DependencyData.resources}}</span></p>
        <div id="resourceInfo"></div>
    </div>

    <script>
        // Resource data
        const resourceData = {{.DependencyDataJSON}};
        
        // D3.js visualization code would go here
        // This is a simplified example - in a real implementation,
        // you would create a full D3.js force-directed graph
        
        function resetZoom() {
            // Reset zoom functionality
        }
        
        function exportData() {
            const blob = new Blob([JSON.stringify(resourceData, null, 2)], {type: 'application/json'});
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'resource_data.json';
            a.click();
        }
        
        {{.CustomJS}}
    </script>
</body>
</html>`

	// Convert dependency data to JSON
	dependencyDataJSON, err := json.Marshal(dependencyData)
	if err != nil {
		return "", err
	}

	data := struct {
		Title              string
		Description        string
		Theme              string
		DependencyData     map[string]interface{}
		DependencyDataJSON string
		CustomCSS          template.CSS
		CustomJS           template.JS
	}{
		Title:              options.Title,
		Description:        options.Description,
		Theme:              options.Theme,
		DependencyData:     dependencyData,
		DependencyDataJSON: string(dependencyDataJSON),
		CustomCSS:          template.CSS(options.CustomCSS),
		CustomJS:           template.JS(options.CustomJS),
	}

	// Parse and execute template
	tmpl, err := template.New("resource").Parse(resourceTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateDriftHTML generates HTML for drift visualization
func (ev *EnhancedVisualization) generateDriftHTML(driftData map[string]interface{}, options *VisualizationOptions) (string, error) {
	const driftTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: {{if eq .Theme "dark"}}#1a1a1a{{else}}#ffffff{{end}}; color: {{if eq .Theme "dark"}}#ffffff{{else}}#000000{{end}}; }
        .container { width: 100%; height: 80vh; border: 1px solid #ccc; }
        .controls { margin-bottom: 20px; }
        .info { margin-top: 20px; padding: 10px; background-color: #f5f5f5; border-radius: 5px; }
        {{.CustomCSS}}
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>{{.Description}}</p>
    
    <div class="controls">
        <button onclick="exportData()">Export Data</button>
    </div>
    
    <div id="drift-charts" class="container"></div>
    
    <div class="info">
        <h3>Drift Analysis Information</h3>
        <p>Total Drifts: <span id="driftCount">{{.DriftData.total_drifts}}</span></p>
        <div id="driftInfo"></div>
    </div>

    <script>
        // Drift data
        const driftData = {{.DriftDataJSON}};
        
        // Chart.js visualization code would go here
        // This is a simplified example - in a real implementation,
        // you would create multiple charts showing drift analysis
        
        function exportData() {
            const blob = new Blob([JSON.stringify(driftData, null, 2)], {type: 'application/json'});
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'drift_data.json';
            a.click();
        }
        
        {{.CustomJS}}
    </script>
</body>
</html>`

	// Convert drift data to JSON
	driftDataJSON, err := json.Marshal(driftData)
	if err != nil {
		return "", err
	}

	data := struct {
		Title         string
		Description   string
		Theme         string
		DriftData     map[string]interface{}
		DriftDataJSON string
		CustomCSS     template.CSS
		CustomJS      template.JS
	}{
		Title:         options.Title,
		Description:   options.Description,
		Theme:         options.Theme,
		DriftData:     driftData,
		DriftDataJSON: string(driftDataJSON),
		CustomCSS:     template.CSS(options.CustomCSS),
		CustomJS:      template.JS(options.CustomJS),
	}

	// Parse and execute template
	tmpl, err := template.New("drift").Parse(driftTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateTimelineHTML generates HTML for timeline visualization
func (ev *EnhancedVisualization) generateTimelineHTML(timeline []TimelineEvent, options *VisualizationOptions) (string, error) {
	const timelineTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: {{if eq .Theme "dark"}}#1a1a1a{{else}}#ffffff{{end}}; color: {{if eq .Theme "dark"}}#ffffff{{else}}#000000{{end}}; }
        .container { width: 100%; height: 80vh; border: 1px solid #ccc; }
        .controls { margin-bottom: 20px; }
        .info { margin-top: 20px; padding: 10px; background-color: #f5f5f5; border-radius: 5px; }
        {{.CustomCSS}}
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>{{.Description}}</p>
    
    <div class="controls">
        <button onclick="exportData()">Export Data</button>
    </div>
    
    <div id="timeline-chart" class="container"></div>
    
    <div class="info">
        <h3>Timeline Information</h3>
        <p>Total Events: <span id="eventCount">{{len .Timeline}}</span></p>
        <div id="timelineInfo"></div>
    </div>

    <script>
        // Timeline data
        const timelineData = {{.TimelineJSON}};
        
        // Chart.js timeline visualization code would go here
        
        function exportData() {
            const blob = new Blob([JSON.stringify(timelineData, null, 2)], {type: 'application/json'});
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'timeline_data.json';
            a.click();
        }
        
        {{.CustomJS}}
    </script>
</body>
</html>`

	// Convert timeline data to JSON
	timelineJSON, err := json.Marshal(timeline)
	if err != nil {
		return "", err
	}

	data := struct {
		Title        string
		Description  string
		Theme        string
		Timeline     []TimelineEvent
		TimelineJSON string
		CustomCSS    template.CSS
		CustomJS     template.JS
	}{
		Title:        options.Title,
		Description:  options.Description,
		Theme:        options.Theme,
		Timeline:     timeline,
		TimelineJSON: string(timelineJSON),
		CustomCSS:    template.CSS(options.CustomCSS),
		CustomJS:     template.JS(options.CustomJS),
	}

	// Parse and execute template
	tmpl, err := template.New("timeline").Parse(timelineTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateMetricsHTML generates HTML for metrics visualization
func (ev *EnhancedVisualization) generateMetricsHTML(metrics map[string]interface{}, options *VisualizationOptions) (string, error) {
	const metricsTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: {{if eq .Theme "dark"}}#1a1a1a{{else}}#ffffff{{end}}; color: {{if eq .Theme "dark"}}#ffffff{{else}}#000000{{end}}; }
        .container { width: 100%; height: 80vh; border: 1px solid #ccc; }
        .controls { margin-bottom: 20px; }
        .info { margin-top: 20px; padding: 10px; background-color: #f5f5f5; border-radius: 5px; }
        {{.CustomCSS}}
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>{{.Description}}</p>
    
    <div class="controls">
        <button onclick="exportData()">Export Data</button>
    </div>
    
    <div id="metrics-charts" class="container"></div>
    
    <div class="info">
        <h3>Metrics Information</h3>
        <div id="metricsInfo"></div>
    </div>

    <script>
        // Metrics data
        const metricsData = {{.MetricsJSON}};
        
        // Chart.js metrics visualization code would go here
        
        function exportData() {
            const blob = new Blob([JSON.stringify(metricsData, null, 2)], {type: 'application/json'});
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'metrics_data.json';
            a.click();
        }
        
        {{.CustomJS}}
    </script>
</body>
</html>`

	// Convert metrics data to JSON
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return "", err
	}

	data := struct {
		Title       string
		Description string
		Theme       string
		Metrics     map[string]interface{}
		MetricsJSON string
		CustomCSS   template.CSS
		CustomJS    template.JS
	}{
		Title:       options.Title,
		Description: options.Description,
		Theme:       options.Theme,
		Metrics:     metrics,
		MetricsJSON: string(metricsJSON),
		CustomCSS:   template.CSS(options.CustomCSS),
		CustomJS:    template.JS(options.CustomJS),
	}

	// Parse and execute template
	tmpl, err := template.New("metrics").Parse(metricsTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateHeatmapHTML generates HTML for heatmap visualization
func (ev *EnhancedVisualization) generateHeatmapHTML(metrics map[string]interface{}, options *VisualizationOptions) (string, error) {
	const heatmapTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: {{if eq .Theme "dark"}}#1a1a1a{{else}}#ffffff{{end}}; color: {{if eq .Theme "dark"}}#ffffff{{else}}#000000{{end}}; }
        .container { width: 100%; height: 80vh; border: 1px solid #ccc; }
        .controls { margin-bottom: 20px; }
        .info { margin-top: 20px; padding: 10px; background-color: #f5f5f5; border-radius: 5px; }
        {{.CustomCSS}}
    </style>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>{{.Description}}</p>
    
    <div class="controls">
        <button onclick="exportData()">Export Data</button>
    </div>
    
    <div id="heatmap-chart" class="container"></div>
    
    <div class="info">
        <h3>Heatmap Information</h3>
        <div id="heatmapInfo"></div>
    </div>

    <script>
        // Heatmap data
        const heatmapData = {{.MetricsJSON}};
        
        // Chart.js heatmap visualization code would go here
        
        function exportData() {
            const blob = new Blob([JSON.stringify(heatmapData, null, 2)], {type: 'application/json'});
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'heatmap_data.json';
            a.click();
        }
        
        {{.CustomJS}}
    </script>
</body>
</html>`

	// Convert metrics data to JSON
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return "", err
	}

	data := struct {
		Title       string
		Description string
		Theme       string
		Metrics     map[string]interface{}
		MetricsJSON string
		CustomCSS   template.CSS
		CustomJS    template.JS
	}{
		Title:       options.Title,
		Description: options.Description,
		Theme:       options.Theme,
		Metrics:     metrics,
		MetricsJSON: string(metricsJSON),
		CustomCSS:   template.CSS(options.CustomCSS),
		CustomJS:    template.JS(options.CustomJS),
	}

	// Parse and execute template
	tmpl, err := template.New("heatmap").Parse(heatmapTemplate)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// loadTemplates loads HTML templates
func (ev *EnhancedVisualization) loadTemplates() error {
	// In a real implementation, you would load templates from files
	// For now, we'll use inline templates
	return nil
}

// UpdateRealTimeData updates real-time data for live visualizations
func (ev *EnhancedVisualization) UpdateRealTimeData(key string, data interface{}) {
	ev.mu.Lock()
	defer ev.mu.Unlock()

	ev.realTimeData[key] = data

	// Notify subscribers
	if subscribers, exists := ev.subscribers[key]; exists {
		for _, ch := range subscribers {
			select {
			case ch <- data:
			default:
				// Channel is full, skip
			}
		}
	}
}

// SubscribeToRealTimeData subscribes to real-time data updates
func (ev *EnhancedVisualization) SubscribeToRealTimeData(key string) chan interface{} {
	ev.mu.Lock()
	defer ev.mu.Unlock()

	ch := make(chan interface{}, 100)
	ev.subscribers[key] = append(ev.subscribers[key], ch)

	return ch
}

// UnsubscribeFromRealTimeData unsubscribes from real-time data updates
func (ev *EnhancedVisualization) UnsubscribeFromRealTimeData(key string, ch chan interface{}) {
	ev.mu.Lock()
	defer ev.mu.Unlock()

	if subscribers, exists := ev.subscribers[key]; exists {
		for i, subscriber := range subscribers {
			if subscriber == ch {
				ev.subscribers[key] = append(subscribers[:i], subscribers[i+1:]...)
				close(ch)
				break
			}
		}
	}
}

// GetRealTimeData gets current real-time data
func (ev *EnhancedVisualization) GetRealTimeData(key string) (interface{}, bool) {
	ev.mu.RLock()
	defer ev.mu.RUnlock()

	data, exists := ev.realTimeData[key]
	return data, exists
}

// ExportVisualization exports a visualization to a specific format
func (ev *EnhancedVisualization) ExportVisualization(filepath string, format string) error {
	// Read the HTML file
	htmlData, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read visualization file: %w", err)
	}

	// Convert to different formats based on the requested format
	switch format {
	case "html":
		// Already in HTML format, no conversion needed
		return nil
	case "svg":
		// Convert HTML to SVG (simplified - in real implementation, you'd use a headless browser)
		return ev.convertToSVG(htmlData, filepath)
	case "png":
		// Convert HTML to PNG (simplified - in real implementation, you'd use a headless browser)
		return ev.convertToPNG(htmlData, filepath)
	case "json":
		// Extract data as JSON
		return ev.extractAsJSON(htmlData, filepath)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// convertToSVG converts HTML visualization to SVG
func (ev *EnhancedVisualization) convertToSVG(htmlData []byte, filepath string) error {
	// This is a simplified implementation
	// In a real implementation, you would use a headless browser like Puppeteer or Playwright
	// to render the HTML and capture it as SVG

	svgFilepath := strings.TrimSuffix(filepath, ".html") + ".svg"

	// For now, just create a placeholder SVG
	svgContent := `<?xml version="1.0" encoding="UTF-8"?>
<svg width="800" height="600" xmlns="http://www.w3.org/2000/svg">
  <rect width="100%" height="100%" fill="white"/>
  <text x="400" y="300" text-anchor="middle" font-family="Arial" font-size="24" fill="black">
    Visualization exported as SVG
  </text>
  <text x="400" y="330" text-anchor="middle" font-family="Arial" font-size="16" fill="gray">
    (SVG export functionality requires headless browser integration)
  </text>
</svg>`

	return os.WriteFile(svgFilepath, []byte(svgContent), 0644)
}

// convertToPNG converts HTML visualization to PNG
func (ev *EnhancedVisualization) convertToPNG(htmlData []byte, filepath string) error {
	// This is a simplified implementation
	// In a real implementation, you would use a headless browser like Puppeteer or Playwright
	// to render the HTML and capture it as PNG

	pngFilepath := strings.TrimSuffix(filepath, ".html") + ".png"

	// For now, just create a placeholder file
	placeholderContent := "PNG export functionality requires headless browser integration"

	return os.WriteFile(pngFilepath, []byte(placeholderContent), 0644)
}

// extractAsJSON extracts data from HTML visualization as JSON
func (ev *EnhancedVisualization) extractAsJSON(htmlData []byte, filepath string) error {
	// This is a simplified implementation
	// In a real implementation, you would parse the HTML and extract the embedded JSON data

	jsonFilepath := strings.TrimSuffix(filepath, ".html") + ".json"

	// For now, just create a placeholder JSON
	jsonContent := `{
  "message": "JSON extraction functionality requires HTML parsing implementation",
  "original_file": "` + filepath + `",
  "timestamp": "` + time.Now().Format(time.RFC3339) + `"
}`

	return os.WriteFile(jsonFilepath, []byte(jsonContent), 0644)
}
