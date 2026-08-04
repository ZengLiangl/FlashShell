package define

import (
	"strings"

	"FlashDock/crypto"
)

// MachineProxyOverride 每主机代理覆盖（inherit=全局 | none=直连 | manual=本机配置）
type MachineProxyOverride struct {
	Mode              string `yaml:"mode,omitempty" json:"mode"` // inherit | none | manual
	Type              string `yaml:"type,omitempty" json:"type"` // http | socks
	Host              string `yaml:"host,omitempty" json:"host"`
	Port              int    `yaml:"port,omitempty" json:"port"`
	User              string `yaml:"user,omitempty" json:"user"`
	EncryptedPassword string `yaml:"encryptedPassword,omitempty" json:"-"`
	Password          string `yaml:"-" json:"password"`
}

// NormalizeMachineProxyOverride 规范化每主机代理字段
func NormalizeMachineProxyOverride(p *MachineProxyOverride) {
	if p == nil {
		return
	}
	mode := strings.ToLower(strings.TrimSpace(p.Mode))
	switch mode {
	case "none", "manual":
		p.Mode = mode
	default:
		p.Mode = "inherit"
	}
	typ := strings.ToLower(strings.TrimSpace(p.Type))
	if typ != "socks" {
		p.Type = "http"
	}
	p.Host = strings.TrimSpace(p.Host)
	p.User = strings.TrimSpace(p.User)
	if p.Port < 0 || p.Port > 65535 {
		p.Port = 0
	}
	if p.Mode != "manual" {
		p.User = ""
		p.Password = ""
	}
}

// SetMachineProxyPassword 加密代理密码
func (p *MachineProxyOverride) SetMachineProxyPassword(password string) error {
	if p == nil {
		return nil
	}
	p.Password = password
	if password == "" {
		p.EncryptedPassword = ""
		return nil
	}
	encrypted, err := crypto.EncryptSensitiveData(&crypto.SensitiveData{
		Name:     "machine-proxy",
		Username: p.User,
		Password: password,
	})
	if err != nil {
		return err
	}
	p.EncryptedPassword = encrypted
	return nil
}

// GetMachineProxyPassword 读取明文代理密码
func (p *MachineProxyOverride) GetMachineProxyPassword() (string, error) {
	if p == nil {
		return "", nil
	}
	if p.Password != "" {
		return p.Password, nil
	}
	if p.EncryptedPassword == "" {
		return "", nil
	}
	data, err := crypto.DecryptSensitiveData(p.EncryptedPassword)
	if err != nil {
		return "", err
	}
	p.Password = data.Password
	return p.Password, nil
}

// HydrateMachineProxyPassword 解密到 Password 供表单展示
func (p *MachineProxyOverride) HydrateMachineProxyPassword() {
	if p == nil {
		return
	}
	pw, err := p.GetMachineProxyPassword()
	if err != nil {
		p.Password = ""
		return
	}
	p.Password = pw
}

// PrepareMachineProxyPasswordForSave 保存前加密密码
func (p *MachineProxyOverride) PrepareMachineProxyPasswordForSave() error {
	if p == nil {
		return nil
	}
	return p.SetMachineProxyPassword(p.Password)
}
