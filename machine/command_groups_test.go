package machine

import (
	"testing"

	"FlashDock/define"
)

func TestGroupParallelCommands(t *testing.T) {
	commands := []define.Command{
		{Name: "a"},
		{Name: "b", Parallel: true},
		{Name: "c", Parallel: true},
		{Name: "d"},
		{Name: "e", Parallel: true},
	}
	groups := GroupParallelCommands(commands)
	if len(groups) != 4 {
		t.Fatalf("expected 4 groups, got %d", len(groups))
	}
	if len(groups[0]) != 1 || groups[0][0].Name != "a" {
		t.Fatalf("group0: %+v", groups[0])
	}
	if len(groups[1]) != 2 || groups[1][0].Name != "b" || groups[1][1].Name != "c" {
		t.Fatalf("group1: %+v", groups[1])
	}
	if len(groups[2]) != 1 || groups[2][0].Name != "d" {
		t.Fatalf("group2: %+v", groups[2])
	}
	if len(groups[3]) != 1 || groups[3][0].Name != "e" {
		t.Fatalf("group3: %+v", groups[3])
	}
}
