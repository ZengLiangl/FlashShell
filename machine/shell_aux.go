package machine

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"FlashDock/define"

	"github.com/pkg/sftp"
)

// ShellAuxManager 辅助 SSH（监控 + SFTP），与 PTY 分离
type ShellAuxManager struct {
	mu          sync.Mutex
	client      *SSHClient
	machineName string
	host        string
	uidNames    map[uint32]string
	gidNames    map[uint32]string
	idMapsReady bool
}

// NewShellAuxManager 创建辅助连接管理器
func NewShellAuxManager() *ShellAuxManager {
	return &ShellAuxManager{}
}

// Connect 建立带 SFTP 的 SSH 连接
func (a *ShellAuxManager) Connect(machine *define.Machine, workVars map[string]string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client != nil && a.client.IsConnected() {
		_ = a.client.Close()
		a.client = nil
	}

	client := NewSSHClient(machine, workVars)
	if err := client.Connect(machine, true); err != nil {
		return err
	}
	sensitive, err := machine.GetSensitiveData()
	if err != nil {
		_ = client.Close()
		return err
	}
	a.client = client
	a.machineName = machine.Name
	a.host = sensitive.Host
	a.uidNames = nil
	a.gidNames = nil
	a.idMapsReady = false
	return nil
}

// Close 关闭辅助连接
func (a *ShellAuxManager) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil {
		return nil
	}
	err := a.client.Close()
	a.client = nil
	a.uidNames = nil
	a.gidNames = nil
	a.idMapsReady = false
	return err
}

// IsConnected 是否已连接
func (a *ShellAuxManager) IsConnected() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.client != nil && a.client.IsConnected()
}

// Exec 在辅助连接上执行命令并返回 stdout+stderr 文本
func (a *ShellAuxManager) Exec(command string) (string, error) {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil || !client.IsConnected() {
		return "", fmt.Errorf("辅助连接未建立")
	}
	session, err := client.remoteMachine.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	out, err := session.CombinedOutput(command)
	return string(out), err
}

// FetchMonitor 拉取监控快照
func (a *ShellAuxManager) FetchMonitor() *define.ShellMonitorSnapshot {
	snap := &define.ShellMonitorSnapshot{
		MachineName: a.machineName,
		Host:        a.host,
		UpdatedAt:   time.Now().Unix(),
		TopMem:      []define.ShellProcessStat{},
	}
	if !a.IsConnected() {
		snap.Error = "辅助连接未建立"
		return snap
	}

	script := strings.Join([]string{
		`echo __UP__; cat /proc/uptime 2>/dev/null | awk '{print $1}'`,
		`echo __CPU__; top -bn1 2>/dev/null | grep -E 'Cpu\(s\)|%Cpu' | head -1`,
		`echo __MEM__; free -b 2>/dev/null | awk '/Mem:/{print $2,$3}'`,
		`echo __TOP__; ps -eo pid,user,pcpu,pmem,comm --sort=-pcpu 2>/dev/null | head -n 5`,
	}, "; ")

	out, err := a.Exec(script)
	if err != nil && strings.TrimSpace(out) == "" {
		snap.Error = err.Error()
		return snap
	}

	sections := parseTaggedSections(out, []string{"__UP__", "__CPU__", "__MEM__", "__TOP__"})
	if v := strings.TrimSpace(sections["__UP__"]); v != "" {
		if sec, err := strconv.ParseFloat(v, 64); err == nil {
			snap.UptimeSec = sec
			snap.UptimeText = formatUptime(sec)
		}
	}
	snap.CPUPercent = parseCPUPercent(sections["__CPU__"])
	if memLine := strings.TrimSpace(sections["__MEM__"]); memLine != "" {
		parts := strings.Fields(memLine)
		if len(parts) >= 2 {
			total, _ := strconv.ParseFloat(parts[0], 64)
			used, _ := strconv.ParseFloat(parts[1], 64)
			if total > 0 {
				snap.MemPercent = used / total * 100
				snap.MemTotal = formatBytes(total)
				snap.MemUsed = formatBytes(used)
			}
		}
	}
	snap.TopMem = parseTopProcesses(sections["__TOP__"], 4)
	return snap
}

