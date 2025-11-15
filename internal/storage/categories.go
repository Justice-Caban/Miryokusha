package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// Category represents a manga category
type Category struct {
	ID        int
	Name      string
	SortOrder int
	IsDefault bool
	CreatedAt time.Time
	UpdatedAt time.Time
	MangaCount int // Number of manga in this category
}

// CategoryManager manages manga categories
type CategoryManager struct {
	db *DB
}

// NewCategoryManager creates a new category manager
func NewCategoryManager(db *DB) *CategoryManager {
	return &CategoryManager{db: db}
}

// Create creates a new category
func (cm *CategoryManager) Create(name string, isDefault bool) (*Category, error) {
	// Get the next sort order
	var maxSort int
	err := cm.db.conn.QueryRow("SELECT COALESCE(MAX(sort_order), -1) + 1 FROM categories").Scan(&maxSort)
	if err != nil {
		return nil, fmt.Errorf("failed to get max sort order: %w", err)
	}

	// If this is the first category, make it default
	var count int
	err = cm.db.conn.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		isDefault = true
	}

	// If setting as default, unset other defaults
	if isDefault {
		_, err = cm.db.conn.Exec("UPDATE categories SET is_default = FALSE")
		if err != nil {
			return nil, fmt.Errorf("failed to unset other defaults: %w", err)
		}
	}

	result, err := cm.db.conn.Exec(`
		INSERT INTO categories (name, sort_order, is_default, created_at, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, name, maxSort, isDefault)

	if err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return cm.GetByID(int(id))
}

// GetByID retrieves a category by ID
func (cm *CategoryManager) GetByID(id int) (*Category, error) {
	var cat Category
	err := cm.db.conn.QueryRow(`
		SELECT id, name, sort_order, is_default, created_at, updated_at
		FROM categories
		WHERE id = ?
	`, id).Scan(&cat.ID, &cat.Name, &cat.SortOrder, &cat.IsDefault, &cat.CreatedAt, &cat.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("category not found")
		}
		return nil, err
	}

	// Get manga count
	err = cm.db.conn.QueryRow(`
		SELECT COUNT(*) FROM manga_categories WHERE category_id = ?
	`, id).Scan(&cat.MangaCount)
	if err != nil {
		cat.MangaCount = 0
	}

	return &cat, nil
}

// GetAll retrieves all categories ordered by sort_order
func (cm *CategoryManager) GetAll() ([]*Category, error) {
	rows, err := cm.db.conn.Query(`
		SELECT c.id, c.name, c.sort_order, c.is_default, c.created_at, c.updated_at,
		       COUNT(mc.manga_id) as manga_count
		FROM categories c
		LEFT JOIN manga_categories mc ON c.id = mc.category_id
		GROUP BY c.id, c.name, c.sort_order, c.is_default, c.created_at, c.updated_at
		ORDER BY c.sort_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*Category
	for rows.Next() {
		var cat Category
		err := rows.Scan(&cat.ID, &cat.Name, &cat.SortOrder, &cat.IsDefault,
			&cat.CreatedAt, &cat.UpdatedAt, &cat.MangaCount)
		if err != nil {
			return nil, err
		}
		categories = append(categories, &cat)
	}

	return categories, rows.Err()
}

