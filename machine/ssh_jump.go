package machine

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"FlashDock/define"

	"golang.org/x/crypto/ssh"
)

// MachineResolver 按名称解析机器配置（用于 ProxyJump 引用）
type MachineResolver func(name string) *define.Machine

var defaultMachineResolver MachineResolver

// SetMachineResolver 设置全局机器解析器（App 启动时注入）
func SetMachineResolver(resolver MachineResolver) {
	defaultMachineResolver = resolver
}

func resolveMachine(name string) *define.Machine {
	if defaultMachineResolver == nil || strings.TrimSpace(name) == "" {
		return nil
	}
	return defaultMachineResolver(strings.TrimSpace(name))
}

// JumpEndpoint 跳板机连接端点
type JumpEndpoint struct {
	Host string
	Port int
	User string
}

// ParseJumpEndpoint 解析跳板配置：机器名由调用方解析；此处处理 host[:port] 或 user@host[:port]
func ParseJumpEndpoint(raw string) (JumpEndpoint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return JumpEndpoint{}, fmt.Errorf("跳板配置为空")
	}
	ep := JumpEndpoint{Port: 22}
	if strings.Contains(raw, "@") {
		userHost, err := splitUserHostPort(raw)
		if err != nil {
			return JumpEndpoint{}, err
		}
		ep.User = userHost.User
		ep.Host = userHost.Host
		ep.Port = userHost.Port
		return ep, nil
	}
	host, port, err := splitHostPort(raw, 22)
	if err != nil {
		return JumpEndpoint{}, err
	}
	ep.Host = host
	ep.Port = port
	return ep, nil
}

type userHostPort struct {
	User string
	Host string
	Port int
}

func splitUserHostPort(raw string) (userHostPort, error) {
	at := strings.LastIndex(raw, "@")
	if at <= 0 {
		return userHostPort{}, fmt.Errorf("无效的跳板地址: %s", raw)
	}
	user := strings.TrimSpace(raw[:at])
	hostPart := strings.TrimSpace(raw[at+1:])
	host, port, err := splitHostPort(hostPart, 22)
	if err != nil {
		return userHostPort{}, err
	}
	return userHostPort{User: user, Host: host, Port: port}, nil
}

func splitHostPort(raw string, defaultPort int) (string, int, error) {
	if raw == "" {
		return "", 0, fmt.Errorf("主机为空")
	}
	if strings.HasPrefix(raw, "[") {
		if h, p, err := net.SplitHostPort(raw); err == nil {
			port, convErr := strconv.Atoi(p)
			if convErr != nil {
				return "", 0, convErr
			}
			return h, port, nil
		}
	}
	if h, p, err := net.SplitHostPort(raw); err == nil {
		port, convErr := strconv.Atoi(p)
		if convErr != nil {
			return "", 0, convErr
		}
		return h, port, nil
	}
	return raw, defaultPort, nil
}

// ResolveJumpEndpoint 将 ProxyJump 字段解析为连接端点；机器名优先查配置。
func ResolveJumpEndpoint(proxyJump string, target *define.Machine) (JumpEndpoint, *define.Machine, error) {
	proxyJump = strings.TrimSpace(proxyJump)
	if proxyJump == "" {
		return JumpEndpoint{}, nil, nil
	}
	if jumpMachine := resolveMachine(proxyJump); jumpMachine != nil {
		sensitive, err := jumpMachine.GetSensitiveData()
		if err != nil {
			return JumpEndpoint{}, nil, fmt.Errorf("读取跳板机敏感数据失败: %w", err)
		}
		port := sensitive.Port
		if port <= 0 {
			port = 22
		}
		return JumpEndpoint{
			Host: sensitive.Host,
			Port: port,
			User: sensitive.User,
		}, jumpMachine, nil
	}
	ep, err := ParseJumpEndpoint(proxyJump)
	if err != nil {
		return JumpEndpoint{}, nil, err
	}
	if ep.User == "" && target != nil {
		if sensitive, err := target.GetSensitiveData(); err == nil && sensitive != nil {
			ep.User = sensitive.User
		}
	}
	return ep, nil, nil
}

