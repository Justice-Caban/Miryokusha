package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// ProgressEntry represents reading progress for a chapter
type ProgressEntry struct {
	ID          int64
	MangaID     string
	MangaTitle string
	ChapterID   string
	CurrentPage int
	TotalPages  int
	IsCompleted bool
	LastReadAt  time.Time
}

// ProgressManager handles reading progress operations
type ProgressManager struct {
	db *DB
}

// NewProgressManager creates a new progress manager
func NewProgressManager(db *DB) *ProgressManager {
	return &ProgressManager{db: db}
}

// UpdateProgress updates or creates reading progress for a chapter
func (pm *ProgressManager) UpdateProgress(mangaID, mangaTitle, chapterID string, currentPage, totalPages int) error {
	isCompleted := currentPage >= totalPages-1

	query := `
		INSERT INTO reading_progress (manga_id, manga_title, chapter_id, current_page, total_pages, is_completed, last_read_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(manga_id, chapter_id)
		DO UPDATE SET
			manga_title = excluded.manga_title,
			current_page = excluded.current_page,
			total_pages = excluded.total_pages,
			is_completed = excluded.is_completed,
			last_read_at = excluded.last_read_at
	`

	_, err := pm.db.conn.Exec(query, mangaID, mangaTitle, chapterID, currentPage, totalPages, isCompleted, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update progress: %w", err)
	}

	return nil
}

// GetProgress retrieves reading progress for a specific chapter
func (pm *ProgressManager) GetProgress(mangaID, chapterID string) (*ProgressEntry, error) {
	query := `
		SELECT id, manga_id, manga_title, chapter_id, current_page, total_pages, is_completed, last_read_at
		FROM reading_progress
		WHERE manga_id = ? AND chapter_id = ?
	`

	entry := &ProgressEntry{}
	err := pm.db.conn.QueryRow(query, mangaID, chapterID).Scan(
		&entry.ID,
		&entry.MangaID,
		&entry.MangaTitle,
		&entry.ChapterID,
		&entry.CurrentPage,
		&entry.TotalPages,
		&entry.IsCompleted,
		&entry.LastReadAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}

	return entry, nil
}

// GetMangaProgress retrieves all progress entries for a manga
func (pm *ProgressManager) GetMangaProgress(mangaID string) ([]*ProgressEntry, error) {
	query := `
		SELECT id, manga_id, manga_title, chapter_id, current_page, total_pages, is_completed, last_read_at
		FROM reading_progress
		WHERE manga_id = ?
		ORDER BY last_read_at DESC
	`

	rows, err := pm.db.conn.Query(query, mangaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query manga progress: %w", err)
	}
	defer rows.Close()

	return pm.scanProgressEntries(rows)
}

// GetRecentlyRead retrieves recently read chapters across all manga
func (pm *ProgressManager) GetRecentlyRead(limit int) ([]*ProgressEntry, error) {
	query := `
		SELECT id, manga_id, manga_title, chapter_id, current_page, total_pages, is_completed, last_read_at
		FROM reading_progress
		ORDER BY last_read_at DESC
		LIMIT ?
	`

	rows, err := pm.db.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent progress: %w", err)
	}
	defer rows.Close()

	return pm.scanProgressEntries(rows)
}

// GetInProgressChapters retrieves all chapters that are not yet completed
func (pm *ProgressManager) GetInProgressChapters() ([]*ProgressEntry, error) {
	query := `
		SELECT id, manga_id, manga_title, chapter_id, current_page, total_pages, is_completed, last_read_at
		FROM reading_progress
		WHERE is_completed = FALSE
		ORDER BY last_read_at DESC
	`

	rows, err := pm.db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query in-progress chapters: %w", err)
	}
	defer rows.Close()

	return pm.scanProgressEntries(rows)
}

// MarkAsCompleted marks a chapter as completed
func (pm *ProgressManager) MarkAsCompleted(mangaID, chapterID string, totalPages int) error {
	query := `
		INSERT INTO reading_progress (manga_id, manga_title, chapter_id, current_page, total_pages, is_completed, last_read_at)
		VALUES (?, '', ?, ?, ?, TRUE, ?)
		ON CONFLICT(manga_id, chapter_id)
		DO UPDATE SET
			current_page = excluded.total_pages,
			is_completed = TRUE,
			last_read_at = excluded.last_read_at
	`

	_, err := pm.db.conn.Exec(query, mangaID, chapterID, totalPages, totalPages, time.Now())
	if err != nil {
		return fmt.Errorf("failed to mark as completed: %w", err)
	}

	return nil
}

// DeleteProgress deletes progress for a specific chapter
func (pm *ProgressManager) DeleteProgress(mangaID, chapterID string) error {
	query := `DELETE FROM reading_progress WHERE manga_id = ? AND chapter_id = ?`
	_, err := pm.db.conn.Exec(query, mangaID, chapterID)
	if err != nil {
		return fmt.Errorf("failed to delete progress: %w", err)
	}
	return nil
}

// DeleteMangaProgress deletes all progress for a specific manga
func (pm *ProgressManager) DeleteMangaProgress(mangaID string) error {
	query := `DELETE FROM reading_progress WHERE manga_id = ?`
	_, err := pm.db.conn.Exec(query, mangaID)
	if err != nil {
		return fmt.Errorf("failed to delete manga progress: %w", err)
	}
	return nil
}

// GetProgressStats returns statistics about reading progress
func (pm *ProgressManager) GetProgressStats(mangaID string) (total, completed, inProgress int, err error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN is_completed = TRUE THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN is_completed = FALSE THEN 1 ELSE 0 END) as in_progress
		FROM reading_progress
		WHERE manga_id = ?
	`

	err = pm.db.conn.QueryRow(query, mangaID).Scan(&total, &completed, &inProgress)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get progress stats: %w", err)
	}

	return total, completed, inProgress, nil
}

// scanProgressEntries scans rows into progress entries
func (pm *ProgressManager) scanProgressEntries(rows *sql.Rows) ([]*ProgressEntry, error) {
	var entries []*ProgressEntry

	for rows.Next() {
		entry := &ProgressEntry{}
		err := rows.Scan(
			&entry.ID,
			&entry.MangaID,
		&entry.MangaTitle,
			&entry.ChapterID,
			&entry.CurrentPage,
			&entry.TotalPages,
			&entry.IsCompleted,
			&entry.LastReadAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan progress entry: %w", err)
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating progress rows: %w", err)
	}

	return entries, nil
}
