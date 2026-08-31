package app

import (
	"fmt"

	"FlashDock/crypto"
	"FlashDock/data"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetVaultStatus 凭据库状态（KEK→DEK）
func (a *App) GetVaultStatus() crypto.Status {
	wasUnlocked := !crypto.IsLocked()
	lockedNow := crypto.CheckIdleLock()
	st := crypto.GetStatus()
	if lockedNow && wasUnlocked {
		a.onVaultLocked()
		st = crypto.GetStatus()
	}
	return st
}

// UnlockVault 解锁凭据库
func (a *App) UnlockVault(masterPassword string) error {
	if err := crypto.Unlock(masterPassword); err != nil {
		return friendlyVaultErr(err)
	}
	if err := a.migrateLegacyCredentials(); err != nil {
		data.AppLogf("解锁后遗留凭据迁移: %v", err)
	}
	a.emitVaultStatus()
	return nil
}

// LockVault 锁定（清空内存 DEK）
func (a *App) LockVault() {
	crypto.Lock()
	a.onVaultLocked()
}

// VaultTouchActivity 用户活动，重置空闲锁定计时
func (a *App) VaultTouchActivity() {
	crypto.TouchActivity()
	if crypto.CheckIdleLock() {
		a.onVaultLocked()
	}
}

// SetVaultMasterPassword 启用主密码
func (a *App) SetVaultMasterPassword(password, confirm string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if err := crypto.SetMasterPassword(password, confirm); err != nil {
		return friendlyVaultErr(err)
	}
	a.emitVaultStatus()
	return nil
}

// ChangeVaultMasterPassword 修改主密码
func (a *App) ChangeVaultMasterPassword(oldPass, newPass, confirm string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if err := crypto.ChangeMasterPassword(oldPass, newPass, confirm); err != nil {
		return friendlyVaultErr(err)
	}
	a.emitVaultStatus()
	return nil
}

// DisableVaultMasterPassword 关闭主密码
func (a *App) DisableVaultMasterPassword(password string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if err := crypto.DisableMasterPassword(password); err != nil {
		return friendlyVaultErr(err)
	}
	a.emitVaultStatus()
	return nil
}

// SetVaultIdleLockMinutes 空闲自动锁定分钟数（0 关闭）
func (a *App) SetVaultIdleLockMinutes(minutes int) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if err := crypto.SetIdleLockMinutes(minutes); err != nil {
		return err
	}
	a.emitVaultStatus()
	return nil
}

// ResetVaultReencrypt 生成新 DEK 并重加密全部凭据
func (a *App) ResetVaultReencrypt(masterPassword string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	err := crypto.ResetReencrypt(masterPassword, func(oldDecrypt, newEncrypt func(string) (string, error)) error {
		return a.reencryptAllCredentials(oldDecrypt, newEncrypt)
	})
	if err != nil {
		return friendlyVaultErr(err)
	}
	a.emitVaultStatus()
	return nil
}

// ResetVaultWipeSecrets 清空所有凭据密文并换新 DEK（保留服务器列表元数据）
func (a *App) ResetVaultWipeSecrets(masterPassword string) error {
	if err := a.requireUnlocked(); err != nil {
		return err
	}
	if err := a.wipeAllCredentialSecrets(); err != nil {
		return err
	}
	if err := crypto.ResetWipeSecrets(masterPassword); err != nil {
		return friendlyVaultErr(err)
	}
	a.emitVaultStatus()
	return nil
}

// ResetVaultForgotMasterPassword 忘记主密码：清空全部凭据密文，换新 DEK，关闭主密码（无需旧密码）
func (a *App) ResetVaultForgotMasterPassword() error {
	if err := a.wipeAllCredentialSecrets(); err != nil {
		return err
	}
	if err := crypto.ResetForgotMasterPassword(); err != nil {
		return friendlyVaultErr(err)
	}
	a.emitVaultStatus()
	return nil
}

func (a *App) emitVaultStatus() {
	if a.ctx == nil {
		return
	}
	wailsRuntime.EventsEmit(a.ctx, "vault:status", crypto.GetStatus())
}

func friendlyVaultErr(err error) error {
	if err == nil {
		return nil
	}
	switch err {
	case crypto.ErrLocked:
		return fmt.Errorf("凭据库已锁定，请先解锁")
	case crypto.ErrBadPassword:
		return fmt.Errorf("主密码错误")
	case crypto.ErrRateLimited:
		return fmt.Errorf("尝试次数过多，请稍后再试")
	case crypto.ErrHasMaster:
		return fmt.Errorf("已启用主密码，请先关闭或修改")
	case crypto.ErrNoMaster:
		return fmt.Errorf("未启用主密码")
	default:
		return err
	}
}

