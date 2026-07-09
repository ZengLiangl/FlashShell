package machine

import "testing"

func TestShellSessionPool_ListEmpty(t *testing.T) {
	pool := NewShellSessionPool()
	if len(pool.ListSessions()) != 0 {
		t.Fatal("expected empty sessions")
	}
}

func TestShellSessionPool_IsAnyConnectedFalse(t *testing.T) {
	pool := NewShellSessionPool()
	if pool.IsAnyConnected() {
		t.Fatal("expected no connection")
	}
}

func TestShellSessionPool_SendInputWithoutSession(t *testing.T) {
	pool := NewShellSessionPool()
	err := pool.SendInput("test", "ls\n")
	if err == nil {
		t.Fatal("expected error")
	}
}
