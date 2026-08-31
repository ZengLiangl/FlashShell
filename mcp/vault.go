package mcp

import (
	"fmt"
	"os"
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
		if server != "" && !strings.EqualFold(it.ServerAlias, server) {
			continue
		}
		out = append(out, map[string]any{
			"id":          it.ID,
			"serverAlias": it.ServerAlias,
			"kind":        it.Kind,
			"label":       it.Label,
			"installPath": it.InstallPath,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out
}

func (v *Vault) PutSecret(id, kind string, secrets, public map[string]string) error {
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

func (v *Vault) Find(idOrLabel string) (VaultItem, map[string]string, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	key := strings.TrimSpace(idOrLabel)
	for _, it := range v.items {
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
