package mcp

import (
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
