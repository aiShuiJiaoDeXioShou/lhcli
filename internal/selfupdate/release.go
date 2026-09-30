package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// DefaultRepo 是 lhcli 自身的发布仓库。
const DefaultRepo = "aiShuiJiaoDeXioShou/lhcli"

// apiBase 是 GitHub REST API 入口；声明为变量以便测试替换。
var apiBase = "https://api.github.com"

// requestTimeout 是单次 API 请求或下载的整体超时时间。
const requestTimeout = 5 * time.Minute

// Release 描述一次发布中自更新关心的字段。
type Release struct {
	TagName     string  `json:"tag_name"`
	PublishedAt string  `json:"published_at"`
	Assets      []Asset `json:"assets"`
}

// Asset 描述发布中的一个附件。
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// fetchRelease 查询指定版本或最新版本的发布信息。
// 当 tag 为空时查询 latest，否则查询对应 tag。
func fetchRelease(ctx context.Context, client *http.Client, repo, tag string) (*Release, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/releases/latest", apiBase, repo)
	if tag != "" {
		endpoint = fmt.Sprintf("%s/repos/%s/releases/tags/%s", apiBase, repo, tag)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "lhcli")
	request.Header.Set("Accept", "application/vnd.github+json")
	if token := githubToken(); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		hint := "请检查网络，或设置 GITHUB_TOKEN 以提升 API 限额"
		switch {
		case response.StatusCode == http.StatusNotFound && tag != "":
			hint = fmt.Sprintf("未找到版本 %s，请确认 tag 是否存在", tag)
		case response.StatusCode == http.StatusNotFound:
			hint = "该仓库还没有发布任何 Release"
		case response.StatusCode == http.StatusForbidden:
			hint = "API 可能已限流，请设置 GITHUB_TOKEN 或稍后重试"
		}
		return nil, fmt.Errorf("查询发布信息失败: HTTP %d: %s（%s）",
			response.StatusCode, strings.TrimSpace(string(body)), hint)
	}

	var release Release
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("解析发布信息失败: %w", err)
	}
	if release.TagName == "" {
		return nil, fmt.Errorf("发布信息缺少 tag_name: %s", endpoint)
	}
	return &release, nil
}

// githubToken 从环境变量读取可选的 GitHub 令牌。
func githubToken() string {
	for _, key := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}
