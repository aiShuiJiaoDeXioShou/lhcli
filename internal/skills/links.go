package skills

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
)

// Selection 指定技能、agent 和作用域；Base 是项目根目录或用户目录。
type Selection struct {
	Name   string
	Names  []string
	All    bool
	Agents []string
	Base   string
	DryRun bool
}

func (selection Selection) normalize() (Selection, error) {
	if len(selection.Agents) == 0 {
		return selection, fmt.Errorf("请用 --agent 指定 codex、claude 或 cursor，可用逗号分隔")
	}
	agents := []string{}
	for _, agent := range selection.Agents {
		agent = strings.TrimSpace(agent)
		if agentDir(agent) == "" {
			return selection, fmt.Errorf("不支持的 agent %q；可选 codex、claude、cursor", agent)
		}
		if !slices.Contains(agents, agent) {
			agents = append(agents, agent)
		}
	}
	selection.Agents = agents
	base, err := filepath.Abs(selection.Base)
	if err != nil {
		return selection, err
	}
	base, err = filepath.EvalSymlinks(base)
	if err != nil {
		return selection, fmt.Errorf("作用域目录不存在或不可访问: %w", err)
	}
	selection.Base = base
	return selection, nil
}

func safeParent(b binding) error {
	for _, path := range []string{filepath.Join(b.Base, agentDir(b.Agent)), filepath.Join(b.Base, agentDir(b.Agent), "skills")} {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("agent 目录必须是普通目录，不能是符号链接: %s", path)
		}
	}
	return nil
}

func linkMatches(b binding) bool {
	target, err := os.Readlink(b.path())
	if err != nil {
		return false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(b.path()), target)
	}
	return filepath.Clean(target) == filepath.Clean(b.Target)
}

func bindingIndex(links []binding, b binding) int {
	return slices.IndexFunc(links, func(existing binding) bool { return existing.path() == b.path() })
}

// Enable 按技能创建链接；批次中的任何冲突都会在写入前报告。
func (m *Manager) Enable(selection Selection, out io.Writer) error {
	return m.changeLinks(selection, true, out)
}

// Disable 只移除登记且目标仍匹配的链接，保留仓库源码。
func (m *Manager) Disable(selection Selection, out io.Writer) error {
	return m.changeLinks(selection, false, out)
}

