//go:build windows

package app

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestIsFlashDockShortcutName(t *testing.T) {
	if !isFlashDockShortcutName("FlashShell.lnk") {
		t.Fatal("expected FlashShell.lnk")
	}
	if !isFlashDockShortcutName("FlashDock.lnk") {
		t.Fatal("expected FlashDock.lnk")
	}
	if !isFlashDockShortcutName("flashdock-1.lnk") {
		t.Fatal("expected flashdock-1.lnk")
	}
	if isFlashDockShortcutName("Notepad.lnk") {
		t.Fatal("notepad should not match")
	}
}

func TestParseICOForResourcesRoundTrip(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	ico, err := encodeWindowsICO(img, []int{16, 32})
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parseICOForResources(ico)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	group := buildGrpIconDir(entries)
	parsed, err := parseGrpIconDir(group)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 2 {
		t.Fatalf("group entries = %d, want 2", len(parsed))
	}
	out := buildICOFromGrpEntries(entries)
	if len(out) < 6 {
		t.Fatal("rebuilt ico too short")
	}
	if !bytes.Equal(out[:6], ico[:6]) {
		t.Fatalf("ico header mismatch")
	}
}
