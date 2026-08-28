// Package cli wires the sk2 cobra command tree against the store.
package cli

import (
	"github.com/spf13/cobra"

	"sk2/internal/store"
)

// NewRootCommand builds the sk2 root command with its subcommands, all wired
// to the provided store. The store is injected so tests can pass an in-memory
// database.
func NewRootCommand(st *store.Store) *cobra.Command {
	root := &cobra.Command{
		Use:   "sk2",
		Short: "Session Key Keeper — store, list and resume terminal-agent sessions",
		// Avoid dumping usage text after a runtime error (e.g. not-found).
		SilenceUsage: true,
		// main() is responsible for printing the error; don't double-print it.
		SilenceErrors: true,
	}
	root.AddCommand(
		NewAddCommand(st),
		NewListCommand(st),
		NewGetCommand(st),
		NewRemoveCommand(st),
		NewUseCommand(st),
	)
	return root
}
