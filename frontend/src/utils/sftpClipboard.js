/**
 * SFTP 应用内剪贴板（复制/剪切/粘贴，不含多选）
 */

/** @type {{ mode: 'copy'|'cut', machineName: string, entry: object } | null} */
let clip = null

export function setSftpClipboard(mode, machineName, entry) {
  if (!entry?.path || !machineName) {
    clip = null
    return
  }
  clip = {
    mode: mode === 'cut' ? 'cut' : 'copy',
    machineName: String(machineName),
    entry: { ...entry },
  }
}

export function clearSftpClipboard() {
  clip = null
}

export function getSftpClipboard() {
  return clip
}

export function hasSftpClipboardFor(machineName) {
  return !!(clip && clip.machineName === machineName && clip.entry?.path)
}

/** 从 paste 事件提取本地文件路径（Wails/系统拖入风格） */
export function extractClipboardLocalPaths(clipboardData) {
  if (!clipboardData) return []
  const paths = []
  const files = clipboardData.files
  if (files?.length) {
    for (let i = 0; i < files.length; i++) {
      const f = files[i]
      const p = f?.path || f?.webkitRelativePath
      if (p && String(p).startsWith('/')) paths.push(String(p))
      // Windows 盘符
      else if (p && /^[a-zA-Z]:[\\/]/.test(String(p))) paths.push(String(p))
    }
  }
  const text = String(clipboardData.getData?.('text/plain') || '').trim()
  if (text) {
    for (const line of text.split(/\r?\n/)) {
      const t = line.trim().replace(/^['"]|['"]$/g, '')
      if (!t) continue
      if (t.startsWith('/') || /^[a-zA-Z]:[\\/]/.test(t) || t.startsWith('file://')) {
        paths.push(t.replace(/^file:\/\//, ''))
      }
    }
  }
  return [...new Set(paths)]
}
