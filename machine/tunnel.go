package machine

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"FlashDock/define"
)

// TunnelType 隧道类型
const (
	TunnelLocal   = "local"   // 本地转发：本机端口 → 远端地址
	TunnelRemote  = "remote"  // 远程转发：远端端口 → 本机地址
	TunnelDynamic = "dynamic" // 动态 SOCKS5（本机）
)

// TunnelRuntime 运行中的隧道
type TunnelRuntime struct {
	Spec      define.SSHTunnel
	Listener  net.Listener
	Cancel    func()
	StartedAt int64
	Error     string
}

// TunnelManager 按机器配置管理 SSH 隧道（与 PTY 会话独立，复用 Aux/PTY SSH）
type TunnelManager struct {
	mu      sync.Mutex
	byMachine map[string][]*TunnelRuntime // configName → tunnels
}

// NewTunnelManager 创建隧道管理器
func NewTunnelManager() *TunnelManager {
	return &TunnelManager{byMachine: make(map[string][]*TunnelRuntime)}
}

// StopAllFor 停止某机器全部隧道
func (tm *TunnelManager) StopAllFor(configName string) {
	tm.mu.Lock()
	list := tm.byMachine[configName]
	delete(tm.byMachine, configName)
	tm.mu.Unlock()
	for _, t := range list {
		stopTunnel(t)
	}
}

// StopAll 停止全部
func (tm *TunnelManager) StopAll() {
	tm.mu.Lock()
	names := make([]string, 0, len(tm.byMachine))
	for n := range tm.byMachine {
		names = append(names, n)
	}
	tm.mu.Unlock()
	for _, n := range names {
		tm.StopAllFor(n)
	}
}

// StatusList 返回某机器隧道状态
func (tm *TunnelManager) StatusList(configName string) []define.SSHTunnelStatus {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	list := tm.byMachine[configName]
	out := make([]define.SSHTunnelStatus, 0, len(list))
	for _, t := range list {
		st := define.SSHTunnelStatus{
			Name:      t.Spec.Name,
			Type:      t.Spec.Type,
			LocalHost: t.Spec.LocalHost,
			LocalPort: t.Spec.LocalPort,
			RemoteHost: t.Spec.RemoteHost,
			RemotePort: t.Spec.RemotePort,
			Active:    t.Listener != nil && t.Error == "",
			Error:     t.Error,
			StartedAt: t.StartedAt,
		}
		out = append(out, st)
	}
	return out
}

// EnsureForMachine 根据配置启动本地/动态隧道；client 为已连接的 SSH
func (tm *TunnelManager) EnsureForMachine(configName string, tunnels []define.SSHTunnel, client *SSHClient) error {
	if client == nil || client.remoteMachine == nil || client.remoteMachine.SSHClient == nil {
		return fmt.Errorf("SSH 未连接，无法启动隧道")
	}
	sshClient := client.remoteMachine.SSHClient

	tm.StopAllFor(configName)

	var firstErr error
	started := make([]*TunnelRuntime, 0, len(tunnels))
	for _, spec := range tunnels {
		spec = normalizeTunnel(spec)
		if !spec.Enabled {
			continue
		}
		rt, err := startTunnel(spec, sshClient)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", tunnelLabel(spec), err)
			}
			rt = &TunnelRuntime{Spec: spec, Error: err.Error(), StartedAt: time.Now().Unix()}
		}
		started = append(started, rt)
	}
	if len(started) > 0 {
		tm.mu.Lock()
		tm.byMachine[configName] = started
		tm.mu.Unlock()
	}
	return firstErr
}

func tunnelLabel(s define.SSHTunnel) string {
	if s.Name != "" {
		return s.Name
	}
	return fmt.Sprintf("%s:%d", s.Type, s.LocalPort)
}

func normalizeTunnel(s define.SSHTunnel) define.SSHTunnel {
	s.Type = strings.ToLower(strings.TrimSpace(s.Type))
	if s.Type == "" {
		s.Type = TunnelLocal
	}
	s.LocalHost = strings.TrimSpace(s.LocalHost)
	if s.LocalHost == "" {
		s.LocalHost = "127.0.0.1"
	}
	s.RemoteHost = strings.TrimSpace(s.RemoteHost)
	if s.RemoteHost == "" && s.Type == TunnelLocal {
		s.RemoteHost = "127.0.0.1"
	}
	return s
}

func stopTunnel(t *TunnelRuntime) {
	if t == nil {
		return
	}
	if t.Cancel != nil {
		t.Cancel()
	}
	if t.Listener != nil {
		_ = t.Listener.Close()
	}
}

func startTunnel(spec define.SSHTunnel, sshClient interface {
	Dial(n, addr string) (net.Conn, error)
	Listen(n, addr string) (net.Listener, error)
}) (*TunnelRuntime, error) {
	switch spec.Type {
	case TunnelLocal:
		return startLocalForward(spec, sshClient)
	case TunnelRemote:
		return startRemoteForward(spec, sshClient)
	case TunnelDynamic:
		return startDynamicForward(spec, sshClient)
	default:
		return nil, fmt.Errorf("不支持的隧道类型: %s", spec.Type)
	}
}

