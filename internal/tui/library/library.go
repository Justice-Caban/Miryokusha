package library

import (
	"fmt"
	"strings"

	"github.com/Justice-Caban/Miryokusha/internal/source"
	"github.com/Justice-Caban/Miryokusha/internal/storage"
	"github.com/Justice-Caban/Miryokusha/internal/tui/kitty"
	"github.com/Justice-Caban/Miryokusha/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
	showImages   bool // Toggle for image display

	// Dependencies
	sourceManager *source.SourceManager
	storage       *storage.Storage
	imageRenderer *kitty.ImageRenderer

	// Loading state
	loading bool
	err     error
}

// NewModel creates a new library model
func NewModel(sm *source.SourceManager, st *storage.Storage, showImages bool) Model {
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
		imageRenderer: kitty.NewImageRenderer(),
		showImages:    showImages,
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

	case "i":
		// Toggle image display
		m.showImages = !m.showImages

	case "enter":
		// Open selected manga
		if m.cursor < len(m.filteredList) {
			selectedManga := m.filteredList[m.cursor]
			return m, m.openManga(selectedManga)
		}
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
	case SortLastRead:
		m.sortByLastRead()
	case SortUnreadCount:
		m.sortByUnreadCount()
	case SortDateAdded:
		m.sortByDateAdded()
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

// sortByLastRead sorts manga by last read time (most recent first)
func (m *Model) sortByLastRead() {
	for i := 0; i < len(m.filteredList); i++ {
		for j := i + 1; j < len(m.filteredList); j++ {
			iTime := m.filteredList[i].LastReadAt
			jTime := m.filteredList[j].LastReadAt

			// nil times go to the end
			if iTime == nil && jTime != nil {
				m.filteredList[i], m.filteredList[j] = m.filteredList[j], m.filteredList[i]
			} else if iTime != nil && jTime != nil && iTime.Before(*jTime) {
				m.filteredList[i], m.filteredList[j] = m.filteredList[j], m.filteredList[i]
			}
		}
	}
}

// sortByUnreadCount sorts manga by unread chapter count (most unread first)
func (m *Model) sortByUnreadCount() {
	for i := 0; i < len(m.filteredList); i++ {
		for j := i + 1; j < len(m.filteredList); j++ {
			if m.filteredList[i].UnreadCount < m.filteredList[j].UnreadCount {
				m.filteredList[i], m.filteredList[j] = m.filteredList[j], m.filteredList[i]
			}
		}
	}
}

// sortByDateAdded sorts manga by when they were added to library (newest first)
// Note: This requires tracking when manga was added - for now, use ID as proxy
func (m *Model) sortByDateAdded() {
	// For manga from Suwayomi, higher IDs are typically newer additions
	// For local manga, sort alphabetically as fallback
	for i := 0; i < len(m.filteredList); i++ {
		for j := i + 1; j < len(m.filteredList); j++ {
			// Try to sort by ID descending (newer first)
			if m.filteredList[i].ID < m.filteredList[j].ID {
				m.filteredList[i], m.filteredList[j] = m.filteredList[j], m.filteredList[i]
			}
		}
	}
}

// View renders the library view
func (m Model) View() string {
	if m.loading {
		return theme.CenteredText(m.width, m.height, "Loading library...")
	}

	if m.err != nil {
		return theme.CenteredText(m.width, m.height, fmt.Sprintf("Error: %v", m.err))
	}

	if len(m.manga) == 0 {
		return theme.CenteredText(m.width, m.height, "No manga found\n\nPress 'r' to refresh")
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

	// Apply consistent horizontal padding/centering
	content := b.String()
	maxWidth := 120
	if m.width < maxWidth {
		maxWidth = m.width - 4
	}

	contentStyle := lipgloss.NewStyle().
		Width(maxWidth).
		Padding(0, 2)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Top,
		contentStyle.Render(content),
	)
}

// renderHeader renders the library header
func (m Model) renderHeader() string {
	title := theme.TitleStyle.Render("Library")

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
		Foreground(theme.ColorSecondary).
		Render(fmt.Sprintf("Sort: %s | Filter: %s | %d manga", sortModeStr, filterModeStr, len(m.filteredList)))

	// Search bar
	searchBar := ""
	if m.searchActive {
		searchBar = "\n" + lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Render(fmt.Sprintf("Search: %s_", m.searchQuery))
	} else if m.searchQuery != "" {
		searchBar = "\n" + lipgloss.NewStyle().
			Foreground(theme.ColorSecondary).
			Render(fmt.Sprintf("Search: %s", m.searchQuery))
	}

	return title + "\n" + info + searchBar
}

// renderMangaList renders the list of manga
func (m Model) renderMangaList() string {
	if len(m.filteredList) == 0 {
		return theme.MutedStyle.Render("No manga match current filters")
	}

	var b strings.Builder

	// Adjust visible items based on whether images are shown
	itemHeight := 1
	if m.showImages {
		itemHeight = 6 // Each item takes ~6 rows when showing images
	}

	visibleItems := (m.height - 10) / itemHeight
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

		if m.showImages && manga.CoverURL != "" {
			// Render with image
			b.WriteString(m.renderMangaItemWithImage(manga, isCursor, uint32(i+1)))
		} else {
			// Render text-only
			b.WriteString(m.renderMangaItemTextOnly(manga, isCursor))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderMangaItemWithImage renders a manga item with cover image
func (m Model) renderMangaItemWithImage(manga *source.Manga, isCursor bool, imageID uint32) string {
	var parts []string

	// Try to render the image
	imageOpts := kitty.ImageOptions{
		Width:               5,
		Height:              5,
		PreserveAspectRatio: true,
		ImageID:             imageID,
	}

	imageStr, err := m.imageRenderer.RenderImageFromURL(manga.CoverURL, imageOpts)
	if err != nil {
		// Fallback to placeholder if image fails
		imageStr = kitty.CreatePlaceholder(5, 5, "[IMG]")
	}

	// Build manga info text
	var infoLines []string

	// Read indicator
	readIndicator := "  "
	if m.readHistory[manga.ID] {
		readIndicator = "✓ "
	}

	// Title line
	titleStyle := lipgloss.NewStyle()
	if isCursor {
		titleStyle = titleStyle.
			Foreground(theme.ColorPrimary).
			Bold(true)
	}
	infoLines = append(infoLines, titleStyle.Render(readIndicator+manga.Title))

	// Author if available
	if manga.Author != "" {
		infoLines = append(infoLines, lipgloss.NewStyle().
			Foreground(theme.ColorSecondary).
			Render("  by "+manga.Author))
	}

	// Source indicator
	sourceIndicator := ""
	switch manga.SourceType {
	case source.SourceTypeLocal:
		sourceIndicator = "[Local]"
	case source.SourceTypeSuwayomi:
		sourceIndicator = "[Server]"
	}
	infoLines = append(infoLines, lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Render("  "+sourceIndicator))

	// Unread count if > 0
	if manga.UnreadCount > 0 {
		infoLines = append(infoLines, lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Render(fmt.Sprintf("  %d unread", manga.UnreadCount)))
	}

	// Join info lines
	infoText := strings.Join(infoLines, "\n")

	// Position image and text side by side
	parts = append(parts, imageStr)
	parts = append(parts, infoText)

	// Join horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderMangaItemTextOnly renders a manga item without images (compact view)
func (m Model) renderMangaItemTextOnly(manga *source.Manga, isCursor bool) string {
	// Item style
	itemStyle := lipgloss.NewStyle()
	if isCursor {
		itemStyle = itemStyle.
			Background(theme.ColorPrimary).
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
		lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(sourceIndicator),
	)

	return itemStyle.Render(line)
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
		"i: toggle images",
		"Enter: open",
		"Esc: back",
	}

	return theme.HelpStyle.Render(strings.Join(controls, " • "))
}

// Messages

type libraryLoadedMsg struct {
	manga       []*source.Manga
	readHistory map[string]bool
}

type libraryErrorMsg struct {
	err error
}

// OpenMangaMsg is sent when a manga should be opened
type OpenMangaMsg struct {
	Manga *source.Manga
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

func (m Model) openManga(manga *source.Manga) tea.Cmd {
	return func() tea.Msg {
		return OpenMangaMsg{
			Manga: manga,
		}
	}
}
