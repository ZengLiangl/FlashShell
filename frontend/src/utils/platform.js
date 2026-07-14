/** 当前是否为 macOS（含 iPad 桌面 Safari 的 MacIntel） */
export function isMacPlatform() {
  if (typeof navigator === 'undefined') return false
  const platform = navigator.platform || ''
  const ua = navigator.userAgent || ''
  return /Mac|iPhone|iPad|iPod/i.test(platform) || /Mac OS X/i.test(ua)
}

/** 修饰键展示文案：Mac = Command，Windows/Linux = Ctrl */
export function modKeyLabel(isMac = isMacPlatform()) {
  return isMac ? 'Command' : 'Ctrl'
}

/**
 * 将快捷键绑定格式化为展示文案
 * @param {{ key?: string, useMod?: boolean }} binding
 * @param {boolean} [isMac]
 */
export function formatShortcutLabel(binding, isMac = isMacPlatform()) {
  if (!binding || !binding.key) return ''
  const key = String(binding.key).length === 1
    ? String(binding.key).toUpperCase()
    : String(binding.key)
  if (binding.useMod === false) return key
  return `${modKeyLabel(isMac)}+${key}`
}
