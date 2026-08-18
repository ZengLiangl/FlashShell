package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"FlashDock/data"
	"FlashDock/netproxy"
	"FlashDock/utils"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const updateDownloadProgressEvent = "update:download-progress"

// UpdateCheckResult 检查更新结果
type UpdateCheckResult struct {
	CurrentVersion  string                     `json:"currentVersion"`
	LatestVersion   string                     `json:"latestVersion"`
	HasUpdate       bool                       `json:"hasUpdate"`
	ReleaseName     string                     `json:"releaseName"`
	ReleaseNotes    string                     `json:"releaseNotes"`
	ReleaseURL      string                     `json:"releaseURL"`
	PublishedAt     string                     `json:"publishedAt"`
	CheckedAt       string                     `json:"checkedAt"`
	AssetName       string                     `json:"assetName,omitempty"`
	DownloadURL     string                     `json:"downloadURL,omitempty"`
	AssetSize       int64                      `json:"assetSize,omitempty"`
	DownloadSources []UpdateDownloadSourceInfo `json:"downloadSources,omitempty"`
	Downloaded      bool                       `json:"downloaded,omitempty"`
	DownloadPath    string                     `json:"downloadPath,omitempty"`
	Error           string                     `json:"error,omitempty"`
}

// UpdateDownloadSourceInfo 前端可选的下载源
type UpdateDownloadSourceInfo struct {
	Label string `json:"label"`
}

// UpdateDownloadResult 下载结果
type UpdateDownloadResult struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	FilePath       string `json:"filePath,omitempty"`
	DirPath        string `json:"dirPath,omitempty"`
	Paused         bool   `json:"paused,omitempty"`
	ReadyToInstall bool   `json:"readyToInstall,omitempty"`
	InstallLogPath string `json:"installLogPath,omitempty"`
	AutoRelaunch   bool   `json:"autoRelaunch,omitempty"`
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
		first := w.lastEmit.IsZero()
		if first || now.Sub(w.lastEmit) >= 200*time.Millisecond || (w.total > 0 && w.written >= w.total) {
			w.lastEmit = now
			w.onProgress(w.written, w.total)
		}
	}
	return n, nil
}

var (
	updateDownloadMu     sync.Mutex
	updateDownloading    bool
	updateDownloadCancel context.CancelFunc
	lastDownloadedDir    string
	updateProgressClosed bool
	updateProgressGen    uint64
)

// CheckForUpdates 查询 GitHub Releases 最新正式版并与本地版本对比。
// 公开仓库无需 Token；若配置了 Token 则可选附带，以提高 API 速率限制。
func (a *App) CheckForUpdates() *UpdateCheckResult {
	result := &UpdateCheckResult{
		CurrentVersion: formatVersionDisplay(Version),
		CheckedAt:      time.Now().Format(time.RFC3339),
	}

	url := githubReleasesLatestAPIURL()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", ProductName+"-UpdateCheck")
	if token := resolveGitHubToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := netproxy.HTTPClient(20 * time.Second)
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
		result.DownloadSources = listDownloadSourceInfos(asset.BrowserDownloadURL)
	} else if result.HasUpdate {
		result.Error = fmt.Sprintf("未找到当前平台安装包（%s/%s）", goruntime.GOOS, goruntime.GOARCH)
	}

	var stagedVersion string
	if result.HasUpdate {
		// 有新版本时始终把目标版本纳入保留集，避免异步 prune 与下载建目录竞态：
		// 若 prune 先删掉 ~/.flashshell/updates 下暂存目录，随后写 .part 会报 no such file。
		stagedVersion = result.LatestVersion
		if reusable := resolveReusableStagedUpdate(result.LatestVersion, result.AssetName); reusable != nil {
			setCurrentStaged(reusable)
			result.Downloaded = true
			result.DownloadPath = reusable.FilePath
			stagedVersion = reusable.Version
		} else {
			clearCurrentStaged()
		}
	} else {
		clearCurrentStaged()
	}
	go pruneUpdateArtifacts(stagedVersion)
	return result
}

