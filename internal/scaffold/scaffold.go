package scaffold

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// 模板仓库与固定约定的默认值。
const (
	// GoTemplateRepo 是 Go 后端模板仓库。
	GoTemplateRepo = "aiShuiJiaoDeXioShou/linghe-go-template"
	// MobileTemplateRepo 是 Flutter 移动端模板仓库。
	MobileTemplateRepo = "aiShuiJiaoDeXioShou/linghe_mobile_template"
	// DefaultRef 是模板的默认分支。
	DefaultRef = "main"
	// RenameScript 是模板内固定的改名脚本路径。
	RenameScript = "scripts/rename.sh"
)

// Options 描述一次初始化的全部参数。
type Options struct {
	Kind         Kind   // 模板类型
	Name         string // 项目名（原始输入）
	Dir          string // 目标目录，默认 ./<name>
	Module       string // Go module 路径，仅 go
	Org          string // 反向域名，仅 mobile
	DisplayName  string // 应用显示名，仅 mobile
	TemplateRepo string // 模板仓库 owner/name
	TemplateRef  string // 模板 ref，默认 main
	Mirror       string // 下载镜像前缀
	DryRun       bool   // 只预览不写入
	Force        bool   // 目标目录非空时覆盖
	NoGit        bool   // 跳过 git 初始化
	NoInstall    bool   // 跳过依赖安装
	Stdout       io.Writer
	Stderr       io.Writer
}

// Run 执行一次完整的项目初始化。
func Run(ctx context.Context, opts Options) error {
	out := opts.Stdout
	if out == nil {
		out = os.Stdout
	}
	errOut := opts.Stderr
	if errOut == nil {
		errOut = os.Stderr
	}

	repo := opts.TemplateRepo
	if repo == "" {
		repo = defaultRepo(opts.Kind)
	}
	ref := opts.TemplateRef
	if ref == "" {
		ref = DefaultRef
	}

	name, err := NormalizeName(opts.Kind, opts.Name)
	if err != nil {
		return err
	}

	dir := opts.Dir
	if dir == "" {
		dir = name
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "模板:   %s@%s\n", repo, ref)
	fmt.Fprintf(out, "项目名: %s\n", name)
	fmt.Fprintf(out, "目标:   %s\n", absDir)

	if !opts.DryRun {
		if err := prepareTarget(absDir, opts.Force, errOut); err != nil {
			return err
		}
	}

	tempDir, err := os.MkdirTemp("", "lhcli-init-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, "template.tar.gz")
	fmt.Fprintf(out, "正在下载模板 ...\n")
	if err := downloadTarball(ctx, repo, ref, opts.Mirror, archivePath); err != nil {
		return err
	}

	stage := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return err
	}
	if err := extractTarGz(archivePath, stage); err != nil {
		return err
	}

	shell, err := findShell()
	if err != nil {
		return err
	}
	if err := runRename(ctx, shell, stage, name, opts, out, errOut); err != nil {
		return err
	}

	if opts.DryRun {
		fmt.Fprintln(out, "干跑完成，未写入目标目录")
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(absDir), 0o755); err != nil {
		return err
	}
	if err := os.Rename(stage, absDir); err != nil {
		if copyErr := copyPath(stage, absDir); copyErr != nil {
			return fmt.Errorf("写入目标目录失败: %w", copyErr)
		}
	}

	// 先安装依赖，再提交，确保首次提交包含 go.sum / pubspec.lock 等变化。
	if !opts.NoInstall {
		installDeps(ctx, opts.Kind, absDir, out, errOut)
	}
	if !opts.NoGit {
		if err := initGit(ctx, absDir, out, errOut); err != nil {
			return err
		}
	}

	printNextSteps(out, opts.Kind, absDir)
	return nil
}

// defaultRepo 返回模板类型的默认仓库。
func defaultRepo(kind Kind) string {
	if kind == KindMobile {
		return MobileTemplateRepo
	}
	return GoTemplateRepo
}

// prepareTarget 校验并清理目标目录。
func prepareTarget(absDir string, force bool, errOut io.Writer) error {
	info, err := os.Stat(absDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("目标路径已存在且不是目录: %s", absDir)
	}

	entries, err := os.ReadDir(absDir)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		if !force {
			return fmt.Errorf("目标目录非空: %s\n如需覆盖请加 --force", absDir)
		}
		fmt.Fprintf(errOut, "警告: 将覆盖已存在的目录 %s\n", absDir)
	}
	return os.RemoveAll(absDir)
}

