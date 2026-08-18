package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"FlashDock/data"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// UpdateInstallResult 安装并重启结果
type UpdateInstallResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	LogPath string `json:"logPath,omitempty"`
}

type stagedUpdate struct {
	Version        string
	AssetName      string
	FilePath       string
	StagedDir      string
	InstallLogPath string
}

var (
	stagedUpdateMu sync.Mutex
	currentStaged  *stagedUpdate
)

// InstallUpdateAndRestart 启动外部更新脚本，替换当前安装并自动打开新版本。
func (a *App) InstallUpdateAndRestart() *UpdateInstallResult {
	stagedUpdateMu.Lock()
	staged := currentStaged
	if staged != nil && strings.TrimSpace(staged.InstallLogPath) == "" {
		staged.InstallLogPath = buildUpdateInstallLogPath(filepath.Dir(staged.FilePath))
	}
	stagedUpdateMu.Unlock()

	if staged == nil {
		return &UpdateInstallResult{Success: false, Message: "尚未下载安装包，请先下载"}
	}
	if _, err := os.Stat(staged.FilePath); err != nil {
		return &UpdateInstallResult{Success: false, Message: "安装包不存在，请重新下载"}
	}

	if err := launchUpdateScript(staged); err != nil {
		msg := "启动安装失败: " + err.Error()
		if staged.InstallLogPath != "" {
			msg = fmt.Sprintf("%s（日志：%s）", msg, staged.InstallLogPath)
		}
		return &UpdateInstallResult{Success: false, Message: msg, LogPath: staged.InstallLogPath}
	}

	// 与 PinkHunkDB 一致：脚本已脱离启动后，宿主自行退出。
	// FlashDock 有退出确认框，安装更新时必须放行，否则脚本会一直等到超时。
	if a != nil {
		a.quitMu.Lock()
		a.allowQuit = true
		a.quitMu.Unlock()
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		if a != nil && a.ctx != nil {
			wailsRuntime.Quit(a.ctx)
		}
	}()

	msg := "正在安装并重启…"
	if staged.InstallLogPath != "" {
		msg = fmt.Sprintf("正在安装并重启…（日志：%s）", staged.InstallLogPath)
	}
	return &UpdateInstallResult{Success: true, Message: msg, LogPath: staged.InstallLogPath}
}

// OpenDownloadedUpdatePackage 在文件管理器中定位已下载的安装包
func (a *App) OpenDownloadedUpdatePackage() error {
	stagedUpdateMu.Lock()
	staged := currentStaged
	stagedUpdateMu.Unlock()
	if staged == nil || strings.TrimSpace(staged.FilePath) == "" {
		return fmt.Errorf("尚未下载安装包")
	}
	assetPath := staged.FilePath
	if info, err := os.Stat(assetPath); err != nil || info.IsDir() {
		return fmt.Errorf("安装包不可用")
	}
	dirPath := filepath.Dir(assetPath)

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", assetPath)
	case "windows":
		cmd = exec.Command("explorer", "/select,", filepath.Clean(assetPath))
	default:
		cmd = exec.Command("xdg-open", dirPath)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("打开安装包位置失败: %w", err)
	}
	return nil
}

func setCurrentStaged(staged *stagedUpdate) {
	stagedUpdateMu.Lock()
	currentStaged = staged
	stagedUpdateMu.Unlock()
}

func getCurrentStaged() *stagedUpdate {
	stagedUpdateMu.Lock()
	defer stagedUpdateMu.Unlock()
	return currentStaged
}

func clearCurrentStaged() {
	setCurrentStaged(nil)
}

func pruneUpdateArtifacts(stagedVersion string) {
	pruneHistoricalUpdateArtifacts(Version, stagedVersion)
}

func buildUpdateInstallLogPath(baseDir string) string {
	logDir := strings.TrimSpace(baseDir)
	if logDir == "" {
		logDir = os.TempDir()
	}
	return filepath.Join(logDir, "update-install.log")
}

