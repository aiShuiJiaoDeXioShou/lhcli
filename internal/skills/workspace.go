package skills

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Create 创建带初始提交的个人仓库并登记为默认仓库。
func (m *Manager) Create(ctx context.Context, path string, dryRun bool, out io.Writer) error {
	if path == "" {
		path = filepath.Join(m.Dir, "repos", "my-skills")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return err
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
	if s.Repo != "" {
		return fmt.Errorf("已配置仓库 %s；运行 lhcli skills info 查看", s.Repo)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		if err != nil {
			return err
		}
		return fmt.Errorf("目标已存在，请使用 --path 登记已有仓库: %s", path)
	}
	fmt.Fprintf(out, "创建个人仓库: %s\n包含 skills/ 和 agents/default/AGENTS.md，自动创建初始提交。\n", path)
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(path), ".create-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if _, err := git(ctx, "", "init", "-b", "main", stage); err != nil {
		return err
	}
	for name, content := range map[string]string{
		"skills/.gitkeep":          "",
		"agents/default/AGENTS.md": "# 协作约定\n\n- 使用中文沟通，代码标识符保持项目约定。\n- 修改前阅读项目说明，沿用现有结构与工具。\n- 保留已有工作；替换或删除文件前先确认影响。\n- 完成后运行与改动相关的检查，说明结果和未验证的部分。\n",
	} {
		file := filepath.Join(stage, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if _, err := git(ctx, stage, "add", "--", "skills", "agents"); err != nil {
		return err
	}
	if _, err := git(ctx, stage, "commit", "-m", "chore(config): 初始化个人技能和指令仓库"); err != nil {
		return fmt.Errorf("无法创建初始提交，请检查 Git 的 user.name、user.email 和签名配置后重试: %w", err)
	}
	if err := os.Rename(stage, path); err != nil {
		return err
	}
	s.Repo, err = repositoryRoot(ctx, path)
	if err != nil {
		return err
	}
	if err := m.save(s); err != nil {
		return fmt.Errorf("保存配置失败；仓库已保留在 %s，可使用 --path 重新登记: %w", path, err)
	}
	fmt.Fprintln(out, "默认仓库已就绪。运行 lhcli skills 导入技能，或 lhcli agents 管理指令模板。")
	return nil
}

// Info 显示默认仓库、配置位置及 Git 状态。
func (m *Manager) Info(ctx context.Context, out io.Writer) error {
	s, err := m.load()
	if err != nil {
		return err
	}
	if err := requireRepo(s); err != nil {
		return err
	}
	fmt.Fprintf(out, "默认仓库: %s\n配置文件: %s\n技能目录: %s\n指令模板: %s\n", s.Repo, filepath.Join(m.Dir, "skills.json"), filepath.Join(s.Repo, "skills"), filepath.Join(s.Repo, "agents"))
	status, err := git(ctx, s.Repo, "status", "--short", "--branch")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, status)
	remote, err := git(ctx, s.Repo, "remote", "-v")
	if err != nil {
		return err
	}
	if remote == "" {
		remote = "尚未关联远端，提交仅保存在本机。"
	}
	fmt.Fprintln(out, remote)
	return nil
}

// Import 将单个本地技能复制进仓库，不移动原文件或接管已有 agent 链接。
func (m *Manager) Import(source string, dryRun bool, out io.Writer) error {
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	name := filepath.Base(source)
	if !validName(name) {
		return fmt.Errorf("技能目录名无效，请使用小写字母、数字和连字符，最多 64 字符")
	}
	if err := regularTree(source); err != nil {
		return err
	}
	if _, err := readDocument(filepath.Join(source, "SKILL.md")); err != nil {
		return err
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
	if err := requireRepo(s); err != nil {
		return err
	}
	root := filepath.Join(s.Repo, "skills")
	if err := plainDirectory(root); err != nil {
		return err
	}
	if rel, err := filepath.Rel(source, root); err == nil && (rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))) {
		return fmt.Errorf("导入来源不能包含目标仓库目录")
	}
	dest := filepath.Join(root, name)
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		if err != nil {
			return err
		}
		return fmt.Errorf("仓库内已有同名技能，不覆盖: %s", dest)
	}
	fmt.Fprintf(out, "复制技能: %s → %s\n原目录保留；导入后可选择启用并提交。\n", source, dest)
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	stage, err := os.MkdirTemp(root, ".import-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := os.CopyFS(stage, os.DirFS(source)); err != nil {
		return err
	}
	if err := regularTree(stage); err != nil {
		return err
	}
	if _, err := readDocument(filepath.Join(stage, "SKILL.md")); err != nil {
		return err
	}
	return os.Rename(stage, dest)
}

func plainDirectory(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("目录不能是符号链接或普通文件: %s", path)
	}
	return nil
}

func regularTree(path string) error {
	return filepath.WalkDir(path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return fmt.Errorf("不能导入符号链接或特殊文件: %s", path)
		}
		if entry.Name() == ".git" {
			return fmt.Errorf("技能内部不能包含 Git 仓库: %s", path)
		}
		return nil
	})
}
