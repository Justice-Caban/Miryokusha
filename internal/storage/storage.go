package storage

import (
	"fmt"
)

// Storage provides a unified interface to all storage managers
type Storage struct {
	db            *DB
	History       *HistoryManager
	Progress      *ProgressManager
	Bookmarks     *BookmarkManager
	Stats         *StatsManager
	Categories    *CategoryManager
	UpdateTracking *UpdateTrackingManager
}

// NewStorage creates a new storage instance with all managers
func NewStorage() (*Storage, error) {
	db, err := NewDB()
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	storage := &Storage{
		db:             db,
		History:        NewHistoryManager(db),
		Progress:       NewProgressManager(db),
		Bookmarks:      NewBookmarkManager(db),
		Stats:          NewStatsManager(db),
		Categories:     NewCategoryManager(db),
		UpdateTracking: NewUpdateTrackingManager(db),
	}

	// Initialize default categories if needed
	if err := storage.Categories.InitializeDefaultCategories(); err != nil {
		return nil, fmt.Errorf("failed to initialize default categories: %w", err)
	}

	return storage, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// GetDB returns the underlying database for advanced operations
func (s *Storage) GetDB() *DB {
	return s.db
}

// ExportData exports all data for backup purposes
func (s *Storage) ExportData() (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// Export history
	history, err := s.History.GetRecentHistory(10000) // Get all history
	if err != nil {
		return nil, fmt.Errorf("failed to export history: %w", err)
	}
	data["history"] = history

	// Export bookmarks
	bookmarks, err := s.Bookmarks.GetAllBookmarks()
	if err != nil {
		return nil, fmt.Errorf("failed to export bookmarks: %w", err)
	}
	data["bookmarks"] = bookmarks

	// Export global stats
	stats, err := s.Stats.GetGlobalStats()
	if err != nil {
		return nil, fmt.Errorf("failed to export stats: %w", err)
	}
	data["stats"] = stats

	return data, nil
}

// ClearAllData removes all data from the database (use with caution!)
func (s *Storage) ClearAllData() error {
	if err := s.History.ClearAllHistory(); err != nil {
		return fmt.Errorf("failed to clear history: %w", err)
	}

	// Clear progress
	_, err := s.db.conn.Exec("DELETE FROM reading_progress")
	if err != nil {
		return fmt.Errorf("failed to clear progress: %w", err)
	}

	// Clear bookmarks
	_, err = s.db.conn.Exec("DELETE FROM bookmarks")
	if err != nil {
		return fmt.Errorf("failed to clear bookmarks: %w", err)
	}

	if err := s.Stats.ClearAllSessions(); err != nil {
		return fmt.Errorf("failed to clear sessions: %w", err)
	}

	// Clear manga cache
	_, err = s.db.conn.Exec("DELETE FROM manga_cache")
	if err != nil {
		return fmt.Errorf("failed to clear manga cache: %w", err)
	}

	return nil
}