// ConnectRemote 建立 SSH（可选经跳板）；withSFTP 为 true 时初始化 SFTP。
func ConnectRemote(rm *define.RemoteMachine, machine *define.Machine, withSFTP bool) error {
	if rm == nil || machine == nil {
		return fmt.Errorf("连接参数无效")
	}
	sensitive, err := machine.GetSensitiveData()
	if err != nil {
		return fmt.Errorf("获取敏感数据失败: %w", err)
	}
	targetPort := sensitive.Port
	if targetPort <= 0 {
		targetPort = 22
	}
	targetAddr := fmt.Sprintf("%s:%d", sensitive.Host, targetPort)

	auth, err := buildSSHAuth(machine, sensitive)
	if err != nil {
		return err
	}
	targetConfig := &ssh.ClientConfig{
		User:            sensitive.User,
		Auth:            auth,
		HostKeyCallback: define.HostKeyCallback(),
		Timeout:         define.SSHHandshakeTimeout(),
	}

	jumpEP, jumpMachine, err := ResolveJumpEndpoint(machine.ProxyJump, machine)
	if err != nil {
		return err
	}

	handshakeTimeout := define.SSHHandshakeTimeout()
	var client *ssh.Client

	if jumpEP.Host != "" {
		jumpAuthMachine := jumpMachine
		if jumpAuthMachine == nil {
			jumpAuthMachine = machine
		}
		jumpSensitive, jErr := jumpAuthMachine.GetSensitiveData()
		if jErr != nil {
			return fmt.Errorf("读取跳板机连接信息失败: %w", jErr)
		}
		jumpUser := jumpEP.User
		if jumpUser == "" {
			jumpUser = jumpSensitive.User
		}
		jumpAuth, aErr := buildSSHAuth(jumpAuthMachine, jumpSensitive)
		if aErr != nil {
			return aErr
		}
		jumpConfig := &ssh.ClientConfig{
			User:            jumpUser,
			Auth:            jumpAuth,
			HostKeyCallback: define.HostKeyCallback(),
			Timeout:         handshakeTimeout,
		}
		jumpAddr := fmt.Sprintf("%s:%d", jumpEP.Host, jumpEP.Port)
		jumpClient, cErr := dialSSH(jumpAddr, jumpConfig, handshakeTimeout)
		if cErr != nil {
			return fmt.Errorf("连接跳板机失败: %w", cErr)
		}
		defer jumpClient.Close()

		conn, dErr := jumpClient.Dial("tcp", targetAddr)
		if dErr != nil {
			return fmt.Errorf("经跳板拨号目标失败: %w", dErr)
		}
		_ = conn.SetDeadline(time.Now().Add(handshakeTimeout))
		c, chans, reqs, nErr := ssh.NewClientConn(conn, targetAddr, targetConfig)
		if nErr != nil {
			_ = conn.Close()
			return fmt.Errorf("SSH 握手目标失败: %w", nErr)
		}
		_ = conn.SetDeadline(time.Time{})
		client = ssh.NewClient(c, chans, reqs)
	} else {
		client, err = dialSSH(targetAddr, targetConfig, handshakeTimeout)
		if err != nil {
			return fmt.Errorf("SSH连接失败: %w", err)
		}
	}

	rm.SSHClient = client
	if withSFTP {
		if err := rm.EnsureSFTP(); err != nil {
			_ = rm.Close()
			rm.SSHClient = nil
			return err
		}
	}
	return nil
}

func dialSSH(addr string, config *ssh.ClientConfig, timeout time.Duration) (*ssh.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := define.DialContext(ctx, "tcp", addr)
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

func buildSSHAuth(machine *define.Machine, sensitive *define.SensitiveData) ([]ssh.AuthMethod, error) {
	var auth []ssh.AuthMethod
	if machine.KeyFile != "" {
		key, err := loadPrivateKey(machine.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("加载私钥失败: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(key))
	}
	if sensitive.Password != "" {
		auth = append(auth, ssh.Password(sensitive.Password))
	}
	if len(auth) == 0 {
		return nil, fmt.Errorf("未配置认证方式")
	}
	return auth, nil
}

func loadPrivateKey(keyPath string) (ssh.Signer, error) {
	if strings.HasPrefix(keyPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		keyPath = filepath.Join(homeDir, keyPath[2:])
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(key)
}
