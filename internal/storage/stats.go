package storage

import (
	"fmt"
	"time"
)

// ReadingSession represents a reading session
type ReadingSession struct {
	ID              int64
	MangaID         string
	ChapterID       string
	DurationSeconds int
	PagesRead       int
	SessionStart    time.Time
	SessionEnd      time.Time
}

// ReadingStats represents reading statistics
type ReadingStats struct {
	TotalReadingTime    int // in seconds
	TotalPagesRead      int
	TotalChaptersRead   int
	TotalMangaRead      int
	AverageSessionTime  int // in seconds
	MostReadManga       string
	LongestReadingStreak int // in days
	CurrentStreak       int // in days
}

// MangaStats represents statistics for a specific manga
type MangaStats struct {
	MangaID          string
	TotalReadingTime int
	TotalPagesRead   int
	ChaptersRead     int
	LastReadAt       time.Time
	FirstReadAt      time.Time
}

// StatsManager handles reading statistics operations
type StatsManager struct {
	db *DB
}

// NewStatsManager creates a new stats manager
func NewStatsManager(db *DB) *StatsManager {
	return &StatsManager{db: db}
}

// RecordSession records a reading session
func (sm *StatsManager) RecordSession(session *ReadingSession) error {
	query := `
		INSERT INTO reading_sessions (manga_id, chapter_id, duration_seconds, pages_read, session_start, session_end)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := sm.db.conn.Exec(query,
		session.MangaID,
		session.ChapterID,
		session.DurationSeconds,
		session.PagesRead,
		session.SessionStart,
		session.SessionEnd,
	)

	if err != nil {
		return fmt.Errorf("failed to record session: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		session.ID = id
	}

	return nil
}

// GetGlobalStats retrieves overall reading statistics
func (sm *StatsManager) GetGlobalStats() (*ReadingStats, error) {
	stats := &ReadingStats{}

	// Get total reading time and pages read
	query := `
		SELECT
			COALESCE(SUM(duration_seconds), 0) as total_time,
			COALESCE(SUM(pages_read), 0) as total_pages,
			COALESCE(AVG(duration_seconds), 0) as avg_session
		FROM reading_sessions
	`

	err := sm.db.conn.QueryRow(query).Scan(&stats.TotalReadingTime, &stats.TotalPagesRead, &stats.AverageSessionTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get session stats: %w", err)
	}

	// Get total chapters read
	countQuery := `SELECT COUNT(*) FROM reading_progress WHERE is_completed = TRUE`
	err = sm.db.conn.QueryRow(countQuery).Scan(&stats.TotalChaptersRead)
	if err != nil {
		return nil, fmt.Errorf("failed to get chapters read: %w", err)
	}

	// Get total unique manga read
	mangaQuery := `SELECT COUNT(DISTINCT manga_id) FROM reading_history`
	err = sm.db.conn.QueryRow(mangaQuery).Scan(&stats.TotalMangaRead)
	if err != nil {
		return nil, fmt.Errorf("failed to get manga count: %w", err)
	}

	// Get most read manga
	mostReadQuery := `
		SELECT manga_id
		FROM reading_sessions
		GROUP BY manga_id
		ORDER BY SUM(duration_seconds) DESC
		LIMIT 1
	`
	err = sm.db.conn.QueryRow(mostReadQuery).Scan(&stats.MostReadManga)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fmt.Errorf("failed to get most read manga: %w", err)
	}

	// Calculate reading streak
	currentStreak, longestStreak, err := sm.calculateReadingStreaks()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate streaks: %w", err)
	}
	stats.CurrentStreak = currentStreak
	stats.LongestReadingStreak = longestStreak

	return stats, nil
}

// GetMangaStats retrieves statistics for a specific manga
func (sm *StatsManager) GetMangaStats(mangaID string) (*MangaStats, error) {
	stats := &MangaStats{MangaID: mangaID}

	// Get reading time and pages
	query := `
		SELECT
			COALESCE(SUM(duration_seconds), 0) as total_time,
			COALESCE(SUM(pages_read), 0) as total_pages,
			MIN(session_start) as first_read,
			MAX(session_end) as last_read
		FROM reading_sessions
		WHERE manga_id = ?
	`

	var firstRead, lastRead *time.Time
	err := sm.db.conn.QueryRow(query, mangaID).Scan(&stats.TotalReadingTime, &stats.TotalPagesRead, &firstRead, &lastRead)
	if err != nil {
		return nil, fmt.Errorf("failed to get manga session stats: %w", err)
	}

	if firstRead != nil {
		stats.FirstReadAt = *firstRead
	}
	if lastRead != nil {
		stats.LastReadAt = *lastRead
	}

	// Get chapters read count
	countQuery := `SELECT COUNT(*) FROM reading_progress WHERE manga_id = ? AND is_completed = TRUE`
	err = sm.db.conn.QueryRow(countQuery, mangaID).Scan(&stats.ChaptersRead)
	if err != nil {
		return nil, fmt.Errorf("failed to get chapters read: %w", err)
	}

	return stats, nil
}

// GetRecentSessions retrieves recent reading sessions
func (sm *StatsManager) GetRecentSessions(limit int) ([]*ReadingSession, error) {
	query := `
		SELECT id, manga_id, chapter_id, duration_seconds, pages_read, session_start, session_end
		FROM reading_sessions
		ORDER BY session_end DESC
		LIMIT ?
	`

	rows, err := sm.db.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*ReadingSession
	for rows.Next() {
		session := &ReadingSession{}
		err := rows.Scan(
			&session.ID,
			&session.MangaID,
			&session.ChapterID,
			&session.DurationSeconds,
			&session.PagesRead,
			&session.SessionStart,
			&session.SessionEnd,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}

	return sessions, nil
}

// GetDailyReadingTime returns total reading time for each day in the given period
func (sm *StatsManager) GetDailyReadingTime(since time.Time) (map[string]int, error) {
	query := `
		SELECT DATE(session_start) as day, SUM(duration_seconds) as total_time
		FROM reading_sessions
		WHERE session_start >= ?
		GROUP BY DATE(session_start)
		ORDER BY day ASC
	`

	rows, err := sm.db.conn.Query(query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily reading time: %w", err)
	}
	defer rows.Close()

	dailyTime := make(map[string]int)
	for rows.Next() {
		var day string
		var totalTime int
		err := rows.Scan(&day, &totalTime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily reading time: %w", err)
		}
		dailyTime[day] = totalTime
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating daily reading time rows: %w", err)
	}

	return dailyTime, nil
}

// calculateReadingStreaks calculates current and longest reading streaks
func (sm *StatsManager) calculateReadingStreaks() (current, longest int, err error) {
	// Get all unique reading dates, ordered
	query := `
		SELECT DISTINCT DATE(read_at) as read_date
		FROM reading_history
		ORDER BY read_date DESC
	`

	rows, err := sm.db.conn.Query(query)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query reading dates: %w", err)
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var dateStr string
		err := rows.Scan(&dateStr)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to scan date: %w", err)
		}
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse date: %w", err)
		}
		dates = append(dates, date)
	}

	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("error iterating date rows: %w", err)
	}

	if len(dates) == 0 {
		return 0, 0, nil
	}

	// Calculate streaks
	current = 1
	longest = 1
	tempStreak := 1

	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.AddDate(0, 0, -1)

	// Check if current streak is active
	if !dates[0].Equal(today) && !dates[0].Equal(yesterday) {
		current = 0
	}

	// Calculate longest streak
	for i := 1; i < len(dates); i++ {
		diff := dates[i-1].Sub(dates[i]).Hours() / 24

		if diff == 1 {
			tempStreak++
			if i < len(dates) && (dates[0].Equal(today) || dates[0].Equal(yesterday)) {
				current = tempStreak
			}
		} else {
			if tempStreak > longest {
				longest = tempStreak
			}
			tempStreak = 1
		}
	}

	if tempStreak > longest {
		longest = tempStreak
	}

	return current, longest, nil
}

// DeleteMangaSessions deletes all sessions for a specific manga
func (sm *StatsManager) DeleteMangaSessions(mangaID string) error {
	query := `DELETE FROM reading_sessions WHERE manga_id = ?`
	_, err := sm.db.conn.Exec(query, mangaID)
	if err != nil {
		return fmt.Errorf("failed to delete manga sessions: %w", err)
	}
	return nil
}

// ClearAllSessions deletes all reading sessions
func (sm *StatsManager) ClearAllSessions() error {
	query := `DELETE FROM reading_sessions`
	_, err := sm.db.conn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to clear sessions: %w", err)
	}
	return nil
}
