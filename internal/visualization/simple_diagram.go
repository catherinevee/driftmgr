package visualization

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/catherinevee/driftmgr/internal/models"
)

// SimpleDiagramGenerator generates basic infrastructure diagrams without Graphviz
type SimpleDiagramGenerator struct {
	outputDir string
}

// NewSimpleDiagramGenerator creates a new simple diagram generator
func NewSimpleDiagramGenerator(outputDir string) (*SimpleDiagramGenerator, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &SimpleDiagramGenerator{
		outputDir: outputDir,
	}, nil
}

// GenerateDiagram generates a simple diagram from Terraform state data
func (dg *SimpleDiagramGenerator) GenerateDiagram(stateFileID string, diagramData models.DiagramData) (*models.DiagramResponse, error) {
	startTime := time.Now()

	// Generate different output formats
	outputs := []models.VisualizationOutput{}
	formats := []string{"html", "svg"}

	for _, format := range formats {
		output, err := dg.generateOutput(stateFileID, diagramData, format)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s output: %w", format, err)
		}
		outputs = append(outputs, *output)
	}

	duration := time.Since(startTime)

	return &models.DiagramResponse{
		StateFileID: stateFileID,
		Status:      "completed",
		Message:     "Simple diagram generated successfully",
		Duration:    duration,
		GeneratedAt: time.Now(),
		DiagramData: diagramData,
	}, nil
}

// generateOutput generates a specific output format
func (dg *SimpleDiagramGenerator) generateOutput(stateFileID string, diagramData models.DiagramData, format string) (*models.VisualizationOutput, error) {
	var content string
	var err error

	switch format {
	case "html":
		content, err = dg.generateHTMLOutput(stateFileID, diagramData)
		if err != nil {
			return nil, err
		}
	case "svg":
		content, err = dg.generateSVGOutput(diagramData)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	// Write to file
	filename := fmt.Sprintf("%s-diagram.%s", stateFileID, format)
	filepath := filepath.Join(dg.outputDir, filename)

	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write output file: %w", err)
	}

	return &models.VisualizationOutput{
		Format: format,
		Path:   filepath,
		URL:    fmt.Sprintf("http://localhost:8080/outputs/%s", filename),
	}, nil
}

