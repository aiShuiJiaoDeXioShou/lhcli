package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// binaryName 返回压缩包内的二进制文件名。
func binaryName(goos string) string {
	if goos == "windows" {
		return "lhcli.exe"
	}
	return "lhcli"
}

// archiveExt 返回指定平台使用的压缩包后缀，与 GoReleaser 配置保持一致。
func archiveExt(goos string) string {
	if goos == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}

// assetName 推导发布资产文件名，形如 lhcli_0.3.0_linux_amd64.tar.gz。
func assetName(tag, goos, goarch string) string {
	trimmed := strings.TrimPrefix(tag, "v")
	return fmt.Sprintf("lhcli_%s_%s_%s%s", trimmed, goos, goarch, archiveExt(goos))
}

// findAsset 在发布中定位目标资产，优先精确匹配，其次按前缀匹配。
func findAsset(release *Release, name string) (*Asset, error) {
	for i := range release.Assets {
		if release.Assets[i].Name == name {
			return &release.Assets[i], nil
		}
	}
	// 退化匹配：忽略大小写与前缀差异（例如不同的压缩后缀）。
	base := strings.TrimSuffix(strings.TrimSuffix(name, ".zip"), ".tar.gz")
	for i := range release.Assets {
		if strings.HasPrefix(strings.ToLower(release.Assets[i].Name), strings.ToLower(base)) {
			return &release.Assets[i], nil
		}
	}

	names := make([]string, 0, len(release.Assets))
	for _, asset := range release.Assets {
		names = append(names, asset.Name)
	}
	return nil, fmt.Errorf("发布 %s 中未找到资产 %s（可用：%s）",
		release.TagName, name, strings.Join(names, ", "))
}

// downloadAsset 下载资产到 dest，失败时尝试镜像前缀。
func downloadAsset(ctx context.Context, client *http.Client, url, mirror, dest string) error {
	candidates := []string{url}
	if mirror != "" {
		candidates = append(candidates, strings.TrimRight(mirror, "/")+"/"+url)
	}

	var lastErr error
	for _, candidate := range candidates {
		if err := fetchToFile(ctx, client, candidate, dest); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	return fmt.Errorf("下载 %s 失败: %w", url, lastErr)
}

// fetchToFile 下载单个 URL 并写入文件。
func fetchToFile(ctx context.Context, client *http.Client, url, dest string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "lhcli")
	if token := githubToken(); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

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

// extractBinary 从压缩包中解出二进制文件并写入 dest。
func extractBinary(archivePath, dest, goos string) error {
	if archiveExt(goos) == ".zip" {
		return extractFromZip(archivePath, dest, binaryName(goos))
	}
	return extractFromTarGz(archivePath, dest, binaryName(goos))
}

// extractFromTarGz 从 tar.gz 中提取指定文件名的条目。
func extractFromTarGz(archivePath, dest, wanted string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("解压更新包失败: %w", err)
	}
	defer gzipReader.Close()

	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取更新包失败: %w", err)
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != wanted {
			continue
		}
		return writeExecutable(dest, reader)
	}
	return fmt.Errorf("更新包中不包含 %s", wanted)
}

// extractFromZip 从 zip 中提取指定文件名的条目。
func extractFromZip(archivePath, dest, wanted string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("解压更新包失败: %w", err)
	}
	defer reader.Close()

	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() || filepath.Base(entry.Name) != wanted {
			continue
		}
		source, err := entry.Open()
		if err != nil {
			return err
		}
		defer source.Close()
		return writeExecutable(dest, source)
	}
	return fmt.Errorf("更新包中不包含 %s", wanted)
}

// writeExecutable 将内容写入具备可执行权限的文件。
func writeExecutable(dest string, source io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, source); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Chmod(dest, 0o755)
}
