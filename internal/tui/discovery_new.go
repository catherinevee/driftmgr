package tui

import (
	"fmt"
	"strings"

	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DiscoveryView represents the resource discovery screen
type DiscoveryView struct {
	engine           *discovery.Engine
	selectedProvider string
	selectedRegions  []string
	isDiscovering    bool
	resources        []models.Resource
	cursor           int
	selectedRes      map[int]bool
	state            DiscoveryState
}

type DiscoveryState int

const (
	ProviderSelection DiscoveryState = iota
	RegionSelection
	Discovering
	ResultsView
)

// NewDiscoveryView creates a new discovery view
func NewDiscoveryView(engine *discovery.Engine) *DiscoveryView {
	return &DiscoveryView{
		engine:      engine,
		selectedRes: make(map[int]bool),
		state:       ProviderSelection,
	}
}

// Update handles messages for the discovery view
func (d *DiscoveryView) Update(msg tea.Msg) (*DiscoveryView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch d.state {
		case ProviderSelection:
			return d.handleProviderSelection(msg)
		case RegionSelection:
			return d.handleRegionSelection(msg)
		case ResultsView:
			return d.handleResultsView(msg)
		}

	case ResourcesFoundMsg:
		d.resources = msg.Resources
		d.isDiscovering = false
		d.state = ResultsView
		return d, SendStatus(fmt.Sprintf("Found %d resources", len(msg.Resources)))
	}

	return d, nil
}

func (d *DiscoveryView) handleProviderSelection(msg tea.KeyMsg) (*DiscoveryView, tea.Cmd) {
	switch msg.String() {
	case "1":
		d.selectedProvider = "aws"
		d.state = RegionSelection
	case "2":
		d.selectedProvider = "azure"
		d.state = RegionSelection
	case "3":
		d.selectedProvider = "gcp"
		d.state = RegionSelection
	case "enter":
		if d.selectedProvider != "" {
			return d.startDiscovery()
		}
	}
	return d, nil
}

func (d *DiscoveryView) handleRegionSelection(msg tea.KeyMsg) (*DiscoveryView, tea.Cmd) {
	switch msg.String() {
	case "1":
		d.selectedRegions = []string{"us-east-1"}
	case "2":
		d.selectedRegions = []string{"us-west-2"}
	case "a":
		d.selectedRegions = []string{} // All regions
	case "enter":
		return d.startDiscovery()
	case "backspace":
		d.state = ProviderSelection
	}
	return d, nil
}

func (d *DiscoveryView) handleResultsView(msg tea.KeyMsg) (*DiscoveryView, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if d.cursor > 0 {
			d.cursor--
		}
	case "down", "j":
		if d.cursor < len(d.resources)-1 {
			d.cursor++
		}
	case " ":
		if d.cursor < len(d.resources) {
			d.selectedRes[d.cursor] = !d.selectedRes[d.cursor]
		}
	case "enter":
		// Export selected resources
		selected := d.getSelectedResources()
		if len(selected) > 0 {
			return d, tea.Cmd(func() tea.Msg { return ChangeScreen{Screen: ScreenImport} })
		}
	case "r":
		// Restart discovery
		d.state = ProviderSelection
		d.resources = nil
		d.selectedRes = make(map[int]bool)
	}
	return d, nil
}

func (d *DiscoveryView) startDiscovery() (*DiscoveryView, tea.Cmd) {
	d.isDiscovering = true
	d.state = Discovering

	return d, func() tea.Msg {
		config := discovery.Config{
			Provider: d.selectedProvider,
			Regions:  d.selectedRegions,
		}

		resources, err := d.engine.Discover(config)
		if err != nil {
			return StatusMsg{Message: fmt.Sprintf("Discovery failed: %v", err)}
		}

		return ResourcesFoundMsg{Resources: resources}
	}
}

func (d *DiscoveryView) getSelectedResources() []models.Resource {
	var selected []models.Resource
	for i, resource := range d.resources {
		if d.selectedRes[i] {
			selected = append(selected, resource)
		}
	}
	return selected
}

// View renders the discovery screen
func (d *DiscoveryView) View() string {
	switch d.state {
	case ProviderSelection:
		return d.renderProviderSelection()
	case RegionSelection:
		return d.renderRegionSelection()
	case Discovering:
		return d.renderDiscovering()
	case ResultsView:
		return d.renderResults()
	}
	return "Unknown state"
}

