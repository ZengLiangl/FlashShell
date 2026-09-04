package mcp

import (
	"strings"
	"testing"
)

func TestBuiltinRedactRulesCount(t *testing.T) {
	if len(builtinRedactRules) < 20 {
		t.Fatalf("builtin rules=%d want >=20", len(builtinRedactRules))
	}
}

func TestRedactRulesAWSAndGitHub(t *testing.T) {
	text := "AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\nghp_abcdefghijklmnopqrstuvwxyz0123456789\n"
	hits := TestRedactRules(text)
	var names []string
	for _, h := range hits {
		names = append(names, h.Rule)
	}
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "aws_secret_key") {
		t.Fatalf("missing aws_secret_key in %v", names)
	}
	if !strings.Contains(joined, "github_token") {
		t.Fatalf("missing github_token in %v", names)
	}
}

func TestApplyRedactPlaceholder(t *testing.T) {
	out := applyRedact("password=SuperSecret99", nil)
	if strings.Contains(out, "SuperSecret99") {
		t.Fatalf("not redacted: %s", out)
	}
	if !strings.Contains(out, "[REDACTED:generic_password_line]") {
		t.Fatalf("unexpected: %s", out)
	}
}
