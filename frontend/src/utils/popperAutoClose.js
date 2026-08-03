/**
 * 1) 下拉面板（Dropdown / Select / Autocomplete）鼠标离开后自动关闭。
 * 2) 按钮上的 hover Tooltip：点击后立即关闭，并兜底清除「loading/disabled 吃掉 mouseleave」后的残留。
 */
const SELECTORS = [
  '.el-dropdown__popper',
  '.el-select__popper',
  '.el-autocomplete__popper',
  '.el-cascader__dropdown',
].join(',')

const MENU_TRIGGER_ANCESTOR =
  '.el-dropdown, .el-select, .el-cascader, .el-autocomplete, .el-popconfirm'

const TOOLTIP_POPPER = '.el-popper.el-tooltip, .el-popper.is-dark, .el-popper.is-light'

const HIDE_DELAY_MS = 160
/** 须大于 Element Plus Tooltip 默认 hideAfter(200)，避免移向气泡途中被误关 */
const ORPHAN_HIDE_DELAY_MS = 240
const bound = new WeakSet()

function isSelectPopper(popper) {
  return (
    popper.classList.contains('el-select__popper') ||
    !!popper.querySelector('.el-select-dropdown')
  )
}

function isDropdownPopper(popper) {
  return (
    popper.classList.contains('el-dropdown__popper') ||
    !!popper.querySelector('.el-dropdown-menu')
  )
}

function isMenuPopper(popper) {
  return (
    isSelectPopper(popper) ||
    isDropdownPopper(popper) ||
    popper.classList.contains('el-autocomplete__popper') ||
    popper.classList.contains('el-cascader__dropdown') ||
    !!popper.closest('.el-popconfirm') ||
    !!popper.querySelector('.el-popconfirm') ||
    !!popper.querySelector('.el-cascader-menu')
  )
}

function isVisible(el) {
  if (!el?.isConnected) return false
  const style = window.getComputedStyle(el)
  if (style.display === 'none' || style.visibility === 'hidden') return false
  const rect = el.getBoundingClientRect()
  return rect.width > 0 && rect.height > 0
}

function clickOutside() {
  const opts = { bubbles: true, cancelable: true, view: window, composed: true }
  const target = document.documentElement
  target.dispatchEvent(new MouseEvent('pointerdown', { ...opts, clientX: 0, clientY: 0 }))
  target.dispatchEvent(new MouseEvent('mousedown', { ...opts, clientX: 0, clientY: 0 }))
}

function findExpandedSelectInput(popper) {
  const listbox = popper.querySelector('[role="listbox"]')
  const id = listbox?.getAttribute('id') || popper.getAttribute('id')
  if (id) {
    try {
      const input = document.querySelector(`input[aria-controls="${CSS.escape(id)}"]`)
      if (input) return input
    } catch {
      const input = document.querySelector(`input[aria-controls="${id}"]`)
      if (input) return input
    }
  }
  return document.querySelector('.el-select input[aria-expanded="true"]')
}

/** 沿 DOM / Vue parent 查找 blur / handleClose */
function callVueClose(startEl) {
  if (!startEl) return false
  const seen = new Set()

  const tryInst = (inst) => {
    while (inst && !seen.has(inst)) {
      seen.add(inst)
      const bags = [inst.exposed, inst.ctx, inst.setupState]
      for (const bag of bags) {
        if (!bag) continue
        if (typeof bag.handleClose === 'function') {
          bag.handleClose()
          return true
        }
        if (typeof bag.blur === 'function') {
          bag.blur()
          return true
        }
        if (typeof bag.hide === 'function') {
          bag.hide()
          return true
        }
        if (typeof bag.onClose === 'function') {
          bag.onClose()
          return true
        }
      }
      // tooltip / popper 内部
      const popperRef = inst.exposed?.popperRef || inst.setupState?.popperRef
      const popper = popperRef?.value ?? popperRef
      if (popper && typeof popper.onClose === 'function') {
        popper.onClose()
        return true
      }
      inst = inst.parent
    }
    return false
  }

  let el = startEl
  while (el) {
    if (tryInst(el.__vueParentComponent)) return true
    el = el.parentElement
  }
  return false
}

