package mcp

import (
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

var dangerCommand = []*regexp.Regexp{
	regexp.MustCompile(`(?i)rm\s+(-[a-zA-Z]*f[a-zA-Z]*|--force).*/\s*$`),
	regexp.MustCompile(`(?i)rm\s+-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+(/|/\*|/\s+\*)`),
	regexp.MustCompile(`(?i)rm\s+-[a-zA-Z]*f[a-zA-Z]*r[a-zA-Z]*\s+(/|/\*)`),
	regexp.MustCompile(`(?i)mkfs(\.\w+)?\s`),
	regexp.MustCompile(`(?i)dd\s+.*\bof=/dev/`),
	regexp.MustCompile(`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;`),
	regexp.MustCompile(`(?i)\b(shutdown|poweroff|halt|init\s+0)\b`),
	regexp.MustCompile(`(?i)\bmkfs\b`),
	regexp.MustCompile(`(?i)>\s*/dev/sd[a-z]`),
	regexp.MustCompile(`(?i)chmod\s+(-R\s+)?777\s+/`),
	regexp.MustCompile(`(?i)chown\s+-R\s+[^\s]+\s+/(\s|$)`),
	regexp.MustCompile(`(?i)\b(wipefs|sgdisk|parted)\b.*\s/dev/`),
	regexp.MustCompile(`(?i)curl\s+[^\n]*\|\s*(ba)?sh`),
	regexp.MustCompile(`(?i)wget\s+[^\n]*\|\s*(ba)?sh`),
	regexp.MustCompile(`(?i)\bDROP\s+(DATABASE|SCHEMA)\b`),
	regexp.MustCompile(`(?i)\bTRUNCATE\s+(TABLE\s+)?`),
	regexp.MustCompile(`(?i)\bFLUSHALL\b`),
	regexp.MustCompile(`(?i)\bFLUSHDB\b`),
}

func commandBlocked(cmd string) bool {
	s := strings.TrimSpace(cmd)
	if s == "" {
		return false
	}
	for _, re := range dangerCommand {
		if re.MatchString(s) {
			return true
		}
	}
	return matchCustomDanger(s, true)
}

func matchCustomDanger(cmd string, lethal bool) bool {
	st := loadSettings()
	for _, p := range st.CustomDangerPatterns {
		re, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		if re.MatchString(cmd) {
			return true
		}
	}
	_ = lethal
	return false
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
