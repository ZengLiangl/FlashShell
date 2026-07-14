package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UpdateCheckResult 检查更新结果
type UpdateCheckResult struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	ReleaseName    string `json:"releaseName"`
	ReleaseNotes   string `json:"releaseNotes"`
	ReleaseURL     string `json:"releaseURL"`
	PublishedAt    string `json:"publishedAt"`
	CheckedAt      string `json:"checkedAt"`
	Error          string `json:"error,omitempty"`
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	Prerelease  bool   `json:"prerelease"`
	Draft       bool   `json:"draft"`
	PublishedAt string `json:"published_at"`
}

// CheckForUpdates 查询 GitHub Releases 最新正式版并与本地版本对比
func (a *App) CheckForUpdates() *UpdateCheckResult {
	result := &UpdateCheckResult{
		CurrentVersion: formatVersionDisplay(Version),
		CheckedAt:      time.Now().Format(time.RFC3339),
	}

	token := resolveGitHubToken()
	if token == "" {
		result.Error = "未配置 GitHub Token。请设置环境变量 FLASHDOCK_GITHUB_TOKEN，或在项目 secrets/github_pat 写入 PAT（私有仓库必需）"
		return result
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "FlashDock-UpdateCheck")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = "网络请求失败: " + err.Error()
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		result.Error = fmt.Sprintf("GitHub API %d: %s", resp.StatusCode, truncate(msg, 240))
		return result
	}

	var rel githubRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		result.Error = "解析 Release 失败: " + err.Error()
		return result
	}
	if rel.Draft || rel.Prerelease {
		result.Error = "最新 Release 为草稿或预发布，已忽略"
		return result
	}

	result.LatestVersion = formatVersionDisplay(rel.TagName)
	result.ReleaseName = rel.Name
	if result.ReleaseName == "" {
		result.ReleaseName = result.LatestVersion
	}
	result.ReleaseNotes = strings.TrimSpace(rel.Body)
	result.ReleaseURL = rel.HTMLURL
	result.PublishedAt = rel.PublishedAt
	result.HasUpdate = compareSemver(rel.TagName, Version) > 0
	return result
}

// OpenReleaseURL 在系统浏览器打开 Release 页面（需已登录 GitHub 才能访问私有仓）
func (a *App) OpenReleaseURL(url string) {
	url = strings.TrimSpace(url)
	if url == "" || a.ctx == nil {
		return
	}
	if !strings.HasPrefix(url, "https://github.com/") {
		return
	}
	runtime.BrowserOpenURL(a.ctx, url)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
