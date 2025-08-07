package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfigView represents the modern configuration view
type ConfigView struct {
	// State management
	state  ConfigState
	width  int
	height int

	// UI components
	inputs     []textinput.Model
	focusIndex int

	// Configuration data
	config Configuration
	saved  bool
	err    error
}

type ConfigState int

const (
	ConfigStateEditing ConfigState = iota
	ConfigStateSaving
	ConfigStateSaved
	ConfigStateError
)

type Configuration struct {
	// Cloud provider credentials
	AWSProfile        string
	AWSRegion         string
	AzureSubscription string
	AzureTenant       string
	GCPProject        string
	GCPCredentials    string

	// Import settings
	OutputDirectory string
	TerraformBinary string
	Parallelism     string
	DryRun          bool

	// Advanced settings
	LogLevel   string
	MaxRetries string
	Timeout    string
}

type ConfigSaveCompleteMsg struct{}
type ConfigSaveErrorMsg struct{ Err error }

// NewConfigView creates a new modern config view
func NewConfigView() *ConfigView {
	// Initialize all text inputs
	inputs := make([]textinput.Model, 10)

	// AWS Configuration
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "default"
	inputs[0].Width = 30
	inputs[0].Focus()

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "us-east-1"
	inputs[1].Width = 30

	// Azure Configuration
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "subscription-id"
	inputs[2].Width = 40

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "tenant-id"
	inputs[3].Width = 40

	// GCP Configuration
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "project-id"
	inputs[4].Width = 30

	inputs[5] = textinput.New()
	inputs[5].Placeholder = "/path/to/credentials.json"
	inputs[5].Width = 50

	// Import Settings
	inputs[6] = textinput.New()
	inputs[6].Placeholder = "./terraform"
	inputs[6].Width = 40

	inputs[7] = textinput.New()
	inputs[7].Placeholder = "terraform"
	inputs[7].Width = 30

	inputs[8] = textinput.New()
	inputs[8].Placeholder = "5"
	inputs[8].Width = 10

	// Advanced Settings
	inputs[9] = textinput.New()
	inputs[9].Placeholder = "info"
	inputs[9].Width = 20

	return &ConfigView{
		state:      ConfigStateEditing,
		inputs:     inputs,
		focusIndex: 0,
		config: Configuration{
			OutputDirectory: "./terraform",
			TerraformBinary: "terraform",
			Parallelism:     "5",
			LogLevel:        "info",
			MaxRetries:      "3",
			Timeout:         "30s",
		},
	}
}

func (v ConfigView) Init() tea.Cmd {
	return textinput.Blink
}

func (v *ConfigView) Update(msg tea.Msg) (*ConfigView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case tea.KeyMsg:
		switch v.state {
		case ConfigStateEditing:
			return v.handleEditingInput(msg)
		case ConfigStateSaved, ConfigStateError:
			return v.handleResultInput(msg)
		}

	case ConfigSaveCompleteMsg:
		v.state = ConfigStateSaved
		v.saved = true
		return v, nil

	case ConfigSaveErrorMsg:
		v.err = msg.Err
		v.state = ConfigStateError
		return v, nil
	}

	// Update focused input
	var cmd tea.Cmd
	if v.state == ConfigStateEditing && v.focusIndex < len(v.inputs) {
		v.inputs[v.focusIndex], cmd = v.inputs[v.focusIndex].Update(msg)
	}

	return v, cmd
}

func (v *ConfigView) handleEditingInput(msg tea.KeyMsg) (*ConfigView, tea.Cmd) {
	switch msg.String() {
	case "tab", "shift+tab", "enter", "up", "down":
		s := msg.String()

		// Handle navigation
		if s == "up" || s == "shift+tab" {
			v.focusIndex--
		} else {
			v.focusIndex++
		}

		if v.focusIndex > len(v.inputs)-1 {
			v.focusIndex = 0
		} else if v.focusIndex < 0 {
			v.focusIndex = len(v.inputs) - 1
		}

		// Update focus
		for i := 0; i < len(v.inputs); i++ {
			if i == v.focusIndex {
				v.inputs[i].Focus()
			} else {
				v.inputs[i].Blur()
			}
		}

		return v, v.inputs[v.focusIndex].Focus()

	case "ctrl+s":
		// Save configuration
		v.updateConfigFromInputs()
		v.state = ConfigStateSaving
		return v, v.saveConfig()

	case "ctrl+r":
		// Reset to defaults
		v.resetToDefaults()
		return v, nil

	case "esc":
		return v, tea.Cmd(func() tea.Msg { return ChangeScreen{Screen: ScreenMain} })
	}

	return v, nil
}

