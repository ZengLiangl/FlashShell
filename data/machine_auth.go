package data

import (
	"fmt"
	"strings"

	"FlashDock/define"
)

// ResolveMachineAuth 若机器引用全局身份，将帐号用户名/密码/密钥覆盖到运行时（不落盘）
func (gcm *GlobalConfigManager) ResolveMachineAuth(machine *define.Machine) error {
	if gcm == nil || machine == nil {
		return nil
	}
	identityID := strings.TrimSpace(machine.IdentityID)
	if identityID == "" {
		return nil
	}
	if gcm.config == nil {
		if _, err := gcm.LoadGlobalConfig(); err != nil {
			return err
		}
	}
	account := gcm.GetGlobalAccountByID(identityID)
	if account == nil {
		return fmt.Errorf("未找到身份: %s", identityID)
	}
	password, err := account.GetPassword()
	if err != nil {
		return err
	}
	if err := machine.OverlaySensitiveFields(account.User, password); err != nil {
		return err
	}
	if k := strings.TrimSpace(account.KeyFile); k != "" {
		machine.KeyFile = k
	}
	return nil
}

// MachineForConnect 返回用于连接的机器副本并应用身份覆盖
func (gcm *GlobalConfigManager) MachineForConnect(machine *define.Machine) (*define.Machine, error) {
	if machine == nil {
		return nil, fmt.Errorf("机器配置为空")
	}
	copy := *machine
	copy.ClearSensitiveData()
	if err := gcm.ResolveMachineAuth(&copy); err != nil {
		return nil, err
	}
	return &copy, nil
}
