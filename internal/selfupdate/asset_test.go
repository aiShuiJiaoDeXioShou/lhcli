package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAssetName(t *testing.T) {
	cases := []struct {
		tag    string
		goos   string
		goarch string
		want   string
	}{
		{"v0.3.0", "linux", "amd64", "lhcli_0.3.0_linux_amd64.tar.gz"},
		{"v0.3.0", "darwin", "arm64", "lhcli_0.3.0_darwin_arm64.tar.gz"},
		{"v0.3.0", "windows", "amd64", "lhcli_0.3.0_windows_amd64.zip"},
		{"0.3.0", "linux", "arm64", "lhcli_0.3.0_linux_arm64.tar.gz"},
	}

	for _, tc := range cases {
		if got := assetName(tc.tag, tc.goos, tc.goarch); got != tc.want {
			t.Errorf("assetName(%q, %q, %q) = %q，期望 %q", tc.tag, tc.goos, tc.goarch, got, tc.want)
		}
	}
}

func TestFindAsset(t *testing.T) {
	release := &Release{
		TagName: "v0.3.0",
		Assets: []Asset{
			{Name: "checksums.txt", URL: "https://example.com/checksums.txt"},
			{Name: "lhcli_0.3.0_windows_amd64.zip", URL: "https://example.com/win.zip"},
		},
	}

	asset, err := findAsset(release, "lhcli_0.3.0_windows_amd64.zip")
	if err != nil {
		t.Fatalf("findAsset 返回错误: %v", err)
	}
	if asset.URL != "https://example.com/win.zip" {
		t.Errorf("findAsset 命中错误的资产: %+v", asset)
	}

	if _, err := findAsset(release, "lhcli_0.3.0_linux_amd64.tar.gz"); err == nil {
		t.Error("缺失资产时 findAsset 期望报错，实际通过")
	}
}

func TestExtractFromTarGz(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(path, tarGzBytes(t, "lhcli", "新二进制内容"), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "lhcli")
	if err := extractBinary(path, dest, "linux"); err != nil {
		t.Fatalf("extractBinary 返回错误: %v", err)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "新二进制内容" {
		t.Errorf("解出的内容 = %q", content)
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	// Windows 上 chmod 只控制只读位，不体现可执行位，故仅在类 Unix 平台断言。
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o100 == 0 {
		t.Errorf("解出的文件缺少可执行权限: %v", info.Mode())
	}
}

func TestExtractFromZip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.zip")
	if err := os.WriteFile(path, zipBytes(t, "lhcli.exe", "windows 二进制"), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "lhcli.exe")
	if err := extractBinary(path, dest, "windows"); err != nil {
		t.Fatalf("extractBinary 返回错误: %v", err)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "windows 二进制" {
		t.Errorf("解出的内容 = %q", content)
	}
}

// tarGzBytes 构造一个包含单个文件的 tar.gz 字节流。
func tarGzBytes(t *testing.T, name, content string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	payload := []byte(content)
	if err := tarWriter.WriteHeader(&tar.Header{
		Name:     name,
		Mode:     0o755,
		Size:     int64(len(payload)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

// zipBytes 构造一个包含单个文件的 zip 字节流。
func zipBytes(t *testing.T, name, content string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
