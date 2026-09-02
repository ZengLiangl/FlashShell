package define

import (
	"FlashDock/crypto"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
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

// ProjectSummary 首页列表用轻量摘要（不含 steps）
type ProjectSummary struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	SubProjectCount int    `json:"subProjectCount"`
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
	Steps       StepList `yaml:"steps" json:"steps"`
	Machine     string   `yaml:"machine,omitempty" json:"machine,omitempty"`
	WorkDir     string   `yaml:"workdir,omitempty" json:"workdir,omitempty"`
	Parallel    bool     `yaml:"parallel,omitempty" json:"parallel,omitempty"`
}

// Machine 远程机器配置
type Machine struct {
	ID            string `yaml:"id" json:"id"`
	EncryptedData string `yaml:"encrypted_data,omitempty" json:"encrypted_data,omitempty"` // 加密后内容
	Name          string `yaml:"name" json:"name"`
	Group         string `yaml:"group,omitempty" json:"group,omitempty"`
	KeyFile       string `yaml:"key_file,omitempty" json:"key_file,omitempty"`
	// ProxyJump 跳板机：引用另一台 Machine 名称，或 host[:port] / user@host[:port]（单跳，与 JumpChain 二选一）
	ProxyJump string `yaml:"proxyJump,omitempty" json:"proxyJump,omitempty"`
	// JumpChain 有序跳板链（机器名或 host[:port] / user@host[:port]）；非空时优先于 ProxyJump
	JumpChain []string `yaml:"jumpChain,omitempty" json:"jumpChain,omitempty"`
	// ProxyOverride 每主机代理覆盖（inherit=全局 | none=直连 | manual=独立代理）
	ProxyOverride *MachineProxyOverride `yaml:"proxyOverride,omitempty" json:"proxyOverride,omitempty"`
	// LegacyAlgorithms 启用旧设备兼容算法（华为/思科等网络设备）
	LegacyAlgorithms bool `yaml:"legacyAlgorithms,omitempty" json:"legacyAlgorithms,omitempty"`
	// SkipEcdsaHostKey 跳过 ECDSA 主机密钥（部分老设备兼容）
	SkipEcdsaHostKey bool `yaml:"skipEcdsaHostKey,omitempty" json:"skipEcdsaHostKey,omitempty"`
	// SftpEncoding SFTP 文件名编码：auto | utf-8 | gb18030
	SftpEncoding string `yaml:"sftpEncoding,omitempty" json:"sftpEncoding,omitempty"`
	// SftpFileProtocol 文件协议：auto（优先 SFTP，失败回退 SCP）| sftp | scp
	SftpFileProtocol string `yaml:"sftpFileProtocol,omitempty" json:"sftpFileProtocol,omitempty"`
	// SftpSudo 以 sudo 提权方式打开 SFTP（需密码；与 SCP 模式互斥）
	SftpSudo bool `yaml:"sftpSudo,omitempty" json:"sftpSudo,omitempty"`
	// StartupCommand 连接后自动执行的启动命令（单行）
	StartupCommand string `yaml:"startupCommand,omitempty" json:"startupCommand,omitempty"`
	// AgentForwarding 启用 SSH Agent 转发
	AgentForwarding bool `yaml:"agentForwarding,omitempty" json:"agentForwarding,omitempty"`
	// LocalEcho 终端本地回显：可打印字符由客户端立即显示，并抑制远端重复回显（高延迟输入优化；非 X11 转发）
	LocalEcho bool `yaml:"localEcho,omitempty" json:"localEcho,omitempty"`
	// TerminalPreset 本机终端配色覆盖（空=跟随全局主题）
	TerminalPreset string `yaml:"terminalPreset,omitempty" json:"terminalPreset,omitempty"`
	// Pinned 首页置顶
	Pinned bool `yaml:"pinned,omitempty" json:"pinned,omitempty"`
	// Tags 主机标签（检索与筛选）
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	// AIPolicy 该主机对 MCP/AI 的策略档：disabled | readonly | approval | allowlist | trusted；空=disabled（历史机器）
	AIPolicy string `yaml:"aiPolicy,omitempty" json:"aiPolicy,omitempty"`
	// AIAllowlist 当 AIPolicy=allowlist 时：命令前缀或正则，命中则 auto，否则审批
	AIAllowlist []string `yaml:"aiAllowlist,omitempty" json:"aiAllowlist,omitempty"`
	// AIAllowSudo 是否允许 AI 经审批后执行含 sudo 的命令；false 则 sudo 直接 [denied]
	AIAllowSudo bool `yaml:"aiAllowSudo,omitempty" json:"aiAllowSudo,omitempty"`
	// Notes 主机备注（纯文本/Markdown）
	Notes string `yaml:"notes,omitempty" json:"notes,omitempty"`
	// Icon 主机图标：预设 id 或单个 emoji
	Icon string `yaml:"icon,omitempty" json:"icon,omitempty"`
	// IdentityID 引用全局帐号（身份）；连接时覆盖用户名/密码/密钥，不落盘
	IdentityID string `yaml:"identityId,omitempty" json:"identityId,omitempty"`
	// Tunnels SSH 隧道（本地/远程/动态），连接后自动建立
	Tunnels []SSHTunnel `yaml:"tunnels,omitempty" json:"tunnels,omitempty"`
	// ListHost/ListPort/ListUser 列表展示与搜索用（明文；密码等仍在 encrypted_data）
	ListHost string `yaml:"list_host,omitempty" json:"list_host,omitempty"`
	ListPort int    `yaml:"list_port,omitempty" json:"list_port,omitempty"`
	ListUser string `yaml:"list_user,omitempty" json:"list_user,omitempty"`
	// ShellMonitorOpen Shell 左侧监控栏是否展开；缺省（空）为展开，false 为收起
	ShellMonitorOpen *bool `yaml:"shellMonitorOpen,omitempty" json:"shellMonitorOpen,omitempty"`
	// 列表展示用（API 响应字段，由 List* 或敏感数据填充）
	Host string `yaml:"-" json:"host,omitempty"`
	Port int    `yaml:"-" json:"port,omitempty"`
	User string `yaml:"-" json:"user,omitempty"`
	// 运行时数据（不序列化）
	sensitiveData *SensitiveData `yaml:"-"`
}

