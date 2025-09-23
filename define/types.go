package define

// Root 配置根结构
type Root struct {
	Projects []Project `yaml:"projects" json:"projects"`
	Machines []Machine `yaml:"machines" json:"machines"`
}

// Project 项目配置
type Project struct {
	Name        string       `yaml:"name" json:"name"`
	Description string       `yaml:"description" json:"description"`
	WorkDir     string       `yaml:"workdir" json:"workdir"`
	SubProjects []SubProject `yaml:"subprojects" json:"subprojects"`
}

// SubProject 子项目配置
type SubProject struct {
	Name        string    `yaml:"name" json:"name"`
	Description string    `yaml:"description" json:"description"`
	Commands    []Command `yaml:"commands" json:"commands"`
}

// Command 命令配置
type Command struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Type        string   `yaml:"type" json:"type"` // batch, remote
	Steps       []string `yaml:"steps" json:"steps"`
	Machine     string   `yaml:"machine,omitempty" json:"machine,omitempty"`
	WorkDir     string   `yaml:"workdir,omitempty" json:"workdir,omitempty"`
}

// Machine 远程机器配置
type Machine struct {
	Name     string `yaml:"name" json:"name"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
	KeyFile  string `yaml:"keyfile,omitempty" json:"keyfile,omitempty"`
}

// Runner 命令执行器接口
type Runner interface {
	Execute(cmd Command, output chan<- string) error
	Stop() error
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// CommandStatus 命令状态
type CommandStatus struct {
	IsRunning bool   `json:"isRunning"`
	Command   string `json:"command"`
	Output    string `json:"output"`
}
