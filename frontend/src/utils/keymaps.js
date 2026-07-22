import { isMacPlatform, modKeyLabel } from './platform'

/** 默认空映射列表 */
export function emptyKeyMapSettings() {
  return { entries: [] }
}

/** 规范化从后端读到的配置 */
export function normalizeKeyMapSettings(raw) {
  const entries = Array.isArray(raw?.entries) ? raw.entries : []
  return {
    entries: entries.map((e, i) => ({
      id: e?.id || `km-${Date.now()}-${i}`,
      enabled: e?.enabled !== false,
      name: e?.name != null ? String(e.name) : '',
      action: e?.action || 'sendString',
      sendString: e?.sendString != null ? String(e.sendString) : '',
      binding: normalizeBinding(e?.binding),
    })),
  }
}

function normalizeBinding(b) {
  return {
    key: b?.key != null ? String(b.key) : '',
    useMod: !!b?.useMod,
    useAlt: !!b?.useAlt,
    useShift: !!b?.useShift,
  }
}

/** 是否为功能键（可无修饰键单独映射） */
export function isFunctionKeyName(key) {
  return /^F([1-9]|1[0-2])$/i.test(String(key || ''))
}

/**
 * 展开发送字符串中的转义
 * 支持：\\ \n \r \t \e/\E \xHH \uHHHH
 */
export function expandSendString(input) {
  if (input == null || input === '') return ''
  const src = String(input)
  let out = ''
  for (let i = 0; i < src.length; i++) {
    const ch = src[i]
    if (ch !== '\\' || i + 1 >= src.length) {
      out += ch
      continue
    }
    const next = src[i + 1]
    switch (next) {
      case 'n':
        out += '\n'
        i++
        break
      case 'r':
        out += '\r'
        i++
        break
      case 't':
        out += '\t'
        i++
        break
      case 'e':
      case 'E':
        out += '\x1b'
        i++
        break
      case '\\':
        out += '\\'
        i++
        break
      case 'x':
      case 'X': {
        const hex = src.slice(i + 2, i + 4)
        if (/^[0-9a-fA-F]{2}$/.test(hex)) {
          out += String.fromCharCode(parseInt(hex, 16))
          i += 3
        } else {
          out += ch
        }
        break
      }
      case 'u':
      case 'U': {
        const hex = src.slice(i + 2, i + 6)
        if (/^[0-9a-fA-F]{4}$/.test(hex)) {
          out += String.fromCharCode(parseInt(hex, 16))
          i += 5
        } else {
          out += ch
        }
        break
      }
      default:
        out += ch
        break
    }
  }
  return out
}

/** 展示用格式化（含 Alt） */
export function formatKeyMapBinding(binding, isMac = isMacPlatform()) {
  if (!binding || !binding.key) return ''
  const key = String(binding.key).length === 1
    ? String(binding.key).toUpperCase()
    : String(binding.key)
  const parts = []
  if (binding.useMod) parts.push(modKeyLabel(isMac))
  if (binding.useAlt) parts.push('Alt')
  if (binding.useShift) parts.push('Shift')
  parts.push(key)
  return parts.join('+')
}

/** 兼容旧 formatShortcutLabel 场景（仅 mod/shift） */
export function formatKeyMapParts(binding) {
  const label = formatKeyMapBinding(binding)
  return label ? label.split('+').filter(Boolean) : []
}

function eventKeyToken(e) {
  const code = String(e.code || '')
  if (/^F([1-9]|1[0-2])$/.test(code)) return code
  if (/^Key[A-Z]$/i.test(code)) return code.slice(3).toLowerCase()
  if (/^Digit[0-9]$/.test(code)) return code.slice(5)
  if (/^Numpad[0-9]$/.test(code)) return code.slice(6)
  if (code === 'Comma') return ','
  if (code === 'Period') return '.'
  if (code === 'Slash') return '/'
  if (code === 'Backslash') return '\\'
  if (code === 'Minus') return '-'
  if (code === 'Equal') return '='
  if (code === 'BracketLeft') return '['
  if (code === 'BracketRight') return ']'
  if (code === 'Semicolon') return ';'
  if (code === 'Quote') return "'"
  if (code === 'Backquote') return '`'

  const pressed = e.key === 'Escape' ? 'Escape' : String(e.key || '')
  if (pressed.length === 1) return pressed.toLowerCase()
  if (/^F([1-9]|1[0-2])$/i.test(pressed)) return pressed.toUpperCase()
  return pressed
}

/**
 * 从 KeyboardEvent 录制按键映射。
 * 普通键需至少带一个修饰键；F1–F12 可单独使用。
 */
export function keymapBindingFromEvent(e) {
  if (!e) return null
  if (e.key === 'Escape') return null
  if (e.key === 'Control' || e.key === 'Meta' || e.key === 'Alt' || e.key === 'Shift') {
    return null
  }

  const key = eventKeyToken(e)
  if (!key) return null

  const useMod = !!(e.metaKey || e.ctrlKey)
  const useAlt = !!e.altKey
  const useShift = !!e.shiftKey
  const isFn = isFunctionKeyName(key)

  if (!isFn && !useMod && !useAlt && !useShift) {
    return null
  }

  return { key, useMod, useAlt, useShift }
}

/** 判断事件是否匹配一条按键映射绑定 */
export function matchesKeyMapBinding(e, binding) {
  if (!e || !binding || !binding.key) return false

  const needMod = !!binding.useMod
  const needAlt = !!binding.useAlt
  const needShift = !!binding.useShift
  const hasMod = !!(e.metaKey || e.ctrlKey)
  const hasAlt = !!e.altKey
  const hasShift = !!e.shiftKey

  if (needMod !== hasMod) return false
  if (needAlt !== hasAlt) return false
  if (needShift !== hasShift) return false

  const target = String(binding.key)
  const pressed = eventKeyToken(e)
  if (!pressed) return false

  if (isFunctionKeyName(target) || isFunctionKeyName(pressed)) {
    return pressed.toUpperCase() === target.toUpperCase()
  }
  return pressed.toLowerCase() === target.toLowerCase()
}

/**
 * 在启用的映射中查找第一条匹配项。
 * @returns {object|null} 匹配的 entry
 */
export function findMatchingKeyMap(e, entries) {
  if (!e || !Array.isArray(entries)) return null
  for (const entry of entries) {
    if (!entry || entry.enabled === false) continue
    if (entry.action && entry.action !== 'sendString') continue
    if (!matchesKeyMapBinding(e, entry.binding)) continue
    return entry
  }
  return null
}
