package storage

import (
	"database/sql"
	"time"
)

// MangaUpdateTracking tracks update patterns for smart updates
type MangaUpdateTracking struct {
	MangaID                string
	LastCheck              time.Time
	LastChapterFound       *time.Time
	ChapterCount           int
	AvgUpdateIntervalDays  *float64
	FetchCount             int
	ConsecutiveFailures    int
	IsCompleted            bool
	IsOngoing              bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// UpdateTrackingManager manages manga update tracking for smart updates
type UpdateTrackingManager struct {
	db *DB
}

// NewUpdateTrackingManager creates a new update tracking manager
func NewUpdateTrackingManager(db *DB) *UpdateTrackingManager {
	return &UpdateTrackingManager{db: db}
}

// RecordUpdateCheck records an update check for a manga
func (utm *UpdateTrackingManager) RecordUpdateCheck(mangaID string, foundNewChapters bool, newChapterCount int) error {
	var existing MangaUpdateTracking
	err := utm.db.conn.QueryRow(`
		SELECT manga_id, last_check, last_chapter_found, chapter_count, avg_update_interval_days,
		       fetch_count, consecutive_failures, is_completed, is_ongoing, created_at, updated_at
		FROM manga_update_tracking
		WHERE manga_id = ?
	`, mangaID).Scan(
		&existing.MangaID,
		&existing.LastCheck,
		&existing.LastChapterFound,
		&existing.ChapterCount,
		&existing.AvgUpdateIntervalDays,
		&existing.FetchCount,
		&existing.ConsecutiveFailures,
		&existing.IsCompleted,
		&existing.IsOngoing,
		&existing.CreatedAt,
		&existing.UpdatedAt,
	)

	now := time.Now()

	if err == sql.ErrNoRows {
		// First time tracking this manga
		var lastChapterFound *time.Time
		if foundNewChapters {
			lastChapterFound = &now
		}

		_, err = utm.db.conn.Exec(`
			INSERT INTO manga_update_tracking
			(manga_id, last_check, last_chapter_found, chapter_count, fetch_count, consecutive_failures, created_at, updated_at)
			VALUES (?, ?, ?, ?, 1, 0, ?, ?)
		`, mangaID, now, lastChapterFound, newChapterCount, now, now)
		return err
	} else if err != nil {
		return err
	}

	// Update existing tracking
	consecutiveFailures := existing.ConsecutiveFailures
	if foundNewChapters {
		consecutiveFailures = 0
	} else {
		consecutiveFailures++
	}

	var avgInterval *float64
	var lastChapterFound *time.Time

	if foundNewChapters && newChapterCount > existing.ChapterCount {
		// New chapters found
		lastChapterFound = &now

		// Calculate average update interval
		if existing.LastChapterFound != nil {
			daysSinceLastChapter := now.Sub(*existing.LastChapterFound).Hours() / 24
			if existing.AvgUpdateIntervalDays != nil {
				// Running average
				newAvg := (*existing.AvgUpdateIntervalDays + daysSinceLastChapter) / 2.0
				avgInterval = &newAvg
			} else {
				avgInterval = &daysSinceLastChapter
			}
		}
	} else {
		lastChapterFound = existing.LastChapterFound
		avgInterval = existing.AvgUpdateIntervalDays
	}

	_, err = utm.db.conn.Exec(`
		UPDATE manga_update_tracking
		SET last_check = ?,
		    last_chapter_found = ?,
		    chapter_count = ?,
		    avg_update_interval_days = ?,
		    fetch_count = fetch_count + 1,
		    consecutive_failures = ?,
		    updated_at = ?
		WHERE manga_id = ?
	`, now, lastChapterFound, newChapterCount, avgInterval, consecutiveFailures, now, mangaID)

	return err
}

// MarkAsCompleted marks a manga as completed (no more updates expected)
func (utm *UpdateTrackingManager) MarkAsCompleted(mangaID string, isCompleted bool) error {
	_, err := utm.db.conn.Exec(`
		UPDATE manga_update_tracking
		SET is_completed = ?, is_ongoing = ?, updated_at = CURRENT_TIMESTAMP
		WHERE manga_id = ?
	`, isCompleted, !isCompleted, mangaID)
	return err
}

// GetTracking retrieves update tracking for a manga
func (utm *UpdateTrackingManager) GetTracking(mangaID string) (*MangaUpdateTracking, error) {
	var tracking MangaUpdateTracking
	err := utm.db.conn.QueryRow(`
		SELECT manga_id, last_check, last_chapter_found, chapter_count, avg_update_interval_days,
		       fetch_count, consecutive_failures, is_completed, is_ongoing, created_at, updated_at
		FROM manga_update_tracking
		WHERE manga_id = ?
	`, mangaID).Scan(
		&tracking.MangaID,
		&tracking.LastCheck,
		&tracking.LastChapterFound,
		&tracking.ChapterCount,
		&tracking.AvgUpdateIntervalDays,
		&tracking.FetchCount,
		&tracking.ConsecutiveFailures,
		&tracking.IsCompleted,
		&tracking.IsOngoing,
		&tracking.CreatedAt,
		&tracking.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &tracking, nil
}

// SmartUpdateConfig configures smart update behavior
type SmartUpdateConfig struct {
	MinIntervalHours       int     // Minimum hours between checks (default: 12)
	UpdateOnlyOngoing      bool    // Only update ongoing series (default: true)
	UpdateOnlyStarted      bool    // Only update series that have been read (default: false)
	MaxConsecutiveFailures int     // Skip after this many failures (default: 10)
	MultiplyIntervalBy     float64 // Multiply expected interval by this (default: 1.5 for safety margin)
}

// DefaultSmartUpdateConfig returns default smart update configuration
func DefaultSmartUpdateConfig() *SmartUpdateConfig {
	return &SmartUpdateConfig{
		MinIntervalHours:       12,
		UpdateOnlyOngoing:      true,
		UpdateOnlyStarted:      false,
		MaxConsecutiveFailures: 10,
		MultiplyIntervalBy:     1.5,
	}
}

// GetMangaForSmartUpdate returns manga IDs that should be checked for updates
// based on smart update logic (similar to Mihon/Tachiyomi)
func (utm *UpdateTrackingManager) GetMangaForSmartUpdate(config *SmartUpdateConfig, allMangaIDs []string) ([]string, error) {
	if config == nil {
		config = DefaultSmartUpdateConfig()
	}

	now := time.Now()
	minCheckTime := now.Add(-time.Duration(config.MinIntervalHours) * time.Hour)

	var smartUpdateIDs []string

	for _, mangaID := range allMangaIDs {
		// Check if manga has been started (if configured to only update started manga)
		if config.UpdateOnlyStarted {
			hasHistory, err := utm.hasMangaBeenRead(mangaID)
			if err != nil {
				return nil, err
			}
			if !hasHistory {
				continue // Skip manga that hasn't been read yet
			}
		}

		tracking, err := utm.GetTracking(mangaID)
		if err != nil {
			return nil, err
		}

		// If no tracking exists, include it (first time)
		if tracking == nil {
			smartUpdateIDs = append(smartUpdateIDs, mangaID)
			continue
		}

		// Skip if completed
		if config.UpdateOnlyOngoing && tracking.IsCompleted {
			continue
		}

		// Skip if too many consecutive failures
		if tracking.ConsecutiveFailures >= config.MaxConsecutiveFailures {
			continue
		}

		// Check if enough time has passed since last check
		if tracking.LastCheck.Before(minCheckTime) {
			// If we have average interval data, use it
			if tracking.AvgUpdateIntervalDays != nil && *tracking.AvgUpdateIntervalDays > 0 {
				// Calculate expected next update time
				expectedIntervalDays := *tracking.AvgUpdateIntervalDays * config.MultiplyIntervalBy
				expectedIntervalHours := expectedIntervalDays * 24

				if tracking.LastChapterFound != nil {
					hoursSinceLastChapter := now.Sub(*tracking.LastChapterFound).Hours()

					// Include if we're past the expected interval
					if hoursSinceLastChapter >= expectedIntervalHours {
						smartUpdateIDs = append(smartUpdateIDs, mangaID)
					}
				} else {
					// No chapters found yet, but respect minimum interval
					smartUpdateIDs = append(smartUpdateIDs, mangaID)
				}
			} else {
				// No interval data, just use minimum interval
				smartUpdateIDs = append(smartUpdateIDs, mangaID)
			}
		}
	}

	return smartUpdateIDs, nil
}

// hasMangaBeenRead checks if a manga has any reading history
func (utm *UpdateTrackingManager) hasMangaBeenRead(mangaID string) (bool, error) {
	var count int
	err := utm.db.conn.QueryRow(`
		SELECT COUNT(*) FROM reading_history WHERE manga_id = ? LIMIT 1
	`, mangaID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUpdateStats returns statistics about update tracking
func (utm *UpdateTrackingManager) GetUpdateStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total tracked manga
	var totalTracked int
	err := utm.db.conn.QueryRow("SELECT COUNT(*) FROM manga_update_tracking").Scan(&totalTracked)
	if err != nil {
		return nil, err
	}
	stats["total_tracked"] = totalTracked

	// Completed manga
	var completed int
	err = utm.db.conn.QueryRow("SELECT COUNT(*) FROM manga_update_tracking WHERE is_completed = TRUE").Scan(&completed)
	if err != nil {
		return nil, err
	}
	stats["completed"] = completed

	// Ongoing manga
	var ongoing int
	err = utm.db.conn.QueryRow("SELECT COUNT(*) FROM manga_update_tracking WHERE is_ongoing = TRUE").Scan(&ongoing)
	if err != nil {
		return nil, err
	}
	stats["ongoing"] = ongoing

	// Average update interval
	var avgInterval sql.NullFloat64
	err = utm.db.conn.QueryRow("SELECT AVG(avg_update_interval_days) FROM manga_update_tracking WHERE avg_update_interval_days IS NOT NULL").Scan(&avgInterval)
	if err != nil {
		return nil, err
	}
	if avgInterval.Valid {
		stats["avg_update_interval_days"] = avgInterval.Float64
	}

	return stats, nil
}
