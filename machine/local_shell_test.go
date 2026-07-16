package machine

import (
	"os"
	"testing"
)

func TestLocalShellStartDir(t *testing.T) {
	dir := localShellStartDir()
	if dir == "" {
		t.Fatal("localShellStartDir returned empty")
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat %q: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("localShellStartDir %q is not a directory", dir)
	}
}
