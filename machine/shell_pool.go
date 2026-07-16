package machine

import (
	"fmt"
	"sync"

	"FlashDock/define"
)

// ShellConnectCallback 异步连接完成回调（err 为 nil 表示成功）
type ShellConnectCallback func(sessionID string, err error)

// ShellSessionPool 管理远程 PTY 会话（同一机器可开多个：web1 / web1-2）
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

// GetSession 获取指定会话（可能为 nil）
func (p *ShellSessionPool) GetSession(sessionID string) *ShellSessionManager {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sessions[sessionID]
}

// nextSessionID 分配最小可用序号（关闭 tab 后复用空位）。
func (p *ShellSessionPool) nextSessionID(configName string) string {
	used := make(map[int]bool)
	for id, sm := range p.sessions {
		st := sm.GetStatus()
		if st == nil || st.ConfigName != configName {
			continue
		}
		if idx := RemoteSessionIndexForConfig(id, configName); idx >= 1 {
			used[idx] = true
			continue
		}
		if id == configName {
			used[1] = true
		}
	}
	for slot := 1; ; slot++ {
		if !used[slot] {
			return FormatRemoteSessionID(configName, slot)
		}
	}
}

func pendingStatus(machine *define.Machine, sessionID string) (host, user string) {
	if machine == nil {
		return "", ""
	}
	if sensitive, err := machine.GetSensitiveData(); err == nil && sensitive != nil {
		return sensitive.Host, sensitive.User
	}
	return "", ""
}

func (p *ShellSessionPool) runConnect(sessionID string, sm *ShellSessionManager, machine *define.Machine, workVars map[string]string, handler ShellOutputHandler, onComplete ShellConnectCallback) {
	err := sm.Connect(sessionID, machine, workVars, handler)
	if err != nil {
		sm.MarkFailed()
		if handler.OnLine != nil {
			handler.OnLine(fmt.Sprintf("连接失败: %v", err))
		}
		// 失败会话移出池，避免占着「连接中」坑位；前端 tab 仍保留以便重试
		p.RemoveSession(sessionID)
	}
	if handler.OnStatus != nil {
		handler.OnStatus(sm.GetStatus())
	}
	if onComplete != nil {
		onComplete(sessionID, err)
	}
}

// Connect 新建远程会话并异步拨号，立即返回会话 ID
func (p *ShellSessionPool) Connect(machine *define.Machine, workVars map[string]string, handlerFor func(sessionID string) ShellOutputHandler, onComplete ShellConnectCallback) (string, error) {
	if machine == nil || machine.Name == "" {
		return "", fmt.Errorf("机器配置无效")
	}
	host, user := pendingStatus(machine, "")
	p.mu.Lock()
	sessionID := p.nextSessionID(machine.Name)
	sm := NewShellSessionManager()
	sm.InitPending(sessionID, machine.Name, host, user)
	p.sessions[sessionID] = sm
	p.mu.Unlock()

	handler := handlerFor(sessionID)
	go p.runConnect(sessionID, sm, machine, workVars, handler, onComplete)
	return sessionID, nil
}

// ConnectID 按指定会话 ID 异步连接（软断开后重连）
func (p *ShellSessionPool) ConnectID(sessionID string, machine *define.Machine, workVars map[string]string, handler ShellOutputHandler, onComplete ShellConnectCallback) error {
	if sessionID == "" || machine == nil {
		return fmt.Errorf("会话或机器配置无效")
	}
	host, user := pendingStatus(machine, sessionID)
	p.mu.Lock()
	if existing, ok := p.sessions[sessionID]; ok && existing.IsConnected() {
		p.mu.Unlock()
		return nil
	}
	sm, ok := p.sessions[sessionID]
	if !ok {
		sm = NewShellSessionManager()
		p.sessions[sessionID] = sm
	}
	sm.InitPending(sessionID, machine.Name, host, user)
	p.mu.Unlock()

	go p.runConnect(sessionID, sm, machine, workVars, handler, onComplete)
	return nil
}

