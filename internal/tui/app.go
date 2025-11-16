package tui

import (
	"fmt"

	"github.com/Justice-Caban/Miryokusha/internal/config"
	"github.com/Justice-Caban/Miryokusha/internal/downloads"
	"github.com/Justice-Caban/Miryokusha/internal/server"
	"github.com/Justice-Caban/Miryokusha/internal/source"
	"github.com/Justice-Caban/Miryokusha/internal/storage"
	"github.com/Justice-Caban/Miryokusha/internal/suwayomi"
	"github.com/Justice-Caban/Miryokusha/internal/tui/categories"
	tuiDownloads "github.com/Justice-Caban/Miryokusha/internal/tui/downloads"
	"github.com/Justice-Caban/Miryokusha/internal/tui/extensions"
	"github.com/Justice-Caban/Miryokusha/internal/tui/history"
	"github.com/Justice-Caban/Miryokusha/internal/tui/library"
	"github.com/Justice-Caban/Miryokusha/internal/tui/reader"
	"github.com/Justice-Caban/Miryokusha/internal/tui/settings"
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
	ViewCategories ViewType = "categories"
)

// AppModel is the root model for the entire TUI application
type AppModel struct {
	currentView ViewType
	ready       bool
	width       int
	height      int
	err         error

	// Dependencies
	config          *config.Config
	sourceManager   *source.SourceManager
	storage         *storage.Storage
	downloadManager *downloads.Manager
	serverManager   *server.Manager

	// View models
	libraryModel     library.Model
	historyModel     history.Model
	extensionsModel  extensions.Model
	downloadsModel   tuiDownloads.Model
	settingsModel    settings.Model
	categoriesModel  categories.Model
	readerModel      *reader.Model

	// Suwayomi client
	suwayomiClient *suwayomi.Client
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

	// Initialize Suwayomi client if server configured
	var suwayomiClient *suwayomi.Client
	if defaultServer := cfg.GetDefaultServer(); defaultServer != nil {
		suwayomiClient = suwayomi.NewClient(defaultServer.URL)
	}

	// Initialize library model
	libModel := library.NewModel(sm, st)

	// Initialize history model
	histModel := history.NewModel(sm, st)

	// Initialize extensions model
	extModel := extensions.NewModel(suwayomiClient)

	// Initialize download manager
	downloadConfig := downloads.DefaultDownloadConfig()
	downloadMgr := downloads.NewManager(downloadConfig, sm)
	downloadMgr.Start() // Auto-start the download manager

	// Initialize downloads model
	dlModel := tuiDownloads.NewModel(downloadMgr)

	// Initialize server manager if enabled
	var serverMgr *server.Manager
	if cfg.ServerManagement.Enabled {
		serverConfig := &server.ManagerConfig{
			ExecutablePath: cfg.ServerManagement.ExecutablePath,
			Args:           cfg.ServerManagement.Args,
			WorkDir:        cfg.ServerManagement.WorkDir,
			MaxLogs:        1000,
		}
		serverMgr = server.NewManager(serverConfig)

		// Auto-start if configured
		if cfg.ServerManagement.AutoStart {
			_ = serverMgr.Start()
		}
	}

	// Initialize settings model
	settingsModel := settings.NewModel(cfg, suwayomiClient, serverMgr)

	// Initialize categories model
	categoriesModel := categories.NewModel(st)

	return AppModel{
		currentView:      ViewHome,
		config:           cfg,
		sourceManager:    sm,
		storage:          st,
		downloadManager:  downloadMgr,
		serverManager:    serverMgr,
		suwayomiClient:   suwayomiClient,
		libraryModel:     libModel,
		historyModel:     histModel,
		extensionsModel:  extModel,
		downloadsModel:   dlModel,
		settingsModel:    settingsModel,
		categoriesModel:  categoriesModel,
	}
}

// Init initializes the application
func (m AppModel) Init() tea.Cmd {
	return nil
}

