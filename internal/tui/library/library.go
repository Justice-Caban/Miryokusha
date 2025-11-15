package library

import (
	"fmt"
	"strings"

	"github.com/Justice-Caban/Miryokusha/internal/source"
	"github.com/Justice-Caban/Miryokusha/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Color palette (duplicated from tui package to avoid import cycle)
var (
	colorPrimary   = lipgloss.Color("205") // Pink
	colorSecondary = lipgloss.Color("99")  // Purple
	colorAccent    = lipgloss.Color("86")  // Cyan
	colorMuted     = lipgloss.Color("242") // Gray
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
)

// centeredText centers text in the given width and height
func centeredText(width, height int, text string) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)
	return style.Render(text)
}

// SortMode represents how the library is sorted
type SortMode int

const (
	SortAlphabetical SortMode = iota
	SortLastRead
	SortUnreadCount
	SortDateAdded
)

// FilterMode represents how the library is filtered
type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterReading
	FilterCompleted
	FilterUnread
)

// Model represents the library view model
type Model struct {
	width  int
	height int

	// Data
	manga        []*source.Manga
	filteredList []*source.Manga
	readHistory  map[string]bool // manga ID -> has been read

	// UI state
	cursor       int
	offset       int
	sortMode     SortMode
	filterMode   FilterMode
	searchQuery  string
	searchActive bool

	// Dependencies
	sourceManager *source.SourceManager
	storage       *storage.Storage

	// Loading state
	loading bool
	err     error
}

// NewModel creates a new library model
func NewModel(sm *source.SourceManager, st *storage.Storage) Model {
	return Model{
		manga:         make([]*source.Manga, 0),
		filteredList:  make([]*source.Manga, 0),
		readHistory:   make(map[string]bool),
		cursor:        0,
		offset:        0,
		sortMode:      SortAlphabetical,
		filterMode:    FilterAll,
		sourceManager: sm,
		storage:       st,
		loading:       false,
	}
}

// Init initializes the library model
func (m Model) Init() tea.Cmd {
	return m.loadLibrary
}

// Update handles messages for the library view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.searchActive {
			return m.handleSearchInput(msg)
		}
		return m.handleKeyPress(msg)

	case libraryLoadedMsg:
		m.manga = msg.manga
		m.readHistory = msg.readHistory
		m.loading = false
		m.applyFiltersAndSort()
		return m, nil

	case libraryErrorMsg:
		m.err = msg.err
		m.loading = false
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.adjustOffset()
		}

	case "down", "j":
		if m.cursor < len(m.filteredList)-1 {
			m.cursor++
			m.adjustOffset()
		}

	case "g":
		// Go to top
		m.cursor = 0
		m.offset = 0

	case "G":
		// Go to bottom
		if len(m.filteredList) > 0 {
			m.cursor = len(m.filteredList) - 1
			m.adjustOffset()
		}

	case "r":
		// Refresh library
		m.loading = true
		return m, m.loadLibrary

	case "/":
		// Start search
		m.searchActive = true
		m.searchQuery = ""

	case "s":
		// Cycle sort mode
		m.sortMode = (m.sortMode + 1) % 4
		m.applyFiltersAndSort()

	case "f":
		// Cycle filter mode
		m.filterMode = (m.filterMode + 1) % 4
		m.applyFiltersAndSort()
	}

	return m, nil
}

// handleSearchInput handles search input
func (m Model) handleSearchInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchActive = false
		m.searchQuery = ""
		m.applyFiltersAndSort()

	case "enter":
		m.searchActive = false
		m.applyFiltersAndSort()

	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}

	default:
		// Add character to search query
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
		}
	}

	return m, nil
}

