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
