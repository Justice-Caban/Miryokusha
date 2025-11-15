package reader

import (
	"github.com/Justice-Caban/Miryokusha/internal/tui/theme"
	"fmt"
	"strings"
	"time"

	"github.com/Justice-Caban/Miryokusha/internal/source"
	"github.com/Justice-Caban/Miryokusha/internal/storage"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ReadingMode represents how pages are displayed
type ReadingMode int

const (
	ModeSinglePage ReadingMode = iota
	ModeDoublePage
	ModeWebtoon
)


// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.ColorPrimary).
			MarginBottom(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			MarginTop(1)

	pageInfoStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Bold(true)

	chapterInfoStyle = lipgloss.NewStyle().
				Foreground(theme.ColorSecondary)
)

// Model represents the manga reader model
type Model struct {
	width  int
	height int

	// Reading state
	manga          *source.Manga
	chapter        *source.Chapter
	chapters       []*source.Chapter
	currentPage    int
	pages          []*source.Page
	mode           ReadingMode
	showControls   bool
	bookmarked     bool

	// Navigation
	chapterIndex int

	// Reading session tracking
	sessionStart time.Time
	pagesRead    int

	// Dependencies
	sourceManager *source.SourceManager
	storage       *storage.Storage

	// Loading state
	loading bool
	err     error
}

// NewModel creates a new reader model
func NewModel(manga *source.Manga, chapter *source.Chapter, sm *source.SourceManager, st *storage.Storage) Model {
	return Model{
		manga:         manga,
		chapter:       chapter,
		currentPage:   0,
		mode:          ModeSinglePage,
		showControls:  true,
		sourceManager: sm,
		storage:       st,
		sessionStart:  time.Now(),
		pagesRead:     0,
		loading:       true,
	}
}

// Init initializes the reader
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadChapter,
		m.loadProgress,
	)
}

// Update handles messages for the reader view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case chapterLoadedMsg:
		m.pages = msg.pages
		m.chapters = msg.chapters
		m.chapterIndex = msg.chapterIndex
		m.loading = false
		return m, nil

	case progressLoadedMsg:
		if msg.progress != nil && !msg.progress.IsCompleted {
			m.currentPage = msg.progress.CurrentPage
		}
		return m, nil

	case chapterErrorMsg:
		m.err = msg.err
		m.loading = false
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h", "pgup":
		return m.prevPage()

	case "right", "l", "pgdn", " ":
		return m.nextPage()

	case "g":
		// Go to first page
		m.currentPage = 0
		return m, m.saveProgress

	case "G":
		// Go to last page
		if len(m.pages) > 0 {
			m.currentPage = len(m.pages) - 1
		}
		return m, m.saveProgress

	case "b":
		// Toggle bookmark
		return m.toggleBookmark()

	case "m":
		// Cycle reading mode
		m.mode = (m.mode + 1) % 3
		return m, nil

	case "c":
		// Toggle controls visibility
		m.showControls = !m.showControls
		return m, nil

	case "n":
		// Next chapter
		return m.nextChapter()

	case "p":
		// Previous chapter
		return m.prevChapter()

	case "[":
		// Jump back 10 pages
		m.currentPage -= 10
		if m.currentPage < 0 {
			m.currentPage = 0
		}
		return m, m.saveProgress

	case "]":
		// Jump forward 10 pages
		m.currentPage += 10
		if m.currentPage >= len(m.pages) {
			m.currentPage = len(m.pages) - 1
		}
		return m, m.saveProgress
	}

	return m, nil
}

// nextPage moves to the next page
func (m Model) nextPage() (Model, tea.Cmd) {
	if m.currentPage < len(m.pages)-1 {
		m.currentPage++
		m.pagesRead++
		return m, m.saveProgress
	}

	// At end of chapter, offer to go to next chapter
	if m.chapterIndex < len(m.chapters)-1 {
		// Auto-advance to next chapter
		return m.nextChapter()
	}

	return m, nil
}

