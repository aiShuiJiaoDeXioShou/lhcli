// Package skills 管理个人技能仓库及各 agent 的技能链接。
package skills

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Manager 保存本次操作的本机目录。
type Manager struct {
	Home string
	Dir  string
}

type state struct {
	Version int       `json:"version"`
	Repo    string    `json:"repo"`
	Links   []binding `json:"links,omitempty"`
}

type binding struct {
	Name   string `json:"name"`
	Agent  string `json:"agent"`
	Base   string `json:"base"`
	Target string `json:"target"`
}

// New 创建管理器；配置独立于 agent 的技能目录。
func New() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("无法获取用户目录: %w", err)
	}
	home, err = filepath.Abs(home)
	if err != nil {
		return nil, err
	}
	return &Manager{Home: home, Dir: filepath.Join(home, ".lhcli")}, nil
}

func (m *Manager) load() (state, error) {
	s := state{Version: 1}
	data, err := os.ReadFile(filepath.Join(m.Dir, "skills.json"))
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, fmt.Errorf("读取技能配置失败: %w", err)
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("技能配置损坏，请检查 %s: %w", filepath.Join(m.Dir, "skills.json"), err)
	}
	if s.Version != 1 || (s.Repo != "" && !filepath.IsAbs(s.Repo)) {
		return s, fmt.Errorf("技能配置版本或仓库路径无效")
	}
	seen := map[string]bool{}
	for _, b := range s.Links {
		if !validName(b.Name) || !filepath.IsAbs(b.Base) || !filepath.IsAbs(b.Target) || agentDir(b.Agent) == "" {
			return s, fmt.Errorf("技能配置中存在无效的链接记录")
		}
		if seen[b.path()] {
			return s, fmt.Errorf("技能配置中存在重复的链接记录: %s", b.path())
		}
		seen[b.path()] = true
	}
	return s, nil
}

func (m *Manager) save(s state) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(m.Dir, ".skills-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(append(data, '\n')); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Rename(file.Name(), filepath.Join(m.Dir, "skills.json")); err != nil {
		return fmt.Errorf("保存技能配置失败: %w", err)
	}
	return nil
}

func (m *Manager) lock() (func(), error) {
	if err := os.MkdirAll(m.Dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(m.Dir, "skills.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("无法锁定技能配置；若上次进程意外退出，请确认没有运行中的 skills 命令后删除 %s: %w", path, err)
	}
	file.Close()
	return func() { _ = os.Remove(path) }, nil
}

func requireRepo(s state) error {
	if s.Repo == "" {
		return fmt.Errorf("尚未配置个人仓库，请运行 lhcli skills init 打开向导，或 lhcli skills init --create 创建默认仓库")
	}
	return nil
}

func agentDir(agent string) string {
	switch agent {
	case "codex":
		return ".agents"
	case "claude":
		return ".claude"
	case "cursor":
		return ".cursor"
	default:
		return ""
	}
}

func (b binding) path() string {
	return filepath.Join(b.Base, agentDir(b.Agent), "skills", b.Name)
}
