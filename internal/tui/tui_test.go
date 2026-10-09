package tui

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/skills"
)

func TestProjectDirectoryValidation(t *testing.T) {
	dir := t.TempDir()
	if err := validateProjectDir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("保留"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", dir, filepath.Join(dir, "keep.txt")} {
		if err := validateProjectDir(value); err == nil {
			t.Fatalf("应拒绝目标 %q", value)
		}
	}
	if err := validateProjectDir(filepath.Join(dir, "new-project")); err != nil {
		t.Fatal(err)
	}
}

func TestAccessibleProjectCancellation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cancelled")
	input := "1\ndemo\n" + dir + "\ngithub.com/example/demo\nn\nn\nn\n"
	var out bytes.Buffer
	if err := Run(context.Background(), strings.NewReader(input), &out, &out, "project", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("取消后不应创建项目")
	}
	if !strings.Contains(out.String(), "已取消") || !strings.Contains(out.String(), "请输入") {
		t.Fatalf("缺少中文确认或提示: %s", out.String())
	}
}

func TestAccessibleEOFDoesNotAcceptDefaults(t *testing.T) {
	var out bytes.Buffer
	if err := Run(context.Background(), strings.NewReader(""), &out, &out, "", true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "已退出向导") {
		t.Fatal("输入结束必须退出，不能继续接受默认值")
	}
}

func TestSkillSelectionWizard(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, "source")
	skill := filepath.Join(repo, "skills", "go-api")
	if err := os.MkdirAll(skill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skill, "SKILL.md"), []byte("技能内容"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", repo).CombinedOutput(); err != nil {
		t.Fatalf("初始化测试仓库失败: %s %v", out, err)
	}
	m := &skills.Manager{Home: home, Dir: filepath.Join(home, ".lhcli")}
	if err := m.Init(context.Background(), "", repo, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	u := &wizard{ctx: context.Background(), out: &out, errOut: &out, accessible: true}
	// 使用默认 Codex，勾选一个技能并先取消，确认没有写入链接。
	u.in = &lineInput{r: bufio.NewReader(strings.NewReader("1\n0\n1\n0\nn\n"))}
	if err := u.links(m, "enable"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents")); !os.IsNotExist(err) {
		t.Fatal("取消向导不应写入 agent 目录")
	}
	probe := filepath.Join(home, "probe")
	if err := os.Symlink(skill, probe); err != nil {
		if runtime.GOOS == "windows" {
			t.Skip("当前 Windows 无符号链接权限，已验证取消流程")
		}
		t.Fatal(err)
	}
	u.in = &lineInput{r: bufio.NewReader(strings.NewReader("1\n0\n1\n0\ny\n"))}
	if err := u.links(m, "enable"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "go-api", "SKILL.md")); err != nil {
		t.Fatalf("确认后应启用选中的技能: %v", err)
	}
}

func TestPipedIOIsNotTerminal(t *testing.T) {
	if IsTerminal(&bytes.Buffer{}, &bytes.Buffer{}) {
		t.Fatal("普通输入输出流不应进入 TUI")
	}
}
