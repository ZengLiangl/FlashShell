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
	for version, paths := range discovered {
		if _, ok := keep[normalizeVersion(version)]; ok {
			continue
		}
		for _, path := range paths {
			if strings.TrimSpace(path) == "" {
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
		result[version] = appendUniquePath(result[version], path)
	}

	scanUpdateWorkspace(resolveUpdateWorkspaceDir(), addPath)
	// 兼容旧版临时目录
	if legacy := resolveLegacyUpdateWorkspaceDir(); !strings.EqualFold(filepath.Clean(legacy), filepath.Clean(resolveUpdateWorkspaceDir())) {
		scanUpdateWorkspace(legacy, addPath)
	}

	if goruntime.GOOS == "darwin" {
		scanMacFlashDockUpdateDirs := func(parentDir string) {
			parentDir = strings.TrimSpace(parentDir)
			if parentDir == "" {
				return
			}
			entries, err := os.ReadDir(parentDir)
			if err != nil {
				return
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				if version := parseMacUpdateDirVersion(entry.Name()); version != "" {
					addPath(version, filepath.Join(parentDir, entry.Name()))
				}
			}
		}
		// 兼容旧版曾放到「下载 / 桌面」的更新目录，一并纳入清理
		if downloadsDir, err := resolveDownloadsDir(); err == nil {
			scanMacFlashDockUpdateDirs(downloadsDir)
		}
		if homeDir, err := os.UserHomeDir(); err == nil && strings.TrimSpace(homeDir) != "" {
			scanMacFlashDockUpdateDirs(filepath.Join(homeDir, "Desktop"))
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
	const prefix = "FlashDock-"
	if !strings.HasPrefix(dirName, prefix) {
		return ""
	}
	return normalizeVersion(strings.TrimPrefix(dirName, prefix))
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
	return normalizeVersion(versionPart)
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
