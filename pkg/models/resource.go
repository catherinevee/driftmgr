package models

import (
	"time"
)

// Resource represents a cloud resource
type Resource struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Provider   string                 `json:"provider"`
	Region     string                 `json:"region"`
	Account    string                 `json:"account"`
	Tags       map[string]string      `json:"tags"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	State      string                 `json:"state"`
}

// StateFile represents a Terraform state file
type StateFile struct {
	Path       string     `json:"path"`
	Version    int        `json:"version"`
	Resources  []Resource `json:"resources"`
	Error      string     `json:"error,omitempty"`
	LastParsed time.Time  `json:"last_parsed"`
}

// DriftResult represents the result of drift analysis
type DriftResult struct {
	ResourceID    string            `json:"resource_id"`
	ResourceName  string            `json:"resource_name"`
	ResourceType  string            `json:"resource_type"`
	Provider      string            `json:"provider"`
	Region        string            `json:"region"`
	DriftType     string            `json:"drift_type"` // "missing", "extra", "modified"
	Changes       map[string]Change `json:"changes"`
	Severity      string            `json:"severity"` // "low", "medium", "high", "critical"
	Description   string            `json:"description"`
	DetectedAt    time.Time         `json:"detected_at"`
	ImportCommand string            `json:"import_command,omitempty"`
}

// Change represents a specific change in a resource
type Change struct {
	Field      string      `json:"field"`
	OldValue   interface{} `json:"old_value"`
	NewValue   interface{} `json:"new_value"`
	ChangeType string      `json:"change_type"` // "added", "removed", "modified"
}

// DiscoveryRequest represents a request for resource discovery
type DiscoveryRequest struct {
	Provider string            `json:"provider"`
	Regions  []string          `json:"regions"`
	Account  string            `json:"account"`
	Filters  map[string]string `json:"filters"`
}

// DiscoveryResponse represents the response from resource discovery
type DiscoveryResponse struct {
	Resources []Resource    `json:"resources"`
	Total     int           `json:"total"`
	Errors    []string      `json:"errors"`
	Duration  time.Duration `json:"duration"`
}

// AnalysisRequest represents a request for drift analysis
type AnalysisRequest struct {
	StateFileID string          `json:"state_file_id"`
	Resources   []Resource      `json:"resources"`
	Options     AnalysisOptions `json:"options"`
}

// AnalysisOptions represents options for drift analysis
type AnalysisOptions struct {
	IncludeTags     bool `json:"include_tags"`
	IncludeMetadata bool `json:"include_metadata"`
	StrictMode      bool `json:"strict_mode"`
	GenerateImports bool `json:"generate_imports"`
}

// AnalysisResponse represents the response from drift analysis
type AnalysisResponse struct {
	Drifts   []DriftResult   `json:"drifts"`
	Summary  AnalysisSummary `json:"summary"`
	Duration time.Duration   `json:"duration"`
	Errors   []string        `json:"errors"`
}

// AnalysisSummary represents a summary of drift analysis
type AnalysisSummary struct {
	TotalResources int `json:"total_resources"`
	DriftsFound    int `json:"drifts_found"`
	Missing        int `json:"missing"`
	Extra          int `json:"extra"`
	Modified       int `json:"modified"`
	Critical       int `json:"critical"`
	High           int `json:"high"`
	Medium         int `json:"medium"`
	Low            int `json:"low"`
}

// DriftAnalysis represents a drift analysis result
type DriftAnalysis struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceType string                 `json:"resource_type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	DriftType    string                 `json:"drift_type"`
	Severity     string                 `json:"severity"`
	Description  string                 `json:"description"`
	Metadata     map[string]interface{} `json:"metadata"`
	Timestamp    time.Time              `json:"timestamp"`
}

// NotificationRequest represents a request to send notifications
type NotificationRequest struct {
	Type       string                 `json:"type"` // "email", "slack", "webhook"
	Recipients []string               `json:"recipients"`
	Subject    string                 `json:"subject"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data"`
	Priority   string                 `json:"priority"` // "low", "normal", "high", "urgent"
}

// NotificationResponse represents the response from notification service
type NotificationResponse struct {
	Success   bool      `json:"success"`
	MessageID string    `json:"message_id,omitempty"`
	Errors    []string  `json:"errors,omitempty"`
	SentAt    time.Time `json:"sent_at"`
}

// PerspectiveRequest represents a request for perspective analysis
type PerspectiveRequest struct {
	StateFileID string `json:"state_file_id"`
	Provider    string `json:"provider"`
}

// PerspectiveResponse represents the response from perspective analysis
type PerspectiveResponse struct {
	StateResources []Resource          `json:"state_resources"`
	LiveResources  []Resource          `json:"live_resources"`
	Missing        []Resource          `json:"missing"`
	Extra          []Resource          `json:"extra"`
	Modified       []PerspectiveChange `json:"modified"`
	Summary        PerspectiveSummary  `json:"summary"`
	ImportCommands []string            `json:"import_commands"`
	Duration       time.Duration       `json:"duration"`
	Errors         []string            `json:"errors"`
}

// PerspectiveChange represents a change detected in perspective analysis
type PerspectiveChange struct {
	ResourceID   string            `json:"resource_id"`
	ResourceName string            `json:"resource_name"`
	ResourceType string            `json:"resource_type"`
	Provider     string            `json:"provider"`
	Region       string            `json:"region"`
	Changes      map[string]Change `json:"changes"`
	Description  string            `json:"description"`
}

