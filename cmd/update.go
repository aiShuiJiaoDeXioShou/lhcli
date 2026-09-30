package cmd

import (
	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/buildinfo"
	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/selfupdate"
	"github.com/spf13/cobra"
)

// updateOpts 保存 update 命令的 flag 取值。
var updateOpts struct {
	check   bool
	version string
	force   bool
	yes     bool
	dryRun  bool
	mirror  string
	repo    string
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "更新 lhcli 到最新版本",
	Long: `从 GitHub Releases 下载并替换当前 lhcli 二进制。

更新前会下载 checksums.txt 校验 SHA256；Windows 下旧版本会备份为
二进制旁的 .old 文件，并在下次启动时自动清理。`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := selfupdate.Run(cmd.Context(), selfupdate.Options{
			CurrentVersion: buildinfo.Version,
			CheckOnly:      updateOpts.check,
			TargetVersion:  updateOpts.version,
			Force:          updateOpts.force,
			AssumeYes:      updateOpts.yes,
			DryRun:         updateOpts.dryRun,
			Mirror:         updateOpts.mirror,
			Repo:           updateOpts.repo,
			Stdout:         cmd.OutOrStdout(),
			Stderr:         cmd.ErrOrStderr(),
			In:             cmd.InOrStdin(),
		})
		return err
	},
}

func init() {
	flags := updateCmd.Flags()
	flags.BoolVar(&updateOpts.check, "check", false, "只检查是否有新版本，不下载也不替换")
	flags.StringVar(&updateOpts.version, "version", "", "安装指定版本（如 v0.3.0），默认为最新版")
	flags.BoolVar(&updateOpts.force, "force", false, "已是最新或本地为 dev 时也强制重装")
	flags.BoolVarP(&updateOpts.yes, "yes", "y", false, "跳过交互确认")
	flags.BoolVar(&updateOpts.dryRun, "dry-run", false, "只打印将要执行的动作，不写入磁盘")
	flags.StringVar(&updateOpts.mirror, "mirror", "", "下载镜像前缀（默认读取环境变量 LHCLI_MIRROR），例如 https://gh-proxy.com/")
	flags.StringVar(&updateOpts.repo, "repo", "", "发布仓库 owner/name，默认 "+selfupdate.DefaultRepo)
	rootCmd.AddCommand(updateCmd)
}
