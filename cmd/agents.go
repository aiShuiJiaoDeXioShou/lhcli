package cmd

import (
	"fmt"
	"os"

	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/skills"
	"github.com/spf13/cobra"
)

func newAgentsCmd() *cobra.Command {
	command := &cobra.Command{Use: "agents", Short: "管理 AGENTS.md 指令模板", Args: exactArgs(0), RunE: func(cmd *cobra.Command, args []string) error {
		return interactive(cmd, "agents", false, os.Getenv("ACCESSIBLE") == "true")
	}}
	for _, action := range []string{"list", "show", "apply", "capture"} {
		var project string
		var replace, dryRun bool
		child := &cobra.Command{Use: action + " <模板>", Args: exactArgs(1)}
		child.Short = map[string]string{"list": "列出指令模板", "show": "查看模板全文", "apply": "将模板复制为项目的 AGENTS.md", "capture": "将项目 AGENTS.md 保存为模板"}[action]
		if action == "list" {
			child.Use, child.Args = "list", exactArgs(0)
		}
		if action == "apply" || action == "capture" {
			child.Flags().StringVar(&project, "project", ".", "已有项目目录，默认当前目录")
			child.Flags().BoolVar(&replace, "replace", false, "先备份再替换已有文件")
			child.Flags().BoolVar(&dryRun, "dry-run", false, "只预览，不写入文件")
			child.Example = "  lhcli agents " + action + " default --project /path/to/project\n  lhcli agents " + action + " default --replace --dry-run"
		}
		child.RunE = func(cmd *cobra.Command, args []string) error {
			m, err := skills.New()
			if err != nil {
				return err
			}
			switch action {
			case "list":
				names, err := m.TemplateNames()
				if err != nil {
					return err
				}
				if len(names) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "暂无模板，运行 lhcli agents capture <模板名> 收录当前项目的 AGENTS.md。")
				}
				for _, name := range names {
					fmt.Fprintln(cmd.OutOrStdout(), name)
				}
				return nil
			case "show":
				return m.ShowTemplate(args[0], cmd.OutOrStdout())
			default:
				return m.TransferTemplate(args[0], project, action == "capture", replace, dryRun, cmd.OutOrStdout())
			}
		}
		command.AddCommand(child)
	}
	command.AddCommand(&cobra.Command{Use: "submit", Short: "提交并可选推送技能和指令模板变更", Args: exactArgs(0), RunE: func(cmd *cobra.Command, args []string) error {
		return interactive(cmd, "submit", true, os.Getenv("ACCESSIBLE") == "true")
	}})
	return command
}

func init() { rootCmd.AddCommand(newAgentsCmd()) }
