package define

// ShellStatus Shell 会话状态
type ShellStatus struct {
	Connected      bool   `json:"connected"`
	MachineName    string `json:"machineName"`
	Host           string `json:"host"`
	User           string `json:"user"`
	IsRunning      bool   `json:"isRunning"`
	CurrentCommand string `json:"currentCommand"`
}

// ShellHistoryRecord 连接历史
type ShellHistoryRecord struct {
	MachineID      string `json:"machineId"`
	MachineName    string `json:"machineName"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	LastConnectedAt int64 `json:"lastConnectedAt"` // unix 秒
	ConnectCount   int    `json:"connectCount"`
}

// ShellProcessStat 进程占用
type ShellProcessStat struct {
	PID     string  `json:"pid"`
	User    string  `json:"user"`
	CPU     float64 `json:"cpu"`
	Mem     float64 `json:"mem"`
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
	TopMem      []ShellProcessStat `json:"topMem"`
	Error       string             `json:"error,omitempty"`
	UpdatedAt   int64              `json:"updatedAt"`
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
}
