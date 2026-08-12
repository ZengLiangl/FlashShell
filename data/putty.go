package data

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"FlashDock/define"

	"github.com/google/uuid"
)

const defaultPuttyPassword = "123456"

var puttySessionSection = regexp.MustCompile(`(?i)^\[HKEY_(?:CURRENT_USER|LOCAL_MACHINE)\\Software\\SimonTatham\\PuTTY\\Sessions\\(.+)\]$`)

// ParsedPuttySession 解析后的 PuTTY 会话
type ParsedPuttySession struct {
	Name     string
	Host     string
	Port     int
	User     string
	Protocol string
}

// ParsePuttyRegFile 解析 PuTTY 注册表导出文件
func ParsePuttyRegFile(path string) ([]ParsedPuttySession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParsePuttyRegContent(string(data))
}

// ParsePuttyRegContent 解析 PuTTY .reg 内容
func ParsePuttyRegContent(text string) ([]ParsedPuttySession, error) {
	var sessions []ParsedPuttySession
	var current *ParsedPuttySession

	flush := func() {
		if current != nil {
			normalizePuttyHostUser(current)
			if strings.TrimSpace(current.Host) != "" {
				sessions = append(sessions, *current)
			}
		}
		current = nil
	}

	scanner := bufio.NewScanner(strings.NewReader(text))
	// PuTTY 会话名可能很长
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" {
			continue
		}
		if m := puttySessionSection.FindStringSubmatch(trimmed); len(m) == 2 {
			flush()
			current = &ParsedPuttySession{
				Name: decodePuttySessionName(m[1]),
				Port: 22,
			}
			continue
		}
		if current == nil {
			continue
		}
		if !strings.HasPrefix(trimmed, `"`) {
			continue
		}
		eq := strings.Index(trimmed, "=")
		if eq <= 0 {
			continue
		}
		key := strings.Trim(trimmed[:eq], `"`)
		value := trimmed[eq+1:]
		switch key {
		case "HostName":
			current.Host = decodeRegString(value)
		case "UserName":
			current.User = decodeRegString(value)
		case "PortNumber":
			if p := parseRegDword(value); p > 0 {
				current.Port = p
			}
		case "Protocol":
			current.Protocol = strings.ToLower(strings.TrimSpace(decodeRegString(value)))
		}
	}
	flush()
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func decodePuttySessionName(raw string) string {
	if decoded, err := url.PathUnescape(raw); err == nil && decoded != "" {
		return decoded
	}
	if decoded, err := url.QueryUnescape(raw); err == nil && decoded != "" {
		return decoded
	}
	return raw
}

// normalizePuttyHostUser 处理 HostName=user@host：拆出用户名，Host 只保留主机。
// 若已有 UserName，以 UserName 为准，仍去掉 HostName 里的 user@ 前缀。
func normalizePuttyHostUser(s *ParsedPuttySession) {
	if s == nil {
		return
	}
	host := strings.TrimSpace(s.Host)
	at := strings.LastIndex(host, "@")
	if at <= 0 || at >= len(host)-1 {
		return
	}
	userPart := host[:at]
	hostPart := host[at+1:]
	// 避免把含冒号的异常串误判成 user@host（如残缺 IPv6）
	if strings.Contains(userPart, ":") {
		return
	}
	if strings.TrimSpace(s.User) == "" {
		s.User = userPart
	}
	s.Host = hostPart
}

func decodeRegString(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) >= 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
		inner := trimmed[1 : len(trimmed)-1]
		inner = strings.ReplaceAll(inner, `\\`, `\`)
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		return inner
	}
	return trimmed
}

func parseRegDword(raw string) int {
	trimmed := strings.TrimSpace(raw)
	const prefix = "dword:"
	if !strings.HasPrefix(strings.ToLower(trimmed), prefix) {
		return 0
	}
	hex := trimmed[len(prefix):]
	n, err := strconv.ParseInt(hex, 16, 64)
	if err != nil || n < 1 || n > 65535 {
		return 0
	}
	return int(n)
}

func isSSHProtocol(proto string) bool {
	p := strings.ToLower(strings.TrimSpace(proto))
	return p == "" || p == "ssh" || p == "ssh2" || p == "ssh-2"
}

func isPuttyRegFile(name string) bool {
	return strings.HasSuffix(name, ".reg")
}

// ImportPuttyFiles 批量导入 PuTTY 会话
func (gcm *GlobalConfigManager) ImportPuttyFiles(paths []string, accountID, group string) (*MachineImportResult, error) {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return nil, err
		}
	}

	files, err := CollectFilesFromPaths(paths, isPuttyRegFile)
	if err != nil {
		return nil, err
	}

	group = strings.TrimSpace(group)
	result := &MachineImportResult{}
	accountUser, accountPassword := "", defaultPuttyPassword
	if accountID != "" {
		user, password, err := gcm.GetGlobalAccountCredentials(accountID)
		if err != nil {
			return nil, err
		}
		accountUser, accountPassword = user, password
	}

	for _, path := range files {
		sessions, err := ParsePuttyRegFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filepath.Base(path), err))
			result.Skipped++
			continue
		}
		if len(sessions) == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: 未找到可导入的 PuTTY 会话", filepath.Base(path)))
			result.Skipped++
			continue
		}
		for _, session := range sessions {
			if !isSSHProtocol(session.Protocol) {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: 不支持的协议 %s", session.Name, session.Protocol))
				result.Skipped++
				continue
			}
			user := session.User
			password := defaultPuttyPassword
			if accountUser != "" {
				user = accountUser
				password = accountPassword
			}
			name := strings.TrimSpace(session.Name)
			if name == "" {
				name = session.Host
			}
			port := session.Port
			if port <= 0 {
				port = 22
			}
			if err := gcm.upsertImportedMachine(name, group, session.Host, port, user, password, result); err != nil {
				continue
			}
			result.Imported++
		}
	}

	return result, nil
}

func (gcm *GlobalConfigManager) upsertImportedMachine(name, group, host string, port int, user, password string, result *MachineImportResult) error {
	machine := gcm.findMachineByName(name)
	if machine == nil {
		machine = &define.Machine{
			ID:    uuid.NewString(),
			Name:  name,
			Group: group,
		}
	} else {
		machine.EnsureID()
		if group != "" {
			machine.Group = group
		}
	}
	sensitive := &define.SensitiveData{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}
	if err := machine.SetSensitiveData(sensitive); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: 加密失败: %v", name, err))
		result.Skipped++
		return err
	}
	gcm.EnsureMachineGroupRegistered(machine.Group)
	if err := gcm.upsertMachine(machine); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", name, err))
		result.Skipped++
		return err
	}
	return nil
}
