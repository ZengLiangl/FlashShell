//go:build darwin

package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFlashDockAppBundleFromExecutable(t *testing.T) {
	exe := "/Applications/FlashShell.app/Contents/MacOS/FlashShell"
	got := flashDockAppBundleFromExecutable(exe)
	if got != "/Applications/FlashShell.app" {
		t.Fatalf("bundle = %q", got)
	}
	if flashDockAppBundleFromExecutable("/tmp/FlashShell") != "" {
		t.Fatal("bare binary should not look like a bundle")
	}
}

func TestIsFlashDockAppBundle(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "FlashShell.app")
	exe := filepath.Join(bundle, "Contents", "MacOS", ProductName)
	if err := os.MkdirAll(filepath.Dir(exe), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}
	if !isFlashDockAppBundle(bundle) {
		t.Fatal("expected fake bundle to be recognized")
	}
	legacy := filepath.Join(root, "FlashDock.app")
	legacyExe := filepath.Join(legacy, "Contents", "MacOS", LegacyProductName)
	if err := os.MkdirAll(filepath.Dir(legacyExe), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyExe, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}
	if !isFlashDockAppBundle(legacy) {
		t.Fatal("expected legacy FlashDock.app to be recognized")
	}
	if isFlashDockAppBundle(root) {
		t.Fatal("plain dir should not be a bundle")
	}
}
