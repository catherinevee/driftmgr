package visualization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"strings"

	"github.com/catherinevee/driftmgr/internal/models"
)

// HTMLExporter exports visualizations to HTML
type HTMLExporter struct {
	template string
}

// NewHTMLExporter creates a new HTML exporter
func NewHTMLExporter() *HTMLExporter {
	return &HTMLExporter{
		template: defaultHTMLTemplate,
	}
}

// Format returns the format name
func (e *HTMLExporter) Format() string {
	return "html"
}

// Export exports the visualization data to HTML
func (e *HTMLExporter) Export(data *VisualizationData, w io.Writer) error {
	// For HTML export, we use a default Mermaid renderer
	renderer := NewMermaidRenderer()
	rendered, err := renderer.Render(data)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	// Create HTML template
	tmpl, err := template.New("visualization").Parse(e.template)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"Title":   "Infrastructure Visualization",
		"Content": string(rendered),
		"Format":  renderer.Format(),
		"Data":    data,
	})
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	_, err = w.Write(buf.Bytes())
	return err
}

// SVGExporter exports visualizations to SVG
type SVGExporter struct{}

// NewSVGExporter creates a new SVG exporter
func NewSVGExporter() *SVGExporter {
	return &SVGExporter{}
}

// Format returns the format name
func (e *SVGExporter) Format() string {
	return "svg"
}

// Export exports the visualization data to SVG
func (e *SVGExporter) Export(data *VisualizationData, w io.Writer) error {
	// For SVG, we use Graphviz renderer
	renderer := NewGraphvizRenderer()

	// Render as DOT
	dot, err := renderer.Render(data)
	if err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}

	// Generate SVG visualization
	svg := e.generateSVG(data, string(dot))
	_, writeErr := w.Write([]byte(svg))
	return writeErr
}

// generateSVG generates an SVG visualization from the data
func (e *SVGExporter) generateSVG(data *VisualizationData, dotGraph string) string {
	var svg strings.Builder

	svg.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	svg.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" viewBox="0 0 1200 800">`)
	svg.WriteString(`<defs>`)
	svg.WriteString(`<style>`)
	svg.WriteString(`.resource { fill: #4a90e2; stroke: #2e5c8a; stroke-width: 2; }`)
	svg.WriteString(`.aws { fill: #ff9900; }`)
	svg.WriteString(`.azure { fill: #0078d4; }`)
	svg.WriteString(`.gcp { fill: #4285f4; }`)
	svg.WriteString(`.connection { stroke: #666; stroke-width: 1; fill: none; }`)
	svg.WriteString(`.label { font-family: Arial, sans-serif; font-size: 12px; }`)
	svg.WriteString(`</style>`)
	svg.WriteString(`</defs>`)

	// Title
	svg.WriteString(`<text x="600" y="30" text-anchor="middle" class="label" font-size="18" font-weight="bold">`)
	svg.WriteString(`Infrastructure Visualization`)
	svg.WriteString(`</text>`)

	// Stats
	svg.WriteString(fmt.Sprintf(`<text x="20" y="60" class="label">Total Resources: %d</text>`, len(data.Resources)))
	svg.WriteString(fmt.Sprintf(`<text x="20" y="80" class="label">Relationships: %d</text>`, len(data.Relationships)))

	// Draw resources as nodes
	nodePositions := e.calculateNodePositions(data.Resources)
	for i, resource := range data.Resources {
		pos := nodePositions[i]
		providerClass := strings.ToLower(resource.Provider)

		// Draw node
		svg.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="120" height="60" rx="5" class="resource %s"/>`,
			pos.X, pos.Y, providerClass))

		// Draw label
		svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" class="label">`,
			pos.X+60, pos.Y+25))
		svg.WriteString(e.truncateText(resource.Name, 15))
		svg.WriteString(`</text>`)

		// Draw type
		svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" class="label" font-size="10">`,
			pos.X+60, pos.Y+40))
		svg.WriteString(resource.Type)
		svg.WriteString(`</text>`)
	}

	// Draw relationships as edges
	for _, rel := range data.Relationships {
		fromIdx := e.findResourceIndex(data.Resources, rel.Source)
		toIdx := e.findResourceIndex(data.Resources, rel.Target)

		if fromIdx >= 0 && toIdx >= 0 {
			fromPos := nodePositions[fromIdx]
			toPos := nodePositions[toIdx]

			svg.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" class="connection" marker-end="url(#arrowhead)"/>`,
				fromPos.X+60, fromPos.Y+60, toPos.X+60, toPos.Y))
		}
	}

	// Arrow marker for dependencies
	svg.WriteString(`<defs>`)
	svg.WriteString(`<marker id="arrowhead" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto">`)
	svg.WriteString(`<polygon points="0 0, 10 3.5, 0 7" fill="#666"/>`)
	svg.WriteString(`</marker>`)
	svg.WriteString(`</defs>`)

	svg.WriteString(`</svg>`)
	return svg.String()
}

