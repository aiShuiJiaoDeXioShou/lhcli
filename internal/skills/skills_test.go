package skills

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testManager(t *testing.T) (*Manager, string) {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := &Manager{Home: home, Dir: filepath.Join(home, ".lhcli")}
	repo := filepath.Join(home, "source")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "技能测试")
	runGit(t, repo, "config", "user.email", "skills@example.invalid")
	runGit(t, repo, "config", "commit.gpgsign", "false")
	writeSkill(t, repo, "go-api", "第一版")
	commit(t, repo)
	return m, repo
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	output, err := git(context.Background(), dir, args...)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func writeSkill(t *testing.T, repo, name, body string) {
	t.Helper()
	dir := filepath.Join(repo, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: 测试技能\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commit(t *testing.T, repo string) {
	t.Helper()
	runGit(t, repo, "add", "--all")
	runGit(t, repo, "commit", "-m", "test(skills): 更新测试技能")
}

func register(t *testing.T, m *Manager, repo string) {
	t.Helper()
	if err := m.Init(context.Background(), "", repo, false, io.Discard); err != nil {
		t.Fatal(err)
	}
}

func requireSymlink(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	err := os.Symlink(dir, filepath.Join(dir, "probe"))
	if err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("当前 Windows 环境不具备符号链接权限: %v", err)
		}
		t.Fatal(err)
	}
}