function closeSelect(popper) {
  const input = findExpandedSelectInput(popper)
  const selectRoot = input?.closest?.('.el-select') || null

  if (selectRoot && callVueClose(selectRoot)) return
  if (input && callVueClose(input)) return

  clickOutside()
  if (input && typeof input.blur === 'function') input.blur()
}

function closeDropdown(popper) {
  const menu = popper.querySelector('.el-dropdown-menu')
  if (menu && callVueClose(menu)) return
  if (callVueClose(popper)) return

  // 通过当前展开的触发器找到 .el-dropdown 根节点
  const openTriggers = document.querySelectorAll(
    '.el-dropdown [aria-expanded="true"], .el-tooltip__trigger[aria-expanded="true"]',
  )
  for (const t of openTriggers) {
    const root = t.closest('.el-dropdown')
    if (root && callVueClose(root)) return
  }

  // 再尝试点触发器收起
  for (const t of openTriggers) {
    try {
      t.click()
      return
    } catch {
      /* ignore */
    }
  }

  clickOutside()
}

function closePopper(popper) {
  if (!isVisible(popper)) return

  if (isSelectPopper(popper)) {
    closeSelect(popper)
    return
  }
  if (isDropdownPopper(popper)) {
    closeDropdown(popper)
    return
  }
  clickOutside()
}

function bindPopper(popper) {
  if (!popper || bound.has(popper)) return
  bound.add(popper)

  let timer = null
  popper.addEventListener('mouseenter', () => {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  })
  popper.addEventListener('mouseleave', (e) => {
    const related = e.relatedTarget
    if (related instanceof Node && popper.contains(related)) return

    if (timer) clearTimeout(timer)
    timer = setTimeout(() => {
      timer = null
      closePopper(popper)
    }, HIDE_DELAY_MS)
  })
}

function scan() {
  document.querySelectorAll(SELECTORS).forEach(bindPopper)
  document.querySelectorAll('.el-select-dropdown, .el-dropdown-menu').forEach((inner) => {
    const popper = inner.closest('.el-popper') || inner
    bindPopper(popper)
  })
}

/** 是否为按钮类悬浮提示触发器（排除下拉/选择等 click 菜单） */
function isButtonTooltipTrigger(trigger) {
  if (!(trigger instanceof Element)) return false
  if (trigger.closest(MENU_TRIGGER_ANCESTOR)) return false
  return (
    trigger.matches('button, .el-button, [role="button"]') ||
    !!trigger.querySelector('button, .el-button, [role="button"]') ||
    // OnlyChild 会把 class 合到按钮上；兜底：任意非菜单的 tooltip 触发器
    trigger.classList.contains('el-tooltip__trigger')
  )
}

function isTooltipComponent(inst) {
  const name = inst?.type?.name || inst?.type?.__name
  return name === 'ElTooltip' || name === 'tooltip'
}

function forceCloseTooltipInst(inst) {
  const bags = [inst.exposed, inst.setupState, inst.ctx, inst.proxy]
  let closed = false

  for (const bag of bags) {
    if (!bag) continue
    // onClose(event, 0) 会取消 useDelayedToggle 里待执行的 onOpen，避免 hide 后又被 setTimeout(0) 打开
    if (typeof bag.onClose === 'function') {
      try {
        bag.onClose(undefined, 0)
        closed = true
      } catch {
        /* ignore */
      }
    }
    if (typeof bag.hide === 'function') {
      try {
        bag.hide()
        closed = true
      } catch {
        /* ignore */
      }
    }
  }

  // 再强制清 open，防止延迟回调把状态拉回
  for (const bag of bags) {
    const open = bag?.open
    if (open && typeof open === 'object' && 'value' in open && open.value) {
      open.value = false
      closed = true
    }
  }

  return closed
}

/** 只关 ElTooltip，避免误调到其它组件的 hide/blur */
function callTooltipClose(startEl) {
  if (!startEl) return false
  const seen = new Set()

  const tryInst = (inst) => {
    while (inst && !seen.has(inst)) {
      seen.add(inst)
      if (isTooltipComponent(inst) && forceCloseTooltipInst(inst)) return true
      inst = inst.parent
    }
    return false
  }

  let el = startEl
  while (el) {
    if (tryInst(el.__vueParentComponent)) return true
    el = el.parentElement
  }
  return false
}