func startLocalForward(spec define.SSHTunnel, sshClient interface {
	Dial(n, addr string) (net.Conn, error)
}) (*TunnelRuntime, error) {
	if spec.LocalPort <= 0 || spec.RemotePort <= 0 {
		return nil, fmt.Errorf("请填写本地端口与远端端口")
	}
	addr := net.JoinHostPort(spec.LocalHost, strconv.Itoa(spec.LocalPort))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("监听本地 %s 失败: %w", addr, err)
	}
	remoteAddr := net.JoinHostPort(spec.RemoteHost, strconv.Itoa(spec.RemotePort))
	done := make(chan struct{})
	go func() {
		for {
			local, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					return
				}
			}
			go func(l net.Conn) {
				r, err := sshClient.Dial("tcp", remoteAddr)
				if err != nil {
					_ = l.Close()
					return
				}
				pipeConn(l, r)
			}(local)
		}
	}()
	rt := &TunnelRuntime{
		Spec:      spec,
		Listener:  ln,
		StartedAt: time.Now().Unix(),
		Cancel:    func() { close(done); _ = ln.Close() },
	}
	// 连通性探测：短暂 dial 本地监听端口，确认可用
	if err := probeLocal(addr, 800*time.Millisecond); err != nil {
		stopTunnel(rt)
		return nil, fmt.Errorf("隧道已监听但探测失败: %w", err)
	}
	return rt, nil
}

func startRemoteForward(spec define.SSHTunnel, sshClient interface {
	Listen(n, addr string) (net.Listener, error)
}) (*TunnelRuntime, error) {
	if spec.RemotePort <= 0 || spec.LocalPort <= 0 {
		return nil, fmt.Errorf("请填写远端端口与本地端口")
	}
	remoteBind := net.JoinHostPort(spec.RemoteHost, strconv.Itoa(spec.RemotePort))
	if spec.RemoteHost == "" {
		remoteBind = net.JoinHostPort("0.0.0.0", strconv.Itoa(spec.RemotePort))
	}
	ln, err := sshClient.Listen("tcp", remoteBind)
	if err != nil {
		return nil, fmt.Errorf("远端监听失败: %w", err)
	}
	localAddr := net.JoinHostPort(spec.LocalHost, strconv.Itoa(spec.LocalPort))
	done := make(chan struct{})
	go func() {
		for {
			remote, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					return
				}
			}
			go func(r net.Conn) {
				l, err := net.DialTimeout("tcp", localAddr, 10*time.Second)
				if err != nil {
					_ = r.Close()
					return
				}
				pipeConn(r, l)
			}(remote)
		}
	}()
	return &TunnelRuntime{
		Spec:      spec,
		Listener:  ln,
		StartedAt: time.Now().Unix(),
		Cancel:    func() { close(done); _ = ln.Close() },
	}, nil
}

func startDynamicForward(spec define.SSHTunnel, sshClient interface {
	Dial(n, addr string) (net.Conn, error)
}) (*TunnelRuntime, error) {
	if spec.LocalPort <= 0 {
		return nil, fmt.Errorf("请填写本地 SOCKS 端口")
	}
	addr := net.JoinHostPort(spec.LocalHost, strconv.Itoa(spec.LocalPort))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("监听 SOCKS %s 失败: %w", addr, err)
	}
	done := make(chan struct{})
	go serveSOCKS5(ln, sshClient, done)
	rt := &TunnelRuntime{
		Spec:      spec,
		Listener:  ln,
		StartedAt: time.Now().Unix(),
		Cancel:    func() { close(done); _ = ln.Close() },
	}
	if err := probeLocal(addr, 800*time.Millisecond); err != nil {
		stopTunnel(rt)
		return nil, fmt.Errorf("SOCKS 隧道探测失败: %w", err)
	}
	return rt, nil
}

func probeLocal(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last error
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return nil
		}
		last = err
		time.Sleep(50 * time.Millisecond)
	}
	if last == nil {
		last = fmt.Errorf("超时")
	}
	return last
}

func pipeConn(a, b net.Conn) {
	defer a.Close()
	defer b.Close()
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(a, b); done <- struct{}{} }()
	go func() { _, _ = io.Copy(b, a); done <- struct{}{} }()
	<-done
}

// 极简 SOCKS5（无认证）转发到 SSH Dial
func serveSOCKS5(ln net.Listener, sshClient interface {
	Dial(n, addr string) (net.Conn, error)
}, done <-chan struct{}) {
	for {
		c, err := ln.Accept()
		if err != nil {
			select {
			case <-done:
				return
			default:
				return
			}
		}
		go handleSOCKS5(c, sshClient)
	}
}

func handleSOCKS5(conn net.Conn, sshClient interface {
	Dial(n, addr string) (net.Conn, error)
}) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))
	buf := make([]byte, 262)
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	if buf[0] != 0x05 {
		return
	}
	nMethods := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:nMethods]); err != nil {
		return
	}
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}
	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return
	}
	if buf[0] != 0x05 || buf[1] != 0x01 {
		_, _ = conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	var host string
	switch buf[3] {
	case 0x01: // IPv4
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case 0x03: // domain
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return
		}
		l := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:l]); err != nil {
			return
		}
		host = string(buf[:l])
	case 0x04: // IPv6
		if _, err := io.ReadFull(conn, buf[:16]); err != nil {
			return
		}
		host = net.IP(buf[:16]).String()
	default:
		_, _ = conn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	port := int(buf[0])<<8 | int(buf[1])
	target := net.JoinHostPort(host, strconv.Itoa(port))
	remote, err := sshClient.Dial("tcp", target)
	if err != nil {
		_, _ = conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	_, _ = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	_ = conn.SetDeadline(time.Time{})
	pipeConn(conn, remote)
}
