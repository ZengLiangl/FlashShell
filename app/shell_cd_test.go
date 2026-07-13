package app

import "testing"

func TestApplyShellCdLogic_KeepWhenMissing(t *testing.T) {
	// 模拟 ApplyShellCd 的核心分支（无网络）：解析后按 exists 选择
	home := "/root"
	current := "/root"

	resolved, err := ResolveRemotePath(current, "app/", home)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != "/root/app" {
		t.Fatalf("resolve=%q", resolved)
	}

	// 不存在 → 保留
	if got := ChooseCdPath(current, resolved, false); got != "/root" {
		t.Fatalf("missing keep=%q", got)
	}
	// 存在 → 跳转
	if got := ChooseCdPath(current, resolved, true); got != "/root/app" {
		t.Fatalf("exists go=%q", got)
	}
}

func TestApplyShellCdLogic_RelativeNested(t *testing.T) {
	home := "/root"
	current := "/root/app"
	resolved, err := ResolveRemotePath(current, "auth-service/", home)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != "/root/app/auth-service" {
		t.Fatalf("got %q want /root/app/auth-service", resolved)
	}
	if got := ChooseCdPath(current, resolved, true); got != "/root/app/auth-service" {
		t.Fatalf("exists go=%q", got)
	}
	if got := ChooseCdPath(current, resolved, false); got != "/root/app" {
		t.Fatalf("missing keep=%q", got)
	}
}
