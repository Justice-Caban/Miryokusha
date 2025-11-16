package history

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

// ViewMode represents what to display
type ViewMode int

const (
	ModeHistory ViewMode = iota
	ModeContinueReading
	ModeStatistics
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

	sectionHeaderStyle = lipgloss.NewStyle().
				Foreground(theme.ColorSecondary).
				Bold(true)

	statStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Bold(true)
)

// Model represents the history view model
type Model struct {
	width  int
	height int

	// Data
	history         []*storage.HistoryEntry
	continueReading []*storage.ProgressEntry
	stats           *storage.ReadingStats

	// UI state
	cursor int
	offset int
	mode   ViewMode

	// Dependencies
	sourceManager *source.SourceManager
	storage       *storage.Storage

	// Loading state
	loading bool
	err     error
}

// NewModel creates a new history model
func NewModel(sm *source.SourceManager, st *storage.Storage) Model {
	return Model{
		history:         make([]*storage.HistoryEntry, 0),
		continueReading: make([]*storage.ProgressEntry, 0),
		cursor:          0,
		offset:          0,
		mode:            ModeContinueReading,
		sourceManager:   sm,
		storage:         st,
		loading:         true,
	}
}

// Init initializes the history model
func (m Model) Init() tea.Cmd {
	return m.loadData
}

// Update handles messages for the history view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case historyLoadedMsg:
		m.history = msg.history
		m.continueReading = msg.continueReading
		m.stats = msg.stats
		m.loading = false
		return m, nil

	case historyErrorMsg:
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
		maxItems := m.getItemCount()
		if m.cursor < maxItems-1 {
			m.cursor++
			m.adjustOffset()
		}

	case "g":
		m.cursor = 0
		m.offset = 0

	case "G":
		maxItems := m.getItemCount()
		if maxItems > 0 {
			m.cursor = maxItems - 1
			m.adjustOffset()
		}

	case "r":
		// Refresh data
		m.loading = true
		return m, m.loadData

	case "tab":
		// Cycle through modes
		m.mode = (m.mode + 1) % 3
		m.cursor = 0
		m.offset = 0

	case "c":
		// Clear history (with confirmation)
		if m.mode == ModeHistory {
			return m, m.clearHistory
		}

	case "enter":
		// Open selected item
		return m, m.openSelected()
	}

	return m, nil
}

// getItemCount returns the number of items in the current mode
func (m Model) getItemCount() int {
	switch m.mode {
	case ModeHistory:
		return len(m.history)
	case ModeContinueReading:
		return len(m.continueReading)
	case ModeStatistics:
		return 0 // Stats view is not selectable
	}
	return 0
}

