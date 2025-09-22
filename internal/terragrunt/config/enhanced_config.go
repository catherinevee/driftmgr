package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TerragruntConfig represents enhanced Terragrunt configuration
type TerragruntConfig struct {
	// Basic configuration
	RemoteState  *RemoteStateConfig     `json:"remote_state,omitempty"`
	Inputs       map[string]interface{} `json:"inputs,omitempty"`
	Locals       map[string]interface{} `json:"locals,omitempty"`
	Dependencies *DependenciesConfig    `json:"dependencies,omitempty"`
	Generate     *GenerateConfig        `json:"generate,omitempty"`

	// Enhanced configuration fields
	RetryableErrors []string       `json:"retryable_errors,omitempty"`
	PreventDestroy  bool           `json:"prevent_destroy,omitempty"`
	IncludeConfig   *IncludeConfig `json:"include,omitempty"`

	// Additional configuration
	Skip                           bool                 `json:"skip,omitempty"`
	IamRole                        string               `json:"iam_role,omitempty"`
	IamAssumeRole                  *IamAssumeRoleConfig `json:"iam_assume_role,omitempty"`
	IamAssumeRoleDuration          string               `json:"iam_assume_role_duration,omitempty"`
	IamAssumeRoleSessionName       string               `json:"iam_assume_role_session_name,omitempty"`
	IamAssumeRoleExternalID        string               `json:"iam_assume_role_external_id,omitempty"`
	IamAssumeRoleTags              map[string]string    `json:"iam_assume_role_tags,omitempty"`
	IamAssumeRoleTransitiveTagKeys []string             `json:"iam_assume_role_transitive_tag_keys,omitempty"`

	// Terraform configuration
	Terraform *TerraformConfig `json:"terraform,omitempty"`

	// Error handling
	ErrorHandling *ErrorHandlingConfig `json:"error_handling,omitempty"`

	// Performance settings
	Performance *PerformanceConfig `json:"performance,omitempty"`
}

// RemoteStateConfig represents remote state configuration
type RemoteStateConfig struct {
	Backend     string                     `json:"backend"`
	Config      map[string]interface{}     `json:"config"`
	DisableInit bool                       `json:"disable_init,omitempty"`
	Generate    *RemoteStateGenerateConfig `json:"generate,omitempty"`
}

// RemoteStateGenerateConfig represents remote state generation configuration
type RemoteStateGenerateConfig struct {
	Path     string `json:"path"`
	IfExists string `json:"if_exists,omitempty"`
}

// DependenciesConfig represents dependencies configuration
type DependenciesConfig struct {
	Paths []string `json:"paths,omitempty"`
}

// GenerateConfig represents code generation configuration
type GenerateConfig struct {
	Provider *GenerateProviderConfig `json:"provider,omitempty"`
}

// GenerateProviderConfig represents provider generation configuration
type GenerateProviderConfig struct {
	Path     string `json:"path"`
	IfExists string `json:"if_exists,omitempty"`
	Contents string `json:"contents"`
}

// IncludeConfig represents include configuration
type IncludeConfig struct {
	Path   string `json:"path"`
	Expose bool   `json:"expose,omitempty"`
}

// IamAssumeRoleConfig represents IAM assume role configuration
type IamAssumeRoleConfig struct {
	RoleARN           string            `json:"role_arn"`
	Duration          string            `json:"duration,omitempty"`
	SessionName       string            `json:"session_name,omitempty"`
	ExternalID        string            `json:"external_id,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`
	TransitiveTagKeys []string          `json:"transitive_tag_keys,omitempty"`
}

// TerraformConfig represents Terraform configuration
type TerraformConfig struct {
	Source      string       `json:"source,omitempty"`
	Version     string       `json:"version,omitempty"`
	ExtraArgs   []string     `json:"extra_arguments,omitempty"`
	BeforeHooks []HookConfig `json:"before_hook,omitempty"`
	AfterHooks  []HookConfig `json:"after_hook,omitempty"`
	ErrorHooks  []HookConfig `json:"error_hook,omitempty"`
}