// runRename 调用模板内的改名脚本。
func runRename(ctx context.Context, shell, dir, name string, opts Options, out, errOut io.Writer) error {
	args := []string{RenameScript, "--name", name}
	switch opts.Kind {
	case KindGo:
		module := opts.Module
		if module == "" {
			module = "github.com/example/" + name
		}
		args = append(args, "--module", module)
	case KindMobile:
		org := opts.Org
		if org == "" {
			org = "com.example"
		}
		args = append(args, "--org", org)
		if opts.DisplayName != "" {
			args = append(args, "--display-name", opts.DisplayName)
		}
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}

	command := exec.CommandContext(ctx, shell, args...)
	command.Dir = dir
	command.Stdout = out
	command.Stderr = errOut
	fmt.Fprintf(out, "执行改名: sh %s\n", strings.Join(args, " "))
	if err := command.Run(); err != nil {
		return fmt.Errorf("改名脚本执行失败: %w", err)
	}
	return nil
}

// initGit 初始化 Git 仓库并尝试首次提交。
func initGit(ctx context.Context, dir string, out, errOut io.Writer) error {
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(errOut, "警告: 未找到 git，已跳过仓库初始化")
		return nil
	}

	if err := runCommand(ctx, dir, out, errOut, "git", "init", "-b", "main"); err != nil {
		return err
	}
	if err := runCommand(ctx, dir, out, errOut, "git", "add", "-A"); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitmessage")); err == nil {
		_ = runCommand(ctx, dir, out, errOut, "git", "config", "commit.template", ".gitmessage")
	}

	if gitConfigValue(ctx, dir, "user.name") == "" || gitConfigValue(ctx, dir, "user.email") == "" {
		fmt.Fprintln(errOut, "警告: 未配置 git user.name/user.email，已跳过首次提交")
		return nil
	}
	return runCommand(ctx, dir, out, errOut, "git", "commit", "-m", "chore: 由 lhcli init 初始化项目")
}

// installDeps 安装项目依赖，失败时仅告警不中断。
func installDeps(ctx context.Context, kind Kind, dir string, out, errOut io.Writer) {
	switch kind {
	case KindGo:
		if _, err := exec.LookPath("go"); err != nil {
			fmt.Fprintln(errOut, "警告: 未找到 go，已跳过 go mod tidy")
			return
		}
		if err := runCommand(ctx, dir, out, errOut, "go", "mod", "tidy"); err != nil {
			fmt.Fprintf(errOut, "警告: go mod tidy 失败: %v\n", err)
		}
	case KindMobile:
		if _, err := exec.LookPath("flutter"); err != nil {
			fmt.Fprintln(errOut, "警告: 未找到 flutter，已跳过 flutter pub get")
			return
		}
		if err := runCommand(ctx, dir, out, errOut, "flutter", "pub", "get"); err != nil {
			fmt.Fprintf(errOut, "警告: flutter pub get 失败: %v\n", err)
		}
	}
}

// runCommand 执行外部命令并回显。
func runCommand(ctx context.Context, dir string, out, errOut io.Writer, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	command.Stdout = out
	command.Stderr = errOut
	fmt.Fprintf(out, "> %s %s\n", name, strings.Join(args, " "))
	return command.Run()
}

// gitConfigValue 读取指定 Git 配置项，未配置时返回空串。
func gitConfigValue(ctx context.Context, dir, key string) string {
	command := exec.CommandContext(ctx, "git", "config", "--get", key)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// printNextSteps 打印后续操作提示。
func printNextSteps(out io.Writer, kind Kind, dir string) {
	fmt.Fprintf(out, "\n项目已创建: %s\n", dir)
	fmt.Fprintln(out, "后续步骤:")
	fmt.Fprintf(out, "  cd %s\n", dir)
	switch kind {
	case KindGo:
		fmt.Fprintln(out, "  检查 configs/config.*.yaml 中的数据库与 Redis 凭据")
		fmt.Fprintln(out, "  go run ./tools/dev check")
	case KindMobile:
		fmt.Fprintln(out, "  flutter pub get")
		fmt.Fprintln(out, "  flutter run")
	}
}
