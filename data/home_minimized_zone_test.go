package data

import "testing"

func TestNormalizeHomeMinimizedZone(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"task", "task"},
		{"shell", "shell"},
		{" task ", "task"},
		{" shell ", "shell"},
		{"TASK", ""},
		{"other", ""},
	}
	for _, c := range cases {
		if got := NormalizeHomeMinimizedZone(c.in); got != c.want {
			t.Fatalf("NormalizeHomeMinimizedZone(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
