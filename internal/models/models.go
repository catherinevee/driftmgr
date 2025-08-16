package models

import (
	"time"
)

// Resource represents a cloud resource
type Resource struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Provider     string                 `json:"provider"`
	Region       string                 `json:"region"`
	Tags         map[string]string      `json:"tags,omitempty"`
	State        string                 `json:"state,omitempty"`
	Status       string                 `json:"status,omitempty"`
	Created      time.Time              `json:"created,omitempty"`
	Updated      time.Time              `json:"updated,omitempty"`
	CreatedAt    time.Time              `json:"created_at,omitempty"`
	LastModified time.Time              `json:"last_modified,omitempty"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// DriftResult represents the result of a drift detection
type DriftResult struct {
	ResourceID    string            `json:"resource_id"`
	ResourceName  string            `json:"resource_name"`
	ResourceType  string            `json:"resource_type"`
	Provider      string            `json:"provider"`
	Region        string            `json:"region"`
	DriftType     string            `json:"drift_type"`
	Severity      string            `json:"severity"`
	Description   string            `json:"description"`
	RiskReasoning string            `json:"risk_reasoning,omitempty"`
	Changes       []DriftChange     `json:"changes,omitempty"`
	DetectedAt    time.Time         `json:"detected_at"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// DriftChange represents a specific change detected in a resource
type DriftChange struct {
	Field      string      `json:"field"`
	OldValue   interface{} `json:"old_value,omitempty"`
	NewValue   interface{} `json:"new_value,omitempty"`
	ChangeType string      `json:"change_type"`
}

// DiscoveryResult represents the result of a resource discovery
type DiscoveryResult struct {
	Resources []Resource       `json:"resources"`
	Summary   DiscoverySummary `json:"summary"`
	Timestamp time.Time        `json:"timestamp"`
}

// DiscoverySummary provides a summary of discovered resources
type DiscoverySummary struct {
	TotalResources int            `json:"total_resources"`
	ByProvider     map[string]int `json:"by_provider"`
	ByRegion       map[string]int `json:"by_region"`
	ByType         map[string]int `json:"by_type"`
}

// AnalysisResult represents the result of a drift analysis
type AnalysisResult struct {
	DriftResults []DriftResult   `json:"drift_results"`
	Summary      AnalysisSummary `json:"summary"`
	Timestamp    time.Time       `json:"timestamp"`
}

// AnalysisSummary provides a summary of drift analysis
type AnalysisSummary struct {
	TotalDrifts    int            `json:"total_drifts"`
	BySeverity     map[string]int `json:"by_severity"`
	ByProvider     map[string]int `json:"by_provider"`
	ByResourceType map[string]int `json:"by_resource_type"`
	CriticalDrifts int            `json:"critical_drifts"`
	HighDrifts     int            `json:"high_drifts"`
	MediumDrifts   int            `json:"medium_drifts"`
	LowDrifts      int            `json:"low_drifts"`

	// Additional fields for perspective analysis
	TotalStateResources   int     `json:"total_state_resources"`
	TotalLiveResources    int     `json:"total_live_resources"`
	Missing               int     `json:"missing"`
	Extra                 int     `json:"extra"`
	Modified              int     `json:"modified"`
	PerspectivePercentage float64 `json:"perspective_percentage"`
	CoveragePercentage    float64 `json:"coverage_percentage"`
	DriftPercentage       float64 `json:"drift_percentage"`
	DriftsFound           int     `json:"drifts_found"`

	// Additional fields for visualization
	TotalResources    int     `json:"total_resources"`
	TotalDependencies int     `json:"total_dependencies"`
	GraphNodes        int     `json:"graph_nodes"`
	GraphEdges        int     `json:"graph_edges"`
	ComplexityScore   float64 `json:"complexity_score"`
	RiskLevel         string  `json:"risk_level"`
}