func sanitizeVersionForPath(version string) string {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return "latest"
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range trimmed {
		isAllowed := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-'
		if isAllowed {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "latest"
	}
	return result
}

var resolveLegacyUpdateWorkspaceDir = func() string {
	return filepath.Join(os.TempDir(), "flashdock-updates")
}

// resolveUpdateWorkspaceRoot 返回 ~/.flashshell/updates（与全局配置同级）
var resolveUpdateWorkspaceRoot = func() string {
	home, err := data.ConfigHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, "updates")
}

func resolveUpdateWorkspaceDir() string {
	if root := strings.TrimSpace(resolveUpdateWorkspaceRoot()); root != "" {
		return root
	}
	return resolveLegacyUpdateWorkspaceDir()
}

func resolveUpdateAssetPath(stagedDir, assetName string) string {
	return filepath.Join(stagedDir, strings.TrimSpace(assetName))
}

func prepareStagedDir(workspaceDir, version string) (string, error) {
	// 保留已有暂存目录与 .part，支持断点续传；勿 RemoveAll
	stagedDir := filepath.Join(workspaceDir, fmt.Sprintf(".flashdock-update-%s-%s", goruntime.GOOS, sanitizeVersionForPath(version)))
	if err := os.MkdirAll(stagedDir, 0o755); err != nil {
		return "", err
	}
	return stagedDir, nil
}

func resolveReusableStagedUpdate(latestVersion, assetName string) *stagedUpdate {
	version := normalizeVersion(latestVersion)
	assetName = strings.TrimSpace(assetName)
	if version == "" || assetName == "" {
		return nil
	}
	if staged := getCurrentStaged(); staged != nil {
		if normalizeVersion(staged.Version) == version && fileExists(staged.FilePath) {
			return staged
		}
	}

	workspaceDir := resolveUpdateWorkspaceDir()
	stagedDirName := fmt.Sprintf(".flashdock-update-%s-%s", goruntime.GOOS, sanitizeVersionForPath(version))
	stagedDir := filepath.Join(workspaceDir, stagedDirName)
	assetPath := resolveUpdateAssetPath(stagedDir, assetName)
	if !fileExists(assetPath) {
		legacy := filepath.Join(stagedDir, assetName)
		if fileExists(legacy) {
			assetPath = legacy
		} else {
			return nil
		}
	}
	return &stagedUpdate{
		Version:        version,
		AssetName:      assetName,
		FilePath:       assetPath,
		StagedDir:      stagedDir,
		InstallLogPath: buildUpdateInstallLogPath(workspaceDir),
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(strings.TrimSpace(path))
	return err == nil && !info.IsDir()
}

func launchUpdateScript(staged *stagedUpdate) error {
	pid := os.Getpid()
	switch goruntime.GOOS {
	case "windows":
		exePath, err := resolveWindowsUpdateTarget()
		if err != nil {
			return err
		}
		return launchWindowsUpdate(staged, exePath, pid)
	case "darwin":
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		exePath, _ = filepath.EvalSymlinks(exePath)
		return launchMacUpdate(staged, exePath, pid)
	case "linux":
		exePath, err := os.Executable()
		if err != nil {
			return err
		}
		exePath, _ = filepath.EvalSymlinks(exePath)
		return launchLinuxUpdate(staged, exePath, pid)
	default:
		return fmt.Errorf("当前平台暂不支持自动安装（%s）", goruntime.GOOS)
	}
}

func launchWindowsUpdate(staged *stagedUpdate, targetExe string, pid int) error {
	logPath := strings.TrimSpace(staged.InstallLogPath)
	if logPath == "" {
		logPath = buildUpdateInstallLogPath(filepath.Dir(staged.FilePath))
		staged.InstallLogPath = logPath
	}

	// 与 PinkHunkDB 一致：用独立 PowerShell 脚本做替换/重启。
	// 不要直接把新 exe 当子进程跑 --apply-update，宿主退出时容易把更新器一起带走。
	scriptPath := filepath.Join(staged.StagedDir, "update.ps1")
	content := buildWindowsPowerShellUpdateScript(pid)
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		return err
	}
	cmd := buildWindowsLaunchCommand(scriptPath)
	cmd.Env = append(os.Environ(), windowsUpdateScriptEnv(staged.FilePath, targetExe, staged.StagedDir, logPath, pid)...)
	if err := cmd.Start(); err != nil {
		return err
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
	return nil
}

func launchMacUpdate(staged *stagedUpdate, targetExe string, pid int) error {
	targetApp := resolveMacUpdateTarget(targetExe)
	mountDir := filepath.Join(staged.StagedDir, "mnt")
	if err := os.MkdirAll(mountDir, 0o755); err != nil {
		return err
	}
	logPath := strings.TrimSpace(staged.InstallLogPath)
	if logPath == "" {
		logPath = buildUpdateInstallLogPath(filepath.Dir(staged.FilePath))
		staged.InstallLogPath = logPath
	}
	scriptPath := filepath.Join(staged.StagedDir, "update.sh")
	content := buildMacScript(staged.FilePath, targetApp, staged.StagedDir, mountDir, logPath, pid)
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		return err
	}
	return exec.Command("/bin/bash", scriptPath).Start()
}

