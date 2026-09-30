package scaffold

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// archiveSymlink 记录待创建的符号链接。
type archiveSymlink struct {
	path   string
	target string
}

// extractTarGz 解压 tarball 到 dest，去掉顶层目录并保留可执行位。
func extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("解压模板失败: %w", err)
	}
	defer gzipReader.Close()

	cleanDest := filepath.Clean(dest)
	var symlinks []archiveSymlink

	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取模板失败: %w", err)
		}

		rel := stripFirstComponent(header.Name)
		if rel == "" {
			continue
		}
		target := filepath.Join(cleanDest, filepath.FromSlash(rel))
		if !withinDir(cleanDest, target) {
			return fmt.Errorf("模板包含非法路径: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := writeFileFromTar(reader, target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			symlinks = append(symlinks, archiveSymlink{path: target, target: header.Linkname})
		}
	}

	// 符号链接延后创建，确保链接目标已存在。
	for _, link := range symlinks {
		if err := createSymlinkOrCopy(link.path, link.target); err != nil {
			return err
		}
	}
	return nil
}

// stripFirstComponent 去掉 tar 条目中的顶层目录。
func stripFirstComponent(name string) string {
	name = strings.TrimPrefix(name, "./")
	index := strings.IndexByte(name, '/')
	if index < 0 {
		return ""
	}
	return name[index+1:]
}

// withinDir 判断 target 是否位于 dir 之内，用于阻止路径穿越。
func withinDir(dir, target string) bool {
	if target == dir {
		return true
	}
	return strings.HasPrefix(target, dir+string(os.PathSeparator))
}

// writeFileFromTar 从 tar 读取内容写入文件。
func writeFileFromTar(reader io.Reader, path string, mode os.FileMode) error {
	perm := mode.Perm()
	if perm == 0 {
		perm = 0o644
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(file, reader); err != nil {
		return err
	}
	return file.Close()
}

// createSymlinkOrCopy 创建符号链接，失败时（如 Windows 无权限）回退为复制目标内容。
func createSymlinkOrCopy(path, linkTarget string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.Symlink(filepath.FromSlash(linkTarget), path); err == nil {
		return nil
	}
	resolved := filepath.Join(filepath.Dir(path), filepath.FromSlash(linkTarget))
	if _, err := os.Lstat(resolved); err != nil {
		// 链接目标不存在时忽略，避免因平台差异中断初始化。
		return nil
	}
	return copyPath(resolved, path)
}

// copyPath 递归复制文件或目录。
func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyPath(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	target, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return err
	}
	return target.Close()
}
