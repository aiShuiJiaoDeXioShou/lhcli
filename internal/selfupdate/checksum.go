package selfupdate

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// checksumFileName 是 GoReleaser 生成的校验和文件名。
const checksumFileName = "checksums.txt"

// fetchChecksums 下载并解析发布目录下的校验和文件。
func fetchChecksums(ctx context.Context, client *http.Client, url, mirror string) (map[string]string, error) {
	temp, err := os.CreateTemp("", "lhcli-checksums-")
	if err != nil {
		return nil, err
	}
	path := temp.Name()
	temp.Close()
	defer os.Remove(path)

	if err := downloadAsset(ctx, client, url, mirror, path); err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseChecksums(file)
}

// parseChecksums 解析 "<sha256>  <filename>" 格式的校验和内容。
func parseChecksums(reader io.Reader) (map[string]string, error) {
	sums := make(map[string]string)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		sums[name] = strings.ToLower(fields[0])
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(sums) == 0 {
		return nil, fmt.Errorf("校验和文件为空或格式不正确")
	}
	return sums, nil
}

// verifyChecksum 校验文件的实际 SHA256 是否与期望值一致。
func verifyChecksum(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("SHA256 校验失败: 期望 %s，实际 %s", expected, actual)
	}
	return nil
}
