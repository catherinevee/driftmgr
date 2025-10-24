package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	var phase = flag.String("phase", "all", "Phase to generate docs for (1, 2, 3, 4, 5, 6, or all)")
	var allPhases = flag.Bool("all-phases", false, "Generate documentation for all phases")
	flag.Parse()

	if *allPhases {
		*phase = "all"
	}

	fmt.Printf("Generating API documentation for phase: %s\n", *phase)

	// Create docs directory if it doesn't exist
	docsDir := "docs/api"
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		fmt.Printf("Error creating docs directory: %v\n", err)
		os.Exit(1)
	}

	// Generate documentation for each phase
	phases := []string{"1", "2", "3", "4", "5", "6"}
	if *phase != "all" {
		phases = []string{*phase}
	}

	for _, phaseNum := range phases {
		generatePhaseDocs(phaseNum, docsDir)
	}

	// Generate main API documentation
	generateMainAPIDocs(docsDir)

	fmt.Println("✅ API documentation generation complete")
}

func generatePhaseDocs(phase, docsDir string) {
	phaseDir := filepath.Join(docsDir, "phase"+phase)
	if err := os.MkdirAll(phaseDir, 0755); err != nil {
		fmt.Printf("Error creating phase directory: %v\n", err)
		return
	}

	// Define phase-specific documentation
	phaseInfo := map[string]struct {
		title       string
		description string
		endpoints   []EndpointInfo
	}{
		"1": {
			title:       "Phase 1: Drift Results & History Management",
			description: "API endpoints for managing drift detection results and historical data.",
			endpoints: []EndpointInfo{
				{
					Method:      "GET",
					Path:        "/api/v1/drift/results/{id}",
					Description: "Get specific drift detection result",
					Parameters:  []Parameter{{Name: "id", Type: "string", Required: true, Description: "Drift result ID"}},
					Response:    "DriftResult",
				},
				{
					Method:      "GET",
					Path:        "/api/v1/drift/history",
					Description: "Get drift detection history",
					Parameters:  []Parameter{{Name: "limit", Type: "integer", Required: false, Description: "Number of results to return"}},
					Response:    "[]DriftResult",
				},
				{
					Method:      "GET",
					Path:        "/api/v1/drift/summary",
					Description: "Get drift summary statistics",
					Response:    "DriftSummary",
				},
				{
					Method:      "GET",
					Path:        "/api/v1/drift/results",
					Description: "List all drift results (paginated)",
					Parameters:  []Parameter{{Name: "page", Type: "integer", Required: false, Description: "Page number"}},
					Response:    "PaginatedDriftResults",
				},
				{
					Method:      "DELETE",
					Path:        "/api/v1/drift/results/{id}",
					Description: "Delete drift result",
					Parameters:  []Parameter{{Name: "id", Type: "string", Required: true, Description: "Drift result ID"}},
					Response:    "SuccessResponse",
				},
			},
		},
		"2": {
			title:       "Phase 2: Remediation Engine",
			description: "API endpoints for automated remediation of drifted resources.",
			endpoints: []EndpointInfo{
				{
					Method:      "POST",
					Path:        "/api/v1/remediation/apply",
					Description: "Apply remediation to drifted resources",
					Body:        "RemediationRequest",
					Response:    "RemediationJob",
				},
				{
					Method:      "POST",
					Path:        "/api/v1/remediation/preview",
					Description: "Preview remediation actions",
					Body:        "RemediationRequest",
					Response:    "RemediationPreview",
				},
				{
					Method:      "GET",
					Path:        "/api/v1/remediation/status/{id}",
					Description: "Get remediation job status",
					Parameters:  []Parameter{{Name: "id", Type: "string", Required: true, Description: "Remediation job ID"}},
					Response:    "RemediationJob",
				},
				{
					Method:      "GET",
					Path:        "/api/v1/remediation/history",
					Description: "Get remediation history",
					Response:    "[]RemediationJob",
				},
				{
					Method:      "POST",
					Path:        "/api/v1/remediation/cancel/{id}",
					Description: "Cancel running remediation",
					Parameters:  []Parameter{{Name: "id", Type: "string", Required: true, Description: "Remediation job ID"}},
					Response:    "SuccessResponse",
				},
				{
					Method:      "GET",
					Path:        "/api/v1/remediation/strategies",
					Description: "List available remediation strategies",
					Response:    "[]RemediationStrategy",
				},
			},
		},
		// Add other phases...
	}

	info, exists := phaseInfo[phase]
	if !exists {
		fmt.Printf("Warning: No documentation defined for phase %s\n", phase)
		return
	}

	// Generate phase documentation
	content := generatePhaseMarkdown(info)
	filename := filepath.Join(phaseDir, "README.md")
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Printf("Error writing phase documentation: %v\n", err)
		return
	}

	// Generate individual endpoint documentation
	for _, endpoint := range info.endpoints {
		endpointContent := generateEndpointMarkdown(endpoint)
		endpointFilename := strings.ReplaceAll(endpoint.Path, "/", "_")
		endpointFilename = strings.ReplaceAll(endpointFilename, "{", "")
		endpointFilename = strings.ReplaceAll(endpointFilename, "}", "")
		endpointFilename = strings.ToLower(endpointFilename) + ".md"
		endpointPath := filepath.Join(phaseDir, endpointFilename)

		if err := os.WriteFile(endpointPath, []byte(endpointContent), 0644); err != nil {
			fmt.Printf("Error writing endpoint documentation: %v\n", err)
		}
	}

	fmt.Printf("✅ Generated documentation for phase %s\n", phase)
}

