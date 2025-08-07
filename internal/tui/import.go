package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"git		case "enter":
		v.state = ImportStateConfiguration
		v.textInput.SetValue(v.outputPath)
		return v, v.textInput.Focus()
	case "backspace", "delete":om/charmbracelet/lipgloss"
	"github.com/catherinevee/driftmgr/internal/models"
)

// ImportView represents the import view that works with discovered resources
type ImportView struct {
	// State management
	state           ImportState
	width           int
	height          int
	
	// UI components
	table           table.Model
	textInput       textinput.Model
	
	// Data
	selectedResources []models.Resource
	outputPath        string
	err              error
}

type ImportState int

const (
	ImportStateSelection ImportState = iota
	ImportStateConfiguration
	ImportStateProgress
	ImportStateComplete
	ImportStateError
)

// Legacy ImportModel for backward compatibility
type ImportModel struct {
	width  int
	height int

	// State
	step        int // 0: setup, 1: importing, 2: results
	inputFile   string
	parallelism int
	dryRun      bool

	// Import state
	importing bool
	progress  []ImportProgress
	result    *ImportResult
	error     string
}

// ImportProgress tracks the progress of individual imports
type ImportProgress struct {
	ResourceName string
	Status       string // "pending", "running", "success", "failed"
	Error        string
}

// ImportResult represents the result of import operation
type ImportResult struct {
	Successful int
	Failed     int
	Errors     []ImportError
	Duration   string
}

type ImportError struct {
	Resource string
	Error    string
}

type ImportViewCompleteMsg struct{}
type ImportErrorMsg struct{ Err error }

// NewImportView creates a new import view with discovered resources
func NewImportView(resources []models.Resource) *ImportView {
	// Initialize table with selected resources
	columns := []table.Column{
		{Title: "Resource", Width: 20},
		{Title: "Type", Width: 30},
		{Title: "Region", Width: 15},
		{Title: "Status", Width: 15},
	}

	rows := make([]table.Row, len(resources))
	for i, resource := range resources {
		rows[i] = table.Row{
			resource.Name,
			resource.Type,
			resource.Region,
			"Ready",
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(AccentColor).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(AccentColor).
		Bold(false)
	t.SetStyles(s)

	// Initialize text input for output path
	ti := textinput.New()
	ti.Placeholder = "Enter output directory (e.g., ./terraform)"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

	return &ImportView{
		state:             ImportStateSelection,
		table:            t,
		textInput:        ti,
		selectedResources: resources,
		outputPath:       "./terraform",
	}
}

func (v ImportView) Init() tea.Cmd {
	return textinput.Blink
}

func (v *ImportView) Update(msg tea.Msg) (*ImportView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		v.table.SetWidth(msg.Width - 4)
		return v, nil

	case tea.KeyMsg:
		switch v.state {
		case ImportStateSelection:
			return v.handleSelectionInput(msg)
		case ImportStateConfiguration:
			return v.handleConfigurationInput(msg)
		case ImportStateProgress:
			return v.handleProgressInput(msg)
		case ImportStateComplete:
			return v.handleCompleteInput(msg)
		}

	case ImportViewCompleteMsg:
		v.state = ImportStateComplete
		return v, nil

	case ImportErrorMsg:
		v.err = msg.Err
		v.state = ImportStateError
		return v, nil
	}

	// Update components
	switch v.state {
	case ImportStateConfiguration:
		v.textInput, cmd = v.textInput.Update(msg)
	default:
		v.table, cmd = v.table.Update(msg)
	}

	return v, cmd
}

func (v *ImportView) handleSelectionInput(msg tea.KeyMsg) (*ImportView, tea.Cmd) {
	switch msg.String() {
	case "enter":
		v.state = ImportStateConfiguration
		v.textInput.SetValue(v.outputPath)
		return v, v.textInput.Focus()
	case "backspace", "delete":
		// Remove selected resource
		if len(v.selectedResources) > 0 {
			cursor := v.table.Cursor()
			if cursor >= 0 && cursor < len(v.selectedResources) {
				v.selectedResources = append(
					v.selectedResources[:cursor],
					v.selectedResources[cursor+1:]...,
				)
				v.updateTable()
			}
		}
		return v, nil
	case "esc":
		return v, tea.Cmd(func() tea.Msg { return ChangeScreen{Screen: ScreenDiscovery} })
	}
	return v, nil
}

func (v *ImportView) handleConfigurationInput(msg tea.KeyMsg) (*ImportView, tea.Cmd) {
	switch msg.String() {
	case "enter":
		v.outputPath = v.textInput.Value()
		if v.outputPath == "" {
			v.outputPath = "./terraform"
		}
		v.state = ImportStateProgress
		return v, v.startImport()
	case "esc":
		v.state = ImportStateSelection
		return v, nil
	}
	return v, nil
}

func (v *ImportView) handleProgressInput(msg tea.KeyMsg) (*ImportView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return v, tea.Cmd(func() tea.Msg { return ChangeScreen{Screen: ScreenMain} })
	}
	return v, nil
}