// prevPage moves to the previous page
func (m Model) prevPage() (Model, tea.Cmd) {
	if m.currentPage > 0 {
		m.currentPage--
		return m, m.saveProgress
	}

	// At beginning of chapter, offer to go to previous chapter
	if m.chapterIndex > 0 {
		// Go to previous chapter (last page)
		return m.prevChapter()
	}

	return m, nil
}

// nextChapter advances to the next chapter
func (m Model) nextChapter() (Model, tea.Cmd) {
	if m.chapterIndex >= len(m.chapters)-1 {
		return m, nil
	}

	// Save session for current chapter
	m.SaveSession()

	m.chapterIndex++
	m.chapter = m.chapters[m.chapterIndex]
	m.currentPage = 0
	m.loading = true
	m.sessionStart = time.Now()
	m.pagesRead = 0

	return m, m.loadChapter
}

// prevChapter goes back to the previous chapter
func (m Model) prevChapter() (Model, tea.Cmd) {
	if m.chapterIndex <= 0 {
		return m, nil
	}

	// Save session for current chapter
	m.SaveSession()

	m.chapterIndex--
	m.chapter = m.chapters[m.chapterIndex]
	m.loading = true
	m.sessionStart = time.Now()
	m.pagesRead = 0

	return m, tea.Batch(
		m.loadChapter,
		func() tea.Msg {
			// Load to last page of previous chapter
			return gotoLastPageMsg{}
		},
	)
}

// toggleBookmark toggles the bookmark for the current page
func (m Model) toggleBookmark() (Model, tea.Cmd) {
	if m.storage == nil {
		return m, nil
	}

	if m.bookmarked {
		// Remove bookmark (would need to track bookmark ID)
		m.bookmarked = false
	} else {
		// Add bookmark
		bookmark := &storage.Bookmark{
			MangaID:       m.manga.ID,
			MangaTitle:    m.manga.Title,
			ChapterID:     m.chapter.ID,
			ChapterNumber: m.chapter.ChapterNumber,
			ChapterTitle:  m.chapter.Title,
			PageNumber:    m.currentPage,
			Note:          "",
		}
		m.storage.Bookmarks.AddBookmark(bookmark)
		m.bookmarked = true
	}

	return m, nil
}

// View renders the reader view
func (m Model) View() string {
	if m.loading {
		return centeredText(m.width, m.height, "Loading chapter...")
	}

	if m.err != nil {
		return centeredText(m.width, m.height, fmt.Sprintf("Error: %v\n\nPress ESC to go back", m.err))
	}

	if len(m.pages) == 0 {
		return centeredText(m.width, m.height, "No pages available\n\nPress ESC to go back")
	}

	var b strings.Builder

	// Header with chapter info
	if m.showControls {
		b.WriteString(m.renderHeader())
		b.WriteString("\n\n")
	}

	// Page content
	b.WriteString(m.renderPage())
	b.WriteString("\n")

	// Footer with controls
	if m.showControls {
		b.WriteString("\n")
		b.WriteString(m.renderFooter())
	}

	return b.String()
}

// renderHeader renders the chapter and page information
func (m Model) renderHeader() string {
	title := chapterInfoStyle.Render(m.manga.Title)
	chapterInfo := fmt.Sprintf("Chapter %.1f", m.chapter.ChapterNumber)
	if m.chapter.Title != "" {
		chapterInfo += fmt.Sprintf(": %s", m.chapter.Title)
	}
	chapter := chapterInfoStyle.Render(chapterInfo)

	pageInfo := pageInfoStyle.Render(fmt.Sprintf("Page %d / %d", m.currentPage+1, len(m.pages)))

	// Reading mode indicator
	modeStr := ""
	switch m.mode {
	case ModeSinglePage:
		modeStr = "Single"
	case ModeDoublePage:
		modeStr = "Double"
	case ModeWebtoon:
		modeStr = "Webtoon"
	}
	mode := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(fmt.Sprintf("[%s]", modeStr))

	// Bookmark indicator
	bookmarkIndicator := ""
	if m.bookmarked {
		bookmarkIndicator = lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render(" ★")
	}

	firstLine := lipgloss.JoinHorizontal(lipgloss.Left, title, "  ", pageInfo, bookmarkIndicator)
	secondLine := lipgloss.JoinHorizontal(lipgloss.Left, chapter, "  ", mode)

	return firstLine + "\n" + secondLine
}

