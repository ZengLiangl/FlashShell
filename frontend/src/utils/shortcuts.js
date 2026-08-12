import { formatShortcutLabel, isMacPlatform } from './platform'
import {
  expandSendString,
  matchesKeyMapBinding,
} from './keymaps'
import { resolveSnippetCommand } from './snippetVariables'

/** 默认快捷键（useMod=true 表示 Cmd/Ctrl） */
export const DEFAULT_SHORTCUTS = {
  newWindow: { key: 'n', useMod: true },
  machineConfig: { key: 'm', useMod: true },
  connectionManager: { key: 'e', useMod: true },
  envVars: { key: 'u', useMod: true },
  systemSettings: { key: ',', useMod: true },
  refreshConfig: { key: 'r', useMod: true },
  find: { key: 'f', useMod: true },
  copy: { key: 'c', useMod: true },
  paste: { key: 'v', useMod: true },
  clearOutput: { key: 'k', useMod: true },
  commandPalette: { key: 'p', useMod: true, useShift: true },
  paneZoom: { key: 'z', useMod: true, useShift: true },
  nextTab: { key: 'Tab', useMod: true },
  prevTab: { key: 'Tab', useMod: true, useShift: true },
  closeTab: { key: 'w', useMod: true },
  toggleBroadcast: { key: 'b', useMod: true },
  openSftp: { key: 'o', useMod: true, useShift: true },
  openLocalShell: { key: 'l', useMod: true },
  splitFocusLeft: { key: 'ArrowLeft', useMod: true, useAlt: true },
  splitFocusRight: { key: 'ArrowRight', useMod: true, useAlt: true },
  splitFocusUp: { key: 'ArrowUp', useMod: true, useAlt: true },
  splitFocusDown: { key: 'ArrowDown', useMod: true, useAlt: true },
}

export const SHORTCUT_LABELS = {
  newWindow: '新建窗口',
  machineConfig: '机器配置',
  connectionManager: '连接 / 快速切换',
  envVars: '环境变量',
  systemSettings: '系统设置',
  refreshConfig: '刷新配置列表',
  find: '查找',
  copy: '复制',
  paste: '粘贴',
  clearOutput: '清空输出',
  commandPalette: '命令面板（历史/片段）',
  paneZoom: '分屏窗格最大化',
  nextTab: '下一个标签',
  prevTab: '上一个标签',
  closeTab: '关闭当前标签',
  toggleBroadcast: '开关命令广播',
  openSftp: '打开/折叠文件面板',
  openLocalShell: '打开本机终端',
  splitFocusLeft: '分屏焦点←',
  splitFocusRight: '分屏焦点→',
  splitFocusUp: '分屏焦点↑',
  splitFocusDown: '分屏焦点↓',
}

/** 设置页分组展示（覆盖全部可配置项） */
export const SHORTCUT_GROUPS = [
  {
    title: '应用',
    ids: ['newWindow', 'systemSettings', 'machineConfig', 'connectionManager', 'envVars', 'openLocalShell'],
  },
  {
    title: '编辑与输出',
    ids: ['find', 'copy', 'paste', 'clearOutput', 'refreshConfig'],
  },
  {
    title: 'Shell 标签与分屏',
    ids: [
      'commandPalette',
      'paneZoom',
      'nextTab',
      'prevTab',
      'closeTab',
      'toggleBroadcast',
      'openSftp',
      'splitFocusLeft',
      'splitFocusRight',
      'splitFocusUp',
      'splitFocusDown',
    ],
  },
]

export function mergeShortcuts(partial) {
  const result = {}
  for (const [id, def] of Object.entries(DEFAULT_SHORTCUTS)) {
    const cur = partial?.[id]
    result[id] = {
      key: (cur?.key != null && String(cur.key).length > 0) ? String(cur.key) : def.key,
      useMod: cur?.useMod !== undefined ? !!cur.useMod : !!def.useMod,
      useShift: cur?.useShift !== undefined ? !!cur.useShift : !!def.useShift,
      useAlt: cur?.useAlt !== undefined ? !!cur.useAlt : !!def.useAlt,
    }
  }
  return result
}

