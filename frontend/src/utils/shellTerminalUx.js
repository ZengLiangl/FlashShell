/** Shell 终端 UX：字号、OSC 52、行时间戳 */

export const SHELL_FONT_SIZE_DEFAULT = 13
export const SHELL_FONT_SIZE_MIN = 8
export const SHELL_FONT_SIZE_MAX = 32

export function clampShellFontSize(size) {
  const n = Number(size)
  if (!Number.isFinite(n) || n <= 0) return SHELL_FONT_SIZE_DEFAULT
  return Math.min(SHELL_FONT_SIZE_MAX, Math.max(SHELL_FONT_SIZE_MIN, Math.round(n)))
}

/** 从绝对路径取末段名（含 ~）；根目录返回 / */
export function cwdBasename(cwd) {
  const raw = String(cwd || '').trim()
  if (!raw) return ''
  if (raw === '/') return '/'
  const cleaned = raw.replace(/\/+$/, '')
  const idx = cleaned.lastIndexOf('/')
  if (idx < 0) return cleaned
  return cleaned.slice(idx + 1) || '/'
}

/**
 * 解析 OSC 52 载荷（Pc;Pd），仅处理写入剪贴板。
 * @returns {string|null} 解码后的文本；查询/空/无效返回 null
 */
export function decodeOsc52ClipboardPayload(data) {
  const raw = String(data || '')
  if (!raw) return null
  const semi = raw.indexOf(';')
  // 允许省略 Pc：`;BASE64` 或 仅 `BASE64`
  let pd = semi >= 0 ? raw.slice(semi + 1) : raw
  pd = pd.replace(/[\r\n]+/g, '').trim()
  if (!pd || pd === '?') return null
  try {
    const binary = atob(pd)
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
    return new TextDecoder('utf-8').decode(bytes)
  } catch {
    return null
  }
}

function pad2(n) {
  return n < 10 ? `0${n}` : String(n)
}

export function formatLineTimestamp(date = new Date()) {
  return `${pad2(date.getHours())}:${pad2(date.getMinutes())}:${pad2(date.getSeconds())}`
}

/** 去掉常见 ANSI / OSC，便于识别 Password: 等可见提示 */
export function stripAnsiForPromptDetect(text) {
  return String(text || '')
    .replace(/\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/g, '')
    .replace(/\x1b\[[0-9;?]*[ -/]*[@-~]/g, '')
    .replace(/\x1b./g, '')
}

/**
 * 终端输出是否像密码 / sudo 提示（末尾附近）。
 * 匹配 Password: / password: / 密码： / 密码:
 */
export function looksLikePasswordPrompt(text) {
  const plain = stripAnsiForPromptDetect(text)
  if (!plain) return false
  const tail = plain.slice(-240)
  return /(?:^|[\r\n\s])(?:\[sudo\]\s*)?(?:password|passwd|密码)\s*[:：]\s*$/im.test(tail)
}

/**
 * 在新行起始处注入灰色时间戳前缀。跳过纯 CSI/OSC 控制块开头。
 * @param {string} text
 * @param {{ atLineStart: boolean }} state 可变：atLineStart
 * @returns {string}
 */
export function prefixLineTimestamps(text, state) {
  if (!text || !state) return text || ''
  let out = ''
  let i = 0
  const len = text.length
  while (i < len) {
    const ch = text[i]
    if (ch === '\n') {
      out += ch
      state.atLineStart = true
      i += 1
      continue
    }
    if (ch === '\r') {
      out += ch
      // \r\n 视为一行结束；单独 \r 也当作行首
      if (text[i + 1] === '\n') {
        out += '\n'
        i += 2
      } else {
        i += 1
      }
      state.atLineStart = true
      continue
    }
    // 透传 ESC 序列，不打断「行首」判断（序列本身不消耗可见行）
    if (ch === '\x1b') {
      const rest = text.slice(i)
      const osc = rest.match(/^\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)/)
      if (osc) {
        out += osc[0]
        i += osc[0].length
        continue
      }
      const csi = rest.match(/^\x1b\[[0-9;?]*[ -/]*[@-~]/)
      if (csi) {
        out += csi[0]
        i += csi[0].length
        continue
      }
      out += ch
      i += 1
      continue
    }
    if (state.atLineStart) {
      // 空字符 / BEL 等不插时间戳
      if (ch >= ' ' || ch === '\t') {
        out += `\x1b[90m[${formatLineTimestamp()}]\x1b[0m `
        state.atLineStart = false
      }
    }
    out += ch
    i += 1
  }
  return out
}
