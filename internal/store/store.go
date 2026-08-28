// Package store provides SQLite-backed persistence for sk2 session records.
//
// The SQLite driver is modernc.org/sqlite (pure Go, no CGO). mattn/go-sqlite3
// is intentionally never imported — it would break the portable build.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned by Delete and Use when no session matches the key.
var ErrNotFound = errors.New("session not found")

// currentSchemaVersion is the latest migration version applied by Open.
const currentSchemaVersion = 1

// Store wraps a SQLite database handle.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite database at path, enables WAL
// mode and runs idempotent migrations. The parent directory is created if it
// does not exist.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	// Enable WAL to guard against data loss / concurrent access.
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// migrate applies the schema idempotently, tracking the version in
// schema_version. Future migrations bump currentSchemaVersion and add steps
// below.
func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS sessions (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  session_key  TEXT NOT NULL UNIQUE,
  agent        TEXT NOT NULL,
  title        TEXT,
  note         TEXT,
  created_at   TEXT NOT NULL,
  last_used_at TEXT
)`); err != nil {
		return fmt.Errorf("create sessions: %w", err)
	}
	// Record current version (idempotent: keep the highest already applied).
	var version int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read schema version: %w", err)
	}
	if version < currentSchemaVersion {
		if _, err := s.db.Exec(`INSERT INTO schema_version (version) VALUES (?)`, currentSchemaVersion); err != nil {
			return fmt.Errorf("record schema version: %w", err)
		}
	}
	return nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// ResolveDBPath returns the database path, honouring XDG_DATA_HOME when set and
// falling back to ~/.local/share/sk2/sk2.db otherwise. It errors if the HOME
// directory cannot be determined.
func ResolveDBPath() (string, error) {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "sk2", "sk2.db"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory (set XDG_DATA_HOME or HOME): %w", err)
	}
	return filepath.Join(home, ".local", "share", "sk2", "sk2.db"), nil
}
