package machine

import (
	"context"
	"fmt"
	"io"
	"sync"

	"quick-cmd/define"

	"golang.org/x/crypto/ssh"
)

// ShellOutputHandler Shell 输出与状态回调
type ShellOutputHandler struct {
	OnLine   func(line string)
	OnData   func(data []byte)
	OnStatus func(status *define.ShellStatus)
	OnClose  func()
}

// ShellSessionManager 独立 PTY Shell 会话管理
type ShellSessionManager struct {
	mu          sync.Mutex
	client      *SSHClient
	machineName string
	host        string
	user        string
	session     *ssh.Session
	stdin       io.WriteCloser
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

func (sm *ShellSessionManager) statusLocked() *define.ShellStatus {
	connected := sm.client != nil && sm.client.IsConnected() && sm.session != nil
	return &define.ShellStatus{
		Connected:   connected,
		MachineName: sm.machineName,
		Host:        sm.host,
		User:        sm.user,
		IsRunning:   false,
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

// Connect 建立 SSH 连接并启动交互式 PTY Shell
func (sm *ShellSessionManager) Connect(machine *define.Machine, workVars map[string]string, handler ShellOutputHandler) error {
	sm.mu.Lock()

	if sm.client != nil && sm.client.IsConnected() {
		sm.mu.Unlock()
		return fmt.Errorf("已连接到 %s，请先断开", sm.machineName)
	}

	client := NewSSHClient(machine, workVars)
	if err := client.Connect(machine, false); err != nil {
		sm.mu.Unlock()
		return err
	}

	sensitive, err := machine.GetSensitiveData()
	if err != nil {
		client.Close()
		sm.mu.Unlock()
		return err
	}

	session, err := client.remoteMachine.NewSession()
	if err != nil {
		client.Close()
		sm.mu.Unlock()
		return fmt.Errorf("创建 SSH 会话失败: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", 120, 40, modes); err != nil {
		session.Close()
		client.Close()
		sm.mu.Unlock()
		return fmt.Errorf("请求 PTY 失败: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		sm.mu.Unlock()
		return fmt.Errorf("获取 stdin 失败: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		sm.mu.Unlock()
		return fmt.Errorf("获取 stdout 失败: %w", err)
	}

	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		sm.mu.Unlock()
		return fmt.Errorf("启动 Shell 失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	readDone := make(chan struct{})

	sm.client = client
	sm.machineName = machine.Name
	sm.host = sensitive.Host
	sm.user = sensitive.User
	sm.session = session
	sm.stdin = stdin
	sm.cancelRead = cancel
	sm.readDone = readDone

	go sm.readPTY(stdout, handler, ctx, readDone)

	connectMsg := fmt.Sprintf("已连接到 %s (%s@%s:%d)", machine.Name, sensitive.User, sensitive.Host, sensitive.Port)
	sm.mu.Unlock()

	if handler.OnLine != nil {
		handler.OnLine(connectMsg)
	}
	sm.notifyStatus(handler)
	return nil
}

func (sm *ShellSessionManager) readPTY(stdout io.Reader, handler ShellOutputHandler, ctx context.Context, done chan struct{}) {
	defer close(done)

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := stdout.Read(buf)
		if n > 0 && handler.OnData != nil {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			handler.OnData(chunk)
		}
		if err != nil {
			var disconnectMsg string
			if err != io.EOF {
				disconnectMsg = fmt.Sprintf("[连接断开] %v", err)
			}
			sm.mu.Lock()
			shouldClose := sm.session != nil
			if shouldClose {
				sm.closeResourcesLocked()
			}
			sm.mu.Unlock()
			if disconnectMsg != "" && handler.OnLine != nil {
				handler.OnLine(disconnectMsg)
			}
			if shouldClose {
				sm.notifyStatus(handler)
				if handler.OnClose != nil {
					handler.OnClose()
				}
			}
			return
		}
	}
}

// Disconnect 断开 Shell 连接
func (sm *ShellSessionManager) Disconnect(handler ShellOutputHandler) error {
	sm.mu.Lock()
	name := sm.machineName
	if sm.cancelRead != nil {
		sm.cancelRead()
	}
	readDone := sm.readDone
	sm.closeResourcesLocked()
	sm.mu.Unlock()

	if readDone != nil {
		<-readDone
	}

	if name != "" && handler.OnLine != nil {
		handler.OnLine(fmt.Sprintf("已断开与 %s 的连接", name))
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
	if sm.client != nil {
		_ = sm.client.Close()
		sm.client = nil
	}
	sm.machineName = ""
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
