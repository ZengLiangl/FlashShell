package machine

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"FlashDock/define"
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
