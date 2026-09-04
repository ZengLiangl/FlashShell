package mcp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const (
	PolicyDisabled  = "disabled"
	PolicyReadonly  = "readonly"
	PolicyApproval  = "approval"
	PolicyAllowlist = "allowlist"
	PolicyTrusted   = "trusted"
)

const (
	kindMeta     = "meta"
	kindReadonly = "readonly"
	kindMutating = "mutating"
)

type toolClass struct {
	kind string
}

var toolKinds = map[string]string{
	"list_servers":             kindMeta,
	"list_skills":              kindMeta,
	"get_skill":                kindMeta,
	"evaluate_skills":          kindMeta,
	"recall_experience":        kindMeta,
	"list_runbooks":            kindMeta,
	"list_installed_services":  kindMeta,
	"list_deploy_history":      kindMeta,
	"system_info":              kindReadonly,
	"disk_usage":               kindReadonly,
	"port_check":               kindReadonly,
	"service_status":           kindReadonly,
	"tail_log":                 kindReadonly,
	"sftp_list":                kindReadonly,
	"sftp_read":                kindReadonly,
	"cert_list":                kindReadonly,
	"web_status":               kindReadonly,
	"web_list_sites":           kindReadonly,
	"deploy_dry_run":           kindReadonly,
	"ssh_exec":                 kindMutating,
	"ssh_exec_script":          kindMutating,
	"ssh_exec_multi":           kindMutating,
	"sftp_write":               kindMutating,
	"sftp_upload":              kindMutating,
	"write_from_vault":         kindMutating,
	"install_app":              kindMutating,
	"install_with_secret":      kindMutating,
	"save_credential":          kindMutating,
	"delete_installed_service": kindMutating,
	"deploy_upsert_target":     kindMutating,
	"deploy_run":               kindMutating,
	"web_install_openresty":    kindMutating,
	"web_create_proxy":         kindMutating,
	"web_create_static":        kindMutating,
	"web_issue_ssl":            kindMutating,
	"run_runbook":              kindMutating,
}

func normalizePolicy(p string) string {
	switch strings.ToLower(strings.TrimSpace(p)) {
	case PolicyDisabled, PolicyReadonly, PolicyApproval, PolicyAllowlist, PolicyTrusted:
		return strings.ToLower(strings.TrimSpace(p))
	default:
		return PolicyTrusted
	}
}

var dangerCommandRules = []struct {
	re  *regexp.Regexp
	why string
}{
	{regexp.MustCompile(`(?i)rm\s+(-[a-zA-Z]*f[a-zA-Z]*|--force).*/\s*$`), "命中危险黑名单: rm -rf /"},
	{regexp.MustCompile(`(?i)rm\s+-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+(/|/\*|/\s+\*)`), "命中危险黑名单: rm -rf /"},
	{regexp.MustCompile(`(?i)rm\s+-[a-zA-Z]*f[a-zA-Z]*r[a-zA-Z]*\s+(/|/\*)`), "命中危险黑名单: rm -rf /"},
	{regexp.MustCompile(`(?i)mkfs(\.\w+)?\s`), "命中危险黑名单: mkfs"},
	{regexp.MustCompile(`(?i)dd\s+.*\bof=/dev/`), "命中危险黑名单: dd of=/dev"},
	{regexp.MustCompile(`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;`), "命中危险黑名单: fork bomb"},
	{regexp.MustCompile(`(?i)\b(shutdown|poweroff|halt|init\s+0)\b`), "命中危险黑名单: 关机/断电"},
	{regexp.MustCompile(`(?i)\bmkfs\b`), "命中危险黑名单: mkfs"},
	{regexp.MustCompile(`(?i)>\s*/dev/sd[a-z]`), "命中危险黑名单: 覆写块设备"},
	{regexp.MustCompile(`(?i)chmod\s+(-R\s+)?777\s+/`), "命中危险黑名单: chmod 777 /"},
	{regexp.MustCompile(`(?i)chown\s+-R\s+[^\s]+\s+/(\s|$)`), "命中危险黑名单: chown -R … /"},
	{regexp.MustCompile(`(?i)\b(wipefs|sgdisk|parted)\b.*\s/dev/`), "命中危险黑名单: 分区/擦除磁盘"},
	{regexp.MustCompile(`(?i)curl\s+[^\n]*\|\s*(ba)?sh`), "命中危险黑名单: curl|sh 管道执行"},
	{regexp.MustCompile(`(?i)wget\s+[^\n]*\|\s*(ba)?sh`), "命中危险黑名单: wget|sh 管道执行"},
	{regexp.MustCompile(`(?i)\bDROP\s+(DATABASE|SCHEMA)\b`), "命中危险黑名单: DROP DATABASE/SCHEMA"},
	{regexp.MustCompile(`(?i)\bTRUNCATE\s+(TABLE\s+)?`), "命中危险黑名单: TRUNCATE"},
	{regexp.MustCompile(`(?i)\bFLUSHALL\b`), "命中危险黑名单: FLUSHALL"},
	{regexp.MustCompile(`(?i)\bFLUSHDB\b`), "命中危险黑名单: FLUSHDB"},
}

