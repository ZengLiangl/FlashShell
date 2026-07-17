/**
 * Shell 日志高亮：对完整行注入 ANSI（风格接近 WindTerm）。
 * - 仅处理换行结束的完整日志行；less 等分页器的行内重绘原样透传
 * - 仅处理「像日志」的行
 * - 可按关键字单独关闭高亮
 */

const R = '\x1b[0m'

export const LOG_HIGHLIGHT_KEYS = [
  'timestamp',
  'threadId',
  'info',
  'debug',
  'warn',
  'error',
  'logger',
  'sql',
  'label',
]

export const DEFAULT_SHELL_LOG_COLORS = {
  timestamp: '#92d050',
  threadId: '#c586c0',
  info: '#569cd6',
  debug: '#ce9178',
  warn: '#dcdcaa',
  error: '#f44747',
  logger: '#4ec9b0',
  sql: '#dcdcaa',
  label: '#9cdcfe',
}

const RE_ANSI = /\x1b(?:\[[0-9;?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\))/g

/** less/vim 等全屏或行内重绘用的控制序列（非单纯配色 SGR） */
const RE_INTERACTIVE_TERMINAL = /\x1b(?:\[\??(?:1049|1047|47)[hl]|\[[0-9;]*[HJKsuABCDEFG]|\[7m|\[27m|\][^\x07\x1b]*(?:\x07|\x1b\\)|[78])/g

const RE_ALT_SCREEN_ENTER = /\x1b\[\??(?:1049|1047|47)h|\x1b\[1049h/g
const RE_ALT_SCREEN_LEAVE = /\x1b\[\??(?:1049|1047|47)l|\x1b\[1049l/g

const RE_TIMESTAMP =
  /\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:[.,]\d{1,9})?|\b\d{2}:\d{2}:\d{2}(?:[.,]\d{1,3})?\b/g

const RE_LEVEL =
  /\b(TRACE|DEBUG|INFO|NOTICE|WARN(?:ING)?|ERROR|FATAL|SEVERE|CRITICAL)\b/gi

const RE_THREAD_AFTER_TS = /(\d{2}:\d{2}:\d{2}(?:[.,]\d{1,9})?\s+)(\d{4,})\b/g

const RE_LOGGER =
  /\b(?:[a-z](?:\.[A-Za-z0-9_$]+){2,}|\b(?:[a-zA-Z_]\w*\.){2,}[A-Za-z_]\w*)\b/g

const RE_SQL_LABEL =
  /\b(?:Preparing|Parameters|Total|Columns)\b:|(?:==>| <==)/g

const RE_SQL_KW =
  /\b(?:SELECT|INSERT|UPDATE|DELETE|FROM|WHERE|JOIN|LEFT|RIGHT|INNER|OUTER|INTO|VALUES|SET|AND|OR|NOT|NULL|AS|ON|ORDER|GROUP|HAVING|LIMIT|OFFSET|CREATE|ALTER|DROP|TABLE|INDEX|UNION|DISTINCT|COUNT|SUM|AVG|MAX|MIN|BETWEEN|LIKE|IN|EXISTS|CASE|WHEN|THEN|ELSE|END)\b/gi

const RE_BRACKET_LEVEL = /\[(TRACE|DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL)\]/gi

const HEX_COLOR = /^#[0-9A-Fa-f]{6}$/

/** 合并用户配色与默认值 */
export function mergeLogHighlightColors(input) {
  const out = { ...DEFAULT_SHELL_LOG_COLORS }
  if (!input || typeof input !== 'object') return out
  const map = {
    timestamp: input.timestamp ?? input.Timestamp,
    threadId: input.threadId ?? input.ThreadId,
    info: input.info ?? input.Info,
    debug: input.debug ?? input.Debug,
    warn: input.warn ?? input.Warn,
    error: input.error ?? input.Error,
    logger: input.logger ?? input.Logger,
    sql: input.sql ?? input.Sql,
    label: input.label ?? input.Label,
  }
  for (const [key, val] of Object.entries(map)) {
    if (typeof val === 'string' && HEX_COLOR.test(val.trim())) {
      out[key] = val.trim()
    }
  }
  return out
}

/** disabled 列表 → 各关键字是否高亮（缺省全部开启） */
export function mergeLogHighlightRules(disabled) {
  const set = new Set(Array.isArray(disabled) ? disabled : [])
  const rules = {}
  for (const key of LOG_HIGHLIGHT_KEYS) {
    rules[key] = !set.has(key)
  }
  return rules
}

/** 规则 → 保存用 disabled 列表 */
export function rulesToDisabled(rules) {
  if (!rules || typeof rules !== 'object') return []
  return LOG_HIGHLIGHT_KEYS.filter((k) => rules[k] === false)
}

/** 合并完整高亮配置 */
export function mergeLogHighlightConfig(input) {
  return {
    colors: mergeLogHighlightColors(input?.colors ?? input?.shellLogHighlightColors),
    rules: mergeLogHighlightRules(input?.disabled ?? input?.shellLogHighlightDisabled),
  }
}

function hexToAnsi(hex) {
  const h = String(hex || '').replace('#', '')
  if (h.length !== 6) return ''
  const r = parseInt(h.slice(0, 2), 16)
  const g = parseInt(h.slice(2, 4), 16)
  const b = parseInt(h.slice(4, 6), 16)
  if ([r, g, b].some((n) => Number.isNaN(n))) return ''
  return `\x1b[38;2;${r};${g};${b}m`
}

function buildPalette(colors) {
  const c = mergeLogHighlightColors(colors)
  return {
    ts: hexToAnsi(c.timestamp),
    tid: hexToAnsi(c.threadId),
    info: hexToAnsi(c.info),
    debug: hexToAnsi(c.debug),
    warn: hexToAnsi(c.warn),
    error: hexToAnsi(c.error),
    logger: hexToAnsi(c.logger),
    sql: hexToAnsi(c.sql),
    label: hexToAnsi(c.label),
  }
}

function isRuleOn(rules, key) {
  if (!rules || typeof rules !== 'object') return true
  return rules[key] !== false
}

/** 剥离 ANSI 与其它 TTY 控制符（less / vim 等） */
export function stripTerminalControls(text) {
  return String(text || '')
    .replace(RE_ANSI, '')
    .replace(/[\x00-\x08\x0b\x0c\x0e-\x1a\x7f]/g, '')
}

/** 是否包含分页器/全屏程序的交互控制序列（搜索反显、清行、光标定位等） */
export function hasInteractiveTerminalSequences(text) {
  RE_INTERACTIVE_TERMINAL.lastIndex = 0
  return RE_INTERACTIVE_TERMINAL.test(String(text || ''))
}

/** 根据 alternate screen 进出序列更新 TUI 嵌套深度 */
export function updateTuiModeDepth(depth, text) {
  const s = String(text || '')
  const enters = s.match(RE_ALT_SCREEN_ENTER)?.length || 0
  const leaves = s.match(RE_ALT_SCREEN_LEAVE)?.length || 0
  return Math.max(0, depth + enters - leaves)
}

function looksLikeLogLine(line) {
  if (!line || line.length < 8) return false
  const hasTs = /\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}|\d{2}:\d{2}:\d{2}[.,]\d/.test(line)
  const hasLevel = /\b(TRACE|DEBUG|INFO|NOTICE|WARN(?:ING)?|ERROR|FATAL|SEVERE|CRITICAL)\b/i.test(line)
  if (hasTs && hasLevel) return true
  if (hasLevel && /\[(TRACE|DEBUG|INFO|WARN|ERROR|FATAL)\]/i.test(line)) return true
  if (/\b(?:Preparing|Parameters|Total)\b:/.test(line)) return true
  if (hasTs && /\b(?:SELECT|INSERT|UPDATE|DELETE)\b/i.test(line)) return true
  return false
}

function pushMatch(matches, start, end, ansiColor, hexColor) {
  if (!ansiColor || start < 0 || end <= start) return
  matches.push({ start, end, ansiColor, hexColor: hexColor || null })
}

function levelRuleKey(level) {
  const lv = String(level).toUpperCase()
  if (lv === 'DEBUG' || lv === 'TRACE') return 'debug'
  if (lv === 'WARN' || lv === 'WARNING') return 'warn'
  if (lv === 'ERROR' || lv === 'FATAL' || lv === 'SEVERE' || lv === 'CRITICAL') return 'error'
  return 'info'
}

function collectMatches(line, config) {
  const colors = mergeLogHighlightColors(config?.colors)
  const rules = mergeLogHighlightRules(config?.disabled)
  if (config?.rules) {
    for (const key of LOG_HIGHLIGHT_KEYS) {
      if (Object.prototype.hasOwnProperty.call(config.rules, key)) {
        rules[key] = config.rules[key] !== false
      }
    }
  }
  const palette = buildPalette(colors)
  const matches = []
  const run = (re, ansiColor, hexColor, ruleKey) => {
    if (!isRuleOn(rules, ruleKey)) return
    re.lastIndex = 0
    let m
    while ((m = re.exec(line)) !== null) {
      pushMatch(matches, m.index, m.index + m[0].length, ansiColor, hexColor)
      if (m[0].length === 0) re.lastIndex++
    }
  }
  const runGroup = (re, groupIdx, ansiColor, hexColor, ruleKey) => {
    if (!isRuleOn(rules, ruleKey)) return
    re.lastIndex = 0
    let m
    while ((m = re.exec(line)) !== null) {
      const g = m[groupIdx]
      if (!g) continue
      const start = m.index + m[0].indexOf(g)
      pushMatch(matches, start, start + g.length, ansiColor, hexColor)
      if (m[0].length === 0) re.lastIndex++
    }
  }
  const runLevel = (re) => {
    re.lastIndex = 0
    let m
    while ((m = re.exec(line)) !== null) {
      const rk = levelRuleKey(m[1])
      if (!isRuleOn(rules, rk)) continue
      let hex = colors.info
      let ansi = palette.info
      if (rk === 'debug') {
        hex = colors.debug
        ansi = palette.debug
      } else if (rk === 'warn') {
        hex = colors.warn
        ansi = palette.warn
      } else if (rk === 'error') {
        hex = colors.error
        ansi = palette.error
      }
      pushMatch(matches, m.index, m.index + m[0].length, ansi, hex)
      if (m[0].length === 0) re.lastIndex++
    }
  }

  run(RE_TIMESTAMP, palette.ts, colors.timestamp, 'timestamp')
  runGroup(RE_THREAD_AFTER_TS, 2, palette.tid, colors.threadId, 'threadId')
  runLevel(RE_BRACKET_LEVEL)
  runLevel(RE_LEVEL)
  run(RE_LOGGER, palette.logger, colors.logger, 'logger')
  run(RE_SQL_LABEL, palette.label, colors.label, 'label')
  if (/\b(?:Preparing|Parameters|SELECT|INSERT|UPDATE|DELETE)\b/i.test(line)) {
    run(RE_SQL_KW, palette.sql, colors.sql, 'sql')
  }

  return matches
}

function pickMatches(matches) {
  matches.sort((a, b) => a.start - b.start || b.end - a.end)
  const picked = []
  let lastEnd = 0
  for (const m of matches) {
    if (m.start < lastEnd) continue
    picked.push(m)
    lastEnd = m.end
  }
  return picked
}

function applyMatches(line, matches) {
  const picked = pickMatches(matches)
  if (!picked.length) return line
  let out = ''
  let pos = 0
  for (const m of picked) {
    out += line.slice(pos, m.start)
    out += m.ansiColor + line.slice(m.start, m.end) + R
    pos = m.end
  }
  out += line.slice(pos)
  return out
}

/** 设置页预览：分段 + hex 颜色 */
export function logHighlightPreviewSegments(sample, colors, rules) {
  const plain = stripTerminalControls(sample)
  if (!plain) return [{ text: sample || '', color: null }]
  if (!looksLikeLogLine(plain)) {
    return [{ text: plain, color: null }]
  }
  const config = { colors, rules }
  const picked = pickMatches(collectMatches(plain, config))
  if (!picked.length) return [{ text: plain, color: null }]
  const segs = []
  let pos = 0
  for (const m of picked) {
    if (m.start > pos) segs.push({ text: plain.slice(pos, m.start), color: null })
    segs.push({ text: plain.slice(m.start, m.end), color: m.hexColor })
    pos = m.end
  }
  if (pos < plain.length) segs.push({ text: plain.slice(pos), color: null })
  return segs
}

/** 高亮单行（不含换行符）；仅处理完整日志行，保留分页器自带的 ANSI */
export function highlightLogLine(line, config) {
  if (!line) return line
  if (hasInteractiveTerminalSequences(line)) return line
  const plain = stripTerminalControls(line).replace(/\r+$/, '')
  if (!plain || !looksLikeLogLine(plain)) return line
  return applyMatches(plain, collectMatches(plain, config))
}

/**
 * 高亮数据块中的完整行；尾部不完整行原样返回。
 * @param {string} text
 * @param {object} [config] colors + rules 或 shellLogHighlightColors/Disabled
 */
export function highlightShellChunk(text, config) {
  if (!text) return text
  if (hasInteractiveTerminalSequences(text)) return text

  let result = ''
  let start = 0
  for (let i = 0; i < text.length; i++) {
    const ch = text[i]
    if (ch === '\n') {
      let line = text.slice(start, i)
      if (line.endsWith('\r')) {
        result += highlightLogLine(line.slice(0, -1), config) + '\r\n'
      } else {
        result += highlightLogLine(line, config) + '\n'
      }
      start = i + 1
      continue
    }
    // less 搜索等行内重绘：不可剥离/重着色，否则字符错位
    if (ch === '\r' && text[i + 1] !== '\n') {
      result += text.slice(start, i + 1)
      start = i + 1
    }
  }
  result += text.slice(start)
  return result
}

/** 粗判二进制，避免把非文本当 UTF-8 高亮 */
export function isProbablyBinary(bytes) {
  if (!bytes || !bytes.length) return false
  const n = Math.min(bytes.length, 512)
  let ctrl = 0
  for (let i = 0; i < n; i++) {
    const b = bytes[i]
    if (b === 0) return true
    if (b < 8 && b !== 9 && b !== 10 && b !== 13) ctrl++
  }
  return ctrl > n * 0.25
}
