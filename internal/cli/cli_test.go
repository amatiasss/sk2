package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"sk2/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:): %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// run executes a command tree with the given args and returns stdout.
func run(t *testing.T, st *store.Store, args ...string) (string, error) {
	t.Helper()
	root := NewRootCommand(st)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func TestAddRequiresAgent(t *testing.T) {
	st := newTestStore(t)
	_, err := run(t, st, "add", "key1")
	if err == nil {
		t.Fatal("expected error when --agent is missing")
	}
	if !strings.Contains(err.Error(), "--agent") {
		t.Fatalf("error should mention --agent: %v", err)
	}
}

func TestAddUpsertReportsUpdated(t *testing.T) {
	st := newTestStore(t)
	out1, err := run(t, st, "add", "key1", "-a", "claude", "-t", "title", "-n", "note")
	if err != nil {
		t.Fatalf("first add: %v", err)
	}
	if !strings.Contains(out1, "added") {
		t.Fatalf("expected 'added', got %q", out1)
	}
	out2, err := run(t, st, "add", "key1", "-a", "opencode")
	if err != nil {
		t.Fatalf("second add (upsert): %v", err)
	}
	if !strings.Contains(out2, "updated") {
		t.Fatalf("expected 'updated' on duplicate, got %q", out2)
	}
}

func TestListEmpty(t *testing.T) {
	st := newTestStore(t)
	out, err := run(t, st, "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "(no records)") {
		t.Fatalf("expected '(no records)', got %q", out)
	}
}

func TestListFieldsModes(t *testing.T) {
	st := newTestStore(t)
	if _, err := run(t, st, "add", "key1", "-a", "claude", "-t", "MyTitle"); err != nil {
		t.Fatal(err)
	}

	// both: shows TITLE and SESSION-KEY.
	both, err := run(t, st, "list", "-f", "both")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(both, "TITLE") || !strings.Contains(both, "SESSION-KEY") {
		t.Fatalf("both mode should show both columns:\n%s", both)
	}
	if !strings.Contains(both, "MyTitle") || !strings.Contains(both, "key1") {
		t.Fatalf("both mode should show title and key:\n%s", both)
	}

	// title: hides SESSION-KEY.
	title, err := run(t, st, "list", "-f", "title")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(title, "TITLE") || strings.Contains(title, "SESSION-KEY") {
		t.Fatalf("title mode should show TITLE only:\n%s", title)
	}

	// key: hides TITLE.
	key, err := run(t, st, "list", "-f", "key")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(key, "SESSION-KEY") || strings.Contains(key, "TITLE") {
		t.Fatalf("key mode should show SESSION-KEY only:\n%s", key)
	}
}

func TestListInvalidFields(t *testing.T) {
	st := newTestStore(t)
	_, err := run(t, st, "list", "-f", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid --fields")
	}
}

func TestListNoteHiddenByDefaultAndShownWithFlag(t *testing.T) {
	st := newTestStore(t)
	if _, err := run(t, st, "add", "k1", "-a", "claude", "-n", "secret note"); err != nil {
		t.Fatal(err)
	}

	def, err := run(t, st, "list")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(def, "NOTE") || strings.Contains(def, "secret note") {
		t.Fatalf("note should be hidden by default in list:\n%s", def)
	}

	withNote, err := run(t, st, "list", "--note")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(withNote, "NOTE") || !strings.Contains(withNote, "secret note") {
		t.Fatalf("--note should show the note column:\n%s", withNote)
	}
}

func TestListAgentFilter(t *testing.T) {
	st := newTestStore(t)
	if _, err := run(t, st, "add", "k1", "-a", "claude"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, st, "add", "k2", "-a", "opencode"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, st, "list", "-a", "claude")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "k2") || !strings.Contains(out, "k1") {
		t.Fatalf("agent filter failed:\n%s", out)
	}
}

func TestUsePrintsKey(t *testing.T) {
	st := newTestStore(t)
	if _, err := run(t, st, "add", "abc123", "-a", "claude"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, st, "use", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "abc123" {
		t.Fatalf("use should print raw key, got %q", out)
	}
}

func TestUseNotFound(t *testing.T) {
	st := newTestStore(t)
	_, err := run(t, st, "use", "nope")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRmSuccessAndNotFound(t *testing.T) {
	st := newTestStore(t)
	if _, err := run(t, st, "add", "abc", "-a", "claude"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, st, "rm", "abc")
	if err != nil {
		t.Fatalf("rm existing: %v", err)
	}
	if !strings.Contains(out, "deleted") {
		t.Fatalf("expected 'deleted', got %q", out)
	}
	// Second rm of the same key -> not found.
	if _, err := run(t, st, "rm", "abc"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on second rm, got %v", err)
	}
}

func TestGetShowsTitleAndNote(t *testing.T) {
	st := newTestStore(t)
	if _, err := run(t, st, "add", "session-xyz", "-a", "claude", "-t", "Billing", "-n", "ctx about payments"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, st, "get", "session-xyz")
	if err != nil {
		t.Fatalf("get by full key: %v", err)
	}
	for _, want := range []string{"session-xyz", "Billing", "ctx about payments"} {
		if !strings.Contains(out, want) {
			t.Fatalf("get output missing %q:\n%s", want, out)
		}
	}

	// Partial key also matches.
	if _, err := run(t, st, "get", "xyz"); err != nil {
		t.Fatalf("get by partial key: %v", err)
	}

	// Match by note substring.
	byNote, err := run(t, st, "get", "payments")
	if err != nil {
		t.Fatalf("get by note: %v", err)
	}
	if !strings.Contains(byNote, "session-xyz") {
		t.Fatalf("note search should return the record:\n%s", byNote)
	}
}

func TestGetNotFound(t *testing.T) {
	st := newTestStore(t)
	if _, err := run(t, st, "add", "abc", "-a", "claude"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, st, "get", "no-such-key"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for no match, got %v", err)
	}
}