func TestInitAndDryRun(t *testing.T) {
	m, repo := testManager(t)
	if err := m.Init(context.Background(), "", repo, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(m.Dir); !os.IsNotExist(err) {
		t.Fatal("预览不应创建配置目录")
	}
	register(t, m, repo)
	register(t, m, repo)
	s, err := m.load()
	if err != nil || s.Repo != repo {
		t.Fatalf("仓库登记错误: %+v, %v", s, err)
	}
	if err := m.Init(context.Background(), "owner/repo", "", false, io.Discard); err == nil {
		t.Fatal("不应覆盖已有仓库配置")
	}
	if err := m.Init(context.Background(), "", filepath.Join(repo, "skills"), false, io.Discard); err == nil {
		t.Fatal("不应接受仓库子目录")
	}
}

func TestEnableDisableAndScopes(t *testing.T) {
	requireSymlink(t)
	m, repo := testManager(t)
	register(t, m, repo)
	selection := Selection{Name: "go-api", Agents: []string{"codex", "claude", "codex"}, Base: m.Home}
	for range 2 {
		if err := m.Enable(selection, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	s, _ := m.load()
	if len(s.Links) != 2 {
		t.Fatalf("重复启用应保持两条记录，实际 %d", len(s.Links))
	}
	project := filepath.Join(m.Home, "project")
	if err := os.Mkdir(project, 0o755); err != nil {
		t.Fatal(err)
	}
	selection.Base = project
	if err := m.Enable(selection, io.Discard); err != nil {
		t.Fatal(err)
	}
	selection.Base = m.Home
	selection.Agents = []string{"claude"}
	if err := m.Disable(selection, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(m.Home, ".claude", "skills", "go-api")); !os.IsNotExist(err) {
		t.Fatal("用户级 Claude 链接应被移除")
	}
	if _, err := os.Stat(filepath.Join(project, ".claude", "skills", "go-api", "SKILL.md")); err != nil {
		t.Fatalf("不应影响项目级链接: %v", err)
	}
	writeSkill(t, repo, "go-api", "修改后")
	content, err := os.ReadFile(filepath.Join(m.Home, ".agents", "skills", "go-api", "SKILL.md"))
	if err != nil || !strings.Contains(string(content), "修改后") {
		t.Fatalf("链接应直接反映源码修改: %s, %v", content, err)
	}
	var out bytes.Buffer
	if err := m.List(Selection{Base: m.Home}, &out); err != nil || !strings.Contains(out.String(), "已启用") {
		t.Fatalf("列表未显示链接状态: %s, %v", out.String(), err)
	}
}

func TestConflictPreflightAndOwnership(t *testing.T) {
	requireSymlink(t)
	m, repo := testManager(t)
	register(t, m, repo)
	foreign := filepath.Join(m.Home, ".claude", "skills", "go-api")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	selection := Selection{Name: "go-api", Agents: []string{"codex", "claude"}, Base: m.Home}
	if err := m.Enable(selection, io.Discard); err == nil {
		t.Fatal("预期同名目录冲突")
	}
	if _, err := os.Lstat(filepath.Join(m.Home, ".agents")); !os.IsNotExist(err) {
		t.Fatal("冲突预检失败时不应安装另一 agent 的链接")
	}
	selection.Agents = []string{"codex"}
	if err := m.Enable(selection, io.Discard); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(m.Home, ".agents", "skills", "go-api")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(link, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := m.Disable(selection, io.Discard); err == nil {
		t.Fatal("被替换为普通目录后必须拒绝删除")
	}
	if _, err := os.Stat(link); err != nil {
		t.Fatal("不应删除非受管内容")
	}
}

func TestUnmanagedMatchingLinkIsNotAdopted(t *testing.T) {
	requireSymlink(t)
	m, repo := testManager(t)
	register(t, m, repo)
	b := binding{Name: "go-api", Agent: "codex", Base: m.Home, Target: filepath.Join(repo, "skills", "go-api")}
	if err := os.MkdirAll(filepath.Dir(b.path()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(b.Target, b.path()); err != nil {
		t.Fatal(err)
	}
	selection := Selection{Name: b.Name, Agents: []string{b.Agent}, Base: b.Base}
	if err := m.Enable(selection, io.Discard); err == nil {
		t.Fatal("不应自动接管手动创建的链接")
	}
	if err := m.Disable(selection, io.Discard); err != nil || !linkMatches(b) {
		t.Fatal("停用不应删除手动链接")
	}
}

func TestEnableRollsBackOnWriteFailure(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("此用例依赖普通用户的 Unix 目录写权限")
	}
	requireSymlink(t)
	m, repo := testManager(t)
	register(t, m, repo)
	blocked := filepath.Join(m.Home, ".claude")
	if err := os.Mkdir(blocked, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })
	err := m.Enable(Selection{Name: "go-api", Agents: []string{"codex", "claude"}, Base: m.Home}, io.Discard)
	if err == nil {
		t.Fatal("预期第二个 agent 目录写入失败")
	}
	if _, err := os.Lstat(filepath.Join(m.Home, ".agents", "skills", "go-api")); !os.IsNotExist(err) {
		t.Fatal("应回滚第一个已创建的链接")
	}
	s, err := m.load()
	if err != nil || len(s.Links) != 0 {
		t.Fatalf("失败时不应留下已启用记录: %+v, %v", s, err)
	}
}

func TestMissingLinksAndSources(t *testing.T) {
	requireSymlink(t)
	m, repo := testManager(t)
	register(t, m, repo)
	selection := Selection{All: true, Agents: []string{"codex"}, Base: m.Home}
	if err := m.Enable(selection, io.Discard); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(m.Home, ".agents", "skills", "go-api")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := m.Enable(selection, io.Discard); err != nil {
		t.Fatalf("应能修复缺失链接: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(repo, "skills", "go-api")); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := m.List(selection, &out); err != nil || !strings.Contains(out.String(), "源码缺失") {
		t.Fatalf("应显示失效链接: %s, %v", out.String(), err)
	}
	if err := m.Disable(selection, io.Discard); err != nil {
		t.Fatalf("源码丢失后仍应允许停用: %v", err)
	}
}

func TestDryRunAndInvalidSelection(t *testing.T) {
	m, repo := testManager(t)
	register(t, m, repo)
	selection := Selection{Name: "go-api", Agents: []string{"codex"}, Base: m.Home, DryRun: true}
	if err := m.Enable(selection, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(m.Home, ".agents")); !os.IsNotExist(err) {
		t.Fatal("预览不应创建 agent 目录")
	}
	for _, name := range []string{"../escape", "UPPER", "a--b", "", "unknown"} {
		selection.Name = name
		if err := m.Enable(selection, io.Discard); err == nil {
			t.Fatalf("应拒绝名称 %q", name)
		}
	}
	selection.Name = "go-api"
	selection.All = true
	if err := m.Enable(selection, io.Discard); err == nil {
		t.Fatal("名称与 --all 不能同时使用")
	}
	selection.All = false
	selection.Agents = []string{"unknown"}
	if err := m.Enable(selection, io.Discard); err == nil {
		t.Fatal("应拒绝未知 agent")
	}
}

func TestRejectSymlinkParentsAndSources(t *testing.T) {
	requireSymlink(t)
	m, repo := testManager(t)
	register(t, m, repo)
	if err := os.Symlink(repo, filepath.Join(m.Home, ".agents")); err != nil {
		t.Fatal(err)
	}
	if err := m.Enable(Selection{Name: "go-api", Agents: []string{"codex"}, Base: m.Home}, io.Discard); err == nil {
		t.Fatal("应拒绝通过 agent 父目录链接写入")
	}
	if err := os.Symlink(repo, filepath.Join(repo, "skills", "go-api", "outside")); err != nil {
		t.Fatal(err)
	}
	if _, err := discover(repo); err == nil {
		t.Fatal("应拒绝源码中的符号链接")
	}
}

func TestLockAndCorruptState(t *testing.T) {
	m, repo := testManager(t)
	unlock, err := m.lock()
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Init(context.Background(), "", repo, false, io.Discard); err == nil {
		t.Fatal("并发写入应被拒绝")
	}
	unlock()
	if err := os.WriteFile(filepath.Join(m.Dir, "skills.json"), []byte("{损坏"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := m.Init(context.Background(), "", repo, false, io.Discard); err == nil {
		t.Fatal("不应覆盖损坏配置")
	}
}

func TestCloneUpdateAndLocalChanges(t *testing.T) {
	m, remote := testManager(t)
	if err := m.Init(context.Background(), remote, "", false, io.Discard); err != nil {
		t.Fatal(err)
	}
	s, _ := m.load()
	before := runGit(t, s.Repo, "rev-parse", "HEAD")
	writeSkill(t, remote, "go-api", "第二版")
	commit(t, remote)
	if err := m.Update(context.Background(), true, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got := runGit(t, s.Repo, "rev-parse", "HEAD"); got != before {
		t.Fatal("检查模式不应改变 HEAD")
	}
	if err := m.Update(context.Background(), false, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got := runGit(t, s.Repo, "rev-parse", "HEAD"); got == before {
		t.Fatal("应快进到新提交")
	}
	writeSkill(t, s.Repo, "go-api", "本地未提交修改")
	if err := m.Update(context.Background(), false, false, io.Discard); err == nil {
		t.Fatal("不应覆盖本地修改")
	}
}

func TestUpdateRejectsDivergenceAndDeletedSkill(t *testing.T) {
	requireSymlink(t)
	m, remote := testManager(t)
	if err := m.Init(context.Background(), remote, "", false, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := m.Enable(Selection{Name: "go-api", Agents: []string{"codex"}, Base: m.Home}, io.Discard); err != nil {
		t.Fatal(err)
	}
	s, _ := m.load()
	before := runGit(t, s.Repo, "rev-parse", "HEAD")
	if err := os.RemoveAll(filepath.Join(remote, "skills", "go-api")); err != nil {
		t.Fatal(err)
	}
	commit(t, remote)
	if err := m.Update(context.Background(), false, false, io.Discard); err == nil || !strings.Contains(err.Error(), "移除了已启用") {
		t.Fatalf("应拒绝移除已启用技能: %v", err)
	}
	if got := runGit(t, s.Repo, "rev-parse", "HEAD"); got != before {
		t.Fatal("校验失败应保留旧版本")
	}
	runGit(t, s.Repo, "config", "user.name", "技能测试")
	runGit(t, s.Repo, "config", "user.email", "skills@example.invalid")
	runGit(t, s.Repo, "config", "commit.gpgsign", "false")
	writeSkill(t, s.Repo, "go-api", "本地提交")
	commit(t, s.Repo)
	if err := m.Update(context.Background(), false, false, io.Discard); err == nil || !strings.Contains(err.Error(), "分叉") {
		t.Fatalf("应拒绝自动合并分叉: %v", err)
	}
}

func TestUnsafeUpstreamAndSourceURL(t *testing.T) {
	requireSymlink(t)
	m, remote := testManager(t)
	if err := m.Init(context.Background(), remote, "", false, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../outside", filepath.Join(remote, "skills", "go-api", "escape")); err != nil {
		t.Fatal(err)
	}
	commit(t, remote)
	if err := m.Update(context.Background(), false, false, io.Discard); err == nil {
		t.Fatal("应拒绝包含越界链接的上游")
	}
	for _, source := range []string{"-evil", "ext::command", "owner/repo\n", "../repo", "owner/../repo"} {
		if _, err := sourceURL(source); err == nil {
			t.Fatalf("应拒绝仓库地址 %q", source)
		}
	}
}
