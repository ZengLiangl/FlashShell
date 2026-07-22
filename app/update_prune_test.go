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
	old := resolveLegacyUpdateWorkspaceDir
	resolveLegacyUpdateWorkspaceDir = func() string { return root }
	t.Cleanup(func() { resolveLegacyUpdateWorkspaceDir = old })

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
