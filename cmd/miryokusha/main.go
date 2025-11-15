package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the application state
type Model struct {
	ready bool
}

// Init initializes the application
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.ready = true
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	return `
╔════════════════════════════════════════════════════════════════╗
║                         MIRYOKUSHA                             ║
║                 Suwayomi TUI Client for Go                     ║
╚════════════════════════════════════════════════════════════════╝

Welcome to Miryokusha!

This is a work-in-progress TUI client for Suwayomi/Tachiyomi servers.

Press 'q' to quit.

[Development Status: Initialization Complete ✓]
`
}

func main() {
	m := Model{}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
