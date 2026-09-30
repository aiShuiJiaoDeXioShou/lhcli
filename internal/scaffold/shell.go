package scaffold

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// findShell 定位用于执行改名脚本的 sh。
//
// 非 Windows 平台直接使用 PATH 中的 sh；Windows 平台优先使用 PATH，
// 其次探测 Git for Windows 自带的 Git Bash。
func findShell() (string, error) {
	if path, err := exec.LookPath("sh"); err == nil {
		return path, nil
	}
	if runtime.GOOS != "windows" {
		return "", errors.New("未找到 sh，请确认系统已安装 /bin/sh")
	}

	candidates := gitBashCandidates()
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("未找到 sh，请安装 Git for Windows 后重试（需要 Git Bash）")
}

// gitBashCandidates 返回 Git for Windows 中 sh.exe 的候选路径。
func gitBashCandidates() []string {
	candidates := make([]string, 0, 8)

	if gitPath, err := exec.LookPath("git"); err == nil {
		// ...\Git\cmd\git.exe -> ...\Git
		base := filepath.Dir(filepath.Dir(gitPath))
		candidates = append(candidates,
			filepath.Join(base, "bin", "sh.exe"),
			filepath.Join(base, "usr", "bin", "sh.exe"),
		)
	}

	candidates = append(candidates,
		`C:\Program Files\Git\bin\sh.exe`,
		`C:\Program Files\Git\usr\bin\sh.exe`,
		`C:\Program Files (x86)\Git\bin\sh.exe`,
		`C:\Program Files (x86)\Git\usr\bin\sh.exe`,
	)
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		candidates = append(candidates, filepath.Join(local, "Programs", "Git", "bin", "sh.exe"))
	}
	return candidates
}
