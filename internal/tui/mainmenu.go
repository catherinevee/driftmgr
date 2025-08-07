package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MainMenu represents the main menu view
type MainMenu struct {
	choices []menuChoice
	cursor  int
	width   int
	height  int
}

type menuChoice struct {
	title       string
	description string
	action      string
}

// NewMainMenu creates a new main menu
func NewMainMenu() *MainMenu {
	return &MainMenu{
		choices: []menuChoice{
			{
				title:       "🔍 Discover Resources",
				description: "Scan cloud providers for existing resources",
				action:      "discovery",
			},
			{
				title:       "📦 Import Resources",
				description: "Import discovered resources into Terraform state",
				action:      "import",
			},
			{
				title:       "📊 View Import History",
				description: "Review previous import operations and results",
				action:      "history",
			},
			{
				title:       "🔧 Configuration",
				description: "Manage cloud provider credentials and settings",
				action:      "config",
			},
			{
				title:       "📚 Help & Documentation",
				description: "View help, examples, and troubleshooting guides",
				action:      "help",
			},
			{
				title:       "🚪 Exit",
				description: "Quit the application",
				action:      "quit",
			},
		},
	}
}

// Update implements tea.Model interface
func (m *MainMenu) Update(msg tea.Msg) (*MainMenu, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.choices) - 1
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "enter", " ":
			choice := m.choices[m.cursor]
			switch choice.action {
			case "quit":
				return m, tea.Quit
			case "discovery":
				return m, ChangeScreen(DiscoveryScreen, "Resource Discovery")
			case "import":
				return m, ChangeScreen(ImportScreen, "Resource Import")
			case "config":
				return m, ChangeScreen(ConfigScreen, "Configuration")
			case "help":
				return m, ChangeScreen(HelpScreen, "Help & Documentation")
			case "history":
				return m, SetStatus("Import history feature coming soon!")
			}
		}
	}
	return m, nil
}

// View implements the display for main menu
func (m *MainMenu) View() string {
	// Title
	title := RenderTitle("📋 Main Menu")

	// Menu items
	var menuItems []string
	for i, choice := range m.choices {
		menuItems = append(menuItems, RenderMenuItem(choice.title, m.cursor == i))
		if m.cursor == i {
			// Show description for selected item
			desc := lipgloss.NewStyle().
				Foreground(mutedColor).
				Margin(0, 0, 0, 4).
				Render(choice.description)
			menuItems = append(menuItems, desc)
		}
		menuItems = append(menuItems, "") // Add spacing
	}

	// Instructions
	instructions := RenderHelp("Use ↑/↓ or j/k to navigate, Enter to select")

	// Status section
	status := lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(1, 0).
		Render("Welcome to the Terraform Import Helper! Select an option to get started.")

	// Combine all sections
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		status,
		"",
		lipgloss.JoinVertical(lipgloss.Left, menuItems...),
		instructions,
	)

	return menuStyle.Render(content)
}
