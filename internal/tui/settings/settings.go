package settings

import (
	"fmt"
	"strings"
	"time"

	"github.com/Justice-Caban/Miryokusha/internal/config"
	"github.com/Justice-Caban/Miryokusha/internal/server"
	"github.com/Justice-Caban/Miryokusha/internal/suwayomi"
	"github.com/Justice-Caban/Miryokusha/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.ColorPrimary).
			MarginBottom(1)

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.ColorSecondary).
			MarginTop(1).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(theme.ColorAccent).
			Width(20)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	helpStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted).
			MarginTop(1)

	successStyle = lipgloss.NewStyle().
			Foreground(theme.ColorSuccess).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(theme.ColorError).
			Bold(true)

	mutedStyle = lipgloss.NewStyle().
			Foreground(theme.ColorMuted)
)

// Model represents the settings view model
type Model struct {
	width  int
	height int

	config         *config.Config
	suwayomiClient *suwayomi.Client
	serverManager  *server.Manager

	// Server health
	serverInfo      *suwayomi.ServerInfo
	checkingHealth  bool
	lastHealthCheck time.Time
	healthError     error

	// View mode
	viewMode ViewMode // "main", "logs", "edit_updates"
	logLines int      // Number of log lines to show

	// Edit mode
	message      string // Status message to display
	messageType  string // "success", "error", "info"
	messageTimer int    // Frames remaining to show message
}

// ViewMode represents the current view mode
type ViewMode string

const (
	ViewModeMain        ViewMode = "main"
	ViewModeLogs        ViewMode = "logs"
	ViewModeEditUpdates ViewMode = "edit_updates"
)

// NewModel creates a new settings model
func NewModel(cfg *config.Config, client *suwayomi.Client, mgr *server.Manager) Model {
	return Model{
		config:         cfg,
		suwayomiClient: client,
		serverManager:  mgr,
		viewMode:       ViewModeMain,
		logLines:       20,
	}
}

// Init initializes the settings model
func (m Model) Init() tea.Cmd {
	return m.performHealthCheck
}

// Update handles messages for the settings view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Handle edit mode separately
		if m.viewMode == ViewModeEditUpdates {
			return m.handleEditUpdatesInput(msg)
		}

		switch msg.String() {
		case "h", "H":
			// Perform health check
			return m, m.performHealthCheck

		case "r", "R":
			// Reload configuration
			cfg, err := config.Load()
			if err == nil {
				m.config = cfg
				m.setMessage("Configuration reloaded", "success")
			} else {
				m.setMessage(fmt.Sprintf("Failed to reload: %v", err), "error")
			}
			return m, nil

		case "e", "E":
			// Enter edit updates mode
			if m.viewMode == ViewModeMain {
				m.viewMode = ViewModeEditUpdates
			}
			return m, nil

		case "s", "S":
			// Start server
			if m.serverManager != nil && m.config.ServerManagement.Enabled {
				if m.serverManager.GetStatus() == server.StatusStopped {
					if err := m.serverManager.Start(); err != nil {
						m.setMessage(fmt.Sprintf("Failed to start: %v", err), "error")
					} else {
						m.setMessage("Server starting...", "success")
					}
				}
			}
			return m, nil

		case "x", "X":
			// Stop server
			if m.serverManager != nil {
				if m.serverManager.GetStatus() == server.StatusRunning {
					if err := m.serverManager.Stop(); err != nil {
						m.setMessage(fmt.Sprintf("Failed to stop: %v", err), "error")
					} else {
						m.setMessage("Server stopped", "success")
					}
				}
			}
			return m, nil

		case "t", "T":
			// Restart server
			if m.serverManager != nil && m.config.ServerManagement.Enabled {
				if err := m.serverManager.Restart(); err != nil {
					m.setMessage(fmt.Sprintf("Failed to restart: %v", err), "error")
				} else {
					m.setMessage("Server restarting...", "success")
				}
			}
			return m, nil

		case "l", "L":
			// Toggle logs view
			if m.viewMode == ViewModeMain {
				m.viewMode = ViewModeLogs
			} else {
				m.viewMode = ViewModeMain
			}
			return m, nil

		case "c", "C":
			// Clear logs
			if m.serverManager != nil {
				m.serverManager.ClearLogs()
				m.setMessage("Logs cleared", "success")
			}
			return m, nil
		}

	case healthCheckResultMsg:
		m.checkingHealth = false
		m.serverInfo = msg.info
		m.healthError = msg.err
		m.lastHealthCheck = time.Now()
		return m, nil
	}

	return m, nil
}

