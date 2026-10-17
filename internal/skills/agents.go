package skills

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func readDocument(path string) ([]byte, error) {
	content, err := readRegularFile(path)
	if err == nil && len(bytes.TrimSpace(content)) == 0 {
		return nil, fmt.Errorf("文件内容不能为空: %s", path)
	}
	return content, err
}

func readRegularFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("需要普通文件，不能使用符号链接: %s", path)
	}
	return os.ReadFile(path)
}

func (m *Manager) templatePath(name string) (string, error) {
	if !validName(name) {
		return "", fmt.Errorf("模板名称应为小写字母、数字和连字符，最多 64 字符")
	}
	s, err := m.load()
	if err != nil {
		return "", err
	}
	if err := requireRepo(s); err != nil {
		return "", err
	}
	root := filepath.Join(s.Repo, "agents")
	for _, dir := range []string{root, filepath.Join(root, name)} {
		if err := plainDirectory(dir); err != nil {
			return "", err
		}
	}
	return filepath.Join(root, name, "AGENTS.md"), nil
}

// TemplateNames 列出默认仓库中的指令模板。
func (m *Manager) TemplateNames() ([]string, error) {
	path, err := m.templatePath("default")
	if err != nil {
		return nil, err
	}
	root := filepath.Dir(filepath.Dir(path))
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path, err := m.templatePath(entry.Name())
		if err != nil {
			return nil, err
		}
		if _, err := readDocument(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		names = append(names, entry.Name())
	}
	return names, nil
}

// ShowTemplate 显示指令模板原文。
func (m *Manager) ShowTemplate(name string, out io.Writer) error {
	path, err := m.templatePath(name)
	if err != nil {
		return err
	}
	content, err := readDocument(path)
	if err != nil {
		return err
	}
	_, err = out.Write(content)
	return err
}

// TransferTemplate 应用指令模板或收回项目指令，替换前备份旧文件。
func (m *Manager) TransferTemplate(name, project string, capture, replace, dryRun bool, out io.Writer) error {
	if !dryRun {
		unlock, err := m.lock()
		if err != nil {
			return err
		}
		defer unlock()
	}
	template, err := m.templatePath(name)
	if err != nil {
		return err
	}
	project, err = filepath.Abs(project)
	if err != nil {
		return err
	}
	project, err = filepath.EvalSymlinks(project)
	if err != nil {
		return err
	}
	info, err := os.Stat(project)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("项目路径必须是已有目录: %s", project)
	}
	source, dest := template, filepath.Join(project, "AGENTS.md")
	if capture {
		source, dest = dest, source
	}
	content, err := readDocument(source)
	if err != nil {
		return err
	}
	old, err := readRegularFile(dest)
	exists := !os.IsNotExist(err)
	if exists && err != nil {
		return err
	}
	if exists && bytes.Equal(old, content) {
		fmt.Fprintf(out, "内容一致，无需修改: %s\n", dest)
		return nil
	}
	if exists && !replace {
		return fmt.Errorf("目标文件已存在: %s；使用 --replace 先备份再替换", dest)
	}
	fmt.Fprintf(out, "复制指令: %s → %s\n", source, dest)
	if exists {
		fmt.Fprintln(out, "目标已有内容，将先保存独立备份再替换。")
	}
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if !exists {
		file, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(content)
		closeErr := file.Close()
		if writeErr != nil {
			_ = os.Remove(dest)
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
	} else {
		backupDir := filepath.Join(m.Dir, "backups")
		if err := os.MkdirAll(backupDir, 0o700); err != nil {
			return err
		}
		backup, err := os.CreateTemp(backupDir, "AGENTS-*.md")
		if err != nil {
			return err
		}
		_, writeErr := backup.Write(old)
		closeErr := backup.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Fprintf(out, "旧文件备份: %s\n", backup.Name())
		file, err := os.CreateTemp(filepath.Dir(dest), ".AGENTS-*")
		if err != nil {
			return err
		}
		defer os.Remove(file.Name())
		original, err := os.Lstat(dest)
		if err != nil {
			file.Close()
			return err
		}
		if err := file.Chmod(original.Mode().Perm()); err != nil {
			file.Close()
			return err
		}
		_, writeErr = file.Write(content)
		closeErr = file.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		current, err := readRegularFile(dest)
		if err != nil {
			return err
		}
		if !bytes.Equal(current, old) {
			return fmt.Errorf("目标文件在操作期间发生变化，请重试；备份已保留")
		}
		if err := os.Rename(file.Name(), dest); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "指令文件已保存；应用到项目的是独立副本，后续不会随模板自动变化。")
	return nil
}

func validateManagedTrees(repo string) error {
	for _, name := range []string{"skills", "agents"} {
		path := filepath.Join(repo, name)
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return err
		}
		if err := plainDirectory(path); err != nil {
			return err
		}
		if err := regularTree(path); err != nil {
			return err
		}
	}
	return nil
}
