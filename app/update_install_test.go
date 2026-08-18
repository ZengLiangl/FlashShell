package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
)

func TestResolveMacAppBinaryRelPrefersNewNameInsideLegacyBundle(t *testing.T) {
	got := resolveMacAppBinaryRelFromEntries([]string{"FlashShell"}, "FlashDock.app")
	if got != "Contents/MacOS/FlashShell" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveMacAppBinaryRelKeepsLegacyBinary(t *testing.T) {
	got := resolveMacAppBinaryRelFromEntries([]string{"FlashDock"}, "FlashDock.app")
	if got != "Contents/MacOS/FlashDock" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveMacAppBinaryRelPrefersFlashShellWhenBothExist(t *testing.T) {
	got := resolveMacAppBinaryRelFromEntries([]string{"FlashDock", "FlashShell"}, "FlashDock.app")
	if got != "Contents/MacOS/FlashShell" {
		t.Fatalf("got %q", got)
	}
}

func TestFindMacAppBinaryRelReadsBundle(t *testing.T) {
	root := t.TempDir()
	bundle := filepath.Join(root, "FlashDock.app")
	macos := filepath.Join(bundle, "Contents", "MacOS")
	if err := os.MkdirAll(macos, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(macos, "FlashShell")
	if err := os.WriteFile(bin, []byte("ok"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := findMacAppBinaryRel(bundle)
	if got != "Contents/MacOS/FlashShell" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultMacApplicationTargetsIncludeBothProductNames(t *testing.T) {
	got := defaultMacApplicationTargets()
	if len(got) < 2 {
		t.Fatalf("want both product bundles, got %#v", got)
	}
	if got[0] != "/Applications/FlashShell.app" {
		t.Fatalf("new name should be first: %#v", got)
	}
	if got[1] != "/Applications/FlashDock.app" {
		t.Fatalf("legacy name missing: %#v", got)
	}
}

func TestFirstExistingPathFallsBackToPreferredNewName(t *testing.T) {
	paths := []string{"/Applications/FlashShell.app", "/Applications/FlashDock.app"}
	got := firstExistingPath(paths, func(string) bool { return false })
	if got != "/Applications/FlashShell.app" {
		t.Fatalf("got %q", got)
	}
}

func TestFirstExistingPathUsesLegacyInstall(t *testing.T) {
	paths := []string{"/Applications/FlashShell.app", "/Applications/FlashDock.app"}
	got := firstExistingPath(paths, func(path string) bool {
		return path == "/Applications/FlashDock.app"
	})
	if got != "/Applications/FlashDock.app" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveMacUpdateTargetKeepsLegacyBundle(t *testing.T) {
	exe := "/Applications/FlashDock.app/Contents/MacOS/FlashDock"
	got := resolveMacUpdateTarget(exe)
	if got != "/Applications/FlashDock.app" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveMacUpdateTargetKeepsNewBundle(t *testing.T) {
	exe := "/Applications/FlashShell.app/Contents/MacOS/FlashShell"
	got := resolveMacUpdateTarget(exe)
	if got != "/Applications/FlashShell.app" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveMacUpdateTargetTranslocationPrefersExistingLegacy(t *testing.T) {
	orig := macAppDirExists
	t.Cleanup(func() { macAppDirExists = orig })
	macAppDirExists = func(path string) bool {
		return path == "/Applications/FlashDock.app"
	}
	exe := "/private/var/folders/xx/AppTranslocation/abc/d/FlashDock.app/Contents/MacOS/FlashDock"
	got := resolveMacUpdateTarget(exe)
	if got != "/Applications/FlashDock.app" {
		t.Fatalf("got %q", got)
	}
}

func TestWindowsKeepOrCanonicalExeName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"FlashDock.exe", "FlashDock.exe"},
		{"FlashShell.exe", "FlashShell.exe"},
		{"flashdock.exe", "FlashDock.exe"},
		{"FlashDock-1.2.3-Windows-Amd64.exe", "FlashShell.exe"},
		{"custom.exe", "FlashShell.exe"},
	}
	for _, tc := range cases {
		if got := windowsKeepOrCanonicalExeName(tc.in); got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestPreferredWindowsSourceExeNames(t *testing.T) {
	got := preferredWindowsSourceExeNames("FlashDock.exe")
	if len(got) < 2 || got[0] != "FlashDock.exe" {
		t.Fatalf("target name first: %#v", got)
	}
	hasShell, hasDock := false, false
	for _, n := range got {
		if n == "FlashShell.exe" {
			hasShell = true
		}
		if n == "FlashDock.exe" {
			hasDock = true
		}
	}
	if !hasShell || !hasDock {
		t.Fatalf("want both product exe names, got %#v", got)
	}
}

func TestBuildMacScriptFindsEitherProductBinary(t *testing.T) {
	script := buildMacScript(
		"/tmp/FlashShell-1.0.0-MacOS-Arm64.dmg",
		"/Applications/FlashDock.app",
		"/tmp/staged",
		"/tmp/mnt",
		"/tmp/log",
		42,
	)
	if script == "" {
		t.Fatal("mac update script is empty")
	}
	if !strings.Contains(script, "resolve_app_bin_rel") {
		t.Fatal("script must resolve binary from bundle contents, not assume bundle basename")
	}
	if !strings.Contains(script, "FlashShell") || !strings.Contains(script, "FlashDock") {
		t.Fatal("script must accept both FlashShell and FlashDock binaries")
	}
	if !strings.Contains(script, "FlashShell.app") || !strings.Contains(script, "FlashDock.app") {
		t.Fatal("script must prefer either product .app inside the dmg")
	}
}

func TestBuildWindowsScriptKeepsEitherProductExe(t *testing.T) {
	script := buildWindowsPowerShellUpdateScript(1)
	if script == "" {
		t.Fatal("windows update script is empty")
	}
	if !strings.Contains(script, "FlashDock.exe") || !strings.Contains(script, "FlashShell.exe") {
		t.Fatal("script must accept both FlashDock.exe and FlashShell.exe")
	}
	if !strings.Contains(script, "Resolve-FinalName") {
		t.Fatal("script must keep current FlashDock.exe / FlashShell.exe instead of always renaming")
	}
	if !strings.Contains(script, "$KnownExeNames") {
		t.Fatal("script must search known product exe names in portable zip")
	}
}

func TestBuildMacScriptIsValidBash(t *testing.T) {
	if goruntime.GOOS == "windows" {
		t.Skip("bash -n is not used on Windows")
	}
	script := buildMacScript("/tmp/a.dmg", "/Applications/FlashDock.app", "/tmp/s", "/tmp/m", "/tmp/l", 7)
	path := filepath.Join(t.TempDir(), "update.sh")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("bash", "-n", path).CombinedOutput()
	if err != nil {
		t.Fatalf("bash -n failed: %v\n%s", err, out)
	}
}

func TestPickReleaseAssetAcceptsLegacyAndNewPrefix(t *testing.T) {
	osName, archName, ext := platformAssetHints()
	if osName == "" {
		t.Skip("unsupported platform")
	}
	legacy := fmt.Sprintf("FlashDock-9.9.9-%s-%s%s", osName, archName, ext)
	modern := fmt.Sprintf("FlashShell-9.9.9-%s-%s%s", osName, archName, ext)
	got := pickReleaseAsset([]githubAsset{{Name: legacy}}, "v9.9.9")
	if got == nil || got.Name != legacy {
		t.Fatalf("legacy fallback failed: %+v", got)
	}
	got = pickReleaseAsset([]githubAsset{{Name: legacy}, {Name: modern}}, "v9.9.9")
	if got == nil || got.Name != modern {
		t.Fatalf("prefer new name, got %+v", got)
	}
}
