package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"FlashDock/define"

	"github.com/google/uuid"
)

const portForwardsFileName = "port_forwards.json"

// PortForwardRule 独立端口转发规则
type PortForwardRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"` // local | remote | dynamic
	LocalHost   string `json:"localHost,omitempty"`
	LocalPort   int    `json:"localPort"`
	RemoteHost  string `json:"remoteHost,omitempty"`
	RemotePort  int    `json:"remotePort,omitempty"`
	MachineName string `json:"machineName"`
	Enabled     bool   `json:"enabled"`
	AutoStart   bool   `json:"autoStart"`
}

// PortForwardStore 独立端口转发持久化
type PortForwardStore struct {
	mu   sync.RWMutex
	path string
	list []PortForwardRule
}

var (
	portForwardStore     *PortForwardStore
	portForwardStoreOnce sync.Once
)

// GlobalPortForwardStore 返回全局端口转发存储
func GlobalPortForwardStore() *PortForwardStore {
	portForwardStoreOnce.Do(func() {
		path, err := portForwardsPath()
		if err != nil {
			path = portForwardsFileName
		}
		portForwardStore = &PortForwardStore{path: path, list: []PortForwardRule{}}
		_ = portForwardStore.load()
	})
	return portForwardStore
}

func portForwardsPath() (string, error) {
	home, err := ConfigHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, portForwardsFileName), nil
}

func (s *PortForwardStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.list = []PortForwardRule{}
			return nil
		}
		return err
	}
	var list []PortForwardRule
	if err := json.Unmarshal(raw, &list); err != nil {
		return err
	}
	changed := false
	for i := range list {
		if list[i].ID == "" {
			list[i].ID = uuid.NewString()
			changed = true
		}
	}
	s.list = list
	if changed {
		return s.saveLocked()
	}
	return nil
}

func (s *PortForwardStore) saveLocked() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}
	raw, err := json.MarshalIndent(s.list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, raw, 0644)
}

// List 返回全部规则副本
func (s *PortForwardStore) List() []PortForwardRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PortForwardRule, len(s.list))
	copy(out, s.list)
	return out
}

// Get 按 ID 获取规则
func (s *PortForwardStore) Get(id string) *PortForwardRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, rule := range s.list {
		if rule.ID == id {
			copy := rule
			return &copy
		}
	}
	return nil
}

// SaveAll 替换全部规则
func (s *PortForwardStore) SaveAll(rules []PortForwardRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range rules {
		rules[i].EnsureID()
		normalizePortForwardRule(&rules[i])
	}
	s.list = rules
	return s.saveLocked()
}

// EnsureID 确保规则有 ID
func (r *PortForwardRule) EnsureID() {
	if r != nil && r.ID == "" {
		r.ID = uuid.NewString()
	}
}

func normalizePortForwardRule(r *PortForwardRule) {
	if r == nil {
		return
	}
	r.Type = strings.ToLower(strings.TrimSpace(r.Type))
	if r.Type == "" {
		r.Type = "local"
	}
	r.LocalHost = strings.TrimSpace(r.LocalHost)
	if r.LocalHost == "" {
		r.LocalHost = "127.0.0.1"
	}
	r.RemoteHost = strings.TrimSpace(r.RemoteHost)
	if r.RemoteHost == "" && r.Type == "local" {
		r.RemoteHost = "127.0.0.1"
	}
	r.MachineName = strings.TrimSpace(r.MachineName)
	r.Name = strings.TrimSpace(r.Name)
}

// ToSSHTunnel 转为机器隧道规格
func (r *PortForwardRule) ToSSHTunnel() define.SSHTunnel {
	if r == nil {
		return define.SSHTunnel{}
	}
	name := r.Name
	if name == "" {
		name = r.ID
	}
	return define.SSHTunnel{
		Enabled:    true,
		Name:       name,
		Type:       r.Type,
		LocalHost:  r.LocalHost,
		LocalPort:  r.LocalPort,
		RemoteHost: r.RemoteHost,
		RemotePort: r.RemotePort,
	}
}
