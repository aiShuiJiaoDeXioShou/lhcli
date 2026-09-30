package selfupdate

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Options 描述一次自更新的全部参数。
type Options struct {
	CurrentVersion string       // 当前版本，取自 buildinfo.Version
	CheckOnly      bool         // 只检查是否有新版本
	TargetVersion  string       // 指定安装的版本，为空表示最新版
	Force          bool         // 已是最新或本地为 dev 时也强制覆盖
	AssumeYes      bool         // 跳过交互确认
	DryRun         bool         // 只打印将要执行的动作
	Mirror         string       // 下载镜像前缀
	Repo           string       // 发布仓库 owner/name，默认 DefaultRepo
	Stdout         io.Writer    // 标准输出，默认 os.Stdout
	Stderr         io.Writer    // 标准错误，默认 os.Stderr
	In             io.Reader    // 交互输入，默认 os.Stdin
	Client         *http.Client // HTTP 客户端，测试可注入
	GOOS           string       // 目标平台，默认当前平台
	GOARCH         string       // 目标架构，默认当前架构
	Executable     string       // 待替换的二进制路径，默认当前进程
}

// Run 执行一次自更新；返回是否真的替换了本地二进制。
func Run(ctx context.Context, opts Options) (bool, error) {
	out := opts.Stdout
	if out == nil {
		out = os.Stdout
	}
	errOut := opts.Stderr
	if errOut == nil {
		errOut = os.Stderr
	}
	in := opts.In
	if in == nil {
		in = os.Stdin
	}

	repo := opts.Repo
	if repo == "" {
		repo = DefaultRepo
	}
	goos := opts.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := opts.GOARCH
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: requestTimeout}
	}
	mirror := resolveMirror(opts.Mirror)

	fmt.Fprintf(out, "当前版本: %s\n", opts.CurrentVersion)

	release, err := fetchRelease(ctx, client, repo, opts.TargetVersion)
	if err != nil {
		return false, err
	}
	label := "最新版本"
	if opts.TargetVersion != "" {
		label = "目标版本"
	}
	fmt.Fprintf(out, "%s: %s", label, release.TagName)
	if published, err := time.Parse(time.RFC3339, release.PublishedAt); err == nil {
		fmt.Fprintf(out, "（发布于 %s）", published.Local().Format("2006-01-02"))
	}
	fmt.Fprintln(out)

	// 仅在两侧版本都可解析时才有可靠的比较结果；dev 等非法版本一律视为需要更新。
	noChange := false
	if IsValid(opts.CurrentVersion) && IsValid(release.TagName) {
		comparison, err := Compare(release.TagName, opts.CurrentVersion)
		if err != nil {
			return false, err
		}
		if opts.TargetVersion != "" {
			// 显式指定版本时只比较是否与目标一致。
			noChange = comparison == 0
		} else {
			// 默认跟随 latest：当前版本不低于 latest 即视为已是最新。
			noChange = comparison <= 0
		}
	}

	if opts.CheckOnly {
		if noChange {
			fmt.Fprintf(out, "已是最新版本 %s\n", opts.CurrentVersion)
		} else {
			fmt.Fprintf(out, "发现新版本 %s，可运行 lhcli update 更新\n", release.TagName)
		}
		return false, nil
	}
	if noChange && !opts.Force {
		fmt.Fprintf(out, "已是最新版本 %s\n", opts.CurrentVersion)
		return false, nil
	}

	// 本地为 dev 或非法版本号时，需要显式 --force 才覆盖。
	if !opts.Force && !IsValid(opts.CurrentVersion) {
		return false, fmt.Errorf("当前版本 %q 不是正式发布版本，如需覆盖请加 --force，或改用 go install / install.sh 安装",
			opts.CurrentVersion)
	}

	asset, err := findAsset(release, assetName(release.TagName, goos, goarch))
	if err != nil {
		return false, err
	}
	checksumAsset, err := findAsset(release, checksumFileName)
	if err != nil {
		return false, err
	}

	fmt.Fprintf(out, "将下载:   %s\n", asset.Name)
	if opts.DryRun {
		fmt.Fprintln(out, "干跑完成，未下载或替换任何文件")
		return false, nil
	}

	if !opts.AssumeYes {
		confirmed, err := confirm(in, out)
		if err != nil {
			return false, err
		}
		if !confirmed {
			fmt.Fprintln(out, "已取消")
			return false, nil
		}
	}

	tempDir, err := os.MkdirTemp("", "lhcli-update-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, asset.Name)
	fmt.Fprintln(out, "正在下载 ...")
	if err := downloadAsset(ctx, client, asset.URL, mirror, archivePath); err != nil {
		return false, err
	}

	sums, err := fetchChecksums(ctx, client, checksumAsset.URL, mirror)
	if err != nil {
		return false, err
	}
	expected, ok := sums[asset.Name]
	if !ok {
		return false, fmt.Errorf("校验和文件中缺少 %s 条目", asset.Name)
	}
	if err := verifyChecksum(archivePath, expected); err != nil {
		return false, err
	}
	fmt.Fprintln(out, "SHA256 校验通过")

	binaryPath := filepath.Join(tempDir, binaryName(goos))
	if err := extractBinary(archivePath, binaryPath, goos); err != nil {
		return false, err
	}

	target := opts.Executable
	if target == "" {
		if target, err = executablePath(); err != nil {
			return false, err
		}
	}
	if err := replaceExecutable(target, binaryPath); err != nil {
		return false, err
	}

	fmt.Fprintf(out, "已更新到 %s\n", release.TagName)
	if goos == "windows" {
		fmt.Fprintf(errOut, "提示: 旧版本已备份为 %s.old，将在下次启动时自动清理\n", target)
	}
	return true, nil
}

// resolveMirror 返回最终使用的下载镜像前缀：命令行参数优先，其次环境变量 LHCLI_MIRROR。
func resolveMirror(flagValue string) string {
	if value := strings.TrimSpace(flagValue); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv("LHCLI_MIRROR"))
}

// confirm 询问用户是否继续更新。
func confirm(in io.Reader, out io.Writer) (bool, error) {
	if !isTerminal(in) {
		return false, fmt.Errorf("当前不是交互式终端，请使用 --yes 确认更新")
	}
	fmt.Fprint(out, "是否更新？[y/N] ")

	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// isTerminal 判断 reader 是否连接到终端。
func isTerminal(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
