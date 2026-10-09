package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var skillName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func validName(name string) bool {
	return len(name) <= 64 && skillName.MatchString(name)
}

func discover(repo string) ([]string, error) {
	root := filepath.Join(repo, "skills")
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("技能根目录必须是普通目录: %s", root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("技能源码不能使用符号链接: %s", path)
		}
		if !entry.IsDir() {
			continue
		}
		info, err := os.Lstat(filepath.Join(path, "SKILL.md"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !validName(entry.Name()) || !info.Mode().IsRegular() || info.Size() == 0 {
			return nil, fmt.Errorf("技能名称或 SKILL.md 无效: %s", path)
		}
		if err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && !d.Type().IsRegular() {
				return fmt.Errorf("技能源码包含符号链接或特殊文件: %s", p)
			}
			return nil
		}); err != nil {
			return nil, err
		}
		names = append(names, entry.Name())
	}
	return names, nil
}
