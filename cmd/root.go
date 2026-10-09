package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo"
	"github.com/spf13/cobra"
)

// rootCmd 是 lhcli 的根命令。
var rootCmd = &cobra.Command{
	Use:           "lhcli",
	Short:         "个人开发工具：初始化项目、管理技能与更新自身",
	Long:          "lhcli 提供 Go / Flutter 项目初始化、个人 skills 多 agent 管理和工具自更新。",
	Version:       buildinfo.Version,
	Example:       "  lhcli init go order-api\n  lhcli skills list --global\n  lhcli update --check",
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute 执行根命令，出错时以非零状态码退出。
func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	rootCmd.InitDefaultHelpCmd()
	localizeHelp(rootCmd)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetVersionTemplate("lhcli {{.Version}}\n")
	rootCmd.SetUsageTemplate(`用法：
  {{.UseLine}}{{if .HasAvailableFlags}} [参数]{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} <命令>{{end}}{{if .HasExample}}

示例：
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

可用命令：{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

参数：
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

通用参数：
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`)
	rootCmd.SetHelpCommand(&cobra.Command{
		Use: "help [命令]", Short: "查看命令帮助",
		RunE: func(cmd *cobra.Command, args []string) error {
			target, remaining, err := cmd.Root().Find(args)
			if err != nil || len(remaining) != 0 {
				return fmt.Errorf("未找到指定命令，请运行 lhcli --help 查看可用命令")
			}
			return target.Help()
		},
	})
	rootCmd.AddCommand(newCompletionCmd())
}

func exactArgs(count int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != count {
			return fmt.Errorf("命令 %s 需要 %d 个位置参数，实际收到 %d 个；使用 --help 查看用法", cmd.CommandPath(), count, len(args))
		}
		return nil
	}
}

func localizeHelp(cmd *cobra.Command) {
	cmd.DisableFlagsInUseLine = true
	cmd.InitDefaultHelpFlag()
	cmd.Flags().Lookup("help").Usage = "查看此命令的帮助"
	cmd.InitDefaultVersionFlag()
	if flag := cmd.Flags().Lookup("version"); flag != nil && cmd == rootCmd {
		flag.Usage = "查看 lhcli 版本"
	}
	for _, child := range cmd.Commands() {
		localizeHelp(child)
	}
}

func newCompletionCmd() *cobra.Command {
	var noDescriptions bool
	command := &cobra.Command{
		Use: "completion <bash|zsh|fish|powershell>", Short: "生成 shell 命令补全脚本", Args: exactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(cmd.OutOrStdout(), !noDescriptions)
			case "zsh":
				if noDescriptions {
					return cmd.Root().GenZshCompletionNoDesc(cmd.OutOrStdout())
				}
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), !noDescriptions)
			case "powershell":
				if noDescriptions {
					return cmd.Root().GenPowerShellCompletion(cmd.OutOrStdout())
				}
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return fmt.Errorf("不支持的 shell %q；可选 bash、zsh、fish、powershell", args[0])
			}
		},
	}
	command.Flags().BoolVar(&noDescriptions, "no-descriptions", false, "补全结果不包含说明文字")
	return command
}
