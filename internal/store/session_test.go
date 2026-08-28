package store

import (
	"errors"
	"testing"
	"time"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:): %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// TestOpenIsIdempotent guards re-opening an existing file DB: the migration
// (including the schema_version read on an empty table) must not error.
func TestOpenIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/sk2.db"
	s1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer s2.Close()
}

func TestAddNewAndList(t *testing.T) {
	s := openTest(t)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	if err := s.Add("abc123", "claude", "ctx project", "a description", now); err != nil {
		t.Fatalf("Add: %v", err)
	}

	sessions, err := s.List("")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	got := sessions[0]
	if got.SessionKey != "abc123" || got.Agent != "claude" {
		t.Fatalf("unexpected session: %+v", got)
	}
	if got.Title == nil || *got.Title != "ctx project" {
		t.Fatalf("title mismatch: %+v", got.Title)
	}
	if got.Note == nil || *got.Note != "a description" {
		t.Fatalf("note mismatch: %+v", got.Note)
	}
	if !got.CreatedAt.Equal(now) {
		t.Fatalf("created_at mismatch: %v", got.CreatedAt)
	}
	if got.LastUsedAt == nil || !got.LastUsedAt.Equal(now) {
		t.Fatalf("last_used_at mismatch: %+v", got.LastUsedAt)
	}
}

func TestAddDuplicateUpsertPreservesCreatedAt(t *testing.T) {
	s := openTest(t)
	t1 := time.Date(2026, 8, 28, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 8, 28, 11, 30, 0, 0, time.UTC)

	if err := s.Add("k1", "claude", "old title", "old note", t1); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	// Re-add same key with new fields and a later timestamp.
	if err := s.Add("k1", "opencode", "new title", "new note", t2); err != nil {
		t.Fatalf("second Add (upsert): %v", err)
	}

	sessions, err := s.List("")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session after upsert, got %d", len(sessions))
	}
	got := sessions[0]
	if !got.CreatedAt.Equal(t1) {
		t.Fatalf("created_at must be preserved, got %v want %v", got.CreatedAt, t1)
	}
	if got.Agent != "opencode" || *got.Title != "new title" || *got.Note != "new note" {
		t.Fatalf("fields not overwritten: %+v", got)
	}
	if got.LastUsedAt == nil || !got.LastUsedAt.Equal(t2) {
		t.Fatalf("last_used_at must be updated, got %+v", got.LastUsedAt)
	}
}

func TestListFilterByAgent(t *testing.T) {
	s := openTest(t)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	if err := s.Add("a1", "claude", "", "", now); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("a2", "opencode", "", "", now); err != nil {
		t.Fatal(err)
	}

	only, err := s.List("claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(only) != 1 || only[0].SessionKey != "a1" {
		t.Fatalf("agent filter failed: %+v", only)
	}

	all, err := s.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(all))
	}
}

func TestListOrderingLastUsedThenCreated(t *testing.T) {
	s := openTest(t)
	// created_at differs, last_used_at equal -> order by created_at desc.
	if err := s.Add("old", "claude", "", "", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("new", "claude", "", "", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	// Bump "old" so it sorts first by last_used_at.
	recent := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	if err := s.Use("old", recent); err != nil {
		t.Fatal(err)
	}

	sessions, err := s.List("")
	if err != nil {
		t.Fatal(err)
	}
	if sessions[0].SessionKey != "old" || sessions[1].SessionKey != "new" {
		t.Fatalf("ordering wrong: %s then %s", sessions[0].SessionKey, sessions[1].SessionKey)
	}
}

func TestUseAndDelete(t *testing.T) {
	s := openTest(t)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	if err := s.Add("k", "claude", "", "", now); err != nil {
		t.Fatal(err)
	}

	later := now.Add(1 * time.Hour)
	if err := s.Use("k", later); err != nil {
		t.Fatalf("Use: %v", err)
	}
	sessions, err := s.List("")
	if err != nil {
		t.Fatal(err)
	}
	if sessions[0].LastUsedAt == nil || !sessions[0].LastUsedAt.Equal(later) {
		t.Fatalf("Use did not update last_used_at: %+v", sessions[0].LastUsedAt)
	}

	if err := s.Delete("k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	sessions, err = s.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions after delete, got %d", len(sessions))
	}
}

func TestUseAndDeleteNotFound(t *testing.T) {
	s := openTest(t)
	if err := s.Use("missing", time.Now()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Use missing key: want ErrNotFound, got %v", err)
	}
	if err := s.Delete("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete missing key: want ErrNotFound, got %v", err)
	}
}

func TestAddEmptyValidation(t *testing.T) {
	s := openTest(t)
	now := time.Now()
	if err := s.Add("", "claude", "", "", now); err == nil {
		t.Fatal("expected error for empty key")
	}
	if err := s.Add("k", "", "", "", now); err == nil {
		t.Fatal("expected error for empty agent")
	}
}

func TestSearchByKeyOrNote(t *testing.T) {
	s := openTest(t)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	// Match by key substring.
	if err := s.Add("session-abc-123", "claude", "Project X", "ctx about billing", now); err != nil {
		t.Fatal(err)
	}
	// Match by note substring (key does not contain it).
	if err := s.Add("other-key", "opencode", "Side", "notes about auth flow", now); err != nil {
		t.Fatal(err)
	}
	// Non-matching record.
	if err := s.Add("unrelated", "claude", "", "nothing here", now); err != nil {
		t.Fatal(err)
	}

	byKey, err := s.Search("abc-123")
	if err != nil {
		t.Fatal(err)
	}
	if len(byKey) != 1 || byKey[0].SessionKey != "session-abc-123" {
		t.Fatalf("key substring search failed: %+v", byKey)
	}

	byNote, err := s.Search("auth")
	if err != nil {
		t.Fatal(err)
	}
	if len(byNote) != 1 || byNote[0].SessionKey != "other-key" {
		t.Fatalf("note substring search failed: %+v", byNote)
	}

	none, err := s.Search("zzz-no-such")
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no matches, got %+v", none)
	}
}

func TestSearchEscapesLikeWildcards(t *testing.T) {
	s := openTest(t)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	if err := s.Add("literal_100", "claude", "", "", now); err != nil {
		t.Fatal(err)
	}
	// A bare "%" must match literally, not act as a wildcard for every row.
	if err := s.Add("anything", "opencode", "", "", now); err != nil {
		t.Fatal(err)
	}
	matches, err := s.Search("%")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("bare %% must not act as wildcard, got %+v", matches)
	}
}
