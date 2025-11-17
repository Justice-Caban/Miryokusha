package categories

import (
	"fmt"
	"strings"

	"github.com/Justice-Caban/Miryokusha/internal/storage"
	"github.com/Justice-Caban/Miryokusha/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViewMode represents the current view mode
type ViewMode string

const (
	ViewModeList   ViewMode = "list"   // List all categories
	ViewModeCreate ViewMode = "create" // Create new category
	ViewModeEdit   ViewMode = "edit"   // Edit category
)

// Model represents the categories view model
type Model struct {
	width  int
	height int

	storage *storage.Storage

	// Category list
	categories     []*storage.Category
	cursor         int
	selectedCategory *storage.Category

	// View mode
	viewMode ViewMode

	// Input fields
	inputValue    string
	inputCursor   int
	isDefault     bool

	// Status
	message      string
	messageType  string // "success", "error", ""
}

// NewModel creates a new categories model
func NewModel(st *storage.Storage) Model {
	return Model{
		storage:  st,
		viewMode: ViewModeList,
	}
}

// Init initializes the categories model
func (m Model) Init() tea.Cmd {
	return m.loadCategories
}

// Update handles messages for the categories view
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case categoriesLoadedMsg:
		if msg.err != nil {
			m.message = fmt.Sprintf("Failed to load categories: %v", msg.err)
			m.messageType = "error"
			m.categories = []*storage.Category{}
		} else {
			m.categories = msg.categories
			// Clear any previous error messages on successful load
			if m.messageType == "error" {
				m.message = ""
				m.messageType = ""
			}
		}
		return m, nil

	case categoryOperationMsg:
		m.message = msg.message
		m.messageType = msg.messageType
		return m, m.loadCategories

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch m.viewMode {
	case ViewModeList:
		return m.handleListKeys(msg)
	case ViewModeCreate, ViewModeEdit:
		return m.handleInputKeys(msg)
	}
	return m, nil
}

// handleListKeys handles keyboard input in list mode
func (m Model) handleListKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.categories)-1 {
			m.cursor++
		}

	case "n", "N":
		// Create new category
		m.viewMode = ViewModeCreate
		m.inputValue = ""
		m.inputCursor = 0
		m.isDefault = false
		m.message = ""

	case "e", "E":
		// Edit selected category
		if m.cursor < len(m.categories) {
			m.selectedCategory = m.categories[m.cursor]
			m.viewMode = ViewModeEdit
			m.inputValue = m.selectedCategory.Name
			m.inputCursor = len(m.inputValue)
			m.isDefault = m.selectedCategory.IsDefault
			m.message = ""
		}

	case "d", "D":
		// Delete selected category
		if m.cursor < len(m.categories) {
			return m, m.deleteCategory(m.categories[m.cursor].ID)
		}

	case "ctrl+up":
		// Move category up
		if m.cursor > 0 {
			return m, m.moveCategoryUp(m.cursor)
		}

	case "ctrl+down":
		// Move category down
		if m.cursor < len(m.categories)-1 {
			return m, m.moveCategoryDown(m.cursor)
		}

	case "r", "R":
		// Refresh
		m.message = ""
		return m, m.loadCategories
	}

	return m, nil
}

// handleInputKeys handles keyboard input in create/edit mode
func (m Model) handleInputKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel
		m.viewMode = ViewModeList
		m.message = ""
		return m, nil

	case "enter":
		// Save
		if len(strings.TrimSpace(m.inputValue)) == 0 {
			m.message = "Category name cannot be empty"
			m.messageType = "error"
			return m, nil
		}

		if m.viewMode == ViewModeCreate {
			return m, m.createCategory(m.inputValue, m.isDefault)
		} else {
			return m, m.updateCategory(m.selectedCategory.ID, m.inputValue, m.isDefault)
		}

	case "tab":
		// Toggle default flag
		m.isDefault = !m.isDefault

	case "backspace":
		if m.inputCursor > 0 {
			m.inputValue = m.inputValue[:m.inputCursor-1] + m.inputValue[m.inputCursor:]
			m.inputCursor--
		}

	case "left":
		if m.inputCursor > 0 {
			m.inputCursor--
		}

	case "right":
		if m.inputCursor < len(m.inputValue) {
			m.inputCursor++
		}

	default:
		// Insert character
		if len(msg.String()) == 1 {
			m.inputValue = m.inputValue[:m.inputCursor] + msg.String() + m.inputValue[m.inputCursor:]
			m.inputCursor++
		}
	}

	return m, nil
}

// View renders the categories view
func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(theme.TitleStyle.Render("📂 Categories"))
	b.WriteString("\n\n")

	// Show different views based on mode
	if m.viewMode == ViewModeList {
		b.WriteString(m.renderCategoryList())
	} else {
		b.WriteString(m.renderCategoryInput())
	}

	b.WriteString("\n")

	// Message
	if m.message != "" {
		b.WriteString("\n")
		if m.messageType == "error" {
			b.WriteString(theme.ErrorStyle.Render(m.message))
		} else if m.messageType == "success" {
			b.WriteString(theme.SuccessStyle.Render(m.message))
		} else {
			b.WriteString(m.message)
		}
	}

	b.WriteString("\n")

	// Footer
	b.WriteString(m.renderFooter())

	return b.String()
}

