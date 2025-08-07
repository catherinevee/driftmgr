package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewMainMenu(t *testing.T) {
	menu := NewMainMenu()
	
	assert.NotNil(t, menu)
	assert.Equal(t, 0, menu.cursor)
	assert.Equal(t, 4, len(menu.options)) // Should have 4 menu options
	
	expectedOptions := []string{
		"🔍 Resource Discovery",
		"📥 Import Resources", 
		"⚙️  Configuration",
		"❓ Help",
	}
	
	for i, expected := range expectedOptions {
		assert.Equal(t, expected, menu.options[i])
	}
}

func TestMainMenu_Init(t *testing.T) {
	menu := NewMainMenu()
	cmd := menu.Init()
	
	// Init should return no command
	assert.Nil(t, cmd)
}

func TestMainMenu_Update_KeyDown(t *testing.T) {
	menu := NewMainMenu()
	
	// Test moving down
	keyMsg := tea.KeyMsg{Type: tea.KeyDown}
	newMenu, _ := menu.Update(keyMsg)
	updatedMenu := newMenu.(*MainMenu)
	
	assert.Equal(t, 1, updatedMenu.cursor)
	
	// Test wrapping to top
	menu.cursor = 3 // Last option
	newMenu, _ = menu.Update(keyMsg)
	updatedMenu = newMenu.(*MainMenu)
	
	assert.Equal(t, 0, updatedMenu.cursor)
}

func TestMainMenu_Update_KeyUp(t *testing.T) {
	menu := NewMainMenu()
	menu.cursor = 1
	
	// Test moving up
	keyMsg := tea.KeyMsg{Type: tea.KeyUp}
	newMenu, _ := menu.Update(keyMsg)
	updatedMenu := newMenu.(*MainMenu)
	
	assert.Equal(t, 0, updatedMenu.cursor)
	
	// Test wrapping to bottom
	menu.cursor = 0 // First option
	newMenu, _ = menu.Update(keyMsg)
	updatedMenu = newMenu.(*MainMenu)
	
	assert.Equal(t, 3, updatedMenu.cursor)
}

func TestMainMenu_Update_Enter(t *testing.T) {
	menu := NewMainMenu()
	
	tests := []struct {
		name           string
		cursor         int
		expectedScreen Screen
	}{
		{
			name:           "select discovery",
			cursor:         0,
			expectedScreen: DiscoveryScreen,
		},
		{
			name:           "select import",
			cursor:         1,
			expectedScreen: ImportScreen,
		},
		{
			name:           "select config",
			cursor:         2,
			expectedScreen: ConfigScreen,
		},
		{
			name:           "select help",
			cursor:         3,
			expectedScreen: HelpScreen,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			menu.cursor = tt.cursor
			
			keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
			_, cmd := menu.Update(keyMsg)
			
			assert.NotNil(t, cmd)
			
			// Execute the command to get the message
			msg := cmd()
			switchMsg, ok := msg.(SwitchScreenMsg)
			assert.True(t, ok)
			assert.Equal(t, tt.expectedScreen, switchMsg.Screen)
		})
	}
}

func TestMainMenu_Update_J_K_Keys(t *testing.T) {
	menu := NewMainMenu()
	
	// Test 'j' key (down)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	newMenu, _ := menu.Update(keyMsg)
	updatedMenu := newMenu.(*MainMenu)
	
	assert.Equal(t, 1, updatedMenu.cursor)
	
	// Test 'k' key (up)
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	newMenu, _ = updatedMenu.Update(keyMsg)
	updatedMenu = newMenu.(*MainMenu)
	
	assert.Equal(t, 0, updatedMenu.cursor)
}

func TestMainMenu_View(t *testing.T) {
	menu := NewMainMenu()
	view := menu.View()
	
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Terraform Import Helper")
	assert.Contains(t, view, "🔍 Resource Discovery")
	assert.Contains(t, view, "📥 Import Resources")
	assert.Contains(t, view, "⚙️  Configuration")
	assert.Contains(t, view, "❓ Help")
	assert.Contains(t, view, "Press q to quit")
}

func TestMainMenu_View_CursorPosition(t *testing.T) {
	menu := NewMainMenu()
	
	for i := 0; i < len(menu.options); i++ {
		menu.cursor = i
		view := menu.View()
		
		assert.NotEmpty(t, view)
		// The view should contain the cursor indicator for the selected option
		assert.Contains(t, view, "→")
	}
}