func (d *DiscoveryView) renderProviderSelection() string {
	title := RenderTitle("🔍 Resource Discovery - Provider Selection")

	content := []string{
		"Select a cloud provider to scan for resources:",
		"",
		RenderMenuItem("1. AWS (Amazon Web Services)", d.selectedProvider == "aws"),
		RenderMenuItem("2. Azure (Microsoft Azure)", d.selectedProvider == "azure"),
		RenderMenuItem("3. GCP (Google Cloud Platform)", d.selectedProvider == "gcp"),
		"",
		RenderHelp("Press number to select provider, Enter to continue"),
	}

	return contentStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			strings.Join(content, "\n"),
		),
	)
}

func (d *DiscoveryView) renderRegionSelection() string {
	title := RenderTitle(fmt.Sprintf("🌍 Region Selection - %s", strings.ToUpper(d.selectedProvider)))

	var regions []string
	switch d.selectedProvider {
	case "aws":
		regions = []string{"1. US East 1 (us-east-1)", "2. US West 2 (us-west-2)", "a. All regions"}
	case "azure":
		regions = []string{"1. East US (eastus)", "2. West US 2 (westus2)", "a. All regions"}
	case "gcp":
		regions = []string{"1. US Central 1 (us-central1)", "2. US East 1 (us-east1)", "a. All regions"}
	}

	content := []string{
		"Select regions to scan:",
		"",
	}
	content = append(content, regions...)
	content = append(content, "", RenderHelp("Press number/letter to select, Enter to start discovery, Backspace to go back"))

	return contentStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			strings.Join(content, "\n"),
		),
	)
}

func (d *DiscoveryView) renderDiscovering() string {
	title := RenderTitle("🔍 Discovering Resources...")

	content := []string{
		fmt.Sprintf("Scanning %s resources...", strings.ToUpper(d.selectedProvider)),
		"",
		"⏳ Please wait while we discover your cloud resources.",
		"This may take a few moments depending on the number of resources.",
		"",
		RenderProgressBar(1, 3, 40), // Simulated progress
	}

	return contentStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			strings.Join(content, "\n"),
		),
	)
}

func (d *DiscoveryView) renderResults() string {
	title := RenderTitle(fmt.Sprintf("📊 Discovery Results - %d Resources Found", len(d.resources)))

	if len(d.resources) == 0 {
		content := []string{
			"No resources found.",
			"",
			RenderHelp("Press 'r' to restart discovery, Esc to return to main menu"),
		}
		return contentStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				title,
				strings.Join(content, "\n"),
			),
		)
	}

	// Table header
	header := lipgloss.JoinHorizontal(lipgloss.Left,
		tableHeaderStyle.Width(3).Render(""),
		tableHeaderStyle.Width(25).Render("NAME"),
		tableHeaderStyle.Width(20).Render("TYPE"),
		tableHeaderStyle.Width(10).Render("PROVIDER"),
		tableHeaderStyle.Width(15).Render("REGION"),
	)

	// Table rows
	var rows []string
	start := 0
	end := len(d.resources)
	if end > 10 { // Show only 10 items at a time
		start = d.cursor
		if start > len(d.resources)-10 {
			start = len(d.resources) - 10
		}
		end = start + 10
	}

	for i := start; i < end && i < len(d.resources); i++ {
		resource := d.resources[i]

		selected := ""
		if d.selectedRes[i] {
			selected = "✓"
		}

		style := tableRowStyle
		if i == d.cursor {
			style = tableSelectedRowStyle
		}

		row := lipgloss.JoinHorizontal(lipgloss.Left,
			style.Width(3).Render(selected),
			style.Width(25).Render(truncate(resource.Name, 23)),
			style.Width(20).Render(truncate(resource.Type, 18)),
			style.Width(10).Render(resource.Provider),
			style.Width(15).Render(resource.Region),
		)
		rows = append(rows, row)
	}

	table := lipgloss.JoinVertical(lipgloss.Left, append([]string{header}, rows...)...)

	// Instructions
	instructions := RenderHelp("↑/↓: navigate, Space: select, Enter: import selected, 'r': restart")

	selectedCount := len(d.getSelectedResources())
	status := ""
	if selectedCount > 0 {
		status = RenderSuccess(fmt.Sprintf("%d resources selected for import", selectedCount))
	}

	return contentStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			table,
			status,
			instructions,
		),
	)
}

// Helper function to truncate strings
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