func generateMainAPIDocs(docsDir string) {
	content := `# DriftMgr API Documentation

## Overview
DriftMgr provides a comprehensive REST API for infrastructure drift detection, remediation, and management across multiple cloud providers.

## Base URL
\`\`\`
https://api.driftmgr.com/v1
\`\`\`

## Authentication
All API requests require authentication using API keys or OAuth tokens.

\`\`\`bash
curl -H "Authorization: Bearer YOUR_API_KEY" https://api.driftmgr.com/v1/health
\`\`\`

## Rate Limiting
API requests are rate limited to 1000 requests per hour per API key.

## Response Format
All API responses follow a consistent JSON format:

\`\`\`json
{
  "data": {},
  "meta": {
    "timestamp": "2024-01-01T00:00:00Z",
    "version": "1.0.0"
  },
  "errors": []
}
\`\`\`

## Error Handling
Errors are returned with appropriate HTTP status codes and error details:

\`\`\`json
{
  "errors": [
    {
      "code": "VALIDATION_ERROR",
      "message": "Invalid request parameters",
      "details": {
        "field": "resource_id",
        "reason": "required field is missing"
      }
    }
  ]
}
\`\`\`

## API Phases

### Phase 1: Drift Results & History Management
- [Drift Results API](./phase1/README.md)

### Phase 2: Remediation Engine  
- [Remediation API](./phase2/README.md)

### Phase 3: Enhanced State Management
- [State Management API](./phase3/README.md)

### Phase 4: Advanced Discovery & Scanning
- [Discovery API](./phase4/README.md)

### Phase 5: Configuration & Provider Management
- [Configuration API](./phase5/README.md)

### Phase 6: Monitoring & Observability
- [Monitoring API](./phase6/README.md)

## SDKs and Examples
- [Go SDK](./sdk/go/README.md)
- [Python SDK](./sdk/python/README.md)
- [JavaScript SDK](./sdk/javascript/README.md)

## Support
For API support and questions, please contact:
- Email: api-support@driftmgr.com
- Documentation: https://docs.driftmgr.com
- Status Page: https://status.driftmgr.com

---
*Generated on ` + time.Now().Format("2006-01-02 15:04:05") + `*
`

	filename := filepath.Join(docsDir, "README.md")
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		fmt.Printf("Error writing main API documentation: %v\n", err)
	}
}

type EndpointInfo struct {
	Method      string
	Path        string
	Description string
	Parameters  []Parameter
	Body        string
	Response    string
}

type Parameter struct {
	Name        string
	Type        string
	Required    bool
	Description string
}

func generatePhaseMarkdown(info struct {
	title       string
	description string
	endpoints   []EndpointInfo
}) string {
	content := fmt.Sprintf("# %s\n\n%s\n\n", info.title, info.description)

	content += "## Endpoints\n\n"
	for _, endpoint := range info.endpoints {
		content += fmt.Sprintf("### %s %s\n", endpoint.Method, endpoint.Path)
		content += fmt.Sprintf("%s\n\n", endpoint.Description)

		if len(endpoint.Parameters) > 0 {
			content += "**Parameters:**\n\n"
			content += "| Name | Type | Required | Description |\n"
			content += "|------|------|----------|-------------|\n"
			for _, param := range endpoint.Parameters {
				required := "No"
				if param.Required {
					required = "Yes"
				}
				content += fmt.Sprintf("| %s | %s | %s | %s |\n", param.Name, param.Type, required, param.Description)
			}
			content += "\n"
		}

		if endpoint.Body != "" {
			content += fmt.Sprintf("**Request Body:** %s\n\n", endpoint.Body)
		}

		content += fmt.Sprintf("**Response:** %s\n\n", endpoint.Response)
		content += "---\n\n"
	}

	return content
}

func generateEndpointMarkdown(endpoint EndpointInfo) string {
	content := fmt.Sprintf("# %s %s\n\n", endpoint.Method, endpoint.Path)
	content += fmt.Sprintf("%s\n\n", endpoint.Description)

	if len(endpoint.Parameters) > 0 {
		content += "## Parameters\n\n"
		for _, param := range endpoint.Parameters {
			content += fmt.Sprintf("- **%s** (`%s`)", param.Name, param.Type)
			if param.Required {
				content += " - Required"
			}
			content += fmt.Sprintf(": %s\n", param.Description)
		}
		content += "\n"
	}

	if endpoint.Body != "" {
		content += fmt.Sprintf("## Request Body\n\n```json\n%s\n```\n\n", endpoint.Body)
	}

	content += fmt.Sprintf("## Response\n\n```json\n%s\n```\n\n", endpoint.Response)

	content += "## Example\n\n"
	content += "```bash\n"
	content += fmt.Sprintf("curl -X %s \\\n", endpoint.Method)
	content += "  -H \"Authorization: Bearer YOUR_API_KEY\" \\\n"
	content += "  -H \"Content-Type: application/json\" \\\n"
	content += fmt.Sprintf("  https://api.driftmgr.com%s\n", endpoint.Path)
	content += "```\n"

	return content
}