// Home 远端登录用户 HOME
func (a *ShellAuxManager) Home() (string, error) {
	out, err := a.Exec("printf '%s\\n' \"$HOME\"")
	if err == nil {
		home := strings.TrimSpace(out)
		if idx := strings.IndexByte(home, '\n'); idx >= 0 {
			home = home[:idx]
		}
		home = strings.TrimSpace(home)
		if home != "" {
			return home, nil
		}
	}
	// fallback: SFTP 起始目录通常即 home
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client != nil && client.remoteMachine != nil && client.remoteMachine.SFTPClient != nil {
		if wd, e := client.remoteMachine.SFTPClient.Getwd(); e == nil && strings.TrimSpace(wd) != "" {
			return strings.TrimSpace(wd), nil
		}
	}
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("HOME 为空")
}

// DirExists 判断远端路径是否为可列出的目录（以 ReadDir 为准，兼容 symlink）
func (a *ShellAuxManager) DirExists(remotePath string) (bool, error) {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil || client.remoteMachine == nil || client.remoteMachine.SFTPClient == nil {
		return false, fmt.Errorf("SFTP 未连接")
	}
	if remotePath == "" {
		remotePath = "."
	}
	sftpClient := client.remoteMachine.SFTPClient

	info, err := sftpClient.Stat(remotePath)
	if err == nil {
		if info.IsDir() {
			return true, nil
		}
		// 非目录
		return false, nil
	}
	errText := strings.ToLower(err.Error())
	if os.IsNotExist(err) || strings.Contains(errText, "no such file") || strings.Contains(errText, "not found") {
		return false, nil
	}

	// Stat 异常时再尝试 ReadDir（部分服务器对目录 Stat 不稳定）
	if _, rdErr := sftpClient.ReadDir(remotePath); rdErr == nil {
		return true, nil
	} else {
		rdText := strings.ToLower(rdErr.Error())
		if os.IsNotExist(rdErr) || strings.Contains(rdText, "no such file") || strings.Contains(rdText, "not found") {
			return false, nil
		}
		return false, err
	}
}

