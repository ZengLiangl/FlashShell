package data

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// ImportOpenSSHKnownHosts 从 OpenSSH known_hosts 文件导入（合并）
func (m *HostKeyManager) ImportOpenSSHKnownHosts(path string) (int, error) {
	if strings.TrimSpace(path) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return 0, err
		}
		path = filepath.Join(home, ".ssh", "known_hosts")
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("打开 known_hosts 失败: %w", err)
	}
	defer f.Close()

	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	br := bufio.NewReader(f)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		host, port, fp, ok := parseOpenSSHKnownHostsLine(line)
		if !ok || fp == "" {
			continue
		}
		addr := hostKeyAddr(host, port)
		m.hosts[addr] = fp
		n++
	}
	if n > 0 {
		if err := m.saveLocked(); err != nil {
			return n, err
		}
	}
	return n, nil
}

func parseOpenSSHKnownHostsLine(line string) (host string, port int, fingerprint string, ok bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return "", 0, "", false
	}
	hostPart := fields[0]
	keyType := fields[1]
	keyBlob := fields[2]
	if strings.HasPrefix(hostPart, "|") {
		// 哈希主机名暂不支持解析，跳过
		return "", 0, "", false
	}
	host, port = parseKnownHostsHostPart(hostPart)
	pubKey, err := base64.StdEncoding.DecodeString(keyBlob)
	if err != nil {
		return "", 0, "", false
	}
	pk, err := ssh.ParsePublicKey(pubKey)
	if err != nil {
		return "", 0, "", false
	}
	if keyType != "" && !strings.HasPrefix(pk.Type(), keyType) {
		// 类型字段可能与实际略有差异，仍以解析结果为准
	}
	return host, port, FingerprintSHA256(pk), true
}

func parseKnownHostsHostPart(raw string) (string, int) {
	port := 22
	if strings.Contains(raw, ",") {
		parts := strings.Split(raw, ",")
		raw = strings.TrimSpace(parts[0])
	}
	if strings.HasPrefix(raw, "[") && strings.Contains(raw, "]") {
		end := strings.Index(raw, "]")
		inner := raw[1:end]
		if h, p, err := splitHostPortKnown(inner); err == nil {
			return h, p
		}
		return inner, 22
	}
	if h, p, err := splitHostPortKnown(raw); err == nil {
		return h, p
	}
	return raw, port
}

func splitHostPortKnown(raw string) (string, int, error) {
	if strings.Contains(raw, ":") {
		idx := strings.LastIndex(raw, ":")
		host := strings.TrimSpace(raw[:idx])
		portStr := strings.TrimSpace(raw[idx+1:])
		if portStr != "" {
			var p int
			if _, err := fmt.Sscanf(portStr, "%d", &p); err == nil && p > 0 {
				return host, p, nil
			}
		}
	}
	return raw, 22, nil
}