// Disconnect 断开会话
func (p *ShellSessionPool) Disconnect(sessionID string, handler ShellOutputHandler) error {
	p.mu.Lock()
	sm, ok := p.sessions[sessionID]
	if ok {
		delete(p.sessions, sessionID)
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
	ids := make([]string, 0, len(p.sessions))
	for id := range p.sessions {
		ids = append(ids, id)
	}
	p.mu.Unlock()
	for _, id := range ids {
		_ = p.Disconnect(id, handler)
	}
}

// RemoveSession 从池中移除会话（远端断开时）
func (p *ShellSessionPool) RemoveSession(sessionID string) {
	p.mu.Lock()
	delete(p.sessions, sessionID)
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

// IsConnected 指定会话是否已连接
func (p *ShellSessionPool) IsConnected(sessionID string) bool {
	p.mu.RLock()
	sm := p.sessions[sessionID]
	p.mu.RUnlock()
	return sm != nil && sm.IsConnected()
}

// HasConnectedConfig 该机器配置是否仍有活动会话
func (p *ShellSessionPool) HasConnectedConfig(configName string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, sm := range p.sessions {
		if !sm.IsConnected() {
			continue
		}
		st := sm.GetStatus()
		if st != nil && st.ConfigName == configName {
			return true
		}
	}
	return false
}

// CountConnectedConfig 该机器配置的活动会话数
func (p *ShellSessionPool) CountConnectedConfig(configName string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	n := 0
	for _, sm := range p.sessions {
		if !sm.IsConnected() {
			continue
		}
		st := sm.GetStatus()
		if st != nil && st.ConfigName == configName {
			n++
		}
	}
	return n
}

// FirstSessionOfConfig 返回该配置下任一活动会话 ID
func (p *ShellSessionPool) FirstSessionOfConfig(configName string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for id, sm := range p.sessions {
		if !sm.IsConnected() {
			continue
		}
		st := sm.GetStatus()
		if st != nil && st.ConfigName == configName {
			return id
		}
	}
	return ""
}

// ListSessions 列出所有活动或连接中的会话状态
func (p *ShellSessionPool) ListSessions() []define.ShellStatus {
	p.mu.RLock()
	sms := make([]*ShellSessionManager, 0, len(p.sessions))
	for _, sm := range p.sessions {
		sms = append(sms, sm)
	}
	p.mu.RUnlock()

	result := make([]define.ShellStatus, 0, len(sms))
	for _, sm := range sms {
		st := sm.GetStatus()
		if st == nil {
			continue
		}
		if st.Connected || st.Connecting {
			result = append(result, *st)
		}
	}
	return result
}

// SendInput 向指定会话发送输入
func (p *ShellSessionPool) SendInput(sessionID, data string) error {
	p.mu.RLock()
	sm := p.sessions[sessionID]
	p.mu.RUnlock()
	if sm == nil || !sm.IsConnected() {
		return fmt.Errorf("未连接: %s", sessionID)
	}
	return sm.SendInput(data)
}

// BroadcastInput 向多个会话广播输入（忽略单个失败）
func (p *ShellSessionPool) BroadcastInput(sessionIDs []string, data string) (ok int, err error) {
	var firstErr error
	for _, id := range sessionIDs {
		if e := p.SendInput(id, data); e != nil {
			if firstErr == nil {
				firstErr = e
			}
			continue
		}
		ok++
	}
	return ok, firstErr
}

// SendInterrupt 向指定会话发送 Ctrl+C
func (p *ShellSessionPool) SendInterrupt(sessionID string) error {
	return p.SendInput(sessionID, "\x03")
}

// Resize 调整指定会话终端尺寸
func (p *ShellSessionPool) Resize(sessionID string, cols, rows int) error {
	p.mu.RLock()
	sm := p.sessions[sessionID]
	p.mu.RUnlock()
	if sm == nil {
		return fmt.Errorf("未连接: %s", sessionID)
	}
	return sm.Resize(cols, rows)
}
