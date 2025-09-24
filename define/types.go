package define

import (
	"quick-cmd/crypto"
)

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
	EncryptedData string `yaml:"encrypted_data,omitempty" json:"encrypted_data,omitempty"` // 加密后内容
	Name          string `yaml:"name" json:"name"`
	KeyFile       string `yaml:"key_file,omitempty" json:"key_file,omitempty"`
	// 运行时数据（不序列化）
	sensitiveData *SensitiveData `yaml:"-"`
}

// SensitiveData 敏感数据
type SensitiveData struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

// SetSensitiveData 设置敏感数据并加密
func (m *Machine) SetSensitiveData(data *SensitiveData) error {
	if data == nil {
		return nil
	}

	// 创建加密用的数据结构
	cryptoData := &crypto.SensitiveData{
		Name:     m.Name,
		Host:     data.Host,
		Port:     data.Port,
		Username: data.User,
		Password: data.Password,
		KeyData:  []byte{}, // 密钥文件内容暂时为空
	}

	// 加密敏感数据
	encryptedStr, err := crypto.EncryptSensitiveData(cryptoData)
	if err != nil {
		return err
	}

	// 设置加密数据和敏感数据
	m.EncryptedData = encryptedStr
	m.sensitiveData = data

	return nil
}

// GetSensitiveData 获取敏感数据（解密）
func (m *Machine) GetSensitiveData() (*SensitiveData, error) {
	if m.sensitiveData != nil {
		return m.sensitiveData, nil
	}

	if m.EncryptedData == "" {
		return &SensitiveData{}, nil
	}

	// 解密数据
	cryptoData, err := crypto.DecryptSensitiveData(m.EncryptedData)
	if err != nil {
		return nil, err
	}

	// 转换为内部数据结构
	data := &SensitiveData{
		Host:     cryptoData.Host,
		Port:     cryptoData.Port,
		User:     cryptoData.Username,
		Password: cryptoData.Password,
	}

	// 缓存解密后的数据
	m.sensitiveData = data

	return data, nil
}

// ClearSensitiveData 清除敏感数据缓存
func (m *Machine) ClearSensitiveData() {
	m.sensitiveData = nil
}

// Runner 命令执行器接口
type Runner interface {
	Execute(cmd Command, output chan<- string) error
	Stop() error
}

// SubProjectExecutor SubProject 执行器接口
type SubProjectExecutor interface {
	ExecuteSubProject(projectName, subProjectName string, output chan<- string) error
	StopSubProject(projectName, subProjectName string) error
	GetExecutionStatus() *SubProjectStatus
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// SubProjectStatus SubProject 执行状态
type SubProjectStatus struct {
	ProjectName       string `json:"projectName"`
	SubProjectName    string `json:"subProjectName"`
	IsRunning         bool   `json:"isRunning"`
	CurrentCommand    string `json:"currentCommand"`
	CompletedCommands int    `json:"completedCommands"`
	TotalCommands     int    `json:"totalCommands"`
	Output            string `json:"output"`
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	ProjectName    string
	SubProjectName string
	Commands       []Command
	CurrentIndex   int
	OutputChannel  chan<- string
	WorkDir        string
}

// CommandStatus 命令状态 (保持向后兼容)
type CommandStatus struct {
	IsRunning bool   `json:"isRunning"`
	Command   string `json:"command"`
	Output    string `json:"output"`
}
