package data

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// 兼容 ssh 包用 %v 包装后无法 errors.As 的情况
var hostKeyUnknownRe = regexp.MustCompile(`未知主机密钥\s+([^（]+)（指纹\s+(SHA256:[A-Za-z0-9+/=]+)）`)

// HostKeyUnknownError 未知主机密钥，需用户确认信任
type HostKeyUnknownError struct {
	Host        string
	Port        int
	Fingerprint string
}

func (e *HostKeyUnknownError) Error() string {
	return fmt.Sprintf("未知主机密钥 %s:%d（指纹 %s），请在连接前信任该主机", e.Host, e.Port, e.Fingerprint)
}

// ParseHostKeyUnknownError 从错误链或文案中解析未知主机密钥
func ParseHostKeyUnknownError(err error) *HostKeyUnknownError {
	if err == nil {
		return nil
	}
	var hk *HostKeyUnknownError
	if errors.As(err, &hk) && hk != nil {
		return hk
	}
	return parseHostKeyUnknownFromMessage(err.Error())
}

func parseHostKeyUnknownFromMessage(msg string) *HostKeyUnknownError {
	m := hostKeyUnknownRe.FindStringSubmatch(msg)
	if m == nil {
		return nil
	}
	hostPort := strings.TrimSpace(m[1])
	host := hostPort
	port := 22
	if i := strings.LastIndex(hostPort, ":"); i >= 0 {
		host = strings.TrimSpace(hostPort[:i])
		if p, err := strconv.Atoi(strings.TrimSpace(hostPort[i+1:])); err == nil && p > 0 {
			port = p
		}
	}
	if host == "" || m[2] == "" {
		return nil
	}
	return &HostKeyUnknownError{Host: host, Port: port, Fingerprint: m[2]}
}

// TrustSessionIfUnknown 若为未知主机密钥则会话级信任（不落盘），返回是否已信任
func TrustSessionIfUnknown(err error) bool {
	hk := ParseHostKeyUnknownError(err)
	if hk == nil {
		return false
	}
	GlobalHostKeyManager().TrustSession(hk.Host, hk.Port, hk.Fingerprint)
	return true
}

// KnownHostRecord 已信任主机记录
type KnownHostRecord struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Fingerprint string `json:"fingerprint"`
}

// HostKeyManager 管理 ~/.flashshell/app_data.json 中的 knownHosts
type HostKeyManager struct {
	mu           sync.RWMutex
	hosts        map[string]string // "host:port" -> SHA256 fingerprint（持久）
	sessionHosts map[string]string // 仅本次会话信任，不落盘
}

var globalHostKeyManager = NewHostKeyManager()

// GlobalHostKeyManager 返回全局 Host Key 管理器
func GlobalHostKeyManager() *HostKeyManager {
	return globalHostKeyManager
}

// NewHostKeyManager 创建管理器并尝试加载磁盘数据
func NewHostKeyManager() *HostKeyManager {
	m := &HostKeyManager{
		hosts:        make(map[string]string),
		sessionHosts: make(map[string]string),
	}
	_ = m.Load()
	return m
}

func hostKeyAddr(host string, port int) string {
	if port <= 0 {
		port = 22
	}
	return fmt.Sprintf("%s:%d", strings.TrimSpace(host), port)
}

// FingerprintSHA256 计算 SSH 公钥 SHA256 指纹（OpenSSH 格式）
func FingerprintSHA256(key ssh.PublicKey) string {
	sum := sha256.Sum256(key.Marshal())
	return "SHA256:" + base64.StdEncoding.EncodeToString(sum[:])
}

// Load 从磁盘加载
func (m *HostKeyManager) Load() error {
	d, err := loadAppDataSection()
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hosts = make(map[string]string)
	for _, rec := range d.KnownHosts {
		addr := hostKeyAddr(rec.Host, rec.Port)
		if rec.Fingerprint != "" {
			m.hosts[addr] = rec.Fingerprint
		}
	}
	return nil
}