func (v *ConfigView) handleResultInput(msg tea.KeyMsg) (*ConfigView, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		return v, tea.Cmd(func() tea.Msg { return ChangeScreen{Screen: ScreenMain} })
	case "e":
		// Edit again
		v.state = ConfigStateEditing
		v.saved = false
		v.err = nil
		return v, nil
	}
	return v, nil
}

func (v ConfigView) saveConfig() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		// Simulate saving configuration
		// TODO: Implement actual configuration persistence
		return ConfigSaveCompleteMsg{}
	})
}

func (v *ConfigView) updateConfigFromInputs() {
	v.config.AWSProfile = v.inputs[0].Value()
	v.config.AWSRegion = v.inputs[1].Value()
	v.config.AzureSubscription = v.inputs[2].Value()
	v.config.AzureTenant = v.inputs[3].Value()
	v.config.GCPProject = v.inputs[4].Value()
	v.config.GCPCredentials = v.inputs[5].Value()
	v.config.OutputDirectory = v.inputs[6].Value()
	v.config.TerraformBinary = v.inputs[7].Value()
	v.config.Parallelism = v.inputs[8].Value()
	v.config.LogLevel = v.inputs[9].Value()
}

func (v *ConfigView) resetToDefaults() {
	defaults := []string{
		"default",     // AWS Profile
		"us-east-1",   // AWS Region
		"",            // Azure Subscription
		"",            // Azure Tenant
		"",            // GCP Project
		"",            // GCP Credentials
		"./terraform", // Output Directory
		"terraform",   // Terraform Binary
		"5",           // Parallelism
		"info",        // Log Level
	}

	for i, defaultVal := range defaults {
		if i < len(v.inputs) {
			v.inputs[i].SetValue(defaultVal)
		}
	}
}

func (v ConfigView) View() string {
	switch v.state {
	case ConfigStateEditing:
		return v.renderEditing()
	case ConfigStateSaving:
		return v.renderSaving()
	case ConfigStateSaved:
		return v.renderSaved()
	case ConfigStateError:
		return v.renderError()
	}
	return ""
}