// Drift represents a drift detection result
type Drift struct {
	ID           string            `json:"id"`
	ResourceID   string            `json:"resource_id"`
	ResourceName string            `json:"resource_name"`
	ResourceType string            `json:"resource_type"`
	Provider     string            `json:"provider"`
	Region       string            `json:"region"`
	DriftType    string            `json:"drift_type"`
	Severity     string            `json:"severity"`
	Description  string            `json:"description"`
	Changes      []DriftChange     `json:"changes,omitempty"`
	DetectedAt   time.Time         `json:"detected_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// StateFile represents a Terraform state file
type StateFile struct {
	Path             string                 `json:"path"`
	Version          int                    `json:"version"`
	TerraformVersion string                 `json:"terraform_version"`
	Serial           int                    `json:"serial"`
	Lineage          string                 `json:"lineage"`
	Outputs          map[string]interface{} `json:"outputs"`
	Resources        []TerraformResource    `json:"resources"`
	// Terragrunt-specific fields
	ManagedByTerragrunt bool              `json:"managed_by_terragrunt,omitempty"`
	TerragruntConfig    *TerragruntConfig `json:"terragrunt_config,omitempty"`
}

// TerragruntConfig represents a Terragrunt configuration file
type TerragruntConfig struct {
	Path                  string                 `json:"path"`
	Source                string                 `json:"source,omitempty"`
	Include               []TerragruntInclude    `json:"include,omitempty"`
	Generate              []TerragruntGenerate   `json:"generate,omitempty"`
	Inputs                map[string]interface{} `json:"inputs,omitempty"`
	RemoteState           *TerragruntRemoteState `json:"remote_state,omitempty"`
	Dependencies          []string               `json:"dependencies,omitempty"`
	BeforeHooks           []TerragruntHook       `json:"before_hooks,omitempty"`
	AfterHooks            []TerragruntHook       `json:"after_hooks,omitempty"`
	ErrorHooks            []TerragruntHook       `json:"error_hooks,omitempty"`
	TerragruntVersion     string                 `json:"terragrunt_version,omitempty"`
	DownloadDir           string                 `json:"download_dir,omitempty"`
	PreventDestroy        bool                   `json:"prevent_destroy,omitempty"`
	Skip                  bool                   `json:"skip,omitempty"`
	IamRole               string                 `json:"iam_role,omitempty"`
	IamAssumeRoleDuration int                    `json:"iam_assume_role_duration,omitempty"`
}

// TerragruntInclude represents an include block in Terragrunt configuration
type TerragruntInclude struct {
	Path   string `json:"path"`
	Name   string `json:"name,omitempty"`
	Expose bool   `json:"expose,omitempty"`
}

// TerragruntGenerate represents a generate block in Terragrunt configuration
type TerragruntGenerate struct {
	Path             string `json:"path"`
	IfExists         string `json:"if_exists,omitempty"`
	Contents         string `json:"contents,omitempty"`
	Comment          string `json:"comment,omitempty"`
	DisableSignature bool   `json:"disable_signature,omitempty"`
}

// TerragruntRemoteState represents remote state configuration in Terragrunt
type TerragruntRemoteState struct {
	Backend                       string                 `json:"backend"`
	Config                        map[string]interface{} `json:"config,omitempty"`
	DisableDependencyOptimization bool                   `json:"disable_dependency_optimization,omitempty"`
	DisableInit                   bool                   `json:"disable_init,omitempty"`
	Generate                      *TerragruntGenerate    `json:"generate,omitempty"`
}

// TerragruntHook represents a hook in Terragrunt configuration
type TerragruntHook struct {
	Name           string   `json:"name"`
	Commands       []string `json:"commands"`
	Execute        []string `json:"execute"`
	RunOnError     bool     `json:"run_on_error,omitempty"`
	SuppressStdout bool     `json:"suppress_stdout,omitempty"`
	SuppressStderr bool     `json:"suppress_stderr,omitempty"`
	WorkingDir     string   `json:"working_dir,omitempty"`
}

// TerragruntFile represents a discovered Terragrunt file
type TerragruntFile struct {
	Path        string            `json:"path"`
	Config      *TerragruntConfig `json:"config"`
	IsRoot      bool              `json:"is_root"`
	ParentPath  string            `json:"parent_path,omitempty"`
	ChildPaths  []string          `json:"child_paths,omitempty"`
	StatePath   string            `json:"state_path,omitempty"`
	ModuleName  string            `json:"module_name,omitempty"`
	Environment string            `json:"environment,omitempty"`
	Region      string            `json:"region,omitempty"`
	Account     string            `json:"account,omitempty"`
}

// TerragruntDiscoveryResult represents the result of Terragrunt file discovery
type TerragruntDiscoveryResult struct {
	RootFiles    []TerragruntFile `json:"root_files"`
	ChildFiles   []TerragruntFile `json:"child_files"`
	TotalFiles   int              `json:"total_files"`
	Environments []string         `json:"environments"`
	Regions      []string         `json:"regions"`
	Accounts     []string         `json:"accounts"`
	Timestamp    time.Time        `json:"timestamp"`
}

// TerraformResource represents a resource in a Terraform state file
type TerraformResource struct {
	Mode      string                      `json:"mode"`
	Type      string                      `json:"type"`
	Name      string                      `json:"name"`
	Provider  string                      `json:"provider"`
	Instances []TerraformResourceInstance `json:"instances"`
}

// TerraformResourceInstance represents an instance of a Terraform resource
type TerraformResourceInstance struct {
	SchemaVersion       int                    `json:"schema_version"`
	Attributes          map[string]interface{} `json:"attributes"`
	SensitiveAttributes []string               `json:"sensitive_attributes,omitempty"`
	Private             string                 `json:"private,omitempty"`
}

// DiscoveryRequest represents a resource discovery request
type DiscoveryRequest struct {
	Provider string   `json:"provider"`
	Regions  []string `json:"regions"`
	Account  string   `json:"account"`
}

// DiscoveryResponse represents a resource discovery response
type DiscoveryResponse struct {
	Resources []Resource    `json:"resources"`
	Total     int           `json:"total"`
	Duration  time.Duration `json:"duration"`
}

// AnalysisRequest represents a drift analysis request
type AnalysisRequest struct {
	StateFileID string          `json:"state_file_id"`
	Resources   []Resource      `json:"resources"`
	Options     AnalysisOptions `json:"options"`
}

// AnalysisOptions represents options for drift analysis
type AnalysisOptions struct {
	IncludeTags     bool `json:"include_tags"`
	IncludeMetadata bool `json:"include_metadata"`
	GenerateImports bool `json:"generate_imports"`
}

// AnalysisResponse represents a drift analysis response
type AnalysisResponse struct {
	Summary  AnalysisSummary `json:"summary"`
	Duration time.Duration   `json:"duration"`
}

// PerspectiveRequest represents a perspective analysis request
type PerspectiveRequest struct {
	StateFileID string `json:"state_file_id"`
	Provider    string `json:"provider"`
}

// PerspectiveResponse represents a perspective analysis response
type PerspectiveResponse struct {
	Summary        AnalysisSummary `json:"summary"`
	ImportCommands []string        `json:"import_commands"`
	Duration       time.Duration   `json:"duration"`
}

// VisualizationRequest represents a visualization request
type VisualizationRequest struct {
	StateFileID   string `json:"state_file_id"`
	TerraformPath string `json:"terraform_path"`
}

// VisualizationResponse represents a visualization response
type VisualizationResponse struct {
	StateFileID   string                `json:"state_file_id"`
	TerraformPath string                `json:"terraform_path"`
	Summary       AnalysisSummary       `json:"summary"`
	Outputs       []VisualizationOutput `json:"outputs"`
	Duration      time.Duration         `json:"duration"`
	GeneratedAt   time.Time             `json:"generated_at"`
}

// VisualizationSummary represents visualization summary data
type VisualizationSummary struct {
	TotalResources    int     `json:"total_resources"`
	TotalDependencies int     `json:"total_dependencies"`
	GraphNodes        int     `json:"graph_nodes"`
	GraphEdges        int     `json:"graph_edges"`
	ComplexityScore   float64 `json:"complexity_score"`
	RiskLevel         string  `json:"risk_level"`
}

// VisualizationOutput represents a visualization output
type VisualizationOutput struct {
	Format string `json:"format"`
	Path   string `json:"path"`
	URL    string `json:"url"`
}

// DiagramResponse represents a diagram generation response
type DiagramResponse struct {
	StateFileID string        `json:"state_file_id"`
	Status      string        `json:"status"`
	Message     string        `json:"message"`
	Duration    time.Duration `json:"duration"`
	GeneratedAt time.Time     `json:"generated_at"`
	DiagramData DiagramData   `json:"diagram_data"`
}

// DiagramData represents diagram data
type DiagramData struct {
	Resources    []Resource   `json:"resources"`
	DataSources  []DataSource `json:"data_sources"`
	Dependencies []Dependency `json:"dependencies"`
	Modules      []Module     `json:"modules"`
	Path         string       `json:"path"`
	ParsedAt     time.Time    `json:"parsed_at"`
}

// DiagramNode represents a node in a diagram
type DiagramNode struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	Position Position          `json:"position"`
	Metadata map[string]string `json:"metadata"`
}

// DiagramEdge represents an edge in a diagram
type DiagramEdge struct {
	Source   string            `json:"source"`
	Target   string            `json:"target"`
	Type     string            `json:"type"`
	Metadata map[string]string `json:"metadata"`
}

// Position represents a position in 2D space
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// DataSource represents a data source
type DataSource struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Provider string            `json:"provider"`
	Region   string            `json:"region"`
	Config   map[string]string `json:"config"`
}

// Module represents a Terraform module
type Module struct {
	Name      string   `json:"name"`
	Source    string   `json:"source"`
	Version   string   `json:"version"`
	Resources []string `json:"resources"`
}

// Dependency represents a dependency relationship
type Dependency struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// ExportRequest represents an export request
type ExportRequest struct {
	Format string `json:"format"`
}

// ExportResponse represents an export response
type ExportResponse struct {
	StateFileID string    `json:"state_file_id"`
	Format      string    `json:"format"`
	OutputPath  string    `json:"output_path"`
	URL         string    `json:"url"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	ExportedAt  time.Time `json:"exported_at"`
}

