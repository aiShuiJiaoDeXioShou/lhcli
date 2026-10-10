package cmd

import (
	"fmt"
	"os"

	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/tui"
	"github.com/spf13/cobra"
)

func interactive(cmd *cobra.Command, start string, required bool, accessible bool) error {
	if !tui.IsTerminal(cmd.InOrStdin(), cmd.OutOrStdout()) {
		if required {
			return fmt.Errorf("交互向导需要终端输入和输出；请在终端直接运行，或使用 --help 查看非交互命令")
		}
		return cmd.Help()
	}
	return tui.Run(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), start, accessible)
}

func newTUICmd() *cobra.Command {
	accessible := os.Getenv("ACCESSIBLE") == "true"
	command := &cobra.Command{
		Use: "tui", Short: "打开项目和技能的交互向导", Args: exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return interactive(cmd, "", true, accessible)
		},
	}
	command.Flags().BoolVar(&accessible, "accessible", accessible, "使用便于屏幕阅读器访问的逐项提示")
	return command
}

func init() {
	rootCmd.AddCommand(newTUICmd())
}
