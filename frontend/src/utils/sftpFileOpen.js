/**
 * SFTP 文件打开方式工具（扩展名关联 / 默认打开 / 二进制判断）
 */

const BINARY_EXTENSIONS = new Set([
  'jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'ico', 'tiff', 'tif',
  'heic', 'heif', 'avif', 'jfif', 'psd', 'ai', 'eps', 'raw', 'cr2', 'nef',
  'mp3', 'wav', 'flac', 'aac', 'ogg', 'wma', 'm4a', 'aiff', 'opus',
  'mp4', 'avi', 'mkv', 'mov', 'wmv', 'flv', 'webm', 'm4v', '3gp', 'mpeg', 'mpg',
  'zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz', 'lz', 'lzma', 'zst',
  'tgz', 'tbz2', 'txz', 'cab', 'iso', 'dmg',
  'exe', 'dll', 'so', 'dylib', 'bin', 'app', 'msi', 'deb', 'rpm',
  'apk', 'ipa', 'jar', 'war', 'ear',
  'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'odt', 'ods', 'odp',
  'ttf', 'otf', 'woff', 'woff2', 'eot',
  'db', 'sqlite', 'sqlite3', 'mdb', 'accdb',
  'o', 'obj', 'pyc', 'pyo', 'class', 'beam',
  'swf', 'fla', 'blend', 'unity3d', 'unitypackage',
])

/** @returns {string} 无点扩展名；无扩展名返回 file */
export function getFileExtension(fileName) {
  const name = String(fileName || '')
  const lastDot = name.lastIndexOf('.')
  if (lastDot <= 0) return 'file'
  return name.slice(lastDot + 1).toLowerCase()
}

export function hasFileExtension(fileName) {
  return getFileExtension(fileName) !== 'file'
}

export function isKnownBinaryFile(fileName) {
  const ext = getFileExtension(fileName)
  if (ext === 'file') return false
  return BINARY_EXTENSIONS.has(ext)
}

export function normalizeSftpDefaultOpener(v) {
  const s = String(v || '').trim()
  if (s === 'builtin-editor' || s === 'system-app') return s
  return 'ask'
}

/**
 * @param {Record<string, any>|null|undefined} config
 * @param {string} fileName
 * @returns {{ openerType: string, systemApp?: { path: string, name: string } } | null}
 */
export function getOpenerForFile(config, fileName) {
  const ext = getFileExtension(fileName)
  const associations = config?.sftpFileAssociations || {}
  if (associations[ext]?.openerType) {
    return associations[ext]
  }
  const opener = normalizeSftpDefaultOpener(config?.sftpDefaultOpener)
  if (opener === 'ask') return null
  if (opener === 'builtin-editor') {
    if (isKnownBinaryFile(fileName)) return null
    return { openerType: 'builtin-editor' }
  }
  const app = config?.sftpDefaultSystemApp
  if (app?.path) {
    return { openerType: 'system-app', systemApp: app }
  }
  return null
}

export function formatAssociationLabel(entry) {
  if (!entry?.openerType) return ''
  if (entry.openerType === 'builtin-editor') return '内置编辑器'
  if (entry.systemApp?.name) return entry.systemApp.name
  if (entry.systemApp?.path) return entry.systemApp.path
  return '应用程序'
}
