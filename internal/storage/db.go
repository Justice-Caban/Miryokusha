package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbFileName = "miryokusha.db"
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
}

// NewDB creates a new database connection
func NewDB() (*DB, error) {
	dbPath, err := getDBPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get database path: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// getDBPath returns the path to the database file
func getDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "miryokusha", dbFileName), nil
}

// initSchema initializes the database schema
func (db *DB) initSchema() error {
	schema := `
	-- Reading history table
	CREATE TABLE IF NOT EXISTS reading_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		manga_id TEXT NOT NULL,
		manga_title TEXT NOT NULL,
		chapter_id TEXT NOT NULL,
		chapter_number REAL NOT NULL,
		chapter_title TEXT,
		source_type TEXT NOT NULL, -- 'suwayomi' or 'local'
		source_id TEXT,
		read_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(manga_id, chapter_id)
	);

	-- Reading progress table
	CREATE TABLE IF NOT EXISTS reading_progress (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		manga_id TEXT NOT NULL,
		chapter_id TEXT NOT NULL,
		current_page INTEGER DEFAULT 0,
		total_pages INTEGER NOT NULL,
		is_completed BOOLEAN DEFAULT FALSE,
		last_read_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(manga_id, chapter_id)
	);

	-- Bookmarks table
	CREATE TABLE IF NOT EXISTS bookmarks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		manga_id TEXT NOT NULL,
		manga_title TEXT NOT NULL,
		chapter_id TEXT NOT NULL,
		chapter_number REAL NOT NULL,
		chapter_title TEXT,
		page_number INTEGER NOT NULL,
		note TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Reading sessions table (for statistics)
	CREATE TABLE IF NOT EXISTS reading_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		manga_id TEXT NOT NULL,
		chapter_id TEXT NOT NULL,
		duration_seconds INTEGER NOT NULL,
		pages_read INTEGER NOT NULL,
		session_start TIMESTAMP NOT NULL,
		session_end TIMESTAMP NOT NULL
	);

	-- Manga metadata cache (optional, for faster lookups)
	CREATE TABLE IF NOT EXISTS manga_cache (
		manga_id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT,
		artist TEXT,
		description TEXT,
		cover_url TEXT,
		source_type TEXT NOT NULL,
		source_id TEXT,
		last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Categories table
	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		sort_order INTEGER NOT NULL DEFAULT 0,
		is_default BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Manga-Category relationship table (many-to-many)
	CREATE TABLE IF NOT EXISTS manga_categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		manga_id TEXT NOT NULL,
		category_id INTEGER NOT NULL,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
		UNIQUE(manga_id, category_id)
	);

	-- Indices for better performance
	CREATE INDEX IF NOT EXISTS idx_history_manga ON reading_history(manga_id);
	CREATE INDEX IF NOT EXISTS idx_history_read_at ON reading_history(read_at DESC);
	CREATE INDEX IF NOT EXISTS idx_progress_manga ON reading_progress(manga_id);
	CREATE INDEX IF NOT EXISTS idx_progress_last_read ON reading_progress(last_read_at DESC);
	CREATE INDEX IF NOT EXISTS idx_bookmarks_manga ON bookmarks(manga_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_manga ON reading_sessions(manga_id);
	CREATE INDEX IF NOT EXISTS idx_manga_categories_manga ON manga_categories(manga_id);
	CREATE INDEX IF NOT EXISTS idx_manga_categories_category ON manga_categories(category_id);
	CREATE INDEX IF NOT EXISTS idx_categories_sort ON categories(sort_order);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// GetConnection returns the underlying database connection
func (db *DB) GetConnection() *sql.DB {
	return db.conn
}
