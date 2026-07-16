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
	ownsClient  bool // false 时复用 PTY 的 SSH，Close 仅释放 SFTP
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

// Connect 建立辅助 SSH；SFTP 尽力初始化，失败不阻断监控/Exec。
func (a *ShellAuxManager) Connect(machine *define.Machine, workVars map[string]string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client != nil && a.client.IsConnected() {
		_ = a.client.Close()
		a.client = nil
	}

	client := NewSSHClient(machine, workVars)
	if err := client.Connect(machine, false); err != nil {
		return err
	}
	sensitive, err := machine.GetSensitiveData()
	if err != nil {
		_ = client.Close()
		return err
	}
	a.client = client
	a.ownsClient = true
	a.machineName = machine.Name
	a.host = sensitive.Host
	a.uidNames = nil
	a.gidNames = nil
	a.idMapsReady = false
	if rm := client.remoteMachine; rm != nil {
		_ = rm.EnsureSFTP()
	}
	return nil
}

// Attach 复用 PTY 已有 SSH 连接初始化 SFTP/Exec（避免第二路 TCP 触发 MaxSessions / packet too long）。
func (a *ShellAuxManager) Attach(client *SSHClient, machineName, host string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client != nil {
		_ = a.releaseLocked()
	}
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("共享 SSH 未连接")
	}
	a.client = client
	a.ownsClient = false
	a.machineName = machineName
	a.host = host
	a.uidNames = nil
	a.gidNames = nil
	a.idMapsReady = false
	if rm := client.remoteMachine; rm != nil {
		if err := rm.EnsureSFTP(); err != nil {
			a.client = nil
			return err
		}
	}
	return nil
}

// EnsureSFTP 按需初始化 SFTP（文件列表/传输需要）。
func (a *ShellAuxManager) EnsureSFTP() error {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("辅助连接未建立")
	}
	if client.remoteMachine == nil {
		return fmt.Errorf("远程未连接")
	}
	return client.remoteMachine.EnsureSFTP()
}

// Close 关闭辅助连接（复用 PTY 时仅关闭 SFTP，不关 SSH）
func (a *ShellAuxManager) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.releaseLocked()
}