// DangerRule 危险黑名单规则（内置只读展示 / 自定义可编辑）
type DangerRule struct {
	Pattern string `json:"pattern"`
	Label   string `json:"label"`
	Kind    string `json:"kind"` // command | path
}

func stripDangerWhyPrefix(why string) string {
	for _, p := range []string{"命中危险黑名单: ", "命中致命黑名单: ", "命中自定义危险黑名单: "} {
		if strings.HasPrefix(why, p) {
			return strings.TrimPrefix(why, p)
		}
	}
	return why
}

func builtinSensitivePathRules() []DangerRule {
	return []DangerRule{
		{Pattern: "/etc/shadow", Label: "敏感路径禁止读写", Kind: "path"},
		{Pattern: "/etc/sudoers", Label: "敏感路径禁止读写", Kind: "path"},
		{Pattern: "/etc/passwd", Label: "敏感路径禁止读写", Kind: "path"},
		{Pattern: "/etc/sudoers.d/*", Label: "敏感路径禁止读写", Kind: "path"},
		{Pattern: "/boot/*", Label: "敏感路径禁止读写", Kind: "path"},
		{Pattern: "~/.ssh 与 */.ssh/*", Label: "敏感路径禁止读写", Kind: "path"},
		{Pattern: "*.pem / *.key / *.env / *.p12 / *.pfx", Label: "敏感路径禁止读写", Kind: "path"},
	}
}

// ListBuiltinDangerRules 返回内置危险黑名单（命令正则 + 敏感路径），只读展示用。
func ListBuiltinDangerRules() []DangerRule {
	seen := map[string]struct{}{}
	out := make([]DangerRule, 0, len(dangerCommandRules)+len(lethalCommandRules)+8)
	appendRule := func(pattern, why, kind string) {
		key := kind + "\x00" + pattern
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, DangerRule{
			Pattern: pattern,
			Label:   stripDangerWhyPrefix(why),
			Kind:    kind,
		})
	}
	for _, r := range lethalCommandRules {
		appendRule(r.re.String(), r.why, "command")
	}
	for _, r := range dangerCommandRules {
		appendRule(r.re.String(), r.why, "command")
	}
	for _, r := range builtinSensitivePathRules() {
		appendRule(r.Pattern, r.Label, r.Kind)
	}
	return out
}

func commandBlocked(cmd string) (bool, string) {
	s := strings.TrimSpace(cmd)
	if s == "" {
		return false, ""
	}
	for _, rule := range dangerCommandRules {
		if rule.re.MatchString(s) {
			return true, rule.why
		}
	}
	if why := matchCustomDangerDetail(s); why != "" {
		return true, why
	}
	return false, ""
}

func matchCustomDangerDetail(cmd string) string {
	st := loadSettings()
	for _, p := range st.CustomDangerPatterns {
		re, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		if re.MatchString(cmd) {
			return fmt.Sprintf("命中自定义危险黑名单: %s", p)
		}
	}
	return ""
}

func containsSudo(cmd string) bool {
	re := regexp.MustCompile(`(?:^|[;&|` + "`" + `(\s])sudo(?:\s|$)`)
	return re.MatchString(cmd)
}

func pathBlocked(p string) bool {
	n := strings.TrimSpace(p)
	if n == "" {
		return false
	}
	low := strings.ToLower(n)
	base := strings.ToLower(filepath.Base(n))
	blockedExact := []string{
		"/etc/shadow", "/etc/sudoers", "/etc/passwd",
	}
	for _, b := range blockedExact {
		if low == b {
			return true
		}
	}
	if strings.Contains(low, "/etc/sudoers.d/") {
		return true
	}
	if strings.HasPrefix(low, "/boot/") || low == "/boot" {
		return true
	}
	if strings.Contains(low, "/.ssh/") || strings.HasSuffix(low, "/.ssh") {
		return true
	}
	for _, ext := range []string{".pem", ".key", ".env", ".p12", ".pfx"} {
		if strings.HasSuffix(base, ext) {
			return true
		}
	}
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	return false
}

func safePathChars(p string) bool {
	if p == "" {
		return true
	}
	for _, r := range p {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '/', '.', '_', '-', '+', '~', ':':
			continue
		default:
			return false
		}
	}
	return true
}

func safeServiceName(n string) bool {
	re := regexp.MustCompile(`^[A-Za-z0-9._\-@]+$`)
	return re.MatchString(n)
}

func policyDeny(policy, kind, command string, allowlist []string) (prefix string, msg string) {
	r := policyDecide(policy, kind, command, allowlist)
	if r.Allow {
		return "", ""
	}
	if r.NeedsApprove {
		return "[approval]", r.Reason
	}
	if r.Prefix != "" {
		return r.Prefix, r.Reason
	}
	return "[denied]", r.Reason
}

func commandAllowed(command string, allowlist []string) bool {
	return commandAllowedRegex(command, allowlist)
}

func blockedErr(prefix, msg string) error {
	return wrapErr(prefix, msg)
}
