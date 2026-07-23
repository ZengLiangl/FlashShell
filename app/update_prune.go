package app

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

func pruneHistoricalUpdateArtifacts(currentVersion, stagedVersion string) {
	discovered := discoverUpdateArtifactPaths()
	if len(discovered) == 0 {
		return
	}
	keep := resolveUpdateArtifactVersionsToKeep(currentVersion, stagedVersion, discovered)
	protected := resolveProtectedSoftwarePaths()
	for version, paths := range discovered {
		if _, ok := keep[normalizeVersion(version)]; ok {
			continue
		}
		for _, path := range paths {
			if strings.TrimSpace(path) == "" {
				continue
			}
			if isProtectedSoftwarePath(path, protected) {
				continue
			}
			_ = os.RemoveAll(path)
		}
	}
}

func resolveUpdateArtifactVersionsToKeep(currentVersion, stagedVersion string, discovered map[string][]string) map[string]struct{} {
	keep := make(map[string]struct{})
	anchor := normalizeVersion(currentVersion)
	staged := normalizeVersion(stagedVersion)
	if staged != "" && compareSemver(staged, anchor) > 0 {
		anchor = staged
	}
	if anchor != "" {
		keep[anchor] = struct{}{}
	}

	var previous string
	for _, version := range sortedUpdateArtifactVersions(discovered) {
		version = normalizeVersion(version)
		if version == "" || version == anchor {
			continue
		}
		if compareSemver(version, anchor) < 0 {
			previous = version
			break
		}
	}
	if previous != "" {
		keep[previous] = struct{}{}
	}
	return keep
}

func sortedUpdateArtifactVersions(discovered map[string][]string) []string {
	versions := make([]string, 0, len(discovered))
	for version := range discovered {
		versions = append(versions, version)
	}
	for i := 0; i < len(versions); i++ {
		for j := i + 1; j < len(versions); j++ {
			if compareSemver(versions[i], versions[j]) < 0 {
				versions[i], versions[j] = versions[j], versions[i]
			}
		}
	}
	return versions
}

func discoverUpdateArtifactPaths() map[string][]string {
	result := make(map[string][]string)
	addPath := func(version, path string) {
		version = normalizeVersion(version)
		path = strings.TrimSpace(path)
		if version == "" || path == "" {
			return
		}
		if !looksLikeSemver(version) {
			return
		}
		result[version] = appendUniquePath(result[version], path)
	}

	scanUpdateWorkspace(resolveUpdateWorkspaceDir(), addPath)
	// 兼容旧版临时目录
	if legacy := resolveLegacyUpdateWorkspaceDir(); !strings.EqualFold(filepath.Clean(legacy), filepath.Clean(resolveUpdateWorkspaceDir())) {
		scanUpdateWorkspace(legacy, addPath)
	}

	// 扫描当前软件安装目录旁的历史版本安装包/目录
	if installDir := resolveSoftwareInstallDir(); installDir != "" {
		scanSoftwareInstallArtifacts(installDir, addPath)
	}

	if goruntime.GOOS == "darwin" {
		// 兼容旧版曾放到「下载 / 桌面」的更新目录，一并纳入清理
		if downloadsDir, err := resolveDownloadsDir(); err == nil {
			scanSoftwareInstallArtifacts(downloadsDir, addPath)
		}
		if homeDir, err := os.UserHomeDir(); err == nil && strings.TrimSpace(homeDir) != "" {
			scanSoftwareInstallArtifacts(filepath.Join(homeDir, "Desktop"), addPath)
		}
	}
	return result
}

func scanUpdateWorkspace(workspaceDir string, addPath func(version, path string)) {
	workspaceDir = strings.TrimSpace(workspaceDir)
	if workspaceDir == "" {
		return
	}
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if version := parseUpdateStagedDirVersion(entry.Name()); version != "" {
			addPath(version, filepath.Join(workspaceDir, entry.Name()))
		}
	}
}

// scanSoftwareInstallArtifacts 扫描安装目录下带版本号的 FlashDock 产物（目录或安装包）。
func scanSoftwareInstallArtifacts(parentDir string, addPath func(version, path string)) {
	parentDir = strings.TrimSpace(parentDir)
	if parentDir == "" {
		return
	}
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		full := filepath.Join(parentDir, name)
		if version := parseUpdateStagedDirVersion(name); version != "" {
			if entry.IsDir() {
				addPath(version, full)
			}
			continue
		}
		if version := parseFlashDockArtifactVersion(name); version != "" {
			addPath(version, full)
		}
	}
}

