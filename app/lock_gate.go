package app

import (
	"FlashDock/crypto"
	"FlashDock/machine"
)

// vaultGuardedMethods 锁定时拒绝的 Wails 绑定方法名（与前端 safeInvoke 清单对齐）。
// 覆盖：测试连接/终端/SFTP、敏感凭据读取、配置改写、主密码变更、端口转发等。
// 允许：列表元数据、配置只读、审计查看、Unlock / 忘记主密码重置、本地终端、Vault 状态查询。
var vaultGuardedMethods = []string{
	// 连接 / 终端 / SFTP
	"TestMachineConnection",
	"TestMachineDraftConnection",
	"ConnectShell",
	"ReconnectShell",
	"OpenMachineInNewWindow",
	"ListShellFiles",
	"DeleteShellFile",
	"GetShellRemoteHome",
	"GetShellRemotePwd",
	"ReadShellRemoteFile",
	"SaveShellRemoteFile",
	"OpenShellRemoteFileExternal",
	"OpenShellRemoteFileWithApp",
	"OpenShellRemoteFileSystemDefault",
	"MkdirShellRemotePath",
	"RenameShellRemotePath",
	"ChmodShellRemotePath",
	"CopyShellRemotePath",
	"MoveShellRemotePath",
	"CheckShellUploadConflict",
	"SendShellCd",
	"StartShellDownload",
	"StartShellUpload",
	"StartShellFolderSync",
	"StartShellCopyToOther",
	"AddShellTemporaryTunnel",
	"StartPortForward",
	"SavePortForwards",
	"ExecuteSubProject",
	"ExecuteCommand",
	// 敏感凭据读取 / 写入
	"GetMachineSensitiveData",
	"SetMachineSensitiveData",
	"ClearMachineSensitiveData",
	// 配置修改（含机器 / 帐号 / 导入）
	"SaveConfig",
	"SaveGlobalConfig",
	"SaveSystemSettings",
	"SaveGlobalAccounts",
	"SaveGlobalAccountsFromDTO",
	"AddMachine",
	"AddMachineWithEvent",
	"UpdateMachine",
	"UpdateMachineWithEvent",
	"DeleteMachine",
	"DeleteMachineWithEvent",
	"CreateMachine",
	"AddMachineGroup",
	"RenameMachineGroup",
	"DeleteMachineGroup",
	"UpdateMachineGroup",
	"SaveMachineGroupDefaults",
	"AddWorkPath",
	"AddWorkPathWithEvent",
	"UpdateWorkPath",
	"UpdateWorkPathWithEvent",
	"DeleteWorkPath",
	"DeleteWorkPathWithEvent",
	"ImportMachineTemplate",
	"ImportMachineTemplateFromFile",
	"ImportXshellPick",
	"ImportXshellFromFile",
	"ImportXshellFromFolder",
	"ImportFinalShellPick",
	"ImportOpenSSHConfigPick",
	"ImportOpenSSHConfigDefault",
	"ImportMachinesCSVPick",
	"ImportPuttyPick",
	"ImportMobaXtermPick",
	"ImportSecureCRTPick",
	"SaveMCPSettings",
	"GenerateMCPToken",
	"IssueMCPToken",
	"InstallMCPClient",
	"InstallMCPClientWith",
	"InstallCursorMCP",
	"SaveMCPCustomDangerPatterns",
	"AddMCPOutboundHost",
	// 主密码 / 凭据库变更（忘记主密码重置除外）
	"SetVaultMasterPassword",
	"ChangeVaultMasterPassword",
	"DisableVaultMasterPassword",
	"SetVaultIdleLockMinutes",
	"ResetVaultReencrypt",
	"ResetVaultWipeSecrets",
}

// ListVaultGuardedMethods 供前端 safeInvoke / 测试对齐守护清单
func (a *App) ListVaultGuardedMethods() []string {
	out := make([]string, len(vaultGuardedMethods))
	copy(out, vaultGuardedMethods)
	return out
}

func (a *App) requireUnlocked() error {
	if crypto.CheckIdleLock() {
		a.onVaultLocked()
	}
	if crypto.IsLocked() {
		return friendlyVaultErr(crypto.ErrLocked)
	}
	return nil
}

// onVaultLocked 锁定副作用：清内存凭据缓存、断开远程 Shell / 辅助通道（本地终端保留）
func (a *App) onVaultLocked() {
	a.clearCredentialCaches()
	a.disconnectRemoteSessionsOnLock()
	a.emitVaultStatus()
}

func (a *App) clearCredentialCaches() {
	if a.configManager == nil {
		return
	}
	gc, err := a.configManager.GetGlobalConfig()
	if err != nil || gc == nil {
		return
	}
	for i := range gc.Machines {
		gc.Machines[i].ClearSensitiveData()
	}
}

func (a *App) disconnectRemoteSessionsOnLock() {
	sessions := a.listAllShellSessions()
	for _, st := range sessions {
		id := st.MachineName
		if id == "" {
			continue
		}
		if machine.IsLocalShellID(id) || st.Kind == "local" {
			continue
		}
		_ = a.DisconnectShell(id)
	}
}
