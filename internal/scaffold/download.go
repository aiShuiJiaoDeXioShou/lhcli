package scaffold

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// codeloadBase 是 GitHub 的源码归档下载入口。
const codeloadBase = "https://codeload.github.com"

// downloadTimeout 是单次模板下载的整体超时时间。
const downloadTimeout = 5 * time.Minute

// tarballURL 拼接模板仓库的 tarball 下载地址。
func tarballURL(repo, ref string) string {
	return fmt.Sprintf("%s/%s/tar.gz/%s", codeloadBase, repo, ref)
}

// downloadTarball 下载模板 tarball 到 dest；失败时尝试镜像前缀。
func downloadTarball(ctx context.Context, repo, ref, mirror, dest string) error {
	url := tarballURL(repo, ref)
	candidates := []string{url}
	if mirror != "" {
		candidates = append(candidates, strings.TrimRight(mirror, "/")+"/"+url)
	}

	var lastErr error
	for _, candidate := range candidates {
		if err := fetchToFile(ctx, candidate, dest); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("下载模板失败: %w", lastErr)
}

// fetchToFile 下载单个 URL 并写入文件。
func fetchToFile(ctx context.Context, url, dest string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "lhcli")

	client := &http.Client{Timeout: downloadTimeout}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s 返回 HTTP %d", url, response.StatusCode)
	}

	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, response.Body); err != nil {
		return err
	}
	return file.Close()
}
