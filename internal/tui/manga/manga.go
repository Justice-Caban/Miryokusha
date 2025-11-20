package manga

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Justice-Caban/Miryokusha/internal/source"
	"github.com/Justice-Caban/Miryokusha/internal/storage"
	"github.com/Justice-Caban/Miryokusha/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the manga details view
type Model struct {
	width  int
	height int

	// Data
	manga    *source.Manga
	chapters []*source.Chapter

	// UI state
	cursor  int
	offset  int
	loading bool
	err     error

	// Dependencies
	sourceManager *source.SourceManager
	storage       *storage.Storage
}

// NewModel creates a new manga details model
func NewModel(manga *source.Manga, sm *source.SourceManager, st *storage.Storage) Model {
	return Model{
		manga:         manga,
		chapters:      nil,
		cursor:        0,
		offset:        0,
		loading:       true,
		sourceManager: sm,
		storage:       st,
	}
}

// Init initializes the manga details model
func (m Model) Init() tea.Cmd {
	return m.loadChapters
}

// Update handles messages for the manga details view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case chaptersLoadedMsg:
		m.chapters = msg.chapters
		m.loading = false
		m.err = msg.err
		return m, nil
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.adjustOffset()
		}

	case "down", "j":
		if m.cursor < len(m.chapters)-1 {
			m.cursor++
			m.adjustOffset()
		}

	case "g":
		// Go to top
		m.cursor = 0
		m.offset = 0

	case "G":
		// Go to bottom
		if len(m.chapters) > 0 {
			m.cursor = len(m.chapters) - 1
			m.adjustOffset()
		}

	case "enter":
		// DEBUG: Log that Enter was pressed
		logFile, _ := os.OpenFile("./miryokusha-reader.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if logFile != nil {
			fmt.Fprintf(logFile, "\n=== Enter key pressed in manga view ===\n")
			fmt.Fprintf(logFile, "Cursor: %d, Total chapters: %d\n", m.cursor, len(m.chapters))
			if m.cursor < len(m.chapters) {
				fmt.Fprintf(logFile, "Selected chapter: %.1f - %s\n", m.chapters[m.cursor].ChapterNumber, m.chapters[m.cursor].Title)
			}
			logFile.Close()
		}

		// Open selected chapter in reader
		if m.cursor < len(m.chapters) {
			selectedChapter := m.chapters[m.cursor]
			return m, m.openChapter(selectedChapter)
		}

	case "r":
		// Refresh chapter list
		m.loading = true
		return m, m.loadChapters
	}

	return m, nil
}

// adjustOffset adjusts the scroll offset to keep cursor visible
func (m *Model) adjustOffset() {
	visibleItems := m.height - 15 // Account for header and footer
	if visibleItems < 1 {
		visibleItems = 1
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+visibleItems {
		m.offset = m.cursor - visibleItems + 1
	}
}

// View renders the manga details view
func (m Model) View() string {
	if m.loading {
		return theme.CenteredText(m.width, m.height, "Loading chapters...")
	}

	if m.err != nil {
		return theme.CenteredText(m.width, m.height, fmt.Sprintf("Error: %v\n\nPress 'r' to retry or Esc to go back", m.err))
	}

	var b strings.Builder

	// Header with manga info
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Chapter list
	b.WriteString(m.renderChapterList())
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

// renderHeader renders the manga information header
func (m Model) renderHeader() string {
	title := theme.TitleStyle.Render(m.manga.Title)

	var info []string

	// Author
	if m.manga.Author != "" {
		info = append(info, fmt.Sprintf("Author: %s", m.manga.Author))
	}

	// Status
	if m.manga.Status != "" {
		info = append(info, fmt.Sprintf("Status: %s", m.manga.Status))
	}

	// Chapter count
	chapterCountStr := fmt.Sprintf("%d chapters", len(m.chapters))
	if m.manga.UnreadCount > 0 {
		chapterCountStr += fmt.Sprintf(" (%d unread)", m.manga.UnreadCount)
	}
	info = append(info, chapterCountStr)

	infoStr := lipgloss.NewStyle().
		Foreground(theme.ColorSecondary).
		Render(strings.Join(info, " • "))

	return title + "\n" + infoStr
}

// renderChapterList renders the list of chapters
func (m Model) renderChapterList() string {
	if len(m.chapters) == 0 {
		return theme.MutedStyle.Render("No chapters available")
	}

	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorPrimary).
		Render("Chapters") + "\n\n")

	visibleItems := m.height - 15
	if visibleItems < 1 {
		visibleItems = 1
	}

	start := m.offset
	end := m.offset + visibleItems
	if end > len(m.chapters) {
		end = len(m.chapters)
	}

	for i := start; i < end; i++ {
		chapter := m.chapters[i]
		isCursor := i == m.cursor

		// Item style
		itemStyle := lipgloss.NewStyle()
		if isCursor {
			itemStyle = itemStyle.
				Background(theme.ColorPrimary).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Width(m.width - 8)
		}

		// Read indicator
		readIndicator := "  "
		if chapter.IsRead {
			readIndicator = "✓ "
		}

		// Downloaded indicator
		downloadIndicator := ""
		if chapter.IsDownloaded {
			downloadIndicator = " 📥"
		}

		// Bookmarked indicator
		bookmarkIndicator := ""
		if chapter.IsBookmarked {
			bookmarkIndicator = " 🔖"
		}

		// Format chapter title
		chapterTitle := chapter.Title
		if chapterTitle == "" {
			chapterTitle = fmt.Sprintf("Chapter %.1f", chapter.ChapterNumber)
		}

		// Build line
		line := fmt.Sprintf("%s %s%s%s",
			readIndicator,
			chapterTitle,
			downloadIndicator,
			bookmarkIndicator,
		)

		b.WriteString(itemStyle.Render(line))
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(m.chapters) > visibleItems {
		scrollInfo := fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(m.chapters))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			Render(scrollInfo))
	}

	return b.String()
}

