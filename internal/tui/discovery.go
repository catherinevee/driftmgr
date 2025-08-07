package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/models"
)

package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/models"
)

// DiscoveryView represents the resource discovery view
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

// ... (rest of the code remains unchanged)
type DiscoveryModel struct {
	width  int
	height int

	// State
	step     int // 0: setup, 1: discovering, 2: results
	provider string
	regions  []string

	// Discovery results
	resources []models.Resource
	selected  map[string]bool
	cursor    int

	// UI state
	discovering bool
	error       string

	// Discovery engine
	engine *discovery.Engine
}

// NewDiscoveryModel creates a new discovery model
func NewDiscoveryModel() *DiscoveryModel {
	return &DiscoveryModel{
		engine:   discovery.NewEngine(),
		selected: make(map[string]bool),
	}
}

// Update implements tea.Model
func (m DiscoveryModel) Update(msg tea.Msg) (DiscoveryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.step {
		case 0: // Setup step
			return m.updateSetup(msg)
		case 2: // Results step
			return m.updateResults(msg)
		}
	case DiscoveryResultMsg:
		m.discovering = false
		if msg.Error != nil {
			m.error = msg.Error.Error()
		} else {
			m.resources = msg.Resources
			m.step = 2
		}
		return m, nil
	}

	return m, nil
}

// updateSetup handles input during the setup step
func (m DiscoveryModel) updateSetup(msg tea.KeyMsg) (DiscoveryModel, tea.Cmd) {
	switch msg.String() {
	case "1":
		m.provider = "aws"
		return m, m.startDiscovery()
	case "2":
		m.provider = "azure"
		return m, m.startDiscovery()
	case "3":
		m.provider = "gcp"
		return m, m.startDiscovery()
	}
	return m, nil
}

// updateResults handles input during the results step
func (m DiscoveryModel) updateResults(msg tea.KeyMsg) (DiscoveryModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.resources)-1 {
			m.cursor++
		}
	case " ":
		// Toggle selection
		if m.cursor < len(m.resources) {
			resource := m.resources[m.cursor]
			m.selected[resource.ID] = !m.selected[resource.ID]
		}
	case "a":
		// Select all
		for _, resource := range m.resources {
			m.selected[resource.ID] = true
		}
	case "n":
		// Select none
		m.selected = make(map[string]bool)
	case "enter":
		// Export selected resources for import
		return m, m.exportSelected()
	case "r":
		// Restart discovery
		m.step = 0
		m.resources = nil
		m.selected = make(map[string]bool)
		m.error = ""
	}
	return m, nil
}

// startDiscovery initiates resource discovery
func (m DiscoveryModel) startDiscovery() tea.Cmd {
	m.step = 1
	m.discovering = true
	m.error = ""

	return func() tea.Msg {
		config := discovery.Config{
			Provider: m.provider,
			Regions:  m.regions,
		}

		resources, err := m.engine.Discover(config)
		return DiscoveryResultMsg{
			Resources: resources,
			Error:     err,
		}
	}
}

// exportSelected exports selected resources for import
func (m DiscoveryModel) exportSelected() tea.Cmd {
	var selectedResources []models.Resource
	for _, resource := range m.resources {
		if m.selected[resource.ID] {
			selectedResources = append(selectedResources, resource)
		}
	}

	return func() tea.Msg {
		// TODO: Save selected resources to file for import
		return ExportCompleteMsg{
			Count: len(selectedResources),
		}
	}
}

// View implements tea.Model
func (m DiscoveryModel) View() string {
	switch m.step {
	case 0:
		return m.viewSetup()
	case 1:
		return m.viewDiscovering()
	case 2:
		return m.viewResults()
	}
	return "Unknown step"
}

// viewSetup renders the setup view
func (m DiscoveryModel) viewSetup() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render("🔍 Resource Discovery")

	content := `Select a cloud provider to discover resources:

1. AWS (Amazon Web Services)
2. Azure (Microsoft Azure)
3. GCP (Google Cloud Platform)

Press the number key to start discovery.`

	if m.error != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
		content += "\n\nError: " + errorStyle.Render(m.error)
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, "", content)
}

// viewDiscovering renders the discovery in progress view
func (m DiscoveryModel) viewDiscovering() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render(fmt.Sprintf("🔍 Discovering %s Resources", strings.ToUpper(m.provider)))

	spinner := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	// TODO: Add actual spinner animation

	content := fmt.Sprintf("%c Scanning for resources...\n\nThis may take a few moments depending on the number of resources in your account.", spinner[0])

	return lipgloss.JoinVertical(lipgloss.Left, title, "", content)
}

// viewResults renders the discovery results view
func (m DiscoveryModel) viewResults() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		Render(fmt.Sprintf("📋 Found %d %s Resources", len(m.resources), strings.ToUpper(m.provider)))

	if len(m.resources) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left, title, "", "No resources found. Press 'r' to try again.")
	}

	// Resource list
	var items []string
	selectedCount := 0

	for i, resource := range m.resources {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checkbox := "☐"
		if m.selected[resource.ID] {
			checkbox = "☑"
			selectedCount++
		}

		style := lipgloss.NewStyle()
		if m.cursor == i {
			style = style.Background(lipgloss.Color("236"))
		}

		item := fmt.Sprintf("%s %s %s (%s) [%s]",
			cursor, checkbox, resource.Name, resource.Type, resource.Region)
		items = append(items, style.Render(item))
	}

	// Display only a subset if there are too many
	maxDisplay := 10
	displayItems := items
	if len(items) > maxDisplay {
		start := m.cursor - maxDisplay/2
		if start < 0 {
			start = 0
		}
		end := start + maxDisplay
		if end > len(items) {
			end = len(items)
			start = end - maxDisplay
			if start < 0 {
				start = 0
			}
		}
		displayItems = items[start:end]
	}

	resourceList := lipgloss.JoinVertical(lipgloss.Left, displayItems...)

	// Controls
	controls := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Render(fmt.Sprintf(`Selected: %d/%d

Controls:
  ↑/↓ - Navigate   Space - Toggle   A - Select all   N - Select none
  Enter - Export selected   R - Restart discovery`, selectedCount, len(m.resources)))

	return lipgloss.JoinVertical(lipgloss.Left, title, "", resourceList, "", controls)
}

// DiscoveryResultMsg is sent when discovery completes
type DiscoveryResultMsg struct {
	Resources []models.Resource
	Error     error
}

// ExportCompleteMsg is sent when export completes
type ExportCompleteMsg struct {
	Count int
}