// navigateToView handles navigation to a specific view from home
func (m AppModel) navigateToView(view ViewType) (AppModel, tea.Cmd) {
	if m.currentView != ViewHome {
		return m, nil
	}

	m.currentView = view
	sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}

	var cmd tea.Cmd
	switch view {
	case ViewLibrary:
		m.libraryModel, cmd = m.libraryModel.Update(sizeMsg)
	case ViewHistory:
		m.historyModel, cmd = m.historyModel.Update(sizeMsg)
	case ViewDownloads:
		m.downloadsModel, cmd = m.downloadsModel.Update(sizeMsg)
	case ViewExtensions:
		m.extensionsModel, cmd = m.extensionsModel.Update(sizeMsg)
	case ViewSettings:
		m.settingsModel, cmd = m.settingsModel.Update(sizeMsg)
	case ViewCategories:
		m.categoriesModel, cmd = m.categoriesModel.Update(sizeMsg)
	}

	return m, cmd
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
		// Try to fetch full details from source
		var manga *source.Manga
		var chapter *source.Chapter

		// Get source for this manga
		src := m.sourceManager.GetSource(msg.MangaID)
		if src == nil {
			// Try to find a source by type (assume Suwayomi for now)
			sources := m.sourceManager.GetSourcesByType(source.SourceTypeSuwayomi)
			if len(sources) > 0 {
				src = sources[0]
			}
		}

		if src != nil {
			// Fetch full manga details
			fetchedManga, err := src.GetManga(msg.MangaID)
			if err == nil && fetchedManga != nil {
				manga = fetchedManga
			}

			// Fetch chapters and find the specific one
			chapters, err := src.ListChapters(msg.MangaID)
			if err == nil {
				for _, ch := range chapters {
					if ch.ID == msg.ChapterID {
						chapter = ch
						break
					}
				}
			}
		}

		// Fallback to minimal objects if fetch failed
		if manga == nil {
			manga = &source.Manga{
				ID:         msg.MangaID,
				Title:      msg.MangaTitle, // Use cached title from history
				SourceType: source.SourceTypeSuwayomi,
			}
		}
		if chapter == nil {
			chapter = &source.Chapter{
				ID:      msg.ChapterID,
				MangaID: msg.MangaID,
			}
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
			return m.navigateToView(ViewHome)
		case "2":
			return m.navigateToView(ViewLibrary)
		case "3":
			return m.navigateToView(ViewHistory)
		case "4":
			return m.navigateToView(ViewBrowse)
		case "5":
			return m.navigateToView(ViewDownloads)
		case "6":
			return m.navigateToView(ViewExtensions)
		case "7":
			return m.navigateToView(ViewSettings)
		case "8":
			return m.navigateToView(ViewCategories)
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

	case ViewDownloads:
		m.downloadsModel, cmd = m.downloadsModel.Update(msg)
		return m, cmd

	case ViewExtensions:
		m.extensionsModel, cmd = m.extensionsModel.Update(msg)
		return m, cmd

	case ViewSettings:
		m.settingsModel, cmd = m.settingsModel.Update(msg)
		return m, cmd

	case ViewCategories:
		m.categoriesModel, cmd = m.categoriesModel.Update(msg)
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
		content = m.downloadsModel.View()
	case ViewExtensions:
		content = m.extensionsModel.View()
	case ViewSettings:
		content = m.settingsModel.View()
	case ViewCategories:
		content = m.categoriesModel.View()
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
  8 - Categories

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

	// Server status
	serverStatus := "Server: "
	if m.suwayomiClient != nil && m.suwayomiClient.Ping() {
		serverStatus += lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Render("✓ Connected")
	} else {
		serverStatus += lipgloss.NewStyle().
			Foreground(ColorMuted).
			Render("○ Not Connected")
	}

	help := "Press ? for help"

	return GetStatusBarText(viewName, serverStatus, dimensions, help)
}

// Messages

// OpenReaderMsg is sent when we want to open the reader
type OpenReaderMsg struct {
	Manga   *source.Manga
	Chapter *source.Chapter
}
