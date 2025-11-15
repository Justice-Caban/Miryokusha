package downloads

import (
	"fmt"
	"strings"
	"time"

	"github.com/Justice-Caban/Miryokusha/internal/downloads"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Color palette (duplicated to avoid import cycle)
var (
	colorPrimary   = lipgloss.Color("205") // Pink
	colorSecondary = lipgloss.Color("99")  // Purple
	colorAccent    = lipgloss.Color("86")  // Cyan
	colorSuccess   = lipgloss.Color("42")  // Green
	colorWarning   = lipgloss.Color("214") // Orange
	colorError     = lipgloss.Color("196") // Red
	colorMuted     = lipgloss.Color("242") // Gray
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary).
			MarginTop(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning)
)

// Model represents the downloads view model
type Model struct {
	width  int
	height int

	manager *downloads.Manager

	// UI state
	cursor       int
	selectedTab  int // 0 = Active, 1 = Queue, 2 = Completed
	autoScroll   bool

	// Download lists (cached)
	activeList    []*downloads.DownloadItem
	queueList     []*downloads.DownloadItem
	completedList []*downloads.DownloadItem
	stats         downloads.DownloadStats

	// Refresh ticker
	lastRefresh time.Time
}

// NewModel creates a new downloads model
func NewModel(manager *downloads.Manager) Model {
	return Model{
		manager:      manager,
		cursor:       0,
		selectedTab:  0,
		autoScroll:   true,
		lastRefresh:  time.Now(),
	}
}

// Init initializes the downloads model
func (m Model) Init() tea.Cmd {
	return m.refreshData
}

// Update handles messages for the downloads view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case refreshDataMsg:
		m.activeList = msg.active
		m.queueList = msg.queue
		m.completedList = msg.completed
		m.stats = msg.stats
		m.lastRefresh = time.Now()

		// Auto-refresh every second
		return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
			return tickMsg{}
		})

	case tickMsg:
		return m, m.refreshData
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		// Switch tabs
		m.selectedTab = (m.selectedTab + 1) % 3
		m.cursor = 0

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		maxCursor := m.getMaxCursor()
		if m.cursor < maxCursor {
			m.cursor++
		}

	case "g":
		// Go to top
		m.cursor = 0

	case "G":
		// Go to bottom
		m.cursor = m.getMaxCursor()

	case "p":
		// Pause downloads
		m.manager.Pause()
		return m, m.refreshData

	case "r":
		// Resume downloads
		m.manager.Resume()
		return m, m.refreshData

	case "s":
		// Start download manager
		m.manager.Start()
		return m, m.refreshData

	case "S":
		// Stop download manager
		m.manager.Stop()
		return m, m.refreshData

	case "c":
		// Cancel selected download (if in queue or active)
		if m.selectedTab == 0 && m.cursor < len(m.activeList) {
			item := m.activeList[m.cursor]
			m.manager.Remove(item.ChapterID)
			return m, m.refreshData
		} else if m.selectedTab == 1 && m.cursor < len(m.queueList) {
			item := m.queueList[m.cursor]
			m.manager.Remove(item.ChapterID)
			return m, m.refreshData
		}

	case "C":
		// Clear completed downloads
		m.manager.ClearCompleted()
		return m, m.refreshData

	case "a":
		// Toggle auto-scroll
		m.autoScroll = !m.autoScroll
	}

	return m, nil
}

// View renders the downloads view
func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n")

	// Content based on selected tab
	switch m.selectedTab {
	case 0:
		b.WriteString(m.renderActiveDownloads())
	case 1:
		b.WriteString(m.renderQueue())
	case 2:
		b.WriteString(m.renderCompleted())
	}

	b.WriteString("\n")

	// Footer
	b.WriteString(m.renderFooter())

	return b.String()
}

// renderHeader renders the downloads header
func (m Model) renderHeader() string {
	title := titleStyle.Render("Downloads")

	info := lipgloss.NewStyle().
		Foreground(colorSecondary).
		Render(fmt.Sprintf(
			"Active: %d | Queued: %d | Completed: %d | Failed: %d",
			m.stats.ActiveDownloads,
			len(m.queueList),
			m.stats.CompletedDownloads,
			m.stats.FailedDownloads,
		))

	return title + "\n" + info
}

// renderTabs renders the tab bar
func (m Model) renderTabs() string {
	tabs := []string{"Active", "Queue", "Completed"}
	var renderedTabs []string

	for i, tab := range tabs {
		style := lipgloss.NewStyle().Padding(0, 2)
		if i == m.selectedTab {
			style = style.
				Background(colorPrimary).
				Foreground(lipgloss.Color("#000000")).
				Bold(true)
		} else {
			style = style.Foreground(colorMuted)
		}
		renderedTabs = append(renderedTabs, style.Render(tab))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, renderedTabs...)
}

