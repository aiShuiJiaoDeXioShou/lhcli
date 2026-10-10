package skills

import "slices"

// Repository 返回已登记的仓库路径；尚未登记时返回空字符串。
func (m *Manager) Repository() (string, error) {
	s, err := m.load()
	return s.Repo, err
}

// Names 返回可启用或已登记的技能名称，供交互界面选择。
func (m *Manager) Names(selection Selection, enabledOnly bool) ([]string, error) {
	s, err := m.load()
	if err != nil {
		return nil, err
	}
	if err := requireRepo(s); err != nil {
		return nil, err
	}
	if !enabledOnly {
		return discover(s.Repo)
	}
	selection, err = selection.normalize()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, b := range s.Links {
		if b.Base == selection.Base && slices.Contains(selection.Agents, b.Agent) && !slices.Contains(names, b.Name) {
			names = append(names, b.Name)
		}
	}
	slices.Sort(names)
	return names, nil
}
