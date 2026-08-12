package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FlashDock/define"

	"gopkg.in/yaml.v3"
)

// ThemeSettings 主题设置
type ThemeSettings struct {
	Mode            string  `yaml:"mode" json:"mode"`                       // light, dark, system
	UiAccent        string  `yaml:"uiAccent" json:"uiAccent"`               // 预设 id（blue/hotpink/...）或自定义 hex（#rrggbb）
	TerminalPreset  string  `yaml:"terminalPreset" json:"terminalPreset"`   // classic, monokai, ...
	UiFontFamily    string  `yaml:"uiFontFamily" json:"uiFontFamily"`       // 界面字体 id
	UiFontSize      int     `yaml:"uiFontSize" json:"uiFontSize"`           // 界面字号，默认 14
	ShellFontFamily string  `yaml:"shellFontFamily" json:"shellFontFamily"` // 终端字体 id
	ShellFontSize   int     `yaml:"shellFontSize" json:"shellFontSize"`     // Shell 终端字号，默认 13
	ShellLineHeight float64 `yaml:"shellLineHeight" json:"shellLineHeight"` // Shell 终端行高倍数，默认 1.2
	// ShellMemorySaver 离开 Shell 时卸载工作区 UI（Go 端会话保持，回 Shell 时重建终端）
	ShellMemorySaver bool `yaml:"shellMemorySaver" json:"shellMemorySaver"`
	// ShellAutoReconnect Shell 意外断开时自动重连（默认关闭）
	ShellAutoReconnect bool `yaml:"shellAutoReconnect" json:"shellAutoReconnect"`
	// ShellUseWebgl 终端使用 WebGL 渲染器（失败时回退 canvas；默认关闭）
	ShellUseWebgl bool `yaml:"shellUseWebgl" json:"shellUseWebgl"`
	// ShellTabHibernate 非活动且非分屏标签休眠（停 fit/卸载 xterm，缓冲保留回放；nil 表示默认开启）
	ShellTabHibernate *bool `yaml:"shellTabHibernate,omitempty" json:"shellTabHibernate"`
}

// ProxySettings HTTP/SOCKS 代理
type ProxySettings struct {
	Mode string `yaml:"mode" json:"mode"` // none | manual
	Type string `yaml:"type" json:"type"` // http | socks
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
	// User 可选代理认证用户名
	User string `yaml:"user,omitempty" json:"user"`
	// EncryptedPassword 加密后的代理密码（落盘）；不回传前端
	EncryptedPassword string `yaml:"encryptedPassword,omitempty" json:"-"`
	// Password 明文密码，仅运行时 / 前端表单；不落盘
	Password string `yaml:"-" json:"password"`
}

// ShellLogHighlightColors Shell 日志高亮配色（hex，如 #92d050）
type ShellLogHighlightColors struct {
	Timestamp string `yaml:"timestamp" json:"timestamp"`
	ThreadId  string `yaml:"threadId" json:"threadId"`
	Info      string `yaml:"info" json:"info"`
	Debug     string `yaml:"debug" json:"debug"`
	Warn      string `yaml:"warn" json:"warn"`
	Error     string `yaml:"error" json:"error"`
	Logger    string `yaml:"logger" json:"logger"`
	Sql       string `yaml:"sql" json:"sql"`
	Label     string `yaml:"label" json:"label"`
}

// ShellLogHighlightRuleKeys 可单独关闭的高亮项
var ShellLogHighlightRuleKeys = []string{
	"timestamp", "threadId", "info", "debug", "warn", "error", "logger", "sql", "label",
}

// ShellLogHighlightCustomKeyword 用户自定义日志高亮关键字
type ShellLogHighlightCustomKeyword struct {
	Keyword string `yaml:"keyword" json:"keyword"`
	Color   string `yaml:"color" json:"color"`
	Enabled *bool  `yaml:"enabled,omitempty" json:"enabled,omitempty"` // nil/缺省视为开启
}

// ShellLogCustomKeywordEnabled 自定义关键字是否启用
func ShellLogCustomKeywordEnabled(k ShellLogHighlightCustomKeyword) bool {
	if k.Enabled == nil {
		return true
	}
	return *k.Enabled
}

