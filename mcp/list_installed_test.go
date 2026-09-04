package mcp

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestListInstalledServicesFilters(t *testing.T) {
	v := &Vault{items: []VaultItem{
		{ID: "vs_a", ServerAlias: "cs", Kind: "token", Label: "stripe"},
		{ID: "vs_b", ServerAlias: "h", Kind: "credential", Label: "root"},
		{ID: "vs_c", ServerAlias: "", Kind: "credential", Label: "shared-pass"},
		{ID: "vs_d", ServerAlias: "other", Kind: "mysql_conn", Label: "db"},
	}}
	s := &Service{vault: v}
	ctx := context.WithValue(context.Background(), ctxKeyToken{}, Token{
		Servers: []string{"cs", "h"},
	})

	_, err := s.handleListInstalledServices(ctx, ListInstalledArgs{})
	if err == nil || !strings.Contains(err.Error(), "缩小范围") {
		t.Fatalf("empty filter must be rejected, got %v", err)
	}

	svr := "cs"
	kind := "token"
	out2, err := s.handleListInstalledServices(ctx, ListInstalledArgs{Server: &svr, Kind: &kind})
	if err != nil {
		t.Fatal(err)
	}
	items2 := out2.(map[string]any)["items"].([]map[string]any)
	if len(items2) != 1 || fmt.Sprint(items2[0]["id"]) != "vs_a" {
		t.Fatalf("kind+server filter: %+v", items2)
	}

	id := "vs_c"
	out3, err := s.handleListInstalledServices(ctx, ListInstalledArgs{ID: &id})
	if err != nil {
		t.Fatal(err)
	}
	items3 := out3.(map[string]any)["items"].([]map[string]any)
	if len(items3) != 1 || fmt.Sprint(items3[0]["id"]) != "vs_c" {
		t.Fatalf("id filter: %+v", items3)
	}

	label := "shared"
	out4, err := s.handleListInstalledServices(ctx, ListInstalledArgs{Label: &label})
	if err != nil {
		t.Fatal(err)
	}
	items4 := out4.(map[string]any)["items"].([]map[string]any)
	if len(items4) != 1 || fmt.Sprint(items4[0]["id"]) != "vs_c" {
		t.Fatalf("label filter: %+v", items4)
	}

	hidden := "vs_d"
	out5, err := s.handleListInstalledServices(ctx, ListInstalledArgs{ID: &hidden})
	if err != nil {
		t.Fatal(err)
	}
	items5 := out5.(map[string]any)["items"].([]map[string]any)
	if len(items5) != 0 {
		t.Fatalf("token scope must hide other server even by id: %+v", items5)
	}
}
