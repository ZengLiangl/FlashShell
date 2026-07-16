/**
 * 下拉面板（Dropdown / Select / Autocomplete）鼠标离开后自动关闭。
 */
const SELECTORS = [
  '.el-dropdown__popper',
  '.el-select__popper',
  '.el-autocomplete__popper',
  '.el-cascader__dropdown',
].join(',')

const HIDE_DELAY_MS = 160
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

export function installPopperAutoClose() {
  if (typeof window === 'undefined' || typeof document === 'undefined') return

  if (window.__flashdockPopperAutoCloseObserver) {
    try {
      window.__flashdockPopperAutoCloseObserver.disconnect()
    } catch {
      /* ignore */
    }
  }

  const start = () => {
    scan()
    const observer = new MutationObserver(() => scan())
    observer.observe(document.body, { childList: true, subtree: true })
    window.__flashdockPopperAutoCloseObserver = observer
  }

  if (document.body) start()
  else document.addEventListener('DOMContentLoaded', start, { once: true })
}