// PauseUpdateDownload 暂停当前安装包下载，便于更换代理源后继续（保留 .part）。
func (a *App) PauseUpdateDownload() {
	updateDownloadMu.Lock()
	cancel := updateDownloadCancel
	updateDownloadMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// DownloadUpdate 按指定下载源下载当前平台安装包到更新暂存目录。
// sourceLabel 为空时使用第一个源（GitHub 直连）；下载中可调用 PauseUpdateDownload 暂停。
// 本地已有完整包则直接复用；有 .part 则断点续传。
func (a *App) DownloadUpdate(sourceLabel string) *UpdateDownloadResult {
	updateDownloadMu.Lock()
	if updateDownloading {
		updateDownloadMu.Unlock()
		return &UpdateDownloadResult{Success: false, Message: "已有下载任务进行中"}
	}
	updateDownloading = true
	ctx, cancel := context.WithCancel(context.Background())
	updateDownloadCancel = cancel
	updateDownloadMu.Unlock()
	defer func() {
		cancel()
		updateDownloadMu.Lock()
		updateDownloading = false
		updateDownloadCancel = nil
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
		a.emitUpdateDownloadProgress("error", 0, check.AssetSize, msg)
		return &UpdateDownloadResult{Success: false, Message: msg}
	}

	workspaceDir := resolveUpdateWorkspaceDir()
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		msg := "创建更新目录失败: " + err.Error()
		a.emitUpdateDownloadProgress("error", 0, check.AssetSize, msg)
		return &UpdateDownloadResult{Success: false, Message: msg}
	}
	stagedDir, err := prepareStagedDir(workspaceDir, check.LatestVersion)
	if err != nil {
		msg := "创建暂存目录失败: " + err.Error()
		a.emitUpdateDownloadProgress("error", 0, check.AssetSize, msg)
		return &UpdateDownloadResult{Success: false, Message: msg}
	}

	dest := resolveUpdateAssetPath(stagedDir, check.AssetName)
	sources := buildDownloadSources(check.DownloadURL)
	if len(sources) == 0 {
		msg := "无可用下载源"
		a.emitUpdateDownloadProgress("error", 0, check.AssetSize, msg)
		return &UpdateDownloadResult{Success: false, Message: msg}
	}

	source, ok := pickDownloadSource(sources, sourceLabel)
	if !ok {
		msg := fmt.Sprintf("未知下载源：%s", strings.TrimSpace(sourceLabel))
		a.emitUpdateDownloadProgress("error", 0, check.AssetSize, msg)
		return &UpdateDownloadResult{Success: false, Message: msg}
	}

	finishOK := func(msg string) *UpdateDownloadResult {
		staged := &stagedUpdate{
			Version:        normalizeVersion(check.LatestVersion),
			AssetName:      check.AssetName,
			FilePath:       dest,
			StagedDir:      stagedDir,
			InstallLogPath: buildUpdateInstallLogPath(workspaceDir),
		}
		setCurrentStaged(staged)
		lastDownloadedDir = filepath.Dir(dest)
		go pruneUpdateArtifacts(staged.Version)
		a.emitUpdateDownloadProgress("done", check.AssetSize, check.AssetSize, dest)
		return &UpdateDownloadResult{
			Success:        true,
			Message:        msg,
			FilePath:       dest,
			DirPath:        filepath.Dir(dest),
			ReadyToInstall: true,
			InstallLogPath: staged.InstallLogPath,
			AutoRelaunch:   true,
		}
	}

	// 本地已有完整安装包：直接复用
	if complete, size := localUpdateAssetComplete(dest, check.AssetSize); complete {
		a.emitUpdateDownloadProgress("start", size, size, "本地已有安装包，跳过下载")
		return finishOK(fmt.Sprintf("已使用本地安装包（%s）", source.Label))
	}

	resumeFrom, _ := localUpdatePartOffset(dest, check.AssetSize)
	startMsg := fmt.Sprintf("使用「%s」下载…", source.Label)
	if resumeFrom > 0 {
		startMsg = fmt.Sprintf("使用「%s」续传（已有 %s）…", source.Label, formatByteSize(resumeFrom))
	}
	a.emitUpdateDownloadProgress("start", resumeFrom, check.AssetSize, startMsg)

	err = downloadFileWithProgress(ctx, source.URL, dest, check.AssetSize, source.Direct, func(downloaded, total int64) {
		a.emitUpdateDownloadProgress("downloading", downloaded, total, "")
	})
	if err == nil {
		return finishOK(fmt.Sprintf("下载完成（%s），可安装并重启", source.Label))
	}
	if ctx.Err() != nil {
		msg := "已暂停，可更换下载源后继续"
		partial, _ := localUpdatePartOffset(dest, check.AssetSize)
		a.emitUpdateDownloadProgress("paused", partial, check.AssetSize, msg)
		return &UpdateDownloadResult{Success: false, Message: msg, Paused: true}
	}

	msg := fmt.Sprintf("下载失败（%s）: %v", source.Label, err)
	partial, _ := localUpdatePartOffset(dest, check.AssetSize)
	a.emitUpdateDownloadProgress("error", partial, check.AssetSize, msg)
	return &UpdateDownloadResult{Success: false, Message: msg}
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
		if percent < 0 {
			percent = 0
		}
	}
	payload := map[string]interface{}{
		"status":     status,
		"downloaded": downloaded,
		"total":      total,
		"percent":    percent,
		"message":    message,
	}

	updateDownloadMu.Lock()
	switch status {
	case "start":
		updateProgressGen++
		updateProgressClosed = false
	case "downloading":
		if updateProgressClosed {
			updateDownloadMu.Unlock()
			return
		}
		gen := updateProgressGen
		updateDownloadMu.Unlock()
		// 异步发进度，但带代数校验，避免 done 之后迟到的 downloading 把 UI 卡死
		go func() {
			updateDownloadMu.Lock()
			defer updateDownloadMu.Unlock()
			if updateProgressClosed || gen != updateProgressGen {
				return
			}
			wailsRuntime.EventsEmit(a.ctx, updateDownloadProgressEvent, payload)
		}()
		return
	case "done", "error", "paused":
		updateProgressClosed = true
	}
	updateDownloadMu.Unlock()
	wailsRuntime.EventsEmit(a.ctx, updateDownloadProgressEvent, payload)
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
		if !strings.HasPrefix(lower, "flashshell-") && !strings.HasPrefix(lower, "flashdock-") {
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
	return fmt.Sprintf("%s-%s-%s-%s%s", ProductName, version, osName, archName, ext)
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
	dir, err := data.ConfigHomeDir()
	if err != nil {
		return filepath.Join(".", data.ConfigHomeDirName, "skipped_update_version")
	}
	return filepath.Join(dir, "skipped_update_version")
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
				Label:  "ghfast.top",
				URL:    withGitHubProxyPrefix("https://ghfast.top", rawURL),
				Direct: false,
			},
		)
	}
	return sources
}

