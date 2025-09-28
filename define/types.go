package define

import (
	"fmt"
	"os"
	"path/filepath"
	"quick-cmd/crypto"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
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
	WorkDir     string    `yaml:"workdir,omitempty" json:"workdir,omitempty"`
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

// RemoteMachine 远程机器包装类，包含 SSH 和 SFTP 客户端
type RemoteMachine struct {
	SSHClient  *ssh.Client
	SFTPClient *sftp.Client
}

// NewRemoteMachine 创建远程机器包装类
func NewRemoteMachine() *RemoteMachine {
	return &RemoteMachine{}
}

// Connect 连接到远程机器并初始化 SSH 和 SFTP 客户端
func (rm *RemoteMachine) Connect(machine *Machine) error {
	// 获取敏感数据
	sensitiveData, err := machine.GetSensitiveData()
	if err != nil {
		return fmt.Errorf("获取敏感数据失败: %w", err)
	}

	var auth []ssh.AuthMethod

	// 密钥认证
	if machine.KeyFile != "" {
		key, err := rm.loadPrivateKey(machine.KeyFile)
		if err != nil {
			return fmt.Errorf("加载私钥失败: %w", err)
		}
		auth = append(auth, ssh.PublicKeys(key))
	}

	// 密码认证
	if sensitiveData.Password != "" {
		auth = append(auth, ssh.Password(sensitiveData.Password))
	}

	if len(auth) == 0 {
		return fmt.Errorf("未配置认证方式")
	}

	config := &ssh.ClientConfig{
		User:            sensitiveData.User,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证主机密钥
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", sensitiveData.Host, sensitiveData.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}

	rm.SSHClient = client

	// 创建 SFTP 客户端
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return fmt.Errorf("SFTP连接失败: %w", err)
	}

	rm.SFTPClient = sftpClient
	return nil
}

// Close 关闭连接
func (rm *RemoteMachine) Close() error {
	var err error
	if rm.SFTPClient != nil {
		err = rm.SFTPClient.Close()
	}
	if rm.SSHClient != nil {
		if closeErr := rm.SSHClient.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

// IsConnected 检查是否已连接
func (rm *RemoteMachine) IsConnected() bool {
	return rm.SSHClient != nil && rm.SFTPClient != nil
}

// loadPrivateKey 加载私钥
func (rm *RemoteMachine) loadPrivateKey(keyPath string) (ssh.Signer, error) {
	// 展开路径
	if strings.HasPrefix(keyPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		keyPath = filepath.Join(homeDir, keyPath[2:])
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	return signer, nil
}

// NewSession 创建新的 SSH session
func (rm *RemoteMachine) NewSession() (*ssh.Session, error) {
	if rm.SSHClient == nil {
		return nil, fmt.Errorf("SSH客户端未连接")
	}
	return rm.SSHClient.NewSession()
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
	ProjectName       string
	SubProjectName    string
	Commands          []Command
	CurrentIndex      int
	OutputChannel     chan<- string
	ProjectWorkDir    string
	SubProjectWorkDir string
}

// CommandStatus 命令状态 (保持向后兼容)
type CommandStatus struct {
	IsRunning bool   `json:"isRunning"`
	Command   string `json:"command"`
	Output    string `json:"output"`
}

// OperationEvent 操作事件
type OperationEvent struct {
	Type        string `json:"type"`        // 事件类型：new_window, open_config, switch_config, refresh_config, machine_config, env_config
	NeedReload  bool   `json:"needReload"`  // 是否需要重新加载
	Message     string `json:"message"`     // 提示信息
	MessageType string `json:"messageType"` // 提示类型：success, error, warning, info
	Timestamp   int64  `json:"timestamp"`   // 时间戳
	Data        any    `json:"data"`        // 额外数据
}

// OperationType 操作类型常量
const (
	OpTypeNewWindow     = "new_window"
	OpTypeOpenConfig    = "open_config"
	OpTypeSwitchConfig  = "switch_config"
	OpTypeRefreshConfig = "refresh_config"
	OpTypeMachineConfig = "machine_config"
	OpTypeEnvConfig     = "env_config"
)

// MessageType 消息类型常量
const (
	MsgTypeSuccess = "success"
	MsgTypeError   = "error"
	MsgTypeWarning = "warning"
	MsgTypeInfo    = "info"
)

type CMDManager struct {
	SpecialCmd map[string]func(*RemoteMachine, []string, chan<- string) error
}

func (cm *CMDManager) RegSpecialCmd(c string, fu func(*RemoteMachine, []string, chan<- string) error) {
	cm.SpecialCmd[c] = fu
}

func (cm *CMDManager) GetSpecialCmd(c string) func(*RemoteMachine, []string, chan<- string) error {
	return cm.SpecialCmd[c]
}
