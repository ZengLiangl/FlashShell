package data

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultOpenSSHConfigPath(t *testing.T) {
	path, err := DefaultOpenSSHConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".ssh", "config")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestParseOpenSSHConfigBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := `
Host jumpbox
  HostName 10.0.0.1
  User ops
  Port 2222
  IdentityFile ~/.ssh/id_ed25519

Host *
  User ignore

Host web app
  HostName 10.0.0.2
  User deploy
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	hosts, err := ParseOpenSSHConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 2 {
		t.Fatalf("hosts len = %d, want 2", len(hosts))
	}
	h := hosts[0]
	if h.Name != "jumpbox" || h.HostName != "10.0.0.1" || h.User != "ops" || h.Port != 2222 {
		t.Fatalf("unexpected host: %+v", h)
	}
	if h.IdentityFile == "" {
		t.Fatal("IdentityFile should be expanded")
	}
	if hosts[1].Name != "web" || hosts[1].HostName != "10.0.0.2" {
		t.Fatalf("unexpected second host: %+v", hosts[1])
	}
}