func launchLinuxUpdate(staged *stagedUpdate, targetExe string, pid int) error {
	scriptPath := filepath.Join(staged.StagedDir, "update.sh")
	content := buildLinuxScript(staged.FilePath, targetExe, staged.StagedDir, pid)
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		return err
	}
	return exec.Command("/bin/sh", scriptPath).Start()
}

func buildWindowsLaunchCommand(scriptPath string) *exec.Cmd {
	cmd := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-NoLogo",
		"-NonInteractive",
		"-WindowStyle", "Hidden",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
	)
	configureWindowsUpdateCommand(cmd)
	return cmd
}

func buildMacScript(dmgPath, targetApp, stagedDir, mountDir, logPath string, pid int) string {
	knownBins := strings.Join(productMacOSBinaryNames(), " ")
	knownApps := strings.Join(productAppBundleFileNames(), " ")
	return fmt.Sprintf(`#!/bin/bash
set -euo pipefail
PID=%d
DMG="%s"
TARGET_APP="%s"
STAGED="%s"
MOUNT_DIR="%s"
LOG_FILE="%s"
KNOWN_BINS="%s"
KNOWN_APPS="%s"
TMP_APP="${TARGET_APP}.new"
BACKUP_APP="${TARGET_APP}.backup"
APP_BIN_REL=""

log() {
  echo "[$(date '+%%Y-%%m-%%d %%H:%%M:%%S')] $*" >> "$LOG_FILE"
}

resolve_app_bin_rel() {
  local app="$1"
  local macos="$app/Contents/MacOS"
  local name found
  if [ ! -d "$macos" ]; then
    return 1
  fi
  for name in $KNOWN_BINS "$(basename "$app" .app)"; do
    if [ -n "$name" ] && [ -x "$macos/$name" ]; then
      echo "Contents/MacOS/$name"
      return 0
    fi
  done
  found=$(find "$macos" -maxdepth 1 \( -type f -o -type l \) ! -name '.*' 2>/dev/null | head -n 1 || true)
  if [ -n "$found" ]; then
    echo "Contents/MacOS/$(basename "$found")"
    return 0
  fi
  return 1
}

run_admin_replace() {
  /usr/bin/osascript <<'APPLESCRIPT' "$APP_SRC" "$TARGET_APP" "$TMP_APP" "$BACKUP_APP" "$APP_BIN_REL" "$LOG_FILE"
on run argv
  set srcPath to item 1 of argv
  set dstPath to item 2 of argv
  set tmpPath to item 3 of argv
  set bakPath to item 4 of argv
  set binRel to item 5 of argv
  set logPath to item 6 of argv
  set cmd to "set -eu; " & ¬
    "rm -rf " & quoted form of tmpPath & " " & quoted form of bakPath & "; " & ¬
    "/usr/bin/ditto " & quoted form of srcPath & " " & quoted form of tmpPath & "; " & ¬
    "if [ ! -x " & quoted form of (tmpPath & "/" & binRel) & " ]; then echo 'tmp app binary missing' >> " & quoted form of logPath & "; exit 1; fi; " & ¬
    "xattr -rd com.apple.quarantine " & quoted form of tmpPath & " >> " & quoted form of logPath & " 2>&1 || true; " & ¬
    "if [ -d " & quoted form of dstPath & " ]; then mv " & quoted form of dstPath & " " & quoted form of bakPath & "; fi; " & ¬
    "mv " & quoted form of tmpPath & " " & quoted form of dstPath & "; " & ¬
    "rm -rf " & quoted form of bakPath & "; " & ¬
    "xattr -rd com.apple.quarantine " & quoted form of dstPath & " >> " & quoted form of logPath & " 2>&1 || true"
  do shell script cmd with administrator privileges
end run
APPLESCRIPT
}

replace_app_direct() {
  rm -rf "$TMP_APP" "$BACKUP_APP" >>"$LOG_FILE" 2>&1 || true
  /usr/bin/ditto "$APP_SRC" "$TMP_APP" >>"$LOG_FILE" 2>&1
  if [ ! -x "$TMP_APP/$APP_BIN_REL" ]; then
    log "tmp app binary missing: $TMP_APP/$APP_BIN_REL"
    return 1
  fi
  xattr -rd com.apple.quarantine "$TMP_APP" >>"$LOG_FILE" 2>&1 || true
  if [ -d "$TARGET_APP" ]; then
    mv "$TARGET_APP" "$BACKUP_APP" >>"$LOG_FILE" 2>&1
  fi
  if ! mv "$TMP_APP" "$TARGET_APP" >>"$LOG_FILE" 2>&1; then
    log "move new app failed, trying rollback"
    rm -rf "$TARGET_APP" >>"$LOG_FILE" 2>&1 || true
    if [ -d "$BACKUP_APP" ]; then
      mv "$BACKUP_APP" "$TARGET_APP" >>"$LOG_FILE" 2>&1 || true
    fi
    return 1
  fi
  rm -rf "$BACKUP_APP" >>"$LOG_FILE" 2>&1 || true
  xattr -rd com.apple.quarantine "$TARGET_APP" >>"$LOG_FILE" 2>&1 || true
  return 0
}

relaunch_app() {
  # -n 强制新实例；失败再直接拉二进制
  if /usr/bin/open -n -a "$TARGET_APP" >>"$LOG_FILE" 2>&1; then
    return 0
  fi
  if /usr/bin/open -n "$TARGET_APP" >>"$LOG_FILE" 2>&1; then
    return 0
  fi
  log "open failed, trying binary launch"
  nohup "$TARGET_APP/$APP_BIN_REL" >>"$LOG_FILE" 2>&1 &
  return 0
}

log "updater started"
while kill -0 $PID 2>/dev/null; do
  sleep 1
done
log "host process exited"
if ! hdiutil attach "$DMG" -nobrowse -quiet -mountpoint "$MOUNT_DIR" >>"$LOG_FILE" 2>&1; then
  log "hdiutil attach failed: $DMG -> $MOUNT_DIR"
  exit 1
fi
# 注意：不能用 ls *.app（会展开目录内容，得到 Contents 而不是 .app 路径）
APP_SRC=""
for name in $KNOWN_APPS; do
  if [ -d "$MOUNT_DIR/$name" ] && [ -d "$MOUNT_DIR/$name/Contents" ]; then
    APP_SRC="$MOUNT_DIR/$name"
    break
  fi
done
if [ -z "$APP_SRC" ]; then
  APP_SRC=$(find "$MOUNT_DIR" -maxdepth 1 -type d -name "*.app" 2>/dev/null | head -n 1 || true)
fi
if [ -z "$APP_SRC" ] || [ ! -d "$APP_SRC/Contents" ]; then
  log "no .app found inside dmg (mount=$MOUNT_DIR, app_src=$APP_SRC)"
  hdiutil detach "$MOUNT_DIR" -quiet >>"$LOG_FILE" 2>&1 || true
  exit 1
fi

if ! APP_BIN_REL=$(resolve_app_bin_rel "$APP_SRC"); then
  log "no binary in source app: $APP_SRC"
  hdiutil detach "$MOUNT_DIR" -quiet >>"$LOG_FILE" 2>&1 || true
  exit 1
fi

log "install source: $APP_SRC"
log "install target: $TARGET_APP"
log "source binary: $APP_BIN_REL"
if ! replace_app_direct; then
  log "direct replace failed, trying admin replace"
  run_admin_replace >>"$LOG_FILE" 2>&1
fi

if ! APP_BIN_REL=$(resolve_app_bin_rel "$TARGET_APP"); then
  log "target app binary missing after replace: $TARGET_APP"
  hdiutil detach "$MOUNT_DIR" -quiet >>"$LOG_FILE" 2>&1 || true
  exit 1
fi
if [ ! -x "$TARGET_APP/$APP_BIN_REL" ]; then
  log "target app binary missing after replace: $TARGET_APP/$APP_BIN_REL"
  hdiutil detach "$MOUNT_DIR" -quiet >>"$LOG_FILE" 2>&1 || true
  exit 1
fi

hdiutil detach "$MOUNT_DIR" -quiet >>"$LOG_FILE" 2>&1 || true
rm -rf "$MOUNT_DIR" "$DMG" "$STAGED" >>"$LOG_FILE" 2>&1 || true
relaunch_app
log "relaunch requested"
`, pid, dmgPath, targetApp, stagedDir, mountDir, logPath, knownBins, knownApps)
}

