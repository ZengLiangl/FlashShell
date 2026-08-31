package mcp

import (
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"strings"

	"FlashDock/crypto"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type redactRule struct {
	Name    string
	Kind    string
	Pattern *regexp.Regexp
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

var builtinRedactRules = []redactRule{
	{Name: "generic_password_line", Kind: "credential", Pattern: regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*(?P<secret>\S+)`)},
	{Name: "mysql_root_password", Kind: "credential", Pattern: regexp.MustCompile(`(?i)(mysql.*?password|password.*?mysql)\s*[:=]\s*(?P<secret>\S+)`)},
	{Name: "api_key_assign", Kind: "token", Pattern: regexp.MustCompile(`(?i)(api[_-]?key|access[_-]?key)\s*[:=]\s*(?P<secret>\S+)`)},
	{Name: "bearer_token", Kind: "token", Pattern: regexp.MustCompile(`(?i)Bearer\s+(?P<secret>[A-Za-z0-9._\-+/=]{8,})`)},
	{Name: "aws_access_key", Kind: "token", Pattern: regexp.MustCompile(`\b(?P<secret>AKIA[0-9A-Z]{16})\b`)},
	{Name: "private_key_pem", Kind: "private_key", Pattern: regexp.MustCompile(`(?s)-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----`)},
	{Name: "url_with_auth", Kind: "url_with_auth", Pattern: regexp.MustCompile(`(?i)https?://[^:\s/]+:(?P<secret>[^@\s]+)@[^\s]+`)},
	{Name: "ssh_authorized_keys", Kind: "token", Pattern: regexp.MustCompile(`(?m)^(?P<secret>ssh-(?:rsa|ed25519|dss)\s+[A-Za-z0-9+/=]{40,})`)},
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
	// 优先 mcp/redaction.yaml，兼容应用数据根目录 redaction.yaml
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
			out = append(out, redactRule{Name: name, Kind: kind, Pattern: re})
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

func redactText(s string) string {
	if s == "" {
		return s
	}
	out := s
	for _, rule := range allRedactRules() {
		out = rule.Pattern.ReplaceAllStringFunc(out, func(m string) string {
			secret := m
			if sub := rule.Pattern.FindStringSubmatch(m); len(sub) > 0 {
				for i, name := range rule.Pattern.SubexpNames() {
					if name == "secret" && i < len(sub) && sub[i] != "" {
						secret = sub[i]
						break
					}
				}
			}
			_ = stashSecret(secret, rule.Name, rule.Kind)
			if strings.Contains(m, secret) && secret != m {
				return strings.Replace(m, secret, "[REDACTED:"+rule.Name+"]", 1)
			}
			return "[REDACTED:" + rule.Name + "]"
		})
	}
	return out
}

func stashSecret(plain, ruleName, kind string) string {
	plain = strings.TrimSpace(plain)
	plain = strings.Trim(plain, `"'`)
	if plain == "" {
		return ruleName
	}
	id := "sv_" + uuid.NewString()[:8]
	v := loadVault()
	_ = v.PutSecret(id, ruleName, map[string]string{"value": plain}, map[string]string{
		"rule": ruleName,
		"kind": kind,
	})
	return ruleName
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
