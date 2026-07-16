package machine

import "testing"

func TestRemoteConfigName(t *testing.T) {
	resolver := func(name string) bool {
		known := map[string]bool{
			"web1": true, "va-test-66": true, "aml-plus-92": true,
		}
		return known[name]
	}
	cases := []struct{ in, want string }{
		{"web1", "web1"},
		{"web1-2", "web1"},
		{"web1-10", "web1"},
		{"va-test-66", "va-test-66"},
		{"va-test-66-2", "va-test-66"},
		{"aml-plus-92-2", "aml-plus-92"},
		{"web1#2", "web1"},
		{"local", "local"},
	}
	known := []string{"web1", "va-test-66", "aml-plus-92"}
	for _, c := range cases {
		if got := RemoteConfigNameForKnown(c.in, known); got != c.want {
			t.Fatalf("RemoteConfigNameForKnown(%q)=%q want %q", c.in, got, c.want)
		}
		if got := RemoteConfigNameWithResolver(c.in, resolver); got != c.want {
			t.Fatalf("RemoteConfigNameWithResolver(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestRemoteSessionIndexForConfig(t *testing.T) {
	if got := RemoteSessionIndexForConfig("va-test-66", "va-test-66"); got != 1 {
		t.Fatalf("got %d", got)
	}
	if got := RemoteSessionIndexForConfig("va-test-66-2", "va-test-66"); got != 2 {
		t.Fatalf("got %d", got)
	}
	if got := RemoteSessionIndexForConfig("web1-2", "web1"); got != 2 {
		t.Fatalf("got %d", got)
	}
}

func TestShellTabLabel(t *testing.T) {
	if got := ShellTabLabel("web1", "web1", ShellKindRemote); got != "web1" {
		t.Fatalf("got %q", got)
	}
	if got := ShellTabLabel("web1-2", "web1", ShellKindRemote); got != "web1-2" {
		t.Fatalf("got %q", got)
	}
	if got := ShellTabLabel("va-test-66", "va-test-66", ShellKindRemote); got != "va-test-66" {
		t.Fatalf("got %q", got)
	}
	if got := ShellTabLabel("local", "local", ShellKindLocal); got != "本机" {
		t.Fatalf("got %q", got)
	}
	if got := ShellTabLabel("local-2", "local", ShellKindLocal); got != "本机-2" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatRemoteSessionID(t *testing.T) {
	if got := FormatRemoteSessionID("web1", 1); got != "web1" {
		t.Fatalf("got %q", got)
	}
	if got := FormatRemoteSessionID("web1", 2); got != "web1-2" {
		t.Fatalf("got %q", got)
	}
	if got := FormatRemoteSessionID("va-test-66", 2); got != "va-test-66-2" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatLocalSessionID(t *testing.T) {
	if got := FormatLocalSessionID(1); got != "local" {
		t.Fatalf("got %q", got)
	}
	if got := FormatLocalSessionID(2); got != "local-2" {
		t.Fatalf("got %q", got)
	}
}

func TestLocalSessionIndex(t *testing.T) {
	if got := LocalSessionIndex("local"); got != 1 {
		t.Fatalf("got %d", got)
	}
	if got := LocalSessionIndex("local-2"); got != 2 {
		t.Fatalf("got %d", got)
	}
}
