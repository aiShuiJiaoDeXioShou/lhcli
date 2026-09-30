package selfupdate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRelease 描述模拟发布服务返回的内容。
type fakeRelease struct {
	tag      string
	payload  []byte
	checksum string // 为空时按 payload 计算正确校验和
}

// newUpdateServer 启动一个模拟 GitHub Releases 的测试服务。
func newUpdateServer(t *testing.T, release fakeRelease) *httptest.Server {
	t.Helper()

	archiveName := assetName(release.tag, "linux", "amd64")
	checksum := release.checksum
	if checksum == "" {
		sum := sha256.Sum256(release.payload)
		checksum = hex.EncodeToString(sum[:])
	}
	checksums := fmt.Sprintf("%s  %s\n", checksum, archiveName)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/owner/repo/releases/latest",
			r.URL.Path == "/repos/owner/repo/releases/tags/"+release.tag:
			fmt.Fprintf(w, `{"tag_name":%q,"published_at":"2025-09-30T00:00:00Z","assets":[{"name":%q,"browser_download_url":%q},{"name":"checksums.txt","browser_download_url":%q}]}`,
				release.tag, archiveName, server.URL+"/dl/archive", server.URL+"/dl/checksums")
		case r.URL.Path == "/dl/archive":
			_, _ = w.Write(release.payload)
		case r.URL.Path == "/dl/checksums":
			fmt.Fprint(w, checksums)
		default:
			http.NotFound(w, r)
		}
	}))
	return server
}

// withAPIBase 在测试期间替换 GitHub API 入口。
func withAPIBase(t *testing.T, base string) {
	t.Helper()
	previous := apiBase
	apiBase = base
	t.Cleanup(func() { apiBase = previous })
}

// newTarget 创建一个待替换的假二进制。
func newTarget(t *testing.T) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), "lhcli")
	if err := os.WriteFile(target, []byte("旧版本"), 0o755); err != nil {
		t.Fatal(err)
	}
	return target
}

// baseOptions 返回一份指向测试服务的基准参数。
func baseOptions(t *testing.T, server *httptest.Server, target string) (Options, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	return Options{
		CurrentVersion: "v0.2.0",
		AssumeYes:      true,
		Repo:           "owner/repo",
		GOOS:           "linux",
		GOARCH:         "amd64",
		Executable:     target,
		Client:         server.Client(),
		Stdout:         out,
		Stderr:         errOut,
	}, out, errOut
}

func TestRunUpdatesBinary(t *testing.T) {
	server := newUpdateServer(t, fakeRelease{tag: "v0.3.0", payload: tarGzBytes(t, "lhcli", "新版本二进制")})
	defer server.Close()
	withAPIBase(t, server.URL)

	target := newTarget(t)
	opts, out, _ := baseOptions(t, server, target)

	updated, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run 返回错误: %v", err)
	}
	if !updated {
		t.Fatal("期望完成更新，实际未更新")
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "新版本二进制" {
		t.Errorf("替换后内容 = %q", content)
	}
	if !strings.Contains(out.String(), "SHA256 校验通过") {
		t.Errorf("输出缺少校验提示: %s", out.String())
	}
}

func TestRunAlreadyLatest(t *testing.T) {
	server := newUpdateServer(t, fakeRelease{tag: "v0.3.0", payload: tarGzBytes(t, "lhcli", "新版本二进制")})
	defer server.Close()
	withAPIBase(t, server.URL)

	target := newTarget(t)
	opts, out, _ := baseOptions(t, server, target)
	opts.CurrentVersion = "v0.3.0"

	updated, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run 返回错误: %v", err)
	}
	if updated {
		t.Error("已是最新版本时不应更新")
	}
	if !strings.Contains(out.String(), "已是最新版本") {
		t.Errorf("输出缺少提示: %s", out.String())
	}
	assertUnchanged(t, target)
}