func (m *Manager) changeLinks(selection Selection, enable bool, out io.Writer) error {
	selection, err := selection.normalize()
	if err != nil {
		return err
	}
	modes := 0
	if selection.Name != "" {
		modes++
	}
	if len(selection.Names) > 0 {
		modes++
	}
	if selection.All {
		modes++
	}
	if modes != 1 {
		return fmt.Errorf("必须且只能指定技能名称、名称列表或 --all")
	}
	if selection.Name != "" {
		selection.Names = []string{selection.Name}
	}
	for _, name := range selection.Names {
		if !validName(name) {
			return fmt.Errorf("技能名称无效: %s", name)
		}
	}
	if !selection.DryRun {
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
	var plan []binding
	if enable {
		names, err := discover(s.Repo)
		if err != nil {
			return err
		}
		if !selection.All {
			selected := []string{}
			for _, name := range selection.Names {
				if !slices.Contains(names, name) {
					return fmt.Errorf("仓库中没有技能 %s，请检查 skills/%s/SKILL.md", name, name)
				}
				if !slices.Contains(selected, name) {
					selected = append(selected, name)
				}
			}
			names = selected
		}
		for _, name := range names {
			for _, agent := range selection.Agents {
				plan = append(plan, binding{Name: name, Agent: agent, Base: selection.Base, Target: filepath.Join(s.Repo, "skills", name)})
			}
		}
	} else {
		for _, b := range s.Links {
			if b.Base == selection.Base && slices.Contains(selection.Agents, b.Agent) && (selection.All || slices.Contains(selection.Names, b.Name)) {
				plan = append(plan, b)
			}
		}
	}
	if len(plan) == 0 {
		fmt.Fprintln(out, "没有需要处理的技能")
		return nil
	}
	for _, b := range plan {
		if err := safeParent(b); err != nil {
			return err
		}
		_, err := os.Lstat(b.path())
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		index := bindingIndex(s.Links, b)
		if index >= 0 && s.Links[index].Target != b.Target {
			return fmt.Errorf("链接记录指向不同源码，请先停用: %s", b.path())
		}
		if err == nil && (index < 0 || !linkMatches(b)) {
			return fmt.Errorf("目标已存在且不属于当前受管链接，拒绝覆盖或删除: %s", b.path())
		}
	}
	action := "启用"
	if !enable {
		action = "停用"
	}
	if selection.DryRun {
		for _, b := range plan {
			fmt.Fprintf(out, "预览：%s %s → %s\n", action, b.path(), b.Target)
		}
		return nil
	}
	var changed []binding
	rollback := func(cause error) error {
		for i := len(changed) - 1; i >= 0; i-- {
			b := changed[i]
			var err error
			if enable {
				if linkMatches(b) {
					err = os.Remove(b.path())
				}
			} else {
				err = os.Symlink(b.Target, b.path())
			}
			if err != nil {
				cause = errors.Join(cause, fmt.Errorf("恢复链接失败 %s: %w", b.path(), err))
			}
		}
		return cause
	}
	for _, b := range plan {
		index := bindingIndex(s.Links, b)
		if enable {
			if !linkMatches(b) {
				if err := os.MkdirAll(filepath.Dir(b.path()), 0o755); err != nil {
					return rollback(err)
				}
				if err := os.Symlink(b.Target, b.path()); err != nil {
					return rollback(fmt.Errorf("创建链接失败 %s；Windows 请开启开发者模式或使用具备链接权限的终端: %w", b.path(), err))
				}
				changed = append(changed, b)
			}
			if index < 0 {
				s.Links = append(s.Links, b)
			}
		} else {
			if linkMatches(b) {
				if err := os.Remove(b.path()); err != nil {
					return rollback(err)
				}
				changed = append(changed, b)
			}
			s.Links = slices.Delete(s.Links, index, index+1)
		}
	}
	if err := m.save(s); err != nil {
		return rollback(err)
	}
	for _, b := range plan {
		fmt.Fprintf(out, "已%s: %s\n", action, b.path())
	}
	return nil
}

// List 展示当前作用域的技能以及实际链接状态。
func (m *Manager) List(selection Selection, out io.Writer) error {
	if len(selection.Agents) == 0 {
		selection.Agents = []string{"codex", "claude", "cursor"}
	}
	selection, err := selection.normalize()
	if err != nil {
		return err
	}
	s, err := m.load()
	if err != nil {
		return err
	}
	if err := requireRepo(s); err != nil {
		return err
	}
	names, err := discover(s.Repo)
	if err != nil {
		return err
	}
	sourceNames := slices.Clone(names)
	for _, b := range s.Links {
		if b.Base == selection.Base && slices.Contains(selection.Agents, b.Agent) && !slices.Contains(names, b.Name) {
			names = append(names, b.Name)
		}
	}
	for _, agent := range selection.Agents {
		b := binding{Base: selection.Base, Agent: agent}
		if err := safeParent(b); err != nil {
			return err
		}
		entries, err := os.ReadDir(filepath.Join(selection.Base, agentDir(agent), "skills"))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		for _, entry := range entries {
			if validName(entry.Name()) && !slices.Contains(names, entry.Name()) {
				names = append(names, entry.Name())
			}
		}
	}
	slices.Sort(names)
	fmt.Fprintf(out, "技能仓库: %s\n作用域: %s\n", s.Repo, selection.Base)
	if len(names) == 0 {
		fmt.Fprintln(out, "没有发现技能，请在仓库 skills/<名称>/SKILL.md 中添加技能")
		return nil
	}
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "技能\t来源\t"+strings.Join(selection.Agents, "\t"))
	for _, name := range names {
		source := "本地仓库"
		if !slices.Contains(sourceNames, name) {
			source = "仓库中不存在"
		}
		fmt.Fprintf(w, "%s\t%s", name, source)
		for _, agent := range selection.Agents {
			b := binding{Name: name, Base: selection.Base, Agent: agent}
			index := bindingIndex(s.Links, b)
			_, statErr := os.Lstat(b.path())
			status := "未启用"
			switch {
			case statErr != nil && !os.IsNotExist(statErr):
				return statErr
			case index >= 0 && os.IsNotExist(statErr):
				status = "链接缺失"
			case index >= 0 && !linkMatches(s.Links[index]):
				status = "冲突"
			case index >= 0 && !slices.Contains(sourceNames, name):
				status = "源码缺失"
			case index >= 0:
				status = "已启用"
			case statErr == nil:
				status = "未受管"
			}
			fmt.Fprintf(w, "\t%s", status)
		}
		fmt.Fprintln(w)
	}
	return w.Flush()
}
