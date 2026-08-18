package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUpdateArtifactVersionsToKeep(t *testing.T) {
	discovered := map[string][]string{
		"1.0.0": {"a"},
		"1.1.0": {"b"},
		"1.2.0": {"c"},
		"1.3.0": {"d"},
	}
	keep := resolveUpdateArtifactVersionsToKeep("1.2.0", "", discovered)
	if _, ok := keep["1.2.0"]; !ok {
		t.Fatalf("should keep current 1.2.0: %#v", keep)
	}
	if _, ok := keep["1.1.0"]; !ok {
		t.Fatalf("should keep previous 1.1.0: %#v", keep)
	}
	if len(keep) != 2 {
		t.Fatalf("want 2 keep versions, got %#v", keep)
	}
}

func TestResolveUpdateArtifactVersionsToKeepWithStaged(t *testing.T) {
	discovered := map[string][]string{
		"1.1.0": {"a"},
		"1.2.0": {"b"},
		"1.3.0": {"c"},
	}
	keep := resolveUpdateArtifactVersionsToKeep("1.2.0", "1.3.0", discovered)
	if _, ok := keep["1.3.0"]; !ok {
		t.Fatalf("should keep staged 1.3.0: %#v", keep)
	}
	if _, ok := keep["1.2.0"]; !ok {
		t.Fatalf("should keep previous 1.2.0: %#v", keep)
	}
}

func TestResolveUpdateArtifactVersionsToKeepTargetLatest(t *testing.T) {
	// 模拟「已检测到更新、尚未下完」：目标版本目录刚创建，必须保留，否则暂存目录会被 prune 删掉
	discovered := map[string][]string{
		"1.1.3": {"~/.flashdock/updates/.flashdock-update-darwin-v1.1.3"},
		"1.1.5": {"~/.flashdock/updates/.flashdock-update-darwin-v1.1.5"},
	}
	keep := resolveUpdateArtifactVersionsToKeep("1.1.3", "1.1.5", discovered)
	if _, ok := keep["1.1.5"]; !ok {
		t.Fatalf("should keep target latest 1.1.5: %#v", keep)
	}
	if _, ok := keep["1.1.3"]; !ok {
		t.Fatalf("should keep current 1.1.3: %#v", keep)
	}
}