func buildLinuxScript(tarPath, targetExe, stagedDir string, pid int) string {
	return fmt.Sprintf(`#!/bin/bash
set -e
PID=%d
ARCHIVE="%s"
TARGET="%s"
STAGED="%s"
while kill -0 $PID 2>/dev/null; do
  sleep 1
done
TMPDIR=$(mktemp -d)
tar -xzf "$ARCHIVE" -C "$TMPDIR"
TARGET_NAME="$(basename "$TARGET")"
NEWBIN="$TMPDIR/$TARGET_NAME"
if [ ! -f "$NEWBIN" ]; then
  NEWBIN=$(find "$TMPDIR" -type f -name "$TARGET_NAME" | head -n 1)
fi
if [ -z "$NEWBIN" ] || [ ! -f "$NEWBIN" ]; then
  NEWBIN=$(find "$TMPDIR" -type f \( -name "FlashShell" -o -name "flashshell" -o -name "FlashDock" -o -name "flashdock" \) | head -n 1)
fi
if [ -z "$NEWBIN" ] || [ ! -f "$NEWBIN" ]; then
  exit 1
fi
cp -f "$NEWBIN" "$TARGET"
chmod +x "$TARGET"
rm -rf "$TMPDIR" "$ARCHIVE" "$STAGED"
"$TARGET" &
`, pid, tarPath, targetExe, stagedDir)
}