func TestRunCheckOnly(t *testing.T) {
	server := newUpdateServer(t, fakeRelease{tag: "v0.3.0", payload: tarGzBytes(t, "lhcli", "新版本二进制")})
	defer server.Close()
	withAPIBase(t, server.URL)

	target := newTarget(t)
	opts, out, _ := baseOptions(t, server, target)
	opts.CheckOnly = true

	updated, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run 返回错误: %v", err)
	}
	if updated {
		t.Error("--check 不应执行更新")
	}
	if !strings.Contains(out.String(), "发现新版本 v0.3.0") {
		t.Errorf("输出缺少提示: %s", out.String())
	}
	assertUnchanged(t, target)
}

func TestRunDevRequiresForce(t *testing.T) {
	server := newUpdateServer(t, fakeRelease{tag: "v0.3.0", payload: tarGzBytes(t, "lhcli", "新版本二进制")})
	defer server.Close()
	withAPIBase(t, server.URL)

	target := newTarget(t)
	opts, _, _ := baseOptions(t, server, target)
	opts.CurrentVersion = "dev"

	if _, err := Run(context.Background(), opts); err == nil {
		t.Fatal("dev 版本未加 --force 时期望报错，实际通过")
	} else if !strings.Contains(err.Error(), "--force") {
		t.Errorf("错误信息未提示 --force: %v", err)
	}
	assertUnchanged(t, target)
}

func TestRunDryRun(t *testing.T) {
	server := newUpdateServer(t, fakeRelease{tag: "v0.3.0", payload: tarGzBytes(t, "lhcli", "新版本二进制")})
	defer server.Close()
	withAPIBase(t, server.URL)

	target := newTarget(t)
	opts, out, _ := baseOptions(t, server, target)
	opts.DryRun = true

	updated, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run 返回错误: %v", err)
	}
	if updated {
		t.Error("--dry-run 不应执行更新")
	}
	if !strings.Contains(out.String(), "干跑完成") {
		t.Errorf("输出缺少提示: %s", out.String())
	}
	assertUnchanged(t, target)
}

func TestRunChecksumMismatch(t *testing.T) {
	server := newUpdateServer(t, fakeRelease{
		tag:      "v0.3.0",
		payload:  tarGzBytes(t, "lhcli", "被篡改的二进制"),
		checksum: strings.Repeat("0", 64),
	})
	defer server.Close()
	withAPIBase(t, server.URL)

	target := newTarget(t)
	opts, _, _ := baseOptions(t, server, target)

	if _, err := Run(context.Background(), opts); err == nil {
		t.Fatal("校验失败时期望报错，实际通过")
	}
	assertUnchanged(t, target)
}

func TestRunTargetVersion(t *testing.T) {
	server := newUpdateServer(t, fakeRelease{tag: "v0.3.0", payload: tarGzBytes(t, "lhcli", "新版本二进制")})
	defer server.Close()
	withAPIBase(t, server.URL)

	target := newTarget(t)
	opts, out, _ := baseOptions(t, server, target)
	opts.TargetVersion = "v0.3.0"

	updated, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatalf("Run 返回错误: %v", err)
	}
	if !updated {
		t.Fatal("指定版本更新失败")
	}
	if !strings.Contains(out.String(), "目标版本: v0.3.0") {
		t.Errorf("输出缺少目标版本提示: %s", out.String())
	}
}

func TestDownloadAssetMirror(t *testing.T) {
	const original = "https://example.com/asset"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+original {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, "镜像内容")
	}))
	defer server.Close()

	dest := filepath.Join(t.TempDir(), "file")
	if err := downloadAsset(context.Background(), server.Client(), original, server.URL, dest); err != nil {
		t.Fatalf("downloadAsset 返回错误: %v", err)
	}
	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "镜像内容" {
		t.Errorf("下载内容 = %q", content)
	}
}

// assertUnchanged 断言目标二进制仍为初始内容。
func assertUnchanged(t *testing.T, target string) {
	t.Helper()
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "旧版本" {
		t.Errorf("目标文件被意外修改: %q", content)
	}
}
