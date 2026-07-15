package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"context"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const updateDownloadProgressEvent = "update:download-progress"

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
	AssetName      string `json:"assetName,omitempty"`
	DownloadURL    string `json:"downloadURL,omitempty"`
	AssetSize      int64  `json:"assetSize,omitempty"`
	Error          string `json:"error,omitempty"`
}

// UpdateDownloadResult 下载结果
type UpdateDownloadResult struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	FilePath string `json:"filePath,omitempty"`
	DirPath  string `json:"dirPath,omitempty"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	Prerelease  bool          `json:"prerelease"`
	Draft       bool          `json:"draft"`
	PublishedAt string        `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type updateDownloadProgressWriter struct {
	written    int64
	total      int64
	lastEmit   time.Time
	onProgress func(downloaded, total int64)
}

func (w *updateDownloadProgressWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.written += int64(n)
	if w.onProgress != nil {
		now := time.Now()
		if now.Sub(w.lastEmit) >= 120*time.Millisecond || w.written >= w.total {
			w.lastEmit = now
			w.onProgress(w.written, w.total)
		}
	}
	return n, nil
}

var (
	updateDownloadMu  sync.Mutex
	updateDownloading bool
	lastDownloadedDir string
)

// CheckForUpdates 查询 GitHub Releases 最新正式版并与本地版本对比。
// 公开仓库无需 Token；若配置了 Token 则可选附带，以提高 API 速率限制。
func (a *App) CheckForUpdates() *UpdateCheckResult {
	result := &UpdateCheckResult{
		CurrentVersion: formatVersionDisplay(Version),
		CheckedAt:      time.Now().Format(time.RFC3339),
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "FlashDock-UpdateCheck")
	if token := resolveGitHubToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = "网络请求失败: " + err.Error()
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
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

	if asset := pickReleaseAsset(rel.Assets, rel.TagName); asset != nil {
		result.AssetName = asset.Name
		result.DownloadURL = asset.BrowserDownloadURL
		result.AssetSize = asset.Size
	} else if result.HasUpdate {
		result.Error = fmt.Sprintf("未找到当前平台安装包（%s/%s）", goruntime.GOOS, goruntime.GOARCH)
	}
	return result
}

// DownloadUpdate 下载当前平台对应的安装包到用户「下载」目录，并打开该目录。
func (a *App) DownloadUpdate() *UpdateDownloadResult {
	updateDownloadMu.Lock()
	if updateDownloading {
		updateDownloadMu.Unlock()
		return &UpdateDownloadResult{Success: false, Message: "已有下载任务进行中"}
	}
	updateDownloading = true
	updateDownloadMu.Unlock()
	defer func() {
		updateDownloadMu.Lock()
		updateDownloading = false
		updateDownloadMu.Unlock()
	}()

	check := a.CheckForUpdates()
	if check.Error != "" && check.DownloadURL == "" {
		a.emitUpdateDownloadProgress("error", 0, 0, check.Error)
		return &UpdateDownloadResult{Success: false, Message: check.Error}
	}
	if !check.HasUpdate {
		msg := "已是最新版本"
		a.emitUpdateDownloadProgress("error", 0, 0, msg)
		return &UpdateDownloadResult{Success: false, Message: msg}
	}
	if check.DownloadURL == "" || check.AssetName == "" {
		msg := "当前平台暂无可用安装包"
		a.emitUpdateDownloadProgress("error", 0, 0, msg)
		return &UpdateDownloadResult{Success: false, Message: msg}
	}

	dir, err := resolveDownloadsDir()
	if err != nil {
		msg := "无法定位下载目录: " + err.Error()
		a.emitUpdateDownloadProgress("error", 0, 0, msg)
		return &UpdateDownloadResult{Success: false, Message: msg}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		msg := "创建下载目录失败: " + err.Error()
		a.emitUpdateDownloadProgress("error", 0, check.AssetSize, msg)
		return &UpdateDownloadResult{Success: false, Message: msg}
	}

	dest := filepath.Join(dir, check.AssetName)
	a.emitUpdateDownloadProgress("start", 0, check.AssetSize, "正在测速选取最快下载源…")

	source := selectFastestDownloadSource(check.DownloadURL)
	a.emitUpdateDownloadProgress("start", 0, check.AssetSize, fmt.Sprintf("使用「%s」下载…", source.Label))

	if err := downloadFileWithProgress(source.URL, dest, check.AssetSize, source.Direct, func(downloaded, total int64) {
		a.emitUpdateDownloadProgress("downloading", downloaded, total, "")
	}); err != nil {
		_ = os.Remove(dest)
		msg := fmt.Sprintf("下载失败（%s）: %v", source.Label, err)
		// 按测速外的其余源依次兜底
		for _, fallback := range buildDownloadSources(check.DownloadURL) {
			if fallback.URL == source.URL {
				continue
			}
			a.emitUpdateDownloadProgress("start", 0, check.AssetSize, fmt.Sprintf("%s 失败，改用「%s」…", source.Label, fallback.Label))
			_ = os.Remove(dest)
			err2 := downloadFileWithProgress(fallback.URL, dest, check.AssetSize, fallback.Direct, func(downloaded, total int64) {
				a.emitUpdateDownloadProgress("downloading", downloaded, total, "")
			})
			if err2 == nil {
				lastDownloadedDir = dir
				a.emitUpdateDownloadProgress("done", check.AssetSize, check.AssetSize, dest)
				_ = a.OpenDownloadsDirectory()
				return &UpdateDownloadResult{
					Success:  true,
					Message:  fmt.Sprintf("下载完成（%s）: %s", fallback.Label, dest),
					FilePath: dest,
					DirPath:  dir,
				}
			}
			msg = fmt.Sprintf("%s；%s 失败: %v", msg, fallback.Label, err2)
		}
		a.emitUpdateDownloadProgress("error", 0, check.AssetSize, msg)
		return &UpdateDownloadResult{Success: false, Message: msg}
	}

	lastDownloadedDir = dir
	a.emitUpdateDownloadProgress("done", check.AssetSize, check.AssetSize, dest)
	_ = a.OpenDownloadsDirectory()
	return &UpdateDownloadResult{
		Success:  true,
		Message:  fmt.Sprintf("下载完成（%s）: %s", source.Label, dest),
		FilePath: dest,
		DirPath:  dir,
	}
}