func detectMacAppPath(exePath string) string {
	parts := strings.Split(exePath, string(filepath.Separator))
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.HasSuffix(parts[i], ".app") {
			appPath := filepath.Join(parts[:i+1]...)
			if !filepath.IsAbs(appPath) {
				appPath = string(filepath.Separator) + appPath
			}
			return appPath
		}
	}
	return ""
}

func productPathUnique(values ...string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	return out
}

func resolveMacAppBinaryRelFromEntries(entries []string, bundleBase string) string {
	bundleName := strings.TrimSuffix(filepath.Base(strings.TrimSpace(bundleBase)), ".app")
	preferred := productPathUnique(append(productMacOSBinaryNames(), bundleName)...)
	found := make(map[string]string, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		found[strings.ToLower(entry)] = entry
	}
	for _, name := range preferred {
		if orig, ok := found[strings.ToLower(name)]; ok {
			return filepath.ToSlash(filepath.Join("Contents", "MacOS", orig))
		}
	}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry != "" {
			return filepath.ToSlash(filepath.Join("Contents", "MacOS", entry))
		}
	}
	return ""
}

func findMacAppBinaryRel(appRoot string) string {
	macos := filepath.Join(strings.TrimSpace(appRoot), "Contents", "MacOS")
	items, err := os.ReadDir(macos)
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		names = append(names, item.Name())
	}
	return resolveMacAppBinaryRelFromEntries(names, filepath.Base(appRoot))
}

