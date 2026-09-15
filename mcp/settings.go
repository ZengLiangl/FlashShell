package mcp

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func loadSettings() Settings {
	s := defaultSettings()
	root, err := homeDir()
	if err != nil {
		return s
	}
	b, err := os.ReadFile(join(root, settingsFile))
	if err != nil {
		_ = saveSettings(s)
		return s
	}
	_ = yaml.Unmarshal(b, &s)
	if s.HTTPPort <= 0 || s.HTTPPort > 65535 {
		s.HTTPPort = 18765
	}
	if s.AIMode == "" {
		s.AIMode = AIModeNormal
	}
	if s.ApprovalTimeoutSecs <= 0 {
		s.ApprovalTimeoutSecs = 300
	}
	if s.AuditRetentionDays < 0 {
		s.AuditRetentionDays = 90
	}
	if s.RedactionTTLDays <= 0 {
		s.RedactionTTLDays = 30
	}
	return s
}

func saveSettings(s Settings) error {
	root, err := homeDir()
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(join(root, settingsFile), b, 0600)
}

func defaultSettings() Settings {
	return Settings{
		Enabled:                   false,
		AutoStart:                 false,
		HTTPPort:                  18765,
		BindLAN:                   false,
		AIMode:                    AIModeNormal,
		AuditRetentionDays:        90,
		ApprovalTimeoutSecs:       300,
		OutboundAllowlistDisabled: false,
		RedactionTTLDays:          30,
	}
}

// Settings MCP / AI 安全设置（对齐 Reeve）
type Settings struct {
	Enabled   bool `yaml:"enabled" json:"enabled"`
	AutoStart bool `yaml:"autoStart" json:"autoStart"` // 应用启动时自动开启 MCP 服务
	HTTPPort  int  `yaml:"httpPort" json:"httpPort"`
	BindLAN   bool `yaml:"bindLan" json:"bindLan"`
	// DefaultPolicy 已废弃，仅兼容旧 YAML；策略以单机 aiPolicy 为准。
	DefaultPolicy string `yaml:"defaultPolicy,omitempty" json:"defaultPolicy,omitempty"`

	// 全局 AI 总开关
	AIMode        string `yaml:"aiMode" json:"aiMode"`                   // normal | armed | emergency
	ArmedUntil    string `yaml:"armedUntil,omitempty" json:"armedUntil"` // RFC3339
	EmergencyStop bool   `yaml:"emergencyStop" json:"emergencyStop"`

	// 审计保留（天；0=永久）
	AuditRetentionDays int `yaml:"auditRetentionDays" json:"auditRetentionDays"`
	// ApprovalTimeoutSecs 审批等待秒数（默认 300，范围 30～3600）
	ApprovalTimeoutSecs int `yaml:"approvalTimeoutSecs,omitempty" json:"approvalTimeoutSecs,omitempty"`

	// 出站白名单（默认启用；设 Disabled=true 关闭）
	OutboundAllowlistDisabled bool     `yaml:"outboundAllowlistDisabled" json:"outboundAllowlistDisabled"`
	OutboundAllowlistEnabled  bool     `yaml:"outboundAllowlistEnabled,omitempty" json:"outboundAllowlistEnabled"` // 兼容旧字段
	OutboundHosts             []string `yaml:"outboundHosts,omitempty" json:"outboundHosts"`

	// 敏感库 TTL
	RedactionTTLDays int `yaml:"redactionTTLDays" json:"redactionTTLDays"`

	// 自定义危险命令正则：命中后升级人工审批（不是永久拦截）
	CustomDangerPatterns []string `yaml:"customDangerPatterns,omitempty" json:"customDangerPatterns"`
}

func (s Settings) ArmedUntilTime() (time.Time, bool) {
	if s.ArmedUntil == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s.ArmedUntil)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
