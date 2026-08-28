package main

import (
	"fmt"
	"os"

	"sk2/internal/cli"
	"sk2/internal/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sk2:", err)
		os.Exit(1)
	}
}

func run() error {
	path, err := store.ResolveDBPath()
	if err != nil {
		return err
	}
	st, err := store.Open(path)
	if err != nil {
		return err
	}
	defer st.Close()

	root := cli.NewRootCommand(st)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	return root.Execute()
}
