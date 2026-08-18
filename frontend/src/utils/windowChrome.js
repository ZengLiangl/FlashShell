import { ToggleWindowMaximised } from '../../wailsjs/go/app/App'

const INTERACTIVE_SEL =
  'button, a, input, textarea, select, option, label, .el-button, .el-dropdown, .el-input, .el-select, .el-checkbox, .el-switch, .el-tooltip__trigger, .session-tab, .win-btn, .chrome-icon-btn, .mode-switcher-host, .conn-toggle, .icon-actions, .home-search, .home-header-actions, .rail-item, .add-session-wrap, .app-chrome-icons, .window-controls, .terminal-actions'

const DBLCLICK_MS = 380
const DBLCLICK_MOVE_PX = 8

export function isChromeInteractive(el) {
  if (!(el instanceof Element)) return false
  return !!el.closest(INTERACTIVE_SEL)
}

function eventElement(target) {
  if (target instanceof Element) return target
  return target && target.parentElement instanceof Element ? target.parentElement : null
}

function isChromeDragRegion(el) {
  const node = eventElement(el)
  if (!node || isChromeInteractive(node)) return false
  if (typeof window.getComputedStyle !== 'function') return false
  const val = window.getComputedStyle(node).getPropertyValue('--wails-draggable').trim()
  return val === 'drag'
}

function cancelWailsDrag() {
  if (window.wails?.flags) window.wails.flags.shouldDrag = false
}

/** 与后端 applyStartupFullscreen 相同：退出独占全屏后最大化或还原 */
let toggling = false
export function toggleChromeWindowMaximise() {
  if (toggling) return Promise.resolve()
  toggling = true
  setTimeout(() => { toggling = false }, 400)
  cancelWailsDrag()
  return ToggleWindowMaximised().catch(() => {})
}

export function onChromeTitleDblActivate(event) {
  if (!event) return
  if (event.type === 'mousedown' && event.button !== 0) return
  if (isChromeInteractive(event.target)) return
  event.preventDefault?.()
  toggleChromeWindowMaximise()
}

let lastAt = 0
let lastX = 0
let lastY = 0
let pendingDragGuard = false
let downX = 0
let downY = 0
let downAt = 0

/** 拖拽区域上 dblclick 常被原生吃掉，用两次 mousedown 识别双击 */
export function onChromeTitlePointerDown(event) {
  if (!event || event.button !== 0) return
  if (isChromeInteractive(event.target)) return
  if (event.detail >= 2) {
    lastAt = 0
    pendingDragGuard = false
    event.preventDefault()
    event.stopPropagation()
    toggleChromeWindowMaximise()
    return
  }
  const now = Date.now()
  const dx = Math.abs((event.clientX || 0) - lastX)
  const dy = Math.abs((event.clientY || 0) - lastY)
  if (now - lastAt < DBLCLICK_MS && dx < DBLCLICK_MOVE_PX && dy < DBLCLICK_MOVE_PX) {
    lastAt = 0
    pendingDragGuard = false
    event.preventDefault()
    event.stopPropagation()
    toggleChromeWindowMaximise()
    return
  }
  lastAt = now
  lastX = event.clientX || 0
  lastY = event.clientY || 0
}

function onCaptureMouseDown(event) {
  if (!event || event.button !== 0) return
  if (!isChromeDragRegion(event.target)) {
    pendingDragGuard = false
    return
  }
  if (event.detail >= 2) {
    pendingDragGuard = false
    cancelWailsDrag()
    event.preventDefault()
    event.stopPropagation()
    toggleChromeWindowMaximise()
    return
  }
  pendingDragGuard = true
  downX = event.clientX || 0
  downY = event.clientY || 0
  downAt = Date.now()
}

function onCaptureMouseMove(event) {
  if (!pendingDragGuard) return
  const dx = Math.abs((event.clientX || 0) - downX)
  const dy = Math.abs((event.clientY || 0) - downY)
  if (Date.now() - downAt < DBLCLICK_MS && dx < DBLCLICK_MOVE_PX && dy < DBLCLICK_MOVE_PX) {
    // 吞掉双击间隙里的微移，避免 Wails 把最大化窗口先「拖还原」，第二次点击再最大化回去
    event.stopImmediatePropagation()
    return
  }
  pendingDragGuard = false
}

function onCaptureMouseUp() {
  pendingDragGuard = false
}

function onCaptureDblClick(event) {
  if (!isChromeDragRegion(event.target)) return
  // 最大化切换已在第二次 mousedown 处理；阻止 dblclick 再触发一次把窗口最大化回去
  event.preventDefault()
  event.stopPropagation()
}

if (typeof window !== 'undefined' && !window.__flashdockChromeDblclickBound) {
  window.__flashdockChromeDblclickBound = true
  window.addEventListener('mousedown', onCaptureMouseDown, true)
  window.addEventListener('mousemove', onCaptureMouseMove, true)
  window.addEventListener('mouseup', onCaptureMouseUp, true)
  window.addEventListener('dblclick', onCaptureDblClick, true)
}
