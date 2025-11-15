package tui

import (
	"github.com/Justice-Caban/Miryokusha/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// Re-export theme colors for backward compatibility
var (
	ColorPrimary   = theme.ColorPrimary
	ColorSecondary = theme.ColorSecondary
	ColorAccent    = theme.ColorAccent
	ColorSuccess   = theme.ColorSuccess
	ColorWarning   = theme.ColorWarning
	ColorError     = theme.ColorError
	ColorMuted     = theme.ColorMuted
	ColorBorder    = theme.ColorBorder
)

// Common styles
var (
	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// Subtitle styles
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			MarginBottom(1)

	// Border styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	// List item styles
	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				PaddingLeft(2)

	UnselectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				PaddingLeft(2)

	// Status styles
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	// Help text styles
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	// Error styles
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// Success styles
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	// Muted text style
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// GetStatusBarText formats a status bar message
func GetStatusBarText(items ...string) string {
	var result string
	for i, item := range items {
		if i > 0 {
			result += " │ "
		}
		result += item
	}
	return StatusBarStyle.Render(result)
}

// CenteredText centers text in the given width and height
func CenteredText(width, height int, text string) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)
	return style.Render(text)
}
