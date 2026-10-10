package skills

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSubmitOnlySkills(t *testing.T) {
	m, repo := testManager(t)
	register(t, m, repo)
	writeSkill(t, repo, "go-api", "待提交修改")
	writeSkill(t, repo, "new-skill", "新增技能")
	outside := filepath.Join(repo, "private.txt")
	if err := os.WriteFile(outside, []byte("不属于技能的内容"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := m.PlanSubmit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Files) != 2 || !strings.Contains(plan.Summary, "go-api/SKILL.md") {
		t.Fatalf("提交预览错误: %+v", plan)
	}
	if err := m.Submit(context.Background(), plan, "feat(skills): 添加技能", false, io.Discard); err != nil {
		t.Fatal(err)
	}
	files := runGit(t, repo, "show", "--format=", "--name-only", "HEAD")
	if strings.Contains(files, "private.txt") || !strings.Contains(files, "new-skill/SKILL.md") {
		t.Fatalf("提交范围错误: %s", files)
	}
	if got := runGit(t, repo, "status", "--porcelain"); !strings.Contains(got, "private.txt") || strings.Contains(got, "skills/") {
		t.Fatalf("提交后工作区状态错误: %s", got)
	}
}

func TestSubmitPreservesExistingIndex(t *testing.T) {
	m, repo := testManager(t)
	register(t, m, repo)
	writeSkill(t, repo, "go-api", "已经暂存")
	runGit(t, repo, "add", "skills")
	before := runGit(t, repo, "diff", "--cached")
	if _, err := m.PlanSubmit(context.Background()); err == nil || !strings.Contains(err.Error(), "已有暂存") {
		t.Fatalf("应拒绝已有暂存内容: %v", err)
	}
	if after := runGit(t, repo, "diff", "--cached"); after != before {
		t.Fatal("不应改变用户的暂存区")
	}
}

func TestSubmitRejectsChangedPreview(t *testing.T) {
	for _, name := range []string{"go-api", "new-skill"} {
		t.Run(name, func(t *testing.T) {
			m, repo := testManager(t)
			register(t, m, repo)
			writeSkill(t, repo, name, "预览内容")
			plan, err := m.PlanSubmit(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			writeSkill(t, repo, name, "确认后已修改")
			before := runGit(t, repo, "rev-parse", "HEAD")
			if err := m.Submit(context.Background(), plan, "test(skills): 不应提交", false, io.Discard); err == nil {
				t.Fatal("不应提交确认之后的变化")
			}
			if after := runGit(t, repo, "rev-parse", "HEAD"); after != before {
				t.Fatal("不应产生新提交")
			}
		})
	}
}

func TestSubmitDeletionAndLiteralPath(t *testing.T) {
	m, repo := testManager(t)
	register(t, m, repo)
	if err := os.Remove(filepath.Join(repo, "skills", "go-api", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, repo, "new-skill", "新技能")
	if err := os.WriteFile(filepath.Join(repo, "skills", "new-skill", "[笔记].md"), []byte("按字面路径处理"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := m.PlanSubmit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Submit(context.Background(), plan, "chore(skills): 调整文件", false, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got := runGit(t, repo, "status", "--porcelain"); got != "" {
		t.Fatalf("提交后应干净: %s", got)
	}
}

func TestSubmitAndRetryPush(t *testing.T) {
	m, repo := testManager(t)
	register(t, m, repo)
	remote := filepath.Join(t.TempDir(), "remote.git")
	runGit(t, repo, "init", "--bare", remote)
	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-u", "origin", "main")
	writeSkill(t, repo, "go-api", "提交并推送")
	plan, err := m.PlanSubmit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Submit(context.Background(), plan, "feat(skills): 更新技能", true, io.Discard); err != nil {
		t.Fatal(err)
	}
	if local, remoteHead := runGit(t, repo, "rev-parse", "HEAD"), runGit(t, remote, "rev-parse", "refs/heads/main"); local != remoteHead {
		t.Fatal("远端没有收到提交")
	}
	plan, err = m.PlanSubmit(context.Background())
	if err != nil || len(plan.Files) != 0 {
		t.Fatalf("提交后应无变更: %+v %v", plan, err)
	}
	if err := m.Submit(context.Background(), plan, "", true, io.Discard); err != nil {
		t.Fatalf("应允许只推送已有提交: %v", err)
	}
}

func TestSubmitRequiresUpstreamBeforeCommit(t *testing.T) {
	m, repo := testManager(t)
	register(t, m, repo)
	writeSkill(t, repo, "go-api", "待提交")
	plan, err := m.PlanSubmit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	before := runGit(t, repo, "rev-parse", "HEAD")
	if err := m.Submit(context.Background(), plan, "chore(skills): 更新", true, io.Discard); err == nil {
		t.Fatal("缺少上游时应在提交前报错")
	}
	if after := runGit(t, repo, "rev-parse", "HEAD"); before != after {
		t.Fatal("不应创建提交")
	}
}

func TestFailedPushKeepsLocalCommit(t *testing.T) {
	m, repo := testManager(t)
	register(t, m, repo)
	runGit(t, repo, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing.git"))
	runGit(t, repo, "config", "branch.main.remote", "origin")
	runGit(t, repo, "config", "branch.main.merge", "refs/heads/main")
	writeSkill(t, repo, "go-api", "远端暂不可用")
	plan, err := m.PlanSubmit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	before := runGit(t, repo, "rev-parse", "HEAD")
	err = m.Submit(context.Background(), plan, "chore(skills): 本地保留", true, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "本地提交已保留") {
		t.Fatalf("应报告推送失败并保留提交: %v", err)
	}
	if after := runGit(t, repo, "rev-parse", "HEAD"); before == after {
		t.Fatal("应保留新创建的本地提交")
	}
	if got := runGit(t, repo, "status", "--porcelain"); got != "" {
		t.Fatalf("推送失败不应恢复旧文件: %s", got)
	}
}

func TestMultiSkillSelection(t *testing.T) {
	requireSymlink(t)
	m, repo := testManager(t)
	writeSkill(t, repo, "review", "审查技能")
	register(t, m, repo)
	selection := Selection{Names: []string{"go-api", "review", "go-api"}, Agents: []string{"codex"}, Base: m.Home}
	if err := m.Enable(selection, io.Discard); err != nil {
		t.Fatal(err)
	}
	names, err := m.Names(selection, true)
	if err != nil || len(names) != 2 {
		t.Fatalf("应返回两个已启用技能: %v %v", names, err)
	}
	selection.Names = []string{"review"}
	if err := m.Disable(selection, io.Discard); err != nil {
		t.Fatal(err)
	}
	names, err = m.Names(selection, true)
	if err != nil || len(names) != 1 || names[0] != "go-api" {
		t.Fatalf("应只停用选中的技能: %v %v", names, err)
	}
}