// Pwd 当前家目录或工作目录（辅助通道无法知 PTY cwd，返回 Getwd）
func (a *ShellAuxManager) Pwd() (string, error) {
	out, err := a.Exec("pwd")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// ListDir 列出目录
func (a *ShellAuxManager) ListDir(dirPath string, showHidden bool) ([]define.SftpEntry, error) {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil || client.remoteMachine == nil || client.remoteMachine.SFTPClient == nil {
		return nil, fmt.Errorf("SFTP 未连接")
	}
	if dirPath == "" {
		dirPath = "."
	}
	infos, err := client.remoteMachine.SFTPClient.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	entries := make([]define.SftpEntry, 0, len(infos))
	for _, info := range infos {
		name := info.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		full := path.Join(dirPath, name)
		mode := info.Mode()
		entry := define.SftpEntry{
			Name:    name,
			Path:    full,
			IsDir:   info.IsDir(),
			Size:    info.Size(),
			Mode:    mode.String(),
			ModTime: info.ModTime().Unix(),
			Type:    fileTypeLabel(mode),
		}
		if stat, ok := info.Sys().(*sftp.FileStat); ok && stat != nil {
			entry.Owner = a.resolveUID(stat.UID)
			entry.Group = a.resolveGID(stat.GID)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (a *ShellAuxManager) ensureIDNameMaps() {
	a.mu.Lock()
	ready := a.idMapsReady
	a.mu.Unlock()
	if ready {
		return
	}

	uidNames := parsePasswdOrGroupMap(a.fetchNameMapText(true))
	gidNames := parsePasswdOrGroupMap(a.fetchNameMapText(false))

	a.mu.Lock()
	a.uidNames = uidNames
	a.gidNames = gidNames
	a.idMapsReady = true
	a.mu.Unlock()
}

func (a *ShellAuxManager) fetchNameMapText(passwd bool) string {
	var cmds []string
	if passwd {
		cmds = []string{"getent passwd 2>/dev/null", "cat /etc/passwd 2>/dev/null"}
	} else {
		cmds = []string{"getent group 2>/dev/null", "cat /etc/group 2>/dev/null"}
	}
	for _, cmd := range cmds {
		out, err := a.Exec(cmd)
		if err == nil && strings.TrimSpace(out) != "" {
			return out
		}
	}
	return ""
}

// parsePasswdOrGroupMap 解析 passwd/group 文本：name:x:id:...
func parsePasswdOrGroupMap(text string) map[uint32]string {
	result := make(map[uint32]string)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		name := parts[0]
		id64, err := strconv.ParseUint(parts[2], 10, 32)
		if err != nil || name == "" {
			continue
		}
		result[uint32(id64)] = name
	}
	return result
}

func (a *ShellAuxManager) resolveUID(uid uint32) string {
	a.ensureIDNameMaps()
	a.mu.Lock()
	defer a.mu.Unlock()
	if name, ok := a.uidNames[uid]; ok && name != "" {
		return name
	}
	return strconv.FormatUint(uint64(uid), 10)
}

func (a *ShellAuxManager) resolveGID(gid uint32) string {
	a.ensureIDNameMaps()
	a.mu.Lock()
	defer a.mu.Unlock()
	if name, ok := a.gidNames[gid]; ok && name != "" {
		return name
	}
	return strconv.FormatUint(uint64(gid), 10)
}

// RemovePath 删除远端文件或空目录；目录非空则递归删除
func (a *ShellAuxManager) RemovePath(remotePath string) error {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil || client.remoteMachine == nil || client.remoteMachine.SFTPClient == nil {
		return fmt.Errorf("SFTP 未连接")
	}
	sftpClient := client.remoteMachine.SFTPClient
	info, err := sftpClient.Stat(remotePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return removeDirRecursive(sftpClient, remotePath)
	}
	return sftpClient.Remove(remotePath)
}

func removeDirRecursive(c *sftp.Client, dir string) error {
	infos, err := c.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, info := range infos {
		p := path.Join(dir, info.Name())
		if info.IsDir() {
			if err := removeDirRecursive(c, p); err != nil {
				return err
			}
		} else if err := c.Remove(p); err != nil {
			return err
		}
	}
	return c.Remove(dir)
}

func fileTypeLabel(mode os.FileMode) string {
	switch mode.Type() {
	case os.ModeDir:
		return "目录"
	case os.ModeSymlink:
		return "链接"
	case os.ModeNamedPipe:
		return "管道"
	case os.ModeSocket:
		return "套接字"
	case os.ModeDevice:
		return "设备"
	default:
		return "文件"
	}
}

func parseTaggedSections(out string, tags []string) map[string]string {
	result := map[string]string{}
	lines := strings.Split(out, "\n")
	var current string
	var buf strings.Builder
	flush := func() {
		if current != "" {
			result[current] = strings.TrimSpace(buf.String())
			buf.Reset()
		}
	}
	tagSet := map[string]bool{}
	for _, t := range tags {
		tagSet[t] = true
	}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if tagSet[trim] {
			flush()
			current = trim
			continue
		}
		if current != "" {
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.WriteString(line)
		}
	}
	flush()
	return result
}

func parseCPUPercent(line string) float64 {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0
	}
	// e.g. %Cpu(s):  3.2 us,  1.1 sy, ... idle
	if idx := strings.Index(strings.ToLower(line), "id"); idx >= 0 {
		// find number before "id"
		chunk := line
		lower := strings.ToLower(chunk)
		idIdx := strings.Index(lower, "id")
		before := chunk[:idIdx]
		fields := strings.FieldsFunc(before, func(r rune) bool {
			return r == ',' || r == ':' || r == ' '
		})
		if len(fields) > 0 {
			idle, err := strconv.ParseFloat(strings.TrimSuffix(fields[len(fields)-1], "%"), 64)
			if err == nil {
				v := 100 - idle
				if v < 0 {
					v = 0
				}
				return v
			}
		}
	}
	return 0
}

func parseTopProcesses(block string, limit int) []define.ShellProcessStat {
	if limit <= 0 {
		limit = 4
	}
	lines := strings.Split(block, "\n")
	out := []define.ShellProcessStat{}
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || i == 0 && strings.Contains(strings.ToLower(line), "pid") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		cmd := strings.Join(fields[4:], " ")
		out = append(out, define.ShellProcessStat{
			PID:     fields[0],
			User:    fields[1],
			CPU:     cpu,
			Mem:     mem,
			Command: cmd,
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

// 兼容旧名
func parseTopMem(block string) []define.ShellProcessStat {
	return parseTopProcesses(block, 4)
}

func formatUptime(sec float64) string {
	s := int64(sec)
	days := s / 86400
	s %= 86400
	hours := s / 3600
	s %= 3600
	mins := s / 60
	if days > 0 {
		return fmt.Sprintf("%d天 %02d:%02d", days, hours, mins)
	}
	return fmt.Sprintf("%02d:%02d", hours, mins)
}

func formatBytes(b float64) string {
	units := []string{"B", "K", "M", "G", "T"}
	v := b
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f%s", v, units[i])
	}
	return fmt.Sprintf("%.1f%s", v, units[i])
}
