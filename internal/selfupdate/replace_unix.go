//go:build !windows

package selfupdate

import (
	"io"
	"os"
	"path/filepath"
)

// replaceExecutable 用 newFile 替换 target：先在同目录写入临时文件，再原子重命名。
func replaceExecutable(target, newFile string) error {
	source, err := os.Open(newFile)
	if err != nil {
		return err
	}
	defer source.Close()

	temp, err := os.CreateTemp(filepath.Dir(target), ".lhcli-update-*")
	if err != nil {
		return wrapReplaceError(err, target)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if _, err := io.Copy(temp, source); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Chmod(0o755); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}

	// 运行中的二进制可以被打断原子替换，进程继续使用旧的 inode。
	if err := os.Rename(tempPath, target); err != nil {
		return wrapReplaceError(err, target)
	}
	return nil
}
