package cmd

import (
	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/scaffold"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "从 Git 模板初始化新项目",
	Long: `从 Git 模板仓库初始化新的后端或移动端项目。

模板下载后会自动执行模板内的 scripts/rename.sh 完成改名，
随后初始化 Git 仓库并安装依赖。`,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.AddCommand(newInitGoCmd())
	initCmd.AddCommand(newInitMobileCmd())
}

// newInitGoCmd 构造后端项目初始化命令。
func newInitGoCmd() *cobra.Command {
	opts := scaffold.Options{Kind: scaffold.KindGo}
	command := &cobra.Command{
		Use:   "go <name>",
		Short: "从 linghe-go-template 初始化 Go 后端项目",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			opts.Stdout = cmd.OutOrStdout()
			opts.Stderr = cmd.ErrOrStderr()
			return scaffold.Run(cmd.Context(), opts)
		},
	}
	flags := command.Flags()
	flags.StringVar(&opts.Module, "module", "", "Go module 路径，默认 github.com/example/<name>")
	flags.StringVarP(&opts.Dir, "dir", "d", "", "目标目录，默认 ./<name>")
	flags.StringVar(&opts.TemplateRepo, "template-repo", "", "模板仓库 owner/name")
	flags.StringVar(&opts.TemplateRef, "template-ref", "", "模板 ref（分支或 tag），默认 main")
	flags.StringVar(&opts.Mirror, "mirror", "", "下载镜像前缀，例如 https://gh-proxy.com/")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "只预览改名结果，不写入磁盘")
	flags.BoolVar(&opts.Force, "force", false, "目标目录非空时先删除")
	flags.BoolVar(&opts.NoGit, "no-git", false, "跳过 Git 仓库初始化")
	flags.BoolVar(&opts.NoInstall, "no-install", false, "跳过依赖安装")
	return command
}

// newInitMobileCmd 构造移动端项目初始化命令。
func newInitMobileCmd() *cobra.Command {
	opts := scaffold.Options{Kind: scaffold.KindMobile}
	command := &cobra.Command{
		Use:   "mobile <name>",
		Short: "从 linghe_mobile_template 初始化 Flutter 项目",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			opts.Stdout = cmd.OutOrStdout()
			opts.Stderr = cmd.ErrOrStderr()
			return scaffold.Run(cmd.Context(), opts)
		},
	}
	flags := command.Flags()
	flags.StringVar(&opts.Org, "org", "", "反向域名前缀，默认 com.example")
	flags.StringVar(&opts.DisplayName, "display-name", "", "应用显示名，默认由项目名推导")
	flags.StringVarP(&opts.Dir, "dir", "d", "", "目标目录，默认 ./<name>")
	flags.StringVar(&opts.TemplateRepo, "template-repo", "", "模板仓库 owner/name")
	flags.StringVar(&opts.TemplateRef, "template-ref", "", "模板 ref（分支或 tag），默认 main")
	flags.StringVar(&opts.Mirror, "mirror", "", "下载镜像前缀，例如 https://gh-proxy.com/")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "只预览改名结果，不写入磁盘")
	flags.BoolVar(&opts.Force, "force", false, "目标目录非空时先删除")
	flags.BoolVar(&opts.NoGit, "no-git", false, "跳过 Git 仓库初始化")
	flags.BoolVar(&opts.NoInstall, "no-install", false, "跳过依赖安装")
	return command
}
