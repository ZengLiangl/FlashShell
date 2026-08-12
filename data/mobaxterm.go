package data

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

const defaultMobaXtermPassword = "123456"

var (
	mobaBookmarkSection = regexp.MustCompile(`(?i)^bookmarks(?:_\d+)?$`)
	mobaCandidateSection = regexp.MustCompile(`(?i)^(?:sessions|bookmarks(?:_\d+)?|bookmarks2|bookmark)$`)
	mobaStandardSession = regexp.MustCompile(`(?i)^(?:;\s*logout)?\s*#\d+(?:#|$)`)
)

// ParsedMobaXtermSession 解析后的 MobaXterm 会话
type ParsedMobaXtermSession struct {
	Name string
	Host string
	Port int
	User string
	Group string
}

// ParseMobaXtermFile 解析 MobaXterm 会话导出文件
func ParseMobaXtermFile(path string) ([]ParsedMobaXtermSession, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	text := decodeMobaXtermBytes(data)
	return ParseMobaXtermContent(text)
}

func decodeMobaXtermBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if utf8.Valid(data) {
		return string(data)
	}
	r := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GB18030.NewDecoder())
	out, err := io.ReadAll(r)
	if err != nil {
		return string(data)
	}
	return string(out)
}

// ParseMobaXtermContent 解析 MobaXterm 文本内容
func ParseMobaXtermContent(text string) ([]ParsedMobaXtermSession, []string, error) {
	type entry struct {
		section string
		key     string
		value   string
	}
	var entries []entry
	sectionGroups := map[string]string{}
	section := ""

	scanner := bufio.NewScanner(strings.NewReader(text))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section = trimmed[1 : len(trimmed)-1]
			continue
		}
		eq := strings.Index(trimmed, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:eq])
		value := strings.TrimSpace(trimmed[eq+1:])
		secTrim := strings.TrimSpace(section)
		if mobaBookmarkSection.MatchString(secTrim) && strings.EqualFold(key, "SubRep") {
			sectionGroups[section] = normalizeImportGroupPath(value)
			continue
		}
		if mobaBookmarkSection.MatchString(secTrim) && strings.EqualFold(key, "ImgNum") {
			continue
		}
		entries = append(entries, entry{section: section, key: key, value: value})
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	var sessions []ParsedMobaXtermSession
	var warnings []string

	for _, e := range entries {
		if !mobaCandidateSection.MatchString(strings.TrimSpace(e.section)) {
			continue
		}
		if e.key == "" || e.value == "" {
			continue
		}

		isBookmark := mobaBookmarkSection.MatchString(strings.TrimSpace(e.section))
		_, hasBookmarkGroup := sectionGroups[e.section]
		keyParts := splitImportPath(e.key)
		label := e.key
		group := ""
		if isBookmark && hasBookmarkGroup {
			label = e.key
			group = sectionGroups[e.section]
		} else if len(keyParts) > 1 {
			label = keyParts[len(keyParts)-1]
			group = strings.Join(keyParts[:len(keyParts)-1], "/")
		} else if len(keyParts) == 1 {
			label = keyParts[0]
		}

		host, user, port, ok, warn := parseMobaXtermSessionValue(label, e.value)
		if warn != "" {
			warnings = append(warnings, warn)
		}
		if !ok {
			continue
		}
		sessions = append(sessions, ParsedMobaXtermSession{
			Name:  label,
			Host:  host,
			Port:  port,
			User:  user,
			Group: group,
		})
	}

	return sessions, warnings, nil
}

