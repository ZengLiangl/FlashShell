package define

// ShellStatus Shell 会话状态
type ShellStatus struct {
	Connected      bool   `json:"connected"`
	Connecting     bool   `json:"connecting"`
	MachineName    string `json:"machineName"` // 会话唯一 ID（远程可为 name / name#2）
	ConfigName     string `json:"configName"`  // 机器配置名（远程）；本地为空或同 MachineName
	TabLabel       string `json:"tabLabel"`    // Tab 显示名
	Host           string `json:"host"`
	User           string `json:"user"`
	IsRunning      bool   `json:"isRunning"`
	CurrentCommand string `json:"currentCommand"`
	Kind           string `json:"kind"` // remote | local
}

// ShellHistoryRecord 连接历史
type ShellHistoryRecord struct {
	MachineID       string `json:"machineId"`
	MachineName     string `json:"machineName"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	User            string `json:"user"`
	LastConnectedAt int64  `json:"lastConnectedAt"` // unix 秒
	ConnectCount    int    `json:"connectCount"`
}

// ShellProcessStat 进程占用
type ShellProcessStat struct {
	PID     string  `json:"pid"`
	User    string  `json:"user"`
	CPU     float64 `json:"cpu"`
	Mem     float64 `json:"mem"`
	MemRSS  string  `json:"memRss"` // top RES 列原始值，如 829.4M
	Command string  `json:"command"`
}

// ShellMonitorSnapshot 机器监控快照
type ShellMonitorSnapshot struct {
	MachineName string             `json:"machineName"`
	Host        string             `json:"host"`
	UptimeSec   float64            `json:"uptimeSec"`
	UptimeText  string             `json:"uptimeText"`
	CPUPercent  float64            `json:"cpuPercent"`
	MemPercent  float64            `json:"memPercent"`
	MemUsed     string             `json:"memUsed"`
	MemTotal    string             `json:"memTotal"`
	SwapPercent float64            `json:"swapPercent"`
	SwapUsed    string             `json:"swapUsed"`
	SwapTotal   string             `json:"swapTotal"`
	TopMem      []ShellProcessStat `json:"topMem"`
	NetIface    string             `json:"netIface"`
	NetIfaces   []string           `json:"netIfaces"`
	NetRxRate   float64            `json:"netRxRate"` // bytes/s
	NetTxRate   float64            `json:"netTxRate"` // bytes/s
	NetRxText   string             `json:"netRxText"`
	NetTxText   string             `json:"netTxText"`
	Error       string             `json:"error,omitempty"`
	UpdatedAt   int64              `json:"updatedAt"`
}

// ShellSystemInfo 系统信息
type ShellSystemInfo struct {
	MachineName string `json:"machineName"`
	Host        string `json:"host"`
	Hostname    string `json:"hostname"`
	OS          string `json:"os"`
	Kernel      string `json:"kernel"`
	Arch        string `json:"arch"`
	CPUModel    string `json:"cpuModel"`
	DiskSummary string `json:"diskSummary"`
	Error       string `json:"error,omitempty"`
}

// SftpEntry 远端文件/目录项
type SftpEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`    // 权限，如 -rw-r--r--
	ModTime int64  `json:"modTime"` // unix 秒
	Owner   string `json:"owner"`   // 用户名或 uid
	Group   string `json:"group"`   // 组名或 gid
	Type    string `json:"type"`    // 目录/文件/链接/...
	LinkTarget string `json:"linkTarget,omitempty"` // 符号链接目标
}

// LocalFileEntry 本地文件/目录项（与 SftpEntry 字段对齐）
type LocalFileEntry = SftpEntry
