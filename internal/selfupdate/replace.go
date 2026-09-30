package selfupdate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// executablePath 返回当前可执行文件的真实路径（解析符号链接）。
func executablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// wrapReplaceError 为替换失败补充中文提示。
func wrapReplaceError(err error, target string) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("无权限替换 %s: %w\n请使用 sudo 重试，或重新运行 install.sh 安装到可写目录", target, err)
	}
	return fmt.Errorf("替换 %s 失败: %w", target, err)
}

// CleanupStale 清理上次更新遗留的 .old 备份；在程序启动时尽力执行，忽略错误。
func CleanupStale() {
	exe, err := executablePath()
	if err != nil {
		return
	}
	backup := exe + ".old"
	if _, err := os.Stat(backup); err != nil {
		return
	}
	// 上次更新在改名后中断：目标缺失时把备份还原回去。
	if _, err := os.Stat(exe); os.IsNotExist(err) {
		_ = os.Rename(backup, exe)
		return
	}
	_ = os.Remove(backup)
}
