package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"FlashDock/crypto"

	"gopkg.in/yaml.v3"
)

type redactRule struct {
	Name    string
	Kind    string
	Pattern *regexp.Regexp
	Builtin bool
}

type redactRuleYAML struct {
	Name    string `yaml:"name"`
	Pattern string `yaml:"pattern"`
	Kind    string `yaml:"kind"`
	Enabled *bool  `yaml:"enabled"`
}

type redactFileYAML struct {
	Rules []redactRuleYAML `yaml:"rules"`
}

// RedactHit 规则测试器命中项（不入库）
type RedactHit struct {
	Rule    string `json:"rule"`
	Kind    string `json:"kind"`
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Match   string `json:"match"`
	Secret  string `json:"secret"`
	Snippet string `json:"snippet"`
}

var builtinRedactRules = []redactRule{
	{Name: "generic_password_line", Kind: "credential", Builtin: true, Pattern: regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*(?P<secret>\S+)`)},
	{Name: "mysql_root_password", Kind: "credential", Builtin: true, Pattern: regexp.MustCompile(`(?i)(mysql.*?password|password.*?mysql)\s*[:=]\s*(?P<secret>\S+)`)},
	{Name: "api_key_assign", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`(?i)(api[_-]?key|access[_-]?key)\s*[:=]\s*(?P<secret>\S+)`)},
	{Name: "bearer_token", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`(?i)Bearer\s+(?P<secret>[A-Za-z0-9._\-+/=]{8,})`)},
	{Name: "aws_access_key", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`\b(?P<secret>AKIA[0-9A-Z]{16})\b`)},
	{Name: "aws_secret_key", Kind: "credential", Builtin: true, Pattern: regexp.MustCompile(`(?i)(aws[_-]?secret[_-]?(?:access[_-]?)?key)\s*[:=]\s*(?P<secret>[A-Za-z0-9/+=]{40})`)},
	{Name: "private_key_pem", Kind: "private_key", Builtin: true, Pattern: regexp.MustCompile(`(?s)-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----`)},
	{Name: "url_with_auth", Kind: "url_with_auth", Builtin: true, Pattern: regexp.MustCompile(`(?i)https?://[^:\s/]+:(?P<secret>[^@\s]+)@[^\s]+`)},
	{Name: "ssh_authorized_keys", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`(?m)^(?P<secret>ssh-(?:rsa|ed25519|dss)\s+[A-Za-z0-9+/=]{40,})`)},
	{Name: "github_token", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`\b(?P<secret>(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,})\b`)},
	{Name: "gitlab_pat", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`\b(?P<secret>glpat-[A-Za-z0-9_\-]{20,})\b`)},
	{Name: "openai_api_key", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`\b(?P<secret>sk-(?:proj-|svcacct-)?[A-Za-z0-9_\-]{20,})\b`)},
	{Name: "anthropic_api_key", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`\b(?P<secret>sk-ant-[A-Za-z0-9_\-]{20,})\b`)},
	{Name: "slack_token", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`\b(?P<secret>xox[baprs]-[A-Za-z0-9-]{10,})\b`)},
	{Name: "stripe_key", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`\b(?P<secret>(?:sk|pk|rk)_(?:live|test)_[A-Za-z0-9]{16,})\b`)},
	{Name: "jwt_token", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`\b(?P<secret>eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+)\b`)},
	{Name: "google_api_key", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`\b(?P<secret>AIza[0-9A-Za-z_\-]{35})\b`)},
	{Name: "npm_token", Kind: "token", Builtin: true, Pattern: regexp.MustCompile(`\b(?P<secret>npm_[A-Za-z0-9]{36})\b`)},
	{Name: "postgres_url", Kind: "url_with_auth", Builtin: true, Pattern: regexp.MustCompile(`(?i)postgres(?:ql)?://[^:\s/]+:(?P<secret>[^@\s]+)@[^\s]+`)},
	{Name: "mysql_url", Kind: "url_with_auth", Builtin: true, Pattern: regexp.MustCompile(`(?i)mysql://[^:\s/]+:(?P<secret>[^@\s]+)@[^\s]+`)},
	{Name: "redis_url", Kind: "url_with_auth", Builtin: true, Pattern: regexp.MustCompile(`(?i)redis://(?:[^:\s/]+:)?(?P<secret>[^@/\s]+)@[^\s]+`)},
	{Name: "connection_string_pwd", Kind: "credential", Builtin: true, Pattern: regexp.MustCompile(`(?i);\s*(?:pwd|password)\s*=\s*(?P<secret>[^;\s"']+)`)},
}