// renderPage renders the current page(s)
func (m Model) renderPage() string {
	switch m.mode {
	case ModeSinglePage:
		return m.renderSinglePage()
	case ModeDoublePage:
		return m.renderDoublePage()
	case ModeWebtoon:
		return m.renderWebtoon()
	default:
		return m.renderSinglePage()
	}
}

// renderSinglePage renders a single page
func (m Model) renderSinglePage() string {
	if m.currentPage >= len(m.pages) {
		return centeredText(m.width, m.height-10, "Invalid page")
	}

	page := m.pages[m.currentPage]

	// For now, show a placeholder since we don't have actual image rendering
	// In a real implementation, this would use a library to display the image
	placeholder := fmt.Sprintf(
		"[Page %d]\n\n"+
			"Image: %s\n\n"+
			"(Image rendering not yet implemented)\n\n"+
			"Use ← → or h l to navigate\n"+
			"Press 'c' to hide controls",
		m.currentPage+1,
		page.URL,
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorMuted).
		Padding(2, 4).
		Width(m.width - 4).
		Height(m.height - 15).
		Align(lipgloss.Center, lipgloss.Center)

	return boxStyle.Render(placeholder)
}

// renderDoublePage renders two pages side by side
func (m Model) renderDoublePage() string {
	// For double page mode, show current and next page
	leftPage := m.currentPage
	rightPage := m.currentPage + 1

	if leftPage >= len(m.pages) {
		return centeredText(m.width, m.height-10, "Invalid page")
	}

	leftPlaceholder := fmt.Sprintf("[Page %d]", leftPage+1)
	rightPlaceholder := ""

	if rightPage < len(m.pages) {
		rightPlaceholder = fmt.Sprintf("[Page %d]", rightPage+1)
	} else {
		rightPlaceholder = "[End]"
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorMuted).
		Padding(2, 2).
		Width((m.width / 2) - 4).
		Height(m.height - 15).
		Align(lipgloss.Center, lipgloss.Center)

	left := boxStyle.Render(leftPlaceholder)
	right := boxStyle.Render(rightPlaceholder)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// renderWebtoon renders in continuous scroll mode (showing multiple pages)
func (m Model) renderWebtoon() string {
	// In webtoon mode, show current page and hint at continuous scroll
	placeholder := fmt.Sprintf(
		"[Webtoon Mode]\n\n"+
			"Page %d / %d\n\n"+
			"(Continuous scroll rendering not yet implemented)\n\n"+
			"Use ← → to navigate pages",
		m.currentPage+1,
		len(m.pages),
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.ColorMuted).
		Padding(2, 4).
		Width(m.width - 4).
		Height(m.height - 15).
		Align(lipgloss.Center, lipgloss.Center)

	return boxStyle.Render(placeholder)
}

// renderFooter renders the footer with controls
func (m Model) renderFooter() string {
	controls := []string{
		"←→/hl/Space: navigate",
		"[]: jump 10 pages",
		"g/G: first/last",
		"n/p: next/prev chapter",
		"b: bookmark",
		"m: mode",
		"c: hide controls",
		"Esc: back",
	}

	// Progress indicator
	progressBar := m.renderProgressBar()

	return helpStyle.Render(strings.Join(controls, " • ")) + "\n" + progressBar
}

// renderProgressBar renders a progress bar
func (m Model) renderProgressBar() string {
	barWidth := m.width - 20
	if barWidth < 10 {
		barWidth = 10
	}

	progressPct := float64(m.currentPage+1) / float64(len(m.pages))
	filled := int(progressPct * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	label := fmt.Sprintf("%.0f%%", progressPct*100)

	progressStyle := lipgloss.NewStyle().Foreground(theme.ColorAccent)

	return lipgloss.JoinHorizontal(lipgloss.Left,
		progressStyle.Render(bar),
		" ",
		lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(label),
	)
}

// centeredText centers text in the given width and height
func centeredText(width, height int, text string) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)
	return style.Render(text)
}

