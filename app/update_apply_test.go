package app

import (
	"path/filepath"
	"testing"
)

func TestIsWindowsReleaseAssetFileName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"FlashShell-1.1.13-Windows-Amd64.exe", true},
		{"FlashDock-1.1.13-Windows-Amd64.exe", true},
		{"FlashDock-1.1.13-Windows-Arm64.exe", true},
		{"flashdock-2.0.0-windows-amd64.exe", true},
		{"FlashShell.exe", false},
		{"FlashDock.exe", false},
		{"FlashDock2222.exe", false},
		{"FlashDock-1.1.13-MacOS-Arm64.dmg", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isWindowsReleaseAssetFileName(tc.name); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestParseApplyUpdateArgs(t *testing.T) {
	cfg, ok := parseApplyUpdateArgs([]string{
		"--apply-update",
		"--update-target", `C:\Apps\old.exe`,
		"--update-pid", "1234",
		"--update-log", `C:\log.txt`,
		"--update-staged", `C:\staged`,
	})
	if !ok {
		t.Fatal("expected apply-update")
	}
	if cfg.Target != `C:\Apps\old.exe` || cfg.PID != 1234 || cfg.Log != `C:\log.txt` || cfg.Staged != `C:\staged` {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}

	cfg, ok = parseApplyUpdateArgs([]string{
		"--apply-update",
		"--update-target=C:\\a.exe",
		"--update-pid=9",
	})
	if !ok || cfg.Target != `C:\a.exe` || cfg.PID != 9 {
		t.Fatalf("inline parse failed: ok=%v cfg=%+v", ok, cfg)
	}

	if _, ok := parseApplyUpdateArgs([]string{"-reg=desk"}); ok {
		t.Fatal("should not treat normal launch as apply-update")
	}
}

func TestWindowsFinalExePath(t *testing.T) {
	input := filepath.Join("Users", "a", "Desktop", "FlashDock2222.exe")
	got := windowsFinalExePath(input)
	want := filepath.Join("Users", "a", "Desktop", "FlashShell.exe")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