// HookConfig represents hook configuration
type HookConfig struct {
	Commands       []string `json:"commands"`
	Execute        []string `json:"execute"`
	RunOnError     bool     `json:"run_on_error,omitempty"`
	SuppressStdout bool     `json:"suppress_stdout,omitempty"`
	SuppressStderr bool     `json:"suppress_stderr,omitempty"`
}

// ErrorHandlingConfig represents error handling configuration
type ErrorHandlingConfig struct {
	RetryableErrors []string `json:"retryable_errors,omitempty"`
	MaxRetries      int      `json:"max_retries,omitempty"`
	RetryDelay      string   `json:"retry_delay,omitempty"`
	IgnoreErrors    []string `json:"ignore_errors,omitempty"`
}

// PerformanceConfig represents performance configuration
type PerformanceConfig struct {
	Parallelism int    `json:"parallelism,omitempty"`
	Timeout     string `json:"timeout,omitempty"`
	Cache       bool   `json:"cache,omitempty"`
}

// NewTerragruntConfig creates a new TerragruntConfig with default values
func NewTerragruntConfig() *TerragruntConfig {
	return &TerragruntConfig{
		Inputs:                         make(map[string]interface{}),
		Locals:                         make(map[string]interface{}),
		RetryableErrors:                []string{},
		PreventDestroy:                 false,
		Skip:                           false,
		IamAssumeRoleTags:              make(map[string]string),
		IamAssumeRoleTransitiveTagKeys: []string{},
	}
}

// AddRetryableError adds a retryable error pattern
func (tc *TerragruntConfig) AddRetryableError(errorPattern string) {
	if errorPattern == "" {
		return
	}

	// Check if error pattern already exists
	for _, existing := range tc.RetryableErrors {
		if existing == errorPattern {
			return
		}
	}

	tc.RetryableErrors = append(tc.RetryableErrors, errorPattern)
}

// RemoveRetryableError removes a retryable error pattern
func (tc *TerragruntConfig) RemoveRetryableError(errorPattern string) {
	if errorPattern == "" {
		return
	}

	var filtered []string
	for _, existing := range tc.RetryableErrors {
		if existing != errorPattern {
			filtered = append(filtered, existing)
		}
	}
	tc.RetryableErrors = filtered
}

// HasRetryableError checks if an error pattern is retryable
func (tc *TerragruntConfig) HasRetryableError(errorPattern string) bool {
	if errorPattern == "" {
		return false
	}

	for _, retryable := range tc.RetryableErrors {
		if strings.Contains(errorPattern, retryable) {
			return true
		}
	}
	return false
}

// SetIncludeConfig sets the include configuration
func (tc *TerragruntConfig) SetIncludeConfig(path string, expose bool) {
	tc.IncludeConfig = &IncludeConfig{
		Path:   path,
		Expose: expose,
	}
}

// GetIncludePath returns the include path
func (tc *TerragruntConfig) GetIncludePath() string {
	if tc.IncludeConfig != nil {
		return tc.IncludeConfig.Path
	}
	return ""
}

// IsIncludeExposed returns whether the include is exposed
func (tc *TerragruntConfig) IsIncludeExposed() bool {
	if tc.IncludeConfig != nil {
		return tc.IncludeConfig.Expose
	}
	return false
}

// SetPreventDestroy sets the prevent destroy flag
func (tc *TerragruntConfig) SetPreventDestroy(prevent bool) {
	tc.PreventDestroy = prevent
}

// ShouldPreventDestroy returns whether destroy should be prevented
func (tc *TerragruntConfig) ShouldPreventDestroy() bool {
	return tc.PreventDestroy
}

// AddInput adds an input value
func (tc *TerragruntConfig) AddInput(key string, value interface{}) {
	if tc.Inputs == nil {
		tc.Inputs = make(map[string]interface{})
	}
	tc.Inputs[key] = value
}

// GetInput gets an input value
func (tc *TerragruntConfig) GetInput(key string) (interface{}, bool) {
	if tc.Inputs == nil {
		return nil, false
	}
	value, exists := tc.Inputs[key]
	return value, exists
}

