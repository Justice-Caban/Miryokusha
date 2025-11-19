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
	viewMode ViewMode // "main", "logs"
	logLines int      // Number of log lines to show

	// Navigation
	cursor      int  // Current cursor position in settings list
	editMode    bool // Whether we're in edit mode for the selected setting
	settingsDirty bool // Whether settings have been modified

	// Message display
	message      string // Status message to display
	messageType  string // "success", "error", "info"
	messageTimer int    // Frames remaining to show message
}

// ViewMode represents the current view mode
type ViewMode string

const (
	ViewModeMain ViewMode = "main"
	ViewModeLogs ViewMode = "logs"
)

// SettingItem represents a configurable setting
type SettingItem struct {
	ID          string
	Label       string
	Description string
	Type        SettingType
	Value       interface{}
	MinValue    int
	MaxValue    int
}

// SettingType represents the type of setting
type SettingType int

const (
	SettingTypeBoolean SettingType = iota
	SettingTypeInteger
	SettingTypeAction
)

// NewModel creates a new settings model
func NewModel(cfg *config.Config, client *suwayomi.Client, mgr *server.Manager) Model {
	return Model{
		config:         cfg,
		suwayomiClient: client,
		serverManager:  mgr,
		viewMode:       ViewModeMain,
		logLines:       20,
		cursor:         0,
		editMode:       false,
		settingsDirty:  false,
	}
}

