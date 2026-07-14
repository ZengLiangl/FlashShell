import { formatShortcutLabel, isMacPlatform } from './platform'

/** 默认快捷键（useMod=true 表示 Cmd/Ctrl） */
export const DEFAULT_SHORTCUTS = {
  newWindow: { key: 'n', useMod: true },
  machineConfig: { key: 'm', useMod: true },
  envVars: { key: 'e', useMod: true },
  systemSettings: { key: ',', useMod: true },
  refreshConfig: { key: 'r', useMod: true },
  find: { key: 'f', useMod: true },
  copy: { key: 'c', useMod: true },
  clearOutput: { key: 'k', useMod: true },
}

export const SHORTCUT_LABELS = {
  newWindow: '新建窗口',
  machineConfig: '机器配置',
  envVars: '环境变量',
  systemSettings: '系统设置',
  refreshConfig: '刷新配置列表',
  find: '查找',
  copy: '复制',
  clearOutput: '清空输出',
}

export function mergeShortcuts(partial) {
  const result = {}
  for (const [id, def] of Object.entries(DEFAULT_SHORTCUTS)) {
    const cur = partial?.[id]
    result[id] = {
      key: (cur?.key != null && String(cur.key).length > 0) ? String(cur.key) : def.key,
      useMod: cur?.useMod !== undefined ? !!cur.useMod : def.useMod,
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
  if (e.altKey) return false

  const pressed = e.key === 'Escape' ? 'Escape' : String(e.key)
  const target = String(binding.key)
  if (target === ',') return pressed === ','
  return pressed.toLowerCase() === target.toLowerCase()
}

export function formatShortcut(binding, isMac = isMacPlatform()) {
  return formatShortcutLabel(binding, isMac)
}

/** 从 KeyboardEvent 生成绑定（用于设置页录制） */
export function bindingFromEvent(e) {
  const key = e.key === 'Escape' ? '' : (e.key.length === 1 ? e.key.toLowerCase() : e.key)
  if (!key || key === 'Control' || key === 'Meta' || key === 'Alt' || key === 'Shift') {
    return null
  }
  return {
    key: key === ',' ? ',' : key.toLowerCase(),
    useMod: true,
  }
}