// View renders the settings view
func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("⚙️  Settings"))
	b.WriteString("\n\n")

	// Show message if present
	if m.messageTimer > 0 && m.message != "" {
		b.WriteString(m.renderMessage())
		b.WriteString("\n")
	}

	// Show different views based on mode
	switch m.viewMode {
	case ViewModeLogs:
		b.WriteString(m.renderServerLogs())
	case ViewModeEditUpdates:
		b.WriteString(m.renderEditUpdates())
	default:
		// Server Configuration Section
		b.WriteString(m.renderServerConfig())
		b.WriteString("\n")

		// Server Management Section
		if m.config != nil && m.config.ServerManagement.Enabled {
			b.WriteString(m.renderServerManagement())
			b.WriteString("\n")
		}

		// Server Health Section
		b.WriteString(m.renderServerHealth())
		b.WriteString("\n")

		// Smart Updates Section
		b.WriteString(m.renderSmartUpdates())
		b.WriteString("\n")

		// Application Info Section
		b.WriteString(m.renderAppInfo())
	}

	b.WriteString("\n")

	// Footer
	b.WriteString(m.renderFooter())

	return b.String()
}

// renderServerConfig renders the server configuration section
func (m Model) renderServerConfig() string {
	var b strings.Builder

	b.WriteString(sectionStyle.Render("Server Configuration"))
	b.WriteString("\n")

	if m.config == nil {
		b.WriteString(mutedStyle.Render("No configuration loaded"))
		return b.String()
	}

	// Get default server
	defaultServer := m.config.GetDefaultServer()
	if defaultServer != nil {
		b.WriteString(m.renderConfigLine("Server Name", defaultServer.Name))
		b.WriteString(m.renderConfigLine("Server URL", defaultServer.URL))
		b.WriteString(m.renderConfigLine("Default", "Yes"))

		if defaultServer.Auth != nil {
			b.WriteString(m.renderConfigLine("Auth Type", defaultServer.Auth.Type))
			b.WriteString(m.renderConfigLine("Username", defaultServer.Auth.Username))
		} else {
			b.WriteString(m.renderConfigLine("Authentication", "None"))
		}
	} else {
		b.WriteString(mutedStyle.Render("No server configured"))
	}

	return b.String()
}

// renderServerHealth renders the server health section
func (m Model) renderServerHealth() string {
	var b strings.Builder

	b.WriteString(sectionStyle.Render("Server Health"))
	b.WriteString("\n")

	if m.checkingHealth {
		b.WriteString(mutedStyle.Render("⟳ Checking server health..."))
		return b.String()
	}

	if m.healthError != nil {
		b.WriteString(errorStyle.Render("✗ Health check failed"))
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render(fmt.Sprintf("Error: %v", m.healthError)))
		return b.String()
	}

	if m.serverInfo == nil {
		b.WriteString(mutedStyle.Render("Press 'h' to perform health check"))
		return b.String()
	}

	// Show server info
	if m.serverInfo.IsHealthy {
		b.WriteString(successStyle.Render("✓ Server is healthy"))
		b.WriteString("\n")
	} else {
		b.WriteString(errorStyle.Render("✗ Server is not responding"))
		b.WriteString("\n")
	}

	if m.serverInfo.IsHealthy {
		b.WriteString(m.renderConfigLine("Version", m.serverInfo.Version))
		b.WriteString(m.renderConfigLine("Build Type", m.serverInfo.BuildType))
		b.WriteString(m.renderConfigLine("Revision", m.serverInfo.Revision))
		b.WriteString(m.renderConfigLine("Extensions", fmt.Sprintf("%d", m.serverInfo.ExtensionCount)))
		b.WriteString(m.renderConfigLine("Manga", fmt.Sprintf("%d", m.serverInfo.MangaCount)))
	}

	if !m.lastHealthCheck.IsZero() {
		b.WriteString("\n")
		b.WriteString(mutedStyle.Render(fmt.Sprintf("Last checked: %s", m.formatRelativeTime(m.lastHealthCheck))))
	}

	return b.String()
}

