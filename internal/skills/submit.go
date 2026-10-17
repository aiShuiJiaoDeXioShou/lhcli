package skills

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// SubmitPlan 保存提交前展示的快照，执行时重新检查，避免提交确认之后的新改动。
type SubmitPlan struct {
	Repo      string
	Branch    string
	Files     []string
	Summary   string
	Remote    string
	RemoteRef string
	digest    string
}

// PlanSubmit 只读检查 skills/ 和 agents/ 下的变更，不触碰已有暂存内容。
func (m *Manager) PlanSubmit(ctx context.Context) (SubmitPlan, error) {
	var plan SubmitPlan
	s, err := m.load()
	if err != nil {
		return plan, err
	}
	if err := requireRepo(s); err != nil {
		return plan, err
	}
	plan.Repo, err = repositoryRoot(ctx, s.Repo)
	if err != nil {
		return plan, err
	}
	plan.Branch, err = git(ctx, s.Repo, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return plan, fmt.Errorf("提交技能前请先切换到一个本地分支")
	}
	for _, marker := range []string{"MERGE_HEAD", "CHERRY_PICK_HEAD", "REVERT_HEAD", "rebase-merge", "rebase-apply"} {
		path, err := git(ctx, s.Repo, "rev-parse", "--git-path", marker)
		if err != nil {
			return plan, err
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(s.Repo, path)
		}
		if _, err := os.Stat(path); err == nil {
			return plan, fmt.Errorf("仓库正在合并、变基或拣选提交，请先用 Git 完成该操作")
		} else if !os.IsNotExist(err) {
			return plan, err
		}
	}
	staged, err := git(ctx, s.Repo, "diff", "--cached", "--name-only")
	if err != nil {
		return plan, err
	}
	if staged != "" {
		return plan, fmt.Errorf("仓库已有暂存内容，请先用 Git 提交或取消暂存；向导不会接管已有暂存区")
	}
	if _, err := discover(s.Repo); err != nil {
		return plan, err
	}
	if err := validateManagedTrees(s.Repo); err != nil {
		return plan, err
	}
	status, err := gitOutput(ctx, s.Repo, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--no-renames", "--", "skills/", "agents/")
	if err != nil {
		return plan, err
	}
	diff, err := gitOutput(ctx, s.Repo, "diff", "--binary", "--no-ext-diff", "--no-textconv", "--", "skills/", "agents/")
	if err != nil {
		return plan, err
	}
	head, err := git(ctx, s.Repo, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return plan, fmt.Errorf("技能仓库需要至少一个初始提交，请先用 Git 完成首次提交")
	}
	plan.Remote, _ = git(ctx, s.Repo, "config", "--get", "branch."+plan.Branch+".remote")
	plan.RemoteRef, _ = git(ctx, s.Repo, "config", "--get", "branch."+plan.Branch+".merge")
	hash := sha256.New()
	fmt.Fprintf(hash, "%q\n", []string{plan.Repo, plan.Branch, head, plan.Remote, plan.RemoteRef, status, diff})
	var summary strings.Builder
	for _, entry := range strings.Split(status, "\x00") {
		if entry == "" {
			continue
		}
		if len(entry) < 4 || entry[2] != ' ' || (!strings.HasPrefix(entry[3:], "skills/") && !strings.HasPrefix(entry[3:], "agents/")) {
			return plan, fmt.Errorf("无法识别 Git 变更条目")
		}
		path := entry[3:]
		plan.Files = append(plan.Files, path)
		label := "变更"
		switch entry[:2] {
		case "??":
			label = "新增"
		case " M":
			label = "修改"
		case " D":
			label = "删除"
		}
		fmt.Fprintf(&summary, "%s  %q\n", label, path)
		if entry[:2] == "??" {
			full := filepath.Join(s.Repo, filepath.FromSlash(path))
			info, err := os.Lstat(full)
			if err != nil {
				return plan, err
			}
			if !info.Mode().IsRegular() {
				return plan, fmt.Errorf("不能提交符号链接或特殊文件: %s", path)
			}
			file, err := os.Open(full)
			if err != nil {
				return plan, err
			}
			fmt.Fprintf(hash, "%q %d %v\n", path, info.Size(), info.Mode())
			_, copyErr := io.Copy(hash, file)
			closeErr := file.Close()
			if copyErr != nil {
				return plan, copyErr
			}
			if closeErr != nil {
				return plan, closeErr
			}
		}
	}
	plan.Summary = summary.String()
	plan.digest = fmt.Sprintf("%x", hash.Sum(nil))
	return plan, nil
}

// Submit 仅提交预览过的技能和指令模板文件，可选择把当前分支推送到其上游。
func (m *Manager) Submit(ctx context.Context, plan SubmitPlan, message string, push bool, out io.Writer) error {
	unlock, err := m.lock()
	if err != nil {
		return err
	}
	defer unlock()
	current, err := m.PlanSubmit(ctx)
	if err != nil {
		return err
	}
	if plan.digest == "" || current.digest != plan.digest {
		return fmt.Errorf("仓库变更或分支已变化，请重新查看并确认提交")
	}
	if push {
		if current.Remote == "" || strings.HasPrefix(current.Remote, "-") || !strings.HasPrefix(current.RemoteRef, "refs/heads/") {
			return fmt.Errorf("当前分支没有可推送的 upstream，请先用 Git 配置；尚未创建提交")
		}
		if _, err := git(ctx, current.Repo, "check-ref-format", current.RemoteRef); err != nil {
			return err
		}
	}
	if len(current.Files) > 0 {
		message = strings.TrimSpace(message)
		if message == "" {
			return fmt.Errorf("提交说明不能为空")
		}
		// 使用字面路径，防止技能文件名中的通配符被 Git 当作路径表达式。
		paths := make([]string, len(current.Files))
		for i, path := range current.Files {
			paths[i] = ":(literal)" + path
		}
		args := append([]string{"add", "-A", "--"}, paths...)
		if _, err := git(ctx, current.Repo, args...); err != nil {
			return err
		}
		args = append([]string{"commit", "--only", "-m", message, "--"}, paths...)
		if _, err := git(ctx, current.Repo, args...); err != nil {
			return fmt.Errorf("提交失败，技能和指令变更保留在暂存区，请用 Git 检查处理: %w", err)
		}
		fmt.Fprintln(out, "技能和指令变更已提交到本地仓库")
	} else {
		fmt.Fprintln(out, "没有新的技能或指令文件变更")
	}
	if push {
		if _, err := git(ctx, current.Repo, "push", "--no-follow-tags", "--", current.Remote, "HEAD:"+current.RemoteRef); err != nil {
			return fmt.Errorf("推送失败，本地提交已保留；处理认证或远端冲突后可在向导中重新推送: %w", err)
		}
		fmt.Fprintf(out, "已推送到 %s/%s\n", current.Remote, strings.TrimPrefix(current.RemoteRef, "refs/heads/"))
	}
	return nil
}
