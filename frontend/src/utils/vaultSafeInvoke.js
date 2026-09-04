/**
 * 凭据库锁定状态 + Wails safeInvoke 拦截器。
 * 与后端 ListVaultGuardedMethods / vaultGuardedMethods 对齐：锁定时拒绝凭据相关调用。
 */
import { reactive } from 'vue'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { GetVaultStatus, ListVaultGuardedMethods } from '../../wailsjs/go/app/App'

const state = reactive({
  hasMasterPassword: false,
  unlocked: true,
  ready: false,
})

const LOCKED_MSG = '凭据库已锁定，请先解锁'

/** 与 app/lock_gate.go 同步的兜底清单（后端 List 不可用时） */
const FALLBACK_GUARDED = [
  'TestMachineConnection', 'TestMachineDraftConnection', 'ConnectShell', 'ReconnectShell',
  'OpenMachineInNewWindow', 'ListShellFiles', 'DeleteShellFile', 'GetShellRemoteHome',
  'GetShellRemotePwd', 'ReadShellRemoteFile', 'SaveShellRemoteFile',
  'OpenShellRemoteFileExternal', 'OpenShellRemoteFileWithApp', 'OpenShellRemoteFileSystemDefault',
  'MkdirShellRemotePath', 'RenameShellRemotePath', 'ChmodShellRemotePath', 'CopyShellRemotePath',
  'MoveShellRemotePath', 'CheckShellUploadConflict', 'SendShellCd', 'StartShellDownload',
  'StartShellUpload', 'StartShellFolderSync', 'StartShellCopyToOther', 'AddShellTemporaryTunnel',
  'StartPortForward', 'SavePortForwards', 'ExecuteSubProject', 'ExecuteCommand',
  'GetMachineSensitiveData', 'SetMachineSensitiveData', 'ClearMachineSensitiveData',
  'SaveConfig', 'SaveGlobalConfig', 'SaveSystemSettings', 'SaveGlobalAccounts',
  'SaveGlobalAccountsFromDTO', 'AddMachine', 'AddMachineWithEvent', 'UpdateMachine',
  'UpdateMachineWithEvent', 'DeleteMachine', 'DeleteMachineWithEvent', 'CreateMachine',
  'AddMachineGroup', 'RenameMachineGroup', 'DeleteMachineGroup', 'UpdateMachineGroup',
  'SaveMachineGroupDefaults', 'AddWorkPath', 'AddWorkPathWithEvent', 'UpdateWorkPath',
  'UpdateWorkPathWithEvent', 'DeleteWorkPath', 'DeleteWorkPathWithEvent',
  'ImportMachineTemplate', 'ImportMachineTemplateFromFile', 'ImportXshellPick',
  'ImportXshellFromFile', 'ImportXshellFromFolder', 'ImportFinalShellPick',
  'ImportOpenSSHConfigPick', 'ImportOpenSSHConfigDefault', 'ImportMachinesCSVPick',
  'ImportPuttyPick', 'ImportMobaXtermPick', 'ImportSecureCRTPick', 'SaveMCPSettings',
  'GenerateMCPToken', 'IssueMCPToken', 'InstallMCPClient', 'InstallMCPClientWith',
  'InstallCursorMCP', 'SaveMCPCustomDangerPatterns', 'AddMCPOutboundHost',
  'RevealMCPSensitive', 'DiscardMCPSensitive', 'PromoteMCPSensitive', 'SaveMCPRedactRules',
  'SetVaultMasterPassword', 'ChangeVaultMasterPassword', 'DisableVaultMasterPassword',
  'SetVaultIdleLockMinutes', 'ResetVaultReencrypt', 'ResetVaultWipeSecrets',
]

export function isVaultLocked() {
  return !!(state.hasMasterPassword && !state.unlocked)
}

export function useVaultLockState() {
  return state
}

function applyStatus(st) {
  if (!st) return
  state.hasMasterPassword = !!st.hasMasterPassword
  state.unlocked = !!st.unlocked
  state.ready = true
}

async function refreshStatus() {
  try {
    applyStatus(await GetVaultStatus())
  } catch {
    state.ready = true
  }
}

function wrapGuarded(api, names) {
  const set = new Set(names || FALLBACK_GUARDED)
  for (const name of set) {
    const orig = api[name]
    if (typeof orig !== 'function') continue
    if (orig.__vaultGuarded) continue
    const wrapped = function (...args) {
      if (isVaultLocked()) {
        return Promise.reject(new Error(LOCKED_MSG))
      }
      return orig.apply(this, args)
    }
    wrapped.__vaultGuarded = true
    api[name] = wrapped
  }
}

/**
 * 安装全局拦截：包装 window.go.app.App 上的守护方法。
 * 同时订阅 vault:status，供 UI 闸门使用。
 */
export async function installVaultSafeInvoke() {
  await refreshStatus()
  EventsOn('vault:status', applyStatus)

  let guarded = FALLBACK_GUARDED
  try {
    const list = await ListVaultGuardedMethods()
    if (Array.isArray(list) && list.length) guarded = list
  } catch {
    /* 绑定未刷新时用兜底清单 */
  }

  const tryWrap = () => {
    const api = window?.go?.app?.App
    if (!api) return false
    wrapGuarded(api, guarded)
    return true
  }
  if (!tryWrap()) {
    const t = setInterval(() => {
      if (tryWrap()) clearInterval(t)
    }, 200)
    setTimeout(() => clearInterval(t), 15000)
  }

  // 解锁后活动心跳：任意指针/键盘重置空闲计时（由后端 VaultTouchActivity）
  let lastTouch = 0
  const touch = () => {
    if (isVaultLocked()) return
    const now = Date.now()
    if (now - lastTouch < 30000) return
    lastTouch = now
    try {
      window.go?.app?.App?.VaultTouchActivity?.()
    } catch { /* ignore */ }
  }
  window.addEventListener('pointerdown', touch, { passive: true })
  window.addEventListener('keydown', touch, { passive: true })
}
