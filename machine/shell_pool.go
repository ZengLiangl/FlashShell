package machine

import (
	"fmt"
	"sync"

	"quick-cmd/define"
)

// ShellSessionPool 管理多台机器的 PTY 会话
type ShellSessionPool struct {
	mu       sync.RWMutex
	sessions map[string]*ShellSessionManager
}

// NewShellSessionPool 创建会话池
func NewShellSessionPool() *ShellSessionPool {
	return &ShellSessionPool{
		sessions: make(map[string]*ShellSessionManager),
	}
}

// Connect 连接指定机器（允许多会话并存；已连接则幂等成功）
func (p *ShellSessionPool) Connect(machineName string, machine *define.Machine, workVars map[string]string, handler ShellOutputHandler) error {
	p.mu.Lock()
	if existing, ok := p.sessions[machineName]; ok {
		if existing.IsConnected() {
			p.mu.Unlock()
			return nil // 已连接，视为成功（前端切到该会话即可）
		}
		delete(p.sessions, machineName)
	}
	sm := NewShellSessionManager()
	p.sessions[machineName] = sm
	p.mu.Unlock()

	if err := sm.Connect(machine, workVars, handler); err != nil {
		p.mu.Lock()
		delete(p.sessions, machineName)
		p.mu.Unlock()
		return err
	}
	return nil
}

// Disconnect 断开指定机器会话
func (p *ShellSessionPool) Disconnect(machineName string, handler ShellOutputHandler) error {
	p.mu.Lock()
	sm, ok := p.sessions[machineName]
	if ok {
		delete(p.sessions, machineName)
	}
	p.mu.Unlock()
	if !ok {
		return nil
	}
	return sm.Disconnect(handler)
}

// DisconnectAll 断开所有会话
func (p *ShellSessionPool) DisconnectAll(handler ShellOutputHandler) {
	p.mu.Lock()
	names := make([]string, 0, len(p.sessions))
	for name := range p.sessions {
		names = append(names, name)
	}
	p.mu.Unlock()
	for _, name := range names {
		_ = p.Disconnect(name, handler)
	}
}

// RemoveSession 从池中移除会话（远端断开时）
func (p *ShellSessionPool) RemoveSession(machineName string) {
	p.mu.Lock()
	delete(p.sessions, machineName)
	p.mu.Unlock()
}

// IsAnyConnected 是否存在活动会话
func (p *ShellSessionPool) IsAnyConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, sm := range p.sessions {
		if sm.IsConnected() {
			return true
		}
	}
	return false
}

// IsConnected 指定机器是否已连接
func (p *ShellSessionPool) IsConnected(machineName string) bool {
	p.mu.RLock()
	sm := p.sessions[machineName]
	p.mu.RUnlock()
	return sm != nil && sm.IsConnected()
}

// ListSessions 列出所有活动会话状态
func (p *ShellSessionPool) ListSessions() []define.ShellStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]define.ShellStatus, 0, len(p.sessions))
	for _, sm := range p.sessions {
		if st := sm.GetStatus(); st.Connected {
			result = append(result, *st)
		}
	}
	return result
}

// SendInput 向指定会话发送输入
func (p *ShellSessionPool) SendInput(machineName, data string) error {
	p.mu.RLock()
	sm := p.sessions[machineName]
	p.mu.RUnlock()
	if sm == nil || !sm.IsConnected() {
		return fmt.Errorf("未连接: %s", machineName)
	}
	return sm.SendInput(data)
}

// SendInterrupt 向指定会话发送 Ctrl+C
func (p *ShellSessionPool) SendInterrupt(machineName string) error {
	return p.SendInput(machineName, "\x03")
}

// Resize 调整指定会话终端尺寸
func (p *ShellSessionPool) Resize(machineName string, cols, rows int) error {
	p.mu.RLock()
	sm := p.sessions[machineName]
	p.mu.RUnlock()
	if sm == nil {
		return fmt.Errorf("未连接: %s", machineName)
	}
	return sm.Resize(cols, rows)
}
