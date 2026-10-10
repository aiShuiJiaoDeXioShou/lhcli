package skills

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func git(ctx context.Context, dir string, args ...string) (string, error) {
	output, err := gitOutput(ctx, dir, args...)
	return strings.TrimSpace(output), err
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	args = append([]string{"--no-optional-locks", "-c", "core.hooksPath=" + os.DevNull}, args...)
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = dir
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := command.Output()
	if err != nil {
		var detail string
		if exit, ok := err.(*exec.ExitError); ok {
			detail = strings.TrimSpace(string(output) + "\n" + string(exit.Stderr))
		}
		return "", fmt.Errorf("Git 操作失败，请检查 Git 安装、仓库权限与认证: %w\n%s", err, detail)
	}
	return string(output), nil
}

func repositoryRoot(ctx context.Context, path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	root, err := git(ctx, abs, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	root, err = filepath.EvalSymlinks(filepath.FromSlash(root))
	if err != nil {
		return "", err
	}
	if root != abs {
		return "", fmt.Errorf("请指定 Git 仓库根目录: %s", root)
	}
	return root, nil
}

func sourceURL(source string) (string, error) {
	if source == "" || strings.HasPrefix(source, "-") || strings.ContainsAny(source, "\r\n") || strings.Contains(source, "::") {
		return "", fmt.Errorf("仓库地址无效")
	}
	if filepath.IsAbs(source) || strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "ssh://") || strings.HasPrefix(source, "git@") || strings.HasPrefix(source, "file://") {
		return source, nil
	}
	parts := strings.Split(source, "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" && !strings.ContainsAny(source, " :\\@") && parts[0] != "." && parts[0] != ".." && parts[1] != "." && parts[1] != ".." {
		return "https://github.com/" + strings.TrimSuffix(source, ".git") + ".git", nil
	}
	return "", fmt.Errorf("仓库地址应为 owner/repo、HTTPS/SSH 地址或本地绝对路径")
}

// Init 克隆或登记一个技能仓库，不修改已有的 agent 目录。
func (m *Manager) Init(ctx context.Context, source, path string, dryRun bool, out io.Writer) error {
	if (source == "") == (path == "") {
		return fmt.Errorf("必须且只能指定 --repo 或 --path")
	}
	if !dryRun {
		unlock, err := m.lock()
		if err != nil {
			return err
		}
		defer unlock()
	}
	s, err := m.load()
	if err != nil {
		return err
	}
	if path != "" {
		path, err = repositoryRoot(ctx, path)
		if err != nil {
			return err
		}
		if _, err := discover(path); err != nil {
			return err
		}
	} else {
		source, err = sourceURL(source)
		if err != nil {
			return err
		}
		path = filepath.Join(m.Dir, "repos", "my-skills")
	}
	if s.Repo != "" {
		if source == "" && s.Repo == path {
			fmt.Fprintf(out, "已登记技能仓库: %s\n", path)
			return nil
		}
		return fmt.Errorf("已配置技能仓库 %s；为保护已有链接，不自动切换仓库", s.Repo)
	}
	if source != "" {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("克隆目标已存在: %s；请使用 --path 登记已有仓库", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if dryRun {
		fmt.Fprintf(out, "预览：将登记技能仓库 %s，未写入配置或克隆仓库\n", path)
		return nil
	}
	if source != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		stage, err := os.MkdirTemp(filepath.Dir(path), ".clone-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(stage)
		fmt.Fprintln(out, "正在克隆技能仓库 ...")
		if _, err = git(ctx, "", "clone", "--", source, stage); err != nil {
			return err
		}
		if _, err := discover(stage); err != nil {
			return err
		}
		if err := os.Rename(stage, path); err != nil {
			return err
		}
		path, err = repositoryRoot(ctx, path)
		if err != nil {
			return err
		}
	}
	s.Repo = path
	if err := m.save(s); err != nil {
		return fmt.Errorf("登记失败；仓库保留在 %s，可用 --path 重试: %w", path, err)
	}
	fmt.Fprintf(out, "已登记技能仓库: %s\n将技能放在 skills/<名称>/SKILL.md，然后运行 lhcli skills list\n", path)
	return nil
}

// Update 获取上游并仅快进更新；检查模式只更新 Git 远端引用。
func (m *Manager) Update(ctx context.Context, check, dryRun bool, out io.Writer) error {
	if !dryRun {
		unlock, err := m.lock()
		if err != nil {
			return err
		}
		defer unlock()
	}
	s, err := m.load()
	if err != nil {
		return err
	}
	if err := requireRepo(s); err != nil {
		return err
	}
	if _, err := repositoryRoot(ctx, s.Repo); err != nil {
		return err
	}
	status, err := git(ctx, s.Repo, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("技能仓库存在本地修改或未跟踪文件，请先提交或自行处理后再更新: %s", s.Repo)
	}
	branch, err := git(ctx, s.Repo, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return fmt.Errorf("当前仓库未处于分支上，请先切换到跟踪远端的分支")
	}
	remote, err := git(ctx, s.Repo, "config", "--get", "branch."+branch+".remote")
	if err != nil || remote == "" || strings.HasPrefix(remote, "-") {
		return fmt.Errorf("当前分支没有有效的上游远端，请先用 Git 配置 upstream")
	}
	if dryRun {
		fmt.Fprintf(out, "预览：将从 %s 获取更新并检查分支 %s；未联网或修改仓库\n", remote, branch)
		return nil
	}
	if _, err := git(ctx, s.Repo, "fetch", "--", remote); err != nil {
		return err
	}
	head, err := git(ctx, s.Repo, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	upstream, err := git(ctx, s.Repo, "rev-parse", "@{upstream}")
	if err != nil {
		return fmt.Errorf("无法解析当前分支的上游，请检查 upstream 配置: %w", err)
	}
	if head == upstream {
		fmt.Fprintln(out, "技能仓库已是最新")
		return nil
	}
	if _, err := git(ctx, s.Repo, "merge-base", "--is-ancestor", head, upstream); err != nil {
		if _, err := git(ctx, s.Repo, "merge-base", "--is-ancestor", upstream, head); err == nil {
			fmt.Fprintln(out, "本地分支领先上游，未修改仓库；可自行推送本地提交")
			return nil
		}
		return fmt.Errorf("本地与上游历史已分叉，拒绝自动合并；请用 Git 处理")
	}
	if err := validateRevision(ctx, s, upstream); err != nil {
		return err
	}
	if check {
		fmt.Fprintf(out, "发现技能更新: %.12s → %.12s；运行 lhcli skills update 应用\n", head, upstream)
		return nil
	}
	if _, err := git(ctx, s.Repo, "merge", "--ff-only", "--no-edit", "--no-overwrite-ignore", upstream); err != nil {
		return err
	}
	fmt.Fprintf(out, "技能仓库已更新: %.12s → %.12s；已有链接继续指向同一份源码\n", head, upstream)
	return nil
}

func validateRevision(ctx context.Context, s state, revision string) error {
	output, err := git(ctx, s.Repo, "ls-tree", "-r", "-z", revision, "--", "skills")
	if err != nil {
		return err
	}
	names := map[string]bool{}
	for _, entry := range strings.Split(output, "\x00") {
		if entry == "" {
			continue
		}
		metadata, path, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(metadata)
		parts := strings.Split(path, "/")
		if !ok || len(fields) != 3 || (fields[0] != "100644" && fields[0] != "100755") {
			return fmt.Errorf("上游技能包含符号链接或特殊条目，拒绝更新: %s", path)
		}
		if len(parts) == 3 && parts[2] == "SKILL.md" {
			body, err := git(ctx, s.Repo, "cat-file", "blob", fields[2])
			if err != nil {
				return err
			}
			if !validName(parts[1]) || body == "" {
				return fmt.Errorf("上游技能名称或 SKILL.md 无效: %s", path)
			}
			names[parts[1]] = true
		}
	}
	for _, b := range s.Links {
		if !names[b.Name] {
			return fmt.Errorf("上游移除了已启用的技能 %s；请先停用该技能的所有链接再更新", b.Name)
		}
	}
	return nil
}
