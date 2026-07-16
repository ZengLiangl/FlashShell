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

func TestShellSessionPool_nextSessionIDReuse(t *testing.T) {
	pool := NewShellSessionPool()
	for _, id := range []string{"web1", "web1-2"} {
		sm := NewShellSessionManager()
		sm.sessionID = id
		sm.configName = "web1"
		pool.sessions[id] = sm
	}

	if got := pool.nextSessionID("web1"); got != "web1-3" {
		t.Fatalf("got %q want web1-3", got)
	}

	delete(pool.sessions, "web1-2")
	if got := pool.nextSessionID("web1"); got != "web1-2" {
		t.Fatalf("after close got %q want web1-2", got)
	}
}

func TestLocalShellPool_nextIDReuse(t *testing.T) {
	pool := NewLocalShellPool()
	pool.sessions["local"] = NewLocalShellSession("local")
	pool.sessions["local-2"] = NewLocalShellSession("local-2")

	if got := pool.nextID(); got != "local-3" {
		t.Fatalf("got %q want local-3", got)
	}

	delete(pool.sessions, "local-2")
	if got := pool.nextID(); got != "local-2" {
		t.Fatalf("after close got %q want local-2", got)
	}
}