// getSettingsList returns the list of configurable settings
func (m Model) getSettingsList() []SettingItem {
	if m.config == nil {
		return []SettingItem{}
	}

	settings := []SettingItem{
		// Server Management Actions
		{
			ID:          "health_check",
			Label:       "Perform Health Check",
			Description: "Check connection to Suwayomi server",
			Type:        SettingTypeAction,
		},
		{
			ID:          "reload_config",
			Label:       "Reload Configuration",
			Description: "Reload config file from disk",
			Type:        SettingTypeAction,
		},

		// Smart Updates Settings
		{
			ID:          "smart_update",
			Label:       "Smart Updates",
			Description: "Enable intelligent update scheduling",
			Type:        SettingTypeBoolean,
			Value:       m.config.Updates.SmartUpdate,
		},
		{
			ID:          "update_only_ongoing",
			Label:       "Update Only Ongoing",
			Description:  "Only update manga that are still ongoing",
			Type:        SettingTypeBoolean,
			Value:       m.config.Updates.UpdateOnlyOngoing,
		},
		{
			ID:          "update_only_started",
			Label:       "Update Only Started",
			Description: "Only update manga you've started reading",
			Type:        SettingTypeBoolean,
			Value:       m.config.Updates.UpdateOnlyStarted,
		},
		{
			ID:          "auto_update",
			Label:       "Auto-Update",
			Description: "Automatically check for new chapters",
			Type:        SettingTypeBoolean,
			Value:       m.config.Updates.AutoUpdateEnabled,
		},
		{
			ID:          "min_interval",
			Label:       "Min Update Interval (hours)",
			Description: "Minimum hours between updates",
			Type:        SettingTypeInteger,
			Value:       m.config.Updates.MinIntervalHours,
			MinValue:    1,
			MaxValue:    168,
		},
		{
			ID:          "auto_interval",
			Label:       "Auto-Update Interval (hours)",
			Description: "Hours between automatic update checks",
			Type:        SettingTypeInteger,
			Value:       m.config.Updates.AutoUpdateIntervalHrs,
			MinValue:    1,
			MaxValue:    168,
		},
	}

	// Add server management actions if enabled
	if m.config.ServerManagement.Enabled && m.serverManager != nil {
		status := m.serverManager.GetStatus()
		if status == server.StatusStopped {
			settings = append(settings, SettingItem{
				ID:          "start_server",
				Label:       "Start Suwayomi Server",
				Description: "Start the local Suwayomi server",
				Type:        SettingTypeAction,
			})
		} else if status == server.StatusRunning {
			settings = append(settings, SettingItem{
				ID:          "stop_server",
				Label:       "Stop Suwayomi Server",
				Description: "Stop the local Suwayomi server",
				Type:        SettingTypeAction,
			})
			settings = append(settings, SettingItem{
				ID:          "restart_server",
				Label:       "Restart Suwayomi Server",
				Description: "Restart the local Suwayomi server",
				Type:        SettingTypeAction,
			})
		}

		settings = append(settings, SettingItem{
			ID:          "view_logs",
			Label:       "View Server Logs",
			Description: "Show Suwayomi server logs",
			Type:        SettingTypeAction,
		})
	}

	return settings
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
		// Handle logs view separately
		if m.viewMode == ViewModeLogs {
			switch msg.String() {
			case "l", "L", "esc":
				m.viewMode = ViewModeMain
				return m, nil
			case "c", "C":
				if m.serverManager != nil {
					m.serverManager.ClearLogs()
					m.setMessage("Logs cleared", "success")
				}
				return m, nil
			}
			return m, nil
		}

		// Main view navigation
		settings := m.getSettingsList()
		if len(settings) == 0 {
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			if m.cursor < len(settings)-1 {
				m.cursor++
			}
			return m, nil

		case "enter", " ":
			return m.handleSettingAction(settings[m.cursor])

		case "+", "=", "right", "l":
			// Increase integer value
			setting := settings[m.cursor]
			if setting.Type == SettingTypeInteger {
				return m.adjustIntegerSetting(setting.ID, 1)
			}
			return m, nil

		case "-", "_", "left", "h":
			// Decrease integer value
			setting := settings[m.cursor]
			if setting.Type == SettingTypeInteger {
				return m.adjustIntegerSetting(setting.ID, -1)
			}
			return m, nil

		case "s", "S":
			// Save settings
			if m.settingsDirty {
				if err := m.saveConfig(); err != nil {
					m.setMessage(fmt.Sprintf("Failed to save: %v", err), "error")
				} else {
					m.setMessage("Settings saved successfully", "success")
					m.settingsDirty = false
				}
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
		b.WriteString("\n\n")
	}

	// Show different views based on mode
	switch m.viewMode {
	case ViewModeLogs:
		b.WriteString(m.renderServerLogs())
	default:
		// Show server health if available
		if m.serverInfo != nil || m.checkingHealth || m.healthError != nil {
			b.WriteString(m.renderServerHealth())
			b.WriteString("\n\n")
		}

		// Dirty indicator
		if m.settingsDirty {
			b.WriteString(lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("● Unsaved changes"))
			b.WriteString("\n\n")
		}

		// Interactive settings list
		b.WriteString(m.renderSettingsList())
	}

	b.WriteString("\n")

	// Footer
	b.WriteString(m.renderFooter())

	// Apply consistent horizontal padding/centering
	content := b.String()
	maxWidth := 100
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
	b.WriteString(m.renderConfigLine("Config Path", config.GetConfigPath()))
	b.WriteString(m.renderConfigLine("Database Path", m.config.Paths.Database))
	b.WriteString(m.renderConfigLine("Cache Path", m.config.Paths.Cache))
	b.WriteString(m.renderConfigLine("Downloads Path", m.config.Paths.Downloads))

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
			"l/Esc: back",
			"c: clear logs",
		}
	default:
		settings := m.getSettingsList()
		if len(settings) > 0 && m.cursor < len(settings) {
			currentSetting := settings[m.cursor]

			controls = []string{
				"↑↓/jk: navigate",
				"Enter: activate",
			}

			if currentSetting.Type == SettingTypeInteger {
				controls = append(controls, "+/-/←→: adjust")
			}

			if m.settingsDirty {
				controls = append(controls, "s: save")
			}
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

// handleSettingAction handles Enter key press on a setting
func (m Model) handleSettingAction(setting SettingItem) (Model, tea.Cmd) {
	if m.config == nil {
		return m, nil
	}

	switch setting.ID {
	case "health_check":
		return m, m.performHealthCheck

	case "reload_config":
		cfg, err := config.Load()
		if err == nil {
			m.config = cfg
			m.settingsDirty = false
			m.setMessage("Configuration reloaded", "success")
		} else {
			m.setMessage(fmt.Sprintf("Failed to reload: %v", err), "error")
		}
		return m, nil

	case "start_server":
		if m.serverManager != nil {
			if err := m.serverManager.Start(); err != nil {
				m.setMessage(fmt.Sprintf("Failed to start: %v", err), "error")
			} else {
				m.setMessage("Server starting...", "success")
			}
		}
		return m, nil

	case "stop_server":
		if m.serverManager != nil {
			if err := m.serverManager.Stop(); err != nil {
				m.setMessage(fmt.Sprintf("Failed to stop: %v", err), "error")
			} else {
				m.setMessage("Server stopped", "success")
			}
		}
		return m, nil

	case "restart_server":
		if m.serverManager != nil {
			if err := m.serverManager.Restart(); err != nil {
				m.setMessage(fmt.Sprintf("Failed to restart: %v", err), "error")
			} else {
				m.setMessage("Server restarting...", "success")
			}
		}
		return m, nil

	case "view_logs":
		m.viewMode = ViewModeLogs
		return m, nil

	case "smart_update":
		m.config.Updates.SmartUpdate = !m.config.Updates.SmartUpdate
		m.settingsDirty = true
		return m, nil

	case "update_only_ongoing":
		m.config.Updates.UpdateOnlyOngoing = !m.config.Updates.UpdateOnlyOngoing
		m.settingsDirty = true
		return m, nil

	case "update_only_started":
		m.config.Updates.UpdateOnlyStarted = !m.config.Updates.UpdateOnlyStarted
		m.settingsDirty = true
		return m, nil

	case "auto_update":
		m.config.Updates.AutoUpdateEnabled = !m.config.Updates.AutoUpdateEnabled
		m.settingsDirty = true
		return m, nil
	}

	return m, nil
}

// adjustIntegerSetting adjusts an integer setting by a delta
func (m Model) adjustIntegerSetting(settingID string, delta int) (Model, tea.Cmd) {
	if m.config == nil {
		return m, nil
	}

	switch settingID {
	case "min_interval":
		newValue := m.config.Updates.MinIntervalHours + delta
		if newValue >= 1 && newValue <= 168 {
			m.config.Updates.MinIntervalHours = newValue
			m.settingsDirty = true
		}

	case "auto_interval":
		newValue := m.config.Updates.AutoUpdateIntervalHrs + delta
		if newValue >= 1 && newValue <= 168 {
			m.config.Updates.AutoUpdateIntervalHrs = newValue
			m.settingsDirty = true
		}
	}

	return m, nil
}

// renderSettingsList renders the interactive settings list
func (m Model) renderSettingsList() string {
	var b strings.Builder

	settings := m.getSettingsList()
	if len(settings) == 0 {
		return mutedStyle.Render("No settings available")
	}

	// Render each setting with cursor highlighting
	for i, setting := range settings {
		isCursor := i == m.cursor

		// Build the setting line
		var line string
		switch setting.Type {
		case SettingTypeBoolean:
			boolVal, _ := setting.Value.(bool)
			status := "Disabled"
			if boolVal {
				status = successStyle.Render("✓ Enabled")
			}
			line = fmt.Sprintf("%s: %s", setting.Label, status)

		case SettingTypeInteger:
			intVal, _ := setting.Value.(int)
			line = fmt.Sprintf("%s: %d", setting.Label, intVal)

		case SettingTypeAction:
			line = setting.Label + " →"
		}

		// Apply cursor style
		if isCursor {
			line = theme.HighlightStyle.Render("▸ " + line)
		} else {
			line = "  " + line
		}

		b.WriteString(line)
		b.WriteString("\n")

		// Show description for selected item
		if isCursor && setting.Description != "" {
			desc := mutedStyle.Render("  " + setting.Description)
			b.WriteString(desc)
			b.WriteString("\n")
		}
	}

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
