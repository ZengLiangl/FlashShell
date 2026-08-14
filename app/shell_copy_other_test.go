package app

import (
	"testing"
)

func TestSameShellHost(t *testing.T) {
	cases := []struct {
		src, dst string
		want     bool
	}{
		{"web1", "web1", true},
		{"web1", "web1-2", false}, // 配置名需已归一化后再比
		{"web1", "web2", false},
		{"", "web1", false},
		{"web1", "", false},
		{"  web1  ", "web1", true},
	}
	for _, c := range cases {
		if got := SameShellHost(c.src, c.dst); got != c.want {
			t.Errorf("SameShellHost(%q,%q)=%v want %v", c.src, c.dst, got, c.want)
		}
	}
}

func TestJoinCopyDestPath(t *testing.T) {
	cases := []struct {
		dstDir, srcPath, want string
	}{
		{"/root/app", "/root/cpp/a.log", "/root/app/a.log"},
		{"/root/app/", "/root/cpp/lib", "/root/app/lib"},
		{"/tmp", "name.txt", "/tmp/name.txt"},
		{"/", "/a/b/c", "/c"},
	}
	for _, c := range cases {
		if got := JoinCopyDestPath(c.dstDir, c.srcPath); got != c.want {
			t.Errorf("JoinCopyDestPath(%q,%q)=%q want %q", c.dstDir, c.srcPath, got, c.want)
		}
	}
}

func TestCopyToOtherMode(t *testing.T) {
	if got := CopyToOtherMode(true); got != "instant" {
		t.Fatalf("same host mode=%q", got)
	}
	if got := CopyToOtherMode(false); got != "transfer" {
		t.Fatalf("cross host mode=%q", got)
	}
}
