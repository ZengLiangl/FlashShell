/**
 * Shell 日志高亮：对完整行注入 ANSI（风格接近 WindTerm）。
 * - 按完整行着色，不区分命令；tail/grep/less 等只要输出整行即可命中
 * - 含光标定位/反显等交互序列的片段原样透传，避免破坏分页器重绘
 * - 支持内置级别/SQL 等规则与自定义关键字
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

export const DEFAULT_CUSTOM_KEYWORD_COLOR = '#e5c07b'
const MAX_CUSTOM_KEYWORDS = 64

/** 日志高亮配色方案（均可再自定义单项颜色） */
export const SHELL_LOG_COLOR_PRESETS = [
  {
    id: 'windterm',
    label: 'WindTerm',
    colors: { ...DEFAULT_SHELL_LOG_COLORS },
  },
  {
    id: 'vscode',
    label: 'VS Code',
    colors: {
      timestamp: '#6a9955',
      threadId: '#c586c0',
      info: '#569cd6',
      debug: '#b5cea8',
      warn: '#dcdcaa',
      error: '#f44747',
      logger: '#4ec9b0',
      sql: '#ce9178',
      label: '#9cdcfe',
    },
  },
  {
    id: 'dracula',
    label: 'Dracula',
    colors: {
      timestamp: '#50fa7b',
      threadId: '#ff79c6',
      info: '#8be9fd',
      debug: '#bd93f9',
      warn: '#f1fa8c',
      error: '#ff5555',
      logger: '#ffb86c',
      sql: '#f1fa8c',
      label: '#8be9fd',
    },
  },
  {
    id: 'nord',
    label: 'Nord',
    colors: {
      timestamp: '#a3be8c',
      threadId: '#b48ead',
      info: '#81a1c1',
      debug: '#88c0d0',
      warn: '#ebcb8b',
      error: '#bf616a',
      logger: '#8fbcbb',
      sql: '#d08770',
      label: '#88c0d0',
    },
  },
  {
    id: 'solarized',
    label: 'Solarized',
    colors: {
      timestamp: '#859900',
      threadId: '#d33682',
      info: '#268bd2',
      debug: '#2aa198',
      warn: '#b58900',
      error: '#dc322f',
      logger: '#6c71c4',
      sql: '#cb4b16',
      label: '#839496',
    },
  },
  {
    id: 'monokai',
    label: 'Monokai',
    colors: {
      timestamp: '#a6e22e',
      threadId: '#f92672',
      info: '#66d9ef',
      debug: '#ae81ff',
      warn: '#e6db74',
      error: '#f92672',
      logger: '#fd971f',
      sql: '#e6db74',
      label: '#a1efe4',
    },
  },
  {
    id: 'one-dark',
    label: 'One Dark',
    colors: {
      timestamp: '#98c379',
      threadId: '#c678dd',
      info: '#61afef',
      debug: '#56b6c2',
      warn: '#e5c07b',
      error: '#e06c75',
      logger: '#d19a66',
      sql: '#e5c07b',
      label: '#abb2bf',
    },
  },
  {
    id: 'tokyo-night',
    label: 'Tokyo Night',
    colors: {
      timestamp: '#9ece6a',
      threadId: '#bb9af7',
      info: '#7aa2f7',
      debug: '#7dcfff',
      warn: '#e0af68',
      error: '#f7768e',
      logger: '#ff9e64',
      sql: '#c0caf5',
      label: '#73daca',
    },
  },
  {
    id: 'github',
    label: 'GitHub',
    colors: {
      timestamp: '#3fb950',
      threadId: '#d2a8ff',
      info: '#58a6ff',
      debug: '#79c0ff',
      warn: '#d29922',
      error: '#ff7b72',
      logger: '#ffa657',
      sql: '#a5d6ff',
      label: '#8b949e',
    },
  },
  {
    id: 'soft',
    label: '柔和',
    colors: {
      timestamp: '#8fbc8f',
      threadId: '#c9a0dc',
      info: '#7eb6d6',
      debug: '#a8c5b0',
      warn: '#d4c07a',
      error: '#e08a8a',
      logger: '#7dbdb5',
      sql: '#d4b896',
      label: '#9bb8d0',
    },
  },
  {
    id: 'vivid',
    label: '高对比',
    colors: {
      timestamp: '#00ff88',
      threadId: '#ff66cc',
      info: '#33bbff',
      debug: '#aa88ff',
      warn: '#ffee00',
      error: '#ff3333',
      logger: '#ffaa33',
      sql: '#ffff66',
      label: '#66ffff',
    },
  },
]

