package theme

import "github.com/charmbracelet/lipgloss"

// Color palette - centralized theme colors for all TUI components
var (
	ColorPrimary   = lipgloss.Color("205") // Pink
	ColorSecondary = lipgloss.Color("99")  // Purple
	ColorAccent    = lipgloss.Color("86")  // Cyan
	ColorSuccess   = lipgloss.Color("42")  // Green
	ColorWarning   = lipgloss.Color("214") // Orange
	ColorError     = lipgloss.Color("196") // Red
	ColorMuted     = lipgloss.Color("242") // Gray
	ColorBorder    = lipgloss.Color("238") // Dark gray
)
