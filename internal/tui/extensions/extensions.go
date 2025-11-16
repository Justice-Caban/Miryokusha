package extensions

import (
	"github.com/Justice-Caban/Miryokusha/internal/tui/theme"
	"fmt"
	"strings"

	"github.com/Justice-Caban/Miryokusha/internal/suwayomi"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewMode represents what to display
type ViewMode int

const (
	ModeBrowse ViewMode = iota
	ModeInstalled
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

	statusStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Bold(true)

	nsfwStyle = lipgloss.NewStyle().
			Foreground(theme.ColorWarning).
			Bold(true)
)

// Model represents the extensions view model
type Model struct {
	width  int
	height int

	// Data
	available []*suwayomi.Extension
	installed []*suwayomi.Extension

	// UI state
	cursor         int
	offset         int
	mode           ViewMode
	languageFilter string // e.g., "en", "ja", "all"
	showNSFW       bool
	searchQuery    string
	searchActive   bool

	// Dependencies
	client *suwayomi.Client

	// Loading state
	loading bool
	err     error
}

// NewModel creates a new extensions model
func NewModel(client *suwayomi.Client) Model {
	return Model{
		available:      make([]*suwayomi.Extension, 0),
		installed:      make([]*suwayomi.Extension, 0),
		cursor:         0,
		offset:         0,
		mode:           ModeInstalled,
		languageFilter: "all",
		showNSFW:       false,
		client:         client,
		loading:        true,
	}
}

// Init initializes the extensions model
func (m Model) Init() tea.Cmd {
	return m.loadExtensions
}

// Update handles messages for the extensions view
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

	case extensionsLoadedMsg:
		m.available = msg.available
		m.installed = msg.installed
		m.loading = false
		return m, nil

	case extensionInstalledMsg:
		// Refresh the lists
		m.loading = true
		return m, m.loadExtensions

	case extensionErrorMsg:
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

	case "tab":
		// Switch between browse and installed
		m.mode = (m.mode + 1) % 2
		m.cursor = 0
		m.offset = 0

	case "r":
		// Refresh
		m.loading = true
		return m, m.loadExtensions

	case "/":
		// Start search
		m.searchActive = true
		m.searchQuery = ""

	case "i", "enter":
		// Install/uninstall
		return m, m.toggleInstall()

	case "u":
		// Update (if installed and has update)
		return m, m.updateExtension()

	case "U":
		// Update all
		return m, m.updateAll()

	case "n":
		// Toggle NSFW filter
		m.showNSFW = !m.showNSFW

	case "l":
		// Cycle language filter
		return m.cycleLanguageFilter(), nil
	}

	return m, nil
}

// handleSearchInput handles search input
func (m Model) handleSearchInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchActive = false
		m.searchQuery = ""

	case "enter":
		m.searchActive = false

	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}

	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
		}
	}

	return m, nil
}

// cycleLanguageFilter cycles through language filters
func (m Model) cycleLanguageFilter() Model {
	languages := []string{"all", "en", "ja", "es", "fr"}
	for i, lang := range languages {
		if lang == m.languageFilter {
			m.languageFilter = languages[(i+1)%len(languages)]
			break
		}
	}
	m.cursor = 0
	m.offset = 0
	return m
}

// getItemCount returns the number of items in the current mode
func (m Model) getItemCount() int {
	switch m.mode {
	case ModeBrowse:
		return len(m.getFilteredExtensions(m.available))
	case ModeInstalled:
		return len(m.getFilteredExtensions(m.installed))
	}
	return 0
}

