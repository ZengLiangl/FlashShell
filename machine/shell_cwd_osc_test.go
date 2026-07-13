package machine

import (
	"bytes"
	"testing"
)

func TestParseOsc777Cwd(t *testing.T) {
	cwd, ok := parseOscCwdPayload([]byte("777;cwd;/root/app/auth-service"))
	if !ok || cwd != "/root/app/auth-service" {
		t.Fatalf("got %q ok=%v", cwd, ok)
	}
}

func TestParseOsc7FileURI(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"7;file://localhost/root/app", "/root/app"},
		{"7;file:///tmp/foo", "/tmp/foo"},
		{"7;file://host.example.com/var/log", "/var/log"},
	}
	for _, c := range cases {
		cwd, ok := parseOscCwdPayload([]byte(c.in))
		if !ok || cwd != c.want {
			t.Fatalf("in=%q got %q ok=%v want %q", c.in, cwd, ok, c.want)
		}
	}
}

func TestOscCwdFilter_StripAndEmit(t *testing.T) {
	var got []string
	f := newOscCwdFilter(func(cwd string) { got = append(got, cwd) })

	raw := []byte("hello\033]777;cwd;/root/app\007world")
	out := f.Feed(raw)
	if string(out) != "helloworld" {
		t.Fatalf("out=%q", out)
	}
	if len(got) != 1 || got[0] != "/root/app" {
		t.Fatalf("cwd events=%v", got)
	}
}

func TestOscCwdFilter_SplitAcrossChunks(t *testing.T) {
	var got []string
	f := newOscCwdFilter(func(cwd string) { got = append(got, cwd) })

	part1 := []byte("x\033]777;cwd;/roo")
	part2 := []byte("t/app\007y")
	out1 := f.Feed(part1)
	out2 := f.Feed(part2)
	combined := string(out1) + string(out2)
	if combined != "xy" {
		t.Fatalf("combined=%q", combined)
	}
	if len(got) != 1 || got[0] != "/root/app" {
		t.Fatalf("cwd=%v", got)
	}
}

func TestOscCwdFilter_OSC7_ST(t *testing.T) {
	var got []string
	f := newOscCwdFilter(func(cwd string) { got = append(got, cwd) })
	raw := bytes.Join([][]byte{
		[]byte("a"),
		[]byte("\033]7;file://localhost/tmp\033\\"),
		[]byte("b"),
	}, nil)
	out := f.Feed(raw)
	if string(out) != "ab" {
		t.Fatalf("out=%q", out)
	}
	if len(got) != 1 || got[0] != "/tmp" {
		t.Fatalf("cwd=%v", got)
	}
}