func (v ConfigView) renderEditing() string {
	title := RenderTitle("Configuration")

	// Create sections
	awsSection := v.renderSection("AWS Configuration", []configField{
		{"Profile", v.inputs[0], "AWS profile name"},
		{"Region", v.inputs[1], "Default AWS region"},
	})

	azureSection := v.renderSection("Azure Configuration", []configField{
		{"Subscription", v.inputs[2], "Azure subscription ID"},
		{"Tenant", v.inputs[3], "Azure tenant ID"},
	})

	gcpSection := v.renderSection("Google Cloud Configuration", []configField{
		{"Project", v.inputs[4], "GCP project ID"},
		{"Credentials", v.inputs[5], "Path to service account JSON"},
	})

	importSection := v.renderSection("Import Settings", []configField{
		{"Output Directory", v.inputs[6], "Where to generate Terraform files"},
		{"Terraform Binary", v.inputs[7], "Path to terraform executable"},
		{"Parallelism", v.inputs[8], "Number of parallel operations"},
	})

	advancedSection := v.renderSection("Advanced Settings", []configField{
		{"Log Level", v.inputs[9], "Logging verbosity (debug, info, warn, error)"},
	})

	controls := lipgloss.NewStyle().
		Foreground(SecondaryTextColor).
		Render("Navigation: ↑/↓ Tab/Shift+Tab • Save: Ctrl+S • Reset: Ctrl+R • Back: Esc")

	sections := []string{
		title,
		"",
		awsSection,
		"",
		azureSection,
		"",
		gcpSection,
		"",
		importSection,
		"",
		advancedSection,
		"",
		controls,
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

type configField struct {
	Label       string
	Input       textinput.Model
	Description string
}

func (v ConfigView) renderSection(title string, fields []configField) string {
	sectionStyle := lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Width(18).
		Align(lipgloss.Right).
		Foreground(PrimaryTextColor)

	descStyle := lipgloss.NewStyle().
		Foreground(SecondaryTextColor).
		Italic(true)

	content := []string{sectionStyle.Render(title)}

	for _, field := range fields {
		fieldLine := lipgloss.JoinHorizontal(
			lipgloss.Top,
			labelStyle.Render(field.Label+":"),
			" ",
			field.Input.View(),
		)
		content = append(content, fieldLine)

		if field.Description != "" {
			descLine := strings.Repeat(" ", 20) + descStyle.Render("# "+field.Description)
			content = append(content, descLine)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func (v ConfigView) renderSaving() string {
	title := RenderTitle("Saving Configuration")

	progressStyle := lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true)

	content := []string{
		progressStyle.Render("⟳ Saving configuration..."),
		"",
		lipgloss.NewStyle().
			Foreground(SecondaryTextColor).
			Render("Please wait..."),
	}

	return lipgloss.JoinVertical(lipgloss.Left, append([]string{title}, content...)...)
}

func (v ConfigView) renderSaved() string {
	title := RenderTitle("Configuration Saved")

	successStyle := lipgloss.NewStyle().
		Foreground(SuccessColor).
		Bold(true)

	content := []string{
		successStyle.Render("✓ Configuration saved successfully!"),
		"",
		"Configuration has been written to:",
		"  ~/.config/driftmgr/config.yaml",
		"",
		lipgloss.NewStyle().
			Foreground(SecondaryTextColor).
			Render("Enter: Return to main menu • E: Edit again"),
	}

	return lipgloss.JoinVertical(lipgloss.Left, append([]string{title}, content...)...)
}

func (v ConfigView) renderError() string {
	title := RenderTitle("Configuration Error")

	errorStyle := lipgloss.NewStyle().
		Foreground(ErrorColor).
		Bold(true)

	content := []string{
		errorStyle.Render("✗ Failed to save configuration"),
		"",
		fmt.Sprintf("Error: %s", v.err.Error()),
		"",
		lipgloss.NewStyle().
			Foreground(SecondaryTextColor).
			Render("Enter: Return to main menu • E: Edit again"),
	}

	return lipgloss.JoinVertical(lipgloss.Left, append([]string{title}, content...)...)
}

// Legacy ConfigModel for backward compatibility
type ConfigModel struct {
	width  int
	height int

	// State
	selectedSection int
	sections        []configSection
}

type configSection struct {
	title   string
	options []configOption
}

type configOption struct {
	key         string
	value       string
	description string
	editable    bool
}

// NewConfigModel creates a new legacy config model
func NewConfigModel() *ConfigModel {
	return &ConfigModel{
		sections: []configSection{
			{
				title: "General Settings",
				options: []configOption{
					{"default_provider", "aws", "Default cloud provider", true},
					{"default_region", "us-east-1", "Default region", true},
					{"parallel_imports", "5", "Number of parallel imports", true},
					{"retry_attempts", "3", "Number of retry attempts", true},
				},
			},
			{
				title: "AWS Configuration",
				options: []configOption{
					{"aws_profile", "default", "AWS profile to use", true},
					{"aws_assume_role", "", "IAM role to assume (optional)", true},
					{"aws_session_duration", "3600", "Session duration in seconds", true},
				},
			},
			{
				title: "Azure Configuration",
				options: []configOption{
					{"azure_subscription_id", "", "Azure subscription ID", true},
					{"azure_tenant_id", "", "Azure tenant ID", true},
					{"azure_client_id", "", "Azure client ID (optional)", true},
				},
			},
			{
				title: "GCP Configuration",
				options: []configOption{
					{"gcp_project_id", "", "GCP project ID", true},
					{"gcp_credentials_file", "", "Path to credentials file", true},
					{"gcp_region", "us-central1", "Default GCP region", true},
				},
			},
			{
				title: "Import Settings",
				options: []configOption{
					{"dry_run", "false", "Default to dry run mode", true},
					{"generate_config", "true", "Generate Terraform configs", true},
					{"validate_after_import", "true", "Validate state after import", true},
					{"backup_state", "true", "Backup state before import", true},
				},
			},
			{
				title: "UI Settings",
				options: []configOption{
					{"theme", "dark", "UI theme (dark/light)", true},
					{"show_progress", "true", "Show progress indicators", true},
					{"log_level", "info", "Log level (debug/info/warn/error)", true},
				},
			},
		},
	}
}

// Legacy Update method for ConfigModel
func (m ConfigModel) Update(msg tea.Msg) (ConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedSection > 0 {
				m.selectedSection--
			} else {
				m.selectedSection = len(m.sections) - 1
			}
		case "down", "j":
			if m.selectedSection < len(m.sections)-1 {
				m.selectedSection++
			} else {
				m.selectedSection = 0
			}
		case "enter", " ":
			// TODO: Implement config editing
			return m, nil
		case "s":
			// Save configuration
			return m, m.saveConfig()
		case "r":
			// Reset to defaults
			return m, m.resetConfig()
		case "t":
			// Test connection for current provider
			return m, m.testConnection()
		}
	}
	return m, nil
}

// saveConfig saves the current configuration
func (m ConfigModel) saveConfig() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement actual config saving
		return ConfigSavedMsg{}
	}
}