// getFilteredExtensions applies filters to extensions
func (m Model) getFilteredExtensions(extensions []*suwayomi.Extension) []*suwayomi.Extension {
	filtered := make([]*suwayomi.Extension, 0)

	for _, ext := range extensions {
		// Language filter
		if m.languageFilter != "all" && ext.Language != m.languageFilter {
			continue
		}

		// NSFW filter
		if ext.IsNSFW && !m.showNSFW {
			continue
		}

		// Search filter
		if m.searchQuery != "" {
			if !strings.Contains(strings.ToLower(ext.Name), strings.ToLower(m.searchQuery)) {
				continue
			}
		}

		filtered = append(filtered, ext)
	}

	return filtered
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

// View renders the extensions view
func (m Model) View() string {
	if m.loading {
		return centeredText(m.width, m.height, "Loading extensions...")
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
	case ModeBrowse:
		b.WriteString(m.renderBrowse())
	case ModeInstalled:
		b.WriteString(m.renderInstalled())
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
	case ModeBrowse:
		modeStr = "Browse Extensions"
	case ModeInstalled:
		modeStr = "Installed Extensions"
	}

	title := titleStyle.Render(modeStr)

	// Filters info
	filters := []string{
		fmt.Sprintf("Language: %s", m.languageFilter),
	}

	if m.showNSFW {
		filters = append(filters, "NSFW: shown")
	} else {
		filters = append(filters, "NSFW: hidden")
	}

	info := lipgloss.NewStyle().
		Foreground(theme.ColorMuted).
		Render(strings.Join(filters, " | "))

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

// renderBrowse renders the browse extensions list
func (m Model) renderBrowse() string {
	filtered := m.getFilteredExtensions(m.available)

	if len(filtered) == 0 {
		if len(m.available) == 0 {
			return centeredText(m.width, m.height-10, "No extensions available\n\nCheck your connection to Suwayomi server")
		}
		return centeredText(m.width, m.height-10, "No extensions match current filters")
	}

	var b strings.Builder

	visibleItems := m.height - 12
	if visibleItems < 1 {
		visibleItems = 1
	}

	start := m.offset
	end := m.offset + visibleItems
	if end > len(filtered) {
		end = len(filtered)
	}

	for i := start; i < end; i++ {
		ext := filtered[i]
		isCursor := i == m.cursor

		itemStyle := lipgloss.NewStyle()
		if isCursor {
			itemStyle = itemStyle.
				Background(theme.ColorPrimary).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Width(m.width - 4)
		}

		// Status indicator
		statusIndicator := ""
		if ext.IsInstalled {
			if ext.HasUpdate {
				statusIndicator = statusStyle.Render("[UPDATE]")
			} else {
				statusIndicator = lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("[✓]")
			}
		} else {
			statusIndicator = lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("[ ]")
		}

		// NSFW indicator
		nsfwIndicator := ""
		if ext.IsNSFW {
			nsfwIndicator = nsfwStyle.Render("[NSFW]")
		}

		// Language badge
		langBadge := lipgloss.NewStyle().
			Foreground(theme.ColorSecondary).
			Render(fmt.Sprintf("[%s]", ext.Language))

		line := fmt.Sprintf("%s %s %s %s v%s",
			statusIndicator,
			ext.Name,
			langBadge,
			nsfwIndicator,
			ext.VersionName,
		)

		b.WriteString(itemStyle.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

// renderInstalled renders the installed extensions list
func (m Model) renderInstalled() string {
	filtered := m.getFilteredExtensions(m.installed)

	if len(filtered) == 0 {
		if len(m.installed) == 0 {
			return centeredText(m.width, m.height-10, "No extensions installed\n\nSwitch to Browse mode to install extensions")
		}
		return centeredText(m.width, m.height-10, "No installed extensions match current filters")
	}

	var b strings.Builder

	visibleItems := m.height - 12
	if visibleItems < 1 {
		visibleItems = 1
	}

	start := m.offset
	end := m.offset + visibleItems
	if end > len(filtered) {
		end = len(filtered)
	}

	for i := start; i < end; i++ {
		ext := filtered[i]
		isCursor := i == m.cursor

		itemStyle := lipgloss.NewStyle()
		if isCursor {
			itemStyle = itemStyle.
				Background(theme.ColorPrimary).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Width(m.width - 4)
		}

		// Status
		status := ""
		if ext.HasUpdate {
			status = statusStyle.Render("[UPDATE]")
		} else if ext.IsObsolete {
			status = lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("[OBSOLETE]")
		} else {
			status = lipgloss.NewStyle().Foreground(theme.ColorSuccess).Render("[✓]")
		}

		// NSFW indicator
		nsfwIndicator := ""
		if ext.IsNSFW {
			nsfwIndicator = nsfwStyle.Render("[NSFW]")
		}

		// Language badge
		langBadge := lipgloss.NewStyle().
			Foreground(theme.ColorSecondary).
			Render(fmt.Sprintf("[%s]", ext.Language))

		line := fmt.Sprintf("%s %s %s %s v%s",
			status,
			ext.Name,
			langBadge,
			nsfwIndicator,
			ext.VersionName,
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
		"Tab: switch mode",
		"/: search",
		"l: language",
		"n: toggle NSFW",
	}

	if m.mode == ModeBrowse {
		controls = append(controls, "i/Enter: install")
	} else {
		controls = append(controls, "i/Enter: uninstall", "u: update", "U: update all")
	}

	controls = append(controls, "r: refresh", "Esc: back")

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

// Messages

type extensionsLoadedMsg struct {
	available []*suwayomi.Extension
	installed []*suwayomi.Extension
}

type extensionInstalledMsg struct {
	packageName string
}

type extensionErrorMsg struct {
	err error
}

// Commands

func (m Model) loadExtensions() tea.Msg {
	if m.client == nil {
		// No client available, return empty lists
		return extensionsLoadedMsg{
			available: []*suwayomi.Extension{},
			installed: []*suwayomi.Extension{},
		}
	}

	// Load available extensions
	available, err := m.client.ListAvailableExtensions()
	if err != nil {
		return extensionErrorMsg{err: fmt.Errorf("failed to load available extensions: %w", err)}
	}

	// Load installed extensions
	installed, err := m.client.ListInstalledExtensions()
	if err != nil {
		return extensionErrorMsg{err: fmt.Errorf("failed to load installed extensions: %w", err)}
	}

	return extensionsLoadedMsg{
		available: available,
		installed: installed,
	}
}

func (m Model) toggleInstall() tea.Cmd {
	var ext *suwayomi.Extension

	switch m.mode {
	case ModeBrowse:
		filtered := m.getFilteredExtensions(m.available)
		if m.cursor >= 0 && m.cursor < len(filtered) {
			ext = filtered[m.cursor]
		}
	case ModeInstalled:
		filtered := m.getFilteredExtensions(m.installed)
		if m.cursor >= 0 && m.cursor < len(filtered) {
			ext = filtered[m.cursor]
		}
	}

	if ext == nil || m.client == nil {
		return nil
	}

	return func() tea.Msg {
		var err error
		if ext.IsInstalled {
			err = m.client.UninstallExtension(ext.PkgName)
		} else {
			err = m.client.InstallExtension(ext.PkgName)
		}

		if err != nil {
			return extensionErrorMsg{err: err}
		}

		return extensionInstalledMsg{packageName: ext.PkgName}
	}
}

func (m Model) updateExtension() tea.Cmd {
	if m.mode != ModeInstalled || m.client == nil {
		return nil
	}

	filtered := m.getFilteredExtensions(m.installed)
	if m.cursor < 0 || m.cursor >= len(filtered) {
		return nil
	}

	ext := filtered[m.cursor]
	if !ext.HasUpdate {
		return nil
	}

	return func() tea.Msg {
		err := m.client.UpdateExtension(ext.PkgName)
		if err != nil {
			return extensionErrorMsg{err: err}
		}
		return extensionInstalledMsg{packageName: ext.PkgName}
	}
}

func (m Model) updateAll() tea.Cmd {
	if m.mode != ModeInstalled || m.client == nil {
		return nil
	}

	// Find all extensions with updates
	toUpdate := make([]*suwayomi.Extension, 0)
	for _, ext := range m.installed {
		if ext.HasUpdate {
			toUpdate = append(toUpdate, ext)
		}
	}

	if len(toUpdate) == 0 {
		return nil
	}

	return func() tea.Msg {
		for _, ext := range toUpdate {
			err := m.client.UpdateExtension(ext.PkgName)
			if err != nil {
				return extensionErrorMsg{err: err}
			}
		}
		return extensionInstalledMsg{packageName: "all"}
	}
}
