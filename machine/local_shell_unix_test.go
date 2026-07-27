//go:build !windows

package machine

import (
	"strings"
	"testing"
)

func TestBuildLocalShellEnv_NormalizesTerminalAndLocale(t *testing.T) {
	env := buildLocalShellEnv([]string{
		"PATH=/usr/bin",
		"TERM=dumb",
		"LANG=C",
		"COLUMNS=999",
		"LINES=40",
	})

	got := envSliceToMap(env)
	if got["TERM"] != "xterm-256color" {
		t.Fatalf("TERM=%q, want %q", got["TERM"], "xterm-256color")
	}
	if got["COLORTERM"] != "truecolor" {
		t.Fatalf("COLORTERM=%q, want %q", got["COLORTERM"], "truecolor")
	}
	if got["LANG"] != "en_US.UTF-8" {
		t.Fatalf("LANG=%q, want %q", got["LANG"], "en_US.UTF-8")
	}
	if got["LC_ALL"] != "en_US.UTF-8" {
		t.Fatalf("LC_ALL=%q, want %q", got["LC_ALL"], "en_US.UTF-8")
	}
	if got["LC_CTYPE"] != "en_US.UTF-8" {
		t.Fatalf("LC_CTYPE=%q, want %q", got["LC_CTYPE"], "en_US.UTF-8")
	}
	if _, ok := got["COLUMNS"]; ok {
		t.Fatalf("COLUMNS should be removed, got %q", got["COLUMNS"])
	}
	if _, ok := got["LINES"]; ok {
		t.Fatalf("LINES should be removed, got %q", got["LINES"])
	}
}

func TestBuildLocalShellEnv_KeepsUtf8Locale(t *testing.T) {
	env := buildLocalShellEnv([]string{
		"LANG=zh_CN.UTF-8",
		"LC_ALL=zh_CN.UTF-8",
		"LC_CTYPE=UTF8",
	})
	got := envSliceToMap(env)
	if got["LANG"] != "zh_CN.UTF-8" {
		t.Fatalf("LANG changed unexpectedly: %q", got["LANG"])
	}
	if got["LC_ALL"] != "zh_CN.UTF-8" {
		t.Fatalf("LC_ALL changed unexpectedly: %q", got["LC_ALL"])
	}
	if got["LC_CTYPE"] != "UTF8" {
		t.Fatalf("LC_CTYPE changed unexpectedly: %q", got["LC_CTYPE"])
	}
}

func envSliceToMap(env []string) map[string]string {
	out := make(map[string]string, len(env))
	for _, pair := range env {
		idx := strings.IndexByte(pair, '=')
		if idx <= 0 {
			continue
		}
		out[pair[:idx]] = pair[idx+1:]
	}
	return out
}
