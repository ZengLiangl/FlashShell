package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"FlashDock/define"
)

func TestAppDataMigratesLegacyJSONFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// ConfigHomeDir 用 UserHomeDir；强制重置 once 无法轻易做，直接写到 home/.flashdock
	dir := filepath.Join(home, ConfigHomeDirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	known := []KnownHostRecord{{Host: "1.2.3.4", Port: 22, Fingerprint: "SHA256:abc"}}
	writeJSON(t, filepath.Join(dir, "known_hosts.json"), known)
	writeJSON(t, filepath.Join(dir, "shell_history.json"), []define.ShellHistoryRecord{
		{MachineID: "m1", MachineName: "web", ConnectCount: 2},
	})
	writeJSON(t, filepath.Join(dir, "shell_command_history.json"), shellCmdHistoryFile{
		Global:  []string{"ls"},
		ByScope: map[string][]string{"web": {"pwd"}},
	})
	writeJSON(t, filepath.Join(dir, "shortcuts.json"), DefaultShortcutSettings())

	// 重置进程内缓存，模拟首次加载
	appDataMu.Lock()
	appDataCache = nil
	appDataLoaded = false
	appDataMu.Unlock()

	// ConfigHomeDir 依赖真实 HOME；macOS 上 UserHomeDir 读 HOME
	d, err := loadAppDataSection()
	if err != nil {
		t.Fatalf("loadAppDataSection: %v", err)
	}
	if len(d.KnownHosts) != 1 || d.KnownHosts[0].Host != "1.2.3.4" {
		t.Fatalf("knownHosts 未迁移: %+v", d.KnownHosts)
	}
	if len(d.ShellHistory) != 1 || d.ShellHistory[0].MachineName != "web" {
		t.Fatalf("shellHistory 未迁移: %+v", d.ShellHistory)
	}
	if len(d.ShellCommandHistory.Global) != 1 || d.ShellCommandHistory.Global[0] != "ls" {
		t.Fatalf("shellCommandHistory 未迁移: %+v", d.ShellCommandHistory)
	}
	if d.Shortcuts.Find.Key == "" {
		t.Fatalf("shortcuts 未迁移")
	}

	merged := filepath.Join(dir, appDataFileName)
	if _, err := os.Stat(merged); err != nil {
		t.Fatalf("应生成 %s: %v", appDataFileName, err)
	}
	for _, name := range []string{"known_hosts.json", "shell_history.json", "shell_command_history.json", "shortcuts.json"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("旧文件应已删除: %s", name)
		}
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
}
