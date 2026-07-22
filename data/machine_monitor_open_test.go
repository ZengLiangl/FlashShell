package data

import (
	"path/filepath"
	"testing"

	"FlashDock/define"
)

func TestUpdateMachineShellMonitorOpen(t *testing.T) {
	dir := t.TempDir()
	gcm := NewGlobalConfigManager(filepath.Join(dir, "global_config.yaml"))
	if _, err := gcm.LoadGlobalConfig(); err != nil {
		t.Fatalf("LoadGlobalConfig: %v", err)
	}
	m := &define.Machine{ID: "m1", Name: "web-1"}
	if err := gcm.AddMachine(m); err != nil {
		t.Fatalf("AddMachine: %v", err)
	}
	got := gcm.GetMachine("m1")
	if got == nil || !got.IsShellMonitorOpen() {
		t.Fatalf("空值默认应为展开, got=%+v", got)
	}
	if err := gcm.UpdateMachineShellMonitorOpen("web-1", false); err != nil {
		t.Fatalf("collapse by name: %v", err)
	}
	got = gcm.GetMachine("m1")
	if got == nil || got.IsShellMonitorOpen() || got.ShellMonitorOpen == nil || *got.ShellMonitorOpen {
		t.Fatalf("收起后应为 false, got=%+v shellMonitorOpen=%v", got, got.ShellMonitorOpen)
	}
	if err := gcm.UpdateMachineShellMonitorOpen("m1", true); err != nil {
		t.Fatalf("open by id: %v", err)
	}
	got = gcm.GetMachine("m1")
	if got == nil || !got.IsShellMonitorOpen() || got.ShellMonitorOpen != nil {
		t.Fatalf("展开后应清空字段（空=展开）, got=%+v shellMonitorOpen=%v", got, got.ShellMonitorOpen)
	}
}
