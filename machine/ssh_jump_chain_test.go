package machine

import (
	"testing"

	"FlashDock/define"
)

func TestResolveJumpChainFallbackProxyJump(t *testing.T) {
	m := &define.Machine{
		Name:      "target",
		ProxyJump: "bastion",
	}
	hops, err := ResolveJumpChain(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(hops) != 1 {
		t.Fatalf("expected 1 hop, got %d", len(hops))
	}
}

func TestResolveJumpChainOrdered(t *testing.T) {
	m := &define.Machine{
		Name:      "target",
		JumpChain: []string{"hop1", "hop2"},
		ProxyJump: "ignored",
	}
	hops, err := ResolveJumpChain(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(hops) != 2 {
		t.Fatalf("expected 2 hops, got %d", len(hops))
	}
}