// renderSmartUpdates renders the smart updates configuration section
func (m Model) renderSmartUpdates() string {
	var b strings.Builder

	b.WriteString(sectionStyle.Render("Smart Updates (Mihon-style)"))
	b.WriteString("\n")

	if m.config == nil {
		b.WriteString(mutedStyle.Render("No configuration loaded"))
		return b.String()
	}

	// Smart update enabled
	enabledValue := "Disabled"
	if m.config.Updates.SmartUpdate {
		enabledValue = successStyle.Render("Enabled ✓")
	}
	b.WriteString(m.renderConfigLine("Smart Updates", enabledValue))

	// Only show details if smart updates are enabled
	if m.config.Updates.SmartUpdate {
		b.WriteString(m.renderConfigLine("Min Interval", fmt.Sprintf("%d hours", m.config.Updates.MinIntervalHours)))

		onlyOngoingValue := "No"
		if m.config.Updates.UpdateOnlyOngoing {
			onlyOngoingValue = "Yes"
		}
		b.WriteString(m.renderConfigLine("Only Ongoing", onlyOngoingValue))

		onlyStartedValue := "No"
		if m.config.Updates.UpdateOnlyStarted {
			onlyStartedValue = "Yes"
		}
		b.WriteString(m.renderConfigLine("Only Started", onlyStartedValue))

		b.WriteString(m.renderConfigLine("Max Failures", fmt.Sprintf("%d", m.config.Updates.MaxConsecutiveFailures)))
		b.WriteString(m.renderConfigLine("Interval Multiplier", fmt.Sprintf("%.1fx", m.config.Updates.IntervalMultiplier)))
	}

	// Auto-update status
	b.WriteString("\n")
	autoUpdateValue := "Disabled"
	if m.config.Updates.AutoUpdateEnabled {
		autoUpdateValue = successStyle.Render(fmt.Sprintf("Enabled (every %dh)", m.config.Updates.AutoUpdateIntervalHrs))
	}
	b.WriteString(m.renderConfigLine("Auto-Update", autoUpdateValue))

	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Press 'e' to edit these settings interactively"))

	return b.String()
}

// renderAppInfo renders the application info section
func (m Model) renderAppInfo() string {
	var b strings.Builder

	b.WriteString(sectionStyle.Render("Application"))
	b.WriteString("\n")

	b.WriteString(m.renderConfigLine("Name", "Miryokusha"))
	b.WriteString(m.renderConfigLine("Version", "0.1.0-dev"))
	b.WriteString(m.renderConfigLine("License", "GPLv3"))
	b.WriteString(m.renderConfigLine("Config Path", "~/.config/miryokusha/config.yaml"))
	b.WriteString(m.renderConfigLine("Data Path", "~/.local/share/miryokusha/"))

	return b.String()
}

