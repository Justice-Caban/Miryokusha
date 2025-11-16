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

	// Open database with performance pragmas
	// WAL mode: Better concurrency for read/write operations
	// Busy timeout: Handles concurrent access gracefully (5 seconds)
	// Synchronous NORMAL: Faster writes while maintaining safety
	// Cache size: 10MB cache for better query performance
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL&_cache_size=-10000&_foreign_keys=ON", dbPath)
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for optimal SQLite performance
	// SQLite benefits from a single writer connection
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0) // Connections live forever

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
	// Create schema version table first
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	// Check current schema version
	var currentVersion int
	err = db.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}

	// If schema is already at latest version, skip
	const latestVersion = 3
	if currentVersion >= latestVersion {
		return nil
	}

	// Apply schema migrations
	if currentVersion < 1 {
		if err := db.applySchemaV1(); err != nil {
			return fmt.Errorf("failed to apply schema v1: %w", err)
		}

		// Record schema version
		_, err = db.conn.Exec("INSERT INTO schema_version (version) VALUES (?)", 1)
		if err != nil {
			return fmt.Errorf("failed to record schema version: %w", err)
		}
	}

	if currentVersion < 2 {
		if err := db.applySchemaV2(); err != nil {
			return fmt.Errorf("failed to apply schema v2: %w", err)
		}

		// Record schema version
		_, err = db.conn.Exec("INSERT INTO schema_version (version) VALUES (?)", 2)
		if err != nil {
			return fmt.Errorf("failed to record schema version: %w", err)
		}
	}

	if currentVersion < 3 {
		if err := db.applySchemaV3(); err != nil {
			return fmt.Errorf("failed to apply schema v3: %w", err)
		}

		// Record schema version
		_, err = db.conn.Exec("INSERT INTO schema_version (version) VALUES (?)", 3)
		if err != nil {
			return fmt.Errorf("failed to record schema version: %w", err)
		}
	}

	return nil
}

// applySchemaV1 applies the initial schema (version 1)
func (db *DB) applySchemaV1() error {
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

// applySchemaV2 adds smart update tracking (version 2)
func (db *DB) applySchemaV2() error {
	schema := `
	-- Manga update tracking for smart updates (Mihon-style)
	CREATE TABLE IF NOT EXISTS manga_update_tracking (
		manga_id TEXT PRIMARY KEY,
		last_check TIMESTAMP NOT NULL,
		last_chapter_found TIMESTAMP,
		chapter_count INTEGER DEFAULT 0,
		avg_update_interval_days REAL,
		fetch_count INTEGER DEFAULT 1,
		consecutive_failures INTEGER DEFAULT 0,
		is_completed BOOLEAN DEFAULT FALSE,
		is_ongoing BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Index for smart update queries
	CREATE INDEX IF NOT EXISTS idx_update_tracking_last_check ON manga_update_tracking(last_check);
	CREATE INDEX IF NOT EXISTS idx_update_tracking_completed ON manga_update_tracking(is_completed);
	CREATE INDEX IF NOT EXISTS idx_update_tracking_ongoing ON manga_update_tracking(is_ongoing);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// applySchemaV3 adds manga_title to reading_progress (version 3)
func (db *DB) applySchemaV3() error {
	schema := `
	-- Add manga_title column to reading_progress for better UX
	ALTER TABLE reading_progress ADD COLUMN manga_title TEXT DEFAULT '';
	`

	_, err := db.conn.Exec(schema)
	return err
}

// GetConnection returns the underlying database connection
func (db *DB) GetConnection() *sql.DB {
	return db.conn
}

// WithTransaction executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, the transaction is committed
func (db *DB) WithTransaction(fn func(*sql.Tx) error) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetSchemaVersion returns the current schema version
func (db *DB) GetSchemaVersion() (int, error) {
	var version int
	err := db.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get schema version: %w", err)
	}
	return version, nil
}
