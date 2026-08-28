package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"sk2/internal/cli"
	"sk2/internal/store"
)

// openTempStore opens a real (file-backed) store under t.TempDir(), exercising
// the same code path main() uses rather than an in-memory DB.
func openTempStore(t *testing.T) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "sk2.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("Open(%s): %v", dbPath, err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func execute(t *testing.T, st *store.Store, args ...string) (string, error) {
	t.Helper()
	root := cli.NewRootCommand(st)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestFullFlow(t *testing.T) {
	st := openTempStore(t)

	// add -> list -> use -> rm
	if _, err := execute(t, st, "add", "abc123", "-a", "claude", "-t", "ctx project", "-n", "desc"); err != nil {
		t.Fatalf("add: %v", err)
	}
	list, err := execute(t, st, "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, want := range []string{"abc123", "claude", "ctx project"} {
		if !strings.Contains(list, want) {
			t.Fatalf("list missing %q:\n%s", want, list)
		}
	}

	use, err := execute(t, st, "use", "abc123")
	if err != nil {
		t.Fatalf("use: %v", err)
	}
	if strings.TrimSpace(use) != "abc123" {
		t.Fatalf("use should print raw key, got %q", use)
	}

	if _, err := execute(t, st, "rm", "abc123"); err != nil {
		t.Fatalf("rm: %v", err)
	}
}

func TestUpsertOnDuplicateAdd(t *testing.T) {
	st := openTempStore(t)
	if _, err := execute(t, st, "add", "k", "-a", "claude", "-t", "old", "-n", "oldnote"); err != nil {
		t.Fatal(err)
	}
	if _, err := execute(t, st, "add", "k", "-a", "opencode", "-t", "new", "-n", "newnote"); err != nil {
		t.Fatalf("second add should not error: %v", err)
	}

	sessions, err := st.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session after upsert, got %d", len(sessions))
	}
	got := sessions[0]
	if got.Agent != "opencode" || *got.Title != "new" || *got.Note != "newnote" {
		t.Fatalf("fields not overwritten on upsert: %+v", got)
	}
}

func TestListFieldsModesIntegration(t *testing.T) {
	st := openTempStore(t)
	if _, err := execute(t, st, "add", "key1", "-a", "claude", "-t", "T"); err != nil {
		t.Fatal(err)
	}

	both, err := execute(t, st, "list", "-f", "both")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(both, "TITLE") || !strings.Contains(both, "SESSION-KEY") {
		t.Fatalf("both mode: %s", both)
	}

	title, err := execute(t, st, "list", "-f", "title")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(title, "SESSION-KEY") {
		t.Fatalf("title mode should hide SESSION-KEY:\n%s", title)
	}

	key, err := execute(t, st, "list", "-f", "key")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(key, "TITLE") {
		t.Fatalf("key mode should hide TITLE:\n%s", key)
	}
}

func TestMissingKeyErrorPaths(t *testing.T) {
	st := openTempStore(t)
	// rm of a missing key -> ErrNotFound.
	_, err := execute(t, st, "rm", "ghost")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("rm missing key: want ErrNotFound, got %v", err)
	}
	// use of a missing key -> ErrNotFound.
	_, err = execute(t, st, "use", "ghost")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("use missing key: want ErrNotFound, got %v", err)
	}
}

// TestResolveDBPathFallback guards the XDG_DATA_HOME / HOME fallback logic.
func TestResolveDBPath(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/tmp/xdgdata")
	if got, _ := store.ResolveDBPath(); got != filepath.Join("/tmp/xdgdata", "sk2", "sk2.db") {
		t.Fatalf("XDG path: got %q", got)
	}
	t.Setenv("XDG_DATA_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got, err := store.ResolveDBPath(); err != nil || got != filepath.Join(home, ".local", "share", "sk2", "sk2.db") {
		t.Fatalf("HOME fallback: got %q err %v", got, err)
	}
}