func defaultMacApplicationTargets() []string {
	return []string{
		"/Applications/" + ProductName + ".app",
		"/Applications/" + LegacyProductName + ".app",
	}
}

var macAppDirExists = func(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func firstExistingPath(paths []string, exists func(string) bool) string {
	if exists == nil {
		exists = macAppDirExists
	}
	for _, p := range paths {
		if exists(p) {
			return p
		}
	}
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func resolveMacUpdateTarget(exePath string) string {
	fallback := firstExistingPath(defaultMacApplicationTargets(), macAppDirExists)
	targetApp := detectMacAppPath(exePath)
	if targetApp == "" {
		return fallback
	}
	targetApp = filepath.Clean(targetApp)
	if strings.Contains(targetApp, string(filepath.Separator)+"AppTranslocation"+string(filepath.Separator)) {
		return fallback
	}
	return targetApp
}

func windowsKeepOrCanonicalExeName(currentBase string) string {
	base := filepath.Base(strings.TrimSpace(currentBase))
	for _, known := range productWindowsExeNames() {
		if strings.EqualFold(base, known) {
			return known
		}
	}
	return windowsFinalExeName
}

func preferredWindowsSourceExeNames(targetBase string) []string {
	names := []string{filepath.Base(strings.TrimSpace(targetBase))}
	names = append(names, productWindowsExeNames()...)
	return productPathUnique(names...)
}

func powershellQuotedList(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, "'"+strings.ReplaceAll(item, "'", "''")+"'")
	}
	return strings.Join(quoted, ", ")
}