func TestParseUpdateStagedDirVersion(t *testing.T) {
	if got := parseUpdateStagedDirVersion(".flashdock-update-windows-1.2.3"); got != "1.2.3" {
		t.Fatalf("got %q", got)
	}
	if got := parseUpdateStagedDirVersion(".flashdock-update-darwin-1.2.3-1710000000"); got != "1.2.3" {
		t.Fatalf("got %q", got)
	}
	if got := parseUpdateStagedDirVersion("other"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestPruneHistoricalUpdateArtifacts(t *testing.T) {
	root := t.TempDir()
	oldLegacy := resolveLegacyUpdateWorkspaceDir
	oldRoot := resolveUpdateWorkspaceRoot
	oldInstall := resolveSoftwareInstallDir
	resolveLegacyUpdateWorkspaceDir = func() string { return root }
	resolveUpdateWorkspaceRoot = func() string { return filepath.Join(root, "unused-root") }
	resolveSoftwareInstallDir = func() string { return filepath.Join(root, "unused-install") }
	t.Cleanup(func() {
		resolveLegacyUpdateWorkspaceDir = oldLegacy
		resolveUpdateWorkspaceRoot = oldRoot
		resolveSoftwareInstallDir = oldInstall
	})

	keepCurrent := filepath.Join(root, ".flashdock-update-windows-1.2.0")
	keepPrev := filepath.Join(root, ".flashdock-update-windows-1.1.0")
	dropOld := filepath.Join(root, ".flashdock-update-windows-1.0.0")
	for _, dir := range []string{keepCurrent, keepPrev, dropOld} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	pruneHistoricalUpdateArtifacts("1.2.0", "")
	if _, err := os.Stat(keepCurrent); err != nil {
		t.Fatalf("current should remain: %v", err)
	}
	if _, err := os.Stat(keepPrev); err != nil {
		t.Fatalf("previous should remain: %v", err)
	}
	if _, err := os.Stat(dropOld); !os.IsNotExist(err) {
		t.Fatalf("old version should be removed, err=%v", err)
	}
}

func TestParseFlashDockArtifactVersion(t *testing.T) {
	cases := map[string]string{
		"FlashShell-1.2.3":                   "1.2.3",
		"FlashShell-1.2.3-Windows-Amd64.exe": "1.2.3",
		"FlashDock-1.2.3":                    "1.2.3",
		"FlashDock-1.2.3-Windows-Amd64.exe":  "1.2.3",
		"FlashDock-1.2.3-MacOS-Arm64.dmg":    "1.2.3",
		"FlashDock-1.2.3-Linux-Amd64.tar.gz": "1.2.3",
		"FlashShell.app":                     "",
		"FlashDock.app":                      "",
		"FlashDock.exe":                      "",
		"FlashDock-README":                   "",
		".flashdock-update-windows-1.2.3":    "",
	}
	for name, want := range cases {
		if got := parseFlashDockArtifactVersion(name); got != want {
			t.Fatalf("%s: got %q want %q", name, got, want)
		}
	}
}

func TestPruneHistoricalArtifactsInSoftwareInstallDir(t *testing.T) {
	root := t.TempDir()
	oldInstall := resolveSoftwareInstallDir
	oldLegacy := resolveLegacyUpdateWorkspaceDir
	oldRoot := resolveUpdateWorkspaceRoot
	resolveSoftwareInstallDir = func() string { return root }
	resolveLegacyUpdateWorkspaceDir = func() string { return filepath.Join(root, "legacy-empty") }
	resolveUpdateWorkspaceRoot = func() string { return filepath.Join(root, "updates-empty") }
	t.Cleanup(func() {
		resolveSoftwareInstallDir = oldInstall
		resolveLegacyUpdateWorkspaceDir = oldLegacy
		resolveUpdateWorkspaceRoot = oldRoot
	})

	keepCurrent := filepath.Join(root, "FlashDock-1.2.0-Windows-Amd64.exe")
	keepPrev := filepath.Join(root, "FlashDock-1.1.0-Windows-Amd64.exe")
	dropOld := filepath.Join(root, "FlashDock-1.0.0-Windows-Amd64.exe")
	for _, path := range []string{keepCurrent, keepPrev, dropOld} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// 当前运行中的同名文件应受保护（即使版本不在 keep 集）
	running := filepath.Join(root, "FlashDock.exe")
	if err := os.WriteFile(running, []byte("run"), 0o644); err != nil {
		t.Fatal(err)
	}

	pruneHistoricalUpdateArtifacts("1.2.0", "")
	if _, err := os.Stat(keepCurrent); err != nil {
		t.Fatalf("current asset should remain: %v", err)
	}
	if _, err := os.Stat(keepPrev); err != nil {
		t.Fatalf("previous asset should remain: %v", err)
	}
	if _, err := os.Stat(dropOld); !os.IsNotExist(err) {
		t.Fatalf("old asset should be removed, err=%v", err)
	}
	if _, err := os.Stat(running); err != nil {
		t.Fatalf("running exe should remain: %v", err)
	}
}

func TestIsProtectedSoftwarePath(t *testing.T) {
	exe := filepath.Join("C:", "Apps", "FlashDock.exe")
	protected := []string{exe}
	if !isProtectedSoftwarePath(exe, protected) {
		t.Fatal("exe itself should be protected")
	}
	if isProtectedSoftwarePath(filepath.Join("C:", "Apps", "FlashDock-1.0.0-Windows-Amd64.exe"), protected) {
		t.Fatal("sibling old installer should not be protected")
	}
}
