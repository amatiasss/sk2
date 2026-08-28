package cli

import (
	"github.com/spf13/cobra"

	"sk2/internal/store"
)

// fields modes controlling which of TITLE / SESSION-KEY are shown.
const (
	fieldsBoth  = "both"
	fieldsTitle = "title"
	fieldsKey   = "key"
)

// NewListCommand returns the `sk2 list` subcommand.
func NewListCommand(st *store.Store) *cobra.Command {
	var agentFilter, fields string
	var showNote bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List stored session keys (optionally filtered by agent)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch fields {
			case fieldsBoth, fieldsTitle, fieldsKey:
			default:
				return &flagError{name: "--fields/-f", value: fields, allowed: []string{fieldsBoth, fieldsTitle, fieldsKey}}
			}
			sessions, err := st.List(agentFilter)
			if err != nil {
				return err
			}
			renderTable(cmd.OutOrStdout(), sessions, fields, showNote)
			return nil
		},
	}
	cmd.Flags().StringVarP(&agentFilter, "agent", "a", "", "only list sessions for this agent")
	cmd.Flags().StringVarP(&fields, "fields", "f", fieldsBoth, "which columns to show: both|title|key")
	cmd.Flags().BoolVar(&showNote, "note", false, "also show the note column")
	return cmd
}
