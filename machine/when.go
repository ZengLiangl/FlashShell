package machine

import (
	"fmt"
	"strconv"
	"strings"
)

// EvaluateWhen 评估步骤 when 表达式；空表达式视为 true。
// 支持：true/false、${var}/$var、==、!=、&&、||、!，以及裸变量名（非空为真）。
func EvaluateWhen(expr string, vars map[string]string) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true, nil
	}
	expr = expandWhenVars(expr, vars)
	return evalWhenOr(expr)
}

func expandWhenVars(expr string, vars map[string]string) string {
	if len(vars) == 0 {
		return expr
	}
	out := expr
	for k, v := range vars {
		out = strings.ReplaceAll(out, "${"+k+"}", v)
		out = strings.ReplaceAll(out, "$"+k, v)
	}
	return out
}

func evalWhenOr(expr string) (bool, error) {
	parts := splitWhenTop(expr, "||")
	if len(parts) > 1 {
		for _, part := range parts {
			ok, err := evalWhenAnd(part)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}
	return evalWhenAnd(expr)
}

func evalWhenAnd(expr string) (bool, error) {
	parts := splitWhenTop(expr, "&&")
	if len(parts) > 1 {
		for _, part := range parts {
			ok, err := evalWhenUnary(part)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	}
	return evalWhenUnary(expr)
}

func evalWhenUnary(expr string) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true, nil
	}
	negated := false
	for strings.HasPrefix(expr, "!") {
		negated = !negated
		expr = strings.TrimSpace(expr[1:])
	}
	ok, err := evalWhenAtom(expr)
	if err != nil {
		return false, err
	}
	if negated {
		return !ok, nil
	}
	return ok, nil
}

func evalWhenAtom(expr string) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true, nil
	}
	lower := strings.ToLower(expr)
	switch lower {
	case "true", "yes", "1":
		return true, nil
	case "false", "no", "0":
		return false, nil
	}
	if strings.Contains(expr, "==") {
		parts := strings.SplitN(expr, "==", 2)
		return strings.TrimSpace(parts[0]) == strings.TrimSpace(parts[1]), nil
	}
	if strings.Contains(expr, "!=") {
		parts := strings.SplitN(expr, "!=", 2)
		return strings.TrimSpace(parts[0]) != strings.TrimSpace(parts[1]), nil
	}
	if b, err := strconv.ParseBool(expr); err == nil {
		return b, nil
	}
	// 裸字符串：非空为真
	return expr != "", nil
}

func splitWhenTop(expr, sep string) []string {
	var parts []string
	var buf strings.Builder
	depth := 0
	quote := byte(0)
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if quote != 0 {
			buf.WriteByte(ch)
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			buf.WriteByte(ch)
			continue
		}
		if ch == '(' {
			depth++
			buf.WriteByte(ch)
			continue
		}
		if ch == ')' && depth > 0 {
			depth--
			buf.WriteByte(ch)
			continue
		}
		if depth == 0 && strings.HasPrefix(expr[i:], sep) {
			parts = append(parts, buf.String())
			buf.Reset()
			i += len(sep) - 1
			continue
		}
		buf.WriteByte(ch)
	}
	parts = append(parts, buf.String())
	if len(parts) == 1 {
		return parts
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// FormatWhenResult 干跑展示用
func FormatWhenResult(expr string, vars map[string]string) (bool, string, error) {
	ok, err := EvaluateWhen(expr, vars)
	if err != nil {
		return false, "", err
	}
	if strings.TrimSpace(expr) == "" {
		return ok, "（无条件）", nil
	}
	label := fmt.Sprintf("%s => %v", expandWhenVars(expr, vars), ok)
	return ok, label, nil
}
