package app

import (
	"testing"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func TestShouldTreatAsMaximised(t *testing.T) {
	cases := []struct {
		known, native, wails, ours, want bool
	}{
		{false, false, false, false, false},
		{false, false, true, false, true},
		{false, false, false, true, true},
		{true, true, false, false, true},
		{true, false, true, true, false}, // 已知未最大化时，忽略过期的内部标记
		{true, true, false, true, true},
	}
	for _, c := range cases {
		if got := shouldTreatAsMaximised(c.known, c.native, c.wails, c.ours); got != c.want {
			t.Fatalf("shouldTreatAsMaximised(%v,%v,%v,%v)=%v want %v",
				c.known, c.native, c.wails, c.ours, got, c.want)
		}
	}
}

func TestChooseRestoreBoundsUsesSavedRect(t *testing.T) {
	x, y, w, h := chooseRestoreBounds(true, 40, 50, 1280, 800, 1920, 1080)
	if x != 40 || y != 50 || w != 1280 || h != 800 {
		t.Fatalf("got %d,%d %dx%d", x, y, w, h)
	}
}

func TestChooseRestoreBoundsFallbackCentersDefault(t *testing.T) {
	x, y, w, h := chooseRestoreBounds(false, 0, 0, 0, 0, 1920, 1080)
	if w != defaultWindowWidth || h != defaultWindowHeight {
		t.Fatalf("size %dx%d", w, h)
	}
	if x != (1920-defaultWindowWidth)/2 || y != (1080-defaultWindowHeight)/2 {
		t.Fatalf("pos %d,%d", x, y)
	}
}

func TestChooseRestoreBoundsIgnoresTinySaved(t *testing.T) {
	_, _, w, h := chooseRestoreBounds(true, 0, 0, 80, 80, 1920, 1080)
	if w != defaultWindowWidth || h != defaultWindowHeight {
		t.Fatalf("tiny saved should fall back, got %dx%d", w, h)
	}
}

func TestScreenLogicalSizePrefersSize(t *testing.T) {
	var screen wailsRuntime.Screen
	screen.Width = 1
	screen.Height = 2
	screen.Size.Width = 1920
	screen.Size.Height = 1080
	w, h := screenLogicalSize(screen)
	if w != 1920 || h != 1080 {
		t.Fatalf("got %dx%d", w, h)
	}
}

func TestScreenLogicalSizeFallsBackToDeprecated(t *testing.T) {
	w, h := screenLogicalSize(wailsRuntime.Screen{Width: 1366, Height: 768})
	if w != 1366 || h != 768 {
		t.Fatalf("got %dx%d", w, h)
	}
}
