package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type HelpView struct {
	width           int
	height          int
	selectedSection int
	sections        []helpSection
}

type helpSection struct {
	title   string
	content []string
}

func NewHelpView() *HelpView {
	sections := []helpSection{
		{
			title: "Getting Started",
			content: []string{
				"Welcome to Drift Manager - the Terraform Import Helper!",
				"",
				"This tool helps you discover cloud resources and generate",
				"Terraform configuration to import them into your state.",
				"",
				"Key Features:",
				"• Multi-cloud resource discovery (AWS, Azure, GCP)",
				"• Interactive terminal user interface",
				"• Automated Terraform configuration generation",
				"• Real-time resource scanning",
				"• Parallel import operations",
			},
		},
		{
			title: "Navigation",
			content: []string{
				"Main Menu:",
				"• D - Resource Discovery",
				"• I - Import Resources",
				"• C - Configuration",
				"• H - Help (this screen)",
				"• Q - Quit",
				"",
				"Universal Keys:",
				"• Esc - Go back to previous screen",
				"• Ctrl+C - Force quit",
				"• ↑/↓ - Navigate lists",
				"• Enter - Select/Confirm",
				"• Tab/Shift+Tab - Navigate form fields",
			},
		},
		{
			title: "Resource Discovery",
			content: []string{
				"The discovery feature scans your cloud accounts for",
				"existing resources that can be imported into Terraform.",
				"",
				"Supported Providers:",
				"• AWS - EC2, VPC, S3, Security Groups",
				"• Azure - Virtual Machines, Resource Groups, Storage",
				"• Google Cloud - Compute Instances, Networks",
				"",
				"Discovery Steps:",
				"1. Select cloud provider",
				"2. Choose regions to scan",
				"3. View discovered resources",
				"4. Select resources for import",
			},
		},
		{
			title: "Configuration",
			content: []string{
				"Configure cloud provider credentials and import settings.",
				"",
				"AWS Configuration:",
				"• Profile - AWS CLI profile name",
				"• Region - Default AWS region",
				"",
				"Azure Configuration:",
				"• Subscription ID - Azure subscription",
				"• Tenant ID - Azure AD tenant",
				"",
				"GCP Configuration:",
				"• Project ID - Google Cloud project",
				"• Credentials - Path to service account JSON",
				"",
				"Import Settings:",
				"• Output Directory - Where to save Terraform files",
				"• Parallelism - Number of concurrent operations",
			},
		},
		{
			title: "Import Process",
			content: []string{
				"Once you've discovered resources, you can generate",
				"Terraform configuration files for them.",
				"",
				"Import generates:",
				"• main.tf - Resource definitions",
				"• variables.tf - Input variables",
				"• import.tf - Import blocks",
				"• terraform.tfvars - Variable values",
				"",
				"After import:",
				"1. Review generated files",
				"2. Run 'terraform plan' to verify",
				"3. Run 'terraform apply' to finalize",
				"4. Remove import blocks if desired",
			},
		},
		{
			title: "Troubleshooting",
			content: []string{
				"Common Issues:",
				"",
				"Authentication Errors:",
				"• Verify cloud provider credentials",
				"• Check IAM permissions",
				"• Ensure correct profile/subscription selection",
				"",
				"Discovery Problems:",
				"• Check network connectivity",
				"• Verify region availability",
				"• Review access permissions for resources",
				"",
				"Import Failures:",
				"• Ensure Terraform binary is in PATH",
				"• Check output directory permissions",
				"• Verify resource still exists",
				"",
				"For more help, check the documentation at:",
				"https://github.com/catherinevee/driftmgr",
			},
		},
		{
			title: "Keyboard Shortcuts",
			content: []string{
				"Global Shortcuts:",
				"• Ctrl+C - Quit application",
				"• Esc - Go back/Cancel",
				"• ↑/↓ - Navigate up/down",
				"• Enter - Select/Confirm",
				"",
				"Discovery View:",
				"• Space - Toggle resource selection",
				"• A - Select all resources",
				"• N - Select none",
				"• F - Filter resources",
				"",
				"Configuration View:",
				"• Ctrl+S - Save configuration",
				"• Ctrl+R - Reset to defaults",
				"• Tab - Next field",
				"• Shift+Tab - Previous field",
				"",
				"Import View:",
				"• Delete - Remove selected resource",
				"• Enter - Start import process",
			},
		},
		{
			title: "Tips & Best Practices",
			content: []string{
				"Discovery Tips:",
				"• Start with one region to test connectivity",
				"• Use filters to find specific resource types",
				"• Review resources before importing",
				"",
				"Import Best Practices:",
				"• Use dry run mode first",
				"• Import resources into separate directories",
				"• Review generated configuration carefully",
				"• Test with 'terraform plan' before applying",
				"",
				"Configuration Management:",
				"• Set up cloud credentials securely",
				"• Use descriptive output directories",
				"• Configure appropriate parallelism",
				"• Enable logging for troubleshooting",
				"",
				"Performance:",
				"• Scan fewer regions for faster discovery",
				"• Adjust parallelism based on API limits",
				"• Use resource filters to reduce scope",
			},
		},
	}

	return &HelpView{
		sections: sections,
	}
}