// calculateNodePositions calculates positions for nodes in a grid layout
func (e *SVGExporter) calculateNodePositions(resources []models.Resource) []struct{ X, Y int } {
	positions := make([]struct{ X, Y int }, len(resources))
	cols := 5
	nodeWidth := 150
	nodeHeight := 100
	startX := 100
	startY := 120

	for i := range resources {
		col := i % cols
		row := i / cols
		positions[i] = struct{ X, Y int }{
			X: startX + col*nodeWidth,
			Y: startY + row*nodeHeight,
		}
	}

	return positions
}

// findResourceIndex finds the index of a resource by ID
func (e *SVGExporter) findResourceIndex(resources []models.Resource, id string) int {
	for i, r := range resources {
		if r.ID == id {
			return i
		}
	}
	return -1
}

// truncateText truncates text to a maximum length
func (e *SVGExporter) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// PNGExporter exports visualizations to PNG
type PNGExporter struct{}

// NewPNGExporter creates a new PNG exporter
func NewPNGExporter() *PNGExporter {
	return &PNGExporter{}
}

// Format returns the format name
func (e *PNGExporter) Format() string {
	return "png"
}

// Export exports the visualization data to PNG
func (e *PNGExporter) Export(data *VisualizationData, w io.Writer) error {
	// Create a basic PNG visualization
	img := e.generatePNG(data)

	// Encode to PNG
	err := png.Encode(w, img)
	if err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	return nil
}

// generatePNG generates a PNG image from the visualization data
func (e *PNGExporter) generatePNG(data *VisualizationData) image.Image {
	// Create image
	width := 1200
	height := 800
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Colors for different providers
	providerColors := map[string]color.Color{
		"aws":     color.RGBA{255, 153, 0, 255},  // Orange
		"azure":   color.RGBA{0, 120, 212, 255},  // Blue
		"gcp":     color.RGBA{66, 133, 244, 255}, // Blue
		"default": color.RGBA{74, 144, 226, 255}, // Default blue
	}

	// Calculate positions
	nodePositions := e.calculateNodePositions(data.Resources)

	// Draw resources
	for i, resource := range data.Resources {
		pos := nodePositions[i]

		// Get provider color
		providerColor, ok := providerColors[strings.ToLower(resource.Provider)]
		if !ok {
			providerColor = providerColors["default"]
		}

		// Draw resource box
		e.drawRectangle(img, pos.X, pos.Y, 120, 60, providerColor)

		// Draw resource name (simplified - in production would use font rendering)
		e.drawLabel(img, pos.X+60, pos.Y+30, resource.Name, color.White)
	}

	// Draw relationships as lines
	for _, rel := range data.Relationships {
		fromIdx := e.findResourceIndex(data.Resources, rel.Source)
		toIdx := e.findResourceIndex(data.Resources, rel.Target)

		if fromIdx >= 0 && toIdx >= 0 {
			fromPos := nodePositions[fromIdx]
			toPos := nodePositions[toIdx]

			e.drawLine(img, fromPos.X+60, fromPos.Y+60, toPos.X+60, toPos.Y, color.RGBA{102, 102, 102, 255})
		}
	}

	return img
}

