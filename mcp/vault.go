package mcp

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// VaultItem 服务凭据（字段值加密）
type VaultItem struct {
	ID          string            `yaml:"id" json:"id"`
	ServerAlias string            `yaml:"serverAlias" json:"serverAlias"`
	Kind        string            `yaml:"kind" json:"kind"`
	Label       string            `yaml:"label" json:"label"`
	InstallPath string            `yaml:"installPath,omitempty" json:"installPath,omitempty"`
	Notes       string            `yaml:"notes,omitempty" json:"notes,omitempty"`
	Public      map[string]string `yaml:"public,omitempty" json:"public,omitempty"`
	Secrets     map[string]string `yaml:"secrets,omitempty" json:"-"`
	CreatedAt   time.Time         `yaml:"createdAt" json:"createdAt"`
}

type vaultDoc struct {
	Items []VaultItem `yaml:"items"`
}

type Vault struct {
	mu    sync.Mutex
	items []VaultItem
}

func loadVault() *Vault {
	v := &Vault{}
	root, err := homeDir()
	if err != nil {
		return v
	}
	b, err := os.ReadFile(join(root, vaultFile))
	if err != nil {
		return v
	}
	var f vaultDoc
	if yaml.Unmarshal(b, &f) == nil {
		v.items = f.Items
	}
	return v
}

func (v *Vault) save() error {
	root, err := homeDir()
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(vaultDoc{Items: v.items})
	if err != nil {
		return err
	}
	return os.WriteFile(join(root, vaultFile), b, 0600)
}

func (v *Vault) ListMeta(server string) []map[string]any {
	v.mu.Lock()
	defer v.mu.Unlock()
	var out []map[string]any
	for _, it := range v.items {
		if isRedactionCapture(it) {
			continue
		}
		if server != "" && it.ServerAlias != "" && !strings.EqualFold(it.ServerAlias, server) {
			continue
		}
		secretFields := make([]string, 0, len(it.Secrets))
		for k := range it.Secrets {
			secretFields = append(secretFields, k)
		}
		sort.Strings(secretFields)
		pub := map[string]string{}
		for k, val := range it.Public {
			if strings.HasPrefix(k, "__") {
				continue
			}
			pub[k] = val
		}
		created := ""
		if !it.CreatedAt.IsZero() {
			created = it.CreatedAt.Format("2006-01-02 15:04:05")
		}
		out = append(out, map[string]any{
			"id":           it.ID,
			"serverAlias":  it.ServerAlias,
			"kind":         it.Kind,
			"label":        it.Label,
			"installPath":  it.InstallPath,
			"notes":        it.Notes,
			"createdAt":    created,
			"public":       pub,
			"secretFields": secretFields,
			"fromSensitive": it.Public["fromSensitive"],
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out
}

func (v *Vault) PutSecret(id, kind string, secrets, public map[string]string) error {
	// 仅用于服务凭据；出口脱敏请走 SensitiveVault.Capture。
	enc, err := encryptMap(secrets)
	if err != nil {
		return err
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.items = append(v.items, VaultItem{
		ID:        id,
		Kind:      kind,
		Label:     kind,
		Public:    public,
		Secrets:   enc,
		CreatedAt: time.Now(),
	})
	return v.save()
}

func (v *Vault) Save(item VaultItem) (VaultItem, error) {
	if item.ID == "" {
		item.ID = "vs_" + uuid.NewString()[:8]
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}
	if len(item.Secrets) > 0 {
		enc, err := encryptMap(item.Secrets)
		if err != nil {
			return item, err
		}
		item.Secrets = enc
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	replaced := false
	for i, it := range v.items {
		if it.ID == item.ID {
			v.items[i] = item
			replaced = true
			break
		}
	}
	if !replaced {
		v.items = append(v.items, item)
	}
	return item, v.save()
}

func (v *Vault) Delete(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	next := v.items[:0]
	found := false
	for _, it := range v.items {
		if it.ID == id {
			found = true
			continue
		}
		next = append(next, it)
	}
	if !found {
		return fmt.Errorf("[notfound] 未找到凭据 %s", id)
	}
	v.items = next
	return v.save()
}

// UpdateNotes 仅更新备注，不动敏感字段。
func (v *Vault) UpdateNotes(id, notes string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	id = strings.TrimSpace(id)
	for i, it := range v.items {
		if it.ID != id {
			continue
		}
		if isRedactionCapture(it) {
			return fmt.Errorf("[notfound] 未找到凭据 %s", id)
		}
		v.items[i].Notes = notes
		return v.save()
	}
	return fmt.Errorf("[notfound] 未找到凭据 %s", id)
}

func (v *Vault) Find(idOrLabel string) (VaultItem, map[string]string, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	key := strings.TrimSpace(idOrLabel)
	for _, it := range v.items {
		if isRedactionCapture(it) {
			continue
		}
		if it.ID == key || strings.EqualFold(it.Label, key) || strings.Contains(strings.ToLower(it.Label), strings.ToLower(key)) {
			sec := decryptMap(it.Secrets)
			return it, sec, true
		}
	}
	return VaultItem{}, nil, false
}

func (v *Vault) SecretValue(id string) (string, bool) {
	_, sec, ok := v.Find(id)
	if !ok {
		return "", false
	}
	if val, has := sec["value"]; has {
		return val, true
	}
	for _, val := range sec {
		return val, true
	}
	return "", false
}

// RevealField 解密指定敏感字段；field 空则取 value 或任意第一个。
func (v *Vault) RevealField(id, field string) (string, error) {
	_, sec, ok := v.Find(id)
	if !ok {
		return "", fmt.Errorf("[notfound] 未找到凭据 %s", id)
	}
	field = strings.TrimSpace(field)
	if field != "" {
		val, has := sec[field]
		if !has || val == "" {
			return "", fmt.Errorf("[notfound] 字段 %s 不存在或为空", field)
		}
		return val, nil
	}
	if val, has := sec["value"]; has && val != "" {
		return val, nil
	}
	for _, val := range sec {
		if val != "" {
			return val, nil
		}
	}
	return "", fmt.Errorf("[notfound] 凭据无明文")
}