// SaveSession saves the reading session
func (m *Model) SaveSession() {
	if m.storage == nil || m.chapter == nil {
		return
	}

	duration := time.Since(m.sessionStart)

	session := &storage.ReadingSession{
		MangaID:         m.manga.ID,
		ChapterID:       m.chapter.ID,
		DurationSeconds: int(duration.Seconds()),
		PagesRead:       m.pagesRead,
		SessionStart:    m.sessionStart,
		SessionEnd:      time.Now(),
	}

	m.storage.Stats.RecordSession(session)

	// Also add to history
	historyEntry := &storage.HistoryEntry{
		MangaID:       m.manga.ID,
		MangaTitle:    m.manga.Title,
		ChapterID:     m.chapter.ID,
		ChapterNumber: m.chapter.ChapterNumber,
		ChapterTitle:  m.chapter.Title,
		SourceType:    string(m.manga.SourceType),
		SourceID:      m.manga.SourceID,
		ReadAt:        time.Now(),
	}

	m.storage.History.AddHistoryEntry(historyEntry)
}

// Messages

type chapterLoadedMsg struct {
	pages        []*source.Page
	chapters     []*source.Chapter
	chapterIndex int
}

type progressLoadedMsg struct {
	progress *storage.ProgressEntry
}

type chapterErrorMsg struct {
	err error
}

type gotoLastPageMsg struct{}

// Commands

func (m Model) loadChapter() tea.Msg {
	if m.sourceManager == nil || m.manga == nil || m.chapter == nil {
		return chapterErrorMsg{err: fmt.Errorf("missing required data")}
	}

	// Get the source for this manga
	src := m.sourceManager.GetSource(m.manga.SourceID)
	if src == nil {
		return chapterErrorMsg{err: fmt.Errorf("source not found")}
	}

	// Load all chapters for navigation
	chapters, err := src.ListChapters(m.manga.ID)
	if err != nil {
		return chapterErrorMsg{err: fmt.Errorf("failed to load chapters: %w", err)}
	}

	// Find current chapter index
	chapterIndex := -1
	for i, ch := range chapters {
		if ch.ID == m.chapter.ID {
			chapterIndex = i
			break
		}
	}

	if chapterIndex == -1 {
		return chapterErrorMsg{err: fmt.Errorf("current chapter not found in chapter list")}
	}

	// Load pages for current chapter
	pages, err := src.GetAllPages(m.chapter.ID)
	if err != nil {
		return chapterErrorMsg{err: fmt.Errorf("failed to load pages: %w", err)}
	}

	return chapterLoadedMsg{
		pages:        pages,
		chapters:     chapters,
		chapterIndex: chapterIndex,
	}
}

func (m Model) loadProgress() tea.Msg {
	if m.storage == nil || m.manga == nil || m.chapter == nil {
		return progressLoadedMsg{progress: nil}
	}

	progress, err := m.storage.Progress.GetProgress(m.manga.ID, m.chapter.ID)
	if err != nil {
		return progressLoadedMsg{progress: nil}
	}

	return progressLoadedMsg{progress: progress}
}

func (m Model) saveProgress() tea.Msg {
	if m.storage == nil || m.manga == nil || m.chapter == nil {
		return nil
	}

	err := m.storage.Progress.UpdateProgress(
		m.manga.ID,
		m.chapter.ID,
		m.currentPage,
		len(m.pages),
	)

	if err != nil {
		// Log error but don't block
		return nil
	}

	return nil
}
