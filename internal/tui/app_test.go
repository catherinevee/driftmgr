package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	app := NewApp()

	assert.NotNil(t, app)
	assert.Equal(t, MainMenuScreen, app.currentScreen)
	assert.NotNil(t, app.mainMenu)
	assert.NotNil(t, app.discovery)
	assert.NotNil(t, app.importView)
	assert.NotNil(t, app.config)
	assert.NotNil(t, app.help)
}

func TestApp_Init(t *testing.T) {
	app := NewApp()
	cmd := app.Init()

	// Init should return no command
	assert.Nil(t, cmd)
}

func TestApp_Update(t *testing.T) {
	app := NewApp()

	tests := []struct {
		name           string
		msg            tea.Msg
		expectedScreen Screen
	}{
		{
			name:           "navigate to discovery",
			msg:            SwitchScreenMsg{Screen: DiscoveryScreen},
			expectedScreen: DiscoveryScreen,
		},
		{
			name:           "navigate to import",
			msg:            SwitchScreenMsg{Screen: ImportScreen},
			expectedScreen: ImportScreen,
		},
		{
			name:           "navigate to config",
			msg:            SwitchScreenMsg{Screen: ConfigScreen},
			expectedScreen: ConfigScreen,
		},
		{
			name:           "navigate to help",
			msg:            SwitchScreenMsg{Screen: HelpScreen},
			expectedScreen: HelpScreen,
		},
		{
			name:           "navigate back to main menu",
			msg:            SwitchScreenMsg{Screen: MainMenuScreen},
			expectedScreen: MainMenuScreen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newApp, _ := app.Update(tt.msg)
			updatedApp := newApp.(*App)

			assert.Equal(t, tt.expectedScreen, updatedApp.currentScreen)
		})
	}
}

func TestApp_Update_KeyMsg(t *testing.T) {
	app := NewApp()

	tests := []struct {
		name     string
		key      string
		expected bool // whether app should quit
	}{
		{
			name:     "ctrl+c quits",
			key:      "ctrl+c",
			expected: true,
		},
		{
			name:     "q quits from main menu",
			key:      "q",
			expected: true,
		},
		{
			name:     "esc goes to main menu",
			key:      "esc",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to main menu for each test
			app.currentScreen = MainMenuScreen

			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "ctrl+c" {
				keyMsg = tea.KeyMsg{Type: tea.KeyCtrlC}
			} else if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			}

			newApp, cmd := app.Update(keyMsg)

			if tt.expected {
				// Should have quit command
				assert.NotNil(t, cmd)
			} else {
				updatedApp := newApp.(*App)
				if tt.key == "esc" {
					assert.Equal(t, MainMenuScreen, updatedApp.currentScreen)
				}
			}
		})
	}
}

func TestApp_View(t *testing.T) {
	app := NewApp()

	tests := []struct {
		name   string
		screen Screen
	}{
		{
			name:   "main menu view",
			screen: MainMenuScreen,
		},
		{
			name:   "discovery view",
			screen: DiscoveryScreen,
		},
		{
			name:   "import view",
			screen: ImportScreen,
		},
		{
			name:   "config view",
			screen: ConfigScreen,
		},
		{
			name:   "help view",
			screen: HelpScreen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.currentScreen = tt.screen
			view := app.View()

			assert.NotEmpty(t, view)
			assert.Contains(t, view, "Terraform Import Helper") // Should contain title
		})
	}
}

func TestSwitchScreenMsg(t *testing.T) {
	msg := SwitchScreenMsg{Screen: DiscoveryScreen}
	assert.Equal(t, DiscoveryScreen, msg.Screen)
}

func TestScreen_Constants(t *testing.T) {
	// Test that all screen constants are defined
	screens := []Screen{
		MainMenuScreen,
		DiscoveryScreen,
		ImportScreen,
		ConfigScreen,
		HelpScreen,
	}

	for i, screen := range screens {
		assert.Equal(t, Screen(i), screen)
	}
}
