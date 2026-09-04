package mcp

import (
	"fmt"
	"regexp"
	"strings"
)

var vaultPlaceholderRe = regexp.MustCompile(`\{\{vault:([A-Za-z0-9_-]+)(?:\.([A-Za-z0-9_]+))?\}\}`)

// SubstituteVaultPlaceholders 将 {{vault:id}} / {{vault:id.field}} 替换为服务凭据明文。
// 任一占位失败则整单失败（不做半替换）。返回替换后文本与用过的明文（供本轮强制脱敏）。
func (s *Service) SubstituteVaultPlaceholders(text string) (out string, used []string, err error) {
	if s == nil || s.vault == nil {
		return text, nil, fmt.Errorf("凭据库未初始化")
	}
	if !strings.Contains(text, "{{vault:") {
		return text, nil, nil
	}
	matches := vaultPlaceholderRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text, nil, nil
	}
	var b strings.Builder
	last := 0
	seenPlain := map[string]struct{}{}
	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		b.WriteString(text[last:m[0]])
		id := text[m[2]:m[3]]
		field := ""
		if m[4] >= 0 && m[5] >= 0 {
			field = text[m[4]:m[5]]
		}
		plain, err := s.vault.RevealField(id, field)
		if err != nil {
			return "", nil, err
		}
		b.WriteString(plain)
		if _, ok := seenPlain[plain]; !ok && plain != "" {
			seenPlain[plain] = struct{}{}
			used = append(used, plain)
		}
		last = m[1]
	}
	b.WriteString(text[last:])
	return b.String(), used, nil
}

// forceRedactPlains 把已知明文从文本里抠掉（防 cat/echo 回显）。
func forceRedactPlains(text string, plains []string) string {
	out := text
	for _, p := range plains {
		if p == "" {
			continue
		}
		out = strings.ReplaceAll(out, p, "[REDACTED:vault]")
	}
	return out
}
