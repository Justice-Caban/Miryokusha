package theme

import "github.com/charmbracelet/lipgloss"

// Common styles used across all TUI views
// These shared styles ensure visual consistency and reduce code duplication

var (
	// Text Styles

	// TitleStyle is used for view titles and headers
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// SectionStyle is used for section headers within views
	SectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			MarginTop(1)

	// HelpStyle is used for help text and keyboard shortcuts
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	// MutedStyle is used for less important text
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// ValueStyle is used for displaying values in key-value pairs
	ValueStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	// Status Styles

	// SuccessStyle is used for success messages and indicators
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// ErrorStyle is used for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	// WarningStyle is used for warnings
	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// Container Styles

	// BoxStyle is a basic box with rounded borders
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	// HighlightStyle is used for selected/highlighted items
	HighlightStyle = lipgloss.NewStyle().
				Background(ColorPrimary).
				Foreground(lipgloss.Color("#000000")).
				Bold(true)

	// ActiveTabStyle is used for the active tab in tab navigation
	ActiveTabStyle = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 2)

	// InactiveTabStyle is used for inactive tabs
	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 2)

	// Badge Styles

	// BadgeStyle is a base style for badges (can be customized per use)
	BadgeStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("#000000"))

	// SuccessBadgeStyle for success badges
	SuccessBadgeStyle = BadgeStyle.Copy().Background(ColorSuccess)

	// ErrorBadgeStyle for error badges
	ErrorBadgeStyle = BadgeStyle.Copy().Background(ColorError)

	// WarningBadgeStyle for warning badges
	WarningBadgeStyle = BadgeStyle.Copy().Background(ColorWarning)

	// InfoBadgeStyle for informational badges
	InfoBadgeStyle = BadgeStyle.Copy().Background(ColorSecondary)
)

// Helper functions for common patterns

// RenderKeyValue renders a key-value pair with consistent styling
func RenderKeyValue(key, value string) string {
	return MutedStyle.Render(key+": ") + ValueStyle.Render(value)
}

// RenderTab renders a tab with appropriate styling based on active state
func RenderTab(label string, isActive bool) string {
	if isActive {
		return ActiveTabStyle.Render(label)
	}
	return InactiveTabStyle.Render(label)
}

// RenderBadge renders a badge with the given text and color
func RenderBadge(text string, style lipgloss.Style) string {
	return style.Render(text)
}

// RenderSection renders a section with a title and content
func RenderSection(title, content string) string {
	return SectionStyle.Render(title) + "\n" + content
}

// CenteredText centers text in the given width and height
func CenteredText(width, height int, text string) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)
	return style.Render(text)
}