var (
	redactRulesMu   sync.RWMutex
	cachedRedactAll []redactRule
)

func loadUserRedactRules() []redactRule {
	root, err := homeDir()
	if err != nil {
		return nil
	}
	paths := []string{
		join(root, "redaction.yaml"),
		filepath.Join(filepath.Dir(root), "redaction.yaml"),
	}
	var out []redactRule
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil || len(b) == 0 {
			continue
		}
		var f redactFileYAML
		if yaml.Unmarshal(b, &f) != nil {
			continue
		}
		for _, r := range f.Rules {
			if r.Enabled != nil && !*r.Enabled {
				continue
			}
			name := strings.TrimSpace(r.Name)
			pat := strings.TrimSpace(r.Pattern)
			if name == "" || pat == "" {
				continue
			}
			re, err := regexp.Compile(pat)
			if err != nil {
				continue
			}
			kind := strings.TrimSpace(r.Kind)
			if kind == "" {
				kind = "generic"
			}
			out = append(out, redactRule{Name: name, Kind: kind, Pattern: re, Builtin: false})
		}
		break
	}
	return out
}

func allRedactRules() []redactRule {
	redactRulesMu.RLock()
	if cachedRedactAll != nil {
		defer redactRulesMu.RUnlock()
		return cachedRedactAll
	}
	redactRulesMu.RUnlock()

	redactRulesMu.Lock()
	defer redactRulesMu.Unlock()
	if cachedRedactAll != nil {
		return cachedRedactAll
	}
	all := append([]redactRule{}, builtinRedactRules...)
	all = append(all, loadUserRedactRules()...)
	cachedRedactAll = all
	return cachedRedactAll
}

// ReloadRedactRules 重载内置 + 用户 redaction.yaml
func ReloadRedactRules() {
	redactRulesMu.Lock()
	cachedRedactAll = nil
	redactRulesMu.Unlock()
	_ = allRedactRules()
}

// ListRedactRulesMeta 规则清单（内置在前，用户自定义在后）
func ListRedactRulesMeta() []map[string]any {
	out := make([]map[string]any, 0)
	for _, r := range allRedactRules() {
		out = append(out, map[string]any{
			"name":    r.Name,
			"kind":    r.Kind,
			"pattern": r.Pattern.String(),
			"builtin": r.Builtin,
		})
	}
	return out
}

// UserRedactRule 用户自定义脱敏规则（落盘 redaction.yaml）
type UserRedactRule struct {
	Name    string `json:"name" yaml:"name"`
	Pattern string `json:"pattern" yaml:"pattern"`
	Kind    string `json:"kind" yaml:"kind"`
}

func userRedactionPath() (string, error) {
	root, err := homeDir()
	if err != nil {
		return "", err
	}
	return join(root, "redaction.yaml"), nil
}

