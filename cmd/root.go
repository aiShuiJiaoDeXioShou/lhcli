package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd 是 lhcli 的根命令。
var rootCmd = &cobra.Command{
	Use:   "lhcli",
	Short: "lhcli is a custom Go CLI tool",
	Long: `lhcli is a custom command-line tool written in Go.

It ships with a small set of example commands that demonstrate the
project layout, flag handling and build-time version injection.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute 执行根命令，出错时以非零状态码退出。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	// 取消注释可禁用自动生成的 shell 补全命令。
	// rootCmd.CompletionOptions.DisableDefaultCmd = true
}