// Update updates a category
func (cm *CategoryManager) Update(id int, name string, isDefault bool) error {
	// If setting as default, unset other defaults
	if isDefault {
		_, err := cm.db.conn.Exec("UPDATE categories SET is_default = FALSE WHERE id != ?", id)
		if err != nil {
			return fmt.Errorf("failed to unset other defaults: %w", err)
		}
	}

	result, err := cm.db.conn.Exec(`
		UPDATE categories
		SET name = ?, is_default = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, isDefault, id)

	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("category not found")
	}

	return nil
}

// Delete deletes a category
func (cm *CategoryManager) Delete(id int) error {
	// Don't allow deleting if it's the only category
	var count int
	err := cm.db.conn.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		return err
	}
	if count <= 1 {
		return fmt.Errorf("cannot delete the last category")
	}

	// Check if it's the default category
	var isDefault bool
	err = cm.db.conn.QueryRow("SELECT is_default FROM categories WHERE id = ?", id).Scan(&isDefault)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("category not found")
		}
		return err
	}

	// Move manga to the first remaining category
	var firstCategoryID int
	err = cm.db.conn.QueryRow("SELECT id FROM categories WHERE id != ? ORDER BY sort_order LIMIT 1", id).Scan(&firstCategoryID)
	if err != nil {
		return fmt.Errorf("failed to find alternative category: %w", err)
	}

	// Update manga_categories to point to the first category
	_, err = cm.db.conn.Exec(`
		UPDATE manga_categories
		SET category_id = ?
		WHERE category_id = ?
	`, firstCategoryID, id)
	if err != nil {
		return fmt.Errorf("failed to reassign manga: %w", err)
	}

	// Delete the category
	result, err := cm.db.conn.Exec("DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("category not found")
	}

	// If we deleted the default category, make the first one default
	if isDefault {
		_, err = cm.db.conn.Exec("UPDATE categories SET is_default = TRUE WHERE id = ?", firstCategoryID)
		if err != nil {
			return fmt.Errorf("failed to set new default: %w", err)
		}
	}

	return nil
}

// Reorder changes the sort order of categories
func (cm *CategoryManager) Reorder(categoryIDs []int) error {
	tx, err := cm.db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("UPDATE categories SET sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i, id := range categoryIDs {
		_, err = stmt.Exec(i, id)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetDefault returns the default category
func (cm *CategoryManager) GetDefault() (*Category, error) {
	var cat Category
	err := cm.db.conn.QueryRow(`
		SELECT c.id, c.name, c.sort_order, c.is_default, c.created_at, c.updated_at,
		       COUNT(mc.manga_id) as manga_count
		FROM categories c
		LEFT JOIN manga_categories mc ON c.id = mc.category_id
		WHERE c.is_default = TRUE
		GROUP BY c.id, c.name, c.sort_order, c.is_default, c.created_at, c.updated_at
	`).Scan(&cat.ID, &cat.Name, &cat.SortOrder, &cat.IsDefault,
		&cat.CreatedAt, &cat.UpdatedAt, &cat.MangaCount)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no default category found")
		}
		return nil, err
	}

	return &cat, nil
}

// AssignManga assigns a manga to a category
func (cm *CategoryManager) AssignManga(mangaID string, categoryID int) error {
	_, err := cm.db.conn.Exec(`
		INSERT OR IGNORE INTO manga_categories (manga_id, category_id, added_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, mangaID, categoryID)

	return err
}

// UnassignManga removes a manga from a category
func (cm *CategoryManager) UnassignManga(mangaID string, categoryID int) error {
	result, err := cm.db.conn.Exec(`
		DELETE FROM manga_categories
		WHERE manga_id = ? AND category_id = ?
	`, mangaID, categoryID)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("manga not in category")
	}

	return nil
}

// GetMangaCategories returns all categories for a manga
func (cm *CategoryManager) GetMangaCategories(mangaID string) ([]*Category, error) {
	rows, err := cm.db.conn.Query(`
		SELECT c.id, c.name, c.sort_order, c.is_default, c.created_at, c.updated_at,
		       0 as manga_count
		FROM categories c
		INNER JOIN manga_categories mc ON c.id = mc.category_id
		WHERE mc.manga_id = ?
		ORDER BY c.sort_order
	`, mangaID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []*Category
	for rows.Next() {
		var cat Category
		err := rows.Scan(&cat.ID, &cat.Name, &cat.SortOrder, &cat.IsDefault,
			&cat.CreatedAt, &cat.UpdatedAt, &cat.MangaCount)
		if err != nil {
			return nil, err
		}
		categories = append(categories, &cat)
	}

	return categories, rows.Err()
}

// GetMangaInCategory returns all manga IDs in a category
func (cm *CategoryManager) GetMangaInCategory(categoryID int) ([]string, error) {
	rows, err := cm.db.conn.Query(`
		SELECT manga_id FROM manga_categories WHERE category_id = ?
	`, categoryID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mangaIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		mangaIDs = append(mangaIDs, id)
	}

	return mangaIDs, rows.Err()
}

// SetMangaCategories replaces all categories for a manga
func (cm *CategoryManager) SetMangaCategories(mangaID string, categoryIDs []int) error {
	tx, err := cm.db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove all existing categories
	_, err = tx.Exec("DELETE FROM manga_categories WHERE manga_id = ?", mangaID)
	if err != nil {
		return err
	}

	// Add new categories
	if len(categoryIDs) > 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO manga_categories (manga_id, category_id, added_at)
			VALUES (?, ?, CURRENT_TIMESTAMP)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, catID := range categoryIDs {
			_, err = stmt.Exec(mangaID, catID)
			if err != nil {
				return err
			}
		}
	} else {
		// If no categories specified, assign to default
		defaultCat, err := cm.GetDefault()
		if err == nil {
			_, err = tx.Exec(`
				INSERT INTO manga_categories (manga_id, category_id, added_at)
				VALUES (?, ?, CURRENT_TIMESTAMP)
			`, mangaID, defaultCat.ID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// InitializeDefaultCategories creates default categories if none exist
func (cm *CategoryManager) InitializeDefaultCategories() error {
	var count int
	err := cm.db.conn.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Categories already exist
	}

	// Create default categories
	defaultCategories := []string{
		"Reading",
		"Completed",
		"Plan to Read",
		"On Hold",
		"Dropped",
	}

	for i, name := range defaultCategories {
		_, err := cm.Create(name, i == 0) // First one is default
		if err != nil {
			return fmt.Errorf("failed to create default category %s: %w", name, err)
		}
	}

	return nil
}