// renderServerManagement renders the server management section
func (m Model) renderServerManagement() string {
	var b strings.Builder

	b.WriteString(sectionStyle.Render("Server Management"))
	b.WriteString("\n")

	if m.serverManager == nil {
		b.WriteString(mutedStyle.Render("Server management not available"))
		return b.String()
	}

	status := m.serverManager.GetStatus()
	pid := m.serverManager.GetPID()
	uptime := m.serverManager.GetUptime()

	// Status display
	var statusDisplay string
	switch status {
	case server.StatusRunning:
		statusDisplay = successStyle.Render("● Running")
	case server.StatusStarting:
		statusDisplay = lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("⟳ Starting")
	case server.StatusStopping:
		statusDisplay = lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("⟳ Stopping")
	case server.StatusStopped:
		statusDisplay = mutedStyle.Render("○ Stopped")
	case server.StatusError:
		statusDisplay = errorStyle.Render("✗ Error")
	}

	b.WriteString(m.renderConfigLine("Status", statusDisplay))

	if status == server.StatusRunning {
		b.WriteString(m.renderConfigLine("PID", fmt.Sprintf("%d", pid)))
		b.WriteString(m.renderConfigLine("Uptime", m.formatDuration(uptime)))
	}

	b.WriteString(m.renderConfigLine("Executable", m.config.ServerManagement.ExecutablePath))
	b.WriteString(m.renderConfigLine("Auto-start", fmt.Sprintf("%v", m.config.ServerManagement.AutoStart)))

	return b.String()
}

