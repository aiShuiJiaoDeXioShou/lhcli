package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd is the base command for lhcli.
var rootCmd = &cobra.Command{
	Use:   "lhcli",
	Short: "lhcli is a custom Go CLI tool",
	Long: `lhcli is a custom command-line tool written in Go.

It ships with a small set of example commands that demonstrate the
project layout, flag handling and build-time version injection.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and exits with a non-zero status on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	// Uncomment to disable the generated shell completion command.
	// rootCmd.CompletionOptions.DisableDefaultCmd = true
}
