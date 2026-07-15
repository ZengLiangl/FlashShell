package machine

import (
	"strings"
	"testing"
)

func TestShellCwdHookScriptContent(t *testing.T) {
	if !strings.Contains(shellCwdHookScript, "777;cwd") {
		t.Fatal("hook should emit OSC 777 cwd")
	}
	if !strings.Contains(shellCwdHookScript, "cd()") {
		t.Fatal("hook should wrap cd")
	}
	if !strings.Contains(shellCwdHookScript, ".flashdock_pwd") {
		t.Fatal("hook should write sidecar pwd file")
	}
	if shellRcMarker != "flashdock-cwd-hook" {
		t.Fatal("unexpected rc marker")
	}
}
