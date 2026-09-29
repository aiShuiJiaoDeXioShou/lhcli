package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var greetName string

var greetCmd = &cobra.Command{
	Use:   "greet",
	Short: "Print a greeting (example command)",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "Hello, %s!\n", greetName)
		return nil
	},
}

func init() {
	greetCmd.Flags().StringVarP(&greetName, "name", "n", "world", "name to greet")
	rootCmd.AddCommand(greetCmd)
}
