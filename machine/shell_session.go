package machine

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"FlashDock/define"

	"golang.org/x/crypto/ssh"
)

// ShellOutputHandler Shell 输出与状态回调
type ShellOutputHandler struct {
	OnLine   func(line string)
	OnData   func(data []byte)
	OnCwd    func(cwd string) // PTY 真实工作目录（OSC 7/777）
	OnStatus func(status *define.ShellStatus)
	OnClose  func()
}

// ShellSessionManager 独立 PTY Shell 会话管理
type ShellSessionManager struct {
	mu          sync.Mutex
	client      *SSHClient
	sessionID   string // 池内唯一键（web1 / web1-2）
	configName  string // 机器配置名
	host        string
	user        string
	connecting  bool
	cancelled   bool // Disconnect 后拒绝把拨号结果写回
	session     *ssh.Session
	stdin       io.WriteCloser
	borrowedSSH bool // 复用同机其它 Tab 的 SSH 传输
	cancelRead  context.CancelFunc
	readDone    chan struct{}
}

// NewShellSessionManager 创建 Shell 会话管理器
func NewShellSessionManager() *ShellSessionManager {
	return &ShellSessionManager{}
}

// GetStatus 获取当前 Shell 状态
func (sm *ShellSessionManager) GetStatus() *define.ShellStatus {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.statusLocked()
}

func (sm *ShellSessionManager) InitPending(sessionID, configName, host, user string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessionID = sessionID
	sm.configName = configName
	sm.host = host
	sm.user = user
	sm.connecting = true
	sm.cancelled = false
}

func (sm *ShellSessionManager) MarkFailed() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.connecting = false
}

func (sm *ShellSessionManager) statusLocked() *define.ShellStatus {
	connected := sm.client != nil && sm.client.IsConnected() && sm.session != nil
	connecting := sm.connecting && !connected
	configName := sm.configName
	if configName == "" {
		configName = sm.sessionID
	}
	return &define.ShellStatus{
		Connected:   connected,
		Connecting:  connecting,
		MachineName: sm.sessionID,
		ConfigName:  configName,
		TabLabel:    ShellTabLabel(sm.sessionID, configName, ShellKindRemote),
		Host:        sm.host,
		User:        sm.user,
		IsRunning:   false,
		Kind:        ShellKindRemote,
	}
}

func (sm *ShellSessionManager) emitStatus(handler ShellOutputHandler) {
	if handler.OnStatus != nil {
		handler.OnStatus(sm.GetStatus())
	}
}

func (sm *ShellSessionManager) notifyStatus(handler ShellOutputHandler) {
	sm.emitStatus(handler)
}

// SharedSSHClient 返回 PTY 使用的 SSH 客户端（供 SFTP 复用）。
func (sm *ShellSessionManager) SharedSSHClient() *SSHClient {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.client
}

