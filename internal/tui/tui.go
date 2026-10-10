// Package tui 提供项目初始化和技能管理的交互式向导。
package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/skills"
	"github.com/charmbracelet/x/term"
)

type wizard struct {
	ctx        context.Context
	in         io.Reader
	out        io.Writer
	errOut     io.Writer
	accessible bool
}

// IsTerminal 判断输入和输出是否都连接到终端，避免流水线等待键盘输入。
func IsTerminal(in io.Reader, out io.Writer) bool {
	input, inOK := in.(*os.File)
	output, outOK := out.(*os.File)
	return inOK && outOK && term.IsTerminal(input.Fd()) && term.IsTerminal(output.Fd())
}

// Run 打开指定入口的向导；空入口表示主菜单。
func Run(ctx context.Context, in io.Reader, out, errOut io.Writer, start string, accessible bool) error {
	u := &wizard{ctx: ctx, in: in, out: out, errOut: errOut, accessible: accessible}
	var err error
	if accessible {
		err = u.runAccessible(start)
	} else {
		err = u.run(start)
	}
	if errors.Is(err, huh.ErrUserAborted) || errors.Is(err, context.Canceled) || ctx.Err() != nil {
		fmt.Fprintln(out, "已退出向导")
		return nil
	}
	return err
}

func (u *wizard) run(start string) error {
	if start == "project" {
		return u.project()
	}
	if start == "agents" {
		return u.agents()
	}
	if start == "skills" {
		return u.skills()
	}
	if start == "setup" || start == "submit" || start == "import" || start == "enable" || start == "disable" {
		m, err := skills.New()
		if err != nil {
			return err
		}
		if start == "setup" {
			return u.setup(m)
		}
		ready, err := u.ensureRepository(m)
		if err != nil || !ready {
			return err
		}
		switch start {
		case "import":
			return u.importSkill(m)
		case "enable", "disable":
			return u.links(m, start)
		}
		return u.submit(m)
	}
	for {
		var action string
		if err := u.form(huh.NewSelect[string]().Title("lhcli · 开始工作").Description("选择要完成的事情").Options(
			huh.NewOption("初始化项目", "project"),
			huh.NewOption("管理 skills", "skills"),
			huh.NewOption("管理 AGENTS.md 指令模板", "agents"),
			huh.NewOption("退出", "exit"),
		).Value(&action)); err != nil {
			return err
		}
		var err error
		switch action {
		case "project":
			err = u.project()
		case "skills":
			err = u.skills()
		case "agents":
			err = u.agents()
		default:
			return nil
		}
		if err := u.report(err); err != nil {
			return err
		}
	}
}

func (u *wizard) form(fields ...huh.Field) error {
	var output io.Writer = u.out
	if u.accessible {
		fmt.Fprintln(u.out, "\n输入编号或文字后回车；确认时 y 表示是、n 表示否；Ctrl+C 退出。")
		output = promptWriter{u.out}
	} else {
		fmt.Fprintln(u.out, "\n↑/↓ 选择 · 空格 多选 · Enter 继续 · Shift+Tab 上一项 · Ctrl+C 退出")
	}
	return huh.NewForm(huh.NewGroup(fields...)).WithInput(u.in).WithOutput(output).
		WithAccessible(u.accessible).WithShowHelp(false).RunWithContext(u.ctx)
}

func (u *wizard) input(title string, value *string, validate func(string) error) *huh.Input {
	initial := *value
	if u.accessible && initial != "" {
		title += "（回车使用 " + initial + "）"
	}
	return huh.NewInput().Title(title).Value(value).Validate(func(input string) error {
		// Huh 逐项模式在补入默认值前校验，这里按实际将采用的值校验。
		if u.accessible && strings.TrimSpace(input) == "" {
			input = initial
		}
		return validate(input)
	})
}

func (u *wizard) confirm(title string) (bool, error) {
	confirmed := false
	err := u.form(huh.NewConfirm().Title(title).Affirmative("确认执行").Negative("取消").Value(&confirmed))
	if err == nil && !confirmed {
		fmt.Fprintln(u.out, "已取消，未执行操作")
	}
	return confirmed, err
}

func (u *wizard) report(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, huh.ErrUserAborted) || u.ctx.Err() != nil {
		return err
	}
	fmt.Fprintln(u.errOut, "操作失败:", err)
	if u.accessible {
		return nil
	}
	return u.form(huh.NewNote().Title("操作未完成").Description("上方保留了错误详情。按 Enter 返回菜单后可重试。").Next(true).NextLabel("返回菜单"))
}

func required(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("此项不能为空")
	}
	return nil
}

func selected(values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("请至少选择一项，按空格勾选")
	}
	return nil
}

func expandPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "~" || strings.HasPrefix(value, "~/") || strings.HasPrefix(value, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		value = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(value[1:], "/"), "\\"))
	}
	return filepath.Abs(value)
}

func (u *wizard) setup(m *skills.Manager) error {
	repo, err := m.Repository()
	if err != nil {
		return err
	}
	if repo != "" {
		return m.Info(u.ctx, u.out)
	}
	var method, value string
	if err := u.form(huh.NewSelect[string]().Title("首次设置 · 个人技能与指令仓库").Description("一个仓库保存 skills 和 AGENTS.md 模板").Options(
		huh.NewOption("创建个人仓库（推荐，自动初始化）", "create"),
		huh.NewOption("登记本机已有的 Git 仓库", "local"),
		huh.NewOption("从远端克隆 Git 仓库", "remote"),
		huh.NewOption("返回", "back"),
	).Value(&method)); err != nil {
		return err
	}
	if method == "back" {
		return nil
	}
	if method == "create" {
		value = filepath.Join(m.Dir, "repos", "my-skills")
		if err := u.form(u.input("新仓库保存位置", &value, required)); err != nil {
			return err
		}
		path, err := expandPath(value)
		if err != nil {
			return err
		}
		if err := m.Create(u.ctx, path, true, u.out); err != nil {
			return err
		}
		confirmed, err := u.confirm("创建并设为默认仓库？")
		if err != nil || !confirmed {
			return err
		}
		return m.Create(u.ctx, path, false, u.out)
	}
	title, placeholder := "本地仓库目录", "~/code/my-skills"
	if method == "remote" {
		title, placeholder = "远端仓库地址", "owner/my-skills 或 git@github.com:owner/my-skills.git"
	}
	if err := u.form(huh.NewInput().Title(title).Placeholder(placeholder).Validate(required).Value(&value)); err != nil {
		return err
	}
	var source, path string
	if method == "local" {
		var err error
		path, err = expandPath(value)
		if err != nil {
			return err
		}
	} else {
		source = strings.TrimSpace(value)
	}
	if err := m.Init(u.ctx, source, path, true, u.out); err != nil {
		return err
	}
	confirmed, err := u.confirm("确认配置这个技能仓库？")
	if err != nil || !confirmed {
		return err
	}
	return m.Init(u.ctx, source, path, false, u.out)
}

func (u *wizard) ensureRepository(m *skills.Manager) (bool, error) {
	repo, err := m.Repository()
	if err != nil {
		return false, err
	}
	if repo == "" {
		if err := u.setup(m); err != nil {
			return false, err
		}
		repo, err = m.Repository()
	}
	return repo != "", err
}

func (u *wizard) pause() error {
	if u.accessible {
		return nil
	}
	return u.form(huh.NewNote().Title("操作结果见上方").Next(true).NextLabel("返回菜单"))
}
