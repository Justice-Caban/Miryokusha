package storage

import (
	"database/sql"
	"fmt"
	"time"
)

// Bookmark represents a bookmarked page
type Bookmark struct {
	ID            int64
	MangaID       string
	MangaTitle    string
	ChapterID     string
	ChapterNumber float64
	ChapterTitle  string
	PageNumber    int
	Note          string
	CreatedAt     time.Time
}

// BookmarkManager handles bookmark operations
type BookmarkManager struct {
	db *DB
}

// NewBookmarkManager creates a new bookmark manager
func NewBookmarkManager(db *DB) *BookmarkManager {
	return &BookmarkManager{db: db}
}

// AddBookmark creates a new bookmark
func (bm *BookmarkManager) AddBookmark(bookmark *Bookmark) error {
	query := `
		INSERT INTO bookmarks (manga_id, manga_title, chapter_id, chapter_number, chapter_title, page_number, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	createdAt := bookmark.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	result, err := bm.db.conn.Exec(query,
		bookmark.MangaID,
		bookmark.MangaTitle,
		bookmark.ChapterID,
		bookmark.ChapterNumber,
		bookmark.ChapterTitle,
		bookmark.PageNumber,
		bookmark.Note,
		createdAt,
	)

	if err != nil {
		return fmt.Errorf("failed to add bookmark: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		bookmark.ID = id
	}

	return nil
}

// GetBookmark retrieves a specific bookmark by ID
func (bm *BookmarkManager) GetBookmark(id int64) (*Bookmark, error) {
	query := `
		SELECT id, manga_id, manga_title, chapter_id, chapter_number, chapter_title, page_number, note, created_at
		FROM bookmarks
		WHERE id = ?
	`

	bookmark := &Bookmark{}
	var chapterTitle, note sql.NullString

	err := bm.db.conn.QueryRow(query, id).Scan(
		&bookmark.ID,
		&bookmark.MangaID,
		&bookmark.MangaTitle,
		&bookmark.ChapterID,
		&bookmark.ChapterNumber,
		&chapterTitle,
		&bookmark.PageNumber,
		&note,
		&bookmark.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get bookmark: %w", err)
	}

	if chapterTitle.Valid {
		bookmark.ChapterTitle = chapterTitle.String
	}
	if note.Valid {
		bookmark.Note = note.String
	}

	return bookmark, nil
}

// GetAllBookmarks retrieves all bookmarks
func (bm *BookmarkManager) GetAllBookmarks() ([]*Bookmark, error) {
	query := `
		SELECT id, manga_id, manga_title, chapter_id, chapter_number, chapter_title, page_number, note, created_at
		FROM bookmarks
		ORDER BY created_at DESC
	`

	rows, err := bm.db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query bookmarks: %w", err)
	}
	defer rows.Close()

	return bm.scanBookmarks(rows)
}

// GetMangaBookmarks retrieves all bookmarks for a specific manga
func (bm *BookmarkManager) GetMangaBookmarks(mangaID string) ([]*Bookmark, error) {
	query := `
		SELECT id, manga_id, manga_title, chapter_id, chapter_number, chapter_title, page_number, note, created_at
		FROM bookmarks
		WHERE manga_id = ?
		ORDER BY chapter_number ASC, page_number ASC
	`

	rows, err := bm.db.conn.Query(query, mangaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query manga bookmarks: %w", err)
	}
	defer rows.Close()

	return bm.scanBookmarks(rows)
}

// GetChapterBookmarks retrieves all bookmarks for a specific chapter
func (bm *BookmarkManager) GetChapterBookmarks(chapterID string) ([]*Bookmark, error) {
	query := `
		SELECT id, manga_id, manga_title, chapter_id, chapter_number, chapter_title, page_number, note, created_at
		FROM bookmarks
		WHERE chapter_id = ?
		ORDER BY page_number ASC
	`

	rows, err := bm.db.conn.Query(query, chapterID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chapter bookmarks: %w", err)
	}
	defer rows.Close()

	return bm.scanBookmarks(rows)
}

// UpdateBookmarkNote updates the note for a bookmark
func (bm *BookmarkManager) UpdateBookmarkNote(id int64, note string) error {
	query := `UPDATE bookmarks SET note = ? WHERE id = ?`
	_, err := bm.db.conn.Exec(query, note, id)
	if err != nil {
		return fmt.Errorf("failed to update bookmark note: %w", err)
	}
	return nil
}

// DeleteBookmark deletes a specific bookmark
func (bm *BookmarkManager) DeleteBookmark(id int64) error {
	query := `DELETE FROM bookmarks WHERE id = ?`
	_, err := bm.db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete bookmark: %w", err)
	}
	return nil
}

// DeleteMangaBookmarks deletes all bookmarks for a specific manga
func (bm *BookmarkManager) DeleteMangaBookmarks(mangaID string) error {
	query := `DELETE FROM bookmarks WHERE manga_id = ?`
	_, err := bm.db.conn.Exec(query, mangaID)
	if err != nil {
		return fmt.Errorf("failed to delete manga bookmarks: %w", err)
	}
	return nil
}

// DeleteChapterBookmarks deletes all bookmarks for a specific chapter
func (bm *BookmarkManager) DeleteChapterBookmarks(chapterID string) error {
	query := `DELETE FROM bookmarks WHERE chapter_id = ?`
	_, err := bm.db.conn.Exec(query, chapterID)
	if err != nil {
		return fmt.Errorf("failed to delete chapter bookmarks: %w", err)
	}
	return nil
}

// GetBookmarkCount returns the total number of bookmarks
func (bm *BookmarkManager) GetBookmarkCount() (int, error) {
	query := `SELECT COUNT(*) FROM bookmarks`
	var count int
	err := bm.db.conn.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get bookmark count: %w", err)
	}
	return count, nil
}

// GetMangaBookmarkCount returns the number of bookmarks for a specific manga
func (bm *BookmarkManager) GetMangaBookmarkCount(mangaID string) (int, error) {
	query := `SELECT COUNT(*) FROM bookmarks WHERE manga_id = ?`
	var count int
	err := bm.db.conn.QueryRow(query, mangaID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get manga bookmark count: %w", err)
	}
	return count, nil
}

// BookmarkExists checks if a bookmark exists for a specific page
func (bm *BookmarkManager) BookmarkExists(mangaID, chapterID string, pageNumber int) (bool, error) {
	query := `SELECT COUNT(*) FROM bookmarks WHERE manga_id = ? AND chapter_id = ? AND page_number = ?`
	var count int
	err := bm.db.conn.QueryRow(query, mangaID, chapterID, pageNumber).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check bookmark existence: %w", err)
	}
	return count > 0, nil
}

// scanBookmarks scans rows into bookmarks
func (bm *BookmarkManager) scanBookmarks(rows *sql.Rows) ([]*Bookmark, error) {
	var bookmarks []*Bookmark

	for rows.Next() {
		bookmark := &Bookmark{}
		var chapterTitle, note sql.NullString

		err := rows.Scan(
			&bookmark.ID,
			&bookmark.MangaID,
			&bookmark.MangaTitle,
			&bookmark.ChapterID,
			&bookmark.ChapterNumber,
			&chapterTitle,
			&bookmark.PageNumber,
			&note,
			&bookmark.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan bookmark: %w", err)
		}

		if chapterTitle.Valid {
			bookmark.ChapterTitle = chapterTitle.String
		}
		if note.Valid {
			bookmark.Note = note.String
		}

		bookmarks = append(bookmarks, bookmark)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bookmark rows: %w", err)
	}

	return bookmarks, nil
}
