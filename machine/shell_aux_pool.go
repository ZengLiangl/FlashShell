package machine

import (
	"fmt"
	"sync"

	"FlashDock/define"
)

// ShellAuxPool 辅助连接池（监控 + SFTP）
type ShellAuxPool struct {
	mu      sync.RWMutex
	clients map[string]*ShellAuxManager
}

// NewShellAuxPool 创建辅助连接池
func NewShellAuxPool() *ShellAuxPool {
	return &ShellAuxPool{clients: make(map[string]*ShellAuxManager)}
}

// AttachFromSession 复用 PTY SSH 连接建立辅助通道（监控 + SFTP）。
func (p *ShellAuxPool) AttachFromSession(machineName string, ptyClient *SSHClient, host string) error {
	p.mu.Lock()
	if existing, ok := p.clients[machineName]; ok {
		_ = existing.Close()
		delete(p.clients, machineName)
	}
	aux := NewShellAuxManager()
	p.clients[machineName] = aux
	p.mu.Unlock()

	if err := aux.Attach(ptyClient, machineName, host); err != nil {
		p.mu.Lock()
		delete(p.clients, machineName)
		p.mu.Unlock()
		return err
	}
	return nil
}

// EnsureAttached 优先复用 PTY SSH；PTY 未连或挂载失败时再独立建连。
func (p *ShellAuxPool) EnsureAttached(machineName string, machine *define.Machine, workVars map[string]string, ptyClient *SSHClient, host string) error {
	p.mu.RLock()
	aux, ok := p.clients[machineName]
	p.mu.RUnlock()
	if ok && aux != nil && aux.IsConnected() {
		return nil
	}
	if ptyClient != nil && ptyClient.IsConnected() {
		if err := p.AttachFromSession(machineName, ptyClient, host); err == nil {
			return nil
		}
		// 复用 PTY 失败时回退独立连接，避免监控/系统信息长期不可用
	}
	return p.Connect(machineName, machine, workVars)
}

// ConnectIfNeeded 已连接则跳过，避免重复建连导致监控/SFTP 短暂不可用。
func (p *ShellAuxPool) ConnectIfNeeded(machineName string, machine *define.Machine, workVars map[string]string) error {
	return p.EnsureAttached(machineName, machine, workVars, nil, "")
}

// Connect 连接辅助通道
func (p *ShellAuxPool) Connect(machineName string, machine *define.Machine, workVars map[string]string) error {
	p.mu.Lock()
	if existing, ok := p.clients[machineName]; ok {
		_ = existing.Close()
		delete(p.clients, machineName)
	}
	aux := NewShellAuxManager()
	p.clients[machineName] = aux
	p.mu.Unlock()

	if err := aux.Connect(machine, workVars); err != nil {
		p.mu.Lock()
		delete(p.clients, machineName)
		p.mu.Unlock()
		return err
	}
	return nil
}

// Disconnect 断开辅助通道
func (p *ShellAuxPool) Disconnect(machineName string) error {
	p.mu.Lock()
	aux, ok := p.clients[machineName]
	if ok {
		delete(p.clients, machineName)
	}
	p.mu.Unlock()
	if !ok {
		return nil
	}
	return aux.Close()
}

// DisconnectAll 断开全部
func (p *ShellAuxPool) DisconnectAll() {
	p.mu.Lock()
	names := make([]string, 0, len(p.clients))
	for name := range p.clients {
		names = append(names, name)
	}
	p.mu.Unlock()
	for _, name := range names {
		_ = p.Disconnect(name)
	}
}

// Get 获取辅助管理器
func (p *ShellAuxPool) Get(machineName string) (*ShellAuxManager, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	aux, ok := p.clients[machineName]
	if !ok || aux == nil || !aux.IsConnected() {
		return nil, fmt.Errorf("辅助连接不存在: %s", machineName)
	}
	return aux, nil
}