func (a *ShellAuxManager) releaseLocked() error {
	if a.client == nil {
		return nil
	}
	var err error
	if a.ownsClient {
		err = a.client.Close()
	} else if a.client.remoteMachine != nil && a.client.remoteMachine.SFTPClient != nil {
		if closeErr := a.client.remoteMachine.SFTPClient.Close(); closeErr != nil {
			err = closeErr
		}
		a.client.remoteMachine.SFTPClient = nil
	}
	a.client = nil
	a.ownsClient = false
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

// ExecBash 通过 stdin 执行 bash 脚本（避免 bash -c + 转义把换行弄丢）
func (a *ShellAuxManager) ExecBash(script string) (string, error) {
	return a.execBashPath("bash", script)
}

func (a *ShellAuxManager) execBashPath(bash, script string) (string, error) {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()
	if client == nil || !client.IsConnected() {
		return "", fmt.Errorf("辅助连接未建立")
	}
	if bash == "" {
		bash = "bash"
	}
	session, err := client.remoteMachine.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	session.Stdin = strings.NewReader(script)
	out, err := session.CombinedOutput(bash + " -s")
	return string(out), err
}

// shellMonitorScript 远端监控：uptime/mem 直读；CPU 与 TOP 用 top -bn2 第二帧（瞬时占用）。
const shellMonitorScript = `set +e
echo __UP__
awk '{print $1; exit}' /proc/uptime 2>/dev/null
echo __MEM__
free -b 2>/dev/null | awk '/^Mem:/{print $2,$3; exit}'
echo __TOPRAW__
LC_ALL=C top -bn2 -d 0.5 -w 512 2>/dev/null
`

// FetchMonitor 拉取监控快照
func (a *ShellAuxManager) FetchMonitor() *define.ShellMonitorSnapshot {
	snap := &define.ShellMonitorSnapshot{
		MachineName: a.machineName,
		Host:        a.host,
		UpdatedAt:   time.Now().Unix(),
		TopMem:      []define.ShellProcessStat{},
	}
	if !a.IsConnected() {
		return snap
	}

	out, err := a.execBashPath("/bin/bash", shellMonitorScript)
	if err != nil && strings.TrimSpace(out) == "" {
		out, err = a.execBashPath("bash", shellMonitorScript)
	}
	if err != nil && strings.TrimSpace(out) == "" {
		snap.Error = err.Error()
		return snap
	}

	sections := parseTaggedSections(out, []string{"__UP__", "__MEM__", "__TOPRAW__"})
	if v := strings.TrimSpace(sections["__UP__"]); v != "" {
		if sec, err := strconv.ParseFloat(v, 64); err == nil {
			snap.UptimeSec = sec
			snap.UptimeText = formatUptime(sec)
		}
	}
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
	sysCPU, procs := parseTopBatch(sections["__TOPRAW__"], 5)
	snap.CPUPercent = sysCPU
	snap.TopMem = procs
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
	if err := a.EnsureSFTP(); err != nil {
		return false, err
	}
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

// PtyCwdFile 读取 PTY hook 写入的 sidecar 文件（~/.flashdock_pwd）。
func (a *ShellAuxManager) PtyCwdFile() (string, error) {
	home, err := a.Home()
	if err != nil || home == "" {
		home = "/tmp"
	}
	pwdFile := path.Join(home, shellCwdPwdFilename)
	out, err := a.Exec("cat " + strconv.Quote(pwdFile) + " 2>/dev/null || true")
	if err != nil {
		return "", err
	}
	if cwd, ok := SanitizePtyCwd(out); ok {
		return cwd, nil
	}
	return "", fmt.Errorf("PTY cwd 未知")
}

// ListDir 列出目录
func (a *ShellAuxManager) ListDir(dirPath string, showHidden bool) ([]define.SftpEntry, error) {
	if err := a.EnsureSFTP(); err != nil {
		return nil, err
	}
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
	if err := a.EnsureSFTP(); err != nil {
		return err
	}
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
	// 新格式：短窗口双采样得到的纯数字
	if first := strings.Fields(line); len(first) > 0 {
		if v, err := strconv.ParseFloat(first[0], 64); err == nil {
			if v < 0 {
				return 0
			}
			if v > 100 {
				return 100
			}
			return v
		}
	}
	// 兼容旧 top 行：%Cpu(s):  3.2 us, ... idle
	if idx := strings.Index(strings.ToLower(line), "id"); idx >= 0 {
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

// parseTopBatch 解析 top -bn2 输出：取最后一帧的整机 CPU 与进程 TOP。
func parseTopBatch(raw string, limit int) (float64, []define.ShellProcessStat) {
	if limit <= 0 {
		limit = 5
	}
	lines := strings.Split(raw, "\n")
	lastCPULine := ""
	headerIdx := -1
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		lower := strings.ToLower(trim)
		if strings.Contains(lower, "cpu(s)") || strings.HasPrefix(lower, "%cpu") {
			lastCPULine = trim
		}
		fields := strings.Fields(trim)
		upperJoined := strings.ToUpper(strings.Join(fields, " "))
		if strings.Contains(upperJoined, "PID") &&
			(strings.Contains(upperJoined, "%CPU") || strings.Contains(upperJoined, " CPU")) &&
			(strings.Contains(upperJoined, "COMMAND") || strings.Contains(upperJoined, "CMD")) {
			headerIdx = i
		}
	}

	sysCPU := parseCPUPercent(lastCPULine)
	if headerIdx < 0 {
		return sysCPU, []define.ShellProcessStat{}
	}

	headerFields := strings.Fields(strings.TrimSpace(lines[headerIdx]))
	idxPID, idxUser, idxCPU, idxMem, idxCmd := -1, -1, -1, -1, -1
	for i, h := range headerFields {
		switch strings.ToUpper(strings.TrimPrefix(h, "%")) {
		case "PID":
			idxPID = i
		case "USER":
			idxUser = i
		case "CPU":
			idxCPU = i
		case "MEM":
			idxMem = i
		case "COMMAND", "CMD":
			idxCmd = i
		}
	}
	// 再按原始表头匹配一次（含 %CPU / %MEM）
	for i, h := range headerFields {
		u := strings.ToUpper(h)
		switch u {
		case "%CPU":
			idxCPU = i
		case "%MEM":
			idxMem = i
		}
	}
	if idxPID < 0 || idxCPU < 0 || idxCmd < 0 {
		return sysCPU, []define.ShellProcessStat{}
	}

	out := make([]define.ShellProcessStat, 0, limit)
	for i := headerIdx + 1; i < len(lines) && len(out) < limit; i++ {
		trim := strings.TrimSpace(lines[i])
		if trim == "" {
			continue
		}
		lower := strings.ToLower(trim)
		if strings.HasPrefix(lower, "top -") || strings.HasPrefix(lower, "tasks:") ||
			strings.Contains(lower, "cpu(s)") || strings.HasPrefix(lower, "%cpu") {
			break
		}
		fields := strings.Fields(trim)
		if len(fields) <= idxCmd || len(fields) <= idxCPU || len(fields) <= idxPID {
			continue
		}
		if _, err := strconv.Atoi(fields[idxPID]); err != nil {
			continue
		}
		cpu, _ := strconv.ParseFloat(fields[idxCPU], 64)
		mem := 0.0
		if idxMem >= 0 && idxMem < len(fields) {
			mem, _ = strconv.ParseFloat(fields[idxMem], 64)
		}
		user := ""
		if idxUser >= 0 && idxUser < len(fields) {
			user = fields[idxUser]
		}
		out = append(out, define.ShellProcessStat{
			PID:     fields[idxPID],
			User:    user,
			CPU:     cpu,
			Mem:     mem,
			Command: strings.Join(fields[idxCmd:], " "),
		})
	}
	return sysCPU, out
}

func parseTopProcesses(block string, limit int) []define.ShellProcessStat {
	if limit <= 0 {
		limit = 5
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
	return parseTopProcesses(block, 5)
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