// OpenDownloadsDirectory 打开用户下载目录（优先最近一次下载目录）
func (a *App) OpenDownloadsDirectory() error {
	dir := strings.TrimSpace(lastDownloadedDir)
	if dir == "" {
		var err error
		dir, err = resolveDownloadsDir()
		if err != nil {
			return err
		}
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("下载目录不可用: %s", dir)
	}

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("打开下载目录失败: %w", err)
	}
	return nil
}

// SkipUpdateVersion 跳过指定版本的更新提示
func (a *App) SkipUpdateVersion(version string) error {
	version = formatVersionDisplay(version)
	if version == "" || version == "v0.0.0" {
		return fmt.Errorf("版本号无效")
	}
	path := skippedUpdateVersionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(version+"\n"), 0o644)
}

// GetSkippedUpdateVersion 读取已跳过的版本
func (a *App) GetSkippedUpdateVersion() string {
	b, err := os.ReadFile(skippedUpdateVersionPath())
	if err != nil {
		return ""
	}
	return formatVersionDisplay(strings.TrimSpace(string(b)))
}

// OpenReleaseURL 在系统浏览器打开 Release 页面
func (a *App) OpenReleaseURL(url string) {
	url = strings.TrimSpace(url)
	if url == "" || a.ctx == nil {
		return
	}
	if !strings.HasPrefix(url, "https://github.com/") {
		return
	}
	wailsRuntime.BrowserOpenURL(a.ctx, url)
}

func (a *App) emitUpdateDownloadProgress(status string, downloaded, total int64, message string) {
	if a == nil || a.ctx == nil {
		return
	}
	percent := 0
	if total > 0 {
		percent = int(float64(downloaded) * 100 / float64(total))
		if percent > 100 {
			percent = 100
		}
	}
	wailsRuntime.EventsEmit(a.ctx, updateDownloadProgressEvent, map[string]interface{}{
		"status":     status,
		"downloaded": downloaded,
		"total":      total,
		"percent":    percent,
		"message":    message,
	})
}

func pickReleaseAsset(assets []githubAsset, tagOrVersion string) *githubAsset {
	want := expectedReleaseAssetName(tagOrVersion)
	if want == "" {
		return nil
	}
	for i := range assets {
		if strings.EqualFold(assets[i].Name, want) {
			return &assets[i]
		}
	}
	// 宽松匹配：前缀 + 平台关键字
	osKey, archKey, ext := platformAssetHints()
	var fallback *githubAsset
	for i := range assets {
		name := assets[i].Name
		lower := strings.ToLower(name)
		if !strings.HasPrefix(lower, "flashdock-") {
			continue
		}
		if osKey != "" && !strings.Contains(lower, strings.ToLower(osKey)) {
			continue
		}
		if archKey != "" && !strings.Contains(lower, strings.ToLower(archKey)) {
			continue
		}
		if ext != "" && !strings.HasSuffix(lower, strings.ToLower(ext)) {
			continue
		}
		fallback = &assets[i]
		break
	}
	return fallback
}

func expectedReleaseAssetName(tagOrVersion string) string {
	version := normalizeVersion(tagOrVersion)
	if version == "" {
		return ""
	}
	osName, archName, ext := platformAssetHints()
	if osName == "" || archName == "" || ext == "" {
		return ""
	}
	return fmt.Sprintf("FlashDock-%s-%s-%s%s", version, osName, archName, ext)
}

func platformAssetHints() (osName, archName, ext string) {
	switch goruntime.GOOS {
	case "windows":
		osName = "Windows"
		ext = ".exe"
	case "darwin":
		osName = "MacOS"
		ext = ".dmg"
	case "linux":
		osName = "Linux"
		ext = ".tar.gz"
	default:
		return "", "", ""
	}
	switch goruntime.GOARCH {
	case "amd64":
		archName = "Amd64"
	case "arm64":
		archName = "Arm64"
	default:
		return "", "", ""
	}
	return osName, archName, ext
}

func resolveDownloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "downloads"),
	}
	if goruntime.GOOS == "windows" {
		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			candidates = append([]string{filepath.Join(userProfile, "Downloads")}, candidates...)
		}
	}
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, nil
		}
	}
	// 不存在则创建标准 Downloads
	dir := filepath.Join(home, "Downloads")
	return dir, nil
}

func skippedUpdateVersionPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".flashdock", "skipped_update_version")
	}
	return filepath.Join(home, ".flashdock", "skipped_update_version")
}

type updateDownloadSource struct {
	Label  string
	URL    string
	Direct bool // 直连 GitHub 时可附带 Token
}

func buildDownloadSources(rawURL string) []updateDownloadSource {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	sources := []updateDownloadSource{
		{Label: "GitHub", URL: rawURL, Direct: true},
	}
	if isGitHubAssetURL(rawURL) {
		sources = append(sources,
			updateDownloadSource{
				Label:  "ghproxy.net",
				URL:    withGitHubProxyPrefix("https://ghproxy.net", rawURL),
				Direct: false,
			},
			updateDownloadSource{
				Label:  "gitclone.com",
				URL:    withGitCloneMirror(rawURL),
				Direct: false,
			},
			updateDownloadSource{
				Label:  "githubproxy.cc",
				URL:    withGitHubProxyPrefix("https://githubproxy.cc", rawURL),
				Direct: false,
			},
			updateDownloadSource{
				Label:  "github.abskoop.workers.dev",
				URL:    withGitHubProxyPrefix("https://github.abskoop.workers.dev", rawURL),
				Direct: false,
			},
		)
	}
	return sources
}

func isGitHubAssetURL(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(lower, "https://github.com/") || strings.HasPrefix(lower, "http://github.com/")
}

func withGitHubProxyPrefix(proxyBase, raw string) string {
	proxyBase = strings.TrimRight(strings.TrimSpace(proxyBase), "/")
	raw = strings.TrimSpace(raw)
	if proxyBase == "" || raw == "" {
		return raw
	}
	prefix := proxyBase + "/"
	if strings.HasPrefix(raw, prefix) {
		return raw
	}
	return prefix + raw
}

// withGitCloneMirror 按 gitclone.com 习惯：https://gitclone.com/github.com/...
// 见 https://gitclone.com/
func withGitCloneMirror(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	trimmed := strings.TrimPrefix(raw, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	if !strings.HasPrefix(strings.ToLower(trimmed), "github.com/") {
		return raw
	}
	return "https://gitclone.com/" + trimmed
}

// selectFastestDownloadSource 并发探测各源首字节延迟，选最快可用源。
func selectFastestDownloadSource(rawURL string) updateDownloadSource {
	sources := buildDownloadSources(rawURL)
	if len(sources) == 0 {
		return updateDownloadSource{Label: "GitHub", URL: rawURL, Direct: true}
	}
	if len(sources) == 1 {
		return sources[0]
	}

	type probeResult struct {
		src updateDownloadSource
		d   time.Duration
		err error
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch := make(chan probeResult, len(sources))
	for _, src := range sources {
		go func(s updateDownloadSource) {
			d, err := probeDownloadLatency(ctx, s)
			ch <- probeResult{src: s, d: d, err: err}
		}(src)
	}

	var best *updateDownloadSource
	var bestLatency time.Duration
	for i := 0; i < len(sources); i++ {
		r := <-ch
		if r.err != nil {
			continue
		}
		if best == nil || r.d < bestLatency {
			cp := r.src
			best = &cp
			bestLatency = r.d
		}
	}
	if best != nil {
		return *best
	}
	return sources[0]
}

func probeDownloadLatency(ctx context.Context, src updateDownloadSource) (time.Duration, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "FlashDock-Updater/"+formatVersionDisplay(Version))
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("Range", "bytes=0-0")
	if src.Direct {
		if token := resolveGitHubToken(); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	// 丢弃极少数据，确认链路可用
	_, _ = io.CopyN(io.Discard, resp.Body, 64)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return time.Since(start), nil
}

func downloadFileWithProgress(url, dest string, knownSize int64, sendGitHubAuth bool, onProgress func(downloaded, total int64)) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "FlashDock-Updater/"+formatVersionDisplay(Version))
	req.Header.Set("Accept", "application/octet-stream")
	// 仅直连 GitHub 时附带 Token；代理域名不应收到 GitHub 凭据
	if sendGitHubAuth {
		if token := resolveGitHubToken(); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	total := knownSize
	if resp.ContentLength > 0 {
		total = resp.ContentLength
	}

	tmp := dest + ".part"
	_ = os.Remove(tmp)
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}

	progressWriter := &updateDownloadProgressWriter{
		total:      total,
		onProgress: onProgress,
	}
	_, copyErr := io.Copy(io.MultiWriter(out, progressWriter), resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	_ = os.Remove(dest)
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if onProgress != nil {
		finalTotal := total
		if finalTotal <= 0 {
			finalTotal = progressWriter.written
		}
		onProgress(progressWriter.written, finalTotal)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