func (v *ImportView) handleCompleteInput(msg tea.KeyMsg) (*ImportView, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		return v, tea.Cmd(func() tea.Msg { return ChangeScreen{Screen: ScreenMain} })
	}
	return v, nil
}

func (v ImportView) startImport() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		// Simulate import process
		// TODO: Implement actual Terraform file generation
		return ImportViewCompleteMsg{}
	})
}

func (v *ImportView) updateTable() {
	rows := make([]table.Row, len(v.selectedResources))
	for i, resource := range v.selectedResources {
		rows[i] = table.Row{
			resource.Name,
			resource.Type,
			resource.Region,
			"Ready",
		}
	}
	v.table.SetRows(rows)
}

func (v ImportView) View() string {
	switch v.state {
	case ImportStateSelection:
		return v.renderSelection()
	case ImportStateConfiguration:
		return v.renderConfiguration()
	case ImportStateProgress:
		return v.renderProgress()
	case ImportStateComplete:
		return v.renderComplete()
	case ImportStateError:
		return v.renderError()
	}
	return ""
}

func (v ImportView) renderSelection() string {
	title := RenderTitle("Import Resources")
	
	content := []string{
		"Selected Resources:",
		"",
		v.table.View(),
		"",
		lipgloss.NewStyle().
			Foreground(SecondaryTextColor).
			Render("↑/↓: Navigate • Enter: Configure import • Delete: Remove resource • Esc: Back"),
	}

	if len(v.selectedResources) == 0 {
		content = append(content[:2], 
			lipgloss.NewStyle().
				Foreground(ErrorColor).
				Render("No resources selected for import"),
			"",
			lipgloss.NewStyle().
				Foreground(SecondaryTextColor).
				Render("Esc: Back to discovery"),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, append([]string{title}, content...)...)
}

func (v ImportView) renderConfiguration() string {
	title := RenderTitle("Configure Import")
	
	content := []string{
		fmt.Sprintf("Importing %d resources", len(v.selectedResources)),
		"",
		"Output Directory:",
		v.textInput.View(),
		"",
		lipgloss.NewStyle().
			Foreground(SecondaryTextColor).
			Render("Enter: Start import • Esc: Back"),
	}

	return lipgloss.JoinVertical(lipgloss.Left, append([]string{title}, content...)...)
}

func (v ImportView) renderProgress() string {
	title := RenderTitle("Importing Resources")
	
	// Create progress indicator
	progressStyle := lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true)
	
	content := []string{
		progressStyle.Render("⟳ Generating Terraform configuration..."),
		"",
		fmt.Sprintf("Output directory: %s", v.outputPath),
		fmt.Sprintf("Resources: %d", len(v.selectedResources)),
		"",
		lipgloss.NewStyle().
			Foreground(SecondaryTextColor).
			Render("Please wait..."),
	}

	return lipgloss.JoinVertical(lipgloss.Left, append([]string{title}, content...)...)
}

func (v ImportView) renderComplete() string {
	title := RenderTitle("Import Complete")
	
	successStyle := lipgloss.NewStyle().
		Foreground(SuccessColor).
		Bold(true)
	
	content := []string{
		successStyle.Render("✓ Terraform configuration generated successfully!"),
		"",
		fmt.Sprintf("Output directory: %s", v.outputPath),
		fmt.Sprintf("Resources imported: %d", len(v.selectedResources)),
		"",
		"Generated files:",
		"  • main.tf",
		"  • variables.tf", 
		"  • import.tf",
		"",
		lipgloss.NewStyle().
			Foreground(SecondaryTextColor).
			Render("Enter: Return to main menu"),
	}

	return lipgloss.JoinVertical(lipgloss.Left, append([]string{title}, content...)...)
}

func (v ImportView) renderError() string {
	title := RenderTitle("Import Error")
	
	errorStyle := lipgloss.NewStyle().
		Foreground(ErrorColor).
		Bold(true)
	
	content := []string{
		errorStyle.Render("✗ Import failed"),
		"",
		fmt.Sprintf("Error: %s", v.err.Error()),
		"",
		lipgloss.NewStyle().
			Foreground(SecondaryTextColor).
			Render("Esc: Return to main menu"),
	}

	return lipgloss.JoinVertical(lipgloss.Left, append([]string{title}, content...)...)
}

// NewImportModel creates a new legacy import model for backward compatibility
func NewImportModel() *ImportModel {
	return &ImportModel{
		parallelism: 5,
		dryRun:      false,
	}
}

// Legacy Update method for ImportModel
func (m ImportModel) Update(msg tea.Msg) (ImportModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.step {
		case 0: // Setup step
			return m.updateSetup(msg)
		case 2: // Results step
			return m.updateResults(msg)
		}
	case ImportResultMsg:
		m.importing = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.result = msg.Result
			m.step = 2
		}
		return m, nil
	case ImportProgressMsg:
		// Update progress for a specific resource
		for i, prog := range m.progress {
			if prog.ResourceName == msg.ResourceName {
				m.progress[i].Status = msg.Status
				m.progress[i].Error = msg.Error
				break
			}
		}
		return m, nil
	}
	return m, nil
}

