package mcp

import (
	"regexp"
	"testing"
)

func TestCustomDangerPatternCompile(t *testing.T) {
	p := `(?i)rm\s+-rf\s+/data/`
	re, err := regexp.Compile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("rm -rf /data/backup") {
		t.Fatal("pattern should match")
	}
}
