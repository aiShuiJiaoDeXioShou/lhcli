package skills

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateDefaultRepository(t *testing.T) {
	home := t.TempDir()
	config := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(config, []byte("[user]\nname = 测试\nemail = test@example.invalid\n[commit]\ngpgsign = false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	m := &Manager{Home: home, Dir: filepath.Join(home, ".lhcli")}
	if err := m.Create(context.Background(), "", true, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(m.Dir); !os.IsNotExist(err) {
		t.Fatal("预览不能写入配置或目录")
	}
	if err := m.Create(context.Background(), "", false, io.Discard); err != nil {
		t.Fatal(err)
	}
	repo, err := m.Repository()
	if err != nil || repo == "" {
		t.Fatalf("默认仓库未登记: %v", err)
	}
	if status := runGit(t, repo, "status", "--porcelain"); status != "" {
		t.Fatalf("初始化后应无变更: %s", status)
	}
	if _, err := m.PlanSubmit(context.Background()); err != nil {
		t.Fatalf("新仓库应可以直接提交: %v", err)
	}
	if names, err := m.TemplateNames(); err != nil || len(names) != 1 || names[0] != "default" {
		t.Fatalf("默认模板缺失: %v %v", names, err)
	}
	if err := m.Create(context.Background(), "", false, io.Discard); err == nil {
		t.Fatal("不能重复创建或覆盖仓库")
	}
}

func TestImportSkill(t *testing.T) {
	m, repo := testManager(t)
	register(t, m, repo)
	origin := t.TempDir()
	writeSkill(t, origin, "example", "完整技能")
	source := filepath.Join(origin, "skills", "example")
	if err := os.WriteFile(filepath.Join(source, "run.sh"), []byte("echo ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := m.Import(source, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(repo, "skills", "example")
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("预览不能复制")
	}
	if err := m.Import(source, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"SKILL.md", "run.sh"} {
		before, _ := os.ReadFile(filepath.Join(source, file))
		after, err := os.ReadFile(filepath.Join(dest, file))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("复制应保留内容和来源: %s %v", file, err)
		}
	}
	if err := m.Import(source, false, io.Discard); err == nil {
		t.Fatal("同名技能不应覆盖")
	}
	if err := m.Import(filepath.Join(repo, "skills"), false, io.Discard); err == nil {
		t.Fatal("不能导入目标的父目录")
	}
}

func TestImportRejectsSymlinks(t *testing.T) {
	requireSymlink(t)
	m, repo := testManager(t)
	register(t, m, repo)
	origin := t.TempDir()
	writeSkill(t, origin, "example", "测试")
	source := filepath.Join(origin, "skills", "example")
	if err := os.Symlink(repo, filepath.Join(source, "outside")); err != nil {
		t.Fatal(err)
	}
	if err := m.Import(source, false, io.Discard); err == nil {
		t.Fatal("应拒绝含符号链接的技能")
	}
	if _, err := os.Stat(filepath.Join(repo, "skills", "example")); !os.IsNotExist(err) {
		t.Fatal("不能留下半成品")
	}
}

func TestTemplateCaptureApplyAndBackup(t *testing.T) {
	m, repo := testManager(t)
	register(t, m, repo)
	project := t.TempDir()
	file := filepath.Join(project, "AGENTS.md")
	if err := os.WriteFile(file, []byte("项目指令\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.TransferTemplate("go-project", project, true, false, true, io.Discard); err != nil {
		t.Fatal(err)
	}
	if names, _ := m.TemplateNames(); len(names) != 0 {
		t.Fatal("预览不应创建模板")
	}
	if err := m.TransferTemplate("go-project", project, true, false, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("保留的项目修改\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.TransferTemplate("go-project", project, false, false, false, io.Discard); err == nil {
		t.Fatal("默认不能覆盖项目指令")
	}
	var output bytes.Buffer
	if err := m.TransferTemplate("go-project", project, false, true, false, &output); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(file)
	if string(content) != "项目指令\n" {
		t.Fatalf("应用内容错误: %s", content)
	}
	backups, _ := filepath.Glob(filepath.Join(m.Dir, "backups", "AGENTS-*.md"))
	if len(backups) != 1 {
		t.Fatalf("应有一个备份: %v", backups)
	}
	backup, _ := os.ReadFile(backups[0])
	if string(backup) != "保留的项目修改\n" || !strings.Contains(output.String(), backups[0]) {
		t.Fatal("备份内容或提示错误")
	}
	if err := m.TransferTemplate("../escape", project, true, true, false, io.Discard); err == nil {
		t.Fatal("应拒绝路径穿越")
	}
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.TransferTemplate("go-project", project, false, true, false, io.Discard); err != nil {
		t.Fatalf("应能替换空文件: %v", err)
	}
	if err := m.TransferTemplate("go-project", t.TempDir(), false, false, false, io.Discard); err != nil {
		t.Fatalf("应能应用到新项目: %v", err)
	}
}

func TestTemplateRejectsSymlinks(t *testing.T) {
	requireSymlink(t)
	m, repo := testManager(t)
	register(t, m, repo)
	project := t.TempDir()
	if err := os.Symlink(filepath.Join(repo, "skills", "go-api", "SKILL.md"), filepath.Join(project, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if err := m.TransferTemplate("example", project, true, true, false, io.Discard); err == nil {
		t.Fatal("不能收录符号链接")
	}
	if err := os.Symlink(project, filepath.Join(repo, "agents")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.TemplateNames(); err == nil {
		t.Fatal("不能使用符号链接作为模板根目录")
	}
}

func TestSubmitIncludesTemplatesAndDetectsChanges(t *testing.T) {
	m, repo := testManager(t)
	register(t, m, repo)
	project := t.TempDir()
	file := filepath.Join(project, "AGENTS.md")
	if err := os.WriteFile(file, []byte("初始模板"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.TransferTemplate("example", project, true, false, false, io.Discard); err != nil {
		t.Fatal(err)
	}
	plan, err := m.PlanSubmit(context.Background())
	if err != nil || len(plan.Files) != 1 || plan.Files[0] != "agents/example/AGENTS.md" {
		t.Fatalf("应包括模板: %+v %v", plan, err)
	}
	if err := os.WriteFile(filepath.Join(repo, "agents", "example", "AGENTS.md"), []byte("预览后修改"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.Submit(context.Background(), plan, "test(agents): 提交模板", false, io.Discard); err == nil {
		t.Fatal("应拒绝预览后变化")
	}
	plan, err = m.PlanSubmit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Submit(context.Background(), plan, "test(agents): 提交模板", false, io.Discard); err != nil {
		t.Fatal(err)
	}
	if files := runGit(t, repo, "show", "--format=", "--name-only", "HEAD"); files != "agents/example/AGENTS.md" {
		t.Fatalf("提交文件错误: %s", files)
	}
}