func parseMobaXtermSessionValue(label, rawValue string) (host, user string, port int, ok bool, warn string) {
	if mobaStandardSession.MatchString(rawValue) {
		outerFields := strings.Split(rawValue, "#")
		sessionFields := []string{}
		if len(outerFields) >= 3 {
			sessionFields = strings.Split(outerFields[2], "%")
		}
		sessionType := ""
		if len(sessionFields) > 0 {
			sessionType = strings.TrimSpace(sessionFields[0])
		}
		if sessionType == "" || !isAllDigits(sessionType) {
			return "", "", 0, false, fmt.Sprintf("%s: 无效的会话类型", label)
		}
		// 0 = SSH, 7 = SSH (部分版本)
		if sessionType != "0" && sessionType != "7" {
			return "", "", 0, false, fmt.Sprintf("%s: 不支持的会话类型 %s", label, sessionType)
		}
		if len(sessionFields) > 1 {
			host = strings.TrimSpace(sessionFields[1])
		}
		if len(sessionFields) > 2 {
			port = parseImportPort(sessionFields[2])
		}
		if len(sessionFields) > 3 {
			rawUser := strings.TrimSpace(sessionFields[3])
			if rawUser != "" && !strings.EqualFold(rawUser, "<default>") {
				user = rawUser
			}
		}
		if host == "" {
			return "", "", 0, false, fmt.Sprintf("%s: 缺少主机名", label)
		}
		if port <= 0 {
			port = 22
		}
		return host, user, port, true, ""
	}

	// 兼容旧版简单 token 格式：deploy@host:2222#ssh
	tokens := []string{}
	for _, t := range strings.Split(rawValue, "#") {
		t = strings.TrimSpace(t)
		if t != "" {
			tokens = append(tokens, t)
		}
	}
	if len(tokens) == 0 {
		return "", "", 0, false, fmt.Sprintf("%s: 缺少主机名", label)
	}

	for _, tok := range tokens {
		clean := strings.TrimSpace(strings.TrimPrefix(tok, "ssh:"))
		clean = strings.TrimSpace(strings.TrimPrefix(clean, "SSH:"))
		h, u, p, found := parseImportTarget(clean)
		if found {
			host, user, port = h, u, p
			break
		}
	}
	if host == "" {
		return "", "", 0, false, fmt.Sprintf("%s: 缺少主机名", label)
	}
	if port <= 0 {
		for _, t := range tokens {
			if p := parseImportPort(t); p > 0 {
				port = p
				break
			}
		}
	}
	if port <= 0 {
		port = 22
	}
	if user == "" {
		for _, t := range tokens {
			if strings.Contains(t, "@") {
				user = strings.SplitN(t, "@", 2)[0]
				break
			}
		}
	}
	return host, user, port, true, ""
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func normalizeImportGroupPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parts := splitImportPath(trimmed)
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "/")
}

func splitImportPath(raw string) []string {
	normalized := strings.ReplaceAll(raw, `\`, "/")
	parts := strings.Split(normalized, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseImportPort(raw string) int {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 65535 {
		return 0
	}
	return n
}

// parseImportTarget 解析 user@host:port / host:port / host
func parseImportTarget(raw string) (host, user string, port int, ok bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", 0, false
	}
	if strings.Contains(trimmed, "://") {
		// ssh://user@host:22
		rest := trimmed
		if idx := strings.Index(rest, "://"); idx >= 0 {
			rest = rest[idx+3:]
		}
		return parseImportTarget(rest)
	}
	if strings.Contains(trimmed, "@") {
		parts := strings.SplitN(trimmed, "@", 2)
		user = parts[0]
		trimmed = parts[1]
	}
	// IPv6 in brackets [::1]:22
	if strings.HasPrefix(trimmed, "[") {
		end := strings.Index(trimmed, "]")
		if end > 0 {
			host = trimmed[1:end]
			rest := trimmed[end+1:]
			if strings.HasPrefix(rest, ":") {
				port = parseImportPort(rest[1:])
			}
			if host != "" {
				return host, user, port, true
			}
		}
	}
	// host:port — 仅当末段为端口数字时拆分
	if i := strings.LastIndex(trimmed, ":"); i > 0 {
		maybePort := trimmed[i+1:]
		if p := parseImportPort(maybePort); p > 0 && !strings.Contains(maybePort, ":") {
			host = trimmed[:i]
			port = p
			if host != "" {
				return host, user, port, true
			}
		}
	}
	if looksLikeHostname(trimmed) {
		return trimmed, user, 0, true
	}
	return "", "", 0, false
}

func looksLikeHostname(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, " \t") {
		return false
	}
	lower := strings.ToLower(s)
	if lower == "ssh" || lower == "ssh2" || lower == "telnet" || lower == "local" {
		return false
	}
	return true
}

func isMobaXtermFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".mxtsessions") ||
		strings.HasSuffix(lower, ".ini") ||
		strings.HasSuffix(lower, ".mobaconf")
}

// ImportMobaXtermFiles 批量导入 MobaXterm 会话
func (gcm *GlobalConfigManager) ImportMobaXtermFiles(paths []string, accountID, group string) (*MachineImportResult, error) {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return nil, err
		}
	}

	files, err := CollectFilesFromPaths(paths, isMobaXtermFile)
	if err != nil {
		return nil, err
	}

	uiGroup := strings.TrimSpace(group)
	result := &MachineImportResult{}
	accountUser, accountPassword := "", defaultMobaXtermPassword
	if accountID != "" {
		user, password, err := gcm.GetGlobalAccountCredentials(accountID)
		if err != nil {
			return nil, err
		}
		accountUser, accountPassword = user, password
	}

	for _, path := range files {
		sessions, warnings, err := ParseMobaXtermFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", filepath.Base(path), err))
			result.Skipped++
			continue
		}
		for _, w := range warnings {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", filepath.Base(path), w))
			result.Skipped++
		}
		if len(sessions) == 0 {
			if len(warnings) == 0 {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: 未找到可导入的 MobaXterm 会话", filepath.Base(path)))
				result.Skipped++
			}
			continue
		}
		for _, session := range sessions {
			user := session.User
			password := defaultMobaXtermPassword
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
				machineGroup = session.Group
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