// NotificationRequest represents a notification request
type NotificationRequest struct {
	Type       string   `json:"type"`
	Recipients []string `json:"recipients"`
	Subject    string   `json:"subject"`
	Message    string   `json:"message"`
	Priority   string   `json:"priority"`
}

// NotificationResponse represents a notification response
type NotificationResponse struct {
	Success   bool      `json:"success"`
	MessageID string    `json:"message_id"`
	Errors    []string  `json:"errors,omitempty"`
	SentAt    time.Time `json:"sent_at"`
}

// EnhancedAnalysisRequest represents a request for enhanced drift analysis
type EnhancedAnalysisRequest struct {
	StateFileID     string         `json:"state_file_id"`
	SensitiveFields []string       `json:"sensitive_fields,omitempty"`
	IgnoreFields    []string       `json:"ignore_fields,omitempty"`
	SeverityRules   []SeverityRule `json:"severity_rules,omitempty"`
	ConfigFile      string         `json:"config_file,omitempty"`
	OutputFormat    string         `json:"output_format,omitempty"`
}

// SeverityRule defines a custom severity rule for drift detection
type SeverityRule struct {
	ResourceType  string `json:"resource_type"`
	AttributePath string `json:"attribute_path"`
	Condition     string `json:"condition"`
	Severity      string `json:"severity"`
	Description   string `json:"description"`
}

