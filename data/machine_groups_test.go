package data

import (
	"path/filepath"
	"testing"

	"FlashDock/define"
)

func TestUpdateMachineGroupPreservesEncryptedData(t *testing.T) {
	dir := t.TempDir()
	gcm := NewGlobalConfigManager(filepath.Join(dir, "global_config.yaml"))
	if _, err := gcm.LoadGlobalConfig(); err != nil {
		t.Fatalf("LoadGlobalConfig: %v", err)
	}

	m := &define.Machine{
		ID:            "m1",
		Name:          "web-1",
		Group:         "prod",
		EncryptedData: "keep-me",
	}
	if err := gcm.AddMachine(m); err != nil {
		t.Fatalf("AddMachine: %v", err)
	}

	if err := gcm.UpdateMachineGroup("m1", "staging"); err != nil {
		t.Fatalf("UpdateMachineGroup: %v", err)
	}
	got := gcm.GetMachine("m1")
	if got == nil {
		t.Fatal("machine missing after update")
	}
	if got.Group != "staging" {
		t.Fatalf("group = %q, want staging", got.Group)
	}
	if got.EncryptedData != "keep-me" {
		t.Fatalf("encrypted_data wiped: %q", got.EncryptedData)
	}

	if err := gcm.UpdateMachineGroup("m1", DefaultMachineGroupName); err != nil {
		t.Fatalf("UpdateMachineGroup default: %v", err)
	}
	got = gcm.GetMachine("m1")
	if got.Group != "" {
		t.Fatalf("default group should clear field, got %q", got.Group)
	}
}
