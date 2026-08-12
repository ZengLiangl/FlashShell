/** 主机图标预设 */
export const HOST_ICON_PRESETS = [
  { id: '', label: '默认', emoji: '' },
  { id: 'server', label: '服务器', emoji: '🖥' },
  { id: 'linux', label: 'Linux', emoji: '🐧' },
  { id: 'ubuntu', label: 'Ubuntu', emoji: '🟠' },
  { id: 'debian', label: 'Debian', emoji: '🔴' },
  { id: 'centos', label: 'CentOS', emoji: '🔵' },
  { id: 'docker', label: 'Docker', emoji: '🐳' },
  { id: 'cloud', label: '云主机', emoji: '☁' },
  { id: 'db', label: '数据库', emoji: '🗄' },
  { id: 'router', label: '网络设备', emoji: '📡' },
]

const PRESET_MAP = Object.fromEntries(HOST_ICON_PRESETS.map((p) => [p.id, p.emoji]))

/** 解析机器图标展示文本（emoji 或首字母） */
export function resolveHostIcon(machine) {
  const raw = String(machine?.icon || '').trim()
  if (!raw) return { kind: 'default', text: '' }
  if (PRESET_MAP[raw] !== undefined) {
    const emoji = PRESET_MAP[raw]
    return emoji ? { kind: 'emoji', text: emoji } : { kind: 'default', text: '' }
  }
  // 直接存的 emoji / 短文本
  if ([...raw].length <= 3) return { kind: 'emoji', text: raw }
  return { kind: 'text', text: raw.slice(0, 2) }
}
