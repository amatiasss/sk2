package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"sk2/internal/store"
)

// NewAddCommand returns the `sk2 add` subcommand (upsert).
func NewAddCommand(st *store.Store) *cobra.Command {
	var agent, title, note string
	cmd := &cobra.Command{
		Use:   "add <session-key>",
		Short: "Store a session key (upserts if the key already exists)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if agent == "" {
				return fmt.Errorf("--agent/-a is required")
			}
			key := args[0]
			exists, err := st.Has(key)
			if err != nil {
				return err
			}
			now := time.Now()
			if err := st.Add(key, agent, title, note, now); err != nil {
				return err
			}
			if exists {
				cmd.Println("updated")
			} else {
				cmd.Println("added")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&agent, "agent", "a", "", "agent/tool that owns the session (required)")
	cmd.Flags().StringVarP(&title, "title", "t", "", "short label for the session")
	cmd.Flags().StringVarP(&note, "note", "n", "", "long description of the session")
	return cmd
}