// renderServerLogs renders the server logs view
func (m Model) renderServerLogs() string {
	var b strings.Builder

	b.WriteString(sectionStyle.Render("Server Logs"))
	b.WriteString("\n")

	if m.serverManager == nil {
		b.WriteString(mutedStyle.Render("No server manager available"))
		return b.String()
	}

	logs := m.serverManager.GetLogs(m.logLines)
	if len(logs) == 0 {
		b.WriteString(mutedStyle.Render("No logs available"))
	} else {
		for _, log := range logs {
			b.WriteString(mutedStyle.Render(log))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderConfigLine renders a configuration line
func (m Model) renderConfigLine(label, value string) string {
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		labelStyle.Render(label+":"),
		valueStyle.Render(value),
	) + "\n"
}

// renderFooter renders the footer with controls
func (m Model) renderFooter() string {
	var controls []string

	switch m.viewMode {
	case ViewModeLogs:
		controls = []string{
			"l: back to main",
			"c: clear logs",
			"Esc: back",
		}
	case ViewModeEditUpdates:
		controls = []string{
			"1-6: toggle settings",
			"+/-: adjust values",
			"s: save",
			"Esc: cancel",
		}
	default:
		controls = []string{
			"h: health check",
			"r: reload config",
			"e: edit updates",
		}

		if m.config != nil && m.config.ServerManagement.Enabled && m.serverManager != nil {
			status := m.serverManager.GetStatus()
			if status == server.StatusStopped {
				controls = append(controls, "s: start server")
			} else if status == server.StatusRunning {
				controls = append(controls, "x: stop server", "t: restart")
			}
			controls = append(controls, "l: view logs")
		}

		controls = append(controls, "Esc: back")
	}

	return helpStyle.Render(strings.Join(controls, " • "))
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

// formatRelativeTime formats a time relative to now
func (m Model) formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// setMessage sets a status message to display
func (m *Model) setMessage(msg, msgType string) {
	m.message = msg
	m.messageType = msgType
	m.messageTimer = 180 // Display for ~3 seconds (60 fps)
}

// renderMessage renders the status message with appropriate styling
func (m Model) renderMessage() string {
	if m.message == "" {
		return ""
	}

	var style lipgloss.Style
	switch m.messageType {
	case "success":
		style = successStyle
	case "error":
		style = errorStyle
	default:
		style = mutedStyle
	}

	return style.Render(m.message)
}

// handleEditUpdatesInput handles keyboard input in edit mode
func (m Model) handleEditUpdatesInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.config == nil {
		return m, nil
	}

	switch msg.String() {
	case "esc":
		// Cancel editing and return to main view
		m.viewMode = ViewModeMain
		return m, nil

	case "1":
		// Toggle smart updates
		m.config.Updates.SmartUpdate = !m.config.Updates.SmartUpdate

	case "2":
		// Toggle update only ongoing
		m.config.Updates.UpdateOnlyOngoing = !m.config.Updates.UpdateOnlyOngoing

	case "3":
		// Toggle update only started
		m.config.Updates.UpdateOnlyStarted = !m.config.Updates.UpdateOnlyStarted

	case "4":
		// Toggle auto-update
		m.config.Updates.AutoUpdateEnabled = !m.config.Updates.AutoUpdateEnabled

	case "+", "=":
		// Increase min interval hours
		if m.config.Updates.MinIntervalHours < 168 {
			m.config.Updates.MinIntervalHours++
		}

	case "-", "_":
		// Decrease min interval hours
		if m.config.Updates.MinIntervalHours > 1 {
			m.config.Updates.MinIntervalHours--
		}

	case "s", "S":
		// Save configuration
		if err := m.saveConfig(); err != nil {
			m.setMessage(fmt.Sprintf("Failed to save: %v", err), "error")
		} else {
			m.setMessage("Configuration saved successfully", "success")
			m.viewMode = ViewModeMain
		}
	}

	return m, nil
}

// renderEditUpdates renders the edit updates interface
func (m Model) renderEditUpdates() string {
	var b strings.Builder

	b.WriteString(sectionStyle.Render("Edit Smart Updates Settings"))
	b.WriteString("\n\n")

	if m.config == nil {
		b.WriteString(mutedStyle.Render("No configuration loaded"))
		return b.String()
	}

	// Smart update toggle
	smartUpdateValue := "Disabled"
	if m.config.Updates.SmartUpdate {
		smartUpdateValue = successStyle.Render("Enabled ✓")
	}
	b.WriteString(m.renderConfigLine("[1] Smart Updates", smartUpdateValue))

	// Only ongoing toggle
	onlyOngoingValue := "No"
	if m.config.Updates.UpdateOnlyOngoing {
		onlyOngoingValue = successStyle.Render("Yes ✓")
	}
	b.WriteString(m.renderConfigLine("[2] Only Ongoing", onlyOngoingValue))

	// Only started toggle
	onlyStartedValue := "No"
	if m.config.Updates.UpdateOnlyStarted {
		onlyStartedValue = successStyle.Render("Yes ✓")
	}
	b.WriteString(m.renderConfigLine("[3] Only Started", onlyStartedValue))

	// Auto-update toggle
	autoUpdateValue := "Disabled"
	if m.config.Updates.AutoUpdateEnabled {
		autoUpdateValue = successStyle.Render("Enabled ✓")
	}
	b.WriteString(m.renderConfigLine("[4] Auto-Update", autoUpdateValue))

	b.WriteString("\n")

	// Min interval hours (adjustable with +/-)
	b.WriteString(m.renderConfigLine("[+/-] Min Interval", fmt.Sprintf("%d hours", m.config.Updates.MinIntervalHours)))

	// Read-only settings
	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("Advanced Settings (edit config.yaml)"))
	b.WriteString("\n")
	b.WriteString(m.renderConfigLine("Max Failures", fmt.Sprintf("%d", m.config.Updates.MaxConsecutiveFailures)))
	b.WriteString(m.renderConfigLine("Interval Multiplier", fmt.Sprintf("%.1fx", m.config.Updates.IntervalMultiplier)))
	if m.config.Updates.AutoUpdateEnabled {
		b.WriteString(m.renderConfigLine("Auto-Update Interval", fmt.Sprintf("%d hours", m.config.Updates.AutoUpdateIntervalHrs)))
	}

	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Press numbers to toggle • +/- to adjust interval • s to save • Esc to cancel"))

	return b.String()
}

// saveConfig saves the current configuration to disk
func (m Model) saveConfig() error {
	if m.config == nil {
		return fmt.Errorf("no configuration loaded")
	}

	return config.Save(m.config)
}

// Messages

type healthCheckResultMsg struct {
	info *suwayomi.ServerInfo
	err  error
}

// Commands

func (m Model) performHealthCheck() tea.Msg {
	if m.suwayomiClient == nil {
		return healthCheckResultMsg{
			info: &suwayomi.ServerInfo{IsHealthy: false},
			err:  fmt.Errorf("no Suwayomi client configured"),
		}
	}

	info, err := m.suwayomiClient.HealthCheck()
	return healthCheckResultMsg{
		info: info,
		err:  err,
	}
}
