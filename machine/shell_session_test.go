package machine

import (
	"testing"

	"FlashDock/define"
)

func TestShellSessionManager_GetStatusDisconnected(t *testing.T) {
	sm := NewShellSessionManager()
	status := sm.GetStatus()
	if status.Connected {
		t.Fatal("expected disconnected")
	}
}

func TestShellSessionManager_SendInputWithoutConnect(t *testing.T) {
	sm := NewShellSessionManager()
	err := sm.SendInput("ls\n")
	if err == nil {
		t.Fatal("expected error when not connected")
	}
}

func TestShellSessionManager_DisconnectWhenNotConnected(t *testing.T) {
	sm := NewShellSessionManager()
	if err := sm.Disconnect(ShellOutputHandler{}); err != nil {
		t.Fatalf("disconnect should succeed: %v", err)
	}
}

func TestShellStatusFields(t *testing.T) {
	status := &define.ShellStatus{
		Connected:   true,
		MachineName: "test",
		Host:        "127.0.0.1",
		User:        "root",
	}
	if !status.Connected || status.MachineName != "test" {
		t.Fatal("unexpected shell status")
	}
}
