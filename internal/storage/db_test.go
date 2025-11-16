package storage

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// NewTestDB creates an in-memory database for testing
func NewTestDB(t *testing.T) *DB {
	conn, err := sql.Open("sqlite3", ":memory:?_foreign_keys=ON")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	db := &DB{conn: conn}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		t.Fatalf("Failed to initialize test schema: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("Failed to close test database: %v", err)
		}
	})

	return db
}

func TestDB_SchemaVersioning(t *testing.T) {
	db := NewTestDB(t)

	// Verify schema version table exists
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query schema_version: %v", err)
	}

	if count == 0 {
		t.Error("Expected at least one schema version entry")
	}

	// Verify current version is 3
	var version int
	err = db.conn.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to get schema version: %v", err)
	}

	if version != 3 {
		t.Errorf("Expected schema version 3, got %d", version)
	}
}

func TestDB_ReadingProgressTable(t *testing.T) {
	db := NewTestDB(t)

	// Verify reading_progress table exists and has correct columns
	rows, err := db.conn.Query("PRAGMA table_info(reading_progress)")
	if err != nil {
		t.Fatalf("Failed to query table info: %v", err)
	}
	defer rows.Close()

	expectedColumns := map[string]bool{
		"id":           false,
		"manga_id":     false,
		"manga_title":  false,
		"chapter_id":   false,
		"current_page": false,
		"total_pages":  false,
		"is_completed": false,
		"last_read_at": false,
	}

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, dflt_value, pk interface{}

		err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt_value, &pk)
		if err != nil {
			t.Fatalf("Failed to scan column info: %v", err)
		}

		if _, exists := expectedColumns[name]; exists {
			expectedColumns[name] = true
		}
	}

	for col, found := range expectedColumns {
		if !found {
			t.Errorf("Expected column %s not found in reading_progress table", col)
		}
	}
}

func TestDB_ReadingHistoryTable(t *testing.T) {
	db := NewTestDB(t)

	// Verify reading_history table exists
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM reading_history WHERE 1=0").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query reading_history: %v", err)
	}
}

func TestDB_CategoriesTable(t *testing.T) {
	db := NewTestDB(t)

	// Verify categories table exists
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM categories WHERE 1=0").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query categories: %v", err)
	}
}

func TestDB_MangaCategoriesTable(t *testing.T) {
	db := NewTestDB(t)

	// Verify manga_categories junction table exists
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM manga_categories WHERE 1=0").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query manga_categories: %v", err)
	}
}

func TestDB_UpdateTrackingTable(t *testing.T) {
	db := NewTestDB(t)

	// Verify manga_update_tracking table exists (schema v2)
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM manga_update_tracking WHERE 1=0").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query manga_update_tracking: %v", err)
	}
}

func TestDB_Transaction(t *testing.T) {
	db := NewTestDB(t)

	// Test WithTransaction helper
	var testValue string
	err := db.WithTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO reading_history (manga_id, manga_title, chapter_id, chapter_number, source_type) VALUES (?, ?, ?, ?, ?)",
			"test-manga", "Test Manga", "test-ch1", 1, "suwayomi")
		if err != nil {
			return err
		}

		// Query within transaction
		return tx.QueryRow("SELECT manga_id FROM reading_history WHERE manga_id = ?", "test-manga").Scan(&testValue)
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	if testValue != "test-manga" {
		t.Errorf("Expected 'test-manga', got '%s'", testValue)
	}

	// Verify data persisted
	var persisted string
	err = db.conn.QueryRow("SELECT manga_id FROM reading_history WHERE manga_id = ?", "test-manga").Scan(&persisted)
	if err != nil {
		t.Fatalf("Failed to verify persisted data: %v", err)
	}

	if persisted != "test-manga" {
		t.Errorf("Data did not persist after transaction")
	}
}

func TestDB_TransactionRollback(t *testing.T) {
	db := NewTestDB(t)

	// Test that rollback works on error
	err := db.WithTransaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO reading_history (manga_id, manga_title, chapter_id, chapter_number, source_type) VALUES (?, ?, ?, ?, ?)",
			"rollback-test", "Rollback Test", "ch1", 1, "suwayomi")
		if err != nil {
			return err
		}

		// Force error to trigger rollback
		return fmt.Errorf("intentional error")
	})

	if err == nil || err.Error() != "intentional error" {
		t.Fatalf("Expected intentional error, got: %v", err)
	}

	// Verify data was rolled back
	var count int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM reading_history WHERE manga_id = ?", "rollback-test").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify rollback: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 rows after rollback, got %d", count)
	}
}

func TestDB_ForeignKeys(t *testing.T) {
	db := NewTestDB(t)

	// Verify foreign keys are enabled
	var fkEnabled int
	err := db.conn.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("Failed to check foreign_keys pragma: %v", err)
	}

	if fkEnabled != 1 {
		t.Error("Foreign keys should be enabled")
	}
}
