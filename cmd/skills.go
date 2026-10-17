package cmd

import (
	"fmt"
	"os"

	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/skills"
	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/tui"
	"github.com/spf13/cobra"
)

func newSkillsCmd() *cobra.Command {
	command := &cobra.Command{
		Use: "skills", Short: "管理个人技能仓库和多个 agent 的技能链接",
		Long: "管理一个个人 Git 技能仓库，按需将 skills/<名称> 链接到 Codex、Claude Code 或 Cursor。\n在终端无参数运行可打开向导。带参数的启停命令默认作用于当前目录；使用 --global 操作用户级技能。",
		Args: exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return interactive(cmd, "skills", false, os.Getenv("ACCESSIBLE") == "true")
		},
	}
	var source, path string
	var initDryRun, create bool
	initCommand := &cobra.Command{
		Use: "init", Short: "创建、克隆或登记默认仓库", Args: exactArgs(0),
		Example: "  lhcli skills init --create\n  lhcli skills init --repo owner/my-skills\n  lhcli skills init --path /path/to/my-skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().NFlag() == 0 {
				return interactive(cmd, "setup", true, os.Getenv("ACCESSIBLE") == "true")
			}
			m, err := skills.New()
			if err != nil {
				return err
			}
			if create {
				if source != "" {
					return fmt.Errorf("--create 不能与 --repo 同时使用")
				}
				return m.Create(cmd.Context(), path, initDryRun, cmd.OutOrStdout())
			}
			return m.Init(cmd.Context(), source, path, initDryRun, cmd.OutOrStdout())
		},
	}
	initCommand.Flags().StringVar(&source, "repo", "", "仓库 owner/repo、HTTPS 或 SSH 地址")
	initCommand.Flags().StringVar(&path, "path", "", "本地仓库目录；配合 --create 创建新目录")
	initCommand.Flags().BoolVar(&create, "create", false, "创建默认本地仓库，包含初始提交和 AGENTS.md 模板")
	initCommand.Flags().BoolVar(&initDryRun, "dry-run", false, "只预览，不克隆或写入配置")
	command.AddCommand(initCommand)
	command.AddCommand(&cobra.Command{Use: "info", Short: "查看默认仓库位置、远端和 Git 状态", Args: exactArgs(0), RunE: func(cmd *cobra.Command, args []string) error {
		m, err := skills.New()
		if err != nil {
			return err
		}
		return m.Info(cmd.Context(), cmd.OutOrStdout())
	}})
	var importDryRun bool
	importCommand := &cobra.Command{Use: "import [技能目录]", Short: "复制本地技能到默认仓库，保留原文件", Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("最多指定一个技能目录")
		}
		return nil
	}, RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			if cmd.Flags().NFlag() != 0 {
				return fmt.Errorf("使用参数时请提供技能目录")
			}
			return interactive(cmd, "import", true, os.Getenv("ACCESSIBLE") == "true")
		}
		m, err := skills.New()
		if err != nil {
			return err
		}
		return m.Import(args[0], importDryRun, cmd.OutOrStdout())
	}}
	importCommand.Flags().BoolVar(&importDryRun, "dry-run", false, "只预览，不复制技能")
	command.AddCommand(importCommand)
	command.AddCommand(&cobra.Command{
		Use: "add <技能目录>", Short: "添加本地技能，然后引导提交和推送", Args: exactArgs(1),
		Long:    "将包含 SKILL.md 的目录复制到默认仓库，随后引导提交并选择是否推送。\n原目录保留；取消提交时，已导入的文件也会保留，可用 lhcli skills submit 继续。",
		Example: "  lhcli skills add ~/code/my-skill",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 必须在复制前检查终端，避免非交互调用失败时已经写入文件。
			if !tui.IsTerminal(cmd.InOrStdin(), cmd.OutOrStdout()) {
				return fmt.Errorf("添加并提交需要终端，请直接运行 lhcli skills add <技能目录>；脚本中只导入可用 lhcli skills import <技能目录>")
			}
			m, err := skills.New()
			if err != nil {
				return err
			}
			if _, err := m.PlanSubmit(cmd.Context()); err != nil {
				return err
			}
			if err := m.Import(args[0], false, cmd.OutOrStdout()); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "技能已导入，接下来提交并选择是否推送。取消时文件保留，可运行 lhcli skills submit 继续。")
			return interactive(cmd, "submit", true, os.Getenv("ACCESSIBLE") == "true")
		},
	})
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
	command.AddCommand(&cobra.Command{
		Use: "submit", Short: "交互式提交技能和指令模板，可选择推送到远端", Args: exactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return interactive(cmd, "submit", true, os.Getenv("ACCESSIBLE") == "true")
		},
	})
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
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return nil
			}
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
		if action != "list" && len(args) == 0 && cmd.Flags().NFlag() == 0 {
			return interactive(cmd, action, true, os.Getenv("ACCESSIBLE") == "true")
		}
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
