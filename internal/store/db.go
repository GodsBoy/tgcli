package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database for tgcli.
type DB struct {
	path       string
	sql        *sql.DB
	ftsEnabled bool
}

// Open opens (or creates) the SQLite database at the given path
// and runs all migrations.
func Open(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000", path)
	sqlDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Set pragmas for performance.
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA temp_store=MEMORY",
	}
	for _, p := range pragmas {
		if _, err := sqlDB.Exec(p); err != nil {
			sqlDB.Close()
			return nil, fmt.Errorf("exec pragma %q: %w", p, err)
		}
	}

	d := &DB{path: path, sql: sqlDB}

	if err := d.ensureSchema(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	return d, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	if d.sql != nil {
		return d.sql.Close()
	}
	return nil
}

// FTSEnabled returns true if FTS5 is available.
func (d *DB) FTSEnabled() bool {
	return d.ftsEnabled
}

// Path returns the database file path.
func (d *DB) Path() string {
	return d.path
}
