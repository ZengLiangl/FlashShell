package mcp

import (
	"strings"
	"testing"
)

func TestSanitizeAuditParamsRedactsSecrets(t *testing.T) {
	raw := sanitizeAuditParams(SftpWriteArgs{
		Server:  "h",
		Path:    "/tmp/a",
		Content: strPtr("super-secret"),
		Intent:  strPtr("写配置"),
	})
	if strings.Contains(raw, "super-secret") {
		t.Fatalf("content leaked: %s", raw)
	}
	if !strings.Contains(raw, "写配置") {
		t.Fatalf("intent should remain: %s", raw)
	}
}

func TestNormalizeOSName(t *testing.T) {
	if normalizeOSName("windows_nt") != "Windows" {
		t.Fatal(normalizeOSName("windows_nt"))
	}
	if normalizeOSName("Linux") != "Linux" {
		t.Fatal(normalizeOSName("Linux"))
	}
}

func TestCustomDangerGoesToApprovalNotBlock(t *testing.T) {
	if hit, _ := lethalBlocked("please-custom-xyz"); hit {
		t.Fatal("lethal should ignore unmatched")
	}
	if hit, _ := commandBlocked("please-custom-xyz"); hit {
		t.Fatal("builtin danger should ignore")
	}
	// 自定义规则走 severeNeedsApproval，不进 commandBlocked/lethalBlocked
	if hit, why := severeNeedsApproval("shutdown -h now"); !hit || why == "" {
		t.Fatalf("severe should catch shutdown: hit=%v why=%q", hit, why)
	}
}

func strPtr(s string) *string { return &s }
