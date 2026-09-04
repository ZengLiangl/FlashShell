package mcp

import "testing"

func TestSubstituteVaultPlaceholders(t *testing.T) {
	v := &Vault{items: []VaultItem{{
		ID:          "vs_demo01",
		ServerAlias: "web1",
		Kind:        "token",
		Label:       "demo",
		Secrets:     map[string]string{"token": "SuperSecretToken99"},
	}}}
	s := &Service{vault: v}
	out, used, err := s.SubstituteVaultPlaceholders("key={{vault:vs_demo01.token}}\n")
	if err != nil {
		t.Fatal(err)
	}
	if out != "key=SuperSecretToken99\n" {
		t.Fatalf("out=%q", out)
	}
	if len(used) != 1 || used[0] != "SuperSecretToken99" {
		t.Fatalf("used=%v", used)
	}
	_, _, err = s.SubstituteVaultPlaceholders("{{vault:vs_missing.token}}")
	if err == nil {
		t.Fatal("expected missing error")
	}
}

func TestForceRedactPlains(t *testing.T) {
	got := forceRedactPlains("pass=abc123 end", []string{"abc123"})
	if got != "pass=[REDACTED:vault] end" {
		t.Fatalf("got=%q", got)
	}
}
