package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"charm.land/huh/v2"
	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/skills"
)

func (u *wizard) agents() error {
	m, err := skills.New()
	if err != nil {
		return err
	}
	ready, err := u.ensureRepository(m)
	if err != nil || !ready {
		return err
	}
	for {
		repo, err := m.Repository()
		if err != nil {
			return err
		}
		var action string
		if err := u.form(huh.NewSelect[string]().Title("AGENTS.md · 指令模板").Description(filepath.Join(repo, "agents")).Options(
			huh.NewOption("应用模板到项目", "apply"),
			huh.NewOption("收录项目的 AGENTS.md", "capture"),
			huh.NewOption("查看模板内容", "show"),
			huh.NewOption("提交技能和指令 / 推送到远端", "submit"),
			huh.NewOption("从远端更新仓库", "update"),
			huh.NewOption("返回", "back"),
		).Value(&action)); err != nil {
			return err
		}
		switch action {
		case "back":
			return nil
		case "submit":
			err = u.submit(m)
		case "update":
			var confirmed bool
			confirmed, err = u.confirm("从远端获取并快进更新个人仓库？")
			if err == nil && confirmed {
				err = m.Update(u.ctx, false, false, u.out)
			}
		default:
			err = u.template(m, action)
		}
		if err := u.report(err); err != nil {
			return err
		}
	}
}

func (u *wizard) template(m *skills.Manager, action string) error {
	var name string
	if action == "capture" {
		if err := u.form(huh.NewInput().Title("保存为哪个模板？").Description("小写字母、数字和连字符，例如 go-project").Validate(required).Value(&name)); err != nil {
			return err
		}
	} else {
		names, err := m.TemplateNames()
		if err != nil {
			return err
		}
		if len(names) == 0 {
			fmt.Fprintln(u.out, "还没有模板，请先选择“收录项目的 AGENTS.md”。")
			return u.pause()
		}
		options := make([]huh.Option[string], len(names))
		for i, name := range names {
			options[i] = huh.NewOption(name, name)
		}
		if err := u.form(huh.NewSelect[string]().Title("选择指令模板").Options(options...).Value(&name)); err != nil {
			return err
		}
		if err := m.ShowTemplate(name, u.out); err != nil {
			return err
		}
		if action == "show" {
			return u.pause()
		}
	}
	project, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := u.form(u.input("项目目录", &project, required).Description("读取或写入这个目录里的 AGENTS.md")); err != nil {
		return err
	}
	project, err = expandPath(project)
	if err != nil {
		return err
	}
	if err := m.TransferTemplate(name, project, action == "capture", true, true, u.out); err != nil {
		return err
	}
	confirmed, err := u.confirm("确认复制指令？已有文件将先备份再替换。")
	if err != nil || !confirmed {
		return err
	}
	if err := m.TransferTemplate(name, project, action == "capture", true, false, u.out); err != nil {
		return err
	}
	return u.pause()
}
