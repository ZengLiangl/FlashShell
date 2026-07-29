package data_test

import (
	"os"
	"path/filepath"
	"testing"

	"FlashDock/data"
)

func TestConfigHomeEnvOverridesUserHome(t *testing.T) {
	dir := data.IsolateConfigHome(t)
	got, err := data.ConfigHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("ConfigHomeDir=%q, want %q", got, dir)
	}
	// 不应落到真实用户目录下的 .flashdock
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(home, data.ConfigHomeDirName)
	if got == real {
		t.Fatalf("仍指向真实配置目录: %s", got)
	}
}
