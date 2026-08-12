//go:build windows

package machine

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"FlashDock/define"

	"github.com/UserExistsError/conpty"
)

// LocalShellSession Windows ConPTY 本地会话
type LocalShellSession struct {
	mu         sync.Mutex
	id         string
	command    string
	cpty       *conpty.ConPty
	cancelRead context.CancelFunc
	readDone   chan struct{}
	connected  bool
}

// NewLocalShellSession 创建本地会话
func NewLocalShellSession(id string) *LocalShellSession {
	return &LocalShellSession{id: id}
}

// SetCommand 指定启动命令行（空则用默认）
func (s *LocalShellSession) SetCommand(cmd string) {
	s.mu.Lock()
	s.command = strings.TrimSpace(cmd)
	s.mu.Unlock()
}

// Start 启动本地 shell
func (s *LocalShellSession) Start(handler ShellOutputHandler) error {
	if !conpty.IsConPtyAvailable() {
		return fmt.Errorf("当前 Windows 版本不支持 ConPTY 本地终端")
	}
	s.mu.Lock()
	cmdLine := s.command
	s.mu.Unlock()
	if cmdLine == "" {
		cmdLine = defaultWindowsShell()
	}
	opts := []conpty.ConPtyOption{conpty.ConPtyDimensions(120, 40)}
	if dir := localShellStartDir(); dir != "" {
		opts = append(opts, conpty.ConPtyWorkDir(dir))
	}
	cpty, err := conpty.Start(cmdLine, opts...)
	if err != nil {
		return fmt.Errorf("启动本地终端失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	s.mu.Lock()
	s.cpty = cpty
	s.cancelRead = cancel
	s.readDone = done
	s.connected = true
	s.mu.Unlock()

	go s.readLoop(handler, ctx, done)
	go s.waitExit(handler)

	if handler.OnStatus != nil {
		handler.OnStatus(s.GetStatus())
	}
	return nil
}

func defaultWindowsShell() string {
	candidates := []string{}
	if sys := os.Getenv("SystemRoot"); sys != "" {
		candidates = append(candidates,
			filepath.Join(sys, "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
			filepath.Join(sys, "System32", "cmd.exe"),
		)
	}
	candidates = append(candidates,
		`C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`,
		`C:\Windows\System32\cmd.exe`,
		"powershell.exe",
		"cmd.exe",
	)
	for _, c := range candidates {
		if c == "powershell.exe" || c == "cmd.exe" {
			return c
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return "cmd.exe"
}

func (s *LocalShellSession) readLoop(handler ShellOutputHandler, ctx context.Context, done chan struct{}) {
	defer close(done)
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		s.mu.Lock()
		cpty := s.cpty
		s.mu.Unlock()
		if cpty == nil {
			return
		}
		n, err := cpty.Read(buf)
		if n > 0 && handler.OnData != nil {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			handler.OnData(chunk)
		}
		if err != nil {
			if err != io.EOF && handler.OnLine != nil {
				handler.OnLine(fmt.Sprintf("[本机退出] %v", err))
			}
			return
		}
	}
}

func (s *LocalShellSession) waitExit(handler ShellOutputHandler) {
	s.mu.Lock()
	cpty := s.cpty
	s.mu.Unlock()
	if cpty == nil {
		return
	}
	_, _ = cpty.Wait(context.Background())
	s.mu.Lock()
	was := s.connected
	s.closeLocked()
	s.mu.Unlock()
	if was {
		if handler.OnClose != nil {
			handler.OnClose()
		}
		if handler.OnStatus != nil {
			handler.OnStatus(s.GetStatus())
		}
	}
}

// IsConnected 是否运行中
func (s *LocalShellSession) IsConnected() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connected
}

// GetStatus 状态
func (s *LocalShellSession) GetStatus() *define.ShellStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return &define.ShellStatus{
		Connected:   s.connected,
		MachineName: s.id,
		ConfigName:  s.id,
		TabLabel:    ShellTabLabel(s.id, s.id, ShellKindLocal),
		Host:        "localhost",
		User:        localUserName(),
		Kind:        ShellKindLocal,
	}
}

// Write 写入
func (s *LocalShellSession) Write(data []byte) error {
	s.mu.Lock()
	cpty := s.cpty
	connected := s.connected
	s.mu.Unlock()
	if !connected || cpty == nil {
		return fmt.Errorf("本地终端未连接")
	}
	_, err := cpty.Write(data)
	return err
}

// Resize 调整窗口
func (s *LocalShellSession) Resize(cols, rows int) error {
	s.mu.Lock()
	cpty := s.cpty
	s.mu.Unlock()
	if cpty == nil {
		return nil
	}
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cpty.Resize(cols, rows)
}

// Close 关闭会话
func (s *LocalShellSession) Close(handler ShellOutputHandler) error {
	s.mu.Lock()
	if s.cancelRead != nil {
		s.cancelRead()
	}
	done := s.readDone
	s.closeLocked()
	s.mu.Unlock()
	if done != nil {
		<-done
	}
	if handler.OnStatus != nil {
		handler.OnStatus(s.GetStatus())
	}
	return nil
}

func (s *LocalShellSession) closeLocked() {
	s.connected = false
	if s.cpty != nil {
		_ = s.cpty.Close()
		s.cpty = nil
	}
}
