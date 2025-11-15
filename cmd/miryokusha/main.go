package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Justice-Caban/Miryokusha/internal/tui"
)

func main() {
	// Create the application model
	m := tui.NewAppModel()

	// Run the TUI program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running Miryokusha: %v\n", err)
		os.Exit(1)
	}
}
