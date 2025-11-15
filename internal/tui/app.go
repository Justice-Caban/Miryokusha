package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewType represents the current active view
type ViewType string

const (
	ViewHome       ViewType = "home"
	ViewLibrary    ViewType = "library"
	ViewHistory    ViewType = "history"
	ViewBrowse     ViewType = "browse"
	ViewDownloads  ViewType = "downloads"
	ViewExtensions ViewType = "extensions"
	ViewSettings   ViewType = "settings"
)

// AppModel is the root model for the entire TUI application
type AppModel struct {
	currentView ViewType
	ready       bool
	width       int
	height      int
	err         error
}

// NewAppModel creates a new application model
func NewAppModel() AppModel {
	return AppModel{
		currentView: ViewHome,
	}
}

// Init initializes the application
func (m AppModel) Init() tea.Cmd {
	return nil
}

// Update handles all messages and routes them appropriately
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.currentView == ViewHome {
				return m, tea.Quit
			}
			m.currentView = ViewHome
			return m, nil

		// View navigation shortcuts
		case "1":
			m.currentView = ViewHome
		case "2":
			m.currentView = ViewLibrary
		case "3":
			m.currentView = ViewHistory
		case "4":
			m.currentView = ViewBrowse
		case "5":
			m.currentView = ViewDownloads
		case "6":
			m.currentView = ViewExtensions
		case "7":
			m.currentView = ViewSettings
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
	}

	return m, nil
}

// View renders the current view
func (m AppModel) View() string {
	if !m.ready {
		return "Initializing Miryokusha..."
	}

	var content string

	// Render the appropriate view based on currentView
	switch m.currentView {
	case ViewHome:
		content = m.renderHomeView()
	case ViewLibrary:
		content = m.renderPlaceholderView("Library", "📚 Your manga library will appear here")
	case ViewHistory:
		content = m.renderPlaceholderView("Reading History", "📖 Your reading history will appear here")
	case ViewBrowse:
		content = m.renderPlaceholderView("Browse", "🔍 Browse manga sources here")
	case ViewDownloads:
		content = m.renderPlaceholderView("Downloads", "📥 Download queue will appear here")
	case ViewExtensions:
		content = m.renderPlaceholderView("Extensions", "🧩 Manage extensions here")
	case ViewSettings:
		content = m.renderPlaceholderView("Settings", "⚙️  Application settings")
	default:
		content = m.renderHomeView()
	}

	// Render status bar
	statusBar := m.renderStatusBar()

	// Combine content and status bar
	mainContent := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height - lipgloss.Height(statusBar)).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, statusBar)
}

// renderHomeView renders the home/welcome screen
func (m AppModel) renderHomeView() string {
	title := TitleStyle.Render("🌸 MIRYOKUSHA")
	subtitle := SubtitleStyle.Render("Suwayomi TUI Client")

	menu := lipgloss.NewStyle().
		MarginTop(2).
		MarginBottom(2).
		Render(`
Navigation:
  1 - Home (this screen)
  2 - Library
  3 - History
  4 - Browse
  5 - Downloads
  6 - Extensions
  7 - Settings

  q - Quit
`)

	status := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render("Development Status: Basic TUI Framework ✓")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		menu,
		status,
	)

	// Center the content
	return lipgloss.Place(
		m.width,
		m.height-3, // Reserve space for status bar
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderPlaceholderView renders a placeholder for unimplemented views
func (m AppModel) renderPlaceholderView(title, description string) string {
	viewTitle := TitleStyle.Render(title)
	viewDesc := lipgloss.NewStyle().
		Foreground(ColorMuted).
		MarginTop(1).
		Render(description)

	backHint := HelpStyle.Render("Press 'q' to go back to home")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		viewTitle,
		viewDesc,
		"",
		backHint,
	)

	// Center the content
	return lipgloss.Place(
		m.width,
		m.height-3,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderStatusBar renders the bottom status bar
func (m AppModel) renderStatusBar() string {
	viewName := fmt.Sprintf("View: %s", m.currentView)
	dimensions := fmt.Sprintf("%dx%d", m.width, m.height)
	help := "Press ? for help"

	return GetStatusBarText(viewName, dimensions, help)
}
