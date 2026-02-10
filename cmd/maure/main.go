package main

import (
	"fmt"
	"os"

	maurecobra "maure/cmd/maure/cobra"
)

func main() {
	rawArgs := os.Args[1:]
	rootCmd := maurecobra.NewRootCommand(maurecobra.Config{
		RawArgs: rawArgs,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	})

	normalizedArgs := maurecobra.NormalizeLegacyArgs(rawArgs, os.Stderr)
	rootCmd.SetArgs(normalizedArgs)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