// IsShellMonitorOpen 监控栏是否展开：字段为空时默认展开
func (m *Machine) IsShellMonitorOpen() bool {
	if m == nil || m.ShellMonitorOpen == nil {
		return true
	}
	return *m.ShellMonitorOpen
}

// SetShellMonitorOpen 设置监控栏展开状态；展开时清空字段（保持「空=展开」）
func (m *Machine) SetShellMonitorOpen(open bool) {
	if m == nil {
		return
	}
	if open {
		m.ShellMonitorOpen = nil
		return
	}
	f := false
	m.ShellMonitorOpen = &f
}

// SSHTunnel 单条隧道配置
type SSHTunnel struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	Name       string `yaml:"name,omitempty" json:"name,omitempty"`
	Type       string `yaml:"type" json:"type"` // local | remote | dynamic
	LocalHost  string `yaml:"localHost,omitempty" json:"localHost,omitempty"`
	LocalPort  int    `yaml:"localPort" json:"localPort"`
	RemoteHost string `yaml:"remoteHost,omitempty" json:"remoteHost,omitempty"`
	RemotePort int    `yaml:"remotePort,omitempty" json:"remotePort,omitempty"`
	Temporary  bool   `yaml:"-" json:"temporary,omitempty"` // 运行时临时隧道，不持久化
}

// SSHTunnelStatus 隧道运行状态
type SSHTunnelStatus struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	LocalHost  string `json:"localHost"`
	LocalPort  int    `json:"localPort"`
	RemoteHost string `json:"remoteHost"`
	RemotePort int    `json:"remotePort"`
	Active     bool   `json:"active"`
	Error      string `json:"error,omitempty"`
	StartedAt  int64  `json:"startedAt"`
	Temporary  bool   `json:"temporary,omitempty"`
}

// EnsureID 确保机器有唯一 ID
func (m *Machine) EnsureID() {
	if m != nil && m.ID == "" {
		m.ID = uuid.NewString()
	}
}

// SensitiveData 敏感数据
type SensitiveData struct {
	Host          string `yaml:"host" json:"host"`
	Port          int    `yaml:"port" json:"port"`
	User          string `yaml:"user" json:"user"`
	Password      string `yaml:"password,omitempty" json:"password,omitempty"`
	KeyPassphrase string `yaml:"keyPassphrase,omitempty" json:"keyPassphrase,omitempty"`
}

// SetSensitiveData 设置敏感数据并加密
func (m *Machine) SetSensitiveData(data *SensitiveData) error {
	if data == nil {
		return nil
	}

	// 创建加密用的数据结构
	cryptoData := &crypto.SensitiveData{
		Name:          m.Name,
		Host:          data.Host,
		Port:          data.Port,
		Username:      data.User,
		Password:      data.Password,
		KeyPassphrase: data.KeyPassphrase,
		KeyData:       []byte{}, // 密钥文件内容暂时为空
	}

	// 加密敏感数据
	encryptedStr, err := crypto.EncryptSensitiveData(cryptoData)
	if err != nil {
		return err
	}

	// 设置加密数据和敏感数据
	m.EncryptedData = encryptedStr
	m.sensitiveData = data
	m.syncListHintsFromSensitive(data)

	return nil
}

func (m *Machine) syncListHintsFromSensitive(data *SensitiveData) {
	if m == nil || data == nil {
		return
	}
	m.ListHost = strings.TrimSpace(data.Host)
	m.ListPort = data.Port
	if m.ListPort <= 0 {
		m.ListPort = 22
	}
	m.ListUser = strings.TrimSpace(data.User)
}

