package scaffold

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// tarEntry 描述一个用于测试的 tar 条目。
type tarEntry struct {
	name     string
	mode     int64
	typeflag byte
	linkname string
	body     string
}

// writeTarGz 按给定条目生成 tar.gz 文件。
func writeTarGz(t *testing.T, path string, entries []tarEntry) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("创建归档失败: %v", err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.name,
			Mode:     entry.mode,
			Typeflag: entry.typeflag,
			Linkname: entry.linkname,
			Size:     int64(len(entry.body)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("写入头失败: %v", err)
		}
		if _, err := tarWriter.Write([]byte(entry.body)); err != nil {
			t.Fatalf("写入内容失败: %v", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("关闭 tar 失败: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("关闭 gzip 失败: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("关闭文件失败: %v", err)
	}
}

// TestExtractTarGz 校验解压、可执行位与符号链接处理。
func TestExtractTarGz(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "template.tar.gz")
	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	writeTarGz(t, archive, []tarEntry{
		{name: "repo-main/", mode: 0o755, typeflag: tar.TypeDir},
		{name: "repo-main/scripts/", mode: 0o755, typeflag: tar.TypeDir},
		{name: "repo-main/scripts/rename.sh", mode: 0o755, typeflag: tar.TypeReg, body: "#!/bin/sh\n"},
		{name: "repo-main/.agents/note.txt", mode: 0o644, typeflag: tar.TypeReg, body: "hello"},
		{name: "repo-main/.claude", mode: 0o777, typeflag: tar.TypeSymlink, linkname: ".agents"},
	})

	if err := extractTarGz(archive, dest); err != nil {
		t.Fatalf("解压失败: %v", err)
	}

	// 顶层目录应被剥离，可执行位应保留（Windows 无 Unix 权限位，跳过）。
	scriptInfo, err := os.Stat(filepath.Join(dest, "scripts", "rename.sh"))
	if err != nil {
		t.Fatalf("脚本不存在: %v", err)
	}
	if runtime.GOOS != "windows" && scriptInfo.Mode().Perm()&0o100 == 0 {
		t.Fatalf("脚本缺少可执行位: %v", scriptInfo.Mode())
	}

	// 普通文件内容应正确。
	content, err := os.ReadFile(filepath.Join(dest, ".agents", "note.txt"))
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("内容不符: %q", content)
	}

	// 符号链接应存在（Windows 无权限时会退化为副本）。
	if _, err := os.Lstat(filepath.Join(dest, ".claude")); err != nil {
		t.Fatalf("符号链接或副本不存在: %v", err)
	}
}

// TestExtractTarGzRejectsTraversal 校验路径穿越会被拒绝。
func TestExtractTarGzRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "evil.tar.gz")
	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	writeTarGz(t, archive, []tarEntry{
		{name: "repo-main/../../evil.txt", mode: 0o644, typeflag: tar.TypeReg, body: "boom"},
	})

	if err := extractTarGz(archive, dest); err == nil {
		t.Fatal("期望路径穿越被拒绝，实际未报错")
	}
}

// TestStripFirstComponent 校验顶层目录剥离。
func TestStripFirstComponent(t *testing.T) {
	cases := map[string]string{
		"repo-main/a/b.txt": "a/b.txt",
		"./repo-main/a":     "a",
		"repo-main":         "",
	}
	for input, want := range cases {
		if got := stripFirstComponent(input); got != want {
			t.Fatalf("stripFirstComponent(%q) = %q，期望 %q", input, got, want)
		}
	}
}