function findTooltipTriggerByPopper(popper) {
  if (!(popper instanceof Element)) return null
  const id = popper.id
  if (id) {
    try {
      const byAria = document.querySelector(`[aria-describedby~="${CSS.escape(id)}"]`)
      if (byAria) return byAria
    } catch {
      const byAria = document.querySelector(`[aria-describedby~="${id}"]`)
      if (byAria) return byAria
    }
  }

  // 从 popper 的 Vue 树找到 ElTooltip，再取 trigger 元素
  let inst = popper.__vueParentComponent
  const seen = new Set()
  while (inst && !seen.has(inst)) {
    seen.add(inst)
    if (isTooltipComponent(inst)) {
      const triggerRef =
        inst.setupState?.contentRef?.value?.triggerRef ||
        inst.exposed?.popperRef?.value?.triggerRef ||
        inst.setupState?.popperRef?.value?.triggerRef
      const el = triggerRef?.value ?? triggerRef
      if (el instanceof Element) return el
      break
    }
    inst = inst.parent
  }
  return null
}

function isHoverTooltipPopper(popper) {
  if (!(popper instanceof Element) || !isVisible(popper)) return false
  if (isMenuPopper(popper)) return false
  // Element Plus tooltip：popperClass 含 el-tooltip；effect 为 is-dark / is-light
  if (popper.classList.contains('el-tooltip')) return true
  if (popper.getAttribute('role') === 'tooltip') return true
  if (
    (popper.classList.contains('is-dark') || popper.classList.contains('is-light')) &&
    popper.classList.contains('is-pure')
  ) {
    return true
  }
  return false
}

function pointInRect(x, y, rect) {
  return x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom
}

function isPointOverTooltip(x, y, popper) {
  if (pointInRect(x, y, popper.getBoundingClientRect())) return true
  const trigger = findTooltipTriggerByPopper(popper)
  if (trigger && pointInRect(x, y, trigger.getBoundingClientRect())) return true
  // aria 找不到时：光标下的节点若仍是该 tooltip 触发器则视为悬停中
  const under = document.elementFromPoint(x, y)
  if (under instanceof Element) {
    const triggerEl = under.closest('.el-tooltip__trigger')
    if (triggerEl && !triggerEl.closest(MENU_TRIGGER_ANCESTOR)) {
      // 同一 tooltip：触发器 aria-describedby 或共同 vue 父级
      if (popper.id && triggerEl.getAttribute('aria-describedby')?.split(/\s+/).includes(popper.id)) {
        return true
      }
    }
  }
  return false
}

/** 关掉鼠标已不在触发器/气泡上的残留 hover tooltip */
function hideOrphanHoverTooltips(clientX, clientY) {
  if (!Number.isFinite(clientX) || !Number.isFinite(clientY)) return

  document.querySelectorAll(TOOLTIP_POPPER).forEach((popper) => {
    if (!isHoverTooltipPopper(popper)) return
    if (isPointOverTooltip(clientX, clientY, popper)) return
    callTooltipClose(popper)
  })
}

function hideButtonTooltipFromEvent(e) {
  const target = e.target
  if (!(target instanceof Element)) return
  const trigger = target.closest('.el-tooltip__trigger')
  if (!isButtonTooltipTrigger(trigger)) return
  callTooltipClose(trigger)

  // 点击后若按钮马上 loading/disabled，mouseleave 不会再来；下一帧再清一次孤儿
  const x = e.clientX
  const y = e.clientY
  requestAnimationFrame(() => {
    hideOrphanHoverTooltips(x, y)
    setTimeout(() => hideOrphanHoverTooltips(x, y), 0)
    setTimeout(() => hideOrphanHoverTooltips(x, y), 50)
  })
}

/** 触发器变为 disabled / loading 时主动关 tooltip（此时不会再有 mouseleave） */
function hideTooltipForDisabledTrigger(el) {
  if (!(el instanceof Element)) return
  const trigger =
    el.closest('.el-tooltip__trigger') ||
    (el.classList.contains('el-tooltip__trigger') ? el : null)
  if (!isButtonTooltipTrigger(trigger)) return
  callTooltipClose(trigger)
}