// adjustOffset adjusts the scroll offset to keep cursor visible
func (m *Model) adjustOffset() {
	visibleItems := m.height - 10
	if visibleItems < 1 {
		visibleItems = 1
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+visibleItems {
		m.offset = m.cursor - visibleItems + 1
	}
}

// View renders the history view
func (m Model) View() string {
	if m.loading {
		return centeredText(m.width, m.height, "Loading history...")
	}

	if m.err != nil {
		return centeredText(m.width, m.height, fmt.Sprintf("Error: %v", m.err))
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Content based on mode
	switch m.mode {
	case ModeHistory:
		b.WriteString(m.renderHistory())
	case ModeContinueReading:
		b.WriteString(m.renderContinueReading())
	case ModeStatistics:
		b.WriteString(m.renderStatistics())
	}

	b.WriteString("\n")

	// Footer
	b.WriteString(m.renderFooter())

	return b.String()
}

// renderHeader renders the header
func (m Model) renderHeader() string {
	var modeStr string
	switch m.mode {
	case ModeHistory:
		modeStr = "Reading History"
	case ModeContinueReading:
		modeStr = "Continue Reading"
	case ModeStatistics:
		modeStr = "Statistics"
	}

	title := titleStyle.Render(modeStr)

	info := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Render("Press Tab to switch modes")

	return title + "\n" + info
}

// renderHistory renders the reading history
func (m Model) renderHistory() string {
	if len(m.history) == 0 {
		return centeredText(m.width, m.height-10, "No reading history\n\nStart reading manga to build your history")
	}

	var b strings.Builder

	// Group history by time
	grouped := m.groupHistoryByTime()

	visibleItems := m.height - 12
	if visibleItems < 1 {
		visibleItems = 1
	}

	currentIndex := 0
	for _, group := range grouped {
		// Section header
		b.WriteString(sectionHeaderStyle.Render(group.label))
		b.WriteString("\n")

		for _, entry := range group.entries {
			if currentIndex >= m.offset && currentIndex < m.offset+visibleItems {
				isCursor := currentIndex == m.cursor

				itemStyle := lipgloss.NewStyle()
				if isCursor {
					itemStyle = itemStyle.
						Background(theme.ColorPrimary).
						Foreground(lipgloss.Color("#000000")).
						Bold(true).
						Width(m.width - 4)
				}

				timeStr := entry.ReadAt.Format("15:04")
				chapterStr := fmt.Sprintf("Ch. %.1f", entry.ChapterNumber)
				if entry.ChapterTitle != "" {
					chapterStr += fmt.Sprintf(": %s", entry.ChapterTitle)
				}

				line := fmt.Sprintf("  %s - %s  %s",
					timeStr,
					entry.MangaTitle,
					lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(chapterStr),
				)

				b.WriteString(itemStyle.Render(line))
				b.WriteString("\n")
			}
			currentIndex++
		}

		b.WriteString("\n")
	}

	return b.String()
}

// historyGroup represents a time-grouped set of history entries
type historyGroup struct {
	label   string
	entries []*storage.HistoryEntry
}

// groupHistoryByTime groups history entries by time periods
func (m Model) groupHistoryByTime() []historyGroup {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)
	weekAgo := today.AddDate(0, 0, -7)

	groups := make(map[string]*historyGroup)
	groupOrder := []string{"Today", "Yesterday", "This Week", "Earlier"}

	for _, key := range groupOrder {
		groups[key] = &historyGroup{label: key, entries: make([]*storage.HistoryEntry, 0)}
	}

	for _, entry := range m.history {
		entryDate := entry.ReadAt.Truncate(24 * time.Hour)

		var groupKey string
		if entryDate.Equal(today) {
			groupKey = "Today"
		} else if entryDate.Equal(yesterday) {
			groupKey = "Yesterday"
		} else if entryDate.After(weekAgo) {
			groupKey = "This Week"
		} else {
			groupKey = "Earlier"
		}

		groups[groupKey].entries = append(groups[groupKey].entries, entry)
	}

	// Return only non-empty groups in order
	result := make([]historyGroup, 0)
	for _, key := range groupOrder {
		if len(groups[key].entries) > 0 {
			result = append(result, *groups[key])
		}
	}

	return result
}

// renderContinueReading renders the continue reading list
func (m Model) renderContinueReading() string {
	if len(m.continueReading) == 0 {
		return centeredText(m.width, m.height-10, "No chapters in progress\n\nStart reading to see your progress here")
	}

	var b strings.Builder

	visibleItems := m.height - 10
	if visibleItems < 1 {
		visibleItems = 1
	}

	start := m.offset
	end := m.offset + visibleItems
	if end > len(m.continueReading) {
		end = len(m.continueReading)
	}

	for i := start; i < end; i++ {
		entry := m.continueReading[i]
		isCursor := i == m.cursor

		itemStyle := lipgloss.NewStyle()
		if isCursor {
			itemStyle = itemStyle.
				Background(theme.ColorPrimary).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Width(m.width - 4)
		}

		// Calculate progress
		progressPct := float64(entry.CurrentPage+1) / float64(entry.TotalPages) * 100
		progressBar := m.renderMiniProgressBar(progressPct, 20)

		// Time since last read
		timeSince := formatTimeSince(entry.LastReadAt)

		line := fmt.Sprintf("  %s - Page %d/%d %s  %s",
			entry.MangaID, // TODO: Get manga title from cache/source
			entry.CurrentPage+1,
			entry.TotalPages,
			progressBar,
			lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(timeSince),
		)

		b.WriteString(itemStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

// renderStatistics renders reading statistics
func (m Model) renderStatistics() string {
	if m.stats == nil {
		return centeredText(m.width, m.height-10, "No statistics available")
	}

	var b strings.Builder

	// Overall stats
	b.WriteString(sectionHeaderStyle.Render("Overall Statistics"))
	b.WriteString("\n\n")

	statsData := []struct {
		label string
		value string
	}{
		{"Total Manga Read", fmt.Sprintf("%d", m.stats.TotalMangaRead)},
		{"Total Chapters Read", fmt.Sprintf("%d", m.stats.TotalChaptersRead)},
		{"Total Pages Read", fmt.Sprintf("%d", m.stats.TotalPagesRead)},
		{"Time Spent Reading", formatDuration(m.stats.TotalReadingTime)},
		{"Average Session Time", formatDuration(m.stats.AverageSessionTime)},
	}

	for _, stat := range statsData {
		b.WriteString(fmt.Sprintf("  %s: %s\n",
			stat.label,
			statStyle.Render(stat.value),
		))
	}

	b.WriteString("\n")

	// Streaks
	b.WriteString(sectionHeaderStyle.Render("Reading Streaks"))
	b.WriteString("\n\n")

	currentStreakIcon := ""
	if m.stats.CurrentStreak > 0 {
		currentStreakIcon = "🔥"
	}

	b.WriteString(fmt.Sprintf("  Current Streak: %s %s\n",
		statStyle.Render(fmt.Sprintf("%d days", m.stats.CurrentStreak)),
		currentStreakIcon,
	))

	b.WriteString(fmt.Sprintf("  Longest Streak: %s 🏆\n",
		statStyle.Render(fmt.Sprintf("%d days", m.stats.LongestReadingStreak)),
	))

	return b.String()
}

// renderMiniProgressBar renders a small progress bar
func (m Model) renderMiniProgressBar(pct float64, width int) string {
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return lipgloss.NewStyle().Foreground(theme.ColorAccent).Render(bar)
}

// renderFooter renders the footer with controls
func (m Model) renderFooter() string {
	controls := []string{
		"↑↓/jk: navigate",
		"g/G: top/bottom",
		"Tab: switch mode",
		"r: refresh",
	}

	if m.mode == ModeHistory {
		controls = append(controls, "c: clear history")
	}

	if m.mode != ModeStatistics {
		controls = append(controls, "Enter: continue")
	}

	controls = append(controls, "Esc: back")

	return helpStyle.Render(strings.Join(controls, " • "))
}

// centeredText centers text in the given width and height
func centeredText(width, height int, text string) string {
	style := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)
	return style.Render(text)
}

// formatTimeSince formats a duration since a time
func formatTimeSince(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		return fmt.Sprintf("%d min ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// formatDuration formats seconds into a readable string
func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	} else {
		hours := seconds / 3600
		mins := (seconds % 3600) / 60
		if mins > 0 {
			return fmt.Sprintf("%dh %dm", hours, mins)
		}
		return fmt.Sprintf("%dh", hours)
	}
}

// Messages

type historyLoadedMsg struct {
	history         []*storage.HistoryEntry
	continueReading []*storage.ProgressEntry
	stats           *storage.ReadingStats
}

type historyErrorMsg struct {
	err error
}

type OpenChapterMsg struct {
	MangaID   string
	ChapterID string
}

// Commands

func (m Model) loadData() tea.Msg {
	if m.storage == nil {
		return historyErrorMsg{err: fmt.Errorf("storage not available")}
	}

	// Load recent history
	history, err := m.storage.History.GetRecentHistory(100)
	if err != nil {
		return historyErrorMsg{err: fmt.Errorf("failed to load history: %w", err)}
	}

	// Load in-progress chapters
	continueReading, err := m.storage.Progress.GetInProgressChapters()
	if err != nil {
		return historyErrorMsg{err: fmt.Errorf("failed to load progress: %w", err)}
	}

	// Load statistics
	stats, err := m.storage.Stats.GetGlobalStats()
	if err != nil {
		return historyErrorMsg{err: fmt.Errorf("failed to load stats: %w", err)}
	}

	return historyLoadedMsg{
		history:         history,
		continueReading: continueReading,
		stats:           stats,
	}
}

func (m Model) clearHistory() tea.Msg {
	if m.storage == nil {
		return nil
	}

	// Clear all history
	err := m.storage.History.ClearAllHistory()
	if err != nil {
		return historyErrorMsg{err: err}
	}

	// Reload data
	return m.loadData()
}

func (m Model) openSelected() tea.Cmd {
	switch m.mode {
	case ModeContinueReading:
		if m.cursor >= 0 && m.cursor < len(m.continueReading) {
			entry := m.continueReading[m.cursor]
			return func() tea.Msg {
				return OpenChapterMsg{
					MangaID:   entry.MangaID,
					ChapterID: entry.ChapterID,
				}
			}
		}

	case ModeHistory:
		if m.cursor >= 0 && m.cursor < len(m.history) {
			entry := m.history[m.cursor]
			return func() tea.Msg {
				return OpenChapterMsg{
					MangaID:   entry.MangaID,
					ChapterID: entry.ChapterID,
				}
			}
		}
	}

	return nil
}
