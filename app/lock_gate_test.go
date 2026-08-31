package app

import (
	"testing"
)

func TestListVaultGuardedMethodsNonEmpty(t *testing.T) {
	a := &App{}
	list := a.ListVaultGuardedMethods()
	if len(list) < 38 {
		t.Fatalf("守护方法过少: got %d, want >= 38", len(list))
	}
	seen := map[string]bool{}
	for _, n := range list {
		if n == "" {
			t.Fatal("空方法名")
		}
		if seen[n] {
			t.Fatalf("重复方法名: %s", n)
		}
		seen[n] = true
	}
	required := []string{
		"ConnectShell",
		"TestMachineConnection",
		"GetMachineSensitiveData",
		"SaveGlobalConfig",
		"SaveSystemSettings",
		"ChangeVaultMasterPassword",
		"StartShellUpload",
		"ListShellFiles",
	}
	for _, n := range required {
		if !seen[n] {
			t.Fatalf("缺少守护方法: %s", n)
		}
	}
	// 解锁 / 忘记重置 / 只读不得在清单里
	forbidden := []string{
		"UnlockVault",
		"ResetVaultForgotMasterPassword",
		"GetVaultStatus",
		"GetMachines",
		"GetGlobalConfig",
		"QueryMCPAudit",
		"ConnectLocalShell",
	}
	for _, n := range forbidden {
		if seen[n] {
			t.Fatalf("不应守护只读/解锁路径: %s", n)
		}
	}
}
