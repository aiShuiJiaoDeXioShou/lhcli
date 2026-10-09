package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var greetName string

var greetCmd = &cobra.Command{
	Use:    "greet",
	Short:  "打印问候语（示例命令）",
	Hidden: true,
	Args:   exactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "你好，%s！\n", greetName)
		return nil
	},
}

func init() {
	greetCmd.Flags().StringVarP(&greetName, "name", "n", "世界", "问候对象的姓名")
	rootCmd.AddCommand(greetCmd)
}