// RemoveInput removes an input value
func (tc *TerragruntConfig) RemoveInput(key string) {
	if tc.Inputs != nil {
		delete(tc.Inputs, key)
	}
}

// AddLocal adds a local value
func (tc *TerragruntConfig) AddLocal(key string, value interface{}) {
	if tc.Locals == nil {
		tc.Locals = make(map[string]interface{})
	}
	tc.Locals[key] = value
}

// GetLocal gets a local value
func (tc *TerragruntConfig) GetLocal(key string) (interface{}, bool) {
	if tc.Locals == nil {
		return nil, false
	}
	value, exists := tc.Locals[key]
	return value, exists
}

// RemoveLocal removes a local value
func (tc *TerragruntConfig) RemoveLocal(key string) {
	if tc.Locals != nil {
		delete(tc.Locals, key)
	}
}

// SetRemoteState sets the remote state configuration
func (tc *TerragruntConfig) SetRemoteState(backend string, config map[string]interface{}) {
	tc.RemoteState = &RemoteStateConfig{
		Backend: backend,
		Config:  config,
	}
}

// GetRemoteStateBackend returns the remote state backend
func (tc *TerragruntConfig) GetRemoteStateBackend() string {
	if tc.RemoteState != nil {
		return tc.RemoteState.Backend
	}
	return ""
}

// GetRemoteStateConfig returns the remote state configuration
func (tc *TerragruntConfig) GetRemoteStateConfig() map[string]interface{} {
	if tc.RemoteState != nil {
		return tc.RemoteState.Config
	}
	return nil
}

// SetDependencies sets the dependencies configuration
func (tc *TerragruntConfig) SetDependencies(paths []string) {
	tc.Dependencies = &DependenciesConfig{
		Paths: paths,
	}
}

// AddDependency adds a dependency path
func (tc *TerragruntConfig) AddDependency(path string) {
	if tc.Dependencies == nil {
		tc.Dependencies = &DependenciesConfig{
			Paths: []string{},
		}
	}

	// Check if path already exists
	for _, existing := range tc.Dependencies.Paths {
		if existing == path {
			return
		}
	}

	tc.Dependencies.Paths = append(tc.Dependencies.Paths, path)
}

// RemoveDependency removes a dependency path
func (tc *TerragruntConfig) RemoveDependency(path string) {
	if tc.Dependencies == nil {
		return
	}

	var filtered []string
	for _, existing := range tc.Dependencies.Paths {
		if existing != path {
			filtered = append(filtered, existing)
		}
	}
	tc.Dependencies.Paths = filtered
}

// SetTerraformSource sets the Terraform source
func (tc *TerragruntConfig) SetTerraformSource(source string) {
	if tc.Terraform == nil {
		tc.Terraform = &TerraformConfig{}
	}
	tc.Terraform.Source = source
}

// GetTerraformSource returns the Terraform source
func (tc *TerragruntConfig) GetTerraformSource() string {
	if tc.Terraform != nil {
		return tc.Terraform.Source
	}
	return ""
}

// SetTerraformVersion sets the Terraform version
func (tc *TerragruntConfig) SetTerraformVersion(version string) {
	if tc.Terraform == nil {
		tc.Terraform = &TerraformConfig{}
	}
	tc.Terraform.Version = version
}

// GetTerraformVersion returns the Terraform version
func (tc *TerragruntConfig) GetTerraformVersion() string {
	if tc.Terraform != nil {
		return tc.Terraform.Version
	}
	return ""
}

// AddTerraformExtraArg adds an extra argument to Terraform
func (tc *TerragruntConfig) AddTerraformExtraArg(arg string) {
	if tc.Terraform == nil {
		tc.Terraform = &TerraformConfig{}
	}

	// Check if argument already exists
	for _, existing := range tc.Terraform.ExtraArgs {
		if existing == arg {
			return
		}
	}

	tc.Terraform.ExtraArgs = append(tc.Terraform.ExtraArgs, arg)
}

// RemoveTerraformExtraArg removes an extra argument from Terraform
func (tc *TerragruntConfig) RemoveTerraformExtraArg(arg string) {
	if tc.Terraform == nil {
		return
	}

	var filtered []string
	for _, existing := range tc.Terraform.ExtraArgs {
		if existing != arg {
			filtered = append(filtered, existing)
		}
	}
	tc.Terraform.ExtraArgs = filtered
}

