package machine

import (
	"fmt"
	"os/user"
	"strings"
	"sync"

	"FlashDock/define"
)

const (
	LocalShellIDPrefix = "local"
	ShellKindLocal     = "local"
	ShellKindRemote    = "remote"
)

// IsLocalShellID 判断是否为本地终端会话 ID
func IsLocalShellID(id string) bool {
	return id == LocalShellIDPrefix || strings.HasPrefix(id, LocalShellIDPrefix+"-")
}

// LocalShellPool 本地 PTY 会话池
type LocalShellPool struct {
	mu       sync.RWMutex
	sessions map[string]*LocalShellSession
	seq      int
}

// NewLocalShellPool 创建本地会话池
func NewLocalShellPool() *LocalShellPool {
	return &LocalShellPool{
		sessions: make(map[string]*LocalShellSession),
	}
}

func (p *LocalShellPool) nextID() string {
	p.seq++
	if p.seq == 1 {
		return LocalShellIDPrefix
	}
	return fmt.Sprintf("%s-%d", LocalShellIDPrefix, p.seq)
}

// Connect 新建本地终端，返回会话 ID
func (p *LocalShellPool) Connect(handlerFor func(id string) ShellOutputHandler) (string, error) {
	p.mu.Lock()
	id := p.nextID()
	sess := NewLocalShellSession(id)
	p.sessions[id] = sess
	p.mu.Unlock()

	if err := sess.Start(handlerFor(id)); err != nil {
		p.mu.Lock()
		delete(p.sessions, id)
		p.mu.Unlock()
		return "", err
	}
	return id, nil
}

// ConnectID 按指定 ID 启动（用于重连）
func (p *LocalShellPool) ConnectID(id string, handlerFor func(id string) ShellOutputHandler) error {
	if id == "" {
		return fmt.Errorf("会话 ID 为空")
	}
	p.mu.Lock()
	if existing, ok := p.sessions[id]; ok && existing.IsConnected() {
		p.mu.Unlock()
		return nil
	}
	sess := NewLocalShellSession(id)
	p.sessions[id] = sess
	p.mu.Unlock()

	if err := sess.Start(handlerFor(id)); err != nil {
		p.mu.Lock()
		delete(p.sessions, id)
		p.mu.Unlock()
		return err
	}
	return nil
}

// GetSession 获取会话
func (p *LocalShellPool) GetSession(id string) *LocalShellSession {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sessions[id]
}

// IsConnected 是否已连接
func (p *LocalShellPool) IsConnected(id string) bool {
	s := p.GetSession(id)
	return s != nil && s.IsConnected()
}

// Disconnect 断开会话
func (p *LocalShellPool) Disconnect(id string, handler ShellOutputHandler) error {
	p.mu.Lock()
	s, ok := p.sessions[id]
	if ok {
		delete(p.sessions, id)
	}
	p.mu.Unlock()
	if !ok {
		return nil
	}
	return s.Close(handler)
}

// DisconnectAll 断开全部
func (p *LocalShellPool) DisconnectAll(handlerFor func(id string) ShellOutputHandler) {
	p.mu.Lock()
	ids := make([]string, 0, len(p.sessions))
	for id := range p.sessions {
		ids = append(ids, id)
	}
	p.mu.Unlock()
	for _, id := range ids {
		_ = p.Disconnect(id, handlerFor(id))
	}
}

// RemoveSession 移除会话记录
func (p *LocalShellPool) RemoveSession(id string) {
	p.mu.Lock()
	delete(p.sessions, id)
	p.mu.Unlock()
}

// ListSessions 列出活动本地会话
func (p *LocalShellPool) ListSessions() []define.ShellStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]define.ShellStatus, 0, len(p.sessions))
	for _, s := range p.sessions {
		if st := s.GetStatus(); st.Connected {
			out = append(out, *st)
		}
	}
	return out
}

// SendInput 发送输入
func (p *LocalShellPool) SendInput(id, data string) error {
	s := p.GetSession(id)
	if s == nil || !s.IsConnected() {
		return fmt.Errorf("未连接: %s", id)
	}
	return s.Write([]byte(data))
}

// SendInterrupt 发送 Ctrl+C
func (p *LocalShellPool) SendInterrupt(id string) error {
	return p.SendInput(id, "\x03")
}

// Resize 调整尺寸
func (p *LocalShellPool) Resize(id string, cols, rows int) error {
	s := p.GetSession(id)
	if s == nil {
		return fmt.Errorf("未连接: %s", id)
	}
	return s.Resize(cols, rows)
}

func localUserName() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return ""
}