// resetConfig resets configuration to defaults
func (m ConfigModel) resetConfig() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement config reset
		return ConfigResetMsg{}
	}
}

// testConnection tests the connection for the current provider
func (m ConfigModel) testConnection() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement connection testing
		return ConnectionTestMsg{
			Success: true,
			Message: "Connection successful",
		}
	}
}

// Legacy View method for ConfigModel
func (m ConfigModel) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("🔧 Configuration")

	// Section list
	var sectionViews []string
	for i, section := range m.sections {
		sectionStyle := lipgloss.NewStyle().
			Padding(0, 2).
			Margin(0, 0, 1, 0)

		if i == m.selectedSection {
			sectionStyle = sectionStyle.
				Background(lipgloss.Color("236")).
				Bold(true)
		}

		// Section title
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))
		sectionTitle := titleStyle.Render(section.title)

		// Section options
		var optionViews []string
		for _, option := range section.options {
			valueStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

			if option.value == "" {
				valueStyle = valueStyle.
					Foreground(lipgloss.Color("241")).
					Italic(true)
				option.value = "<not set>"
			}

			optionView := fmt.Sprintf("  %s: %s",
				option.key,
				valueStyle.Render(option.value))

			if option.description != "" {
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("243")).
					Italic(true)
				optionView += fmt.Sprintf("\n    %s", descStyle.Render(option.description))
			}

			optionViews = append(optionViews, optionView)
		}

		sectionContent := lipgloss.JoinVertical(
			lipgloss.Left,
			sectionTitle,
			lipgloss.JoinVertical(lipgloss.Left, optionViews...),
		)

		sectionViews = append(sectionViews, sectionStyle.Render(sectionContent))
	}

	configContent := lipgloss.JoinVertical(lipgloss.Left, sectionViews...)

	// Controls
	controls := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Padding(1, 0, 0, 0).
		Render(`Controls:
  ↑/↓ - Navigate sections   Enter - Edit section   S - Save config
  R - Reset to defaults   T - Test connection`)

	return lipgloss.JoinVertical(lipgloss.Left, title, "", configContent, controls)
}

// ConfigSavedMsg is sent when configuration is saved
type ConfigSavedMsg struct{}

// ConfigResetMsg is sent when configuration is reset
type ConfigResetMsg struct{}

// ConnectionTestMsg is sent when connection test completes
type ConnectionTestMsg struct {
	Success bool
	Message string
}

// ConfigModel represents the configuration view
type ConfigModel struct {
	width  int
	height int

	// State
	selectedSection int
	sections        []configSection
}

type configSection struct {
	title   string
	options []configOption
}

type configOption struct {
	key         string
	value       string
	description string
	editable    bool
}