// NormalizeShellLogHighlightKeywords 清洗自定义关键字列表（最多 64 条，按关键字去重）
func NormalizeShellLogHighlightKeywords(list []ShellLogHighlightCustomKeyword) []ShellLogHighlightCustomKeyword {
	if len(list) == 0 {
		return nil
	}
	out := make([]ShellLogHighlightCustomKeyword, 0, len(list))
	seen := map[string]struct{}{}
	for _, item := range list {
		kw := strings.TrimSpace(item.Keyword)
		if kw == "" {
			continue
		}
		key := strings.ToLower(kw)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		color := strings.TrimSpace(item.Color)
		if !isHexColor(color) {
			color = "#e5c07b"
		}
		en := ShellLogCustomKeywordEnabled(item)
		out = append(out, ShellLogHighlightCustomKeyword{
			Keyword: kw,
			Color:   color,
			Enabled: &en,
		})
		if len(out) >= 64 {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NormalizeShellLogHighlightDisabled 校验并去重 disabled 列表
func NormalizeShellLogHighlightDisabled(list []string) []string {
	if len(list) == 0 {
		return nil
	}
	allowed := map[string]struct{}{
		"timestamp": {}, "threadId": {}, "info": {}, "debug": {}, "warn": {},
		"error": {}, "logger": {}, "sql": {}, "label": {},
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(list))
	for _, k := range list {
		k = strings.TrimSpace(k)
		if _, ok := allowed[k]; !ok {
			continue
		}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// DefaultShellLogHighlightColors 默认配色（接近 WindTerm）
func DefaultShellLogHighlightColors() ShellLogHighlightColors {
	return ShellLogHighlightColors{
		Timestamp: "#92d050",
		ThreadId:  "#c586c0",
		Info:      "#569cd6",
		Debug:     "#ce9178",
		Warn:      "#dcdcaa",
		Error:     "#f44747",
		Logger:    "#4ec9b0",
		Sql:       "#dcdcaa",
		Label:     "#9cdcfe",
	}
}

// NormalizeShellLogHighlightColors 合并缺省项并校验 hex
func NormalizeShellLogHighlightColors(c ShellLogHighlightColors) ShellLogHighlightColors {
	def := DefaultShellLogHighlightColors()
	out := def
	if isHexColor(c.Timestamp) {
		out.Timestamp = c.Timestamp
	}
	if isHexColor(c.ThreadId) {
		out.ThreadId = c.ThreadId
	}
	if isHexColor(c.Info) {
		out.Info = c.Info
	}
	if isHexColor(c.Debug) {
		out.Debug = c.Debug
	}
	if isHexColor(c.Warn) {
		out.Warn = c.Warn
	}
	if isHexColor(c.Error) {
		out.Error = c.Error
	}
	if isHexColor(c.Logger) {
		out.Logger = c.Logger
	}
	if isHexColor(c.Sql) {
		out.Sql = c.Sql
	}
	if isHexColor(c.Label) {
		out.Label = c.Label
	}
	return out
}

func isHexColor(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, ch := range s[1:] {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}

// MachineGroupDefaults 分组级默认连接配置
type MachineGroupDefaults struct {
	Name            string                       `yaml:"name" json:"name"`
	User            string                       `yaml:"user,omitempty" json:"user,omitempty"`
	KeyFile         string                       `yaml:"keyFile,omitempty" json:"keyFile,omitempty"`
	ProxyJump       string                       `yaml:"proxyJump,omitempty" json:"proxyJump,omitempty"`
	StartupCommand  string                       `yaml:"startupCommand,omitempty" json:"startupCommand,omitempty"`
	SftpEncoding    string                       `yaml:"sftpEncoding,omitempty" json:"sftpEncoding,omitempty"`
	AgentForwarding bool                         `yaml:"agentForwarding,omitempty" json:"agentForwarding,omitempty"`
	ProxyOverride   *define.MachineProxyOverride `yaml:"proxyOverride,omitempty" json:"proxyOverride,omitempty"`
	Tags            []string                     `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// GlobalConfig 全局配置结构
type GlobalConfig struct {
	AppId          string            `yaml:"appId" json:"appId"`
	WindowsName    string            `yaml:"windowsName" json:"windowsName"`
	ConfigFiles    []string          `yaml:"configFile" json:"configFile"`
	LastOpenedFile string            `yaml:"lastOpenedFile" json:"lastOpenedFile"`
	WorkPaths      map[string]string `yaml:"workPaths" json:"workPaths"`
	Machines       []define.Machine  `yaml:"machines,omitempty" json:"machines,omitempty"`
	MachineGroups            []string               `yaml:"machineGroups,omitempty" json:"machineGroups,omitempty"`
	MachineGroupDefaultsList []MachineGroupDefaults `yaml:"machineGroupDefaults,omitempty" json:"machineGroupDefaults,omitempty"`
	GlobalAccounts []GlobalAccount `yaml:"globalAccounts,omitempty" json:"globalAccounts,omitempty"`
	ThemeSettings  ThemeSettings   `yaml:"themeSettings" json:"themeSettings"`
	ProxySettings  ProxySettings     `yaml:"proxySettings" json:"proxySettings"`
	// ShellMonitorIntervalMs Shell 监控面板刷新间隔（毫秒），默认 1000
	ShellMonitorIntervalMs int `yaml:"shellMonitorIntervalMs" json:"shellMonitorIntervalMs"`
	// SSHHandshakeTimeoutSec SSH 握手超时（秒）：TCP 连接 + SSH 协商，Shell 与任务远程执行共用，默认 30
	SSHHandshakeTimeoutSec int `yaml:"sshHandshakeTimeoutSec" json:"sshHandshakeTimeoutSec"`
	// ShellTerminalScrollback xterm 滚动缓冲行数上限，默认 2000
	ShellTerminalScrollback int `yaml:"shellTerminalScrollback" json:"shellTerminalScrollback"`
	// TaskOutputMaxLines 任务执行终端输出行数上限，默认 1000
	TaskOutputMaxLines int `yaml:"taskOutputMaxLines" json:"taskOutputMaxLines"`
	// ShellCommandHistoryMax Shell 命令历史每作用域条数上限，默认 200
	ShellCommandHistoryMax int `yaml:"shellCommandHistoryMax" json:"shellCommandHistoryMax"`
	// AppIconPreset Dock/任务栏图标预设：default | helm | pipeline | shell | split | broadcast | sftp | tunnel | yaml | parallel | secure | custom
	AppIconPreset string `yaml:"appIconPreset" json:"appIconPreset"`
	// StartupFullscreen 启动时最大化窗口（非系统独占全屏）
	StartupFullscreen bool `yaml:"startupFullscreen" json:"startupFullscreen"`
	// HomeMinimizedZone 首页分区最小化："" 双栏；"task" 收起任务；"shell" 收起 Shell（另一侧多列展示）
	HomeMinimizedZone string `yaml:"homeMinimizedZone,omitempty" json:"homeMinimizedZone"`
	// ShellMonitorIntervalSec 旧字段（秒），仅用于迁移
	ShellMonitorIntervalSec int `yaml:"shellMonitorIntervalSec,omitempty" json:"-"`
	// ShellLogHighlight Shell 终端日志关键字高亮；nil 表示默认开启
	ShellLogHighlight *bool `yaml:"shellLogHighlight,omitempty" json:"shellLogHighlight"`
	// ShellLogHighlightColors 日志高亮配色
	ShellLogHighlightColors ShellLogHighlightColors `yaml:"shellLogHighlightColors,omitempty" json:"shellLogHighlightColors"`
	// ShellLogHighlightDisabled 关闭高亮的关键字（缺省或空表示全部开启）
	ShellLogHighlightDisabled []string `yaml:"shellLogHighlightDisabled,omitempty" json:"shellLogHighlightDisabled"`
	// ShellLogHighlightKeywords 自定义高亮关键字
	ShellLogHighlightKeywords []ShellLogHighlightCustomKeyword `yaml:"shellLogHighlightKeywords,omitempty" json:"shellLogHighlightKeywords"`
	// ShellAsciiInput Shell 终端获得焦点时临时关闭中文组词（失焦/离开 Shell 后恢复）；nil 表示默认开启
	ShellAsciiInput *bool `yaml:"shellAsciiInput,omitempty" json:"shellAsciiInput"`
	// SftpUseCompressedUpload 目录上传默认走压缩包（zip）再远端解压；nil 表示默认开启
	SftpUseCompressedUpload *bool `yaml:"sftpUseCompressedUpload,omitempty" json:"sftpUseCompressedUpload"`
	// ShellSessionRestore 启动时恢复上次打开的 Shell 标签页；nil 表示默认开启
	ShellSessionRestore *bool `yaml:"shellSessionRestore,omitempty" json:"shellSessionRestore"`
	// ShellCursorLineHighlight 终端光标行高亮；nil/缺省表示关闭
	ShellCursorLineHighlight *bool `yaml:"shellCursorLineHighlight,omitempty" json:"shellCursorLineHighlight"`
	// ShellLineTimestamps 新输出行前缀时间戳；nil/缺省表示关闭
	ShellLineTimestamps *bool `yaml:"shellLineTimestamps,omitempty" json:"shellLineTimestamps"`
	// ShellPasswordAssist 检测到 Password:/密码 提示时显示终端底部输入条；nil 表示默认开启
	ShellPasswordAssist *bool `yaml:"shellPasswordAssist,omitempty" json:"shellPasswordAssist"`
	// ExternalEditorCommand 外置编辑器命令（空则系统默认打开）；可用 {path} 占位
	ExternalEditorCommand string `yaml:"externalEditorCommand,omitempty" json:"externalEditorCommand,omitempty"`
	// FileAssociations 扩展名 → 打开命令（如 ".go": "code {path}"）
	FileAssociations map[string]string `yaml:"fileAssociations,omitempty" json:"fileAssociations,omitempty"`
}

// ShellSessionRestoreEnabled 启动时是否恢复 Shell 会话（功能已下线，恒为 false）
func ShellSessionRestoreEnabled(cfg *GlobalConfig) bool {
	return false
}

// NormalizeHomeMinimizedZone 校验首页最小化分区
func NormalizeHomeMinimizedZone(zone string) string {
	switch strings.TrimSpace(zone) {
	case "task", "shell":
		return strings.TrimSpace(zone)
	default:
		return ""
	}
}

// NormalizeAppIconPreset 校验 Dock 图标预设 id
func NormalizeAppIconPreset(preset string) string {
	switch strings.TrimSpace(preset) {
	case "helm", "pipeline", "shell", "split", "broadcast", "sftp", "tunnel", "yaml", "parallel", "secure", "custom":
		return strings.TrimSpace(preset)
	// 兼容旧预设 id → 相近模块
	case "aurora", "ocean", "midnight", "slate":
		return "helm"
	case "emerald", "lime", "teal", "green":
		return "pipeline"
	case "frost":
		return "shell"
	case "sunset", "gold", "amber":
		return "sftp"
	case "nebula", "purple", "rose":
		return "parallel"
	case "ember":
		return "secure"
	default:
		return "default"
	}
}

// GlobalConfigManager 全局配置管理器
type GlobalConfigManager struct {
	configPath string
	config     *GlobalConfig
}

// NewGlobalConfigManager 创建全局配置管理器
func NewGlobalConfigManager(configPath string) *GlobalConfigManager {
	if configPath == "" {
		configHome, err := ConfigHomeDir()
		if err != nil {
			configPath = "global_config.yaml"
		} else {
			configPath = filepath.Join(configHome, "global_config.yaml")
		}
	}
	return &GlobalConfigManager{
		configPath: configPath,
	}
}

// LoadGlobalConfig 加载全局配置文件
func (gcm *GlobalConfigManager) LoadGlobalConfig() (*GlobalConfig, error) {
	expandedPath := expandPath(gcm.configPath)

	configDir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %w", err)
	}

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := gcm.createDefaultGlobalConfig(); err != nil {
				return nil, fmt.Errorf("创建默认全局配置失败: %w", err)
			}
			data, err = os.ReadFile(expandedPath)
			if err != nil {
				return nil, fmt.Errorf("读取全局配置文件失败: %w", err)
			}
		} else {
			return nil, fmt.Errorf("读取全局配置文件失败: %w", err)
		}
	}

	// 文件已存在但为空时，不覆盖为默认配置
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("全局配置文件为空: %s", expandedPath)
	}

	var config GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析全局配置文件失败: %w", err)
	}

	gcm.config = &config
	dirty := gcm.ensureMachineIDs() || gcm.ensureGlobalAccountIDs()
	if gcm.migrateShellMonitorInterval() {
		dirty = true
	}
	normalizedZone := NormalizeHomeMinimizedZone(gcm.config.HomeMinimizedZone)
	if gcm.config.HomeMinimizedZone != normalizedZone {
		gcm.config.HomeMinimizedZone = normalizedZone
		dirty = true
	}
	if dirty {
		if err := gcm.SaveGlobalConfig(gcm.config); err != nil {
			return nil, fmt.Errorf("迁移配置失败: %w", err)
		}
	}
	return &config, nil
}

// migrateShellMonitorInterval 将旧的秒级间隔迁到毫秒
func (gcm *GlobalConfigManager) migrateShellMonitorInterval() bool {
	if gcm.config == nil {
		return false
	}
	if gcm.config.ShellMonitorIntervalMs > 0 {
		if gcm.config.ShellMonitorIntervalSec != 0 {
			gcm.config.ShellMonitorIntervalSec = 0
			return true
		}
		return false
	}
	if gcm.config.ShellMonitorIntervalSec > 0 {
		gcm.config.ShellMonitorIntervalMs = gcm.config.ShellMonitorIntervalSec * 1000
		gcm.config.ShellMonitorIntervalSec = 0
		return true
	}
	return false
}

// migrateLegacyLogPath 已废弃（执行日志功能已移除）

// SaveGlobalConfig 保存全局配置文件
func (gcm *GlobalConfigManager) SaveGlobalConfig(config *GlobalConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化全局配置失败: %w", err)
	}

	expandedPath := expandPath(gcm.configPath)

	// 确保目录存在
	configDir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	if err := os.WriteFile(expandedPath, data, 0644); err != nil {
		return fmt.Errorf("保存全局配置文件失败: %w", err)
	}

	gcm.config = config
	return nil
}

// GetConfig 获取全局配置
func (gcm *GlobalConfigManager) GetConfig() *GlobalConfig {
	return gcm.config
}

// GetConfigPath 获取全局配置文件路径
func (gcm *GlobalConfigManager) GetConfigPath() string {
	return gcm.configPath
}

// normalizeConfigPath 规范化路径用于比较（Windows 下忽略大小写）
func normalizeConfigPath(path string) string {
	path = expandPath(path)
	path = filepath.Clean(path)
	return strings.ToLower(path)
}

// UpdateLastOpenedFile 更新最后打开的配置文件（仅在实际变更时写入磁盘）
func (gcm *GlobalConfigManager) UpdateLastOpenedFile(filePath string) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}

	if normalizeConfigPath(gcm.config.LastOpenedFile) == normalizeConfigPath(filePath) {
		return nil
	}

	gcm.config.LastOpenedFile = filePath
	return gcm.SaveGlobalConfig(gcm.config)
}

// GetLastOpenedFile 获取最后打开的配置文件路径
func (gcm *GlobalConfigManager) GetLastOpenedFile() string {
	if gcm.config == nil {
		return ""
	}
	return gcm.config.LastOpenedFile
}

// AddConfigFile 添加配置文件路径
func (gcm *GlobalConfigManager) AddConfigFile(filePath string) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}

	// 检查是否已存在
	for _, existing := range gcm.config.ConfigFiles {
		if existing == filePath {
			return nil // 已存在，不需要添加
		}
	}

	gcm.config.ConfigFiles = append(gcm.config.ConfigFiles, filePath)
	return gcm.SaveGlobalConfig(gcm.config)
}

// RemoveConfigFile 移除配置文件路径
func (gcm *GlobalConfigManager) RemoveConfigFile(filePath string) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}

	for i, existing := range gcm.config.ConfigFiles {
		if existing == filePath {
			gcm.config.ConfigFiles = append(gcm.config.ConfigFiles[:i], gcm.config.ConfigFiles[i+1:]...)
			break
		}
	}

	return gcm.SaveGlobalConfig(gcm.config)
}

// ReplaceWorkPaths 替换字符串中的工作路径变量
func (gcm *GlobalConfigManager) ReplaceWorkPaths(input string) string {
	if gcm.config == nil || gcm.config.WorkPaths == nil {
		return input
	}

	result := input
	for key, value := range gcm.config.WorkPaths {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// createDefaultGlobalConfig 创建默认全局配置
func (gcm *GlobalConfigManager) createDefaultGlobalConfig() error {
	defaultCfgPath := DefaultConfigPath()
	defaultConfig := &GlobalConfig{
		AppId:       "com.runner",
		WindowsName: "FlashDock",
		ConfigFiles: []string{
			defaultCfgPath,
		},
		LastOpenedFile: defaultCfgPath,
		WorkPaths: map[string]string{
			"HOME": "~",
		},
		ThemeSettings: ThemeSettings{
			Mode:            "light",
			UiAccent:        "blue",
			TerminalPreset:  "classic",
			UiFontFamily:    "system",
			UiFontSize:      14,
			ShellFontFamily: "consolas",
			ShellFontSize:   13,
			ShellLineHeight: 1.2,
		},
		ProxySettings: ProxySettings{
			Mode: "none",
			Type: "http",
			Host: "",
			Port: 7890,
		},
		ShellMonitorIntervalMs:  1000,
		SSHHandshakeTimeoutSec:  30,
		ShellTerminalScrollback: 2000,
		TaskOutputMaxLines:      1000,
		ShellCommandHistoryMax:  200,
		AppIconPreset:           "default",
		ShellLogHighlight:       boolPtr(true),
		ShellLogHighlightColors: DefaultShellLogHighlightColors(),
		ShellAsciiInput:         boolPtr(true),
		SftpUseCompressedUpload: boolPtr(true),
	}

	return gcm.SaveGlobalConfig(defaultConfig)
}

func boolPtr(v bool) *bool { return &v }

// ShellLogHighlightEnabled 日志高亮是否开启（缺省 true）
func ShellLogHighlightEnabled(cfg *GlobalConfig) bool {
	if cfg == nil || cfg.ShellLogHighlight == nil {
		return true
	}
	return *cfg.ShellLogHighlight
}

// ShellAsciiInputEnabled Shell 终端临时英文输入是否开启（缺省 true）
func ShellAsciiInputEnabled(cfg *GlobalConfig) bool {
	if cfg == nil || cfg.ShellAsciiInput == nil {
		return true
	}
	return *cfg.ShellAsciiInput
}

// SftpUseCompressedUploadEnabled 目录上传是否默认压缩（缺省 true）
func SftpUseCompressedUploadEnabled(cfg *GlobalConfig) bool {
	if cfg == nil || cfg.SftpUseCompressedUpload == nil {
		return true
	}
	return *cfg.SftpUseCompressedUpload
}

// ShellCursorLineHighlightEnabled 光标行高亮是否开启（缺省 false）
func ShellCursorLineHighlightEnabled(cfg *GlobalConfig) bool {
	if cfg == nil || cfg.ShellCursorLineHighlight == nil {
		return false
	}
	return *cfg.ShellCursorLineHighlight
}

// ShellLineTimestampsEnabled 行时间戳是否开启（缺省 false）
func ShellLineTimestampsEnabled(cfg *GlobalConfig) bool {
	if cfg == nil || cfg.ShellLineTimestamps == nil {
		return false
	}
	return *cfg.ShellLineTimestamps
}

// ShellTabHibernateEnabled 非活动标签是否休眠（缺省 true）
func ShellTabHibernateEnabled(cfg *GlobalConfig) bool {
	if cfg == nil || cfg.ThemeSettings.ShellTabHibernate == nil {
		return true
	}
	return *cfg.ThemeSettings.ShellTabHibernate
}

// ThemeShellTabHibernateEnabled 主题设置中的标签休眠开关（缺省 true）
func ThemeShellTabHibernateEnabled(ts ThemeSettings) bool {
	if ts.ShellTabHibernate == nil {
		return true
	}
	return *ts.ShellTabHibernate
}

// ShellPasswordAssistEnabled 密码提示辅助输入是否开启（缺省 true）
func ShellPasswordAssistEnabled(cfg *GlobalConfig) bool {
	if cfg == nil || cfg.ShellPasswordAssist == nil {
		return true
	}
	return *cfg.ShellPasswordAssist
}

// AddMachine 添加或更新机器配置（按 ID）
func (gcm *GlobalConfigManager) AddMachine(machine *define.Machine) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}

	machine.EnsureID()
	gcm.EnsureMachineGroupRegistered(machine.Group)
	for i, existing := range gcm.config.Machines {
		if existing.ID == machine.ID {
			gcm.config.Machines[i] = *machine
			return gcm.SaveGlobalConfig(gcm.config)
		}
	}

	gcm.config.Machines = append(gcm.config.Machines, *machine)
	return gcm.SaveGlobalConfig(gcm.config)
}

// GetMachine 根据 ID 获取机器配置
func (gcm *GlobalConfigManager) GetMachine(id string) *define.Machine {
	if gcm.config == nil || id == "" {
		return nil
	}

	for _, machine := range gcm.config.Machines {
		if machine.ID == id {
			return &machine
		}
	}
	return nil
}

// GetMachineByName 根据名称获取机器配置
func (gcm *GlobalConfigManager) GetMachineByName(name string) *define.Machine {
	if gcm.config == nil || name == "" {
		return nil
	}

	for _, machine := range gcm.config.Machines {
		if machine.Name == name {
			return &machine
		}
	}
	return nil
}

// RemoveMachine 按 ID 移除机器配置
func (gcm *GlobalConfigManager) RemoveMachine(id string) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}

	for i, machine := range gcm.config.Machines {
		if machine.ID == id {
			gcm.config.Machines = append(gcm.config.Machines[:i], gcm.config.Machines[i+1:]...)
			return gcm.SaveGlobalConfig(gcm.config)
		}
	}

	return fmt.Errorf("机器配置 '%s' 不存在", id)
}

func (gcm *GlobalConfigManager) ensureMachineIDs() bool {
	if gcm.config == nil {
		return false
	}
	changed := false
	for i := range gcm.config.Machines {
		if gcm.config.Machines[i].ID == "" {
			gcm.config.Machines[i].EnsureID()
			changed = true
		}
	}
	return changed
}

// GetAllMachines 获取所有机器配置
func (gcm *GlobalConfigManager) GetAllMachines() []define.Machine {
	if gcm.config == nil {
		return []define.Machine{}
	}

	return gcm.config.Machines
}

// AddWorkPath 添加工作路径
func (gcm *GlobalConfigManager) AddWorkPath(key, value string) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}

	if gcm.config.WorkPaths == nil {
		gcm.config.WorkPaths = make(map[string]string)
	}

	gcm.config.WorkPaths[key] = value
	return gcm.SaveGlobalConfig(gcm.config)
}

// UpdateWorkPath 更新工作路径
func (gcm *GlobalConfigManager) UpdateWorkPath(key, value string) error {
	return gcm.AddWorkPath(key, value) // 添加和更新逻辑相同
}

// RemoveWorkPath 移除工作路径
func (gcm *GlobalConfigManager) RemoveWorkPath(key string) error {
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}

	if gcm.config.WorkPaths == nil {
		return fmt.Errorf("工作路径 '%s' 不存在", key)
	}

	if _, exists := gcm.config.WorkPaths[key]; !exists {
		return fmt.Errorf("工作路径 '%s' 不存在", key)
	}

	delete(gcm.config.WorkPaths, key)
	return gcm.SaveGlobalConfig(gcm.config)
}

// GetWorkPath 获取工作路径
func (gcm *GlobalConfigManager) GetWorkPath(key string) (string, bool) {
	if gcm.config == nil || gcm.config.WorkPaths == nil {
		return "", false
	}

	value, exists := gcm.config.WorkPaths[key]
	return value, exists
}

// GetAllWorkPaths 获取所有工作路径
func (gcm *GlobalConfigManager) GetAllWorkPaths() map[string]string {
	if gcm.config == nil {
		return make(map[string]string)
	}

	if gcm.config.WorkPaths == nil {
		return make(map[string]string)
	}

	// 返回副本，避免外部修改
	result := make(map[string]string)
	for k, v := range gcm.config.WorkPaths {
		result[k] = v
	}

	return result
}