func (m *HostKeyManager) saveLocked() error {
	list := make([]KnownHostRecord, 0, len(m.hosts))
	for addr, fp := range m.hosts {
		host, port := splitHostPort(addr)
		list = append(list, KnownHostRecord{Host: host, Port: port, Fingerprint: fp})
	}
	return updateAppData(func(d *AppDataFile) {
		d.KnownHosts = list
	})
}

func splitHostPort(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 22
	}
	port := 22
	fmt.Sscanf(portStr, "%d", &port)
	return host, port
}

// List 返回全部已信任记录
func (m *HostKeyManager) List() []KnownHostRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]KnownHostRecord, 0, len(m.hosts))
	for addr, fp := range m.hosts {
		host, port := splitHostPort(addr)
		out = append(out, KnownHostRecord{Host: host, Port: port, Fingerprint: fp})
	}
	return out
}

// Trust 持久信任主机密钥
func (m *HostKeyManager) Trust(host string, port int, fingerprint string) error {
	addr := hostKeyAddr(host, port)
	fp := strings.TrimSpace(fingerprint)
	m.mu.Lock()
	m.hosts[addr] = fp
	delete(m.sessionHosts, addr) // 已持久信任则无需会话级
	err := m.saveLocked()
	m.mu.Unlock()
	return err
}

// TrustSession 仅本次应用会话信任（内存，不落盘）
func (m *HostKeyManager) TrustSession(host string, port int, fingerprint string) {
	addr := hostKeyAddr(host, port)
	m.mu.Lock()
	m.sessionHosts[addr] = strings.TrimSpace(fingerprint)
	m.mu.Unlock()
}

// ClearSessionTrust 清除单次会话信任
func (m *HostKeyManager) ClearSessionTrust(host string, port int) {
	addr := hostKeyAddr(host, port)
	m.mu.Lock()
	delete(m.sessionHosts, addr)
	m.mu.Unlock()
}

// Remove 移除信任
func (m *HostKeyManager) Remove(host string, port int) error {
	return m.revokeTrust(host, port)
}

// revokeTrust 移除持久与会话级信任并落盘
func (m *HostKeyManager) revokeTrust(host string, port int) error {
	addr := hostKeyAddr(host, port)
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.hosts, addr)
	delete(m.sessionHosts, addr)
	return m.saveLocked()
}

// IsTrusted 检查是否已信任
func (m *HostKeyManager) IsTrusted(host string, port int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.hosts[hostKeyAddr(host, port)]
	return ok
}

// Callback 返回 ssh.HostKeyCallback
func (m *HostKeyManager) Callback() ssh.HostKeyCallback {
	return func(hostname string, _ net.Addr, key ssh.PublicKey) error {
		host, port := splitHostPort(hostname)
		fp := FingerprintSHA256(key)
		addr := hostKeyAddr(host, port)
		m.mu.RLock()
		expected, ok := m.hosts[addr]
		if !ok {
			expected, ok = m.sessionHosts[addr]
		}
		m.mu.RUnlock()
		if !ok {
			return &HostKeyUnknownError{Host: host, Port: port, Fingerprint: fp}
		}
		if expected != fp {
			if err := m.revokeTrust(host, port); err != nil {
				return fmt.Errorf("主机密钥冲突且无法更新本地记录: %w", err)
			}
			return &HostKeyUnknownError{Host: host, Port: port, Fingerprint: fp}
		}
		return nil
	}
}

// ExportJSON 导出 JSON
func (m *HostKeyManager) ExportJSON() ([]byte, error) {
	return json.MarshalIndent(m.List(), "", "  ")
}

// ImportJSON 导入 JSON（合并）
func (m *HostKeyManager) ImportJSON(data []byte) (int, error) {
	var list []KnownHostRecord
	if err := json.Unmarshal(data, &list); err != nil {
		return 0, fmt.Errorf("解析导入数据失败: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, rec := range list {
		if rec.Host == "" || rec.Fingerprint == "" {
			continue
		}
		addr := hostKeyAddr(rec.Host, rec.Port)
		m.hosts[addr] = rec.Fingerprint
		n++
	}
	return n, m.saveLocked()
}