// Connect 建立 SSH 连接并启动交互式 PTY Shell。sharedClient 非空时复用已有 SSH 传输（免二次认证）。
func (sm *ShellSessionManager) Connect(sessionID string, machine *define.Machine, workVars map[string]string, handler ShellOutputHandler, sharedClient *SSHClient) error {
	sm.mu.Lock()
	if sm.client != nil && sm.client.IsConnected() {
		sm.mu.Unlock()
		return nil // 幂等
	}
	sm.connecting = true
	sm.mu.Unlock()

	client := NewSSHClient(machine, workVars)
	borrowed := false
	if sharedClient != nil && sharedClient.IsConnected() && sharedClient.remoteMachine != nil {
		client = sharedClient
		borrowed = true
	} else if err := client.Connect(machine, false); err != nil {
		return err
	}

	sensitive, err := machine.GetSensitiveData()
	if err != nil {
		_ = client.Close()
		return err
	}

	session, err := client.remoteMachine.NewSession()
	if err != nil {
		_ = client.Close()
		return fmt.Errorf("创建 SSH 会话失败: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", 120, 40, modes); err != nil {
		_ = session.Close()
		_ = client.Close()
		return fmt.Errorf("请求 PTY 失败: %w", err)
	}

	if machine.AgentForwarding {
		if err := requestSessionAgentForwarding(session); err != nil {
			_ = session.Close()
			_ = client.Close()
			return fmt.Errorf("启用 Agent 转发失败: %w", err)
		}
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return fmt.Errorf("获取 stdin 失败: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		_ = client.Close()
		return fmt.Errorf("获取 stdout 失败: %w", err)
	}

	if err := session.Shell(); err != nil {
		_ = session.Close()
		_ = client.Close()
		return fmt.Errorf("启动 Shell 失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	readDone := make(chan struct{})

	if sessionID == "" {
		sessionID = machine.Name
	}

	sm.mu.Lock()
	if sm.cancelled {
		sm.mu.Unlock()
		cancel()
		_ = session.Close()
		_ = client.Close()
		return fmt.Errorf("会话已关闭")
	}
	if sm.client != nil && sm.client.IsConnected() {
		sm.mu.Unlock()
		cancel()
		_ = session.Close()
		_ = client.Close()
		return nil
	}
	sm.client = client
	sm.sessionID = sessionID
	sm.configName = machine.Name
	sm.host = sensitive.Host
	sm.user = sensitive.User
	sm.session = session
	sm.stdin = stdin
	sm.borrowedSSH = borrowed
	sm.cancelRead = cancel
	sm.readDone = readDone
	sm.connecting = false
	sm.mu.Unlock()

	if startup := strings.TrimSpace(machine.StartupCommand); startup != "" {
		_, _ = stdin.Write([]byte(startup + "\n"))
	}

	go sm.readPTY(stdout, handler, ctx, readDone)
	sm.notifyStatus(handler)
	return nil
}

func (sm *ShellSessionManager) readPTY(stdout io.Reader, handler ShellOutputHandler, ctx context.Context, done chan struct{}) {
	defer close(done)

	filter := newOscCwdFilter(func(cwd string) {
		if handler.OnCwd != nil {
			handler.OnCwd(cwd)
		}
	})

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := stdout.Read(buf)
		if n > 0 {
			cleaned := filter.Feed(buf[:n])
			if len(cleaned) > 0 && handler.OnData != nil {
				chunk := make([]byte, len(cleaned))
				copy(chunk, cleaned)
				handler.OnData(chunk)
			}
		}
		if err != nil {
			var disconnectMsg string
			if err != io.EOF {
				disconnectMsg = fmt.Sprintf("[连接断开] %v", err)
			}
			sm.mu.Lock()
			shouldClose := sm.session != nil
			sm.mu.Unlock()
			if disconnectMsg != "" && handler.OnLine != nil {
				handler.OnLine(disconnectMsg)
			}
			if shouldClose {
				if handler.OnClose != nil {
					handler.OnClose()
				}
				sm.mu.Lock()
				sm.closeResourcesLocked()
				sm.mu.Unlock()
				sm.notifyStatus(handler)
			}
			return
		}
	}
}

// Disconnect 断开 Shell 连接
func (sm *ShellSessionManager) Disconnect(handler ShellOutputHandler) error {
	sm.mu.Lock()
	sm.cancelled = true
	sm.connecting = false
	label := ShellTabLabel(sm.sessionID, sm.configName, ShellKindRemote)
	if sm.cancelRead != nil {
		sm.cancelRead()
	}
	readDone := sm.readDone
	sm.closeResourcesLocked()
	sm.mu.Unlock()

	if readDone != nil {
		<-readDone
	}

	if label != "" && handler.OnLine != nil {
		handler.OnLine(fmt.Sprintf("已断开与 %s 的连接", label))
	}
	sm.emitStatus(handler)
	return nil
}

func (sm *ShellSessionManager) closeResourcesLocked() {
	sm.cancelRead = nil
	sm.readDone = nil
	if sm.stdin != nil {
		_ = sm.stdin.Close()
		sm.stdin = nil
	}
	if sm.session != nil {
		_ = sm.session.Close()
		sm.session = nil
	}
	if sm.client != nil && !sm.borrowedSSH {
		_ = sm.client.Close()
	}
	sm.client = nil
	sm.borrowedSSH = false
	sm.sessionID = ""
	sm.configName = ""
	sm.host = ""
	sm.user = ""
}

// IsConnected 是否已连接
func (sm *ShellSessionManager) IsConnected() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.client != nil && sm.client.IsConnected() && sm.session != nil
}

// SendInput 向 PTY 写入输入
func (sm *ShellSessionManager) SendInput(data string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.stdin == nil {
		return fmt.Errorf("未连接远程机器")
	}
	_, err := sm.stdin.Write([]byte(data))
	return err
}

// SendInterrupt 发送 Ctrl+C
func (sm *ShellSessionManager) SendInterrupt() error {
	return sm.SendInput("\x03")
}

// Resize 调整 PTY 窗口大小
func (sm *ShellSessionManager) Resize(cols, rows int) error {
	sm.mu.Lock()
	session := sm.session
	sm.mu.Unlock()
	if session == nil {
		return nil
	}
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}
	return session.WindowChange(rows, cols)
}
