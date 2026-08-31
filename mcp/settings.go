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
	if s.DefaultPolicy == "" {
		s.DefaultPolicy = PolicyTrusted
	}
	if s.AIMode == "" {
		s.AIMode = AIModeNormal
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
		BindLAN:                  false,
		DefaultPolicy:            PolicyTrusted,
		AIMode:                   AIModeNormal,
		AuditRetentionDays:       90,
		OutboundAllowlistDisabled: false,
		RedactionTTLDays:         30,
	}
}

// Settings MCP / AI 安全设置（对齐 Reeve）
type Settings struct {
	Enabled       bool   `yaml:"enabled" json:"enabled"`
	AutoStart     bool   `yaml:"autoStart" json:"autoStart"` // 应用启动时自动开启 MCP 服务
	HTTPPort      int    `yaml:"httpPort" json:"httpPort"`
	BindLAN       bool   `yaml:"bindLan" json:"bindLan"`
	DefaultPolicy string `yaml:"defaultPolicy" json:"defaultPolicy"`

	// 全局 AI 总开关
	AIMode         string `yaml:"aiMode" json:"aiMode"`                   // normal | armed | emergency
	ArmedUntil     string `yaml:"armedUntil,omitempty" json:"armedUntil"` // RFC3339
	EmergencyStop  bool   `yaml:"emergencyStop" json:"emergencyStop"`

	// 审计保留（天；0=永久）
	AuditRetentionDays int `yaml:"auditRetentionDays" json:"auditRetentionDays"`

	// 出站白名单（默认启用；设 Disabled=true 关闭）
	OutboundAllowlistDisabled bool     `yaml:"outboundAllowlistDisabled" json:"outboundAllowlistDisabled"`
	OutboundAllowlistEnabled  bool     `yaml:"outboundAllowlistEnabled,omitempty" json:"outboundAllowlistEnabled"` // 兼容旧字段
	OutboundHosts             []string `yaml:"outboundHosts,omitempty" json:"outboundHosts"`

	// 敏感库 TTL
	RedactionTTLDays int `yaml:"redactionTTLDays" json:"redactionTTLDays"`

	// 自定义危险命令正则（任何档位永拦或升级审批，见 matchCustomDanger）
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
