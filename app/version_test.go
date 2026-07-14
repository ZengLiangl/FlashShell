package app

import "testing"

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.2.1", "1.0.0", 1},
		{"1.0.0", "v1.3.0", -1},
		{"1.10.0", "1.9.9", 1},
		{"2.0.0", "1.9.9", 1},
	}
	for _, c := range cases {
		if got := compareSemver(c.a, c.b); got != c.want {
			t.Fatalf("compareSemver(%q,%q)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestFormatVersionDisplay(t *testing.T) {
	if formatVersionDisplay("1.0.0") != "v1.0.0" {
		t.Fatal(formatVersionDisplay("1.0.0"))
	}
	if formatVersionDisplay("v1.0.0") != "v1.0.0" {
		t.Fatal(formatVersionDisplay("v1.0.0"))
	}
}
