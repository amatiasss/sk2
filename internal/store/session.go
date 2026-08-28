package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Session is a single stored session-key record.
//
// Title, Note and LastUsedAt are optional and therefore represented as
// pointers. Timestamps are time.Time and persisted in RFC3339 format.
type Session struct {
	ID         int64
	SessionKey string
	Agent      string
	Title      *string
	Note       *string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// Add upserts a session. If the key already exists, created_at is preserved
// while agent/title/note/last_used_at are overwritten with the provided values.
// If the key is new, created_at and last_used_at are set to now.
func (s *Store) Add(key, agent, title, note string, now time.Time) error {
	if key == "" {
		return errors.New("session key must not be empty")
	}
	if agent == "" {
		return errors.New("agent must not be empty")
	}
	// Non-empty strings are stored directly; empty means "not provided".
	var titlePtr, notePtr *string
	if title != "" {
		titlePtr = &title
	}
	if note != "" {
		notePtr = &note
	}
	nowStr := now.UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`
INSERT INTO sessions (session_key, agent, title, note, created_at, last_used_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(session_key) DO UPDATE SET
  agent        = excluded.agent,
  title        = excluded.title,
  note         = excluded.note,
  last_used_at = excluded.last_used_at`,
		key, agent, titlePtr, notePtr, nowStr, nowStr)
	if err != nil {
		return fmt.Errorf("upsert session: %w", err)
	}
	return nil
}

// List returns all sessions, optionally filtered by agent. Results are ordered
// by last_used_at descending, falling back to created_at descending.
func (s *Store) List(agentFilter string) ([]Session, error) {
	query := `
SELECT id, session_key, agent, title, note, created_at, last_used_at
FROM sessions`
	args := []any{}
	if agentFilter != "" {
		query += ` WHERE agent = ?`
		args = append(args, agentFilter)
	}
	query += ` ORDER BY last_used_at DESC, created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	return scanSessions(rows)
}

// Search returns the sessions whose session_key or note contains the given
// query (case-insensitive substring match). Results are ordered by last_used_at
// descending, falling back to created_at descending.
func (s *Store) Search(query string) ([]Session, error) {
	pattern := "%" + escapeLike(query) + "%"
	rows, err := s.db.Query(`
SELECT id, session_key, agent, title, note, created_at, last_used_at
FROM sessions
WHERE session_key LIKE ? ESCAPE '\' OR note LIKE ? ESCAPE '\'
ORDER BY last_used_at DESC, created_at DESC`, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("query sessions (search): %w", err)
	}
	defer rows.Close()

	return scanSessions(rows)
}

// scanSessions maps the current result-set rows into []Session.
func scanSessions(rows *sql.Rows) ([]Session, error) {
	var sessions []Session
	for rows.Next() {
		var (
			se          Session
			title, note sql.NullString
			createdRaw  string
			lastRaw     sql.NullString
		)
		if err := rows.Scan(&se.ID, &se.SessionKey, &se.Agent, &title, &note, &createdRaw, &lastRaw); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		se.Title = nullStringPtr(title)
		se.Note = nullStringPtr(note)
		createdAt, err := time.Parse(time.RFC3339, createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
		}
		se.CreatedAt = createdAt
		if lastRaw.Valid {
			t, err := time.Parse(time.RFC3339, lastRaw.String)
			if err != nil {
				return nil, fmt.Errorf("parse last_used_at %q: %w", lastRaw.String, err)
			}
			se.LastUsedAt = &t
		}
		sessions = append(sessions, se)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return sessions, nil
}

// escapeLike escapes the LIKE wildcards % and _ (and the escape char itself)
// so user input is treated as a literal substring, not a pattern.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// Has reports whether a session with the given key already exists.
func (s *Store) Has(key string) (bool, error) {
	var one int
	err := s.db.QueryRow(`SELECT 1 FROM sessions WHERE session_key = ?`, key).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check session: %w", err)
	}
	return true, nil
}

// Delete removes the session with the given key. It returns ErrNotFound when
// no such session exists.
func (s *Store) Delete(key string) error {
	res, err := s.db.Exec(`DELETE FROM sessions WHERE session_key = ?`, key)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete session (rows affected): %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Use bumps last_used_at for the session with the given key. It returns
// ErrNotFound when no such session exists.
func (s *Store) Use(key string, now time.Time) error {
	res, err := s.db.Exec(`UPDATE sessions SET last_used_at = ? WHERE session_key = ?`,
		now.UTC().Format(time.RFC3339), key)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update session (rows affected): %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func nullStringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := ns.String
	return &v
}
