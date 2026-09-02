package data

import "testing"

func TestNormalizeWindowOpacity(t *testing.T) {
	if got := NormalizeWindowOpacity(0); got != 1 {
		t.Fatalf("0 => %v", got)
	}
	if got := NormalizeWindowOpacity(0.2); got != 0.4 {
		t.Fatalf("0.2 => %v", got)
	}
	if got := NormalizeWindowOpacity(0.85); got != 0.85 {
		t.Fatalf("0.85 => %v", got)
	}
	if got := NormalizeWindowOpacity(1.5); got != 1 {
		t.Fatalf("1.5 => %v", got)
	}
}