function isDisabledLike(el) {
  if (!(el instanceof Element)) return false
  if (el.matches(':disabled, [disabled], .is-disabled, .is-loading')) return true
  if (el.classList.contains('is-loading') || el.classList.contains('is-disabled')) return true
  return el.getAttribute('aria-disabled') === 'true'
}

export function installPopperAutoClose() {
  if (typeof window === 'undefined' || typeof document === 'undefined') return

  if (window.__flashdockPopperAutoCloseObserver) {
    try {
      window.__flashdockPopperAutoCloseObserver.disconnect()
    } catch {
      /* ignore */
    }
  }

  if (window.__flashdockTooltipClickHide) {
    try {
      document.removeEventListener('pointerdown', window.__flashdockTooltipClickHide, true)
    } catch {
      /* ignore */
    }
  }

  if (window.__flashdockTooltipOrphanMove) {
    try {
      document.removeEventListener('pointermove', window.__flashdockTooltipOrphanMove, true)
    } catch {
      /* ignore */
    }
  }

  if (window.__flashdockTooltipOrphanLeave) {
    try {
      document.documentElement.removeEventListener('mouseleave', window.__flashdockTooltipOrphanLeave)
      window.removeEventListener('blur', window.__flashdockTooltipOrphanLeave)
    } catch {
      /* ignore */
    }
  }

  const start = () => {
    scan()
    let lastX = 0
    let lastY = 0
    let orphanTimer = null

    const scheduleOrphanCheck = () => {
      if (orphanTimer) clearTimeout(orphanTimer)
      orphanTimer = setTimeout(() => {
        orphanTimer = null
        hideOrphanHoverTooltips(lastX, lastY)
      }, ORPHAN_HIDE_DELAY_MS)
    }

    const observer = new MutationObserver((mutations) => {
      scan()
      for (const m of mutations) {
        if (m.type === 'attributes') {
          const el = m.target
          if (!(el instanceof Element)) continue
          if (
            (m.attributeName === 'class' ||
              m.attributeName === 'disabled' ||
              m.attributeName === 'aria-disabled') &&
            isDisabledLike(el)
          ) {
            hideTooltipForDisabledTrigger(el)
          }
        } else if (m.type === 'childList') {
          m.addedNodes.forEach((node) => {
            if (!(node instanceof Element)) return
            if (isDisabledLike(node)) hideTooltipForDisabledTrigger(node)
            node.querySelectorAll?.('button:disabled, .el-button.is-loading, .el-button.is-disabled').forEach((el) => {
              hideTooltipForDisabledTrigger(el)
            })
          })
        }
      }
    })
    observer.observe(document.body, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ['class', 'disabled', 'aria-disabled'],
    })
    window.__flashdockPopperAutoCloseObserver = observer

    const onPointerDown = (e) => hideButtonTooltipFromEvent(e)
    document.addEventListener('pointerdown', onPointerDown, true)
    window.__flashdockTooltipClickHide = onPointerDown

    const onPointerMove = (e) => {
      lastX = e.clientX
      lastY = e.clientY
      // 延迟清理：给「触发器 → 气泡」的 enterable 移动留出时间；真离开后会关掉残留
      scheduleOrphanCheck()
    }
    document.addEventListener('pointermove', onPointerMove, true)
    window.__flashdockTooltipOrphanMove = onPointerMove

    const onLeaveWindow = () => {
      if (orphanTimer) {
        clearTimeout(orphanTimer)
        orphanTimer = null
      }
      document.querySelectorAll(TOOLTIP_POPPER).forEach((popper) => {
        if (isHoverTooltipPopper(popper)) callTooltipClose(popper)
      })
    }
    // 鼠标离开文档/窗口时清掉残留
    document.documentElement.addEventListener('mouseleave', onLeaveWindow)
    window.addEventListener('blur', onLeaveWindow)
    window.__flashdockTooltipOrphanLeave = onLeaveWindow
  }

  if (document.body) start()
  else document.addEventListener('DOMContentLoaded', start, { once: true })
}
