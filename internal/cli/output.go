package cli

import (
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"sk2/internal/store"
)

// flagError reports an invalid flag value.
type flagError struct {
	name    string
	value   string
	allowed []string
}

func (e *flagError) Error() string {
	return fmt.Sprintf("invalid value %q for %s: must be one of %v", e.value, e.name, e.allowed)
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.UTC().Format(time.RFC3339)
}

// renderTable writes the sessions to w as a tabwriter-aligned table. The ID,
// AGENT and timestamp columns are always present; TITLE / SESSION-KEY follow
// the fields mode. The NOTE column is shown only when showNote is true (get
// shows it, list hides it by default). Both `list` and `get` use this renderer
// so their output stays standardized.
func renderTable(w io.Writer, sessions []store.Session, fields string, showNote bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	defer tw.Flush()

	header := "ID\tAGENT"
	format := "%d\t%s"
	titleMode := fields == fieldsBoth || fields == fieldsTitle
	keyMode := fields == fieldsBoth || fields == fieldsKey
	if titleMode {
		header += "\tTITLE"
		format += "\t%s"
	}
	if keyMode {
		header += "\tSESSION-KEY"
		format += "\t%s"
	}
	if showNote {
		header += "\tNOTE"
		format += "\t%s"
	}
	header += "\tCREATED\tLAST-USED"
	format += "\t%s\t%s\n"
	fmt.Fprintln(tw, header)

	if len(sessions) == 0 {
		fmt.Fprintln(tw, "(no records)")
		return
	}
	for _, s := range sessions {
		args := []any{s.ID, s.Agent}
		if titleMode {
			args = append(args, deref(s.Title))
		}
		if keyMode {
			args = append(args, s.SessionKey)
		}
		if showNote {
			args = append(args, deref(s.Note))
		}
		args = append(args, fmtTime(&s.CreatedAt), fmtTime(s.LastUsedAt))
		fmt.Fprintf(tw, format, args...)
	}
}

func deref(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}
