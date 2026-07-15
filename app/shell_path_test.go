package app

import (
	"reflect"
	"testing"
)

func TestNormalizeRemoteAbs(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "/"},
		{"/", "/"},
		{"/root", "/root"},
		{"/root/", "/root"},
		{"root", "/root"},
		{"  /app/  ", "/app"},
	}
	for _, c := range cases {
		if got := NormalizeRemoteAbs(c.in); got != c.want {
			t.Errorf("NormalizeRemoteAbs(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestPathJoinRemote(t *testing.T) {
	cases := []struct {
		base, rel, want string
	}{
		{"/root", "app", "/root/app"},
		{"/root", "app/", "/root/app"},
		{"/root/", "app/", "/root/app"},
		{"/root", "app/sub", "/root/app/sub"},
		{"/root/app", "..", "/root"},
		{"/root/app", "../other", "/root/other"},
		{"/root", ".", "/root"},
		{"/root", "", "/root"},
		{"/root", "/tmp", "/tmp"},
	}
	for _, c := range cases {
		if got := PathJoinRemote(c.base, c.rel); got != c.want {
			t.Errorf("PathJoinRemote(%q,%q)=%q want %q", c.base, c.rel, got, c.want)
		}
	}
}

func TestResolveRemotePath(t *testing.T) {
	cases := []struct {
		name                     string
		base, target, home, want string
		wantErr                  bool
	}{
		{"relative from base", "/root", "app", "/home/u", "/root/app", false},
		{"relative trailing slash", "/root", "app/", "/home/u", "/root/app", false},
		{"absolute", "/root", "/var/log", "/home/u", "/var/log", false},
		{"tilde home", "/root", "~", "/home/u", "/home/u", false},
		{"tilde sub", "/root", "~/work", "/home/u", "/home/u/work", false},
		{"empty base uses home", "", "app", "/home/u", "/home/u/app", false},
		{"dot keeps base", "/root", ".", "/home/u", "/root", false},
		{"empty target keeps base", "/root", "", "/home/u", "/root", false},
		{"cd parent", "/root/app", "..", "/home/u", "/root", false},
		{"quoted", "/root", `"app"`, "/home/u", "/root/app", false},
		{"relative no base no home", "", "app", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveRemotePath(c.base, c.target, c.home)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestResolveShellCdTarget(t *testing.T) {
	cases := []struct {
		name                        string
		current, target, home, want string
		wantErr                     bool
	}{
		{"bare cd goes home", "/root/app", "", "/root", "/root", false},
		{"cd tilde goes home", "/var/log", "~", "/root", "/root", false},
		{"relative still works", "/root", "app", "/root", "/root/app", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveShellCdTarget(c.current, c.target, c.home)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestChooseCdPath(t *testing.T) {
	if got := ChooseCdPath("/root", "/root/app", true); got != "/root/app" {
		t.Fatalf("exists: got %q", got)
	}
	if got := ChooseCdPath("/root", "/root/ap", false); got != "/root" {
		t.Fatalf("missing keeps current: got %q want /root", got)
	}
	if got := ChooseCdPath("/root/", "/root/app/", false); got != "/root" {
		t.Fatalf("normalize keep: got %q", got)
	}
}

func TestRemoteAncestorPaths(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"/", []string{"/"}},
		{"/root", []string{"/", "/root"}},
		{"/root/app", []string{"/", "/root", "/root/app"}},
		{"/root/app/", []string{"/", "/root", "/root/app"}},
	}
	for _, c := range cases {
		got := RemoteAncestorPaths(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("RemoteAncestorPaths(%q)=%v want %v", c.in, got, c.want)
		}
	}
}
