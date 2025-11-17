// Critical errors that need user attention
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ErrorNotification represents a user-facing error that requires attention
type ErrorNotification struct {
	Title       string // Brief title (e.g., "Storage Error")
	Message     string // Detailed message
	Severity    ErrorSeverity
	Actionable  bool   // Whether user can fix this
	Suggestion  string // What user should do
	Dismissible bool   // Can be dismissed
}

type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

// ErrorNotificationList manages multiple error notifications
type ErrorNotificationList struct {
	notifications []ErrorNotification
}

// Add adds a new error notification
func (e *ErrorNotificationList) Add(notification ErrorNotification) {
	e.notifications = append(e.notifications, notification)
}

// AddError adds an error with custom fields
func (e *ErrorNotificationList) AddError(title, message, suggestion string, severity ErrorSeverity) {
	e.Add(ErrorNotification{
		Title:       title,
		Message:     message,
		Severity:    severity,
		Actionable:  suggestion != "",
		Suggestion:  suggestion,
		Dismissible: severity != SeverityCritical,
	})
}

// HasErrors returns true if there are any notifications
func (e *ErrorNotificationList) HasErrors() bool {
	return len(e.notifications) > 0
}

// HasCritical returns true if there are critical errors
func (e *ErrorNotificationList) HasCritical() bool {
	for _, n := range e.notifications {
		if n.Severity == SeverityCritical {
			return true
		}
	}
	return false
}

// Count returns the number of notifications
func (e *ErrorNotificationList) Count() int {
	return len(e.notifications)
}

// Clear removes all notifications
func (e *ErrorNotificationList) Clear() {
	e.notifications = nil
}

// Render renders all notifications as a styled string
func (e *ErrorNotificationList) Render(width int) string {
	if len(e.notifications) == 0 {
		return ""
	}

	var sections []string

	for i, notif := range e.notifications {
		sections = append(sections, e.renderNotification(notif, width, i))
	}

	return strings.Join(sections, "\n\n")
}

func (e *ErrorNotificationList) renderNotification(notif ErrorNotification, width int, index int) string {
	// Choose style based on severity
	var (
		iconStyle  lipgloss.Style
		borderCol  lipgloss.Color
		icon       string
	)

	switch notif.Severity {
	case SeverityInfo:
		iconStyle = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
		borderCol = ColorSecondary
		icon = "ℹ"
	case SeverityWarning:
		iconStyle = lipgloss.NewStyle().Foreground(ColorWarning).Bold(true)
		borderCol = ColorWarning
		icon = "⚠"
	case SeverityError:
		iconStyle = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
		borderCol = ColorError
		icon = "✗"
	case SeverityCritical:
		iconStyle = lipgloss.NewStyle().Foreground(ColorError).Bold(true).Underline(true)
		borderCol = ColorError
		icon = "🛑"
	}

	// Build notification content
	var content strings.Builder

	// Title with icon
	titleLine := iconStyle.Render(fmt.Sprintf("%s %s", icon, notif.Title))
	content.WriteString(titleLine)
	content.WriteString("\n\n")

	// Message
	msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	content.WriteString(msgStyle.Render(notif.Message))

	// Suggestion if present
	if notif.Suggestion != "" {
		content.WriteString("\n\n")
		suggestionStyle := lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true)
		content.WriteString(suggestionStyle.Render("→ " + notif.Suggestion))
	}

	// Wrap in box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderCol).
		Padding(1, 2).
		Width(width - 4)

	return boxStyle.Render(content.String())
}

// GetAll returns all notifications
func (e *ErrorNotificationList) GetAll() []ErrorNotification {
	return e.notifications
}