// resolveSoftwareInstallDir 返回当前正在运行的软件所在目录（mac 为 .app 的父目录）。
var resolveSoftwareInstallDir = func() string {
	exe := ""
	if goruntime.GOOS == "windows" {
		if path, err := resolveWindowsUpdateTarget(); err == nil {
			exe = path
		}
	}
	if exe == "" {
		path, err := os.Executable()
		if err != nil {
			return ""
		}
		exe = path
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	exe = filepath.Clean(exe)
	if app := detectMacAppPath(exe); app != "" {
		return filepath.Dir(app)
	}
	return filepath.Dir(exe)
}

func resolveProtectedSoftwarePaths() []string {
	out := make([]string, 0, 4)
	add := func(path string) {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "" || path == "." {
			return
		}
		for _, existing := range out {
			if strings.EqualFold(existing, path) {
				return
			}
		}
		out = append(out, path)
	}

	if goruntime.GOOS == "windows" {
		if path, err := resolveWindowsUpdateTarget(); err == nil {
			add(path)
		}
	}
	if path, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = resolved
		}
		add(path)
		if app := detectMacAppPath(path); app != "" {
			add(app)
		}
	}
	return out
}

func isProtectedSoftwarePath(path string, protected []string) bool {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" {
		return true
	}
	cleanLower := strings.ToLower(clean)
	sep := string(filepath.Separator)
	for _, p := range protected {
		pLower := strings.ToLower(filepath.Clean(p))
		if pLower == "" {
			continue
		}
		if cleanLower == pLower {
			return true
		}
		// 禁止删除包含当前可执行文件 / .app 的目录
		if strings.HasPrefix(pLower, cleanLower+sep) {
			return true
		}
	}
	return false
}

func parseUpdateStagedDirVersion(dirName string) string {
	const prefix = ".flashdock-update-"
	if !strings.HasPrefix(dirName, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(dirName, prefix)
	for _, osPrefix := range []string{"windows-", "darwin-", "linux-"} {
		if !strings.HasPrefix(rest, osPrefix) {
			continue
		}
		return normalizeUpdateArtifactVersionSuffix(strings.TrimPrefix(rest, osPrefix))
	}
	return ""
}

func parseMacUpdateDirVersion(dirName string) string {
	return parseFlashDockArtifactVersion(dirName)
}

// parseFlashDockArtifactVersion 解析 FlashDock-{version}[ -平台后缀][.ext]
// 例如：FlashDock-1.2.3、FlashDock-1.2.3-Windows-Amd64.exe、FlashDock-1.2.3-MacOS-Arm64.dmg
func parseFlashDockArtifactVersion(name string) string {
	const prefix = "FlashDock-"
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(name, prefix)
	rest = stripKnownReleaseAssetExt(rest)
	parts := strings.Split(rest, "-")
	if len(parts) == 0 {
		return ""
	}
	version := normalizeVersion(parts[0])
	if !looksLikeSemver(version) {
		return ""
	}
	return version
}

func stripKnownReleaseAssetExt(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, ext := range []string{".tar.gz", ".exe", ".dmg", ".zip", ".appimage"} {
		if strings.HasSuffix(lower, ext) {
			return name[:len(name)-len(ext)]
		}
	}
	return name
}

func looksLikeSemver(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return false
	}
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if !isAllDigits(part) {
			return false
		}
	}
	return true
}

func normalizeUpdateArtifactVersionSuffix(versionPart string) string {
	versionPart = strings.TrimSpace(versionPart)
	if versionPart == "" {
		return ""
	}
	if idx := strings.LastIndex(versionPart, "-"); idx > 0 {
		tail := versionPart[idx+1:]
		if isAllDigits(tail) && len(tail) >= 10 {
			versionPart = versionPart[:idx]
		}
	}
	version := normalizeVersion(versionPart)
	if !looksLikeSemver(version) {
		return ""
	}
	return version
}

func isAllDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func appendUniquePath(paths []string, path string) []string {
	for _, existing := range paths {
		if strings.EqualFold(filepath.Clean(existing), filepath.Clean(path)) {
			return paths
		}
	}
	return append(paths, path)
}