// RemediationRequest represents a request for remediation
type RemediationRequest struct {
	DriftID     string `json:"drift_id"`
	AutoApprove bool   `json:"auto_approve"`
	DryRun      bool   `json:"dry_run"`
}

// RemediationResult represents the result of a remediation action
type RemediationResult struct {
	DriftID      string    `json:"drift_id"`
	Status       string    `json:"status"`
	Commands     []string  `json:"commands"`
	Approved     bool      `json:"approved"`
	Executed     bool      `json:"executed"`
	Timestamp    time.Time `json:"timestamp"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// BatchRemediationRequest represents a request for batch remediation
type BatchRemediationRequest struct {
	StateFileID    string `json:"state_file_id"`
	SeverityFilter string `json:"severity_filter,omitempty"`
	AutoApprove    bool   `json:"auto_approve"`
	DryRun         bool   `json:"dry_run"`
}

// BatchRemediationResult represents the result of batch remediation
type BatchRemediationResult struct {
	StateFileID string              `json:"state_file_id"`
	TotalDrifts int                 `json:"total_drifts"`
	Remediated  int                 `json:"remediated"`
	Failed      int                 `json:"failed"`
	Results     []RemediationResult `json:"results"`
	Timestamp   time.Time           `json:"timestamp"`
}

// RemediationHistory represents the history of remediation actions
type RemediationHistory struct {
	History   []RemediationResult `json:"history"`
	Timestamp time.Time           `json:"timestamp"`
}

// RollbackRequest represents a request to rollback to a previous state
type RollbackRequest struct {
	SnapshotID string `json:"snapshot_id"`
}

// RollbackResult represents the result of a rollback action
type RollbackResult struct {
	SnapshotID   string    `json:"snapshot_id"`
	Status       string    `json:"status"`
	RolledBack   bool      `json:"rolled_back"`
	Timestamp    time.Time `json:"timestamp"`
	ErrorMessage string    `json:"error_message,omitempty"`
}