func (h HelpView) Init() tea.Cmd {
	return nil
}

func (h *HelpView) Update(msg tea.Msg) (*HelpView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height
		return h, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if h.selectedSection > 0 {
				h.selectedSection--
			}
		case "down", "j":
			if h.selectedSection < len(h.sections)-1 {
				h.selectedSection++
			}
		case "home":
			h.selectedSection = 0
		case "end":
			h.selectedSection = len(h.sections) - 1
		case "esc":
			return h, tea.Cmd(func() tea.Msg { return ChangeScreen{Screen: ScreenMain} })
		}
	}

	return h, nil
}

func (h HelpView) View() string {
	title := RenderTitle("Help & Documentation")

	// Create navigation sidebar
	sidebar := h.renderSidebar()

	// Create content area
	content := h.renderContent()

	// Layout sidebar and content side by side
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		lipgloss.NewStyle().Width(2).Render(""), // Spacer
		content,
	)

	// Controls
	controls := lipgloss.NewStyle().
		Foreground(SecondaryTextColor).
		Render("Navigation: ↑/↓ or J/K • Home/End • Back: Esc")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		mainContent,
		"",
		controls,
	)
}

func (h HelpView) renderSidebar() string {
	sidebarStyle := lipgloss.NewStyle().
		Width(25).
		Height(h.height - 10).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(AccentColor).
		Padding(1)

	var items []string
	for i, section := range h.sections {
		itemStyle := lipgloss.NewStyle().
			Foreground(PrimaryTextColor)

		if i == h.selectedSection {
			itemStyle = itemStyle.
				Background(AccentColor).
				Foreground(BackgroundColor).
				Bold(true).
				Padding(0, 1)
		}

		items = append(items, itemStyle.Render(section.title))
	}

	sidebarContent := lipgloss.JoinVertical(lipgloss.Left, items...)
	return sidebarStyle.Render(sidebarContent)
}

func (h HelpView) renderContent() string {
	contentStyle := lipgloss.NewStyle().
		Width(h.width - 30). // Account for sidebar width
		Height(h.height - 10).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(SecondaryTextColor).
		Padding(1)

	if h.selectedSection >= len(h.sections) {
		return contentStyle.Render("No content available")
	}

	section := h.sections[h.selectedSection]

	// Section title
	sectionTitle := lipgloss.NewStyle().
		Foreground(AccentColor).
		Bold(true).
		Underline(true).
		Render(section.title)

	// Section content
	var contentLines []string
	contentLines = append(contentLines, sectionTitle, "")

	for _, line := range section.content {
		if line == "" {
			contentLines = append(contentLines, "")
			continue
		}

		// Style bullets and headers
		if strings.HasPrefix(line, "•") {
			line = lipgloss.NewStyle().
				Foreground(AccentColor).
				Render("•") + " " + line[2:]
		} else if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			line = lipgloss.NewStyle().
				Foreground(AccentColor).
				Bold(true).
				Render(line)
		}

		contentLines = append(contentLines, line)
	}

	contentText := lipgloss.JoinVertical(lipgloss.Left, contentLines...)
	return contentStyle.Render(contentText)
}