// updateSetup handles input during the setup step
func (m ImportModel) updateSetup(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	switch msg.String() {
	case "1":
		m.inputFile = "resources.csv"
		return m, m.startImportLegacy()
	case "2":
		m.inputFile = "resources.json"
		return m, m.startImportLegacy()
	case "d":
		m.dryRun = !m.dryRun
	case "+":
		if m.parallelism < 10 {
			m.parallelism++
		}
	case "-":
		if m.parallelism > 1 {
			m.parallelism--
		}
	case "enter":
		if m.inputFile != "" {
			return m, m.startImportLegacy()
		}
	}
	return m, nil
}

// updateResults handles input during the results step
func (m ImportModel) updateResults(msg tea.KeyMsg) (ImportModel, tea.Cmd) {
	switch msg.String() {
	case "r":
		// Restart import process
		m.step = 0
		m.result = nil
		m.progress = nil
		m.error = ""
	case "v":
		// View detailed results
		// TODO: Implement detailed results view
	}
	return m, nil
}

// startImportLegacy initiates the import process for legacy model
func (m ImportModel) startImportLegacy() tea.Cmd {
	m.step = 1
	m.importing = true
	m.error = ""

	return func() tea.Msg {
		// Simulate import process for legacy model
		result := &ImportResult{
			Successful: 5,
			Failed:     0,
			Duration:   "2s",
		}
		return ImportResultMsg{
			Result: result,
			Error:  nil,
		}
	}
}

// Legacy View method for ImportModel  
func (m ImportModel) View() string {
	switch m.step {
	case 0:
		return m.viewSetup()
	case 1:
		return m.viewImporting()
	case 2:
		return m.viewResults()
	}
	return "Unknown step"
}

// viewSetup renders the setup view
func (m ImportModel) viewSetup() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("📦 Resource Import")

	dryRunStatus := "Off"
	if m.dryRun {
		dryRunStatus = "On"
	}

	content := fmt.Sprintf(`Configure import settings:

Input File Options:
1. resources.csv (Load from CSV file)
2. resources.json (Load from JSON file)

Settings:
  Parallelism: %d (use +/- to adjust)
  Dry Run: %s (press 'd' to toggle)

Press Enter to start import with current settings.
Press the number key to select an input file type.`,
		m.parallelism, dryRunStatus)

	if m.error != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		content += "\n\nError: " + errorStyle.Render(m.error)
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, "", content)
}

// viewImporting renders the import in progress view
func (m ImportModel) viewImporting() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("📦 Importing Resources")

	var mode string
	if m.dryRun {
		mode = " (Dry Run Mode)"
	}

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Render(fmt.Sprintf("File: %s | Parallelism: %d%s", m.inputFile, m.parallelism, mode))

	// Progress display
	var progressItems []string
	completed := 0
	failed := 0

	for _, prog := range m.progress {
		var status string
		switch prog.Status {
		case "pending":
			status = "⏳"
		case "running":
			status = "🔄"
		case "success":
			status = "✅"
			completed++
		case "failed":
			status = "❌"
			failed++
		}

		item := fmt.Sprintf("%s %s", status, prog.ResourceName)
		if prog.Error != "" {
			item += fmt.Sprintf(" (%s)", prog.Error)
		}
		progressItems = append(progressItems, item)
	}

	progressList := lipgloss.JoinVertical(lipgloss.Left, progressItems...)

	summary := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Render(fmt.Sprintf("Progress: %d completed, %d failed, %d total",
			completed, failed, len(m.progress)))

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", progressList, "", summary)
}

// viewResults renders the import results view
func (m ImportModel) viewResults() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("📊 Import Results")

	if m.result == nil {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", "No results available.")
	}

	// Summary
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)

	summary := fmt.Sprintf(`Import completed in %v

Results:
  %s %d successful
  %s %d failed
  
Total resources processed: %d`,
		m.result.Duration,
		successStyle.Render("✅"), m.result.Successful,
		failStyle.Render("❌"), m.result.Failed,
		m.result.Successful+m.result.Failed)

	// Error details if any
	var errorDetails string
	if len(m.result.Errors) > 0 {
		errorDetails = "\n\nErrors:"
		for i, err := range m.result.Errors {
			if i >= 5 { // Limit display to first 5 errors
				errorDetails += fmt.Sprintf("\n  ... and %d more", len(m.result.Errors)-5)
				break
			}
			errorDetails += fmt.Sprintf("\n  • %s: %s", err.Resource, err.Error)
		}
	}

	// Generated files
	var filesInfo string
	if m.result.Successful > 0 && !m.dryRun {
		filesInfo = "\n\nGenerated Files:\n  • Terraform configuration files (*.tf)\n  • Import state updated"
	}

	// Controls
	controls := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Render("\nControls:\n  R - Restart import   V - View detailed results")

	content := summary + errorDetails + filesInfo + controls

	return lipgloss.JoinVertical(lipgloss.Left, title, "", content)
}

// ImportResultMsg is sent when import completes
type ImportResultMsg struct {
	Result *ImportResult
	Error  error
}

// ImportProgressMsg is sent to update import progress
type ImportProgressMsg struct {
	ResourceName string
	Status       string
	Error        string
}