// drawRectangle draws a filled rectangle on the image
func (e *PNGExporter) drawRectangle(img *image.RGBA, x, y, width, height int, c color.Color) {
	for i := x; i < x+width && i < img.Bounds().Max.X; i++ {
		for j := y; j < y+height && j < img.Bounds().Max.Y; j++ {
			img.Set(i, j, c)
		}
	}
}

// drawLine draws a line between two points
func (e *PNGExporter) drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	// Bresenham's line algorithm
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	sy := 1

	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}

	err := dx - dy

	for {
		img.Set(x1, y1, c)

		if x1 == x2 && y1 == y2 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// drawLabel draws a text label (simplified - just a marker)
func (e *PNGExporter) drawLabel(img *image.RGBA, x, y int, text string, c color.Color) {
	// In production, would use golang.org/x/image/font to render text
	// For now, just draw a small marker
	for i := x - 2; i <= x+2 && i >= 0 && i < img.Bounds().Max.X; i++ {
		for j := y - 2; j <= y+2 && j >= 0 && j < img.Bounds().Max.Y; j++ {
			img.Set(i, j, c)
		}
	}
}

// calculateNodePositions calculates positions for nodes
func (e *PNGExporter) calculateNodePositions(resources []models.Resource) []struct{ X, Y int } {
	positions := make([]struct{ X, Y int }, len(resources))
	cols := 5
	nodeWidth := 150
	nodeHeight := 100
	startX := 100
	startY := 120

	for i := range resources {
		col := i % cols
		row := i / cols
		positions[i] = struct{ X, Y int }{
			X: startX + col*nodeWidth,
			Y: startY + row*nodeHeight,
		}
	}

	return positions
}

// findResourceIndex finds resource index by ID
func (e *PNGExporter) findResourceIndex(resources []models.Resource, id string) int {
	for i, r := range resources {
		if r.ID == id {
			return i
		}
	}
	return -1
}

// abs returns absolute value
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// JSONExporter exports visualizations to JSON
type JSONExporter struct {
	pretty bool
}

// NewJSONExporter creates a new JSON exporter
func NewJSONExporter(pretty bool) *JSONExporter {
	return &JSONExporter{pretty: pretty}
}

// Format returns the format name
func (e *JSONExporter) Format() string {
	return "json"
}

// Export exports the visualization data to JSON
func (e *JSONExporter) Export(data *VisualizationData, renderer Renderer) ([]byte, error) {
	// For JSON, we can directly export the data
	if e.pretty {
		return json.MarshalIndent(data, "", "  ")
	}
	return json.Marshal(data)
}

// Default HTML template
const defaultHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #4CAF50;
            padding-bottom: 10px;
        }
        .visualization {
            margin: 20px 0;
            padding: 20px;
            background-color: #fafafa;
            border: 1px solid #ddd;
            border-radius: 4px;
        }
        .metadata {
            margin-top: 20px;
            padding: 10px;
            background-color: #f0f0f0;
            border-radius: 4px;
        }
        pre {
            background-color: #f5f5f5;
            padding: 10px;
            border-radius: 4px;
            overflow-x: auto;
        }
        .mermaid {
            text-align: center;
        }
    </style>
    {{if eq .Format "mermaid"}}
    <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
    <script>mermaid.initialize({startOnLoad:true});</script>
    {{end}}
</head>
<body>
    <div class="container">
        <h1>{{.Title}}</h1>
        
        <div class="visualization">
            {{if eq .Format "mermaid"}}
            <div class="mermaid">
{{.Content}}
            </div>
            {{else}}
            <pre>{{.Content}}</pre>
            {{end}}
        </div>
        
        <div class="metadata">
            <h3>Statistics</h3>
            <p>Total Resources: {{len .Data.Resources}}</p>
            <p>Total Relationships: {{len .Data.Relationships}}</p>
            {{if .Data.Options.ShowDrift}}
            <p>Total Drifts: {{len .Data.Drifts}}</p>
            {{end}}
        </div>
    </div>
</body>
</html>`
