package machine

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"FlashDock/define"
	"FlashDock/netproxy"

	"golang.org/x/crypto/ssh"
)

// JumpHop 跳板链中的一跳
type JumpHop struct {
	Endpoint JumpEndpoint
	Machine  *define.Machine
}

// ResolveJumpChain 解析跳板链；JumpChain 优先，否则回退单跳 ProxyJump
func ResolveJumpChain(target *define.Machine) ([]JumpHop, error) {
	if target == nil {
		return nil, nil
	}
	raws := target.JumpChain
	if len(raws) == 0 {
		if strings.TrimSpace(target.ProxyJump) == "" {
			return nil, nil
		}
		raws = []string{target.ProxyJump}
	}
	hops := make([]JumpHop, 0, len(raws))
	for _, raw := range raws {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		ep, jumpMachine, err := ResolveJumpEndpoint(raw, target)
		if err != nil {
			return nil, err
		}
		if ep.Host == "" {
			return nil, fmt.Errorf("跳板配置无效: %s", raw)
		}
		hops = append(hops, JumpHop{Endpoint: ep, Machine: jumpMachine})
	}
	return hops, nil
}

func machineDialContext(machine *define.Machine) func(context.Context, string, string) (net.Conn, error) {
	if machine == nil || machine.ProxyOverride == nil {
		return define.DialContext
	}
	override := *machine.ProxyOverride
	define.NormalizeMachineProxyOverride(&override)
	switch override.Mode {
	case "none":
		return directDialContext
	case "manual":
		if override.Host == "" || override.Port <= 0 {
			return define.DialContext
		}
		password, _ := override.GetMachineProxyPassword()
		settings := netproxy.Settings{
			Mode:     netproxy.ModeManual,
			Type:     override.Type,
			Host:     override.Host,
			Port:     override.Port,
			User:     override.User,
			Password: password,
		}
		return func(ctx context.Context, network, address string) (net.Conn, error) {
			return netproxy.DialWith(ctx, settings, network, address)
		}
	default:
		return define.DialContext
	}
}

func directDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	return d.DialContext(ctx, network, address)
}

func connectThroughHops(
	hops []JumpHop,
	targetAddr string,
	targetConfig *ssh.ClientConfig,
	targetMachine *define.Machine,
	handshakeTimeout time.Duration,
) (*ssh.Client, error) {
	dialFn := machineDialContext(targetMachine)
	if len(hops) == 0 {
		return dialSSHWith(targetAddr, targetConfig, handshakeTimeout, dialFn)
	}

	var lastClient *ssh.Client
	defer func() {
		if lastClient != nil {
			_ = lastClient.Close()
		}
	}()

	for i, hop := range hops {
		hopMachine := hop.Machine
		if hopMachine == nil {
			hopMachine = targetMachine
		}
		hopDial := machineDialContext(hopMachine)
		jumpSensitive, err := hopMachine.GetSensitiveData()
		if err != nil {
			return nil, fmt.Errorf("读取跳板机连接信息失败: %w", err)
		}
		jumpUser := hop.Endpoint.User
		if jumpUser == "" {
			jumpUser = jumpSensitive.User
		}
		jumpAuth, err := buildSSHAuth(hopMachine, jumpSensitive)
		if err != nil {
			return nil, err
		}
		jumpConfig := buildSSHClientConfig(jumpUser, jumpAuth, hopMachine, handshakeTimeout)
		jumpAddr := fmt.Sprintf("%s:%d", hop.Endpoint.Host, hop.Endpoint.Port)

		var jumpClient *ssh.Client
		if lastClient == nil {
			jumpClient, err = dialSSHWith(jumpAddr, jumpConfig, handshakeTimeout, hopDial)
		} else {
			jumpClient, err = dialSSHThrough(lastClient, jumpAddr, jumpConfig, handshakeTimeout)
		}
		if err != nil {
			return nil, fmt.Errorf("连接跳板 %d (%s) 失败: %w", i+1, hop.Endpoint.Host, err)
		}
		if lastClient != nil {
			_ = lastClient.Close()
		}
		lastClient = jumpClient
	}

	client, err := dialSSHThrough(lastClient, targetAddr, targetConfig, handshakeTimeout)
	if err != nil {
		return nil, err
	}
	lastClient = nil
	return client, nil
}

func dialSSHThrough(existing *ssh.Client, addr string, config *ssh.ClientConfig, timeout time.Duration) (*ssh.Client, error) {
	if existing == nil {
		return nil, fmt.Errorf("跳板 SSH 未连接")
	}
	conn, err := existing.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return ssh.NewClient(c, chans, reqs), nil
}

func dialSSHWith(addr string, config *ssh.ClientConfig, timeout time.Duration, dialFn func(context.Context, string, string) (net.Conn, error)) (*ssh.Client, error) {
	if dialFn == nil {
		dialFn = define.DialContext
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := dialFn(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return ssh.NewClient(c, chans, reqs), nil
}

func dialSSH(addr string, config *ssh.ClientConfig, timeout time.Duration) (*ssh.Client, error) {
	return dialSSHWith(addr, config, timeout, define.DialContext)
}
