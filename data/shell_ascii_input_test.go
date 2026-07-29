package data

import "testing"

func TestShellAsciiInputEnabled(t *testing.T) {
	if !ShellAsciiInputEnabled(nil) {
		t.Fatal("nil config should default to true")
	}
	off := false
	on := true
	if ShellAsciiInputEnabled(&GlobalConfig{ShellAsciiInput: &off}) {
		t.Fatal("explicit false should disable")
	}
	if !ShellAsciiInputEnabled(&GlobalConfig{ShellAsciiInput: &on}) {
		t.Fatal("explicit true should enable")
	}
	if !ShellAsciiInputEnabled(&GlobalConfig{}) {
		t.Fatal("nil field should default to true")
	}
}
