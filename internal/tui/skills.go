package tui

import (
	"fmt"
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/aiShuiJiaoDeXioShou/lhcli/internal/skills"
)

func (u *wizard) skills() error {
	m, err := skills.New()
	if err != nil {
		return err
	}
	for {
		repo, err := m.Repository()
		if err != nil {
			return err
		}
		if repo == "" {
			fmt.Fprintln(u.out, "还没有配置个人技能仓库，先选择源码存放位置。")
			if err := u.setup(m); err != nil {
				if err := u.report(err); err != nil {
					return err
				}
				continue
			}
			repo, err = m.Repository()
			if err != nil || repo == "" {
				return err
			}
		}
		var action string
		if err := u.form(huh.NewSelect[string]().Title("个人 skills").Description(repo).Options(
			huh.NewOption("启用技能到 agent", "enable"),
			huh.NewOption("停用已安装的技能", "disable"),
			huh.NewOption("查看技能状态", "list"),
			huh.NewOption("提交技能变更 / 推送到远端", "submit"),
			huh.NewOption("从远端更新技能", "update"),
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
			confirmed, err = u.confirm("从远端获取并快进更新技能？本地修改会受到保护。")
			if err == nil && confirmed {
				err = m.Update(u.ctx, false, false, u.out)
			}
		default:
			err = u.links(m, action)
		}
		if err := u.report(err); err != nil {
			return err
		}
	}
}

func (u *wizard) links(m *skills.Manager, action string) error {
	global := true
	selection := skills.Selection{Agents: []string{"codex"}}
	if err := u.form(
		huh.NewSelect[bool]().Title("技能生效范围").Options(
			huh.NewOption("用户级 · 所有项目可用", true),
			huh.NewOption("项目级 · 当前目录", false),
		).Value(&global),
		huh.NewMultiSelect[string]().Title("选择 agent").Description("空格勾选；不同工具可能同时扫描共享目录").Options(
			huh.NewOption("Codex", "codex"), huh.NewOption("Claude Code", "claude"), huh.NewOption("Cursor", "cursor"),
		).Value(&selection.Agents).Validate(selected),
	); err != nil {
		return err
	}
	selection.Base = m.Home
	if !global {
		var err error
		selection.Base, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	if action == "list" {
		return m.List(selection, u.out)
	}
	names, err := m.Names(selection, action == "disable")
	if err != nil {
		return err
	}
	if len(names) == 0 {
		fmt.Fprintln(u.out, "没有可选择的技能；源码应位于仓库的 skills/<名称>/SKILL.md 中。")
		return nil
	}
	options := make([]huh.Option[string], len(names))
	for i, name := range names {
		options[i] = huh.NewOption(name, name)
	}
	if err := u.form(huh.NewMultiSelect[string]().Title("选择技能").Description("空格勾选，Ctrl+A 全选，/ 搜索").Options(options...).Height(10).
		Value(&selection.Names).Validate(selected)); err != nil {
		return err
	}
	change := m.Enable
	title := "确认启用这些技能？"
	if action == "disable" {
		change, title = m.Disable, "确认停用这些技能？源码将保留。"
	}
	selection.DryRun = true
	if err := change(selection, u.out); err != nil {
		return err
	}
	confirmed, err := u.confirm(title)
	if err != nil || !confirmed {
		return err
	}
	selection.DryRun = false
	return change(selection, u.out)
}

func (u *wizard) submit(m *skills.Manager) error {
	plan, err := m.PlanSubmit(u.ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(u.out, "\n仓库: %s\n分支: %s\n", plan.Repo, plan.Branch)
	fmt.Fprint(u.out, plan.Summary)
	message := "chore(skills): 更新个人技能"
	push := false
	var fields []huh.Field
	if len(plan.Files) > 0 {
		fields = append(fields, huh.NewInput().Title("提交说明").Value(&message).Validate(required))
	} else {
		fmt.Fprintln(u.out, "没有新的技能文件变更，可以选择推送已有提交。")
	}
	fields = append(fields, huh.NewConfirm().Title("推送到当前分支的远端上游？").
		Description("会包含当前分支尚未推送的提交；推送失败时保留本地提交。").
		Affirmative("推送到远端").Negative("不推送").Value(&push))
	if err := u.form(fields...); err != nil {
		return err
	}
	if push {
		if plan.Remote == "" || plan.RemoteRef == "" {
			return fmt.Errorf("当前分支还没有 upstream，请先用 Git 配置远端跟踪关系；尚未提交")
		}
		fmt.Fprintf(u.out, "推送到: %s/%s\n", plan.Remote, strings.TrimPrefix(plan.RemoteRef, "refs/heads/"))
	}
	if len(plan.Files) == 0 && !push {
		return nil
	}
	confirmed, err := u.confirm("确认执行上面的技能提交操作？")
	if err != nil || !confirmed {
		return err
	}
	return m.Submit(u.ctx, plan, message, push, u.out)
}