// renderFooter renders the footer with controls
func (m Model) renderFooter() string {
	controls := []string{
		"↑↓/jk: navigate",
		"g/G: top/bottom",
		"Enter: read chapter",
		"r: refresh",
		"Esc: back",
	}

	return theme.HelpStyle.Render(strings.Join(controls, " • "))
}

// Messages

type chaptersLoadedMsg struct {
	chapters []*source.Chapter
	err      error
}

// OpenChapterMsg is DEPRECATED - no longer used since we switched to standalone reader
// The manga view now uses tea.ExecProcess to launch the standalone reader directly
// This type is kept for backward compatibility with history view, which still uses the old TUI reader
// TODO: Remove this once history view is updated to use standalone reader
type OpenChapterMsg struct {
	Manga   *source.Manga
	Chapter *source.Chapter
}

// Commands

func (m Model) loadChapters() tea.Msg {
	// Get the appropriate source
	var src source.Source
	for _, s := range m.sourceManager.GetSources() {
		if s.GetID() == m.manga.SourceID {
			src = s
			break
		}
	}

	if src == nil {
		return chaptersLoadedMsg{
			chapters: nil,
			err:      fmt.Errorf("source not found: %s", m.manga.SourceID),
		}
	}

	// Load chapters
	chapters, err := src.ListChapters(m.manga.ID)
	if err != nil {
		return chaptersLoadedMsg{
			chapters: nil,
			err:      err,
		}
	}

	return chaptersLoadedMsg{
		chapters: chapters,
		err:      nil,
	}
}

func (m Model) openChapter(chapter *source.Chapter) tea.Cmd {
	// DEBUG: Log that we're attempting to open a chapter
	logFile, _ := os.OpenFile("./miryokusha-reader.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if logFile != nil {
		fmt.Fprintf(logFile, "\n=== openChapter called ===\n")
		fmt.Fprintf(logFile, "Chapter: %.1f - %s\n", chapter.ChapterNumber, chapter.Title)
		fmt.Fprintf(logFile, "Manga: %s (ID: %s)\n", m.manga.Title, m.manga.ID)
		fmt.Fprintf(logFile, "Chapter ID: %s\n", chapter.ID)
		logFile.Close()
	}

	// Launch standalone reader using tea.Exec to bypass alt-screen limitations
	// This allows Kitty graphics protocol to work properly

	// NOTE: We only pass IDs, not full JSON data, to avoid "argument list too long" error
	// The reader will re-fetch the data from the source using these IDs

	// Get the absolute path to the current binary
	execPath, err := os.Executable()
	if err != nil {
		// Fallback to os.Args[0]
		execPath = os.Args[0]
		logFile, _ = os.OpenFile("./miryokusha-reader.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if logFile != nil {
			fmt.Fprintf(logFile, "WARNING: Could not get executable path, using os.Args[0]: %s\n", execPath)
			logFile.Close()
		}
	}

	// DEBUG: Log that we're about to launch reader
	logFile2, _ := os.OpenFile("./miryokusha-reader.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if logFile2 != nil {
		fmt.Fprintf(logFile2, "Launching tea.ExecProcess with IDs only\n")
		fmt.Fprintf(logFile2, "Executable path: %s\n", execPath)
		fmt.Fprintf(logFile2, "Manga ID: %s, Chapter ID: %s\n", m.manga.ID, chapter.ID)
		logFile2.Close()
	}

	// Create the command with just IDs (not full JSON data)
	cmd := exec.Command(
		execPath,
		"reader",
		"--manga-id", m.manga.ID,
		"--chapter-id", chapter.ID,
	)

	// Make sure stdin/stdout/stderr are properly connected
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Launch reader mode using tea.ExecProcess
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		// DEBUG: Log when reader exits
		logFile3, _ := os.OpenFile("./miryokusha-reader.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if logFile3 != nil {
			if err != nil {
				fmt.Fprintf(logFile3, "Reader exited with error: %v\n", err)
			} else {
				fmt.Fprintf(logFile3, "Reader exited successfully\n")
			}
			logFile3.Close()
		}

		// When reader exits, return to this view
		if err != nil {
			return fmt.Errorf("reader error: %w", err)
		}
		return nil
	})
}