export function getLogHighlightPreset(id) {
  return SHELL_LOG_COLOR_PRESETS.find((p) => p.id === id) || SHELL_LOG_COLOR_PRESETS[0]
}

/** 当前配色匹配的方案 id；无匹配则为 custom */
export function matchLogHighlightPreset(colors) {
  const merged = mergeLogHighlightColors(colors)
  for (const preset of SHELL_LOG_COLOR_PRESETS) {
    const same = LOG_HIGHLIGHT_KEYS.every(
      (key) => String(merged[key]).toLowerCase() === String(preset.colors[key]).toLowerCase(),
    )
    if (same) return preset.id
  }
  return 'custom'
}

/** 方案中的全部色值，供 color-picker 预置 */
export function collectLogHighlightPredefineColors() {
  const set = new Set(Object.values(DEFAULT_SHELL_LOG_COLORS))
  for (const preset of SHELL_LOG_COLOR_PRESETS) {
    Object.values(preset.colors).forEach((c) => set.add(c))
  }
  return [...set]
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

/** 清洗自定义关键字 */
export function normalizeCustomKeywords(list) {
  if (!Array.isArray(list)) return []
  const out = []
  const seen = new Set()
  for (const item of list) {
    const keyword = String(item?.keyword || item?.Keyword || '').trim()
    if (!keyword) continue
    const key = keyword.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    let color = String(item?.color || item?.Color || '').trim()
    if (!HEX_COLOR.test(color)) color = DEFAULT_CUSTOM_KEYWORD_COLOR
    const enabledRaw = item?.enabled ?? item?.Enabled
    const enabled = enabledRaw !== false
    out.push({ keyword, color, enabled })
    if (out.length >= MAX_CUSTOM_KEYWORDS) break
  }
  return out
}

/** 合并完整高亮配置 */
export function mergeLogHighlightConfig(input) {
  return {
    colors: mergeLogHighlightColors(input?.colors ?? input?.shellLogHighlightColors),
    rules: mergeLogHighlightRules(input?.disabled ?? input?.shellLogHighlightDisabled),
    keywords: normalizeCustomKeywords(input?.keywords ?? input?.shellLogHighlightKeywords),
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

function escapeRegExp(s) {
  return String(s).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
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

  const customs = normalizeCustomKeywords(config?.keywords)
  for (const item of customs) {
    if (!item.enabled || !item.keyword) continue
    const ansi = hexToAnsi(item.color)
    if (!ansi) continue
    const re = new RegExp(escapeRegExp(item.keyword), 'gi')
    let m
    while ((m = re.exec(line)) !== null) {
      pushMatch(matches, m.index, m.index + m[0].length, ansi, item.color)
      if (m[0].length === 0) re.lastIndex++
    }
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
export function logHighlightPreviewSegments(sample, colors, rules, keywords) {
  const plain = stripTerminalControls(sample)
  if (!plain) return [{ text: sample || '', color: null }]
  const config = { colors, rules, keywords }
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

/** 高亮单行（不含换行符）；含交互控制序列则原样返回 */
export function highlightLogLine(line, config) {
  if (!line) return line
  if (hasInteractiveTerminalSequences(line)) return line
  const plain = stripTerminalControls(line).replace(/\r+$/, '')
  if (!plain) return line
  const matches = collectMatches(plain, config)
  if (!matches.length) return line
  return applyMatches(plain, matches)
}

/**
 * 高亮数据块中的完整行；尾部不完整行原样返回。
 * 不按命令区分；整块含交互序列时仍按行处理，仅跳过带交互序列的行。
 * @param {string} text
 * @param {object} [config] colors + rules + keywords
 */
export function highlightShellChunk(text, config) {
  if (!text) return text

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
