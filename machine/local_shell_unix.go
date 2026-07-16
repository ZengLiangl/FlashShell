//go:build !windows

package machine

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"FlashDock/define"

	"github.com/creack/pty"
)

// LocalShellSession Unix 本地 PTY 会话
type LocalShellSession struct {
	mu         sync.Mutex
	id         string
	cmd        *exec.Cmd
	ptmx       *os.File
	cancelRead context.CancelFunc
	readDone   chan struct{}
	connected  bool
}

// NewLocalShellSession 创建本地会话
func NewLocalShellSession(id string) *LocalShellSession {
	return &LocalShellSession{id: id}
}

// Start 启动本地 shell
func (s *LocalShellSession) Start(handler ShellOutputHandler) error {
	shell, args := defaultUnixShell()
	cmd := exec.Command(shell, args...)
	cmd.Env = os.Environ()
	if dir := localShellStartDir(); dir != "" {
		cmd.Dir = dir
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("启动本地终端失败: %w", err)
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 40, Cols: 120})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	s.mu.Lock()
	s.cmd = cmd
	s.ptmx = ptmx
	s.cancelRead = cancel
	s.readDone = done
	s.connected = true
	s.mu.Unlock()

	go s.readLoop(handler, ctx, done)

	if handler.OnStatus != nil {
		handler.OnStatus(s.GetStatus())
	}
	return nil
}

func defaultUnixShell() (string, []string) {
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh, []string{"-l"}
	}
	if _, err := os.Stat("/bin/zsh"); err == nil {
		return "/bin/zsh", []string{"-l"}
	}
	if _, err := os.Stat("/bin/bash"); err == nil {
		return "/bin/bash", []string{"-l"}
	}
	return "/bin/sh", nil
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
		ptmx := s.ptmx
		s.mu.Unlock()
		if ptmx == nil {
			return
		}
		n, err := ptmx.Read(buf)
		if n > 0 && handler.OnData != nil {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			handler.OnData(chunk)
		}
		if err != nil {
			if err != io.EOF && handler.OnLine != nil {
				handler.OnLine(fmt.Sprintf("[本机退出] %v", err))
			}
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
			return
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

// Write 写入 PTY
func (s *LocalShellSession) Write(data []byte) error {
	s.mu.Lock()
	ptmx := s.ptmx
	connected := s.connected
	s.mu.Unlock()
	if !connected || ptmx == nil {
		return fmt.Errorf("本地终端未连接")
	}
	_, err := ptmx.Write(data)
	return err
}

// Resize 调整窗口
func (s *LocalShellSession) Resize(cols, rows int) error {
	s.mu.Lock()
	ptmx := s.ptmx
	s.mu.Unlock()
	if ptmx == nil {
		return nil
	}
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
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
	if s.ptmx != nil {
		_ = s.ptmx.Close()
		s.ptmx = nil
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		s.cmd = nil
	}
}
