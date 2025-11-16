package theme

import "github.com/charmbracelet/lipgloss"

// Color palette - uses terminal's native color scheme for better integration
// These colors adapt to the user's terminal theme (dark/light mode, custom themes)
// Using the standard 16-color ANSI palette (0-15):
//   0-7: Normal colors (black, red, green, yellow, blue, magenta, cyan, white)
//   8-15: Bright colors (bright versions of the above)
var (
	ColorPrimary   = lipgloss.Color("13") // Bright Magenta - primary accent
	ColorSecondary = lipgloss.Color("12") // Bright Blue - secondary accent
	ColorAccent    = lipgloss.Color("14") // Bright Cyan - highlights
	ColorSuccess   = lipgloss.Color("10") // Bright Green - success states
	ColorWarning   = lipgloss.Color("11") // Bright Yellow - warnings
	ColorError     = lipgloss.Color("9")  // Bright Red - errors
	ColorMuted     = lipgloss.Color("8")  // Bright Black (Gray) - muted text
	ColorBorder    = lipgloss.Color("8")  // Bright Black (Gray) - borders
)
