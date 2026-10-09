package cmd

import (
	"fmt"
	"os"

	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/skills"
	"github.com/spf13/cobra"
)

func newSkillsCmd() *cobra.Command {
	command := &cobra.Command{
		Use: "skills", Short: "管理个人技能仓库和多个 agent 的技能链接",
		Long: "管理一个个人 Git 技能仓库，按需将 skills/<名称> 链接到 Codex、Claude Code 或 Cursor。\n启用与停用默认作用于当前目录；使用 --global 操作用户级技能。",
	}
	var source, path string
	var initDryRun bool
	initCommand := &cobra.Command{
		Use: "init", Short: "克隆或登记技能仓库", Args: exactArgs(0),
		Example: "  lhcli skills init --repo owner/my-skills\n  lhcli skills init --path /path/to/my-skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := skills.New()
			if err != nil {
				return err
			}
			return m.Init(cmd.Context(), source, path, initDryRun, cmd.OutOrStdout())
		},
	}
	initCommand.Flags().StringVar(&source, "repo", "", "仓库 owner/repo、HTTPS 或 SSH 地址")
	initCommand.Flags().StringVar(&path, "path", "", "已有的本地 Git 仓库根目录")
	initCommand.Flags().BoolVar(&initDryRun, "dry-run", false, "只预览，不克隆或写入配置")
	command.AddCommand(initCommand)
	for _, action := range []string{"list", "enable", "disable"} {
		command.AddCommand(newSkillsLinksCmd(action))
	}
	var check, updateDryRun bool
	updateCommand := &cobra.Command{
		Use: "update", Short: "从上游快进更新技能仓库", Args: exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := skills.New()
			if err != nil {
				return err
			}
			return m.Update(cmd.Context(), check, updateDryRun, cmd.OutOrStdout())
		},
	}
	updateCommand.Flags().BoolVar(&check, "check", false, "获取远端引用并检查更新，不修改技能文件")
	updateCommand.Flags().BoolVar(&updateDryRun, "dry-run", false, "只预览，不联网或修改仓库")
	command.AddCommand(updateCommand)
	return command
}

func newSkillsLinksCmd(action string) *cobra.Command {
	var selection skills.Selection
	var global bool
	description := map[string]string{"list": "查看技能及链接状态", "enable": "为指定 agent 启用技能", "disable": "停用受管链接，保留技能源码"}
	command := &cobra.Command{Use: action, Short: description[action]}
	if action == "list" {
		command.Args = exactArgs(0)
	} else {
		command.Use += " [名称]"
		command.Args = func(cmd *cobra.Command, args []string) error {
			if (selection.All && len(args) == 0) || (!selection.All && len(args) == 1) {
				return nil
			}
			return fmt.Errorf("请指定一个技能名称或 --all，二者不能同时使用")
		}
		command.Flags().BoolVar(&selection.All, "all", false, "处理当前作用域内所有技能")
		command.Flags().BoolVar(&selection.DryRun, "dry-run", false, "只预览，不修改链接或配置")
		command.Example = "  lhcli skills " + action + " go-api --agent codex,claude --global\n  lhcli skills " + action + " --all --agent codex --dry-run"
	}
	command.Flags().StringSliceVar(&selection.Agents, "agent", nil, "目标 agent：codex、claude、cursor，可用逗号分隔；list 默认显示全部")
	command.Flags().BoolVar(&global, "global", false, "操作用户目录，默认操作当前目录")
	command.RunE = func(cmd *cobra.Command, args []string) error {
		m, err := skills.New()
		if err != nil {
			return err
		}
		selection.Base = m.Home
		if !global {
			selection.Base, err = os.Getwd()
			if err != nil {
				return err
			}
		}
		selection.Name = ""
		if len(args) == 1 {
			selection.Name = args[0]
		}
		switch action {
		case "enable":
			return m.Enable(selection, cmd.OutOrStdout())
		case "disable":
			return m.Disable(selection, cmd.OutOrStdout())
		default:
			return m.List(selection, cmd.OutOrStdout())
		}
	}
	_ = command.RegisterFlagCompletionFunc("agent", cobra.FixedCompletions([]string{"codex", "claude", "cursor"}, cobra.ShellCompDirectiveNoFileComp))
	return command
}

func init() {
	rootCmd.AddCommand(newSkillsCmd())
}
