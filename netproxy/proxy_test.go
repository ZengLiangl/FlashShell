package netproxy

import "testing"

func TestNormalizeKeepsAuth(t *testing.T) {
	s := Normalize(Settings{
		Mode:     ModeManual,
		Type:     TypeHTTP,
		Host:     "127.0.0.1",
		Port:     7890,
		User:     " alice ",
		Password: "p@ss",
	})
	if s.User != "alice" || s.Password != "p@ss" {
		t.Fatalf("auth not preserved: user=%q password=%q", s.User, s.Password)
	}
}

func TestNormalizeClearsAuthWhenDisabled(t *testing.T) {
	s := Normalize(Settings{
		Mode:     ModeNone,
		Type:     TypeHTTP,
		Host:     "127.0.0.1",
		Port:     7890,
		User:     "alice",
		Password: "p@ss",
	})
	if s.User != "" || s.Password != "" {
		t.Fatalf("auth should clear when mode=none: %+v", s)
	}
}