// adjustOffset adjusts the scroll offset to keep cursor visible
func (m *Model) adjustOffset() {
	visibleItems := m.height - 10 // Account for header and footer
	if visibleItems < 1 {
		visibleItems = 1
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+visibleItems {
		m.offset = m.cursor - visibleItems + 1
	}
}

// applyFiltersAndSort applies current filters and sorting
func (m *Model) applyFiltersAndSort() {
	// Start with all manga
	m.filteredList = make([]*source.Manga, 0, len(m.manga))

	for _, manga := range m.manga {
		// Apply search filter
		if m.searchQuery != "" {
			if !strings.Contains(strings.ToLower(manga.Title), strings.ToLower(m.searchQuery)) {
				continue
			}
		}

		// Apply read filter
		switch m.filterMode {
		case FilterReading:
			if !m.readHistory[manga.ID] {
				continue
			}
		case FilterUnread:
			if m.readHistory[manga.ID] {
				continue
			}
		}

		m.filteredList = append(m.filteredList, manga)
	}

	// Apply sorting
	switch m.sortMode {
	case SortAlphabetical:
		m.sortAlphabetically()
	// TODO: Implement other sort modes when we have more metadata
	}

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filteredList) {
		m.cursor = len(m.filteredList) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
}

// sortAlphabetically sorts manga alphabetically by title
func (m *Model) sortAlphabetically() {
	// Simple bubble sort for now
	for i := 0; i < len(m.filteredList); i++ {
		for j := i + 1; j < len(m.filteredList); j++ {
			if m.filteredList[i].Title > m.filteredList[j].Title {
				m.filteredList[i], m.filteredList[j] = m.filteredList[j], m.filteredList[i]
			}
		}
	}
}

// View renders the library view
func (m Model) View() string {
	if m.loading {
		return centeredText(m.width, m.height, "Loading library...")
	}

	if m.err != nil {
		return centeredText(m.width, m.height, fmt.Sprintf("Error: %v", m.err))
	}

	if len(m.manga) == 0 {
		return centeredText(m.width, m.height, "No manga found\n\nPress 'r' to refresh")
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Manga list
	b.WriteString(m.renderMangaList())
	b.WriteString("\n")

	// Footer
	b.WriteString(m.renderFooter())

	return b.String()
}

// renderHeader renders the library header
func (m Model) renderHeader() string {
	title := titleStyle.Render("Library")

	// Sort mode indicator
	sortModeStr := ""
	switch m.sortMode {
	case SortAlphabetical:
		sortModeStr = "A-Z"
	case SortLastRead:
		sortModeStr = "Last Read"
	case SortUnreadCount:
		sortModeStr = "Unread"
	case SortDateAdded:
		sortModeStr = "Date Added"
	}

	// Filter mode indicator
	filterModeStr := ""
	switch m.filterMode {
	case FilterAll:
		filterModeStr = "All"
	case FilterReading:
		filterModeStr = "Reading"
	case FilterCompleted:
		filterModeStr = "Completed"
	case FilterUnread:
		filterModeStr = "Unread"
	}

	info := lipgloss.NewStyle().
		Foreground(colorSecondary).
		Render(fmt.Sprintf("Sort: %s | Filter: %s | %d manga", sortModeStr, filterModeStr, len(m.filteredList)))

	// Search bar
	searchBar := ""
	if m.searchActive {
		searchBar = "\n" + lipgloss.NewStyle().
			Foreground(colorAccent).
			Render(fmt.Sprintf("Search: %s_", m.searchQuery))
	} else if m.searchQuery != "" {
		searchBar = "\n" + lipgloss.NewStyle().
			Foreground(colorSecondary).
			Render(fmt.Sprintf("Search: %s", m.searchQuery))
	}

	return title + "\n" + info + searchBar
}

// renderMangaList renders the list of manga
func (m Model) renderMangaList() string {
	if len(m.filteredList) == 0 {
		return mutedStyle.Render("No manga match current filters")
	}

	var b strings.Builder

	visibleItems := m.height - 10
	if visibleItems < 1 {
		visibleItems = 1
	}

	start := m.offset
	end := m.offset + visibleItems
	if end > len(m.filteredList) {
		end = len(m.filteredList)
	}

	for i := start; i < end; i++ {
		manga := m.filteredList[i]
		isCursor := i == m.cursor

		// Item style
		itemStyle := lipgloss.NewStyle()
		if isCursor {
			itemStyle = itemStyle.
				Background(colorPrimary).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Width(m.width - 4)
		}

		// Read indicator
		readIndicator := "  "
		if m.readHistory[manga.ID] {
			readIndicator = "✓ "
		}

		// Source indicator
		sourceIndicator := ""
		switch manga.SourceType {
		case source.SourceTypeLocal:
			sourceIndicator = "[Local]"
		case source.SourceTypeSuwayomi:
			sourceIndicator = "[Server]"
		}

		// Format line
		line := fmt.Sprintf("%s %s %s",
			readIndicator,
			manga.Title,
			lipgloss.NewStyle().Foreground(colorMuted).Render(sourceIndicator),
		)

		b.WriteString(itemStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

// renderFooter renders the footer with controls
func (m Model) renderFooter() string {
	controls := []string{
		"↑↓/jk: navigate",
		"g/G: top/bottom",
		"r: refresh",
		"/: search",
		"s: sort",
		"f: filter",
		"Enter: open",
		"Esc: back",
	}

	return helpStyle.Render(strings.Join(controls, " • "))
}

// Messages

type libraryLoadedMsg struct {
	manga       []*source.Manga
	readHistory map[string]bool
}

type libraryErrorMsg struct {
	err error
}

// Commands

func (m Model) loadLibrary() tea.Msg {
	// Load manga from all sources
	manga, err := m.sourceManager.ListAllManga()
	if err != nil {
		return libraryErrorMsg{err: err}
	}

	// Load read history
	readHistory := make(map[string]bool)
	if m.storage != nil {
		history, err := m.storage.History.GetRecentHistory(10000)
		if err == nil {
			for _, entry := range history {
				readHistory[entry.MangaID] = true
			}
		}
	}

	return libraryLoadedMsg{
		manga:       manga,
		readHistory: readHistory,
	}
}