// NewConfigModel creates a new config model
func NewConfigModel() *ConfigModel {
	return &ConfigModel{
		sections: []configSection{
			{
				title: "General Settings",
				options: []configOption{
					{"default_provider", "aws", "Default cloud provider", true},
					{"default_region", "us-east-1", "Default region", true},
					{"parallel_imports", "5", "Number of parallel imports", true},
					{"retry_attempts", "3", "Number of retry attempts", true},
				},
			},
			{
				title: "AWS Configuration",
				options: []configOption{
					{"aws_profile", "default", "AWS profile to use", true},
					{"aws_assume_role", "", "IAM role to assume (optional)", true},
					{"aws_session_duration", "3600", "Session duration in seconds", true},
				},
			},
			{
				title: "Azure Configuration",
				options: []configOption{
					{"azure_subscription_id", "", "Azure subscription ID", true},
					{"azure_tenant_id", "", "Azure tenant ID", true},
					{"azure_client_id", "", "Azure client ID (optional)", true},
				},
			},
			{
				title: "GCP Configuration",
				options: []configOption{
					{"gcp_project_id", "", "GCP project ID", true},
					{"gcp_credentials_file", "", "Path to credentials file", true},
					{"gcp_region", "us-central1", "Default GCP region", true},
				},
			},
			{
				title: "Import Settings",
				options: []configOption{
					{"dry_run", "false", "Default to dry run mode", true},
					{"generate_config", "true", "Generate Terraform configs", true},
					{"validate_after_import", "true", "Validate state after import", true},
					{"backup_state", "true", "Backup state before import", true},
				},
			},
			{
				title: "UI Settings",
				options: []configOption{
					{"theme", "dark", "UI theme (dark/light)", true},
					{"show_progress", "true", "Show progress indicators", true},
					{"log_level", "info", "Log level (debug/info/warn/error)", true},
				},
			},
		},
	}
}

// Update implements tea.Model
func (m ConfigModel) Update(msg tea.Msg) (ConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedSection > 0 {
				m.selectedSection--
			} else {
				m.selectedSection = len(m.sections) - 1
			}
		case "down", "j":
			if m.selectedSection < len(m.sections)-1 {
				m.selectedSection++
			} else {
				m.selectedSection = 0
			}
		case "enter", " ":
			// TODO: Implement config editing
			return m, nil
		case "s":
			// Save configuration
			return m, m.saveConfig()
		case "r":
			// Reset to defaults
			return m, m.resetConfig()
		case "t":
			// Test connection for current provider
			return m, m.testConnection()
		}
	}
	return m, nil
}

// saveConfig saves the current configuration
func (m ConfigModel) saveConfig() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement actual config saving
		return ConfigSavedMsg{}
	}
}

// resetConfig resets configuration to defaults
func (m ConfigModel) resetConfig() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement config reset
		return ConfigResetMsg{}
	}
}

// testConnection tests the connection for the current provider
func (m ConfigModel) testConnection() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement connection testing
		return ConnectionTestMsg{
			Success: true,
			Message: "Connection successful",
		}
	}
}

// View implements tea.Model
func (m ConfigModel) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("🔧 Configuration")

	// Section list
	var sectionViews []string
	for i, section := range m.sections {
		sectionStyle := lipgloss.NewStyle().
			Padding(0, 2).
			Margin(0, 0, 1, 0)

		if i == m.selectedSection {
			sectionStyle = sectionStyle.
				Background(lipgloss.Color("236")).
				Bold(true)
		}

		// Section title
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))
		sectionTitle := titleStyle.Render(section.title)

		// Section options
		var optionViews []string
		for _, option := range section.options {
			valueStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

			if option.value == "" {
				valueStyle = valueStyle.
					Foreground(lipgloss.Color("241")).
					Italic(true)
				option.value = "<not set>"
			}

			optionView := fmt.Sprintf("  %s: %s",
				option.key,
				valueStyle.Render(option.value))

			if option.description != "" {
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("243")).
					Italic(true)
				optionView += fmt.Sprintf("\n    %s", descStyle.Render(option.description))
			}

			optionViews = append(optionViews, optionView)
		}

		sectionContent := lipgloss.JoinVertical(
			lipgloss.Left,
			sectionTitle,
			lipgloss.JoinVertical(lipgloss.Left, optionViews...),
		)

		sectionViews = append(sectionViews, sectionStyle.Render(sectionContent))
	}

	configContent := lipgloss.JoinVertical(lipgloss.Left, sectionViews...)

	// Controls
	controls := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Padding(1, 0, 0, 0).
		Render(`Controls:
  ↑/↓ - Navigate sections   Enter - Edit section   S - Save config
  R - Reset to defaults   T - Test connection`)

	return lipgloss.JoinVertical(lipgloss.Left, title, "", configContent, controls)
}

// ConfigSavedMsg is sent when configuration is saved
type ConfigSavedMsg struct{}

// ConfigResetMsg is sent when configuration is reset
type ConfigResetMsg struct{}

// ConnectionTestMsg is sent when connection test completes
type ConnectionTestMsg struct {
	Success bool
	Message string
}
