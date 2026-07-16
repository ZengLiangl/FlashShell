package data

import (
	"os"
	"testing"
)

func TestSessionManager_Delete(t *testing.T) {
	dir := t.TempDir()
	sm := &SessionManager{
		baseDir: dir,
		state:   &SessionState{SessionID: "test-session"},
	}
	if err := sm.save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	path := sm.sessionPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat before delete: %v", err)
	}
	if err := sm.Delete(); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("want not exist after delete, got %v", err)
	}
	if err := sm.Delete(); err != nil {
		t.Fatalf("delete again: %v", err)
	}
}
