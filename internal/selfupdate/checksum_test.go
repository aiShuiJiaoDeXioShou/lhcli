package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseChecksums(t *testing.T) {
	input := strings.Join([]string{
		"# 注释行",
		"",
		"abc123  lhcli_0.3.0_linux_amd64.tar.gz",
		"DEF456 *lhcli_0.3.0_windows_amd64.zip",
		"缺少文件名",
	}, "\n")

	sums, err := parseChecksums(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseChecksums 返回错误: %v", err)
	}
	if sums["lhcli_0.3.0_linux_amd64.tar.gz"] != "abc123" {
		t.Errorf("解析结果不符: %+v", sums)
	}
	if sums["lhcli_0.3.0_windows_amd64.zip"] != "def456" {
		t.Errorf("未去掉 * 前缀或未转小写: %+v", sums)
	}
}

func TestParseChecksumsEmpty(t *testing.T) {
	if _, err := parseChecksums(strings.NewReader("\n# 只有注释\n")); err == nil {
		t.Error("空校验和期望报错，实际通过")
	}
}

func TestVerifyChecksum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact")
	content := []byte("待校验内容")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	expected := hex.EncodeToString(sum[:])

	if err := verifyChecksum(path, expected); err != nil {
		t.Fatalf("校验通过的文件却报错: %v", err)
	}
	if err := verifyChecksum(path, strings.Repeat("0", 64)); err == nil {
		t.Error("校验和不匹配时期望报错，实际通过")
	}
}