func listDownloadSourceInfos(rawURL string) []UpdateDownloadSourceInfo {
	sources := buildDownloadSources(rawURL)
	if len(sources) == 0 {
		return nil
	}
	out := make([]UpdateDownloadSourceInfo, 0, len(sources))
	for _, s := range sources {
		out = append(out, UpdateDownloadSourceInfo{Label: s.Label})
	}
	return out
}

func pickDownloadSource(sources []updateDownloadSource, label string) (updateDownloadSource, bool) {
	if len(sources) == 0 {
		return updateDownloadSource{}, false
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return sources[0], true
	}
	for _, s := range sources {
		if strings.EqualFold(s.Label, label) {
			return s, true
		}
	}
	return updateDownloadSource{}, false
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

// selectFastestDownloadSource 保留给兼容调用；现已改为优先直连 GitHub。
func selectFastestDownloadSource(rawURL string) updateDownloadSource {
	sources := buildDownloadSources(rawURL)
	if len(sources) == 0 {
		return updateDownloadSource{Label: "GitHub", URL: rawURL, Direct: true}
	}
	return sources[0]
}

func probeDownloadLatency(ctx context.Context, src updateDownloadSource) (time.Duration, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src.URL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", ProductName+"-Updater/"+formatVersionDisplay(Version))
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("Range", "bytes=0-0")
	if src.Direct {
		if token := resolveGitHubToken(); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	client := &http.Client{Timeout: 5 * time.Second, Transport: netproxy.HTTPTransport()}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.CopyN(io.Discard, resp.Body, 64)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return time.Since(start), nil
}

func downloadFileWithProgress(ctx context.Context, url, dest string, knownSize int64, sendGitHubAuth bool, onProgress func(downloaded, total int64)) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("创建下载目录失败: %w", err)
	}

	if complete, size := localUpdateAssetComplete(dest, knownSize); complete {
		if onProgress != nil {
			onProgress(size, size)
		}
		return nil
	}

	tmp := dest + ".part"
	startOffset, err := prepareUpdatePartFile(dest, tmp, knownSize)
	if err != nil {
		return err
	}
	if knownSize > 0 && startOffset >= knownSize {
		if err := os.Rename(tmp, dest); err != nil {
			return err
		}
		if onProgress != nil {
			onProgress(knownSize, knownSize)
		}
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", ProductName+"-Updater/"+formatVersionDisplay(Version))
	req.Header.Set("Accept", "application/octet-stream")
	if startOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startOffset))
	}
	// 仅直连 GitHub 时附带 Token；代理域名不应收到 GitHub 凭据
	if sendGitHubAuth {
		if token := resolveGitHubToken(); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	client := &http.Client{Timeout: 0, Transport: netproxy.HTTPTransport()}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch {
	case startOffset > 0 && resp.StatusCode == http.StatusOK:
		// 服务端忽略 Range，从头重下
		startOffset = 0
		_ = os.Remove(tmp)
	case startOffset > 0 && resp.StatusCode == http.StatusPartialContent:
		// 续传成功
	case startOffset == 0 && resp.StatusCode == http.StatusOK:
		// 正常整包下载
	default:
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	total := resolveDownloadTotal(knownSize, startOffset, resp)
	var out *os.File
	if startOffset > 0 {
		out, err = os.OpenFile(tmp, os.O_WRONLY|os.O_APPEND, 0o644)
	} else {
		out, err = os.Create(tmp)
	}
	if err != nil {
		return err
	}

	progressWriter := &updateDownloadProgressWriter{
		written:    startOffset,
		total:      total,
		onProgress: onProgress,
	}
	if onProgress != nil {
		onProgress(startOffset, total)
	}

	_, copyErr := utils.CopyBuffer(io.MultiWriter(out, progressWriter), resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		// 暂停/失败均保留 .part，便于下次续传
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	_ = os.Remove(dest)
	if err := os.Rename(tmp, dest); err != nil {
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

func localUpdateAssetComplete(dest string, knownSize int64) (bool, int64) {
	st, err := os.Stat(dest)
	if err != nil || st.IsDir() {
		return false, 0
	}
	size := st.Size()
	if knownSize > 0 {
		return size == knownSize, size
	}
	return size > 0, size
}

func localUpdatePartOffset(dest string, knownSize int64) (int64, error) {
	tmp := dest + ".part"
	if st, err := os.Stat(tmp); err == nil && !st.IsDir() {
		size := st.Size()
		if knownSize > 0 && size > knownSize {
			_ = os.Remove(tmp)
			return 0, nil
		}
		return size, nil
	}
	if st, err := os.Stat(dest); err == nil && !st.IsDir() {
		size := st.Size()
		if knownSize > 0 && size >= knownSize {
			return knownSize, nil
		}
		return size, nil
	}
	return 0, nil
}

func prepareUpdatePartFile(dest, tmp string, knownSize int64) (int64, error) {
	if st, err := os.Stat(tmp); err == nil && !st.IsDir() {
		size := st.Size()
		if knownSize > 0 && size > knownSize {
			_ = os.Remove(tmp)
			return 0, nil
		}
		return size, nil
	}
	// 不完整的最终文件当作续传起点
	if st, err := os.Stat(dest); err == nil && !st.IsDir() {
		size := st.Size()
		if knownSize > 0 && size == knownSize {
			return knownSize, nil
		}
		if knownSize > 0 && size > knownSize {
			_ = os.Remove(dest)
			return 0, nil
		}
		if err := os.Rename(dest, tmp); err != nil {
			return 0, err
		}
		return size, nil
	}
	return 0, nil
}

func resolveDownloadTotal(knownSize, startOffset int64, resp *http.Response) int64 {
	// 优先用 GitHub API 的资源大小；代理常返回错误/偏小的 Content-Length，会导致进度瞬间 100%
	if knownSize > 0 {
		return knownSize
	}
	if resp == nil {
		return 0
	}
	if total := parseContentRangeTotal(resp.Header.Get("Content-Range")); total > 0 {
		return total
	}
	if resp.ContentLength > 0 {
		if resp.StatusCode == http.StatusPartialContent {
			return startOffset + resp.ContentLength
		}
		return resp.ContentLength
	}
	return 0
}

func parseContentRangeTotal(h string) int64 {
	h = strings.TrimSpace(h)
	if h == "" {
		return 0
	}
	lower := strings.ToLower(h)
	if !strings.HasPrefix(lower, "bytes ") {
		return 0
	}
	rest := strings.TrimSpace(h[len("bytes "):])
	parts := strings.Split(rest, "/")
	if len(parts) != 2 {
		return 0
	}
	totalStr := strings.TrimSpace(parts[1])
	if totalStr == "*" || totalStr == "" {
		return 0
	}
	n, err := strconv.ParseInt(totalStr, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func formatByteSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	if n < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", float64(n)/(1024*1024*1024))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
