package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"FlashDock/define"
)

func TestAppDataMigratesLegacyJSONFiles(t *testing.T) {
	dir := IsolateConfigHome(t)

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
