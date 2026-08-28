package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"sk2/internal/store"
)

// NewUseCommand returns the `sk2 use` subcommand, which bumps last_used_at and
// prints the raw session key (suitable for piping).
func NewUseCommand(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <session-key>",
		Short: "Mark a session as used and print its key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := st.Use(args[0], time.Now()); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return fmt.Errorf("no session found for key %q: %w", args[0], store.ErrNotFound)
				}
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), args[0])
			return nil
		},
	}
	return cmd
}
