package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateConfigHome(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, ".cmd-config")
	newDir := filepath.Join(root, ".flashdock")

	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(oldDir, "global_config.yaml")
	if err := os.WriteFile(marker, []byte("ok"), 0644); err != nil {
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
}

func TestMigrateConfigHomeSkipsWhenNewExists(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, ".cmd-config")
	newDir := filepath.Join(root, ".flashdock")

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

	if _, err := os.Stat(filepath.Join(oldDir, "a.yaml")); err != nil {
		t.Fatalf("old dir should remain when new exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newDir, "b.yaml")); err != nil {
		t.Fatalf("new dir should remain: %v", err)
	}
}
