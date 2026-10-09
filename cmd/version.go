package cmd

import (
	"fmt"

	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "打印版本信息",
	Args:  exactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "lhcli %s（提交 %s，构建时间 %s）\n",
			buildinfo.Version, buildinfo.Commit, buildinfo.Date)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
