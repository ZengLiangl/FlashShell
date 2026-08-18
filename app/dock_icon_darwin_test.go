//go:build darwin

package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFlashDockAppBundleFromExecutable(t *testing.T) {
	exe := "/Applications/FlashDock.app/Contents/MacOS/FlashDock"
	got := flashDockAppBundleFromExecutable(exe)
	if got != "/Applications/FlashDock.app" {
		t.Fatalf("bundle = %q", got)
	}
	if flashDockAppBundleFromExecutable("/tmp/FlashDock") != "" {
		t.Fatal("bare binary should not look like a bundle")
	}
}

func TestIsFlashDockAppBundle(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "FlashDock.app")
	exe := filepath.Join(bundle, "Contents", "MacOS", "FlashDock")
	if err := os.MkdirAll(filepath.Dir(exe), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}
	if !isFlashDockAppBundle(bundle) {
		t.Fatal("expected fake bundle to be recognized")
	}
	if isFlashDockAppBundle(root) {
		t.Fatal("plain dir should not be a bundle")
	}
}
