package app

import (
	"os"
	"path/filepath"
	"testing"

	"FlashDock/data"
	"gopkg.in/yaml.v3"
)

func TestPersistAppIconPresetWritesGlobalConfig(t *testing.T) {
	home := data.IsolateConfigHome(t)
	a := NewApp("")
	got, err := a.persistAppIconPreset("helm")
	if err != nil {
		t.Fatalf("persistAppIconPreset: %v", err)
	}
	if got != "helm" {
		t.Fatalf("preset = %q, want helm", got)
	}

	cfg, err := a.configManager.GetGlobalConfig()
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if cfg.AppIconPreset != "helm" {
		t.Fatalf("in-memory appIconPreset = %q, want helm", cfg.AppIconPreset)
	}

	raw, err := os.ReadFile(filepath.Join(home, "global_config.yaml"))
	if err != nil {
		t.Fatalf("read yaml: %v", err)
	}
	var disk data.GlobalConfig
	if err := yaml.Unmarshal(raw, &disk); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	if disk.AppIconPreset != "helm" {
		t.Fatalf("disk appIconPreset = %q, want helm", disk.AppIconPreset)
	}
}

func TestPersistAppIconPresetRejectsUnknown(t *testing.T) {
	data.IsolateConfigHome(t)
	a := NewApp("")
	got, err := a.persistAppIconPreset("not-a-real-preset")
	if err != nil {
		t.Fatalf("persistAppIconPreset: %v", err)
	}
	if got != "default" {
		t.Fatalf("unknown preset normalized to %q, want default", got)
	}
}
