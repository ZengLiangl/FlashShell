package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateConfigHome(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, ".flashdock")
	newDir := filepath.Join(root, ".flashshell")

	if err := os.MkdirAll(filepath.Join(oldDir, "icons"), 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(oldDir, "global_config.yaml")
	if err := os.WriteFile(marker, []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(oldDir, "icons", "custom.png")
	if err := os.WriteFile(nested, []byte("img"), 0644); err != nil {
		t.Fatal(err)
	}

	migrateConfigHome(oldDir, newDir)

	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("old dir should be gone, err=%v", err)
	}
	migrated := filepath.Join(newDir, "global_config.yaml")
	data, err := os.ReadFile(migrated)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "ok" {
		t.Fatalf("unexpected content: %q", data)
	}
	gotNested, err := os.ReadFile(filepath.Join(newDir, "icons", "custom.png"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotNested) != "img" {
		t.Fatalf("nested file = %q", gotNested)
	}
}

func TestMigrateConfigHomeRemovesOldWhenNewExists(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, ".flashdock")
	newDir := filepath.Join(root, ".flashshell")

	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "a.yaml"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "b.yaml"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	migrateConfigHome(oldDir, newDir)

	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("leftover old dir should be removed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(newDir, "b.yaml")); err != nil {
		t.Fatalf("new dir should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newDir, "a.yaml")); !os.IsNotExist(err) {
		t.Fatal("should not overwrite existing new dir with old files")
	}
}

func TestMigrateConfigHomeFromCmdConfig(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, ".cmd-config")
	newDir := filepath.Join(root, ".flashshell")
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(oldDir, "global_config.yaml"), []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}
	migrateConfigHome(oldDir, newDir)
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("cmd-config should be gone, err=%v", err)
	}
	got, err := os.ReadFile(filepath.Join(newDir, "global_config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "legacy" {
		t.Fatalf("content = %q", got)
	}
}