// ApplyListFieldsForDisplay 用 List* hint 填充展示字段；有 hint 返回 true
func (m *Machine) ApplyListFieldsForDisplay() bool {
	if m == nil || m.ListHost == "" {
		return false
	}
	m.Host = m.ListHost
	m.Port = m.ListPort
	if m.Port <= 0 {
		m.Port = 22
	}
	m.User = m.ListUser
	return true
}

// EnsureListHints 为缺少 hint 的旧配置从加密块迁移（仅 host/port/user）
func (m *Machine) EnsureListHints() (changed bool, err error) {
	if m == nil || m.ListHost != "" || m.EncryptedData == "" {
		return false, nil
	}
	s, err := m.GetSensitiveData()
	if err != nil || s == nil || strings.TrimSpace(s.Host) == "" {
		return false, err
	}
	m.syncListHintsFromSensitive(s)
	return true, nil
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
		Host:          cryptoData.Host,
		Port:          cryptoData.Port,
		User:          cryptoData.Username,
		Password:      cryptoData.Password,
		KeyPassphrase: cryptoData.KeyPassphrase,
	}

	// 缓存解密后的数据
	m.sensitiveData = data

	return data, nil
}

// ClearSensitiveData 清除敏感数据缓存
func (m *Machine) ClearSensitiveData() {
	m.sensitiveData = nil
}

// OverlaySensitiveFields 覆盖运行时用户名/密码/口令（身份引用等，不落盘）
func (m *Machine) OverlaySensitiveFields(user, password, keyPassphrase string) error {
	sensitive, err := m.GetSensitiveData()
	if err != nil {
		return err
	}
	if sensitive == nil {
		sensitive = &SensitiveData{}
	}
	if u := strings.TrimSpace(user); u != "" {
		sensitive.User = u
	}
	if password != "" {
		sensitive.Password = password
	}
	if keyPassphrase != "" {
		sensitive.KeyPassphrase = keyPassphrase
	}
	m.sensitiveData = sensitive
	return nil
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

// Connect 连接到远程机器。withSFTP 为 true 时额外初始化 SFTP 客户端（文件上传需要）。
func (rm *RemoteMachine) Connect(machine *Machine, withSFTP bool) error {
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
		HostKeyCallback: currentHostKeyCallback(),
		Timeout:         SSHHandshakeTimeout(),
	}

	addr := fmt.Sprintf("%s:%d", sensitiveData.Host, sensitiveData.Port)
	handshakeTimeout := SSHHandshakeTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), handshakeTimeout)
	defer cancel()
	conn, err := DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("SSH连接失败: %w", err)
	}
	_ = conn.SetDeadline(time.Now().Add(handshakeTimeout))
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("SSH连接失败: %w", err)
	}
	_ = conn.SetDeadline(time.Time{})
	client := ssh.NewClient(c, chans, reqs)

	rm.SSHClient = client

	if withSFTP {
		if err := rm.EnsureSFTP(); err != nil {
			client.Close()
			rm.SSHClient = nil
			return err
		}
	}

	return nil
}

// EnsureSFTP 在已有 SSH 连接上初始化 SFTP（可重复调用）。
func (rm *RemoteMachine) EnsureSFTP() error {
	if rm.SFTPClient != nil {
		return nil
	}
	if rm.SSHClient == nil {
		return fmt.Errorf("SSH客户端未连接")
	}
	// 默认 maxPacket=32KiB：过大的 MaxPacketUnchecked 在部分服务器上会直接 EOF。
	// UseConcurrentWrites + ReadFromWithConcurrency（见 utils.CopySFTPUpload）才能叠包跑满高延迟链路。
	sftpClient, err := sftp.NewClient(rm.SSHClient,
		sftp.UseConcurrentWrites(true),
		sftp.MaxConcurrentRequestsPerFile(64),
	)
	if err != nil {
		return fmt.Errorf("SFTP连接失败: %w", err)
	}
	rm.SFTPClient = sftpClient
	return nil
}

// SetSFTPClient 注入已打开的 SFTP 客户端（如 sudo 提权通道）
func (rm *RemoteMachine) SetSFTPClient(c *sftp.Client) {
	rm.SFTPClient = c
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

// IsConnected 检查 SSH 是否已连接
func (rm *RemoteMachine) IsConnected() bool {
	return rm.SSHClient != nil
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
	Execute(cmd Command, output chan<- string, onStepStart func(step string), onStepComplete func(), shouldStop func() bool) error
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
	CurrentStep       string `json:"currentStep"`
	CompletedCommands int    `json:"completedCommands"`
	CompletedSteps    int    `json:"completedSteps"`
	TotalCommands     int    `json:"totalCommands"`
	TotalSteps        int    `json:"totalSteps"`
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
	WorkPathVars      map[string]string
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