/** 判断键盘事件是否匹配绑定 */
export function matchesShortcut(e, binding) {
  if (!binding || !binding.key || !e) return false
  const needMod = binding.useMod !== false
  const hasMod = !!(e.metaKey || e.ctrlKey)
  if (needMod !== hasMod) return false

  const needAlt = !!binding.useAlt
  if (needAlt !== !!e.altKey) return false

  const needShift = !!binding.useShift
  if (needShift !== !!e.shiftKey) return false

  const target = String(binding.key)
  if (target === ',') {
    return e.key === ',' || e.code === 'Comma'
  }
  if (target === 'Tab') {
    return e.key === 'Tab' || e.code === 'Tab'
  }
  if (target.startsWith('Arrow')) {
    return e.key === target || e.code === target
  }

  const pressed = e.key === 'Escape' ? 'Escape' : String(e.key || '')
  if (pressed.toLowerCase() === target.toLowerCase()) return true

  const code = String(e.code || '')
  if (/^Key[A-Z]$/i.test(code)) {
    return code.slice(3).toLowerCase() === target.toLowerCase()
  }
  if (/^Digit[0-9]$/.test(code)) {
    return code.slice(5) === target
  }
  return false
}

/**
 * 检测快捷键表内部冲突。返回 [{ a, b, label }]
 */
export function findShortcutConflicts(shortcuts) {
  const entries = Object.entries(shortcuts || {}).filter(([, b]) => b?.key)
  const conflicts = []
  for (let i = 0; i < entries.length; i++) {
    for (let j = i + 1; j < entries.length; j++) {
      const [idA, a] = entries[i]
      const [idB, b] = entries[j]
      if (
        String(a.key).toLowerCase() === String(b.key).toLowerCase()
        && !!a.useMod === !!b.useMod
        && !!a.useShift === !!b.useShift
        && !!a.useAlt === !!b.useAlt
      ) {
        conflicts.push({
          a: idA,
          b: idB,
          label: `${SHORTCUT_LABELS[idA] || idA} 与 ${SHORTCUT_LABELS[idB] || idB}`,
        })
      }
    }
  }
  return conflicts
}

export function formatShortcut(binding, isMac = isMacPlatform()) {
  return formatShortcutLabel(binding, isMac)
}

/** 是否为 xterm 内部输入框 */
export function isXtermInput(el) {
  if (!el || el.nodeType !== 1) return false
  return !!el.classList?.contains('xterm-helper-textarea')
}

/**
 * 是否为应跳过全局快捷键的表单输入（不含 xterm）。
 * 终端聚焦时焦点在 textarea 上，不能当作普通输入框屏蔽。
 */
export function isFormFieldTarget(el) {
  if (!el || el.nodeType !== 1) return false
  if (isXtermInput(el)) return false
  const tag = el.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
  if (el.isContentEditable || el.contentEditable === 'true') return true
  return false
}

/** 从 KeyboardEvent 生成绑定（用于设置页录制） */
export function bindingFromEvent(e) {
  let key = e.key === 'Escape' ? '' : e.key
  if (!key || key === 'Control' || key === 'Meta' || key === 'Alt' || key === 'Shift') {
    return null
  }
  if (key === 'Tab' || key.startsWith('Arrow')) {
    // keep
  } else if (key.length === 1) {
    key = key.toLowerCase()
  }
  return {
    key: key === ',' ? ',' : key,
    useMod: true,
    useShift: !!e.shiftKey,
    useAlt: !!e.altKey,
  }
}

export function emptySnippetBinding() {
  return { key: '', useMod: false, useAlt: false, useShift: false }
}

/** 规范化单条命令片段 */
export function normalizeSnippet(raw, index = 0) {
  const binding = raw?.binding
  return {
    id: raw?.id || `sn-${Date.now()}-${index}`,
    name: raw?.name != null ? String(raw.name) : '',
    command: raw?.command != null ? String(raw.command) : '',
    scope: raw?.scope ? String(raw.scope) : 'global',
    execute: raw?.execute !== undefined ? !!raw.execute : true,
    onConnect: !!raw?.onConnect,
    binding: {
      key: binding?.key != null ? String(binding.key) : '',
      useMod: !!binding?.useMod,
      useAlt: !!binding?.useAlt,
      useShift: !!binding?.useShift,
    },
  }
}

export function normalizeSnippets(list) {
  return (Array.isArray(list) ? list : []).map((s, i) => normalizeSnippet(s, i))
}

/** Shell 终端快捷键：匹配绑定了组合键的片段 */
export function findMatchingSnippet(e, snippets) {
  if (!e || !Array.isArray(snippets)) return null
  for (const s of snippets) {
    if (!s?.binding?.key) continue
    if (matchesKeyMapBinding(e, s.binding)) return s
  }
  return null
}

/** 片段发送内容：支持转义；execute 时自动补换行 */
export async function buildSnippetPayload(snippet, { promptVars = true } = {}) {
  const resolved = await resolveSnippetCommand(snippet, { prompt: promptVars })
  if (resolved == null) return ''
  let text = expandSendString(resolved)
  if (snippet?.execute && text && !/[\r\n]$/.test(text)) {
    text += '\n'
  }
  return text
}
