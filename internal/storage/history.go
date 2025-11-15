package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// HistoryEntry represents a reading history entry
type HistoryEntry struct {
	ID            int64
	MangaID       string
	MangaTitle    string
	ChapterID     string
	ChapterNumber float64
	ChapterTitle  string
	SourceType    string
	SourceID      string
	ReadAt        time.Time
}

// HistoryManager handles reading history operations
type HistoryManager struct {
	db *DB
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(db *DB) *HistoryManager {
	return &HistoryManager{db: db}
}

// AddHistoryEntry adds or updates a history entry
func (hm *HistoryManager) AddHistoryEntry(entry *HistoryEntry) error {
	query := `
		INSERT INTO reading_history (manga_id, manga_title, chapter_id, chapter_number, chapter_title, source_type, source_id, read_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(manga_id, chapter_id)
		DO UPDATE SET
			manga_title = excluded.manga_title,
			chapter_number = excluded.chapter_number,
			chapter_title = excluded.chapter_title,
			read_at = excluded.read_at
	`

	readAt := entry.ReadAt
	if readAt.IsZero() {
		readAt = time.Now()
	}

	_, err := hm.db.conn.Exec(query,
		entry.MangaID,
		entry.MangaTitle,
		entry.ChapterID,
		entry.ChapterNumber,
		entry.ChapterTitle,
		entry.SourceType,
		entry.SourceID,
		readAt,
	)

	if err != nil {
		return fmt.Errorf("failed to add history entry: %w", err)
	}

	return nil
}

// GetRecentHistory retrieves recent reading history
func (hm *HistoryManager) GetRecentHistory(limit int) ([]*HistoryEntry, error) {
	query := `
		SELECT id, manga_id, manga_title, chapter_id, chapter_number, chapter_title, source_type, source_id, read_at
		FROM reading_history
		ORDER BY read_at DESC
		LIMIT ?
	`

	rows, err := hm.db.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	return hm.scanHistoryEntries(rows)
}

// GetMangaHistory retrieves reading history for a specific manga
func (hm *HistoryManager) GetMangaHistory(mangaID string) ([]*HistoryEntry, error) {
	query := `
		SELECT id, manga_id, manga_title, chapter_id, chapter_number, chapter_title, source_type, source_id, read_at
		FROM reading_history
		WHERE manga_id = ?
		ORDER BY chapter_number DESC
	`

	rows, err := hm.db.conn.Query(query, mangaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query manga history: %w", err)
	}
	defer rows.Close()

	return hm.scanHistoryEntries(rows)
}

// GetHistorySince retrieves history entries since a given time
func (hm *HistoryManager) GetHistorySince(since time.Time) ([]*HistoryEntry, error) {
	query := `
		SELECT id, manga_id, manga_title, chapter_id, chapter_number, chapter_title, source_type, source_id, read_at
		FROM reading_history
		WHERE read_at >= ?
		ORDER BY read_at DESC
	`

	rows, err := hm.db.conn.Query(query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query history since: %w", err)
	}
	defer rows.Close()

	return hm.scanHistoryEntries(rows)
}

// DeleteHistoryEntry deletes a specific history entry
func (hm *HistoryManager) DeleteHistoryEntry(id int64) error {
	query := `DELETE FROM reading_history WHERE id = ?`
	_, err := hm.db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete history entry: %w", err)
	}
	return nil
}

// DeleteMangaHistory deletes all history for a specific manga
func (hm *HistoryManager) DeleteMangaHistory(mangaID string) error {
	query := `DELETE FROM reading_history WHERE manga_id = ?`
	_, err := hm.db.conn.Exec(query, mangaID)
	if err != nil {
		return fmt.Errorf("failed to delete manga history: %w", err)
	}
	return nil
}

// ClearAllHistory deletes all history entries
func (hm *HistoryManager) ClearAllHistory() error {
	query := `DELETE FROM reading_history`
	_, err := hm.db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}
	return nil
}

// GetHistoryCount returns the total number of history entries
func (hm *HistoryManager) GetHistoryCount() (int, error) {
	query := `SELECT COUNT(*) FROM reading_history`
	var count int
	err := hm.db.conn.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get history count: %w", err)
	}
	return count, nil
}

// scanHistoryEntries scans rows into history entries
func (hm *HistoryManager) scanHistoryEntries(rows *sql.Rows) ([]*HistoryEntry, error) {
	var entries []*HistoryEntry

	for rows.Next() {
		entry := &HistoryEntry{}
		var sourceID sql.NullString
		var chapterTitle sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.MangaID,
			&entry.MangaTitle,
			&entry.ChapterID,
			&entry.ChapterNumber,
			&chapterTitle,
			&entry.SourceType,
			&sourceID,
			&entry.ReadAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan history entry: %w", err)
		}

		if sourceID.Valid {
			entry.SourceID = sourceID.String
		}
		if chapterTitle.Valid {
			entry.ChapterTitle = chapterTitle.String
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating history rows: %w", err)
	}

	return entries, nil
}