// renderActiveDownloads renders the active downloads list
func (m Model) renderActiveDownloads() string {
	if len(m.activeList) == 0 {
		return mutedStyle.Render("\nNo active downloads")
	}

	var b strings.Builder
	b.WriteString("\n")

	for i, item := range m.activeList {
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

		// Progress indicator
		progressBar := m.renderProgressBar(item.Progress())
		speedInfo := fmt.Sprintf("Page %d/%d", item.CurrentPage, item.TotalPages)

		// Format line
		line := fmt.Sprintf("▼ %s - %s\n  %s  %s",
			item.MangaTitle,
			item.ChapterName,
			progressBar,
			speedInfo,
		)

		b.WriteString(itemStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

// renderQueue renders the queued downloads
func (m Model) renderQueue() string {
	if len(m.queueList) == 0 {
		return mutedStyle.Render("\nQueue is empty")
	}

	var b strings.Builder
	b.WriteString("\n")

	visibleItems := m.height - 15
	if visibleItems < 1 {
		visibleItems = 1
	}

	start := 0
	end := len(m.queueList)
	if end > visibleItems {
		end = visibleItems
	}

	for i := start; i < end; i++ {
		item := m.queueList[i]
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

		// Status icon
		statusIcon := "○"
		if item.Status == downloads.StatusPaused {
			statusIcon = "⏸"
		}

		// Priority indicator
		priorityStr := ""
		if item.Priority < 10 {
			priorityStr = fmt.Sprintf(" [P%d]", item.Priority)
		}

		// Format line
		line := fmt.Sprintf("%s %s - %s%s",
			statusIcon,
			item.MangaTitle,
			item.ChapterName,
			mutedStyle.Render(priorityStr),
		)

		b.WriteString(itemStyle.Render(line))
		b.WriteString("\n")
	}

	if len(m.queueList) > visibleItems {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("\n... and %d more", len(m.queueList)-visibleItems)))
	}

	return b.String()
}

// renderCompleted renders the completed downloads
func (m Model) renderCompleted() string {
	if len(m.completedList) == 0 {
		return mutedStyle.Render("\nNo completed downloads")
	}

	var b strings.Builder
	b.WriteString("\n")

	visibleItems := m.height - 15
	if visibleItems < 1 {
		visibleItems = 1
	}

	start := 0
	end := len(m.completedList)
	if end > visibleItems {
		end = visibleItems
	}

	for i := start; i < end; i++ {
		item := m.completedList[i]
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

		// Time info
		duration := item.CompletedAt.Sub(item.StartedAt)
		timeStr := fmt.Sprintf("%s", m.formatDuration(duration))

		// Format line
		line := fmt.Sprintf("✓ %s - %s  %s",
			item.MangaTitle,
			item.ChapterName,
			mutedStyle.Render(timeStr),
		)

		b.WriteString(itemStyle.Render(line))
		b.WriteString("\n")
	}

	if len(m.completedList) > visibleItems {
		b.WriteString(mutedStyle.Render(fmt.Sprintf("\n... and %d more", len(m.completedList)-visibleItems)))
	}

	return b.String()
}

// renderProgressBar renders a progress bar
func (m Model) renderProgressBar(progress float64) string {
	barWidth := 20
	filled := int(progress / 100.0 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]"

	return successStyle.Render(fmt.Sprintf("%s %3.0f%%", bar, progress))
}

// renderFooter renders the footer with controls
func (m Model) renderFooter() string {
	controls := []string{
		"↑↓/jk: navigate",
		"Tab: switch tabs",
		"p: pause",
		"r: resume",
		"s: start",
		"S: stop",
		"c: cancel",
		"C: clear completed",
		"Esc: back",
	}

	return helpStyle.Render(strings.Join(controls, " • "))
}

// getMaxCursor returns the maximum cursor position for the current tab
func (m Model) getMaxCursor() int {
	switch m.selectedTab {
	case 0:
		return len(m.activeList) - 1
	case 1:
		return len(m.queueList) - 1
	case 2:
		return len(m.completedList) - 1
	}
	return 0
}

// formatDuration formats a duration for display
func (m Model) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// Messages

type refreshDataMsg struct {
	active    []*downloads.DownloadItem
	queue     []*downloads.DownloadItem
	completed []*downloads.DownloadItem
	stats     downloads.DownloadStats
}

type tickMsg struct{}

// Commands

func (m Model) refreshData() tea.Msg {
	return refreshDataMsg{
		active:    m.manager.GetActive(),
		queue:     m.manager.GetQueue(),
		completed: m.manager.GetCompleted(),
		stats:     m.manager.GetStats(),
	}
}