// AddBeforeHook adds a before hook
func (tc *TerragruntConfig) AddBeforeHook(hook HookConfig) {
	if tc.Terraform == nil {
		tc.Terraform = &TerraformConfig{}
	}
	tc.Terraform.BeforeHooks = append(tc.Terraform.BeforeHooks, hook)
}

// AddAfterHook adds an after hook
func (tc *TerragruntConfig) AddAfterHook(hook HookConfig) {
	if tc.Terraform == nil {
		tc.Terraform = &TerraformConfig{}
	}
	tc.Terraform.AfterHooks = append(tc.Terraform.AfterHooks, hook)
}

// AddErrorHook adds an error hook
func (tc *TerragruntConfig) AddErrorHook(hook HookConfig) {
	if tc.Terraform == nil {
		tc.Terraform = &TerraformConfig{}
	}
	tc.Terraform.ErrorHooks = append(tc.Terraform.ErrorHooks, hook)
}

// SetErrorHandling sets the error handling configuration
func (tc *TerragruntConfig) SetErrorHandling(config *ErrorHandlingConfig) {
	tc.ErrorHandling = config
}

// GetErrorHandling returns the error handling configuration
func (tc *TerragruntConfig) GetErrorHandling() *ErrorHandlingConfig {
	return tc.ErrorHandling
}

// SetPerformance sets the performance configuration
func (tc *TerragruntConfig) SetPerformance(config *PerformanceConfig) {
	tc.Performance = config
}

// GetPerformance returns the performance configuration
func (tc *TerragruntConfig) GetPerformance() *PerformanceConfig {
	return tc.Performance
}

// Validate validates the Terragrunt configuration
func (tc *TerragruntConfig) Validate() error {
	// Validate include path if present
	if tc.IncludeConfig != nil && tc.IncludeConfig.Path == "" {
		return fmt.Errorf("include path cannot be empty")
	}

	// Validate Terraform source if present
	if tc.Terraform != nil && tc.Terraform.Source == "" {
		return fmt.Errorf("terraform source cannot be empty")
	}

	// Validate remote state configuration if present
	if tc.RemoteState != nil {
		if tc.RemoteState.Backend == "" {
			return fmt.Errorf("remote state backend cannot be empty")
		}
		if tc.RemoteState.Config == nil {
			return fmt.Errorf("remote state config cannot be nil")
		}
	}

	// Validate error handling configuration if present
	if tc.ErrorHandling != nil {
		if tc.ErrorHandling.MaxRetries < 0 {
			return fmt.Errorf("max retries cannot be negative")
		}
	}

	// Validate performance configuration if present
	if tc.Performance != nil {
		if tc.Performance.Parallelism < 0 {
			return fmt.Errorf("parallelism cannot be negative")
		}
	}

	return nil
}

// ToJSON converts the configuration to JSON
func (tc *TerragruntConfig) ToJSON() ([]byte, error) {
	return json.MarshalIndent(tc, "", "  ")
}

// FromJSON creates a configuration from JSON
func FromJSON(data []byte) (*TerragruntConfig, error) {
	var config TerragruntConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Terragrunt config: %w", err)
	}

	// Initialize maps if they are nil
	if config.Inputs == nil {
		config.Inputs = make(map[string]interface{})
	}
	if config.Locals == nil {
		config.Locals = make(map[string]interface{})
	}
	if config.IamAssumeRoleTags == nil {
		config.IamAssumeRoleTags = make(map[string]string)
	}

	return &config, nil
}

// Clone creates a deep copy of the configuration
func (tc *TerragruntConfig) Clone() *TerragruntConfig {
	// This is a simplified clone - in production, you'd want a proper deep copy
	jsonData, err := tc.ToJSON()
	if err != nil {
		return NewTerragruntConfig()
	}

	clone, err := FromJSON(jsonData)
	if err != nil {
		return NewTerragruntConfig()
	}

	return clone
}