// SaveUserRedactRules 保存自定义脱敏规则（不影响内置）；校验正则后写 redaction.yaml 并重载。
func SaveUserRedactRules(rules []UserRedactRule) error {
	norm := make([]redactRuleYAML, 0, len(rules))
	seen := map[string]struct{}{}
	for _, r := range rules {
		name := strings.TrimSpace(r.Name)
		pat := strings.TrimSpace(r.Pattern)
		if name == "" || pat == "" {
			continue
		}
		if _, err := regexp.Compile(pat); err != nil {
			return fmt.Errorf("无效正则 %s: %w", name, err)
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("规则名重复: %s", name)
		}
		seen[key] = struct{}{}
		for _, b := range builtinRedactRules {
			if strings.EqualFold(b.Name, name) {
				return fmt.Errorf("不能覆盖内置规则名: %s", name)
			}
		}
		kind := strings.TrimSpace(r.Kind)
		if kind == "" {
			kind = "generic"
		}
		norm = append(norm, redactRuleYAML{Name: name, Pattern: pat, Kind: kind})
	}
	path, err := userRedactionPath()
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(redactFileYAML{Rules: norm})
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		return err
	}
	ReloadRedactRules()
	return nil
}

// TestRedactRules 对文本跑规则测试（不入库、不替换持久化）
func TestRedactRules(text string) []RedactHit {
	if text == "" {
		return []RedactHit{}
	}
	var hits []RedactHit
	for _, rule := range allRedactRules() {
		idxs := rule.Pattern.FindAllStringSubmatchIndex(text, -1)
		for _, idx := range idxs {
			if len(idx) < 2 {
				continue
			}
			start, end := idx[0], idx[1]
			match := text[start:end]
			secret := match
			if names := rule.Pattern.SubexpNames(); len(names) > 0 {
				for i, name := range names {
					if name == "secret" && i*2+1 < len(idx) && idx[i*2] >= 0 {
						secret = text[idx[i*2]:idx[i*2+1]]
						break
					}
				}
			}
			snippet := match
			if len(snippet) > 80 {
				snippet = snippet[:40] + "…" + snippet[len(snippet)-20:]
			}
			hits = append(hits, RedactHit{
				Rule:    rule.Name,
				Kind:    rule.Kind,
				Start:   start,
				End:     end,
				Match:   match,
				Secret:  "[hidden len=" + strconv.Itoa(len(secret)) + "]",
				Snippet: snippet,
			})
		}
	}
	if hits == nil {
		hits = []RedactHit{}
	}
	return hits
}

func applyRedact(s string, stash func(plain, rule, kind string)) string {
	if s == "" {
		return s
	}
	out := s
	for _, rule := range allRedactRules() {
		out = rule.Pattern.ReplaceAllStringFunc(out, func(m string) string {
			if strings.Contains(m, "[REDACTED:") {
				return m
			}
			secret := m
			if sub := rule.Pattern.FindStringSubmatch(m); len(sub) > 0 {
				for i, name := range rule.Pattern.SubexpNames() {
					if name == "secret" && i < len(sub) && sub[i] != "" {
						secret = sub[i]
						break
					}
				}
			}
			if strings.Contains(secret, "[REDACTED:") {
				return m
			}
			if stash != nil {
				stash(secret, rule.Name, rule.Kind)
			}
			if strings.Contains(m, secret) && secret != m {
				return strings.Replace(m, secret, "[REDACTED:"+rule.Name+"]", 1)
			}
			return "[REDACTED:" + rule.Name + "]"
		})
	}
	return out
}

func (s *Service) redactText(ctx context.Context, text, server string) string {
	if s == nil || s.sensitive == nil {
		return applyRedact(text, nil)
	}
	days := 30
	if st := s.GetSettings(); st.RedactionTTLDays > 0 {
		days = st.RedactionTTLDays
	}
	return applyRedact(text, func(plain, rule, kind string) {
		ent, err := s.sensitive.Capture(plain, rule, kind, server, days)
		if err == nil {
			noteSensID(ctx, ent.ID)
		}
	})
}

func encryptMap(fields map[string]string) (map[string]string, error) {
	out := map[string]string{}
	for k, val := range fields {
		enc, err := crypto.EncryptText(val)
		if err != nil {
			return nil, err
		}
		out[k] = enc
	}
	return out, nil
}

func decryptMap(fields map[string]string) map[string]string {
	out := map[string]string{}
	for k, val := range fields {
		plain, err := crypto.DecryptText(val)
		if err != nil {
			out[k] = val
			continue
		}
		out[k] = plain
	}
	return out
}
