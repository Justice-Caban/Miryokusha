package tui

import (
	"fmt"

	"github.com/Justice-Caban/Miryokusha/internal/config"
	"github.com/Justice-Caban/Miryokusha/internal/source"
	"github.com/Justice-Caban/Miryokusha/internal/storage"
	"github.com/Justice-Caban/Miryokusha/internal/tui/history"
	"github.com/Justice-Caban/Miryokusha/internal/tui/library"
	"github.com/Justice-Caban/Miryokusha/internal/tui/reader"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewType represents the current active view
type ViewType string

const (
	ViewHome       ViewType = "home"
	ViewLibrary    ViewType = "library"
	ViewReader     ViewType = "reader"
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

	// Dependencies
	config        *config.Config
	sourceManager *source.SourceManager
	storage       *storage.Storage

	// View models
	libraryModel library.Model
	historyModel history.Model
	readerModel  *reader.Model
}

// NewAppModel creates a new application model
func NewAppModel() AppModel {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Use default config if loading fails
		cfg = config.DefaultConfig()
	}

	// Initialize storage
	st, err := storage.NewStorage()
	if err != nil {
		// Handle error but continue (storage is optional)
		st = nil
	}

	// Initialize source manager
	sm := source.NewSourceManager()

	// Initialize library model
	libModel := library.NewModel(sm, st)

	// Initialize history model
	histModel := history.NewModel(sm, st)

	return AppModel{
		currentView:   ViewHome,
		config:        cfg,
		sourceManager: sm,
		storage:       st,
		libraryModel:  libModel,
		historyModel:  histModel,
	}
}

// Init initializes the application
func (m AppModel) Init() tea.Cmd {
	return nil
}

// Update handles all messages and routes them appropriately
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case OpenReaderMsg:
		// Launch reader with manga and chapter
		readerModel := reader.NewModel(msg.Manga, msg.Chapter, m.sourceManager, m.storage)
		m.readerModel = &readerModel
		m.currentView = ViewReader
		return m, m.readerModel.Init()

	case history.OpenChapterMsg:
		// Launch reader from history by loading manga and chapter from source
		// For now, create a minimal manga/chapter to launch reader
		// TODO: Fetch full manga/chapter details from source
		manga := &source.Manga{
			ID:         msg.MangaID,
			SourceType: source.SourceTypeSuwayomi, // TODO: Get from history entry
		}
		chapter := &source.Chapter{
			ID:      msg.ChapterID,
			MangaID: msg.MangaID,
		}
		readerModel := reader.NewModel(manga, chapter, m.sourceManager, m.storage)
		m.readerModel = &readerModel
		m.currentView = ViewReader
		return m, m.readerModel.Init()

	case tea.KeyMsg:
		// Handle global shortcuts
		if m.currentView != ViewHome {
			switch msg.String() {
			case "esc":
				// Save reader session before closing
				if m.currentView == ViewReader && m.readerModel != nil {
					m.readerModel.SaveSession()
				}
				m.currentView = ViewHome
				m.readerModel = nil
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+c":
			// Save reader session before quitting
			if m.currentView == ViewReader && m.readerModel != nil {
				m.readerModel.SaveSession()
			}
			return m, tea.Quit

		case "q":
			if m.currentView == ViewHome {
				return m, tea.Quit
			}
			// Save reader session before going home
			if m.currentView == ViewReader && m.readerModel != nil {
				m.readerModel.SaveSession()
			}
			m.currentView = ViewHome
			m.readerModel = nil
			return m, nil

		// View navigation shortcuts (only from home)
		case "1":
			if m.currentView == ViewHome {
				m.currentView = ViewHome
			}
		case "2":
			if m.currentView == ViewHome {
				m.currentView = ViewLibrary
				m.libraryModel, cmd = m.libraryModel.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
				return m, cmd
			}
		case "3":
			if m.currentView == ViewHome {
				m.currentView = ViewHistory
				m.historyModel, cmd = m.historyModel.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
				return m, cmd
			}
		case "4":
			if m.currentView == ViewHome {
				m.currentView = ViewBrowse
			}
		case "5":
			if m.currentView == ViewHome {
				m.currentView = ViewDownloads
			}
		case "6":
			if m.currentView == ViewHome {
				m.currentView = ViewExtensions
			}
		case "7":
			if m.currentView == ViewHome {
				m.currentView = ViewSettings
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
	}

	// Route messages to active view
	switch m.currentView {
	case ViewLibrary:
		m.libraryModel, cmd = m.libraryModel.Update(msg)
		return m, cmd

	case ViewHistory:
		m.historyModel, cmd = m.historyModel.Update(msg)
		return m, cmd

	case ViewReader:
		if m.readerModel != nil {
			updated, cmd := m.readerModel.Update(msg)
			m.readerModel = &updated
			return m, cmd
		}
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
		content = m.libraryModel.View()
	case ViewReader:
		if m.readerModel != nil {
			content = m.readerModel.View()
		} else {
			content = m.renderPlaceholderView("Reader", "No manga loaded")
		}
	case ViewHistory:
		content = m.historyModel.View()
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

// Messages

// OpenReaderMsg is sent when we want to open the reader
type OpenReaderMsg struct {
	Manga   *source.Manga
	Chapter *source.Chapter
}
