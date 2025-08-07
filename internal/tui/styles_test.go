package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestColors(t *testing.T) {
	// Test that all color constants are defined
	colors := []lipgloss.Color{
		PrimaryColor,
		SecondaryColor,
		AccentColor,
		SuccessColor,
		WarningColor,
		ErrorColor,
		MutedColor,
		BackgroundColor,
		BorderColor,
		HighlightColor,
	}

	for _, color := range colors {
		assert.NotEmpty(t, string(color))
	}
}

func TestBaseStyle(t *testing.T) {
	style := BaseStyle.Render("test")
	assert.NotEmpty(t, style)
}

func TestTitleStyle(t *testing.T) {
	title := TitleStyle.Render("Test Title")
	assert.NotEmpty(t, title)
	assert.Contains(t, title, "Test Title")
}

func TestMenuItemStyle(t *testing.T) {
	item := MenuItemStyle.Render("Test Menu Item")
	assert.NotEmpty(t, item)
	assert.Contains(t, item, "Test Menu Item")
}

func TestSelectedMenuItemStyle(t *testing.T) {
	item := SelectedMenuItemStyle.Render("Selected Item")
	assert.NotEmpty(t, item)
	assert.Contains(t, item, "Selected Item")
}

func TestButtonStyle(t *testing.T) {
	button := ButtonStyle.Render("Test Button")
	assert.NotEmpty(t, button)
	assert.Contains(t, button, "Test Button")
}

func TestActiveButtonStyle(t *testing.T) {
	button := ActiveButtonStyle.Render("Active Button")
	assert.NotEmpty(t, button)
	assert.Contains(t, button, "Active Button")
}

func TestTableHeaderStyle(t *testing.T) {
	header := TableHeaderStyle.Render("Header")
	assert.NotEmpty(t, header)
	assert.Contains(t, header, "Header")
}

func TestTableCellStyle(t *testing.T) {
	cell := TableCellStyle.Render("Cell Content")
	assert.NotEmpty(t, cell)
	assert.Contains(t, cell, "Cell Content")
}

func TestSelectedTableRowStyle(t *testing.T) {
	row := SelectedTableRowStyle.Render("Selected Row")
	assert.NotEmpty(t, row)
	assert.Contains(t, row, "Selected Row")
}

func TestErrorStyle(t *testing.T) {
	error := ErrorStyle.Render("Error Message")
	assert.NotEmpty(t, error)
	assert.Contains(t, error, "Error Message")
}

func TestSuccessStyle(t *testing.T) {
	success := SuccessStyle.Render("Success Message")
	assert.NotEmpty(t, success)
	assert.Contains(t, success, "Success Message")
}

func TestWarningStyle(t *testing.T) {
	warning := WarningStyle.Render("Warning Message")
	assert.NotEmpty(t, warning)
	assert.Contains(t, warning, "Warning Message")
}

func TestInputStyle(t *testing.T) {
	input := InputStyle.Render("Input Text")
	assert.NotEmpty(t, input)
	assert.Contains(t, input, "Input Text")
}

func TestFocusedInputStyle(t *testing.T) {
	input := FocusedInputStyle.Render("Focused Input")
	assert.NotEmpty(t, input)
	assert.Contains(t, input, "Focused Input")
}

func TestHelpStyle(t *testing.T) {
	help := HelpStyle.Render("Help Text")
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "Help Text")
}

func TestBorderStyle(t *testing.T) {
	border := BorderStyle.Render("Bordered Content")
	assert.NotEmpty(t, border)
	assert.Contains(t, border, "Bordered Content")
}

func TestRenderTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{
			name:  "simple title",
			title: "Test Title",
		},
		{
			name:  "title with special characters",
			title: "Test Title with 🎉 Emojis!",
		},
		{
			name:  "empty title",
			title: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderTitle(tt.title)
			assert.NotEmpty(t, result)
			if tt.title != "" {
				assert.Contains(t, result, tt.title)
			}
		})
	}
}

func TestRenderMenuItem(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		isSelected bool
	}{
		{
			name:       "unselected menu item",
			text:       "Menu Item",
			isSelected: false,
		},
		{
			name:       "selected menu item",
			text:       "Selected Item",
			isSelected: true,
		},
		{
			name:       "empty menu item",
			text:       "",
			isSelected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderMenuItem(tt.text, tt.isSelected)
			assert.NotEmpty(t, result)

			if tt.isSelected {
				assert.Contains(t, result, "→")
			}

			if tt.text != "" {
				assert.Contains(t, result, tt.text)
			}
		})
	}
}

func TestRenderButton(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		isActive bool
	}{
		{
			name:     "inactive button",
			text:     "Button",
			isActive: false,
		},
		{
			name:     "active button",
			text:     "Active Button",
			isActive: true,
		},
		{
			name:     "empty button",
			text:     "",
			isActive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderButton(tt.text, tt.isActive)
			assert.NotEmpty(t, result)

			if tt.text != "" {
				assert.Contains(t, result, tt.text)
			}
		})
	}
}
