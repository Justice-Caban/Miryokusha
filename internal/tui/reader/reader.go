package reader

import (
	"fmt"
	"strings"
	"time"

	"github.com/Justice-Caban/Miryokusha/internal/source"
	"github.com/Justice-Caban/Miryokusha/internal/storage"
	"github.com/Justice-Caban/Miryokusha/internal/tui/kitty"
	"github.com/Justice-Caban/Miryokusha/internal/tui/theme"
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
	imageRenderer *kitty.ImageRenderer

	// Loading state
	loading bool
	err     error
	warning string // Non-critical warnings (e.g., progress tracking disabled)
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
		imageRenderer: kitty.NewImageRenderer(),
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
		if msg.err != nil {
			m.warning = fmt.Sprintf("Progress tracking unavailable: %v", msg.err)
		} else if msg.progress != nil && !msg.progress.IsCompleted {
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
		return theme.CenteredText(m.width, m.height, "Loading chapter...")
	}

	if m.err != nil {
		return theme.CenteredText(m.width, m.height, fmt.Sprintf("Error: %v\n\nPress ESC to go back", m.err))
	}

	if len(m.pages) == 0 {
		return theme.CenteredText(m.width, m.height, "No pages available\n\nPress ESC to go back")
	}

	var b strings.Builder

	// Header with chapter info
	if m.showControls {
		b.WriteString(m.renderHeader())
		b.WriteString("\n\n")
	}

	// Warning banner (if any)
	if m.warning != "" {
		warningBanner := lipgloss.NewStyle().
			Foreground(theme.ColorWarning).
			Bold(true).
			Render("⚠ " + m.warning)
		b.WriteString(warningBanner)
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
	title := theme.MutedStyle.Render(m.manga.Title)
	chapterInfo := fmt.Sprintf("Chapter %.1f", m.chapter.ChapterNumber)
	if m.chapter.Title != "" {
		chapterInfo += fmt.Sprintf(": %s", m.chapter.Title)
	}
	chapter := theme.MutedStyle.Render(chapterInfo)

	pageInfo := theme.ValueStyle.Render(fmt.Sprintf("Page %d / %d", m.currentPage+1, len(m.pages)))

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
		return theme.CenteredText(m.width, m.height-10, "Invalid page")
	}

	page := m.pages[m.currentPage]

	// Calculate image dimensions based on terminal size
	// Reserve space for header (5 lines) and footer (3 lines) if controls are shown
	availableHeight := m.height
	if m.showControls {
		availableHeight -= 8
	}

	// Use most of the terminal width and height for the image
	imageWidth := m.width - 4
	imageHeight := availableHeight - 2

	// Convert terminal cells to approximate cells for Kitty protocol
	// Kitty measures in cells, so we use the terminal dimensions directly
	cellWidth := imageWidth / 10  // Approximate character width
	cellHeight := imageHeight / 2 // Approximate character height

	if cellWidth < 1 {
		cellWidth = 1
	}
	if cellHeight < 1 {
		cellHeight = 1
	}

	// Configure image options
	imageOpts := kitty.ImageOptions{
		Width:               cellWidth,
		Height:              cellHeight,
		PreserveAspectRatio: true,
		ImageID:             uint32(m.currentPage + 1000), // Offset to avoid conflicts
	}

	var imageStr string
	var err error

	// Check if we have image data already loaded
	if len(page.ImageData) > 0 {
		// Render from loaded data
		imageStr, err = m.imageRenderer.RenderImage(page.ImageData, imageOpts)
	} else if page.URL != "" {
		// Fetch and render from URL
		imageStr, err = m.imageRenderer.RenderImageFromURL(page.URL, imageOpts)
	} else {
		err = fmt.Errorf("no image data or URL available")
	}

	if err != nil {
		// Fallback to text placeholder on error
		placeholder := fmt.Sprintf(
			"[Page %d]\n\n"+
				"Failed to load image: %v\n\n"+
				"URL: %s\n\n"+
				"Use ← → or h l to navigate\n"+
				"Press 'c' to hide controls",
			m.currentPage+1,
			err,
			page.URL,
		)

		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.ColorError).
			Padding(2, 4).
			Width(imageWidth).
			Height(availableHeight).
			Align(lipgloss.Center, lipgloss.Center)

		return boxStyle.Render(placeholder)
	}

	// Center the image in the available space
	return lipgloss.Place(
		m.width,
		availableHeight,
		lipgloss.Center,
		lipgloss.Center,
		imageStr,
	)
}

