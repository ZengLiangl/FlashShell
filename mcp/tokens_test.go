package mcp

import (
	"os"
	"strings"
	"testing"
)

func TestNormalizeCIDRRejectsPublic(t *testing.T) {
	_, err := normalizeCIDRs([]string{"0.0.0.0/0"})
	if err == nil {
		t.Fatal("expected public cidr rejected")
	}
	out, err := normalizeCIDRs([]string{"127.0.0.1/32", "192.168.1.0/24"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("got %v", out)
	}
	if !ipAllowed("127.0.0.1:1234", out) {
		t.Fatal("loopback should allow")
	}
	if ipAllowed("8.8.8.8", out) {
		t.Fatal("public ip should deny")
	}
}

func TestTokenSeesServer(t *testing.T) {
	tok := Token{Servers: []string{"web1"}}
	if !tok.SeesServer("web1") || tok.SeesServer("db1") {
		t.Fatal("scope filter failed")
	}
	all := Token{}
	if !all.SeesServer("anything") {
		t.Fatal("empty servers = all")
	}
}

func TestTokenUpdateKeepsHash(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLASHSHELL_CONFIG_HOME", dir)

	st := loadTokens()
	issued, err := st.Issue(IssueOpts{Name: "t1", Client: "manual", Servers: []string{"a"}, CIDRs: []string{"127.0.0.1/32"}})
	if err != nil {
		t.Fatal(err)
	}
	hashBefore := ""
	st.mu.Lock()
	for _, t0 := range st.tokens {
		if t0.ID == issued.ID {
			hashBefore = t0.Hash
		}
	}
	st.mu.Unlock()
	if hashBefore == "" {
		t.Fatal("missing hash after issue")
	}
	updated, err := st.Update(UpdateTokenOpts{ID: issued.ID, Name: "t2", Servers: []string{"b", "c"}, CIDRs: []string{"127.0.0.1/32"}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "t2" || len(updated.Servers) != 2 {
		t.Fatalf("update failed: %+v", updated)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, t0 := range st.tokens {
		if t0.ID == issued.ID {
			if t0.Hash != hashBefore {
				t.Fatal("hash should not rotate on update")
			}
			return
		}
	}
	t.Fatal("token missing")
}

func TestTokenValidFromDoesNotClobberDiskServers(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLASHSHELL_CONFIG_HOME", dir)

	ui := loadTokens()
	issued, err := ui.Issue(IssueOpts{Name: "t", Client: "cursor", Servers: []string{"a", "b"}, CIDRs: []string{"127.0.0.1/32"}})
	if err != nil {
		t.Fatal(err)
	}
	plain := issued.Plaintext

	stdio := loadTokens()
	stdio.mu.Lock()
	for i := range stdio.tokens {
		if stdio.tokens[i].ID == issued.ID {
			stdio.tokens[i].Servers = []string{"a", "b"}
		}
	}
	stdio.mu.Unlock()

	if _, err := ui.Update(UpdateTokenOpts{ID: issued.ID, Servers: []string{"a"}, CIDRs: []string{"127.0.0.1/32"}}); err != nil {
		t.Fatal(err)
	}

	got, ok := stdio.ValidFrom(plain, "127.0.0.1")
	if !ok {
		t.Fatal("valid expected")
	}
	if len(got.Servers) != 1 || got.Servers[0] != "a" {
		t.Fatalf("stdio should see updated servers, got %+v", got.Servers)
	}
	reloaded := loadTokens()
	tok, ok := reloaded.Get(issued.ID)
	if !ok || len(tok.Servers) != 1 || tok.Servers[0] != "a" {
		t.Fatalf("disk clobbered: %+v", tok.Servers)
	}
}

func TestIssuedPlaintextNotPersistedOrListed(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FLASHSHELL_CONFIG_HOME", dir)

	st := loadTokens()
	issued, err := st.Issue(IssueOpts{Name: "t", Client: "manual", Servers: []string{"a"}, CIDRs: []string{"127.0.0.1/32"}})
	if err != nil {
		t.Fatal(err)
	}
	if issued.Plaintext == "" {
		t.Fatal("expected one-time plaintext")
	}

	path, err := tokensFilePath()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), issued.Plaintext) {
		t.Fatal("plaintext must not be written to tokens.yaml")
	}

	for _, row := range st.List() {
		if row.ID != issued.ID {
			continue
		}
		if row.Hash != "" {
			t.Fatal("list must strip hash")
		}
	}
	got, ok := st.Get(issued.ID)
	if !ok {
		t.Fatal("missing token")
	}
	if got.Hash != "" {
		t.Fatal("get must strip hash")
	}
	if _, ok := st.ValidFrom(issued.Plaintext, "127.0.0.1"); !ok {
		t.Fatal("hash validation should still work")
	}
}

func TestUpsertGuidanceBlock(t *testing.T) {
	block := guidanceBody([]string{"a"})
	next := upsertGuidanceBlock("hello\n", block)
	if !strings.Contains(next, guidanceBegin) || !strings.Contains(next, "hello") {
		t.Fatal(next)
	}
	next2 := upsertGuidanceBlock(next, guidanceBody([]string{"b"}))
	if strings.Count(next2, guidanceBegin) != 1 {
		t.Fatal("should replace once")
	}
}

func TestHashTokenStable(t *testing.T) {
	a := hashToken("abc")
	b := hashToken("abc")
	if a != b || a == "" {
		t.Fatal(a, b)
	}
}