// PerspectiveSummary represents a summary of perspective analysis
type PerspectiveSummary struct {
	TotalStateResources   int     `json:"total_state_resources"`
	TotalLiveResources    int     `json:"total_live_resources"`
	Missing               int     `json:"missing"`
	Extra                 int     `json:"extra"`
	Modified              int     `json:"modified"`
	PerspectivePercentage float64 `json:"perspective_percentage"`
	CoveragePercentage    float64 `json:"coverage_percentage"`
	DriftPercentage       float64 `json:"drift_percentage"`
}

// VisualizationRequest represents a request for visualization generation
type VisualizationRequest struct {
	StateFileID   string `json:"state_file_id"`
	TerraformPath string `json:"terraform_path"`
	Format        string `json:"format,omitempty"`
	OutputDir     string `json:"output_dir,omitempty"`
}

// VisualizationResponse represents the response from visualization service
type VisualizationResponse struct {
	StateFileID   string               `json:"state_file_id"`
	TerraformPath string               `json:"terraform_path"`
	DiagramData   DiagramData          `json:"diagram_data"`
	Outputs       []DiagramOutput      `json:"outputs"`
	Summary       VisualizationSummary `json:"summary"`
	Duration      time.Duration        `json:"duration"`
	GeneratedAt   time.Time            `json:"generated_at"`
}

// DiagramData represents the data structure for dependency diagrams
type DiagramData struct {
	Resources    []Resource   `json:"resources"`
	DataSources  []DataSource `json:"data_sources"`
	Dependencies []Dependency `json:"dependencies"`
	Modules      []Module     `json:"modules"`
	Path         string       `json:"path"`
	ParsedAt     time.Time    `json:"parsed_at"`
}

// DataSource represents a Terraform data source
type DataSource struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Provider string `json:"provider"`
	Region   string `json:"region"`
}

// Dependency represents a dependency between resources
type Dependency struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // "resource", "data", "module"
}

// Module represents a Terraform module
type Module struct {
	Name      string   `json:"name"`
	Source    string   `json:"source"`
	Version   string   `json:"version"`
	Resources []string `json:"resources"`
}

// DiagramOutput represents an output file from diagram generation
type DiagramOutput struct {
	Format string `json:"format"`
	Path   string `json:"path"`
	URL    string `json:"url"`
}

// VisualizationSummary represents a summary of visualization analysis
type VisualizationSummary struct {
	TotalResources    int     `json:"total_resources"`
	TotalDependencies int     `json:"total_dependencies"`
	GraphNodes        int     `json:"graph_nodes"`
	GraphEdges        int     `json:"graph_edges"`
	ComplexityScore   float64 `json:"complexity_score"`
	RiskLevel         string  `json:"risk_level"`
}

// DiagramResponse represents a response for diagram generation
type DiagramResponse struct {
	StateFileID string        `json:"state_file_id"`
	DiagramData DiagramData   `json:"diagram_data"`
	GeneratedAt time.Time     `json:"generated_at"`
	Duration    time.Duration `json:"duration"`
	Status      string        `json:"status"`
	Message     string        `json:"message"`
}

// ExportRequest represents a request for diagram export
type ExportRequest struct {
	Format    string `json:"format"` // "html", "svg", "png", "json"
	OutputDir string `json:"output_dir,omitempty"`
}

// ExportResponse represents a response for diagram export
type ExportResponse struct {
	StateFileID string    `json:"state_file_id"`
	Format      string    `json:"format"`
	OutputPath  string    `json:"output_path"`
	URL         string    `json:"url"`
	ExportedAt  time.Time `json:"exported_at"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
}

// AnalysisResult represents the result of drift analysis
type AnalysisResult struct {
	Drifts   []DriftResult   `json:"drifts"`
	Summary  AnalysisSummary `json:"summary"`
	Duration time.Duration   `json:"duration"`
	Errors   []string        `json:"errors"`
}

// TerragruntDiscoveryResult represents the result of Terragrunt configuration discovery
type TerragruntDiscoveryResult struct {
	TotalFiles   int              `json:"total_files"`
	RootFiles    []TerragruntFile `json:"root_files"`
	ChildFiles   []TerragruntFile `json:"child_files"`
	Environments []string         `json:"environments"`
	Regions      []string         `json:"regions"`
	Accounts     []string         `json:"accounts"`
	Duration     time.Duration    `json:"duration"`
	Errors       []string         `json:"errors"`
}

// TerragruntFile represents a discovered Terragrunt configuration file
type TerragruntFile struct {
	Path     string            `json:"path"`
	Type     string            `json:"type"` // "root" or "child"
	Config   *TerragruntConfig `json:"config"`
	ParsedAt time.Time         `json:"parsed_at"`
	Error    string            `json:"error,omitempty"`
}

// TerragruntConfig represents the configuration within a Terragrunt file
type TerragruntConfig struct {
	Source       string                   `json:"source"`
	Inputs       map[string]interface{}   `json:"inputs"`
	Backend      map[string]interface{}   `json:"backend"`
	Hooks        []map[string]interface{} `json:"hooks"`
	Generate     map[string]interface{}   `json:"generate"`
	Dependencies []string                 `json:"dependencies"`
}