// renderCategoryList renders the category list
func (m Model) renderCategoryList() string {
	var b strings.Builder

	if len(m.categories) == 0 {
		b.WriteString(theme.MutedStyle.Render("No categories yet. Press 'n' to create one."))
		return b.String()
	}

	for i, cat := range m.categories {
		isCursor := i == m.cursor

		// Format category line
		var line string
		if cat.IsDefault {
			line = fmt.Sprintf("★ %s (%d)", cat.Name, cat.MangaCount)
		} else {
			line = fmt.Sprintf("  %s (%d)", cat.Name, cat.MangaCount)
		}

		// Apply style
		if isCursor {
			b.WriteString(theme.HighlightStyle.Render(line))
		} else {
			b.WriteString(lipgloss.NewStyle().Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderCategoryInput renders the create/edit input form
func (m Model) renderCategoryInput() string {
	var b strings.Builder

	if m.viewMode == ViewModeCreate {
		b.WriteString(theme.SectionStyle.Render("Create New Category"))
	} else {
		b.WriteString(theme.SectionStyle.Render("Edit Category"))
	}
	b.WriteString("\n\n")

	// Name input
	b.WriteString("Name: ")
	b.WriteString(m.inputValue[:m.inputCursor])
	b.WriteString("█") // Cursor
	b.WriteString(m.inputValue[m.inputCursor:])
	b.WriteString("\n\n")

	// Default toggle
	if m.isDefault {
		b.WriteString("[✓] Set as default category")
	} else {
		b.WriteString("[ ] Set as default category")
	}
	b.WriteString("\n")
	b.WriteString(theme.MutedStyle.Render("(Press Tab to toggle)"))

	return b.String()
}

// renderFooter renders the footer with controls
func (m Model) renderFooter() string {
	var controls []string

	if m.viewMode == ViewModeList {
		controls = []string{
			"↑↓/jk: navigate",
			"n: new",
			"e: edit",
			"d: delete",
			"ctrl+↑↓: reorder",
			"r: refresh",
			"Esc: back",
		}
	} else {
		controls = []string{
			"Enter: save",
			"Tab: toggle default",
			"Esc: cancel",
		}
	}

	return theme.HelpStyle.Render(strings.Join(controls, " • "))
}

// Messages

type categoriesLoadedMsg struct {
	categories []*storage.Category
	err        error
}

type categoryOperationMsg struct {
	message     string
	messageType string
}

// Commands

func (m Model) loadCategories() tea.Msg {
	if m.storage == nil {
		return categoriesLoadedMsg{
			categories: []*storage.Category{},
			err:        fmt.Errorf("storage not initialized"),
		}
	}

	categories, err := m.storage.Categories.GetAll()
	if err != nil {
		return categoriesLoadedMsg{
			categories: []*storage.Category{},
			err:        err,
		}
	}

	return categoriesLoadedMsg{
		categories: categories,
		err:        nil,
	}
}

func (m Model) createCategory(name string, isDefault bool) tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil {
			return categoryOperationMsg{message: "Storage not available", messageType: "error"}
		}

		_, err := m.storage.Categories.Create(name, isDefault)
		if err != nil {
			return categoryOperationMsg{message: fmt.Sprintf("Failed to create category: %v", err), messageType: "error"}
		}

		m.viewMode = ViewModeList
		return categoryOperationMsg{message: fmt.Sprintf("Category '%s' created", name), messageType: "success"}
	}
}

func (m Model) updateCategory(id int, name string, isDefault bool) tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil {
			return categoryOperationMsg{message: "Storage not available", messageType: "error"}
		}

		err := m.storage.Categories.Update(id, name, isDefault)
		if err != nil {
			return categoryOperationMsg{message: fmt.Sprintf("Failed to update category: %v", err), messageType: "error"}
		}

		m.viewMode = ViewModeList
		return categoryOperationMsg{message: fmt.Sprintf("Category '%s' updated", name), messageType: "success"}
	}
}

func (m Model) deleteCategory(id int) tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil {
			return categoryOperationMsg{message: "Storage not available", messageType: "error"}
		}

		err := m.storage.Categories.Delete(id)
		if err != nil {
			return categoryOperationMsg{message: fmt.Sprintf("Failed to delete category: %v", err), messageType: "error"}
		}

		return categoryOperationMsg{message: "Category deleted", messageType: "success"}
	}
}

func (m Model) moveCategoryUp(index int) tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil || index <= 0 || index >= len(m.categories) {
			return categoryOperationMsg{message: "Cannot move category up", messageType: "error"}
		}

		// Swap with previous category
		categoryIDs := make([]int, len(m.categories))
		for i, cat := range m.categories {
			categoryIDs[i] = cat.ID
		}

		// Swap
		categoryIDs[index], categoryIDs[index-1] = categoryIDs[index-1], categoryIDs[index]

		err := m.storage.Categories.Reorder(categoryIDs)
		if err != nil {
			return categoryOperationMsg{message: fmt.Sprintf("Failed to reorder: %v", err), messageType: "error"}
		}

		m.cursor = index - 1
		return categoryOperationMsg{message: "", messageType: ""}
	}
}

func (m Model) moveCategoryDown(index int) tea.Cmd {
	return func() tea.Msg {
		if m.storage == nil || index < 0 || index >= len(m.categories)-1 {
			return categoryOperationMsg{message: "Cannot move category down", messageType: "error"}
		}

		// Swap with next category
		categoryIDs := make([]int, len(m.categories))
		for i, cat := range m.categories {
			categoryIDs[i] = cat.ID
		}

		// Swap
		categoryIDs[index], categoryIDs[index+1] = categoryIDs[index+1], categoryIDs[index]

		err := m.storage.Categories.Reorder(categoryIDs)
		if err != nil {
			return categoryOperationMsg{message: fmt.Sprintf("Failed to reorder: %v", err), messageType: "error"}
		}

		m.cursor = index + 1
		return categoryOperationMsg{message: "", messageType: ""}
	}
}