// renderDoublePage renders two pages side by side
func (m Model) renderDoublePage() string {
	// For double page mode, show current and next page
	leftPageIdx := m.currentPage
	rightPageIdx := m.currentPage + 1

	if leftPageIdx >= len(m.pages) {
		return theme.CenteredText(m.width, m.height-10, "Invalid page")
	}

	// Calculate dimensions for side-by-side layout
	availableHeight := m.height
	if m.showControls {
		availableHeight -= 8
	}

	pageWidth := (m.width / 2) - 4
	cellWidth := pageWidth / 10
	cellHeight := (availableHeight - 2) / 2

	if cellWidth < 1 {
		cellWidth = 1
	}
	if cellHeight < 1 {
		cellHeight = 1
	}

	// Render left page
	leftPage := m.pages[leftPageIdx]
	leftOpts := kitty.ImageOptions{
		Width:               cellWidth,
		Height:              cellHeight,
		PreserveAspectRatio: true,
		ImageID:             uint32(leftPageIdx + 2000),
	}

	var leftImageStr string
	var leftErr error

	if len(leftPage.ImageData) > 0 {
		leftImageStr, leftErr = m.imageRenderer.RenderImage(leftPage.ImageData, leftOpts)
	} else if leftPage.URL != "" {
		leftImageStr, leftErr = m.imageRenderer.RenderImageFromURL(leftPage.URL, leftOpts)
	} else {
		leftErr = fmt.Errorf("no image data")
	}

	if leftErr != nil {
		leftImageStr = kitty.CreatePlaceholder(cellWidth, cellHeight, fmt.Sprintf("Page %d\nError", leftPageIdx+1))
	}

	// Render right page (if available)
	var rightImageStr string
	if rightPageIdx < len(m.pages) {
		rightPage := m.pages[rightPageIdx]
		rightOpts := kitty.ImageOptions{
			Width:               cellWidth,
			Height:              cellHeight,
			PreserveAspectRatio: true,
			ImageID:             uint32(rightPageIdx + 2000),
		}

		var rightErr error
		if len(rightPage.ImageData) > 0 {
			rightImageStr, rightErr = m.imageRenderer.RenderImage(rightPage.ImageData, rightOpts)
		} else if rightPage.URL != "" {
			rightImageStr, rightErr = m.imageRenderer.RenderImageFromURL(rightPage.URL, rightOpts)
		} else {
			rightErr = fmt.Errorf("no image data")
		}

		if rightErr != nil {
			rightImageStr = kitty.CreatePlaceholder(cellWidth, cellHeight, fmt.Sprintf("Page %d\nError", rightPageIdx+1))
		}
	} else {
		// End of chapter
		rightImageStr = kitty.CreatePlaceholder(cellWidth, cellHeight, "End of\nChapter")
	}

	// Join pages horizontally
	doublePage := lipgloss.JoinHorizontal(lipgloss.Top, leftImageStr, "  ", rightImageStr)

	return lipgloss.Place(
		m.width,
		availableHeight,
		lipgloss.Center,
		lipgloss.Center,
		doublePage,
	)
}

// renderWebtoon renders in continuous scroll mode (showing multiple pages)
func (m Model) renderWebtoon() string {
	if m.currentPage >= len(m.pages) {
		return theme.CenteredText(m.width, m.height-10, "Invalid page")
	}

	// In webtoon mode, show current page in full width
	// Future enhancement: show multiple pages vertically
	availableHeight := m.height
	if m.showControls {
		availableHeight -= 8
	}

	// Use full width for webtoon images (they're typically vertical)
	imageWidth := m.width - 4
	cellWidth := imageWidth / 8  // Slightly wider than single page mode
	cellHeight := (availableHeight - 2) / 2

	if cellWidth < 1 {
		cellWidth = 1
	}
	if cellHeight < 1 {
		cellHeight = 1
	}

	page := m.pages[m.currentPage]
	imageOpts := kitty.ImageOptions{
		Width:               cellWidth,
		Height:              cellHeight,
		PreserveAspectRatio: true,
		ImageID:             uint32(m.currentPage + 3000), // Offset for webtoon mode
	}

	var imageStr string
	var err error

	if len(page.ImageData) > 0 {
		imageStr, err = m.imageRenderer.RenderImage(page.ImageData, imageOpts)
	} else if page.URL != "" {
		imageStr, err = m.imageRenderer.RenderImageFromURL(page.URL, imageOpts)
	} else {
		err = fmt.Errorf("no image data or URL available")
	}

	if err != nil {
		placeholder := fmt.Sprintf(
			"[Webtoon Mode - Page %d]\n\n"+
				"Failed to load image: %v\n\n"+
				"Use ← → to navigate",
			m.currentPage+1,
			err,
		)

		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.ColorError).
			Padding(2, 4).
			Width(imageWidth).
			Height(availableHeight).
			Align(lipgloss.Center, lipgloss.Center)

		return boxStyle.Render(placeholder)
	}

	return lipgloss.Place(
		m.width,
		availableHeight,
		lipgloss.Center,
		lipgloss.Center,
		imageStr,
	)
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

	return theme.HelpStyle.Render(strings.Join(controls, " • ")) + "\n" + progressBar
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
	err      error
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
		return progressLoadedMsg{
			progress: nil,
			err:      nil,
		}
	}

	progress, err := m.storage.Progress.GetProgress(m.manga.ID, m.chapter.ID)
	if err != nil {
		return progressLoadedMsg{
			progress: nil,
			err:      err,
		}
	}

	return progressLoadedMsg{
		progress: progress,
		err:      nil,
	}
}

func (m Model) saveProgress() tea.Msg {
	if m.storage == nil || m.manga == nil || m.chapter == nil {
		return nil
	}

	err := m.storage.Progress.UpdateProgress(
		m.manga.ID,
		m.manga.Title,
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
