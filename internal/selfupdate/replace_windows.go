//go:build windows

package selfupdate

import "os"

// replaceExecutable 用 newFile 替换 target。
// Windows 下无法覆盖正在运行的 exe，采取「改名备份 + 落位 + 下次启动清理」策略。
func replaceExecutable(target, newFile string) error {
	backup := target + ".old"
	_ = os.Remove(backup)

	if err := os.Rename(target, backup); err != nil {
		return wrapReplaceError(err, target)
	}
	if err := os.Rename(newFile, target); err != nil {
		// 落位失败时回滚，避免留下无法启动的状态。
		_ = os.Rename(backup, target)
		return wrapReplaceError(err, target)
	}

	// 备份文件此时仍被运行中的进程占用，删除失败属正常，交由 CleanupStale 处理。
	_ = os.Remove(backup)
	return nil
}