// generateHTMLOutput generates an interactive HTML diagram
func (dg *SimpleDiagramGenerator) generateHTMLOutput(stateFileID string, diagramData models.DiagramData) (string, error) {
	// Generate SVG content for embedding
	svgContent, err := dg.generateSVGOutput(diagramData)
	if err != nil {
		return "", err
	}

	// Create interactive HTML with CSS and JavaScript
	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Infrastructure Diagram - %s</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            padding: 20px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 24px;
        }
        .header p {
            margin: 5px 0 0 0;
            opacity: 0.9;
        }
        .diagram-container {
            padding: 20px;
            text-align: center;
            overflow-x: auto;
        }
        .diagram-container svg {
            max-width: 100%%;
            height: auto;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        .controls {
            padding: 20px;
            background: #f8f9fa;
            border-top: 1px solid #dee2e6;
        }
        .control-group {
            display: flex;
            gap: 10px;
            align-items: center;
            margin-bottom: 10px;
        }
        .control-group label {
            font-weight: bold;
            min-width: 100px;
        }
        .control-group input, .control-group select {
            padding: 8px;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 14px;
        }
        .control-group button {
            padding: 8px 16px;
            background: #007bff;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
        }
        .control-group button:hover {
            background: #0056b3;
        }
        .info {
            padding: 20px;
            background: #e7f3ff;
            border-left: 4px solid #007bff;
            margin: 20px 0;
        }
        .info h3 {
            margin: 0 0 10px 0;
            color: #007bff;
        }
        .info p {
            margin: 5px 0;
        }
        .resource-list {
            padding: 20px;
            background: #f8f9fa;
            margin: 20px 0;
        }
        .resource-list h3 {
            margin: 0 0 15px 0;
            color: #333;
        }
        .resource-item {
            padding: 10px;
            margin: 5px 0;
            background: white;
            border-radius: 4px;
            border-left: 4px solid #007bff;
        }
        .resource-name {
            font-weight: bold;
            color: #007bff;
        }
        .resource-type {
            color: #666;
            font-size: 12px;
        }
        .resource-region {
            color: #999;
            font-size: 11px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Infrastructure Diagram</h1>
            <p>Generated by DriftMgr - %s</p>
        </div>
        
        <div class="info">
            <h3>Diagram Information</h3>
            <p><strong>State File:</strong> %s</p>
            <p><strong>Generated:</strong> %s</p>
            <p><strong>Total Resources:</strong> %d</p>
            <p><strong>Total Dependencies:</strong> %d</p>
            <p><strong>Data Sources:</strong> %d</p>
            <p><strong>Modules:</strong> %d</p>
        </div>

        <div class="controls">
            <div class="control-group">
                <label for="zoom">Zoom:</label>
                <input type="range" id="zoom" min="0.5" max="2" step="0.1" value="1">
                <span id="zoomValue">100%%</span>
                <button onclick="resetZoom()">Reset</button>
            </div>
            <div class="control-group">
                <label for="search">Search:</label>
                <input type="text" id="search" placeholder="Search resources...">
                <button onclick="searchResources()">Search</button>
                <button onclick="clearSearch()">Clear</button>
            </div>
        </div>

        <div class="diagram-container" id="diagramContainer">
            %s
        </div>

        <div class="resource-list">
            <h3>Resource List</h3>
            %s
        </div>
    </div>

    <script>
        // Zoom functionality
        const zoomSlider = document.getElementById('zoom');
        const zoomValue = document.getElementById('zoomValue');
        const svg = document.querySelector('svg');
        
        if (zoomSlider && svg) {
            zoomSlider.addEventListener('input', function() {
                const zoom = this.value;
                zoomValue.textContent = Math.round(zoom * 100) + '%%';
                svg.style.transform = 'scale(' + zoom + ')';
            });
        }

        function resetZoom() {
            if (zoomSlider && svg) {
                zoomSlider.value = 1;
                zoomValue.textContent = '100%%';
                svg.style.transform = 'scale(1)';
            }
        }

        // Search functionality
        const searchInput = document.getElementById('search');
        
        function searchResources() {
            const searchTerm = searchInput.value.toLowerCase();
            const resources = document.querySelectorAll('.resource-item');
            
            resources.forEach(resource => {
                const text = resource.textContent.toLowerCase();
                if (text.includes(searchTerm)) {
                    resource.style.opacity = '1';
                    resource.style.backgroundColor = '#fff3cd';
                } else {
                    resource.style.opacity = '0.3';
                    resource.style.backgroundColor = 'white';
                }
            });
        }

        function clearSearch() {
            searchInput.value = '';
            const resources = document.querySelectorAll('.resource-item');
            resources.forEach(resource => {
                resource.style.opacity = '1';
                resource.style.backgroundColor = 'white';
            });
        }

        // Keyboard shortcuts
        document.addEventListener('keydown', function(e) {
            if (e.ctrlKey || e.metaKey) {
                switch(e.key) {
                    case '0':
                        e.preventDefault();
                        resetZoom();
                        break;
                    case 'f':
                        e.preventDefault();
                        searchInput.focus();
                        break;
                }
            }
        });
    </script>
</body>
</html>`

	// Generate resource list HTML
	resourceListHTML := dg.generateResourceListHTML(diagramData)

	return fmt.Sprintf(htmlTemplate,
		stateFileID,
		time.Now().Format("2006-01-02 15:04:05"),
		stateFileID,
		time.Now().Format("2006-01-02 15:04:05"),
		len(diagramData.Resources),
		len(diagramData.Dependencies),
		len(diagramData.DataSources),
		len(diagramData.Modules),
		svgContent,
		resourceListHTML,
	), nil
}

// generateSVGOutput generates a simple SVG diagram
func (dg *SimpleDiagramGenerator) generateSVGOutput(diagramData models.DiagramData) (string, error) {
	// Calculate dimensions based on number of resources
	width := 800
	height := 600
	if len(diagramData.Resources) > 10 {
		width = 1200
		height = 800
	}

	// Start SVG
	var svg strings.Builder
	svg.WriteString(fmt.Sprintf(`<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
    <defs>
        <style>
            .resource { fill: #e3f2fd; stroke: #1976d2; stroke-width: 2; }
            .compute { fill: #e8f5e8; stroke: #2e7d32; }
            .storage { fill: #ffebee; stroke: #c62828; }
            .network { fill: #fff3e0; stroke: #ef6c00; }
            .database { fill: #f3e5f5; stroke: #7b1fa2; }
            .serverless { fill: #e0f2f1; stroke: #00695c; }
            .security { fill: #f5f5f5; stroke: #424242; }
            .text { font-family: Arial, sans-serif; font-size: 12px; }
            .title { font-family: Arial, sans-serif; font-size: 16px; font-weight: bold; }
            .dependency { stroke: #666; stroke-width: 1; marker-end: url(#arrowhead); }
        </style>
        <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto">
            <polygon points="0 0, 10 3.5, 0 7" fill="#666" />
        </marker>
    </defs>
    <rect width="%d" height="%d" fill="white"/>
    <text x="%d" y="30" class="title" text-anchor="middle">Infrastructure Diagram</text>`,
		width, height, width, height, width/2))

	// Calculate positions for resources
	positions := dg.calculateResourcePositions(diagramData.Resources, width, height)

	// Draw resources
	for i, resource := range diagramData.Resources {
		pos := positions[i]
		resourceClass := dg.getResourceClass(resource.Type)

		// Draw resource box
		svg.WriteString(fmt.Sprintf(`
    <rect x="%d" y="%d" width="120" height="60" class="resource %s" rx="5"/>`,
			pos.X, pos.Y, resourceClass))

		// Draw resource text
		svg.WriteString(fmt.Sprintf(`
    <text x="%d" y="%d" class="text" text-anchor="middle">%s</text>`,
			pos.X+60, pos.Y+25, dg.truncateString(resource.Name, 15)))
		svg.WriteString(fmt.Sprintf(`
    <text x="%d" y="%d" class="text" text-anchor="middle">%s</text>`,
			pos.X+60, pos.Y+40, dg.truncateString(resource.Type, 15)))
	}

	// Draw dependencies
	for _, dep := range diagramData.Dependencies {
		fromPos := dg.findResourcePosition(dep.From, positions, diagramData.Resources)
		toPos := dg.findResourcePosition(dep.To, positions, diagramData.Resources)

		if fromPos != nil && toPos != nil {
			svg.WriteString(fmt.Sprintf(`
    <line x1="%d" y1="%d" x2="%d" y2="%d" class="dependency"/>`,
				fromPos.X+120, fromPos.Y+30, toPos.X, toPos.Y+30))
		}
	}

	svg.WriteString(`
</svg>`)

	return svg.String(), nil
}

// generateResourceListHTML generates HTML for the resource list
func (dg *SimpleDiagramGenerator) generateResourceListHTML(diagramData models.DiagramData) string {
	var html strings.Builder

	for _, resource := range diagramData.Resources {
		html.WriteString(fmt.Sprintf(`
            <div class="resource-item">
                <div class="resource-name">%s</div>
                <div class="resource-type">%s</div>
                <div class="resource-region">%s</div>
            </div>`,
			resource.Name, resource.Type, resource.Region))
	}

	return html.String()
}

// calculateResourcePositions calculates positions for resources in a grid layout
func (dg *SimpleDiagramGenerator) calculateResourcePositions(resources []models.Resource, width, height int) []struct {
	X, Y int
} {
	positions := make([]struct{ X, Y int }, len(resources))

	cols := 4
	if len(resources) > 12 {
		cols = 6
	}

	for i := range resources {
		row := i / cols
		col := i % cols

		positions[i] = struct{ X, Y int }{
			X: 50 + col*150,
			Y: 80 + row*100,
		}
	}

	return positions
}

// findResourcePosition finds the position of a resource by name
func (dg *SimpleDiagramGenerator) findResourcePosition(name string, positions []struct{ X, Y int }, resources []models.Resource) *struct{ X, Y int } {
	for i, resource := range resources {
		if resource.Name == name {
			return &positions[i]
		}
	}
	return nil
}

// getResourceClass returns the CSS class for a resource type
func (dg *SimpleDiagramGenerator) getResourceClass(resourceType string) string {
	switch {
	case strings.Contains(resourceType, "instance") || strings.Contains(resourceType, "virtual_machine"):
		return "compute"
	case strings.Contains(resourceType, "bucket") || strings.Contains(resourceType, "storage"):
		return "storage"
	case strings.Contains(resourceType, "vpc") || strings.Contains(resourceType, "network"):
		return "network"
	case strings.Contains(resourceType, "rds") || strings.Contains(resourceType, "sql"):
		return "database"
	case strings.Contains(resourceType, "lambda") || strings.Contains(resourceType, "function"):
		return "serverless"
	case strings.Contains(resourceType, "iam") || strings.Contains(resourceType, "key_vault"):
		return "security"
	default:
		return ""
	}
}

// truncateString truncates a string to the specified length
func (dg *SimpleDiagramGenerator) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
