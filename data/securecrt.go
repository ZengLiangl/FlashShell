package data

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const defaultSecureCRTPassword = "123456"

var secureCRTKV = regexp.MustCompile(`^[SDB]:"([^"]+)"=(.*)$`)

var secureCRTMetadataFiles = map[string]struct{}{
	"__folderdata__.ini": {},
	"default.ini":        {},
}

// ParsedSecureCRTSession 解析后的 SecureCRT 会话
type ParsedSecureCRTSession struct {
	Name       string
	Host       string
	Port       int
	User       string
	Protocol   string
	SSHVersion string // ssh1 | ssh2
	SSH1Port   int
	SSH2Port   int
}

// ParseSecureCRTFile 解析单个 SecureCRT .ini 会话文件
func ParseSecureCRTFile(path string) ([]ParsedSecureCRTSession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	base := filepath.Base(path)
	fallback := strings.TrimSuffix(base, filepath.Ext(base))
	return ParseSecureCRTContent(string(data), fallback)
}

// ParseSecureCRTContent 解析 SecureCRT 会话内容
func ParseSecureCRTContent(text, fallbackLabel string) ([]ParsedSecureCRTSession, error) {
	type session struct {
		label      string
		hostname   string
		username   string
		port       int
		ssh1Port   int
		ssh2Port   int
		sshVersion string
		protocol   string
	}

	var sessions []session
	current := session{}

	flush := func() {
		if strings.TrimSpace(current.hostname) != "" {
			sessions = append(sessions, current)
		}
		current = session{}
	}

	scanner := bufio.NewScanner(strings.NewReader(text))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" {
			continue
		}
		m := secureCRTKV.FindStringSubmatch(trimmed)
		if len(m) != 3 {
			continue
		}
		key := m[1]
		rawValue := strings.TrimSpace(m[2])
		value := strings.Trim(rawValue, `"`)

		switch key {
		case "Hostname":
			if current.hostname != "" {
				flush()
			}
			current.hostname = value
		case "Username":
			current.username = value
		case "Port":
			current.port = parseSecureCRTPort(value)
		case "[SSH2] Port":
			current.ssh2Port = parseSecureCRTPort(value)
		case "[SSH1] Port":
			current.ssh1Port = parseSecureCRTPort(value)
		case "Protocol Name":
			normalized := strings.ToLower(strings.TrimSpace(value))
			switch normalized {
			case "ssh1":
				current.sshVersion = "ssh1"
				current.protocol = "ssh"
			case "ssh2", "ssh-2", "ssh":
				current.sshVersion = "ssh2"
				current.protocol = "ssh"
			case "telnet":
				current.protocol = "telnet"
			default:
				if p := normalizeImportProtocol(value); p != "" {
					current.protocol = p
				}
			}
		case "Session Name":
			current.label = value
		}
	}
	flush()
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if fallbackLabel == "" {
		fallbackLabel = "SecureCRT Session"
	}

	out := make([]ParsedSecureCRTSession, 0, len(sessions))
	for i, s := range sessions {
		proto := s.protocol
		if proto == "" {
			proto = "ssh"
		}
		label := s.label
		if label == "" {
			if len(sessions) > 1 {
				label = fmt.Sprintf("%s %d", fallbackLabel, i+1)
			} else {
				label = fallbackLabel
			}
		}
		port := s.port
		if proto == "ssh" {
			switch s.sshVersion {
			case "ssh1":
				if s.ssh1Port > 0 {
					port = s.ssh1Port
				}
			case "ssh2":
				if s.ssh2Port > 0 {
					port = s.ssh2Port
				}
			default:
				if s.ssh2Port > 0 {
					port = s.ssh2Port
				} else if s.ssh1Port > 0 {
					port = s.ssh1Port
				}
			}
		}
		if port <= 0 {
			port = 22
		}
		out = append(out, ParsedSecureCRTSession{
			Name:       label,
			Host:       s.hostname,
			Port:       port,
			User:       s.username,
			Protocol:   proto,
			SSHVersion: s.sshVersion,
			SSH1Port:   s.ssh1Port,
			SSH2Port:   s.ssh2Port,
		})
	}
	return out, nil
}

