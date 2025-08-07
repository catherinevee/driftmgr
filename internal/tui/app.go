package tui

import (
	"fmt"

	"github.com/catherinevee/driftmgr/internal/discovery"
	"github.com/catherinevee/driftmgr/internal/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Screen represents different screens in the TUI
type Screen int

const (
	ScreenMain Screen = iota
	ScreenDiscovery
	ScreenImport
	ScreenConfig
	ScreenHelp
)

// App represents the main TUI application
type App struct {
	currentScreen   Screen
	width           int
	height          int
	mainMenu        *MainMenu
	discoveryView   *DiscoveryView
	importView      *ImportView
	configView      *ConfigView
	helpView        *HelpView
	statusMessage   string
	discoveryEngine *discovery.Engine
}

// NewApp creates a new TUI application
func NewApp() (*App, error) {
	discoveryEngine, err := discovery.NewEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize discovery engine: %w", err)
	}

	app := &App{
		currentScreen:   ScreenMain,
		discoveryEngine: discoveryEngine,
	}

	// Initialize all views
	app.mainMenu = NewMainMenu()
	app.discoveryView = NewDiscoveryView(discoveryEngine)
	app.importView = NewImportView([]models.Resource{}) // Initialize with empty resources
	app.configView = NewConfigView()
	app.helpView = NewHelpView()

	return app, nil
}

// Init implements the tea.Model interface
func (a *App) Init() tea.Cmd {
	return tea.EnterAltScreen
}

// Update implements the tea.Model interface
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit

		case "esc":
			// Return to main menu from any screen
			a.currentScreen = ScreenMain
			a.statusMessage = ""
			return a, nil
		}

	case ChangeScreen:
		a.currentScreen = msg.Screen
		a.statusMessage = ""
		return a, nil

	case StatusMsg:
		a.statusMessage = msg.Message
		return a, nil
	}

	// Forward messages to the appropriate view
	var cmd tea.Cmd
	switch a.currentScreen {
	case ScreenMain:
		a.mainMenu, cmd = a.mainMenu.Update(msg)
	case ScreenDiscovery:
		a.discoveryView, cmd = a.discoveryView.Update(msg)
	case ScreenImport:
		a.importView, cmd = a.importView.Update(msg)
	case ScreenConfig:
		a.configView, cmd = a.configView.Update(msg)
	case ScreenHelp:
		a.helpView, cmd = a.helpView.Update(msg)
	}

	return a, cmd
}

// View implements the tea.Model interface
func (a *App) View() string {
	// Header
	header := RenderTitle("🚀 Terraform Import Helper v2.0")

	// Content based on current screen
	var content string
	switch a.currentScreen {
	case ScreenMain:
		content = a.mainMenu.View()
	case ScreenDiscovery:
		content = a.discoveryView.View()
	case ScreenImport:
		content = a.importView.View()
	case ScreenConfig:
		content = a.configView.View()
	case ScreenHelp:
		content = a.helpView.View()
	default:
		content = "Unknown screen"
	}

	// Status bar
	statusBar := ""
	if a.statusMessage != "" {
		statusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Render("Status: " + a.statusMessage)
	}

	// Help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true).
		Render("Press 'q' to quit, 'esc' to return to main menu")

	// Combine all parts
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		statusBar,
		helpText,
	)

	// Ensure the view fits the terminal
	if a.height > 0 {
		availableHeight := a.height - 2 // Leave space for status and help
		if lipgloss.Height(view) > availableHeight {
			view = lipgloss.NewStyle().Height(availableHeight).Render(view)
		}
	}

	return view
}

// Run starts the TUI application
func (a *App) Run() error {
	// For backward compatibility, provide a simple fallback
	if a.mainMenu.choices == nil {
		fmt.Println("🚀 Interactive TUI Mode")
		fmt.Println()
		fmt.Println("📋 Main Menu:")
		fmt.Println("  1. 🔍 Discover Resources")
		fmt.Println("  2. 📦 Import Resources")
		fmt.Println("  3. 📊 View Import History")
		fmt.Println("  4. 🔧 Configuration")
		fmt.Println("  5. 📚 Help & Documentation")
		fmt.Println("  6. 🚪 Exit")
		fmt.Println()
		fmt.Println("Note: Full interactive TUI is now available!")
		fmt.Println("Run with: driftmgr interactive --tui")
		return nil
	}

	// Start the Bubble Tea program
	p := tea.NewProgram(a, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ChangeScreen represents a screen change message
type ChangeScreen struct {
	Screen Screen
}

type StatusMsg struct {
	Message string
}

type ResourcesFoundMsg struct {
	Resources []models.Resource
}

type ImportCompleteMsg struct {
	Results []models.ImportResult
}

// Helper functions for sending messages
func SendStatus(message string) tea.Cmd {
	return func() tea.Msg {
		return StatusMsg{Message: message}
	}
}

func SendResourcesFound(resources []models.Resource) tea.Cmd {
	return func() tea.Msg {
		return ResourcesFoundMsg{Resources: resources}
	}
}

func SendImportComplete(results []models.ImportResult) tea.Cmd {
	return func() tea.Msg {
		return ImportCompleteMsg{Results: results}
	}
}
