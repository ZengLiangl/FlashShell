/**
 * Shell 终端临时英文输入：焦点时关闭中文组词，失焦/离开 Shell/窗口失焦后恢复。
 * 仅在设置开启且终端真正持有焦点时生效。
 */
import * as App from '../../wailsjs/go/app/App'

let enabled = true
/** 终端侧期望处于英文态（textarea 持有焦点） */
let desired = false
let leaveTimer = null
let listenersBound = false

const LEAVE_DEBOUNCE_MS = 140

export function setShellAsciiInputEnabled(value) {
  enabled = value !== false
  if (!enabled) {
    desired = false
    void forceLeave()
    return
  }
  if (desired) {
    void syncEnter()
  }
}

export function notifyShellTerminalFocus() {
  desired = true
  cancelLeaveTimer()
  void syncEnter()
}

export function notifyShellTerminalBlur() {
  desired = false
  scheduleLeave()
}

/** 离开 Shell 视图时立即恢复 */
export function notifyLeaveShellMode() {
  desired = false
  void forceLeave()
}

export function ensureShellAsciiInputListeners() {
  if (listenersBound || typeof window === 'undefined') return
  listenersBound = true
  window.addEventListener('blur', onWindowBlur)
  window.addEventListener('focus', onWindowFocus)
}

function onWindowBlur() {
  // 切到其它 App：立刻恢复，避免系统一直停在英文态
  cancelLeaveTimer()
  void forceLeave()
}

function onWindowFocus() {
  if (desired && enabled) {
    void syncEnter()
  }
}

async function syncEnter() {
  if (!enabled || !desired) return
  try {
    await App.ShellAsciiInputEnter()
  } catch {
    // 静默：输入法切换失败不打断终端使用
  }
}

async function forceLeave() {
  cancelLeaveTimer()
  try {
    await App.ShellAsciiInputLeave()
  } catch {
    // ignore
  }
}

function scheduleLeave() {
  cancelLeaveTimer()
  leaveTimer = setTimeout(() => {
    leaveTimer = null
    if (!desired) {
      void forceLeave()
    }
  }, LEAVE_DEBOUNCE_MS)
}

function cancelLeaveTimer() {
  if (leaveTimer != null) {
    clearTimeout(leaveTimer)
    leaveTimer = null
  }
}