func parseSecureCRTPort(raw string) int {
	trimmed := strings.TrimSpace(strings.Trim(raw, `"`))
	if trimmed == "" {
		return 0
	}
	if len(trimmed) == 8 {
		allHex := true
		for _, r := range trimmed {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				allHex = false
				break
			}
		}
		if allHex {
			n, err := strconv.ParseInt(trimmed, 16, 64)
			if err == nil && n >= 1 && n <= 65535 {
				return int(n)
			}
		}
	}
	return parseImportPort(trimmed)
}

func normalizeImportProtocol(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "ssh", "ssh2", "ssh-2", "ssh1":
		return "ssh"
	case "telnet":
		return "telnet"
	default:
		return ""
	}
}

func isSecureCRTFile(name string) bool {
	lower := strings.ToLower(name)
	if _, skip := secureCRTMetadataFiles[lower]; skip {
		return false
	}
	return strings.HasSuffix(lower, ".ini")
}

// CollectSecureCRTFiles 收集 SecureCRT 会话文件（目录递归）
func CollectSecureCRTFiles(paths []string) ([]string, error) {
	return CollectFilesRecursive(paths, isSecureCRTFile)
}

func secureCrtGroupFromRel(rel string) string {
	parts := splitImportPath(rel)
	if len(parts) <= 1 {
		return ""
	}
	parts = parts[:len(parts)-1]
	if len(parts) > 0 && strings.EqualFold(parts[0], "Sessions") {
		parts = parts[1:]
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "/")
}

// ImportSecureCRTFiles 批量导入 SecureCRT 会话
func (gcm *GlobalConfigManager) ImportSecureCRTFiles(paths []string, accountID, group string) (*MachineImportResult, error) {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return nil, err
		}
	}

	files, err := CollectSecureCRTFiles(paths)
	if err != nil {
		return nil, err
	}

	uiGroup := strings.TrimSpace(group)
	result := &MachineImportResult{}
	accountUser, accountPassword := "", defaultSecureCRTPassword
	if accountID != "" {
		user, password, err := gcm.GetGlobalAccountCredentials(accountID)
		if err != nil {
			return nil, err
		}
		accountUser, accountPassword = user, password
	}

	// 计算相对路径用的根：若只选一个目录则以其为根
	var rootDir string
	if len(paths) == 1 {
		if info, err := os.Stat(paths[0]); err == nil && info.IsDir() {
			rootDir = paths[0]
		}
	}

	for _, path := range files {
		sessions, err := ParseSecureCRTFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filepath.Base(path), err))
			result.Skipped++
			continue
		}
		if len(sessions) == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: 未找到可导入的 SecureCRT 会话", filepath.Base(path)))
			result.Skipped++
			continue
		}

		fileGroup := ""
		if rootDir != "" {
			if rel, err := filepath.Rel(rootDir, path); err == nil {
				fileGroup = secureCrtGroupFromRel(rel)
			}
		}

		for _, session := range sessions {
			if !isSSHProtocol(session.Protocol) {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: 不支持的协议 %s", session.Name, session.Protocol))
				result.Skipped++
				continue
			}
			user := session.User
			password := defaultSecureCRTPassword
			if accountUser != "" {
				user = accountUser
				password = accountPassword
			}
			name := strings.TrimSpace(session.Name)
			if name == "" {
				name = session.Host
			}
			machineGroup := uiGroup
			if machineGroup == "" {
				machineGroup = fileGroup
			}
			port := session.Port
			if port <= 0 {
				port = 22
			}
			if err := gcm.upsertImportedMachine(name, machineGroup, session.Host, port, user, password, result); err != nil {
				continue
			}
			result.Imported++
		}
	}

	return result, nil
}
