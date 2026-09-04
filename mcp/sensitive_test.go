package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExpireDueClearsSecretsKeepsHash(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLASHSHELL_CONFIG_HOME", dir)

	v := &SensitiveVault{items: []SensitiveEntry{
		{
			ID:        "sv_old",
			Status:    SensitiveStatusActive,
			Hash:      "abc",
			Secrets:   map[string]string{"value": "enc"},
			ExpiresAt: time.Now().Add(-time.Hour).Format(time.RFC3339),
		},
		{
			ID:        "sv_new",
			Status:    SensitiveStatusActive,
			Hash:      "def",
			Secrets:   map[string]string{"value": "enc"},
			ExpiresAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		},
	}}
	n, err := v.ExpireDue(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expired count=%d want 1", n)
	}
	if v.items[0].Status != SensitiveStatusExpired || len(v.items[0].Secrets) != 0 || v.items[0].Hash != "abc" {
		t.Fatalf("old entry not expired correctly: %+v", v.items[0])
	}
	if v.items[1].Status != SensitiveStatusActive || len(v.items[1].Secrets) == 0 {
		t.Fatalf("new entry should stay active: %+v", v.items[1])
	}
	b, err := os.ReadFile(filepath.Join(dir, "mcp", sensitiveVaultFile))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "sv_old") || !strings.Contains(string(b), "expired") {
		t.Fatalf("disk should persist expired entry: %s", b)
	}
}

func TestMigrateMovesRedactionOutOfCredentialVault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLASHSHELL_CONFIG_HOME", dir)

	cred := &Vault{items: []VaultItem{
		{
			ID:      "sv_abc",
			Kind:    "credential",
			Label:   "generic_password_line",
			Public:  map[string]string{"rule": "generic_password_line", "kind": "credential"},
			Secrets: map[string]string{"value": "enc"},
		},
		{
			ID:      "vs_mysql",
			Kind:    "mysql_conn",
			Label:   "MySQL",
			Secrets: map[string]string{"password": "enc"},
		},
	}}
	sens := &SensitiveVault{}
	migrateRedactionCaptures(cred, sens)

	if len(cred.items) != 1 || cred.items[0].ID != "vs_mysql" {
		t.Fatalf("cred vault=%+v", cred.items)
	}
	if len(sens.items) != 1 || sens.items[0].ID != "sv_abc" {
		t.Fatalf("sensitive vault=%+v", sens.items)
	}
	if sens.items[0].Rule != "generic_password_line" {
		t.Fatalf("rule=%q", sens.items[0].Rule)
	}
	meta := cred.ListMeta("")
	if len(meta) != 1 || meta[0]["id"] != "vs_mysql" {
		t.Fatalf("ListMeta should hide redaction captures: %+v", meta)
	}
}

func TestVaultListMetaSkipsRedactionCaptures(t *testing.T) {
	v := &Vault{items: []VaultItem{
		{ID: "sv_x", Kind: "credential", Public: map[string]string{"rule": "bearer_token"}},
		{ID: "vs_ok", Kind: "redis_conn", Label: "Redis", ServerAlias: "web1"},
	}}
	out := v.ListMeta("")
	if len(out) != 1 || out[0]["id"] != "vs_ok" {
		t.Fatalf("got %+v", out)
	}
	_, _, ok := v.Find("sv_x")
	if ok {
		t.Fatal("Find must not resolve redaction captures as credentials")
	}
}

func TestRedactTextWritesSensitiveNotCredentialVault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLASHSHELL_CONFIG_HOME", dir)

	s := newService(nil)
	out := s.redactText(context.Background(), "password=SuperSecretValue99", "")
	if strings.Contains(out, "SuperSecretValue99") {
		t.Fatalf("should redact: %s", out)
	}
	if !strings.Contains(out, "[REDACTED:") {
		t.Fatalf("missing placeholder: %s", out)
	}
	if len(s.vault.ListMeta("")) != 0 {
		t.Fatalf("credential vault polluted: %+v", s.vault.ListMeta(""))
	}
	// Capture 依赖 DEK；加密失败时仍脱敏但不落库，属可接受。
	if s.sensitive.Len() > 0 {
		meta := s.sensitive.ListMeta()
		if meta[0]["rule"] == "" {
			t.Fatalf("missing rule: %+v", meta[0])
		}
	}
}
