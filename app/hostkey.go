package app

import (
	"errors"
	"fmt"
	"strings"

	"FlashDock/data"
)

// GetKnownHosts 返回已信任主机列表
func (a *App) GetKnownHosts() []data.KnownHostRecord {
	return data.GlobalHostKeyManager().List()
}

// TrustHostKey 持久信任主机密钥（写入 known_hosts）
func (a *App) TrustHostKey(host string, port int, fingerprint string) error {
	return data.GlobalHostKeyManager().Trust(host, port, fingerprint)
}

// TrustHostKeyOnce 仅信任本次会话（内存，应用退出或断开该主机全部连接后失效）
func (a *App) TrustHostKeyOnce(host string, port int, fingerprint string) {
	data.GlobalHostKeyManager().TrustSession(host, port, fingerprint)
}

func (a *App) clearSessionHostKeyTrust(configName string) {
	configName = strings.TrimSpace(configName)
	if configName == "" {
		return
	}
	machineConfig := a.configManager.GetMachine(configName)
	if machineConfig == nil {
		return
	}
	sensitive, err := machineConfig.GetSensitiveData()
	if err != nil || sensitive == nil {
		return
	}
	data.GlobalHostKeyManager().ClearSessionTrust(sensitive.Host, sensitive.Port)
}

// RemoveKnownHost 移除已信任主机
func (a *App) RemoveKnownHost(host string, port int) error {
	return data.GlobalHostKeyManager().Remove(host, port)
}

// ExportKnownHosts 导出 known_hosts JSON
func (a *App) ExportKnownHosts() (string, error) {
	raw, err := data.GlobalHostKeyManager().ExportJSON()
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ImportKnownHosts 导入 known_hosts JSON
func (a *App) ImportKnownHosts(jsonData string) (int, error) {
	return data.GlobalHostKeyManager().ImportJSON([]byte(jsonData))
}

// IsHostKeyUnknownError 判断是否为未知主机密钥错误
func IsHostKeyUnknownError(err error) (*data.HostKeyUnknownError, bool) {
	var hk *data.HostKeyUnknownError
	if errors.As(err, &hk) {
		return hk, true
	}
	// 兼容包装错误
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "未知主机密钥") {
			return nil, true
		}
	}
	return nil, false
}

// FormatHostKeyError 格式化主机密钥错误供前端展示
func FormatHostKeyError(err error) string {
	if hk, ok := IsHostKeyUnknownError(err); ok && hk != nil {
		return fmt.Sprintf("%s:%d|%s", hk.Host, hk.Port, hk.Fingerprint)
	}
	if err != nil {
		return err.Error()
	}
	return ""
}