func buildWindowsPowerShellUpdateScript(pid int) string {
	script := `$ErrorActionPreference = 'Stop'
$Source = $env:FLASHDOCK_UPDATE_SOURCE
$Target = $env:FLASHDOCK_UPDATE_TARGET
$Staged = $env:FLASHDOCK_UPDATE_STAGED
$LogFile = $env:FLASHDOCK_UPDATE_LOG
$HostPid = [int]$env:FLASHDOCK_UPDATE_PID
$CanonicalFinalName = '` + windowsFinalExeName + `'
$KnownExeNames = @(` + powershellQuotedList(productWindowsExeNames()) + `)

function Write-UpdateLog([string]$Message) {
  $line = '[{0}] {1}' -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $Message
  Add-Content -Path $LogFile -Value $line
}

function Wait-ForHostExit {
  $deadline = (Get-Date).AddSeconds(90)
  while ((Get-Process -Id $HostPid -ErrorAction SilentlyContinue) -and (Get-Date) -lt $deadline) {
    Start-Sleep -Seconds 1
  }
  if (Get-Process -Id $HostPid -ErrorAction SilentlyContinue) {
    Write-UpdateLog "host process still running after 90 seconds, aborting update"
    exit 1
  }
}

function Resolve-FinalName([string]$TargetPath) {
  $name = [System.IO.Path]::GetFileName($TargetPath)
  foreach ($n in $KnownExeNames) {
    if ($name -ieq $n) {
      return $name
    }
  }
  return $CanonicalFinalName
}

function Resolve-SourceExecutable([string]$SourcePath, [string]$TargetPath, [string]$StagedDir) {
  $targetName = [System.IO.Path]::GetFileName($TargetPath)
  $sourceExt = [System.IO.Path]::GetExtension($SourcePath)
  if ($sourceExt -ieq '.zip') {
    $extractDir = Join-Path $StagedDir '_extract'
    if (Test-Path -Path $extractDir) {
      Remove-Item -Path $extractDir -Recurse -Force
    }
    New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
    Expand-Archive -Path $SourcePath -DestinationPath $extractDir -Force
    $ordered = @()
    if ($targetName) { $ordered += $targetName }
    foreach ($n in $KnownExeNames) {
      if ($ordered -notcontains $n) { $ordered += $n }
    }
    foreach ($n in $ordered) {
      $direct = Join-Path $extractDir $n
      if (Test-Path -Path $direct) {
        return $direct
      }
      $nested = Get-ChildItem -Path $extractDir -Filter $n -Recurse -File -ErrorAction SilentlyContinue |
        Select-Object -First 1 -ExpandProperty FullName
      if ($nested) {
        return $nested
      }
    }
    $found = Get-ChildItem -Path $extractDir -Filter '*.exe' -Recurse -File |
      Select-Object -First 1 -ExpandProperty FullName
    if ($found) {
      return $found
    }
    throw "no executable found in portable zip: $SourcePath"
  }
  return $SourcePath
}

function Replace-TargetExecutable([string]$SourceExe, [string]$TargetExe) {
  $targetOld = "$TargetExe.old"
  for ($retry = 0; $retry -lt 15; $retry++) {
    Write-UpdateLog "attempt ${retry}: trying rename-then-copy strategy"
    try {
      if (Test-Path -Path $TargetExe) {
        if (Test-Path -Path $targetOld) {
          Remove-Item -Path $targetOld -Force
        }
        Move-Item -Path $TargetExe -Destination $targetOld -Force
      }
      Copy-Item -Path $SourceExe -Destination $TargetExe -Force
      if (Test-Path -Path $targetOld) {
        Remove-Item -Path $targetOld -Force
      }
      return
    } catch {
      Write-UpdateLog "rename strategy failed: $($_.Exception.Message)"
      if (Test-Path -Path $targetOld) {
        try {
          if (Test-Path -Path $TargetExe) {
            Remove-Item -Path $TargetExe -Force
          }
          Move-Item -Path $targetOld -Destination $TargetExe -Force
        } catch {
          Write-UpdateLog "restore old executable failed: $($_.Exception.Message)"
        }
      }
    }

    Write-UpdateLog 'rename strategy failed, trying direct move'
    try {
      Move-Item -Path $SourceExe -Destination $TargetExe -Force
      return
    } catch {
      Write-UpdateLog "direct move failed: $($_.Exception.Message)"
    }
    try {
      Copy-Item -Path $SourceExe -Destination $TargetExe -Force
      return
    } catch {
      Write-UpdateLog "direct copy failed: $($_.Exception.Message)"
    }

    $wait = 1
    if ($retry -ge 3) { $wait = 2 }
    if ($retry -ge 6) { $wait = 3 }
    if ($retry -ge 9) { $wait = 5 }
    Write-UpdateLog "waiting $wait seconds before retry"
    Start-Sleep -Seconds $wait
  }
  throw 'replace failed after retries (portable mode, no elevation): check directory write permission or file lock'
}

function Resolve-FinalTargetPath([string]$TargetExe, [string]$DesiredName) {
  $DesiredName = ($DesiredName + '').Trim()
  if (-not $DesiredName) {
    return $TargetExe
  }
  if ($DesiredName -notmatch '(?i)\.exe$') {
    Write-UpdateLog "final name is not exe, skip rename: $DesiredName"
    return $TargetExe
  }
  if ($DesiredName -match '[\\/]') {
    Write-UpdateLog "final name contains path separator, skip rename: $DesiredName"
    return $TargetExe
  }
  $dir = [System.IO.Path]::GetDirectoryName($TargetExe)
  return (Join-Path $dir $DesiredName)
}

function Rename-TargetToFinalName([string]$TargetExe, [string]$FinalPath) {
  if ($TargetExe -ieq $FinalPath) {
    return $TargetExe
  }
  if (-not (Test-Path -Path $TargetExe)) {
    throw "target missing before rename: $TargetExe"
  }
  if (Test-Path -Path $FinalPath) {
    Write-UpdateLog "removing existing final path before rename: $FinalPath"
    Remove-Item -Path $FinalPath -Force
  }
  Move-Item -Path $TargetExe -Destination $FinalPath -Force
  Write-UpdateLog "renamed target: $TargetExe -> $FinalPath"
  return $FinalPath
}

function Start-UpdatedApplication([string]$TargetExe) {
  $targetDir = [System.IO.Path]::GetDirectoryName($TargetExe)
  # 必须用 Normal：Hidden 会把 Wails GUI 主窗口藏掉，用户感觉「没有自动打开」
  $proc = Start-Process -FilePath $TargetExe -WorkingDirectory $targetDir -WindowStyle Normal -PassThru -ErrorAction Stop
  if (-not $proc -or $proc.HasExited) {
    throw "relaunch failed for target: $TargetExe"
  }
  Start-Sleep -Milliseconds 800
  if ($proc.HasExited) {
    throw ("relaunch exited immediately: pid={0} code={1} path={2}" -f $proc.Id, $proc.ExitCode, $TargetExe)
  }
  Write-UpdateLog ("started updated application: pid={0} path={1}" -f $proc.Id, $TargetExe)
}

try {
  Write-UpdateLog 'updater started'
  Write-UpdateLog "source=$Source"
  Write-UpdateLog "target=$Target"
  $FinalName = Resolve-FinalName -TargetPath $Target
  Write-UpdateLog "finalName=$FinalName"

  if (-not (Test-Path -Path $Source)) {
    throw "source file not found: $Source"
  }
  if (-not (Test-Path -Path $Target)) {
    throw "target executable not found: $Target"
  }

  $sourceExe = Resolve-SourceExecutable -SourcePath $Source -TargetPath $Target -StagedDir $Staged
  Write-UpdateLog "resolved source executable: $sourceExe"

  Wait-ForHostExit
  Write-UpdateLog 'host process exited'
  Start-Sleep -Seconds 3
  Write-UpdateLog 'cooldown finished, starting file replace'

  Replace-TargetExecutable -SourceExe $sourceExe -TargetExe $Target
  # FlashDock.exe / FlashShell.exe 原名保留；带版本号的资源名再归一到 FlashShell.exe
  $finalPath = Resolve-FinalTargetPath -TargetExe $Target -DesiredName $FinalName
  $launchPath = $Target
  try {
    $launchPath = Rename-TargetToFinalName -TargetExe $Target -FinalPath $finalPath
  } catch {
    Write-UpdateLog "rename to final name failed, launch original path: $($_.Exception.Message)"
    $launchPath = $Target
  }
  Start-UpdatedApplication -TargetExe $launchPath
  if (Test-Path -Path $Staged) {
    Remove-Item -Path $Staged -Recurse -Force
  }
  Write-UpdateLog 'update finished'
  exit 0
} catch {
  Write-UpdateLog ("update failed: " + $_.Exception.Message)
  exit 1
}
`
	_ = pid
	return strings.ReplaceAll(script, "\n", "\r\n")
}