func (a *App) initCredentialVault() {
	if err := crypto.InitVault(); err != nil {
		data.AppLogf("凭据库初始化失败: %v", err)
		return
	}
	if crypto.IsLocked() {
		data.AppLogf("凭据库已锁定（主密码模式），等待解锁")
		return
	}
	if err := a.migrateLegacyCredentials(); err != nil {
		data.AppLogf("遗留凭据迁移失败: %v", err)
	}
}

func (a *App) migrateLegacyCredentials() error {
	if a.configManager == nil {
		return nil
	}
	gc, err := a.configManager.GetGlobalConfig()
	if err != nil || gc == nil {
		return err
	}
	changed := false
	for i := range gc.Machines {
		m := &gc.Machines[i]
		if m.EncryptedData == "" {
			continue
		}
		out, ok, err := crypto.MigrateLegacyCiphertext(m.EncryptedData)
		if err != nil {
			return err
		}
		if ok {
			m.EncryptedData = out
			changed = true
		}
		if m.ProxyOverride != nil && m.ProxyOverride.EncryptedPassword != "" {
			out, ok, err := crypto.MigrateLegacyCiphertext(m.ProxyOverride.EncryptedPassword)
			if err != nil {
				return err
			}
			if ok {
				m.ProxyOverride.EncryptedPassword = out
				changed = true
			}
		}
	}
	for i := range gc.GlobalAccounts {
		acc := &gc.GlobalAccounts[i]
		if acc.EncryptedPassword != "" {
			out, ok, err := crypto.MigrateLegacyCiphertext(acc.EncryptedPassword)
			if err != nil {
				return err
			}
			if ok {
				acc.EncryptedPassword = out
				changed = true
			}
		}
		if acc.EncryptedKeyPassphrase != "" {
			out, ok, err := crypto.MigrateLegacyCiphertext(acc.EncryptedKeyPassphrase)
			if err != nil {
				return err
			}
			if ok {
				acc.EncryptedKeyPassphrase = out
				changed = true
			}
		}
	}
	if gc.ProxySettings.EncryptedPassword != "" {
		out, ok, err := crypto.MigrateLegacyCiphertext(gc.ProxySettings.EncryptedPassword)
		if err != nil {
			return err
		}
		if ok {
			gc.ProxySettings.EncryptedPassword = out
			changed = true
		}
	}
	if !changed {
		return nil
	}
	if err := a.configManager.SaveGlobalConfig(gc); err != nil {
		return err
	}
	data.AppLogf("已将遗留硬编码密钥密文迁移到 OS keyring DEK")
	return nil
}

func (a *App) reencryptAllCredentials(oldDecrypt, newEncrypt func(string) (string, error)) error {
	gc, err := a.configManager.GetGlobalConfig()
	if err != nil || gc == nil {
		return err
	}
	re := func(enc *string) error {
		if enc == nil || *enc == "" {
			return nil
		}
		plain, err := oldDecrypt(*enc)
		if err != nil {
			return nil // 解不开则跳过保留
		}
		out, err := newEncrypt(plain)
		if err != nil {
			return err
		}
		*enc = out
		return nil
	}
	for i := range gc.Machines {
		m := &gc.Machines[i]
		if err := re(&m.EncryptedData); err != nil {
			return err
		}
		if m.ProxyOverride != nil {
			if err := re(&m.ProxyOverride.EncryptedPassword); err != nil {
				return err
			}
		}
	}
	for i := range gc.GlobalAccounts {
		acc := &gc.GlobalAccounts[i]
		if err := re(&acc.EncryptedPassword); err != nil {
			return err
		}
		if err := re(&acc.EncryptedKeyPassphrase); err != nil {
			return err
		}
	}
	if err := re(&gc.ProxySettings.EncryptedPassword); err != nil {
		return err
	}
	return a.configManager.SaveGlobalConfig(gc)
}

func (a *App) wipeAllCredentialSecrets() error {
	gc, err := a.configManager.GetGlobalConfig()
	if err != nil || gc == nil {
		return err
	}
	for i := range gc.Machines {
		gc.Machines[i].EncryptedData = ""
		if gc.Machines[i].ProxyOverride != nil {
			gc.Machines[i].ProxyOverride.EncryptedPassword = ""
		}
	}
	for i := range gc.GlobalAccounts {
		gc.GlobalAccounts[i].EncryptedPassword = ""
		gc.GlobalAccounts[i].EncryptedKeyPassphrase = ""
	}
	gc.ProxySettings.EncryptedPassword = ""
	return a.configManager.SaveGlobalConfig(gc)
}
