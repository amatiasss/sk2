package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"sk2/internal/store"
)

// NewGetCommand returns the `sk2 get` subcommand, which shows the title and
// note of the session(s) matching a substring of the key or the note.
func NewGetCommand(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <query>",
		Short: "Show a session's title and note (matches part of key or note)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessions, err := st.Search(args[0])
			if err != nil {
				return err
			}
			if len(sessions) == 0 {
				return fmt.Errorf("no session matches %q: %w", args[0], store.ErrNotFound)
			}
			renderTable(cmd.OutOrStdout(), sessions, fieldsBoth, true)
			return nil
		},
	}
	return cmd
}
