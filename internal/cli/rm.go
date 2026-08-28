package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"sk2/internal/store"
)

// NewRemoveCommand returns the `sk2 rm` subcommand.
func NewRemoveCommand(st *store.Store) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <session-key>",
		Short: "Delete a stored session key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := st.Delete(args[0]); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return fmt.Errorf("no session found for key %q: %w", args[0], store.ErrNotFound)
				}
				return err
			}
			cmd.Println("deleted")
			return nil
		},
	}
	return cmd
}
