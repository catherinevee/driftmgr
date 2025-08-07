package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#00D4AA")
	secondaryColor = lipgloss.Color("#FF6B6B")
	accentColor    = lipgloss.Color("#4ECDC4")
	warningColor   = lipgloss.Color("#FFE66D")
	errorColor     = lipgloss.Color("#FF6B6B")
	successColor   = lipgloss.Color("#4ECDC4")
	textColor      = lipgloss.Color("#FFFFFF")
	mutedColor     = lipgloss.Color("#666666")
	borderColor    = lipgloss.Color("#3C3C3C")

	// Base styles
	baseStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(lipgloss.Color("#1a1a1a"))

	// Header styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Background(lipgloss.Color("#2a2a2a")).
			Padding(1, 2).
			Margin(0, 0, 1, 0).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Align(lipgloss.Center).
			Margin(1)

	// Menu styles
	menuStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1).
			Margin(1)

	selectedMenuItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				Background(lipgloss.Color("#2a2a2a")).
				Padding(0, 1)

	unselectedMenuItemStyle = lipgloss.NewStyle().
				Foreground(textColor).
				Padding(0, 1)

	// Content styles
	contentStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1).
			Margin(1).
			Height(20).
			Width(80)

	// Status styles
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2a2a2a")).
			Foreground(mutedColor).
			Padding(0, 1).
			Margin(1, 0, 0, 0)

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(successColor)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(errorColor)

	warningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(warningColor)

	// Table styles
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				Background(lipgloss.Color("#2a2a2a")).
				Padding(0, 1)

	tableRowStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Padding(0, 1)

	tableSelectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor).
				Background(lipgloss.Color("#2a2a2a")).
				Padding(0, 1)

	// Progress styles
	progressBarStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderColor).
				Padding(1).
				Margin(1)

	progressCompleteStyle = lipgloss.NewStyle().
				Background(successColor).
				Foreground(lipgloss.Color("#000000"))

	progressIncompleteStyle = lipgloss.NewStyle().
					Background(mutedColor)

	// Button styles
	buttonStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textColor).
			Background(primaryColor).
			Padding(0, 2).
			Margin(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor)

	activeButtonStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#000000")).
				Background(accentColor).
				Padding(0, 2).
				Margin(0, 1).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor)

	// Input styles
	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1).
			Margin(0, 1)

	focusedInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1).
				Margin(0, 1)

	// Help styles
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Margin(1, 0)

	// Loading styles
	spinnerStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Margin(0, 1)
)

// Helper functions for consistent styling
func RenderTitle(text string) string {
	return titleStyle.Render(text)
}

func RenderHeader(text string) string {
	return headerStyle.Render(text)
}

func RenderSuccess(text string) string {
	return successStyle.Render("✅ " + text)
}

func RenderError(text string) string {
	return errorStyle.Render("❌ " + text)
}

func RenderWarning(text string) string {
	return warningStyle.Render("⚠️  " + text)
}

func RenderMenuItem(text string, selected bool) string {
	if selected {
		return selectedMenuItemStyle.Render("▶ " + text)
	}
	return unselectedMenuItemStyle.Render("  " + text)
}

func RenderButton(text string, active bool) string {
	if active {
		return activeButtonStyle.Render(text)
	}
	return buttonStyle.Render(text)
}

func RenderProgressBar(current, total int, width int) string {
	if total == 0 {
		return progressBarStyle.Width(width).Render("No progress")
	}

	progress := float64(current) / float64(total)
	completed := int(float64(width) * progress)
	
	completeBar := progressCompleteStyle.Width(completed).Render("")
	incompleteBar := progressIncompleteStyle.Width(width - completed).Render("")
	
	progressText := lipgloss.JoinHorizontal(lipgloss.Left, completeBar, incompleteBar)
	statusText := lipgloss.NewStyle().Align(lipgloss.Center).Width(width).
		Render(lipgloss.JoinVertical(lipgloss.Center, 
			progressText, 
			fmt.Sprintf("%d/%d (%.1f%%)", current, total, progress*100)))
	
	return progressBarStyle.Width(width + 4).Render(statusText)
}

func RenderStatusBar(text string) string {
	return statusBarStyle.Render(text)
}

func RenderHelp(text string) string {
	return helpStyle.Render(text)
}
