package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const defaultShellCmdHistoryMax = 200

// ShellCommandRecord 单条 Shell 命令历史
type ShellCommandRecord struct {
	Command   string `json:"command"`
	Scope     string `json:"scope"` // global 或机器配置名
	Timestamp int64  `json:"timestamp"`
}

type shellCmdHistoryFile struct {
	Global  []string            `json:"global"`
	ByScope map[string][]string `json:"byScope"`
}

// ShellCommandHistoryManager 跨会话命令历史
type ShellCommandHistoryManager struct {
	mu          sync.Mutex
	data        shellCmdHistoryFile
	maxPerScope int
}

// NewShellCommandHistoryManager 创建并加载
func NewShellCommandHistoryManager() *ShellCommandHistoryManager {
	m := &ShellCommandHistoryManager{
		data:        shellCmdHistoryFile{ByScope: make(map[string][]string)},
		maxPerScope: defaultShellCmdHistoryMax,
	}
	_ = m.Load()
	return m
}

// SetMaxPerScope 设置每个作用域的历史条数上限
func (m *ShellCommandHistoryManager) SetMaxPerScope(max int) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxPerScope = NormalizeShellCommandHistoryMax(max)
}

// NormalizeShellCommandHistoryMax 校验命令历史上限
func NormalizeShellCommandHistoryMax(max int) int {
	if max <= 0 {
		return defaultShellCmdHistoryMax
	}
	if max < 50 {
		return 50
	}
	if max > 20000 {
		return 20000
	}
	return max
}

const defaultShellTerminalScrollback = 500
const defaultTaskOutputMaxLines = 2000

// NormalizeShellTerminalScrollback 校验 xterm 滚动缓冲行数
func NormalizeShellTerminalScrollback(n int) int {
	if n <= 0 {
		return defaultShellTerminalScrollback
	}
	if n < 100 {
		return 100
	}
	if n > 100000 {
		return 100000
	}
	return n
}

// NormalizeTaskOutputMaxLines 校验任务输出行数上限
func NormalizeTaskOutputMaxLines(n int) int {
	if n <= 0 {
		return defaultTaskOutputMaxLines
	}
	if n < 100 {
		return 100
	}
	if n > 100000 {
		return 100000
	}
	return n
}

func shellCmdHistoryPath() (string, error) {
	home, err := ConfigHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "shell_command_history.json"), nil
}

// Load 从磁盘加载
func (m *ShellCommandHistoryManager) Load() error {
	path, err := shellCmdHistoryPath()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return json.Unmarshal(raw, &m.data)
}

func (m *ShellCommandHistoryManager) saveLocked() error {
	path, err := shellCmdHistoryPath()
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m.data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}

func trimHistory(list []string, max int) []string {
	if len(list) <= max {
		return list
	}
	return list[len(list)-max:]
}

func normalizeCmd(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	cmd = strings.TrimSuffix(cmd, "\n")
	cmd = strings.TrimSuffix(cmd, "\r")
	return cmd
}

// Record 记录命令（去重：连续相同不重复写入）
func (m *ShellCommandHistoryManager) Record(scope, command string) error {
	command = normalizeCmd(command)
	if command == "" || len(command) > 4096 {
		return nil
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = "global"
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	appendUnique := func(list []string, cmd string) []string {
		if len(list) > 0 && list[len(list)-1] == cmd {
			return list
		}
		// 若已存在则移到末尾
		for i, c := range list {
			if c == cmd {
				list = append(list[:i], list[i+1:]...)
				break
			}
		}
		list = append(list, cmd)
		return trimHistory(list, m.maxPerScope)
	}

	m.data.Global = appendUnique(m.data.Global, command)
	if scope != "global" {
		m.data.ByScope[scope] = appendUnique(m.data.ByScope[scope], command)
	}
	return m.saveLocked()
}

// Search 搜索历史（scope: global / 机器名 / all）
func (m *ShellCommandHistoryManager) Search(scope, query string, limit int) []string {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query = strings.ToLower(strings.TrimSpace(query))
	m.mu.Lock()
	defer m.mu.Unlock()

	collect := func(list []string) {
		// reversed for newest first - handled in merge
	}
	_ = collect

	seen := make(map[string]struct{})
	var out []string
	add := func(cmd string) {
		if _, ok := seen[cmd]; ok {
			return
		}
		if query != "" && !strings.Contains(strings.ToLower(cmd), query) {
			return
		}
		seen[cmd] = struct{}{}
		out = append(out, cmd)
	}

	appendList := func(list []string) {
		for i := len(list) - 1; i >= 0; i-- {
			add(list[i])
			if len(out) >= limit {
				return
			}
		}
	}

	scope = strings.TrimSpace(scope)
	switch scope {
	case "", "all":
		appendList(m.data.Global)
		for _, list := range m.data.ByScope {
			appendList(list)
			if len(out) >= limit {
				break
			}
		}
	case "global":
		appendList(m.data.Global)
	default:
		appendList(m.data.ByScope[scope])
		appendList(m.data.Global)
	}
	return out
}

// Clear 清空历史
func (m *ShellCommandHistoryManager) Clear(scope string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	scope = strings.TrimSpace(scope)
	if scope == "" || scope == "all" {
		m.data.Global = nil
		m.data.ByScope = make(map[string][]string)
	} else if scope == "global" {
		m.data.Global = nil
	} else {
		delete(m.data.ByScope, scope)
	}
	return m.saveLocked()
}
