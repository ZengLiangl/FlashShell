package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

const (
	SensitiveStatusActive    = "active"
	SensitiveStatusExpired   = "expired"
	SensitiveStatusDiscarded = "discarded"
)

// SensitiveEntry 出口脱敏被动捕获（与服务凭据库分离）。
// 到期后清空加密明文，保留 hash 供审计。
type SensitiveEntry struct {
	ID        string            `yaml:"id" json:"id"`
	Rule      string            `yaml:"rule" json:"rule"`
	Kind      string            `yaml:"kind" json:"kind"`
	Label     string            `yaml:"label" json:"label"`
	Server    string            `yaml:"server,omitempty" json:"server,omitempty"`
	AuditID   string            `yaml:"auditId,omitempty" json:"auditId,omitempty"`
	Status    string            `yaml:"status" json:"status"`
	Hash      string            `yaml:"hash,omitempty" json:"hash,omitempty"`
	HasValue  bool              `yaml:"-" json:"hasValue"`
	Secrets   map[string]string `yaml:"secrets,omitempty" json:"-"`
	CreatedAt string            `yaml:"createdAt" json:"createdAt"`
	ExpiresAt string            `yaml:"expiresAt,omitempty" json:"expiresAt,omitempty"`
	ExpiredAt string            `yaml:"expiredAt,omitempty" json:"expiredAt,omitempty"`
}

// FalsePositiveSample 标假阳性后保留的样本（无明文）。
type FalsePositiveSample struct {
	ID        string `yaml:"id" json:"id"`
	Rule      string `yaml:"rule" json:"rule"`
	Kind      string `yaml:"kind" json:"kind"`
	Hash      string `yaml:"hash,omitempty" json:"hash,omitempty"`
	Server    string `yaml:"server,omitempty" json:"server,omitempty"`
	CreatedAt string `yaml:"createdAt" json:"createdAt"`
}

type sensitiveDoc struct {
	Items []SensitiveEntry `yaml:"items"`
}

type falsePositivesDoc struct {
	Items []FalsePositiveSample `yaml:"items"`
}

type SensitiveVault struct {
	mu    sync.Mutex
	items []SensitiveEntry
}

func loadSensitiveVault() *SensitiveVault {
	v := &SensitiveVault{}
	root, err := homeDir()
	if err != nil {
		return v
	}
	b, err := os.ReadFile(join(root, sensitiveVaultFile))
	if err != nil {
		return v
	}
	var f sensitiveDoc
	if yaml.Unmarshal(b, &f) == nil {
		v.items = f.Items
	}
	return v
}

func (v *SensitiveVault) saveLocked() error {
	root, err := homeDir()
	if err != nil {
		return err
	}
	out := make([]SensitiveEntry, len(v.items))
	copy(out, v.items)
	b, err := yaml.Marshal(sensitiveDoc{Items: out})
	if err != nil {
		return err
	}
	return os.WriteFile(join(root, sensitiveVaultFile), b, 0600)
}

func parseEntryTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func hashSensitivePlain(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func (v *SensitiveVault) Capture(plain, rule, kind, server string, ttlDays int) (SensitiveEntry, error) {
	plain = strings.TrimSpace(plain)
	plain = strings.Trim(plain, `"'`)
	if plain == "" {
		return SensitiveEntry{}, fmt.Errorf("empty secret")
	}
	if ttlDays <= 0 {
		ttlDays = 30
	}
	enc, err := encryptMap(map[string]string{"value": plain})
	if err != nil {
		return SensitiveEntry{}, err
	}
	now := time.Now()
	ent := SensitiveEntry{
		ID:        "sv_" + uuid.NewString()[:8],
		Rule:      rule,
		Kind:      kind,
		Label:     rule,
		Server:    strings.TrimSpace(server),
		Status:    SensitiveStatusActive,
		Hash:      hashSensitivePlain(plain),
		Secrets:   enc,
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: now.Add(time.Duration(ttlDays) * 24 * time.Hour).Format(time.RFC3339),
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.items = append(v.items, ent)
	if err := v.saveLocked(); err != nil {
		v.items = v.items[:len(v.items)-1]
		return SensitiveEntry{}, err
	}
	return ent, nil
}

func (v *SensitiveVault) LinkAudit(ids []string, auditID, server string) {
	if len(ids) == 0 || strings.TrimSpace(auditID) == "" {
		return
	}
	want := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			want[id] = struct{}{}
		}
	}
	if len(want) == 0 {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	changed := false
	server = strings.TrimSpace(server)
	for i := range v.items {
		it := &v.items[i]
		if _, ok := want[it.ID]; !ok {
			continue
		}
		if it.AuditID == "" {
			it.AuditID = auditID
			changed = true
		}
		if it.Server == "" && server != "" {
			it.Server = server
			changed = true
		}
	}
	if changed {
		_ = v.saveLocked()
	}
}

func (v *SensitiveVault) Reveal(id string) (string, error) {
	id = strings.TrimSpace(id)
	v.mu.Lock()
	defer v.mu.Unlock()
	_, _ = v.expireDueLocked(time.Now())
	for _, it := range v.items {
		if it.ID != id {
			continue
		}
		if it.Status == SensitiveStatusDiscarded {
			return "", fmt.Errorf("该条目已标为假阳性并丢弃")
		}
		if it.Status == SensitiveStatusExpired || len(it.Secrets) == 0 {
			return "", fmt.Errorf("明文已过期或不可用，仅保留 hash")
		}
		sec := decryptMap(it.Secrets)
		if val, ok := sec["value"]; ok && val != "" {
			return val, nil
		}
		for _, val := range sec {
			if val != "" {
				return val, nil
			}
		}
		return "", fmt.Errorf("解密失败或无明文")
	}
	return "", fmt.Errorf("[notfound] 未找到敏感条目 %s", id)
}

func (v *SensitiveVault) Discard(id string) error {
	id = strings.TrimSpace(id)
	v.mu.Lock()
	defer v.mu.Unlock()
	for i := range v.items {
		it := &v.items[i]
		if it.ID != id {
			continue
		}
		sample := FalsePositiveSample{
			ID:        it.ID,
			Rule:      it.Rule,
			Kind:      it.Kind,
			Hash:      it.Hash,
			Server:    it.Server,
			CreatedAt: time.Now().Format(time.RFC3339),
		}
		it.Secrets = nil
		it.Status = SensitiveStatusDiscarded
		it.ExpiredAt = time.Now().Format(time.RFC3339)
		if err := v.saveLocked(); err != nil {
			return err
		}
		_ = appendFalsePositive(sample)
		return nil
	}
	return fmt.Errorf("[notfound] 未找到敏感条目 %s", id)
}

func appendFalsePositive(sample FalsePositiveSample) error {
	root, err := homeDir()
	if err != nil {
		return err
	}
	path := join(root, falsePositivesFile)
	var doc falsePositivesDoc
	if b, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(b, &doc)
	}
	doc.Items = append(doc.Items, sample)
	b, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func ListFalsePositives() []FalsePositiveSample {
	root, err := homeDir()
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(join(root, falsePositivesFile))
	if err != nil {
		return nil
	}
	var doc falsePositivesDoc
	if yaml.Unmarshal(b, &doc) != nil {
		return nil
	}
	if doc.Items == nil {
		return []FalsePositiveSample{}
	}
	return doc.Items
}

func (v *SensitiveVault) ExpireDue(now time.Time) (int, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.expireDueLocked(now)
}

func (v *SensitiveVault) expireDueLocked(now time.Time) (int, error) {
	n := 0
	for i := range v.items {
		it := &v.items[i]
		if it.Status != "" && it.Status != SensitiveStatusActive {
			continue
		}
		exp, ok := parseEntryTime(it.ExpiresAt)
		if !ok || now.Before(exp) {
			continue
		}
		it.Secrets = nil
		it.Status = SensitiveStatusExpired
		it.ExpiredAt = now.Format(time.RFC3339)
		n++
	}
	if n == 0 {
		return 0, nil
	}
	return n, v.saveLocked()
}

func (v *SensitiveVault) ListMeta() []map[string]any {
	v.mu.Lock()
	defer v.mu.Unlock()
	_, _ = v.expireDueLocked(time.Now())
	out := make([]map[string]any, 0, len(v.items))
	for _, it := range v.items {
		status := it.Status
		if status == "" {
			status = SensitiveStatusActive
		}
		out = append(out, map[string]any{
			"id":        it.ID,
			"rule":      it.Rule,
			"kind":      it.Kind,
			"label":     it.Label,
			"server":    it.Server,
			"auditId":   it.AuditID,
			"status":    status,
			"hash":      it.Hash,
			"hasValue":  len(it.Secrets) > 0,
			"createdAt": it.CreatedAt,
			"expiresAt": it.ExpiresAt,
			"expiredAt": it.ExpiredAt,
		})
	}
	return out
}

func (v *SensitiveVault) FindMeta(id string) (map[string]any, bool) {
	id = strings.TrimSpace(id)
	for _, m := range v.ListMeta() {
		if fmt.Sprint(m["id"]) == id {
			return m, true
		}
	}
	return nil, false
}

func (v *SensitiveVault) Len() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.items)
}

func isRedactionCapture(it VaultItem) bool {
	if strings.HasPrefix(it.ID, "sv_") {
		return true
	}
	if it.Public != nil {
		if strings.TrimSpace(it.Public["rule"]) != "" {
			return true
		}
	}
	return false
}

// migrateRedactionCaptures 把误写入服务凭据库的脱敏捕获迁到独立敏感库。
func migrateRedactionCaptures(cred *Vault, sens *SensitiveVault) {
	if cred == nil || sens == nil {
		return
	}
	cred.mu.Lock()
	moved := make([]VaultItem, 0)
	keep := make([]VaultItem, 0, len(cred.items))
	for _, it := range cred.items {
		if isRedactionCapture(it) {
			moved = append(moved, it)
			continue
		}
		keep = append(keep, it)
	}
	if len(moved) == 0 {
		cred.mu.Unlock()
		return
	}
	old := append([]VaultItem(nil), cred.items...)
	cred.items = keep
	if err := cred.save(); err != nil {
		cred.items = old
		cred.mu.Unlock()
		return
	}
	cred.mu.Unlock()

	sens.mu.Lock()
	defer sens.mu.Unlock()
	seen := map[string]struct{}{}
	for _, it := range sens.items {
		seen[it.ID] = struct{}{}
	}
	for _, it := range moved {
		if _, ok := seen[it.ID]; ok {
			continue
		}
		rule := it.Kind
		if it.Public != nil && it.Public["rule"] != "" {
			rule = it.Public["rule"]
		}
		kind := it.Kind
		if it.Public != nil && it.Public["kind"] != "" {
			kind = it.Public["kind"]
		}
		created := ""
		if !it.CreatedAt.IsZero() {
			created = it.CreatedAt.Format(time.RFC3339)
		}
		sens.items = append(sens.items, SensitiveEntry{
			ID:        it.ID,
			Rule:      rule,
			Kind:      kind,
			Label:     it.Label,
			Server:    it.ServerAlias,
			Status:    SensitiveStatusActive,
			Secrets:   it.Secrets,
			CreatedAt: created,
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
		})
	}
	_ = sens.saveLocked()
}
